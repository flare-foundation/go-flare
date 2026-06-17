'use client';

type StockfishCallback = (bestMove: string) => void;

class StockfishEngine {
  private worker: Worker | null = null;
  private callback: StockfishCallback | null = null;
  private isReady = false;
  private pendingFen: string | null = null;
  private pendingDepth: number = 10;

  initialize(): Promise<void> {
    return new Promise((resolve, reject) => {
      try {
        // Use stockfish.js as a web worker
        this.worker = new Worker('/stockfish.js');
        
        this.worker.onmessage = (e: MessageEvent) => {
          const message = e.data as string;
          
          if (message === 'readyok' || message === 'uciok') {
            this.isReady = true;
            resolve();
          }
          
          if (message.startsWith('bestmove')) {
            const parts = message.split(' ');
            const bestMove = parts[1];
            if (bestMove && bestMove !== '(none)' && this.callback) {
              this.callback(bestMove);
            }
          }
        };

        this.worker.onerror = (e) => {
          console.error('Stockfish worker error:', e);
          reject(e);
        };

        this.worker.postMessage('uci');
        this.worker.postMessage('isready');
      } catch (error) {
        reject(error);
      }
    });
  }

  getBestMove(fen: string, depth: number, callback: StockfishCallback): void {
    this.callback = callback;
    
    if (!this.worker || !this.isReady) {
      this.pendingFen = fen;
      this.pendingDepth = depth;
      return;
    }

    this.worker.postMessage('stop');
    this.worker.postMessage(`position fen ${fen}`);
    this.worker.postMessage(`go depth ${depth}`);
  }

  setSkillLevel(level: number): void {
    if (!this.worker) return;
    // Stockfish skill level 0-20 maps to our 1-10 slider
    const sfLevel = Math.round((level / 10) * 20);
    this.worker.postMessage(`setoption name Skill Level value ${sfLevel}`);
  }

  stop(): void {
    this.worker?.postMessage('stop');
  }

  terminate(): void {
    this.worker?.postMessage('quit');
    this.worker?.terminate();
    this.worker = null;
    this.isReady = false;
  }
}

// Singleton
let engine: StockfishEngine | null = null;

export function getStockfishEngine(): StockfishEngine {
  if (!engine) {
    engine = new StockfishEngine();
  }
  return engine;
}

export type { StockfishCallback };
