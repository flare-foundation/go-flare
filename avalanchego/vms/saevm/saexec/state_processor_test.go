// (c) 2026, Flare Network. All rights reserved.
// See the file LICENSE for licensing terms.

package saexec

import (
	"crypto/ecdsa"
	"math"
	"math/big"
	"testing"
	"time"

	"github.com/holiman/uint256"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ava-labs/libevm/common"
	"github.com/ava-labs/libevm/consensus"
	"github.com/ava-labs/libevm/core"
	"github.com/ava-labs/libevm/core/rawdb"
	"github.com/ava-labs/libevm/core/state"
	"github.com/ava-labs/libevm/core/types"
	"github.com/ava-labs/libevm/core/vm"
	"github.com/ava-labs/libevm/crypto"

	"github.com/ava-labs/avalanchego/graft/coreth/nativeasset"
	"github.com/ava-labs/avalanchego/graft/coreth/params/extras"
	"github.com/ava-labs/avalanchego/graft/coreth/plugin/evm/upgrade/ap4"
	"github.com/ava-labs/avalanchego/graft/evm/constants"
	"github.com/ava-labs/avalanchego/snow"
	networkconstants "github.com/ava-labs/avalanchego/utils/constants"

	dcore "github.com/ava-labs/avalanchego/graft/coreth/core"
	corethparams "github.com/ava-labs/avalanchego/graft/coreth/params"
	corethevm "github.com/ava-labs/avalanchego/graft/coreth/plugin/evm"
	ethparams "github.com/ava-labs/libevm/params"
)

// withCChainExtras runs fn with the libevm extras registered exactly as
// main/main.go does for the production binary (RegisterAllLibEVMExtras),
// scoped to fn. [ApplyTransactionWithExtras] relies on this registration:
// coreth's params.RulesExtra.ShouldCreditBaseFeeToCoinbase makes libevm credit
// the base fee to the coinbase ([constants.BlackholeAddr]), without which
// settleFees would find the coinbase underfunded.
func withCChainExtras(t *testing.T, fn func(t *testing.T)) {
	t.Helper()
	require.NoError(t, corethevm.WithTempRegisteredLibEVMExtras(func() error {
		fn(t)
		return nil
	}), "WithTempRegisteredLibEVMExtras()")
}

var (
	deadBurnAddress    = common.HexToAddress("0x000000000000000000000000000000000000dEaD")
	ftsoContractAddr   = common.HexToAddress("0x1000000000000000000000000000000000000003")
	daemonContractAddr = common.HexToAddress("0x1000000000000000000000000000000000000002")
	governanceAddr     = common.HexToAddress("0x1000000000000000000000000000000000000007")

	// Prioritised FTSO calldata prefixes, per network family (see
	// graft/coreth/core/daemon.go).
	flareFTSOPrefix    = []byte{0x8f, 0xc6, 0xf6, 0x67}
	songbirdFTSOPrefix = []byte{0xc5, 0xad, 0xc5, 0x39}
)

func newTestStateDB(t *testing.T) *state.StateDB {
	t.Helper()
	sdb, err := state.New(types.EmptyRootHash, state.NewDatabase(rawdb.NewMemoryDatabase()), nil)
	require.NoError(t, err, "state.New()")
	return sdb
}

func TestNewExtrasConfig(t *testing.T) {
	otherCoinbase := common.HexToAddress("0x0abc000000000000000000000000000000000abc")

	tests := []struct {
		name     string
		chainID  *big.Int
		coinbase common.Address
		want     *ExtrasConfig
		wantErr  bool
	}{
		{
			name:     "flare_burns_to_dead",
			chainID:  corethparams.FlareChainID,
			coinbase: constants.BlackholeAddr,
			want: &ExtrasConfig{
				BurnAddress:     deadBurnAddress,
				NominalGasPrice: uint64(ap4.MinBaseFee),
				CallDaemon:      true,
			},
		},
		{
			name:     "localflare_ignores_coinbase",
			chainID:  corethparams.LocalFlareChainID,
			coinbase: otherCoinbase,
			want: &ExtrasConfig{
				BurnAddress:     deadBurnAddress,
				NominalGasPrice: uint64(ap4.MinBaseFee),
				CallDaemon:      true,
			},
		},
		{
			name:     "songbird_burns_to_coinbase",
			chainID:  corethparams.SongbirdChainID,
			coinbase: constants.BlackholeAddr,
			want: &ExtrasConfig{
				BurnAddress:     constants.BlackholeAddr,
				NominalGasPrice: 225 * ethparams.GWei,
				CallDaemon:      true,
			},
		},
		{
			name:     "songbird_rejects_foreign_coinbase",
			chainID:  corethparams.SongbirdChainID,
			coinbase: otherCoinbase,
			wantErr:  true,
		},
		{
			name:     "coston_rejects_zero_coinbase",
			chainID:  corethparams.CostonChainID,
			coinbase: common.Address{},
			wantErr:  true,
		},
		{
			name:     "non_flare_chain_matches_upstream",
			chainID:  big.NewInt(1337),
			coinbase: otherCoinbase,
			want: &ExtrasConfig{
				BurnAddress:     otherCoinbase,
				NominalGasPrice: uint64(ap4.MinBaseFee),
				CallDaemon:      false,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := NewExtrasConfig(tt.chainID, tt.coinbase)
			if tt.wantErr {
				require.Error(t, err, "NewExtrasConfig()")
				return
			}
			require.NoError(t, err, "NewExtrasConfig()")
			assert.Equal(t, tt.want, got, "NewExtrasConfig()")
		})
	}
}

