# Runbook: FaucetRateLimitSpike

`kubectl logs -n faucet deploy/faucet | grep "rate limit"`. Adjust `perIPLimit`/`perAddrLimit` in `main.go` if misconfigured.
