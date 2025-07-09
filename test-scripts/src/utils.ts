import { evm, pvm, Context, addTxSignatures, UnsignedTx, utils} from '@flarenetwork/flarejs';
import { JsonRpcProvider } from 'ethers'

export const LocalURL = 'http://localhost:9650';
export const TestCAddress = '0x8db97C7cEcE249c2b98bDC0226Cc4C2A57BF52FC';
export const TestPAddress = 'P-localflare18jma8ppw3nhx5r4ap8clazz0dps7rv5uj3gy4v';
export const TestPrivateKey = '0x56289e99c94b6912bfc12adc093c9b51124f0dc54ac7a766b2bc5ccf558d8027';

export interface TestContext {
    context: Context.Context;
    evmapi: evm.EVMApi;
    pvmapi: pvm.PVMApi;
    provider: JsonRpcProvider;
    addressC: string;
    addressP: string;
    privateKey: string;
}

export async function CChainBalance(): Promise<BigInt> {
    const context = await localFlareContext();
    return context.provider.getBalance(TestCAddress);
}

export async function PChainBalance(): Promise<BigInt> {
    const context = await localFlareContext();
    const { balance } = await context.pvmapi.getBalance({
        addresses: [TestPAddress],
    });
    return balance
}

export async function localFlareContext(): Promise<TestContext> {
    const evmapi = new evm.EVMApi(LocalURL);
    const pvmapi = new pvm.PVMApi(LocalURL);
    const context = await Context.getContextFromURI(LocalURL);
    const provider = new JsonRpcProvider(LocalURL + '/ext/bc/C/rpc');
    return {
        context,
        evmapi,
        pvmapi,
        provider,
        addressC: TestCAddress,
        addressP: TestPAddress,
        privateKey: TestPrivateKey,
    }
}

export async function delay(ms: number): Promise<void> {
    return new Promise(resolve => setTimeout(resolve, ms));
}

export function formatDecimal(int: BigInt, decimals: number): string {
    if (int === 0n) {
        return '0'
    }
    let strInt = int.toString()
    strInt = strInt.padStart(decimals, '0')
    const decPart = strInt.slice(0, -decimals) || '0'
    const fracPart = strInt.slice(-decimals).replace(/0+$/, '')
    return fracPart === '' ? decPart : decPart + '.' + fracPart
}

export async function issuePChainTx(pvmapi: pvm.PVMApi, tx: UnsignedTx, privateKey: string): Promise<void> {
    await addTxSignatures({
        unsignedTx: tx,
        privateKeys: [utils.hexToBuffer(privateKey)],
    });

    const exportResponse = await pvmapi.issueSignedTx(tx.getSignedTx());
    const txID = exportResponse.txID;
    console.log(`Issued transaction with ID: ${txID}`);    

    console.log(`Waiting for transaction ${txID} to be processed...`);
    let txStatus;
    do {
        await delay(1000);
        txStatus = await pvmapi.getTxStatus({ txID });
    } while (txStatus.status === 'Processing');

    if (txStatus.status === 'Committed') {
        console.log(`Transaction ${txID} accepted`);
    } else {
        throw new Error(`Transaction ${txID} failed with status: ${txStatus.status}`);
    }
}

export async function issueCChainTx(evmapi: evm.EVMApi, tx: UnsignedTx, privateKey: string): Promise<void> {
    await addTxSignatures({
        unsignedTx: tx,
        privateKeys: [utils.hexToBuffer(privateKey)],
    });    

    const exportResponse = await evmapi.issueSignedTx(tx.getSignedTx());
    const txID = exportResponse.txID;
    console.log(`Issued transaction with ID: ${txID}`);    

    console.log(`Waiting for transaction ${txID} to be processed...`);
    let txStatus;
    do {
        await delay(1000);
        txStatus = await evmapi.getAtomicTxStatus(txID);
    } while (txStatus.status !== 'Accepted' && txStatus.status !== 'Rejected');

    if (txStatus.status === 'Accepted') {
        console.log(`Transaction ${txID} accepted`);
    } else {
        throw new Error(`Transaction ${txID} failed with status: ${txStatus.status}`);
    }
}