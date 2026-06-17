import type { Abi } from "viem";

import { CONTRACT_TEMPLATES } from "@/lib/titan/contract-templates";
import type { DeployedContractRecord } from "@/lib/titan/deployed-contracts-storage";

export type SandboxTemplateId = "greeter" | "counter" | "simple-storage";

export const SANDBOX_ABIS: Record<SandboxTemplateId, Abi> = {
  greeter: [
    {
      type: "function",
      name: "greet",
      stateMutability: "view",
      inputs: [],
      outputs: [{ name: "", type: "string" }],
    },
    {
      type: "function",
      name: "setGreeting",
      stateMutability: "nonpayable",
      inputs: [{ name: "_greeting", type: "string" }],
      outputs: [],
    },
  ],
  counter: [
    {
      type: "function",
      name: "count",
      stateMutability: "view",
      inputs: [],
      outputs: [{ name: "", type: "uint256" }],
    },
    {
      type: "function",
      name: "increment",
      stateMutability: "nonpayable",
      inputs: [],
      outputs: [],
    },
    {
      type: "function",
      name: "decrement",
      stateMutability: "nonpayable",
      inputs: [],
      outputs: [],
    },
  ],
  "simple-storage": [
    {
      type: "function",
      name: "get",
      stateMutability: "view",
      inputs: [],
      outputs: [{ name: "", type: "uint256" }],
    },
    {
      type: "function",
      name: "set",
      stateMutability: "nonpayable",
      inputs: [{ name: "value", type: "uint256" }],
      outputs: [],
    },
  ],
};

export function templateIdForContractName(contractName: string): SandboxTemplateId | null {
  const template = CONTRACT_TEMPLATES.find((item) => item.name === contractName);
  if (!template) return null;
  return isSandboxTemplateId(template.id) ? template.id : null;
}

export function resolveSandboxTemplateId(record: DeployedContractRecord): SandboxTemplateId | null {
  if (record.templateId && isSandboxTemplateId(record.templateId)) {
    return record.templateId;
  }
  return templateIdForContractName(record.contractName);
}

export function isSandboxContract(record: DeployedContractRecord): boolean {
  return resolveSandboxTemplateId(record) !== null;
}

export function sandboxLabel(templateId: SandboxTemplateId): string {
  return CONTRACT_TEMPLATES.find((item) => item.id === templateId)?.name ?? templateId;
}

function isSandboxTemplateId(value: string): value is SandboxTemplateId {
  return value === "greeter" || value === "counter" || value === "simple-storage";
}