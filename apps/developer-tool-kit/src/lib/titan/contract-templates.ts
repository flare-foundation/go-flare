import { TITAN_CHESS_ESCROW_SOURCE } from "@/lib/titan/sources/titan-chess-escrow.source";

export type ContractTemplate = {
  id: string;
  name: string;
  description: string;
  source: string;
  fileName?: string;
  /** Default constructor arg values keyed by parameter name */
  constructorDefaults?: Record<string, string>;
};

export const CONTRACT_TEMPLATES: ContractTemplate[] = [
  {
    id: "greeter",
    name: "Greeter",
    description: "Simple string storage with greet() and setGreeting().",
    source: `// SPDX-License-Identifier: MIT
pragma solidity ^0.8.19;

contract Greeter {
    string private greeting;

    constructor(string memory _greeting) {
        greeting = _greeting;
    }

    function greet() public view returns (string memory) {
        return greeting;
    }

    function setGreeting(string memory _greeting) public {
        greeting = _greeting;
    }
}
`,
  },
  {
    id: "counter",
    name: "Counter",
    description: "Minimal counter with increment and decrement.",
    source: `// SPDX-License-Identifier: MIT
pragma solidity ^0.8.19;

contract Counter {
    uint256 public count;

    constructor(uint256 initial) {
        count = initial;
    }

    function increment() public {
        count += 1;
    }

    function decrement() public {
        require(count > 0, "Counter: cannot decrement below zero");
        count -= 1;
    }
}
`,
  },
  {
    id: "simple-storage",
    name: "SimpleStorage",
    description: "Store and retrieve a single uint256 value.",
    source: `// SPDX-License-Identifier: MIT
pragma solidity ^0.8.19;

contract SimpleStorage {
    uint256 private storedValue;

    event ValueChanged(uint256 newValue);

    constructor(uint256 initialValue) {
        storedValue = initialValue;
    }

    function set(uint256 value) public {
        storedValue = value;
        emit ValueChanged(value);
    }

    function get() public view returns (uint256) {
        return storedValue;
    }
}
`,
  },
  {
    id: "titan-chess-escrow",
    name: "TitanChessEscrow",
    description:
      "FIFO wager queue: player stakes TITAN, Stockfish operator matches stake, winner takes the pot. For apps/titan-chess.",
    fileName: "TitanChessEscrow.sol",
    source: TITAN_CHESS_ESCROW_SOURCE,
    constructorDefaults: {
      _stockfishOperator: "0x49077293fe7049400A91D14395dbCad16A98Ea47",
      _minStake: "10000000000000000",
      _maxStake: "1000000000000000000",
    },
  },
];

export const DEFAULT_TEMPLATE_ID = "greeter";