package core

import (
	"math/big"
	"testing"
	"time"

	"github.com/ava-labs/coreth/core/rawdb"
	"github.com/ava-labs/coreth/core/state"
	"github.com/ava-labs/coreth/core/state/snapshot"
	"github.com/ava-labs/coreth/core/types"
	"github.com/ava-labs/coreth/core/vm"
	"github.com/ava-labs/coreth/eth/tracers/logger"
	"github.com/ava-labs/coreth/params"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethdb"
)

// Test prioritized contract (Submitter) being partially refunded when fee is high
func TestStateTransitionPrioritizedContract(t *testing.T) {
	configs := []*params.ChainConfig{params.CostonChainConfig, params.CostwoChainConfig, params.SongbirdChainConfig, params.FlareChainConfig}

	for _, config := range configs {
		key, _ := crypto.HexToECDSA("b71c71a67e1177ad4e901695e1b4b9ee17ae16c6668d313eac2f96dbcda3f291")
		from := crypto.PubkeyToAddress(key.PublicKey)
		gas := uint64(3000000)
		to := prioritisedSubmitterContractAddress
		daemon := common.HexToAddress(GetDaemonContractAddr(0))
		signer := types.LatestSignerForChainID(config.ChainID)
		tx, err := types.SignNewTx(key, signer,
			&types.LegacyTx{
				Nonce:    1,
				GasPrice: big.NewInt(1250000000000000),
				Gas:      gas,
				To:       &to,
			})
		if err != nil {
			t.Fatal(err)
		}
		txContext := vm.TxContext{
			Origin:   from,
			GasPrice: tx.GasPrice(),
		}
		context := vm.BlockContext{
			CanTransfer: CanTransfer,
			Transfer:    Transfer,
			Coinbase:    common.HexToAddress("0x0100000000000000000000000000000000000000"),
			BlockNumber: big.NewInt(5),
			Time:        uint64(time.Date(2024, time.May, 1, 0, 0, 0, 0, time.UTC).Unix()), // Time after setting Submitter contract address on all chains
			Difficulty:  big.NewInt(0xffffffff),
			GasLimit:    gas,
			BaseFee:     big.NewInt(8),
		}
		alloc := GenesisAlloc{}
		balance := new(big.Int)
		balance.SetString("10000000000000000000000000000000000", 10)
		alloc[from] = GenesisAccount{
			Nonce:   1,
			Code:    []byte{},
			Balance: balance,
		}
		alloc[to] = GenesisAccount{
			Nonce:   2,
			Code:    code,
			Balance: balance,
		}
		alloc[daemon] = GenesisAccount{
			Nonce:   3,
			Code:    daemonCode,
			Balance: balance,
		}
		_, statedb := makePreState(rawdb.NewMemoryDatabase(), alloc, false)

		// Create the tracer, the EVM environment and run it
		tracer := logger.NewStructLogger(&logger.Config{
			Debug: false,
		})
		cfg := vm.Config{Tracer: tracer}
		evm := vm.NewEVM(context, txContext, statedb, config, cfg)

		msg, err := TransactionToMessage(tx, signer, nil)
		if err != nil {
			t.Fatalf("failed to prepare transaction for tracing: %v", err)
		}

		st := NewStateTransition(evm, msg, new(GasPool).AddGas(tx.Gas()))

		balanceBefore := st.state.GetBalance(st.msg.From)
		_, err = st.TransitionDb()
		if err != nil {
			t.Fatal(err)
		}
		balanceAfter := st.state.GetBalance(st.msg.From)

		// max fee (funds above which are returned) depends on the chain used
		_, limit, _, _, _ := stateTransitionVariants.GetValue(config.ChainID)(st)
		maxFee := new(big.Int).Mul(new(big.Int).SetUint64(params.TxGas), new(big.Int).SetUint64(limit))
		diff := new(big.Int).Sub(balanceBefore, balanceAfter)

		if maxFee.Cmp(diff) != 0 {
			t.Fatalf("want %v, have %v", maxFee, diff)
		}
	}
}