func TestSettleFees(t *testing.T) {
	var (
		sender = common.HexToAddress("0x00000000000000000000000000000000000005e4")

		flareExtras = &ExtrasConfig{
			BurnAddress:     deadBurnAddress,
			NominalGasPrice: 25 * ethparams.GWei,
			CallDaemon:      true,
		}
		songbirdExtras = &ExtrasConfig{
			BurnAddress:     constants.BlackholeAddr,
			NominalGasPrice: 225 * ethparams.GWei,
			CallDaemon:      true,
		}

		// 50,000 gas at 100 GWei.
		fee = uint256.NewInt(5_000_000_000_000_000)
		// 21,000 gas at the respective nominal prices.
		nominalFlare    = uint256.NewInt(525_000_000_000_000)
		nominalSongbird = uint256.NewInt(4_725_000_000_000_000)
	)

	tests := []struct {
		name        string
		coinbase    common.Address
		extras      *ExtrasConfig
		coinbaseBal *uint256.Int // pre-credited, as left by core.ApplyMessage
		usedGas     uint64
		gasPrice    *big.Int
		prioritised bool
		wantErr     string
		// Balances asserted after the call.
		want map[common.Address]*uint256.Int
	}{
		{
			name:        "flare_burns_full_fee",
			coinbase:    constants.BlackholeAddr,
			extras:      flareExtras,
			coinbaseBal: fee,
			usedGas:     50_000,
			gasPrice:    big.NewInt(100 * ethparams.GWei),
			want: map[common.Address]*uint256.Int{
				deadBurnAddress:         fee,
				constants.BlackholeAddr: uint256.NewInt(0),
				sender:                  uint256.NewInt(0),
			},
		},
		{
			name:        "flare_prioritised_refunds_excess",
			coinbase:    constants.BlackholeAddr,
			extras:      flareExtras,
			coinbaseBal: fee,
			usedGas:     50_000,
			gasPrice:    big.NewInt(100 * ethparams.GWei),
			prioritised: true,
			want: map[common.Address]*uint256.Int{
				deadBurnAddress:         nominalFlare,
				sender:                  new(uint256.Int).Sub(fee, nominalFlare),
				constants.BlackholeAddr: uint256.NewInt(0),
			},
		},
		{
			name:        "flare_prioritised_fee_below_nominal_burns_all",
			coinbase:    constants.BlackholeAddr,
			extras:      flareExtras,
			coinbaseBal: uint256.NewInt(210_000_000_000_000), // 21,000 gas at 10 GWei
			usedGas:     21_000,
			gasPrice:    big.NewInt(10 * ethparams.GWei),
			prioritised: true,
			want: map[common.Address]*uint256.Int{
				deadBurnAddress:         uint256.NewInt(210_000_000_000_000),
				sender:                  uint256.NewInt(0),
				constants.BlackholeAddr: uint256.NewInt(0),
			},
		},
		{
			name:        "songbird_fee_stays_on_coinbase",
			coinbase:    constants.BlackholeAddr,
			extras:      songbirdExtras,
			coinbaseBal: fee,
			usedGas:     50_000,
			gasPrice:    big.NewInt(100 * ethparams.GWei),
			want: map[common.Address]*uint256.Int{
				constants.BlackholeAddr: fee,
				deadBurnAddress:         uint256.NewInt(0),
				sender:                  uint256.NewInt(0),
			},
		},
		{
			name:        "songbird_prioritised_refunds_excess",
			coinbase:    constants.BlackholeAddr,
			extras:      songbirdExtras,
			coinbaseBal: fee,
			usedGas:     50_000,
			gasPrice:    big.NewInt(100 * ethparams.GWei),
			prioritised: true,
			want: map[common.Address]*uint256.Int{
				constants.BlackholeAddr: nominalSongbird,
				sender:                  new(uint256.Int).Sub(fee, nominalSongbird),
			},
		},
		{
			// Upstream-parity mode: when the burn address IS the coinbase and
			// the call is not prioritised, settlement must be a no-op even if
			// the coinbase holds less than the full fee (libevm without
			// coreth's ShouldCreditBaseFeeToCoinbase hook credits only the tip).
			name:        "burn_address_is_coinbase_noop_even_underfunded",
			coinbase:    constants.BlackholeAddr,
			extras:      songbirdExtras,
			coinbaseBal: uint256.NewInt(1), // far below the full fee
			usedGas:     50_000,
			gasPrice:    big.NewInt(100 * ethparams.GWei),
			want: map[common.Address]*uint256.Int{
				constants.BlackholeAddr: uint256.NewInt(1),
				deadBurnAddress:         uint256.NewInt(0),
				sender:                  uint256.NewInt(0),
			},
		},
		{
			name:     "zero_fee_leaves_state_untouched",
			coinbase: constants.BlackholeAddr,
			extras:   flareExtras,
			usedGas:  50_000,
			gasPrice: big.NewInt(0),
			want: map[common.Address]*uint256.Int{
				constants.BlackholeAddr: uint256.NewInt(0),
				deadBurnAddress:         uint256.NewInt(0),
				sender:                  uint256.NewInt(0),
			},
		},
		{
			name:        "underfunded_coinbase_errors",
			coinbase:    constants.BlackholeAddr,
			extras:      flareExtras,
			coinbaseBal: new(uint256.Int).SubUint64(fee, 1),
			usedGas:     50_000,
			gasPrice:    big.NewInt(100 * ethparams.GWei),
			wantErr:     "balance below transaction fee",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sdb := newTestStateDB(t)
			if tt.coinbaseBal != nil {
				sdb.AddBalance(tt.coinbase, tt.coinbaseBal)
			}
			msg := &core.Message{From: sender, GasPrice: tt.gasPrice}
			result := &core.ExecutionResult{UsedGas: tt.usedGas}

			err := settleFees(sdb, tt.coinbase, msg, result, tt.prioritised, tt.extras)
			if tt.wantErr != "" {
				require.ErrorContains(t, err, tt.wantErr, "settleFees()")
				return
			}
			require.NoError(t, err, "settleFees()")
			for addr, want := range tt.want {
				assert.Equalf(t, want, sdb.GetBalance(addr), "balance of %s", addr)
			}
		})
	}
}

