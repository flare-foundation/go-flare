// Copyright (C) 2019-2022, Ava Labs, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package avm

import (
	"context"
	"fmt"

	"github.com/ava-labs/avalanchego/api"
	"github.com/ava-labs/avalanchego/ids"
	"github.com/ava-labs/avalanchego/utils/constants"
	"github.com/ava-labs/avalanchego/utils/formatting"
	"github.com/ava-labs/avalanchego/utils/json"
	"github.com/ava-labs/avalanchego/utils/rpc"
)

var _ WalletClient = &client{}

// interface of an AVM wallet client for interacting with avm managed wallet on [chain]
type WalletClient interface {
	// IssueTx issues a transaction to a node and returns the TxID
	IssueTx(ctx context.Context, tx []byte, options ...rpc.Option) (ids.ID, error)
	// Send [amount] of [assetID] to address [to]
	Send(
		ctx context.Context,
		user api.UserPass,
		from []ids.ShortID,
		changeAddr ids.ShortID,
		amount uint64,
		assetID string,
		to ids.ShortID,
		memo string,
		options ...rpc.Option,
	) (ids.ID, error)
	// SendMultiple sends a transaction from [user] funding all [outputs]
	SendMultiple(
		ctx context.Context,
		user api.UserPass,
		from []ids.ShortID,
		changeAddr ids.ShortID,
		outputs []ClientSendOutput,
		memo string,
		options ...rpc.Option,
	) (ids.ID, error)
}

// implementation of an AVM wallet client for interacting with avm managed wallet on [chain]
type walletClient struct {
	requester rpc.EndpointRequester
}

// NewWalletClient returns an AVM wallet client for interacting with avm managed wallet on [chain]
func NewWalletClient(uri, chain string) WalletClient {
	path := fmt.Sprintf(
		"%s/ext/%s/%s/wallet",
		uri,
		constants.ChainAliasPrefix,
		chain,
	)
	return &walletClient{
		requester: rpc.NewEndpointRequester(path, "wallet"),
	}
}

func (c *walletClient) IssueTx(ctx context.Context, txBytes []byte, options ...rpc.Option) (ids.ID, error) {
	txStr, err := formatting.Encode(formatting.Hex, txBytes)
	if err != nil {
		return ids.ID{}, err
	}
	res := &api.JSONTxID{}
	err = c.requester.SendRequest(ctx, "issueTx", &api.FormattedTx{
		Tx:       txStr,
		Encoding: formatting.Hex,
	}, res, options...)
	return res.TxID, err
}

// ClientSendOutput specifies that [Amount] of asset [AssetID] be sent to [To]
type ClientSendOutput struct {
	// The amount of funds to send
	Amount uint64

	// ID of the asset being sent
	AssetID string

	// Address of the recipient
	To ids.ShortID
}

func (c *walletClient) Send(
	ctx context.Context,
	user api.UserPass,
	from []ids.ShortID,
	changeAddr ids.ShortID,
	amount uint64,
	assetID string,
	to ids.ShortID,
	memo string,
	options ...rpc.Option,
) (ids.ID, error) {
	res := &api.JSONTxID{}
	err := c.requester.SendRequest(ctx, "send", &SendArgs{
		JSONSpendHeader: api.JSONSpendHeader{
			UserPass:       user,
			JSONFromAddrs:  api.JSONFromAddrs{From: ids.ShortIDsToStrings(from)},
			JSONChangeAddr: api.JSONChangeAddr{ChangeAddr: changeAddr.String()},
		},
		SendOutput: SendOutput{
			Amount:  json.Uint64(amount),
			AssetID: assetID,
			To:      to.String(),
		},
		Memo: memo,
	}, res, options...)
	return res.TxID, err
}

func (c *walletClient) SendMultiple(
	ctx context.Context,
	user api.UserPass,
	from []ids.ShortID,
	changeAddr ids.ShortID,
	outputs []ClientSendOutput,
	memo string,
	options ...rpc.Option,
) (ids.ID, error) {
	res := &api.JSONTxID{}
	serviceOutputs := make([]SendOutput, len(outputs))
	for i, output := range outputs {
		serviceOutputs[i].Amount = json.Uint64(output.Amount)
		serviceOutputs[i].AssetID = output.AssetID
		serviceOutputs[i].To = output.To.String()
	}
	err := c.requester.SendRequest(ctx, "sendMultiple", &SendMultipleArgs{
		JSONSpendHeader: api.JSONSpendHeader{
			UserPass:       user,
			JSONFromAddrs:  api.JSONFromAddrs{From: ids.ShortIDsToStrings(from)},
			JSONChangeAddr: api.JSONChangeAddr{ChangeAddr: changeAddr.String()},
		},
		Outputs: serviceOutputs,
		Memo:    memo,
	}, res, options...)
	return res.TxID, err
}
