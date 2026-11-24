// (c) 2021, Flare Networks Limited. All rights reserved.
// Please see the file LICENSE for licensing terms.

package core

import (
	"context"
	"errors"
	"math/big"
	"testing"
	"time"

	"github.com/holiman/uint256"

	"github.com/ava-labs/avalanchego/graft/coreth/params"
	"github.com/ava-labs/libevm/common"
	"github.com/ava-labs/libevm/core/vm"
	"github.com/ava-labs/libevm/log"

	"golang.org/x/exp/slog"
)

// Define a mock structure to spy and mock values for daemon calls
type MockEVMCallerData struct {
	callCalls             int
	addBalanceCalls       int
	revertToSnapshotCalls int
	lastSnapshotValue     int
	blockTime             uint64
	gasLimit              uint64
	mintRequestReturn     uint256.Int
	lastAddBalanceAddr    common.Address
	lastAddBalanceAmount  *uint256.Int
}

// Define a mock structure to spy and mock values for logger calls
type MockLoggerData struct {
	warnCalls int
}

// Set up default mock method calls
func defaultDaemonCall(e *MockEVMCallerData, caller vm.ContractRef, addr common.Address, input []byte, gas uint64) (snapshot int, ret []byte, leftOverGas uint64, err error) {
	e.callCalls++

	bytes := e.mintRequestReturn.Bytes32()
	return 0, bytes[:], 0, nil
}

func defaultRevertToSnapshot(e *MockEVMCallerData, snapshot int) {
	e.revertToSnapshotCalls++
	e.lastSnapshotValue = snapshot
}

func defaultGetBlockTime(e *MockEVMCallerData) uint64 {
	return e.blockTime
}

func defaultGetGasLimit(e *MockEVMCallerData) uint64 {
	return e.gasLimit
}

func defaultAddBalance(e *MockEVMCallerData, addr common.Address, amount *uint256.Int) {
	e.addBalanceCalls++
	e.lastAddBalanceAddr = addr
	e.lastAddBalanceAmount = amount
}

// Define the default EVM mock and define default mock receiver functions
type DefaultEVMMock struct {
	mockEVMCallerData MockEVMCallerData
}

func (e *DefaultEVMMock) DaemonCall(caller vm.ContractRef, addr common.Address, input []byte, gas uint64) (snapshot int, ret []byte, leftOverGas uint64, err error) {
	return defaultDaemonCall(&e.mockEVMCallerData, caller, addr, input, gas)
}

func (e *DefaultEVMMock) DaemonRevertToSnapshot(snapshot int) {
	defaultRevertToSnapshot(&e.mockEVMCallerData, snapshot)
}

func (e *DefaultEVMMock) GetBlockTime() uint64 {
	return defaultGetBlockTime(&e.mockEVMCallerData)
}

func (e *DefaultEVMMock) GetGasLimit() uint64 {
	return defaultGetGasLimit(&e.mockEVMCallerData)
}

func (e *DefaultEVMMock) GetChainID() *big.Int {
	return params.FlareChainID
}

func (e *DefaultEVMMock) AddBalance(addr common.Address, amount *uint256.Int) {
	defaultAddBalance(&e.mockEVMCallerData, addr, amount)
}

func TestDaemonShouldReturnMintRequest(t *testing.T) {
	mintRequestReturn := new(uint256.Int)
	mintRequestReturn.SetFromDecimal("60000000000000000000000000")
	mockEVMCallerData := &MockEVMCallerData{
		blockTime:         0,
		gasLimit:          0,
		mintRequestReturn: *mintRequestReturn,
	}
	defaultEVMMock := &DefaultEVMMock{
		mockEVMCallerData: *mockEVMCallerData,
	}

	_, mintRequest, _ := daemon(defaultEVMMock)

	if mintRequest.Cmp(mintRequestReturn) != 0 {
		t.Errorf("got %s want %q", mintRequest.String(), "60000000000000000000000000")
	}
}