// stubChainContext satisfies [core.ChainContext] for direct calls to
// [ApplyTransactionWithExtras]. Neither method is reached: the author is
// always provided and no test executes the BLOCKHASH opcode.
type stubChainContext struct{}

func (stubChainContext) Engine() consensus.Engine                    { return nil }
func (stubChainContext) GetHeader(common.Hash, uint64) *types.Header { return nil }

// mintReturnCode returns runtime bytecode that returns `value` as a 32-byte
// word for any calldata, imitating the daemon contract's mint request.
func mintReturnCode(value *uint256.Int) []byte {
	word := value.Bytes32()
	code := append([]byte{0x7f}, word[:]...)          // PUSH32 value
	code = append(code, 0x60, 0x00, 0x52)             // PUSH1 0, MSTORE
	return append(code, 0x60, 0x20, 0x60, 0x00, 0xf3) // PUSH1 32, PUSH1 0, RETURN
}

// delegateCallMintReturnCode returns runtime bytecode that DELEGATECALLs
// `target` with empty calldata, discards the result, and then returns `value`
// as a 32-byte word (the daemon mint request). DELEGATECALL pops, top-first:
// gas, addr, argsOffset, argsSize, retOffset, retSize.
func delegateCallMintReturnCode(target common.Address, value *uint256.Int) []byte {
	code := []byte{
		0x60, 0x00, // PUSH1 0 (retSize)
		0x60, 0x00, // PUSH1 0 (retOffset)
		0x60, 0x00, // PUSH1 0 (argsSize)
		0x60, 0x00, // PUSH1 0 (argsOffset)
	}
	code = append(code, 0x73)         // PUSH20
	code = append(code, target[:]...) // target address
	code = append(code,
		0x5a, // GAS (forward all remaining)
		0xf4, // DELEGATECALL
		0x50, // POP (discard the success flag)
	)
	return append(code, mintReturnCode(value)...)
}

