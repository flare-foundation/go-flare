# Release Notes

## [v1.9.0](https://github.com/ava-labs/avalanchego/releases/tag/v1.9.0)

This upgrade adds support for creating Proof-of-Stake Subnets.

This version is not backwards compatible. The changes in the upgrade go into effect at 12 PM EDT, October 18th 2022 on Mainnet.

**All Mainnet nodes should upgrade before 12 PM EDT, October 18th 2022.**

The supported plugin version is `17`.

### Upgrades

- Activated P2P serialization format change to Protobuf
- Activated non-AVAX `ImportTx`/`ExportTx`s to/from the P-chain
- Activated `Banff*` blocks on the P-chain
- Deactivated `Apricot*` blocks on the P-chain
- Activated `RemoveSubnetValidatorTx`s on the P-chain
- Activated `TransformSubnetTx`s on the P-chain
- Activated `AddPermissionlessValidatorTx`s on the P-chain
- Activated `AddPermissionlessDelegatorTx`s on the P-chain
- Deactivated ANT `ImportTx`/`ExportTx`s on the C-chain
- Deactivated ANT precompiles on the C-chain

### Deprecations

- Ubuntu 18.04 releases are deprecated and will not be provided for `>=v1.9.1`

### Miscellaneous

- Fixed locked input signing in the P-chain wallet
- Removed assertions from the logger interface
- Removed `--assertions-enabled` flag
- Fixed typo in `--bootstrap-max-time-get-ancestors` flag
- Standardized exported P-Chain codec usage
- Improved isolation and execution of the E2E tests
- Updated the linked hashmap implementation to use generics

## [v1.8.6](https://github.com/ava-labs/avalanchego/releases/tag/v1.8.6)

