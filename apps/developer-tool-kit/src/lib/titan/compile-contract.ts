import solc from "solc";

export type AbiConstructorInput = {
  name: string;
  type: string;
  internalType?: string;
};

export type CompiledContract = {
  contractName: string;
  abi: readonly Record<string, unknown>[];
  bytecode: string;
  constructorInputs: AbiConstructorInput[];
};

export type CompileContractResult =
  | { ok: true; contract: CompiledContract }
  | { ok: false; errors: string[] };

type SolcOutput = {
  errors?: Array<{ severity: string; formattedMessage?: string; message?: string }>;
  contracts?: Record<
    string,
    Record<
      string,
      {
        abi?: Record<string, unknown>[];
        evm?: { bytecode?: { object?: string } };
      }
    >
  >;
};

export function compileSoliditySource(source: string, fileName = "Contract.sol"): CompileContractResult {
  const input = JSON.stringify({
    language: "Solidity",
    sources: {
      [fileName]: { content: source },
    },
    settings: {
      optimizer: { enabled: true, runs: 200 },
      // Titan C-Chain does not support PUSH0 (Shanghai); target Paris EVM.
      evmVersion: "paris",
      outputSelection: {
        "*": {
          "*": ["abi", "evm.bytecode"],
        },
      },
    },
  });

  let outputRaw: string;
  try {
    outputRaw = solc.compile(input);
  } catch (error) {
    return {
      ok: false,
      errors: [error instanceof Error ? error.message : "Solidity compiler failed."],
    };
  }

  const output = JSON.parse(outputRaw) as SolcOutput;
  const errors =
    output.errors
      ?.filter((e) => e.severity === "error")
      .map((e) => e.formattedMessage ?? e.message ?? "Unknown compiler error") ?? [];

  if (errors.length > 0) {
    return { ok: false, errors };
  }

  const fileContracts = output.contracts?.[fileName];
  if (!fileContracts) {
    return { ok: false, errors: ["No contracts found in source file."] };
  }

  const entries = Object.entries(fileContracts).filter(([, data]) => {
    const bytecode = data.evm?.bytecode?.object ?? "";
    return bytecode.length > 0;
  });

  if (entries.length === 0) {
    return { ok: false, errors: ["Compilation succeeded but no deployable bytecode was produced."] };
  }

  const [contractName, artifact] = entries[entries.length - 1];
  const abi = artifact.abi ?? [];
  const bytecode = artifact.evm?.bytecode?.object ?? "";

  if (!bytecode) {
    return { ok: false, errors: ["Compiled contract bytecode is empty."] };
  }

  const constructor = abi.find((item) => item.type === "constructor") as
    | { inputs?: AbiConstructorInput[] }
    | undefined;

  return {
    ok: true,
    contract: {
      contractName,
      abi,
      bytecode,
      constructorInputs: constructor?.inputs ?? [],
    },
  };
}