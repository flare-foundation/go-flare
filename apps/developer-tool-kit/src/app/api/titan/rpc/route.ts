import { NextResponse } from "next/server";

const NODES = [
  { node: "node1", rpc: "http://localhost:9650", port: 9650 },
  { node: "node2", rpc: "http://localhost:9652", port: 9652 },
  { node: "node3", rpc: "http://localhost:9654", port: 9654 },
];

async function jsonRpc(url: string, method: string, params: unknown[] = []) {
  const res = await fetch(url, {
    method: "POST",
    headers: { "content-type": "application/json" },
    body: JSON.stringify({ jsonrpc: "2.0", id: 1, method, params }),
    signal: AbortSignal.timeout(3000),
  });
  return res.json();
}

export async function GET() {
  const results = await Promise.allSettled(
    NODES.map(async ({ node, rpc, port }) => {
      const cRpc = `${rpc}/ext/bc/C/rpc`;
      const infoRpc = `${rpc}/ext/info`;
      const healthRpc = `${rpc}/ext/health`;

      const [healthRes, peersRes, chainIdRes, blockRes, gasPriceRes] =
        await Promise.allSettled([
          fetch(healthRpc, { signal: AbortSignal.timeout(3000) }).then((r) =>
            r.json()
          ),
          jsonRpc(infoRpc, "info.peers"),
          jsonRpc(cRpc, "eth_chainId"),
          jsonRpc(cRpc, "eth_blockNumber"),
          jsonRpc(cRpc, "eth_gasPrice"),
        ]);

      const healthy =
        healthRes.status === "fulfilled" &&
        healthRes.value?.healthy === true;
      const peers =
        peersRes.status === "fulfilled"
          ? Number(peersRes.value?.result?.numPeers ?? 0)
          : 0;
      const chainIdHex =
        chainIdRes.status === "fulfilled"
          ? chainIdRes.value?.result
          : undefined;
      const chainId = chainIdHex
        ? String(parseInt(chainIdHex, 16))
        : undefined;
      const blockHex =
        blockRes.status === "fulfilled" ? blockRes.value?.result : undefined;
      const blockNumber = blockHex
        ? String(parseInt(blockHex, 16))
        : undefined;
      const gasPriceHex =
        gasPriceRes.status === "fulfilled"
          ? gasPriceRes.value?.result
          : undefined;
      const gasPrice = gasPriceHex
        ? `${(BigInt(gasPriceHex) / BigInt(1e9)).toString()} gwei`
        : undefined;

      return { node, port, healthy, peers, chainId, blockNumber, gasPrice };
    })
  );

  const nodes = results.map((r, i) =>
    r.status === "fulfilled"
      ? r.value
      : {
          node: NODES[i].node,
          port: NODES[i].port,
          healthy: false,
          peers: 0,
          error: String((r as PromiseRejectedResult).reason),
        }
  );

  return NextResponse.json({ nodes });
}
