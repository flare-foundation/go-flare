#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
ROOT_WIN="$(cygpath -w "$ROOT")"
TITAN_MOUNT="$ROOT_WIN\\titan-network:/app/titan"
DB_MOUNT="$ROOT_WIN\\db-node1:/app/db"
STAKING_MOUNT="$ROOT_WIN\\staking-node1:/root/.avalanchego/staking"

bash "$ROOT/titan-network/scripts/create-network.sh"
docker rm -f titan-node1 >/dev/null 2>&1 || true

MSYS_NO_PATHCONV=1 MSYS2_ARG_CONV_EXCL='*' docker run -d \
  --name titan-node1 \
  --network titan-net \
  -p 9650:9650 \
  -p 9651:9651 \
  -v "$TITAN_MOUNT" \
  -v "$DB_MOUNT" \
  -v "$STAKING_MOUNT" \
  -e AUTOCONFIGURE_PUBLIC_IP=0 \
  -e AUTOCONFIGURE_BOOTSTRAP=0 \
  -e DB_DIR=/app/db \
  -e EXTRA_ARGUMENTS="--genesis=/app/titan/origin.json --network-id=781337 --http-host=0.0.0.0 --http-port=9650 --staking-port=9651 --db-dir=/app/db --log-level=info" \
  flarefoundation/go-flare:latest
