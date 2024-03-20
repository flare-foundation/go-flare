package core

import (
	"errors"
	"math/big"

	"github.com/ava-labs/coreth/params"
	"github.com/ava-labs/coreth/utils"
	"github.com/ethereum/go-ethereum/common"
)

var (
	stateTransitionVariants = utils.NewChainValue(invalidChain).
		AddValues([]*big.Int{params.FlareChainID, params.CostwoChainID, params.StagingChainID, params.LocalFlareChainID}, stateTransitionParamsFlare).
		AddValues([]*big.Int{params.SongbirdChainID, params.CostonChainID, params.LocalChainID}, stateTransitionParamsSongbird)
)

func invalidChain(st *StateTransition) (common.Address, uint64, bool, bool, error) {
	return common.Address{}, 0, false, false, errors.New("invalid chain ID")
}

// Returns the state transition parameters for the given chain ID
// burnAddress, nominalGasPrice, isFlare chain, isSongbird chain, error
func stateTransitionParamsFlare(st *StateTransition) (common.Address, uint64, bool, bool, error) {
	return common.HexToAddress("0x000000000000000000000000000000000000dEaD"),
		uint64(params.ApricotPhase4MinBaseFee),
		true,
		false,
		nil
}

func stateTransitionParamsSongbird(st *StateTransition) (common.Address, uint64, bool, bool, error) {
	burnAddress := st.evm.Context.Coinbase
	if burnAddress != common.HexToAddress("0x0100000000000000000000000000000000000000") {
		return common.Address{}, 0, false, true, errors.New("invalid value for block.coinbase")
	}
	return burnAddress, 225_000_000_000, false, true, nil
}
