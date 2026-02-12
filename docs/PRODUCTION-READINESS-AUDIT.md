# Production Readiness Audit

**Date:** 2025-02  
**Reviewer:** Staff+ SRE (Head of Platform)  
**Repo:** arkiv-platform-reference / arkiv-sre-blueprint

---

## A) Repo Intent Check

This repo proves:

1. **GitOps + K8s:** Argo CD ApplicationSet (`infra/k8s/argocd/application-set.yaml`) deploys secrets (KSOPS), observability (kube-prometheus-stack), ingress, blockscout, arkiv-ingestion. Standalone Application (`application-faucet.yaml`) enables gameday overlay switching. `make up` creates kind cluster, installs Argo CD, bootstraps apps.

2. **Observability/SLOs:** kube-prometheus-stack with SLO burn-rate alerts (Faucet, Blockscout, Ingestion), Grafana dashboards (slo-dashboard, app-overview, cluster-overview), runbooks linked from alerts.

3. **Blockscout, onboarding, partner pilot, incident response:** Blockscout Helm app with persistence; partner-pilot Docker Compose; `docs/incident-response.md` + runbooks; gameday scenario (GitOps-native) + postmortem template.

---

## B) Reproducibility Checks

### Fresh machine path

**Exact commands from zero → running:**

```bash
# 1. Install tools (macOS)
brew install kind kubectl make sops age
# Docker Desktop required for kind and image builds

# 2. Clone and cd
git clone <repo-url> && cd <repo-dir>

# 3. Edit secrets: replace CHANGE_ME in infra/k8s/secrets/dev/*.secret.yaml.example
#    Use admin/postgres for local dev (docs/security/secrets.md)

# 4. Generate age key and encrypt
make secrets-init

# 5. Commit encrypted secrets (required for Argo sync)
git add infra/k8s/secrets/dev/*.enc.yaml
git commit -m "Add encrypted secrets" || true

# 6. Export key and push to Git remote (REPO_URL must be reachable by Argo)
export SOPS_AGE_KEY=$(cat infra/k8s/secrets/dev/age.agekey)
git push <remote> main

# 7. Bring up cluster
make up

# 8. Build app images (required for faucet + arkiv-ingestion to become Healthy)
make faucet-build
make ingestion-build

# 9. Verify
make status
make port-forward
```

| Check | PASS/FAIL | Evidence |
|-------|-----------|----------|
| `make up` | PASS | Makefile L24-26: up → create-cluster, configure-argocd-ksops, bootstrap. Precondition: `grafana-admin.enc.yaml` exists (L54). |
| `make down` | PASS | Makefile L76-77: `kind delete cluster` |
| `make status` | PASS | L79-83: kind clusters, Argo apps, pods |
| `make logs` | PASS | L84-85: `kubectl logs` with COMPONENT=server\|repo-server |
| Argo apps Healthy/Synced | PASS | wait-sync L66-74: loops until synced/healthy ≥ total |
| Port-forward URLs | PASS | README L48-51: Grafana 3000, blockscout 8080, faucet 8081, ingestion 8082 |
| Creds (Argo password) | PASS | README L39-40: `kubectl -n argocd get secret argocd-initial-admin-secret ...` |

**FAIL:** README Quickstart L22-31 omits (a) `export SOPS_AGE_KEY` before `make up`, (b) commit enc.yaml before push. Bootstrap requires grafana-admin.enc.yaml; Argo secrets app requires *.enc.yaml in repo. **P1.**

**WARN:** `wait-sync` may timeout (10×15s) if faucet/arkiv-ingestion stay Degraded (images not built). Bootstrap runs before build in Quickstart order.

---

## C) Security Checks

| Check | PASS/FAIL | Evidence |
|-------|-----------|----------|
| No plaintext secrets in git | PASS | `.gitleaks.toml` allowlist: REDACTED, CHANGE_ME; `*_test.go` paths. `*.secret.yaml.example` use placeholders. `.gitignore` L22: `infra/k8s/secrets/**/age.agekey` |
| SOPS/age setup | PASS | `scripts/secrets-init.sh`, `infra/k8s/secrets/.sops.yaml`, `docs/security/secrets.md`. KSOPS patch: `infra/k8s/argocd/repo-server-ksops-patch.yaml` |
| RBAC least privilege | PASS | Dedicated SAs: faucet-sa, arkiv-ingestion-sa, arkiv-ingestion-db-sa, loadgen-faucet-sa; automountServiceAccountToken: false. `docs/security/rbac.md` |
| Network exposure | PASS | Services ClusterIP; ingress controller hostPort for kind |

**Gap:** `*.enc.yaml` not in repo by default. User must run `make secrets-init` and commit enc.yaml. `docs/security/secrets.md` L17 mentions "commit *.enc.yaml" but README does not.

---

## D) Observability & SLO Checks

### SLIs/SLOs implemented

| SLO | Target | Defined in |
|-----|--------|------------|
| Faucet availability | ≥ 99.9% | `infra/k8s/monitoring/values.yaml` L29-56, `docs/slos.md` |
| Faucet p95 latency | < 2s | values.yaml L66-74 |
| Ingestion availability | ≥ 99.9% | values.yaml L88-96 |
| Blockscout availability | ≥ 99.9% | values.yaml L110-156, docs/slos.md |
| Node readiness | 100% | values.yaml L162-169 |

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
| SLO | `infra/k8s/monitoring/dashboards/slo-dashboard.json` | Via `templates/dashboards-configmap.yaml` (label grafana_dashboard: "1") |
| App overview | `dashboards/app-overview.json` | Yes |
| Cluster overview | `dashboards/cluster-overview.json` | Yes |

### Runbooks

