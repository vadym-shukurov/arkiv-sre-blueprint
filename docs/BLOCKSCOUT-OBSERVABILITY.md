# Blockscout Observability — Verification

Prerequisites: `make up`, `make faucet-build`, `make ingestion-build`, Blockscout Healthy in Argo CD.

## Verification steps

```bash
# 1. ServiceMonitor exists
kubectl get servicemonitor -n blockscout

# 2. Port-forward Prometheus, then open http://localhost:9090/targets — filter "blockscout", expect UP
kubectl port-forward -n monitoring svc/observability-kube-prometheus-prometheus 9090:9090

# 3. Metrics and SLOs (run in another terminal while port-forward active)
curl -s 'http://localhost:9090/api/v1/query?query=http_requests_total{namespace="blockscout"}' | jq '.data.result | length'  # > 0
curl -s 'http://localhost:9090/api/v1/query?query=slo:blockscout:error_budget_remaining' | jq '.data.result[0].value[1]'  # numeric
```

## Dashboard and alerts

- **Grafana:** `kubectl port-forward -n monitoring svc/observability-grafana 3000:80` → http://localhost:3000 → SLO Dashboard
- **Alerts:** Prometheus http://localhost:9090/alerts → BlockscoutSLOBurnRateFast, BlockscoutSLOBurnRateSlow, BlockscoutHighErrorRate
- **Runbook:** [runbooks/BlockscoutHighErrorRate.md](runbooks/BlockscoutHighErrorRate.md)
