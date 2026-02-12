# Secrets (SOPS + age)

Template files use placeholders (`CHANGE_ME`, `REDACTED`). Do not commit real secrets.

## Main path: K8s (`make up`)

1. Edit `infra/k8s/secrets/dev/*.secret.yaml.example` and replace each `CHANGE_ME` with real values (or `admin`/`postgres` for local dev).
2. Run:

```bash
make secrets-init
export SOPS_AGE_KEY=$(cat infra/k8s/secrets/dev/age.agekey)
make up
```

Key management: `make secrets-init` encrypts templates → `*.enc.yaml`. `age.agekey` is gitignored. To rotate: edit `.secret.yaml.example`, re-run `make secrets-init`, commit `*.enc.yaml`.

## Other paths

- **Partner-pilot:** `cd partner-pilot && cp .env.example .env` — set `POSTGRES_PASSWORD`, then `docker compose up -d`.
- **Arkiv-ingestion standalone:** Set `DATABASE_URL` env var.
