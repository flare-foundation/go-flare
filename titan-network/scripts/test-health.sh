#!/usr/bin/env bash
set -euo pipefail

echo node1
curl -s http://127.0.0.1:9650/ext/health
echo
echo node2
curl -s http://127.0.0.1:9652/ext/health
echo
echo node3
curl -s http://127.0.0.1:9654/ext/health
echo
