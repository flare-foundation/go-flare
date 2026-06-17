"use client";

import { useState } from "react";

import { Copy, Loader2 } from "lucide-react";

import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Sheet, SheetContent, SheetDescription, SheetHeader, SheetTitle } from "@/components/ui/sheet";

export interface ExplorerTx {
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
}

export interface ExplorerBlock {
  number: string;
  hash: string;
  parentHash: string;
  timestamp: string;
  gasUsed: string;
  gasLimit: string;
  baseFeePerGas?: string;
  transactions: ExplorerTx[] | string[];
  transactionCount?: number;
}

export interface ExplorerReceipt {
  status: string;
  gasUsed: string;
  effectiveGasPrice?: string;
  logs: unknown[];
  blockNumber: string;
  transactionHash: string;
}

export interface AddressDetail {
  address: string;
  balanceHex: string;
  balanceTitan: string;
}

type ExplorerDetailDrawerProps = {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  block: ExplorerBlock | null;
  blockLoading: boolean;
  onSelectBlock: (block: ExplorerBlock) => void;
  onLoadTx: (hash: string) => void;
  txHash: string | null;
  tx: ExplorerTx | null;
  receipt: ExplorerReceipt | null;
  txLoading: boolean;
  addressDetail: AddressDetail | null;
  shortHash: (h?: string | null, left?: number, right?: number) => string;
  formatTimestamp: (tsHex?: string) => { full: string; ago: string };
  hexToNumber: (hex?: string) => number | null;
  formatWeiToTitan: (hexOrBig?: string | bigint) => string;
  formatGwei: (hex?: string) => string;
};

export function ExplorerDetailDrawer({
  open,
  onOpenChange,
  block,
  blockLoading,
  onSelectBlock,
  onLoadTx,
  txHash,
  tx,
  receipt,
  txLoading,
  addressDetail,
  shortHash,
  formatTimestamp,
  hexToNumber,
  formatWeiToTitan,
  formatGwei,
}: ExplorerDetailDrawerProps) {
  const showTx = Boolean(txHash);
  const showAddress = !showTx && Boolean(addressDetail);
  const showBlock = !showTx && !showAddress && Boolean(block);

  return (
    <Sheet open={open} onOpenChange={onOpenChange}>
      <SheetContent side="right" className="w-full gap-0 overflow-y-auto p-0 sm:max-w-2xl">
        {showTx && (
          <>
            <SheetHeader className="border-b bg-muted/30">
              <SheetTitle>Transaction</SheetTitle>
              <SheetDescription className="font-mono text-xs break-all">{txHash}</SheetDescription>
            </SheetHeader>
            <div className="px-4 py-4">
              {txLoading ? (
                <div className="flex gap-2 py-6 text-sm text-muted-foreground">
                  <Loader2 className="h-4 w-4 animate-spin" /> Loading transaction…
                </div>
              ) : !tx ? (
                <p className="text-sm text-muted-foreground">Transaction not found or still loading.</p>
              ) : (
                <TxDetailBody
                  tx={tx}
                  txHash={txHash!}
                  receipt={receipt}
                  onSelectBlock={onSelectBlock}
                  shortHash={shortHash}
                  hexToNumber={hexToNumber}
                  formatWeiToTitan={formatWeiToTitan}
                  formatGwei={formatGwei}
                />
              )}
            </div>
          </>
        )}

        {showAddress && addressDetail && (
          <>
            <SheetHeader className="border-b bg-muted/30">
              <SheetTitle>Address</SheetTitle>
              <SheetDescription className="font-mono text-xs break-all">{addressDetail.address}</SheetDescription>
            </SheetHeader>
            <div className="space-y-3 px-4 py-4 text-sm">
              <DetailRow label="Balance" value={`${addressDetail.balanceTitan} TITAN`} />
              <DetailRow label="Raw balance" value={addressDetail.balanceHex} mono copyValue={addressDetail.balanceHex} />
            </div>
          </>
        )}

        {showBlock && block && (
          <>
            <SheetHeader className="border-b bg-muted/30">
              <SheetTitle>Block #{hexToNumber(block.number)?.toLocaleString()}</SheetTitle>
              <SheetDescription className="font-mono text-xs break-all">{block.hash}</SheetDescription>
            </SheetHeader>
            <div className="px-4 py-4">
              {blockLoading ? (
                <div className="flex items-center gap-2 py-6 text-sm text-muted-foreground">
                  <Loader2 className="h-4 w-4 animate-spin" /> Loading block…
                </div>
              ) : (
                <BlockDetailBody
                  block={block}
                  selectedTxHash={txHash}
                  onLoadTx={onLoadTx}
                  shortHash={shortHash}
                  formatTimestamp={formatTimestamp}
                  hexToNumber={hexToNumber}
                  formatWeiToTitan={formatWeiToTitan}
                  formatGwei={formatGwei}
                />
              )}
            </div>
          </>
        )}
      </SheetContent>
    </Sheet>
  );
}

