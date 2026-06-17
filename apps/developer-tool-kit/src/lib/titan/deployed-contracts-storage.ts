import { getLocalStorageValue, setLocalStorageValue } from "@/lib/local-storage.client";

export type DeployedContractRecord = {
  id: string;
  contractName: string;
  contractAddress: string;
  transactionHash: string | null;
  deployer: string | null;
  deployedAt: string;
};

const STORAGE_KEY = "titan-deployed-contracts";

export function loadDeployedContracts(): DeployedContractRecord[] {
  const raw = getLocalStorageValue(STORAGE_KEY);
  if (!raw) return [];

  try {
    const parsed = JSON.parse(raw) as DeployedContractRecord[];
    return Array.isArray(parsed) ? parsed : [];
  } catch {
    return [];
  }
}

export function saveDeployedContracts(records: DeployedContractRecord[]) {
  setLocalStorageValue(STORAGE_KEY, JSON.stringify(records));
}

export function addDeployedContract(record: Omit<DeployedContractRecord, "id" | "deployedAt"> & { deployedAt?: string }) {
  const existing = loadDeployedContracts();
  const normalized = record.contractAddress.toLowerCase();
  const next = existing.filter((item) => item.contractAddress.toLowerCase() !== normalized);

  next.unshift({
    id: `deploy-${normalized}-${Date.now()}`,
    deployedAt: record.deployedAt ?? new Date().toISOString(),
    ...record,
  });

  const trimmed = next.slice(0, 50);
  saveDeployedContracts(trimmed);
  return trimmed;
}

export function removeDeployedContract(contractAddress: string) {
  const normalized = contractAddress.toLowerCase();
  const next = loadDeployedContracts().filter((item) => item.contractAddress.toLowerCase() !== normalized);
  saveDeployedContracts(next);
  return next;
}

export async function isContractAddress(address: string): Promise<boolean> {
  const res = await fetch("/api/titan/rpc", {
    method: "POST",
    headers: { "content-type": "application/json" },
    body: JSON.stringify({
      method: "eth_getCode",
      params: [address, "latest"],
      node: "node1",
      chain: "C",
    }),
  });

  const json = (await res.json()) as { result?: string; error?: unknown };
  if (json.error || !json.result) return false;
  return json.result !== "0x" && json.result !== "0x0";
}