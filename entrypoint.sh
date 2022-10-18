#!/bin/bash

set -eo pipefail

if [ "$AUTOCONFIGURE_PUBLIC_IP" = "1" ];
then
	if [ "$PUBLIC_IP" = "" ];
	then
		echo "Autoconfiguring public IP"
		PUBLIC_IP=$(curl https://api.ipify.org/)
	else
		echo "/!\\ AUTOCONFIGURE_PUBLIC_IP is enabled, but PUBLIC_IP is already set to '$PUBLIC_IP'! Skipping autoconfigure and using current PUBLIC_IP value!"
	fi
fi

if [ "$AUTOCONFIGURE_BOOTSTRAP" = "1" ];
then
	echo "Autoconfiguring bootstrap IPs and IDs"

	BOOTSTRAP_IPS=$(curl -m 10 -sX POST --data '{ "jsonrpc":"2.0", "id":1, "method":"info.getNodeIP" }' -H 'content-type:application/json;' "$AUTOCONFIGURE_BOOTSTRAP_ENDPOINT" | jq -r ".result.ip")
	BOOTSTRAP_IDS=$(curl -m 10 -sX POST --data '{ "jsonrpc":"2.0", "id":1, "method":"info.getNodeID" }' -H 'content-type:application/json;' "$AUTOCONFIGURE_BOOTSTRAP_ENDPOINT" | jq -r ".result.nodeID")
fi

/app/build/avalanchego \
	--http-host=$HTTP_HOST \
	--http-port=$HTTP_PORT \
	--staking-port=$STAKING_PORT \
	--public-ip=$PUBLIC_IP \
	--db-dir=$DB_DIR \
	--db-type=$DB_TYPE \
	--bootstrap-ips=$BOOTSTRAP_IPS \
	--bootstrap-ids=$BOOTSTRAP_IDS \
	--chain-config-dir=$CHAIN_CONFIG_DIR \
	--log-dir=$LOG_DIR \
	--log-level=$LOG_LEVEL \
	--network-id=$NETWORK_ID \
	$EXTRA_ARGUMENTS
