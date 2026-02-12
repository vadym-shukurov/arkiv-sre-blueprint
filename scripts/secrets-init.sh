#!/usr/bin/env bash
# Generate age key, encrypt secrets. Run once before make up.
# Prerequisites: brew install sops age
# Edit *.secret.yaml.example (replace CHANGE_ME) before running.
set -e

cd "$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
DEV="infra/k8s/secrets/dev"
KEY="$DEV/age.agekey"

command -v sops >/dev/null || { echo "brew install sops"; exit 1; }
command -v age-keygen >/dev/null || { echo "brew install age"; exit 1; }

# Create age key if missing; update .sops.yaml with public key.
[[ -f "$KEY" ]] || { age-keygen -o "$KEY"; chmod 600 "$KEY"; }
PUBLIC=$(grep "public key:" "$KEY" | cut -d: -f2 | tr -d ' ')
sed "s|age: >-.*|age: >-$PUBLIC|" infra/k8s/secrets/.sops.yaml > infra/k8s/secrets/.sops.yaml.tmp
mv infra/k8s/secrets/.sops.yaml.tmp infra/k8s/secrets/.sops.yaml

cd "$DEV"
for f in grafana-admin blockscout-postgres blockscout-app arkiv-ingestion-db; do
  sops -e "$f.secret.yaml.example" > "$f.enc.yaml"
done

echo "Done. export SOPS_AGE_KEY=\$(cat $KEY) && make up"
