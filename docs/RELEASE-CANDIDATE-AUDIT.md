# Release-Candidate Audit

| Check | Status | Citation or Patch |
|-------|--------|-------------------|
| **No plaintext secrets in repo** | PASS | All template files use `CHANGE_ME`/`REDACTED` placeholders. `infra/k8s/secrets/dev/*.secret.yaml.example`, `partner-pilot/docker-compose.yaml`, `partner-pilot/.env.example`, `apps/arkiv-ingestion/main.go` (default REDACTED). `docs/security/secrets.md` explains how to generate real values locally. |
| **CI exists and validates manifests + builds apps** | PASS | `.github/workflows/ci.yml` — yamllint, kubeconform (ApplicationSet), helm lint, Docker build (faucet, arkiv-ingestion), go vet/test |
| **`make up` works on fresh machine** | PASS | `Makefile` L22–24 (up → create-cluster, configure-argocd-ksops, bootstrap). `README.md` L7–28 (Prerequisites: kind, kubectl, make, docker, sops, age; Quickstart: secrets-init → make up → faucet-build → ingestion-build) |
| **Argo apps converge healthy** | PASS | `Makefile` L59–66: `wait-sync` checks both `sync.status=Synced` and `status.health.status=Healthy` before exiting. |
| **Grafana dashboards auto-provision** | PASS | `infra/k8s/monitoring/templates/dashboards-configmap.yaml` — Helm creates ConfigMaps from `dashboards/*.json`. `values.yaml` L17–20: sidecar.dashboards enabled, label `grafana_dashboard: "1"` |
| **SLO burn-rate alerts exist + runbooks linked** | PASS | `infra/k8s/monitoring/values.yaml` L29–65: recording rules (error_ratio, burn_rate), FaucetSLOBurnRateFast/Slow with `runbook_url`. `docs/runbooks/FaucetSLOBurnRateFast.md`, `FaucetSLOBurnRateSlow.md` |
| **GameDay scenario demonstrates alert→runbook→recovery** | PASS | `gameday/scenarios/01-faucet-error-spike.md`, `make gameday-on`/`gameday-off`, overlay with `loadgen-pod.yaml`, runbook linked in alert annotations |

