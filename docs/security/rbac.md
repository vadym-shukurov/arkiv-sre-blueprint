# RBAC and Service Accounts

## Principles

1. **Dedicated SA per workload** — Each app runs under its own ServiceAccount, not the namespace default.
2. **Token automount disabled by default** — `automountServiceAccountToken: false` unless the workload needs Kubernetes API access.
3. **RBAC only if necessary** — No Role/RoleBinding unless app code explicitly uses the Kubernetes API.

## Workloads

| Namespace        | ServiceAccount        | automountServiceAccountToken | RBAC |
|------------------|-----------------------|-----------------------------|------|
| faucet           | faucet-sa             | false                       | none |
| faucet           | loadgen-faucet-sa     | false                       | none |
| arkiv-ingestion  | arkiv-ingestion-sa    | false                       | none |
| arkiv-ingestion  | arkiv-ingestion-db-sa | false                       | none |
| blockscout       | blockscout-sa         | false                       | none |
| blockscout       | blockscout-frontend-sa| false                       | none |
| blockscout       | blockscout-db-sa      | false                       | none |

## Why no Role/RoleBinding?

App code (faucet, arkiv-ingestion) does not use the Kubernetes API (no client-go, InClusterConfig, or pod/secret listing). Blockscout and Postgres do not require API access. If a future workload needs it, add:

1. Smallest possible Role (namespaced, specific verbs/resources)
2. RoleBinding to the workload's ServiceAccount
3. Set `automountServiceAccountToken: true` for that workload only
4. Document the reason in this file

## Verification

```bash
# List ServiceAccounts (expect dedicated SAs, not default)
kubectl get sa -A | grep -E "faucet|arkiv-ingestion|blockscout"

# Expected: faucet-sa, loadgen-faucet-sa in faucet; arkiv-ingestion-sa, arkiv-ingestion-db-sa in arkiv-ingestion;
#           blockscout-sa, blockscout-frontend-sa, blockscout-db-sa in blockscout (if chart supports)

# Confirm automountServiceAccountToken on pod (false = no token mount)
kubectl get pod -n faucet -l app=faucet -o jsonpath='{.items[0].spec.automountServiceAccountToken}'
kubectl describe pod -n faucet -l app=faucet | grep -A2 "Service Account"
# Expected: automountServiceAccountToken: false; Service Account: faucet-sa

# Loadgen (gameday overlay only)
kubectl get pod -n faucet loadgen-faucet -o jsonpath='{.spec.serviceAccountName}{"\n"}{.spec.automountServiceAccountToken}'
# Expected: loadgen-faucet-sa, false
```

**Loadgen:** The gameday load generator (`loadgen-faucet` Pod) uses dedicated `loadgen-faucet-sa` with token automount disabled. Deployed only when overlay `01-faucet-error-spike` is applied.

**Note:** Blockscout serviceAccount values depend on blockscout-stack chart support. If a deploy fails, remove blockscout.serviceAccount / frontend.serviceAccount from values and document in this file.
