export function shortAddress(address: string, left = 6, right = 4): string {
  if (!address) return "";
  if (address.length <= left + right + 2) return address;
  return `${address.slice(0, left)}…${address.slice(-right)}`;
}

export function formatWeiToTitan(hexOrBig?: string | bigint): string {
  if (hexOrBig == null) return "0";
  try {
    const wei = typeof hexOrBig === "string" ? BigInt(hexOrBig) : hexOrBig;
    const titan = Number(wei) / 1e18;
    if (Math.abs(titan) < 0.0001 && titan !== 0) return titan.toExponential(2);
    return titan.toLocaleString(undefined, { maximumFractionDigits: 4 });
  } catch {
    return "0";
  }
}