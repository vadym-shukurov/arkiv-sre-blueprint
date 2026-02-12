# arkiv-sre-blueprint

[![CI](https://github.com/vadym-shukurov/arkiv-sre-blueprint/actions/workflows/ci.yml/badge.svg)](https://github.com/vadym-shukurov/arkiv-sre-blueprint/actions/workflows/ci.yml)

Platform reference stack. Run locally on **kind**: `make up`. Teardown: `make down`.

## Prerequisites

kind, kubectl, make, docker, sops, age

```bash
brew install kind kubectl make sops age
# Docker Desktop required for kind and image builds
# For make ci-local: brew install yamllint kubeconform gitleaks helm
```

**CI:** `make ci-local` (requires yamllint, kubeconform, helm, go, gitleaks). Runs on push/PR; manual run available via workflow_dispatch.

## Quickstart

```bash
# 1. Generate random dev secrets (or edit *.secret.yaml.example manually)
make secrets-dev
# 2. Encrypt secrets
make secrets-init
export SOPS_AGE_KEY=$(cat infra/k8s/secrets/dev/age.agekey)
# 3. Commit enc.yaml and push (Argo syncs secrets from repo)
git add infra/k8s/secrets/dev/*.enc.yaml && git commit -m "Add encrypted secrets" || true
git push origin main
# 4. Bring up cluster (includes app builds)
make up
make status
make pf-grafana   # see Port-forward for others
make down
```

Secrets: [docs/security/secrets.md](docs/security/secrets.md) | RBAC: [docs/security/rbac.md](docs/security/rbac.md)

**Security:** Workloads do not run under default SA; token automount disabled by default.

**Argo CD password:**
```bash
kubectl -n argocd get secret argocd-initial-admin-secret -o jsonpath="{.data.password}" | base64 -d
```

## Port-forward

`make port-forward` → Argo CD @ https://localhost:8080. For other services, run in separate terminals:

| Target | Command | URL |
|--------|---------|-----|
| Argo CD | `make port-forward` | https://localhost:8080 |
| Grafana | `make pf-grafana` | http://localhost:3000 |
| Faucet | `make pf-faucet` | http://localhost:8081 |
| Arkiv-ingestion | `make pf-ingestion` | http://localhost:8082 |
| Blockscout | `make pf-blockscout` | http://localhost:8080 |

## Apps

Faucet: [docs/faucet.md](docs/faucet.md) | Arkiv-ingestion: [partner-pilot/README.md](partner-pilot/README.md)

## Run a GameDay

Full on-call loop: inject → observe dashboard → alert fires → runbook → recover → postmortem. GitOps-native; no Argo sync pause required.

```bash
make up
make pf-grafana
make gameday-on
# Wait 2–5 min for FaucetSLOBurnRateFast. Follow runbook. Then:
make gameday-off
```

→ [gameday/README.md](gameday/README.md) | Postmortem: [gameday/templates/postmortem.md](gameday/templates/postmortem.md)

## Structure

```
config/kind/       # Kind config
infra/k8s/         # Argo apps, monitoring (Helm), secrets
apps/              # blockscout, faucet, arkiv-ingestion
docs/              # runbooks, incident-response, slos, security
gameday/           # chaos scenarios, postmortem template
```

## REPO_URL

Push to a Git remote before `make up` (Argo syncs from REPO_URL). Default: `https://github.com/vadym-shukurov/arkiv-sre-blueprint`. Override: `REPO_URL=https://github.com/your-org/repo make up`. If forked, update `runbook_url` in `infra/k8s/monitoring/values.yaml` (grep for arkiv-platform-reference).

## Security checks

Secret scanning runs in CI and locally via [gitleaks](https://github.com/gitleaks/gitleaks). Fail the build if secrets are detected.

```bash
make secrets-scan
```

Placeholders (`REDACTED`, `CHANGE_ME`) are allowlisted in `.gitleaks.toml`. See [docs/security/secrets.md](docs/security/secrets.md) for how to set real values.

## Evidence / Production Readiness

Audit: [docs/RELEASE-CANDIDATE-AUDIT.md](docs/RELEASE-CANDIDATE-AUDIT.md) | [docs/PRODUCTION-READINESS-AUDIT.md](docs/PRODUCTION-READINESS-AUDIT.md) | QE: [docs/RELEASE-CANDIDATE-QE.md](docs/RELEASE-CANDIDATE-QE.md) | **Verdict:** [docs/RELEASE-CANDIDATE-QE-VERDICT.md](docs/RELEASE-CANDIDATE-QE-VERDICT.md)

**Evidence screenshots:** (1) Argo CD UI — all apps Synced/Healthy; (2) Grafana SLO Dashboard — Faucet burn rate panels; (3) Prometheus Alerts with runbook links; (4) `curl localhost:8081/healthz` → ok; (5) GameDay: burn rate before/after gameday-off; (6) Prometheus Targets showing Blockscout target UP.

## License

MIT
