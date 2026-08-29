#!/bin/bash

# Define node IDs
NODE_IDS=("NodeID-MFrZFVCXPv5iCn6M9K6XduxGTYp891xXZ" "NodeID-NFBbbJ4qCmNaCzeW7sxErhvWqvEQMnYcN" "NodeID-GWPcbFJZFfZreETSoWjPimr846mXEKCtu" "NodeID-P7oB2McjBGgW2NXXWVYjV8JEDFoW9xDE5")
flare-stake-tool exportCP -a 6000000     --network localflare  --env-path empty_secret --get-hacked
sleep 1
flare-stake-tool importCP  --network localflare  --env-path empty_secret --get-hacked
sleep 1
# Loop over each Node ID and execute the command
for NODE_ID in "${NODE_IDS[@]}"; do
  flare-stake-tool transaction stake -n "$NODE_ID" -a 1000000 -s $(date -d "+5 minutes" +%s) -e $(date -d "+52 weeks" +%s) --network localflare --env-path ./empty_secret --get-hacked
  sleep 2
done

sleep 20

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
