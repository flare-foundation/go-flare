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
./scripts/build.sh       # Build avalanchego
./scripts/localflare.sh  # Run 5 localflare nodes
```

Check that the nodes are running by visiting `http://localhost:9650/ext/health` in your browser.

Then, in a separate terminal, run the test scripts in this (`test-scripts`) directory:

```bash
nvm use
corepack enable
pnpm install
pnpm run p-chain-import
pnpm run p-chain-export
pnpm run p-chain-transfer
pnpm run add-validator
pnpm run add-delegator
pnpm run x-chain-import
pnpm run x-chain-export
```

or if Etna has already started, use:

```bash
nvm use
corepack enable
pnpm install
pnpm run etna-p-chain-import
pnpm run etna-p-chain-export
pnpm run etna-x-chain-import
pnpm run etna-x-chain-export
pnpm run etna-add-validator
pnpm run etna-add-delegator
```

Note:

- Nodejs version >=24 is required to run the scripts, also make sure you have the `pnpm` package manager installed.
- The scripts assume that the localflare network is running and accessible at the default ports.
