// (c) 2026, Flare Network. All rights reserved.
// See the file LICENSE for licensing terms.

package saexec

import (
	"fmt"
	"math/big"

	"github.com/holiman/uint256"

	"github.com/ava-labs/libevm/common"
	"github.com/ava-labs/libevm/core"
	"github.com/ava-labs/libevm/core/state"
	"github.com/ava-labs/libevm/core/types"
	"github.com/ava-labs/libevm/core/vm"
	"github.com/ava-labs/libevm/crypto"
	"github.com/ava-labs/libevm/log"
	"github.com/ava-labs/libevm/params"

	dconsensus "github.com/ava-labs/avalanchego/graft/coreth/consensus"
	dcore "github.com/ava-labs/avalanchego/graft/coreth/core"
	corethextras "github.com/ava-labs/avalanchego/graft/coreth/params/extras"
)

// ExtrasConfig parameterises the divergences from upstream transaction
// application.
type ExtrasConfig struct {
	// Flare has a burn address 0x000000000000000000000000000000000000dEaD while
	// Songbird has a burn address 0x0100000000000000000000000000000000000000 for the transaction fees.
	BurnAddress common.Address

	// NominalGasPrice (wei/gas) sets the nominal fee,
	// params.TxGas * NominalGasPrice, that prioritised calls pay.
	NominalGasPrice uint64

	// Whether to call the daemon after each transaction
	CallDaemon bool
}

// NewExtrasConfig derives the Flare/Songbird transaction-application extras
// for the given chain ID. On Songbird-family chains it also validates that the
// block coinbase is the expected 0x0100…00 burn address and errors otherwise.
func NewExtrasConfig(chainID *big.Int, coinbase common.Address) (*ExtrasConfig, error) {
	burnAddress, nominalGasPrice, isFlare, isSongbird, err := dcore.StateTransitionVariants.GetValue(chainID)(coinbase)
	if err != nil {
		return nil, err
	}
	return &ExtrasConfig{
		BurnAddress:     burnAddress,
		NominalGasPrice: nominalGasPrice,
		CallDaemon:      isFlare || isSongbird,
	}, nil
}

// daemonEvmCaller adapts the transaction-application environment to
// [dcore.EVMCaller], the dependency interface of the daemon/mint logic. It
// mirrors the methods that coreth's StateTransition provides pre-Helicon.
type daemonEvmCaller struct {
	evm   *vm.EVM
	state *state.StateDB
}

var _ dcore.EVMCaller = (*daemonEvmCaller)(nil)

func (e *daemonEvmCaller) GetChainID() *big.Int {
	return e.evm.ChainConfig().ChainID
}

func (e *daemonEvmCaller) AddBalance(addr common.Address, amount *uint256.Int) {
	e.state.AddBalance(addr, amount)
}

func (e *daemonEvmCaller) DaemonCall(caller vm.ContractRef, addr common.Address, input []byte, gas uint64) (snapshot int, ret []byte, leftOverGas uint64, err error) {
	return dcore.DaemonCall(e.evm, caller, addr, input, gas)
}

func (e *daemonEvmCaller) DaemonRevertToSnapshot(snapshot int) {
	e.evm.StateDB.RevertToSnapshot(snapshot)
}

func (e *daemonEvmCaller) GetBlockTime() uint64 {
	return e.evm.Context.Time
}

func (e *daemonEvmCaller) GetGasLimit() uint64 {
	return e.evm.Context.GasLimit
}

// legacyChainContext bridges SAE's libevm chain context to grafted coreth's
// legacy context. Header lookup is the only operation used after the explicit
// block author has been supplied; it preserves BLOCKHASH behavior. Engine is
// deliberately nil because SAE always passes the header coinbase as author.
type legacyChainContext struct {
	core.ChainContext
}

var _ dcore.ChainContext = legacyChainContext{}

func (legacyChainContext) Engine() dconsensus.Engine { return nil }

// IsLegacyCorethBlock reports whether a block with the given timestamp is
// applied through the legacy pre-Helicon coreth StateTransition (see
// [ApplyTransaction]) rather than [ApplyTransactionWithExtras]. Only these
// blocks use the header base fee directly; SAE-era blocks (post-Helicon coreth,
// or any config without coreth extras) carry a worst-case base-fee bound whose
// executed value must be derived (see [DeriveExecutedBaseFee]). A config without
// coreth extras, or with no scheduled Helicon, is never legacy.
func IsLegacyCorethBlock(config *params.ChainConfig, blockTime uint64) bool {
	upgrades, hasCorethExtras := config.Hooks().(*corethextras.ChainConfig)
	return hasCorethExtras && upgrades != nil && upgrades.HeliconTimestamp != nil && !upgrades.IsHelicon(blockTime)
}

