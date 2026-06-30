LAUNCH_DIR=$(pwd)
printf "\x1b[34mCoston2 Deployment\x1b[0m\n\n"

export WEB3_API=debug
export COMPLETE_GET_VALIDATORS="true"

# NODE 1
printf "Launching Node 1 at 127.0.0.1:9650\n"
nohup ./build/avalanchego \
  --public-ip=127.0.0.1 \
  --http-port=9650 \
  --staking-port=9651 \
  --db-dir=db/node1 \
  --network-id=costwo \
  --index-enabled=true \
  --staking-tls-cert-file=$LAUNCH_DIR/staking/local/staker1.crt \
  --staking-tls-key-file=$LAUNCH_DIR/staking/local/staker1.key \
  --staking-signer-key-file=$LAUNCH_DIR/staking/local/signer1.key \
  --bootstrap-ips="$(curl -m 10 -sX POST --data '{ "jsonrpc":"2.0", "id":1, "method":"info.getNodeIP" }' -H 'content-type:application/json;' https://coston2-bootstrap.flare.network/ext/info | jq -r ".result.ip")" \
  --bootstrap-ids="$(curl -m 10 -sX POST --data '{ "jsonrpc":"2.0", "id":1, "method":"info.getNodeID" }' -H 'content-type:application/json;' https://coston2-bootstrap.flare.network/ext/info | jq -r ".result.nodeID")" \
  --log-level=debug >$LAUNCH_DIR/logs/costwo/node1/launch.log 2>&1 &
NODE_1_PID=$(echo $!)
sleep 3

printf "\n"
read -p "Press enter to stop background node processes"
kill $NODE_1_PID
