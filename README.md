# go-titan

Local development monorepo for the **Titan Network** — a private UAT Avalanche-compatible chain built on a fork of [go-flare](README-goflare.md) (`avalanchego` + `coreth`). This repo wires together the node binaries, a 3-node Docker topology, genesis tooling, and the **Titan Explorer** developer dashboard.

## What is in this repo

| Path | Purpose |
|------|---------|
| `avalanchego/` | Forked AvalancheGo node (based on go-flare / v1.14.0 lineage) |
| `coreth/` | C-Chain EVM implementation |
| `titan-network/` | Genesis (`origin.json`), network scripts, launch docs |
| `apps/developer-tool-kit/` | **Titan Explorer** — Next.js dashboard (explorer, contracts, logs, wallet) |
| `test-scripts/` | Flare P-chain / X-chain test utilities (upstream) |
| `tools/avalanche-cli/` | Avalanche CLI submodule |
| `Dockerfile` | Build a local `go-titan-local` node image from source |

**Network ID / Chain ID:** `781337` (`0xbec19`)  
**Native token:** TITAN (18 decimals)  
**Default C-Chain RPC:** `http://localhost:9650/ext/bc/C/rpc`

## Prerequisites

- **Docker** — recommended for running the 3-node local network
- **Node.js** ≥ 20 and **pnpm** 9 (`corepack enable` or `npm i -g pnpm`)
- **Go** 1.24 + gcc/g++ — only needed to build `avalanchego` from source
- **Git Bash** or WSL on Windows — shell scripts under `titan-network/scripts/` use bash
- **MetaMask** (optional) — connect to Titan in the dashboard sidebar

## Deploy on Ubuntu server (one line)

On a fresh Ubuntu machine, install Docker, clone this repo, edit `.env` in nano, and run the node as a **systemd** service (survives SSH disconnect and reboot):

```bash
curl -fsSL https://raw.githubusercontent.com/pakeku/go-titan/dev/explorer/scripts/install-titan-node.sh | sudo bash
```

- **Parent (first server):** keep `TITAN_ROLE=bootstrap` in `.env`
- **Joining servers:** set `PARENT_HOST`, `TITAN_AUTOCONFIGURE_BOOTSTRAP=1`, and `TITAN_BOOTSTRAP_ENDPOINT`

After install:

```bash
sudo systemctl status titan-node
curl -sf http://localhost:9650/ext/health
```

Full guide: [DEPLOY-docker.md](DEPLOY-docker.md)

## Quick start (recommended)

### 1. Install JavaScript dependencies

```bash
pnpm install
```

### 2. Start the 3-node Titan network (Docker)

```bash
pnpm run node:docker:stop:all    # clean up any previous run
pnpm run node:docker:start:all   # node1 → node2 → node3
```

Verify nodes are healthy and peered:

```bash
pnpm run node:docker:test:health
pnpm run node:docker:test:peers
```

| Node | API port | Staking port | Container |
|------|----------|--------------|-----------|
| node1 | 9650 | 9651 | `titan-node1` |
| node2 | 9652 | 9653 | `titan-node2` |
| node3 | 9654 | 9655 | `titan-node3` |

Tail logs:

```bash
pnpm run node:docker:logs:node1
pnpm run node:docker:logs:node2
pnpm run node:docker:logs:node3
```

Stop everything:

```bash
pnpm run node:docker:stop:all
```

### 3. Start Titan Explorer (developer dashboard)

```bash
pnpm run explorer:dev
```