// ApplyTransaction selects the transaction implementation appropriate for the
// block timestamp. Historical pre-Helicon C-Chain blocks must keep using
// coreth's StateTransition because it contains the one-time Flare/Songbird
// transition-contract calls. From Helicon onward, only the prioritised fee
// refund and daemon call remain, as implemented by ApplyTransactionWithExtras.
//
// A nil Helicon timestamp identifies generic SAE configurations outside the
// C-Chain; they retain SAE's transaction implementation.
//
// Besides [Execute]'s transaction loop, this is the per-transaction replay
// primitive of the SAE rpc block-level tracers, which inject a tracer via cfg.
func ApplyTransaction(config *params.ChainConfig, bc core.ChainContext, author *common.Address, gp *core.GasPool, statedb *state.StateDB, header *types.Header, tx *types.Transaction, usedGas *uint64, cfg vm.Config) (*types.Receipt, error) {
	if IsLegacyCorethBlock(config, header.Time) {
		legacyBC := legacyChainContext{ChainContext: bc}
		blockContext := dcore.NewEVMBlockContext(header, legacyBC, author)
		return dcore.ApplyTransaction(
			config,
			legacyBC,
			blockContext,
			(*dcore.GasPool)(gp),
			statedb,
			header,
			tx,
			usedGas,
			cfg,
		)
	}

	return ApplyTransactionWithExtras(config, bc, author, gp, statedb, header, tx, usedGas, cfg)
}

// ApplyTransactionWithExtras mirrors core.ApplyTransaction for Helicon and
// later blocks.
// Check for any differences in the upstream implementation when updating libevm
func ApplyTransactionWithExtras(config *params.ChainConfig, bc core.ChainContext, author *common.Address, gp *core.GasPool, statedb *state.StateDB, header *types.Header, tx *types.Transaction, usedGas *uint64, cfg vm.Config) (*types.Receipt, error) {
	msg, err := core.TransactionToMessage(tx, types.MakeSigner(config, header.Number, header.Time), header.BaseFee)
	if err != nil {
		return nil, err
	}
	// Create a new context to be used in the EVM environment
	blockContext := core.NewEVMBlockContext(header, bc, author)
	txContext := core.NewEVMTxContext(msg)
	vmenv := vm.NewEVM(blockContext, txContext, statedb, config, cfg)

	// Specific for Flare
	extras, err := NewExtrasConfig(config.ChainID, vmenv.Context.Coinbase)
	if err != nil {
		return nil, err
	}
	return applyTransactionWithExtras(msg, config, gp, statedb, header.Number, header.Hash(), tx, usedGas, vmenv, extras)
}

// applyTransactionWithExtras is a copy of libevm's unexported
// core.applyTransaction (core/state_processor.go), with the addition of the
// Flare-specific fee settlement (burn + prioritised-contract fee refund) and
// the daemon call.
// Check for any differences in the upstream implementation when updating libevm.
func applyTransactionWithExtras(msg *core.Message, config *params.ChainConfig, gp *core.GasPool, statedb *state.StateDB, blockNumber *big.Int, blockHash common.Hash, tx *types.Transaction, usedGas *uint64, evm *vm.EVM, extras *ExtrasConfig) (*types.Receipt, error) {
	// Create a new context to be used in the EVM environment.
	txContext := core.NewEVMTxContext(msg)
	evm.Reset(txContext, statedb)

	txSnapshot := statedb.Snapshot() // revert if the daemon invalidates execution

	// Apply the transaction to the current state (included in the env).
	result, err := core.ApplyMessage(evm, msg, gp)
	if err != nil {
		return nil, err
	}

	// Flare-specific features, mirroring the pre-Helicon
	// coreth StateTransition.TransitionDb. Both insertions MUST stay before
	// statedb.Finalise/IntermediateRoot (Finalise invalidates snapshot
	// revisions) and before GetLogs (so daemon logs reach the receipt).

	// Feature 1/2: relocate the fee that core.ApplyMessage credited to the
	// coinbase into the burn address (0x…dEaD on Flare; on Songbird the burn
	// address is the coinbase itself, so the move is a no-op). This applies
	// to failed transactions too. For prioritised contract calls only the
	// nominal fee is burned and the excess is returned to the sender. The
	// gas-cap check inside IsPrioritisedContractCall receives the transaction
	// gas limit, matching the pre-Helicon st.initialGas semantics.
	prioritised := result.Err == nil &&
		dcore.IsPrioritisedContractCall(config.ChainID, evm.Context.Time, msg.To, msg.Data, result.ReturnData, msg.GasLimit)
	if err := settleFees(statedb, evm.Context.Coinbase, msg, result, prioritised, extras); err != nil {
		return nil, err
	}

	if result.Err == nil && extras.CallDaemon {
		// Feature 2/2: daemon call with a free, fixed gas allowance. Errors
		// are deliberately ignored (logged inside AtomicDaemonAndMint):
		// DaemonCall snapshots before the call and AtomicDaemonAndMint
		// reverts on mint failure, leaving the transaction and receipt
		// untouched. No evm.Reset here — it would corrupt
		// receipt.ContractAddress (built below from evm.TxContext.Origin) for
		// creation transactions.
		evmCaller := &daemonEvmCaller{
			evm:   evm,
			state: statedb,
		}

		dcore.AtomicDaemonAndMint(evmCaller, log.Root())

		// The daemon runs after core.ApplyMessage, so libevm's only
		// invalidation check (inside TransitionDb) cannot observe a
		// daemon-triggered vm.EVM.InvalidateExecution
		if err := evm.ExecutionInvalidated(); err != nil {
			statedb.RevertToSnapshot(txSnapshot)
			return nil, fmt.Errorf("execution invalidated: %w", err)
		}
	}
	// End of Flare-specific features.

	// Update the state with pending changes.
	var root []byte
	if config.IsByzantium(blockNumber) {
		statedb.Finalise(true)
	} else {
		root = statedb.IntermediateRoot(config.IsEIP158(blockNumber)).Bytes()
	}
	*usedGas += result.UsedGas

	// Create a new receipt for the transaction, storing the intermediate root and gas used
	// by the tx.
	receipt := &types.Receipt{Type: tx.Type(), PostState: root, CumulativeGasUsed: *usedGas}
	if result.Failed() {
		receipt.Status = types.ReceiptStatusFailed
	} else {
		receipt.Status = types.ReceiptStatusSuccessful
	}
	receipt.TxHash = tx.Hash()
	receipt.GasUsed = result.UsedGas

	if tx.Type() == types.BlobTxType {
		receipt.BlobGasUsed = uint64(len(tx.BlobHashes()) * params.BlobTxBlobGasPerBlob)
		receipt.BlobGasPrice = evm.Context.BlobBaseFee
	}

	// If the transaction created a contract, store the creation address in the receipt.
	if msg.To == nil {
		receipt.ContractAddress = crypto.CreateAddress(evm.TxContext.Origin, tx.Nonce())
	}

	// Set the receipt logs and create the bloom filter.
	receipt.Logs = statedb.GetLogs(tx.Hash(), blockNumber.Uint64(), blockHash)
	receipt.Bloom = types.CreateBloom(types.Receipts{receipt})
	receipt.BlockHash = blockHash
	receipt.BlockNumber = blockNumber
	receipt.TransactionIndex = uint(statedb.TxIndex())
	return receipt, err
}

