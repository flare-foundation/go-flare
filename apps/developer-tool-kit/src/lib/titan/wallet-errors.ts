function pickMessage(value: unknown): string | null {
  if (typeof value === "string" && value.trim()) {
    return value.trim();
  }
  return null;
}

function digWalletError(error: unknown, depth = 0): string | null {
  if (depth > 4 || error == null) return null;

  if (error instanceof Error) {
    return pickMessage(error.message) ?? digWalletError((error as Error & { cause?: unknown }).cause, depth + 1);
  }

  if (typeof error !== "object") {
    return pickMessage(error);
  }

  const record = error as Record<string, unknown>;

  for (const key of ["shortMessage", "message", "reason", "details"] as const) {
    const msg = pickMessage(record[key]);
    if (msg && !/^Internal JSON-RPC error\.?$/i.test(msg)) {
      return msg;
    }
  }

  if (typeof record.code === "number" && record.code === -32603) {
    const nested =
      digWalletError(record.data, depth + 1) ??
      digWalletError(record.error, depth + 1) ??
      digWalletError(record.cause, depth + 1);
    if (nested) return nested;
    return "RPC rejected the transaction. Ensure MetaMask is on Titan Local UAT, your wallet has TITAN for gas, and nodes are running.";
  }

  if (typeof record.data === "object" && record.data !== null) {
    const nested = digWalletError(record.data, depth + 1);
    if (nested) return nested;
  }

  if (typeof record.cause === "object" && record.cause !== null) {
    const nested = digWalletError(record.cause, depth + 1);
    if (nested) return nested;
  }

  return null;
}

export function parseWalletError(error: unknown, fallback = "Wallet request failed."): string {
  return digWalletError(error) ?? fallback;
}