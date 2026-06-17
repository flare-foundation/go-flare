import fs from "node:fs";
import path from "node:path";

export const TITAN_GITHUB_REPO = "https://github.com/pakeku/go-titan";
export const TITAN_GITHUB_BRANCH = "dev/explorer";
export const TITAN_ORIGIN_GITHUB_PATH = "titan-network/origin.json";
export const TITAN_ORIGIN_GITHUB_RAW_URL =
  `https://raw.githubusercontent.com/pakeku/go-titan/${TITAN_GITHUB_BRANCH}/${TITAN_ORIGIN_GITHUB_PATH}`;
export const TITAN_ORIGIN_GITHUB_BLOB_URL =
  `${TITAN_GITHUB_REPO}/blob/${TITAN_GITHUB_BRANCH}/${TITAN_ORIGIN_GITHUB_PATH}`;

export interface TitanOriginAllocation {
  ethAddr: string;
  avaxAddr: string;
  initialAmount: number;
  unlockSchedule: Array<{ amount: number; locktime: number }>;
}

export interface TitanOriginStaker {
  nodeID: string;
  rewardAddress: string;
  delegationFee: number;
}

export interface TitanCChainGenesis {
  config?: {
    chainId?: number;
    feeConfig?: Record<string, number>;
  };
  alloc?: Record<string, { balance: string }>;
}

export interface TitanOrigin {
  networkID: number;
  allocations: TitanOriginAllocation[];
  message: string;
  startTime: number;
  initialStakeDuration: number;
  initialStakeDurationOffset: number;
  initialStakedFunds: string[];
  initialStakers: TitanOriginStaker[];
  cChainGenesis: string;
}

export interface TitanOriginSummary {
  origin: TitanOrigin;
  cChain: TitanCChainGenesis;
  prefundedAccounts: Array<{ address: string; balanceWei: string; balanceTitan: string }>;
  sourcePath: string;
  githubRawUrl: string;
  githubBlobUrl: string;
}

function repoOriginPath(): string {
  return path.resolve(process.cwd(), "../../titan-network/origin.json");
}

export function readOriginFile(): TitanOriginSummary {
  const sourcePath = repoOriginPath();
  const raw = fs.readFileSync(sourcePath, "utf8");
  const origin = JSON.parse(raw) as TitanOrigin;
  const cChain = JSON.parse(origin.cChainGenesis) as TitanCChainGenesis;

  const prefundedAccounts = Object.entries(cChain.alloc ?? {})
    .map(([address, { balance }]) => ({
      address,
      balanceWei: balance,
      balanceTitan: formatWeiToTitan(balance),
    }))
    .sort((a, b) => a.address.localeCompare(b.address));

  return {
    origin,
    cChain,
    prefundedAccounts,
    sourcePath,
    githubRawUrl: TITAN_ORIGIN_GITHUB_RAW_URL,
    githubBlobUrl: TITAN_ORIGIN_GITHUB_BLOB_URL,
  };
}

export function formatWeiToTitan(wei: string): string {
  const value = BigInt(wei);
  const whole = value / 10n ** 18n;
  const fraction = value % 10n ** 18n;
  if (fraction === 0n) {
    return `${whole} TITAN`;
  }
  const fractionStr = fraction.toString().padStart(18, "0").replace(/0+$/, "");
  return `${whole}.${fractionStr} TITAN`;
}

export function formatUnixTime(seconds: number): string {
  return new Date(seconds * 1000).toUTCString();
}