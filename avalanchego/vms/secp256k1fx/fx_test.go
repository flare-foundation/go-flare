// Copyright (C) 2019-2022, Ava Labs, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package secp256k1fx

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/ava-labs/avalanchego/codec/linearcodec"
	"github.com/ava-labs/avalanchego/ids"
	"github.com/ava-labs/avalanchego/utils/cb58"
	"github.com/ava-labs/avalanchego/utils/crypto"
	"github.com/ava-labs/avalanchego/utils/logging"
)

var (
	txBytes  = []byte{0, 1, 2, 3, 4, 5}
	sigBytes = [crypto.SECP256K1RSigLen]byte{ // signature of addr on txBytes
		0x0e, 0x33, 0x4e, 0xbc, 0x67, 0xa7, 0x3f, 0xe8,
		0x24, 0x33, 0xac, 0xa3, 0x47, 0x88, 0xa6, 0x3d,
		0x58, 0xe5, 0x8e, 0xf0, 0x3a, 0xd5, 0x84, 0xf1,
		0xbc, 0xa3, 0xb2, 0xd2, 0x5d, 0x51, 0xd6, 0x9b,
		0x0f, 0x28, 0x5d, 0xcd, 0x3f, 0x71, 0x17, 0x0a,
		0xf9, 0xbf, 0x2d, 0xb1, 0x10, 0x26, 0x5c, 0xe9,
		0xdc, 0xc3, 0x9d, 0x7a, 0x01, 0x50, 0x9d, 0xe8,
		0x35, 0xbd, 0xcb, 0x29, 0x3a, 0xd1, 0x49, 0x32,
		0x00,
	}
	addr = ids.ShortID{
		0x01, 0x5c, 0xce, 0x6c, 0x55, 0xd6, 0xb5, 0x09,
		0x84, 0x5c, 0x8c, 0x4e, 0x30, 0xbe, 0xd9, 0x8d,
		0x39, 0x1a, 0xe7, 0xf0,
	}
	addr2     ids.ShortID
	sig2Bytes [crypto.SECP256K1RSigLen]byte // signature of addr2 on txBytes
)

func init() {
	b, err := cb58.Decode("31SoC6ehdWUWFcuzkXci7ymFEQ8HGTJgw")
	if err != nil {
		panic(err)
	}
	copy(addr2[:], b)
	b, err = cb58.Decode("c7doHa86hWYyfXTVnNsdP1CG1gxhXVpZ9Q5CiHi2oFRdnaxh2YR2Mvu2cUNMgyQy4BNQaXAxWWPt36BJ5pDWX1Xeos4h9L")
	if err != nil {
		panic(err)
	}
	copy(sig2Bytes[:], b)
}

func TestFxInitialize(t *testing.T) {
	require := require.New(t)
	vm := TestVM{
		Codec: linearcodec.NewDefault(),
		Log:   logging.NoLog{},
	}
	fx := Fx{}
	require.NoError(fx.Initialize(&vm))
}

func TestFxInitializeInvalid(t *testing.T) {
	require := require.New(t)
	fx := Fx{}
	require.ErrorIs(fx.Initialize(nil), errWrongVMType)
}

func TestFxVerifyTransfer(t *testing.T) {
	require := require.New(t)
	vm := TestVM{
		Codec: linearcodec.NewDefault(),
		Log:   logging.NoLog{},
	}
	date := time.Date(2019, time.January, 19, 16, 25, 17, 3, time.UTC)
	vm.CLK.Set(date)
	fx := Fx{}
	require.NoError(fx.Initialize(&vm))
	require.NoError(fx.Bootstrapping())
	require.NoError(fx.Bootstrapped())
	tx := &TestTx{UnsignedBytes: txBytes}
	out := &TransferOutput{
		Amt: 1,
		OutputOwners: OutputOwners{
			Locktime:  0,
			Threshold: 1,
			Addrs: []ids.ShortID{
				addr,
			},
		},
	}
	in := &TransferInput{
		Amt: 1,
		Input: Input{
			SigIndices: []uint32{0},
		},
	}
	cred := &Credential{
		Sigs: [][crypto.SECP256K1RSigLen]byte{
			sigBytes,
		},
	}

	require.NoError(fx.VerifyTransfer(tx, in, cred, out))
}

