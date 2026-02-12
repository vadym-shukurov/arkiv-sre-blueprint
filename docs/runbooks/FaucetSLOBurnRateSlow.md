# Runbook: FaucetSLOBurnRateSlow

## Alert

**Summary:** Faucet error budget burning slowly (6x in 1h) — **ticket**  
**SLO:** Faucet availability >= 99.9% (0.1% error budget)  
**Severity:** warning  
**Meaning:** Error rate would exhaust 30-day budget in ~5 hours. Non-urgent but needs investigation.

## What good looks like

- Burn rate < 6 (1h window)
- Error budget remaining > 99.9%
- `slo:faucet:burn_rate_1h` < 6
- Faucet returning 2xx for normal requests

## Triage

1. Check error ratio over last hour:
   ```bash
   kubectl port-forward -n monitoring svc/observability-kube-prometheus-prometheus 9090:9090
   # Query: slo:faucet:error_ratio_1h
   # Query: sum(rate(http_requests_total{namespace="faucet",status=~"5.."}[1h])) / sum(rate(http_requests_total{namespace="faucet"}[1h]))
   ```

2. Check logs for intermittent errors:
   ```bash
   kubectl logs -n faucet deployment/faucet --tail=500 --since=1h | grep -E "5[0-9]{2}|error|Error"
   ```

3. Check if issue is transient or sustained:
   ```bash
   # Grafana SLO Dashboard → Faucet burn rate (1h line)
   # If burn rate is trending down, may resolve without action
   ```

## Recovery

1. **If errors are intermittent:** May self-resolve. Add a ticket to investigate root cause.

2. **If sustained:** Same steps as [FaucetSLOBurnRateFast](FaucetSLOBurnRateFast.md):
   - Check if gameday overlay active (`make gameday-off`), restart deployment, scale, fix upstream.

3. **If approaching fast burn:** Escalate — if 1h burn rate exceeds 6 and 5m burn rate approaches 14.4, treat as page.

## Verification

```bash
# Burn rate should drop below 6 within 1–2 hours after fix
curl -s "http://localhost:9090/api/v1/query?query=slo:faucet:burn_rate_1h" | jq .
```

## Prevention

- Track slow burn tickets to identify recurring patterns.
- Consider decreasing alert threshold if slow burns are frequent.
