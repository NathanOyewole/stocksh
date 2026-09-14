# STOCK.sh — Terminal xStocks on Solana

Keyboard-only TUI for trading tokenized stocks (xStocks) on Solana.
Built for **Stocklana** hackathon.

## Install

```bash
git clone https://github.com/NathanOyewole/stocksh.git
cd stocksh
go mod tidy
go build -o stocksh .
# Windows:
go build -o stocksh.exe .
```

## Run

```bash
./stocksh          # Linux/Mac
.\stocksh.exe      # Windows
```

Defaults: **Devnet + Dry-run**. Live prices from Jupiter mainnet.

### Optional .env

```bash
cp .env.example .env
# then edit:
# SOLANA_PRIVATE_KEY=your_base58_key
# SOLANA_RPC=https://api.devnet.solana.com
# STOCKSH_LIVE=0
```

## Controls

| Key | Action |
|-----|--------|
| ↑↓ / j k | Navigate |
| Enter | Get quote |
| +/- | Change size |
| s | Toggle BUY/SELL |
| y | Confirm / prepare tx |
| p | Portfolio |
| ? / h | Help |
| r | Refresh |
| a | Airdrop 1 SOL (Devnet) |
| Esc | Back |
| q | Quit |

## Structure

```
stocksh/
├── main.go
├── ui/          # Bubbletea TUI
├── jupiter/     # Quotes + swaps
└── solana/      # Wallet + RPC
```

## Notes

- xStocks liquidity is **mainnet-only**
- Dry-run never broadcasts
- For live: `STOCKSH_LIVE=1` + mainnet RPC + funded key
