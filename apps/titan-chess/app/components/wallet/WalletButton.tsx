'use client';

import { ConnectButton } from '@rainbow-me/rainbowkit';
import { motion } from 'framer-motion';

export function WalletButton() {
  return (
    <ConnectButton.Custom>
      {({
        account,
        chain,
        openAccountModal,
        openChainModal,
        openConnectModal,
        authenticationStatus,
        mounted,
      }) => {
        const ready = mounted && authenticationStatus !== 'loading';
        const connected =
          ready &&
          account &&
          chain &&
          (!authenticationStatus || authenticationStatus === 'authenticated');

        return (
          <div
            {...(!ready && {
              'aria-hidden': true,
              style: { opacity: 0, pointerEvents: 'none', userSelect: 'none' },
            })}
          >
            {!connected ? (
              <motion.button
                onClick={openConnectModal}
                whileHover={{ scale: 1.02 }}
                whileTap={{ scale: 0.98 }}
                className="relative px-4 py-2 rounded-lg text-sm font-semibold overflow-hidden"
                style={{
                  background: 'linear-gradient(135deg, #c9a84c, #e8c97a)',
                  color: '#0f0f11',
                }}
              >
                <span className="relative z-10">Connect Wallet</span>
                <motion.div
                  className="absolute inset-0 opacity-0"
                  style={{ background: 'linear-gradient(135deg, #e8c97a, #c9a84c)' }}
                  whileHover={{ opacity: 1 }}
                  transition={{ duration: 0.2 }}
                />
              </motion.button>
            ) : chain.unsupported ? (
              <motion.button
                onClick={openChainModal}
                whileHover={{ scale: 1.02 }}
                className="px-4 py-2 rounded-lg text-sm font-semibold"
                style={{ background: 'rgba(220,50,50,0.2)', color: '#ff6b6b', border: '1px solid rgba(220,50,50,0.4)' }}
              >
                Wrong Network
              </motion.button>
            ) : (
              <div className="flex items-center gap-2">
                <motion.button
                  onClick={openChainModal}
                  whileHover={{ scale: 1.02 }}
                  className="flex items-center gap-1.5 px-3 py-1.5 rounded-lg text-xs"
                  style={{
                    background: 'var(--bg-glass)',
                    border: '1px solid var(--bg-glass-border)',
                    color: 'var(--text-secondary)',
                  }}
                >
                  {chain.hasIcon && chain.iconUrl && (
                    <img
                      alt={chain.name ?? 'Chain icon'}
                      src={chain.iconUrl}
                      className="w-3.5 h-3.5 rounded-full"
                    />
                  )}
                  <span>{chain.name}</span>
                </motion.button>
                <motion.button
                  onClick={openAccountModal}
                  whileHover={{ scale: 1.02 }}
                  className="flex items-center gap-2 px-3 py-1.5 rounded-lg text-xs font-mono"
                  style={{
                    background: 'var(--bg-glass)',
                    border: '1px solid var(--gold-primary)',
                    color: 'var(--gold-secondary)',
                  }}
                >
                  <div
                    className="w-2 h-2 rounded-full"
                    style={{ background: 'var(--gold-primary)' }}
                  />
                  {account.displayName}
                  {account.displayBalance ? ` · ${account.displayBalance}` : ''}
                </motion.button>
              </div>
            )}
          </div>
        );
      }}
    </ConnectButton.Custom>
  );
}
