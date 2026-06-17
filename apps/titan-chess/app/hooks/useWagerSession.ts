'use client';

import { useCallback, useEffect, useRef, useState } from 'react';
import { useAccount } from 'wagmi';
import { formatEther, parseEther } from 'viem';
import type { Color } from 'chess.js';
import { useEscrow, useEscrowMatchStarted } from './useEscrow';
import { useMatchmaking } from './useMatchmaking';

export type OpponentType = 'stockfish' | 'human';
export type WagerPhase = 'idle' | 'modal' | 'waiting' | 'playing' | 'settled';

export interface WagerSession {
  phase: WagerPhase;
  opponentType: OpponentType | null;
  stake: string;
  potTitan: string;
  opponentLabel: string;
  escrowGameId: bigint | null;
  roomId: string | null;
  queuePosition: number | null;
  error: string | null;
  isPractice: boolean;
}

const INITIAL_SESSION: WagerSession = {
  phase: 'idle',
  opponentType: null,
  stake: '0.1',
  potTitan: '0',
  opponentLabel: '',
  escrowGameId: null,
  roomId: null,
  queuePosition: null,
  error: null,
  isPractice: false,
};

interface WagerCallbacks {
  onMatchStart: (opponentType: OpponentType, playerColor: Color) => void;
  onMatchEnd: () => void;
}

