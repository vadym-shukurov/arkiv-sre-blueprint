# Runbook: IngestionHighErrorRate

## Alert

**Summary:** Arkiv ingestion error rate above 0.1% (SLO breach)  
**SLO:** Ingestion availability >= 99.9%  
**Severity:** warning

## Triage

1. Check if the ingestion pod is running:
   ```bash
   kubectl get pods -n arkiv-ingestion
   ```

2. Check logs:
   ```bash
   kubectl logs -n arkiv-ingestion deployment/arkiv-ingestion --tail=100
   ```

3. Verify Postgres (arkiv-ingestion-db) is running.

## Recovery

1. **Pod failures:** Restart deployment.
   ```bash
   kubectl rollout restart deployment/arkiv-ingestion -n arkiv-ingestion
   ```

2. **Postgres down:** Check arkiv-ingestion-db pod. `kubectl get pods -n arkiv-ingestion`
