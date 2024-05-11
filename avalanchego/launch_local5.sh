#!/bin/sh
EXTRA=--min-stake-duration=600s
./build/avalanchego --public-ip=127.0.0.1 --http-port=9650 --staking-port=9651 --db-dir=db/node1 --network-id=localflare --staking-tls-cert-file=$(pwd)/staking/local/staker1.crt --staking-tls-key-file=$(pwd)/staking/local/staker1.key --chain-config-dir=$(pwd)/../config/localflare $EXTRA &
./build/avalanchego --public-ip=127.0.0.1 --http-port=9652 --staking-port=9653 --db-dir=db/node2 --network-id=localflare --bootstrap-ips=127.0.0.1:9651 --bootstrap-ids=NodeID-7Xhw2mDxuDS44j42TCB6U5579esbSt3Lg --staking-tls-cert-file=$(pwd)/staking/local/staker2.crt --staking-tls-key-file=$(pwd)/staking/local/staker2.key $EXTRA & 
./build/avalanchego --public-ip=127.0.0.1 --http-port=9654 --staking-port=9655 --db-dir=db/node3 --network-id=localflare --bootstrap-ips=127.0.0.1:9651 --bootstrap-ids=NodeID-7Xhw2mDxuDS44j42TCB6U5579esbSt3Lg --staking-tls-cert-file=$(pwd)/staking/local/staker3.crt --staking-tls-key-file=$(pwd)/staking/local/staker3.key $EXTRA &
./build/avalanchego --public-ip=127.0.0.1 --http-port=9656 --staking-port=9657 --db-dir=db/node4 --network-id=localflare --bootstrap-ips=127.0.0.1:9651 --bootstrap-ids=NodeID-7Xhw2mDxuDS44j42TCB6U5579esbSt3Lg --staking-tls-cert-file=$(pwd)/staking/local/staker4.crt --staking-tls-key-file=$(pwd)/staking/local/staker4.key $EXTRA &
./build/avalanchego --public-ip=127.0.0.1 --http-port=9658 --staking-port=9659 --db-dir=db/node5 --network-id=localflare --bootstrap-ips=127.0.0.1:9651 --bootstrap-ids=NodeID-7Xhw2mDxuDS44j42TCB6U5579esbSt3Lg --staking-tls-cert-file=$(pwd)/staking/local/staker5.crt --staking-tls-key-file=$(pwd)/staking/local/staker5.key $EXTRA &
curl -s --location --request POST 'http://localhost:9650/ext/bc/P' \
--header 'Content-Type: application/json' \
--data-raw '{
    "jsonrpc": "2.0",
    "method": "platform.getCurrentValidators",
    "id": 1
}' | jq .result


#!/usr/bin/env bash

printf "Creating new user: "
curl -s --location --request POST 'http://localhost:9650/ext/keystore' \
--header 'Content-Type: application/json' \
--data-raw '{
    "jsonrpc":"2.0",
    "id"     :1,
    "method" :"keystore.createUser",
    "params" :{
        "username": "user1234",
        "password": "b39d642078d2ca0517cafe008ddc9326fa1c4d71248078c67bf0d508993720e4"
    }
}' | jq .

sleep 5

printf "\nUpdating user's P-chain public key to P-localflare18jma8ppw3nhx5r4ap8clazz0dps7rv5uj3gy4v: "
curl -s --location --request POST 'http://localhost:9650/ext/bc/P' \
--header 'Content-Type: application/json' \
--data-raw '{
    "jsonrpc":"2.0",
    "id"     :1,
    "method" :"platform.importKey",
    "params" :{
        "username":"user1234",
        "password":"b39d642078d2ca0517cafe008ddc9326fa1c4d71248078c67bf0d508993720e4",
        "privateKey":"PrivateKey-ewoqjP7PxY4yr3iLTpLisriqt94hdyDFNgchSxGGztUrTXtNN"
    }
}' | jq .

sleep 5

printf "\nGet pending validators (currently empty): "
curl -s --location --request POST 'http://localhost:9650/ext/bc/P' \
--header 'Content-Type: application/json' \
--data-raw '{
    "jsonrpc": "2.0",
    "method": "platform.getPendingValidators",
    "params": {
        "subnetID": null,
        "nodeIDs": []
    },
    "id": 1
}' | jq .

sleep 5