func TestFxVerifyTransferNilTx(t *testing.T) {
	require := require.New(t)
	vm := TestVM{
		Codec: linearcodec.NewDefault(),
		Log:   logging.NoLog{},
	}
	date := time.Date(2019, time.January, 19, 16, 25, 17, 3, time.UTC)
	vm.CLK.Set(date)
	fx := Fx{}
	require.NoError(fx.Initialize(&vm))
	out := &TransferOutput{
		Amt: 1,
		OutputOwners: OutputOwners{
			Locktime:  0,
			Threshold: 1,
			Addrs: []ids.ShortID{
				addr,
			},
		},
	}
	in := &TransferInput{
		Amt: 1,
		Input: Input{
			SigIndices: []uint32{0},
		},
	}
	cred := &Credential{
		Sigs: [][crypto.SECP256K1RSigLen]byte{
			sigBytes,
		},
	}

	require.ErrorIs(fx.VerifyTransfer(nil, in, cred, out), errWrongTxType)
}

func TestFxVerifyTransferNilOutput(t *testing.T) {
	require := require.New(t)
	vm := TestVM{
		Codec: linearcodec.NewDefault(),
		Log:   logging.NoLog{},
	}
	date := time.Date(2019, time.January, 19, 16, 25, 17, 3, time.UTC)
	vm.CLK.Set(date)
	fx := Fx{}
	require.NoError(fx.Initialize(&vm))
	tx := &TestTx{UnsignedBytes: txBytes}
	in := &TransferInput{
		Amt: 1,
		Input: Input{
			SigIndices: []uint32{0},
		},
	}
	cred := &Credential{
		Sigs: [][crypto.SECP256K1RSigLen]byte{
			sigBytes,
		},
	}

	require.ErrorIs(fx.VerifyTransfer(tx, in, cred, nil), errWrongUTXOType)
}

func TestFxVerifyTransferNilInput(t *testing.T) {
	require := require.New(t)
	vm := TestVM{
		Codec: linearcodec.NewDefault(),
		Log:   logging.NoLog{},
	}
	date := time.Date(2019, time.January, 19, 16, 25, 17, 3, time.UTC)
	vm.CLK.Set(date)
	fx := Fx{}
	require.NoError(fx.Initialize(&vm))
	tx := &TestTx{UnsignedBytes: txBytes}
	out := &TransferOutput{
		Amt: 1,
		OutputOwners: OutputOwners{
			Locktime:  0,
			Threshold: 1,
			Addrs: []ids.ShortID{
				addr,
			},
		},
	}
	cred := &Credential{
		Sigs: [][crypto.SECP256K1RSigLen]byte{
			sigBytes,
		},
	}

	require.ErrorIs(fx.VerifyTransfer(tx, nil, cred, out), errWrongInputType)
}

func TestFxVerifyTransferNilCredential(t *testing.T) {
	require := require.New(t)
	vm := TestVM{
		Codec: linearcodec.NewDefault(),
		Log:   logging.NoLog{},
	}
	date := time.Date(2019, time.January, 19, 16, 25, 17, 3, time.UTC)
	vm.CLK.Set(date)
	fx := Fx{}
	require.NoError(fx.Initialize(&vm))
	tx := &TestTx{UnsignedBytes: txBytes}
	out := &TransferOutput{
		Amt: 1,
		OutputOwners: OutputOwners{
			Locktime:  0,
			Threshold: 1,
			Addrs: []ids.ShortID{
				addr,
			},
		},
	}
	in := &TransferInput{
		Amt: 1,
		Input: Input{
			SigIndices: []uint32{0},
		},
	}

	require.ErrorIs(fx.VerifyTransfer(tx, in, nil, out), errWrongCredentialType)
}

