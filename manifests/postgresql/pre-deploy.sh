#!/usr/bin/env
set -eu

PGDATABASE="$(cat /dev/urandom | tr -dc 'a-z' | fold -w 10 | head -n 1)"
PGUSER="$(cat /dev/urandom | tr -dc 'a-z' | fold -w 10 | head -n 1)"
PGPASSWORD="$(cat /dev/urandom | tr -dc 'a-z' | fold -w 10 | head -n 1)"
PGHOST="postgresql.default.svc.cluster.local"

echo "Checking for secret"
OLD_SECRET="$(kubectl get secret pg-config --ignore-not-found -o yaml)"

if [[ -z "$OLD_SECRET" ]]; then
    echo "Creating secret"
    set +x
    kubectl create secret generic pg-config -o yaml --dry-run \
        --from-literal=PGDATABASE="${PGDATABASE}" \
        --from-literal=PGHOST="${PGHOST}" \
        --from-literal=PGUSER="${PGUSER}" \
        --from-literal=PGPASSWORD="${PGPASSWORD}" \
        --from-literal=PGPASS="${PGHOST}:5432:${PGDATABASE}:${PGUSER}:${PGPASSWORD}" \
        --from-literal=PGPASSFILE="/tmp/postgres/.pgpass" > pg-config.yaml
    set -x
    kubectl apply -f pg-config.yaml
    rm pg-config.yaml || true
fi
