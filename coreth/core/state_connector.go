// (c) 2021, Flare Networks Limited. All rights reserved.
// Please see the file LICENSE for licensing terms.

package core

import (
	"encoding/hex"
	"fmt"
	"math/big"
	"os"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum/common"

	"github.com/ava-labs/coreth/core/vm"
	"github.com/ava-labs/coreth/params"
	"github.com/ava-labs/coreth/utils"
)

const (
	localAttestorEnv  = "SC_LOCAL_ATTESTATORS"
	forkingEnabledEnv = "SC_FORKING_ENABLED"
)

var (
	flareActivationTime      = uint64(time.Date(2022, time.June, 1, 0, 0, 0, 0, time.UTC).Unix())
	costwoActivationTime     = uint64(time.Date(2022, time.June, 1, 0, 0, 0, 0, time.UTC).Unix())
	stagingActivationTime    = uint64(time.Date(2022, time.June, 1, 0, 0, 0, 0, time.UTC).Unix())
	localFlareActivationTime = uint64(time.Date(2022, time.June, 1, 0, 0, 0, 0, time.UTC).Unix())

	songbirdActivationTime = uint64(time.Date(2022, time.March, 28, 14, 0, 0, 0, time.UTC).Unix())
	songbirdOct22ForkTime  = uint64(time.Date(2022, time.October, 19, 15, 0, 0, 0, time.UTC).Unix())

	costonActivationTime = uint64(time.Date(2022, time.February, 25, 17, 0, 0, 0, time.UTC).Unix())
	costonOct22ForkTime  = uint64(time.Date(2022, time.October, 6, 15, 0, 0, 0, time.UTC).Unix())
)

type AttestationVotes struct {
	reachedMajority    bool
	majorityDecision   string
	majorityAttestors  []common.Address
	divergentAttestors []common.Address
	abstainedAttestors []common.Address
}

var (
	stateConnectorActivationVariants = utils.NewChainValue(func(uint64, common.Address) bool { return false }).
		AddValue(params.FlareChainID, GetStateConnectorIsActivatedAndCalledFlare).
		AddValue(params.SongbirdChainID, GetStateConnectorIsActivatedAndCalledSongbird).
		AddValue(params.CostwoChainID, GetStateConnectorIsActivatedAndCalledCostwo).
		AddValue(params.CostonChainID, GetStateConnectorIsActivatedAndCalledCoston).
		AddValue(params.LocalFlareChainID, GetStateConnectorIsActivatedAndCalledLocalFlare).
		AddValue(params.LocalChainID, GetStateConnectorIsActivatedAndCalledLocal)
)

func GetStateConnectorIsActivatedAndCalled(isDurango bool, chainID *big.Int, blockTime uint64, to common.Address) bool {
	return !isDurango && stateConnectorActivationVariants.GetValue(chainID)(blockTime, to)
}

func GetStateConnectorIsActivatedAndCalledFlare(blockTime uint64, to common.Address) bool {
	return blockTime >= flareActivationTime &&
		to == common.HexToAddress("0x1000000000000000000000000000000000000001")
}

func GetStateConnectorIsActivatedAndCalledCostwo(blockTime uint64, to common.Address) bool {
	return blockTime >= costwoActivationTime &&
		to == common.HexToAddress("0x1000000000000000000000000000000000000001")
}

func GetStateConnectorIsActivatedAndCalledStaging(blockTime uint64, to common.Address) bool {
	return blockTime >= stagingActivationTime &&
		to == common.HexToAddress("0x1000000000000000000000000000000000000001")
}

func GetStateConnectorIsActivatedAndCalledLocalFlare(blockTime uint64, to common.Address) bool {
	return blockTime >= localFlareActivationTime &&
		to == common.HexToAddress("0x1000000000000000000000000000000000000001")
}

func GetStateConnectorIsActivatedAndCalledLocal(_ uint64, to common.Address) bool {
	return to == common.HexToAddress("0x1000000000000000000000000000000000000001")
}

