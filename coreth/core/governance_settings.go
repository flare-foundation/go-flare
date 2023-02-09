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

	flareInitialAirdropChangeActivationTime      = big.NewInt(time.Date(2022, time.November, 10, 15, 0, 0, 0, time.UTC).Unix())
	costwoInitialAirdropChangeActivationTime     = big.NewInt(time.Date(2022, time.October, 27, 20, 0, 0, 0, time.UTC).Unix())
	localFlareInitialAirdropChangeActivationTime = big.NewInt(time.Date(2022, time.January, 1, 0, 0, 0, 0, time.UTC).Unix())

	flareDistributionChangeActivationTime      = big.NewInt(time.Date(2023, time.March, 1, 15, 0, 0, 0, time.UTC).Unix())
	costwoDistributionChangeActivationTime     = big.NewInt(time.Date(2023, time.January, 26, 15, 0, 0, 0, time.UTC).Unix())
	localFlareDistributionChangeActivationTime = big.NewInt(time.Date(2023, time.January, 1, 0, 0, 0, 0, time.UTC).Unix())
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

func GetInitialAirdropChangeIsActivatedAndCalled(chainID *big.Int, blockTime *big.Int, to common.Address) bool {
	switch {
	case chainID.Cmp(params.FlareChainID) == 0 && blockTime.Cmp(flareInitialAirdropChangeActivationTime) >= 0:
		switch blockTime {
		default:
			return to == common.HexToAddress("0x4AeE563140E36abA778944E2Ca68c3988CAd5730")
		}
	case chainID.Cmp(params.CostwoChainID) == 0 && blockTime.Cmp(costwoInitialAirdropChangeActivationTime) >= 0:
		switch blockTime {
		default:
			return to == common.HexToAddress("0x28561B938342efD0677f60Fd0912e1931367a612")
		}
	case chainID.Cmp(params.LocalFlareChainID) == 0 && blockTime.Cmp(localFlareInitialAirdropChangeActivationTime) >= 0:
		switch blockTime {
		default:
			return to == common.HexToAddress("0x1000000000000000000000000000000000000008")
		}
	default:
		return false
	}
}

