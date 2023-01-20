package hashing

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestToValidatorConfigHash(t *testing.T) {
	require := require.New(t)
	networkId := "162"
	duration := "86400"
	weight := "10000000000000"

	// pChainPkReadable := "P-localflare18jma8ppw3nhx5r4ap8clazz0dps7rv5uj3gy4v"
	// pChainPk, err := address.ParseToID(pChainPkReadable)
	// require.NoError(err)
	pChainPk := "6Y3kysjF9jnHnYkdS9yGAuoHyae2eNmeV"

	nodeId1 := "NodeID-MFrZFVCXPv5iCn6M9K6XduxGTYp891xXZ"
	nodeId2 := "NodeID-NFBbbJ4qCmNaCzeW7sxErhvWqvEQMnYcN"
	nodeId3 := "NodeID-GWPcbFJZFfZreETSoWjPimr846mXEKCtu"
	nodeId4 := "NodeID-P7oB2McjBGgW2NXXWVYjV8JEDFoW9xDE5"

	result1 := ToValidatorConfigHash(networkId, pChainPk, nodeId1, weight, duration)
	t.Logf("Result: %v", result1)
	require.True(result1 == "3b66ad21620fe6d0dd1665b89b7c5f7a3b18e34d7d18ca56b732f833b8259108")

	result2 := ToValidatorConfigHash(networkId, pChainPk, nodeId2, weight, duration)
	t.Logf("Result: %v", result2)
	require.True(result2 == "0f750b09f2702ecea445657120e7dcb0cdb46a8c87d1eae9a508fa4e3bfa5a32")

	result3 := ToValidatorConfigHash(networkId, pChainPk, nodeId3, weight, duration)
	t.Logf("Result: %v", result3)
	require.True(result3 == "39e596bdd2e00f5cb7fd86069c94159b98bdd79a4ed8684a88c0faf49f63bfab")

	result4 := ToValidatorConfigHash(networkId, pChainPk, nodeId4, weight, duration)
	t.Logf("Result: %v", result4)
	require.True(result4 == "aa43e6ef2d60823406e7e3cb6fcdb148fdd3ffcd137a261457f9ee6d541f9ca9")
}
