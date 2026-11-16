# Release Notes: Flare and Songbird networks

Here are listed specific changes to the code for the Flare and Songbird networks. For a comprehensive list of general changes, see [here](./avalanchego/RELEASES.md) for the AvalancheGo project — which, since v1.14.x, also contains Coreth (grafted in at `avalanchego/graft/coreth`).

## v1.15.2

This release updates the Flare and Songbird codebase to AvalancheGo v1.15.2.

The upgrade is optional but encouraged. It is compatible with v1.15.1 and does not schedule any new fork times.

**Update to this version only from v1.15.1, on a node that has already passed the Helicon activation.** v1.15.2 no longer contains the pre-Helicon C-chain VM and cannot continue from a database that was last used before Helicon activated on its network. A node that still runs v1.14.x or older, or that has been offline since before the activation, must first run v1.15.1 until its C-chain has bootstrapped past the Helicon activation, and only then be updated to v1.15.2. Alternatively, such a node can be started on v1.15.2 with an empty database and synced from scratch.

See the [AvalancheGo v1.15.2 release notes](./avalanchego/RELEASES.md) for the complete list of changes.

### Flare-specific changes

 * `--sybil-protection-enabled=false` is rejected on the Flare, Songbird, Coston and Coston2 networks, as it is on Avalanche mainnet; only local and custom networks can disable sybil protection.
 * Fixed bootstrapping the C-chain from genesis (an empty database with state sync disabled) on the Flare and Coston2 networks, where the first block was rejected.
 * The `proposervm.getProposerByEVMBlockHash` and `proposervm.getProposerByEVMBlockNumber` lookups hold the chain lock for a shorter time.

### Note:

 * The `eth-apis` C-chain config option, deprecated in v1.15.1, is removed. The node ignores it and logs a warning, as for any other unrecognized option. Set `apis` instead.
 * The `avax.getAtomicTxStatus` C-chain RPC is removed. Use `avax.getAtomicTx`, which fails for a transaction that has not been accepted.
 * C-chain metrics no longer include the `transition` prefix; for example, `avalanche_evm_transition_sae_last_executed_height` is now `avalanche_evm_sae_last_executed_height`. Dashboards and alerts that use the old names must be updated.
 * The C-chain config option `api-max-duration` again limits every call to the `/rpc` and `/ws` endpoints; since the Helicon activation it limited only `eth_call`.
 * The C-chain config option `api-max-blocks-per-request` is available again. It limits the number of blocks an `eth_getLogs` request can cover (default `0`, no limit).
 * The P-chain config option `pull-gossip-requests-per-validator` is removed. It has had no effect since v1.14.2 and is ignored if a config still sets it.
 * `platform.getCurrentValidators` returns the new fields `restakedValidationRewards` and `restakedDelegateeRewards` for auto-renewed validators. They are always 0 on the Flare and Songbird networks, where the P-chain pays no staking rewards.
 * The plugin (RPCChainVM) protocol version is updated to `47`.

## v1.15.1

This release updates the Flare and Songbird codebase to AvalancheGo v1.15.1 (Helicon upgrade). It supersedes the v1.15.1-coston2 pre-release.

The changes go into effect
 * November 3, 2026 at 12:00:00 UTC for the Coston2 network (already active, as scheduled by the v1.15.1-coston2 pre-release),
 * November 11, 2026 at 7:00:00 UTC for the Coston network,
 * November 11, 2026 at 8:00:00 UTC for the Songbird network,
 * November 11, 2026 at 14:00:00 UTC for the Flare network.

Nodes should be updated to this version before these times.

### Helicon upgrade

See the [AvalancheGo v1.15.0 and v1.15.1 release notes](./avalanchego/RELEASES.md) for the complete list of API, configuration and metrics changes.

### Flare-specific changes

 * A new read-only **P-chain gateway precompile** at `0x0300000000000000000000000000000000000000` lets C-chain contracts read accepted P-chain data during execution: the P-chain height, blocks, transactions with their inputs and outputs, and stake owners. See its [documentation](./avalanchego/graft/coreth/precompile/contracts/pchaingateway/README.md).
 * The daemon contract call, the burning of transaction fees and the fee refunds for prioritized contracts are preserved under the new asynchronous execution. The state connector handling, inactive since the Durango upgrade, is removed.
 * For the Coston2 network, the minimum validator staking duration is reduced from 60 days to 7 days and the minimum delegation duration from 14 days to 24 hours after the activation.

### Note:

 * After the activation, the C-chain ignores a number of configuration options and logs a warning for each of them, `eth-apis` is deprecated in favor of `apis`, and `state-sync-enabled` defaults to `true`. The `commit-interval` option must be left at its default (4096). See the [C-chain config reference](./avalanchego/vms/saevm/cchain/config.md).
 * The `admin`, `warp` and `personal` API namespaces, the `eth_accounts`, `eth_coinbase` and `eth_etherbase` RPCs and several `debug` RPCs are removed.
 * The C-chain config option `api-max-duration` accepts only a duration string (for example `"30s"`) after the activation; a numeric value (nanoseconds) is rejected and the node fails to start. Update the value before the activation, the string form is also accepted by the current VM.
 * The C-chain keeps its execution results in the chain data directory (`--chain-data-dir`, default `<data-dir>/chainData`), outside the node database. This directory must be on persistent storage together with the database directory; a node that loses it cannot start and has to be resynced. The Docker images now expose it as the `CHAIN_DATA_DIR` environment variable, defaulting to `/app/db/chainData` inside the database volume, so a mount at `/app/db` persists both.
 * Building from source now requires Go 1.26.10 or later.
 * The staking parameter flags (`--min-validator-stake`, `--max-validator-stake`, `--min-delegator-stake`, `--min-delegation-fee`, `--min-stake-duration`, `--helicon-min-stake-duration`, `--max-stake-duration`, `--uptime-requirement` and the `--stake-*` reward flags) are ignored on the Flare, Songbird, Coston and Coston2 networks, as they are on Avalanche mainnet; only local and custom networks honor them. The built-in values equal the previous defaults where they take effect, so nodes that did not set these flags behave as before. The primary network's staking limits are unchanged and remain those built into the node for each network.

