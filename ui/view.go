package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

func (m Model) View() string {
	if m.width == 0 {
		return "Loading STOCK.sh..."
	}

	var body string
	switch m.mode {
	case viewTickers:
		body = m.viewTickers()
	case viewConfirm:
		body = m.viewConfirm()
	case viewResult:
		body = m.viewResult()
	case viewPortfolio:
		body = m.viewPortfolio()
	case viewHelp:
		body = m.viewHelp()
	}

	netBadge := m.styles.Yellow.Render("[ DEVNET ]")
	if m.network == "mainnet" {
		netBadge = m.styles.Green.Render("[ MAINNET ]")
	}
	modeBadge := m.styles.Cyan.Render("[ DRY-RUN ]")
	if !m.dryRun {
		modeBadge = m.styles.Error.Render("[ LIVE ]")
	}

	title := m.styles.Title.Render("  STOCK.sh")
	subtitle := m.styles.Cyan.Render("  Terminal xStocks")
	header := lipgloss.JoinHorizontal(lipgloss.Center, title, subtitle, "   ", netBadge, "  ", modeBadge)

	footer := m.styles.Status.Render("  " + m.status)
	if m.errMsg != "" {
		footer += "\n" + m.styles.Error.Render("  ERR: "+m.errMsg)
	}

	panelW := m.width - 2
	if panelW < 40 {
		panelW = 40
	}
	body = m.styles.Border.Width(panelW).Padding(1, 2).Render(body)

	content := lipgloss.JoinVertical(lipgloss.Left, "", header, "", body)
	used := lipgloss.Height(content) + lipgloss.Height(footer) + 2
	spacer := ""
	if m.height > used {
		spacer = strings.Repeat("\n", m.height-used-1)
	}
	return content + spacer + "\n" + footer
}

func (m Model) viewTickers() string {
	var b strings.Builder
	sideLabel := m.styles.Green.Bold(true).Render("BUY")
	if m.side == "sell" {
		sideLabel = m.styles.Yellow.Bold(true).Render("SELL")
	}
	b.WriteString(m.styles.Header.Render(fmt.Sprintf("TICKERS                                    side: %s", sideLabel)))
	b.WriteString("\n\n")
	b.WriteString(m.styles.Dim.Render(fmt.Sprintf("  %-10s  %-12s  %-10s  %-12s", "SYMBOL", "PRICE", "24h", "SIZE")))
	b.WriteString("\n")
	b.WriteString(m.styles.Dim.Render("  " + strings.Repeat("-", 52)))
	b.WriteString("\n\n")
	for i, t := range m.tickers {
		price := t.Price
		if price == "" {
			price = "-"
		}
		chg := t.Change
		if chg == "" {
			chg = "-"
		}
		row := fmt.Sprintf("  %-10s  %-12s  %-10s  %.0f USDC", t.Symbol, price, chg, m.amountUSDC)
		if i == m.cursor {
			b.WriteString(m.styles.Selected.Render(row))
		} else {
			prefix := fmt.Sprintf("  %-10s  %-12s  ", t.Symbol, price)
			b.WriteString(m.styles.Normal.Render(prefix))
			if strings.HasPrefix(chg, "+") {
				b.WriteString(m.styles.Green.Render(fmt.Sprintf("%-10s", chg)))
			} else if strings.HasPrefix(chg, "-") {
				b.WriteString(m.styles.Error.Render(fmt.Sprintf("%-10s", chg)))
			} else {
				b.WriteString(m.styles.Dim.Render(fmt.Sprintf("%-10s", chg)))
			}
			b.WriteString(m.styles.Normal.Render(fmt.Sprintf("  %.0f USDC", m.amountUSDC)))
		}
		b.WriteString("\n")
	}
	b.WriteString("\n")
	b.WriteString(m.styles.Dim.Render("  up/down / j k    navigate          Enter      quote"))
	b.WriteString("\n")
	b.WriteString(m.styles.Dim.Render("  + / -            size              s          buy/sell"))
	b.WriteString("\n")
	b.WriteString(m.styles.Dim.Render("  p                portfolio         ? / h      help"))
	b.WriteString("\n")
	b.WriteString(m.styles.Dim.Render("  r                refresh           q          quit"))
	b.WriteString("\n\n")
	b.WriteString(m.styles.Cyan.Render("  Size: ") + m.styles.Green.Bold(true).Render(fmt.Sprintf("%.0f USDC", m.amountUSDC)))
	b.WriteString("\n")
	b.WriteString(m.styles.Dim.Render("  Live prices via Jupiter  -  auto-refresh every 12s"))
	return b.String()
}

func (m Model) viewConfirm() string {
	if m.quote == nil {
		return "No quote"
	}
	var b strings.Builder
	action := m.styles.Green.Bold(true).Render("BUY")
	if m.side == "sell" {
		action = m.styles.Yellow.Bold(true).Render("SELL")
	}
	b.WriteString(m.styles.Header.Render("CONFIRM  -  ") + action)
	b.WriteString("\n\n")
	if m.side == "buy" {
		b.WriteString(fmt.Sprintf("  Buy         %s\n", m.styles.Green.Bold(true).Render(m.selected.Symbol)))
		b.WriteString(fmt.Sprintf("  Pay         %s USDC\n", m.styles.Cyan.Render(formatUSDC(m.quote.InAmount))))
		b.WriteString(fmt.Sprintf("  Receive    ~%s %s\n", m.styles.Green.Render(formatStock(m.quote.OutAmount)), m.selected.Symbol))
	} else {
		b.WriteString(fmt.Sprintf("  Sell        %s\n", m.styles.Yellow.Bold(true).Render(m.selected.Symbol)))
		b.WriteString(fmt.Sprintf("  Receive    ~%s USDC\n", m.styles.Cyan.Render(formatUSDC(m.quote.OutAmount))))
		b.WriteString(fmt.Sprintf("  Pay        ~%s %s\n", formatStock(m.quote.InAmount), m.selected.Symbol))
	}
	b.WriteString(fmt.Sprintf("\n  Impact      %s%%\n", m.quote.PriceImpactPct))
	b.WriteString(fmt.Sprintf("  Slippage    %d bps\n", m.quote.SlippageBps))
	b.WriteString("\n")
	if m.dryRun || m.network == "devnet" {
		b.WriteString(m.styles.Yellow.Render("  -> Will prepare tx only (dry-run / devnet)"))
		b.WriteString("\n")
	}
	b.WriteString("\n")
	b.WriteString(m.styles.Dim.Render("  y / Enter     prepare          Esc / n     cancel"))
	return b.String()
}

