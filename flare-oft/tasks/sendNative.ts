import path from 'path'

import { BigNumber, Contract, ContractTransaction } from 'ethers'
import { parseUnits } from 'ethers/lib/utils'
import { HardhatRuntimeEnvironment } from 'hardhat/types'

import { OmniPointHardhat, createGetHreByEid } from '@layerzerolabs/devtools-evm-hardhat'
import { createLogger, promptToContinue } from '@layerzerolabs/io-devtools'
import { ChainType, endpointIdToChainType, endpointIdToNetwork } from '@layerzerolabs/lz-definitions'
import { Options, addressToBytes32 } from '@layerzerolabs/lz-v2-utilities'

import { SendResult } from './types'
import { DebugLogger, KnownErrors, MSG_TYPE, getLayerZeroScanLink, isEmptyOptionsEvm } from './utils'

const logger = createLogger()

/**
 * Get OApp contract address by EID from LayerZero config
 */
async function getOAppAddressByEid(
    eid: number,
    oappConfig: string,
    hre: HardhatRuntimeEnvironment,
    overrideAddress?: string
): Promise<string> {
    if (overrideAddress) {
        return overrideAddress
    }

    const layerZeroConfig = (await import(path.resolve('./', oappConfig))).default
    const { contracts } = typeof layerZeroConfig === 'function' ? await layerZeroConfig() : layerZeroConfig
    const wrapper = contracts.find((c: { contract: OmniPointHardhat }) => c.contract.eid === eid)
    if (!wrapper) throw new Error(`No config for EID ${eid}`)

    return wrapper.contract.contractName
        ? (await hre.deployments.get(wrapper.contract.contractName)).address
        : wrapper.contract.address || ''
}

export interface NativeArgs {
    srcEid: number
    dstEid: number
    amount: string
    to: string
    oappConfig: string
    minAmount?: string
    extraLzReceiveOptions?: string[]
    extraLzComposeOptions?: string[]
    extraNativeDropOptions?: string[]
    composeMsg?: string
    nativeAdapterAddress?: string
}