func TestDaemonShouldNotLetMintRequestOverflow(t *testing.T) {
	var mintRequestReturn uint256.Int
	// TODO: Compact with exponent?
	buffer := []byte{0, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255}
	mintRequestReturn.SetBytes(buffer)

	mockEVMCallerData := &MockEVMCallerData{
		blockTime:         0,
		gasLimit:          0,
		mintRequestReturn: mintRequestReturn,
	}
	defaultEVMMock := &DefaultEVMMock{
		mockEVMCallerData: *mockEVMCallerData,
	}

	snapshot, mintRequest, mintRequestError := daemon(defaultEVMMock)

	if mintRequestError != nil {
		t.Errorf("received unexpected error %s", mintRequestError)
	}

	if mintRequest.Sign() < 1 {
		t.Errorf("unexpected mintRequest negative")
	}

	if snapshot < 0 {
		t.Errorf("unexpected snapshot negative")
	}
}

// Define a bad mint request return size mock
type BadMintReturnSizeEVMMock struct {
	mockEVMCallerData MockEVMCallerData
}

func (e *BadMintReturnSizeEVMMock) DaemonCall(caller vm.ContractRef, addr common.Address, input []byte, gas uint64) (snapshot int, ret []byte, leftOverGas uint64, err error) {
	e.mockEVMCallerData.callCalls++
	bytes := e.mockEVMCallerData.mintRequestReturn.Bytes()
	return 0, bytes[:], 0, nil
}

func (e *BadMintReturnSizeEVMMock) DaemonRevertToSnapshot(snapshot int) {
	defaultRevertToSnapshot(&e.mockEVMCallerData, snapshot)
}

func (e *BadMintReturnSizeEVMMock) GetBlockTime() uint64 {
	return defaultGetBlockTime(&e.mockEVMCallerData)
}

func (e *BadMintReturnSizeEVMMock) GetGasLimit() uint64 {
	return defaultGetGasLimit(&e.mockEVMCallerData)
}

func (e *BadMintReturnSizeEVMMock) GetChainID() *big.Int {
	return params.FlareChainID
}

func (e *BadMintReturnSizeEVMMock) AddBalance(addr common.Address, amount *uint256.Int) {
	defaultAddBalance(&e.mockEVMCallerData, addr, amount)
}

func TestDaemonValidatesMintRequestReturnValueSize(t *testing.T) {
	var mintRequestReturn uint256.Int
	// TODO: Compact with exponent?
	buffer := []byte{255}
	mintRequestReturn.SetBytes(buffer)

	mockEVMCallerData := &MockEVMCallerData{
		blockTime:         0,
		gasLimit:          0,
		mintRequestReturn: mintRequestReturn,
	}
	badMintReturnSizeEVMMock := &BadMintReturnSizeEVMMock{
		mockEVMCallerData: *mockEVMCallerData,
	}
	// Call to return less than 32 bytes
	_, _, err := daemon(badMintReturnSizeEVMMock)

	if err != nil {
		if err, ok := err.(*ErrInvalidDaemonData); !ok {
			want := &ErrInvalidDaemonData{}
			t.Errorf("got '%s' want '%s'", err.Error(), want.Error())
		}
	} else {
		t.Errorf("no error returned as expected")
	}
}

// Define a mock to simulate daemon returning an error from Call
type BadDaemonCallEVMMock struct {
	mockEVMCallerData MockEVMCallerData
}

func (e *BadDaemonCallEVMMock) DaemonCall(caller vm.ContractRef, addr common.Address, input []byte, gas uint64) (snapshot int, ret []byte, leftOverGas uint64, err error) {
	e.mockEVMCallerData.callCalls++

	bytes := e.mockEVMCallerData.mintRequestReturn.Bytes32()
	return 0, bytes[:], 0, errors.New("Call error happened")
}