// applyTxEnv is a minimal environment to exercise
// [ApplyTransactionWithExtras] against a single block header, without the
// full [Executor] machinery.
type applyTxEnv struct {
	config  *ethparams.ChainConfig
	statedb *state.StateDB
	header  *types.Header
	signer  types.Signer
	key     *ecdsa.PrivateKey
	sender  common.Address
	gp      core.GasPool
	usedGas uint64
}

// testBlockTime is after every prioritised-contract activation time defined
// in graft/coreth/core/daemon.go.
var testBlockTime = uint64(time.Date(2026, time.January, 1, 0, 0, 0, 0, time.UTC).Unix())

// newTestChainConfig returns a fresh all-forks-active config (mirroring
// [ethparams.MergedTestChainConfig]) for the given chain ID. A fresh object —
// rather than a copy of the shared global — is required because other tests
// in this package lazily attach libevm extra payloads of incompatible types
// to the global config.
func newTestChainConfig(chainID *big.Int) *ethparams.ChainConfig {
	return &ethparams.ChainConfig{
		ChainID:                       new(big.Int).Set(chainID),
		HomesteadBlock:                big.NewInt(0),
		EIP150Block:                   big.NewInt(0),
		EIP155Block:                   big.NewInt(0),
		EIP158Block:                   big.NewInt(0),
		ByzantiumBlock:                big.NewInt(0),
		ConstantinopleBlock:           big.NewInt(0),
		PetersburgBlock:               big.NewInt(0),
		IstanbulBlock:                 big.NewInt(0),
		MuirGlacierBlock:              big.NewInt(0),
		BerlinBlock:                   big.NewInt(0),
		LondonBlock:                   big.NewInt(0),
		ArrowGlacierBlock:             big.NewInt(0),
		GrayGlacierBlock:              big.NewInt(0),
		MergeNetsplitBlock:            big.NewInt(0),
		ShanghaiTime:                  new(uint64),
		CancunTime:                    new(uint64),
		TerminalTotalDifficulty:       big.NewInt(0),
		TerminalTotalDifficultyPassed: true,
	}
}

// newApplyTxEnv returns an environment on the given chain ID with a funded
// sender and the daemon contract (0x…02) coded to request `mintRequestWei`.
// The coinbase is [constants.BlackholeAddr], as guaranteed for verified SAE
// blocks.
func newApplyTxEnv(t *testing.T, chainID *big.Int, mintRequestWei *uint256.Int) *applyTxEnv {
	t.Helper()

	config := newTestChainConfig(chainID)

	key, err := crypto.GenerateKey()
	require.NoError(t, err, "crypto.GenerateKey()")
	sender := crypto.PubkeyToAddress(key.PublicKey)

	sdb := newTestStateDB(t)
	sdb.AddBalance(sender, uint256.MustFromDecimal("1000000000000000000000")) // 1000 native tokens
	sdb.SetCode(daemonContractAddr, mintReturnCode(mintRequestWei))

	header := &types.Header{
		Number:        big.NewInt(1),
		Time:          testBlockTime,
		BaseFee:       big.NewInt(25 * ethparams.GWei),
		GasLimit:      8_000_000,
		Coinbase:      constants.BlackholeAddr,
		Difficulty:    big.NewInt(0),
		BlobGasUsed:   new(uint64),
		ExcessBlobGas: new(uint64),
	}
	return &applyTxEnv{
		config:  config,
		statedb: sdb,
		header:  header,
		signer:  types.LatestSigner(config),
		key:     key,
		sender:  sender,
		gp:      core.GasPool(math.MaxUint64),
	}
}

func (e *applyTxEnv) apply(t *testing.T, txData types.TxData) (*types.Receipt, error) {
	t.Helper()
	tx := types.MustSignNewTx(e.key, e.signer, txData)
	return ApplyTransactionWithExtras(
		e.config, stubChainContext{}, &e.header.Coinbase, &e.gp,
		e.statedb, e.header, tx, &e.usedGas, vm.Config{},
	)
}

