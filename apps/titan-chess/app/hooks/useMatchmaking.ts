'use client';

import { useCallback, useEffect, useState } from 'react';
import type { ActiveRoom } from '@/types/matchmaking';
import type { Color } from 'chess.js';

export interface WaitingPlayer {
  address: string;
  shortAddress: string;
  stake: string;
  joinedAt: number;
}

export function useMatchmaking(address: string | undefined) {
  const [waitingPlayers, setWaitingPlayers] = useState<WaitingPlayer[]>([]);
  const [queueLength, setQueueLength] = useState(0);
  const [myQueuePosition, setMyQueuePosition] = useState<number | null>(null);
  const [activeRoom, setActiveRoom] = useState<ActiveRoom | null>(null);
  const [isJoining, setIsJoining] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const poll = useCallback(async () => {
    if (!address) return;
    try {
      const res = await fetch(`/api/matchmaking?address=${address}`);
      if (!res.ok) return;
      const data = await res.json();
      setWaitingPlayers(data.waitingPlayers ?? []);
      setQueueLength(data.queueLength ?? 0);
      setMyQueuePosition(data.myQueuePosition ?? null);
      if (data.myRoom) {
        setActiveRoom(data.myRoom);
      }
    } catch {
      // API unavailable in static export — ignore
    }
  }, [address]);

  useEffect(() => {
    poll();
    const id = setInterval(poll, 2_000);
    return () => clearInterval(id);
  }, [poll]);

  const joinQueue = useCallback(
    async (stake: string): Promise<{
      matched: boolean;
      room?: ActiveRoom;
      playerColor?: Color;
      opponent?: { address: string; shortAddress: string };
      queuePosition?: number;
    }> => {
      if (!address) throw new Error('Wallet not connected');
      setIsJoining(true);
      setError(null);
      try {
        const res = await fetch('/api/matchmaking', {
          method: 'POST',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify({ action: 'join', address, stake }),
        });
        const data = await res.json();
        if (!res.ok) throw new Error(data.error ?? 'Failed to join queue');
        if (data.matched && data.room) {
          setActiveRoom(data.room);
          setMyQueuePosition(null);
        } else {
          setMyQueuePosition(data.queuePosition ?? null);
        }
        await poll();
        return data;
      } catch (e) {
        const msg = e instanceof Error ? e.message : 'Matchmaking failed';
        setError(msg);
        throw e;
      } finally {
        setIsJoining(false);
      }
    },
    [address, poll]
  );

  const leaveQueue = useCallback(async () => {
    if (!address) return;
    await fetch('/api/matchmaking', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ action: 'leave', address }),
    });
    setMyQueuePosition(null);
    await poll();
  }, [address, poll]);

  const syncRoom = useCallback(
    async (
      roomId: string,
      fen: string,
      lastMove: { from: string; to: string } | null,
      winner?: 'white' | 'black' | 'draw'
    ) => {
      if (!address) return;
      const res = await fetch('/api/matchmaking', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ action: 'sync', address, roomId, fen, lastMove, winner }),
      });
      if (res.ok) {
        const data = await res.json();
        setActiveRoom(data.room);
      }
    },
    [address]
  );

  const fetchRoom = useCallback(
    async (roomId: string) => {
      const res = await fetch(`/api/matchmaking?roomId=${roomId}`);
      if (res.ok) {
        const data = await res.json();
        setActiveRoom(data.room);
        return data.room as ActiveRoom;
      }
      return null;
    },
    []
  );

  const clearRoom = useCallback(() => {
    setActiveRoom(null);
    setMyQueuePosition(null);
  }, []);

  return {
    waitingPlayers,
    queueLength,
    myQueuePosition,
    activeRoom,
    isJoining,
    error,
    joinQueue,
    leaveQueue,
    syncRoom,
    fetchRoom,
    clearRoom,
    poll,
  };
}