func GetStateConnectorIsActivatedAndCalledSongbird(blockTime uint64, to common.Address) bool {
	switch {
	case blockTime > songbirdOct22ForkTime:
		return to == common.HexToAddress("0x0c13aDA1C7143Cf0a0795FFaB93eEBb6FAD6e4e3")
	case blockTime > songbirdActivationTime:
		return to == common.HexToAddress("0x3A1b3220527aBA427d1e13e4b4c48c31460B4d91")
	default:
		return false
	}
}

func GetStateConnectorIsActivatedAndCalledCoston(blockTime uint64, to common.Address) bool {
	switch {
	case blockTime > costonOct22ForkTime:
		return to == common.HexToAddress("0x0c13aDA1C7143Cf0a0795FFaB93eEBb6FAD6e4e3")
	case blockTime > costonActivationTime:
		return to == common.HexToAddress("0x947c76694491d3fD67a73688003c4d36C8780A97")
	default:
		return false
	}
}

// Signalling block.coinbase value
// address public constant SIGNAL_COINBASE = address(0x00000000000000000000000000000000000DEaD1);
// https://gitlab.com/flarenetwork/flare-smart-contracts/-/blob/6b6e5480c3cf769b5a650b961992b4f082761d76/contracts/genesis/implementation/StateConnector.sol#L17

func GetStateConnectorCoinbaseSignalAddr(chainID *big.Int, blockTime uint64) common.Address {
	switch {
	case chainID.Cmp(params.FlareChainID) == 0 || chainID.Cmp(params.CostwoChainID) == 0 || chainID.Cmp(params.LocalFlareChainID) == 0:
		return common.HexToAddress("0x00000000000000000000000000000000000DEaD1")
	case chainID.Cmp(params.SongbirdChainID) == 0:
		switch {
		case blockTime > songbirdOct22ForkTime:
			return common.HexToAddress("0x00000000000000000000000000000000000DEaD1")
		default:
			return common.HexToAddress("0x000000000000000000000000000000000000dEaD")
		}
	case chainID.Cmp(params.CostonChainID) == 0:
		switch {
		case blockTime > costonOct22ForkTime:
			return common.HexToAddress("0x00000000000000000000000000000000000DEaD1")
		default:
			return common.HexToAddress("0x000000000000000000000000000000000000dEaD")
		}
	default:
		return common.HexToAddress("0x000000000000000000000000000000000000dEaD")
	}
}

// function submitAttestation(uint256 _bufferNumber, bytes32 _commitHash, bytes32 _merkleRoot, bytes32 _randomNumber) external returns (bool _isInitialBufferSlot)
// https://gitlab.com/flarenetwork/flare-smart-contracts/-/blob/6b6e5480c3cf769b5a650b961992b4f082761d76/contracts/genesis/implementation/StateConnector.sol#L98

func SubmitAttestationSelector(chainID *big.Int, blockTime uint64) []byte {
	switch {
	default:
		return []byte{0xcf, 0xd1, 0xfd, 0xad}
	}
}

// function getAttestation(uint256 _bufferNumber) external view returns (bytes32 _merkleRoot)
// https://gitlab.com/flarenetwork/flare-smart-contracts/-/blob/6b6e5480c3cf769b5a650b961992b4f082761d76/contracts/genesis/implementation/StateConnector.sol#L123

func GetAttestationSelector(chainID *big.Int, blockTime uint64) []byte {
	switch {
	default:
		return []byte{0x29, 0xbe, 0x4d, 0xb2}
	}
}

// function finaliseRound(uint256 _bufferNumber, bytes32 _merkleRoot) external
// https://gitlab.com/flarenetwork/flare-smart-contracts/-/blob/6b6e5480c3cf769b5a650b961992b4f082761d76/contracts/genesis/implementation/StateConnector.sol#L137

func FinaliseRoundSelector(chainID *big.Int, blockTime uint64) []byte {
	switch {
	default:
		return []byte{0xea, 0xeb, 0xf6, 0xd3}
	}
}

