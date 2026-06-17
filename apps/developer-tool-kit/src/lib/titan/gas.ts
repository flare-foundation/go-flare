/** Genesis min base fee on Titan local UAT (25 gwei). MetaMask fails when RPC returns 0. */
export const TITAN_FALLBACK_GAS_PRICE_WEI = BigInt(25_000_000_000);

type RpcResponse<T> = { result?: T; error?: { message?: string; code?: number } };

async function titanRpc<T>(method: string, params: unknown[] = []): Promise<T> {
  const res = await fetch("/api/titan/rpc", {
    method: "POST",
    headers: { "content-type": "application/json" },
    body: JSON.stringify({ method, params, node: "node1", chain: "C" }),
  });

  const json = (await res.json()) as RpcResponse<T>;
  if (json.error) {
    throw new Error(json.error.message ?? `RPC ${method} failed.`);
  }
  return json.result as T;
}

export async function fetchTitanGasPriceWei(): Promise<bigint> {
  try {
    const hex = await titanRpc<string>("eth_gasPrice");
    const price = hex ? BigInt(hex) : BigInt(0);
    return price > BigInt(0) ? price : TITAN_FALLBACK_GAS_PRICE_WEI;
  } catch {
    return TITAN_FALLBACK_GAS_PRICE_WEI;
  }
}

export async function estimateGasViaRpc(from: string, data: string): Promise<bigint> {
  const hex = await titanRpc<string>("eth_estimateGas", [
    {
      from,
      data,
      value: "0x0",
    },
  ]);
  return BigInt(hex);
}

export function toHex(value: bigint): string {
  return `0x${value.toString(16)}`;
}