import { evm, pvm, UnsignedTx, utils } from '@flarenetwork/flarejs';
import { issueCChainTx, issuePChainTx, localFlareContext } from './utils';
import { runTest } from './runner';

async function CtoPExport(amountFLR: number) {
    const ctx = await localFlareContext();
    const fee = 1; // in FLR
    const txCount = await ctx.provider.getTransactionCount(ctx.addressC);

    // Create and issue a C to P export transaction
    console.log(`Creating C to P export transaction for ${amountFLR} FLR...`);

    const exportTx = evm.newExportTx(
        ctx.context,
        BigInt(amountFLR * 1e9),
        ctx.context.pBlockchainID,
        utils.hexToBuffer(ctx.addressC),
        [utils.bech32ToBytes(ctx.addressP)],
        BigInt(fee * 1e9),
        BigInt(txCount),
    )

    await issueCChainTx(ctx.evmapi, exportTx, ctx.privateKey);

    // Create and issue a C to P chain import transaction
    console.log('\nCreating C to P chain import transaction');

    const { utxos } = await ctx.pvmapi.getUTXOs({
        sourceChain: 'C',
        addresses: [ctx.addressP]
    });

    let importTx: UnsignedTx;
    if (ctx.isEtnaForkActive) {
        console.log("Etna fork is active, using new transaction format for import.");
        const feeState = await ctx.pvmapi.getFeeState();
        importTx = pvm.e.newImportTx(
            {
                feeState,
                utxos,
                sourceChainId: ctx.context.cBlockchainID,
                fromAddressesBytes: [utils.bech32ToBytes(ctx.addressP)],
                toAddressesBytes: [utils.bech32ToBytes(ctx.addressP)],
            },
            ctx.context,
        );
    } else {
        console.log("Etna fork is not active, using legacy transaction format for import.");
        importTx = pvm.newImportTx(
            ctx.context,
            ctx.context.cBlockchainID,
            utxos,
            [utils.bech32ToBytes(ctx.addressP)],
            [utils.bech32ToBytes(ctx.addressP)]
        );
    }
    await issuePChainTx(ctx.pvmapi, importTx, ctx.privateKey);
}

runTest(() => CtoPExport(100))