// The default attestation providers for the state connector will be drawn from the top weighted/performing FTSOs.
func GetDefaultAttestors(chainID *big.Int, blockTime uint64) []common.Address {
	switch {
	case chainID.Cmp(params.FlareChainID) == 0:
		if blockTime > submitterContractActivationTimeFlare {
			return []common.Address{
				common.HexToAddress("0x4E07E1F3DB3Dc9BAd56Cc829747cc0148234329F"),
				common.HexToAddress("0xB264Fad6Fdc65767998f93501945aB8F9108809d"),
				common.HexToAddress("0x366BeC54195bfD45DBB34b79Ad2dEC4010598947"),
				common.HexToAddress("0x2665B179d5fCE1118f06e23B5d6E7617c5Ff733A"),
				common.HexToAddress("0x65cBaFaDD7C914179aabcE9C35f918a4E36AfFf9"),
				common.HexToAddress("0x7eC6a7C7c4Ef003A75DC6c06352B48B37Ac2191B"),
				common.HexToAddress("0xEa9bC2F98eFFC6A27E2C31733c1905961826f73B"),
				common.HexToAddress("0xA4aA75a9B49c7f2B4be62b2999d7103E78D004C7"),
				common.HexToAddress("0x4DF8436D7578C2d3bc73d33B6644913e131B70FC"),
			}
		} else {
			return []common.Address{
				common.HexToAddress("0x0988Cf4828F4e4eD0cE7c07467E70e19095Ee152"),
				common.HexToAddress("0x6BC7DCa62010D418eB72CCdc58561e00C5868Ef1"),
				common.HexToAddress("0xE34Bb361536610a9DCcEa5292262e36AfF65c06c"),
				common.HexToAddress("0x8A3D627D86A81F5D21683F4963565C63DB5e1309"),
				common.HexToAddress("0x2D3e7e4b19bDc920fd9C57BD3072A31F5a59FeC8"),
				common.HexToAddress("0x6455dC38fdF739b6fE021b30C7D9672C1c6DEb5c"),
				common.HexToAddress("0x49893c5Dfc035F4eE4E46faC014f6D4bC80F7f92"),
				common.HexToAddress("0x08e8b2Af4874e920de27723576A13d66008Af523"),
				common.HexToAddress("0x5D2f75392DdDa69a2818021dd6a64937904c8352"),
			}
		}
	case chainID.Cmp(params.SongbirdChainID) == 0:
		switch {
		case blockTime > submitterContractActivationTimeSongbird:
			return []common.Address{
				common.HexToAddress("0xcE397b9a395ace2e328030699bDDf4E2F049A05B"),
				common.HexToAddress("0xeDBb013BBC314124a9f842c1887e34cfeB03B052"),
				common.HexToAddress("0xb9eF3951ac2D04C6bdD886bF042041E3954E86aF"),
				common.HexToAddress("0x816Cec8f3A37Fd673Cfd4229441c59cA8DbD0641"),
				common.HexToAddress("0x14c9c4583F0b1af8a69452Ec1b29884240f83bDC"),
				common.HexToAddress("0x0049081C2D6def64800cC011Bd9aDe8682c6593a"),
				common.HexToAddress("0x53Fcb50a22aFd6e5438d754CB22c4726032d2488"),
				common.HexToAddress("0x35f4F0Bb73a6040F24927e1735B089d7769F7674"),
				common.HexToAddress("0x3B583C919fD4C863F3A17d11929346C687FfB7c3"),
			}
		case blockTime > songbirdOct22ForkTime:
			return []common.Address{
				common.HexToAddress("0x2D3e7e4b19bDc920fd9C57BD3072A31F5a59FeC8"),
				common.HexToAddress("0x442DD539Fe78D43A1a9358FF3460CfE63e2bC9CC"),
				common.HexToAddress("0x49893c5Dfc035F4eE4E46faC014f6D4bC80F7f92"),
				common.HexToAddress("0x5D2f75392DdDa69a2818021dd6a64937904c8352"),
				common.HexToAddress("0x6455dC38fdF739b6fE021b30C7D9672C1c6DEb5c"),
				common.HexToAddress("0x808441Ec3Fa1721330226E69527Bc160D8d9386a"),
				common.HexToAddress("0x823B0f5c7758E9d3bE55bA1EA840E29ccd5D5CcB"),
				common.HexToAddress("0x85016969b9eBDB8977975a4743c9FCEeabCEAf8A"),
				common.HexToAddress("0x8A3D627D86A81F5D21683F4963565C63DB5e1309"),
			}
		default:
			return []common.Address{
				common.HexToAddress("0x0c19f3B4927abFc596353B0f9Ddad5D817736F70"),
			}
		}
	case chainID.Cmp(params.CostwoChainID) == 0:
		return []common.Address{
			common.HexToAddress("0x30e4b4542b4aAf615838B113f14c46dE1469212e"),
			common.HexToAddress("0x3519E14183252794aaA52aA824f34482ef44cE1d"),
			common.HexToAddress("0xb445857476181ec378Ec453ab3d122183CfC3b78"),
			common.HexToAddress("0x6D755cd7A61A9DCFc96FaE0f927C3a73bE986ce4"),
			common.HexToAddress("0xdC0fD24846303D58d2D66AA8820be2685735dBd2"),
			common.HexToAddress("0x3F52c41c0500a4f018A38c9f8273b254aD7e2FCc"),
			common.HexToAddress("0xdA6d6aA9F1f770c279c5DA0C71f4DC1142A70d5D"),
			common.HexToAddress("0x3d895D00d2802120D39d4D2554F7ef09d6845E99"),
			common.HexToAddress("0xc36141CFBe5Af6eB2F8b21550Ccd457DA7FaF3C6"),
		}
	case chainID.Cmp(params.CostonChainID) == 0:
		switch {
		case blockTime > costonOct22ForkTime:
			return []common.Address{
				common.HexToAddress("0x30e4b4542b4aAf615838B113f14c46dE1469212e"),
				common.HexToAddress("0x3519E14183252794aaA52aA824f34482ef44cE1d"),
				common.HexToAddress("0xb445857476181ec378Ec453ab3d122183CfC3b78"),
				common.HexToAddress("0x6D755cd7A61A9DCFc96FaE0f927C3a73bE986ce4"),
				common.HexToAddress("0xdC0fD24846303D58d2D66AA8820be2685735dBd2"),
				common.HexToAddress("0x3F52c41c0500a4f018A38c9f8273b254aD7e2FCc"),
				common.HexToAddress("0xdA6d6aA9F1f770c279c5DA0C71f4DC1142A70d5D"),
				common.HexToAddress("0x3d895D00d2802120D39d4D2554F7ef09d6845E99"),
				common.HexToAddress("0xc36141CFBe5Af6eB2F8b21550Ccd457DA7FaF3C6"),
			}
		default:
			return []common.Address{
				common.HexToAddress("0x3a6e101103ec3d9267d08f484a6b70e1440a8255"),
			}
		}
	case chainID.Cmp(params.LocalFlareChainID) == 0:
		return []common.Address{
			common.HexToAddress("0x8db97C7cEcE249c2b98bDC0226Cc4C2A57BF52FC"), // Private key: 56289e99c94b6912bfc12adc093c9b51124f0dc54ac7a766b2bc5ccf558d8027
		}
	default:
		return []common.Address{}
	}
}

