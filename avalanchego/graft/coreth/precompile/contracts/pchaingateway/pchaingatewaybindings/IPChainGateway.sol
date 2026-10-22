// (c) 2026, Flare Network. All rights reserved.
// See the file LICENSE for licensing terms.

// SPDX-License-Identifier: Ecosystem
pragma solidity ^0.8.0;

/// @title P-chain gateway (Flare stateful precompile at 0x0300000000000000000000000000000000000000)
/// @notice Read-only access to accepted P-chain blocks and transactions during EVM execution:
/// primary-network staking transactions, imports, exports and plain transfers. Every read is bounded
/// by the P-chain height committed in the C-chain block header (`getPChainHeight()`), so all nodes
/// return identical data.
interface IPChainGateway {
    // Block kinds returned by `getBlock`:
    //   0 Standard, 1 Proposal (decision txs first, proposal tx last; the outcome is the kind of the
    //   block at height+1), 2 Commit, 3 Abort, 4 Atomic (pre-Apricot-Phase-5 blocks only).
    //
    // Transaction types (bit `t` of a type filter selects type `t`; a zero filter selects all):
    //   1 AddPermissionlessValidator, 2 AddPermissionlessDelegator, 3 AddAutoRenewedValidator,
    //   4 SetAutoRenewedValidatorConfig, 5 RewardAutoRenewedValidator,
    //   6 AddValidator (pre-Durango legacy), 7 AddDelegator (pre-Durango legacy),
    //   8 Import (from the C-chain or the X-chain), 9 Export (to the C-chain or the X-chain),
    //   10 Base (plain P-chain transfer), 11 RewardValidator (removal of a bounded staker of type 1, 2, 6 or 7;
    //   a proposal tx whose outcome is the kind of the block at height+1).
    //
    // Inputs are listed in canonical order (the transaction's serialized order): the P-chain inputs,
    // then the imported inputs of an import tx, each group sorted by UTXO tx id bytes, then output index.
    //
    // Output kinds:
    //   0 transfer: a P-chain UTXO (txId, position),
    //   1 stake: the staked amount, returned as a P-chain UTXO (txId, position) when the staker is
    //     removed (auto-renewed validators on exit or abort only),
    //   2 exported: a UTXO (txId, position) on the destination chain; its addresses are that chain's
    //     secp256k1fx owner addresses, not EVM addresses.
    // The position of an output in a transaction's output list (BaseTx outputs first, then the stake
    // outputs of a staker or the exported outputs of an export) is the `utxoOutputIndex` of the UTXO it
    // becomes or creates.
    //
    // Assets: the P-chain UTXO set holds the native asset (FLR, SGB) and any asset created on the
    // X-chain, so the inputs and outputs of imports, exports, transfers and the fee/change side of
    // staking transactions may carry other assets. Every input and output therefore reports its
    // `assetId`, and `amount` is in the smallest unit of that asset; an amount is native funds only
    // if `assetId == getNativeAssetId()`. Stakes are always native: `weight` and outputs of kind 1
    // are in nFLR (the P-chain rejects primary-network stakes of any other asset).

    struct Input {
        bytes32 utxoTxId;
        uint32 utxoOutputIndex;
        bytes32 assetId; // asset of the consumed output (native if equal to `getNativeAssetId()`)
        uint64 amount; // in units of `assetId`
        bool imported; // consumed from the source chain (import txs); such inputs are never resolved
        bool resolved; // false if the consumed output could not be located; owner fields are then zero
        uint64 locktime; // owner locktime
        uint64 stakeableLocktime; // stakeable lock of the consumed output, 0 if not locked
        uint32 threshold;
        bytes20[] addresses; // all owners of the consumed output, on-chain (byte) order; P-chain addresses
        uint32[] sigIndices; // positions in `addresses` whose signatures authorized the spend
    }

    struct Output {
        uint8 kind;
        bytes32 assetId; // asset of the output; always the native asset for kind 1 (stake)
        uint64 amount; // in units of `assetId`
        uint64 locktime; // owner locktime
        uint64 stakeableLocktime; // stakeable lock of the output, 0 if not locked
        uint32 threshold;
        bytes20[] addresses; // all owners, on-chain (byte) order
    }

