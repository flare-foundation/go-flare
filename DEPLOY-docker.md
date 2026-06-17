# Deploy Titan nodes with Docker

This guide deploys Titan Network nodes from [github.com/pakeku/go-titan](https://github.com/pakeku/go-titan) using Docker. Every node downloads the same genesis (`titan-network/origin.json`) at startup. The **parent node** (first server) bootstraps the network; **every other server** runs one joining node that uses the parent’s IP and NodeID.

Browse the live origin in Titan Explorer: `/dashboard/origin`  
Download endpoints:

| Source | URL |
|--------|-----|
| GitHub (raw, default for Docker) | `https://raw.githubusercontent.com/pakeku/go-titan/dev/explorer/titan-network/origin.json` |
| Explorer API (when dashboard is running) | `http://localhost:3000/api/titan/origin?format=raw` |

---

## Prerequisites

- Docker 24+ with Compose v2
- Ports open on each server: API `9650` + staking `9651` (multi-node local compose offsets ports on one host; see port map below)
- Git (optional — only needed to clone the repo or build locally)

**Default image:** `ghcr.io/pakeku/go-titan:dev-explorer`  
**Network / chain ID:** `781337`

---

## Quick install — fresh Ubuntu server (one line)

Installs Docker, clones the repo, opens **nano** to edit `.env`, and registers a **systemd** service so the node keeps running after you close SSH and on reboot. No Go/Java on the host — the container uses the pre-built image.

```bash
curl -fsSL https://raw.githubusercontent.com/pakeku/go-titan/dev/explorer/scripts/install-titan-node.sh | sudo bash
```

**Parent server** — leave the bootstrap preset in `.env`.  
**Joining server** — set `PARENT_HOST`, `TITAN_AUTOCONFIGURE_BOOTSTRAP=1`, and `TITAN_BOOTSTRAP_ENDPOINT`.

After install:

```bash
sudo systemctl status titan-node
curl -sf http://localhost:9650/ext/health
```

Reconfigure later: `sudo nano /opt/titan-node/.env` then `sudo systemctl restart titan-node`.

---

## Option A — Multi-node Compose (local dev on one machine)

Clone the repo (or download only `docker-compose.yml`):

```bash
git clone https://github.com/pakeku/go-titan.git
cd go-titan
```

### Step 1 — Deploy node 1 (bootstrap)

Node 1 downloads `origin.json` from GitHub and starts the network. No bootstrap peers are required.

```bash
docker compose up -d node1
```

Wait until healthy:

```bash
docker compose ps
curl -sf http://localhost:9650/ext/health
```

**Record node 1 identity** (needed for remote node 2+):

```bash
# Node ID
curl -s -X POST http://localhost:9650/ext/info \
  -H 'Content-Type: application/json' \
  -d '{"jsonrpc":"2.0","id":1,"method":"info.getNodeID"}' | jq -r .result.nodeID

# Staking IP:port (use the host-reachable address for remote machines)
curl -s -X POST http://localhost:9650/ext/info \
  -H 'Content-Type: application/json' \
  -d '{"jsonrpc":"2.0","id":1,"method":"info.getNodeIP"}' | jq -r .result.ip
```

Example output:

- NodeID: `NodeID-KrAHHbov7fC2FH5j1pNgkfAsYL8d5Dfms`
- IP: `1.2.3.4:9651` (public or LAN address + staking port)

### Step 2 — Deploy node 2 (same machine)

On the same host, Compose bootstraps node 2 from node 1 automatically:

```bash
docker compose up -d node2
```

Verify:

```bash
curl -sf http://localhost:9652/ext/health
```

### Step 3 — Deploy node 3 (and beyond)

```bash
docker compose up -d node3
```

Full stack in one command:

```bash
docker compose up -d
```

Stop everything:

```bash
docker compose down
```

---

## Option B — Single-node Compose (recommended for production)

One Titan node per server. Copy the same two files to every machine: `docker-compose.single-node.yml` and a server-specific `.env`.

```bash
git clone https://github.com/pakeku/go-titan.git
cd go-titan
cp .env.single-node.example .env
```

### Step 1 — Parent server (bootstrap)

On the **first** machine, set `.env` to the bootstrap preset:

```bash
TITAN_ROLE=bootstrap
TITAN_AUTOCONFIGURE_BOOTSTRAP=0
```

Start the node:

```bash
docker compose -f docker-compose.single-node.yml up -d
```

Wait until healthy, then **record the parent identity** (used by every future server):

```bash
curl -sf http://localhost:9650/ext/health

# Node ID
curl -s -X POST http://localhost:9650/ext/info \
  -H 'Content-Type: application/json' \
  -d '{"jsonrpc":"2.0","id":1,"method":"info.getNodeID"}' | jq -r .result.nodeID

# Staking IP:port (public or LAN address peers can reach)
curl -s -X POST http://localhost:9650/ext/info \
  -H 'Content-Type: application/json' \
  -d '{"jsonrpc":"2.0","id":1,"method":"info.getNodeIP"}' | jq -r .result.ip
```

On remote hosts, also set `PUBLIC_IP` in `.env` (or `AUTOCONFIGURE_PUBLIC_IP=1`) so other nodes can dial this server’s staking port.

### Step 2 — Joining servers

On each **additional** machine, copy the repo and `.env.single-node.example`, then set the joiner preset (replace `PARENT_HOST` with the parent’s reachable IP or hostname):

```bash
TITAN_ROLE=joiner
PARENT_HOST=1.2.3.4
TITAN_AUTOCONFIGURE_BOOTSTRAP=1
TITAN_BOOTSTRAP_ENDPOINT=http://1.2.3.4:9650/ext/info
PUBLIC_IP=<this-server-public-ip>
```

Deploy:

```bash
docker compose -f docker-compose.single-node.yml up -d
curl -sf http://localhost:9650/ext/health
```

The entrypoint calls `info.getNodeIP` and `info.getNodeID` on the parent and sets `BOOTSTRAP_IPS` / `BOOTSTRAP_IDS` automatically.

**Always bootstrap from the parent node**, not from other joiners. Node 3, 4, … use the same joiner `.env` pattern with the same `PARENT_HOST`.

### Manual bootstrap (optional)

If you already know the parent’s staking address and NodeID, set in `.env`:

```bash
TITAN_AUTOCONFIGURE_BOOTSTRAP=0
BOOTSTRAP_IPS=1.2.3.4:9651
BOOTSTRAP_IDS=NodeID-xxxxxxxxxxxxxxxxxxxxxxxxxxxx
```

### Stop / restart

```bash
docker compose -f docker-compose.single-node.yml down
docker compose -f docker-compose.single-node.yml up -d
```

---

## Option C — Remote node (`docker run`)

Deploy the parent on **machine A** first (Option B Step 1 or Option D). On **machine B**, run a joining node with the parent’s **staking IP:port** and **NodeID**.

### Method 1 — Auto-bootstrap from node 1 API (easiest)

Replace `NODE1_HOST` with machine A’s reachable IP or hostname:

```bash
docker run -d \
  --name titan-node2 \
  -p 9650:9650 \
  -p 9651:9651 \
  -e ORIGIN_URL=https://raw.githubusercontent.com/pakeku/go-titan/dev/explorer/titan-network/origin.json \
  -e AUTOCONFIGURE_PUBLIC_IP=0 \
  -e AUTOCONFIGURE_BOOTSTRAP=1 \
  -e AUTOCONFIGURE_BOOTSTRAP_ENDPOINT=http://NODE1_HOST:9650/ext/info \
  -e EXTRA_ARGUMENTS="--genesis=/app/titan/origin.json --network-id=781337 --http-host=0.0.0.0 --http-port=9650 --staking-port=9651 --db-dir=/app/db --log-level=warn" \
  ghcr.io/pakeku/go-titan:dev-explorer
```

The entrypoint calls `info.getNodeIP` and `info.getNodeID` on node 1 and sets `BOOTSTRAP_IPS` / `BOOTSTRAP_IDS` automatically.

### Method 2 — Explicit bootstrap IP and NodeID

Use this when you already know node 1’s values:

```bash
docker run -d \
  --name titan-node2 \
  -p 9650:9650 \
  -p 9651:9651 \
  -e ORIGIN_URL=https://raw.githubusercontent.com/pakeku/go-titan/dev/explorer/titan-network/origin.json \
  -e AUTOCONFIGURE_PUBLIC_IP=0 \
  -e AUTOCONFIGURE_BOOTSTRAP=0 \
  -e BOOTSTRAP_IPS="NODE1_HOST:9651" \
  -e BOOTSTRAP_IDS="NodeID-xxxxxxxxxxxxxxxxxxxxxxxxxxxx" \
  -e EXTRA_ARGUMENTS="--genesis=/app/titan/origin.json --network-id=781337 --http-host=0.0.0.0 --http-port=9650 --staking-port=9651 --db-dir=/app/db --log-level=warn" \
  ghcr.io/pakeku/go-titan:dev-explorer
```

### Node 3, node 4, …

Same pattern as node 2 — always bootstrap from **node 1** (the genesis validator), not from node 2. Use unique host ports if multiple nodes share one machine.

---

## Option D — Parent node only (`docker run`)

Minimal bootstrap node without Compose:

```bash
docker run -d \
  --name titan-node1 \
  -p 9650:9650 \
  -p 9651:9651 \
  -e ORIGIN_URL=https://raw.githubusercontent.com/pakeku/go-titan/dev/explorer/titan-network/origin.json \
  -e AUTOCONFIGURE_PUBLIC_IP=0 \
  -e AUTOCONFIGURE_BOOTSTRAP=0 \
  -e EXTRA_ARGUMENTS="--genesis=/app/titan/origin.json --network-id=781337 --http-host=0.0.0.0 --http-port=9650 --staking-port=9651 --db-dir=/app/db --log-level=warn" \
  ghcr.io/pakeku/go-titan:dev-explorer
```

At container start the entrypoint downloads origin from GitHub into `/app/titan/origin.json`.

---

## Origin URL sources

| Variable | Purpose |
|----------|---------|
| `ORIGIN_URL` | HTTP(S) URL fetched at container start (GitHub raw or Explorer API) |
| `GENESIS_FILE` | Local path written by the download (default `/app/titan/origin.json`) |
| `TITAN_NETWORK_ID` | Metadata / docs default (`781337`) |

Override the GitHub branch/path when using a fork or release tag:

```bash
export TITAN_ORIGIN_URL=https://raw.githubusercontent.com/pakeku/go-titan/main/titan-network/origin.json
docker compose -f docker-compose.single-node.yml up -d
```

Use the Explorer as origin (dashboard must be reachable from the container):

```bash
-e ORIGIN_URL=http://host.docker.internal:3000/api/titan/origin?format=raw
```

---

## Bootstrap environment reference

| Variable | Node 1 | Node 2+ |
|----------|--------|---------|
| `ORIGIN_URL` | Required (unless volume-mounted) | Same URL as node 1 |
| `AUTOCONFIGURE_BOOTSTRAP` | `0` | `1` (auto) or `0` (manual) |
| `AUTOCONFIGURE_BOOTSTRAP_ENDPOINT` | — | `http://<node1-host>:9650/ext/info` |
| `BOOTSTRAP_IPS` | — | `<node1-host>:9651` (manual) |
| `BOOTSTRAP_IDS` | — | `NodeID-…` from node 1 (manual) |

---

## Build image from source

```bash
git clone https://github.com/pakeku/go-titan.git
cd go-titan
docker build -t go-titan-local .
export TITAN_NODE_IMAGE=go-titan-local
docker compose -f docker-compose.single-node.yml up -d
```

---

## Verify the cluster

```bash
# Health
curl -sf http://localhost:9650/ext/health
curl -sf http://localhost:9652/ext/health

# Peers (repeat for each API port)
curl -s -X POST http://localhost:9650/ext/info \
  -H 'Content-Type: application/json' \
  -d '{"jsonrpc":"2.0","id":1,"method":"info.peers"}' | jq .result.peers

# C-Chain RPC
curl -s -X POST http://localhost:9650/ext/bc/C/rpc \
  -H 'Content-Type: application/json' \
  -d '{"jsonrpc":"2.0","id":1,"method":"eth_chainId","params":[]}'
```

With Titan Explorer running (`pnpm run explorer:dev`), open:

- [http://localhost:3000/dashboard/origin](http://localhost:3000/dashboard/origin) — genesis browser
- [http://localhost:3000/dashboard/nodes](http://localhost:3000/dashboard/nodes) — live node status

---

## Port map (multi-node local Compose)

| Node | Container | API (host) | Staking (host) |
|------|-----------|------------|----------------|
| node1 | `titan-node1` | 9650 | 9651 |
| node2 | `titan-node2` | 9652 | 9653 |
| node3 | `titan-node3` | 9654 | 9655 |

Staking port **9651 on node 1** must be reachable from other nodes when deploying across machines.