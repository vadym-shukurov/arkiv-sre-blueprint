# Postmortem: Faucet Error Spike (GameDay Sample)

## Summary

Faucet returned 500 on ~20% of requests due to FORCE_ERROR_RATE=0.2 (gameday overlay).

## Impact

- **Duration:** ~15 min (inject to restore)
- **Users affected:** All faucet users
- **Services affected:** faucet

## Timeline

| Time (UTC) | Event |
|------------|-------|
| T+0 | GameDay: `make gameday-on` — overlay applied, loadgen pod started |
| T+2 | Load generator hits /faucet; 5xx rate rises |
| T+7 | FaucetSLOBurnRateFast alert fires |
| T+10 | On-call observes SLO dashboard, follows runbook |
| T+12 | Runbook: `make gameday-off` — overlay reverted, loadgen removed |
| T+15 | Alert resolved; burn rate drops |

## Root Cause

Gameday overlay set FORCE_ERROR_RATE=0.2 for chaos injection.

## Resolution

`make gameday-off`

## Action Items

| # | Action | Owner |
|---|--------|-------|
| 1 | Add gameday scenario docs to README | |
| 2 | Consider shorter alert `for` for staging | |

## Lessons Learned

- GameDay validated alert and runbook flow.
- Dashboard panels matched runbook queries.