export function useWagerSession(callbacks: WagerCallbacks) {
  const { address, isConnected } = useAccount();
  const escrow = useEscrow();
  const matchmaking = useMatchmaking(address);
  const [session, setSession] = useState<WagerSession>(INITIAL_SESSION);
  const [showModal, setShowModal] = useState(false);
  const matchStartedRef = useRef(false);

  const openNewGameModal = useCallback(() => {
    setShowModal(true);
    setSession((s) => ({ ...s, phase: 'modal', error: null }));
  }, []);

  const closeModal = useCallback(() => {
    setShowModal(false);
    if (session.phase === 'modal') {
      setSession((s) => ({ ...s, phase: 'idle' }));
    }
  }, [session.phase]);

  const startStockfishWager = useCallback(
    async (stake: string) => {
      if (!isConnected) {
        setSession((s) => ({ ...s, error: 'Connect your wallet first' }));
        return;
      }

      const stakeWei = parseEther(stake);
      if (stakeWei < escrow.minStake || stakeWei > escrow.maxStake) {
        setSession((s) => ({
          ...s,
          error: `Stake must be between ${escrow.stakeBounds.min} and ${escrow.stakeBounds.max} TITAN`,
        }));
        return;
      }

      setSession((s) => ({
        ...s,
        phase: 'waiting',
        opponentType: 'stockfish',
        stake,
        potTitan: (Number(stake) * 2).toFixed(4),
        opponentLabel: 'Stockfish',
        error: null,
        isPractice: !escrow.enabled,
        queuePosition: null,
      }));
      setShowModal(false);
      matchStartedRef.current = false;

      if (escrow.enabled) {
        if (escrow.inActiveGame) {
          matchStartedRef.current = true;
          callbacks.onMatchStart('stockfish', 'w');
          setSession((s) => ({ ...s, phase: 'playing' }));
        } else {
          escrow.joinQueue(stake);
        }
      } else {
        matchStartedRef.current = true;
        callbacks.onMatchStart('stockfish', 'w');
        setSession((s) => ({ ...s, phase: 'playing', isPractice: true }));
      }
    },
    [isConnected, escrow, callbacks]
  );

  const startHumanWager = useCallback(
    async (stake: string, opponentAddress?: string) => {
      if (!isConnected || !address) {
        setSession((s) => ({ ...s, error: 'Connect your wallet first' }));
        return;
      }

      setSession((s) => ({
        ...s,
        phase: 'waiting',
        opponentType: 'human',
        stake,
        potTitan: (Number(stake) * 2).toFixed(4),
        opponentLabel: opponentAddress
          ? `${opponentAddress.slice(0, 6)}…${opponentAddress.slice(-4)}`
          : 'Opponent',
        error: null,
        isPractice: false,
        escrowGameId: null,
        roomId: null,
      }));
      setShowModal(false);
      matchStartedRef.current = false;

      try {
        const result = await matchmaking.joinQueue(stake);
        if (result.matched && result.room) {
          matchStartedRef.current = true;
          const color = (result.playerColor ?? 'w') as Color;
          callbacks.onMatchStart('human', color);
          setSession((s) => ({
            ...s,
            phase: 'playing',
            roomId: result.room!.id,
            opponentLabel: result.opponent?.shortAddress ?? s.opponentLabel,
          }));
        } else {
          setSession((s) => ({
            ...s,
            queuePosition: result.queuePosition ?? null,
          }));
        }
      } catch (e) {
        setSession((s) => ({
          ...s,
          phase: 'modal',
          error: e instanceof Error ? e.message : 'Failed to join queue',
        }));
        setShowModal(true);
      }
    },
    [isConnected, address, matchmaking, callbacks]
  );

  const cancelWaiting = useCallback(async () => {
    if (session.opponentType === 'stockfish' && escrow.inQueue) {
      escrow.leaveQueue();
    }
    if (session.opponentType === 'human') {
      await matchmaking.leaveQueue();
    }
    matchStartedRef.current = false;
    callbacks.onMatchEnd();
    setSession(INITIAL_SESSION);
  }, [session.opponentType, escrow, matchmaking, callbacks]);

  const settleSession = useCallback(() => {
    setSession((s) => ({ ...s, phase: 'settled' }));
  }, []);

  const resetSession = useCallback(() => {
    matchStartedRef.current = false;
    matchmaking.clearRoom();
    setSession(INITIAL_SESSION);
  }, [matchmaking]);

  // Escrow: watch for match start
  useEscrowMatchStarted(
    useCallback(
      (gameId, player, stake) => {
        if (!address || player.toLowerCase() !== address.toLowerCase()) return;
        if (matchStartedRef.current) return;
        matchStartedRef.current = true;
        callbacks.onMatchStart('stockfish', 'w');
        setSession((s) => ({
          ...s,
          phase: 'playing',
          escrowGameId: gameId,
          stake: formatEther(stake),
          potTitan: (Number(formatEther(stake)) * 2).toFixed(4),
        }));
      },
      [address, callbacks]
    )
  );

  // Escrow: poll inActiveGame when waiting for Stockfish
  useEffect(() => {
    if (session.phase !== 'waiting' || session.opponentType !== 'stockfish') return;
    if (!escrow.enabled) return;
    if (escrow.inActiveGame && !matchStartedRef.current) {
      matchStartedRef.current = true;
      callbacks.onMatchStart('stockfish', 'w');
      setSession((s) => ({ ...s, phase: 'playing' }));
    }
    if (escrow.inQueue) {
      setSession((s) => ({
        ...s,
        queuePosition: 1,
        phase: 'waiting',
      }));
    }
  }, [
    session.phase,
    session.opponentType,
    escrow.enabled,
    escrow.inActiveGame,
    escrow.inQueue,
    callbacks,
  ]);

  // Human: poll for match while waiting
  useEffect(() => {
    if (session.phase !== 'waiting' || session.opponentType !== 'human') return;
    if (matchmaking.activeRoom && !matchStartedRef.current) {
      matchStartedRef.current = true;
      const isWhite =
        matchmaking.activeRoom.white.toLowerCase() === address?.toLowerCase();
      callbacks.onMatchStart('human', isWhite ? 'w' : 'b');
      setSession((s) => ({
        ...s,
        phase: 'playing',
        roomId: matchmaking.activeRoom!.id,
        opponentLabel: isWhite
          ? `${matchmaking.activeRoom!.black.slice(0, 6)}…${matchmaking.activeRoom!.black.slice(-4)}`
          : `${matchmaking.activeRoom!.white.slice(0, 6)}…${matchmaking.activeRoom!.white.slice(-4)}`,
      }));
    } else if (matchmaking.myQueuePosition) {
      setSession((s) => ({
        ...s,
        queuePosition: matchmaking.myQueuePosition,
      }));
    }
  }, [
    session.phase,
    session.opponentType,
    matchmaking.activeRoom,
    matchmaking.myQueuePosition,
    address,
    callbacks,
  ]);

  // Human: sync opponent moves from room poll
  useEffect(() => {
    if (session.phase !== 'playing' || session.opponentType !== 'human' || !session.roomId) {
      return;
    }
    const room = matchmaking.activeRoom;
    if (!room || room.id !== session.roomId) return;
    if (room.status === 'finished') {
      settleSession();
    }
  }, [session, matchmaking.activeRoom, settleSession]);

  return {
    session,
    showModal,
    openNewGameModal,
    closeModal,
    startStockfishWager,
    startHumanWager,
    cancelWaiting,
    settleSession,
    resetSession,
    escrow,
    matchmaking,
    waitingPlayers: matchmaking.waitingPlayers,
    isConnected,
  };
}