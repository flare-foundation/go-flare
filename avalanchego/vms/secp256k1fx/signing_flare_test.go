// (c) 2026, Flare Network. All rights reserved.
// See the file LICENSE for licensing terms.

package secp256k1fx

import (
	"encoding/hex"
	"testing"

	"github.com/ava-labs/libevm/accounts"
	"github.com/stretchr/testify/require"

	"github.com/ava-labs/avalanchego/ids"
	"github.com/ava-labs/avalanchego/utils/crypto/secp256k1"
	"github.com/ava-labs/avalanchego/utils/hashing"
)

func TestNewSigningHashes(t *testing.T) {
	require := require.New(t)

	hashes := NewSigningHashes(txBytes)
	plain := hashing.ComputeHash256(txBytes)
	require.Equal(plain, hashes.Plain)
	// The Ethereum scheme signs the hex string of the digest (what signing
	// tools are given), not the digest bytes.
	require.Equal(accounts.TextHash([]byte(hex.EncodeToString(plain))), hashes.EthPrefixed)
	require.NotEqual(hashes.Plain, hashes.EthPrefixed)
}

func TestRecoverSigner(t *testing.T) {
	keys := secp256k1.TestKeys()
	signer, other := keys[0], keys[1]
	hashes := NewSigningHashes(txBytes)
	plainSig, err := signer.SignHash(hashes.Plain)
	require.NoError(t, err)
	ethSig, err := signer.SignHash(hashes.EthPrefixed)
	require.NoError(t, err)

	tests := []struct {
		name     string
		cache    *secp256k1.RecoverCache
		sig      []byte
		expected ids.ShortID
		wantErr  error
	}{
		{name: "plain signature", sig: plainSig, expected: signer.Address()},
		{name: "eth-prefixed signature", sig: ethSig, expected: signer.Address()},
		{name: "plain signature, cached", cache: secp256k1.NewRecoverCache(4), sig: plainSig, expected: signer.Address()},
		{name: "eth-prefixed signature, cached", cache: secp256k1.NewRecoverCache(4), sig: ethSig, expected: signer.Address()},
		{name: "plain signature of another key", sig: plainSig, expected: other.Address(), wantErr: ErrWrongSig},
		{name: "eth-prefixed signature of another key", sig: ethSig, expected: other.Address(), wantErr: ErrWrongSig},
		{name: "malformed signature", sig: make([]byte, secp256k1.SignatureLen), expected: signer.Address(), wantErr: secp256k1.ErrInvalidSig},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require := require.New(t)

			pk, err := hashes.RecoverSigner(tt.cache, tt.sig, tt.expected)
			require.ErrorIs(err, tt.wantErr)
			if tt.wantErr != nil {
				require.Nil(pk)
				return
			}
			require.Equal(signer.Address(), pk.Address())
			require.Equal(signer.EthAddress(), pk.EthAddress())
			// Recovering twice (cache hit or not) gives the same key.
			again, err := hashes.RecoverSigner(tt.cache, tt.sig, tt.expected)
			require.NoError(err)
			require.Equal(pk.Bytes(), again.Bytes())
		})
	}
}
