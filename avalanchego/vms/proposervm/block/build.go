// Copyright (C) 2019-2022, Ava Labs, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package block

import (
	"crypto"
	"crypto/rand"
	"crypto/x509"
	"time"

	"github.com/ava-labs/avalanchego/codec"
	"github.com/ava-labs/avalanchego/ids"
	"github.com/ava-labs/avalanchego/utils/hashing"
	"github.com/ava-labs/avalanchego/utils/wrappers"
)

func BuildUnsignedApricot(
	parentID ids.ID,
	timestamp time.Time,
	pChainHeight uint64,
	blockBytes []byte,
) (SignedBlock, error) {
	return buildUnsigned(apricotCodec, parentID, timestamp, pChainHeight, blockBytes)
}

func BuildUnsignedBanff(
	parentID ids.ID,
	timestamp time.Time,
	pChainHeight uint64,
	blockBytes []byte,
) (SignedBlock, error) {
	return buildUnsigned(banffCodec, parentID, timestamp, pChainHeight, blockBytes)
}

func buildUnsigned(
	cm codec.Manager,
	parentID ids.ID,
	timestamp time.Time,
	pChainHeight uint64,
	blockBytes []byte,
) (SignedBlock, error) {
	var block SignedBlock = &statelessBlock{
		StatelessBlock: statelessUnsignedBlock{
			ParentID:     parentID,
			Timestamp:    timestamp.Unix(),
			PChainHeight: pChainHeight,
			Certificate:  nil,
			Block:        blockBytes,
		},
		timestamp: timestamp,
	}

	bytes, err := cm.Marshal(codecVersion, &block)
	if err != nil {
		return nil, err
	}
	return block, block.initialize(bytes)
}

func BuildApricot(
	parentID ids.ID,
	timestamp time.Time,
	pChainHeight uint64,
	cert *x509.Certificate,
	blockBytes []byte,
	chainID ids.ID,
	key crypto.Signer,
) (SignedBlock, error) {
	return build(apricotCodec, parentID, timestamp, pChainHeight, cert, blockBytes, chainID, key)
}

func BuildBanff(
	parentID ids.ID,
	timestamp time.Time,
	pChainHeight uint64,
	cert *x509.Certificate,
	blockBytes []byte,
	chainID ids.ID,
	key crypto.Signer,
) (SignedBlock, error) {
	return build(banffCodec, parentID, timestamp, pChainHeight, cert, blockBytes, chainID, key)
}

func build(
	cm codec.Manager,
	parentID ids.ID,
	timestamp time.Time,
	pChainHeight uint64,
	cert *x509.Certificate,
	blockBytes []byte,
	chainID ids.ID,
	key crypto.Signer,
) (SignedBlock, error) {
	block := &statelessBlock{
		StatelessBlock: statelessUnsignedBlock{
			ParentID:     parentID,
			Timestamp:    timestamp.Unix(),
			PChainHeight: pChainHeight,
			Certificate:  cert.Raw,
			Block:        blockBytes,
		},
		timestamp: timestamp,
		cert:      cert,
		proposer:  ids.NodeIDFromCert(cert),
	}
	var blockIntf SignedBlock = block

	unsignedBytesWithEmptySignature, err := cm.Marshal(codecVersion, &blockIntf)
	if err != nil {
		return nil, err
	}

	// The serialized form of the block is the unsignedBytes followed by the
	// signature, which is prefixed by a uint32. Because we are marshalling the
	// block with an empty signature, we only need to strip off the length
	// prefix to get the unsigned bytes.
	lenUnsignedBytes := len(unsignedBytesWithEmptySignature) - wrappers.IntLen
	unsignedBytes := unsignedBytesWithEmptySignature[:lenUnsignedBytes]
	block.id = hashing.ComputeHash256Array(unsignedBytes)

	header, err := BuildHeader(chainID, parentID, block.id)
	if err != nil {
		return nil, err
	}

	headerHash := hashing.ComputeHash256(header.Bytes())
	block.Signature, err = key.Sign(rand.Reader, headerHash, crypto.SHA256)
	if err != nil {
		return nil, err
	}

	block.bytes, err = cm.Marshal(codecVersion, &blockIntf)
	return block, err
}

func BuildHeader(
	chainID ids.ID,
	parentID ids.ID,
	bodyID ids.ID,
) (Header, error) {
	header := statelessHeader{
		Chain:  chainID,
		Parent: parentID,
		Body:   bodyID,
	}

	bytes, err := banffCodec.Marshal(codecVersion, &header)
	header.bytes = bytes
	return &header, err
}

// BuildOption the option block
// [parentID] is the ID of this option's wrapper parent block
// [innerBytes] is the byte representation of a child option block
func BuildOption(
	parentID ids.ID,
	innerBytes []byte,
) (Block, error) {
	var block Block = &option{
		PrntID:     parentID,
		InnerBytes: innerBytes,
	}

	bytes, err := banffCodec.Marshal(codecVersion, &block)
	if err != nil {
		return nil, err
	}
	return block, block.initialize(bytes)
}
