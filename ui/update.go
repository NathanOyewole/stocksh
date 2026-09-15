package ui

import (
	"fmt"
	"strconv"
	"strings"

	tea "github.com/charmbracelet/bubbletea"

	"stocksh/solana"
)

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil
	case splashMsg:
		m.splashTicks += msg.n
		if m.splashTicks >= 2 {
			m.mode = viewTickers
			m.status = "Ready to trade"
			return m, nil
		}
		return m, splashCmd()
	case tea.KeyMsg:
		if m.mode == viewSplash {
			m.mode = viewTickers
			if m.wallet != nil {
				m.status = fmt.Sprintf("Wallet %s \u00b7 %s \u00b7 DRY-RUN", truncatePubkey(m.wallet.PubKey.String()), m.network)
			} else {
				m.status = "No wallet found - set SOLANA_PRIVATE_KEY for live trading"
			}
			return m, nil
		}
		switch m.mode {
		case viewTickers:
			return m.updateTickers(msg)
		case viewCustomAmount:
			return m.updateCustomAmount(msg)
		case viewConfirm:
			return m.updateConfirm(msg)
		case viewResult:
			return m.updateResult(msg)
		case viewPortfolio:
			return m.updatePortfolio(msg)
		case viewHistory:
			return m.updateHistory(msg)
		case viewWatchlist:
			return m.updateWatchlist(msg)
		case viewHelp:
			return m.updateHelp(msg)
		}
	case walletLoadedMsg:
		if msg.err != nil {
			m.status = "No wallet (set SOLANA_PRIVATE_KEY for live / portfolio)"
			m.walletStatus = "No wallet found"
		} else {
			m.wallet = msg.wallet
			mode := "DRY-RUN"
			if !m.dryRun {
				mode = "LIVE"
			}
			m.status = fmt.Sprintf("Wallet %s \u00b7 %s \u00b7 %s", truncatePubkey(msg.wallet.PubKey.String()), m.network, mode)
			m.walletStatus = fmt.Sprintf("Wallet connected \u00b7 %s", mode)
			return m, m.fetchBalance()
		}
		return m, nil
	case balanceMsg:
		if msg.err != nil {
			m.errMsg = msg.err.Error()
		} else {
			m.errMsg = ""
			m.solBalance = msg.sol
			m.solLoaded = true
		}
		return m, nil
	case tokensMsg:
		if msg.err == nil {
			m.tokenBalances = msg.balances
			m.tokensLoaded = true
		}
		return m, nil
	case pricesMsg:
		if msg.err == nil && msg.prices != nil {
			for i, t := range m.tickers {
				if p, ok := msg.prices[t.Mint]; ok {
					m.tickers[i].PriceV = p.USDPrice
					m.tickers[i].Price = fmt.Sprintf("$%.2f", p.USDPrice)
					m.tickers[i].ChgV = p.PriceChange24h
					if p.PriceChange24h >= 0 {
						m.tickers[i].Change = fmt.Sprintf("+%.1f%%", p.PriceChange24h)
					} else {
						m.tickers[i].Change = fmt.Sprintf("%.1f%%", p.PriceChange24h)
					}
					m.tickers[i].History = append(t.History, p.USDPrice)
					if len(m.tickers[i].History) > 42 {
						m.tickers[i].History = m.tickers[i].History[len(m.tickers[i].History)-42:]
					}
					m.checkAlert(i)
				}
			}
		}
		return m, nil
	case quoteMsg:
		if msg.err != nil {
			m.errMsg = msg.err.Error()
			m.status = "Quote failed"
			m.mode = viewTickers
			return m, nil
		}
		m.quote = msg.quote
		m.errMsg = ""
		m.mode = viewConfirm
		m.status = "Quote ready \u2014 y to prepare \u00b7 Esc cancel"
		return m, nil
	case swapResultMsg:
		if msg.err != nil {
			m.errMsg = msg.err.Error()
			m.status = "Swap prep failed"
			m.mode = viewResult
			return m, nil
		}
		m.lastSig = msg.sig
		m.errMsg = ""
		m.status = "READY"
		m.mode = viewResult
		m.recordTrade()
		return m, nil
	case tickMsg:
		cmds := []tea.Cmd{tickCmd(), m.fetchAllPrices()}
		if m.wallet != nil && m.mode == viewPortfolio {
			cmds = append(cmds, m.fetchBalance())
			if !m.dryRun {
				cmds = append(cmds, m.fetchTokens())
			}
		}
		return m, tea.Batch(cmds...)
	}
	return m, nil
}

