'use client';

import { useState, useCallback, useEffect, useRef } from 'react';
import { motion, AnimatePresence } from 'framer-motion';
import { Chess } from 'chess.js';
import { ChessBoard } from '@/components/board/ChessBoard';
import { GameHUD } from '@/components/hud/GameHUD';
import { GameOverOverlay } from '@/components/ui/GameOverOverlay';
import { NewGameModal } from '@/components/ui/NewGameModal';
import { WagerBanner } from '@/components/ui/WagerBanner';
import { useChessGame } from '@/hooks/useChessGame';
import { useWagerSession } from '@/hooks/useWagerSession';
import type { Color } from 'chess.js';

export function GamePage() {
  const {
    gameState,
    selectedSquare,
    legalMoves,
    stockfishDepth,
    isAiThinking,
    playerColor,
    opponentType,
    isMatchActive,
    selectSquare,
    startMatch,
    endMatch,
    updateDifficulty,
    applyRemoteFen,
    setMoveSyncHandler,
  } = useChessGame();

  const wagerCallbacks = useRef({
    onMatchStart: (_type: 'stockfish' | 'human', _color: Color) => {},
    onMatchEnd: () => {},
  });

  wagerCallbacks.current.onMatchStart = (type, color) => startMatch(type, color);
  wagerCallbacks.current.onMatchEnd = () => endMatch();

  const wager = useWagerSession({
    onMatchStart: (type, color) => wagerCallbacks.current.onMatchStart(type, color),
    onMatchEnd: () => wagerCallbacks.current.onMatchEnd(),
  });

  const [boardFlipped, setBoardFlipped] = useState(false);
  const lastRoomFenRef = useRef('');

  const handleFlipBoard = useCallback(() => {
    setBoardFlipped((f) => !f);
  }, []);

  const effectiveColor: Color = boardFlipped
    ? playerColor === 'w' ? 'b' : 'w'
    : playerColor;

  // Sync local moves to PvP room
  useEffect(() => {
    if (wager.session.phase !== 'playing' || wager.session.opponentType !== 'human') {
      setMoveSyncHandler(null);
      return;
    }
    if (!wager.session.roomId) return;

    setMoveSyncHandler((fen, lastMove) => {
      let winner: 'white' | 'black' | 'draw' | undefined;
      const chess = new Chess(fen);
      if (chess.isGameOver()) {
        if (chess.isDraw()) winner = 'draw';
        else winner = chess.turn() === 'w' ? 'black' : 'white';
        wager.settleSession();
      }
      wager.matchmaking.syncRoom(wager.session.roomId!, fen, lastMove, winner);
    });
  }, [
    wager.session.phase,
    wager.session.opponentType,
    wager.session.roomId,
    setMoveSyncHandler,
    wager.matchmaking,
    wager.settleSession,
  ]);

  // Poll opponent moves in PvP
  useEffect(() => {
    if (wager.session.phase !== 'playing' || wager.session.opponentType !== 'human') return;
    if (!wager.session.roomId) return;

    const room = wager.matchmaking.activeRoom;
    if (!room || room.id !== wager.session.roomId) return;
    if (room.fen === lastRoomFenRef.current) return;
    if (room.fen === gameState.fen) {
      lastRoomFenRef.current = room.fen;
      return;
    }

    lastRoomFenRef.current = room.fen;
    applyRemoteFen(room.fen, room.lastMove);

    if (room.status === 'finished') {
      wager.settleSession();
    }
  }, [
    wager.session,
    wager.matchmaking.activeRoom,
    gameState.fen,
    applyRemoteFen,
    wager.settleSession,
  ]);

  // Settle Stockfish wagers on game over
  useEffect(() => {
    if (!isMatchActive || opponentType !== 'stockfish') return;
    if (!gameState.isGameOver) return;
    wager.settleSession();
  }, [isMatchActive, opponentType, gameState.isGameOver, wager.settleSession]);

  const handleNewGame = useCallback(() => {
    wager.resetSession();
    endMatch();
    wager.openNewGameModal();
  }, [wager, endMatch]);

  const showOverlay =
    isMatchActive && gameState.isGameOver && wager.session.phase !== 'idle';

  return (
    <div
      className="min-h-screen flex flex-col"
      style={{
        background:
          'radial-gradient(ellipse at top, #1a1a22 0%, #0f0f11 60%)',
      }}
    >
      <motion.header
        initial={{ opacity: 0, y: -10 }}
        animate={{ opacity: 1, y: 0 }}
        transition={{ duration: 0.5 }}
        className="flex items-center justify-between px-6 py-4 border-b"
        style={{ borderColor: 'var(--bg-glass-border)' }}
      >
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
          Titan Local UAT
        </div>
      </motion.header>

      <main className="flex-1 flex flex-col lg:flex-row gap-4 lg:gap-6 p-4 lg:p-6 max-w-7xl mx-auto w-full">
        <motion.div
          initial={{ opacity: 0, scale: 0.96 }}
          animate={{ opacity: 1, scale: 1 }}
          transition={{ duration: 0.5, delay: 0.1 }}
          className="flex-1 flex items-center justify-center"
        >
          <div className="relative w-full max-w-[600px]">
            {!isMatchActive && wager.session.phase !== 'waiting' && (
              <div
                className="absolute inset-0 z-10 flex items-center justify-center rounded-xl"
                style={{ background: 'rgba(0,0,0,0.45)', backdropFilter: 'blur(2px)' }}
              >
                <button
                  onClick={handleNewGame}
                  className="px-6 py-3 rounded-xl font-semibold text-sm"
                  style={{
                    background: 'linear-gradient(135deg, var(--gold-primary), var(--gold-secondary))',
                    color: '#0f0f11',
                  }}
                >
                  New Wagered Game
                </button>
              </div>
            )}
            <ChessBoard
              gameState={gameState}
              selectedSquare={selectedSquare}
              legalMoves={legalMoves}
              onSquareClick={selectSquare}
              playerColor={effectiveColor}
              isAiThinking={isAiThinking}
            />
            <AnimatePresence>
              {showOverlay && (
                <GameOverOverlay
                  gameState={gameState}
                  playerColor={playerColor}
                  wagerSession={wager.session}
                  onRematch={handleNewGame}
                />
              )}
            </AnimatePresence>
          </div>
        </motion.div>

        <GameHUD
          gameState={gameState}
          isAiThinking={isAiThinking}
          stockfishDepth={stockfishDepth}
          playerColor={playerColor}
          opponentType={opponentType}
          isMatchActive={isMatchActive}
          wagerSession={wager.session}
          onDifficultyChange={updateDifficulty}
          onNewGame={handleNewGame}
          onFlipBoard={handleFlipBoard}
          onCancelWager={wager.cancelWaiting}
        />
      </main>

      <NewGameModal
        open={wager.showModal}
        onClose={wager.closeModal}
        onPlayStockfish={wager.startStockfishWager}
        onPlayHuman={wager.startHumanWager}
        waitingPlayers={wager.waitingPlayers}
        stakeBounds={wager.escrow.stakeBounds}
        escrowEnabled={wager.escrow.enabled}
        isConnected={wager.isConnected}
        isLoading={wager.escrow.isWritePending || wager.escrow.isConfirming || wager.matchmaking.isJoining}
        error={wager.session.error}
      />
    </div>
  );
}