func (m Model) viewResult() string {
	var b strings.Builder
	if m.lastSig != "" {
		b.WriteString(m.styles.Success.Render("  *  TRADE PREPARED"))
		b.WriteString("\n\n")
		for _, line := range wrap(m.lastSig, max(m.width-12, 40)) {
			b.WriteString("  " + line + "\n")
		}
		b.WriteString("\n")
		b.WriteString(m.styles.Dim.Render("  Real xStocks live on mainnet only."))
	} else {
		b.WriteString(m.styles.Error.Render("  X  FAILED"))
		b.WriteString("\n\n")
		b.WriteString("  " + m.errMsg)
	}
	b.WriteString("\n\n")
	b.WriteString(m.styles.Dim.Render("  Enter / Esc     back to tickers"))
	return b.String()
}

func (m Model) viewPortfolio() string {
	var b strings.Builder
	b.WriteString(m.styles.Header.Render("PORTFOLIO"))
	b.WriteString("\n\n")
	if m.wallet == nil {
		b.WriteString(m.styles.Dim.Render("  No wallet loaded."))
		b.WriteString("\n\n")
		b.WriteString(m.styles.Dim.Render("  Set SOLANA_PRIVATE_KEY in .env"))
		b.WriteString("\n")
		b.WriteString(m.styles.Dim.Render("  or place keypair at ~/.config/solana/id.json"))
	} else {
		short := m.wallet.PubKey.String()
		if len(short) > 16 {
			short = short[:8] + "..." + short[len(short)-6:]
		}
		b.WriteString(fmt.Sprintf("  Address     %s\n", m.styles.Cyan.Render(short)))
		b.WriteString("\n")
		if m.solLoaded {
			b.WriteString(fmt.Sprintf("  SOL         %s\n", m.styles.Green.Bold(true).Render(fmt.Sprintf("%.4f", m.solBalance))))
		} else {
			b.WriteString("  SOL         loading...\n")
		}
		b.WriteString("\n")
		b.WriteString(m.styles.Header.Render("  xStocks holdings"))
		b.WriteString("\n\n")
		if !m.tokensLoaded {
			b.WriteString(m.styles.Dim.Render("  loading token balances..."))
			b.WriteString("\n")
		} else if len(m.tokenBalances) == 0 {
			b.WriteString(m.styles.Dim.Render("  (none - buy some on the ticker screen)"))
			b.WriteString("\n")
			if m.network == "devnet" {
				b.WriteString(m.styles.Dim.Render("  note: xStocks liquidity is mainnet-only"))
				b.WriteString("\n")
			}
		} else {
			for _, tb := range m.tokenBalances {
				b.WriteString(fmt.Sprintf("  %-10s  %s\n", tb.Symbol, m.styles.Green.Bold(true).Render(fmt.Sprintf("%.6f", tb.Amount))))
			}
		}
	}
	b.WriteString("\n\n")
	b.WriteString(m.styles.Dim.Render("  a     airdrop 1 SOL (devnet only)"))
	b.WriteString("\n")
	b.WriteString(m.styles.Dim.Render("  r     refresh balances"))
	b.WriteString("\n")
	b.WriteString(m.styles.Dim.Render("  Esc / p     back to tickers"))
	return b.String()
}

func (m Model) viewHelp() string {
	var b strings.Builder
	b.WriteString(m.styles.Header.Render("HELP  -  KEYBINDINGS"))
	b.WriteString("\n\n")
	b.WriteString(m.styles.Cyan.Bold(true).Render("  NAVIGATION") + "\n")
	b.WriteString("  up/down / j k   Move cursor\n")
	b.WriteString("  Enter / Space    Get quote\n")
	b.WriteString("  Esc / n         Cancel / go back\n")
	b.WriteString("  q               Quit\n\n")
	b.WriteString(m.styles.Cyan.Bold(true).Render("  TRADING") + "\n")
	b.WriteString("  + / -           Change USDC size\n")
	b.WriteString("  s               Toggle BUY / SELL\n")
	b.WriteString("  y / Enter       Confirm & prepare swap\n\n")
	b.WriteString(m.styles.Cyan.Bold(true).Render("  SCREENS") + "\n")
	b.WriteString("  p               Portfolio\n")
	b.WriteString("  ? / h           This help\n")
	b.WriteString("  r               Refresh prices / balances\n")
	b.WriteString("  a               Airdrop 1 SOL (devnet)\n\n")
	b.WriteString(m.styles.Dim.Render("  Esc / ? / h     back to tickers"))
	return b.String()
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func wrap(s string, width int) []string {
	if width < 8 {
		width = 8
	}
	if len(s) <= width {
		return []string{s}
	}
	var lines []string
	for len(s) > width {
		lines = append(lines, s[:width])
		s = s[width:]
	}
	if len(s) > 0 {
		lines = append(lines, s)
	}
	return lines
}
