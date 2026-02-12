# Blockscout Observability — Verification Checklist

## Prerequisites

- `make up`, `make faucet-build`, `make ingestion-build`
- Blockscout app deployed and Healthy in Argo CD

## 1. ServiceMonitor exists

```bash
kubectl get servicemonitor -n blockscout
# Expected: blockscout-blockscout-stack-blockscout-svm (or similar)
```

## 2. Prometheus targets show UP

```bash
kubectl port-forward -n monitoring svc/observability-kube-prometheus-prometheus 9090:9090
```

Then in browser: http://localhost:9090/targets

- Filter by `blockscout` or `namespace="blockscout"`
- Expected: Targets for blockscout backend `/metrics` show **UP**

## 3. Metrics are scraped

```bash
# Query Prometheus for http_requests_total from blockscout
curl -s 'http://localhost:9090/api/v1/query?query=http_requests_total{namespace="blockscout"}' | jq '.data.result | length'
# Expected: > 0 (at least one time series)
```

## 4. Recording rules evaluate

```bash
curl -s 'http://localhost:9090/api/v1/query?query=slo:blockscout:error_budget_remaining' | jq '.data.result[0].value[1]'
# Expected: numeric value (e.g. "1" for 100% when no errors)
```

## 5. Grafana SLO dashboard

```bash
kubectl port-forward -n monitoring svc/observability-grafana 3000:80
```

- Open http://localhost:3000
- Navigate to **SLO Dashboard**
- Expected: **Blockscout availability**, **Blockscout error budget remaining**, **Blockscout burn rate** panels show data (not "No data")

## 6. Alerts configured

```bash
curl -s 'http://localhost:9090/api/v1/rules' | jq '.data.groups[].rules[] | select(.name | startswith("Blockscout")) | .name'
# Expected: BlockscoutSLOBurnRateFast, BlockscoutSLOBurnRateSlow, BlockscoutHighErrorRate
```

## Quick pass/fail

| Check | Command | Pass criteria |
|-------|---------|---------------|
| ServiceMonitor | `kubectl get servicemonitor -n blockscout` | At least one SM |
| Targets UP | Prometheus UI → Targets | blockscout job UP |
| http_requests_total | `curl .../query?query=http_requests_total{namespace="blockscout"}` | Result count > 0 |
| slo:blockscout:* | `curl .../query?query=slo:blockscout:error_budget_remaining` | Has value |
| Grafana panels | SLO Dashboard | No "No data" on Blockscout panels |