func (e *BadDaemonCallEVMMock) DaemonRevertToSnapshot(snapshot int) {
	defaultRevertToSnapshot(&e.mockEVMCallerData, snapshot)
}

func (e *BadDaemonCallEVMMock) GetBlockTime() uint64 {
	return defaultGetBlockTime(&e.mockEVMCallerData)
}

func (e *BadDaemonCallEVMMock) GetGasLimit() uint64 {
	return defaultGetGasLimit(&e.mockEVMCallerData)
}

func (e *BadDaemonCallEVMMock) GetChainID() *big.Int {
	return params.FlareChainID
}

func (e *BadDaemonCallEVMMock) AddBalance(addr common.Address, amount *uint256.Int) {
	defaultAddBalance(&e.mockEVMCallerData, addr, amount)
}

func TestDaemonReturnsCallError(t *testing.T) {
	mockEVMCallerData := &MockEVMCallerData{}
	badDaemonCallEVMMock := &BadDaemonCallEVMMock{
		mockEVMCallerData: *mockEVMCallerData,
	}
	// Call to return less than 32 bytes
	_, _, err := daemon(badDaemonCallEVMMock)

	if err == nil {
		t.Errorf("no error received")
	} else {
		if err.Error() != "Call error happened" {
			t.Errorf("did not get expected error")
		}
	}
}

type LoggerMock struct {
	mockLoggerData MockLoggerData
}

func (l *LoggerMock) New(ctx ...interface{}) log.Logger {
	return nil
}
func (l *LoggerMock) With(ctx ...interface{}) log.Logger {
	return nil
}

func (l *LoggerMock) Trace(msg string, ctx ...interface{})             {}
func (l *LoggerMock) Debug(msg string, ctx ...interface{})             {}
func (l *LoggerMock) Info(msg string, ctx ...interface{})              {}
func (l *LoggerMock) Error(msg string, ctx ...interface{})             {}
func (l *LoggerMock) Crit(msg string, ctx ...interface{})              {}
func (l *LoggerMock) Write(level slog.Level, msg string, attrs ...any) {}

func (l *LoggerMock) Log(level slog.Level, msg string, ctx ...interface{}) {
	if level == slog.LevelWarn {
		l.mockLoggerData.warnCalls++
	}
}

func (l *LoggerMock) Warn(msg string, ctx ...interface{}) {
	l.mockLoggerData.warnCalls++
}

func (l *LoggerMock) Enabled(ctx context.Context, level slog.Level) bool {
	return true
}

func TestAtomicDaemonAndMintLogsError(t *testing.T) {
	// Assemble
	// Set up mock EVM call to return an error
	mockEVMCallerData := &MockEVMCallerData{}
	badDaemonCallEVMMock := &BadDaemonCallEVMMock{
		mockEVMCallerData: *mockEVMCallerData,
	}
	// Set up a mock logger
	mockLoggerData := &MockLoggerData{}
	loggerMock := &LoggerMock{
		mockLoggerData: *mockLoggerData,
	}

	// Act
	AtomicDaemonAndMint(badDaemonCallEVMMock, loggerMock)

	// Assert
	if loggerMock.mockLoggerData.warnCalls != 1 {
		t.Errorf("Logger.Warn not called as expected")
	}
}

// Define a mock to simulate daemon returning nil for mint request
type ReturnNilMintRequestEVMMock struct {
	mockEVMCallerData MockEVMCallerData
}

func (e *ReturnNilMintRequestEVMMock) DaemonCall(caller vm.ContractRef, addr common.Address, input []byte, gas uint64) (snapshot int, ret []byte, leftOverGas uint64, err error) {
	e.mockEVMCallerData.callCalls++

	return 0, nil, 0, nil
}

func (e *ReturnNilMintRequestEVMMock) DaemonRevertToSnapshot(snapshot int) {
	defaultRevertToSnapshot(&e.mockEVMCallerData, snapshot)
}