func GetLocalAttestors() []common.Address {
	var localAttestors []common.Address
	localAttestorList := os.Getenv(localAttestorEnv)
	if localAttestorList != "" {
		localAttestorEntries := strings.Split(localAttestorList, ",")
		for _, localAttestorEntry := range localAttestorEntries {
			localAttestors = append(localAttestors, common.HexToAddress(localAttestorEntry))
		}
	}
	return localAttestors
}

func (st *StateTransition) GetAttestation(attestor common.Address, instructions []byte) (string, error) {
	_, merkleRootHash, _, err := st.evm.DaemonCall(vm.AccountRef(attestor), st.to(), instructions, params.TxGas)
	return hex.EncodeToString(merkleRootHash), err
}

func (st *StateTransition) GetAttestations(attestors []common.Address, instructions []byte) (AttestationVotes, int, map[string][]common.Address) {
	var attestationVotes AttestationVotes
	hashFrequencies := make(map[string][]common.Address)
	for i, a := range attestors {
		h, err := st.GetAttestation(a, instructions)
		if err != nil {
			attestationVotes.abstainedAttestors = append(attestationVotes.abstainedAttestors, a)
		}
		hashFrequencies[h] = append(hashFrequencies[h], attestors[i])
	}
	return attestationVotes, len(attestors), hashFrequencies
}