func GetDistributionChangeIsActivatedAndCalled(chainID *big.Int, blockTime *big.Int, to common.Address) bool {
	switch {
	case chainID.Cmp(params.FlareChainID) == 0 && blockTime.Cmp(flareDistributionChangeActivationTime) >= 0:
		switch blockTime {
		default:
			return to == common.HexToAddress("0x4d1c42F41555Ae35DfC1819bd718f7D9Fb28abdD")
		}
	case chainID.Cmp(params.CostwoChainID) == 0 && blockTime.Cmp(costwoDistributionChangeActivationTime) >= 0:
		switch blockTime {
		default:
			return to == common.HexToAddress("0xdF1deD5f1905C5012cbeE8367e3F4849afEAE545")
		}
	case chainID.Cmp(params.LocalFlareChainID) == 0 && blockTime.Cmp(localFlareDistributionChangeActivationTime) >= 0:
		switch blockTime {
		default:
			return to == common.HexToAddress("0x1000000000000000000000000000000000000009")
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

func GetInitialAirdropChangeCoinbaseSignalAddr(chainID *big.Int, blockTime *big.Int) common.Address {
	switch {
	default:
		return common.HexToAddress("0x00000000000000000000000000000000000dead2")
	}
}

func GetDistributionChangeCoinbaseSignalAddr(chainID *big.Int, blockTime *big.Int) common.Address {
	switch {
	default:
		return common.HexToAddress("0x00000000000000000000000000000000000deAD3")
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

// function updateInitialAirdropAddress() external
// https://flare-explorer.flare.network/address/0x4AeE563140E36abA778944E2Ca68c3988CAd5730/contracts#address-tabs

func UpdateInitialAirdropAddressSelector(chainID *big.Int, blockTime *big.Int) []byte {
	switch {
	default:
		return []byte{0x7d, 0x1f, 0x99, 0x46}
	}
}

// function updateDistributionAddress() external
// https://flare-explorer.flare.network/address/0x4d1c42F41555Ae35DfC1819bd718f7D9Fb28abdD/contracts#address-tabs

func UpdateDistributionAddressSelector(chainID *big.Int, blockTime *big.Int) []byte {
	switch {
	default:
		return []byte{0x5a, 0xce, 0x4f, 0x0d}
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
			return newGovernanceAddress == common.HexToAddress("0x100000000000000000000000000000000000000f")
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

func GetInitialAirdropContractAddress(chainID *big.Int, blockTime *big.Int) common.Address {
	switch {
	default:
		return common.HexToAddress("0x1000000000000000000000000000000000000006")
	}
}

func GetDistributionContractAddress(chainID *big.Int, blockTime *big.Int) common.Address {
	switch {
	default:
		return common.HexToAddress("0x1000000000000000000000000000000000000004")
	}
}

func GetTargetAirdropContractAddress(chainID *big.Int, blockTime *big.Int) common.Address {
	switch {
	case chainID.Cmp(params.FlareChainID) == 0:
		switch {
		default:
			return common.HexToAddress("0xbe653C54DF337F13Fcb726101388F4a4803049F3")
		}
	case chainID.Cmp(params.CostwoChainID) == 0:
		switch {
		default:
			return common.HexToAddress("0xC83Ec6a4aFf2099942836860A28C7e248Fabc32C")
		}
	default:
		return common.HexToAddress("0x000000000000000000000000000000000000dEaD")
	}
}

func GetTargetDistributionContractAddress(chainID *big.Int, blockTime *big.Int) common.Address {
	switch {
	case chainID.Cmp(params.FlareChainID) == 0:
		switch {
		default:
			return common.HexToAddress("0xbe653C54DF337F13Fcb726101388F4a4803049F3")
		}
	case chainID.Cmp(params.CostwoChainID) == 0:
		switch {
		default:
			return common.HexToAddress("0xC83Ec6a4aFf2099942836860A28C7e248Fabc32C")
		}
	default:
		return common.HexToAddress("0x000000000000000000000000000000000000dEaD")
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
		_, _, _, err := st.evm.DaemonCall(vm.AccountRef(coinbaseSignal), st.to(), st.data, st.evm.Context.GasLimit)
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
		_, _, _, err := st.evm.DaemonCall(vm.AccountRef(coinbaseSignal), st.to(), st.data, st.evm.Context.GasLimit)
		if err != nil {
			return err
		}
	}
	return nil
}

func (st *StateTransition) UpdateInitialAirdropAddress(chainID *big.Int, timestamp *big.Int) error {
	coinbaseSignal := GetInitialAirdropChangeCoinbaseSignalAddr(chainID, timestamp)
	originalCoinbase := st.evm.Context.Coinbase
	defer func() {
		st.evm.Context.Coinbase = originalCoinbase
	}()
	st.evm.Context.Coinbase = coinbaseSignal
	_, _, _, err := st.evm.DaemonCall(vm.AccountRef(coinbaseSignal), st.to(), st.data, st.evm.Context.GasLimit)
	if err != nil {
		return err
	}
	initialAirdropAddress := GetInitialAirdropContractAddress(chainID, timestamp)
	targetAidropAddress := GetTargetAirdropContractAddress(chainID, timestamp)
	airdropBalance := st.state.GetBalance(initialAirdropAddress)
	st.state.SubBalance(initialAirdropAddress, airdropBalance)
	st.state.AddBalance(targetAidropAddress, airdropBalance)
	return nil
}

func (st *StateTransition) UpdateDistributionAddress(chainID *big.Int, timestamp *big.Int) error {
	coinbaseSignal := GetDistributionChangeCoinbaseSignalAddr(chainID, timestamp)
	originalCoinbase := st.evm.Context.Coinbase
	defer func() {
		st.evm.Context.Coinbase = originalCoinbase
	}()
	st.evm.Context.Coinbase = coinbaseSignal
	_, _, _, err := st.evm.DaemonCall(vm.AccountRef(coinbaseSignal), st.to(), st.data, st.evm.Context.GasLimit)
	if err != nil {
		return err
	}
	distributionAddress := GetDistributionContractAddress(chainID, timestamp)
	targetDistributionAddress := GetTargetDistributionContractAddress(chainID, timestamp)
	distributionBalance := st.state.GetBalance(distributionAddress)
	st.state.SubBalance(distributionAddress, distributionBalance)
	st.state.AddBalance(targetDistributionAddress, distributionBalance)
	return nil
}
