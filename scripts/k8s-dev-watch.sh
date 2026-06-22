#!/bin/sh
# SPDX-License-Identifier: MIT
set -eu

ROOT_DIR=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)
cd "$ROOT_DIR"

hash_sources() {
  find backend/cmd backend/internal backend/migrations deploy/k8s/orbstack backend/Dockerfile backend/go.mod backend/go.sum -type f \
    ! -path 'backend/internal/db/generated/*' \
    -exec cksum {} \; | sort | cksum
}

rebuild_and_restart() {
  echo "[idbridge] rebuilding open-idb:dev"
  docker build -t open-idb:dev backend/
  echo "[idbridge] restarting deployment/idbridge"
  kubectl -n open-idb rollout restart deployment/idbridge
  kubectl -n open-idb rollout status deployment/idbridge --timeout=120s
}

echo "[idbridge] watching backend/cmd/, backend/internal/, backend/migrations/, deploy/k8s/orbstack/, backend/Dockerfile, backend/go.mod, backend/go.sum"
echo "[idbridge] press Ctrl-C to stop"

last_hash=$(hash_sources)

while :; do
  sleep "${IDB_WATCH_INTERVAL_SECONDS:-2}"
  next_hash=$(hash_sources)
  if [ "$next_hash" != "$last_hash" ]; then
    last_hash=$next_hash
    if rebuild_and_restart; then
      echo "[idbridge] refreshed at $(date '+%H:%M:%S')"
    else
      echo "[idbridge] refresh failed at $(date '+%H:%M:%S')" >&2
    fi
  fi
done