func CountAttestations(attestationVotes AttestationVotes, numAttestors int, hashFrequencies map[string][]common.Address) AttestationVotes {
	// Find the plurality
	var pluralityNum int
	var pluralityKey string
	for key, val := range hashFrequencies {
		if len(val) > pluralityNum && len(key) > 0 {
			pluralityNum = len(val)
			pluralityKey = key
		}
	}
	if pluralityNum > numAttestors/2 {
		attestationVotes.reachedMajority = true
		attestationVotes.majorityDecision = pluralityKey
		attestationVotes.majorityAttestors = hashFrequencies[pluralityKey]
	}
	for key, val := range hashFrequencies {
		if key != pluralityKey {
			attestationVotes.divergentAttestors = append(attestationVotes.divergentAttestors, val...)
		}
	}
	return attestationVotes
}

func (st *StateTransition) FinalisePreviousRound(chainID *big.Int, timestamp uint64, currentRoundNumber []byte) error {
	getAttestationSelector := GetAttestationSelector(chainID, timestamp)
	instructions := append(getAttestationSelector[:], currentRoundNumber[:]...)
	defaultAttestors := GetDefaultAttestors(chainID, timestamp)
	defaultAttestationVotes := CountAttestations(st.GetAttestations(defaultAttestors, instructions))
	localAttestors := GetLocalAttestors()
	finalityReached := defaultAttestationVotes.reachedMajority
	if len(localAttestors) > 0 {
		localAttestationVotes := CountAttestations(st.GetAttestations(localAttestors, instructions))
		if finalityReached && defaultAttestationVotes.majorityDecision != localAttestationVotes.majorityDecision && os.Getenv(forkingEnabledEnv) == "1" {
			// Fork this node now from the default path
			return fmt.Errorf(
				"default state connector decision (%s) does not match this node's local state connector decision (%s), forking node",
				defaultAttestationVotes.majorityDecision,
				localAttestationVotes.majorityDecision,
			)
		}
	}
	if finalityReached {
		// Finalise defaultAttestationVotes.majorityDecision
		finaliseRoundSelector := FinaliseRoundSelector(chainID, timestamp)
		finalisedData := append(finaliseRoundSelector[:], currentRoundNumber[:]...)
		merkleRootHashBytes, err := hex.DecodeString(defaultAttestationVotes.majorityDecision)
		if err != nil {
			return err
		}
		finalisedData = append(finalisedData[:], merkleRootHashBytes[:]...)
		coinbaseSignal := GetStateConnectorCoinbaseSignalAddr(chainID, timestamp)
		originalCoinbase := st.evm.Context.Coinbase
		defer func() {
			st.evm.Context.Coinbase = originalCoinbase
		}()
		// Setting msg.sender = block.coinbase and block.coinbase = SIGNAL_COINBASE signals to the EVM to finalise this round
		// See: https://gitlab.com/flarenetwork/flare-smart-contracts/-/blob/57ac7259f1708832201b774fc3445e0fbfb94ef4/contracts/genesis/implementation/StateConnector.sol#L132
		st.evm.Context.Coinbase = coinbaseSignal
		// In order to break the State Connector's signalling mechanism, one would have to both:
		// 		1) Change the Flare validator code to enable them to control the block.coinbase variable. This is mitigated in state_transition.go
		//				by this check: burnAddress == common.HexToAddress("0x0100000000000000000000000000000000000000") on line 373, which occurs
		//				right before st.FinalisePreviousRound(chainID, timestamp, st.data[4:36]) is called.
		//		2) Know the private key to the address 0x00000000000000000000000000000000000DEaD1 in order to become msg.sender.
		_, _, _, err = st.evm.DaemonCall(vm.AccountRef(coinbaseSignal), st.to(), finalisedData, st.evm.Context.GasLimit)
		if err != nil {
			return err
		}
	}
	return nil
}
