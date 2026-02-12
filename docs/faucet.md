# Faucet

`make faucet-build` then port-forward to 8081:

```bash
kubectl port-forward -n faucet svc/faucet 8081:80
curl -X POST http://localhost:8081/faucet -H "Content-Type: application/json" -d '{"address":"0x1234"}'
```

Rate limits: 10/min per IP, 2/hour per address.
