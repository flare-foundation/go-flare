'use client';

import { useState, useCallback, useRef, useEffect } from 'react';
import { Chess } from 'chess.js';
import type { Square, PieceSymbol, Color } from 'chess.js';
import type { GameState, MoveHistory } from '@/types/chess';
import { getStockfishEngine } from '@/lib/stockfish';
import {
  playSelect,
  playMove,
  playCapture,
  playCheck,
  playCheckmate,
  resumeAudioContext,
} from '@/lib/audio';

const INITIAL_FEN = 'rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNR w KQkq - 0 1';

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
  const engineInitialized = useRef(false);

  // Initialize stockfish
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

  const triggerAiMove = useCallback(
    (currentHistory: MoveHistory[]) => {
      const chess = chessRef.current;
      if (chess.isGameOver()) return;
      if (chess.turn() === playerColor) return;

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
    [playerColor, stockfishDepth, syncState]
  );

  const selectSquare = useCallback(
    (square: Square) => {
      resumeAudioContext();
      const chess = chessRef.current;
      if (chess.isGameOver()) return;

      // If it's not player's turn, do nothing
      if (chess.turn() !== playerColor) return;

      const piece = chess.get(square);

      // If clicking on own piece, select it
      if (piece && piece.color === playerColor) {
        if (selectedSquare === square) {
          // Deselect
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

      // If a square is selected, try to move
      if (selectedSquare) {
        const targetPiece = chess.get(square);
        const movingPiece = chess.get(selectedSquare);

        // Check promotion
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

          // Trigger AI response
          setTimeout(() => triggerAiMove(newHistory), 300);
        } else {
          // Invalid move — try selecting the clicked square if it has a piece
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
    [selectedSquare, moveHistory, playerColor, syncState, triggerAiMove]
  );

  const resetGame = useCallback(() => {
    chessRef.current = new Chess();
    setSelectedSquare(null);
    setLegalMoves([]);
    setIsAiThinking(false);
    const emptyHistory: MoveHistory[] = [];
    syncState(emptyHistory);
  }, [syncState]);

  const updateDifficulty = useCallback(
    (level: number) => {
      setStockfishDepth(level);
      try {
        getStockfishEngine().setSkillLevel(level);
      } catch (e) {
        // Engine not ready yet
      }
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
    selectSquare,
    resetGame,
    updateDifficulty,
    setPlayerColor,
  };
}
