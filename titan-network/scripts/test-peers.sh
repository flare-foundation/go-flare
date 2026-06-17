#!/usr/bin/env bash
set -euo pipefail

PAYLOAD='{"jsonrpc":"2.0","id":1,"method":"info.peers"}'

echo node1
curl -s -X POST -H 'content-type: application/json' --data "$PAYLOAD" http://127.0.0.1:9650/ext/info
echo
echo node2
curl -s -X POST -H 'content-type: application/json' --data "$PAYLOAD" http://127.0.0.1:9652/ext/info
echo
echo node3
curl -s -X POST -H 'content-type: application/json' --data "$PAYLOAD" http://127.0.0.1:9654/ext/info
echo
