/** Local Titan L1 defaults — aligned with apps/developer-tool-kit and titan-network/origin.json */
export const TITAN_NETWORK = {
  name: process.env.NEXT_PUBLIC_TITAN_NETWORK_NAME ?? 'Titan Local UAT',
  chainId: Number.parseInt(process.env.NEXT_PUBLIC_TITAN_CHAIN_ID ?? '781337', 10),
  chainIdHex: process.env.NEXT_PUBLIC_TITAN_CHAIN_ID_HEX ?? '0xbec19',
  rpcUrl: process.env.NEXT_PUBLIC_TITAN_RPC_URL ?? 'http://localhost:9650/ext/bc/C/rpc',
  explorerUrl:
    process.env.NEXT_PUBLIC_TITAN_EXPLORER_URL ?? 'http://localhost:3000/dashboard/activity',
  nativeCurrency: {
    decimals: 18,
    name: 'Titan',
    symbol: 'TITAN',
  },
} as const;