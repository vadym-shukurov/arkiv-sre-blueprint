# Runbook: PrometheusDown

## Alert

**Summary:** Prometheus is down (observability failure)  
**Severity:** critical

## Triage

1. Check Prometheus pods:
   ```bash
   kubectl get pods -n monitoring -l app.kubernetes.io/name=prometheus
   ```

2. Check Grafana (may also be affected):
   ```bash
   kubectl get pods -n monitoring -l app.kubernetes.io/name=grafana
   ```

## Recovery

1. Scale Prometheus back up:
   ```bash
   kubectl scale statefulset -n monitoring observability-kube-prometheus-prometheus --replicas=1
   ```