func (m Model) updateTickers(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "q", "ctrl+c":
		return m, tea.Quit
	case "up", "k":
		if m.cursor > 0 {
			m.cursor--
		}
	case "down", "j":
		if m.cursor < len(m.tickers)-1 {
			m.cursor++
		}
	case "enter", " ":
		m.selected = m.tickers[m.cursor]
		m.errMsg = ""
		m.status = fmt.Sprintf("Quoting %s\u2026", m.selected.Symbol)
		return m, m.fetchQuote()
	case "+", "=":
		m.amountUSDC += 5
		if m.amountUSDC > 500 {
			m.amountUSDC = 500
		}
	case "-", "_":
		if m.amountUSDC > 5 {
			m.amountUSDC -= 5
		}
	case "s":
		if m.side == "buy" {
			m.side = "sell"
		} else {
			m.side = "buy"
		}
	case "p":
		m.mode = viewPortfolio
		m.status = "Portfolio"
		if m.wallet != nil {
			cmds := []tea.Cmd{m.fetchBalance()}
			if !m.dryRun {
				cmds = append(cmds, m.fetchTokens())
			}
			return m, tea.Batch(cmds...)
		}
		return m, nil
	case "t":
		m.mode = viewHistory
		m.status = "Trade history"
		return m, nil
	case "w":
		m.mode = viewWatchlist
		m.status = "Watchlist"
		return m, nil
	case "*":
		m.tickers[m.cursor].Watch = !m.tickers[m.cursor].Watch
		sym := m.tickers[m.cursor].Symbol
		if m.tickers[m.cursor].Watch {
			m.status = fmt.Sprintf("%s added to watchlist (%s to set alert)", sym, "w")
		} else {
			m.status = fmt.Sprintf("%s removed from watchlist", sym)
			m.tickers[m.cursor].Alert = 0
		}
		return m, nil
	case "r":
		return m, m.fetchAllPrices()
	case "?", "h":
		m.mode = viewHelp
		return m, nil
	case "c":
		m.mode = viewCustomAmount
		m.customBuf = ""
		m.status = "Custom size \u2014 type digits, Enter to set, Esc to cancel"
		return m, nil
	case "0":
		m.mode = viewSplash
		m.splashTicks = 0
		return m, nil
	}
	return m, nil
}

func (m Model) updateCustomAmount(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "q", "ctrl+c":
		return m, tea.Quit
	case "esc":
		m.mode = viewTickers
		m.customBuf = ""
		m.errMsg = ""
		return m, nil
	case "enter":
		if m.customBuf == "" {
			m.mode = viewTickers
			m.errMsg = ""
			return m, nil
		}
		val, err := strconv.ParseFloat(m.customBuf, 64)
		if err != nil || val < 0.01 || val > 50000 {
			m.errMsg = "invalid amount \u2014 use 0.01 to 50000 USDC"
			m.customBuf = ""
			return m, nil
		}
		m.amountUSDC = val
		m.mode = viewTickers
		m.customBuf = ""
		m.errMsg = ""
		m.status = fmt.Sprintf("Size set to $%.2f USDC", val)
		return m, nil
	case "backspace":
		if len(m.customBuf) > 0 {
			m.customBuf = m.customBuf[:len(m.customBuf)-1]
		}
		return m, nil
	default:
		ch := msg.String()
		if len(ch) == 1 {
			c := ch[0]
			if (c >= '0' && c <= '9') || (c == '.' && !strings.Contains(m.customBuf, ".")) {
				if len(m.customBuf) < 9 {
					m.customBuf += ch
				}
			}
		}
		return m, nil
	}
}

func (m Model) updateConfirm(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "q", "ctrl+c":
		return m, tea.Quit
	case "esc", "n", "backspace":
		m.mode = viewTickers
		m.quote = nil
		return m, nil
	case "y", "enter":
		return m, m.executeSwap()
	}
	return m, nil
}

func (m Model) updateResult(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "q", "ctrl+c":
		return m, tea.Quit
	case "esc", "enter", "n", "y", " ":
		m.mode = viewTickers
		m.quote = nil
		m.lastSig = ""
		m.errMsg = ""
	}
	return m, nil
}

func (m Model) updatePortfolio(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "q", "ctrl+c":
		return m, tea.Quit
	case "esc", "p", "enter", " ":
		m.mode = viewTickers
		return m, nil
	case "a":
		if m.wallet == nil {
			m.errMsg = "no wallet found — set SOLANA_PRIVATE_KEY or place id.json at ~/.config/solana/id.json"
			return m, nil
		}
		if m.network != "devnet" {
			m.errMsg = "airdrop is devnet-only — use default config or set SOLANA_RPC to an endpoint containing 'devnet'"
			return m, nil
		}
		m.errMsg = ""
		m.status = "Requesting airdrop (trying fallback endpoints)..."
		return m, m.doAirdrop()
	case "r":
		cmds := []tea.Cmd{m.fetchBalance(), m.fetchAllPrices()}
		if !m.dryRun {
			cmds = append(cmds, m.fetchTokens())
		}
		return m, tea.Batch(cmds...)
	}
	return m, nil
}

func (m Model) updateHistory(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "q", "ctrl+c":
		return m, tea.Quit
	case "esc", "t", "p", "enter", " ":
		m.mode = viewTickers
		return m, nil
	}
	return m, nil
}

