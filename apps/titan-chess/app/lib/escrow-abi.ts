export const TITAN_CHESS_ESCROW_ABI = [
  {
    type: 'function',
    name: 'joinQueue',
    stateMutability: 'payable',
    inputs: [],
    outputs: [],
  },
  {
    type: 'function',
    name: 'leaveQueue',
    stateMutability: 'nonpayable',
    inputs: [],
    outputs: [],
  },
  {
    type: 'function',
    name: 'minStake',
    stateMutability: 'view',
    inputs: [],
    outputs: [{ type: 'uint256' }],
  },
  {
    type: 'function',
    name: 'maxStake',
    stateMutability: 'view',
    inputs: [],
    outputs: [{ type: 'uint256' }],
  },
  {
    type: 'function',
    name: 'queueLength',
    stateMutability: 'view',
    inputs: [],
    outputs: [{ type: 'uint256' }],
  },
  {
    type: 'function',
    name: 'activeGames',
    stateMutability: 'view',
    inputs: [],
    outputs: [{ type: 'uint256' }],
  },
  {
    type: 'function',
    name: 'playerInQueue',
    stateMutability: 'view',
    inputs: [{ name: '', type: 'address' }],
    outputs: [{ type: 'bool' }],
  },
  {
    type: 'function',
    name: 'playerInActiveGame',
    stateMutability: 'view',
    inputs: [{ name: '', type: 'address' }],
    outputs: [{ type: 'bool' }],
  },
  {
    type: 'function',
    name: 'getQueueEntry',
    stateMutability: 'view',
    inputs: [{ name: 'index', type: 'uint256' }],
    outputs: [
      { name: 'player', type: 'address' },
      { name: 'stake', type: 'uint256' },
      { name: 'queuedAt', type: 'uint256' },
    ],
  },
  {
    type: 'function',
    name: 'peekNextPlayer',
    stateMutability: 'view',
    inputs: [],
    outputs: [
      { name: 'player', type: 'address' },
      { name: 'stake', type: 'uint256' },
      { name: 'queuedAt', type: 'uint256' },
    ],
  },
  {
    type: 'function',
    name: 'getGame',
    stateMutability: 'view',
    inputs: [{ name: 'gameId', type: 'uint256' }],
    outputs: [
      { name: 'player', type: 'address' },
      { name: 'playerStake', type: 'uint256' },
      { name: 'stockfishStake', type: 'uint256' },
      { name: 'status', type: 'uint8' },
      { name: 'outcome', type: 'uint8' },
      { name: 'winner', type: 'address' },
      { name: 'startedAt', type: 'uint256' },
      { name: 'finishedAt', type: 'uint256' },
    ],
  },
  {
    type: 'event',
    name: 'MatchStarted',
    inputs: [
      { name: 'gameId', type: 'uint256', indexed: true },
      { name: 'player', type: 'address', indexed: true },
      { name: 'stake', type: 'uint256', indexed: false },
    ],
  },
  {
    type: 'event',
    name: 'MatchResolved',
    inputs: [
      { name: 'gameId', type: 'uint256', indexed: true },
      { name: 'outcome', type: 'uint8', indexed: false },
      { name: 'winner', type: 'address', indexed: true },
      { name: 'playerPayout', type: 'uint256', indexed: false },
      { name: 'stockfishPayout', type: 'uint256', indexed: false },
    ],
  },
] as const;

export enum EscrowGameStatus {
  Active = 0,
  Finished = 1,
  Cancelled = 2,
}

export enum EscrowOutcome {
  None = 0,
  PlayerWins = 1,
  StockfishWins = 2,
  Draw = 3,
}