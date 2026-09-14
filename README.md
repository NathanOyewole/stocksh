# STOCK.sh — Terminal xStocks on Solana

> Blazing-fast, keyboard-only terminal execution engine for tokenized stocks (xStocks) on Solana.

Built for **Stocklana** · Sept 11–18 2026 · $100K prize pool

## The Wedge

Retail brokerage apps are slow, bloated, and closed on weekends.  
`stock.sh` gives crypto/TradFi degens a **24/7 high-speed CLI** to swap USDC ↔ tokenized equities without ever touching a mouse.

## Current Mode (safe defaults)

- **Network**: Devnet by default
- **Execution**: Dry-run by default (prepares fully-formed Jupiter swap tx but does **not** broadcast)
- **Quotes & Prices**: Real mainnet Jupiter so numbers are realistic

## Quick Start

```bash
git clone https://github.com/NathanOyewole/stocksh.git
cd stocksh
go run .
```

### Controls

| Key          | Action                          |
|--------------|---------------------------------|
| ↑ ↓ / j k    | Navigate tickers                |
| Enter / Space| Fetch live quote                |
| + / -        | Change USDC size (±5)           |
| s            | Toggle Buy / Sell               |
| p            | Portfolio view                  |
| r            | Refresh prices                  |
| a            | Airdrop 1 SOL (Devnet)          |
| y / Enter    | Confirm & prepare swap tx       |
| Esc / n      | Cancel / back                   |
| q            | Quit                            |

## Live Features

- Live USD prices + 24h change via Jupiter Price API
- Buy & Sell flows
- Portfolio (SOL balance + airdrop)
- Dry-run / Live toggle via `STOCKSH_LIVE=1`

## Architecture

```
stock.sh/
├── main.go
├── ui/          # Bubbletea + Lipgloss TUI
├── jupiter/     # Quote + Swap + Prices
└── solana/      # Wallet, airdrop, sign+send
```

## Notes

- xStocks liquidity lives on **mainnet**. Devnet has no real pools.
- Free Jupiter tier is rate-limited — fine for demo.
- xStocks are **not available to US persons**. DYOR.

**Keep building.**
