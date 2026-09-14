# STOCK.sh — Terminal xStocks on Solana

> Blazing-fast, keyboard-only terminal execution engine for tokenized stocks (xStocks) on Solana.

Built for **Stocklana** · Sept 11–18 2026 · $100K prize pool

## Quick Start

```bash
git clone https://github.com/NathanOyewole/stocksh.git
cd stocksh
cp .env.example .env   # optional: add SOLANA_PRIVATE_KEY
go mod tidy
go build -o stocksh .
./stocksh              # Windows: stocksh.exe
```

## Controls

| Key | Action |
|-----|--------|
| ↑↓ / jk | Navigate |
| Enter | Quote |
| +/- | Size |
| s | Buy/Sell toggle |
| y | Confirm |
| p | Portfolio |
| ? / h | Help |
| r | Refresh |
| a | Airdrop (Devnet) |
| q | Quit |

Defaults: **Devnet + Dry-run**. Live prices from Jupiter mainnet.
