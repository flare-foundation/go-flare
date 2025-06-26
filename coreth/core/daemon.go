// (c) 2021, Flare Networks Limited. All rights reserved.
// Please see the file LICENSE for licensing terms.

package core

import (
	"bytes"
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
	submitterContractActivationTimeFlare  = uint64(time.Date(2024, time.March, 26, 12, 0, 0, 0, time.UTC).Unix())
	submitterContractActivationTimeCostwo = uint64(time.Date(2024, time.March, 7, 12, 0, 0, 0, time.UTC).Unix())

	submitterContractActivationTimeSongbird = uint64(time.Date(2024, time.March, 15, 12, 0, 0, 0, time.UTC).Unix())
	submitterContractActivationTimeCoston   = uint64(time.Date(2024, time.February, 29, 12, 0, 0, 0, time.UTC).Unix())

	// Define ftso and submitter contract addresses
	prioritisedFTSOContractAddress = common.HexToAddress("0x1000000000000000000000000000000000000003")

	prioritisedSubmitterContractAddress    = common.HexToAddress("0x2cA6571Daa15ce734Bbd0Bf27D5C9D16787fc33f") // for flare, costwo, songbird and coston
	prioritisedSubmitterContractAddressEnv = common.HexToAddress(os.Getenv("SUBMITTER_CONTRACT_ADDRESS"))      // for local and staging chains

	// Define data prefixes for submitter and prioritized ftso contracts
	submitterDataPrefixes = [][4]byte{
		{0x6c, 0x53, 0x2f, 0xae},
		{0x9d, 0x00, 0xc9, 0xfd},
		{0xe1, 0xb1, 0x57, 0xe7},
		{0x57, 0xee, 0xd5, 0x80},
		{0x83, 0x3b, 0xf6, 0xc0},
	}

	prioritisedFTSOContractDataPrefixesFlareNetworks = [][4]byte{
		{0x8f, 0xc6, 0xf6, 0x67},
		{0xe2, 0xdb, 0x5a, 0x52},
	}

	prioritisedFTSOContractDataPrefixesSongbirdNetworks = [][4]byte{
		{0xc5, 0xad, 0xc5, 0x39},
		{0x60, 0x84, 0x8b, 0x44},
	}
)

const (
	prioritisedCallDataCap = 4500 // 4500 bytes
)

type prioritisedParams struct {
	submitterActivationTime  uint64
	submitterAddress         common.Address
	maxGasLimit              uint64
	dataPrefixActivationTime uint64
	submitterDataPrefixes    [][4]byte
	ftsoDataPrefixes         [][4]byte
}

var (
	prioritisedContractVariants = utils.NewChainValue(&prioritisedParams{
		0, common.Address{}, 0, 0, [][4]byte{}, [][4]byte{},
	}).
		AddValue(params.FlareChainID, &prioritisedParams{
			submitterContractActivationTimeFlare,
			prioritisedSubmitterContractAddress,
			3000000,
			uint64(time.Date(2024, time.October, 10, 15, 0, 0, 0, time.UTC).Unix()),
			submitterDataPrefixes,
			prioritisedFTSOContractDataPrefixesFlareNetworks,
		}).
		AddValue(params.CostwoChainID, &prioritisedParams{
			submitterContractActivationTimeCostwo,
			prioritisedSubmitterContractAddress,
			3000000,
			uint64(time.Date(2024, time.October, 10, 10, 0, 0, 0, time.UTC).Unix()),
			submitterDataPrefixes,
			prioritisedFTSOContractDataPrefixesFlareNetworks,
		}).
		AddValue(params.SongbirdChainID, &prioritisedParams{
			submitterContractActivationTimeSongbird,
			prioritisedSubmitterContractAddress,
			math.MaxUint64,
			uint64(time.Date(2024, time.October, 10, 13, 0, 0, 0, time.UTC).Unix()),
			submitterDataPrefixes,
			prioritisedFTSOContractDataPrefixesSongbirdNetworks,
		}).
		AddValue(params.CostonChainID, &prioritisedParams{
			submitterContractActivationTimeCoston,
			prioritisedSubmitterContractAddress,
			math.MaxUint64,
			uint64(time.Date(2024, time.October, 10, 8, 0, 0, 0, time.UTC).Unix()),
			submitterDataPrefixes,
			prioritisedFTSOContractDataPrefixesSongbirdNetworks,
		}).
		AddValue(params.LocalFlareChainID, &prioritisedParams{
			0,
			prioritisedSubmitterContractAddressEnv,
			3000000,
			0,
			[][4]byte{},
			[][4]byte{},
		}).
		AddValue(params.LocalChainID, &prioritisedParams{
			0,
			prioritisedSubmitterContractAddressEnv,
			math.MaxUint64,
			0,
			[][4]byte{},
			[][4]byte{},
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
	GetBlockTime() uint64
	GetGasLimit() uint64
	AddBalance(addr common.Address, amount *big.Int)
}

func GetDaemonGasMultiplier(blockTime uint64) uint64 {
	switch {
	default:
		return 100
	}
}

func GetDaemonContractAddr(blockTime uint64) string {
	switch {
	default:
		return "0x1000000000000000000000000000000000000002"
	}
}

func GetDaemonSelector(blockTime uint64) []byte {
	switch {
	default:
		return []byte{0x7f, 0xec, 0x8d, 0x38}
	}
}

func IsPrioritisedContractCall(chainID *big.Int, blockTime uint64, to *common.Address, data []byte, ret []byte, initialGas uint64) bool {
	if to == nil || chainID == nil {
		return false
	}

	chainValue := prioritisedContractVariants.GetValue(chainID)

	switch {
	case initialGas > chainValue.maxGasLimit:
		return false
	case *to == prioritisedFTSOContractAddress:
		if blockTime > chainValue.dataPrefixActivationTime {
			return checkDataPrefix(data, chainValue.ftsoDataPrefixes)
		}
		return true
	case *to == chainValue.submitterAddress && blockTime > chainValue.submitterActivationTime && !isZeroSlice(ret):
		if blockTime > chainValue.dataPrefixActivationTime {
			return len(data) <= prioritisedCallDataCap && checkDataPrefix(data, chainValue.submitterDataPrefixes)
		}
		return true
	default:
		return false
	}
}

func GetMaximumMintRequest(chainID *big.Int, blockTime uint64) *big.Int {
	switch {
	case chainID.Cmp(params.FlareChainID) == 0 || chainID.Cmp(params.CostwoChainID) == 0 || chainID.Cmp(params.LocalFlareChainID) == 0:
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

func checkDataPrefix(data []byte, prefixes [][4]byte) bool {
	if len(data) < 4 {
		return false
	}
	dataPrefix := data[:4]
	for _, prefix := range prefixes {
		if bytes.Equal(dataPrefix, prefix[:]) {
			return true
		}
	}
	return false
}
