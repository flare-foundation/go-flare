package core

import (
	"errors"
	"math/big"

	"github.com/ava-labs/avalanchego/graft/coreth/params"
	"github.com/ava-labs/avalanchego/graft/coreth/plugin/evm/upgrade/ap4"
	"github.com/ava-labs/avalanchego/graft/evm/utils"
	"github.com/ava-labs/libevm/common"
)

var (
	StateTransitionVariants = utils.NewChainValue(nonFlareChain).
		AddValues([]*big.Int{params.FlareChainID, params.CostwoChainID, params.LocalFlareChainID}, stateTransitionParamsFlare).
		AddValues([]*big.Int{params.SongbirdChainID, params.CostonChainID, params.LocalChainID}, stateTransitionParamsSongbird)
)

// Chain IDs outside the Flare/Songbird families occur only in tests. Fees are
// credited to the coinbase, matching upstream coreth exactly, so that
// upstream-generated chain fixtures (e.g. plugin/evm/upgradechaintest) and
// replaying VMs reproduce the same state.
func nonFlareChain(coinbase common.Address) (common.Address, uint64, bool, bool, error) {
	return coinbase,
		uint64(ap4.MinBaseFee),
		false,
		false,
		nil
}

// Returns the state transition parameters for the given chain ID
// burnAddress, nominalGasPrice, isFlare chain, isSongbird chain, error
func stateTransitionParamsFlare(coinbase common.Address) (common.Address, uint64, bool, bool, error) {
	return common.HexToAddress("0x000000000000000000000000000000000000dEaD"),
		uint64(ap4.MinBaseFee),
		true,
		false,
		nil
}

func stateTransitionParamsSongbird(coinbase common.Address) (common.Address, uint64, bool, bool, error) {
	burnAddress := coinbase
	if burnAddress != common.HexToAddress("0x0100000000000000000000000000000000000000") {
		return common.Address{}, 0, false, true, errors.New("invalid value for block.coinbase")
	}
	return burnAddress, 225_000_000_000, false, true, nil
}
