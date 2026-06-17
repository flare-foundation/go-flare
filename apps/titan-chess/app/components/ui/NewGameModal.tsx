'use client';

import { useState } from 'react';
import { motion, AnimatePresence } from 'framer-motion';
import type { WaitingPlayer } from '@/hooks/useMatchmaking';

interface NewGameModalProps {
  open: boolean;
  onClose: () => void;
  onPlayStockfish: (stake: string) => void;
  onPlayHuman: (stake: string, opponentAddress?: string) => void;
  waitingPlayers: WaitingPlayer[];
  stakeBounds: { min: string; max: string };
  escrowEnabled: boolean;
  isConnected: boolean;
  isLoading: boolean;
  error: string | null;
}

const PRESET_STAKES = ['0.01', '0.05', '0.1', '0.25', '0.5', '1'];

export function NewGameModal({
  open,
  onClose,
  onPlayStockfish,
  onPlayHuman,
  waitingPlayers,
  stakeBounds,
  escrowEnabled,
  isConnected,
  isLoading,
  error,
}: NewGameModalProps) {
  const [stake, setStake] = useState('0.1');
  const [customStake, setCustomStake] = useState('');

  const effectiveStake = customStake || stake;
  const pot = (Number(effectiveStake) * 2).toFixed(4);

  const matchingOpponent = waitingPlayers.find((p) => p.stake === effectiveStake);
  const otherWaiting = waitingPlayers.filter((p) => p.stake !== effectiveStake);

  return (
    <AnimatePresence>
      {open && (
        <motion.div
          initial={{ opacity: 0 }}
          animate={{ opacity: 1 }}
          exit={{ opacity: 0 }}
          className="fixed inset-0 z-50 flex items-center justify-center p-4"
          style={{ background: 'rgba(0,0,0,0.75)', backdropFilter: 'blur(6px)' }}
          onClick={onClose}
        >
          <motion.div
            initial={{ scale: 0.92, opacity: 0, y: 16 }}
            animate={{ scale: 1, opacity: 1, y: 0 }}
            exit={{ scale: 0.95, opacity: 0 }}
            transition={{ type: 'spring', stiffness: 320, damping: 28 }}
            className="glass rounded-2xl p-6 w-full max-w-md"
            style={{ borderColor: 'rgba(201,168,76,0.25)' }}
            onClick={(e) => e.stopPropagation()}
          >
            <h2
              className="text-xl font-bold mb-1"
              style={{
                background: 'linear-gradient(135deg, var(--gold-primary), var(--gold-secondary))',
                WebkitBackgroundClip: 'text',
                WebkitTextFillColor: 'transparent',
              }}
            >
              New Wagered Game
            </h2>
            <p className="text-sm mb-5" style={{ color: 'var(--text-secondary)' }}>
              Choose your opponent and stake TITAN to play for the pot.
            </p>

            {!isConnected && (
              <div
                className="mb-4 px-3 py-2 rounded-lg text-sm"
                style={{ background: 'rgba(255,107,107,0.1)', color: '#ff8a8a' }}
              >
                Connect your wallet to place a wager.
              </div>
            )}

            {error && (
              <div
                className="mb-4 px-3 py-2 rounded-lg text-sm"
                style={{ background: 'rgba(255,107,107,0.1)', color: '#ff8a8a' }}
              >
                {error}
              </div>
            )}

            {/* Stake picker */}
            <div className="mb-5">
              <label className="text-xs uppercase tracking-widest mb-2 block" style={{ color: 'var(--text-secondary)' }}>
                Stake (TITAN)
              </label>
              <div className="flex flex-wrap gap-2 mb-2">
                {PRESET_STAKES.filter(
                  (s) => Number(s) >= Number(stakeBounds.min) && Number(s) <= Number(stakeBounds.max)
                ).map((s) => (
                  <button
                    key={s}
                    onClick={() => {
                      setStake(s);
                      setCustomStake('');
                    }}
                    className="px-3 py-1.5 rounded-lg text-sm font-medium transition-colors"
                    style={{
                      background: !customStake && stake === s ? 'var(--gold-dim)' : 'var(--bg-glass)',
                      border: `1px solid ${!customStake && stake === s ? 'rgba(201,168,76,0.4)' : 'var(--bg-glass-border)'}`,
                      color: !customStake && stake === s ? 'var(--gold-secondary)' : 'var(--text-secondary)',
                    }}
                  >
                    {s}
                  </button>
                ))}
              </div>
              <input
                type="number"
                min={stakeBounds.min}
                max={stakeBounds.max}
                step="0.01"
                placeholder={`Custom (${stakeBounds.min}–${stakeBounds.max})`}
                value={customStake}
                onChange={(e) => setCustomStake(e.target.value)}
                className="w-full px-3 py-2 rounded-lg text-sm"
                style={{
                  background: 'var(--bg-secondary)',
                  border: '1px solid var(--bg-glass-border)',
                  color: 'var(--text-primary)',
                }}
              />
              <p className="text-xs mt-2" style={{ color: 'var(--text-secondary)' }}>
                Pot: <span style={{ color: 'var(--gold-secondary)' }}>{pot} TITAN</span>
                {!escrowEnabled && ' · Practice mode (escrow not deployed)'}
              </p>
            </div>

            {/* Opponent options */}
            <div className="space-y-3 mb-5">
              <button
                disabled={!isConnected || isLoading}
                onClick={() => onPlayStockfish(effectiveStake)}
                className="w-full p-4 rounded-xl text-left transition-opacity disabled:opacity-40"
                style={{
                  background: 'var(--gold-dim)',
                  border: '1px solid rgba(201,168,76,0.35)',
                }}
              >
                <div className="flex items-center justify-between">
                  <div>
                    <div className="font-semibold text-sm" style={{ color: 'var(--gold-secondary)' }}>
                      vs Stockfish
                    </div>
                    <div className="text-xs mt-0.5" style={{ color: 'var(--text-secondary)' }}>
                      Bet against the house AI · {escrowEnabled ? 'On-chain escrow' : 'Instant play'}
                    </div>
                  </div>
                  <span className="text-2xl">🤖</span>
                </div>
              </button>

              {matchingOpponent && (
                <button
                  disabled={!isConnected || isLoading}
                  onClick={() => onPlayHuman(effectiveStake, matchingOpponent.address)}
                  className="w-full p-4 rounded-xl text-left transition-opacity disabled:opacity-40"
                  style={{
                    background: 'rgba(34,197,94,0.08)',
                    border: '1px solid rgba(34,197,94,0.3)',
                  }}
                >
                  <div className="flex items-center justify-between">
                    <div>
                      <div className="font-semibold text-sm" style={{ color: '#4ade80' }}>
                        vs {matchingOpponent.shortAddress}
                      </div>
                      <div className="text-xs mt-0.5" style={{ color: 'var(--text-secondary)' }}>
                        Player waiting · {matchingOpponent.stake} TITAN stake · Instant match
                      </div>
                    </div>
                    <span className="text-2xl">👤</span>
                  </div>
                </button>
              )}

              <button
                disabled={!isConnected || isLoading}
                onClick={() => onPlayHuman(effectiveStake)}
                className="w-full p-4 rounded-xl text-left transition-opacity disabled:opacity-40"
                style={{
                  background: 'var(--bg-glass)',
                  border: '1px solid var(--bg-glass-border)',
                }}
              >
                <div className="flex items-center justify-between">
                  <div>
                    <div className="font-semibold text-sm" style={{ color: 'var(--text-primary)' }}>
                      vs Player from Queue
                    </div>
                    <div className="text-xs mt-0.5" style={{ color: 'var(--text-secondary)' }}>
                      {waitingPlayers.length > 0
                        ? `${waitingPlayers.length} player${waitingPlayers.length > 1 ? 's' : ''} waiting · Join queue at ${effectiveStake} TITAN`
                        : 'No players waiting — join queue and wait for a match'}
                    </div>
                  </div>
                  <span className="text-2xl">🎯</span>
                </div>
              </button>

              {otherWaiting.length > 0 && !matchingOpponent && (
                <p className="text-xs px-1" style={{ color: 'var(--text-secondary)' }}>
                  Waiting at other stakes:{' '}
                  {otherWaiting.map((p) => `${p.shortAddress} (${p.stake})`).join(', ')}
                </p>
              )}
            </div>

            <button
              onClick={onClose}
              className="w-full py-2.5 rounded-xl text-sm"
              style={{
                background: 'var(--bg-glass)',
                border: '1px solid var(--bg-glass-border)',
                color: 'var(--text-secondary)',
              }}
            >
              Cancel
            </button>
          </motion.div>
        </motion.div>
      )}
    </AnimatePresence>
  );
}