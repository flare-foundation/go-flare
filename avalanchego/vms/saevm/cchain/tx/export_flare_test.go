// (c) 2026, Flare Network. All rights reserved.
// See the file LICENSE for licensing terms.

package tx_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/ava-labs/avalanchego/database/memdb"
	"github.com/ava-labs/avalanchego/ids"
	"github.com/ava-labs/avalanchego/utils/crypto/secp256k1"
	"github.com/ava-labs/avalanchego/vms/components/avax"
	"github.com/ava-labs/avalanchego/vms/saevm/cchain/tx/txtest"
	"github.com/ava-labs/avalanchego/vms/secp256k1fx"

	chainsatomic "github.com/ava-labs/avalanchego/chains/atomic"

	. "github.com/ava-labs/avalanchego/vms/saevm/cchain/tx"
)

// signEth signs u the way Ethereum signing tools do on Flare: over the EIP-191
// prefixed hash of the hex-encoded sha256 digest of the unsigned bytes
// (secp256k1fx.SigningHashes.EthPrefixed) rather than over the digest itself,
// which is what txtest.Sign produces.
func signEth(tb testing.TB, u Unsigned, sk *secp256k1.PrivateKey) txtest.Signature {
	tb.Helper()

	b, err := UnsignedBytes(u)
	require.NoErrorf(tb, err, "UnsignedBytes(%T)", u)
	sig, err := sk.SignHash(secp256k1fx.NewSigningHashes(b).EthPrefixed)
	require.NoError(tb, err, "SignHash()")
	return txtest.Signature(sig)
}

// Flare-specific: from Helicon on, the C-chain's atomic transactions are
// verified here rather than by coreth's tx_semantic_verifier.go, which has
// accepted EIP-191-prefixed signatures since Banff. Imports reach
// secp256k1fx.Fx.VerifyCredentials and inherit the two-scheme recovery;
// exports have their own check, so this locks in the same acceptance for both
// directions, with the plain scheme still accepted alongside.
func TestVerifyCredentialsEthSignature(t *testing.T) {
	owner := txtest.NewKey(t)
	other := txtest.NewKey(t)

	utxoID := avax.UTXOID{}
	inputID := utxoID.InputID()
	utxos := []*chainsatomic.Element{{
		Key: inputID[:],
		Value: txtest.MarshalUTXO(t, &avax.UTXO{
			UTXOID: utxoID,
			Asset:  avax.Asset{ID: avaxAssetID},
			Out: &secp256k1fx.TransferOutput{
				Amt: 100,
				OutputOwners: secp256k1fx.OutputOwners{
					Threshold: 1,
					Addrs:     []ids.ShortID{owner.Address()},
				},
			},
		}),
	}}

	newExport := func() *Export {
		return &Export{
			Ins: []Input{{Address: owner.EthAddress(), Amount: 100, AssetID: avaxAssetID}},
		}
	}
	newImport := func() *Import {
		return &Import{
			SourceChain: xChainID,
			ImportedInputs: []*avax.TransferableInput{{
				UTXOID: utxoID,
				Asset:  avax.Asset{ID: avaxAssetID},
				In: &secp256k1fx.TransferInput{
					Amt:   100,
					Input: secp256k1fx.Input{SigIndices: []uint32{0}},
				},
			}},
		}
	}
	signed := func(u Unsigned, sig txtest.Signature) *Tx {
		return &Tx{
			Unsigned: u,
			Creds:    []Credential{&secp256k1fx.Credential{Sigs: []txtest.Signature{sig}}},
		}
	}

	tests := []struct {
		name    string
		tx      *Tx
		wantErr error
	}{
		{
			name: "export_eth_signature",
			tx: func() *Tx {
				e := newExport()
				return signed(e, signEth(t, e, owner))
			}(),
		},
		{
			name: "export_plain_signature",
			tx: func() *Tx {
				e := newExport()
				return signed(e, txtest.Sign(t, e, owner))
			}(),
		},
		{
			name: "export_eth_signature_wrong_signer",
			tx: func() *Tx {
				e := newExport()
				return signed(e, signEth(t, e, other))
			}(),
			wantErr: ErrAddressMismatch,
		},
		{
			name: "import_eth_signature",
			tx: func() *Tx {
				i := newImport()
				return signed(i, signEth(t, i, owner))
			}(),
		},
		{
			name: "import_eth_signature_wrong_signer",
			tx: func() *Tx {
				i := newImport()
				return signed(i, signEth(t, i, other))
			}(),
			wantErr: ErrVerifyingTransfer,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			require := require.New(t)

			memory := chainsatomic.NewMemory(memdb.New())
			xMemory := memory.NewSharedMemory(xChainID)
			require.NoError(xMemory.Apply(map[ids.ID]*chainsatomic.Requests{
				cChainID: {PutRequests: utxos},
			}))

			cMemory := memory.NewSharedMemory(cChainID)
			require.ErrorIs(test.tx.VerifyCredentials(cMemory), test.wantErr)
		})
	}
}
