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
| c | Type a custom USDC size |
| s | Toggle BUY/SELL |
| y | Confirm / prepare tx |
| p | Portfolio |
| t | Trade history |
| w | Watchlist + alerts |
| i | Activity feed (trades, airdrops, alerts) |
| * | Star / track symbol |
| 0 | Back to splash screen (a digit in custom-size input) |
| a | Airdrop SOL — paper +1 without a wallet, real devnet faucet with one |
| ? / h | Help |
| r | Refresh |
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
- when a price crosses its target you'll get an alert in the status bar
  even from any other screen (portfolio `p` · history `t`)

In live mode (`STOCKSH_LIVE=1`) real on-chain balances are used,
and P&L is still computed from whatever history the ledger holds.

## UI layout

The whole app runs on one **full-screen responsive frame** that always
fits the terminal exactly (resize it and every view re-lays-out on the
fly; nothing scrolls or leaves ghost fragments behind).

- **Header row** — `[ STOCK.sh ] Terminal xStocks` on the left, badges
  flush to the right edge (`[ DEVNET ]`/`[ MAINNET ]`, `[ DRY-RUN ]`/`[ LIVE ]`)
- **Ticker table** — 10 columns:
  `SYMBOL · PRICE · 24H · TREND (sparkline) · LIQ · MARK · POS · POS VAL · P&L · SIZE`
  - `LIQ` is real USDC liquidity from Jupiter; `MARK` is the stock oracle
    price (distinct from Jupiter's swap price)
  - `POS` / `POS VAL` / `P&L` reflect your paper (or live) holdings per symbol
  - columns shrink responsively on narrow terminals; the selected row is
    highlighted full-width
- **Docked hint footer** — minimal bottom bar inside the frame showing
  `[?] Help · [q] Quit` plus a live-refresh note
- **Status bar** — a single row pinned below the frame border for status
  text, errors, and size-lock/log messages (never scrolls the terminal)

## Demo walkthrough

A 90-second scripted run that shows off the whole app. Works with the
default **Devnet + Dry-run** setup — no wallet needed for the paper flow.

```bash
# 1. Boot (any key to skip the splash once it appears)
.\stocksh.exe

# 2. Ticker view — prices warm up within 5s, sparklines tick along
#    + / -        bump order size
#    s            flip BUY -> SELL (see it toggle in the header)
#    c            type a one-shot custom USDC size (reverts after the trade)

# 3. Get a live quote
#    Enter  on NVDAx -> quote panel: price, 24h change, size, account

# 4. Simulated buy (no wallet needed — paper SOL account is seeded)
#    y               confirm -> "DRY-RUN OK - tx prepared" and a paper
#                   position is recorded against the paper SOL balance

# 5. Portfolio P&L
#    p               value / avg cost / unrealized P&L / allocation bars

# 6. Trend sparklines
#    Esc              back to tickers, watch the sparkline crawl every 5s

# 7. Watchlist + price alert
#    a                airdrop +1 SOL (paper without a wallet; real devnet faucet with one)
#    *                star the symbol, w to open watchlist
#    +                 raise target alert $5 at a time

# 8. Trade history + activity feed
#    t                fills from the paper ledger (~/.stocksh/trades.json)
#    i                activity feed: trades, airdrops, alerts, size locks

# 9. Out
#    q                exit
```

Flip to live — with `STOCKSH_LIVE=1`, a mainnet RPC, and a funded
keypair — and the same keys run a **real xStocks swap**. The `p`
portfolio screen always reflects live prices, so a demo can start in
paper mode and go live by just swapping env vars.

## Notes

- xStocks liquidity is **mainnet-only**
- Dry-run never broadcasts but records paper trades
- For live: `STOCKSH_LIVE=1` + mainnet RPC + funded key
