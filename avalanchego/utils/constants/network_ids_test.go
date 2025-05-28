// Copyright (C) 2019-2024, Ava Labs, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package constants

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestGetHRP(t *testing.T) {
	tests := []struct {
		id  uint32
		hrp string
	}{
		{
			id:  MainnetID,
			hrp: MainnetHRP,
		},
		{
			id:  CostonID,
			hrp: CostonHRP,
		},
		{
			id:  SongbirdID,
			hrp: SongbirdName,
		},
		{
			id:  LocalID,
			hrp: LocalHRP,
		},
		{
			id:  4294967295,
			hrp: FallbackHRP,
		},
		{
			id:  FlareID,
			hrp: FlareHRP,
		},
		{
			id:  CostwoID,
			hrp: CostwoHRP,
		},
		{
			id:  LocalFlareID,
			hrp: LocalFlareHRP,
		},
	}
	for _, test := range tests {
		t.Run(test.hrp, func(t *testing.T) {
			require.Equal(t, test.hrp, GetHRP(test.id))
		})
	}
}

func TestNetworkName(t *testing.T) {
	tests := []struct {
		id   uint32
		name string
	}{
		{
			id:   MainnetID,
			name: MainnetName,
		},
		{
			id:   CostonID,
			name: CostonName,
		},
		{
			id:   SongbirdID,
			name: SongbirdName,
		},
		{
			id:   LocalID,
			name: LocalName,
		},
		{
			id:   4294967295,
			name: "network-4294967295",
		},
		{
			id:   FlareID,
			name: FlareName,
		},
		{
			id:   CostwoID,
			name: CostwoName,
		},
		{
			id:   LocalFlareID,
			name: LocalFlareName,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			require.Equal(t, test.name, NetworkName(test.id))
		})
	}
}

func TestNetworkID(t *testing.T) {
	tests := []struct {
		name        string
		id          uint32
		expectedErr error
	}{
		{
			name: MainnetName,
			id:   MainnetID,
		},
		{
			name: "MaInNeT",
			id:   MainnetID,
		},
		{
			name: SongbirdName,
			id:   SongbirdID,
		},
		{
			name: LocalName,
			id:   LocalID,
		},
		{
			name: FlareName,
			id:   FlareID,
		},
		{
			name: CostwoName,
			id:   CostwoID,
		},
		{
			name: LocalFlareName,
			id:   LocalFlareID,
		},
		{
			name: "network-4294967295",
			id:   4294967295,
		},
		{
			name: "4294967295",
			id:   4294967295,
		},
		{
			name:        "networ-4294967295",
			expectedErr: ErrParseNetworkName,
		},
		{
			name:        "network-4294967295123123",
			expectedErr: ErrParseNetworkName,
		},
		{
			name:        "4294967295123123",
			expectedErr: ErrParseNetworkName,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			require := require.New(t)

			id, err := NetworkID(test.name)
			require.ErrorIs(err, test.expectedErr)
			require.Equal(test.id, id)
		})
	}
}
