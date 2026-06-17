'use client';

import { motion } from 'framer-motion';

interface DifficultySliderProps {
  value: number;
  onChange: (v: number) => void;
  disabled?: boolean;
}

const LEVEL_LABELS: Record<number, string> = {
  1: 'Beginner',
  2: 'Casual',
  3: 'Easy',
  4: 'Developing',
  5: 'Intermediate',
  6: 'Club',
  7: 'Advanced',
  8: 'Expert',
  9: 'Master',
  10: 'Grandmaster',
};

export function DifficultySlider({ value, onChange, disabled }: DifficultySliderProps) {
  return (
    <div className="glass rounded-xl p-4">
      <div className="flex items-center justify-between mb-3">
        <span className="text-xs uppercase tracking-widest" style={{ color: 'var(--text-secondary)' }}>
          AI Difficulty
        </span>
        <motion.span
          key={value}
          initial={{ opacity: 0, y: -4 }}
          animate={{ opacity: 1, y: 0 }}
          className="text-xs font-semibold px-2 py-0.5 rounded-full"
          style={{
            background: 'var(--gold-dim)',
            color: 'var(--gold-secondary)',
          }}
        >
          {LEVEL_LABELS[value]}
        </motion.span>
      </div>

      <div className="flex items-center gap-3">
        <span className="text-xs w-4 text-center" style={{ color: 'var(--text-secondary)' }}>1</span>
        <div className="relative flex-1">
          <input
            type="range"
            min={1}
            max={10}
            step={1}
            value={value}
            onChange={(e) => onChange(Number(e.target.value))}
            disabled={disabled}
            className="w-full h-1.5 rounded-full appearance-none cursor-pointer disabled:opacity-50 disabled:cursor-not-allowed"
            style={{
              background: `linear-gradient(to right, var(--gold-primary) 0%, var(--gold-secondary) ${(value - 1) * 11.1}%, rgba(255,255,255,0.1) ${(value - 1) * 11.1}%)`,
              WebkitAppearance: 'none',
              outline: 'none',
            }}
          />
          {/* Tick marks */}
          <div className="flex justify-between mt-1.5 px-0">
            {Array.from({ length: 10 }, (_, i) => (
              <div
                key={i}
                className="w-px h-1.5 rounded-full"
                style={{
                  background: i < value ? 'var(--gold-primary)' : 'rgba(255,255,255,0.15)',
                }}
              />
            ))}
          </div>
        </div>
        <span className="text-xs w-4 text-center" style={{ color: 'var(--text-secondary)' }}>10</span>
      </div>

      <style jsx>{`
        input[type='range']::-webkit-slider-thumb {
          -webkit-appearance: none;
          width: 16px;
          height: 16px;
          border-radius: 50%;
          background: linear-gradient(135deg, #c9a84c, #e8c97a);
          cursor: pointer;
          box-shadow: 0 0 8px rgba(201, 168, 76, 0.5);
          border: 2px solid #0f0f11;
        }
        input[type='range']::-moz-range-thumb {
          width: 16px;
          height: 16px;
          border-radius: 50%;
          background: linear-gradient(135deg, #c9a84c, #e8c97a);
          cursor: pointer;
          border: 2px solid #0f0f11;
        }
      `}</style>
    </div>
  );
}