function BlockDetailBody({
  block,
  selectedTxHash,
  onLoadTx,
  shortHash,
  formatTimestamp,
  hexToNumber,
  formatWeiToTitan,
  formatGwei,
}: {
  block: ExplorerBlock;
  selectedTxHash: string | null;
  onLoadTx: (hash: string) => void;
  shortHash: (h?: string | null, left?: number, right?: number) => string;
  formatTimestamp: (tsHex?: string) => { full: string; ago: string };
  hexToNumber: (hex?: string) => number | null;
  formatWeiToTitan: (hexOrBig?: string | bigint) => string;
  formatGwei: (hex?: string) => string;
}) {
  const txCount =
    block.transactionCount ?? (Array.isArray(block.transactions) ? block.transactions.length : 0);

  return (
    <div className="space-y-4 text-sm">
      <div className="grid grid-cols-1 gap-x-6 gap-y-1">
        <DetailRow label="Timestamp" value={formatTimestamp(block.timestamp).full} />
        <DetailRow label="Age" value={formatTimestamp(block.timestamp).ago} />
        <DetailRow label="Transactions" value={String(txCount)} />
        <DetailRow
          label="Gas Used / Limit"
          value={`${hexToNumber(block.gasUsed)?.toLocaleString() ?? "—"} / ${hexToNumber(block.gasLimit)?.toLocaleString() ?? "—"}`}
        />
        {block.baseFeePerGas && <DetailRow label="Base Fee" value={formatGwei(block.baseFeePerGas)} />}
        <DetailRow label="Parent Hash" value={shortHash(block.parentHash)} mono copyValue={block.parentHash} />
      </div>

      <div>
        <h3 className="mb-2 text-xs font-medium uppercase tracking-widest text-muted-foreground">
          Transactions ({txCount})
        </h3>
        {Array.isArray(block.transactions) && block.transactions.length > 0 ? (
          <div className="overflow-hidden rounded-md border">
            <table className="w-full text-xs">
              <thead>
                <tr className="bg-muted/60 text-muted-foreground">
                  <th className="px-3 py-1.5 text-left font-normal">#</th>
                  <th className="px-3 py-1.5 text-left font-normal">Hash</th>
                  <th className="px-3 py-1.5 text-right font-normal">Value</th>
                </tr>
              </thead>
              <tbody className="divide-y">
                {block.transactions.map((t, idx) => {
                  const tx = typeof t === "string" ? ({ hash: t } as ExplorerTx) : (t as ExplorerTx);
                  return (
                    <tr
                      key={tx.hash}
                      onClick={() => onLoadTx(tx.hash)}
                      className={`cursor-pointer hover:bg-muted/50 ${selectedTxHash === tx.hash ? "bg-muted" : ""}`}
                    >
                      <td className="px-3 py-1.5 font-mono tabular-nums text-muted-foreground">{idx}</td>
                      <td className="px-3 py-1.5 font-mono text-primary">{shortHash(tx.hash)}</td>
                      <td className="px-3 py-1.5 text-right font-medium tabular-nums">
                        {formatWeiToTitan(tx.value)}
                      </td>
                    </tr>
                  );
                })}
              </tbody>
            </table>
          </div>
        ) : (
          <p className="text-xs text-muted-foreground py-2">No transactions in this block.</p>
        )}
      </div>
    </div>
  );
}

