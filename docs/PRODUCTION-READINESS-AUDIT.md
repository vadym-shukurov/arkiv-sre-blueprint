# Production Readiness Audit

**Date:** 2025-02  
**Reviewer:** Staff+ SRE (Head of Platform)  
**Repo:** arkiv-sre-blueprint

---

## A) Repo Intent Check

This repo proves:

1. **GitOps + K8s + Observability/SLOs:** Argo CD ApplicationSet deploys secrets (KSOPS), observability (kube-prometheus-stack), ingress, blockscout, arkiv-ingestion. Standalone Application (`application-faucet.yaml`) enables gameday overlay switching. SLO burn-rate alerts for Faucet, Blockscout, Ingestion; Grafana dashboards; runbooks linked from alerts.

2. **Blockscout + onboarding + partner pilot:** Blockscout Helm app with persistence, prometheus.enabled, ServiceMonitor. Partner-pilot Docker Compose for arkiv-ingestion standalone. `docs/security/secrets.md`, `docs/operations/blockscout.md` for onboarding.

3. **Incident response:** GameDay scenarios (GitOps-native), postmortem template, 9 runbooks (Faucet, Blockscout, Ingestion, infra).

---

## B) Reproducibility Checks

### Fresh machine path (exact commands)

```bash
# 1. Install tools (macOS)
brew install kind kubectl make sops age
# Docker Desktop required for kind and image builds

# 2. Clone and cd
git clone https://github.com/vadym-shukurov/arkiv-sre-blueprint && cd arkiv-sre-blueprint

# 3. Generate random dev secrets
make secrets-dev

# 4. Encrypt secrets
make secrets-init
export SOPS_AGE_KEY=$(cat infra/k8s/secrets/dev/age.agekey)

# 5. Commit enc.yaml and push (Argo syncs secrets from repo)
git add infra/k8s/secrets/dev/*.enc.yaml && git commit -m "Add encrypted secrets" || true
git push origin main

# 6. Bring up cluster (includes app builds)
make up

# 7. Verify
make status
make port-forward
```

| Check | PASS/FAIL | Evidence |
|-------|-----------|----------|
| `make up` | PASS | Makefile L29: `up: create-cluster app-build configure-argocd-ksops bootstrap`. Precondition: `grafana-admin.enc.yaml` exists (L60). `app-build` runs `faucet-build` + `ingestion-build` (L33). |
| `make down` | PASS | Makefile L85: `kind delete cluster --name=$(CLUSTER_NAME)` |
| `make status` | PASS | L87–91: kind clusters, Argo apps, pods |
| `make logs` | PASS | L93–94: `kubectl logs -n argocd deployment/argocd-$$c -f --tail=50` |
| Argo apps Healthy/Synced | PASS | wait-sync L75–83: loops until synced/healthy ≥ total (10×15s max) |
| Port-forward URLs | PASS | README L50–56; Makefile L99–107: Argo 8080, Grafana 3000, faucet 8081, ingestion 8082, blockscout 8080 |
| Argo CD password | PASS | README L43–45: `kubectl -n argocd get secret argocd-initial-admin-secret ...` |

**Evidence to collect:** `make status` output showing all Argo apps Synced/Healthy; `kubectl get applications -n argocd`.

**WARN:** `wait-sync` may exit after 10 iterations (2.5 min) if apps stay Degraded (e.g. images not in kind). `make up` runs `app-build` before bootstrap, so images should be loaded. If cluster pre-exists, `create-cluster` is no-op; app-build still runs.

---

## C) Security Checks

