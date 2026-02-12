# SLIs and SLOs

| SLO | Target |
|-----|--------|
| Faucet availability | ≥ 99.9% |
| Faucet p95 latency | < 2s |
| Ingestion availability | ≥ 99.9% |
| Blockscout availability | ≥ 99.9% |
| Node readiness | 100% |

Alerts → [runbooks/](runbooks/)

## How to test faucet SLO alerts

GameDay scenario 1 uses an in-cluster load generator. No manual traffic needed.

```bash
make gameday-on
# Wait ~2–5 min for FaucetSLOBurnRateFast
make gameday-off
```
