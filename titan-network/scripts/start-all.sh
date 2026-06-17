#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"

bash "$ROOT/titan-network/scripts/free-ports.sh"
bash "$ROOT/titan-network/scripts/start-node1-bg.sh"

# Wait for node1 info endpoint before bootstrapping node2/node3.
ATTEMPTS=90
PAYLOAD='{"jsonrpc":"2.0","id":1,"method":"info.getNodeID"}'
for i in $(seq 1 "$ATTEMPTS"); do
	if curl -s -X POST -H "content-type: application/json" --data "$PAYLOAD" http://127.0.0.1:9650/ext/info >/dev/null; then
		break
	fi

	if [[ "$i" -eq "$ATTEMPTS" ]]; then
		echo "node1 did not become ready in time; tailing logs" >&2
		docker logs titan-node1 --tail 120 || true
		exit 1
	fi

	sleep 1
done

bash "$ROOT/titan-network/scripts/start-node2-bg.sh"
bash "$ROOT/titan-network/scripts/start-node3-bg.sh"

docker ps --format 'table {{.Names}}\t{{.Ports}}\t{{.Status}}' | grep titan-node || true
