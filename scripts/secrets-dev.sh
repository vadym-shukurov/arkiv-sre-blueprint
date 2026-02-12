#!/usr/bin/env bash
# Generate random dev secrets into .secret.yaml (gitignored). Run before make secrets-init.
# Local dev only; .example templates stay pristine.
set -e

cd "$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
DEV="infra/k8s/secrets/dev"

pw_grafana=$(openssl rand -base64 12)
pw_blockscout=$(openssl rand -base64 12)
pw_ingestion=$(openssl rand -base64 12)
secret_blockscout=$(openssl rand -base64 32)

# Copy template to .secret.yaml (gitignored), replace CHANGE_ME. Use | delimiter so / in base64 is safe.
for name in grafana-admin blockscout-postgres blockscout-app arkiv-ingestion-db; do
  f="$DEV/$name.secret.yaml.example"
  out="$DEV/$name.secret.yaml"
  case "$name" in
    grafana-admin)     cp "$f" "$out"; sed -i.bak "s|CHANGE_ME|$pw_grafana|g" "$out" ;;
    blockscout-postgres) cp "$f" "$out"; sed -i.bak "s|CHANGE_ME|$pw_blockscout|g" "$out" ;;
    blockscout-app)    cp "$f" "$out"; sed -i.bak "s|CHANGE_ME|$secret_blockscout|g" "$out" ;;
    arkiv-ingestion-db) cp "$f" "$out"; sed -i.bak "s|CHANGE_ME|$pw_ingestion|g" "$out" ;;
  esac
  rm -f "${out}.bak"
done

echo ">>> Generated .secret.yaml. Run: make secrets-init"