func (e *ReturnNilMintRequestEVMMock) GetBlockTime() uint64 {
	return defaultGetBlockTime(&e.mockEVMCallerData)
}

func (e *ReturnNilMintRequestEVMMock) GetGasLimit() uint64 {
	return defaultGetGasLimit(&e.mockEVMCallerData)
}

func (e *ReturnNilMintRequestEVMMock) GetChainID() *big.Int {
	return params.FlareChainID
}

func (e *ReturnNilMintRequestEVMMock) AddBalance(addr common.Address, amount *uint256.Int) {
	defaultAddBalance(&e.mockEVMCallerData, addr, amount)
}

func TestDaemonHandlesNilMintRequest(t *testing.T) {
	mockEVMCallerData := &MockEVMCallerData{}
	returnNilMintRequestEVMMock := &ReturnNilMintRequestEVMMock{
		mockEVMCallerData: *mockEVMCallerData,
	}
	// Call to return less than 32 bytes
	_, _, err := daemon(returnNilMintRequestEVMMock)

	if err != nil {
		if err, ok := err.(*ErrDaemonDataEmpty); !ok {
			want := &ErrDaemonDataEmpty{}
			t.Errorf("got '%s' want '%s'", err.Error(), want.Error())
		}
	} else {
		t.Errorf("no error returned as expected")
	}
}

func TestDaemonShouldNotMintMoreThanMax(t *testing.T) {
	mintRequest := new(uint256.Int)
	mintRequest.SetFromDecimal("60000000000000000000000001")
	mockEVMCallerData := &MockEVMCallerData{
		blockTime:         0,
		gasLimit:          0,
		mintRequestReturn: *uint256.NewInt(0),
	}
	defaultEVMMock := &DefaultEVMMock{
		mockEVMCallerData: *mockEVMCallerData,
	}

	err := mint(defaultEVMMock, mintRequest)

	if err != nil {
		if err, ok := err.(*ErrMaxMintExceeded); !ok {
			want := &ErrMaxMintExceeded{
				mintRequest: mintRequest,
				mintMax:     GetMaximumMintRequest(params.FlareChainID, 0),
			}
			t.Errorf("got '%s' want '%s'", err.Error(), want.Error())
		}
	} else {
		t.Errorf("no error returned as expected")
	}
}

func TestDaemonShouldMint(t *testing.T) {
	// Assemble
	mintRequest := new(uint256.Int)
	mintRequest.SetFromDecimal("60000000000000000000000000")
	mockEVMCallerData := &MockEVMCallerData{
		blockTime:         0,
		gasLimit:          0,
		mintRequestReturn: *uint256.NewInt(0),
	}
	defaultEVMMock := &DefaultEVMMock{
		mockEVMCallerData: *mockEVMCallerData,
	}

	// Act
	err := mint(defaultEVMMock, mintRequest)

	// Assert
	if err == nil {
		if defaultEVMMock.mockEVMCallerData.addBalanceCalls != 1 {
			t.Errorf("AddBalance not called as expected")
		}
		if defaultEVMMock.mockEVMCallerData.lastAddBalanceAddr.String() != GetDaemonContractAddr(0) {
			t.Errorf("wanted addr %s; got addr %s", GetDaemonContractAddr(0), defaultEVMMock.mockEVMCallerData.lastAddBalanceAddr)
		}
		if defaultEVMMock.mockEVMCallerData.lastAddBalanceAmount.Cmp(mintRequest) != 0 {
			t.Errorf("wanted amount %s; got amount %s", mintRequest.String(), defaultEVMMock.mockEVMCallerData.lastAddBalanceAmount.String())
		}
	} else {
		t.Errorf("unexpected error returned; was = %s", err.Error())
	}
}

