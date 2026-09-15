package ui

import (
	"fmt"
	"strconv"
	"strings"

	tea "github.com/charmbracelet/bubbletea"

	"stocksh/jupiter"
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
		case viewActivity:
			return m.updateActivity(msg)
		}
	case walletLoadedMsg:
		if msg.err != nil {
			m.status = "No wallet (set SOLANA_PRIVATE_KEY for live / portfolio)"
			m.walletStatus = "No wallet found"
			// Devnet dry-run still needs a SOL account to debit: seed a paper
			// balance once so buys/sells settle against it.
			if !m.led.SolSeeded && m.dryRun {
				if err := m.led.SeedSol(100); err == nil {
					m.solBalance = 100
					m.solLoaded = true
					_ = m.led.RecordActivity("size", "Paper SOL account opened with 100 SOL")
				}
			}
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
			return m, nil
		}
		// The ledger is the source of truth for the displayed SOL balance, so
		// it only seeds the on-chain number once as a baseline. After that the
		// app's buys/sells/airdrops adjust it, and it persists across restarts.
		if msg.airdrop {
			if m.led.SolSeeded {
				_ = m.led.AdjSol(1) // faucet delta on top of the persisted baseline
			} else {
				_ = m.led.SeedSol(msg.sol) // no baseline yet: the post-airdrop balance becomes it
			}
			_ = m.led.RecordActivity("airdrop",
				fmt.Sprintf("Devnet faucet +1 SOL (balance %.4f)", m.led.SolBalance))
		} else if !m.led.SolSeeded {
			_ = m.led.SeedSol(msg.sol)
		}
		m.solBalance = m.led.SolBalance
		m.solLoaded = true
		return m, nil
	case tokensMsg:
		if msg.err == nil {
			m.tokenBalances = msg.balances
			m.tokensLoaded = true
		}
		return m, nil
	case pricesMsg:
		if msg.err == nil && msg.prices != nil {
			if p, ok := msg.prices[jupiter.SolMint]; ok {
				m.solPrice = p.USDPrice
			}
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
		m.mode = viewResult
		m.orderUSDC = 0 // one-shot order consumed
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
			_ = m.led.RecordActivity("watch", fmt.Sprintf("%s added to watchlist", sym))
		} else {
			m.status = fmt.Sprintf("%s removed from watchlist", sym)
			m.tickers[m.cursor].Alert = 0
			_ = m.led.RecordActivity("watch", fmt.Sprintf("%s removed from watchlist", sym))
		}
		return m, nil
	case "i":
		m.mode = viewActivity
		m.status = "Activity feed"
		return m, nil
	case "r":
		return m, m.fetchAllPrices()
	case "?", "h":
		m.mode = viewHelp
		return m, tea.ClearScreen
	case "c":
		m.mode = viewCustomAmount
		m.customBuf = ""
		m.status = "Custom size — type digits, Enter to lock next order, Esc to cancel"
		return m, nil
	case "a":
		return m.triggerAirdrop()
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
		m.orderUSDC = val // one-shot: applies to the next order only
		m.mode = viewTickers
		m.customBuf = ""
		m.errMsg = ""
		_ = m.led.RecordActivity("size", fmt.Sprintf("Order size locked at $%.2f USDC for %s", val, m.selected.Symbol))
		m.status = fmt.Sprintf("One-shot size $%.2f locked for %s", val, m.selected.Symbol)
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
		return m.triggerAirdrop()
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
		_ = m.led.RecordActivity("alert",
			fmt.Sprintf("%s hit target $%.2f (current $%.2f)", t.Symbol, t.PriceV, t.Alert))
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
		return m, tea.ClearScreen
	}
	return m, nil
}

func (m Model) updateActivity(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "q", "ctrl+c":
		return m, tea.Quit
	case "esc", "i", "enter", " ":
		m.mode = viewTickers
		return m, nil
	case "t":
		m.mode = viewHistory
		m.status = "Trade history"
		return m, nil
	}
	return m, nil
}

// triggerAirdrop runs a real devnet faucet airdrop when a wallet is present.
// With no wallet (paper demo) it instead credits +1 SOL to the persistent
// ledger so the balance visibly moves without a keypair on the machine.
func (m Model) triggerAirdrop() (tea.Model, tea.Cmd) {
	if m.wallet == nil {
		if m.network != "devnet" && !m.dryRun {
			m.errMsg = "no wallet found — set SOLANA_PRIVATE_KEY or place id.json at ~/.config/solana/id.json"
			return m, nil
		}
		if m.led.SolSeeded {
			_ = m.led.AdjSol(1)
		} else {
			_ = m.led.SeedSol(1)
		}
		m.solBalance = m.led.SolBalance
		m.solLoaded = true
		_ = m.led.RecordActivity("airdrop",
			fmt.Sprintf("Paper airdrop +1 SOL (balance %.4f)", m.led.SolBalance))
		m.status = "Airdrop +1 SOL (paper)"
		return m, nil
	}
	if m.network != "devnet" {
		m.errMsg = "airdrop is devnet-only — use default config or set SOLANA_RPC to an endpoint containing 'devnet'"
		return m, nil
	}
	m.errMsg = ""
	m.status = "Requesting airdrop (trying fallback endpoints)..."
	return m, m.doAirdrop()
}

func (m Model) doAirdrop() tea.Cmd {
	return func() tea.Msg {
		_, err := solana.RequestAirdrop(m.wallet.PubKey, 1_000_000_000)
		if err != nil {
			return balanceMsg{err: err, airdrop: true}
		}
		client := solana.NewRPC("")
		bal, err := solana.GetBalance(client, m.wallet.PubKey)
		return balanceMsg{sol: bal, err: err, airdrop: true}
	}
}

func truncatePubkey(pk string) string {
	if len(pk) > 12 {
		return pk[:6] + "\u2026" + pk[len(pk)-4:]
	}
	return pk
}

// recordTrade writes the last executed swap into the paper/live ledger and
// adjusts the persistent SOL balance so buys/sells settle even after the app
// is quit and relaunched.  Amounts come from the Jupiter quote (both are 1e6
// lamports).
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

	// Convert the trade into a SOL account debit / credit so the displayed
	// balance stays correct even after the app is quit and re-launched.
	var solDelta float64
	if m.side == "buy" {
		spend := float64(inAmt) / 1_000_000
		if m.solPrice > 0 {
			solDelta = spend / m.solPrice
		} else {
			solDelta = spend // fallback when SOL/USDC not loaded yet
		}
		_ = m.led.AdjSol(-solDelta)
	} else {
		spend := float64(outAmt) / 1_000_000
		if m.solPrice > 0 {
			solDelta = spend / m.solPrice
		} else {
			solDelta = spend
		}
		_ = m.led.AdjSol(solDelta)
	}
	m.solBalance = m.led.SolBalance
	side := "BUY"
	if m.side == "sell" {
		side = "SELL"
	}
	absQty := signedQty
	if absQty < 0 {
		absQty = -absQty
	}
	_ = m.led.RecordActivity("trade",
		fmt.Sprintf("%s %.4f %s @ $%.2f (SOL %.4f)", side, absQty, m.selected.Symbol, price, solDelta))
	m.status = fmt.Sprintf("%s %.4f %s \u2014 SOL balance %.4f", side, absQty, m.selected.Symbol, m.solBalance)
}
