'use client';

import { useState, useCallback } from 'react';
import { motion, AnimatePresence } from 'framer-motion';
import { ChessBoard } from '@/components/board/ChessBoard';
import { GameHUD } from '@/components/hud/GameHUD';
import { GameOverOverlay } from '@/components/ui/GameOverOverlay';
import { useChessGame } from '@/hooks/useChessGame';
import type { Color } from 'chess.js';

export function GamePage() {
  const {
    gameState,
    selectedSquare,
    legalMoves,
    stockfishDepth,
    isAiThinking,
    playerColor,
    selectSquare,
    resetGame,
    updateDifficulty,
    setPlayerColor,
  } = useChessGame();

  const [boardFlipped, setBoardFlipped] = useState(false);

  const handleFlipBoard = useCallback(() => {
    setBoardFlipped((f) => !f);
  }, []);

  const effectiveColor: Color = boardFlipped
    ? playerColor === 'w' ? 'b' : 'w'
    : playerColor;

  return (
    <div
      className="min-h-screen flex flex-col"
      style={{
        background:
          'radial-gradient(ellipse at top, #1a1a22 0%, #0f0f11 60%)',
      }}
    >
      {/* Header */}
      <motion.header
        initial={{ opacity: 0, y: -10 }}
        animate={{ opacity: 1, y: 0 }}
        transition={{ duration: 0.5 }}
        className="flex items-center justify-between px-6 py-4 border-b"
        style={{ borderColor: 'var(--bg-glass-border)' }}
      >
        {/* Logo */}
        <div className="flex items-center gap-3">
          <div
            className="w-8 h-8 rounded-lg flex items-center justify-center text-lg font-bold"
            style={{
              background: 'linear-gradient(135deg, var(--gold-primary), var(--bronze))',
              color: '#0f0f11',
            }}
          >
            ♟
          </div>
          <div>
            <h1
              className="text-lg font-bold tracking-tight leading-none"
              style={{
                background: 'linear-gradient(135deg, var(--gold-primary), var(--gold-secondary))',
                WebkitBackgroundClip: 'text',
                WebkitTextFillColor: 'transparent',
              }}
            >
              TITAN CHESS
            </h1>
            <p className="text-xs leading-none" style={{ color: 'var(--text-secondary)' }}>
              Decentralized · On-Chain · Avalanche L1
            </p>
          </div>
        </div>

        {/* Network badge */}
        <div
          className="hidden sm:flex items-center gap-2 px-3 py-1.5 rounded-full text-xs"
          style={{
            background: 'var(--gold-dim)',
            border: '1px solid rgba(201,168,76,0.2)',
            color: 'var(--gold-secondary)',
          }}
        >
          <div
            className="w-1.5 h-1.5 rounded-full"
            style={{
              background: 'var(--gold-primary)',
              boxShadow: '0 0 4px var(--gold-primary)',
            }}
          />
          Titan Subnet
        </div>
      </motion.header>

      {/* Main content */}
      <main className="flex-1 flex flex-col lg:flex-row gap-4 lg:gap-6 p-4 lg:p-6 max-w-7xl mx-auto w-full">
        {/* Board */}
        <motion.div
          initial={{ opacity: 0, scale: 0.96 }}
          animate={{ opacity: 1, scale: 1 }}
          transition={{ duration: 0.5, delay: 0.1 }}
          className="flex-1 flex items-center justify-center"
        >
          <div className="relative w-full max-w-[600px]">
            <ChessBoard
              gameState={gameState}
              selectedSquare={selectedSquare}
              legalMoves={legalMoves}
              onSquareClick={selectSquare}
              playerColor={effectiveColor}
              isAiThinking={isAiThinking}
            />
            <AnimatePresence>
              {gameState.isGameOver && (
                <GameOverOverlay
                  gameState={gameState}
                  onRematch={resetGame}
                />
              )}
            </AnimatePresence>
          </div>
        </motion.div>

        {/* HUD */}
        <GameHUD
          gameState={gameState}
          isAiThinking={isAiThinking}
          stockfishDepth={stockfishDepth}
          playerColor={playerColor}
          onDifficultyChange={updateDifficulty}
          onReset={resetGame}
          onFlipBoard={handleFlipBoard}
        />
      </main>
    </div>
  );
}
