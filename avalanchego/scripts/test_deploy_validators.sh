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
START_TIME=$(($CURR_TIME+300))
END_TIME=$(($START_TIME+1209600))

curl -s --location --request POST 'http://localhost:9650/ext/bc/P' \
--header 'Content-Type: application/json' \
--data-raw '{
    "jsonrpc": "2.0",
    "method": "platform.addValidator",
    "params": {
        "nodeID":"NodeID-MFrZFVCXPv5iCn6M9K6XduxGTYp891xXZ",
        "startTime":'$START_TIME',
        "endTime":'$END_TIME',
        "stakeAmount": 10000000000000,
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
        "stakeAmount": 10000000000000,
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
        "stakeAmount": 10000000000000,
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
        "stakeAmount": 10000000000000,
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