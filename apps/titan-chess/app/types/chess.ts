import type { Square, PieceSymbol, Color } from 'chess.js';

export type { Square, PieceSymbol, Color };

export interface ChessPiece {
  type: PieceSymbol;
  color: Color;
  square: Square;
}

export interface MoveHistory {
  san: string;
  from: Square;
  to: Square;
  piece: PieceSymbol;
  captured?: PieceSymbol;
  promotion?: PieceSymbol;
  color: Color;
  moveNumber: number;
}

export interface GameState {
  fen: string;
  turn: Color;
  isCheck: boolean;
  isCheckmate: boolean;
  isDraw: boolean;
  isGameOver: boolean;
  moveHistory: MoveHistory[];
  lastMove: { from: Square; to: Square } | null;
}

export interface BoardSquare {
  square: Square;
  piece: ChessPiece | null;
  rank: number;
  file: number;
}

export type GamePhase = 'idle' | 'playing' | 'checkmate' | 'draw' | 'stalemate';

export interface AudioContextState {
  playSelect: () => void;
  playMove: () => void;
  playCapture: () => void;
  playCheck: () => void;
  playCheckmate: () => void;
}
