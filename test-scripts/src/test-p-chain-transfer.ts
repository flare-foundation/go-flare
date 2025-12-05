import { pvm, utils, TransferableOutput } from '@flarenetwork/flarejs';
import { issuePChainTx, localFlareContext } from './utils';
import { runTest } from './runner';

async function PtoPTransfer(amountFLR: number, toAddress: string) {
    const ctx = await localFlareContext();

    // Create and issue a P chain transfer transaction (BaseTx)
    console.log(`Creating P chain transfer transaction for ${amountFLR} FLR to address ${toAddress}...`);

    const { utxos: utxosp } = await ctx.pvmapi.getUTXOs({
        addresses: [ctx.addressP]
    });
    const feeState = await ctx.pvmapi.getFeeState();

    const tx = pvm.e.newBaseTx(
        {
            feeState,
            utxos: utxosp,
            fromAddressesBytes: [utils.bech32ToBytes(ctx.addressP)],
            outputs: [
                TransferableOutput.fromNative(
                    ctx.context.avaxAssetID,
                    BigInt(amountFLR * 1e9),
                    [utils.bech32ToBytes(toAddress)]
                )
            ]
        },
        ctx.context,
    );
    await issuePChainTx(ctx.pvmapi, tx, ctx.privateKey);
}


runTest(() => PtoPTransfer(100, "P-localflare1zjaa3yjnzn5cjx9r56x59raam2jgwnmztlg995"))