// Test that daemon contract is invoked after a transaction is successfully executed
func TestStateTransitionDaemon(t *testing.T) {
	configs := []*params.ChainConfig{params.CostonChainConfig, params.CostwoChainConfig, params.SongbirdChainConfig, params.FlareChainConfig}

	for _, config := range configs {
		key, _ := crypto.HexToECDSA("b71c71a67e1177ad4e901695e1b4b9ee17ae16c6668d313eac2f96dbcda3f291")
		from := crypto.PubkeyToAddress(key.PublicKey)
		gas := uint64(3000000)
		daemon := common.HexToAddress(GetDaemonContractAddr(0))
		to := common.HexToAddress("0x7e22C4A78675ae3Be11Fb389Da9b9fb15996bb6a")
		signer := types.LatestSignerForChainID(config.ChainID)
		tx, err := types.SignNewTx(key, signer,
			&types.LegacyTx{
				Nonce:    1,
				GasPrice: big.NewInt(1250000000000000),
				Gas:      gas,
				To:       &to,
			})
		if err != nil {
			t.Fatal(err)
		}
		txContext := vm.TxContext{
			Origin:   from,
			GasPrice: tx.GasPrice(),
		}
		context := vm.BlockContext{
			CanTransfer: CanTransfer,
			Transfer:    Transfer,
			Coinbase:    common.HexToAddress("0x0100000000000000000000000000000000000000"),
			BlockNumber: big.NewInt(5),
			Time:        uint64(time.Date(2024, time.May, 1, 0, 0, 0, 0, time.UTC).Unix()),
			Difficulty:  big.NewInt(0xffffffff),
			GasLimit:    gas,
			BaseFee:     big.NewInt(8),
		}
		alloc := GenesisAlloc{}
		balance := new(big.Int)
		balance.SetString("10000000000000000000000000000000000", 10)
		alloc[from] = GenesisAccount{
			Nonce:   1,
			Code:    []byte{},
			Balance: balance,
		}
		alloc[to] = GenesisAccount{
			Nonce:   2,
			Code:    code, // Reuse the code from the previous test
			Balance: balance,
		}
		alloc[daemon] = GenesisAccount{
			Nonce:   3,
			Code:    daemonCode,
			Balance: balance,
		}
		_, statedb := makePreState(rawdb.NewMemoryDatabase(), alloc, false)

		// Create the tracer, the EVM environment and run it
		tracer := logger.NewStructLogger(&logger.Config{
			Debug: false,
		})
		cfg := vm.Config{Tracer: tracer}
		evm := vm.NewEVM(context, txContext, statedb, config, cfg)
		msg, err := TransactionToMessage(tx, signer, nil)
		if err != nil {
			t.Fatalf("failed to prepare transaction for tracing: %v", err)
		}

		st := NewStateTransition(evm, msg, new(GasPool).AddGas(tx.Gas()))

		balanceBefore := st.state.GetBalance(daemon)
		_, err = st.TransitionDb()
		if err != nil {
			t.Fatal(err)
		}
		balanceAfter := st.state.GetBalance(daemon)

		if balanceAfter.Cmp(balanceBefore) <= 0 {
			t.Fatalf("want daemon balance increase, have %v before and %v after", balanceBefore, balanceAfter)
		}
	}
}

// This is a copy of the function from tests/state_test_util.go, to create a starting state for the test EVM
// We need to copy it here due to import cycle.
func makePreState(db ethdb.Database, accounts GenesisAlloc, snapshotter bool) (*snapshot.Tree, *state.StateDB) {
	sdb := state.NewDatabase(db)
	statedb, _ := state.New(common.Hash{}, sdb, nil)
	for addr, a := range accounts {
		statedb.SetCode(addr, a.Code)
		statedb.SetNonce(addr, a.Nonce)
		statedb.SetBalance(addr, a.Balance)
		for k, v := range a.Storage {
			statedb.SetState(addr, k, v)
		}
	}
	// Commit and re-open to start with a clean state.
	root, _ := statedb.Commit(false, false)

	snapConfig := snapshot.Config{
		CacheSize:  64,
		AsyncBuild: false,
		NoBuild:    false,
		SkipVerify: true,
	}
	var snaps *snapshot.Tree
	if snapshotter {
		snaps, _ = snapshot.New(snapConfig, sdb.DiskDB(), sdb.TrieDB(), common.Hash{}, root)
	}
	statedb, _ = state.New(root, sdb, snaps)
	return snaps, statedb
}

// This is a simple EVM code that returns 0x01 (necessary for a prioritized contract to be refunded)
var code = []byte{
	byte(vm.PUSH1), 0x01, byte(vm.PUSH1), 0x0, byte(vm.MSTORE8), // store 1 memory at offset 0 for return
	byte(vm.PUSH1), 0x01, byte(vm.PUSH1), 0x0, // set return value size to 1, offset 0
	byte(vm.RETURN), // return 0x01
}

// return a 32-bit value, set to 1 for daemon balance change
var daemonCode = []byte{
	byte(vm.PUSH1), 0x01, byte(vm.PUSH1), 0x0, byte(vm.MSTORE), // store 1 memory at offset 0 for return
	byte(vm.PUSH1), 0x20, byte(vm.PUSH1), 0x0, // set return value size to 32 bits, offset 0
	byte(vm.RETURN), // return 0x0..01
}
