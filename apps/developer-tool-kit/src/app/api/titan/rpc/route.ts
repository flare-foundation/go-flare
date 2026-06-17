import { NextRequest, NextResponse } from "next/server";

const NODES = [
  { node: "node1", rpc: "http://localhost:9650", port: 9650 },
  { node: "node2", rpc: "http://localhost:9652", port: 9652 },
  { node: "node3", rpc: "http://localhost:9654", port: 9654 },
];

async function jsonRpc(url: string, method: string, params: unknown[] = [], timeoutMs = 3000) {
  const res = await fetch(url, {
    method: "POST",
    headers: { "content-type": "application/json" },
    body: JSON.stringify({ jsonrpc: "2.0", id: Date.now(), method, params }),
    signal: AbortSignal.timeout(timeoutMs),
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

// POST: Proxy generic JSON-RPC calls for the explorer (and other clients).
// Body: { method: string, params?: unknown[], node?: "node1"|"node2"|"node3", chain?: "C"|"info" }
export async function POST(req: NextRequest) {
  try {
    const body = (await req.json()) as {
      method?: string;
      params?: unknown[];
      node?: string;
      chain?: string;
    };

    const method = body.method;
    if (!method || typeof method !== "string") {
      return NextResponse.json(
        { jsonrpc: "2.0", error: { code: -32600, message: "method is required" } },
        { status: 400 },
      );
    }

    const params = Array.isArray(body.params) ? body.params : [];
    const nodeName = body.node ?? "node1";
    const chain = (body.chain ?? "C").toUpperCase();

    const target = NODES.find((n) => n.node === nodeName) ?? NODES[0];
    const base = target.rpc;

    let path = "/ext/bc/C/rpc";
    if (chain === "INFO") path = "/ext/info";
    else if (chain === "HEALTH") path = "/ext/health";
    else if (chain === "P") path = "/ext/bc/P/rpc"; // if needed later
    // default C-chain for eth_*

    const url = `${base}${path}`;

    const resp = await fetch(url, {
      method: "POST",
      headers: { "content-type": "application/json" },
      body: JSON.stringify({ jsonrpc: "2.0", id: Date.now(), method, params }),
      signal: AbortSignal.timeout(15000),
    });

    const j = await resp.json();
    return NextResponse.json(j);
  } catch (err: unknown) {
    const message = err instanceof Error ? err.message : String(err);
    return NextResponse.json(
      { jsonrpc: "2.0", error: { code: -32603, message } },
      { status: 500 },
    );
  }
}