// checkAlert fires a status alert when a watchlisted symbol's price reaches
// its configured target. Re-arms once the price drops back below the target.
func (m *Model) checkAlert(i int) {
	t := &m.tickers[i]
	if !t.Watch || t.Alert <= 0 {
		return
	}
	above := t.PriceV >= t.Alert
	if above && !t.alertUp {
		m.errMsg = ""
		m.status = fmt.Sprintf("ALERT: %s reached $%.2f (target $%.2f)", t.Symbol, t.PriceV, t.Alert)
	}
	t.alertUp = above
}

func (m Model) updateWatchlist(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "q", "ctrl+c":
		return m, tea.Quit
	case "esc", "w", "enter", " ":
		m.mode = viewTickers
		return m, nil
	case "*":
		// untrack the current watchlist selection
		if i := m.watchTickerIndex(); i >= 0 {
			m.tickers[i].Watch = false
			m.tickers[i].Alert = 0
			m.status = fmt.Sprintf("%s removed from watchlist", m.tickers[i].Symbol)
			if m.watchCursor > 0 {
				m.watchCursor--
			}
		}
		return m, nil
	case "+", "=", "-", "_":
		return m, m.adjustAlertTarget(msg.String())
	case "x":
		if i := m.watchTickerIndex(); i >= 0 {
			m.tickers[i].Alert = 0
			m.tickers[i].alertUp = false
			m.status = fmt.Sprintf("Alert cleared for %s", m.tickers[i].Symbol)
		}
		return m, nil
	case "up", "k":
		if m.watchCursor > 0 {
			m.watchCursor--
		}
		return m, nil
	case "down", "j":
		if m.watchCursor < m.watchCount()-1 {
			m.watchCursor++
		}
		return m, nil
	}
	return m, nil
}

func (m *Model) watchTickerIndex() int {
	n := 0
	for i := range m.tickers {
		if !m.tickers[i].Watch {
			continue
		}
		if n == m.watchCursor {
			return i
		}
		n++
	}
	return -1
}

func (m *Model) watchCount() int {
	n := 0
	for i := range m.tickers {
		if m.tickers[i].Watch {
			n++
		}
	}
	return n
}

// adjustAlertTarget bumps the alert for the current watchlist selection by
// $5 steps (rounded to whole dollars).
func (m Model) adjustAlertTarget(key string) tea.Cmd {
	i := m.watchTickerIndex()
	if i < 0 {
		m.status = "Star a symbol with * first"
		return nil
	}
	step := 5.0
	if key == "-" || key == "_" {
		step = -5.0
	}
	if m.tickers[i].Alert <= 0 {
		// start near the current price, rounded up to the nearest step
		m.tickers[i].Alert = float64(int(m.tickers[i].PriceV/step))*step + step
	} else {
		m.tickers[i].Alert = float64(int(m.tickers[i].Alert/step))*step + step
	}
	if m.tickers[i].Alert < 0 {
		m.tickers[i].Alert = 0
	}
	m.tickers[i].alertUp = m.tickers[i].PriceV >= m.tickers[i].Alert
	m.status = fmt.Sprintf("%s target: $%.2f", m.tickers[i].Symbol, m.tickers[i].Alert)
	return nil
}

func (m Model) updateHelp(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "q", "ctrl+c":
		return m, tea.Quit
	case "esc", "?", "h", "enter", " ":
		m.mode = viewTickers
		return m, nil
	}
	return m, nil
}

func (m Model) doAirdrop() tea.Cmd {
	return func() tea.Msg {
		_, err := solana.RequestAirdrop(m.wallet.PubKey, 1_000_000_000)
		if err != nil {
			return balanceMsg{err: err}
		}
		client := solana.NewRPC("")
		bal, err := solana.GetBalance(client, m.wallet.PubKey)
		return balanceMsg{sol: bal, err: err}
	}
}

func truncatePubkey(pk string) string {
	if len(pk) > 12 {
		return pk[:6] + "\u2026" + pk[len(pk)-4:]
	}
	return pk
}

// recordTrade writes the last executed swap into the paper/live ledger so
// the portfolio has a cost basis to compute P&L from. Amounts come from the
// Jupiter quote (both are 1e6 lamports).
func (m *Model) recordTrade() {
	if m.quote == nil {
		return
	}
	inAmt, err := strconv.ParseUint(m.quote.InAmount, 10, 64)
	if err != nil || inAmt == 0 {
		return
	}
	outAmt, err := strconv.ParseUint(m.quote.OutAmount, 10, 64)
	if err != nil || outAmt == 0 {
		return
	}
	var signedQty, price float64
	if m.side == "buy" {
		price = float64(inAmt) / float64(outAmt)
		signedQty = float64(outAmt) / 1_000_000
	} else {
		price = float64(outAmt) / float64(inAmt)
		signedQty = -float64(inAmt) / 1_000_000
	}
	if err := m.led.Record(m.selected.Symbol, m.side, signedQty, price); err != nil {
		m.errMsg = fmt.Sprintf("Could not persist trade: %v", err)
	}
}
