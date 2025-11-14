// Copyright (C) 2019-2024, Ava Labs, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package json

import (
	"math/big"
)

type BigInt struct {
	value *big.Int
}

func NewBigInt(value *big.Int) BigInt {
	return BigInt{value: new(big.Int).Set(value)}
}

func NewBigIntFromInt(value int64) BigInt {
	return BigInt{value: big.NewInt(value)}
}

func (b *BigInt) Set(value *big.Int) {
	if value == nil {
		b.value = nil
	} else if b.value == nil {
		b.value = new(big.Int).Set(value)
	} else {
		b.value.Set(value)
	}
}

func (b BigInt) ToBigInt() *big.Int {
	if b.value == nil {
		return nil
	}
	return new(big.Int).Set(b.value)
}

func (b BigInt) MarshalJSON() ([]byte, error) {
	if b.value == nil {
		return []byte("null"), nil
	}
	text, err := b.value.MarshalText()
	if err != nil {
		return nil, err
	}
	return []byte(`"` + string(text) + `"`), nil
}

func (b *BigInt) UnmarshalJSON(data []byte) error {
	if len(data) >= 2 && data[0] == '"' && data[len(data)-1] == '"' {
		data = data[1 : len(data)-1]
	}
	if string(data) == "null" {
		b.value = nil
		return nil
	}
	if b.value == nil {
		b.value = new(big.Int)
	}
	return b.value.UnmarshalText(data)
}
