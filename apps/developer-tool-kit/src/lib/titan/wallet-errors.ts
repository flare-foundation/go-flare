export function parseWalletError(error: unknown, fallback = "Wallet request failed."): string {
  if (error instanceof Error && error.message) {
    return error.message;
  }

  if (typeof error === "string" && error.trim()) {
    return error;
  }

  if (typeof error === "object" && error !== null) {
    const record = error as Record<string, unknown>;

    if (typeof record.message === "string" && record.message.trim()) {
      return record.message;
    }

    if (typeof record.shortMessage === "string" && record.shortMessage.trim()) {
      return record.shortMessage;
    }

    if (typeof record.code === "number" && record.code === -32603) {
      return "RPC node rejected the request. If deploying, recompile the contract and try again.";
    }

    const data = record.data;
    if (typeof data === "object" && data !== null) {
      const dataRecord = data as Record<string, unknown>;
      if (typeof dataRecord.message === "string" && dataRecord.message.trim()) {
        return dataRecord.message;
      }
    }
  }

  return fallback;
}