# GameDay

Run the chaos scenario to validate the full on-call loop. Prerequisites: `make up`, `make faucet-build`, `make ingestion-build`.

**GitOps-native:** `make gameday-on` switches the faucet app to the overlay path; `make gameday-off` reverts.

## Full on-call loop

| Step | Action |
|------|--------|
| 1. Inject | `make gameday-on` |
| 2. Observe | Grafana SLO Dashboard — watch error budget, burn rate |
| 3. Alert | Prometheus/Alertmanager fires (FaucetSLOBurnRateFast) |
| 4. Runbook | Follow runbook from alert `runbook_url` |
| 5. Recover | `make gameday-off` |
| 6. Postmortem | Fill [templates/postmortem.md](templates/postmortem.md) and save to `postmortems/` |

## Quick start

```bash
kubectl port-forward -n monitoring svc/observability-grafana 3000:80
make gameday-on
# Wait 2–5 min for FaucetSLOBurnRateFast. Follow runbook. Then:
make gameday-off
```

**Expected:** Alert fires within 2–5 min. After `make gameday-off`: burn rate drops, alert resolves within 5–10 min.

## Scenario

| Scenario | Alert | Inject | Restore |
|---------|-------|--------|---------|
| [SLO burn-rate (faucet)](scenarios/01-faucet-error-spike.md) | FaucetSLOBurnRateFast | `make gameday-on` | `make gameday-off` |

## Postmortem

After the scenario: [templates/postmortem.md](templates/postmortem.md)  
Sample: [postmortems/faucet-error-spike-sample.md](postmortems/faucet-error-spike-sample.md)
