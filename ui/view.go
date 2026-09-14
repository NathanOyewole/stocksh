package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

func (m Model) View() string {
	if m.width == 0 {
		return "Loading STOCK.sh\u2026"
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
	netBadge := m.styles.Yellow.Render("[DEVNET]")
	if m.network == "mainnet" {
		netBadge = m.styles.Green.Render("[MAINNET]")
	}
	modeBadge := m.styles.Cyan.Render("DRY-RUN")
	if !m.dryRun {
		modeBadge = m.styles.Error.Render("LIVE")
	}
	header := m.styles.Title.Render("STOCK.sh  \u00b7  Terminal xStocks") + "  " + netBadge + "  " + modeBadge
	footer := m.styles.Status.Render(m.status)
	if m.errMsg != "" {
		footer += "\n" + m.styles.Error.Render("ERR: "+m.errMsg)
	}
	content := lipgloss.JoinVertical(lipgloss.Left, header, body)
	used := lipgloss.Height(content) + lipgloss.Height(footer) + 1
	spacer := ""
	if m.height > used {
		spacer = strings.Repeat("\n", m.height-used-1)
	}
	return content + spacer + "\n" + footer
}

func (m Model) viewTickers() string {
	var b strings.Builder
	sideLabel := m.styles.Green.Render("BUY")
	if m.side == "sell" {
		sideLabel = m.styles.Yellow.Render("SELL")
	}
	b.WriteString(m.styles.Header.Render(fmt.Sprintf("  TICKERS  \u00b7  side: %s", sideLabel)))
	b.WriteString("\n")
	b.WriteString(m.styles.Dim.Render("  SYMBOL    PRICE       24h        SIZE"))
	b.WriteString("\n")
	b.WriteString(m.styles.Dim.Render("  \u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500"))
	b.WriteString("\n")
	for i, t := range m.tickers {
		price := t.Price
		if price == "" {
			price = "\u2014"
		}
		chg := t.Change
		if chg == "" {
			chg = "\u2014"
		}
		row := fmt.Sprintf("  %-8s  %-10s  %-8s  %.0f USDC", t.Symbol, price, chg, m.amountUSDC)
		if i == m.cursor {
			b.WriteString(m.styles.Selected.Render(row))
		} else {
			prefix := fmt.Sprintf("  %-8s  %-10s  ", t.Symbol, price)
			b.WriteString(m.styles.Normal.Render(prefix))
			if strings.HasPrefix(chg, "+") {
				b.WriteString(m.styles.Green.Render(fmt.Sprintf("%-8s", chg)))
			} else if strings.HasPrefix(chg, "-") {
				b.WriteString(m.styles.Error.Render(fmt.Sprintf("%-8s", chg)))
			} else {
				b.WriteString(m.styles.Dim.Render(fmt.Sprintf("%-8s", chg)))
			}
			b.WriteString(m.styles.Normal.Render(fmt.Sprintf("  %.0f USDC", m.amountUSDC)))
		}
		b.WriteString("\n")
	}
	b.WriteString("\n")
	b.WriteString(m.styles.Dim.Render("  \u2191\u2193/jk  navigate   Enter  quote   +/-  size"))
	b.WriteString("\n")
	b.WriteString(m.styles.Dim.Render("  s      buy/sell   p  portfolio   ?  help   r  refresh   q  quit"))
	b.WriteString("\n\n")
	b.WriteString(m.styles.Cyan.Render("  Size: ") + m.styles.Green.Render(fmt.Sprintf("%.0f USDC", m.amountUSDC)))
	b.WriteString("\n")
	b.WriteString(m.styles.Dim.Render("  Live prices via Jupiter \u00b7 auto-refresh every 12s"))
	return m.styles.Border.Width(max(m.width-4, 40)).Render(b.String())
}

func (m Model) viewConfirm() string {
	if m.quote == nil {
		return "No quote"
	}
	var b strings.Builder
	action := "BUY"
	if m.side == "sell" {
		action = "SELL"
	}
	b.WriteString(m.styles.Header.Render(fmt.Sprintf("  CONFIRM %s", action)))
	b.WriteString("\n\n")
	if m.side == "buy" {
		b.WriteString(fmt.Sprintf("  Buy      %s\n", m.styles.Green.Render(m.selected.Symbol)))
		b.WriteString(fmt.Sprintf("  Pay      %s USDC\n", formatUSDC(m.quote.InAmount)))
		b.WriteString(fmt.Sprintf("  Receive  ~%s %s\n", formatStock(m.quote.OutAmount), m.selected.Symbol))
	} else {
		b.WriteString(fmt.Sprintf("  Sell     %s\n", m.styles.Yellow.Render(m.selected.Symbol)))
		b.WriteString(fmt.Sprintf("  Receive  ~%s USDC\n", formatUSDC(m.quote.OutAmount)))
		b.WriteString(fmt.Sprintf("  Pay      ~%s %s\n", formatStock(m.quote.InAmount), m.selected.Symbol))
	}
	b.WriteString(fmt.Sprintf("  Impact   %s%%\n", m.quote.PriceImpactPct))
	b.WriteString(fmt.Sprintf("  Slippage %d bps\n", m.quote.SlippageBps))
	b.WriteString("\n")
	if m.dryRun || m.network == "devnet" {
		b.WriteString(m.styles.Yellow.Render("  \u2192 Will prepare tx only (dry-run / devnet)"))
		b.WriteString("\n")
	}
	b.WriteString(m.styles.Dim.Render("  y / Enter   prepare     Esc / n   cancel"))
	return m.styles.Border.Width(max(m.width-4, 40)).Render(b.String())
}

func (m Model) viewResult() string {
	var b strings.Builder
	if m.lastSig != "" {
		b.WriteString(m.styles.Success.Render("  \U0001F7E2  TRADE PREPARED"))
		b.WriteString("\n\n")
		for _, line := range wrap(m.lastSig, 48) {
			b.WriteString("  " + line + "\n")
		}
		b.WriteString("\n")
		b.WriteString(m.styles.Dim.Render("  Real xStocks live on mainnet only."))
	} else {
		b.WriteString(m.styles.Error.Render("  \u2715  FAILED"))
		b.WriteString("\n\n")
		b.WriteString("  " + m.errMsg)
	}
	b.WriteString("\n\n")
	b.WriteString(m.styles.Dim.Render("  Enter / Esc   back to tickers"))
	return m.styles.Border.Width(max(m.width-4, 40)).Render(b.String())
}

func (m Model) viewPortfolio() string {
	var b strings.Builder
	b.WriteString(m.styles.Header.Render("  PORTFOLIO"))
	b.WriteString("\n")
	b.WriteString(m.styles.Dim.Render("  \u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500"))
	b.WriteString("\n\n")
	if m.wallet == nil {
		b.WriteString(m.styles.Dim.Render("  No wallet loaded."))
		b.WriteString("\n")
		b.WriteString(m.styles.Dim.Render("  Set SOLANA_PRIVATE_KEY or place id.json"))
	} else {
		short := m.wallet.PubKey.String()
		if len(short) > 12 {
			short = short[:6] + "\u2026" + short[len(short)-4:]
		}
		b.WriteString(fmt.Sprintf("  Address   %s\n", m.styles.Cyan.Render(short)))
		b.WriteString("\n")
		if m.solLoaded {
			b.WriteString(fmt.Sprintf("  SOL       %s\n", m.styles.Green.Render(fmt.Sprintf("%.4f", m.solBalance))))
		} else {
			b.WriteString("  SOL       loading\u2026\n")
		}
		b.WriteString("\n")
		b.WriteString(m.styles.Header.Render("  xStocks holdings"))
		b.WriteString("\n")
		if !m.tokensLoaded {
			b.WriteString(m.styles.Dim.Render("  loading token balances\u2026"))
			b.WriteString("\n")
		} else if len(m.tokenBalances) == 0 {
			b.WriteString(m.styles.Dim.Render("  (none \u2014 buy some on the ticker screen)"))
			b.WriteString("\n")
			if m.network == "devnet" {
				b.WriteString(m.styles.Dim.Render("  note: xStocks live on mainnet only"))
				b.WriteString("\n")
			}
		} else {
			for _, tb := range m.tokenBalances {
				b.WriteString(fmt.Sprintf("  %-8s  %s\n", tb.Symbol, m.styles.Green.Render(fmt.Sprintf("%.6f", tb.Amount))))
			}
		}
	}
	b.WriteString("\n\n")
	b.WriteString(m.styles.Dim.Render("  a   airdrop 1 SOL (devnet only)"))
	b.WriteString("\n")
	b.WriteString(m.styles.Dim.Render("  r   refresh balances"))
	b.WriteString("\n")
	b.WriteString(m.styles.Dim.Render("  Esc / p   back to tickers"))
	return m.styles.Border.Width(max(m.width-4, 40)).Render(b.String())
}

func (m Model) viewHelp() string {
	var b strings.Builder
	b.WriteString(m.styles.Header.Render("  HELP  \u00b7  KEYBINDINGS"))
	b.WriteString("\n\n")
	b.WriteString(m.styles.Cyan.Render("  NAVIGATION") + "\n")
	b.WriteString("  \u2191 \u2193 / j k     Move cursor\n")
	b.WriteString("  Enter / Space  Get quote\n")
	b.WriteString("  Esc / n       Cancel / go back\n")
	b.WriteString("  q             Quit\n\n")
	b.WriteString(m.styles.Cyan.Render("  TRADING") + "\n")
	b.WriteString("  + / -         Change USDC size\n")
	b.WriteString("  s             Toggle BUY / SELL\n")
	b.WriteString("  y / Enter     Confirm & prepare swap\n\n")
	b.WriteString(m.styles.Cyan.Render("  SCREENS") + "\n")
	b.WriteString("  p             Portfolio\n")
	b.WriteString("  ? / h         This help\n")
	b.WriteString("  r             Refresh\n\n")
	b.WriteString(m.styles.Dim.Render("  Esc / ? / h   back to tickers"))
	return m.styles.Border.Width(max(m.width-4, 40)).Render(b.String())
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