func TestFxVerifyTransferInvalidOutput(t *testing.T) {
	require := require.New(t)
	vm := TestVM{
		Codec: linearcodec.NewDefault(),
		Log:   logging.NoLog{},
	}
	date := time.Date(2019, time.January, 19, 16, 25, 17, 3, time.UTC)
	vm.CLK.Set(date)
	fx := Fx{}
	require.NoError(fx.Initialize(&vm))
	tx := &TestTx{UnsignedBytes: txBytes}
	out := &TransferOutput{
		Amt: 1,
		OutputOwners: OutputOwners{
			Locktime:  0,
			Threshold: 0,
			Addrs: []ids.ShortID{
				addr,
			},
		},
	}
	in := &TransferInput{
		Amt: 1,
		Input: Input{
			SigIndices: []uint32{0},
		},
	}
	cred := &Credential{
		Sigs: [][crypto.SECP256K1RSigLen]byte{
			sigBytes,
		},
	}

	require.ErrorIs(fx.VerifyTransfer(tx, in, cred, out), errOutputUnoptimized)
}

func TestFxVerifyTransferWrongAmounts(t *testing.T) {
	require := require.New(t)
	vm := TestVM{
		Codec: linearcodec.NewDefault(),
		Log:   logging.NoLog{},
	}
	date := time.Date(2019, time.January, 19, 16, 25, 17, 3, time.UTC)
	vm.CLK.Set(date)
	fx := Fx{}
	require.NoError(fx.Initialize(&vm))
	tx := &TestTx{UnsignedBytes: txBytes}
	out := &TransferOutput{
		Amt: 1,
		OutputOwners: OutputOwners{
			Locktime:  0,
			Threshold: 1,
			Addrs: []ids.ShortID{
				addr,
			},
		},
	}
	in := &TransferInput{
		Amt: 2,
		Input: Input{
			SigIndices: []uint32{0},
		},
	}
	cred := &Credential{
		Sigs: [][crypto.SECP256K1RSigLen]byte{
			sigBytes,
		},
	}

	require.Error(fx.VerifyTransfer(tx, in, cred, out))
}

func TestFxVerifyTransferTimelocked(t *testing.T) {
	require := require.New(t)
	vm := TestVM{
		Codec: linearcodec.NewDefault(),
		Log:   logging.NoLog{},
	}
	date := time.Date(2019, time.January, 19, 16, 25, 17, 3, time.UTC)
	vm.CLK.Set(date)
	fx := Fx{}
	require.NoError(fx.Initialize(&vm))
	tx := &TestTx{UnsignedBytes: txBytes}
	out := &TransferOutput{
		Amt: 1,
		OutputOwners: OutputOwners{
			Locktime:  uint64(date.Add(time.Second).Unix()),
			Threshold: 1,
			Addrs: []ids.ShortID{
				addr,
			},
		},
	}
	in := &TransferInput{
		Amt: 1,
		Input: Input{
			SigIndices: []uint32{0},
		},
	}
	cred := &Credential{
		Sigs: [][crypto.SECP256K1RSigLen]byte{
			sigBytes,
		},
	}

	require.ErrorIs(fx.VerifyTransfer(tx, in, cred, out), errTimelocked)
}

func TestFxVerifyTransferTooManySigners(t *testing.T) {
	require := require.New(t)
	vm := TestVM{
		Codec: linearcodec.NewDefault(),
		Log:   logging.NoLog{},
	}
	date := time.Date(2019, time.January, 19, 16, 25, 17, 3, time.UTC)
	vm.CLK.Set(date)
	fx := Fx{}
	require.NoError(fx.Initialize(&vm))
	tx := &TestTx{UnsignedBytes: txBytes}
	out := &TransferOutput{
		Amt: 1,
		OutputOwners: OutputOwners{
			Locktime:  0,
			Threshold: 1,
			Addrs: []ids.ShortID{
				addr,
			},
		},
	}
	in := &TransferInput{
		Amt: 1,
		Input: Input{
			SigIndices: []uint32{0, 1},
		},
	}
	cred := &Credential{
		Sigs: [][crypto.SECP256K1RSigLen]byte{
			sigBytes,
			{},
		},
	}

	require.ErrorIs(fx.VerifyTransfer(tx, in, cred, out), errTooManySigners)
}