func (e *applyTxEnv) applyForkAware(t *testing.T, txData types.TxData) (*types.Receipt, error) {
	t.Helper()
	tx := types.MustSignNewTx(e.key, e.signer, txData)
	return ApplyTransaction(
		e.config, stubChainContext{}, &e.header.Coinbase, &e.gp,
		e.statedb, e.header, tx, &e.usedGas, vm.Config{},
	)
}

// feeOf returns gasUsed * gasPrice as a uint256.
func feeOf(t *testing.T, gasUsed uint64, gasPrice *big.Int) *uint256.Int {
	t.Helper()
	price, overflow := uint256.FromBig(gasPrice)
	require.False(t, overflow, "gas price overflows uint256")
	return new(uint256.Int).Mul(uint256.NewInt(gasUsed), price)
}

func TestApplyTransactionWithExtrasFlare(t *testing.T) {
	withCChainExtras(t, testApplyTransactionWithExtrasFlare)
}

func testApplyTransactionWithExtrasFlare(t *testing.T) {
	var (
		mint       = uint256.NewInt(1e18)
		gasPrice   = big.NewInt(100 * ethparams.GWei)
		nominalFee = new(uint256.Int).Mul(
			uint256.NewInt(ethparams.TxGas),
			uint256.NewInt(uint64(ap4.MinBaseFee)),
		)
		ftsoData = append(append([]byte{}, flareFTSOPrefix...), make([]byte, 32)...)
	)

	t.Run("prioritised_ftso_call_refunds_excess_fee", func(t *testing.T) {
		env := newApplyTxEnv(t, corethparams.FlareChainID, mint)
		initial := env.statedb.GetBalance(env.sender).Clone()

		receipt, err := env.apply(t, &types.LegacyTx{
			Nonce:    0,
			To:       &ftsoContractAddr,
			Gas:      100_000,
			GasPrice: gasPrice,
			Data:     ftsoData,
		})
		require.NoError(t, err, "ApplyTransactionWithExtras()")
		require.Equal(t, types.ReceiptStatusSuccessful, receipt.Status, "receipt status")

		actualFee := feeOf(t, receipt.GasUsed, gasPrice)
		require.True(t, actualFee.Gt(nominalFee), "test setup: actual fee must exceed the nominal fee")

		assert.Equal(t, nominalFee, env.statedb.GetBalance(deadBurnAddress), "burn address receives only the nominal fee")
		assert.Equal(t, new(uint256.Int).Sub(initial, nominalFee), env.statedb.GetBalance(env.sender), "sender pays only the nominal fee")
		assert.True(t, env.statedb.GetBalance(constants.BlackholeAddr).IsZero(), "coinbase fully drained on Flare")
		assert.Equal(t, mint, env.statedb.GetBalance(daemonContractAddr), "daemon mint request honoured")
	})

	t.Run("gas_limit_above_cap_is_not_prioritised", func(t *testing.T) {
		env := newApplyTxEnv(t, corethparams.FlareChainID, mint)
		initial := env.statedb.GetBalance(env.sender).Clone()

		// The prioritised-contract gas cap on Flare is 3M and, as pre-Helicon,
		// MUST be compared against the transaction gas limit, not the gas
		// used. This transaction uses far less than 3M gas, so comparing
		// against gas used would (incorrectly) grant the fee refund.
		receipt, err := env.apply(t, &types.LegacyTx{
			Nonce:    0,
			To:       &ftsoContractAddr,
			Gas:      3_000_001,
			GasPrice: gasPrice,
			Data:     ftsoData,
		})
		require.NoError(t, err, "ApplyTransactionWithExtras()")
		require.Equal(t, types.ReceiptStatusSuccessful, receipt.Status, "receipt status")
		require.Less(t, receipt.GasUsed, uint64(3_000_000), "test setup: gas used must be below the cap")

		actualFee := feeOf(t, receipt.GasUsed, gasPrice)
		assert.Equal(t, actualFee, env.statedb.GetBalance(deadBurnAddress), "full fee burned, no refund")
		assert.Equal(t, new(uint256.Int).Sub(initial, actualFee), env.statedb.GetBalance(env.sender), "sender pays the full fee")
		assert.Equal(t, mint, env.statedb.GetBalance(daemonContractAddr), "daemon still runs for successful txs")
	})

	t.Run("failed_tx_burns_fee_and_skips_daemon", func(t *testing.T) {
		env := newApplyTxEnv(t, corethparams.FlareChainID, mint)
		initial := env.statedb.GetBalance(env.sender).Clone()

		revertAddr := common.HexToAddress("0x00000000000000000000000000000000000000ff")
		env.statedb.SetCode(revertAddr, []byte{0x60, 0x00, 0x60, 0x00, 0xfd}) // PUSH1 0, PUSH1 0, REVERT

		receipt, err := env.apply(t, &types.LegacyTx{
			Nonce:    0,
			To:       &revertAddr,
			Gas:      100_000,
			GasPrice: gasPrice,
		})
		require.NoError(t, err, "ApplyTransactionWithExtras()")
		require.Equal(t, types.ReceiptStatusFailed, receipt.Status, "receipt status")

		actualFee := feeOf(t, receipt.GasUsed, gasPrice)
		assert.Equal(t, actualFee, env.statedb.GetBalance(deadBurnAddress), "failed txs still burn the full fee")
		assert.Equal(t, new(uint256.Int).Sub(initial, actualFee), env.statedb.GetBalance(env.sender), "sender pays the full fee")
		assert.True(t, env.statedb.GetBalance(daemonContractAddr).IsZero(), "daemon must not run after a failed tx")
	})
}

