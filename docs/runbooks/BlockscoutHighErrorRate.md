# Runbook: Blockscout SLO Alerts

## Alerts

| Alert | Summary | Severity |
|-------|---------|----------|
| BlockscoutSLOBurnRateFast | Error budget burning fast (14.4x in 5m) | page |
| BlockscoutSLOBurnRateSlow | Error budget burning slowly (6x in 1h) | warning |
| BlockscoutHighErrorRate | API error rate above 0.1% (SLO 99.9% breach) | warning |

**SLO:** Blockscout HTTP availability ≥ 99.9% (30d)  
**Error budget:** 0.1% (43 min/month)

## What good looks like

- Burn rate < 14.4 (5m window)
- Error budget remaining > 99.9%
- `slo:blockscout:burn_rate_5m` < 14.4
- Blockscout returning 2xx for normal API requests

## Triage

1. Check Blockscout pods:
   ```bash
   kubectl get pods -n blockscout
   kubectl get events -n blockscout --sort-by='.lastTimestamp'
   ```
   Expected: Pods `Running`, no recent OOMKilled or CrashLoopBackOff.

2. Check logs (deployment names vary by chart; use label selector):
   ```bash
   kubectl logs -n blockscout -l app.kubernetes.io/name=blockscout-stack --tail=100
   # Or: kubectl logs -n blockscout deployment/blockscout-blockscout-stack-blockscout --tail=100
   ```

3. Check burn rate in Prometheus:
   ```bash
   kubectl port-forward -n monitoring svc/observability-kube-prometheus-prometheus 9090:9090
   # Query: slo:blockscout:burn_rate_5m
   # Query: slo:blockscout:error_ratio_5m
   ```

4. Verify metrics are being scraped:
   ```bash
   # In Prometheus UI: Status → Targets. Look for blockscout job.
   # Query: up{namespace="blockscout"}
   ```

## Recovery

1. **Pod failures:** Restart Blockscout deployment.
   ```bash
   kubectl rollout restart deployment/blockscout-blockscout-stack-blockscout -n blockscout
   kubectl rollout status deployment/blockscout-blockscout-stack-blockscout -n blockscout
   ```

2. **DB connection issues:** Restart to reset connections.
   ```bash
   kubectl rollout restart deployment/blockscout-blockscout-stack-blockscout -n blockscout
   ```

3. **Resource exhaustion:** Scale or increase resources.
   ```bash
   kubectl top pods -n blockscout
   kubectl scale deployment/blockscout-blockscout-stack-blockscout -n blockscout --replicas=2
   ```

4. **Indexer lag:** Blockscout indexer may be behind. Check indexing status.
   ```bash
   kubectl logs -n blockscout -l app.kubernetes.io/name=blockscout-stack | grep -i index
   ```

## Verification

After mitigation, confirm:
```bash
# Burn rate should drop below 14.4 within 5–10 minutes
curl -s "http://localhost:9090/api/v1/query?query=slo:blockscout:burn_rate_5m" | jq .
# Or: Grafana SLO Dashboard → Blockscout burn rate panel
```

## Prevention

- Prometheus must scrape Blockscout `/metrics` (ServiceMonitor from blockscout-stack with `config.prometheus.enabled: true`).
- Blockscout must expose `http_requests_total` with `status` label (5xx for errors).
