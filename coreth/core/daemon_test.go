// (c) 2021, Flare Networks Limited. All rights reserved.
// Please see the file LICENSE for licensing terms.

package core

import (
	"context"
	"errors"
	"math/big"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"
	"github.com/holiman/uint256"

	"github.com/ava-labs/coreth/core/vm"
	"github.com/ava-labs/coreth/params"

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
	atomicDaemonAndMint(badDaemonCallEVMMock, loggerMock)

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
	atomicDaemonAndMint(defaultEVMMock, log)

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
	atomicDaemonAndMint(defaultEVMMock, log)

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
