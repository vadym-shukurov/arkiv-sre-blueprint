# Partner Pilot

Demo: synthetic data → Postgres every 10s.

## Setup

```bash
cp .env.example .env
# Edit .env: replace POSTGRES_PASSWORD=CHANGE_ME with your password (see docs/security/secrets.md)
```

## Run

```bash
docker compose up -d
curl http://localhost:8082/healthz
curl http://localhost:8082/metrics
docker compose exec postgres psql -U postgres -d arkiv -c "SELECT * FROM ingestion_records;"
```

## Env

| Var | Default | Notes |
|-----|---------|-------|
| POSTGRES_PASSWORD | CHANGE_ME | Set in `.env`; required for postgres + arkiv-ingestion |
| DATABASE_URL | derived from POSTGRES_PASSWORD | Override to use external DB |
| CHAIN_ID | 1 | |
| INGEST_INTERVAL_SEC | 30 | |

## K8s

```bash
make ingestion-build
kubectl apply -k apps/arkiv-ingestion/k8s
kubectl port-forward -n arkiv-ingestion svc/arkiv-ingestion 8082:80
```

## Troubleshooting

- **Connection refused:** Start postgres first: `docker compose up postgres -d`
- **No data:** `SELECT * FROM ingestion_records;` — records use idempotency key `{chainID}-{blockNum}`
