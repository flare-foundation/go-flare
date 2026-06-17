'use client';

import { useMemo, useCallback } from 'react';
import { motion, AnimatePresence } from 'framer-motion';
import { Chess } from 'chess.js';
import type { Square, Color } from 'chess.js';
import type { GameState } from '@/types/chess';
import { getPieceSVG } from '@/lib/pieces';

interface ChessBoardProps {
  gameState: GameState;
  selectedSquare: Square | null;
  legalMoves: Square[];
  onSquareClick: (square: Square) => void;
  playerColor?: Color;
  isAiThinking?: boolean;
}

const FILES = ['a', 'b', 'c', 'd', 'e', 'f', 'g', 'h'];

function coordsToSquare(col: number, row: number, flipped: boolean): Square {
  const file = flipped ? 7 - col : col;
  const rank = flipped ? row : 7 - row;
  return `${FILES[file]}${rank + 1}` as Square;
}

function isLightSquare(col: number, row: number): boolean {
  return (col + row) % 2 === 0;
}

export function ChessBoard({
  gameState,
  selectedSquare,
  legalMoves,
  onSquareClick,
  playerColor = 'w',
  isAiThinking,
}: ChessBoardProps) {
  const flipped = playerColor === 'b';
  
  const chess = useMemo(() => {
    const c = new Chess();
    try { c.load(gameState.fen); } catch {}
    return c;
  }, [gameState.fen]);

  const CELL = 80;
  const BOARD = CELL * 8;
  const LABEL = 20;

  const squares = useMemo(() => {
    const result = [];
    for (let row = 0; row < 8; row++) {
      for (let col = 0; col < 8; col++) {
        const sq = coordsToSquare(col, row, flipped);
        const piece = chess.get(sq);
        result.push({ sq, col, row, piece: piece || null });
      }
    }
    return result;
  }, [chess, flipped]);

  const isLastMove = useCallback(
    (sq: Square) => gameState.lastMove?.from === sq || gameState.lastMove?.to === sq,
    [gameState.lastMove]
  );

  const isKingInCheck = useCallback(
    (sq: Square) => {
      if (!gameState.isCheck) return false;
      const piece = chess.get(sq);
      return piece?.type === 'k' && piece.color === gameState.turn;
    },
    [chess, gameState.isCheck, gameState.turn]
  );

  const isLegal = useCallback((sq: Square) => legalMoves.includes(sq), [legalMoves]);
  const isCapture = useCallback((sq: Square) => isLegal(sq) && !!chess.get(sq), [chess, isLegal]);

  const rankLabels = flipped ? [1,2,3,4,5,6,7,8] : [8,7,6,5,4,3,2,1];
  const fileLabels = flipped ? [...FILES].reverse() : FILES;

  return (
    <div className="relative select-none" style={{ touchAction: 'none' }}>
      <svg
        viewBox={`0 0 ${BOARD + LABEL} ${BOARD + LABEL}`}
        className="w-full max-w-[640px]"
        style={{ filter: 'drop-shadow(0 20px 60px rgba(0,0,0,0.6))' }}
      >
        {/* Board squares */}
        {squares.map(({ sq, col, row, piece }) => {
          const x = LABEL + col * CELL;
          const y = row * CELL;
          const light = isLightSquare(col, row);
          const selected = selectedSquare === sq;
          const lastMove = isLastMove(sq);
          const check = isKingInCheck(sq);
          const legal = isLegal(sq);
          const capture = isCapture(sq);

          let fillColor = light ? '#f0d9b5' : '#b58863';
          if (lastMove) fillColor = light ? '#cdd26a' : '#aaa23a';
          if (selected) fillColor = light ? '#7fc97a' : '#4d9a4d';
          if (check) fillColor = '#cc3333';

          return (
            <g key={sq}>
              <rect
                x={x} y={y}
                width={CELL} height={CELL}
                fill={fillColor}
                onClick={() => onSquareClick(sq)}
                style={{ cursor: 'pointer' }}
              />
              {legal && !capture && (
                <circle
                  cx={x + CELL / 2} cy={y + CELL / 2}
                  r={CELL * 0.15}
                  fill="rgba(0,0,0,0.2)"
                  onClick={() => onSquareClick(sq)}
                  style={{ cursor: 'pointer', animation: 'legalPulse 1.4s ease-in-out infinite' }}
                />
              )}
              {legal && capture && (
                <circle
                  cx={x + CELL / 2} cy={y + CELL / 2}
                  r={CELL * 0.46}
                  fill="none"
                  stroke="rgba(0,0,0,0.2)"
                  strokeWidth={CELL * 0.12}
                  onClick={() => onSquareClick(sq)}
                  style={{ cursor: 'pointer', animation: 'legalPulse 1.4s ease-in-out infinite' }}
                />
              )}
              {selected && (
                <rect x={x} y={y} width={CELL} height={CELL}
                  fill="none" stroke="rgba(255,255,255,0.35)" strokeWidth="2" />
              )}
            </g>
          );
        })}

        {/* File labels */}
        {fileLabels.map((file, i) => (
          <text key={file} x={LABEL + i * CELL + CELL - 8} y={BOARD + LABEL - 4}
            fontSize={10} fontWeight="600"
            fill={isLightSquare(i, 7) ? '#b58863' : '#f0d9b5'}
            fontFamily="Inter, sans-serif">{file}</text>
        ))}

        {/* Rank labels */}
        {rankLabels.map((rank, i) => (
          <text key={rank} x={3} y={i * CELL + 14}
            fontSize={10} fontWeight="600"
            fill={isLightSquare(0, i) ? '#f0d9b5' : '#b58863'}
            fontFamily="Inter, sans-serif">{rank}</text>
        ))}

        {/* Pieces */}
        {squares.map(({ sq, col, row, piece }) => {
          if (!piece) return null;
          const x = LABEL + col * CELL;
          const y = row * CELL;
          const svgContent = getPieceSVG(piece.type, piece.color);

          return (
            <foreignObject
              key={`${sq}-${piece.color}${piece.type}`}
              x={x} y={y}
              width={CELL} height={CELL}
              onClick={() => onSquareClick(sq)}
              style={{ cursor: 'pointer', overflow: 'visible' }}
            >
              <div style={{ width: CELL, height: CELL, display: 'flex', alignItems: 'center', justifyContent: 'center' }}>
                <div
                  style={{
                    width: CELL * 0.86,
                    height: CELL * 0.86,
                    filter: selectedSquare === sq
                      ? 'drop-shadow(0 4px 12px rgba(201,168,76,0.8))'
                      : 'drop-shadow(0 2px 5px rgba(0,0,0,0.45))',
                    transition: 'filter 0.15s ease',
                    transform: selectedSquare === sq ? 'scale(1.07)' : 'scale(1)',
                  }}
                  dangerouslySetInnerHTML={{ __html: svgContent }}
                />
              </div>
            </foreignObject>
          );
        })}

        {/* AI thinking overlay */}
        {isAiThinking && (
          <rect x={LABEL} y={0} width={BOARD} height={BOARD}
            fill="rgba(0,0,0,0.06)" style={{ pointerEvents: 'none' }} />
        )}
      </svg>

      <style>{`
        @keyframes legalPulse {
          0%, 100% { opacity: 0.4; }
          50% { opacity: 0.85; }
        }
      `}</style>
    </div>
  );
}