func TestDaemonShouldNotErrorMintingZero(t *testing.T) {
	// Assemble
	mintRequest := uint256.NewInt(0)
	mockEVMCallerData := &MockEVMCallerData{
		blockTime:         0,
		gasLimit:          0,
		mintRequestReturn: *uint256.NewInt(0),
	}
	defaultEVMMock := &DefaultEVMMock{
		mockEVMCallerData: *mockEVMCallerData,
	}

	// Act
	err := mint(defaultEVMMock, mintRequest)

	// Assert
	if err == nil {
		if defaultEVMMock.mockEVMCallerData.addBalanceCalls != 0 {
			t.Errorf("AddBalance called unexpectedly")
		}
	} else {
		t.Errorf("unexpected error returned; was %s", err.Error())
	}
}

func TestDaemonFiredAndMinted(t *testing.T) {
	mintRequestReturn := new(uint256.Int)
	mintRequestReturn.SetFromDecimal("60000000000000000000000000")
	mockEVMCallerData := &MockEVMCallerData{
		blockTime:         0,
		gasLimit:          0,
		mintRequestReturn: *mintRequestReturn,
	}
	defaultEVMMock := &DefaultEVMMock{
		mockEVMCallerData: *mockEVMCallerData,
	}

	log := log.New()
	AtomicDaemonAndMint(defaultEVMMock, log)

	// EVM Call function calling the daemon should have been cqlled
	if defaultEVMMock.mockEVMCallerData.callCalls != 1 {
		t.Errorf("EVM Call count not as expected. got %d want 1", defaultEVMMock.mockEVMCallerData.callCalls)
	}
	// AddBalance should have been called on the state database, minting the request asked for
	if defaultEVMMock.mockEVMCallerData.addBalanceCalls != 1 {
		t.Errorf("Add balance call count not as expected. got %d want 1", defaultEVMMock.mockEVMCallerData.addBalanceCalls)
	}
}

func TestDaemonShouldNotMintMoreThanLimit(t *testing.T) {
	mintRequestReturn := new(uint256.Int)
	mintRequestReturn.SetFromDecimal("60000000000000000000000001")
	mockEVMCallerData := &MockEVMCallerData{
		blockTime:         0,
		gasLimit:          0,
		mintRequestReturn: *mintRequestReturn,
	}
	defaultEVMMock := &DefaultEVMMock{
		mockEVMCallerData: *mockEVMCallerData,
	}

	log := log.New()
	AtomicDaemonAndMint(defaultEVMMock, log)

	// EVM Call function calling the daemon should have been called
	if defaultEVMMock.mockEVMCallerData.callCalls != 1 {
		t.Errorf("EVM Call count not as expected. got %d want 1", defaultEVMMock.mockEVMCallerData.callCalls)
	}
	// AddBalance should not have been called on the state database, as the mint request was over the limit
	if defaultEVMMock.mockEVMCallerData.addBalanceCalls != 0 {
		t.Errorf("Add balance call count not as expected. got %d want 1", defaultEVMMock.mockEVMCallerData.addBalanceCalls)
	}
}

