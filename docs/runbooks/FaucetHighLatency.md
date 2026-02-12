# Runbook: FaucetHighLatency

## Alert

**Summary:** Faucet p95 latency above 2s  
**SLO:** p95 latency < 2s  
**Severity:** warning

## Triage

1. Check current latency from metrics:
   ```bash
   kubectl port-forward -n monitoring svc/observability-kube-prometheus-prometheus 9090:9090
   # PromQL: histogram_quantile(0.95, sum(rate(http_request_duration_seconds_bucket{namespace="faucet"}[5m])) by (le))
   ```

2. Check pod resource usage:
   ```bash
   kubectl top pods -n faucet
   ```
   High CPU/memory can cause latency.

3. Check downstream dependencies (RPC, DB) for slowness.

## Recovery

1. **CPU/memory throttling:** Increase resources or add replicas.
   ```bash
   kubectl scale deployment/faucet -n faucet --replicas=2
   ```

2. **Downstream slowness:** Identify and fix the bottleneck (RPC node, DB).

3. **Connection pool exhaustion:** Restart to clear pooled connections:
   ```bash
   kubectl rollout restart deployment/faucet -n faucet
   ```

## Prevention

- Ensure faucet exposes `http_request_duration_seconds` histogram.
- Add caching for read-heavy paths.
