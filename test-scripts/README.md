# Test scripts for Flare's P-Chain operations

Tests can be performed on **localflare** network and include:

- C-chain to P-chain transfer (100 FLR)
- P-chain to C-chain transfer (100 FLR)
- Transfer to another address on the P-chain (100 FLR to `P-localflare1zjaa3yjnzn5cjx9r56x59raam2jgwnmztlg995`)
- Add validator (`NodeID-MFrZFVCXPv5iCn6M9K6XduxGTYp891xXZ` with weight 10,000 FLR and duration of 14 days)
- Add delegator (To node `NodeID-7Xhw2mDxuDS44j42TCB6U5579esbSt3Lg` with weight 10,000 FLR and duration of 1 hour)
- C-chain to X-chain transfer (100 FLR)
- X-chain to C-chain transfer (100 FLR)

Run localflare in `avalanchego` directory:

```bash
./scripts/build.sh. # Build avalanchego
./scripts/localflare.sh. # Run 5 localflare nodes
```

Check that the nodes are running by visiting `http://localhost:9650/ext/health` in your browser.

Then, in a separate terminal, run the test scripts in this (`test-scripts`) directory:

```bash
yarn
yarn run p-chain-import
yarn run p-chain-export
yarn run p-chain-transfer
yarn run add-validator
yarn run add-delegator
yarn run x-chain-import
yarn run x-chain-export
```

Note:

- Nodejs version >=20 is required to run the scripts, also make sure you have the `yarn` package manager installed.
- The scripts assume that the localflare network is running and accessible at the default ports.