printf "\nCreating new validator: NodeID-MFrZFVCXPv5iCn6M9K6XduxGTYp891xXZ "

CURR_TIME=$(date +%s)
START_TIME=$(($CURR_TIME+160))
END_TIME=$(($START_TIME+86400*365))

curl -s --location --request POST 'http://localhost:9650/ext/bc/P' \
--header 'Content-Type: application/json' \
--data-raw '{
    "jsonrpc": "2.0",
    "method": "platform.addValidator",
    "params": {
        "nodeID":"NodeID-MFrZFVCXPv5iCn6M9K6XduxGTYp891xXZ",
        "startTime":'$START_TIME',
        "endTime":'$END_TIME',
        "stakeAmount": 1000000000000000,
        "rewardAddress": "P-localflare1pz6dhzxvfmztknw35ukl8fav6gzjt9xwmkngua",
        "delegationFeeRate":0,
        "username": "user1234",
        "password": "b39d642078d2ca0517cafe008ddc9326fa1c4d71248078c67bf0d508993720e4"
    },
    "id": 1
}' | jq .

sleep 5

printf "\nCreating new validator: NodeID-NFBbbJ4qCmNaCzeW7sxErhvWqvEQMnYcN "

curl -s --location --request POST 'http://localhost:9650/ext/bc/P' \
--header 'Content-Type: application/json' \
--data-raw '{
    "jsonrpc": "2.0",
    "method": "platform.addValidator",
    "params": {
        "nodeID":"NodeID-NFBbbJ4qCmNaCzeW7sxErhvWqvEQMnYcN",
        "startTime":'$START_TIME',
        "endTime":'$END_TIME',
        "stakeAmount": 1000000000000000,
        "rewardAddress": "P-localflare1pz6dhzxvfmztknw35ukl8fav6gzjt9xwmkngua",
        "delegationFeeRate":0,
        "username": "user1234",
        "password": "b39d642078d2ca0517cafe008ddc9326fa1c4d71248078c67bf0d508993720e4"
    },
    "id": 1
}' | jq .

sleep 5

printf "\nCreating new validator: NodeID-GWPcbFJZFfZreETSoWjPimr846mXEKCtu "

curl -s --location --request POST 'http://localhost:9650/ext/bc/P' \
--header 'Content-Type: application/json' \
--data-raw '{
    "jsonrpc": "2.0",
    "method": "platform.addValidator",
    "params": {
        "nodeID":"NodeID-GWPcbFJZFfZreETSoWjPimr846mXEKCtu",
        "startTime":'$START_TIME',
        "endTime":'$END_TIME',
        "stakeAmount": 1000000000000000,
        "rewardAddress": "P-localflare1pz6dhzxvfmztknw35ukl8fav6gzjt9xwmkngua",
        "delegationFeeRate":0,
        "username": "user1234",
        "password": "b39d642078d2ca0517cafe008ddc9326fa1c4d71248078c67bf0d508993720e4"
    },
    "id": 1
}' | jq .

sleep 5

printf "\nCreating new validator: NodeID-P7oB2McjBGgW2NXXWVYjV8JEDFoW9xDE5 "

curl -s --location --request POST 'http://localhost:9650/ext/bc/P' \
--header 'Content-Type: application/json' \
--data-raw '{
    "jsonrpc": "2.0",
    "method": "platform.addValidator",
    "params": {
        "nodeID":"NodeID-P7oB2McjBGgW2NXXWVYjV8JEDFoW9xDE5",
        "startTime":'$START_TIME',
        "endTime":'$END_TIME',
        "stakeAmount": 1000000000000000,
        "rewardAddress": "P-localflare1pz6dhzxvfmztknw35ukl8fav6gzjt9xwmkngua",
        "delegationFeeRate":0,
        "username": "user1234",
        "password": "b39d642078d2ca0517cafe008ddc9326fa1c4d71248078c67bf0d508993720e4"
    },
    "id": 1
}' | jq .

sleep 5
printf "\nGet pending validators:\n"

curl -s --location --request POST 'http://localhost:9650/ext/bc/P' \
--header 'Content-Type: application/json' \
--data-raw '{
    "jsonrpc": "2.0",
    "method": "platform.getPendingValidators",
    "params": {
        "subnetID": null,
        "nodeIDs": []
    },
    "id": 1
}' | jq .

printf "\nNew validators will start in $(($START_TIME-$(date +%s))) seconds\n"
