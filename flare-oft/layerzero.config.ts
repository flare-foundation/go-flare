import { EndpointId } from '@layerzerolabs/lz-definitions'
import { OmniPointHardhat } from '@layerzerolabs/toolbox-hardhat'

import { getEnforcedOptions, getMultisigAddress, getRequiredDVNs } from './consts/wire'

// Define all contracts
const CONTRACTS: OmniPointHardhat[] = [
    { eid: EndpointId.FLARE_V2_MAINNET, contractName: 'FlareNativeOFTAdapterUpgradeable_Proxy' },
    { eid: EndpointId.HYPERLIQUID_V2_MAINNET, contractName: 'FlareOFTFeeUpgradeable_Proxy' },
]

// Generate all possible connections
const generateConnections = async () => {
    const connections = []

    // Generate all directional pairs first (including both directions)
    const pairs = []
    for (let i = 0; i < CONTRACTS.length; i++) {
        for (let j = 0; j < CONTRACTS.length; j++) {
            if (i !== j) {
                // Skip self-connections
                pairs.push([CONTRACTS[i], CONTRACTS[j]]) // from -> to
            }
        }
    }

    // Iterate through all directional pairs
    for (const [from, to] of pairs) {
        connections.push({
            from,
            to,
            config: {
                enforcedOptions: getEnforcedOptions(to.eid),
                sendConfig: {
                    ulnConfig: {
                        requiredDVNs: getRequiredDVNs(from.eid),
                    },
                },
                receiveConfig: {
                    ulnConfig: {
                        requiredDVNs: getRequiredDVNs(from.eid),
                    },
                },
            },
        })
    }

    return connections
}

export default async function () {
    const connections = await generateConnections()

    return {
        contracts: CONTRACTS.map((contract) => ({
            contract,
            config: {
                owner: getMultisigAddress(contract.eid),
                delegate: getMultisigAddress(contract.eid),
            },
        })),
        connections,
    }
}
