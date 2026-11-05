// (c) 2026, Flare Network. All rights reserved.
// See the file LICENSE for licensing terms.

package secp256k1fx

import (
	"encoding/hex"

	"github.com/ava-labs/libevm/accounts"
	"github.com/ava-labs/libevm/common"

	"github.com/ava-labs/avalanchego/ids"
	"github.com/ava-labs/avalanchego/utils/crypto/secp256k1"
	"github.com/ava-labs/avalanchego/utils/hashing"
)

// SigningHashes are the message hashes a secp256k1 credential may sign on
// Flare: the plain sha256 of the unsigned transaction bytes (the Avalanche
// scheme) and the Ethereum signed-message hash (accounts.TextHash, EIP-191) of
// the hex encoding of that digest, which Ethereum signing tools produce when
// asked to sign the hex string. [Fx.VerifyCredentials] accepts either;
// [SigningHashes.RecoverSigner] and [SigningHashes.RecoverEthSigner] try them
// in that order.
type SigningHashes struct {
	Plain       []byte
	EthPrefixed []byte
}

// NewSigningHashes computes the hashes of unsignedBytes, the serialized
// unsigned transaction.
func NewSigningHashes(unsignedBytes []byte) SigningHashes {
	plain := hashing.ComputeHash256(unsignedBytes)
	return SigningHashes{
		Plain:       plain,
		EthPrefixed: accounts.TextHash([]byte(hex.EncodeToString(plain))),
	}
}

// RecoverSigner returns the public key that produced sig over one of the
// hashes and whose Avalanche address ([secp256k1.PublicKey.Address]) is
// expected. A malformed signature is an error, as reported by
// [secp256k1.RecoverPublicKeyFromHash]; a well-formed signature that recovers
// to another address under both hashes is [ErrWrongSig]. cache may be nil.
func (h SigningHashes) RecoverSigner(cache *secp256k1.RecoverCache, sig []byte, expected ids.ShortID) (*secp256k1.PublicKey, error) {
	return h.recoverSigner(cache, sig, func(pk *secp256k1.PublicKey) bool {
		return pk.Address() == expected
	})
}

// RecoverEthSigner is [SigningHashes.RecoverSigner] for a signer identified by
// its Ethereum address ([secp256k1.PublicKey.EthAddress]), as the inputs of a
// C-chain export transaction are.
func (h SigningHashes) RecoverEthSigner(cache *secp256k1.RecoverCache, sig []byte, expected common.Address) (*secp256k1.PublicKey, error) {
	return h.recoverSigner(cache, sig, func(pk *secp256k1.PublicKey) bool {
		return pk.EthAddress() == expected
	})
}

func (h SigningHashes) recoverSigner(cache *secp256k1.RecoverCache, sig []byte, isExpected func(*secp256k1.PublicKey) bool) (*secp256k1.PublicKey, error) {
	// Try to recover the address from the signature of the transaction hash without a prefix
	// (Standard Avalanche approach, but unsupported/deprecated by most signing tools)
	pk, err := cache.RecoverPublicKeyFromHash(h.Plain, sig)
	if err != nil {
		return nil, err
	}
	if isExpected(pk) {
		return pk, nil
	}

	// Try to recover the address from the signature of the prefixed message hash
	// (with the standard Ethereum prefix, see accounts.TextHash)
	pk, err = cache.RecoverPublicKeyFromHash(h.EthPrefixed, sig)
	if err != nil {
		return nil, err
	}
	if isExpected(pk) {
		return pk, nil
	}
	return nil, ErrWrongSig
}