All 9 runbooks exist: FaucetSLOBurnRateFast/Slow, FaucetHighLatency, FaucetRateLimitSpike, IngestionHighErrorRate, BlockscoutHighErrorRate, NodeNotReady, HighPodRestartRate, PrometheusDown.

**Gap:** `runbook_url` hardcoded to `https://github.com/arkiv/arkiv-platform-reference`. If forked, links 404. **P1.**

---

## E) App Checks

### Faucet

| Check | PASS/FAIL | Evidence |
|-------|-----------|----------|
| /healthz | PASS | `apps/faucet/main.go` L175-183, deployment.yaml L32-42 |
| /metrics | PASS | main.go L119, ServiceMonitor L61-72 |
| Rate limit | PASS | perIPLimit=10, perAddrLimit=2 (main.go L19-23); faucet_rate_limit_total |
| Liveness/readiness | PASS | deployment.yaml L33-42 |

### Ingestion

| Check | PASS/FAIL | Evidence |
|-------|-----------|----------|
| Idempotency | PASS | `ingester_postgres.go` L46-49: ON CONFLICT (idempotency_key) DO NOTHING |
| Retries/backoff | PASS | `main.go` L107-123: 3 attempts, exponential backoff |
| Config | PASS | DATABASE_URL, INGEST_INTERVAL_SEC, CHAIN_ID from env |
| Demo mode | PASS | `fetcher_synthetic.go`: synthetic blocks |
| Liveness/readiness | PASS | deployment.yaml L41-51 |

### Blockscout

| Check | PASS/FAIL | Evidence |
|-------|-----------|----------|
| Persistence | PASS | values.yaml L24-27: persistence.enabled: true, 10Gi |
| Resources | PASS | L27-34, L71-76 |
| Prometheus | PASS | values.yaml L38: prometheus.enabled: true |
| Operational docs | PASS | `docs/operations/blockscout.md` |

**Gap:** `arkiv-ingestion-db` uses emptyDir (deployment.yaml L123-124). Comment: "demo only; data lost on pod restart."

---

## F) GameDay & Incident Response

| Check | PASS/FAIL | Evidence |
|-------|-----------|----------|
| Scenario executable | PASS | `make gameday-on` (Makefile L100-101) patches application path to `gameday/overlays/01-faucet-error-spike` |
| Overlay | PASS | `gameday/overlays/01-faucet-error-spike`: deployment-patch (FORCE_ERROR_RATE=0.2), loadgen-pod |
| Postmortem template | PASS | `gameday/templates/postmortem.md` |
| Example filled | PASS | `gameday/postmortems/faucet-error-spike-sample.md` |

**Evidence:** `kubectl kustomize gameday/overlays/01-faucet-error-spike` — overlay includes loadgen-pod.yaml.

---

## G) CI Checks

| Step | PASS/FAIL | Evidence |
|------|-----------|----------|
| Secret scan (gitleaks) | PASS | `.github/workflows/ci.yml` L18-21 |
| YAML lint | PASS | L23-26, `.yamllint` |
| Kubeconform | PASS | L28-34 |
| Kustomize build | PASS | L34-35 |
| Helm lint | PASS | L37-41 |
| Docker build | PASS | L43-55 |
| Go vet/test | PASS | L57-64 |

**Triggers:** push/PR to main, master; workflow_dispatch.

**Missing (high value):** Terraform fmt/validate (N/A — no Terraform). Consider: ApplicationSet dry-run sync, Prometheus rule validation.

---

## H) Required Fixes

### P0 (must fix)

None.

### P1

1. **README Quickstart: SOPS_AGE_KEY and enc.yaml**
   - **File:** `README.md`
   - **Change:** Add after `make secrets-init`:
     ```bash
     export SOPS_AGE_KEY=$(cat infra/k8s/secrets/dev/age.agekey)
     # Commit enc.yaml before push (Argo syncs secrets from repo)
     git add infra/k8s/secrets/dev/*.enc.yaml && git commit -m "Add encrypted secrets" || true
     ```
   - **Evidence:** Bootstrap requires grafana-admin.enc.yaml; Argo secrets app requires *.enc.yaml in repo path.

2. **runbook_url for forks**
   - **File:** `README.md` (REPO_URL section)
   - **Change:** Already documents "If forked, update runbook_url... grep for arkiv-platform-reference." — verify prominenence.

### P2

3. **ci-local: gitleaks first fails**
   - **File:** `README.md`
   - **Change:** Prominently list gitleaks in Prerequisites.

4. **emptyDir for arkiv-ingestion-db**
   - Documented in deployment comment. OK for demo.

---

## Go/No-Go

**Verdict: GO**

No P0 blockers. Repo delivers GitOps, K8s, observability, SLO burn-rate alerts, runbooks, gameday. P1 fixes improve reproducibility (SOPS_AGE_KEY, enc.yaml commit); recommended before release.

---

## Evidence Screenshots (add to README)

1. **Argo CD UI** — Applications list: all apps Synced (green), Health Healthy (green). URL: https://localhost:8080 (after `make port-forward`).
2. **Grafana SLO Dashboard** — Panels: Faucet error budget remaining, Faucet burn rate (5m/1h), Blockscout burn rate. URL: http://localhost:3000 → SLO Dashboard.
3. **Prometheus Alerts** — Firing or Pending with runbook_url. URL: http://localhost:9090/alerts (port-forward to Prometheus).
4. **Faucet health** — Terminal: `curl -s http://localhost:8081/healthz` → `ok`.
5. **GameDay recovery** — Grafana: burn rate > 14.4 (gameday-on) vs < 14.4 (gameday-off). Or Prometheus: alert Firing → Resolved.