This version is backwards compatible to [v1.8.0](https://github.com/ava-labs/avalanchego/releases/tag/v1.8.0). It is optional, but encouraged. The supported plugin version is `16`.

### BLS

- Added BLS key file at `--staking-signer-key-file`
- Exposed BLS proof of possession in the `info.getNodeID` API
- Added BLS proof of possession to `AddPermissionlessValidatorTx`s for the Primary Network

The default value of `--staking-signer-key-file` is `~/.avalanchego/staking/signer.key`. If the key file doesn't exist, it will be populated with a new key.

### Networking

- Added P2P proto support to be activated in a future release
- Fixed inbound bandwidth spike after leaving the validation set
- Removed support for `ChitsV2` messages
- Removed `ContainerID`s from `Put` and `PushQuery` messages
- Added `pending_timeouts` metric to track the number of active timeouts a node is tracking
- Fixed overflow in gzip decompression
- Optimized memory usage in `peer.MessageQueue`

### Miscellaneous

- Fixed bootstrapping ETA metric
- Removed unused `unknown_txs_count` metric
- Replaced duplicated code with generic implementations

### Coreth

- Added failure reason to bad block API

## [v1.8.5](https://github.com/ava-labs/avalanchego/releases/tag/v1.8.5)

Please upgrade your node as soon as possible.

The supported plugin version is `16`.

### Fixes

- Fixed stale block reference by evicting blocks upon successful verification

### [Coreth](https://medium.com/avalancheavax/apricot-phase-6-native-asset-call-deprecation-a7b7a77b850a)

- Removed check for Apricot Phase6 incompatible fork to unblock nodes that did not upgrade ahead of the activation time

## [v1.8.4](https://github.com/ava-labs/avalanchego/releases/tag/v1.8.4)

Please upgrade your node as soon as possible.

The supported plugin version is `16`.

### Caching

- Added temporarily invalid block caching to reduce repeated network requests
- Added caching to the proposervm's inner block parsing

### [Coreth](https://medium.com/avalancheavax/apricot-phase-6-native-asset-call-deprecation-a7b7a77b850a)

- Reduced the log level of `BAD BLOCK`s from `ERROR` to `DEBUG`
- Deprecated Native Asset Call

## [v1.8.2](https://github.com/ava-labs/avalanchego/releases/tag/v1.8.2)

Please upgrade your node as soon as possible.

The changes in `v1.8.x` go into effect at 4 PM EDT on September 6th, 2022 on both Fuji and Mainnet. You should upgrade your node before the changes go into effect, otherwise they may experience loss of uptime.

The supported plugin version is `16`.

### [Coreth](https://medium.com/avalancheavax/apricot-phase-6-native-asset-call-deprecation-a7b7a77b850a)

- Fixed live-lock in bootstrapping, after performing state-sync, by properly reporting `database.ErrNotFound` in `GetBlockIDAtHeight` rather than a formatted error
- Increased the log level of `BAD BLOCK`s from `DEBUG` to `ERROR`
- Fixed typo in Chain Config `String` function

## [v1.8.1](https://github.com/ava-labs/avalanchego/releases/tag/v1.8.1)

Please upgrade your node as soon as possible.

The changes in `v1.8.x` go into effect at 4 PM EDT on September 6th, 2022 on both Fuji and Mainnet. You should upgrade your node before the changes go into effect, otherwise they may experience loss of uptime.

The supported plugin version is `16`.

### Miscellaneous

- Reduced the severity of not quickly connecting to bootstrap nodes from `FATAL` to `WARN`

### [Coreth](https://medium.com/avalancheavax/apricot-phase-6-native-asset-call-deprecation-a7b7a77b850a)

- Reduced the log level of `BAD BLOCK`s from `ERROR` to `DEBUG`
- Added Apricot Phase6 to Chain Config `String` function

## [v1.8.0](https://github.com/ava-labs/avalanchego/releases/tag/v1.8.0)

This is a mandatory security upgrade. Please upgrade your node **as soon as possible.**

The changes in the upgrade go into effect at **4 PM EDT on September 6th, 2022** on both Fuji and Mainnet. You should upgrade your node before the changes go into effect, otherwise they may experience loss of uptime.

You may see some extraneous ERROR logs ("BAD BLOCK") on your node after upgrading. These may continue until the Apricot Phase 6 activation (at 4 PM EDT on September 6th).

The supported plugin version is `16`.

### PlatformVM APIs

- Fixed `GetBlock` API when requesting the encoding as `json`
- Changed the json key in `AddSubnetValidatorTx`s from `subnet` to `subnetID`
- Added multiple asset support to `getBalance`
- Updated `PermissionlessValidator`s returned from `getCurrentValidators` and `getPendingValidators` to include `validationRewardOwner` and `delegationRewardOwner`
- Deprecated `rewardOwner` in `PermissionlessValidator`s returned from `getCurrentValidators` and `getPendingValidators`
- Added `subnetID` argument to `getCurrentSupply`
- Added multiple asset support to `getStake`
- Added `subnetID` argument to `getMinStake`

### PlatformVM Structures

- Renamed existing blocks
  - `ProposalBlock` -> `ApricotProposalBlock`
  - `AbortBlock` -> `ApricotAbortBlock`
  - `CommitBlock` -> `ApricotCommitBlock`
  - `StandardBlock` -> `ApricotStandardBlock`
  - `AtomicBlock` -> `ApricotAtomicBlock`
- Added new block types **to be enabled in a future release**
  - `BlueberryProposalBlock`
    - Introduces a `Time` field and an unused `Txs` field before the remaining `ApricotProposalBlock` fields
  - `BlueberryAbortBlock`
    - Introduces a `Time` field before the remaining `ApricotAbortBlock` fields
  - `BlueberryCommitBlock`
    - Introduces a `Time` field before the remaining `ApricotCommitBlock` fields
  - `BlueberryStandardBlock`
    - Introduces a `Time` field before the remaining `ApricotStandardBlock` fields
- Added new transaction types **to be enabled in a future release**
  - `RemoveSubnetValidatorTx`
    - Can be included into `BlueberryStandardBlock`s
    - Allows a subnet owner to remove a validator from their subnet
  - `TransformSubnetTx`
    - Can be included into `BlueberryStandardBlock`s
    - Allows a subnet owner to convert their subnet into a permissionless subnet
  - `AddPermissionlessValidatorTx`
    - Can be included into `BlueberryStandardBlock`s
    - Adds a new validator to the requested permissionless subnet
  - `AddPermissionlessDelegatorTx`
    - Can be included into `BlueberryStandardBlock`s
    - Adds a new delegator to the requested permissionless validator on the requested subnet

### PlatformVM Block Building

- Fixed race in `AdvanceTimeTx` creation to avoid unnecessary block construction
- Added `block_formation_logic.md` to describe how blocks are created
- Refactored `BlockBuilder` into `ApricotBlockBuilder`
- Added `BlueberryBlockBuilder`
- Added `OptionBlock` builder visitor
- Refactored `Mempool` issuance and removal logic to use transaction visitors

### PlatformVM Block Execution

- Added support for executing `AddValidatorTx`, `AddDelegatorTx`, and `AddSubnetValidatorTx` inside of a `BlueberryStandardBlock`
- Refactored time advancement into a standard state modification structure
- Refactored `ProposalTxExecutor` to abstract state diff creation
- Standardized upgrade checking rules
- Refactored subnet authorization checking

### Wallet

- Added support for new transaction types in the P-chain wallet
- Fixed fee amounts used in the Primary Network wallet to reduce unnecessary fee burning

### Networking

- Defined `p2p.proto` to be used for future network messages
- Added `--network-tls-key-log-file-unsafe` to support inspecting p2p messages
- Added `avalanche_network_accept_failed` metrics to track networking `Accept` errors

### Miscellaneous

- Removed reserved fields from proto files and renumbered the existing fields
- Added generic dynamically resized ring buffer
- Updated gRPC version to `v1.49.0` to fix non-deterministic errors reported in the `rpcchainvm`
- Removed `--signature-verification-enabled` flag
- Removed dead code
  - `ids.QueueSet`
  - `timer.Repeater`
  - `timer.NewStagedTimer`
  - `timer.TimedMeter`

### [Coreth](https://medium.com/avalancheavax/apricot-phase-6-native-asset-call-deprecation-a7b7a77b850a)

- Incorrectly deprecated Native Asset Call
- Migrated to go-ethereum v1.10.23
- Added API to fetch Chain Config

## [v1.7.18](https://github.com/ava-labs/avalanchego/releases/tag/v1.7.18)

This version is backwards compatible to [v1.7.0](https://github.com/ava-labs/avalanchego/releases/tag/v1.7.0). It is optional, but encouraged. The supported plugin version is `15`.

### Fixes

- Fixed bug in `codeToFetch` database accessors that caused an error when starting/stopping state sync
- Fixed rare BAD BLOCK errors during C-chain bootstrapping
- Fixed platformvm `couldn't get preferred block state` log due to attempted block building during bootstrapping
- Fixed platformvm `failed to fetch next staker to reward` error log due to an incorrect `lastAcceptedID` reference
- Fixed AWS AMI creation

### PlatformVM

- Refactored platformvm metrics handling
- Refactored platformvm block creation
- Introduced support to prevent empty nodeID use on the P-chain to be activated in a future upgrade

### Coreth

- Updated gas price estimation to limit lookback window based on block timestamps
- Added metrics for processed/accepted gas
- Simplified syntactic block verification
- Ensured statedb errors during block processing are logged
- Removed deprecated gossiper/block building logic from pre-Apricot Phase 4
- Added marshal function for duration to improve config output

### Miscellaneous

- Updated local network genesis to use a newer start time
- Updated minimum golang version to go1.18.1
- Removed support for RocksDB
- Bumped go-ethereum version to v1.10.21
- Added various additional tests
- Introduced additional database invariants for all database implementations
- Added retries to windows CI installations
- Removed useless ID aliasing during chain creation

## [v1.7.17](https://github.com/ava-labs/avalanchego/releases/tag/v1.7.17)

This version is backwards compatible to [v1.7.0](https://github.com/ava-labs/avalanchego/releases/tag/v1.7.0). It is optional, but encouraged. The supported plugin version is `15`.

### VMs

- Refactored P-chain block state management
  - Supporting easier parsing and usage of blocks
  - Improving separation of block execution with block definition
  - Unifying state definitions
- Introduced support to send custom X-chain assets to the P-chain to be activated in a future upgrade
- Introduced support to use custom assets on the P-chain to be activated in a future upgrade
- Added VMs README to begin fully documenting plugin invariants
- Added various comments around expected usages of VM tools

### Coreth

- Added optional JSON logging
- Added interface for supporting stateful precompiles
- Removed legacy code format from the database

### Fixes

- Fixed ungraceful gRPC connection closure during very long running requests
- Fixed LevelDB panic during shutdown
- Fixed verification of `--stake-max-consumption-rate` to include the upper-bound
- Fixed various CI failures
- Fixed flaky unit tests

### Miscellaneous

- Added bootstrapping ETA metrics
- Converted all logs to support structured fields
- Improved Snowman++ oracle block verification error messages
- Removed deprecated or unused scripts

## [v1.7.16](https://github.com/ava-labs/avalanchego/releases/tag/v1.7.16)

This version is backwards compatible to [v1.7.0](https://github.com/ava-labs/avalanchego/releases/tag/v1.7.0). It is optional, but encouraged. The supported plugin version is `15`.

### LevelDB

- Fix rapid disk growth by manually specifying the maximum manifest file size

## [v1.7.15](https://github.com/ava-labs/avalanchego/releases/tag/v1.7.15)

This version is backwards compatible to [v1.7.0](https://github.com/ava-labs/avalanchego/releases/tag/v1.7.0). It is optional, but encouraged. The supported plugin version is `15`.

### PlatformVM

- Replaced copy-on-write validator set data-structure to use tree diffs to optimize validator set additions
- Replaced validation transactions with a standardized representation to remove transaction type handling
- Migrated transaction execution to its own package
- Removed child pointers from processing blocks
- Added P-chain wallet helper for providing initial transactions

### Coreth

- Bumped go-ethereum dependency to v1.10.20
- Updated API names used to enable services in `eth-api` config flag. Prior names are supported but deprecated, please update configurations [accordingly](https://docs.avax.network/nodes/maintain/chain-config-flags#c-chain-configs)
- Optimized state sync by parallelizing trie syncing
- Added `eth_syncing` API for compatibility. Note: This API is only accessible after bootstrapping and always returns `"false"`, since the node will no longer be syncing at that point
- Added metrics to the atomic transaction mempool
- Added metrics for incoming/outgoing mempool gossip

### Fixes

- Updated Snowman and Avalanche consensus engines to report original container preferences before processing the provided container
- Fixed inbound message byte throttler context cancellation cleanup
- Removed case sensitivity of IP resolver services
- Added failing health check when a whitelisted subnet fails to initialize a chain

### Miscellaneous

- Added gRPC client metrics for dynamically created connections
- Added uninitialized continuous time averager for when initial predictions are unreliable
- Updated linter version
- Documented various platform invariants
- Cleaned up various dead parameters
- Improved various tests

## [v1.7.14](https://github.com/ava-labs/avalanchego/releases/tag/v1.7.14)

This version is backwards compatible to [v1.7.0](https://github.com/ava-labs/avalanchego/releases/tag/v1.7.0). It is optional, but encouraged.

### APIs

**These API format changes are breaking changes. https://api.avax.network and https://api.avax-test.network have been updated with this format. If you are using AvalancheGo APIs in your code, please ensure you have updated to the latest versions. See  https://docs.avax.network/apis/avalanchego/cb58-deprecation for details about the CB58 removal.**

- Removed `CB58` as an encoding option from all APIs
- Added `HexC` and `HexNC` as encoding options for all APIs that accept an encoding format
- Removed the `Success` response from all APIs
- Replaced `containerID` with `id` in the indexer API

### PlatformVM

- Fixed incorrect `P-chain` height in `Snowman++` when staking is disabled
- Moved `platformvm` transactions to be defined in a sub-package
- Moved `platformvm` genesis management to be defined in a sub-package
- Moved `platformvm` state to be defined in a sub-package
- Standardized `platformvm` transactions to always be referenced via pointer
- Moved the `platformvm` transaction builder to be defined in a sub-package
- Fixed uptime rounding during node shutdown

### Coreth

- Bumped go-ethereum dependency to v1.10.18
- Parallelized state sync code fetching

### Networking

- Updated `Connected` and `Disconnected` messages to only be sent to chains if the peer is tracking the subnet
- Updated the minimum TLS version on the p2p network to `v1.3`
- Supported context cancellation in the networking rate limiters
- Added `ChitsV2` message format for the p2p network to be used in a future upgrade

### Miscellaneous

- Fixed `--public-ip-resolution-frequency` invalid overwrite of the resolution service
- Added additional metrics to distinguish between virtuous and rogue currently processing transactions
- Suppressed the super cool `avalanchego` banner when `stdout` is not directed to a terminal
- Updated linter version
- Improved various comments and documentation
- Standardized primary network handling across subnet maps

## [v1.7.13](https://github.com/ava-labs/avalanchego/releases/tag/v1.7.13)

This version is backwards compatible to [v1.7.0](https://github.com/ava-labs/avalanchego/releases/tag/v1.7.0). It is optional, but encouraged.

### State Sync

- Added peer bandwidth tracking to optimize `coreth` state sync message routing
- Fixed `coreth` leaf request handler bug to ensure the handler delivers a valid range proof
- Removed redundant proof keys from `coreth` leafs response message format
- Improved `coreth` state sync request retry logic
- Improved `coreth` state sync handler metrics
- Improved `coreth` state sync ETA
- Added `avalanche_{chainID}_handler_async_expired` metric

### Miscellaneous

- Fixed `platform.getCurrentValidators` API to correctly mark a node as connected to itself on subnets.
- Fixed `platform.getBlockchainStatus` to correctly report `Unknown` for blockchains that are not managed by the `P-Chain`
- Added process metrics by default in the `rpcchainvm#Server`
- Added `Database` health checks
- Removed the deprecated `Database.Stat` call from the `rpcdb#Server`
- Added fail fast logic to duplicated Snowman additions to avoid undefined behavior
- Added additional testing around Snowman diverged voting tests
- Deprecated `--dynamic-update-duration` and `--dynamic-public-ip` CLI flags
- Added `--public-ip-resolution-frequency` and `--public-ip-resolution-service` to replace `--dynamic-update-duration` and `--dynamic-public-ip`, respectively

## [v1.7.12](https://github.com/ava-labs/avalanchego/releases/tag/v1.7.12)

This version is backwards compatible to [v1.7.0](https://github.com/ava-labs/avalanchego/releases/tag/v1.7.0). It is optional, but encouraged.

### State Sync

- Fixed proposervm state summary acceptance to only accept state summaries with heights higher than the locally last accepted block
- Fixed proposervm state summary serving to only respond to requests after height indexing has finished
- Improved C-chain state sync leaf request serving by optimistically reading leaves from snapshot
- Refactored C-chain state sync block fetching

### Networking

- Reduced default peerlist and accepted frontier gossipping
- Increased the default at-large outbound buffer size to 32 MiB

### Metrics

- Added leveldb metrics
- Added process and golang metrics for the avalanchego binary
- Added available disk space health check
  - Ensured that the disk space will not be fully utilized by shutting down the node if there is a critically low amount of free space remaining
- Improved C-chain state sync metrics

### Performance

- Added C-chain acceptor queue within `core/blockchain.go`
- Removed rpcdb locking when committing batches and using iterators
- Capped C-chain TrieDB dirties cache size during block acceptance to reduce commit size at 4096 block interval

### Cleanup

- Refactored the avm to utilize the external txs package
- Unified platformvm dropped tx handling
- Clarified snowman child block acceptance calls
- Fixed small consensus typos
- Reduced minor duplicated code in consensus
- Moved the platformvm key factory out of the VM into the test file
- Removed unused return values from the timeout manager
- Removed weird json rpc private interface
- Standardized json imports
- Added vm factory interface checks

## [v1.7.11](https://github.com/ava-labs/avalanchego/releases/tag/v1.7.11)

This version is backwards compatible to [v1.7.0](https://github.com/ava-labs/avalanchego/releases/tag/v1.7.0). It is optional, but encouraged.

**The first startup of the C-Chain will cause an increase in CPU and IO usage due to an index update. This index update runs in the background and does not impact restart time.**

### State Sync

- Added state syncer engine to facilitate VM state syncing, rather than full historical syncing
- Added `GetStateSummaryFrontier`, `StateSummaryFrontier`, `GetAcceptedStateSummary`, `AcceptedStateSummary` as P2P messages
- Updated `Ancestors` message specification to expect an empty response if the container is unknown
- Added `--state-sync-ips` and `--state-sync-ids` flags to allow manual overrides of which nodes to query for accepted state summaries
- Updated networking library to permanently track all manually tracked peers, rather than just beacons
- Added state sync support to the `metervm`
- Added state sync support to the `proposervm`
- Added state sync support to the `rpcchainvm`
- Added beta state sync support to `coreth`

### ProposerVM

- Prevented rejected blocks from overwriting the `proposervm` height index
- Optimized `proposervm` block rewind to utilize the height index if available
- Ensured `proposervm` height index is marked as repaired in `Initialize` if it is fully repaired on startup
- Removed `--reset-proposervm-height-index`. The height index will be reset upon first restart
- Optimized `proposervm` height index resetting to periodically flush deletions

### Bug Fixes

- Fixed IPC message issuance and restructured consensus event callbacks to be checked at compile time
- Fixed `coreth` metrics initialization
- Fixed bootstrapping startup logic to correctly startup if initially connected to enough stake
- Fixed `coreth` panic during metrics collection
- Fixed panic on concurrent map read/write in P-chain wallet SDK
- Fixed `rpcchainvm` panic by sanitizing http response codes
- Fixed incorrect JSON tag on `platformvm.BaseTx`
- Fixed `AppRequest`, `AppResponse`, and `AppGossip` stringers used in logging

### API/Client

- Supported client implementations pointing to non-standard URIs
- Introduced `ids.NodeID` type to standardize logging and simplify API service and client implementations
- Changed client implementations to use standard types rather than `string`s wherever possible
- Added `subnetID` as an argument to `platform.getTotalStake`
- Added `connected` to the subnet validators in responses to `platform.getCurrentValidators` and `platform.getPendingValidators`
- Add missing `admin` API client methods
- Improved `indexer` API client implementation to avoid encoding edge cases

### Networking

- Added `--snow-mixed-query-num-push-vdr` and `--snow-mixed-query-num-push-non-vdr` to allow parameterization of sending push queries
  - By default, non-validators now send only pull queries, not push queries.
  - By default, validators now send both pull queries and push queries upon inserting a container into consensus. Previously, nodes sent only push queries.
- Added metrics to track the amount of over gossipping of `peerlist` messages
- Added custom message queueing support to outbound `Peer` messages
- Reused `Ping` messages to avoid needless memory allocations

### Logging

- Replaced AvalancheGo's internal logger with [uber-go/zap](https://github.com/uber-go/zap).
- Replaced AvalancheGo's log rotation with [lumberjack](https://github.com/natefinch/lumberjack).
- Renamed `log-display-highlight` to `log-format` and added `json` option.
- Added `log-rotater-max-size`, `log-rotater-max-files`, `log-rotater-max-age`, `log-rotater-compress-enabled` options for log rotation.

### Miscellaneous

- Added `--data-dir` flag to easily move all default file locations to a custom location
- Standardized RPC specification of timestamp fields
- Logged health checks whenever a failing health check is queried
- Added callback support for the validator set manager
- Increased `coreth` trie tip buffer size to 32
- Added CPU usage metrics for AvalancheGo and all sub-processes
- Added Disk IO usage metrics for AvalancheGo and all sub-processes

### Cleanup

- Refactored easily separable `platformvm` files into separate smaller packages
- Simplified default version parsing
- Fixed various typos
- Converted some structs to interfaces to better support mocked testing
- Refactored IP utils

### Documentation

- Increased recommended disk size to 1 TB
- Updated issue template
- Documented additional `snowman.Block` invariants

## [v1.7.10](https://github.com/ava-labs/avalanchego/releases/tag/v1.7.10)

This version is backwards compatible to [v1.7.0](https://github.com/ava-labs/avalanchego/releases/tag/v1.7.0). It is optional, but encouraged.

### Networking

- Improved vertex and block gossiping for validators with low stake weight.
- Added peers metric by subnet.
- Added percentage of stake connected metric by subnet.

### APIs

- Added support for specifying additional headers and query params in the RPC client implementations.
- Added static API clients for the `platformvm` and the `avm`.

### PlatformVM

- Introduced time based windowing of accepted P-chain block heights to ensure that local networks update the proposer list timely in the `proposervm`.
- Improved selection of decision transactions from the mempool.

### RPCChainVM

- Increased `buf` version to `v1.3.1`.
- Migrated all proto definitions to a dedicated `/proto` folder.
- Removed the dependency on the non-standard grpc broker to better support other language implementations.
- Added grpc metrics.
- Added grpc server health checks.

### Coreth

- Fixed a bug where a deadlock on shutdown caused historical re-generation on restart.
- Added an API endpoint to fetch the current VM Config.
- Added AvalancheGo custom log formatting to the logs.
- Removed support for the JS Tracer.

### Logging

- Added piping of subnet logs to stdout.
- Lazily initialized logs to avoid opening files that are never written to.
- Added support for arbitrarily deleted log files while avalanchego is running.
- Removed redundant logging configs.

### Miscellaneous

- Updated minimum go version to `v1.17.9`.
- Added subnet bootstrapping health checks.
- Supported multiple tags per codec instantiation.
- Added minor fail-fast optimization to string packing.
- Removed dead code.
- Fixed typos.
- Simplified consensus engine `Shutdown` notification dispatching.
- Removed `Sleep` call in the inbound connection throttler.

## [v1.7.9](https://github.com/ava-labs/avalanchego/releases/tag/v1.7.9)

This version is backwards compatible to [v1.7.0](https://github.com/ava-labs/avalanchego/releases/tag/v1.7.0). It is optional, but encouraged.

### Updates

- Improved subnet gossip to only send messages to nodes participating in that subnet.
- Fixed inlined VM initialization to correctly register static APIs.
- Added logging for file descriptor limit errors.
- Removed dead code from network packer.
- Improved logging of invalid hash length errors.

## [v1.7.8](https://github.com/ava-labs/avalanchego/releases/tag/v1.7.8)

This version is backwards compatible to [v1.7.0](https://github.com/ava-labs/avalanchego/releases/tag/v1.7.0). It is optional, but encouraged.

### Networking

- Fixed duplicate reference decrease when closing a peer.
- Freed allocated message buffers immediately after sending.
- Added `--network-peer-read-buffer-size` and `--network-peer-write-buffer-size` config options.
- Moved peer IP signature verification to enable concurrent verifications.
- Reduced the number of connection flushes when sending messages.
- Canceled outbound connection requests on shutdown.
- Reused dialer across multiple outbound connections.
- Exported `NewTestNetwork` for easier external testing.

### Coreth

- Reduced log level of snapshot regeneration logs.
- Enabled atomic tx replacement with higher gas fees.
- Parallelized trie index re-generation.

### Miscellaneous

- Fixed incorrect `BlockchainID` usage in the X-chain `ImportTx` builder.
- Fixed incorrect `OutputOwners` in the P-chain `ImportTx` builder.
- Improved FD limit error logging and warnings.
- Rounded bootstrapping ETAs to the nearest second.
- Added gossip config support to the subnet configs.
- Optimized various queue removals for improved memory freeing.
- Added a basic X-chain E2E usage test to the new testing framework.

## [v1.7.7](https://github.com/ava-labs/avalanchego/releases/tag/v1.7.7)

This version is backwards compatible to [v1.7.0](https://github.com/ava-labs/avalanchego/releases/tag/v1.7.0). It is optional, but encouraged.

### Networking

- Refactored the networking library to track potential peers by nodeID rather than IP.
- Separated peer connections from the mesh network implementation to simplify testing.
- Fixed duplicate `Connected` messages bug.
- Supported establishing outbound connections with peers reporting different inbound and outbound IPs.

### Database

- Disabled seek compaction in leveldb by default.

### GRPC

- Increased protocol version, this requires all plugin definitions to update their communication dependencies.
- Merged services to be served using the same server when possible.
- Implemented a fast path for simple HTTP requests.
- Removed duplicated message definitions.
- Improved error reporting around invalid plugins.

### Coreth

- Optimized FeeHistory API.
- Added protection to prevent accidental corruption of archival node trie index.
- Added capability to restore complete trie index on best effort basis.
- Rounded up fastcache sizes to utilize all mmap'd memory in chunks of 64MB.

### Configs

- Removed `--inbound-connection-throttling-max-recent`
- Renamed `--network-peer-list-size` to `--network-peer-list-num-validator-ips`
- Removed `--network-peer-list-gossip-size`
- Removed `--network-peer-list-staker-gossip-fraction`
- Added `--network-peer-list-validator-gossip-size`
- Added `--network-peer-list-non-validator-gossip-size`
- Removed `--network-get-version-timeout`
- Removed `--benchlist-peer-summary-enabled`
- Removed `--peer-alias-timeout`

### Miscellaneous

- Fixed error reporting when making Avalanche chains that did not manually specify a primary alias.
- Added beacon utils for easier programmatic handling of beacon nodes.
- Resolved the default log directory on initialization to avoid additional error handling.
- Added support to the chain state module to specify an arbitrary new accepted block.

## [v1.7.6](https://github.com/ava-labs/avalanchego/releases/tag/v1.7.6)

This version is backwards compatible to [v1.7.0](https://github.com/ava-labs/avalanchego/releases/tag/v1.7.0). It is optional, but encouraged.

### Consensus

- Introduced a new vertex type to support future `Avalanche` based network upgrades.
- Added pending message metrics to the chain message queues.
- Refactored event dispatchers to simplify dependencies and remove dead code.

### PlatformVM

- Added `json` encoding option to the `platform.getTx` call.
- Added `platform.getBlock` API.
- Cleaned up block building logic to be more modular and testable.

### Coreth

- Increased `FeeHistory` maximum historical limit to improve MetaMask UI on the C-Chain.
- Enabled chain state metrics.
- Migrated go-ethereum v1.10.16 changes.

### Miscellaneous

- Added the ability to load new VM plugins dynamically.
- Implemented X-chain + P-chain wallet that can be used to build and sign transactions. Without providing a full node private keys.
- Integrated e2e testing to the repo to avoid maintaining multiple synced repos.
- Fixed `proposervm` height indexing check to correctly mark the indexer as repaired.
- Introduced message throttling overrides to be used in future improvements to reliably send messages.
- Introduced a cap on the client specified request deadline.
- Increased the default `leveldb` open files limit to `1024`.
- Documented the `leveldb` configurations.
- Extended chain shutdown timeout.
- Performed various cleanup passes.

## [v1.7.5](https://github.com/ava-labs/avalanchego/releases/tag/v1.7.5)

This version is backwards compatible to [v1.7.0](https://github.com/ava-labs/avalanchego/releases/tag/v1.7.0). It is optional, but encouraged.

### Consensus

- Added asynchronous processing of `App.*` messages.
- Added height indexing support to the `proposervm` and `rpcchainvm`. If a node is updated to `>=v1.7.5` and then downgraded to `<v1.7.5`, the user must enable the `--reset-proposervm-height-index=true` flag to ensure the `proposervm` height index is correctly updated going forward.
- Fixed bootstrapping job counter initialization that could cause negative ETAs to be reported.
- Fixed incorrect processing check that could log incorrect information.
- Removed incorrect warning logs.

### Miscellaneous

- Added tracked subnets to be reported in calls to the `info.peers` API.
- Updated gRPC implementations to use `buf` tooling and standardized naming and locations.
- Added a consistent hashing implementation to be used in future improvements.
- Fixed database iteration invariants to report `ErrClosed` rather than silently exiting.
- Added additional sanity checks to prevent users from incorrectly configuring their node.
- Updated log timestamps to include milliseconds.

### Coreth

- Added beta support for offline pruning.
- Refactored peer networking layer.
- Enabled cheap metrics by default.
- Marked RPC call metrics as expensive.
- Added Abigen support for native asset call precompile.
- Fixed bug in BLOCKHASH opcode during traceBlock.
- Fixed bug in handling updated chain config on startup.

## [v1.7.4](https://github.com/ava-labs/avalanchego/releases/tag/v1.7.4)

This version is backwards compatible to [v1.7.0](https://github.com/ava-labs/avalanchego/releases/tag/v1.7.0). It is optional, but encouraged.

**The first startup of the C-Chain will take a few minutes longer due to an index update.**

### Consensus

- Removed deprecated Snowstorm consensus implementation that no longer aligned with the updated specification.
- Updated bootstrapping logs to no longer reset counters after a node restart.
- Added bootstrapping ETAs for fetching Snowman blocks and executing operations.
- Renamed the `MultiPut` message to the `Ancestors` message to match other message naming conventions.
- Introduced Whitelist conflicts into the Snowstorm specification that will be used in future X-chain improvements.
- Refactored the separation between the Bootstrapping engine and the Consensus engine to support Fast-Sync.

### Coreth

- Added an index mapping height to the list of accepted atomic operations at that height in a trie. Generating this index will cause the node to take a few minutes longer to startup the C-Chain for the first restart.
- Updated Geth dependency to `v1.10.15`.
- Updated `networkID` to match `chainID`.

### VMs

- Refactored `platformvm` rewards calculations to enable usage from an external library.
- Fixed `platformvm` and `avm` UTXO fetching to not re-iterate the UTXO set if no UTXOs are fetched.
- Refactored `platformvm` status definitions.
- Added support for multiple address balance lookups in the `platformvm`.
- Refactored `platformvm` and `avm` keystore users to reuse similar code.

### RPCChainVM

- Returned a `500 InternalServerError` if an unexpected gRPC error occurs during the handling of an HTTP request to a plugin.
- Updated gRPC server's max message size to enable responses larger than 4MiB from the plugin's handling of an HTTP request.

### Configs

- Added `--stake-max-consumption-rate` which defaults to `120,000`.
- Added `--stake-min-consumption-rate` which defaults to `100,000`.
- Added `--stake-supply-cap` which defaults to `720,000,000,000,000,000` nAVAX.
- Renamed `--bootstrap-multiput-max-containers-sent` to `--bootstrap-ancestors-max-containers-sent`.
- Renamed `--bootstrap-multiput-max-containers-received` to `--bootstrap-ancestors-max-containers-received`.
- Enforced that `--staking-enabled=false` can not be specified on public networks (`Fuji` and `Mainnet`).

### Metrics

- All `multi_put` metrics were converted to `ancestors` metrics.

### Miscellaneous

- Improved `corruptabledb` error reporting by tracking the first reported error.
- Updated CPU tracking to use the proper EWMA tracker rather than a linear approximation.
- Separated health checks into `readiness`, `healthiness`, and `liveness` checks to support more fine-grained monitoring.
- Refactored API client utilities to use a `Context` rather than an explicit timeout.

## [v1.7.3](https://github.com/ava-labs/avalanchego/releases/tag/v1.7.3)

This version is backwards compatible to [v1.7.0](https://github.com/ava-labs/avalanchego/releases/tag/v1.7.0). It is optional, but encouraged.

### Consensus

- Introduced a notion of vertex conflicts that will be used in future X-chain improvements.

### Coreth

- Added an index mapping height to the list of accepted atomic transactions at that height. Generating this index will cause the node to take approximately 2 minutes longer to startup the C-Chain for the first restart.
- Fixed bug in base fee estimation API that impacted custom defined networks.
- Decreased minimum transaction re-gossiping interval from 1s to 500ms.
- Removed websocket handler from the static vm APIs.

### Database

- Reduced lock contention in `prefixDB`s.

### Networking

- Increase the gossip size from `6` to `10` validators.
- Prioritized `Connected` and `Disconnected` messages in the message handler.

### Miscellaneous

- Notified VMs of peer versions on `Connected`.
- Fixed acceptance broadcasting over IPC.
- Fixed 32-bit architecture builds for AvalancheGo (not Coreth).

## [v1.7.2](https://github.com/ava-labs/avalanchego/releases/tag/v1.7.2)

This version is backwards compatible to [v1.7.0](https://github.com/ava-labs/avalanchego/releases/tag/v1.7.0). It is optional, but encouraged.

### Coreth

- Fixed memory leak in the estimate gas API.
- Reduced the default RPC gas limit to 50,000,000 gas.
- Improved RPC logging.
- Removed pre-AP5 legacy code.

### PlatformVM

- Optimized validator set change calculations.
- Removed storage of non-decided blocks.
- Simplified error handling.
- Removed pre-AP5 legacy code.

### Networking

- Explicitly fail requests with responses that failed to be parsed.
- Removed pre-AP5 legacy code.

### Configs

- Introduced the ability for a delayed graceful node shutdown.
- Added the ability to take all configs as environment variables for containerized deployments.

### Utils

- Fixed panic bug in logging library when importing from external projects.

## [v1.7.1](https://github.com/ava-labs/avalanchego/releases/tag/v1.7.1)

This update is backwards compatible with [v1.7.0](https://github.com/ava-labs/avalanchego/releases/tag/v1.7.0). Please see the expected update times in the v1.7.0 release.

### Coreth

- Reduced fee estimate volatility.

### Consensus

- Fixed vote bubbling for unverified block chits.

## [v1.7.0](https://github.com/ava-labs/avalanchego/releases/tag/v1.7.0)

This upgrade adds support for issuing multiple atomic transactions into a single block and directly transferring assets between the P-chain and the C-chain.

The changes in the upgrade go into effect at 1 PM EST, December 2nd 2021 on Mainnet. One should upgrade their node before the changes go into effect, otherwise they may experience loss of uptime.

**All nodes should upgrade before 1 PM EST, December 2nd 2021.**

### Networking

- Added peer uptime reports as metrics.
- Removed IP rate limiting over local networks.

### PlatformVM

- Enabled `AtomicTx`s to be issued into `StandardBlock`s and deprecated `AtomicBlock`s.
- Added the ability to export/import AVAX to/from the C-chain.

### Coreth

- Enabled multiple `AtomicTx`s to be issued per block.
- Added the ability to export/import AVAX to/from the P-chain.
- Updated dynamic fee calculations.

### ProposerVM

- Removed storage of undecided blocks.

### RPCChainVM

- Added support for metrics to be reported by plugin VMs.

### Configs

- Removed `--snow-epoch-first-transition` and `snow-epoch-duration` as command line arguments.

## [v1.6.5](https://github.com/ava-labs/avalanchego/releases/tag/v1.6.5)

This version is backwards compatible to [v1.6.0](https://github.com/ava-labs/avalanchego/releases/tag/v1.6.0). It is optional, but encouraged.

### Bootstrapping

- Drop inbound messages to a chain if that chain is in the execution phase of bootstrapping.
- Print beacon nodeIDs upon failure to connect to them.

### Metrics

- Added `avalanche_{ChainID}_bootstrap_finished`, which is 1 if the chain is done bootstrapping, 0 otherwise.

### APIs

- Added `info.uptime` API call that attempts to report the network's view of the local node.
- Added `observedUptime` to each peer's result in `info.peers`.

### Network

- Added reported uptime to pong messages to be able to better track a local node's uptime as viewed by the network.
- Refactored request timeout registry to avoid a potential race condition.

## [v1.6.4](https://github.com/ava-labs/avalanchego/releases/tag/v1.6.4)

This version is backwards compatible to [v1.6.0](https://github.com/ava-labs/avalanchego/releases/tag/v1.6.0). It is optional, but encouraged.

### Config

- Added flag `throttler-inbound-bandwidth-refill-rate`, which specifies the max average inbound bandwidth usage of a peer.
- Added flag `throttler-inbound-bandwidth-max-burst-size`, which specifies the max inbound bandwidth usage of a peer.

### Networking

- Updated peerlist gossiping to use the same mechanism as other gossip calls.
- Added inbound message throttling based on recent bandwidth usage.

### Metrics

- Updated `avalanche_{ChainID}_handler_gossip_{count,sum}` to `avalanche_{ChainID}_handler_gossip_request_{count,sum}`.
- Updated `avalanche_{ChainID}_lat_get_accepted_{count,sum}` to `avalanche_{ChainID}_lat_accepted_{count,sum}`.
- Updated `avalanche_{ChainID}_lat_get_accepted_frontier_{count,sum}` to `avalanche_{ChainID}_lat_accepted_frontier_{count,sum}`.
- Updated `avalanche_{ChainID}_lat_get_ancestors_{count,sum}` to `avalanche_{ChainID}_lat_multi_put_{count,sum}`.
- Combined `avalanche_{ChainID}_lat_pull_query_{count,sum}` and `avalanche_{ChainID}_lat_push_query_{count,sum}` to `avalanche_{ChainID}_lat_chits_{count,sum}`.
- Added `avalanche_{ChainID}_app_response_{count,sum}`.
- Added `avalanche_network_bandwidth_throttler_inbound_acquire_latency_{count,sum}`
- Added `avalanche_network_bandwidth_throttler_inbound_awaiting_acquire`
- Added `avalanche_P_vm_votes_won`
- Added `avalanche_P_vm_votes_lost`

### Indexer

- Added method `GetContainerByID` to client implementation.
- Client methods now return `[]byte` rather than `string` representations of a container.

### C-Chain

- Updated Geth dependency to 1.10.11.
- Added a new admin API for updating the log level and measuring performance.
- Added a new `--allow-unprotected-txs` flag to allow issuance of transactions without EIP-155 replay protection.

### Subnet & Custom VMs

- Ensured that all possible chains are run in `--staking-enabled=false` networks.

---

## [v1.6.3](https://github.com/ava-labs/avalanchego/releases/tag/v1.6.3)

This version is backwards compatible to [v1.6.0](https://github.com/ava-labs/avalanchego/releases/tag/v1.6.0). It is optional, but encouraged.

### Config Options

- Updated the default value of `--inbound-connection-throttling-max-conns-per-sec` to `256`.
- Updated the default value of `--meter-vms-enabled` to `true`.
- Updated the default value of `--staking-disabled-weight` to `100`.

### Metrics

- Changed the behavior of `avalanche_network_buffer_throttler_inbound_awaiting_acquire` to only increment if the message is actually blocking.
- Changed the behavior of `avalanche_network_byte_throttler_inbound_awaiting_acquire` to only increment if the message is actually blocking.
- Added `Block/Tx` metrics on `meterVM`s.
  - Added `avalanche_{ChainID}_vm_metervm_build_block_err_{count,sum}`.
  - Added `avalanche_{ChainID}_vm_metervm_parse_block_err_{count,sum}`.
  - Added `avalanche_{ChainID}_vm_metervm_get_block_err_{count,sum}`.
  - Added `avalanche_{ChainID}_vm_metervm_verify_{count,sum}`.
  - Added `avalanche_{ChainID}_vm_metervm_verify_err_{count,sum}`.
  - Added `avalanche_{ChainID}_vm_metervm_accept_{count,sum}`.
  - Added `avalanche_{ChainID}_vm_metervm_reject_{count,sum}`.
  - Added `avalanche_{DAGID}_vm_metervm_parse_tx_err_{count,sum}`.
  - Added `avalanche_{DAGID}_vm_metervm_get_tx_err_{count,sum}`.
  - Added `avalanche_{DAGID}_vm_metervm_verify_tx_{count,sum}`.
  - Added `avalanche_{DAGID}_vm_metervm_verify_tx_err_{count,sum}`.
  - Added `avalanche_{DAGID}_vm_metervm_accept_{count,sum}`.
  - Added `avalanche_{DAGID}_vm_metervm_reject_{count,sum}`.

### Coreth

- Applied callTracer fault handling fix.
- Initialized multicoin functions in the runtime environment.

### ProposerVM

- Updated block `Delay` in `--staking-enabled=false` networks to be `0`.
