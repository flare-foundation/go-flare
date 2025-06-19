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

if [ "$AUTOCONFIGURE_BOOTSTRAP" = "1" ];
then


    __BOOTSTRAP_ENDPOINTS=("${AUTOCONFIGURE_BOOTSTRAP_ENDPOINT}" ${AUTOCONFIGURE_FALLBACK_ENDPOINTS//,/ })

	echo "Trying provided bootstrap endpoints"
	for __ENDPOINT in "${__BOOTSTRAP_ENDPOINTS[@]}"; do
        echo "  Trying endpoint $__ENDPOINT"

        RESPONSE_CODE=$(curl -X POST -m 5 -s -o /dev/null -w '%{http_code}' "$__ENDPOINT" -H 'Content-Type: application/json' --data '{ "jsonrpc":"2.0", "id":1, "method":"info.getNodeIP" }' || true)
        if [ "$RESPONSE_CODE" = "200" ]; then
            __BOOTSTRAP_ENDPOINT="$__ENDPOINT"
            break
        else
			echo "    Failed! The endpoint is unreachable."
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
	"$EXTRA_ARGUMENTS"