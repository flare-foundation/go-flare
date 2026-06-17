#!/usr/bin/env bash
set -euo pipefail

docker network inspect titan-net >/dev/null 2>&1 || docker network create titan-net
