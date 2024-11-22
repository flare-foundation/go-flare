// Copyright (C) 2019-2022, Ava Labs, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package signer

import (
	"github.com/ava-labs/avalanchego/utils/crypto/bls"
)

var _ Signer = &Empty{}

type Empty struct{}

func (*Empty) Verify() error       { return nil }
func (*Empty) Key() *bls.PublicKey { return nil }