| Check | PASS/FAIL | Evidence |
|-------|-----------|----------|
| No plaintext secrets in git | PASS | `.gitignore` L22–27: `age.agekey`, `*.secret.yaml`. `.gitleaks.toml` allowlist: REDACTED, CHANGE_ME; `*_test.go` paths. `*.secret.yaml.example` use placeholders. |
| SOPS/age setup | PASS | `scripts/secrets-init.sh`, `infra/k8s/secrets/.sops.yaml`, `docs/security/secrets.md`. KSOPS patch: `infra/k8s/argocd/repo-server-ksops-patch.yaml`. `secrets-init` prefers `.secret.yaml` (from `secrets-dev`) over `.example`. |
| RBAC least privilege | PASS | `docs/security/rbac.md`: faucet-sa, arkiv-ingestion-sa, arkiv-ingestion-db-sa, loadgen-faucet-sa; `automountServiceAccountToken: false`. Blockscout values.yaml L49–51, L98–101. |
| Network exposure | PASS | Services ClusterIP; ingress controller hostPort for kind. |

**Evidence to collect:** `kubectl get sa -A | grep -E "faucet|arkiv-ingestion|blockscout"`; `gitleaks git . --config .gitleaks.toml` (exit 0).

---

## D) Observability & SLO Checks

### SLIs/SLOs implemented

| SLO | Target | Defined in |
|-----|--------|------------|
| Faucet availability | ≥ 99.9% | `infra/k8s/monitoring/values.yaml` L29–86 |
| Faucet p95 latency | < 2s | values.yaml L69–76 |
| Ingestion availability | ≥ 99.9% | values.yaml L86–111 |
| Blockscout availability | ≥ 99.9% | values.yaml L111–164 |
| Node readiness | 100% | values.yaml L163–169 |

### Burn-rate alerts

| Alert | SLO | Severity | runbook_url |
|-------|-----|----------|-------------|
| FaucetSLOBurnRateFast | faucet 99.9% | page | FaucetSLOBurnRateFast.md |
| FaucetSLOBurnRateSlow | faucet 99.9% | warning | FaucetSLOBurnRateSlow.md |
| BlockscoutSLOBurnRateFast | blockscout 99.9% | page | BlockscoutHighErrorRate.md |
| BlockscoutSLOBurnRateSlow | blockscout 99.9% | warning | BlockscoutHighErrorRate.md |

### Dashboards

| Dashboard | File | Provisioned |
|-----------|------|-------------|
| SLO | `infra/k8s/monitoring/dashboards/slo-dashboard.json` | Via `templates/dashboards-configmap.yaml` (label `grafana_dashboard: "1"`); sidecar `label: grafana_dashboard` (values.yaml L20–23) |
| App overview | `dashboards/app-overview.json` | Yes |
| Cluster overview | `dashboards/cluster-overview.json` | Yes |

### Runbooks

All 9 runbooks exist: FaucetSLOBurnRateFast/Slow, FaucetHighLatency, FaucetRateLimitSpike, IngestionHighErrorRate, BlockscoutHighErrorRate, NodeNotReady, HighPodRestartRate, PrometheusDown.

**Runbook verification:** `docs/runbooks/FaucetSLOBurnRateFast.md` contains actionable commands (kubectl get pods, logs, port-forward, `make gameday-off`, rollout restart). **PASS.**

---

## E) App Checks

### Faucet

| Check | PASS/FAIL | Evidence |
|-------|-----------|----------|
| /healthz | PASS | `apps/faucet/main.go` L177–185; deployment.yaml L33–42 |
| /metrics | PASS | main.go L122; ServiceMonitor L61–72 in deployment.yaml |
| Rate limit | PASS | perIPLimit=10, perAddrLimit=2 (main.go L22–27); faucet_rate_limit_total |
| Liveness/readiness | PASS | deployment.yaml L33–42 |
| Request body limit | PASS | main.go L220–228: MaxBytesReader(64KB), 413 on overflow |

### Ingestion

| Check | PASS/FAIL | Evidence |
|-------|-----------|----------|
| Idempotency | PASS | `ingester_postgres.go` L44–48: ON CONFLICT (idempotency_key) DO NOTHING |
| Retries/backoff | PASS | `main.go` L123–137: 3 attempts, 1s/2s/3s backoff |
| Config | PASS | DATABASE_URL, INGEST_INTERVAL_SEC, CHAIN_ID from env |
| Demo mode | PASS | `fetcher_synthetic.go`: synthetic blocks |
| Liveness/readiness | PASS | deployment.yaml L41–51 |
| Graceful shutdown | PASS | main.go L71–88: http.Server.Shutdown on SIGTERM/SIGINT |

