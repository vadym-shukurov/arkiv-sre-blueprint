# Runbook: HighPodRestartRate

## Alert

**Summary:** Pod {{ $labels.pod }} in {{ $labels.namespace }} restarting frequently  
**Severity:** warning

## Triage

1. Check pod status and restart count:
   ```bash
   kubectl get pods -n <namespace>
   kubectl describe pod <pod-name> -n <namespace>
   ```

2. Check previous container logs (before restart):
   ```bash
   kubectl logs <pod-name> -n <namespace> --previous --tail=100
   ```

3. Check events:
   ```bash
   kubectl get events -n <namespace> --sort-by='.lastTimestamp'
   ```

## Recovery

1. **OOMKilled:** Increase memory limits.
   ```bash
   kubectl get deployment <name> -n <namespace> -o yaml
   # Edit resources.limits.memory
   ```

2. **CrashLoopBackOff:** Fix the application bug or config. Check logs for root cause.

3. **Liveness probe failure:** Adjust probe thresholds or fix the app:
   ```bash
   kubectl get pod <pod-name> -n <namespace> -o yaml | grep -A20 livenessProbe
   ```

## Prevention

- Set appropriate resource limits.
- Ensure liveness/readiness probes match app behavior.
