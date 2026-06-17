import { parseEther } from 'viem';

/** Deploy TitanChessEscrow via Contract Studio, then set this address. */
export const ESCROW_ADDRESS = process.env.NEXT_PUBLIC_ESCROW_ADDRESS as `0x${string}` | undefined;

export const ESCROW_ENABLED = Boolean(
  ESCROW_ADDRESS && ESCROW_ADDRESS.startsWith('0x') && ESCROW_ADDRESS.length === 42
);

/** Defaults match Contract Studio template (0.01 – 1 TITAN). */
export const DEFAULT_MIN_STAKE = parseEther('0.01');
export const DEFAULT_MAX_STAKE = parseEther('1');