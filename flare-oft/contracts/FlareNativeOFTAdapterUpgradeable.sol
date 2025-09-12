// SPDX-License-Identifier: MIT
pragma solidity ^0.8.22;

import { NativeOFTAdapterUpgradeable } from "@layerzerolabs/oft-evm-upgradeable/contracts/oft/NativeOFTAdapterUpgradeable.sol";
/**
 * @title NativeOFTAdapterUpgradeable Contract
 * @dev NativeOFTAdapterUpgradeable is a contract that adapts native currency to the OFT functionality.
 *
 * @dev WARNING: ONLY 1 of these should exist for a given global mesh,
 * unless you make a NON-default implementation of OFT, which needs to be done very carefully.
 * @dev WARNING: The default NativeOFTAdapterUpgradeable implementation assumes LOSSLESS transfers, ie. 1 native in, 1 native out.
 */
contract FlareNativeOFTAdapterUpgradeable is NativeOFTAdapterUpgradeable {
    constructor(
        uint8 _localDecimals,
        address _lzEndpoint
    ) NativeOFTAdapterUpgradeable(_localDecimals, _lzEndpoint) {
        _disableInitializers();
    }

    function initialize(address _delegate) public initializer {
        __NativeOFTAdapter_init(_delegate);
        __Ownable_init(_delegate);
    }
}