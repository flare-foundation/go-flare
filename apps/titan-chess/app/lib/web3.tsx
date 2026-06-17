'use client';

import { RainbowKitProvider, getDefaultConfig, darkTheme } from '@rainbow-me/rainbowkit';
import '@rainbow-me/rainbowkit/styles.css';
import { WagmiProvider, http } from 'wagmi';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import type { Chain } from 'wagmi/chains';
import { TITAN_NETWORK } from './titan-config';

export const titanSubnet: Chain = {
  id: TITAN_NETWORK.chainId,
  name: TITAN_NETWORK.name,
  nativeCurrency: TITAN_NETWORK.nativeCurrency,
  rpcUrls: {
    default: { http: [TITAN_NETWORK.rpcUrl] },
    public: { http: [TITAN_NETWORK.rpcUrl] },
  },
  blockExplorers: {
    default: {
      name: 'Titan Explorer',
      url: TITAN_NETWORK.explorerUrl,
    },
  },
};

const config = getDefaultConfig({
  appName: 'Titan Chess',
  projectId: process.env.NEXT_PUBLIC_WALLETCONNECT_PROJECT_ID || 'titan-chess-dev',
  chains: [titanSubnet],
  transports: {
    [titanSubnet.id]: http(TITAN_NETWORK.rpcUrl),
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
        <RainbowKitProvider theme={rainbowKitTheme} coolMode initialChain={titanSubnet}>
          {children}
        </RainbowKitProvider>
      </QueryClientProvider>
    </WagmiProvider>
  );
}