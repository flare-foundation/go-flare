'use client';

import { useEffect, useRef } from 'react';
import { motion, AnimatePresence } from 'framer-motion';
import type { MoveHistory } from '@/types/chess';

interface MoveListProps {
  moves: MoveHistory[];
}

// Group moves into pairs (white + black)
function groupMoves(moves: MoveHistory[]) {
  const pairs: { number: number; white?: MoveHistory; black?: MoveHistory }[] = [];
  for (let i = 0; i < moves.length; i += 2) {
    pairs.push({
      number: Math.floor(i / 2) + 1,
      white: moves[i],
      black: moves[i + 1],
    });
  }
  return pairs;
}

function MoveCell({ move, isLatest }: { move?: MoveHistory; isLatest: boolean }) {
  if (!move) return <div className="w-full px-2 py-1" />;

  const isCapture = move.san.includes('x');
  const isCheck = move.san.includes('+');
  const isMate = move.san.includes('#');

  return (
    <motion.div
      initial={{ opacity: 0, x: -4 }}
      animate={{ opacity: 1, x: 0 }}
      className="w-full px-2 py-1 rounded text-sm font-mono"
      style={{
        background: isLatest ? 'var(--gold-dim)' : 'transparent',
        color: isLatest
          ? 'var(--gold-secondary)'
          : isMate
          ? '#ff6b6b'
          : isCheck
          ? '#ffa94d'
          : 'var(--text-primary)',
        fontWeight: isLatest ? 600 : 400,
      }}
    >
      {move.san}
    </motion.div>
  );
}

export function MoveList({ moves }: MoveListProps) {
  const scrollRef = useRef<HTMLDivElement>(null);
  const pairs = groupMoves(moves);

  useEffect(() => {
    if (scrollRef.current) {
      scrollRef.current.scrollTop = scrollRef.current.scrollHeight;
    }
  }, [moves.length]);

  return (
    <div className="glass rounded-xl overflow-hidden flex flex-col" style={{ height: '260px' }}>
      <div className="px-4 py-3 border-b" style={{ borderColor: 'var(--bg-glass-border)' }}>
        <span className="text-xs uppercase tracking-widest" style={{ color: 'var(--text-secondary)' }}>
          Move History
        </span>
      </div>

      <div
        ref={scrollRef}
        className="flex-1 overflow-y-auto p-2"
        style={{ overscrollBehavior: 'contain' }}
      >
        {pairs.length === 0 ? (
          <div className="flex items-center justify-center h-full">
            <span className="text-xs" style={{ color: 'var(--text-secondary)' }}>
              No moves yet
            </span>
          </div>
        ) : (
          <AnimatePresence>
            {pairs.map(({ number, white, black }, i) => (
              <motion.div
                key={number}
                initial={{ opacity: 0 }}
                animate={{ opacity: 1 }}
                className="flex items-center gap-1 mb-0.5"
              >
                {/* Move number */}
                <span
                  className="text-xs w-6 shrink-0 text-right pr-1"
                  style={{ color: 'var(--text-secondary)' }}
                >
                  {number}.
                </span>

                {/* White move */}
                <div className="flex-1">
                  <MoveCell
                    move={white}
                    isLatest={
                      i === pairs.length - 1 &&
                      moves.length % 2 !== 0
                        ? true
                        : false
                    }
                  />
                </div>

                {/* Black move */}
                <div className="flex-1">
                  <MoveCell
                    move={black}
                    isLatest={i === pairs.length - 1 && moves.length % 2 === 0 && !!black}
                  />
                </div>
              </motion.div>
            ))}
          </AnimatePresence>
        )}
      </div>
    </div>
  );
}
