package utils

import "math/big"

type ChainValue[T any] struct {
	valueMap     map[uint64]T
	defaultValue T
}

func NewChainValue[T any](defaultValue T) *ChainValue[T] {
	return &ChainValue[T]{
		valueMap:     make(map[uint64]T),
		defaultValue: defaultValue,
	}
}

func (ca *ChainValue[T]) AddValue(chainID *big.Int, action T) *ChainValue[T] {
	ca.valueMap[chainID.Uint64()] = action
	return ca
}

func (ca *ChainValue[T]) AddValues(chainIDs []*big.Int, action T) *ChainValue[T] {
	for _, chainID := range chainIDs {
		ca.valueMap[chainID.Uint64()] = action
	}
	return ca
}

func (ca *ChainValue[T]) GetValue(chainID *big.Int) T {
	if action, ok := ca.valueMap[chainID.Uint64()]; ok {
		return action
	}
	return ca.defaultValue
}
