import { NextRequest, NextResponse } from 'next/server';
import type { ActiveRoom, MatchmakingEntry } from '@/types/matchmaking';

const globalStore = globalThis as typeof globalThis & {
  __titanChessQueue?: MatchmakingEntry[];
  __titanChessRooms?: Map<string, ActiveRoom>;
};

const queue = globalStore.__titanChessQueue ?? (globalStore.__titanChessQueue = []);
const rooms = globalStore.__titanChessRooms ?? (globalStore.__titanChessRooms = new Map());

function shortAddr(addr: string) {
  return `${addr.slice(0, 6)}…${addr.slice(-4)}`;
}

function makeRoomId() {
  return `room-${Date.now()}-${Math.random().toString(36).slice(2, 8)}`;
}

function removeFromQueue(address: string) {
  const idx = queue.findIndex((e) => e.address.toLowerCase() === address.toLowerCase());
  if (idx >= 0) queue.splice(idx, 1);
}

export async function GET(req: NextRequest) {
  const address = req.nextUrl.searchParams.get('address')?.toLowerCase();
  const roomId = req.nextUrl.searchParams.get('roomId');

  if (roomId) {
    const room = rooms.get(roomId);
    if (!room) {
      return NextResponse.json({ error: 'Room not found' }, { status: 404 });
    }
    return NextResponse.json({ room });
  }

  const waiting = queue.filter((e) => e.address.toLowerCase() !== address);
  const myEntry = address ? queue.find((e) => e.address.toLowerCase() === address) : undefined;
  const myRoom = address
    ? [...rooms.values()].find(
        (r) =>
          r.status === 'active' &&
          (r.white.toLowerCase() === address || r.black.toLowerCase() === address)
      )
    : undefined;

  return NextResponse.json({
    queueLength: queue.length,
    waitingPlayers: waiting.map((e) => ({
      address: e.address,
      shortAddress: shortAddr(e.address),
      stake: e.stake,
      joinedAt: e.joinedAt,
    })),
    myQueuePosition: myEntry ? queue.indexOf(myEntry) + 1 : null,
    myRoom: myRoom ?? null,
  });
}

export async function POST(req: NextRequest) {
  const body = await req.json();
  const { action, address, stake, roomId, fen, lastMove, winner } = body as {
    action: string;
    address?: string;
    stake?: string;
    roomId?: string;
    fen?: string;
    lastMove?: { from: string; to: string } | null;
    winner?: 'white' | 'black' | 'draw';
  };

  if (!address) {
    return NextResponse.json({ error: 'Address required' }, { status: 400 });
  }

  const addr = address.toLowerCase();

  if (action === 'join') {
    if (!stake) {
      return NextResponse.json({ error: 'Stake required' }, { status: 400 });
    }

    removeFromQueue(addr);

    const opponentIdx = queue.findIndex(
      (e) => e.address.toLowerCase() !== addr && e.stake === stake
    );

    if (opponentIdx >= 0) {
      const opponent = queue.splice(opponentIdx, 1)[0];
      const id = makeRoomId();
      const room: ActiveRoom = {
        id,
        white: opponent.address,
        black: addr,
        stake,
        fen: 'rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNR w KQkq - 0 1',
        lastMove: null,
        status: 'active',
        winner: null,
        createdAt: Date.now(),
      };
      rooms.set(id, room);
      return NextResponse.json({
        matched: true,
        room,
        playerColor: 'b' as const,
        opponent: { address: opponent.address, shortAddress: shortAddr(opponent.address) },
      });
    }

    queue.push({ address: addr, stake, joinedAt: Date.now() });
    return NextResponse.json({
      matched: false,
      queuePosition: queue.length,
    });
  }

  if (action === 'leave') {
    removeFromQueue(addr);
    return NextResponse.json({ ok: true });
  }

  if (action === 'sync') {
    if (!roomId || !fen) {
      return NextResponse.json({ error: 'roomId and fen required' }, { status: 400 });
    }
    const room = rooms.get(roomId);
    if (!room) {
      return NextResponse.json({ error: 'Room not found' }, { status: 404 });
    }
    if (room.white.toLowerCase() !== addr && room.black.toLowerCase() !== addr) {
      return NextResponse.json({ error: 'Not in room' }, { status: 403 });
    }

    room.fen = fen;
    room.lastMove = lastMove ?? null;
    if (winner) {
      room.status = 'finished';
      room.winner = winner;
    }
    rooms.set(roomId, room);
    return NextResponse.json({ room });
  }

  return NextResponse.json({ error: 'Unknown action' }, { status: 400 });
}