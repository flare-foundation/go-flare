# Avalanchego (Flare Fork)

Fork of [ava-labs/avalanchego](https://github.com/ava-labs/avalanchego) with Flare and Songbird network support.
Module: `github.com/ava-labs/avalanchego` | Go 1.25.8 (see `go.mod` toolchain).

**Upstream version:** Pinned to avalanchego **v1.14.2** (branch `flare-merge-1_14_2`) — this is **not** the latest upstream release. Newer upstream tags exist; do not assume features/APIs from later versions are present here.

**Coreth is grafted into this repo.** Since v1.14.x upstream, coreth lives at `graft/coreth` (module `github.com/ava-labs/avalanchego/graft/coreth`), alongside `graft/evm` (shared EVM/sync code) and `graft/subnet-evm`. All are wired via `replace` directives in the root `go.mod` plus the `go.work` workspace, and are compiled directly into the `avalanchego` binary. There is no separate `evm` plugin binary and the build no longer copies anything into `GOPATH`.

> This is the comprehensive technical **and** upstream-merge guide for the node. The go-flare repo-root `CLAUDE.md` is only a short overview that points here. Paths below are relative to this `avalanchego/` directory.

## Build & Test

```bash
./scripts/build.sh                              # Build node (coreth included; binary at build/avalanchego)
go test $(go list ./... | grep -v /tests/)      # Root-module unit tests (also matches graft pkgs in workspace mode)
cd graft/coreth && go test ./...   # Coreth unit tests
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

## Coreth (`graft/coreth`) specifics

Coreth implements the C-Chain EVM for Flare/Songbird (module `github.com/ava-labs/avalanchego/graft/coreth`); the shared EVM/sync/triedb/rpc code upstream split out lives in the sibling `graft/evm` module.

**Chain IDs** (distinct from the network IDs above; defined in `graft/coreth/params/config.go`): Flare 14, Costwo 114, Songbird 19, Coston 16, LocalFlare 162, Local 4294967295.

**libevm:** consumed as the upstream-published `github.com/ava-labs/libevm` with **no local fork / no `replace`**. The daemon-call helper that previously required a libevm fork now lives in `graft/coreth/core/daemon_call.go` (`DaemonCall` = state snapshot + tracer-disabled `evm.Call`).

**Flare mechanisms** — applied post-EVM-execution in `graft/coreth/core/state_transition.go` via `handleFlareTransitionDbContracts` / `handleSongbirdTransitionDbContracts`:
- **Daemon contract (inflation)** at `0x1000000000000000000000000000000000000002` — called each block; returns a mint request (uint256); if valid and ≤ max, tokens are minted to the daemon contract. Max mint: 60M/block on Flare/Costwo/LocalFlare, 50M on Songbird/Coston.
- **Prioritized contracts (fee refund)** — the FTSO contract `0x1000000000000000000000000000000000000003` and the submitter contract get gas fees refunded to the caller. Gas cap: 3,000,000 on Flare/Costwo/LocalFlare; unlimited on Songbird/Coston/Local. Activation times and data prefixes are network-specific (see `graft/coreth/core/daemon.go`).
- **State Connector** (`graft/coreth/core/state_connector.go`) — Merkle attestation protocol; per-network contract addresses, activated at per-chain timestamps.
- **Governance** (`graft/coreth/core/governance_settings.go`) — updates the governance address/timelock and the airdrop/distribution contracts on Flare and Costwo, using the coinbase value as a signal.

**Accessing extras in coreth code:** `rulesExtra := params.GetRulesExtra(rules)` (Avalanche/Flare rules) and `configExtra := params.GetExtra(chainConfig)` (chain-config extras). `RulesExtra.IsSongbirdCode` / `extras.IsFlareFamilyCode` branch on network.

## Key Patterns

- **Network detection**: Use `constants.IsFlareNetworkID(networkID)` / `constants.IsSgbNetworkID(networkID)` to branch on network type.
- **Fork-time gating**: Check timestamps against fork times in `upgrade/upgrade.go` for time-dependent behavior.
- **Granite ACP-176 params**: Flare-family chains use `graft/coreth/plugin/evm/upgrade/granite.DefaultParams` after Granite (see `extras.ChainConfig.ACP176Params`).

## Merge Guide for Upstream Updates

This documents merging newer **ava-labs/avalanchego** upstream releases into this fork. The merge is performed in this avalanchego tree; since v1.14.x coreth is part of the avalanchego repository (`graft/coreth`), a single upstream merge brings the avalanchego **and** coreth changes — there is no longer a separate coreth merge.

### Architecture Overview (as of v1.14.2; coreth grafted at `graft/coreth`)

Coreth (`graft/coreth`), the shared EVM/sync code (`graft/evm`), and `graft/subnet-evm` are in-tree Go modules wired to the root avalanchego module via `replace` directives in `go.mod` and the `go.work` workspace. They compile directly into the single `avalanchego` binary (no separate `evm` plugin binary, no GOPATH copy).

### Merge Process

1. **Create a merge branch:**
   ```bash
   git checkout -b flare-merge-<version>
   ```

2. **Add upstream remote (if not already done):**
   ```bash
   git remote add upstream-ava https://github.com/ava-labs/avalanchego.git
   ```
   Coreth no longer has its own upstream remote — it ships inside avalanchego at `graft/coreth`.

3. **Fetch and merge:**
   ```bash
   git fetch upstream-ava && git merge v<version>
   ```
   This single merge also brings the `graft/coreth`, `graft/evm` and `graft/subnet-evm` changes.

4. **Resolve conflicts following the guidelines below**

5. **Run module hygiene:** `GOWORK=off go mod tidy` in each module (`.`, `graft/evm`, `graft/coreth`, `graft/subnet-evm`), then `go work sync`

6. **Build and test**

### Critical Flare-Specific Files

**Always preserve Flare's version of these files** (described in detail in the sections above):

In avalanchego:
- `genesis/*.json` - Flare/Songbird/Coston genesis files
- `utils/constants/network_ids.go` - Flare network IDs
- `vms/platformvm/txs/executor/staker_tx_verification.go` - Flare staking rules
- `vms/platformvm/reward/calculator.go` - Reward calculations (0 rewards)
- `vms/secp256k1fx/fx.go` - eth-style prefix support
- `upgrade/upgrade.go` - Flare-specific fork times
- `vms/evm/acp176/acp176.go` - Keep the extended `Params` struct (with `TimeToFillCapacity`, `TargetToMax`, `TargetToPriceUpdateConversion`) and the `*With` methods reading from `p.*` instead of package constants
- `scripts/git_commit.sh` - Flare patch: derives the build commit via `git -C "${AVALANCHE_PATH}"` discovery instead of upstream's hardcoded `--git-dir="${AVALANCHE_PATH}/.git"`. avalanchego is nested in the go-flare repo (the `.git` is at the repo root; there is no `avalanchego/.git`), so the upstream form makes `build.sh` fail with "not a git repository".
- Any file with "Flare", "Songbird", "Coston" specific code

In coreth (`graft/coreth/`):
- `graft/coreth/core/daemon.go` - Daemon contract calls (inflation minting); prioritized-contract detection (FTSO, submitter)
- `graft/coreth/core/daemon_call.go` - `DaemonCall` helper (snapshot + tracer-disabled `evm.Call`); replaces the method formerly added in the Flare libevm fork
- `graft/coreth/core/daemon_test.go` - Tests for daemon logic
- `graft/coreth/core/state_transition.go` - Integrates daemon/governance/state-connector; fee refunds for prioritized contracts
- `graft/coreth/core/governance_settings.go` - Governance address/timelock, airdrop and distribution contract management
- `graft/coreth/core/state_connector.go` - State Connector attestation protocol
- `graft/coreth/params/config.go` - Flare/Songbird chain IDs, `TestFlareChainConfig`
- `graft/coreth/params/config_libevm.go` - `RulesExtra.IsSongbirdCode` derivation from `SnowCtx.NetworkID`
- `graft/coreth/params/extras/config.go` - `SnowCtx` in `ChainConfig`, `IsSongbirdCode`/`IsFlareFamilyCode`, `ACP176Params()` (Granite params on Flare-family networks)
- `graft/coreth/params/extras/network_upgrades.go` - Flare-specific network upgrade timing, `SongbirdTransitionTimestamp`
- `graft/coreth/plugin/evm/atomic/export_tx.go` - Export tx logic
- `graft/coreth/plugin/evm/atomic/vm/tx_semantic_verifier.go` - Eth-style signature prefix support (`accounts.TextHash`) for export txs
- `graft/coreth/plugin/evm/atomic/vm/vm.go` - Atomic transaction functions (`verifyTxAtTip`, `verifyTx`, `verifyTxs`, `GetAtomicUTXOs`) — moved here from `plugin/evm/vm.go`
- `graft/coreth/plugin/evm/upgrade/sgbt/` - Songbird transition gas-limit parameters
- `graft/coreth/plugin/evm/upgrade/granite/params.go` - Flare-family ACP-176 parameter set; keep all seven fields in `DefaultParams`
- `graft/coreth/plugin/evm/customheader/gas_limit.go` - Songbird gas-limit schedule between AP1 and Cortina; `MinimumBuildableGasCapacity`
- `graft/coreth/miner/worker.go` - Keep the `w.config.WaitForGasCapacityRefill &&` guard in front of the `IsFortuna` capacity-wait block. The flag defaults to `true` following AvalancheGo upstream, and operators can set it `false` to build regardless of bucket state. The wait threshold is computed by `customheader.MinimumBuildableGasCapacity` (`min(4*target, 12M)`), not inline, so `worker.go` no longer imports `acp176`/`cortina`
- `graft/evm/sync/client/client.go` - `StateSyncVersionSgb` minimum version for Songbird state sync (lives in the `graft/evm` module)

### Common Import Path Changes

- Coreth packages: `github.com/ava-labs/coreth/...` → `github.com/ava-labs/avalanchego/graft/coreth/...`
- Shared EVM/sync/triedb/rpc code that upstream split out of coreth: now under `github.com/ava-labs/avalanchego/graft/evm/...`

### Resolving Specific Conflicts

**`graft/coreth/core/state_transition.go`:**
- Keep Flare's `DaemonCall` integration
- Keep prioritized contract fee handling
- Keep `handleFlareTransitionDbContracts` and `handleSongbirdTransitionDbContracts`
- Use `rulesExtra := params.GetRulesExtra(rules)` to access Avalanche-specific rules
- Use `configExtra := params.GetExtra(chainConfig)` to access chain config extras

**`graft/coreth/plugin/evm/vm.go`:**
- Keep Flare's atomic transaction functions (`verifyTxAtTip`, `verifyTx`, `verifyTxs`, `GetAtomicUTXOs`, `ParseAddress`)
- These are removed in upstream but still needed for Flare. Note: upstream relocated these atomic functions to `graft/coreth/plugin/evm/atomic/vm/vm.go` — keep Flare's versions there.

**Test files with JSON data (`graft/coreth/internal/ethapi/testdata/`):**
- The conflict is usually about transaction hashes, state roots, block hashes

### Post-Merge Checklist

1. [ ] Run module hygiene: `GOWORK=off go mod tidy` in each module (`.`, `graft/evm`, `graft/coreth`, `graft/subnet-evm`), then `go work sync`
2. [ ] Build: `./scripts/build.sh`
3. [ ] Run avalanchego tests: `go test $(go list ./... | grep -v /tests/)`
4. [ ] Run coreth tests: `cd graft/coreth && go test ./...`
5. [ ] Verify Flare-specific tests pass:
   - `TestStateTransitionPrioritizedContract`
   - Tests in `graft/coreth/core/daemon_test.go`
6. [ ] Update JSON test data hashes if needed

### Known Issues

- **Network binding tests:** Tests requiring `listen tcp 127.0.0.1:0` will fail in sandboxed environments. This is not a code issue.

- **`TestCheckCompatible`:** May fail due to error message changes. Update test expectations if the logic is correct.
