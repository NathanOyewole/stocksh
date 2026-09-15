package ui

import (
	"fmt"

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
		case viewConfirm:
			return m.updateConfirm(msg)
		case viewResult:
			return m.updateResult(msg)
		case viewPortfolio:
			return m.updatePortfolio(msg)
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
					m.tickers[i].Price = fmt.Sprintf("$%.2f", p.USDPrice)
					if p.PriceChange24h >= 0 {
						m.tickers[i].Change = fmt.Sprintf("+%.1f%%", p.PriceChange24h)
					} else {
						m.tickers[i].Change = fmt.Sprintf("%.1f%%", p.PriceChange24h)
					}
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
		return m, nil
	case tickMsg:
		cmds := []tea.Cmd{tickCmd(), m.fetchAllPrices()}
		if m.wallet != nil && m.mode == viewPortfolio {
			cmds = append(cmds, m.fetchBalance(), m.fetchTokens())
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
			return m, tea.Batch(m.fetchBalance(), m.fetchTokens())
		}
		return m, nil
	case "r":
		return m, m.fetchAllPrices()
	case "?", "h":
		m.mode = viewHelp
		return m, nil
	}
	return m, nil
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
		if m.network != "devnet" || m.wallet == nil {
			return m, nil
		}
		m.errMsg = ""
		m.status = "Requesting airdrop..."
		return m, m.doAirdrop()
	case "r":
		return m, tea.Batch(m.fetchBalance(), m.fetchTokens())
	}
	return m, nil
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
		client := solana.NewRPC("")
		_, err := solana.RequestAirdrop(client, m.wallet.PubKey, 1_000_000_000)
		if err != nil {
			return balanceMsg{err: err}
		}
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
