export type ContractTemplate = {
  id: string;
  name: string;
  description: string;
  source: string;
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
];

export const DEFAULT_TEMPLATE_ID = "greeter";