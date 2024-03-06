// (c) 2021, Flare Networks Limited. All rights reserved.
// Please see the file LICENSE for licensing terms.

package core

import (
	"fmt"
	"math/big"
	"os"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"

	"github.com/ava-labs/coreth/core/vm"
	"github.com/ava-labs/coreth/params"
)

var (
	// Define activation times for submitter contract
	submitterContractActivationTimeFlare  = big.NewInt(time.Date(2024, time.March, 26, 12, 0, 0, 0, time.UTC).Unix())
	submitterContractActivationTimeCostwo = big.NewInt(time.Date(2024, time.March, 7, 12, 0, 0, 0, time.UTC).Unix())

	// Define ftso and submitter contract addresses
	prioritisedFTSOContractAddress = common.HexToAddress("0x1000000000000000000000000000000000000003")

	prioritisedSubmitterContractAddress    = common.HexToAddress("0x2cA6571Daa15ce734Bbd0Bf27D5C9D16787fc33f")
	prioritisedSubmitterContractAddressEnv = common.HexToAddress(os.Getenv("SUBMITTER_CONTRACT_ADDRESS")) // for local and staging chains
)

// Define errors
type ErrInvalidDaemonData struct{}

func (e *ErrInvalidDaemonData) Error() string { return "invalid return data from daemon" }

type ErrDaemonDataEmpty struct{}

func (e *ErrDaemonDataEmpty) Error() string { return "return data from daemon empty" }

type ErrMaxMintExceeded struct {
	mintMax     *big.Int
	mintRequest *big.Int
}

func (e *ErrMaxMintExceeded) Error() string {
	return fmt.Sprintf("mint request of %s exceeded max of %s", e.mintRequest.Text(10), e.mintMax.Text(10))
}

type ErrMintNegative struct{}

func (e *ErrMintNegative) Error() string { return "mint request cannot be negative" }

// Define interface for dependencies
type EVMCaller interface {
	DaemonCall(caller vm.ContractRef, addr common.Address, input []byte, gas uint64) (snapshot int, ret []byte, leftOverGas uint64, err error)
	DaemonRevertToSnapshot(snapshot int)
	GetBlockTime() *big.Int
	GetGasLimit() uint64
	AddBalance(addr common.Address, amount *big.Int)
}

// Define maximums that can change by block time
func GetMaxFTSOGasLimit(blockTime *big.Int) uint64 {
	switch {
	default:
		return 3000000
	}
}

func GetDaemonGasMultiplier(blockTime *big.Int) uint64 {
	switch {
	default:
		return 100
	}
}

func GetDaemonContractAddr(blockTime *big.Int) string {
	switch {
	default:
		return "0x1000000000000000000000000000000000000002"
	}
}

func GetDaemonSelector(blockTime *big.Int) []byte {
	switch {
	default:
		return []byte{0x7f, 0xec, 0x8d, 0x38}
	}
}

func isPrioritisedFTSOContract(to *common.Address) bool {
	return to != nil && *to == prioritisedFTSOContractAddress
}

func isPrioritisedSubmitterContract(chainID *big.Int, to *common.Address, blockTime *big.Int) bool {
	switch {
	case to == nil || chainID == nil || blockTime == nil:
		return false
	case chainID.Cmp(params.FlareChainID) == 0:
		return *to == prioritisedSubmitterContractAddress &&
			blockTime.Cmp(submitterContractActivationTimeFlare) > 0
	case chainID.Cmp(params.CostwoChainID) == 0:
		return *to == prioritisedSubmitterContractAddress &&
			blockTime.Cmp(submitterContractActivationTimeCostwo) > 0
	case chainID.Cmp(params.LocalFlareChainID) == 0 || chainID.Cmp(params.StagingChainID) == 0:
		return *to == prioritisedSubmitterContractAddressEnv
	default:
		return false
	}
}

func IsPrioritisedContractCall(chainID *big.Int, to *common.Address, ret []byte, blockTime *big.Int) bool {
	switch {
	case isPrioritisedFTSOContract(to):
		return true
	case isPrioritisedSubmitterContract(chainID, to, blockTime):
		return !isZeroSlice(ret)
	default:
		return false
	}
}

func GetMaximumMintRequest(blockTime *big.Int) *big.Int {
	switch {
	default:
		maxRequest, _ := new(big.Int).SetString("60000000000000000000000000", 10)
		return maxRequest
	}
}

func daemon(evm EVMCaller) (int, *big.Int, error) {
	bigZero := big.NewInt(0)
	// Get the contract to call
	daemonContract := common.HexToAddress(GetDaemonContractAddr(evm.GetBlockTime()))

	// Call the method
	daemonSnapshot, daemonRet, _, daemonErr := evm.DaemonCall(
		vm.AccountRef(daemonContract),
		daemonContract,
		GetDaemonSelector(evm.GetBlockTime()),
		GetDaemonGasMultiplier(evm.GetBlockTime())*evm.GetGasLimit())
	// If no error and a value came back...
	if daemonErr == nil && daemonRet != nil {
		// Did we get one big int?
		if len(daemonRet) == 32 {
			// Convert to big int
			// Mint request cannot be less than 0 as SetBytes treats value as unsigned
			mintRequest := new(big.Int).SetBytes(daemonRet)
			// return the mint request
			return daemonSnapshot, mintRequest, nil
		} else {
			// Returned length was not 32 bytes
			return 0, bigZero, &ErrInvalidDaemonData{}
		}
	} else {
		if daemonErr != nil {
			return 0, bigZero, daemonErr
		} else {
			return 0, bigZero, &ErrDaemonDataEmpty{}
		}
	}
}

func mint(evm EVMCaller, mintRequest *big.Int) error {
	// If the mint request is greater than zero and less than max
	max := GetMaximumMintRequest(evm.GetBlockTime())
	if mintRequest.Cmp(big.NewInt(0)) > 0 &&
		mintRequest.Cmp(max) <= 0 {
		// Mint the amount asked for on to the daemon contract
		evm.AddBalance(common.HexToAddress(GetDaemonContractAddr(evm.GetBlockTime())), mintRequest)
	} else if mintRequest.Cmp(max) > 0 {
		// Return error
		return &ErrMaxMintExceeded{
			mintRequest: mintRequest,
			mintMax:     max,
		}
	} else if mintRequest.Cmp(big.NewInt(0)) < 0 {
		// Cannot mint negatives
		return &ErrMintNegative{}
	}
	// No error
	return nil
}

func atomicDaemonAndMint(evm EVMCaller, log log.Logger) {
	// Call the daemon
	daemonSnapshot, mintRequest, daemonErr := daemon(evm)
	// If no error...
	if daemonErr == nil {
		// time to mint
		if mintError := mint(evm, mintRequest); mintError != nil {
			log.Warn("Error minting inflation request", "error", mintError)
			// Revert to snapshot to unwind daemon state transition
			evm.DaemonRevertToSnapshot(daemonSnapshot)
		}
	} else {
		log.Warn("Daemon error", "error", daemonErr)
	}
}

func isZeroSlice(s []byte) bool {
	for i := len(s) - 1; i >= 0; i-- {
		if s[i] != 0 {
			return false
		}
	}
	return true
}
