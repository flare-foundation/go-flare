# Coreth — Flare Fork (grafted into avalanchego)

This is the coreth component of the Flare node, forked from [ava-labs/coreth](https://github.com/ava-labs/coreth). It implements the EVM execution environment (C-Chain) for Flare and Songbird networks.

**Since avalanchego v1.14.x, coreth is no longer a standalone repository.** Upstream "grafted" it into the avalanchego repo as the Go module `github.com/ava-labs/avalanchego/graft/coreth` (this directory). It is wired to the root module via `replace` directives in the root `go.mod` and a `go.work` workspace, and is compiled directly into the `avalanchego` binary — there is no separate `evm` plugin binary and no copying into `GOPATH` anymore. The pre-graft standalone project (module `github.com/ava-labs/coreth`) is kept only for reference in the repo-root sibling folder `coreth/`.

## Version

- Part of go-flare based on avalanchego **v1.14.2** (branch `flare-merge-1_14_2`). This is a pinned release, **not** tracking the latest upstream.
- `github.com/ava-labs/libevm` version is pinned in the **root** avalanchego `go.mod` and in this module's `go.mod` (currently `v1.13.14-0.4.0.rc.2`). Because of the `go.work` workspace, the highest requirement wins for the whole build — keep the root and graft modules on the **same** libevm version or precompile ABI helpers (`PackOutput`, `UnpackInputIntoInterface`, `PackEvent`) will break.
- Newer upstream versions of coreth/avalanchego/libevm exist — do not assume features, types, or APIs from those releases are present here.

## Module and Dependencies

- Module: `github.com/ava-labs/avalanchego/graft/coreth`
- Sibling graft modules: `graft/evm` (shared EVM/sync/triedb code split out by upstream) and `graft/subnet-evm`.
- `github.com/ava-labs/avalanchego` resolves to the repo root via `replace ../..` semantics (root `go.mod` + `go.work`).
- `github.com/ava-labs/libevm` is consumed as a normal upstream dependency (no local fork / no `replace`). Previously-forked behavior is implemented directly in coreth — see [core/daemon_call.go](core/daemon_call.go).

## Supported Networks

| Network     | Chain ID     | Role                    |
|-------------|--------------|-------------------------|
| Flare       | 14           | Flare mainnet           |
| Costwo      | 114          | Flare testnet           |
| Songbird    | 19           | Songbird mainnet        |
| Coston      | 16           | Songbird testnet        |
| LocalFlare  | 162          | Local Flare testing     |
| Local       | 4294967295   | Local Songbird testing  |

Chain IDs are defined in [params/config.go](params/config.go). Network IDs (different from chain IDs) live in the root module's `utils/constants/network_ids.go`.

## Build and Test

```bash
# Build the node (from the avalanchego repo root; includes coreth)
./scripts/build.sh

# Build only this module
cd graft/coreth && go build ./...

# Run unit tests for this module
cd graft/coreth && go test $(go list ./... | grep -v /tests/)

# Module hygiene (run from repo root; see Taskfile go-mod-tidy)
GOWORK=off go mod tidy   # in each of: ., graft/evm, graft/coreth, graft/subnet-evm
go work sync
```

**Note:** Tests in `internal/ethapi/testdata/` compare against JSON files containing tx hashes and state roots. When merging upstream, these may need to be updated for the different network ID (1 vs 14).

**Note:** Tests requiring `listen tcp 127.0.0.1:0` will fail in sandboxed environments — this is not a code issue.

## Directory Structure

```
core/          - Blockchain logic, state transitions, Flare-specific contracts
params/        - Chain configs, network upgrades, Flare parameters
plugin/evm/    - VM implementation; atomic transactions in plugin/evm/atomic
precompile/    - Smart contract precompiles
eth/           - Ethereum protocol
miner/         - Block production
warp/          - Avalanche Warp Messaging
../evm/        - graft/evm module: shared sync/, triedb/, rpc/ code (moved out of coreth by upstream)
```

## Flare-Specific Files (Always Preserve on Merge)

### Core

| File | Purpose |
|------|---------|
| [core/daemon.go](core/daemon.go) | Daemon contract calls for inflation minting; prioritized contract detection (FTSO, submitter) |
| [core/daemon_call.go](core/daemon_call.go) | `DaemonCall` helper (snapshot + tracer-disabled `evm.Call`); replaces the method previously added in the Flare libevm fork |
| [core/daemon_test.go](core/daemon_test.go) | Tests for daemon logic |
| [core/governance_settings.go](core/governance_settings.go) | Governance address/timelock updates, airdrop and distribution contract management |
| [core/state_connector.go](core/state_connector.go) | State Connector attestation protocol integration |
| [core/state_transition.go](core/state_transition.go) | Modified to integrate daemon, governance, state connector; fee refunds for prioritized contracts |

### Params

| File | Purpose |
|------|---------|
| [params/config.go](params/config.go) | Flare/Songbird chain IDs, `TestFlareChainConfig` |
| [params/config_libevm.go](params/config_libevm.go) | `RulesExtra.IsSongbirdCode` derivation from `SnowCtx.NetworkID` |
| [params/extras/config.go](params/extras/config.go) | `SnowCtx` in `ChainConfig`, `IsSongbirdCode`/`IsFlareFamilyCode`, `ACP176Params()` (Granite params on Flare-family networks) |
| [params/extras/network_upgrades.go](params/extras/network_upgrades.go) | Flare-specific network upgrade timing, `SongbirdTransitionTimestamp` |

### Plugin

| File | Purpose |
|------|---------|
| [plugin/evm/atomic/export_tx.go](plugin/evm/atomic/export_tx.go) | Export tx logic |
| [plugin/evm/atomic/vm/tx_semantic_verifier.go](plugin/evm/atomic/vm/tx_semantic_verifier.go) | Eth-style signature prefix support (`accounts.TextHash`) for export txs |
| [plugin/evm/atomic/vm/vm.go](plugin/evm/atomic/vm/vm.go) | Atomic transaction functions (`verifyTxAtTip`, `verifyTx`, `verifyTxs`, `GetAtomicUTXOs`) — moved here from plugin/evm/vm.go |
| [plugin/evm/upgrade/sgbt/](plugin/evm/upgrade/sgbt/) | Songbird transition gas limit parameters |
| [plugin/evm/upgrade/granite/](plugin/evm/upgrade/granite/) | Flare-family ACP-176 parameter overrides for the Granite fork (min gas price, capacity) |
| [plugin/evm/customheader/gas_limit.go](plugin/evm/customheader/gas_limit.go) | Songbird gas limit schedule between AP1 and Cortina |
| [sync/client/client.go (graft/evm)](../evm/sync/client/client.go) | `StateSyncVersionSgb` minimum version for Songbird state sync |

## Flare-Specific Mechanisms

### Daemon Contract (Inflation)
- Called each block on Flare and Songbird networks
- Contract address: `0x1000000000000000000000000000000000000002`
- Returns a mint request (uint256); if valid and ≤ max, tokens are minted to daemon contract
- Max mint: 60M tokens/block on Flare/Costwo/LocalFlare, 50M on Songbird/Coston

### Prioritized Contracts (Fee Refunds)
- FTSO contract (`0x1000000000000000000000000000000000000003`) and submitter contract get gas fees returned to the caller
- Activation times and data prefixes are network-specific (see [core/daemon.go](core/daemon.go))
- Gas cap: 3,000,000 on Flare/Costwo/LocalFlare; unlimited on Songbird/Coston/Local

### State Connector
- Merkle attestation protocol; different contract addresses per network
- Activated at specific timestamps per chain

### Governance Contracts
- Update governance address, timelock, airdrop, and distribution contracts
- Specific to Flare and Costwo; uses coinbase value as a signal mechanism

### State Transition Flow
1. Standard EVM transaction execution
2. Post-execution (Flare-specific):
   - `handleFlareTransitionDbContracts` — governance, state connector, daemon
   - `handleSongbirdTransitionDbContracts` — state connector, daemon
   - Prioritized contract fee refund if applicable

## Accessing Extras in State Transition Code

```go
// Access Avalanche-specific rules
rulesExtra := params.GetRulesExtra(rules)

// Access chain config extras
configExtra := params.GetExtra(chainConfig)
```

## Network Upgrades

Block-based: Homestead, EIP150/155/158, Byzantium, Constantinople, Petersburg, Istanbul, MuirGlacier, Berlin, London

Timestamp-based (Avalanche/Flare): Apricot (phases 1-6), SongbirdTransition (Songbird-family only), Banff, Cortina, Durango, Etna, Fortuna, Granite, Helicon

Flare-specific timestamps differ from Avalanche mainnet — do not copy upgrade times from upstream. The authoritative per-network schedule is the root module's `upgrade/upgrade.go`.

## Known Upstream Differences

1. **Atomic transactions** — export transactions support signing with eth-style prefix, see `plugin/evm/atomic/vm/tx_semantic_verifier.go`
2. **No Fuji testnet** — removed due to network ID conflicts (Songbird uses network ID 5); tests that use `upgrade.Fuji` upstream are adapted to Mainnet/Flare configs
3. **libevm dependency** — uses the upstream published `github.com/ava-labs/libevm` directly (no local fork / no `replace` directive). The daemon-call logic that previously required a libevm fork lives in [core/daemon_call.go](core/daemon_call.go)
4. **big.Int validator weights** — `validators.WarpSet.TotalWeight` and warp signature weight sums are `*big.Int` in the Flare fork (token supply exceeds uint64); warp tests here construct weights accordingly
5. **Gossip semantics** — `gossip.Every` (root module) matches upstream (a non-positive frequency clamps to the default period). A prior Flare "disable on non-positive frequency" audit fix was reverted because it left the push-gossip queue unbounded when disabled; do not re-apply it on merge.
