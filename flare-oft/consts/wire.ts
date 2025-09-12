import { EndpointId } from '@layerzerolabs/lz-definitions'
import { ExecutorOptionType } from '@layerzerolabs/lz-v2-utilities'
import { OAppEnforcedOption } from '@layerzerolabs/toolbox-hardhat'

export const DVNS = {
    LZ_LABS: {
        [EndpointId.FLARE_V2_MAINNET]: '0x9c061c9a4782294eef65ef28cb88233a987f4bdd',
        [EndpointId.HYPERLIQUID_V2_MAINNET]: '0xc097ab8cd7b053326dfe9fb3e3a31a0cce3b526f',
    } as Partial<Record<EndpointId, string>>,
    NETHERMIND: {
        [EndpointId.FLARE_V2_MAINNET]: '0x9bcd17a654bffaa6f8fea38d19661a7210e22196',
        [EndpointId.HYPERLIQUID_V2_MAINNET]: '0x8e49ef1dfae17e547ca0e7526ffda81fbaca810a',
    } as Partial<Record<EndpointId, string>>,
}

// Define enforced options per specific endpoint ID
export const ENFORCED_OPTIONS: Partial<Record<EndpointId, OAppEnforcedOption[]>> = {
    [EndpointId.FLARE_V2_MAINNET]: [{ msgType: 1, optionType: ExecutorOptionType.LZ_RECEIVE, gas: 100000, value: 0 }],
    [EndpointId.HYPERLIQUID_V2_MAINNET]: [
        { msgType: 1, optionType: ExecutorOptionType.LZ_RECEIVE, gas: 100000, value: 0 },
    ],
}

export const MULTISIGS: Partial<Record<EndpointId, string>> = {
    [EndpointId.FLARE_V2_MAINNET]: '0x78547dDA5C18db47bb8Ecd1d3c368955b513F355',
    [EndpointId.HYPERLIQUID_V2_MAINNET]: '0x0144603c28313F30fa573448406721792E46cfd8',
} as const

export const getRequiredDVNs = (eid: EndpointId): string[] => {
    return [DVNS.LZ_LABS[eid], DVNS.NETHERMIND[eid]].filter(Boolean) as string[]
}

export const getEnforcedOptions = (eid: EndpointId): OAppEnforcedOption[] => {
    return ENFORCED_OPTIONS[eid] ?? [{ msgType: 1, optionType: ExecutorOptionType.LZ_RECEIVE, gas: 80000, value: 0 }]
}

export const getMultisigAddress = (eid: EndpointId): string => {
    const address = MULTISIGS[eid]

    if (!address || address === 'TODO' || address === '0x0000000000000000000000000000000000000000') {
        throw new Error(
            `Multisig address not configured for endpoint ${eid}. Please update MULTISIGS in consts/wire.ts`
        )
    }

    return address
}