func TestApplyTransactionWithExtrasSongbird(t *testing.T) {
	withCChainExtras(t, testApplyTransactionWithExtrasSongbird)
}

func testApplyTransactionWithExtrasSongbird(t *testing.T) {
	var (
		mint       = uint256.NewInt(1e18)
		gasPrice   = big.NewInt(300 * ethparams.GWei)
		nominalFee = new(uint256.Int).Mul(
			uint256.NewInt(ethparams.TxGas),
			uint256.NewInt(225*ethparams.GWei),
		)
		ftsoData = append(append([]byte{}, songbirdFTSOPrefix...), make([]byte, 32)...)
	)

	env := newApplyTxEnv(t, corethparams.SongbirdChainID, mint)
	initial := env.statedb.GetBalance(env.sender).Clone()

	receipt, err := env.apply(t, &types.LegacyTx{
		Nonce:    0,
		To:       &ftsoContractAddr,
		Gas:      100_000,
		GasPrice: gasPrice,
		Data:     ftsoData,
	})
	require.NoError(t, err, "ApplyTransactionWithExtras()")
	require.Equal(t, types.ReceiptStatusSuccessful, receipt.Status, "receipt status")

	actualFee := feeOf(t, receipt.GasUsed, gasPrice)
	require.True(t, actualFee.Gt(nominalFee), "test setup: actual fee must exceed the nominal fee")

	assert.Equal(t, nominalFee, env.statedb.GetBalance(constants.BlackholeAddr), "nominal fee stays on the Songbird coinbase")
	assert.Equal(t, new(uint256.Int).Sub(initial, nominalFee), env.statedb.GetBalance(env.sender), "sender pays only the nominal fee")
	assert.True(t, env.statedb.GetBalance(deadBurnAddress).IsZero(), "nothing burned to the dEaD address on Songbird")
	assert.Equal(t, mint, env.statedb.GetBalance(daemonContractAddr), "daemon mint request honoured")
}

// TestApplyTransactionWithExtrasNonFlareChain asserts upstream-equivalent
// behaviour on chain IDs outside the Flare/Songbird families: the fee stays
// with the coinbase and the daemon is never invoked, even if a contract
// exists at the daemon address.
func TestApplyTransactionWithExtrasNonFlareChain(t *testing.T) {
	withCChainExtras(t, testApplyTransactionWithExtrasNonFlareChain)
}

func testApplyTransactionWithExtrasNonFlareChain(t *testing.T) {
	gasPrice := big.NewInt(100 * ethparams.GWei)
	ftsoData := append(append([]byte{}, flareFTSOPrefix...), make([]byte, 32)...)

	env := newApplyTxEnv(t, big.NewInt(1337), uint256.NewInt(1e18))
	initial := env.statedb.GetBalance(env.sender).Clone()

	receipt, err := env.apply(t, &types.LegacyTx{
		Nonce:    0,
		To:       &ftsoContractAddr,
		Gas:      100_000,
		GasPrice: gasPrice,
		Data:     ftsoData,
	})
	require.NoError(t, err, "ApplyTransactionWithExtras()")
	require.Equal(t, types.ReceiptStatusSuccessful, receipt.Status, "receipt status")

	actualFee := feeOf(t, receipt.GasUsed, gasPrice)
	assert.Equal(t, actualFee, env.statedb.GetBalance(constants.BlackholeAddr), "coinbase keeps the full fee, matching upstream")
	assert.True(t, env.statedb.GetBalance(deadBurnAddress).IsZero(), "nothing burned")
	assert.True(t, env.statedb.GetBalance(daemonContractAddr).IsZero(), "daemon must not run on non-Flare chains")
	assert.Equal(t, new(uint256.Int).Sub(initial, actualFee), env.statedb.GetBalance(env.sender), "sender pays the full fee")
}

