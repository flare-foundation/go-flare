#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
ROOT_WIN="$(cygpath -w "$ROOT")"
TITAN_MOUNT="$ROOT_WIN\\titan-network:/app/titan"
DB_MOUNT="$ROOT_WIN\\db-node3:/app/db"
STAKING_MOUNT="$ROOT_WIN\\staking-node3:/root/.avalanchego/staking"

bash "$ROOT/titan-network/scripts/create-network.sh"
docker rm -f titan-node3 >/dev/null 2>&1 || true

MSYS_NO_PATHCONV=1 MSYS2_ARG_CONV_EXCL='*' docker run -d \
  --name titan-node3 \
  --network titan-net \
  -p 9654:9650 \
  -p 9655:9651 \
  -v "$TITAN_MOUNT" \
  -v "$DB_MOUNT" \
  -v "$STAKING_MOUNT" \
  -e AUTOCONFIGURE_PUBLIC_IP=0 \
  -e AUTOCONFIGURE_BOOTSTRAP=1 \
  -e AUTOCONFIGURE_BOOTSTRAP_ENDPOINT=http://titan-node1:9650/ext/info \
  -e DB_DIR=/app/db \
  -e EXTRA_ARGUMENTS="--genesis=/app/titan/origin.json --network-id=781337 --http-host=0.0.0.0 --http-port=9650 --staking-port=9651 --db-dir=/app/db --log-level=warn" \
  flarefoundation/go-flare:latest
