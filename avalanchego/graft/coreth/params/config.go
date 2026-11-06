// Copyright (C) 2019, Ava Labs, Inc. All rights reserved.
// See the file LICENSE for licensing terms.
//
// This file is a derived work, based on the go-ethereum library whose original
// notices appear below.
//
// It is distributed under a license compatible with the licensing terms of the
// original code from which it is derived.
//
// Much love to the original authors for their work.
// **********
// Copyright 2016 The go-ethereum Authors
// This file is part of the go-ethereum library.
//
// The go-ethereum library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The go-ethereum library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the go-ethereum library. If not, see <http://www.gnu.org/licenses/>.

package params

import (
	"math/big"

	"github.com/ava-labs/avalanchego/graft/coreth/params/extras"
	"github.com/ava-labs/avalanchego/graft/evm/utils"
	"github.com/ava-labs/libevm/libevm"
	ethparams "github.com/ava-labs/libevm/params"
)

// Flare Chain IDs.
var (
	FlareChainID      = big.NewInt(14)  // https://github.com/ethereum-lists/chains/blob/master/_data/chains/eip155-14.json
	CostonChainID     = big.NewInt(16)  // https://github.com/ethereum-lists/chains/blob/master/_data/chains/eip155-16.json
	SongbirdChainID   = big.NewInt(19)  // https://github.com/ethereum-lists/chains/blob/master/_data/chains/eip155-19.json
	CostwoChainID     = big.NewInt(114) // TO-DO: Register with https://github.com/ethereum-lists
	LocalFlareChainID = big.NewInt(162)
	LocalChainID      = big.NewInt(4294967295)
)

func init() {
	libevm.WithTemporaryExtrasLock(func(l libevm.ExtrasLock) error {
		return WithTempRegisteredExtras(l, func() error {
			initialiseChainConfigs()
			return nil
		})
	})
}

var (
	TestChainConfig,
	TestFlareChainConfig,
	TestApricotPhase4Config,
	TestApricotPhase5Config *ChainConfig
)

// initialiseChainConfigs MUST be called inside [WithTempRegisteredExtras] to
// allow [WithExtra] to work without global registration of libevm extras.
func initialiseChainConfigs() {
	TestChainConfig = WithExtra(
		&ChainConfig{
			ChainID:             big.NewInt(1),
			HomesteadBlock:      big.NewInt(0),
			DAOForkBlock:        big.NewInt(0),
			DAOForkSupport:      true,
			EIP150Block:         big.NewInt(0),
			EIP155Block:         big.NewInt(0),
			EIP158Block:         big.NewInt(0),
			ByzantiumBlock:      big.NewInt(0),
			ConstantinopleBlock: big.NewInt(0),
			PetersburgBlock:     big.NewInt(0),
			IstanbulBlock:       big.NewInt(0),
			MuirGlacierBlock:    big.NewInt(0),
			BerlinBlock:         big.NewInt(0),
			LondonBlock:         big.NewInt(0),
			ShanghaiTime:        utils.PointerTo[uint64](0),
			CancunTime:          utils.PointerTo[uint64](0),
		},
		extras.TestChainConfig,
	)

	TestFlareChainConfig = WithExtra(
		&ChainConfig{
			ChainID:             FlareChainID,
			HomesteadBlock:      big.NewInt(0),
			DAOForkBlock:        big.NewInt(0),
			DAOForkSupport:      true,
			EIP150Block:         big.NewInt(0),
			EIP155Block:         big.NewInt(0),
			EIP158Block:         big.NewInt(0),
			ByzantiumBlock:      big.NewInt(0),
			ConstantinopleBlock: big.NewInt(0),
			PetersburgBlock:     big.NewInt(0),
			IstanbulBlock:       big.NewInt(0),
			MuirGlacierBlock:    big.NewInt(0),
			BerlinBlock:         big.NewInt(0),
			LondonBlock:         big.NewInt(0),
			ShanghaiTime:        utils.PointerTo[uint64](0),
			CancunTime:          utils.PointerTo[uint64](0),
		},
		extras.TestChainConfig,
	)

	TestApricotPhase4Config = WithExtra(
		&ChainConfig{
			ChainID:             big.NewInt(1),
			HomesteadBlock:      big.NewInt(0),
			DAOForkBlock:        big.NewInt(0),
			DAOForkSupport:      true,
			EIP150Block:         big.NewInt(0),
			EIP155Block:         big.NewInt(0),
			EIP158Block:         big.NewInt(0),
			ByzantiumBlock:      big.NewInt(0),
			ConstantinopleBlock: big.NewInt(0),
			PetersburgBlock:     big.NewInt(0),
			IstanbulBlock:       big.NewInt(0),
			MuirGlacierBlock:    big.NewInt(0),
			BerlinBlock:         big.NewInt(0),
			LondonBlock:         big.NewInt(0),
		},
		extras.TestApricotPhase4Config,
	)

	TestApricotPhase5Config = WithExtra(
		&ChainConfig{
			ChainID:             big.NewInt(1),
			HomesteadBlock:      big.NewInt(0),
			DAOForkBlock:        big.NewInt(0),
			DAOForkSupport:      true,
			EIP150Block:         big.NewInt(0),
			EIP155Block:         big.NewInt(0),
			EIP158Block:         big.NewInt(0),
			ByzantiumBlock:      big.NewInt(0),
			ConstantinopleBlock: big.NewInt(0),
			PetersburgBlock:     big.NewInt(0),
			IstanbulBlock:       big.NewInt(0),
			MuirGlacierBlock:    big.NewInt(0),
			BerlinBlock:         big.NewInt(0),
			LondonBlock:         big.NewInt(0),
		},
		extras.TestApricotPhase5Config,
	)
}

// ChainConfig is the core config which determines the blockchain settings.
//
// ChainConfig is stored in the database on a per block basis. This means
// that any network, identified by its genesis block, can have its own
// set of configuration options.
type ChainConfig = ethparams.ChainConfig

// Rules wraps ChainConfig and is merely syntactic sugar or can be used for functions
// that do not have or require information about the block.
//
// Rules is a one time interface meaning that it shouldn't be used in between transition
// phases.
type Rules = ethparams.Rules
