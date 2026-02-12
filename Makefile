# arkiv-platform-reference — make up | make down
CLUSTER_NAME ?= arkiv-platform
ARGOCD_NS ?= argocd
REPO_URL ?= https://github.com/vadym-shukurov/arkiv-sre-blueprint
AGE_KEY_FILE ?= infra/k8s/secrets/dev/age.agekey

.PHONY: up down status logs port-forward faucet-build ingestion-build secrets-init secrets-scan gameday-on gameday-off ci-local help create-cluster install-argocd configure-argocd-ksops bootstrap
.DEFAULT_GOAL := help

help:
	@echo "make up               Create cluster + Argo CD + bootstrap ApplicationSet (REPO_URL)"
	@echo "make down             Delete cluster"
	@echo "make status           Cluster, Argo apps, pods"
	@echo "make logs             Tail Argo CD logs (COMPONENT=server|repo-server)"
	@echo "make port-forward     Argo CD @ https://localhost:8080"
	@echo "make faucet-build     Build faucet image and load into kind"
	@echo "make ingestion-build  Build arkiv-ingestion image and load into kind"
	@echo "make secrets-init     Generate age key, encrypt secrets (run once)"
	@echo "make secrets-scan     Scan repo for secrets (gitleaks)"
	@echo "make gameday-on       Enable gameday overlay"
	@echo "make gameday-off      Revert faucet to normal path"
	@echo "make ci-local         Run CI locally"

up: create-cluster configure-argocd-ksops bootstrap
	@echo ">>> Waiting for Argo apps to sync..."
	@$(MAKE) wait-sync

create-cluster:
	@if ! kind get clusters 2>/dev/null | grep -q "^$(CLUSTER_NAME)$$"; then \
		kind create cluster --name=$(CLUSTER_NAME) --config=config/kind/config.yaml; \
	fi

install-argocd:
	kubectl create namespace $(ARGOCD_NS) 2>/dev/null || true
	kubectl apply -n $(ARGOCD_NS) -f https://raw.githubusercontent.com/argoproj/argo-cd/stable/manifests/install.yaml
	kubectl apply -n $(ARGOCD_NS) -f https://raw.githubusercontent.com/argoproj/applicationset/v0.4.1/manifests/install.yaml
	kubectl wait --for=condition=Available deployment/argocd-server -n $(ARGOCD_NS) --timeout=120s

configure-argocd-ksops: install-argocd
	@echo ">>> Creating sops-age secret..."
	@if [ -n "$$SOPS_AGE_KEY" ]; then \
		echo "$$SOPS_AGE_KEY" | kubectl create secret generic sops-age -n $(ARGOCD_NS) --from-file=keys.txt=/dev/stdin --dry-run=client -o yaml | kubectl apply -f -; \
	elif [ -f "$(AGE_KEY_FILE)" ]; then \
		kubectl create secret generic sops-age -n $(ARGOCD_NS) --from-file=keys.txt="$(AGE_KEY_FILE)" --dry-run=client -o yaml | kubectl apply -f -; \
	else \
		echo "ERROR: Export SOPS_AGE_KEY or run 'make secrets-init' and ensure $(AGE_KEY_FILE) exists"; exit 1; \
	fi
	@echo ">>> Patching Argo CD for KSOPS..."
	kubectl patch configmap argocd-cm -n $(ARGOCD_NS) --type=merge -p '{"data":{"kustomize.buildOptions":"--enable-alpha-plugins --enable-exec"}}'
	kubectl patch deployment argocd-repo-server -n $(ARGOCD_NS) --patch-file=infra/k8s/argocd/repo-server-ksops-patch.yaml
	kubectl rollout status deployment/argocd-repo-server -n $(ARGOCD_NS) --timeout=120s

bootstrap:
	@test -f infra/k8s/secrets/dev/grafana-admin.enc.yaml || { echo ">>> Run 'make secrets-init' first, then export SOPS_AGE_KEY"; exit 1; }
	@echo ">>> Bootstrapping ApplicationSet (REPO_URL=$(REPO_URL))"
	sed -e 's|REPO_URL_PLACEHOLDER|$(REPO_URL)|g' infra/k8s/argocd/application-set.yaml | kubectl apply -f -
	sed -e 's|REPO_URL_PLACEHOLDER|$(REPO_URL)|g' infra/k8s/argocd/application-faucet.yaml | kubectl apply -f -