export async function sendNative(
    {
        srcEid,
        dstEid,
        amount,
        to,
        oappConfig,
        minAmount,
        extraLzReceiveOptions,
        extraLzComposeOptions,
        extraNativeDropOptions,
        composeMsg,
        nativeAdapterAddress,
    }: NativeArgs,
    hre: HardhatRuntimeEnvironment
): Promise<SendResult> {
    if (endpointIdToChainType(srcEid) !== ChainType.EVM) {
        throw new Error(`non-EVM srcEid (${srcEid}) not supported here`)
    }

    const getHreByEid = createGetHreByEid(hre)
    let srcEidHre: HardhatRuntimeEnvironment
    try {
        srcEidHre = await getHreByEid(srcEid)
    } catch (error) {
        DebugLogger.printErrorAndFixSuggestion(
            KnownErrors.ERROR_GETTING_HRE,
            `For network: ${endpointIdToNetwork(srcEid)}, Native Adapter: ${nativeAdapterAddress}`
        )
        throw error
    }
    const signer = (await srcEidHre.ethers.getSigners())[0]

    // 1️⃣ resolve the Native OFT Adapter address
    const adapterAddress = await getOAppAddressByEid(srcEid, oappConfig, srcEidHre, nativeAdapterAddress)

    // 2️⃣ load NativeOFTAdapterUpgradeable ABI
    const adapterArtifact = await srcEidHre.artifacts.readArtifact('FlareNativeOFTAdapterUpgradeable')

    // now attach
    const nativeAdapter = await srcEidHre.ethers.getContractAt(adapterArtifact.abi, adapterAddress, signer)

    // 🔗 Get LayerZero endpoint contract
    const endpointDep = await srcEidHre.deployments.get('EndpointV2')
    const _endpointContract = new Contract(endpointDep.address, endpointDep.abi, signer)

    // Get destination OApp address for outboundNonce call
    const dstEidHre = await getHreByEid(dstEid)
    const dstAdapterAddress = await getOAppAddressByEid(dstEid, oappConfig, dstEidHre, nativeAdapterAddress)
    const dstAdapterBytes32 = addressToBytes32(dstAdapterAddress)

    // 3️⃣ verify this is a native adapter
    const tokenAddress = await nativeAdapter.token()
    if (tokenAddress !== '0x0000000000000000000000000000000000000000') {
        throw new Error(`Contract at ${adapterAddress} is not a Native OFT Adapter (token() returns ${tokenAddress})`)
    }

    // 4️⃣ verify approval is not required
    const approvalRequired = await nativeAdapter.approvalRequired()
    if (approvalRequired) {
        throw new Error(`Contract at ${adapterAddress} requires approval, which is unexpected for Native OFT Adapters`)
    }

    logger.info('✅ Confirmed Native OFT Adapter - no ERC20 approvals required')

    // 5️⃣ Get native token decimals (typically 18 for most chains)
    const decimals = 18 // Native tokens typically have 18 decimals

    // 6️⃣ normalize the user-supplied amount
    const amountUnits: BigNumber = parseUnits(amount, decimals)

    // 7️⃣ hex string → Uint8Array → zero-pad to 32 bytes
    const toBytes = addressToBytes32(to)

    // 8️⃣ Build options dynamically using Options.newOptions()
    let options = Options.newOptions()

    // Add lzReceive options
    if (extraLzReceiveOptions && extraLzReceiveOptions.length > 0) {
        if (extraLzReceiveOptions.length % 2 !== 0) {
            throw new Error(
                `Invalid lzReceive options: received ${extraLzReceiveOptions.length} values, but expected pairs of gas,value`
            )
        }

        for (let i = 0; i < extraLzReceiveOptions.length; i += 2) {
            const gas = Number(extraLzReceiveOptions[i])
            const value = Number(extraLzReceiveOptions[i + 1]) || 0
            options = options.addExecutorLzReceiveOption(gas, value)
            logger.info(`Added lzReceive option: ${gas} gas, ${value} value`)
        }
    }

    // Add lzCompose options
    if (extraLzComposeOptions && extraLzComposeOptions.length > 0) {
        if (extraLzComposeOptions.length % 3 !== 0) {
            throw new Error(
                `Invalid lzCompose options: received ${extraLzComposeOptions.length} values, but expected triplets of index,gas,value`
            )
        }

        for (let i = 0; i < extraLzComposeOptions.length; i += 3) {
            const index = Number(extraLzComposeOptions[i])
            const gas = Number(extraLzComposeOptions[i + 1])
            const value = Number(extraLzComposeOptions[i + 2]) || 0
            options = options.addExecutorComposeOption(index, gas, value)
            logger.info(`Added lzCompose option: index ${index}, ${gas} gas, ${value} value`)
        }
    }

    // Add native drop options
    if (extraNativeDropOptions && extraNativeDropOptions.length > 0) {
        if (extraNativeDropOptions.length % 2 !== 0) {
            throw new Error(
                `Invalid native drop options: received ${extraNativeDropOptions.length} values, but expected pairs of amount,recipient`
            )
        }

        for (let i = 0; i < extraNativeDropOptions.length; i += 2) {
            const amountStr = extraNativeDropOptions[i]
            const recipient = extraNativeDropOptions[i + 1]

            if (!amountStr || !recipient) {
                throw new Error(
                    `Invalid native drop option: Both amount and recipient must be provided. Got amount="${amountStr}", recipient="${recipient}"`
                )
            }

            try {
                options = options.addExecutorNativeDropOption(amountStr.trim(), recipient.trim())
                logger.info(`Added native drop option: ${amountStr.trim()} wei to ${recipient.trim()}`)
            } catch (error) {
                const maxUint128 = BigInt('340282366920938463463374607431768211455') // 2^128 - 1
                const maxUint128Ether = Number(maxUint128) / 1e18

                throw new Error(
                    `Failed to add native drop option with amount ${amountStr.trim()} wei. ` +
                        `LayerZero protocol constrains native drop amounts to uint128 maximum ` +
                        `(${maxUint128.toString()} wei ≈ ${maxUint128Ether.toFixed(2)} ETH). ` +
                        `Original error: ${error instanceof Error ? error.message : String(error)}`
                )
            }
        }
    }
    const extraOptions = options.toHex()

    // Check whether there are extra options or enforced options
    if (isEmptyOptionsEvm(extraOptions)) {
        try {
            const enforcedOptions = composeMsg
                ? await nativeAdapter.enforcedOptions(dstEid, MSG_TYPE.SEND_AND_CALL)
                : await nativeAdapter.enforcedOptions(dstEid, MSG_TYPE.SEND)

            if (isEmptyOptionsEvm(enforcedOptions)) {
                const proceed = await promptToContinue(
                    'No extra options were included and Native Adapter has no set enforced options. Your quote / send will most likely fail. Continue?'
                )
                if (!proceed) {
                    throw new Error('Aborted due to missing options')
                }
            }
        } catch (error) {
            logger.debug(`Failed to check enforced options: ${error}`)
        }
    }

    // 9️⃣ build sendParam
    const sendParam = {
        dstEid,
        to: toBytes,
        amountLD: amountUnits.toString(),
        minAmountLD: minAmount ? parseUnits(minAmount, decimals).toString() : amountUnits.toString(),
        extraOptions: extraOptions,
        composeMsg: composeMsg ? composeMsg.toString() : '0x',
        oftCmd: '0x',
    }

    // 🔟 Quote the LayerZero messaging fee
    logger.info('Quoting the LayerZero messaging fee...')
    let msgFee: { nativeFee: BigNumber; lzTokenFee: BigNumber }
    try {
        msgFee = await nativeAdapter.quoteSend(sendParam, false)
    } catch (error) {
        DebugLogger.printErrorAndFixSuggestion(
            KnownErrors.ERROR_QUOTING_NATIVE_GAS_COST,
            `For network: ${endpointIdToNetwork(srcEid)}, Native Adapter: ${nativeAdapterAddress}`
        )
        throw error
    }

    // Calculate total msg.value needed: LayerZero fee + native amount to send
    const totalMsgValue = msgFee.nativeFee.add(amountUnits)

    logger.info(`💰 Amount to send: ${amount} native tokens (${amountUnits.toString()} wei)`)
    logger.info(`💸 LayerZero messaging fee: ${msgFee.nativeFee.toString()} wei`)
    logger.info(`💳 Total msg.value required: ${totalMsgValue.toString()} wei`)

    // Get the outbound nonce that will be used for this transaction (before sending)
    const outboundNonce = (await _endpointContract.outboundNonce(adapterAddress, dstEid, dstAdapterBytes32)).add(1)

    logger.info('🚀 Sending the native token transaction...')
    let tx: ContractTransaction
    try {
        tx = await nativeAdapter.send(sendParam, msgFee, signer.address, {
            value: totalMsgValue, // LayerZero fee + native amount to send
        })
    } catch (error) {
        DebugLogger.printErrorAndFixSuggestion(
            KnownErrors.ERROR_SENDING_TRANSACTION,
            `For network: ${endpointIdToNetwork(srcEid)}, Native Adapter: ${nativeAdapterAddress}`
        )
        throw error
    }
    const receipt = await tx.wait()

    const txHash = receipt.transactionHash
    const scanLink = getLayerZeroScanLink(txHash, srcEid >= 40_000 && srcEid < 50_000)

    logger.info(`✅ Transaction sent successfully!`)
    logger.info(`📄 Transaction hash: ${txHash}`)
    logger.info(`🔍 LayerZero scan: ${scanLink}`)

    return { txHash, scanLink, outboundNonce: outboundNonce.toString(), extraOptions }
}
