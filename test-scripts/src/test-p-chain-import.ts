import { evm, pvm, utils } from '@flarenetwork/flarejs';
import { issueCChainTx, issuePChainTx, localFlareContext } from './utils';
import { runTest } from './runner';

async function CtoPExport(amountFLR: number) {
    const ctx = await localFlareContext();
    const baseFee = await ctx.evmapi.getBaseFee();
    const txCount = await ctx.provider.getTransactionCount(ctx.addressC);

    // Create and issue a C to P export transaction
    console.log(`Creating C to P export transaction for ${amountFLR} FLR...`);

    const exportTx = evm.newExportTxFromBaseFee(
        ctx.context,
        baseFee / BigInt(1e9),
        BigInt(amountFLR * 1e9),
        ctx.context.pBlockchainID,
        utils.hexToBuffer(ctx.addressC),
        [utils.bech32ToBytes(ctx.addressP)],
        BigInt(txCount),
    );

    await issueCChainTx(ctx.evmapi, exportTx, ctx.privateKey);

    // Create and issue a C to P chain import transaction
    console.log('\nCreating C to P chain import transaction');

    const { utxos } = await ctx.pvmapi.getUTXOs({
        sourceChain: 'C',
        addresses: [ctx.addressP]
    });
    const importTx = pvm.newImportTx(
        ctx.context,
        ctx.context.cBlockchainID,
        utxos,
        [utils.bech32ToBytes(ctx.addressP)],
        [utils.bech32ToBytes(ctx.addressP)]
    );

    await issuePChainTx(ctx.pvmapi, importTx, ctx.privateKey);
}

runTest(() => CtoPExport(100))