## v1.14.2

This release updates the Flare and Songbird codebase to AvalancheGo v1.14.2.

The upgrade is optional but encouraged. It is backwards compatible with v1.14.0 (the minimum compatible version remains v1.14.0) and does not schedule any new fork times.

### Note:

 * Coreth and Subnet-EVM are now grafted into the AvalancheGo tree — there is no separate `coreth` module anymore. Go import paths change from `github.com/ava-labs/coreth/...` to `github.com/ava-labs/avalanchego/graft/coreth/...`, which affects downstream tools that import Coreth packages.
 * Building from source now requires Go 1.25.11 or later.

## v1.14.0

This release updates the Flare and Songbird codebase to AvalancheGo v1.14.0 (Granite upgrade).

The changes go into effect
 * June 11, 2026 at 12 PM UTC for the Coston network,
 * June 16, 2026 at 12 PM UTC for the Coston2 network,
 * July 7, 2026 at 12 PM UTC for the Songbird network,
 * July 14, 2026 at 12 PM UTC for the Flare network.

Nodes should be updated to this version before these times.

### Note:

 * The minimum C-chain gas price (base fee floor) is raised to 500 GWei.
 * For the Flare network, the maximum validator stake amount is increased from 200 million to 300 million FLR.
 * For the Flare and Coston2 networks, the minimum delegation fee for validators is set to 20% (previously 0%).
 * The deprecated top-level `rewardOwner` field has been removed from validator entries returned by `platform.getCurrentValidators`. It previously duplicated `validationRewardOwner`.

## v1.13.0

The changes go into effect
 * March 17, 2026 at 12 PM UTC for the Coston network,
 * March 24, 2026 at 12 PM UTC for the Coston2 network,
 * March 31, 2026 at 12 PM UTC for the Songbird network,
 * April 14, 2026 at 12 PM UTC for the Flare network.

## v1.12.0

The changes go into effect
 * November 13, 2025 at 10 AM UTC for the Coston network,
 * November 13, 2025 at 2 PM UTC for the Coston2 network,
 * November 25, 2025 at 12 PM UTC for the Songbird network,
 * December 2, 2025 at 12 PM UTC for the Flare network.

### Note:

 * In contrast to Avalanche, the minimum C-chain base fee after the fork remains at 25 GWei.
 * Minimum price per unit of gas used in P-chain [dynamic transaction fee calculation](https://github.com/avalanche-foundation/ACPs/blob/main/ACPs/103-dynamic-fees) is set to 250 nFLR.

## v1.11.0

The changes go into effect
 * June 24, 2025 at 12 PM UTC for the Coston2 network,
 * July 1, 2025 at 12 PM UTC for the Coston network,
 * July 22, 2025 at 12 PM UTC for the Songbird network,
 * August 5, 2025 at 12 PM UTC for the Flare network.

### Note:

- Avalanche added in v1.10.3 a new config `--http-allowed-hosts` with a default value of `localhost`. Set `--http-allowed-hosts="*"` to allow RPC calls for all hosts.

## v1.10.0

The changes go into effect on May 6, 2025 at 12 PM UTC for the Songbird network and on May 13, 2025 at 12 PM UTC for the Flare network.

### Specific changes:

- Return delegators in `platform.getCurrentValidators` when the environment variable `COMPLETE_GET_VALIDATORS` is set to `true`. (Only the number of delegators is returned by default unless only one node is specified.)

- `AddPermissionlessValidatorTx` and `AddPermissionlessDelegatorTx` transactions are enabled for the primary network (they were disabled in 1.9.0) and are the preferred way to add delegators and validators as we plan to disable `AddValidatorTx` and `AddDelegatorTx` transactions in v1.11.0 to align with Avalanche.

- Reduce the target gas (for the 10-second rolling window used for dynamic fee calculation) to 15,000,000 from 150,000,000 for the Songbird network to be consistent with the Flare network.

### Note:

- We recommend using `AddPermissionlessValidatorTx` and `AddPermissionlessDelegatorTx` transactions for adding validators and delegators after the fork. When creating `AddPermissionlessValidatorTx` you need to specify the BLS proof of posession (BLS public key and signature). You can find the BLS proof of posession in the response of `info.getNodeID` API. Path to the BLS private key is specified with `--staking-signer-key-file`. The default is `~/.avalanchego/staking/signer.key`. If it does not exist, it will be populated with a new key. You should copy it to your persistent storage along with the existing `staker.crt` and `staker.key` files.

## v1.9.0

### Specific changes:

- `AddPermissionlessValidatorTx` and `AddPermissionlessDelegatorTx` transactions are disabled.

- Subnets are disabled.

- We allow Ethereum-style signatures for P-chain transactions ([EIP-191](https://eips.ethereum.org/EIPS/eip-191)).