func TestFxVerifyTransferTooFewSigners(t *testing.T) {
	require := require.New(t)
	vm := TestVM{
		Codec: linearcodec.NewDefault(),
		Log:   logging.NoLog{},
	}
	date := time.Date(2019, time.January, 19, 16, 25, 17, 3, time.UTC)
	vm.CLK.Set(date)
	fx := Fx{}
	require.NoError(fx.Initialize(&vm))
	tx := &TestTx{UnsignedBytes: txBytes}
	out := &TransferOutput{
		Amt: 1,
		OutputOwners: OutputOwners{
			Locktime:  0,
			Threshold: 1,
			Addrs: []ids.ShortID{
				addr,
			},
		},
	}
	in := &TransferInput{
		Amt: 1,
		Input: Input{
			SigIndices: []uint32{},
		},
	}
	cred := &Credential{
		Sigs: [][crypto.SECP256K1RSigLen]byte{},
	}

	require.ErrorIs(fx.VerifyTransfer(tx, in, cred, out), errTooFewSigners)
}

func TestFxVerifyTransferMismatchedSigners(t *testing.T) {
	require := require.New(t)
	vm := TestVM{
		Codec: linearcodec.NewDefault(),
		Log:   logging.NoLog{},
	}
	date := time.Date(2019, time.January, 19, 16, 25, 17, 3, time.UTC)
	vm.CLK.Set(date)
	fx := Fx{}
	require.NoError(fx.Initialize(&vm))
	tx := &TestTx{UnsignedBytes: txBytes}
	out := &TransferOutput{
		Amt: 1,
		OutputOwners: OutputOwners{
			Locktime:  0,
			Threshold: 1,
			Addrs: []ids.ShortID{
				addr,
			},
		},
	}
	in := &TransferInput{
		Amt: 1,
		Input: Input{
			SigIndices: []uint32{0},
		},
	}
	cred := &Credential{
		Sigs: [][crypto.SECP256K1RSigLen]byte{
			sigBytes,
			{},
		},
	}

	require.ErrorIs(fx.VerifyTransfer(tx, in, cred, out), errInputCredentialSignersMismatch)
}

func TestFxVerifyTransferInvalidSignature(t *testing.T) {
	require := require.New(t)
	vm := TestVM{
		Codec: linearcodec.NewDefault(),
		Log:   logging.NoLog{},
	}
	date := time.Date(2019, time.January, 19, 16, 25, 17, 3, time.UTC)
	vm.CLK.Set(date)
	fx := Fx{}
	require.NoError(fx.Initialize(&vm))
	require.NoError(fx.Bootstrapping())
	tx := &TestTx{UnsignedBytes: txBytes}
	out := &TransferOutput{
		Amt: 1,
		OutputOwners: OutputOwners{
			Locktime:  0,
			Threshold: 1,
			Addrs: []ids.ShortID{
				addr,
			},
		},
	}
	in := &TransferInput{
		Amt: 1,
		Input: Input{
			SigIndices: []uint32{0},
		},
	}
	cred := &Credential{
		Sigs: [][crypto.SECP256K1RSigLen]byte{
			{},
		},
	}

	require.NoError(fx.VerifyTransfer(tx, in, cred, out))
	require.NoError(fx.Bootstrapped())
	require.Error(fx.VerifyTransfer(tx, in, cred, out), errAddrsNotSortedUnique)
}

func TestFxVerifyTransferWrongSigner(t *testing.T) {
	require := require.New(t)
	vm := TestVM{
		Codec: linearcodec.NewDefault(),
		Log:   logging.NoLog{},
	}
	date := time.Date(2019, time.January, 19, 16, 25, 17, 3, time.UTC)
	vm.CLK.Set(date)
	fx := Fx{}
	require.NoError(fx.Initialize(&vm))
	require.NoError(fx.Bootstrapping())
	tx := &TestTx{UnsignedBytes: txBytes}
	out := &TransferOutput{
		Amt: 1,
		OutputOwners: OutputOwners{
			Locktime:  0,
			Threshold: 1,
			Addrs: []ids.ShortID{
				ids.ShortEmpty,
			},
		},
	}
	in := &TransferInput{
		Amt: 1,
		Input: Input{
			SigIndices: []uint32{0},
		},
	}
	cred := &Credential{
		Sigs: [][crypto.SECP256K1RSigLen]byte{
			sigBytes,
		},
	}

	require.NoError(fx.VerifyTransfer(tx, in, cred, out))
	require.NoError(fx.Bootstrapped())
	require.Error(fx.VerifyTransfer(tx, in, cred, out))
}

