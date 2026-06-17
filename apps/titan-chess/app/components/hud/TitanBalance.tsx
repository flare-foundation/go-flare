'use client';

import { motion } from 'framer-motion';
import { useAccount, useBalance } from 'wagmi';
import { formatEther } from 'viem';
import { titanSubnet } from '@/lib/web3';

function useTitanBalance() {
  const { address, isConnected } = useAccount();
  const { data, isLoading, isError } = useBalance({
    address,
    chainId: titanSubnet.id,
    query: { enabled: isConnected && !!address },
  });

  const balance =
    data != null
      ? Number(formatEther(data.value)).toLocaleString('en-US', { maximumFractionDigits: 4 })
      : null;

  return { balance, isConnected, isLoading, isError };
}

export function TitanBalance() {
  const { balance, isConnected, isLoading, isError } = useTitanBalance();

  return (
    <div
      className="glass rounded-xl p-4"
      style={{ borderColor: 'rgba(201, 168, 76, 0.2)' }}
    >
      <div className="flex items-center justify-between mb-1">
        <span className="text-xs uppercase tracking-widest" style={{ color: 'var(--text-secondary)' }}>
          TITAN Balance
        </span>
        <div
          className="w-2 h-2 rounded-full"
          style={{
            background: isConnected ? 'var(--gold-primary)' : 'var(--text-secondary)',
            boxShadow: isConnected ? '0 0 6px var(--gold-primary)' : 'none',
          }}
        />
      </div>

      {isConnected && balance != null ? (
        <motion.div
          initial={{ opacity: 0, y: 4 }}
          animate={{ opacity: 1, y: 0 }}
          transition={{ duration: 0.4 }}
        >
          <div className="flex items-baseline gap-1">
            <span
              className="text-2xl font-bold tabular-nums"
              style={{
                background: 'linear-gradient(135deg, var(--gold-primary), var(--gold-secondary))',
                WebkitBackgroundClip: 'text',
                WebkitTextFillColor: 'transparent',
              }}
            >
              {balance}
            </span>
            <span className="text-xs font-medium" style={{ color: 'var(--bronze)' }}>
              TITAN
            </span>
          </div>
          <div className="mt-2 flex items-center gap-1.5">
            <svg viewBox="0 0 10 10" className="w-2.5 h-2.5">
              <circle cx="5" cy="5" r="4" fill="none" stroke="currentColor" strokeWidth="1.5"
                style={{ color: '#22c55e' }} />
              <path d="M3 5.5l1.5 1.5L7 4" stroke="currentColor" strokeWidth="1" fill="none"
                strokeLinecap="round" strokeLinejoin="round" style={{ color: '#22c55e' }} />
            </svg>
            <span className="text-xs" style={{ color: 'var(--text-secondary)' }}>
              Live on Titan Local UAT
            </span>
          </div>
        </motion.div>
      ) : (
        <div className="mt-1">
          <span className="text-sm" style={{ color: 'var(--text-secondary)' }}>
            {!isConnected
              ? 'Connect wallet to view'
              : isLoading
                ? 'Fetching...'
                : isError
                  ? 'Unable to load balance — is node1 running?'
                  : 'Fetching...'}
          </span>
        </div>
      )}
    </div>
  );
}