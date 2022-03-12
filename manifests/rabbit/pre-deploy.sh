#!/usr/bin/env
set -eu

COOKIE="$(cat /dev/urandom | tr -dc 'a-zA-Z0-9' | fold -w 96 | head -n 1)"

echo "Checking for secret"
OLD_SECRET="$(kubectl get secret rabbit-secret --ignore-not-found -o yaml)"

if [[ -z "$OLD_SECRET" ]]; then
    echo "Creating secret"
    set +x
    kubectl create secret generic rabbit-secret -o yaml --dry-run \
        --from-literal=COOKIE="${COOKIE}" > rabbit-secret.yaml
    set -x
    kubectl apply -f rabbit-secret.yaml
    rm rabbit-secret.yaml || true
fi
