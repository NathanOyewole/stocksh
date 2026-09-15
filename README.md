# STOCK.sh — Terminal xStocks on Solana

Keyboard-only TUI for trading tokenized stocks (xStocks) on Solana.
Built for **Stocklana** hackathon.

The app boots to a full-screen splash (logo, wallet status, market-data
feed), then drops you straight into the ticker view — fully wired up in
the background so prices and balances are already warm by the time you
start trading.

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
| any-key / Esc | Skip splash screen |
| ↑↓ / j k | Navigate |
| Enter | Get quote |
| +/- | Change size |
| s | Toggle BUY/SELL |
| y | Confirm / prepare tx |
| p | Portfolio |
| t | Trade history |
| w | Watchlist + alerts |
| * | Star / track symbol |
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
├── ledger/      # Trade history + position P&L math
└── solana/      # Wallet + RPC
```

## Portfolio & Paper Trading

Every dry-run swap is recorded as a paper position in
`~/.stocksh/trades.json`. The `p` screen shows:

- Per-token **market value** at live prices
- **Average cost basis** and **unrealized P&L** ($ and %)
- **Allocation bars** (% of total portfolio value)
- **24-hour portfolio change** estimate

The ticker screen refreshes prices every **5 seconds** and draws a live
**trend sparkline** per symbol so you can watch momentum build in real
time.

## Watchlist & price alerts

Star symbols with `*` on the ticker screen, then open the watchlist with
`w`:

- `+` / `-` set a **target price alert** on the selected symbol (in $5 steps)
- `x` clears the alert, `*` untracks the symbol
- when a price crosses its target you'll get an alert in the status bar,
  even from any other screen | `p` · history `t`

In live mode (`STOCKSH_LIVE=1`) real on-chain balances are used,
and P&L is still computed from whatever history the ledger holds.

## Notes

- xStocks liquidity is **mainnet-only**
- Dry-run never broadcasts but records paper trades
- For live: `STOCKSH_LIVE=1` + mainnet RPC + funded key
