# Runbook: FaucetSLOBurnRateFast

## Alert

**Summary:** Faucet error budget burning fast (14.4x in 5m) — **page**  
**SLO:** Faucet availability >= 99.9% (0.1% error budget)  
**Severity:** page  
**Meaning:** Error rate is high enough to exhaust 30-day budget in ~1 hour.

## What good looks like

- Burn rate < 14.4 (5m window)
- Error budget remaining > 99.9%
- `slo:faucet:burn_rate_5m` < 14.4
- Faucet returning 2xx for normal requests

## Triage

1. Check pod status:
   ```bash
   kubectl get pods -n faucet
   kubectl get events -n faucet --sort-by='.lastTimestamp'
   ```
   Expected: Pods `Running`, no recent OOMKilled or CrashLoopBackOff.

2. Check logs for 5xx:
   ```bash
   kubectl logs -n faucet deployment/faucet --tail=200 | grep -E "5[0-9]{2}|error|Error"
   ```

3. Check burn rate in Prometheus:
   ```bash
   kubectl port-forward -n monitoring svc/observability-kube-prometheus-prometheus 9090:9090
   # Query: slo:faucet:burn_rate_5m
   # Query: slo:faucet:error_ratio_5m
   ```

4. Check if GameDay overlay is active:
   ```bash
   kubectl get application faucet -n argocd -o jsonpath='{.spec.source.path}'
   # If gameday/overlays/01-faucet-error-spike: make gameday-off
   ```

## Recovery

1. **If GameDay overlay is active:**
   ```bash
   make gameday-off
   ```

2. **If pod is failing:** Restart deployment.
   ```bash
   kubectl rollout restart deployment/faucet -n faucet
   kubectl rollout status deployment/faucet -n faucet
   ```

3. **If resource exhaustion:** Scale or increase resources.
   ```bash
   kubectl top pods -n faucet
   kubectl scale deployment/faucet -n faucet --replicas=2
   ```

4. **If downstream (RPC/DB) failure:** Fix upstream and restart.
   ```bash
   kubectl rollout restart deployment/faucet -n faucet
   ```

## Verification

After mitigation, confirm:
```bash
# Burn rate should drop below 14.4 within 5–10 minutes
curl -s http://localhost:9090/api/v1/query?query=slo:faucet:burn_rate_5m | jq .
# Or: Grafana SLO Dashboard → Faucet burn rate panel
```

## Prevention

- Ensure faucet exposes `http_requests_total` with `status` label (5xx for errors).
- Monitor SLO dashboard for early burn rate trends.