// TestIsLegacyCorethBlock pins the era predicate that gates both the legacy
// transaction-application branch and the rpc tracers' base-fee normalisation:
// only pre-Helicon coreth blocks are legacy; post-Helicon coreth blocks and
// configs without coreth extras (generic SAE) are SAE-era.
func TestIsLegacyCorethBlock(t *testing.T) {
	const helicon = uint64(1_000)

	withCChainExtras(t, func(t *testing.T) {
		corethConfig := func(heliconTime *uint64) *ethparams.ChainConfig {
			config := newTestChainConfig(corethparams.LocalFlareChainID)
			chainExtras := *extras.TestHeliconChainConfig
			chainExtras.HeliconTimestamp = heliconTime
			chainExtras.SnowCtx = &snow.Context{NetworkID: networkconstants.LocalFlareID}
			corethparams.WithExtra(config, &chainExtras)
			return config
		}
		heliconTime := helicon

		tests := []struct {
			name      string
			config    *ethparams.ChainConfig
			blockTime uint64
			want      bool
		}{
			{"no_coreth_extras_before", newTestChainConfig(corethparams.LocalFlareChainID), helicon - 1, false},
			{"no_coreth_extras_after", newTestChainConfig(corethparams.LocalFlareChainID), helicon + 1, false},
			{"coreth_pre_helicon", corethConfig(&heliconTime), helicon - 1, true},
			{"coreth_at_helicon", corethConfig(&heliconTime), helicon, false},
			{"coreth_post_helicon", corethConfig(&heliconTime), helicon + 1, false},
			{"coreth_helicon_unscheduled", corethConfig(nil), helicon - 1, false},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				assert.Equal(t, tt.want, IsLegacyCorethBlock(tt.config, tt.blockTime))
			})
		}
	})
}

func TestApplyTransactionHeliconBoundary(t *testing.T) {
	withCChainExtras(t, testApplyTransactionHeliconBoundary)
}

// testApplyTransactionHeliconBoundary proves that historical execution keeps
// the one-time transition-contract behavior until Helicon, and stops it at the
// inclusive fork timestamp. The governance contract stores COINBASE in slot 0:
// the user's call writes the normal 0x0100... coinbase, while the legacy
// transition handler repeats the call with the governance signal coinbase.
func testApplyTransactionHeliconBoundary(t *testing.T) {
	heliconTime := testBlockTime
	newGovernance := common.HexToAddress("0x100000000000000000000000000000000000000f")
	governanceSignal := dcore.GetGovernanceSettingsCoinbaseSignalAddr(
		corethparams.LocalFlareChainID,
		heliconTime,
	)
	data := append(
		append([]byte{}, dcore.SetGovernanceAddressSelector(corethparams.LocalFlareChainID, heliconTime)...),
		common.LeftPadBytes(newGovernance.Bytes(), common.HashLength)...,
	)

	tests := []struct {
		name         string
		blockTime    uint64
		wantCoinbase common.Address
	}{
		{
			name:         "before_helicon_uses_legacy_transition_calls",
			blockTime:    heliconTime - 1,
			wantCoinbase: governanceSignal,
		},
		{
			name:         "at_helicon_uses_reduced_extras",
			blockTime:    heliconTime,
			wantCoinbase: constants.BlackholeAddr,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			env := newApplyTxEnv(t, corethparams.LocalFlareChainID, uint256.NewInt(0))
			chainExtras := *extras.TestHeliconChainConfig
			chainExtras.HeliconTimestamp = &heliconTime
			chainExtras.SnowCtx = &snow.Context{NetworkID: networkconstants.LocalFlareID}
			corethparams.WithExtra(env.config, &chainExtras)
			env.header.Time = tt.blockTime

			// COINBASE; PUSH0; SSTORE; STOP. The second (legacy-only) call
			// overwrites slot 0 with the transition signal address.
			env.statedb.SetCode(governanceAddr, []byte{byte(vm.COINBASE), byte(vm.PUSH0), byte(vm.SSTORE), byte(vm.STOP)})

			receipt, err := env.applyForkAware(t, &types.LegacyTx{
				Nonce:    0,
				To:       &governanceAddr,
				Gas:      100_000,
				GasPrice: big.NewInt(100 * ethparams.GWei),
				Data:     data,
			})
			require.NoError(t, err, "ApplyTransaction()")
			require.Equal(t, types.ReceiptStatusSuccessful, receipt.Status, "receipt status")

			assert.Equal(
				t,
				common.BytesToHash(tt.wantCoinbase.Bytes()),
				env.statedb.GetState(governanceAddr, common.Hash{}),
				"coinbase observed by the last governance-contract call",
			)
		})
	}
}