func TestFxVerifyOperation(t *testing.T) {
	require := require.New(t)
	vm := TestVM{
		Codec: linearcodec.NewDefault(),
		Log:   logging.NoLog{},
	}
	date := time.Date(2019, time.January, 19, 16, 25, 17, 3, time.UTC)
	vm.CLK.Set(date)
	fx := Fx{}
	require.NoError(fx.Initialize(&vm))
	tx := &TestTx{UnsignedBytes: txBytes}
	utxo := &MintOutput{
		OutputOwners: OutputOwners{
			Threshold: 1,
			Addrs: []ids.ShortID{
				addr,
			},
		},
	}
	op := &MintOperation{
		MintInput: Input{
			SigIndices: []uint32{0},
		},
		MintOutput: MintOutput{
			OutputOwners: OutputOwners{
				Threshold: 1,
				Addrs: []ids.ShortID{
					addr,
				},
			},
		},
		TransferOutput: TransferOutput{
			Amt: 1,
			OutputOwners: OutputOwners{
				Locktime:  0,
				Threshold: 1,
				Addrs: []ids.ShortID{
					addr,
				},
			},
		},
	}
	cred := &Credential{
		Sigs: [][crypto.SECP256K1RSigLen]byte{
			sigBytes,
		},
	}

	utxos := []interface{}{utxo}
	require.NoError(fx.VerifyOperation(tx, op, cred, utxos))
}

func TestFxVerifyOperationUnknownTx(t *testing.T) {
	require := require.New(t)
	vm := TestVM{
		Codec: linearcodec.NewDefault(),
		Log:   logging.NoLog{},
	}
	date := time.Date(2019, time.January, 19, 16, 25, 17, 3, time.UTC)
	vm.CLK.Set(date)
	fx := Fx{}
	require.NoError(fx.Initialize(&vm))
	utxo := &MintOutput{
		OutputOwners: OutputOwners{
			Threshold: 1,
			Addrs: []ids.ShortID{
				addr,
			},
		},
	}
	op := &MintOperation{
		MintInput: Input{
			SigIndices: []uint32{0},
		},
		MintOutput: MintOutput{
			OutputOwners: OutputOwners{
				Threshold: 1,
				Addrs: []ids.ShortID{
					addr,
				},
			},
		},
		TransferOutput: TransferOutput{
			Amt: 1,
			OutputOwners: OutputOwners{
				Locktime:  0,
				Threshold: 1,
				Addrs: []ids.ShortID{
					addr,
				},
			},
		},
	}
	cred := &Credential{
		Sigs: [][crypto.SECP256K1RSigLen]byte{
			sigBytes,
		},
	}

	utxos := []interface{}{utxo}
	require.ErrorIs(fx.VerifyOperation(nil, op, cred, utxos), errWrongTxType)
}

func TestFxVerifyOperationUnknownOperation(t *testing.T) {
	require := require.New(t)
	vm := TestVM{
		Codec: linearcodec.NewDefault(),
		Log:   logging.NoLog{},
	}
	date := time.Date(2019, time.January, 19, 16, 25, 17, 3, time.UTC)
	vm.CLK.Set(date)
	fx := Fx{}
	require.NoError(fx.Initialize(&vm))
	tx := &TestTx{UnsignedBytes: txBytes}
	utxo := &MintOutput{
		OutputOwners: OutputOwners{
			Threshold: 1,
			Addrs: []ids.ShortID{
				addr,
			},
		},
	}
	cred := &Credential{
		Sigs: [][crypto.SECP256K1RSigLen]byte{
			sigBytes,
		},
	}

	utxos := []interface{}{utxo}
	require.ErrorIs(fx.VerifyOperation(tx, nil, cred, utxos), errWrongOpType)
}

