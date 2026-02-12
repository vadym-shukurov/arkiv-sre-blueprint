# Blockscout

`kubectl port-forward -n blockscout svc/blockscout-blockscout-stack-frontend-svc 8080:80` → http://localhost:8080

- **Backup:** `kubectl exec -n blockscout blockscout-postgresql-0 -- pg_dump -U postgres blockscout > backup.sql`
- **Upgrade:** Chart.yaml deps → `helm dependency update` → push.
- **DB size:** `SELECT pg_size_pretty(pg_database_size('blockscout'));`
- **Issues:** CrashLoopBackOff → check logs; RPC unreachable → verify `ETHEREUM_JSONRPC_HTTP_URL`; OOM → increase limits.