    struct Tx {
        uint8 txType;
        uint32 index; // position in the block's raw transaction list
        bytes32 txId;
        bytes20 nodeId;
        uint64 startTime; // bounded stakers: block timestamp post-Durango, else the tx's own start; other types: block timestamp
        uint64 endTime; // stake end / first auto-renew cycle end / reward cycle end / removal time (type 11)
        uint64 period; // auto-renew cycle length; for config txs the next cycle length (0 = exit)
        uint64 weight; // staked amount, nFLR (0 for non-staking types); stakes are always the native asset
        uint32 delegationShares; // delegation fee times 10,000
        uint32 autoCompoundRewardShares; // millionths
        bytes32 stakerTxId; // the AddAutoRenewedValidatorTx for types 4 and 5, the removed staker for type 11
        bytes32 remoteChainId; // source chain of an import, destination chain of an export
        uint32 inputCount;
        uint32 outputCount;
        Input[] inputs; // first `maxInputsPerTx` inputs in canonical order
        Output[] outputs; // first `maxOutputsPerTx` outputs
    }

    /// @notice P-chain height committed in the current block header; every read must be at or below it.
    function getPChainHeight() external view returns (uint64);

    /// @notice Asset ID of the native token (FLR on Flare, SGB on Songbird) on the P-chain. An input's
    /// or output's `amount` is native funds only if its `assetId` equals this value; other assets are
    /// X-chain assets that were imported to the P-chain.
    function getNativeAssetId() external view returns (bytes32);

    /// @notice Accepted P-chain block at `height`. `timestamp` is 0 for pre-Banff blocks.
    function getBlock(
        uint64 height
    ) external view returns (bytes32 blockId, bytes32 parentId, uint64 timestamp, uint8 kind, uint32 txCount);

    /// @notice Decoded transactions of the block at `height`, scanning raw indexes from `fromIndex`,
    /// returning at most `maxTxs` transactions matching `typeFilter` with up to `maxInputsPerTx` inline
    /// inputs and `maxOutputsPerTx` inline outputs each. `nextIndex` is the raw index to continue from;
    /// it equals `txCount` when the block is exhausted.
    function getTransactions(
        uint64 height,
        uint32 fromIndex,
        uint32 maxTxs,
        uint256 typeFilter,
        uint32 maxInputsPerTx,
        uint32 maxOutputsPerTx
    ) external view returns (Tx[] memory txs, uint32 nextIndex);

    /// @notice Inputs `[fromInput, fromInput+maxInputs)` of the transaction at `txIndex`, in canonical
    /// order, and the total number of inputs.
    function getTransactionInputs(
        uint64 height,
        uint32 txIndex,
        uint32 fromInput,
        uint32 maxInputs
    ) external view returns (Input[] memory inputs, uint32 inputCount);

    /// @notice Outputs `[fromOutput, fromOutput+maxOutputs)` of the transaction at `txIndex`, and the
    /// total number of outputs.
    function getTransactionOutputs(
        uint64 height,
        uint32 txIndex,
        uint32 fromOutput,
        uint32 maxOutputs
    ) external view returns (Output[] memory outputs, uint32 outputCount);

    /// @notice Stake owner attribution compatible with the Flare P-chain indexer: the first input (in
    /// canonical order), restricted to the owners that signed it, choosing the lexically smallest
    /// bech32 address string. `pChainOwner` is that owner's P-chain address and `cChainOwner` the
    /// EVM address of the same key, recovered from the owner's signature on the transaction (the
    /// P-chain accepts signatures over the plain transaction hash and Ethereum-prefixed ones).
    /// `resolved` is false, and both addresses zero, when the input is missing, imported, unresolved,
    /// a genesis UTXO or an ownerless output (threshold 0, no addresses), which resolves for
    /// `getTransactionInputs` but cannot be attributed, or when the owner's signature does not recover
    /// to it (impossible for accepted transactions). On transfers it attributes the sender (fee payer).
    function getStakeOwner(
        uint64 height,
        uint32 txIndex
    ) external view returns (bytes20 pChainOwner, address cChainOwner, bool resolved);
}