func TestPrioritisedContract(t *testing.T) {
	address := common.HexToAddress("0x123456789aBCdEF123456789aBCdef123456789A")
	preForkTime := uint64(time.Date(2024, time.March, 20, 12, 0, 0, 0, time.UTC).Unix())
	postForkTime := uint64(time.Date(2024, time.March, 27, 12, 0, 0, 0, time.UTC).Unix())
	postPrefixForkTime := uint64(time.Date(2024, time.October, 11, 0, 0, 0, 0, time.UTC).Unix())
	initialGas := uint64(0)
	ret0 := [32]byte{}
	ret1 := [32]byte{}
	ret1[31] = 1
	data := []byte{0x01, 0x02, 0x03, 0x04, 0x05}

	if IsPrioritisedContractCall(params.FlareChainID, preForkTime, &address, data, nil, initialGas) {
		t.Errorf("Expected false for wrong address")
	}
	if !IsPrioritisedContractCall(params.FlareChainID, preForkTime, &prioritisedFTSOContractAddress, nil, nil, initialGas) {
		t.Errorf("Expected true for FTSO contract")
	}
	if IsPrioritisedContractCall(params.FlareChainID, preForkTime, &prioritisedSubmitterContractAddress, data, ret1[:], initialGas) {
		t.Errorf("Expected false for submitter contract before activation")
	}
	if !IsPrioritisedContractCall(params.FlareChainID, postForkTime, &prioritisedSubmitterContractAddress, data, ret1[:], initialGas) {
		t.Errorf("Expected true for submitter contract after activation")
	}
	if IsPrioritisedContractCall(params.FlareChainID, postForkTime, &prioritisedSubmitterContractAddress, data, ret0[:], initialGas) {
		t.Errorf("Expected false for submitter contract with wrong return value")
	}
	if IsPrioritisedContractCall(params.FlareChainID, postForkTime, &prioritisedSubmitterContractAddress, data, nil, initialGas) {
		t.Errorf("Expected false for submitter contract with no return value")
	}
	if IsPrioritisedContractCall(params.FlareChainID, postPrefixForkTime, &prioritisedSubmitterContractAddress, data, ret1[:], initialGas) {
		t.Errorf("Expected false for submitter contract after prefix activation with wrong data")
	}
	if !IsPrioritisedContractCall(params.FlareChainID, postPrefixForkTime, &prioritisedSubmitterContractAddress, []byte{0xe1, 0xb1, 0x57, 0xe7, 0x00, 0x00}, ret1[:], initialGas) {
		t.Errorf("Expected true for submitter contract after prefix activation with correct data")
	}
	if IsPrioritisedContractCall(params.FlareChainID, postPrefixForkTime, &prioritisedSubmitterContractAddress, make([]byte, prioritisedCallDataCap+1), ret1[:], initialGas) {
		t.Errorf("Expected false for submitter contract after prefix activation with too long data")
	}
	if IsPrioritisedContractCall(params.FlareChainID, postPrefixForkTime, &prioritisedFTSOContractAddress, data, nil, initialGas) {
		t.Errorf("Expected false for FTSO contract after prefix activation with wrong data")
	}
	if !IsPrioritisedContractCall(params.FlareChainID, postPrefixForkTime, &prioritisedFTSOContractAddress, []byte{0x8f, 0xc6, 0xf6, 0x67, 0x05}, nil, initialGas) {
		t.Errorf("Expected true for FTSO contract after prefix activation with correct data")
	}
}

