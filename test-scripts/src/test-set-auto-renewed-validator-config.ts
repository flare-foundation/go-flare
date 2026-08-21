import { pvm, secp256k1, utils } from "@flarenetwork/flarejs";
import { issuePChainTx, localFlareContext } from "./utils";
import { runTest } from "./runner";

// Node ID of the auto-renewed validator added by test-add-auto-renewed-validator.ts
const nodeID = 'NodeID-GWPcbFJZFfZreETSoWjPimr846mXEKCtu';

async function setAutoRenewedValidatorConfig(nodeId: string, period: number) {
    const ctx = await localFlareContext();

    // The config tx references the validator by the ID of the transaction that
    // created it: take it from the command line or look it up by node ID.
    let validatorTxId = process.argv[2];
    if (!validatorTxId) {
        const { validators } = await ctx.pvmapi.getCurrentValidators({ nodeIDs: [nodeId] });
        if (validators.length === 0) {
            throw new Error(`No current validator with node ID ${nodeId}; run add-auto-renewed-validator first or pass the validator txID as an argument`);
        }
        validatorTxId = validators[0].txID;
    }

    // Create the transaction to update the auto-renew config of the validator
    console.log(`Creating set auto-renewed validator config transaction for validator ${validatorTxId} with period ${period}s...`);

    const { utxos } = await ctx.pvmapi.getUTXOs({
        addresses: [ctx.addressP]
    });
    const feeState = await ctx.pvmapi.getFeeState();

    const tx = pvm.newSetAutoRenewedValidatorConfigTx(
        {
            feeState,
            utxos,
            fromAddressesBytes: [utils.bech32ToBytes(ctx.addressP)],
            validatorTxId,
            // Indices into the validator authority owner defined by the
            // AddAutoRenewedValidatorTx (a single address with threshold 1).
            auth: [0],
            autoCompoundRewardShares: 0,
            // A period of 0 stops auto-renewal: the validator is removed and its
            // stake returned at the end of the current validation cycle.
            period: BigInt(period),
        },
        ctx.context,
    );

    // The builder's address maps only cover the base-tx inputs, so addTxSignatures
    // (called in issuePChainTx) does not fill the validator-authority credential,
    // which is appended after the input credentials. Sign it explicitly; the
    // signature is over the unsigned tx bytes, so it can be added first.
    const authSig = await secp256k1.sign(tx.toBytes(), utils.hexToBuffer(ctx.privateKey));
    tx.addSignatureAt(authSig, tx.getCredentials().length - 1, 0);

    await issuePChainTx(ctx.pvmapi, tx, ctx.privateKey);

    // The auto-renew config fields are flattened into the validator object
    // (not part of the flarejs response type yet, hence the cast)
    const { validators } = await ctx.pvmapi.getCurrentValidators({ nodeIDs: [nodeId] });
    const validator = validators[0] as any;
    console.log(`Validator config updated, next period: ${validator?.nextPeriod}s, auto compound reward shares: ${validator?.autoCompoundRewardShares}`);
}

runTest(() => setAutoRenewedValidatorConfig(
    nodeID,
    100 // stop auto-renewing at the end of the current validation cycle
))
