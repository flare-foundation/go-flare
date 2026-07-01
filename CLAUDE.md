# Flare node implementation

This repository implements a Flare (and Songbird) node — a fork of [ava-labs/avalanchego](https://github.com/ava-labs/avalanchego) with the Flare/Songbird networks and features such as prioritized-contract handling, daemon-contract inflation, and eth-style P-chain signatures. Since avalanchego v1.14.x the coreth EVM ([ava-labs/coreth](https://github.com/ava-labs/coreth)) is **grafted in-tree** at `avalanchego/graft/coreth/` rather than being a separate project.

## Repository layout

- `avalanchego/` — the node itself (Go module `github.com/ava-labs/avalanchego`). Build with `cd avalanchego && ./scripts/build.sh` (binary at `avalanchego/build/avalanchego`).
- `avalanchego/graft/coreth/`, `avalanchego/graft/evm/`, `avalanchego/graft/subnet-evm/` — in-tree graft modules (coreth C-Chain EVM + shared EVM/sync code), wired via `replace` directives + the `go.work` workspace and compiled into the single `avalanchego` binary.
- `Dockerfile`, `Dockerfile.dless` — container builds (build context = repo root; they `COPY ./avalanchego` + `./config`). `README.md`, `README-docker.md`, `RELEASES-flare.md` — docs and release notes.

## Documentation

**All technical and upstream-merge documentation lives in [`avalanchego/CLAUDE.md`](avalanchego/CLAUDE.md)**, kept in sync with its twin [`avalanchego/AGENTS.md`](avalanchego/AGENTS.md). That guide covers:

- module layout, build & test, module hygiene (`go.work`, libevm pinning), protobuf regeneration;
- Flare-specific **avalanchego** modifications — network IDs, genesis, fork times, staking/0-rewards, eth-style signatures, big.Int validator weights, gossip;
- **coreth** (`graft/coreth`) specifics — chain IDs, the daemon/prioritized-contract/state-connector/governance mechanisms, the libevm-no-fork note, and the per-file Flare-specific list;
- the **Merge Guide** for merging ava-labs/avalanchego upstream into this fork (process, critical files, conflict resolution, post-merge checklist, known issues).

(This file and root `AGENTS.md` are intentionally just this overview; keep them in sync.)
