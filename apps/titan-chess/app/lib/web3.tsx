'use client';

import { RainbowKitProvider, getDefaultConfig, darkTheme } from '@rainbow-me/rainbowkit';
import '@rainbow-me/rainbowkit/styles.css';
import { WagmiProvider, http } from 'wagmi';
import { avalanche, avalancheFuji } from 'wagmi/chains';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import type { Chain } from 'wagmi/chains';

// Titan Chess custom Avalanche L1 Subnet — update RPC_URL when available
const TITAN_SUBNET_RPC = process.env.NEXT_PUBLIC_TITAN_RPC_URL || 'https://placeholder-rpc.titanchess.io';
const TITAN_CHAIN_ID = parseInt(process.env.NEXT_PUBLIC_TITAN_CHAIN_ID || '66666');

export const titanSubnet: Chain = {
  id: TITAN_CHAIN_ID,
  name: 'Titan Chess L1',
  nativeCurrency: {
    decimals: 18,
    name: 'TITAN',
    symbol: 'TITAN',
  },
  rpcUrls: {
    default: { http: [TITAN_SUBNET_RPC] },
    public: { http: [TITAN_SUBNET_RPC] },
  },
  blockExplorers: {
    default: {
      name: 'Titan Explorer',
      url: 'https://explorer.titanchess.io',
    },
  },
};

const config = getDefaultConfig({
  appName: 'Titan Chess',
  projectId: process.env.NEXT_PUBLIC_WALLETCONNECT_PROJECT_ID || 'titan-chess-dev',
  chains: [titanSubnet, avalanche, avalancheFuji],
  transports: {
    [titanSubnet.id]: http(TITAN_SUBNET_RPC),
    [avalanche.id]: http(),
    [avalancheFuji.id]: http(),
  },
  ssr: true,
});

const queryClient = new QueryClient();

const rainbowKitTheme = darkTheme({
  accentColor: '#c9a84c',
  accentColorForeground: '#0f0f11',
  borderRadius: 'medium',
  fontStack: 'system',
  overlayBlur: 'small',
});

export function Web3Provider({ children }: { children: React.ReactNode }) {
  return (
    <WagmiProvider config={config}>
      <QueryClientProvider client={queryClient}>
        <RainbowKitProvider theme={rainbowKitTheme} coolMode>
          {children}
        </RainbowKitProvider>
      </QueryClientProvider>
    </WagmiProvider>
  );
}
