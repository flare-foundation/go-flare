#!/usr/bin/env bash
set -euo pipefail

docker rm -f titan-node1 titan-node2 titan-node3 >/dev/null 2>&1 || true
