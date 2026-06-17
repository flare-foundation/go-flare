'use client';

import { motion, AnimatePresence } from 'framer-motion';
import type { GameState } from '@/types/chess';

interface StatusBarProps {
  gameState: GameState;
  isAiThinking: boolean;
  playerColor: 'w' | 'b';
}

export function StatusBar({ gameState, isAiThinking, playerColor }: StatusBarProps) {
  const { turn, isCheck, isCheckmate, isDraw, isGameOver } = gameState;

  const getStatusText = () => {
    if (isCheckmate) {
      const winner = turn === 'w' ? 'Black' : 'White';
      return `${winner} wins by checkmate!`;
    }
    if (isDraw) return 'Game drawn';
    if (isCheck) return `${turn === 'w' ? 'White' : 'Black'} is in check!`;
    if (isAiThinking) return 'AI is thinking...';
    if (turn === playerColor) return 'Your move';
    return `${turn === 'w' ? 'White' : 'Black'} to move`;
  };

  const getStatusColor = () => {
    if (isCheckmate) return '#e8c97a';
    if (isCheck) return '#ff6b6b';
    if (isAiThinking) return 'var(--text-secondary)';
    if (turn === playerColor) return 'var(--gold-secondary)';
    return 'var(--text-secondary)';
  };

  return (
    <div className="flex items-center gap-3">
      {/* Turn indicator dot */}
      <AnimatePresence mode="wait">
        {isAiThinking ? (
          <motion.div
            key="thinking"
            className="flex gap-1"
          >
            {[0, 1, 2].map((i) => (
              <motion.div
                key={i}
                className="w-1.5 h-1.5 rounded-full"
                style={{ background: 'var(--text-secondary)' }}
                animate={{ opacity: [0.3, 1, 0.3] }}
                transition={{ duration: 1.2, repeat: Infinity, delay: i * 0.2 }}
              />
            ))}
          </motion.div>
        ) : (
          <motion.div
            key="dot"
            className="w-2 h-2 rounded-full"
            style={{
              background: isCheck || isCheckmate ? '#ff6b6b' : 'var(--gold-primary)',
              boxShadow: `0 0 8px ${isCheck || isCheckmate ? '#ff6b6b' : 'var(--gold-primary)'}`,
            }}
            animate={
              isCheck
                ? { scale: [1, 1.3, 1], opacity: [1, 0.6, 1] }
                : { scale: 1, opacity: 1 }
            }
            transition={{ duration: 0.8, repeat: isCheck ? Infinity : 0 }}
          />
        )}
      </AnimatePresence>

      <AnimatePresence mode="wait">
        <motion.span
          key={getStatusText()}
          initial={{ opacity: 0, y: -4 }}
          animate={{ opacity: 1, y: 0 }}
          exit={{ opacity: 0, y: 4 }}
          transition={{ duration: 0.2 }}
          className="text-sm font-medium"
          style={{ color: getStatusColor() }}
        >
          {getStatusText()}
        </motion.span>
      </AnimatePresence>
    </div>
  );
}
