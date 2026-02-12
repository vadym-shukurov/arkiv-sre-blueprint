# Scenario 1: SLO Burn-Rate Alert (Faucet)

Demonstrates the full on-call loop: inject → observe → alert → runbook → recover → postmortem. GitOps-native; no Argo sync pause required.

## On-call loop

| Step | Action |
|------|--------|
| 1. Inject | `make gameday-on` — faucet app switches to overlay with FORCE_ERROR_RATE=0.2, loadgen pod deploys |
| 2. Observe | Grafana SLO Dashboard → Faucet error budget remaining, Faucet burn rate |
| 3. Alert | FaucetSLOBurnRateFast fires (burn rate > 14.4 for 2m) |
| 4. Runbook | [FaucetSLOBurnRateFast](../../docs/runbooks/FaucetSLOBurnRateFast.md) — triage, recovery commands |
| 5. Recover | `make gameday-off` |
| 6. Postmortem | [templates/postmortem.md](../templates/postmortem.md) |

## Run

```bash
kubectl port-forward -n monitoring svc/observability-grafana 3000:80
make gameday-on
# Wait 2–5 min. Watch Grafana SLO Dashboard. When alert fires, follow runbook.
make gameday-off
```

## Prerequisites

- `make up`, `make faucet-build`, `make ingestion-build`
- Port-forward: Grafana 3000 (optional: Prometheus 9090 for alerts UI)

## Expected alert firing

- **FaucetSLOBurnRateFast** — burn rate > 14.4 (5m), page  
- Runbook: [FaucetSLOBurnRateFast](../../docs/runbooks/FaucetSLOBurnRateFast.md)

## Recovery verification

After `make gameday-off`:

1. **Grafana SLO Dashboard:** Burn rate drops below 14.4 within 5–10 min; error budget remaining returns toward 100%.
2. **Prometheus:** Alert clears (Firing → Pending → Resolved).
3. **Pods:** `kubectl get pods -n faucet` — loadgen-faucet removed (overlay no longer applied).
4. **Faucet health:** `curl -s http://localhost:8081/healthz` → `ok`.
