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

## Multi-node Docker (node1/node2/node3)

Nodes run from the **go-titan** image built from this repo (`Dockerfile`), not `flarefoundation/go-flare`.

**Full deployment guide:** [DEPLOY-docker.md](../DEPLOY-docker.md) — node 1 bootstrap, node 2+ join, `docker compose`, remote machines.

**Docker Compose (downloads `origin.json` from GitHub):**

```bash
docker compose up -d node1    # bootstrap first
docker compose up -d node2 node3
```

Browse genesis in Titan Explorer: `/dashboard/origin` or `GET /api/titan/origin?format=raw`.

Local (builds `go-titan-local` automatically if missing):

1. `pnpm run node:docker:stop:all`
2. `pnpm run node:docker:start:all`

Force a fresh image build first:

```bash
pnpm run node:docker:build-and-start:all
```

Deployed image from CI (`dev/explorer` branch → `ghcr.io/pakeku/go-titan:dev-explorer`):

```bash
TITAN_NODE_IMAGE=ghcr.io/pakeku/go-titan:dev-explorer pnpm run node:docker:start:all
```

Ports:

- node1 API: `9650`, staking: `9651`
- node2 API: `9652`, staking: `9653`
- node3 API: `9654`, staking: `9655`

### How to test the nodes

1. Health endpoints:

```bash
pnpm run node:docker:test:health
```

2. Peer discovery (JSON-RPC `info.peers` on all nodes):

```bash
pnpm run node:docker:test:peers
```

3. Follow logs:

```bash
pnpm run node:docker:logs:node1
pnpm run node:docker:logs:node2
pnpm run node:docker:logs:node3
```

Expected behavior in this local/UAT setup:

- node1 starts first and acts as the initial bootstrap target
- node2 and node3 auto-bootstrap from `http://titan-node1:9650/ext/info`
- health should become stable once peer links are established

## Add more nodes later

The current pattern is intentionally simple to extend.

1. Copy `titan-network/scripts/start-node3-bg.sh` to a new file (example: `start-node4-bg.sh`).
2. Update container name, DB folder, and host ports:
	- `--name titan-node4`
	- `-p 9656:9650` and `-p 9657:9651`
	- `db-node4`
3. Keep bootstrap endpoint pointed to node1:
	- `AUTOCONFIGURE_BOOTSTRAP_ENDPOINT=http://titan-node1:9650/ext/info`
4. Add a root `package.json` script entry:
	- `"node:docker:start:node4:bg": "bash ./titan-network/scripts/start-node4-bg.sh"`
5. Append the new node start command in `node:docker:start:all`.
6. Expand `test-health.sh` and `test-peers.sh` with the node4 API port (`9656`).

Node DB folders are ignored from git by `.gitignore` rule `db-node*/`, so adding `db-node4`, `db-node5`, etc. will not pollute commits.