func TestFxVerifyOperationUnknownCredential(t *testing.T) {
	require := require.New(t)
	vm := TestVM{
		Codec: linearcodec.NewDefault(),
		Log:   logging.NoLog{},
	}
	date := time.Date(2019, time.January, 19, 16, 25, 17, 3, time.UTC)
	vm.CLK.Set(date)
	fx := Fx{}
	require.NoError(fx.Initialize(&vm))
	tx := &TestTx{UnsignedBytes: txBytes}
	utxo := &MintOutput{
		OutputOwners: OutputOwners{
			Threshold: 1,
			Addrs: []ids.ShortID{
				addr,
			},
		},
	}
	op := &MintOperation{
		MintInput: Input{
			SigIndices: []uint32{0},
		},
		MintOutput: MintOutput{
			OutputOwners: OutputOwners{
				Threshold: 1,
				Addrs: []ids.ShortID{
					addr,
				},
			},
		},
		TransferOutput: TransferOutput{
			Amt: 1,
			OutputOwners: OutputOwners{
				Locktime:  0,
				Threshold: 1,
				Addrs: []ids.ShortID{
					addr,
				},
			},
		},
	}

	utxos := []interface{}{utxo}
	require.ErrorIs(fx.VerifyOperation(tx, op, nil, utxos), errWrongCredentialType)
}

func TestFxVerifyOperationWrongNumberOfUTXOs(t *testing.T) {
	require := require.New(t)
	vm := TestVM{
		Codec: linearcodec.NewDefault(),
		Log:   logging.NoLog{},
	}
	date := time.Date(2019, time.January, 19, 16, 25, 17, 3, time.UTC)
	vm.CLK.Set(date)
	fx := Fx{}
	require.NoError(fx.Initialize(&vm))
	tx := &TestTx{UnsignedBytes: txBytes}
	utxo := &MintOutput{
		OutputOwners: OutputOwners{
			Threshold: 1,
			Addrs: []ids.ShortID{
				addr,
			},
		},
	}
	op := &MintOperation{
		MintInput: Input{
			SigIndices: []uint32{0},
		},
		MintOutput: MintOutput{
			OutputOwners: OutputOwners{
				Threshold: 1,
				Addrs: []ids.ShortID{
					addr,
				},
			},
		},
		TransferOutput: TransferOutput{
			Amt: 1,
			OutputOwners: OutputOwners{
				Locktime:  0,
				Threshold: 1,
				Addrs: []ids.ShortID{
					addr,
				},
			},
		},
	}
	cred := &Credential{
		Sigs: [][crypto.SECP256K1RSigLen]byte{
			sigBytes,
		},
	}

	utxos := []interface{}{utxo, utxo}
	require.ErrorIs(fx.VerifyOperation(tx, op, cred, utxos), errWrongNumberOfUTXOs)
}

func TestFxVerifyOperationUnknownUTXOType(t *testing.T) {
	require := require.New(t)
	vm := TestVM{
		Codec: linearcodec.NewDefault(),
		Log:   logging.NoLog{},
	}
	date := time.Date(2019, time.January, 19, 16, 25, 17, 3, time.UTC)
	vm.CLK.Set(date)
	fx := Fx{}
	require.NoError(fx.Initialize(&vm))
	tx := &TestTx{UnsignedBytes: txBytes}
	op := &MintOperation{
		MintInput: Input{
			SigIndices: []uint32{0},
		},
		MintOutput: MintOutput{
			OutputOwners: OutputOwners{
				Threshold: 1,
				Addrs: []ids.ShortID{
					addr,
				},
			},
		},
		TransferOutput: TransferOutput{
			Amt: 1,
			OutputOwners: OutputOwners{
				Locktime:  0,
				Threshold: 1,
				Addrs: []ids.ShortID{
					addr,
				},
			},
		},
	}
	cred := &Credential{
		Sigs: [][crypto.SECP256K1RSigLen]byte{
			sigBytes,
		},
	}

	utxos := []interface{}{nil}
	require.ErrorIs(fx.VerifyOperation(tx, op, cred, utxos), errWrongUTXOType)
}

