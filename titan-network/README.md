# Titan Network

This folder holds Titan Network launch artifacts, a deterministic origin generator input, and the technical design for the chain startup path.

## Files

- `Release-titan.md` release checklist and launch notes
- `TECHNICAL-DESIGN.md` code-referenced network flow document
- `origin.config.example.json` generator input for test wallets and allocations
- `origin.json` generated origin output

## pnpm

The root of the repo now includes a pnpm workspace file so we can install JavaScript dependencies in one shot when we add or update tooling:

```bash
pnpm install
```

Current workspace coverage is `test-scripts/`, and the root `package.json` is set up to support workspace installs.

## Generate the origin file

```bash
cd avalanchego
go run ./cmd/titan-genesis -config ../titan-network/origin.config.example.json -out ../titan-network/origin.json
```

The generator is deterministic:

- it sorts the wallet list before writing allocations
- it validates the config shape before writing output
- it writes stable JSON formatting

The `avaxAddr` values in the example config must be valid X-chain addresses for the Titan network you are creating; the example file is just a placeholder template.

## MVP order

1. Define the test wallet array.
2. Generate `origin.json`.
3. Start the bootstrap node.
4. Start provider nodes.
5. Verify node sync and chain behavior.