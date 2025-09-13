import { type DeployFunction } from 'hardhat-deploy/types'

import { EndpointId, endpointIdToNetwork } from '@layerzerolabs/lz-definitions'
import { getDeploymentAddressAndAbi } from '@layerzerolabs/lz-evm-sdk-v2'

const contractName = 'FlareOFTFeeUpgradeable'

const deploy: DeployFunction = async (hre) => {
    const { deploy } = hre.deployments
    const signer = (await hre.ethers.getSigners())[0]
    console.log(`deploying ${contractName} on network: ${hre.network.name} with ${signer.address}`)

    const eid = hre.network.config.eid as EndpointId
    const lzNetworkName = endpointIdToNetwork(eid)

    const { address } = getDeploymentAddressAndAbi(lzNetworkName, 'EndpointV2')

    // First deploy the ProxyAdmin contract
    const proxyAdminName = `${contractName}_ProxyAdmin`
    await deploy(proxyAdminName, {
        from: signer.address,
        contract: 'ProxyAdmin',
        args: [signer.address],
        log: true,
        waitConfirmations: 1,
        skipIfAlreadyDeployed: false,
    })

    // Then deploy the main contract using the ProxyAdmin
    await deploy(contractName, {
        from: signer.address,
        args: [address],
        log: true,
        waitConfirmations: 1,
        skipIfAlreadyDeployed: false,
        proxy: {
            proxyContract: 'OpenZeppelinTransparentProxy',
            viaAdminContract: {
                name: proxyAdminName,
            },
            execute: {
                init: {
                    methodName: 'initialize',
                    args: ['Flare', 'FLR', signer.address],
                },
            },
        },
    })
}

deploy.tags = [contractName]

export default deploy