Open [http://localhost:3000/dashboard/default](http://localhost:3000/dashboard/default).

## Titan Explorer features

The dashboard (`apps/developer-tool-kit`) is the primary UI for local development:

| Route | Feature |
|-------|---------|
| `/dashboard/default` | Network overview, MetaMask connect, node health strip |
| `/dashboard/nodes` | Per-node RPC URLs and status |
| `/dashboard/activity` | Block explorer — infinite-scroll feed, search, shareable `?q=` URLs, right-hand detail drawer for blocks / transactions / addresses |
| `/dashboard/contracts` | **Contract Studio** — write Solidity, compile via `/api/titan/compile`, deploy with MetaMask (Paris EVM target) |
| `/dashboard/origin` | **Network origin** — browse genesis, download URLs for Docker, stakers & prefunded accounts |
| `/dashboard/logs` | Live Docker log viewer for `titan-node1`–`titan-node3` |
| `/dashboard/containers` | Container management view |

**Wallet:** Connect MetaMask from the sidebar. When connected, the overview page hides redundant connect buttons and shows your TITAN balance.

**Add Titan to MetaMask**

- Network name: Titan Local UAT
- RPC URL: `http://localhost:9650/ext/bc/C/rpc`
- Chain ID: `781337`
- Currency symbol: `TITAN`

### Genesis prefunded C-Chain addresses

These accounts are funded in `titan-network/origin.json` for local testing:

| Address | Notes |
|---------|-------|
| `0xf56e38f35d52d30c512086B1564cFaAA5686B769` | 1B TITAN |
| `0x0FA0f5B26763b3baE5fF9d4d156542c0CA8AaA02` | 1B TITAN |
| `0x22e6d2e3e613006Fc39620760A4322872fBe336d` | 1B TITAN |
| `0x49077293fe7049400A91D14395dbCad16A98Ea47` | 5B TITAN |
| `0xC2Ff7887DED0C04F64F10677fb828CE4b4178e91` | 2B TITAN |

Import a private key for one of these into MetaMask to deploy contracts or send transactions.

### API routes (Next.js)

| Endpoint | Description |
|----------|-------------|
| `GET/POST /api/titan/rpc` | Proxy to node1–3 JSON-RPC and health |
| `POST /api/titan/compile` | Solidity compile (`solc`, Paris EVM) |
| `GET /api/titan/logs` | Stream stamped Docker logs from Titan containers |
| `GET /api/titan/origin` | Network genesis summary; `?format=raw` downloads `origin.json` for nodes |

## Root scripts reference

All commands run from the repo root via `pnpm run <script>`.

### Node — build & run natively (Windows)

| Script | Description |
|--------|-------------|
| `node:build:avalanchego` | Build `avalanchego/build/avalanchego.exe` |
| `node:start:node1` | Run node1 against `titan-network/origin.json` |
| `node:start:node1:genesis` | Run node1 against `titan-network/genesis.json` |
| `node:build-and-start:node1` | Build then start node1 |

DB and staking data land in `db-node1/` (gitignored).

### Node — Docker (single node)

| Script | Description |
|--------|-------------|
| `node:docker:build` | Build `go-titan-local` image from repo `Dockerfile` |
| `node:docker:start:node1` | Interactive single-node run (local image) |
| `node:docker:build-and-start:node1` | Build + run |
| `node:docker:build-and-start:all` | Build image then start 3-node network |
| `node:docker:start:node1:remote` | Run using CI image `ghcr.io/pakeku/go-titan:dev-explorer` |

### Node — Docker (3-node network)

| Script | Description |
|--------|-------------|
| `node:docker:network:create` | Create `titan-net` Docker network |
| `node:docker:free-ports` | Stop containers holding Titan ports |
| `node:docker:start:node1:bg` | Start node1 in background |
| `node:docker:start:node2:bg` | Start node2 (bootstraps from node1) |
| `node:docker:start:node3:bg` | Start node3 (bootstraps from node1) |
| `node:docker:start:all` | Full orchestrated startup (auto-builds `go-titan-local` if needed) |
| `node:docker:stop:all` | Stop and remove all Titan containers |
| `node:docker:test:health` | Hit health endpoints on all nodes |
| `node:docker:test:peers` | Check `info.peers` on all nodes |
| `node:docker:logs:node1` | Follow node1 logs |
| `node:docker:logs:node2` | Follow node2 logs |
| `node:docker:logs:node3` | Follow node3 logs |

### Explorer

| Script | Description |
|--------|-------------|
| `explorer:dev` | `next dev` on port 3000 |
| `explorer:build` | Production build |

### Workspace

| Script | Description |
|--------|-------------|
| `install:all` | `pnpm install -r` across workspace packages |

### Solidity workspace (planned)

`pnpm-workspace.yaml` lists a `solidity/` package for standalone compile/deploy scripts (`sol:compile`, `sol:deploy:local`). Contract workflows are fully supported today via **Contract Studio** in the dashboard; the standalone package may be added later.

## Genesis tooling

Regenerate `origin.json` from a config file:

```bash
cd avalanchego
go run ./cmd/titan-genesis -config ../titan-network/origin.config.example.json -out ../titan-network/origin.json
```

See [titan-network/README.md](titan-network/README.md) for multi-node topology details and how to add node4+.

## Building the node from source

```bash
pnpm run node:build:avalanchego
# or
cd avalanchego && ./scripts/build.sh
```

Requirements match the upstream fork: Go 1.24, gcc, g++ — see [README-goflare.md](README-goflare.md).

## Project layout (pnpm workspace)

```
go-titan/
├── apps/developer-tool-kit/   # Titan Explorer (titan-explorer)
├── test-scripts/              # Flare chain test scripts
├── solidity/                  # (planned) standalone Solidity tooling
├── titan-network/             # genesis + docker orchestration
├── avalanchego/                 # node binary
└── coreth/                    # EVM plugin
```

## Testing

- **Go unit tests:** see [tests/README.md](tests/README.md)
- **Flare P-chain scripts:** see [test-scripts/README.md](test-scripts/README.md) (targets upstream localflare, not Titan genesis)
- **Network smoke tests:** `pnpm run node:docker:test:health` and `node:docker:test:peers`

## Further reading

- [README-goflare.md](README-goflare.md) — upstream go-flare documentation (validator deploy, observation nodes, CI images)
- [DEPLOY-docker.md](DEPLOY-docker.md) — **step-by-step Titan node deployment** (node 1 bootstrap, node 2+ join, docker compose)
- [README-docker.md](README-docker.md) — `flarefoundation/go-flare` container configuration
- [titan-network/README.md](titan-network/README.md) — Titan genesis, ports, extending the node cluster
- [apps/developer-tool-kit/README.md](apps/developer-tool-kit/README.md) — dashboard template upstream docs

## Typical dev workflow

```bash
# Terminal 1 — network
pnpm run node:docker:start:all
pnpm run node:docker:test:health

# Terminal 2 — dashboard
pnpm run explorer:dev
```

Then browse blocks at `/dashboard/activity`, deploy a contract at `/dashboard/contracts`, and watch node output at `/dashboard/logs`.