func TestFxVerifyOperationInvalidOperationVerify(t *testing.T) {
	require := require.New(t)
	vm := TestVM{
		Codec: linearcodec.NewDefault(),
		Log:   logging.NoLog{},
	}
	date := time.Date(2019, time.January, 19, 16, 25, 17, 3, time.UTC)
	vm.CLK.Set(date)
	fx := Fx{}
	require.NoError(fx.Initialize(&vm))
	tx := &TestTx{UnsignedBytes: txBytes}
	utxo := &MintOutput{
		OutputOwners: OutputOwners{
			Threshold: 1,
			Addrs: []ids.ShortID{
				addr,
			},
		},
	}
	op := &MintOperation{
		MintInput: Input{
			SigIndices: []uint32{0},
		},
		MintOutput: MintOutput{
			OutputOwners: OutputOwners{
				Threshold: 1,
				Addrs: []ids.ShortID{
					addr,
				},
			},
		},
		TransferOutput: TransferOutput{
			Amt: 1,
			OutputOwners: OutputOwners{
				Locktime:  0,
				Threshold: 1,
			},
		},
	}
	cred := &Credential{
		Sigs: [][crypto.SECP256K1RSigLen]byte{
			sigBytes,
		},
	}

	utxos := []interface{}{utxo}
	require.ErrorIs(fx.VerifyOperation(tx, op, cred, utxos), errOutputUnspendable)
}

func TestFxVerifyOperationMismatchedMintOutputs(t *testing.T) {
	require := require.New(t)
	vm := TestVM{
		Codec: linearcodec.NewDefault(),
		Log:   logging.NoLog{},
	}
	date := time.Date(2019, time.January, 19, 16, 25, 17, 3, time.UTC)
	vm.CLK.Set(date)
	fx := Fx{}
	require.NoError(fx.Initialize(&vm))
	tx := &TestTx{UnsignedBytes: txBytes}
	utxo := &MintOutput{
		OutputOwners: OutputOwners{
			Threshold: 1,
			Addrs: []ids.ShortID{
				addr,
			},
		},
	}
	op := &MintOperation{
		MintInput: Input{
			SigIndices: []uint32{0},
		},
		MintOutput: MintOutput{
			OutputOwners: OutputOwners{},
		},
		TransferOutput: TransferOutput{
			Amt: 1,
			OutputOwners: OutputOwners{
				Locktime:  0,
				Threshold: 1,
				Addrs: []ids.ShortID{
					addr,
				},
			},
		},
	}
	cred := &Credential{
		Sigs: [][crypto.SECP256K1RSigLen]byte{
			sigBytes,
		},
	}

	utxos := []interface{}{utxo}
	require.ErrorIs(fx.VerifyOperation(tx, op, cred, utxos), errWrongMintCreated)
}

