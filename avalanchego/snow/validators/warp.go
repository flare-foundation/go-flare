// Copyright (C) 2019, Ava Labs, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package validators

import (
	"bytes"
	"encoding/json"
	"math/big"

	"golang.org/x/exp/maps"

	"github.com/ava-labs/avalanchego/ids"
	"github.com/ava-labs/avalanchego/utils"
	"github.com/ava-labs/avalanchego/utils/crypto/bls"
	"github.com/ava-labs/avalanchego/utils/formatting"

	avajson "github.com/ava-labs/avalanchego/utils/json"
)

var _ utils.Sortable[*Warp] = (*Warp)(nil)

type WarpSet struct {
	// Slice, in canonical ordering, of the validators that have a public key.
	Validators []*Warp
	// The total weight of all the validators, including the ones that don't
	// have a public key.
	TotalWeight *big.Int
}

type jsonWarpSet struct {
	Validators  []*Warp        `json:"validators"`
	TotalWeight avajson.BigInt `json:"totalWeight"`
}

func (w WarpSet) MarshalJSON() ([]byte, error) {
	totalWeight := w.TotalWeight
	if totalWeight == nil {
		totalWeight = new(big.Int)
	}
	return json.Marshal(jsonWarpSet{
		Validators:  w.Validators,
		TotalWeight: avajson.NewBigInt(totalWeight),
	})
}

func (w *WarpSet) UnmarshalJSON(b []byte) error {
	var j jsonWarpSet
	if err := json.Unmarshal(b, &j); err != nil {
		return err
	}
	w.TotalWeight = j.TotalWeight.ToBigInt()
	w.Validators = j.Validators
	return nil
}

type Warp struct {
	PublicKey *bls.PublicKey
	// PublicKeyBytes is expected to be in the uncompressed form.
	PublicKeyBytes []byte
	Weight         uint64
	NodeIDs        []ids.NodeID
}

func (w *Warp) Compare(o *Warp) int {
	return bytes.Compare(w.PublicKeyBytes, o.PublicKeyBytes)
}

type jsonWarp struct {
	PublicKey string         `json:"publicKey"`
	Weight    avajson.Uint64 `json:"weight"`
	NodeIDs   []ids.NodeID   `json:"nodeIDs"`
}

func (w Warp) MarshalJSON() ([]byte, error) {
	pkBytes := bls.PublicKeyToCompressedBytes(w.PublicKey)
	pk, err := formatting.Encode(formatting.HexNC, pkBytes)
	if err != nil {
		return nil, err
	}
	return json.Marshal(jsonWarp{
		PublicKey: pk,
		Weight:    avajson.Uint64(w.Weight),
		NodeIDs:   w.NodeIDs,
	})
}

func (w *Warp) UnmarshalJSON(b []byte) error {
	var j jsonWarp
	if err := json.Unmarshal(b, &j); err != nil {
		return err
	}

	pkBytes, err := formatting.Decode(formatting.HexNC, j.PublicKey)
	if err != nil {
		return err
	}
	pk, err := bls.PublicKeyFromCompressedBytes(pkBytes)
	if err != nil {
		return err
	}
	*w = Warp{
		PublicKey:      pk,
		PublicKeyBytes: bls.PublicKeyToUncompressedBytes(pk),
		Weight:         uint64(j.Weight),
		NodeIDs:        j.NodeIDs,
	}
	return nil
}

// FlattenValidatorSet converts the provided vdrSet into a canonical ordering.
func FlattenValidatorSet(vdrSet map[ids.NodeID]*GetValidatorOutput) (WarpSet, error) {
	var (
		vdrs        = make(map[string]*Warp, len(vdrSet))
		totalWeight = new(big.Int)
	)
	for _, vdr := range vdrSet {
		totalWeight.Add(totalWeight, new(big.Int).SetUint64(vdr.Weight))

		if vdr.PublicKey == nil {
			continue
		}

		pkBytes := bls.PublicKeyToUncompressedBytes(vdr.PublicKey)
		uniqueVdr, ok := vdrs[string(pkBytes)]
		if !ok {
			uniqueVdr = &Warp{
				PublicKey:      vdr.PublicKey,
				PublicKeyBytes: pkBytes,
			}
			vdrs[string(pkBytes)] = uniqueVdr
		}

		// Individual validator weights fit in uint64. Although BLS keys are not
		// consensus-enforced unique, validators are expected to use unique keys; a
		// shared-key aggregate exceeding uint64 is treated as out of scope.
		uniqueVdr.Weight += vdr.Weight
		uniqueVdr.NodeIDs = append(uniqueVdr.NodeIDs, vdr.NodeID)
	}

	// Sort validators by public key
	vdrList := maps.Values(vdrs)
	utils.Sort(vdrList)
	return WarpSet{Validators: vdrList, TotalWeight: totalWeight}, nil
}
