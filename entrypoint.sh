#!/bin/bash

set -eo pipefail

if [ "$AUTOCONFIGURE_PUBLIC_IP" = "1" ];
then
	if [ -z "$PUBLIC_IP" ];
	then
		echo "Autoconfiguring public IP"
		PUBLIC_IP=$(curl -s -m 10 https://flare.network/cdn-cgi/trace | grep 'ip=' | cut -d'=' -f2)
		echo "  Got public address '${PUBLIC_IP}'" 
	else
		echo "/!\\ AUTOCONFIGURE_PUBLIC_IP is enabled, but PUBLIC_IP is already set to '$PUBLIC_IP'! Skipping autoconfigure and using current PUBLIC_IP value!"
	fi
fi

# Check if we can connect to the bootstrap endpoint (whitelisting)
BOOTSTRAP_STATUS=$(curl -m 10 -s -w %{http_code} -X POST --data '{ "jsonrpc":"2.0", "id":1, "method":"info.getNodeIP" }' -H 'content-type:application/json;' "$AUTOCONFIGURE_BOOTSTRAP_ENDPOINT" -o /dev/null)
if [ "$BOOTSTRAP_STATUS" != "200" ]; then
	echo "Could not connect to bootstrap endpoint. Is your IP whitelisted?"
	exit 1
fi

if [ "$AUTOCONFIGURE_BOOTSTRAP" = "1" ];
then
	echo "Autoconfiguring bootstrap IPs and IDs"

	BOOTSTRAP_IPS=$(curl -m 10 -sX POST --data '{ "jsonrpc":"2.0", "id":1, "method":"info.getNodeIP" }' -H 'content-type:application/json;' "$AUTOCONFIGURE_BOOTSTRAP_ENDPOINT" | jq -r ".result.ip")
	BOOTSTRAP_IDS=$(curl -m 10 -sX POST --data '{ "jsonrpc":"2.0", "id":1, "method":"info.getNodeID" }' -H 'content-type:application/json;' "$AUTOCONFIGURE_BOOTSTRAP_ENDPOINT" | jq -r ".result.nodeID")
fi

exec /app/build/avalanchego \
	--http-host=$HTTP_HOST \
	--http-port=$HTTP_PORT \
	--staking-port=$STAKING_PORT \
	--public-ip=$PUBLIC_IP \
	--db-dir=$DB_DIR \
	--db-type=$DB_TYPE \
	--bootstrap-ips=$BOOTSTRAP_IPS \
	--bootstrap-ids=$BOOTSTRAP_IDS \
	--bootstrap-beacon-connection-timeout=$BOOTSTRAP_BEACON_CONNECTION_TIMEOUT \
	--chain-config-dir=$CHAIN_CONFIG_DIR \
	--log-dir=$LOG_DIR \
	--log-level=$LOG_LEVEL \
	--network-id=$NETWORK_ID \
	$EXTRA_ARGUMENTS
