# ♟ Titan Chess

**Decentralized Chess on Avalanche L1 · Next.js 15 · Dark-Mode Luxury UI**

---

## Quick Start

```bash
tar -xzf titan-chess.tar.gz
cd titan-chess
npm install
npm run dev
```

Open `http://localhost:3000`

---

## Project Structure

```
titan-chess/
├── app/
│   ├── components/
│   │   ├── board/
│   │   │   └── ChessBoard.tsx      # SVG board + Framer Motion pieces
│   │   ├── hud/
│   │   │   ├── GameHUD.tsx         # Sidebar orchestrator
│   │   │   ├── TitanBalance.tsx    # Token balance (placeholder → real contract)
│   │   │   ├── DifficultySlider.tsx
│   │   │   ├── MoveList.tsx        # Algebraic notation, auto-scroll
│   │   │   └── StatusBar.tsx       # Turn / check / AI thinking indicator
│   │   ├── wallet/
│   │   │   └── WalletButton.tsx    # RainbowKit custom button
│   │   ├── ui/
│   │   │   └── GameOverOverlay.tsx # Checkmate / draw modal
│   │   └── GamePage.tsx            # Root game layout
│   ├── hooks/
│   │   └── useChessGame.ts         # All game logic (chess.js + Stockfish)
│   ├── lib/
│   │   ├── audio.ts                # Web Audio API sound engine
│   │   ├── pieces.ts               # Wikimedia SVG piece set
│   │   ├── stockfish.ts            # Stockfish worker manager
│   │   └── web3.tsx                # Wagmi + RainbowKit + Titan subnet config
│   ├── types/
│   │   └── chess.ts                # Shared TypeScript types
│   ├── globals.css                 # CSS vars, glassmorphism, animations
│   └── layout.tsx                  # Root layout with Web3Provider
├── public/
│   └── stockfish.js                # Stockfish engine (served as Web Worker)
└── next.config.ts
```

---

## Connecting Your Avalanche Subnet

Edit `app/lib/web3.tsx` and add your `.env.local`:

```env
NEXT_PUBLIC_TITAN_RPC_URL=https://your-subnet-rpc.example.com
NEXT_PUBLIC_TITAN_CHAIN_ID=12345
NEXT_PUBLIC_WALLETCONNECT_PROJECT_ID=your_project_id
```

---

## Wiring the TITAN Token Contract

In `app/components/hud/TitanBalance.tsx`, replace the mock balance:

```tsx
import { useReadContract } from 'wagmi';
import { TITAN_TOKEN_ABI } from '@/lib/abis';

const TITAN_TOKEN_ADDRESS = '0xYourContractAddress';

// Inside component:
const { data: rawBalance } = useReadContract({
  address: TITAN_TOKEN_ADDRESS,
  abi: TITAN_TOKEN_ABI,
  functionName: 'balanceOf',
  args: [address],
});
const balance = rawBalance
  ? (Number(rawBalance) / 1e18).toLocaleString('en-US', { maximumFractionDigits: 2 })
  : null;
```

---

## Sound System

All sounds are procedurally generated via Web Audio API — no files needed:

| Event       | Sound                     |
|-------------|---------------------------|
| Piece select | Soft double-tone click   |
| Move         | White noise whoosh        |
| Capture      | Low-frequency thud        |
| Check        | Three-note ascending tone |
| Checkmate    | Five-note C major arpeggio|

Replace with `.mp3` files by calling `new Audio('/sounds/move.mp3').play()` in `lib/audio.ts`.

---

## Stockfish Difficulty Mapping

| Slider (1–10) | Stockfish Skill | ELO Approximate |
|---------------|-----------------|-----------------|
| 1             | 0               | ~800            |
| 5             | 10              | ~1800           |
| 10            | 20              | ~3400 (max)     |

---

## Design Tokens

```css
--bg-primary: #0f0f11        /* deep charcoal background */
--gold-primary: #c9a84c      /* TITAN gold accent */
--gold-secondary: #e8c97a    /* hover/highlight gold */
--bronze: #8a6234            /* secondary accent */
--board-light: #f0d9b5       /* Lichess-standard board colors */
--board-dark: #b58863
```