// settleFees relocates the transaction fee that core.ApplyMessage credited to
// the block coinbase into extras.BurnAddress. For prioritised calls, only the
// nominal fee is burned and the excess is returned to the sender. When the
// burn address IS the coinbase (Songbird families and non-Flare chain IDs)
// the non-prioritised settlement is a no-op and no balances are touched.
func settleFees(statedb *state.StateDB, coinbase common.Address, msg *core.Message, result *core.ExecutionResult, prioritised bool, extras *ExtrasConfig) error {
	// msg.GasPrice is the effective gas price (set by core.TransactionToMessage):
	// the exact per-gas amount the sender paid and the coinbase received.
	gasPrice, overflow := uint256.FromBig(msg.GasPrice)
	if overflow {
		return core.ErrGasUintOverflow
	}
	actualFee, overflow := new(uint256.Int).MulOverflow(uint256.NewInt(result.UsedGas), gasPrice)
	if overflow {
		return core.ErrGasUintOverflow
	}
	if actualFee.IsZero() {
		// Zero-price simulations (NoBaseFee): ApplyMessage skipped fee
		// payment, so there is nothing to relocate.
		return nil
	}
	if extras.BurnAddress == coinbase && !prioritised {
		// The fee is already exactly where it belongs: Songbird-family
		// chains burn to the coinbase, and on non-Flare chain IDs the burn
		// address is defined as the coinbase so that upstream semantics are
		// preserved byte-for-byte. Skipping the round trip also keeps this
		// path independent of how much of the fee reached the coinbase.
		return nil
	}
	if statedb.GetBalance(coinbase).Cmp(actualFee) < 0 {
		// Unreachable for verified blocks: libevm's ApplyMessage credits the
		// effective tip to the coinbase, coreth's
		// RulesExtra.AfterExecutingTransaction hook credits the base fee to
		// constants.BlackholeAddr, and SAE block verification forces
		// header.Coinbase == constants.BlackholeAddr — so the coinbase holds
		// at least usedGas * effectiveGasPrice at this point.
		return fmt.Errorf("coinbase %s balance below transaction fee %s", coinbase, actualFee)
	}
	statedb.SubBalance(coinbase, actualFee)

	if prioritised {
		nominalFee, overflow := new(uint256.Int).MulOverflow(uint256.NewInt(params.TxGas), uint256.NewInt(extras.NominalGasPrice))
		if !overflow && actualFee.Gt(nominalFee) {
			statedb.AddBalance(extras.BurnAddress, nominalFee)
			statedb.AddBalance(msg.From, new(uint256.Int).Sub(actualFee, nominalFee))
			return nil
		}
	}
	statedb.AddBalance(extras.BurnAddress, actualFee)
	return nil
}
