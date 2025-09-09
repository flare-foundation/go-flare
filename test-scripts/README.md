# Test scripts for Flare's P-Chain operations

Tests are performed on **localflare** network and include:

- C-chain to P-chain transfer (100 FLR)
- P-chain to C-chain transfer (100 FLR)
- Add validator (`NodeID-MFrZFVCXPv5iCn6M9K6XduxGTYp891xXZ` with weight 10,000 FLR and duration of 14 days)
- Add delegator (To node `NodeID-7Xhw2mDxuDS44j42TCB6U5579esbSt3Lg` with weight 10,000 FLR and duration of 1 hour)

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
yarn run x-chain-import
yarn run x-chain-export
yarn run add-validator
yarn run add-delegator
```

or if Etna has already started, use:

```bash
yarn
yarn run etna-p-chain-import
yarn run etna-p-chain-export
yarn run etna-x-chain-import
yarn run etna-x-chain-export
yarn run etna-add-validator
yarn run etna-add-delegator
```

Note:

- Nodejs version 20 is required to run the scripts, also make sure you have the `yarn` package manager installed.
- The scripts assume that the localflare network is running and accessible at the default ports.
