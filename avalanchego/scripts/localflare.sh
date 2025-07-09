LAUNCH_DIR=$(pwd)
printf "\x1b[34mLocalflare 5-Node Deployment\x1b[0m\n\n"

export WEB3_API=debug
export COMPLETE_GET_VALIDATORS="true"

if ! echo $1 | grep -e "--existing" -q
then
	rm -rf $LAUNCH_DIR/logs/local
	mkdir -p $LAUNCH_DIR/logs/local
	rm -rf $LAUNCH_DIR/db/local
	mkdir -p $LAUNCH_DIR/db/local
	mkdir -p $LAUNCH_DIR/logs/local/node1
	mkdir -p $LAUNCH_DIR/logs/local/node2
	mkdir -p $LAUNCH_DIR/logs/local/node3
	mkdir -p $LAUNCH_DIR/logs/local/node4
	mkdir -p $LAUNCH_DIR/logs/local/node5
	mkdir -p $LAUNCH_DIR/db/local/node1
	mkdir -p $LAUNCH_DIR/db/local/node2
	mkdir -p $LAUNCH_DIR/db/local/node3
	mkdir -p $LAUNCH_DIR/db/local/node4
	mkdir -p $LAUNCH_DIR/db/local/node5
fi

# NODE 1
printf "Launching Node 1 at 127.0.0.1:9650\n"
nohup ./build/avalanchego \
--public-ip=127.0.0.1 \
--http-port=9650 \
--staking-port=9651 \
--db-dir=db/node1 \
--network-id=localflare \
--index-enabled=true \
--staking-tls-cert-file=$LAUNCH_DIR/staking/local/staker1.crt \
--staking-tls-key-file=$LAUNCH_DIR/staking/local/staker1.key \
--log-level=debug > $LAUNCH_DIR/logs/local/node1/launch.log 2>&1 &
NODE_1_PID=`echo $!`
sleep 3

# NODE 2
printf "Launching Node 2 at 127.0.0.1:9652\n"
export WEB3_API=disabled
nohup ./build/avalanchego \
--public-ip=127.0.0.1 \
--http-port=9652 \
--staking-port=9653 \
--db-dir=db/node2 \
--network-id=localflare \
--index-enabled=true \
--bootstrap-ips=127.0.0.1:9651 \
--bootstrap-ids=NodeID-7Xhw2mDxuDS44j42TCB6U5579esbSt3Lg \
--staking-tls-cert-file=$LAUNCH_DIR/staking/local/staker2.crt \
--staking-tls-key-file=$LAUNCH_DIR/staking/local/staker2.key \
--log-level=debug > $LAUNCH_DIR/logs/local/node2/launch.log 2>&1 &
NODE_2_PID=`echo $!`
sleep 3


# NODE 3
printf "Launching Node 3 at 127.0.0.1:9654\n"
nohup ./build/avalanchego \
--public-ip=127.0.0.1 \
--http-port=9654 \
--staking-port=9655 \
--db-dir=db/node3 \
--network-id=localflare \
--bootstrap-ips=127.0.0.1:9651 \
--bootstrap-ids=NodeID-7Xhw2mDxuDS44j42TCB6U5579esbSt3Lg \
--staking-tls-cert-file=$LAUNCH_DIR/staking/local/staker3.crt \
--staking-tls-key-file=$LAUNCH_DIR/staking/local/staker3.key \
--log-level=debug > $LAUNCH_DIR/logs/local/node3/launch.log 2>&1 &
NODE_3_PID=`echo $!`
sleep 3


# NODE 4
printf "Launching Node 4 at 127.0.0.1:9656\n"
nohup ./build/avalanchego \
--public-ip=127.0.0.1 \
--http-port=9656 \
--staking-port=9657 \
--db-dir=db/node4 \
--network-id=localflare \
--bootstrap-ips=127.0.0.1:9651 \
--bootstrap-ids=NodeID-7Xhw2mDxuDS44j42TCB6U5579esbSt3Lg \
--staking-tls-cert-file=$LAUNCH_DIR/staking/local/staker4.crt \
--staking-tls-key-file=$LAUNCH_DIR/staking/local/staker4.key \
--log-level=debug > $LAUNCH_DIR/logs/local/node4/launch.log 2>&1 &
NODE_4_PID=`echo $!`
sleep 3


# NODE 5
printf "Launching Node 5 at 127.0.0.1:9658\n"
nohup ./build/avalanchego \
--public-ip=127.0.0.1 \
--http-port=9658 \
--staking-port=9659 \
--db-dir=db/node5 \
--network-id=localflare \
--bootstrap-ips=127.0.0.1:9651 \
--bootstrap-ids=NodeID-7Xhw2mDxuDS44j42TCB6U5579esbSt3Lg \
--staking-tls-cert-file=$LAUNCH_DIR/staking/local/staker5.crt \
--staking-tls-key-file=$LAUNCH_DIR/staking/local/staker5.key \
--log-level=debug > $LAUNCH_DIR/logs/local/node5/launch.log 2>&1 &
NODE_5_PID=`echo $!`
sleep 3

printf "\n"
read -p "Press enter to stop background node processes"
kill $NODE_1_PID
kill $NODE_2_PID
kill $NODE_3_PID
kill $NODE_4_PID
kill $NODE_5_PID

