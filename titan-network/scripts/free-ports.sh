#!/usr/bin/env bash
set -euo pipefail

(docker ps -q --filter publish=9650; \
 docker ps -q --filter publish=9651; \
 docker ps -q --filter publish=9652; \
 docker ps -q --filter publish=9653; \
 docker ps -q --filter publish=9654; \
 docker ps -q --filter publish=9655) | sort -u | xargs -r docker rm -f
