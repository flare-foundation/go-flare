// Copyright (C) 2019-2022, Ava Labs, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package mempool

import "github.com/ava-labs/avalanchego/vms/platformvm/txs"

var _ txs.Visitor = &remover{}

type remover struct {
	m  *mempool
	tx *txs.Tx
}

func (r *remover) AddValidatorTx(*txs.AddValidatorTx) error {
	r.m.removeStakerTx(r.tx)
	return nil
}

func (r *remover) AddSubnetValidatorTx(*txs.AddSubnetValidatorTx) error {
	r.m.removeStakerTx(r.tx)
	return nil
}

func (r *remover) AddDelegatorTx(*txs.AddDelegatorTx) error {
	r.m.removeStakerTx(r.tx)
	return nil
}

func (r *remover) RemoveSubnetValidatorTx(tx *txs.RemoveSubnetValidatorTx) error {
	r.m.removeDecisionTxs([]*txs.Tx{r.tx})
	return nil
}

func (r *remover) CreateChainTx(*txs.CreateChainTx) error {
	r.m.removeDecisionTxs([]*txs.Tx{r.tx})
	return nil
}

func (r *remover) CreateSubnetTx(*txs.CreateSubnetTx) error {
	r.m.removeDecisionTxs([]*txs.Tx{r.tx})
	return nil
}

func (r *remover) ImportTx(*txs.ImportTx) error {
	r.m.removeDecisionTxs([]*txs.Tx{r.tx})
	return nil
}

func (r *remover) ExportTx(*txs.ExportTx) error {
	r.m.removeDecisionTxs([]*txs.Tx{r.tx})
	return nil
}

func (r *remover) TransformSubnetTx(tx *txs.TransformSubnetTx) error {
	r.m.removeDecisionTxs([]*txs.Tx{r.tx})
	return nil
}

func (r *remover) AddPermissionlessValidatorTx(tx *txs.AddPermissionlessValidatorTx) error {
	r.m.removeStakerTx(r.tx)
	return nil
}

func (r *remover) AddPermissionlessDelegatorTx(tx *txs.AddPermissionlessDelegatorTx) error {
	r.m.removeStakerTx(r.tx)
	return nil
}

func (r *remover) AdvanceTimeTx(*txs.AdvanceTimeTx) error {
	// this tx is never in mempool
	return nil
}

func (r *remover) RewardValidatorTx(*txs.RewardValidatorTx) error {
	// this tx is never in mempool
	return nil
}