// TestPrioritisedContractGasLimitCapSongbirdNetworks pins the time-gated rules
// of prioritised calls on Songbird and Coston. Up to and including the
// activation time any gas limit is refunded, on the FTSO and the submitter
// branch alike. After it the submitter branch is capped at
// prioritisedMaxGasLimit (the Flare cap) and the FTSO v1 branch is retired: no
// call to the FTSO contract is prioritised any more, whatever its gas limit.
// Flare keeps its cap and its FTSO branch. The local networks define no FTSO
// prefixes, so the branch never matches there.
func TestPrioritisedContractGasLimitCapSongbirdNetworks(t *testing.T) {
	ret1 := [32]byte{}
	ret1[31] = 1
	ftsoData := []byte{0xc5, 0xad, 0xc5, 0x39, 0x05}       // submitPriceHashes selector (Songbird networks)
	ftsoRevealData := []byte{0x60, 0x84, 0x8b, 0x44, 0x05} // revealPrices selector (Songbird networks)
	submitterData := []byte{0x6c, 0x53, 0x2f, 0xae, 0x05}  // submit1 selector
	const blockGasLimit = 70_000_000                       // the SAE block gas limit on Songbird at the time of writing

	for _, test := range []struct {
		name       string
		chainID    *big.Int
		activation uint64
	}{
		{"songbird", params.SongbirdChainID, prioritisedGasLimitCapActivationTimeSongbird},
		{"coston", params.CostonChainID, prioritisedGasLimitCapActivationTimeCoston},
	} {
		t.Run(test.name, func(t *testing.T) {
			for _, blockTime := range []uint64{test.activation - 1, test.activation} {
				for _, data := range [][]byte{ftsoData, ftsoRevealData} {
					if !IsPrioritisedContractCall(test.chainID, blockTime, &prioritisedFTSOContractAddress, data, nil, blockGasLimit) {
						t.Errorf("block time %d: expected true for FTSO call %x with gas limit %d before the cap activation", blockTime, data[:4], blockGasLimit)
					}
				}
				if !IsPrioritisedContractCall(test.chainID, blockTime, &prioritisedSubmitterContractAddress, submitterData, ret1[:], blockGasLimit) {
					t.Errorf("block time %d: expected true for submitter call with gas limit %d before the cap activation", blockTime, blockGasLimit)
				}
			}

			// The FTSO v1 branch is retired at the activation: not even a call
			// within the cap is prioritised.
			after := test.activation + 1
			for _, data := range [][]byte{ftsoData, ftsoRevealData} {
				for _, gasLimit := range []uint64{blockGasLimit, prioritisedMaxGasLimit + 1, prioritisedMaxGasLimit, 100_000} {
					if IsPrioritisedContractCall(test.chainID, after, &prioritisedFTSOContractAddress, data, nil, gasLimit) {
						t.Errorf("expected false for FTSO call %x with gas limit %d after the cap activation", data[:4], gasLimit)
					}
				}
			}
			if IsPrioritisedContractCall(test.chainID, after, &prioritisedSubmitterContractAddress, submitterData, ret1[:], prioritisedMaxGasLimit+1) {
				t.Errorf("expected false for submitter call with gas limit %d after the cap activation", prioritisedMaxGasLimit+1)
			}
			if !IsPrioritisedContractCall(test.chainID, after, &prioritisedSubmitterContractAddress, submitterData, ret1[:], prioritisedMaxGasLimit) {
				t.Errorf("expected true for submitter call with gas limit %d after the cap activation", prioritisedMaxGasLimit)
			}
		})
	}

	// Flare and Coston2 have had the cap since the beginning and keep their
	// FTSO branch (their PriceSubmitter rejects senders that are not
	// whitelisted).
	flareFTSOData := []byte{0x8f, 0xc6, 0xf6, 0x67, 0x05} // submitHash selector (Flare networks)
	blockTime := uint64(time.Date(2027, time.January, 1, 0, 0, 0, 0, time.UTC).Unix())
	for _, chainID := range []*big.Int{params.FlareChainID, params.CostwoChainID} {
		if IsPrioritisedContractCall(chainID, blockTime, &prioritisedFTSOContractAddress, flareFTSOData, nil, prioritisedMaxGasLimit+1) {
			t.Errorf("chain %s: expected false for Flare FTSO call with gas limit %d", chainID, prioritisedMaxGasLimit+1)
		}
		if !IsPrioritisedContractCall(chainID, blockTime, &prioritisedFTSOContractAddress, flareFTSOData, nil, prioritisedMaxGasLimit) {
			t.Errorf("chain %s: expected true for Flare FTSO call with gas limit %d", chainID, prioritisedMaxGasLimit)
		}
	}

	// The local networks define no FTSO prefixes, so no call to the FTSO
	// contract is prioritised there at any block time above zero.
	for _, chainID := range []*big.Int{params.LocalChainID, params.LocalFlareChainID} {
		for _, data := range [][]byte{ftsoData, ftsoRevealData, flareFTSOData} {
			for _, localTime := range []uint64{1, blockTime} {
				if IsPrioritisedContractCall(chainID, localTime, &prioritisedFTSOContractAddress, data, nil, 100_000) {
					t.Errorf("chain %s, block time %d: expected false for FTSO call %x on a local network", chainID, localTime, data[:4])
				}
			}
		}
	}
}
