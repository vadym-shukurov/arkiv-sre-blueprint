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

**CI:** `make ci-local` (requires yamllint, kubeconform, helm, go, gitleaks). Runs on PR/push; you can also run manually via workflow_dispatch.

## Quickstart

```bash
# 1. Edit infra/k8s/secrets/dev/*.secret.yaml.example — replace CHANGE_ME with real values (use admin/postgres for local dev: docs/security/secrets.md)
# 2. Generate encrypted secrets
make secrets-init
export SOPS_AGE_KEY=$(cat infra/k8s/secrets/dev/age.agekey)
# 3. Commit enc.yaml and push (Argo syncs secrets from repo)
git add infra/k8s/secrets/dev/*.enc.yaml && git commit -m "Add encrypted secrets" || true
git push origin main
# 4. Bring up cluster
make up
make faucet-build
make ingestion-build
make status
make port-forward  # Argo CD only; Grafana/faucet/etc. in separate terminals (see Port-forward)
make down
```

Secrets: [docs/security/secrets.md](docs/security/secrets.md) | RBAC: [docs/security/rbac.md](docs/security/rbac.md)

**Security:** Workloads do not run under default SA; token automount disabled by default.

**Argo CD password:**
```bash
kubectl -n argocd get secret argocd-initial-admin-secret -o jsonpath="{.data.password}" | base64 -d
```

## Port-forward

`make port-forward` starts Argo CD only. For Grafana, faucet, etc., run these in separate terminals:

```bash
kubectl port-forward -n monitoring svc/observability-grafana 3000:80
kubectl port-forward -n blockscout svc/blockscout-blockscout-stack-frontend-svc 8080:80
kubectl port-forward -n faucet svc/faucet 8081:80
kubectl port-forward -n arkiv-ingestion svc/arkiv-ingestion 8082:80
```

## Faucet

`make faucet-build` → [docs/faucet.md](docs/faucet.md)

## Arkiv Ingestion

`make ingestion-build` → [partner-pilot/README.md](partner-pilot/README.md)

## Run a GameDay

Full on-call loop: inject → observe dashboard → alert fires → runbook → recover → postmortem. GitOps-native; no Argo sync pause required.

```bash
make up && make faucet-build && make ingestion-build
kubectl port-forward -n monitoring svc/observability-grafana 3000:80
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

**Prerequisites for `make ci-local`:** `brew install yamllint kubeconform gitleaks helm` (in addition to kind, kubectl, make, sops, age).

## Security checks

Secret scanning runs in CI and locally via [gitleaks](https://github.com/gitleaks/gitleaks). Fail the build if secrets are detected.

```bash
make secrets-scan
```

Placeholders (`REDACTED`, `CHANGE_ME`) are allowlisted in `.gitleaks.toml`. See [docs/security/secrets.md](docs/security/secrets.md) for how to set real values.

## Evidence / Production Readiness

Audit: [docs/RELEASE-CANDIDATE-AUDIT.md](docs/RELEASE-CANDIDATE-AUDIT.md) | [docs/PRODUCTION-READINESS-AUDIT.md](docs/PRODUCTION-READINESS-AUDIT.md) | QE: [docs/RELEASE-CANDIDATE-QE.md](docs/RELEASE-CANDIDATE-QE.md)

**Evidence screenshots:** (1) Argo CD UI — all apps Synced/Healthy; (2) Grafana SLO Dashboard — Faucet burn rate panels; (3) Prometheus Alerts with runbook links; (4) `curl localhost:8081/healthz` → ok; (5) GameDay: burn rate before/after gameday-off; (6) Prometheus Targets showing Blockscout target UP.

## License

MIT