func TestApplyTransactionWithExtrasDaemonInvalidation(t *testing.T) {
	withCChainExtras(t, testApplyTransactionWithExtrasDaemonInvalidation)
}

// testApplyTransactionWithExtrasDaemonInvalidation asserts that a
// vm.EVM.InvalidateExecution raised DURING THE DAEMON CALL aborts the
// transaction. libevm's only invalidation check runs inside core.ApplyMessage,
// i.e. before the daemon call, so without the post-daemon re-check in
// applyTransactionWithExtras the invalidation would be silently discarded and
// a successful receipt returned. The trigger is coreth's pre-Granite guard
// (makePrecompile in graft/coreth/params/hooks_libevm.go): DELEGATECALL into a
// stateful precompile at block time >= corethparams.InvalidateDelegateUnix.
func testApplyTransactionWithExtrasDaemonInvalidation(t *testing.T) {
	var (
		mint      = uint256.NewInt(1e18)
		recipient = common.HexToAddress("0x00000000000000000000000000000000000000ee")
	)

	// ApricotPhase2 active from genesis and Granite unscheduled:
	// currentPrecompiles() returns the nativeasset builtins, wrapped with the
	// delegate-call guard. The daemon contract delegatecalls one of them and
	// still returns its 32-byte mint request.
	newEnv := func(t *testing.T, blockTime uint64) *applyTxEnv {
		env := newApplyTxEnv(t, corethparams.FlareChainID, mint)
		corethparams.WithExtra(env.config, extras.TestApricotPhase2Config)
		env.statedb.SetCode(daemonContractAddr,
			delegateCallMintReturnCode(nativeasset.NativeAssetBalanceAddr, mint))
		env.header.Time = blockTime
		return env
	}
	transfer := func(t *testing.T, env *applyTxEnv) (*types.Receipt, error) {
		return env.apply(t, &types.LegacyTx{
			Nonce:    0,
			To:       &recipient,
			Gas:      ethparams.TxGas,
			GasPrice: big.NewInt(100 * ethparams.GWei),
		})
	}

	t.Run("post_cutoff_daemon_invalidation_errors", func(t *testing.T) {
		env := newEnv(t, corethparams.InvalidateDelegateUnix)
		initial := env.statedb.GetBalance(env.sender).Clone()

		receipt, err := transfer(t, env)
		require.ErrorContains(t, err, "execution invalidated",
			"a daemon-triggered InvalidateExecution must surface as an error")
		assert.Nil(t, receipt, "no receipt for an invalidated execution")

		// The whole transaction is reverted to the pre-ApplyMessage snapshot,
		// mirroring libevm's own invalidation handling.
		assert.Equal(t, initial, env.statedb.GetBalance(env.sender), "sender balance reverted")
		assert.Zero(t, env.statedb.GetNonce(env.sender), "sender nonce reverted")
		assert.True(t, env.statedb.GetBalance(deadBurnAddress).IsZero(), "fee burn reverted")
		assert.True(t, env.statedb.GetBalance(daemonContractAddr).IsZero(), "daemon mint reverted")
	})

	t.Run("pre_cutoff_delegatecall_is_allowed", func(t *testing.T) {
		env := newEnv(t, corethparams.InvalidateDelegateUnix-1)
		receipt, err := transfer(t, env)
		require.NoError(t, err, "ApplyTransactionWithExtras()")
		require.Equal(t, types.ReceiptStatusSuccessful, receipt.Status, "receipt status")
		assert.Equal(t, mint, env.statedb.GetBalance(daemonContractAddr), "daemon mint request honoured")
	})
}
