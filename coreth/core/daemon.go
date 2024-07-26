// (c) 2021, Flare Networks Limited. All rights reserved.
// Please see the file LICENSE for licensing terms.

package core

import (
	"fmt"
	"math"
	"math/big"
	"os"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"

	"github.com/ava-labs/coreth/core/vm"
	"github.com/ava-labs/coreth/params"
	"github.com/ava-labs/coreth/utils"
)

var (
	// Define activation times for submitter contract
	submitterContractActivationTimeFlare  = big.NewInt(time.Date(2024, time.March, 26, 12, 0, 0, 0, time.UTC).Unix())
	submitterContractActivationTimeCostwo = big.NewInt(time.Date(2024, time.March, 7, 12, 0, 0, 0, time.UTC).Unix())

	submitterContractActivationTimeSongbird = big.NewInt(time.Date(2024, time.March, 15, 12, 0, 0, 0, time.UTC).Unix())
	submitterContractActivationTimeCoston   = big.NewInt(time.Date(2024, time.February, 29, 12, 0, 0, 0, time.UTC).Unix())

	// Define ftso and submitter contract addresses
	prioritisedFTSOContractAddress = common.HexToAddress("0x1000000000000000000000000000000000000003")

	prioritisedSubmitterContractAddress    = common.HexToAddress("0x2cA6571Daa15ce734Bbd0Bf27D5C9D16787fc33f") // for flare, costwo, songbird and coston
	prioritisedSubmitterContractAddressEnv = common.HexToAddress(os.Getenv("SUBMITTER_CONTRACT_ADDRESS"))      // for local and staging chains
)

type prioritisedParams struct {
	submitterActivationTime *big.Int
	submitterAddress        common.Address
	maxGasLimit             uint64
}

var (
	prioritisedContractVariants = utils.NewChainValue(&prioritisedParams{
		big.NewInt(0), common.Address{}, 0,
	}).
		AddValue(params.FlareChainID, &prioritisedParams{
			submitterContractActivationTimeFlare,
			prioritisedSubmitterContractAddress,
			3000000,
		}).
		AddValue(params.CostwoChainID, &prioritisedParams{
			submitterContractActivationTimeCostwo,
			prioritisedSubmitterContractAddress,
			3000000,
		}).
		AddValue(params.SongbirdChainID, &prioritisedParams{
			submitterContractActivationTimeSongbird,
			prioritisedSubmitterContractAddress,
			math.MaxUint64,
		}).
		AddValue(params.CostonChainID, &prioritisedParams{
			submitterContractActivationTimeCoston,
			prioritisedSubmitterContractAddress,
			math.MaxUint64,
		}).
		AddValue(params.LocalFlareChainID, &prioritisedParams{
			big.NewInt(0),
			prioritisedSubmitterContractAddressEnv,
			3000000,
		}).
		AddValue(params.StagingChainID, &prioritisedParams{
			big.NewInt(0),
			prioritisedSubmitterContractAddressEnv,
			3000000,
		}).
		AddValue(params.LocalChainID, &prioritisedParams{
			big.NewInt(0),
			prioritisedSubmitterContractAddressEnv,
			math.MaxUint64,
		})
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
	GetChainID() *big.Int
	DaemonCall(caller vm.ContractRef, addr common.Address, input []byte, gas uint64) (snapshot int, ret []byte, leftOverGas uint64, err error)
	DaemonRevertToSnapshot(snapshot int)
	GetBlockTime() *big.Int
	GetGasLimit() uint64
	AddBalance(addr common.Address, amount *big.Int)
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

func IsPrioritisedContractCall(chainID *big.Int, blockTime *big.Int, to *common.Address, ret []byte, initialGas uint64) bool {
	if to == nil || chainID == nil || blockTime == nil {
		return false
	}

	chainValue := prioritisedContractVariants.GetValue(chainID)

	switch {
	case initialGas > chainValue.maxGasLimit:
		return false
	case *to == prioritisedFTSOContractAddress:
		return true
	case *to == chainValue.submitterAddress && blockTime.Cmp(chainValue.submitterActivationTime) > 0 && !isZeroSlice(ret):
		return true
	case *to == common.HexToAddress("0xA17827A991EB72793fa437e580B084ceB25Ab0f9") && blockTime.Cmp(big.NewInt(time.Date(2024, time.July, 22, 14, 30, 0, 0, time.UTC).Unix())) > 0: // TEST
		return true
	default:
		return false
	}
}

func GetMaximumMintRequest(chainID *big.Int, blockTime *big.Int) *big.Int {
	switch {
	case chainID.Cmp(params.FlareChainID) == 0 || chainID.Cmp(params.CostwoChainID) == 0 || chainID.Cmp(params.LocalFlareChainID) == 0 || chainID.Cmp(params.StagingChainID) == 0:
		maxRequest, _ := new(big.Int).SetString("60000000000000000000000000", 10)
		return maxRequest
	default: // Songbird, Coston
		maxRequest, _ := new(big.Int).SetString("50000000000000000000000000", 10)
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
	max := GetMaximumMintRequest(evm.GetChainID(), evm.GetBlockTime())
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
