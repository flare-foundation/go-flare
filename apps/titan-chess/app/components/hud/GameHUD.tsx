'use client';

import { motion } from 'framer-motion';
import { TitanBalance } from './TitanBalance';
import { DifficultySlider } from './DifficultySlider';
import { MoveList } from './MoveList';
import { StatusBar } from './StatusBar';
import { WalletButton } from '@/components/wallet/WalletButton';
import type { GameState } from '@/types/chess';

interface GameHUDProps {
  gameState: GameState;
  isAiThinking: boolean;
  stockfishDepth: number;
  playerColor: 'w' | 'b';
  onDifficultyChange: (v: number) => void;
  onReset: () => void;
  onFlipBoard: () => void;
}

export function GameHUD({
  gameState,
  isAiThinking,
  stockfishDepth,
  playerColor,
  onDifficultyChange,
  onReset,
  onFlipBoard,
}: GameHUDProps) {
  return (
    <motion.aside
      initial={{ opacity: 0, x: 20 }}
      animate={{ opacity: 1, x: 0 }}
      transition={{ duration: 0.5, delay: 0.2 }}
      className="flex flex-col gap-3 w-full lg:w-72 xl:w-80"
    >
      {/* Wallet */}
      <div
        className="glass rounded-xl px-4 py-3 flex items-center justify-between"
        style={{ borderColor: 'rgba(201,168,76,0.15)' }}
      >
        <div>
          <div className="text-xs uppercase tracking-widest mb-0.5" style={{ color: 'var(--text-secondary)' }}>
            Titan Chess
          </div>
          <div className="text-xs" style={{ color: 'var(--text-secondary)' }}>
            Avalanche L1
          </div>
        </div>
        <WalletButton />
      </div>

      {/* TITAN Balance */}
      <TitanBalance />

      {/* Status */}
      <div className="glass rounded-xl px-4 py-3">
        <StatusBar
          gameState={gameState}
          isAiThinking={isAiThinking}
          playerColor={playerColor}
        />
      </div>

      {/* AI Difficulty */}
      <DifficultySlider
        value={stockfishDepth}
        onChange={onDifficultyChange}
        disabled={isAiThinking}
      />

      {/* Move list */}
      <MoveList moves={gameState.moveHistory} />

      {/* Actions */}
      <div className="flex gap-2">
        <motion.button
          whileHover={{ scale: 1.02 }}
          whileTap={{ scale: 0.97 }}
          onClick={onFlipBoard}
          className="flex-1 py-2.5 rounded-xl text-sm font-medium"
          style={{
            background: 'var(--bg-glass)',
            border: '1px solid var(--bg-glass-border)',
            color: 'var(--text-secondary)',
          }}
        >
          Flip Board
        </motion.button>
        <motion.button
          whileHover={{ scale: 1.02 }}
          whileTap={{ scale: 0.97 }}
          onClick={onReset}
          className="flex-1 py-2.5 rounded-xl text-sm font-semibold"
          style={{
            background: 'var(--gold-dim)',
            border: '1px solid rgba(201,168,76,0.3)',
            color: 'var(--gold-secondary)',
          }}
        >
          New Game
        </motion.button>
      </div>

      {/* Footer */}
      <div className="text-center">
        <span className="text-xs" style={{ color: 'var(--text-secondary)', opacity: 0.5 }}>
          Powered by Stockfish · chess.js
        </span>
      </div>
    </motion.aside>
  );
}
