#!/bin/bash

set -eo pipefail

GENESIS_FILE="${GENESIS_FILE:-/app/titan/origin.json}"

if [ -n "$ORIGIN_URL" ];
then
	mkdir -p "$(dirname "$GENESIS_FILE")"
	echo "Fetching origin from '${ORIGIN_URL}'"
	curl -fsSL "$ORIGIN_URL" -o "$GENESIS_FILE"
	echo "  Saved origin to '${GENESIS_FILE}'"
fi

if echo "$EXTRA_ARGUMENTS" | grep -q -- '--genesis=';
then
	__GENESIS_PATH=$(echo "$EXTRA_ARGUMENTS" | sed -n 's/.*--genesis=\([^[:space:]]*\).*/\1/p')
	__GENESIS_PATH="${__GENESIS_PATH:-$GENESIS_FILE}"
	if [ ! -f "$__GENESIS_PATH" ];
	then
		echo "Genesis file '${__GENESIS_PATH}' not found."
		echo "Set ORIGIN_URL, mount titan-network/origin.json, or provide a valid --genesis path."
		exit 1
	fi
fi

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

if [ "$AUTOCONFIGURE_BOOTSTRAP" = "1" ];
then
	__BOOTSTRAP_ENDPOINTS=("${AUTOCONFIGURE_BOOTSTRAP_ENDPOINT}" ${AUTOCONFIGURE_FALLBACK_ENDPOINTS//,/ })

	echo "Trying provided bootstrap endpoints"
	for __ENDPOINT in "${__BOOTSTRAP_ENDPOINTS[@]}"; do
		echo "  Trying endpoint $__ENDPOINT"

		TEMP_RESPONSE=$(mktemp)
		RESPONSE_CODE=$(curl -X POST -m 5 -s -w '%{http_code}' -o "$TEMP_RESPONSE" "$__ENDPOINT" -H 'Content-Type: application/json' --data '{ "jsonrpc":"2.0", "id":1, "method":"info.getNodeIP" }' 2>/dev/null || echo "NETWORK_ERROR")

		if [ "$RESPONSE_CODE" = "200" ]; then
			__BOOTSTRAP_ENDPOINT="$__ENDPOINT"
			rm -f "$TEMP_RESPONSE"
			break
		elif [ "$RESPONSE_CODE" = "NETWORK_ERROR" ]; then
			echo "    Failed! Network error (connection timeout, DNS failure, or unreachable host)"
			rm -f "$TEMP_RESPONSE"
			continue
		else
			RESPONSE_BODY=$(head -c 200 "$TEMP_RESPONSE" 2>/dev/null || echo "")
			if [ -n "$RESPONSE_BODY" ]; then
				echo "    Failed! HTTP $RESPONSE_CODE, response body: $RESPONSE_BODY"
			else
				echo "    Failed! HTTP $RESPONSE_CODE"
			fi
			rm -f "$TEMP_RESPONSE"
			continue
		fi
	done

	if [ -z "$__BOOTSTRAP_ENDPOINT" ]; then
		echo "  None of provided bootstrap endpoints worked!"
		exit 1
	fi

	echo "Autoconfiguring bootstrap IPs and IDs"

	BOOTSTRAP_IPS=$(curl -m 10 -sX POST --data '{ "jsonrpc":"2.0", "id":1, "method":"info.getNodeIP" }' -H 'content-type:application/json;' "$__BOOTSTRAP_ENDPOINT" | jq -r ".result.ip")
	BOOTSTRAP_IDS=$(curl -m 10 -sX POST --data '{ "jsonrpc":"2.0", "id":1, "method":"info.getNodeID" }' -H 'content-type:application/json;' "$__BOOTSTRAP_ENDPOINT" | jq -r ".result.nodeID")

	echo "  Got bootstrap ips: '${BOOTSTRAP_IPS}'"
	echo "  Got bootstrap ids: '${BOOTSTRAP_IDS}'"
fi

exec /app/build/avalanchego \
	--http-host="$HTTP_HOST" \
	--http-port="$HTTP_PORT" \
	--staking-port="$STAKING_PORT" \
	--public-ip="$PUBLIC_IP" \
	--db-dir="$DB_DIR" \
	--db-type="$DB_TYPE" \
	--bootstrap-ips="$BOOTSTRAP_IPS" \
	--bootstrap-ids="$BOOTSTRAP_IDS" \
	--bootstrap-beacon-connection-timeout="$BOOTSTRAP_BEACON_CONNECTION_TIMEOUT" \
	--chain-config-dir="$CHAIN_CONFIG_DIR" \
	--log-dir="$LOG_DIR" \
	--log-level="$LOG_LEVEL" \
	--network-id="$NETWORK_ID" \
	--http-allowed-hosts="$HTTP_ALLOWED_HOSTS" \
	$EXTRA_ARGUMENTS