"use client";

import { Suspense, useCallback, useEffect, useRef, useState } from "react";
import { usePathname, useRouter, useSearchParams } from "next/navigation";
import {
  Activity,
  Clock,
  Hash,
  Loader2,
  RefreshCw,
  Search,
  Zap,
} from "lucide-react";

import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import {
  AddressDetail,
  ExplorerDetailDrawer,
} from "@/app/(main)/dashboard/activity/_components/explorer-detail-drawer";

interface NodeInfo {
  node: string;
  port: number;
  healthy: boolean;
  peers: number;
  chainId?: string;
  blockNumber?: string;
  gasPrice?: string;
}

interface Tx {
  blockHash?: string;
  blockNumber?: string;
  from: string;
  gas: string;
  gasPrice?: string;
  hash: string;
  input: string;
  nonce: string;
  to: string | null;
  transactionIndex?: string;
  value: string;
  type?: string;
  // extended when full
}

interface Block {
  number: string;
  hash: string;
  parentHash: string;
  timestamp: string;
  miner?: string; // coinbase
  gasUsed: string;
  gasLimit: string;
  baseFeePerGas?: string;
  difficulty?: string;
  extraData?: string;
  transactions: Tx[] | string[]; // full objects when requested with true
  transactionCount?: number;
}

interface Receipt {
  status: string;
  gasUsed: string;
  effectiveGasPrice?: string;
  logs: unknown[];
  blockNumber: string;
  transactionHash: string;
}

const NODES = [
  { node: "node1", port: 9650 },
  { node: "node2", port: 9652 },
  { node: "node3", port: 9654 },
];

const BLOCKS_PAGE_SIZE = 20;

// Small RPC helper via our proxy (targets C-Chain by default)
async function rpc(method: string, params: unknown[] = [], node = "node1"): Promise<unknown> {
  const res = await fetch("/api/titan/rpc", {
    method: "POST",
    headers: { "content-type": "application/json" },
    body: JSON.stringify({ method, params, node, chain: "C" }),
  });
  const j = await res.json();
  if (j?.error) {
    const msg = typeof j.error === "string" ? j.error : j.error?.message || JSON.stringify(j.error);
    throw new Error(msg);
  }
  return j?.result;
}

function shortHash(h?: string | null, left = 6, right = 4): string {
  if (!h) return "—";
  if (h.length <= left + right + 2) return h;
  return `${h.slice(0, left)}…${h.slice(-right)}`;
}

function formatWeiToTitan(hexOrBig?: string | bigint): string {
  if (hexOrBig == null) return "0";
  try {
    const wei = typeof hexOrBig === "string" ? BigInt(hexOrBig) : hexOrBig;
    const titan = Number(wei) / 1e18;
    if (Math.abs(titan) < 0.0001 && titan !== 0) return titan.toExponential(2);
    return titan.toFixed(4);
  } catch {
    return "0";
  }
}

function formatGwei(hex?: string): string {
  if (!hex) return "—";
  try {
    const wei = BigInt(hex);
    const g = Number(wei) / 1e9;
    return g.toFixed(g < 1 ? 4 : 2) + " gwei";
  } catch {
    return "—";
  }
}

function hexToNumber(hex?: string): number | null {
  if (!hex) return null;
  try {
    return parseInt(hex, 16);
  } catch {
    return null;
  }
}

function formatTimestamp(tsHex?: string): { full: string; ago: string } {
  if (!tsHex) return { full: "—", ago: "—" };
  const sec = hexToNumber(tsHex) ?? 0;
  const d = new Date(sec * 1000);
  const full = d.toLocaleString();
  const diff = Math.floor((Date.now() - d.getTime()) / 1000);
  let ago = "";
  if (diff < 60) ago = `${diff}s ago`;
  else if (diff < 3600) ago = `${Math.floor(diff / 60)}m ago`;
  else if (diff < 86400) ago = `${Math.floor(diff / 3600)}h ago`;
  else ago = `${Math.floor(diff / 86400)}d ago`;
  return { full, ago };
}

function isTxHash(s: string) {
  return /^0x[0-9a-fA-F]{64}$/.test(s);
}
function isAddress(s: string) {
  return /^0x[0-9a-fA-F]{40}$/.test(s);
}
function isBlockNumber(s: string) {
  if (/^\d+$/.test(s)) return true;
  // 64-char hex values are block/tx hashes, not block numbers (RPC max is 64 bits).
  if (/^0x[0-9a-fA-F]{1,16}$/i.test(s)) return true;
  return false;
}

