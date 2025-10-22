import { evm, pvm, utils, TransferableOutput, UnsignedTx } from '@flarenetwork/flarejs';
import { issueCChainTx, issuePChainTx, localFlareContext } from './utils';
import { runTest } from './runner';

async function PtoCExport(amountFLR: number) {
    const ctx = await localFlareContext();
    const fee = 1; // in FLR

    // Create and issue a P to C export transaction
    console.log(`Creating P to C export transaction for ${amountFLR} FLR...`);

    const { utxos: utxosp } = await ctx.pvmapi.getUTXOs({
        addresses: [ctx.addressP]
    });

    let exportTx: UnsignedTx;
    if (ctx.isEtnaForkActive) {
        console.log("Etna fork is active, using new transaction format for export.");
        const feeState = await ctx.pvmapi.getFeeState();
        exportTx = pvm.e.newExportTx(
            {
                feeState,
                utxos: utxosp,
                destinationChainId: ctx.context.cBlockchainID,
                fromAddressesBytes: [utils.bech32ToBytes(ctx.addressP)],
                outputs: [
                    TransferableOutput.fromNative(
                        ctx.context.avaxAssetID,
                        BigInt(amountFLR * 1e9),
                        [utils.bech32ToBytes(ctx.addressP)]
                    )
                ]
            },
            ctx.context,
        );
    } else {
        console.log("Etna fork is not active, using legacy transaction format for export.");
        exportTx = pvm.newExportTx(
            ctx.context,
            ctx.context.cBlockchainID,
            [utils.bech32ToBytes(ctx.addressP)],
            utxosp,
            [
                TransferableOutput.fromNative(
                    ctx.context.avaxAssetID,
                    BigInt(amountFLR * 1e9),
                    [utils.bech32ToBytes(ctx.addressP)]
                )
            ]
        );
    }
    await issuePChainTx(ctx.pvmapi, exportTx, ctx.privateKey);

    // Create and issue a P to C chain import transaction
    console.log('\nCreating P to C chain import transaction');

    const { utxos: utxosc } = await ctx.evmapi.getUTXOs({
        sourceChain: 'P',
        addresses: ['C-' + ctx.addressP.slice(2)],
    });

    const importTx = evm.newImportTx(
        ctx.context,
        utils.hexToBuffer(ctx.addressC),
        [utils.bech32ToBytes(ctx.addressP)],
        utxosc,
        ctx.context.pBlockchainID,
        BigInt(fee * 1e9),
    );

    await issueCChainTx(ctx.evmapi, importTx, ctx.privateKey);
}


runTest(() => PtoCExport(100))

