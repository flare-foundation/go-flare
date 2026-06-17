export interface MatchmakingEntry {
  address: string;
  stake: string;
  joinedAt: number;
}

export interface ActiveRoom {
  id: string;
  white: string;
  black: string;
  stake: string;
  fen: string;
  lastMove: { from: string; to: string } | null;
  status: 'active' | 'finished';
  winner: 'white' | 'black' | 'draw' | null;
  createdAt: number;
}