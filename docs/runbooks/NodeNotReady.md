# Runbook: NodeNotReady

## Alert

**Summary:** Node {{ $labels.node }} is not Ready  
**Severity:** critical

## Triage

1. Check node status:
   ```bash
   kubectl get nodes
   kubectl describe node <node-name>
   ```

2. Check node conditions:
   ```bash
   kubectl get nodes -o jsonpath='{range .items[*]}{.metadata.name}{"\t"}{.status.conditions[?(@.type=="Ready")].status}{"\n"}{end}'
   ```

3. Check kubelet on the node (if accessible):
   ```bash
   # SSH to node or use kind/ssh
   systemctl status kubelet
   journalctl -u kubelet -n 50
   ```

## Recovery

1. **Resource pressure:** Check DiskPressure, MemoryPressure.
   ```bash
   kubectl describe node <node-name> | grep -A5 Conditions
   ```

2. **Cordon and drain if unrecoverable:**
   ```bash
   kubectl cordon <node-name>
   kubectl drain <node-name> --ignore-daemonsets --delete-emptydir-data
   ```

3. **For kind:** Restart the kind node or recreate the cluster.
   ```bash
   kind delete cluster --name arkiv-platform
   make up
   ```

## Prevention

- Set resource requests/limits to avoid node OOM.
- Monitor node disk space.
