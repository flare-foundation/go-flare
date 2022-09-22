// Copyright (C) 2019-2021, Ava Labs, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package genesis

import (
	"time"

	_ "embed"

	"github.com/ava-labs/avalanchego/utils/units"
	"github.com/ava-labs/avalanchego/vms/platformvm/reward"
)

var (
	//go:embed genesis_flare.json
	flareGenesisConfigJSON []byte

	// FlareParams are the params used for the flare mainnet
	FlareParams = Params{
		TxFeeConfig: TxFeeConfig{
			TxFee:                 units.MilliAvax,
			CreateAssetTxFee:      units.MilliAvax,
			CreateSubnetTxFee:     100 * units.MegaAvax,
			CreateBlockchainTxFee: 100 * units.MegaAvax,
		},
		StakingConfig: StakingConfig{
			UptimeRequirement: .8, // 80%
			MinValidatorStake: 1 * units.Avax,
			MaxValidatorStake: 10000 * units.Avax,
			MinDelegatorStake: 0,
			MinDelegationFee:  0,
			MinStakeDuration:  24 * time.Hour,
			MaxStakeDuration:  365 * 24 * time.Hour,
			RewardConfig: reward.Config{
				MaxConsumptionRate: .12 * reward.PercentDenominator,
				MinConsumptionRate: .10 * reward.PercentDenominator,
				MintingPeriod:      365 * 24 * time.Hour,
				SupplyCap:          0 * units.MegaAvax,
			},
		},
	}
)
