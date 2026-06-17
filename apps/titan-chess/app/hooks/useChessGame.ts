'use client';

import { useState, useCallback, useRef, useEffect } from 'react';
import { Chess } from 'chess.js';
import type { Square, PieceSymbol, Color } from 'chess.js';
import type { GameState, MoveHistory } from '@/types/chess';
import type { OpponentType } from './useWagerSession';
import { getStockfishEngine } from '@/lib/stockfish';
import {
  playSelect,
  playMove,
  playCapture,
  playCheck,
  playCheckmate,
  resumeAudioContext,
} from '@/lib/audio';

function buildGameState(chess: Chess, history: MoveHistory[]): GameState {
  const lastHistoryEntry = history[history.length - 1];
  return {
    fen: chess.fen(),
    turn: chess.turn(),
    isCheck: chess.isCheck(),
    isCheckmate: chess.isCheckmate(),
    isDraw: chess.isDraw(),
    isGameOver: chess.isGameOver(),
    moveHistory: history,
    lastMove: lastHistoryEntry
      ? { from: lastHistoryEntry.from, to: lastHistoryEntry.to }
      : null,
  };
}

export function useChessGame() {
  const chessRef = useRef(new Chess());
  const [gameState, setGameState] = useState<GameState>(() =>
    buildGameState(chessRef.current, [])
  );
  const [selectedSquare, setSelectedSquare] = useState<Square | null>(null);
  const [legalMoves, setLegalMoves] = useState<Square[]>([]);
  const [stockfishDepth, setStockfishDepth] = useState(10);
  const [isAiThinking, setIsAiThinking] = useState(false);
  const [playerColor, setPlayerColor] = useState<Color>('w');
  const [moveHistory, setMoveHistoryState] = useState<MoveHistory[]>([]);
  const [opponentType, setOpponentType] = useState<OpponentType | null>(null);
  const [isMatchActive, setIsMatchActive] = useState(false);
  const engineInitialized = useRef(false);
  const onMoveSyncRef = useRef<
    ((fen: string, lastMove: { from: string; to: string } | null) => void) | null
  >(null);
  const lastSyncedFenRef = useRef<string>('');

  useEffect(() => {
    if (!engineInitialized.current) {
      engineInitialized.current = true;
      getStockfishEngine()
        .initialize()
        .catch((e) => console.warn('Stockfish not available:', e));
    }
  }, []);

  const syncState = useCallback((history: MoveHistory[]) => {
    setGameState(buildGameState(chessRef.current, history));
    setMoveHistoryState(history);
  }, []);

  const rebuildHistoryFromChess = useCallback((chess: Chess): MoveHistory[] => {
    const verbose = chess.history({ verbose: true });
    return verbose.map((m, i) => ({
      san: m.san,
      from: m.from,
      to: m.to,
      piece: m.piece,
      captured: m.captured,
      color: m.color,
      moveNumber: Math.ceil(i / 2) + 1,
    }));
  }, []);

  const triggerAiMove = useCallback(
    (currentHistory: MoveHistory[]) => {
      const chess = chessRef.current;
      if (chess.isGameOver()) return;
      if (chess.turn() === playerColor) return;
      if (opponentType !== 'stockfish') return;

      setIsAiThinking(true);

      try {
        const engine = getStockfishEngine();
        engine.setSkillLevel(stockfishDepth);
        engine.getBestMove(chess.fen(), stockfishDepth, (bestMove: string) => {
          if (!bestMove || bestMove === '(none)') {
            setIsAiThinking(false);
            return;
          }

          const from = bestMove.slice(0, 2) as Square;
          const to = bestMove.slice(2, 4) as Square;
          const promotion = bestMove[4] as PieceSymbol | undefined;

          const existingPiece = chess.get(to);
          const move = chess.move({ from, to, promotion: promotion || 'q' });

          if (move) {
            const newHistory: MoveHistory[] = [
              ...currentHistory,
              {
                san: move.san,
                from: move.from,
                to: move.to,
                piece: move.piece,
                captured: move.captured,
                color: move.color,
                moveNumber: Math.ceil(currentHistory.length / 2) + 1,
              },
            ];

            if (existingPiece || move.captured) {
              playCapture();
            } else {
              playMove();
            }

            if (chess.isCheckmate()) playCheckmate();
            else if (chess.isCheck()) playCheck();

            syncState(newHistory);
          }

          setIsAiThinking(false);
        });
      } catch (e) {
        console.warn('AI move failed:', e);
        setIsAiThinking(false);
      }
    },
    [playerColor, stockfishDepth, syncState, opponentType]
  );

  const applyRemoteFen = useCallback(
    (fen: string, lastMove: { from: string; to: string } | null) => {
      if (fen === lastSyncedFenRef.current) return;
      lastSyncedFenRef.current = fen;

      const chess = new Chess(fen);
      chessRef.current = chess;
      const history = rebuildHistoryFromChess(chess);
      setSelectedSquare(null);
      setLegalMoves([]);
      syncState(history);

      if (lastMove) {
        const captured = chess.get(lastMove.to as Square);
        if (captured) playCapture();
        else playMove();
      }
      if (chess.isCheckmate()) playCheckmate();
      else if (chess.isCheck()) playCheck();
    },
    [rebuildHistoryFromChess, syncState]
  );

  const selectSquare = useCallback(
    (square: Square) => {
      if (!isMatchActive) return;
      resumeAudioContext();
      const chess = chessRef.current;
      if (chess.isGameOver()) return;
      if (chess.turn() !== playerColor) return;

      const piece = chess.get(square);

      if (piece && piece.color === playerColor) {
        if (selectedSquare === square) {
          setSelectedSquare(null);
          setLegalMoves([]);
        } else {
          setSelectedSquare(square);
          const moves = chess.moves({ square, verbose: true });
          setLegalMoves(moves.map((m) => m.to as Square));
          playSelect();
        }
        return;
      }

      if (selectedSquare) {
        const targetPiece = chess.get(square);
        const movingPiece = chess.get(selectedSquare);

        const isPromotion =
          movingPiece?.type === 'p' &&
          ((movingPiece.color === 'w' && square[1] === '8') ||
            (movingPiece.color === 'b' && square[1] === '1'));

        const move = chess.move({
          from: selectedSquare,
          to: square,
          promotion: isPromotion ? 'q' : undefined,
        });

        if (move) {
          const newHistory: MoveHistory[] = [
            ...moveHistory,
            {
              san: move.san,
              from: move.from,
              to: move.to,
              piece: move.piece,
              captured: move.captured,
              color: move.color,
              moveNumber: Math.ceil(moveHistory.length / 2) + 1,
            },
          ];

          if (targetPiece || move.captured) {
            playCapture();
          } else {
            playMove();
          }

          if (chess.isCheckmate()) {
            setTimeout(playCheckmate, 100);
          } else if (chess.isCheck()) {
            playCheck();
          }

          setSelectedSquare(null);
          setLegalMoves([]);
          syncState(newHistory);

          const fen = chess.fen();
          const lastMove = { from: move.from, to: move.to };
          lastSyncedFenRef.current = fen;
          onMoveSyncRef.current?.(fen, lastMove);

          if (opponentType === 'stockfish') {
            setTimeout(() => triggerAiMove(newHistory), 300);
          }
        } else {
          if (piece && piece.color === playerColor) {
            setSelectedSquare(square);
            const moves = chess.moves({ square, verbose: true });
            setLegalMoves(moves.map((m) => m.to as Square));
            playSelect();
          } else {
            setSelectedSquare(null);
            setLegalMoves([]);
          }
        }
      }
    },
    [
      isMatchActive,
      selectedSquare,
      moveHistory,
      playerColor,
      syncState,
      triggerAiMove,
      opponentType,
    ]
  );

  const resetGame = useCallback(() => {
    chessRef.current = new Chess();
    setSelectedSquare(null);
    setLegalMoves([]);
    setIsAiThinking(false);
    lastSyncedFenRef.current = '';
    const emptyHistory: MoveHistory[] = [];
    syncState(emptyHistory);
  }, [syncState]);

  const startMatch = useCallback(
    (type: OpponentType, color: Color) => {
      resetGame();
      setOpponentType(type);
      setPlayerColor(color);
      setIsMatchActive(true);
    },
    [resetGame]
  );

  const endMatch = useCallback(() => {
    setIsMatchActive(false);
    setOpponentType(null);
    resetGame();
  }, [resetGame]);

  const updateDifficulty = useCallback((level: number) => {
    setStockfishDepth(level);
    try {
      getStockfishEngine().setSkillLevel(level);
    } catch {
      // Engine not ready yet
    }
  }, []);

  const setMoveSyncHandler = useCallback(
    (
      handler:
        | ((fen: string, lastMove: { from: string; to: string } | null) => void)
        | null
    ) => {
      onMoveSyncRef.current = handler;
    },
    []
  );

  return {
    gameState,
    selectedSquare,
    legalMoves,
    stockfishDepth,
    isAiThinking,
    playerColor,
    opponentType,
    isMatchActive,
    selectSquare,
    resetGame,
    startMatch,
    endMatch,
    updateDifficulty,
    setPlayerColor,
    applyRemoteFen,
    setMoveSyncHandler,
  };
}