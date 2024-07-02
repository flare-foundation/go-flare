package validators

import (
	"testing"
	"time"

	"github.com/ava-labs/avalanchego/utils/constants"
	"github.com/stretchr/testify/require"
)

func TestValidatorsBeforeExpiration(t *testing.T) {
	songbirdValidatorsExpTime = time.Date(2024, time.February, 1, 0, 0, 0, 0, time.UTC)

	vs := defaultValidatorSet{}
	vs.initialize(constants.SongbirdID, time.Date(2024, time.January, 1, 0, 0, 0, 0, time.UTC))

	vds := vs.list()
	require.Len(t, vds, 20)

	expVdrs := vs.expiredValidators(constants.SongbirdID, time.Date(2024, time.February, 2, 0, 0, 0, 0, time.UTC))
	require.Len(t, expVdrs, 20)
}

func TestValidatorsAfterExpiration(t *testing.T) {
	songbirdValidatorsExpTime = time.Date(2024, time.February, 1, 0, 0, 0, 0, time.UTC)

	vs := defaultValidatorSet{}
	vs.initialize(constants.SongbirdID, time.Date(2024, time.March, 1, 0, 0, 0, 0, time.UTC))

	vds := vs.list()
	require.Len(t, vds, 0)
}
