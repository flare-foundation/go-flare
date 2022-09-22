// (c) 2021-2022, Ava Labs, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package constants

import (
	"math/big"
	"time"

	"github.com/ethereum/go-ethereum/common"
)

var (
	BlackholeAddr = common.Address{
		1, 0, 0, 0, 0, 0, 0, 0, 0, 0,
		0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
	}

	NativeAssetCallDeprecationTime = big.NewInt(time.Date(2022, time.September, 16, 15, 0, 0, 0, time.UTC).Unix())
)