function TxDetailBody({
  tx,
  txHash,
  receipt,
  onSelectBlock,
  shortHash,
  hexToNumber,
  formatWeiToTitan,
  formatGwei,
}: {
  tx: ExplorerTx;
  txHash: string;
  receipt: ExplorerReceipt | null;
  onSelectBlock: (block: ExplorerBlock) => void;
  shortHash: (h?: string | null, left?: number, right?: number) => string;
  hexToNumber: (hex?: string) => number | null;
  formatWeiToTitan: (hexOrBig?: string | bigint) => string;
  formatGwei: (hex?: string) => string;
}) {
  return (
    <div className="space-y-3 text-sm">
      <div className="grid grid-cols-1 gap-x-6 gap-y-1">
        <DetailRow label="From" value={shortHash(tx.from)} mono copyValue={tx.from} />
        <DetailRow
          label="To"
          value={tx.to ? shortHash(tx.to) : "Contract creation"}
          mono
          copyValue={tx.to ?? undefined}
        />
        <DetailRow label="Value" value={`${formatWeiToTitan(tx.value)} TITAN`} />
        <DetailRow label="Gas Price" value={tx.gasPrice ? formatGwei(tx.gasPrice) : "—"} />
        <DetailRow
          label="Gas Limit / Used"
          value={`${hexToNumber(tx.gas)?.toLocaleString() ?? "—"}${receipt ? ` / ${hexToNumber(receipt.gasUsed)?.toLocaleString()}` : ""}`}
        />
        {receipt && (
          <DetailRow
            label="Status"
            value={
              <span
                className={
                  receipt.status === "0x1" || receipt.status === "0x01"
                    ? "text-green-600 font-medium"
                    : "text-red-600 font-medium"
                }
              >
                {receipt.status === "0x1" || receipt.status === "0x01" ? "Success" : "Failed"}
              </span>
            }
          />
        )}
        <DetailRow label="Nonce" value={String(hexToNumber(tx.nonce) ?? tx.nonce)} />
        {receipt?.effectiveGasPrice && (
          <DetailRow label="Effective Gas Price" value={formatGwei(receipt.effectiveGasPrice)} />
        )}
      </div>

      {receipt && (
        <div>
          <div className="mb-1 text-[10px] uppercase tracking-widest text-muted-foreground">Logs emitted</div>
          <Badge variant="secondary">
            {receipt.logs?.length ?? 0} log{receipt.logs?.length === 1 ? "" : "s"}
          </Badge>
        </div>
      )}

      <div>
        <div className="mb-1 text-[10px] uppercase tracking-widest text-muted-foreground">Input data</div>
        <pre className="max-h-40 overflow-auto rounded bg-muted p-2 font-mono text-[10px] break-all">
          {tx.input && tx.input !== "0x" ? tx.input : "(empty)"}
        </pre>
      </div>

      <div className="flex flex-wrap gap-2">
        <Button size="sm" variant="outline" onClick={() => navigator.clipboard.writeText(txHash)}>
          <Copy className="mr-1.5 h-3.5 w-3.5" /> Copy tx hash
        </Button>
        {tx.blockHash && (
          <Button size="sm" variant="outline" onClick={() => onSelectBlock({ hash: tx.blockHash } as ExplorerBlock)}>
            View containing block
          </Button>
        )}
      </div>
    </div>
  );
}

function DetailRow({
  label,
  value,
  mono,
  copyValue,
}: {
  label: string;
  value: React.ReactNode;
  mono?: boolean;
  copyValue?: string;
}) {
  const [copied, setCopied] = useState(false);

  async function doCopy() {
    if (!copyValue) return;
    await navigator.clipboard.writeText(copyValue);
    setCopied(true);
    setTimeout(() => setCopied(false), 1200);
  }

  return (
    <div className="flex justify-between gap-3 border-b border-dashed border-border/60 py-1 last:border-none">
      <span className="shrink-0 text-muted-foreground">{label}</span>
      <span
        className={`${mono ? "font-mono text-xs break-all" : "font-medium break-all"} flex items-center gap-1.5 text-right`}
      >
        {value}
        {copyValue && (
          <button type="button" onClick={doCopy} className="text-muted-foreground hover:text-foreground" title="Copy">
            <Copy className="h-3 w-3" />
          </button>
        )}
        {copied && <span className="text-[10px] text-emerald-600">copied</span>}
      </span>
    </div>
  );
}