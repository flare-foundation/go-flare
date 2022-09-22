// (c) 2021, Flare Networks Limited. All rights reserved.
// Please see the file LICENSE for licensing terms.

package core

import (
	"testing"

	"github.com/ethereum/go-ethereum/common"
)

// TestCountAttestationsEmpty checks that the CountAttestations function in state_connector.go
// correctly handles an empty set
func TestCountAttestationsEmpty(t *testing.T) {
	var attestationVotes AttestationVotes
	numAttestors := 0
	hashFrequencies := make(map[string][]common.Address)
	returnedAttestationVotes := CountAttestations(attestationVotes, numAttestors, hashFrequencies)

	want := false
	if returnedAttestationVotes.reachedMajority != want {
		t.Fatalf(`reachedMajority = %t, want %t`, returnedAttestationVotes.reachedMajority, want)
	}
}

// TestCountAttestationsMajorityReached checks that the CountAttestations function in state_connector.go
// correctly handles a majority result
func TestCountAttestationsMajorityReached(t *testing.T) {
	var attestationVotes AttestationVotes
	numAttestors := 5
	hashFrequencies := make(map[string][]common.Address)

	// Hash 1
	majorityHash := "953fbdd4ac2d5a2f1e413cbd378be0f3135010d81b4b643c6020e96ca49fc0c9"
	hashFrequencies[majorityHash] = []common.Address{
		common.HexToAddress("0x0c19f3B4927abFc596353B0f9Ddad5D817736F70"),
		common.HexToAddress("0x3a6e101103ec3d9267d08f484a6b70e1440a8255"),
		common.HexToAddress("0xe51605047a50fc70143d98cb0b090bb1b157b6ae"),
	}

	// Hash 2
	minorityHash := "1447bc5b8b57cc4afcba95b545aa0e3e166da0e12bd1beec9377bfaa4b84b6a6"
	hashFrequencies[minorityHash] = []common.Address{
		common.HexToAddress("0x0b63d67989fa94e702bd976fc33d83308b7ca1b7"),
		common.HexToAddress("0x3ec71a4c026a315d7d2415e901a1cc923aee1474"),
	}

	returnedAttestationVotes := CountAttestations(attestationVotes, numAttestors, hashFrequencies)

	wantMajority := true
	wantHash := majorityHash
	if returnedAttestationVotes.reachedMajority != wantMajority || returnedAttestationVotes.majorityDecision != wantHash {
		t.Fatalf(`reachedMajority = %t and majorityDecision = %s, want %t and %s`, returnedAttestationVotes.reachedMajority, returnedAttestationVotes.majorityDecision, wantMajority, wantHash)
	}
}

// TestCountAttestationsNoMajorityReached checks that the CountAttestations function in state_connector.go
// correctly handles a no majority reached result
func TestCountAttestationsNoMajorityReached(t *testing.T) {
	var attestationVotes AttestationVotes
	numAttestors := 5
	hashFrequencies := make(map[string][]common.Address)

	// Hash 1
	minorityHash1 := "451af0bda5d105216ee500ec403f93e1597922f96427fd17a0b1025d9ba39f0f"
	hashFrequencies[minorityHash1] = []common.Address{
		common.HexToAddress("0x1133e938080622e323ca9522040fafcdcb40b926"),
		common.HexToAddress("0xba42574bacb0487343c7d3c1765ac798c1002c76"),
	}

	// Hash 2
	minorityHash3 := "1c3094c661450e4df987445546d37442a70e7696a6d81a8f74cd9bf56441b6f0"
	hashFrequencies[minorityHash3] = []common.Address{
		common.HexToAddress("0x1f4ab9148bfec95df02025df27cc270202a25663"),
	}

	// Hash 3
	minorityHash4 := "291c4f5e443fc97ed86fbf1269aac512520291cbb8c0c682058d44a306382b5c"
	hashFrequencies[minorityHash4] = []common.Address{
		common.HexToAddress("0x42581d4393f6244d9859df14d7f7e8e96ea3542b"),
	}

	// Hash 4
	minorityHash5 := "991e45909a27ece438f57dd5ed5bbe30870a8ba68a0ccea6f377fdeafdaac0a8"
	hashFrequencies[minorityHash5] = []common.Address{
		common.HexToAddress("0xdc5e866f6dd7e1a69eb807e1adf445dae895b3b0"),
	}

	returnedAttestationVotes := CountAttestations(attestationVotes, numAttestors, hashFrequencies)

	want := false
	if returnedAttestationVotes.reachedMajority != want {
		t.Fatalf(`reachedMajority = %t, want %t`, returnedAttestationVotes.reachedMajority, want)
	}
}

// TestCountAttestationsBlankVotes checks that the CountAttestations function in state_connector.go
// correctly handles a result where attestors abstain or vote with a blank string
func TestCountAttestationsBlankVotes(t *testing.T) {
	var attestationVotes AttestationVotes
	numAttestors := 5
	hashFrequencies := make(map[string][]common.Address)

	// Blank Hash
	emptyHash := ""
	hashFrequencies[emptyHash] = []common.Address{
		common.HexToAddress("0x1133e938080622e323ca9522040fafcdcb40b926"),
		common.HexToAddress("0xba42574bacb0487343c7d3c1765ac798c1002c76"),
		common.HexToAddress("0x1f4ab9148bfec95df02025df27cc270202a25663"),
		common.HexToAddress("0x42581d4393f6244d9859df14d7f7e8e96ea3542b"),
		common.HexToAddress("0xdc5e866f6dd7e1a69eb807e1adf445dae895b3b0"),
	}

	returnedAttestationVotes := CountAttestations(attestationVotes, numAttestors, hashFrequencies)

	want := false
	if returnedAttestationVotes.reachedMajority != want {
		t.Fatalf(`reachedMajority = %t, want %t`, returnedAttestationVotes.reachedMajority, want)
	}
}