func TestVerifyPermission(t *testing.T) {
	require := require.New(t)
	vm := TestVM{
		Codec: linearcodec.NewDefault(),
		Log:   logging.NoLog{},
	}
	fx := Fx{}
	require.NoError(fx.Initialize(&vm))
	require.NoError(fx.Bootstrapping())
	require.NoError(fx.Bootstrapped())

	type test struct {
		description string
		tx          UnsignedTx
		in          *Input
		cred        *Credential
		cg          *OutputOwners
		shouldErr   bool
	}
	tests := []test{
		{
			"threshold 0, no sigs, has addrs",
			&TestTx{UnsignedBytes: txBytes},
			&Input{SigIndices: []uint32{}},
			&Credential{Sigs: [][crypto.SECP256K1RSigLen]byte{}},
			&OutputOwners{
				Threshold: 0,
				Addrs:     []ids.ShortID{addr},
			},
			true,
		},
		{
			"threshold 0, no sigs, no addrs",
			&TestTx{UnsignedBytes: txBytes},
			&Input{SigIndices: []uint32{}},
			&Credential{Sigs: [][crypto.SECP256K1RSigLen]byte{}},
			&OutputOwners{
				Threshold: 0,
				Addrs:     []ids.ShortID{},
			},
			false,
		},
		{
			"threshold 1, 1 sig",
			&TestTx{UnsignedBytes: txBytes},
			&Input{SigIndices: []uint32{0}},
			&Credential{Sigs: [][crypto.SECP256K1RSigLen]byte{sigBytes}},
			&OutputOwners{
				Threshold: 1,
				Addrs:     []ids.ShortID{addr},
			},
			false,
		},
		{
			"threshold 0, 1 sig (too many sigs)",
			&TestTx{UnsignedBytes: txBytes},
			&Input{SigIndices: []uint32{0}},
			&Credential{Sigs: [][crypto.SECP256K1RSigLen]byte{sigBytes}},
			&OutputOwners{
				Threshold: 0,
				Addrs:     []ids.ShortID{addr},
			},
			true,
		},
		{
			"threshold 1, 0 sigs (too few sigs)",
			&TestTx{UnsignedBytes: txBytes},
			&Input{SigIndices: []uint32{}},
			&Credential{Sigs: [][crypto.SECP256K1RSigLen]byte{}},
			&OutputOwners{
				Threshold: 1,
				Addrs:     []ids.ShortID{addr},
			},
			true,
		},
		{
			"threshold 1, 1 incorrect sig",
			&TestTx{UnsignedBytes: txBytes},
			&Input{SigIndices: []uint32{0}},
			&Credential{Sigs: [][crypto.SECP256K1RSigLen]byte{sigBytes}},
			&OutputOwners{
				Threshold: 1,
				Addrs:     []ids.ShortID{ids.GenerateTestShortID()},
			},
			true,
		},
		{
			"repeated sig",
			&TestTx{UnsignedBytes: txBytes},
			&Input{SigIndices: []uint32{0, 0}},
			&Credential{Sigs: [][crypto.SECP256K1RSigLen]byte{sigBytes, sigBytes}},
			&OutputOwners{
				Threshold: 2,
				Addrs:     []ids.ShortID{addr, addr2},
			},
			true,
		},
		{
			"threshold 2, repeated address and repeated sig",
			&TestTx{UnsignedBytes: txBytes},
			&Input{SigIndices: []uint32{0, 1}},
			&Credential{Sigs: [][crypto.SECP256K1RSigLen]byte{sigBytes, sigBytes}},
			&OutputOwners{
				Threshold: 2,
				Addrs:     []ids.ShortID{addr, addr},
			},
			true,
		},
		{
			"threshold 2, 2 sigs",
			&TestTx{UnsignedBytes: txBytes},
			&Input{SigIndices: []uint32{0, 1}},
			&Credential{Sigs: [][crypto.SECP256K1RSigLen]byte{sigBytes, sig2Bytes}},
			&OutputOwners{
				Threshold: 2,
				Addrs:     []ids.ShortID{addr, addr2},
			},
			false,
		},
		{
			"threshold 2, 2 sigs reversed (should be sorted)",
			&TestTx{UnsignedBytes: txBytes},
			&Input{SigIndices: []uint32{1, 0}},
			&Credential{Sigs: [][crypto.SECP256K1RSigLen]byte{sig2Bytes, sigBytes}},
			&OutputOwners{
				Threshold: 2,
				Addrs:     []ids.ShortID{addr, addr2},
			},
			true,
		},
		{
			"threshold 1, 1 sig, index out of bounds",
			&TestTx{UnsignedBytes: txBytes},
			&Input{SigIndices: []uint32{1}},
			&Credential{Sigs: [][crypto.SECP256K1RSigLen]byte{sigBytes}},
			&OutputOwners{
				Threshold: 1,
				Addrs:     []ids.ShortID{addr},
			},
			true,
		},
	}

	for _, test := range tests {
		err := fx.VerifyPermission(test.tx, test.in, test.cred, test.cg)
		if test.shouldErr {
			require.Errorf(err, "test '%s' should have errored but didn't", test.description)
		} else {
			require.NoErrorf(err, "test '%s' errored but it shouldn't have", test.description)
		}
	}
}
