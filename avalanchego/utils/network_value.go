package utils

type NetworkValue[T any] struct {
	valueMap     map[uint32]T
	defaultValue T
}

func NewNetworkValue[T any](defaultValue T) *NetworkValue[T] {
	return &NetworkValue[T]{
		valueMap:     make(map[uint32]T),
		defaultValue: defaultValue,
	}
}

func (ca *NetworkValue[T]) AddValue(networkID uint32, action T) *NetworkValue[T] {
	ca.valueMap[networkID] = action
	return ca
}

func (ca *NetworkValue[T]) AddValues(networkIDs []uint32, action T) *NetworkValue[T] {
	for _, networkID := range networkIDs {
		ca.valueMap[networkID] = action
	}
	return ca
}

func (ca *NetworkValue[T]) GetValue(networkID uint32) T {
	if action, ok := ca.valueMap[networkID]; ok {
		return action
	}
	return ca.defaultValue
}