secrets-init:
	@./scripts/secrets-init.sh

secrets-scan:
	@command -v gitleaks >/dev/null 2>&1 || { echo "Install gitleaks: brew install gitleaks"; exit 1; }
	gitleaks git --config .gitleaks.toml --verbose .

wait-sync:
	@for i in 1 2 3 4 5 6 7 8 9 10; do \
		synced=$$(kubectl get applications -n argocd -o jsonpath='{.items[*].status.sync.status}' 2>/dev/null | tr ' ' '\n' | grep -c Synced || echo 0); \
		healthy=$$(kubectl get applications -n argocd -o jsonpath='{.items[?(@.status.health.status=="Healthy")].metadata.name}' 2>/dev/null | wc -w | tr -d ' '); \
		total=$$(kubectl get applications -n argocd --no-headers 2>/dev/null | wc -l); \
		echo ">>> Synced $$synced/$$total, Healthy $$healthy/$$total..."; \
		[ "$$synced" -ge "$$total" ] 2>/dev/null && [ "$$healthy" -ge "$$total" ] 2>/dev/null && [ "$$total" -gt 0 ] && break; \
		sleep 15; \
	done

down:
	kind delete cluster --name=$(CLUSTER_NAME) 2>/dev/null || true

status:
	@echo ">>> Cluster:"; kind get clusters 2>/dev/null; echo
	@echo ">>> Argo CD applications:"; kubectl get applications -n $(ARGOCD_NS) 2>/dev/null; echo
	@echo ">>> Pods:"; kubectl get pods -A 2>/dev/null | head -40

logs:
	@c=$${COMPONENT:-server}; kubectl logs -n $(ARGOCD_NS) deployment/argocd-$$c -f --tail=50

port-forward:
	kubectl port-forward -n $(ARGOCD_NS) svc/argocd-server 8080:443

faucet-build:
	docker build -t faucet:latest apps/faucet
	kind load docker-image faucet:latest --name=$(CLUSTER_NAME)

ingestion-build:
	docker build -t arkiv-ingestion:latest apps/arkiv-ingestion
	kind load docker-image arkiv-ingestion:latest --name=$(CLUSTER_NAME)

# Gameday: GitOps-native failure injection (no Argo sync pause required)
gameday-on:
	@echo ">>> Switching faucet to gameday overlay"
	kubectl patch application faucet -n $(ARGOCD_NS) --type=merge -p '{"spec":{"source":{"path":"gameday/overlays/01-faucet-error-spike"}}}'
	kubectl annotate application faucet -n $(ARGOCD_NS) argocd.argoproj.io/refresh=hard --overwrite
	@echo ">>> Gameday on. Faucet returns ~20%% errors. Wait 2-5m for FaucetSLOBurnRateFast."
	@echo ">>> Run: make gameday-off to restore."

gameday-off:
	@echo ">>> Reverting faucet to normal path"
	kubectl patch application faucet -n $(ARGOCD_NS) --type=merge -p '{"spec":{"source":{"path":"apps/faucet/k8s"}}}'
	kubectl annotate application faucet -n $(ARGOCD_NS) argocd.argoproj.io/refresh=hard --overwrite
	@echo ">>> Gameday off. Verify: burn rate drops, alert resolves."

ci-local:
	$(MAKE) secrets-scan
	yamllint -c .yamllint .
	sed -e 's|REPO_URL_PLACEHOLDER|$(REPO_URL)|g' infra/k8s/argocd/application-set.yaml | kubeconform -output text -kubernetes-version 1.28 -strict -ignore-missing-schemas
	sed -e 's|REPO_URL_PLACEHOLDER|$(REPO_URL)|g' infra/k8s/argocd/application-faucet.yaml | kubeconform -output text -kubernetes-version 1.28 -strict -ignore-missing-schemas
	kubectl kustomize gameday/overlays/01-faucet-error-spike > /dev/null
	helm dependency update infra/k8s/monitoring && helm dependency update apps/blockscout
	helm lint infra/k8s/monitoring && helm lint apps/blockscout
	docker build -t faucet:ci apps/faucet && docker build -t arkiv-ingestion:ci apps/arkiv-ingestion
	cd apps/faucet && go mod tidy && go vet ./... && go test ./...
	cd apps/arkiv-ingestion && go mod tidy && go vet ./... && go test ./...
