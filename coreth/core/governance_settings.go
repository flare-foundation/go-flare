// (c) 2021, Flare Networks Limited. All rights reserved.
// Please see the file LICENSE for licensing terms.

package core

import (
	"encoding/binary"
	"math/big"
	"time"

	"github.com/ava-labs/coreth/core/vm"
	"github.com/ava-labs/coreth/params"
	"github.com/ethereum/go-ethereum/common"
)

var (
	flareGovActivationTime      = big.NewInt(time.Date(2022, time.September, 9, 0, 0, 0, 0, time.UTC).Unix())
	costwoGovActivationTime     = big.NewInt(time.Date(2022, time.September, 8, 0, 0, 0, 0, time.UTC).Unix())
	stagingGovActivationTime    = big.NewInt(time.Date(2022, time.January, 1, 0, 0, 0, 0, time.UTC).Unix())
	localFlareGovActivationTime = big.NewInt(time.Date(2022, time.January, 1, 0, 0, 0, 0, time.UTC).Unix())
)

func GetGovernanceSettingIsActivatedAndCalled(chainID *big.Int, blockTime *big.Int, to common.Address) bool {
	switch {
	case chainID.Cmp(params.FlareChainID) == 0 && blockTime.Cmp(flareGovActivationTime) >= 0:
		switch blockTime {
		default:
			return to == common.HexToAddress("0x1000000000000000000000000000000000000007")
		}
	case chainID.Cmp(params.CostwoChainID) == 0 && blockTime.Cmp(costwoGovActivationTime) >= 0:
		switch blockTime {
		default:
			return to == common.HexToAddress("0x1000000000000000000000000000000000000007")
		}
	case chainID.Cmp(params.StagingChainID) == 0 && blockTime.Cmp(stagingGovActivationTime) >= 0:
		switch blockTime {
		default:
			return to == common.HexToAddress("0x1000000000000000000000000000000000000007")
		}
	case chainID.Cmp(params.LocalFlareChainID) == 0 && blockTime.Cmp(localFlareGovActivationTime) >= 0:
		switch blockTime {
		default:
			return to == common.HexToAddress("0x1000000000000000000000000000000000000007")
		}
	default:
		return false
	}
}

// Signalling block.coinbase value
// address public constant SIGNAL_COINBASE = address(0x00000000000000000000000000000000000DEaD0);
//https://gitlab.com/flarenetwork/flare-smart-contracts/-/blob/4bb79bfe7266b43ea46e681f8a86ab8b9ef36446/contracts/genesis/implementation/GovernanceSettings.sol#L17

func GetGovernanceSettingsCoinbaseSignalAddr(chainID *big.Int, blockTime *big.Int) common.Address {
	switch {
	default:
		return common.HexToAddress("0x00000000000000000000000000000000000DEaD0")
	}
}

// function setGovernanceAddress(address _newGovernance) external
// https://gitlab.com/flarenetwork/flare-smart-contracts/-/blob/4bb79bfe7266b43ea46e681f8a86ab8b9ef36446/contracts/genesis/implementation/GovernanceSettings.sol#L73

func SetGovernanceAddressSelector(chainID *big.Int, blockTime *big.Int) []byte {
	switch {
	default:
		return []byte{0xcf, 0xc1, 0x62, 0x54}
	}
}

// function setTimelock(uint256 _newTimelock) external
// https://gitlab.com/flarenetwork/flare-smart-contracts/-/blob/4bb79bfe7266b43ea46e681f8a86ab8b9ef36446/contracts/genesis/implementation/GovernanceSettings.sol#L85

func SetTimelockSelector(chainID *big.Int, blockTime *big.Int) []byte {
	switch {
	default:
		return []byte{0x1e, 0x89, 0x1c, 0x0a}
	}
}

func NewGovernanceAddressIsPermitted(chainID *big.Int, blockTime *big.Int, newGovernanceAddress common.Address) bool {
	switch {
	case chainID.Cmp(params.FlareChainID) == 0:
		switch {
		default:
			return false
		}
	case chainID.Cmp(params.CostwoChainID) == 0:
		switch {
		default:
			return false
		}
	case chainID.Cmp(params.StagingChainID) == 0:
		switch {
		default:
			return false
		}
	case chainID.Cmp(params.LocalFlareChainID) == 0:
		switch {
		case blockTime.Cmp(big.NewInt(time.Date(2022, time.January, 1, 0, 0, 0, 0, time.UTC).Unix())) >= 0:
			return newGovernanceAddress == common.HexToAddress("0x1000000000000000000000000000000000000008")
		default:
			return false
		}
	default:
		return false
	}
}

func NewTimelockIsPermitted(chainID *big.Int, blockTime *big.Int, newTimelock uint64) bool {
	switch {
	case chainID.Cmp(params.FlareChainID) == 0:
		switch {
		case blockTime.Cmp(big.NewInt(time.Date(2022, time.September, 9, 0, 0, 0, 0, time.UTC).Unix())) >= 0:
			return newTimelock == 3600
		default:
			return false
		}
	case chainID.Cmp(params.CostwoChainID) == 0:
		switch {
		case blockTime.Cmp(big.NewInt(time.Date(2022, time.September, 8, 0, 0, 0, 0, time.UTC).Unix())) >= 0:
			return newTimelock == 3600
		default:
			return false
		}
	case chainID.Cmp(params.StagingChainID) == 0:
		switch {
		case blockTime.Cmp(big.NewInt(time.Date(2022, time.January, 1, 0, 0, 0, 0, time.UTC).Unix())) >= 0:
			return newTimelock == 3600
		default:
			return false
		}
	case chainID.Cmp(params.LocalFlareChainID) == 0:
		switch {
		case blockTime.Cmp(big.NewInt(time.Date(2022, time.January, 1, 0, 0, 0, 0, time.UTC).Unix())) >= 0:
			return newTimelock == 3600
		default:
			return false
		}
	default:
		return false
	}
}

func (st *StateTransition) SetGovernanceAddress(chainID *big.Int, timestamp *big.Int, newGovernanceAddress []byte) error {
	if NewGovernanceAddressIsPermitted(chainID, timestamp, common.BytesToAddress(newGovernanceAddress)) {
		coinbaseSignal := GetGovernanceSettingsCoinbaseSignalAddr(chainID, timestamp)
		originalCoinbase := st.evm.Context.Coinbase
		defer func() {
			st.evm.Context.Coinbase = originalCoinbase
		}()
		st.evm.Context.Coinbase = coinbaseSignal
		_, _, err := st.evm.Call(vm.AccountRef(coinbaseSignal), st.to(), st.data, st.evm.Context.GasLimit, big.NewInt(0))
		if err != nil {
			return err
		}
	}
	return nil
}

func (st *StateTransition) SetTimelock(chainID *big.Int, timestamp *big.Int, newTimelock []byte) error {
	if NewTimelockIsPermitted(chainID, timestamp, binary.BigEndian.Uint64(newTimelock[24:32])) {
		coinbaseSignal := GetGovernanceSettingsCoinbaseSignalAddr(chainID, timestamp)
		originalCoinbase := st.evm.Context.Coinbase
		defer func() {
			st.evm.Context.Coinbase = originalCoinbase
		}()
		st.evm.Context.Coinbase = coinbaseSignal
		_, _, err := st.evm.Call(vm.AccountRef(coinbaseSignal), st.to(), st.data, st.evm.Context.GasLimit, big.NewInt(0))
		if err != nil {
			return err
		}
	}
	return nil
}
