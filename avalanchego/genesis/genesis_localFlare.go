// Copyright (C) 2019-2021, Ava Labs, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package genesis

import (
	"time"

	_ "embed"

	"github.com/ava-labs/avalanchego/utils/units"
	"github.com/ava-labs/avalanchego/vms/platformvm/reward"
	txfee "github.com/ava-labs/avalanchego/vms/platformvm/txs/fee"
)

var (
	//go:embed genesis_localFlare.json
	localFlareGenesisConfigJSON []byte

	// LocalFlareParams are the params used for the flare local network
	LocalFlareParams = Params{
		TxFeeConfig: TxFeeConfig{
			CreateAssetTxFee: units.MilliAvax,
			// For reference only, fees are set via program arguments (defaults are below)
			StaticFeeConfig: txfee.StaticConfig{
				TxFee:                         units.MilliAvax,
				CreateSubnetTxFee:             100 * units.MilliAvax,
				TransformSubnetTxFee:          100 * units.MilliAvax,
				CreateBlockchainTxFee:         100 * units.MilliAvax,
				AddPrimaryNetworkValidatorFee: 0,
				AddPrimaryNetworkDelegatorFee: 0,
				AddSubnetValidatorFee:         units.MilliAvax,
				AddSubnetDelegatorFee:         units.MilliAvax,
			},
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
