// (c) 2021-2022, Ava Labs, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package constants

import (
	"github.com/ethereum/go-ethereum/common"
)

// Network IDs: copied from avalanchego/utils/constants/network_ids.go
// There is an "import cycle" between coreth and avalanchego on Avalanche GitHub repository which lacks
// Flare and Songbird network ids.
const (
	MainnetID uint32 = 1
	CascadeID uint32 = 2
	DenaliID  uint32 = 3
	EverestID uint32 = 4

	UnitTestID   uint32 = 10
	LocalID      uint32 = 12345
	FlareID      uint32 = 14
	CostwoID     uint32 = 114
	LocalFlareID uint32 = 162
	SongbirdID   uint32 = 5
	CostonID     uint32 = 7
)

var (
	BlackholeAddr = common.Address{
		1, 0, 0, 0, 0, 0, 0, 0, 0, 0,
		0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
	}
)
