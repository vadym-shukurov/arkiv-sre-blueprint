# Incident Response

1. Acknowledge alert → open runbook from `runbook_url`.
2. Triage (P1/P2/P3), investigate, mitigate, resolve.
3. Postmortem: [gameday/templates/postmortem.md](../gameday/templates/postmortem.md)

## Runbooks

| Alert | Runbook |
|-------|---------|
| FaucetSLOBurnRateFast | [FaucetSLOBurnRateFast.md](runbooks/FaucetSLOBurnRateFast.md) |
| FaucetSLOBurnRateSlow | [FaucetSLOBurnRateSlow.md](runbooks/FaucetSLOBurnRateSlow.md) |
| FaucetHighLatency | [FaucetHighLatency.md](runbooks/FaucetHighLatency.md) |
| FaucetRateLimitSpike | [FaucetRateLimitSpike.md](runbooks/FaucetRateLimitSpike.md) |
| IngestionHighErrorRate | [IngestionHighErrorRate.md](runbooks/IngestionHighErrorRate.md) |
| BlockscoutSLOBurnRateFast | [BlockscoutHighErrorRate.md](runbooks/BlockscoutHighErrorRate.md) |
| BlockscoutSLOBurnRateSlow | [BlockscoutHighErrorRate.md](runbooks/BlockscoutHighErrorRate.md) |
| BlockscoutHighErrorRate | [BlockscoutHighErrorRate.md](runbooks/BlockscoutHighErrorRate.md) |
| NodeNotReady | [NodeNotReady.md](runbooks/NodeNotReady.md) |
| HighPodRestartRate | [HighPodRestartRate.md](runbooks/HighPodRestartRate.md) |
| IngestionErrorSpike | [IngestionHighErrorRate.md](runbooks/IngestionHighErrorRate.md) |
| PrometheusDown | [PrometheusDown.md](runbooks/PrometheusDown.md) |
