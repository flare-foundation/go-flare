# Avalanchego (Flare Fork)

Fork of [ava-labs/avalanchego](https://github.com/ava-labs/avalanchego) with Flare and Songbird network support.
Module: `github.com/ava-labs/avalanchego` | Go 1.25.8 (see `go.mod` toolchain) | See parent `../CLAUDE.md` for the full merge guide.

**Upstream version:** Pinned to avalanchego **v1.14.2** (branch `flare-merge-1_14_2`) — this is **not** the latest upstream release. Newer upstream tags exist; do not assume features/APIs from later versions are present here.

**Coreth is grafted into this repo.** Since v1.14.x upstream, coreth lives at `graft/coreth` (module `github.com/ava-labs/avalanchego/graft/coreth`), alongside `graft/evm` (shared EVM/sync code) and `graft/subnet-evm`. All are wired via `replace` directives in the root `go.mod` plus the `go.work` workspace, and are compiled directly into the `avalanchego` binary. There is no separate `evm` plugin binary and the build no longer copies anything into `GOPATH`. See `graft/coreth/CLAUDE.md` for coreth-specific details.

## Build & Test

```bash
./scripts/build.sh                              # Build node (coreth included; binary at build/avalanchego)
go test $(go list ./... | grep -v /tests/)      # Root-module unit tests (also matches graft pkgs in workspace mode)
cd graft/coreth && go test $(go list ./... | grep -v /tests/)   # Coreth unit tests
./scripts/localflare.sh                         # 5-node local network (ports 9650-9654)
./scripts/coston2.sh                            # Coston2 node(s)
```

Module hygiene after dependency changes (mirrors `Taskfile.yml` `go-mod-tidy`):

```bash
GOWORK=off go mod tidy                          # repo root
(cd tools/external && GOWORK=off go mod tidy)
(cd graft/evm && GOWORK=off go mod tidy)
(cd graft/coreth && GOWORK=off go mod tidy)
(cd graft/subnet-evm && GOWORK=off go mod tidy)
go work sync
```

**libevm pinning:** the root `go.mod` and all graft modules must require the **same** `github.com/ava-labs/libevm` version (currently `v1.13.14-0.4.0.rc.2`). The workspace selects the highest requirement; a newer pseudo-version in any module silently downgrades/upgrades everyone and breaks the ava-labs ABI helpers (`PackOutput`, `UnpackInputIntoInterface`, `PackEvent`).

**Protobuf:** generated code lives in `proto/pb`. Regenerate with `scripts/protobuf_codegen.sh` (pins buf 1.59.0, protoc-gen-go v1.36.10, protoc-gen-go-grpc 1.5.1), or `buf generate --path <dir>` from `proto/` with matching plugin versions.

## Directory Structure

| Directory | Purpose |
|-----------|---------|
| `main/` | Entry point → `app/` → `node/` |
| `genesis/` | Network genesis configs (JSON + Go loaders) |
| `upgrade/` | Fork times per network (incl. SongbirdTransition, Granite, Helicon) |
| `graft/coreth/` | C-Chain EVM (coreth) — Flare daemon/state-connector logic lives here |
| `graft/evm/` | Shared EVM module: sync client/handlers, triedb, rpc |
| `graft/subnet-evm/` | Subnet-EVM (upstream graft) |
| `vms/platformvm/` | P-chain: validators, staking, subnets |
| `vms/avm/` | X-chain: UTXO model |
| `vms/secp256k1fx/` | Signature verification (eth-style support) |
| `snow/` | Consensus engine (Snowman/Snowball) |
| `network/` | P2P networking |
| `utils/constants/` | Network IDs, HRPs, constants |
| `scripts/` | Build, test, local network scripts |

## Flare-Specific Modifications

### Network IDs (`utils/constants/network_ids.go`)
- Flare: 14, Songbird: 5, Costwo (Flare testnet): 114, Coston (Songbird testnet): 7, LocalFlare: 162, Local: 12345
- Helpers: `IsFlareNetworkID()`, `IsSgbNetworkID()`
- Avalanche Fuji was removed (network ID conflict with Songbird); upstream code/tests referencing `upgrade.Fuji` or `constants.FujiID` must be adapted (use Mainnet or Flare configs).

### Genesis Files (`genesis/`)
- `genesis_flare.go/.json`, `genesis_songbird.go`, `genesis_costwo.go/.json`, `genesis_coston.go`, `genesis_localFlare.go/.json`, `genesis_local.go`

### Fork Times (`upgrade/upgrade.go`)
- Per-network upgrade schedules (Apricot phases, SongbirdTransition, Banff, Cortina, Durango, Etna, Fortuna, Granite, Helicon)
- Every config must set **all** fork fields (the `upgrade` package validates ordering; unscheduled forks use `UnscheduledActivationTime`)
- `ZeroTime` is pinned to UTC so configs survive proto round-trips under deep equality
- `SongbirdTransitionTime` is carried over the plugin VM proto (`proto/vm/vm.proto` field 19, `vms/rpcchainvm/vm_client.go` + `vm_server.go`); `TestConvertNetworkUpgrades_AllFieldsHandled` guards this

### Staking Parameters (`vms/platformvm/txs/executor/inflation_settings.go`)
- Returns different validator/delegator settings based on network ID and timestamp (`getValidatorRules`/`getDelegatorRules` take a leading `time.Time`)
- All networks return **0 rewards** (rewards handled via smart contracts); see `vms/platformvm/reward/calculator.go`
- Verification logic in `staker_tx_verification.go`

### Eth-Style Signatures (`vms/secp256k1fx/fx.go`)
- `VerifyCredentials()` tries two verification methods: standard Avalanche (no prefix) and Ethereum-style (`accounts.TextHash`)
- Allows P-chain transactions signed with standard Ethereum tools

### big.Int Weight Calculations
- Validator weight aggregates use `*big.Int` instead of `uint64` to prevent overflow (Flare has more tokens than Avalanche):
  - `validators.Manager.TotalWeight/SubsetWeight` return `*big.Int`
  - `validators.WarpSet.TotalWeight` is `*big.Int`; the `validatorstate` proto encodes it as `bytes` (regenerate proto after changes)
  - Consumers (benchlist, chains/manager, syncer, tracker, snowman bootstrap) convert via `Float64()`/`IsUint64()` where ratios are needed
- Per-validator weights remain `uint64`

### Gossip (`network/p2p/gossip/gossip.go`)
- `Every()` matches upstream: a non-positive frequency clamps to the default period (`defaultRequestPeriod`, 1s). A prior Flare audit fix made it *disable* the loop instead (`return`), but that left the push-gossip queue (`PushGossiper.Add`) growing unbounded with nothing to drain it, so it was reverted to upstream. **Do not re-apply the disable behavior on merge.**

## Key Patterns

- **Network detection**: Use `constants.IsFlareNetworkID(networkID)` / `constants.IsSgbNetworkID(networkID)` to branch on network type.
- **Fork-time gating**: Check timestamps against fork times in `upgrade/upgrade.go` for time-dependent behavior.
- **Granite ACP-176 params**: Flare-family chains use `graft/coreth/plugin/evm/upgrade/granite.DefaultParams` after Granite (see `extras.ChainConfig.ACP176Params`).