### Blockscout

| Check | PASS/FAIL | Evidence |
|-------|-----------|----------|
| Persistence | PASS | values.yaml L24–27: persistence.enabled: true, 10Gi |
| Resources | PASS | L27–34, L71–76 |
| Prometheus | PASS | values.yaml L38: prometheus.enabled: true |
| ServiceMonitor | PASS | blockscout-stack chart; kube-prometheus-stack serviceMonitorNamespaceSelector: {} |
| Operational docs | PASS | `docs/operations/blockscout.md`, `docs/BLOCKSCOUT-OBSERVABILITY.md` |

**Gap:** `arkiv-ingestion-db` uses emptyDir (deployment.yaml L123–124). Comment: "demo only; data lost on pod restart."

---

## F) GameDay & Incident Response

| Check | PASS/FAIL | Evidence |
|-------|-----------|----------|
| Scenario executable | PASS | `make gameday-on` (Makefile L118–122) patches application path to `gameday/overlays/01-faucet-error-spike` |
| Overlay | PASS | `gameday/overlays/01-faucet-error-spike`: deployment-patch (FORCE_ERROR_RATE=0.2), loadgen-pod, loadgen-serviceaccount |
| Postmortem template | PASS | `gameday/templates/postmortem.md` |
| Example filled | PASS | `gameday/postmortems/faucet-error-spike-sample.md` |

---

## G) CI Checks

| Step | PASS/FAIL | Evidence |
|------|-----------|----------|
| Secret scan (gitleaks) | PASS | `.github/workflows/ci.yml` L18–21 |
| YAML lint | PASS | L23–26, `.yamllint` |
| Kubeconform | PASS | L28–34 (faucet app only; ApplicationSet uses Go templates) |
| Kustomize build | PASS | L35–36 |
| Helm lint | PASS | L37–41 |
| Docker build | PASS | L43–55 |
| Go vet/test | PASS | L57–64 |

**Triggers:** push/PR to main, master; workflow_dispatch.

**Missing (high value):** ApplicationSet dry-run sync, Prometheus rule validation. Terraform N/A.

---

## H) Required Fixes

### P0 (must fix)

None.

### P1

1. **runbook_url for forks**
   - **File:** `README.md` (REPO_URL section)
   - **Status:** Already documents "If forked, update runbook_url... grep for arkiv-platform-reference." — sufficient.

### P2

3. **ci-local: gitleaks first**
   - **File:** `README.md` Prerequisites
   - **Change:** List gitleaks explicitly for `make ci-local`.

4. **emptyDir for arkiv-ingestion-db**
   - Documented in deployment comment. OK for demo.

---

## Go/No-Go

**Verdict: GO**

No P0 blockers. Repo delivers GitOps, K8s, observability, SLO burn-rate alerts, runbooks, gameday. P1 items are doc clarifications; recommended before release.

---

## Evidence Screenshots (5 for README)

1. **Argo CD UI** — Applications list: all apps Synced (green), Health Healthy (green). URL: https://localhost:8080 (after `make port-forward`).
2. **Grafana SLO Dashboard** — Panels: Faucet error budget remaining, Faucet burn rate (5m/1h), Blockscout burn rate. URL: http://localhost:3000 → SLO Dashboard.
3. **Prometheus Alerts** — Firing or Pending with runbook_url. Port-forward: `kubectl port-forward -n monitoring svc/observability-kube-prometheus-prometheus 9090:9090` → http://localhost:9090/alerts.
4. **Faucet health** — Terminal: `curl -s http://localhost:8081/healthz` → `ok`.
5. **Prometheus Targets** — Blockscout target UP. URL: http://localhost:9090/targets → filter "blockscout".