function ExplorerPageContent() {
  const searchParams = useSearchParams();
  const router = useRouter();
  const pathname = usePathname();
  const skipUrlSearchRef = useRef(false);
  const [nodes, setNodes] = useState<NodeInfo[]>([]);
  const [nodesLoading, setNodesLoading] = useState(true);
  const [lastUpdated, setLastUpdated] = useState<Date | null>(null);

  const [blocks, setBlocks] = useState<Block[]>([]);
  const [blocksLoading, setBlocksLoading] = useState(true);
  const [loadingMore, setLoadingMore] = useState(false);
  const [hasMoreBlocks, setHasMoreBlocks] = useState(true);
  const [blocksError, setBlocksError] = useState<string>("");
  const loadMoreRef = useRef<HTMLDivElement>(null);

  const [selectedBlock, setSelectedBlock] = useState<Block | null>(null);
  const [selectedBlockLoading, setSelectedBlockLoading] = useState(false);

  const [selectedTxHash, setSelectedTxHash] = useState<string | null>(null);
  const [selectedTx, setSelectedTx] = useState<Tx | null>(null);
  const [selectedReceipt, setSelectedReceipt] = useState<Receipt | null>(null);
  const [txLoading, setTxLoading] = useState(false);

  const [searchValue, setSearchValue] = useState("");
  const [searchLoading, setSearchLoading] = useState(false);
  const [searchError, setSearchError] = useState("");
  const [addressDetail, setAddressDetail] = useState<AddressDetail | null>(null);

  const syncUrlParam = useCallback(
    (q: string | null) => {
      skipUrlSearchRef.current = true;
      if (q) {
        router.replace(`${pathname}?q=${encodeURIComponent(q)}`, { scroll: false });
      } else {
        router.replace(pathname, { scroll: false });
      }
    },
    [pathname, router],
  );

  function closeDrawer() {
    setSelectedBlock(null);
    setSelectedTxHash(null);
    setSelectedTx(null);
    setSelectedReceipt(null);
    setAddressDetail(null);
    syncUrlParam(null);
  }

  // Load node overview (reused from old activity)
  async function loadNodes() {
    setNodesLoading(true);
    try {
      const r = await fetch("/api/titan/rpc");
      const j = await r.json();
      setNodes(j.nodes ?? []);
      setLastUpdated(new Date());
    } catch {
      setNodes([]);
    } finally {
      setNodesLoading(false);
    }
  }

  // Fetch a single block (with full tx objects)
  const fetchBlock = useCallback(async (identifier: string | number, isHash = false): Promise<Block | null> => {
    try {
      const param = isHash
        ? String(identifier)
        : typeof identifier === "number"
          ? `0x${identifier.toString(16)}`
          : identifier; // can be "latest" or hex block number
      const method = isHash ? "eth_getBlockByHash" : "eth_getBlockByNumber";
      const raw = await rpc(method, [param, true]);
      if (!raw) return null;
      const b = raw as Block;
      // normalize tx count
      const txArr = Array.isArray(b.transactions) ? b.transactions : [];
      return {
        ...b,
        transactionCount: txArr.length,
      } as Block;
    } catch (e: unknown) {
      console.error("fetchBlock error", e);
      return null;
    }
  }, []);

  // Load recent N blocks starting from head
  const loadRecentBlocks = useCallback(async (count = BLOCKS_PAGE_SIZE) => {
    setBlocksLoading(true);
    setBlocksError("");
    setHasMoreBlocks(true);
    try {
      const headHex = (await rpc("eth_blockNumber")) as string | null;
      if (!headHex) throw new Error("Could not read head block");
      const head = Number.parseInt(headHex, 16);

      const toFetch: number[] = [];
      for (let i = 0; i < count; i++) {
        const n = head - i;
        if (n < 0) break;
        toFetch.push(n);
      }

      const results = await Promise.all(toFetch.map((n) => fetchBlock(n)));
      const valid = results.filter(Boolean) as Block[];
      setBlocks(valid);
      setHasMoreBlocks(head - count > 0);
    } catch (e: unknown) {
      setBlocksError(e instanceof Error ? e.message : "Failed to load blocks");
      setBlocks([]);
      setHasMoreBlocks(false);
    } finally {
      setBlocksLoading(false);
    }
  }, [fetchBlock]);

  // Load older blocks (paginated batches for infinite scroll)
  const loadOlder = useCallback(
    async (count = BLOCKS_PAGE_SIZE) => {
      if (loadingMore || blocksLoading) return;

      if (blocks.length === 0) {
        await loadRecentBlocks(count);
        return;
      }

      const lowest = hexToNumber(blocks[blocks.length - 1].number) ?? 0;
      if (lowest <= 0) {
        setHasMoreBlocks(false);
        return;
      }

      setLoadingMore(true);
      try {
        const toFetch: number[] = [];
        for (let i = 1; i <= count; i++) {
          const n = lowest - i;
          if (n < 0) break;
          toFetch.push(n);
        }

        if (toFetch.length === 0) {
          setHasMoreBlocks(false);
          return;
        }

        const results = await Promise.all(toFetch.map((n) => fetchBlock(n)));
        const older = results.filter(Boolean) as Block[];
        if (older.length) {
          setBlocks((prev) => [...prev, ...older]);
        }
        if (lowest - count <= 0 || older.length < count) {
          setHasMoreBlocks(false);
        }
      } finally {
        setLoadingMore(false);
      }
    },
    [blocks, blocksLoading, fetchBlock, loadRecentBlocks, loadingMore],
  );

  // Select and fully load a block (ensures full tx objects)
  async function selectBlock(b: Block | null) {
    if (!b) {
      closeDrawer();
      return;
    }
    setAddressDetail(null);
    // if we already have full txs, use it
    const needsRefetch = !b.transactions || (b.transactions.length > 0 && typeof b.transactions[0] === "string");
    if (needsRefetch) {
      setSelectedBlockLoading(true);
      const full = await fetchBlock(b.hash, true);
      setSelectedBlock(full ?? b);
      setSelectedBlockLoading(false);
    } else {
      setSelectedBlock(b);
    }
    // clear tx selection
    setSelectedTxHash(null);
    setSelectedTx(null);
    setSelectedReceipt(null);
    const blockNum = hexToNumber(b.number);
    syncUrlParam(blockNum != null ? String(blockNum) : b.hash);
  }

  // Fetch a transaction + receipt
  async function loadTx(hash: string) {
    setAddressDetail(null);
    setTxLoading(true);
    setSelectedTxHash(hash);
    setSelectedTx(null);
    setSelectedReceipt(null);
    setSearchError("");
    syncUrlParam(hash);
    try {
      const [txRaw, receiptRaw] = await Promise.all([
        rpc("eth_getTransactionByHash", [hash]),
        rpc("eth_getTransactionReceipt", [hash]),
      ]);
      setSelectedTx(txRaw as Tx | null);
      setSelectedReceipt(receiptRaw as Receipt | null);

      // If we can, also select/ensure the parent block is loaded in the list/details
      if (txRaw && (txRaw as Tx).blockNumber) {
        const bn = (txRaw as Tx).blockNumber as string;
        // try to find in current blocks
        const existing = blocks.find((bb) => bb.number === bn);
        if (existing) {
          setSelectedBlock(existing);
        } else {
          // fetch it
          const bl = await fetchBlock(bn);
          if (bl) {
            setBlocks((prev) => {
              // insert sorted if possible, else just put at front
              const exists = prev.some((p) => p.number === bl.number);
              return exists ? prev : [bl, ...prev].sort((a, b) => hexToNumber(b.number)! - hexToNumber(a.number)!);
            });
            setSelectedBlock(bl);
          }
        }
      }
    } catch (e: unknown) {
      setSearchError(e instanceof Error ? e.message : "Failed to load transaction");
      setSelectedTxHash(null);
    } finally {
      setTxLoading(false);
    }
  }

  // Generic search dispatcher
  async function performSearch(raw: string) {
    const q = raw.trim();
    if (!q) return;
    setSearchLoading(true);
    setSearchError("");

    try {
      if (isAddress(q)) {
        const balHex = (await rpc("eth_getBalance", [q, "latest"])) as string;
        const titan = formatWeiToTitan(balHex);
        setSelectedBlock(null);
        setSelectedTxHash(null);
        setSelectedTx(null);
        setSelectedReceipt(null);
        setAddressDetail({ address: q, balanceHex: balHex, balanceTitan: titan });
        syncUrlParam(q);
        setSearchValue("");
      } else if (isTxHash(q)) {
        const txRaw = await rpc("eth_getTransactionByHash", [q]);
        if (txRaw) {
          await loadTx(q);
          setSearchValue("");
        } else {
          const b = await fetchBlock(q, true);
          if (b) {
            setBlocks((prev) => {
              const exists = prev.some((bb) => bb.hash === b.hash);
              const next = exists ? prev : [b, ...prev];
              return next.sort((a, bb) => (hexToNumber(bb.number) ?? 0) - (hexToNumber(a.number) ?? 0));
            });
            await selectBlock(b);
            setSearchValue("");
          } else {
            setSearchError("Transaction or block not found");
          }
        }
      } else if (isBlockNumber(q)) {
        const num = q.startsWith("0x") ? Number.parseInt(q, 16) : Number.parseInt(q, 10);
        const b = await fetchBlock(num);
        if (b) {
          setBlocks((prev) => {
            const exists = prev.some((bb) => bb.hash === b.hash);
            const next = exists ? prev : [b, ...prev];
            return next.sort((a, bb) => (hexToNumber(bb.number) ?? 0) - (hexToNumber(a.number) ?? 0));
          });
          await selectBlock(b);
          setSearchValue("");
        } else {
          setSearchError("Block not found");
        }
      } else {
        setSearchError("Unrecognized input. Use block number, 0x-block/tx hash, or 0x-address.");
      }
    } catch (e: unknown) {
      setSearchError(e instanceof Error ? e.message : "Search failed");
    } finally {
      setSearchLoading(false);
    }
  }

  function onSearchKey(e: React.KeyboardEvent<HTMLInputElement>) {
    if (e.key === "Enter") {
      performSearch(searchValue);
    }
  }

  useEffect(() => {
    const q = searchParams.get("q")?.trim();
    if (!q) {
      skipUrlSearchRef.current = false;
      return;
    }
    if (skipUrlSearchRef.current) {
      skipUrlSearchRef.current = false;
      return;
    }
    setSearchValue(q);
    void performSearch(q);
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [searchParams]);

  // Auto-refresh nodes + detect new head and refresh blocks
  useEffect(() => {
    loadNodes();
    const id = setInterval(loadNodes, 10000);
    return () => clearInterval(id);
  }, []);

  // Initial blocks + periodic head check
  useEffect(() => {
    loadRecentBlocks(BLOCKS_PAGE_SIZE);

    const headPoll = setInterval(async () => {
      try {
        const headHex = (await rpc("eth_blockNumber")) as string;
        const head = hexToNumber(headHex);
        if (head == null || blocks.length === 0) return;

        const currentHead = hexToNumber(blocks[0]?.number);
        if (head > (currentHead ?? -1)) {
          await loadRecentBlocks(BLOCKS_PAGE_SIZE);
        }
      } catch {
        /* ignore poll errors */
      }
    }, 8000);

    return () => clearInterval(headPoll);
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  // Infinite scroll: load older blocks when sentinel enters view
  useEffect(() => {
    const sentinel = loadMoreRef.current;
    if (!sentinel || !hasMoreBlocks || loadingMore || blocksLoading) return;

    const observer = new IntersectionObserver(
      (entries) => {
        if (entries[0]?.isIntersecting) {
          void loadOlder(BLOCKS_PAGE_SIZE);
        }
      },
      { rootMargin: "240px" },
    );

    observer.observe(sentinel);
    return () => observer.disconnect();
  }, [blocks.length, blocksLoading, hasMoreBlocks, loadOlder, loadingMore]);

  const headBlock = blocks[0]?.number ? hexToNumber(blocks[0].number)?.toLocaleString() : nodes.find((n) => n.blockNumber)?.blockNumber;
  const chainId = nodes.find((n) => n.chainId)?.chainId ?? "—";
  const gasPrice = nodes.find((n) => n.gasPrice)?.gasPrice ?? "—";
  const highestBlock = blocks[0]?.number ? hexToNumber(blocks[0].number) : null;
  const lowestBlock = blocks[blocks.length - 1]?.number ? hexToNumber(blocks[blocks.length - 1].number) : null;
  const loadedPages = Math.max(1, Math.ceil(blocks.length / BLOCKS_PAGE_SIZE));
  const drawerOpen = Boolean(selectedBlock || selectedTxHash || addressDetail);

  return (
    <div className="flex flex-col gap-5">
      {/* Header */}
      <div className="flex flex-col gap-3 sm:flex-row sm:items-end sm:justify-between">
        <div>
          <h1 className="text-2xl font-bold tracking-tight flex items-center gap-2">
            <Hash className="h-6 w-6" /> Explorer
          </h1>
          <p className="text-sm text-muted-foreground">Browse the Titan C-Chain · live blocks &amp; transactions</p>
        </div>

        <Button
          variant="outline"
          size="sm"
          onClick={() => {
            loadNodes();
            loadRecentBlocks(BLOCKS_PAGE_SIZE);
            closeDrawer();
          }}
          disabled={blocksLoading || nodesLoading}
        >
          {blocksLoading || nodesLoading ? <Loader2 className="h-4 w-4 animate-spin" /> : <RefreshCw className="h-4 w-4" />}
          Refresh
        </Button>
      </div>

      {/* Compact network strip */}
      <div className="flex flex-wrap items-center gap-x-6 gap-y-2 rounded-lg border bg-muted/20 px-4 py-3 text-sm">
        <div className="flex items-center gap-2">
          <Zap className="h-4 w-4 text-muted-foreground" />
          <span className="text-muted-foreground">Head</span>
          <span className="font-mono font-semibold tabular-nums">{headBlock ?? "—"}</span>
        </div>
        <div className="flex items-center gap-2">
          <span className="text-muted-foreground">Chain</span>
          <span className="font-mono font-semibold">{chainId}</span>
        </div>
        <div className="flex items-center gap-2">
          <span className="text-muted-foreground">Gas</span>
          <span className="font-mono text-xs">{gasPrice}</span>
        </div>
        <div className="flex flex-wrap items-center gap-1.5">
          {NODES.map(({ node, port }) => {
            const info = nodes.find((n) => n.node === node);
            const ok = info?.healthy;
            return (
              <Badge key={node} variant={ok ? "default" : "secondary"} className={ok ? "bg-green-600" : ""}>
                {node}:{port}
              </Badge>
            );
          })}
          {nodesLoading && <Loader2 className="h-3 w-3 animate-spin" />}
        </div>
        {lastUpdated && (
          <span className="text-xs text-muted-foreground ml-auto">
            <Clock className="inline h-3 w-3 mr-1" />
            {lastUpdated.toLocaleTimeString()}
          </span>
        )}
      </div>

      {/* Search */}
      <div className="rounded-lg border px-4 py-3">
        <div className="flex flex-col gap-2 sm:flex-row">
          <div className="relative flex-1">
            <Search className="absolute left-3 top-2.5 h-4 w-4 text-muted-foreground" />
            <Input
              value={searchValue}
              onChange={(e) => setSearchValue(e.target.value)}
              onKeyDown={onSearchKey}
              placeholder="Block number, block/tx hash (0x...), or address"
              className="pl-9 font-mono text-sm"
              disabled={searchLoading}
            />
          </div>
          <Button onClick={() => performSearch(searchValue)} disabled={searchLoading || !searchValue.trim()}>
            {searchLoading ? <Loader2 className="h-4 w-4 animate-spin" /> : <Search className="h-4 w-4" />}
            Search
          </Button>
        </div>
        {searchError && <p className="mt-2 text-xs text-amber-600 break-all">{searchError}</p>}
      </div>

      {/* Blocks feed — single column, infinite scroll */}
      <section className="rounded-lg border overflow-hidden">
        <div className="flex flex-wrap items-center justify-between gap-2 border-b bg-muted/30 px-4 py-3">
          <div className="flex items-center gap-2 text-sm font-semibold">
            <Activity className="h-4 w-4" />
            Blocks
          </div>
          <div className="text-xs text-muted-foreground">
            {blocks.length > 0 && highestBlock != null && lowestBlock != null ? (
              <>
                #{highestBlock.toLocaleString()} → #{lowestBlock.toLocaleString()} · {blocks.length} loaded · page {loadedPages}
              </>
            ) : (
              "Loading chain history…"
            )}
          </div>
        </div>

        <table className="w-full text-sm">
          <thead className="bg-muted/40 text-muted-foreground text-xs uppercase tracking-wider border-b">
            <tr>
              <th className="px-4 py-2.5 text-left font-medium w-28">Block</th>
              <th className="px-4 py-2.5 text-left font-medium w-24">Age</th>
              <th className="px-4 py-2.5 text-right font-medium w-16">Txs</th>
              <th className="px-4 py-2.5 text-right font-medium hidden sm:table-cell">Gas Used</th>
              <th className="px-4 py-2.5 text-left font-medium hidden md:table-cell">Hash</th>
            </tr>
          </thead>
          <tbody className="divide-y">
            {blocksLoading && blocks.length === 0 ? (
              <tr>
                <td colSpan={5} className="px-4 py-12 text-center text-muted-foreground">
                  <Loader2 className="mx-auto h-5 w-5 animate-spin" />
                </td>
              </tr>
            ) : blocks.length === 0 ? (
              <tr>
                <td colSpan={5} className="px-4 py-8 text-center text-muted-foreground">
                  {blocksError || "No blocks loaded"}
                </td>
              </tr>
            ) : (
              blocks.map((b) => {
                const num = hexToNumber(b.number);
                const { ago } = formatTimestamp(b.timestamp);
                const isSel = selectedBlock?.hash === b.hash;
                const txCount = b.transactionCount ?? (Array.isArray(b.transactions) ? b.transactions.length : 0);
                return (
                  <tr
                    key={b.hash}
                    onClick={() => selectBlock(b)}
                    className={`cursor-pointer transition-colors hover:bg-muted/50 ${isSel ? "bg-muted/70" : ""}`}
                  >
                    <td className="px-4 py-2.5 font-mono font-semibold tabular-nums">#{num?.toLocaleString()}</td>
                    <td className="px-4 py-2.5 text-muted-foreground text-xs">{ago}</td>
                    <td className="px-4 py-2.5 text-right font-mono tabular-nums">{txCount}</td>
                    <td className="px-4 py-2.5 text-right font-mono text-xs text-muted-foreground hidden sm:table-cell">
                      {hexToNumber(b.gasUsed)?.toLocaleString() ?? "—"}
                    </td>
                    <td className="px-4 py-2.5 font-mono text-xs text-muted-foreground hidden md:table-cell">
                      {shortHash(b.hash)}
                    </td>
                  </tr>
                );
              })
            )}
          </tbody>
        </table>

        <div ref={loadMoreRef} className="border-t bg-muted/20 px-4 py-3 flex flex-wrap items-center justify-between gap-3">
          <div className="text-xs text-muted-foreground">
            {loadingMore ? (
              <span className="inline-flex items-center gap-1.5">
                <Loader2 className="h-3.5 w-3.5 animate-spin" />
                Loading {BLOCKS_PAGE_SIZE} older blocks…
              </span>
            ) : hasMoreBlocks ? (
              "Scroll down to load older blocks"
            ) : (
              "Reached the earliest loaded block"
            )}
          </div>
          <div className="flex items-center gap-2">
            <Button
              variant="outline"
              size="sm"
              onClick={() => loadOlder(BLOCKS_PAGE_SIZE)}
              disabled={!hasMoreBlocks || loadingMore || blocksLoading || blocks.length === 0}
            >
              Load {BLOCKS_PAGE_SIZE} more
            </Button>
            <Button variant="ghost" size="sm" onClick={() => loadRecentBlocks(BLOCKS_PAGE_SIZE)} disabled={blocksLoading}>
              Back to latest
            </Button>
          </div>
        </div>
      </section>

      <ExplorerDetailDrawer
        open={drawerOpen}
        onOpenChange={(open) => {
          if (!open) closeDrawer();
        }}
        block={selectedBlock}
        blockLoading={selectedBlockLoading}
        onSelectBlock={(b) => void selectBlock(b as Block)}
        onLoadTx={(hash) => void loadTx(hash)}
        txHash={selectedTxHash}
        tx={selectedTx}
        receipt={selectedReceipt}
        txLoading={txLoading}
        addressDetail={addressDetail}
        shortHash={shortHash}
        formatTimestamp={formatTimestamp}
        hexToNumber={hexToNumber}
        formatWeiToTitan={formatWeiToTitan}
        formatGwei={formatGwei}
      />

      <p className="text-[10px] text-muted-foreground px-1">
        Data is fetched live from local Titan nodes via C-Chain JSON-RPC. Blocks load in batches of {BLOCKS_PAGE_SIZE}.
      </p>
    </div>
  );
}

export default function ExplorerPage() {
  return (
    <Suspense
      fallback={
        <div className="flex items-center justify-center py-24 text-muted-foreground">
          <Loader2 className="h-6 w-6 animate-spin" />
        </div>
      }
    >
      <ExplorerPageContent />
    </Suspense>
  );
}
