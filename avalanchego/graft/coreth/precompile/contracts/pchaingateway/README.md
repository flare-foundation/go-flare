# P-chain gateway precompile (Flare)

Address: `0x0300000000000000000000000000000000000000` · config key: `pChainGatewayConfig` · interface:
[`pchaingatewaybindings/IPChainGateway.sol`](pchaingatewaybindings/IPChainGateway.sol) · active from the Helicon
upgrade on the SAE C-chain (`vms/saevm/cchain`).

The gateway lets contracts **pull** accepted P-chain blocks and transactions during EVM execution: primary-network
staking transactions, imports from and exports to the other chains of the primary network, plain P-chain transfers
and the removal of bounded stakers. The node never decides which P-chain data enters a C-chain block; a contract
(typically one driven by the Flare daemon) chooses what to read. It replaces the off-chain attestation feed of
`PChainStakeMirror`.

## Determinism

Reading another chain during execution is only safe if every node reads the same thing, live, during bootstrapping
and in every historical replay (`eth_call`, `debug_trace*`). Three mechanisms guarantee that:

1. **Header-committed bound.** The SAE block builder writes the ProposerVM block's own P-chain height
   (`block.Context.ProposerPChainHeight`: the height the proposer sealed into the outer block, chosen via
   `GetMinimumHeight`, so at least ~30 s old, monotonic, and already accepted by every verifier's P-chain before the
   block is verified in normal operation) into `HeaderExtra.PChainHeight`. Verifiers rebuild the block with their own
   outer block's height and require a hash match, so the value is consensus-verified. The ACP-181 *epoch* height
   (`block.Context.PChainHeight`, pinned for 5 minutes) stays with warp predicate verification and is **not** the bound.
   `getPChainHeight()` returns the bound and **every read must be at or below it**; higher heights revert. It advances
   with every C-chain block, so a new P-chain transaction becomes readable roughly 30 s later, in the next C-chain block.
   The bound is also exposed as `pChainHeight` in the block JSON of `eth_getBlockByNumber/ByHash`.
2. **Accepted data only.** The reader (`vms/platformvm/pchainreader`, reachable through `snow.Context.PChainReader`)
   serves only accepted P-chain blocks, transactions and their owners; nothing depends on the local tip, mempool or
   wall clock. Everything a transaction exposes comes from its own bytes and the block timestamp, except the owners of
   consumed inputs, which are resolved from the (necessarily earlier) producing transaction.
3. **Execution barrier.** Before executing a block canonically the SAE executor waits until the local P-chain has
   accepted the committed height (`hook.ExecutionGate`, implemented by the cchain hooks). This only bites at the
   bootstrapping edge, where blocks are accepted without verification while the P-chain still catches up. RPC replays
   do not wait: on a lagging node they fail with an error instead.

## Error model

| Situation | Behaviour |
|---|---|
| Caller errors: height above the bound, no committed height (pre-Helicon block), page size above a cap, tx index / input index / output index out of range, unsupported tx type, malformed calldata | Solidity `Error(string)` revert with a `PChainGateway: …` reason; remaining gas is kept |
| Node-local failures: no P-chain reader on the chain context, committed height not accepted locally (RPC on a lagging node), database errors | `InvalidateExecution` + error: canonical execution halts loudly instead of diverging, RPC returns the error. Never a catchable revert, so a daemon cannot swallow it and advance its cursor. |

## Gas

Gas is charged **before** the corresponding work and depends only on the arguments (page sizes) and on consensus
data (the stored size of the P-chain block and of the producing transactions that are decoded, the number of owner
addresses returned). It never depends on node-local caches: a cached block or owner is charged exactly like a cold
one. There are no refunds for short pages. Constants in `gas.go`:

| Item | Gas |
|---|---|
| `getPChainHeight` | 200 |
| `getNativeAssetId` | 200 |
| block read (every other call) | 20 000 |
| per requested transaction slot (`maxTxs`) | 5 000 |
| per requested input slot (`maxInputsPerTx` per tx slot, `maxInputs`, one for `getStakeOwner`) | 10 000 |
| per requested output slot (`maxOutputsPerTx` per tx slot, `maxOutputs`) | 1 000 |
| per byte of P-chain data decoded: the block's stored bytes on every call, plus the stored bytes of each producing transaction resolved for an input (genesis UTXOs and imported inputs: none) | 10 |
| per returned owner address, of inputs and outputs (per candidate for `getStakeOwner`) | 200 |
| signature recovery of the attributed owner (`getStakeOwner`, when the first input has a candidate owner): up to two secp256k1 public-key recoveries, one per accepted signing scheme | 6 000 |

Caps: `maxTxs ≤ 32`, `maxInputsPerTx ≤ 8`, `maxOutputsPerTx ≤ 8`, `maxInputs ≤ 64`, `maxOutputs ≤ 64`. A full
`getTransactions(…, 32, …, 8, 8)` page costs 2 996 000 gas plus bytes and addresses. Byte pricing follows the measured
cold path: parsing plus decoding runs at about 65 MB/s on an M-series laptop (`BenchmarkParse*` in
`vms/platformvm/pchainreader`), about 1.5 gas/byte at 100 Mgas/s, so 10 gas/byte leaves a margin of more than 6× for
slower validators. A typical staking block (a few KB) costs tens of thousands of gas; a 128 KiB block (the P-chain
builder's target) about 1.3M; the 2 MiB maximum about 21M. The node never parses under the P-chain lock: it reads the
stored bytes under the lock (`state.GetBlockBytes`, `state.GetTxBytes`), charges them, and parses outside. Reads that
cannot pay for the bytes run out of gas before any parsing happens. The local tip height (bound checks, the execution
gate) comes from the accepted-height index, never from parsing the tip block, so a P-chain tip change costs a few
index lookups rather than an unmetered parse. Input pages are sliced directly from the decoded transaction, in its
serialized order, so a page costs nothing beyond its requested slots and the producing transactions it resolves. A
call decodes and pays for its block once: `pchain.Reader.GetBlock` returns a
handle (`pchain.BlockReader`) over the decoded block and the per-transaction input and output reads of
`getTransactions` go through that handle, so they charge only the producing transactions the inputs resolve. Outputs
need no lookup at all: the decoded block keeps references into the parsed transaction and a page is converted on
demand, which is why an output slot costs a tenth of an input slot.

## Data model

Type codes (bit `t` of `typeFilter` selects type `t`; `0` selects all, **including the transfer types below**):
1 `AddPermissionlessValidatorTx`, 2 `AddPermissionlessDelegatorTx`, 3 `AddAutoRenewedValidatorTx`,
4 `SetAutoRenewedValidatorConfigTx`, 5 `RewardAutoRenewedValidatorTx`, 6 `AddValidatorTx` (legacy),
7 `AddDelegatorTx` (legacy), 8 `ImportTx`, 9 `ExportTx`, 10 `BaseTx` (plain transfer, Durango and later),
11 `RewardValidatorTx` (removal of a bounded staker of type 1, 2, 6 or 7). Everything else, and permissionless stakers
of other subnets, is skipped but still counted in `txCount` and in the raw indexes.

| Type | startTime | endTime | period | weight | shares | stakerTxId | remoteChainId |
|---|---|---|---|---|---|---|---|
| 1 / 2 / 6 / 7 | block timestamp if Durango is active at that block, else the tx's own start | tx end | 0 | tx weight | `DelegationShares` (validators) | 0 | 0 |
| 3 | block timestamp | start + `Period` | `Period` | sum of stake outputs | `DelegationShares`, `AutoCompoundRewardShares` | 0 | 0 |
| 4 | block timestamp | 0 | next `Period` (0 = exit at cycle end) | 0 | `AutoCompoundRewardShares` | add tx | 0 |
| 5 | block timestamp | cycle end | 0 | 0 | 0 | add tx | 0 |
| 8 import | block timestamp | 0 | 0 | 0 | 0 | 0 | source chain |
| 9 export | block timestamp | 0 | 0 | 0 | 0 | 0 | destination chain |
| 10 transfer | block timestamp | 0 | 0 | 0 | 0 | 0 | 0 |
| 11 reward validator | block timestamp | block timestamp (the staker's end) | 0 | 0 | 0 | removed staker | 0 |

Block kinds: 0 Standard, 1 Proposal, 2 Commit, 3 Abort, 4 Atomic (pre-Apricot-Phase-5; one import or export). A
Proposal block lists its decision transactions first and the proposal transaction (type 5 or 11, or a legacy staking
tx in Apricot-era blocks) last; the proposal's outcome is the kind of the block at `height+1`, readable once
`height+1 ≤ bound`. For type 11 a Commit removes the staker with its reward, an Abort without it (rewards are 0 on
Flare either way); the returned stake and any reward UTXOs are keyed by the *staker* transaction, so the reward
transaction itself has no inputs and no outputs. Auto-renewed validators keep their original transaction id across
cycles; contracts own the lifecycle interpretation. Pre-Banff blocks report `timestamp = 0`. Transfers have no
amount field of their own: sum the outputs (or the imported inputs) instead, per asset.

### Assets

The P-chain UTXO set holds the native asset (FLR, SGB) and any asset created on the X-chain: custom assets can be
imported to the P-chain, transferred there and exported again, and a staking transaction may pass one through on its
fee side (the flow checker only requires every asset to balance). Every input and output therefore carries the
`assetId` of its own bytes, and its `amount` is in the smallest unit of that asset; an amount is native funds only if
`assetId == getNativeAssetId()`, the primary network's AVAX asset ID from the chain context (a genesis constant,
identical on every node). No lookup is involved: an input's own asset field is reported, which consensus verified
against the consumed output (for imported inputs against the shared-memory UTXO), so imported and unresolved inputs
report their asset too. Stakes are always native: `weight` and outputs of kind 1 are in nFLR, because the P-chain
rejects primary-network stakes of any other asset (legacy, permissionless and auto-renewed alike). A contract that
credits FLR must check the asset of every transfer, import or export input and output it sums. The gateway never
filters non-native entries: that would break the UTXO indexes and the pagination.

### Inputs and owners

`getTransactionInputs` pages through the consumed UTXOs of one transaction. Each input carries the UTXO id, the
`assetId` and `amount` of the consumed output, the owner of the consumed output (`locktime`, all `addresses` in on-chain byte order, `threshold`, plus the
`stakeableLocktime` of a stakeable-locked output, 0 otherwise; resolved from the producing transaction: BaseTx
outputs, returned stake, reward UTXOs and genesis UTXOs) and `sigIndices`, the owner slots whose signatures authorized
the spend. `resolved == false` means the producing output is unknown (owner fields zero). Addresses are P-chain
addresses (`bytes20`), not EVM addresses; only `getStakeOwner` derives an EVM address, from the owner's signature.

An import's inputs are its P-chain inputs (fee payment, usually none) followed by its **imported** inputs, the UTXOs
exported by the source chain (`utxoTxId` is that chain's export transaction). Imported inputs are flagged
`imported`, carry their `assetId`, `amount` and `sigIndices`, and are never resolved (`resolved == false`): their producing
transaction is not on the P-chain, and the reader does not look it up (a lookup would fail as a node-local error).

Inputs are listed in canonical order, the transaction's serialized order: the P-chain inputs, then the imported
inputs, each group sorted by UTXO transaction id **bytes**, then output index (the indexer's `in_idx`). Pages are
slices of that list.

### Outputs

`getTransactionOutputs` pages through the outputs a transaction produces, in the **P-chain's UTXO index space**: the
BaseTx outputs first, then the type-specific extra outputs at the indexes the P-chain assigns them. Output `i` is
therefore exactly the `(utxoTxId = txId, utxoOutputIndex = i)` an input consuming it reports, and `outputCount` the
total. Each output carries its `kind`, `assetId`, `amount` and owner (`locktime`, `stakeableLocktime`, `threshold`,
`addresses`; an output of any other type, which the P-chain never produces, reports no owner). Kinds:

| Kind | Meaning |
|---|---|
| 0 transfer | a P-chain UTXO: the BaseTx outputs of every type (change, plain transfers, imported funds) |
| 1 stake | the stake outputs of types 1, 2, 3, 6, 7: locked while the staker is active, returned as a P-chain UTXO **with the same id and index** when the staker is removed (bounded stakers at their end time via a type-11 tx; auto-renewed validators on exit or abort, never on renewal) |
| 2 exported | the exported outputs of type 9: UTXOs created on `remoteChainId` with the same id and index; their addresses are that chain's secp256k1fx owner addresses, not EVM addresses |

Types 4 (config) and 10 (transfer) have transfer outputs only; types 5 and 11 have none.

### `getStakeOwner`

Reproduces the attribution rule of the Flare P-chain indexer (`utils/staking/utils.go` `DedupeTxs` and
`indexer/shared` `selectSigners`): take the first input (in canonical order: the indexer's `in_idx` 0), keep only
the owners that signed it (all owners if there are no usable signature indices), and pick the lexically smallest
**bech32** string with the
network's HRP. The owner is returned twice: `pChainOwner`, its P-chain address, and `cChainOwner`, the C-chain (EVM)
address of the same key. A P-chain address (`ripemd160(sha256(compressed key))`) cannot be converted into an EVM
address (`keccak256(uncompressed key)[12:]`), but the owner signed the transaction: the node recovers the public key
from the owner's signature in the input's credential (the signature at the owner's slot of `sigIndices`) over the
unsigned transaction bytes, under either signing scheme the P-chain accepts — the plain sha256 of the bytes, or the
Ethereum signed-message hash of its hex encoding that Ethereum wallets produce — exactly as `vms/secp256k1fx`
`VerifyCredentials` does (both call `secp256k1fx.SigningHashes.RecoverSigner`, so the gateway cannot diverge from
consensus), and derives both addresses from the key. The reader exposes the material as `pchain.Tx.UnsignedBytes` and
`pchain.Input.Signatures` (credential *i* of a transaction belongs to input *i*; the trailing validator-authority
credential of a config transaction is not an input credential). `resolved` is true only when an owner is attributed
**and** its signature recovers to it, so `resolved` implies both addresses are set. Consensus verified exactly that
signature before accepting the transaction, so the recovery cannot fail for accepted data; a failure (data the
P-chain never accepts) leaves the stake unattributed, `(0, 0, false)`, deterministically, rather than reverting.

Genesis inputs (UTXOs of the genesis allocation, which have no producing transaction; their `utxoTxId` is zero on
production networks), imported inputs and unresolved inputs yield `resolved == false`, even though
`getTransactionInputs` reports the genesis allocation's owner for genesis inputs. So does a located output with
threshold 0: secp256k1fx permits an owner with no addresses (spendable without signatures), `getTransactionInputs`
reports it as `resolved` with an empty `addresses`, but there is nobody to attribute the stake to, so `getStakeOwner`
returns `(0, 0, false)`. Output resolution and stake attribution are different conditions. Bech32 string order is not
byte order, which is why the node does this comparison. On transfers (types 8, 9, 10) the rule attributes the sender
of the first input — the fee payer of an import, which has no owner when the import is funded from imported UTXOs
alone.

## Activation

`vms/saevm/cchain/genesis.go` adds a `PrecompileUpgrade` keyed on the Helicon time and activates the account at
genesis (when Helicon is active at genesis) or in `StartExecutingBlock` at the first Helicon block. Before Helicon the
header carries no `PChainHeight`, so nothing is readable.

## Regenerating the ABI

`pchaingatewaybindings/compile.go` carries the `go generate` directive for a native `solc` (`--abi --pretty-json`).
Without one, the solc-js pinned by `local-scripts/p-chain-precompile-test` (0.8.28) produces the same ABI through the
standard-JSON API, provided the keys of every parameter object are reordered to `internalType, name, type, components`
(solc-js emits `components` first) and the JSON is written with two-space indentation and a trailing newline; the
result is byte-identical to the native output. `TestABISelectorsAndSignatures` and `TestPackUnpackRoundTrips` verify
the embedded file.
