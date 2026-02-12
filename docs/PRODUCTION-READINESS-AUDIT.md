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

# 3. Edit secrets: replace CHANGE_ME in infra/k8s/secrets/dev/*.secret.yaml.example
#    Use admin/postgres for local dev (docs/security/secrets.md)

# 4. Generate age key and encrypt
make secrets-init

# 5. Export key and commit enc.yaml (required for Argo sync)
export SOPS_AGE_KEY=$(cat infra/k8s/secrets/dev/age.agekey)
git add infra/k8s/secrets/dev/*.enc.yaml && git commit -m "Add encrypted secrets" || true
git push origin main

# 6. Bring up cluster
make up

# 7. Build app images (required for faucet + arkiv-ingestion to become Healthy)
make faucet-build
make ingestion-build

# 8. Verify
make status
make port-forward
```

| Check | PASS/FAIL | Evidence |
|-------|-----------|----------|
| `make up` | PASS | Makefile L24-26: up → create-cluster, configure-argocd-ksops, bootstrap. Precondition: `grafana-admin.enc.yaml` exists (L53). |
| `make down` | PASS | Makefile L76-77: `kind delete cluster` |
| `make status` | PASS | L79-83: kind clusters, Argo apps, pods |
| `make logs` | PASS | L84-85: `kubectl logs` with COMPONENT=server\|repo-server |
| Argo apps Healthy/Synced | PASS | wait-sync L66-74: loops until synced/healthy ≥ total |
| Port-forward URLs | PASS | README L52-56: Grafana 3000, blockscout 8080, faucet 8081, ingestion 8082 |
| Argo CD password | PASS | README L43-46: `kubectl -n argocd get secret argocd-initial-admin-secret ...` |

**FAIL:** `docs/security/secrets.md` L5-15 does not mention "commit and push *.enc.yaml before make up". Argo secrets app syncs from repo; enc.yaml must be committed. **P1.**

**WARN:** `wait-sync` may timeout (10×15s) if faucet/arkiv-ingestion stay Degraded (images not built). Bootstrap runs before build in Quickstart order.

---

## C) Security Checks

| Check | PASS/FAIL | Evidence |
|-------|-----------|----------|
| No plaintext secrets in git | PASS | `.gitleaks.toml` allowlist: REDACTED, CHANGE_ME; `*_test.go` paths. `*.secret.yaml.example` use placeholders. `.gitignore` L22: `infra/k8s/secrets/**/age.agekey` |
| SOPS/age setup | PASS | `scripts/secrets-init.sh`, `infra/k8s/secrets/.sops.yaml`, `docs/security/secrets.md`. KSOPS patch: `infra/k8s/argocd/repo-server-ksops-patch.yaml` |
| RBAC least privilege | PASS | Dedicated SAs: faucet-sa, arkiv-ingestion-sa, arkiv-ingestion-db-sa, loadgen-faucet-sa; automountServiceAccountToken: false. `docs/security/rbac.md` |
| Network exposure | PASS | Services ClusterIP; ingress controller hostPort for kind |

**Gap:** `*.enc.yaml` not in repo by default. User must run `make secrets-init` and commit enc.yaml. `docs/security/secrets.md` L17 mentions "commit *.enc.yaml" for rotation but not in initial path.

---

## D) Observability & SLO Checks

### SLIs/SLOs implemented

| SLO | Target | Defined in |
|-----|--------|------------|
| Faucet availability | ≥ 99.9% | `infra/k8s/monitoring/values.yaml` L29-86, `docs/slos.md` |
| Faucet p95 latency | < 2s | values.yaml L69-76 |
| Ingestion availability | ≥ 99.9% | values.yaml L86-111 |
| Blockscout availability | ≥ 99.9% | values.yaml L111-164, docs/slos.md |
| Node readiness | 100% | values.yaml L163-169 |

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

**Runbook verification:** `docs/runbooks/FaucetSLOBurnRateFast.md` contains actionable commands (kubectl get pods, logs, port-forward, make gameday-off, rollout restart). **PASS.**

---

## E) App Checks

### Faucet

| Check | PASS/FAIL | Evidence |
|-------|-----------|----------|
| /healthz | PASS | `apps/faucet/main.go` L175-184, deployment.yaml L34-41 |
| /metrics | PASS | main.go L119, ServiceMonitor L61-72 in deployment.yaml |
| Rate limit | PASS | perIPLimit=10, perAddrLimit=2 (main.go L19-25); faucet_rate_limit_total |
| Liveness/readiness | PASS | deployment.yaml L33-42 |
| Request body limit | PASS | main.go L220: MaxBytesReader(64KB), 413 on overflow |

### Ingestion

| Check | PASS/FAIL | Evidence |
|-------|-----------|----------|
| Idempotency | PASS | `ingester_postgres.go` L46-49: ON CONFLICT (idempotency_key) DO NOTHING |
| Retries/backoff | PASS | `main.go` L123-137: 3 attempts, 1s/2s/3s backoff |
| Config | PASS | DATABASE_URL, INGEST_INTERVAL_SEC, CHAIN_ID from env |
| Demo mode | PASS | `fetcher_synthetic.go`: synthetic blocks |
| Liveness/readiness | PASS | deployment.yaml L41-51 |
| Graceful shutdown | PASS | main.go L71-88: http.Server.Shutdown on SIGTERM/SIGINT |

### Blockscout

| Check | PASS/FAIL | Evidence |
|-------|-----------|----------|
| Persistence | PASS | values.yaml L24-27: persistence.enabled: true, 10Gi |
| Resources | PASS | L27-34, L71-76 |
| Prometheus | PASS | values.yaml L38: prometheus.enabled: true |
| ServiceMonitor | PASS | blockscout-stack chart creates ServiceMonitor; kube-prometheus-stack serviceMonitorNamespaceSelector discovers blockscout |
| Operational docs | PASS | `docs/operations/blockscout.md`, `docs/BLOCKSCOUT-OBSERVABILITY.md` |

**Gap:** `arkiv-ingestion-db` uses emptyDir (deployment.yaml L123-124). Comment: "demo only; data lost on pod restart."

---

## F) GameDay & Incident Response

| Check | PASS/FAIL | Evidence |
|-------|-----------|----------|
| Scenario executable | PASS | `make gameday-on` (Makefile L100-101) patches application path to `gameday/overlays/01-faucet-error-spike` |
| Overlay | PASS | `gameday/overlays/01-faucet-error-spike`: deployment-patch (FORCE_ERROR_RATE=0.2), loadgen-pod |
| Postmortem template | PASS | `gameday/templates/postmortem.md` |
| Example filled | PASS | `gameday/postmortems/faucet-error-spike-sample.md` |

---

## G) CI Checks

| Step | PASS/FAIL | Evidence |
|------|-----------|----------|
| Secret scan (gitleaks) | PASS | `.github/workflows/ci.yml` L18-21 |
| YAML lint | PASS | L23-26, `.yamllint` |
| Kubeconform | PASS | L28-34 (faucet app only; ApplicationSet skipped due to Go templates) |
| Kustomize build | PASS | L35-36 |
| Helm lint | PASS | L37-41 |
| Docker build | PASS | L43-55 |
| Go vet/test | PASS | L57-64 |

**Triggers:** push/PR to main, master; workflow_dispatch.

**Missing (high value):** ApplicationSet dry-run sync, Prometheus rule validation. Terraform N/A (no Terraform).

---

## H) Required Fixes

### P0 (must fix)

None.

### P1

1. **docs/security/secrets.md — Commit enc.yaml before make up**
   - **File:** `docs/security/secrets.md`
   - **Change:** In "Main path: K8s" section, add step between secrets-init and make up:
     ```bash
     git add infra/k8s/secrets/dev/*.enc.yaml && git commit -m "Add encrypted secrets" || true
     git push origin main  # Argo syncs secrets from repo
     ```
   - **Evidence:** Argo secrets app path `infra/k8s/secrets/dev`; enc.yaml must be in repo for sync.

2. **runbook_url for forks**
   - **File:** `README.md` (REPO_URL section)
   - **Change:** Already documents "If forked, update runbook_url... grep for arkiv-platform-reference." — verify prominence. Runbook URLs point to vadym-shukurov/arkiv-sre-blueprint; forks need to update.

### P2

3. **ci-local: gitleaks first fails**
   - **File:** `README.md`
   - **Change:** Prominently list gitleaks in Prerequisites.

4. **emptyDir for arkiv-ingestion-db**
   - Documented in deployment comment. OK for demo.

---

## Go/No-Go

**Verdict: GO**

No P0 blockers. Repo delivers GitOps, K8s, observability, SLO burn-rate alerts, runbooks, gameday. P1 fixes improve reproducibility (secrets doc commit step); recommended before release.

---

## Evidence Screenshots (min. 5 for README)

1. **Argo CD UI** — Applications list: all apps Synced (green), Health Healthy (green). URL: https://localhost:8080 (after `make port-forward`).
2. **Grafana SLO Dashboard** — Panels: Faucet error budget remaining, Faucet burn rate (5m/1h), Blockscout burn rate. URL: http://localhost:3000 → SLO Dashboard.
3. **Prometheus Alerts** — Firing or Pending with runbook_url. Port-forward: `kubectl port-forward -n monitoring svc/observability-kube-prometheus-prometheus 9090:9090` → http://localhost:9090/alerts.
4. **Faucet health** — Terminal: `curl -s http://localhost:8081/healthz` → `ok`.
5. **Prometheus Targets** — Blockscout target UP. URL: http://localhost:9090/targets → filter "blockscout".
6. **GameDay recovery** — Grafana: burn rate > 14.4 (gameday-on) vs < 14.4 (gameday-off). Or Prometheus: alert Firing → Resolved.
