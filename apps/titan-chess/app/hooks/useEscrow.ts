'use client';

import { useCallback, useMemo } from 'react';
import {
  useAccount,
  useReadContract,
  useWriteContract,
  useWaitForTransactionReceipt,
  useWatchContractEvent,
} from 'wagmi';
import { formatEther, parseEther } from 'viem';
import { titanSubnet } from '@/lib/web3';
import { TITAN_CHESS_ESCROW_ABI } from '@/lib/escrow-abi';
import {
  ESCROW_ADDRESS,
  ESCROW_ENABLED,
  DEFAULT_MIN_STAKE,
  DEFAULT_MAX_STAKE,
} from '@/lib/escrow-config';

export interface QueueEntry {
  player: `0x${string}`;
  stake: bigint;
  queuedAt: bigint;
}

export function useEscrow() {
  const { address, isConnected } = useAccount();

  const enabled = ESCROW_ENABLED && isConnected;

  const { data: minStakeRaw } = useReadContract({
    address: ESCROW_ADDRESS,
    abi: TITAN_CHESS_ESCROW_ABI,
    functionName: 'minStake',
    chainId: titanSubnet.id,
    query: { enabled: ESCROW_ENABLED },
  });

  const { data: maxStakeRaw } = useReadContract({
    address: ESCROW_ADDRESS,
    abi: TITAN_CHESS_ESCROW_ABI,
    functionName: 'maxStake',
    chainId: titanSubnet.id,
    query: { enabled: ESCROW_ENABLED },
  });

  const { data: queueLengthRaw, refetch: refetchQueue } = useReadContract({
    address: ESCROW_ADDRESS,
    abi: TITAN_CHESS_ESCROW_ABI,
    functionName: 'queueLength',
    chainId: titanSubnet.id,
    query: { enabled: ESCROW_ENABLED, refetchInterval: 5_000 },
  });

  const { data: activeGamesRaw, refetch: refetchActive } = useReadContract({
    address: ESCROW_ADDRESS,
    abi: TITAN_CHESS_ESCROW_ABI,
    functionName: 'activeGames',
    chainId: titanSubnet.id,
    query: { enabled: ESCROW_ENABLED, refetchInterval: 3_000 },
  });

  const { data: inQueue, refetch: refetchInQueue } = useReadContract({
    address: ESCROW_ADDRESS,
    abi: TITAN_CHESS_ESCROW_ABI,
    functionName: 'playerInQueue',
    args: address ? [address] : undefined,
    chainId: titanSubnet.id,
    query: { enabled: enabled && !!address, refetchInterval: 3_000 },
  });

  const { data: inActiveGame, refetch: refetchInActiveGame } = useReadContract({
    address: ESCROW_ADDRESS,
    abi: TITAN_CHESS_ESCROW_ABI,
    functionName: 'playerInActiveGame',
    args: address ? [address] : undefined,
    chainId: titanSubnet.id,
    query: { enabled: enabled && !!address, refetchInterval: 2_000 },
  });

  const minStake = minStakeRaw ?? DEFAULT_MIN_STAKE;
  const maxStake = maxStakeRaw ?? DEFAULT_MAX_STAKE;
  const queueLength = Number(queueLengthRaw ?? BigInt(0));
  const activeGames = Number(activeGamesRaw ?? BigInt(0));

  const {
    writeContract,
    data: txHash,
    isPending: isWritePending,
    error: writeError,
    reset: resetWrite,
  } = useWriteContract();

  const { isLoading: isConfirming, isSuccess: isConfirmed } = useWaitForTransactionReceipt({
    hash: txHash,
    chainId: titanSubnet.id,
  });

  const joinQueue = useCallback(
    (stakeTitan: string) => {
      if (!ESCROW_ENABLED || !ESCROW_ADDRESS) return;
      const value = parseEther(stakeTitan);
      writeContract({
        address: ESCROW_ADDRESS,
        abi: TITAN_CHESS_ESCROW_ABI,
        functionName: 'joinQueue',
        value,
        chain: titanSubnet,
        account: address,
      });
    },
    [writeContract, address]
  );

  const leaveQueue = useCallback(() => {
    if (!ESCROW_ENABLED || !ESCROW_ADDRESS) return;
    writeContract({
      address: ESCROW_ADDRESS,
      abi: TITAN_CHESS_ESCROW_ABI,
      functionName: 'leaveQueue',
      chain: titanSubnet,
      account: address,
    });
  }, [writeContract, address]);

  const refetchAll = useCallback(() => {
    refetchQueue();
    refetchActive();
    refetchInQueue();
    refetchInActiveGame();
  }, [refetchQueue, refetchActive, refetchInQueue, refetchInActiveGame]);

  const stakeBounds = useMemo(
    () => ({
      min: formatEther(minStake),
      max: formatEther(maxStake),
      minWei: minStake,
      maxWei: maxStake,
    }),
    [minStake, maxStake]
  );

  return {
    enabled: ESCROW_ENABLED,
    address: ESCROW_ADDRESS,
    isConnected,
    minStake,
    maxStake,
    stakeBounds,
    queueLength,
    activeGames,
    inQueue: Boolean(inQueue),
    inActiveGame: Boolean(inActiveGame),
    joinQueue,
    leaveQueue,
    txHash,
    isWritePending,
    isConfirming,
    isConfirmed,
    writeError,
    resetWrite,
    refetchAll,
  };
}

export function useEscrowMatchStarted(onMatch: (gameId: bigint, player: `0x${string}`, stake: bigint) => void) {
  useWatchContractEvent({
    address: ESCROW_ADDRESS,
    abi: TITAN_CHESS_ESCROW_ABI,
    eventName: 'MatchStarted',
    chainId: titanSubnet.id,
    enabled: ESCROW_ENABLED,
    onLogs(logs) {
      for (const log of logs) {
        const { gameId, player, stake } = log.args;
        if (gameId != null && player && stake != null) {
          onMatch(gameId, player, stake);
        }
      }
    },
  });
}