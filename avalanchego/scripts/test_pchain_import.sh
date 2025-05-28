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

printf "\nUpdating user's C-chain public key to 0x8db97C7cEcE249c2b98bDC0226Cc4C2A57BF52FC: "
curl -s --location --request POST 'http://localhost:9650/ext/bc/C/avax' \
--header 'Content-Type: application/json' \
--data-raw '{
    "method": "avax.importKey",
    "params": {
        "username":"user1234",
        "password":"b39d642078d2ca0517cafe008ddc9326fa1c4d71248078c67bf0d508993720e4",
        "privateKey":"PrivateKey-ewoqjP7PxY4yr3iLTpLisriqt94hdyDFNgchSxGGztUrTXtNN"
    },
    "jsonrpc": "2.0",
    "id": 1
}' | jq .

sleep 5

printf "\nTesting P->C export: "
curl -s --location --request POST 'http://localhost:9650/ext/bc/P' \
--header 'Content-Type: application/json' \
--data-raw '{
    "jsonrpc": "2.0",
    "method": "platform.exportAVAX",
    "params": {
        "to":"C-localflare18jma8ppw3nhx5r4ap8clazz0dps7rv5uj3gy4v",
        "amount":1000000000,
        "username": "user1234",
        "password": "b39d642078d2ca0517cafe008ddc9326fa1c4d71248078c67bf0d508993720e4"
    },
    "id": 1
}' | jq .

sleep 5

printf "\nTesting P->C import: "
curl -s --location --request POST 'http://localhost:9650/ext/bc/C/avax' \
--header 'Content-Type: application/json' \
--data-raw '{
    "method": "avax.importAVAX",
    "params": {
        "username":"user1234",
        "password":"b39d642078d2ca0517cafe008ddc9326fa1c4d71248078c67bf0d508993720e4",
        "sourceChain": "P",
        "to":"0x8db97C7cEcE249c2b98bDC0226Cc4C2A57BF52FC"
    },
    "jsonrpc": "2.0",
    "id": 1
}' | jq .

sleep 5

printf "\nTesting C->P export: "
curl -s --location --request POST 'http://localhost:9650/ext/bc/C/avax' \
--header 'Content-Type: application/json' \
--data-raw '{
    "method" :"avax.exportAVAX",
    "params" :{
        "from": ["0x8db97C7cEcE249c2b98bDC0226Cc4C2A57BF52FC"],
        "to":"P-localflare18jma8ppw3nhx5r4ap8clazz0dps7rv5uj3gy4v",
        "amount": 100000000,
        "destinationChain": "P",
        "changeAddr": "0x8db97C7cEcE249c2b98bDC0226Cc4C2A57BF52FC",
        "username":"user1234",
        "password":"b39d642078d2ca0517cafe008ddc9326fa1c4d71248078c67bf0d508993720e4" 
    },
    "jsonrpc":"2.0",
    "id"     :1
}' | jq .

sleep 5

printf "\nTesting C->P import: "
curl -s --location --request POST 'http://localhost:9650/ext/bc/P' \
--header 'Content-Type: application/json' \
--data-raw '{
    "jsonrpc": "2.0",
    "method": "platform.importAVAX",
    "params": {
        "username":"user1234",
        "password":"b39d642078d2ca0517cafe008ddc9326fa1c4d71248078c67bf0d508993720e4",
        "sourceChain": "C",
        "to":"P-localflare18jma8ppw3nhx5r4ap8clazz0dps7rv5uj3gy4v"
    },
    "id": 1
}' | jq .
