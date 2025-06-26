package params

import (
	"time"

	"github.com/ava-labs/coreth/constants"
)

// Fork times: copied from avalanchego/version/constants.go
// There is an "import cycle" between coreth and avalanchego on Avalanche GitHub repository which lacks
// times for Flare and Songbird networks.
var (
	ApricotPhase1Times = map[uint32]time.Time{
		constants.MainnetID:  time.Date(2021, time.March, 31, 14, 0, 0, 0, time.UTC),
		constants.CostonID:   time.Date(2000, time.January, 1, 0, 0, 0, 0, time.UTC),
		constants.SongbirdID: time.Date(2000, time.January, 1, 0, 0, 0, 0, time.UTC),
	}

	ApricotPhase2Times = map[uint32]time.Time{
		constants.MainnetID:  time.Date(2021, time.May, 10, 11, 0, 0, 0, time.UTC),
		constants.CostonID:   time.Date(2000, time.January, 1, 0, 0, 0, 0, time.UTC),
		constants.SongbirdID: time.Date(2000, time.January, 1, 0, 0, 0, 0, time.UTC),
	}

	ApricotPhase3Times = map[uint32]time.Time{
		constants.MainnetID:  time.Date(2021, time.August, 24, 14, 0, 0, 0, time.UTC),
		constants.CostonID:   time.Date(2022, time.February, 25, 14, 0, 0, 0, time.UTC),
		constants.SongbirdID: time.Date(2022, time.March, 7, 14, 0, 0, 0, time.UTC),
	}

	ApricotPhase4Times = map[uint32]time.Time{
		constants.MainnetID:  time.Date(2021, time.September, 22, 21, 0, 0, 0, time.UTC),
		constants.CostonID:   time.Date(2022, time.February, 25, 15, 0, 0, 0, time.UTC),
		constants.SongbirdID: time.Date(2022, time.March, 7, 15, 0, 0, 0, time.UTC),
	}

	ApricotPhase5Times = map[uint32]time.Time{
		constants.MainnetID:  time.Date(2021, time.December, 2, 18, 0, 0, 0, time.UTC),
		constants.CostonID:   time.Date(2022, time.February, 25, 16, 0, 0, 0, time.UTC),
		constants.SongbirdID: time.Date(2022, time.March, 7, 16, 0, 0, 0, time.UTC),
	}

	SongbirdTransitionTimes = map[uint32]time.Time{
		constants.SongbirdID: time.Date(2024, time.October, 29, 12, 0, 0, 0, time.UTC),
		constants.CostonID:   time.Date(2024, time.July, 23, 12, 0, 0, 0, time.UTC),
	}

	ApricotPhasePre6Times = map[uint32]time.Time{
		constants.MainnetID:  time.Date(2022, time.September, 5, 1, 30, 0, 0, time.UTC),
		constants.FlareID:    time.Date(2024, time.December, 17, 12, 0, 0, 0, time.UTC),
		constants.CostwoID:   time.Date(2024, time.November, 26, 12, 0, 0, 0, time.UTC),
		constants.CostonID:   time.Date(2025, time.January, 7, 12, 0, 0, 0, time.UTC),
		constants.SongbirdID: time.Date(2025, time.January, 28, 12, 0, 0, 0, time.UTC),
	}

	ApricotPhase6Times = map[uint32]time.Time{
		constants.MainnetID:  time.Date(2022, time.September, 6, 20, 0, 0, 0, time.UTC),
		constants.FlareID:    time.Date(2024, time.December, 17, 13, 0, 0, 0, time.UTC),
		constants.CostwoID:   time.Date(2024, time.November, 26, 13, 0, 0, 0, time.UTC),
		constants.CostonID:   time.Date(2025, time.January, 7, 13, 0, 0, 0, time.UTC),
		constants.SongbirdID: time.Date(2025, time.January, 28, 13, 0, 0, 0, time.UTC),
	}

	ApricotPhasePost6Times = map[uint32]time.Time{
		constants.MainnetID:  time.Date(2022, time.September, 7, 3, 0, 0, 0, time.UTC),
		constants.FlareID:    time.Date(2024, time.December, 17, 14, 0, 0, 0, time.UTC),
		constants.CostwoID:   time.Date(2024, time.November, 26, 14, 0, 0, 0, time.UTC),
		constants.CostonID:   time.Date(2025, time.January, 7, 14, 0, 0, 0, time.UTC),
		constants.SongbirdID: time.Date(2025, time.January, 28, 14, 0, 0, 0, time.UTC),
	}

	BanffTimes = map[uint32]time.Time{
		constants.MainnetID:  time.Date(2022, time.October, 18, 16, 0, 0, 0, time.UTC),
		constants.FlareID:    time.Date(2024, time.December, 17, 15, 0, 0, 0, time.UTC),
		constants.CostwoID:   time.Date(2024, time.November, 26, 15, 0, 0, 0, time.UTC),
		constants.CostonID:   time.Date(2025, time.January, 7, 15, 0, 0, 0, time.UTC),
		constants.SongbirdID: time.Date(2025, time.January, 28, 15, 0, 0, 0, time.UTC),
	}

	CortinaTimes = map[uint32]time.Time{
		constants.MainnetID:  time.Date(2023, time.April, 25, 15, 0, 0, 0, time.UTC),
		constants.FlareID:    time.Date(2025, time.May, 13, 12, 0, 0, 0, time.UTC),
		constants.CostwoID:   time.Date(2025, time.April, 8, 12, 0, 0, 0, time.UTC),
		constants.CostonID:   time.Date(2025, time.March, 27, 13, 0, 0, 0, time.UTC),
		constants.SongbirdID: time.Date(2025, time.May, 6, 12, 0, 0, 0, time.UTC),
	}

	DurangoTimes = map[uint32]time.Time{
		constants.MainnetID:  time.Date(2024, time.March, 6, 16, 0, 0, 0, time.UTC),
		constants.FlareID:    time.Date(2025, time.August, 5, 12, 0, 0, 0, time.UTC),
		constants.CostwoID:   time.Date(2025, time.June, 24, 12, 0, 0, 0, time.UTC),
		constants.CostonID:   time.Date(2025, time.July, 1, 12, 0, 0, 0, time.UTC),
		constants.SongbirdID: time.Date(2025, time.July, 22, 12, 0, 0, 0, time.UTC),
		constants.LocalID:    time.Date(10000, time.December, 1, 0, 0, 0, 0, time.UTC),
	}
)
