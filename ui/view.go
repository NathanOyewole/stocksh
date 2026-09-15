package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// boxWidth returns the total outer width (including the thick border) of the
// centered panel. Capped so the layout stays readable — and looks intentional
// rather than edge-to-edge — on very wide terminals.
func (m Model) boxWidth() int {
	w := m.width - 6
	if w > 88 {
		w = 88
	}
	if w < 44 {
		w = 44
	}
	return w
}

// innerWidth is the usable text width inside the panel (outer box minus the
// 1-col thick border and the 3-col horizontal padding on each side).
func (m Model) innerWidth() int {
	return m.boxWidth() - 8
}

// spaceBetween justifies left/right within width, padding with spaces.
func spaceBetween(left, right string, width int) string {
	gap := width - lipgloss.Width(left) - lipgloss.Width(right)
	if gap < 1 {
		gap = 1
	}
	return left + strings.Repeat(" ", gap) + right
}

func (m Model) View() string {
	if m.width == 0 {
		return "Loading STOCK.sh..."
	}

	if m.mode == viewSplash {
		return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, m.viewSplash())
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

	netBadge := m.styles.Yellow.Bold(true).Render("[ DEVNET ]")
	if m.network == "mainnet" {
		netBadge = m.styles.Green.Bold(true).Render("[ MAINNET ]")
	}
	modeBadge := m.styles.Cyan.Bold(true).Render("[ DRY-RUN ]")
	if !m.dryRun {
		modeBadge = m.styles.Error.Render("[ LIVE ]")
	}

	boxW := m.boxWidth()
	innerW := m.innerWidth()

	title := m.styles.Title.Render("STOCK.sh")
	subtitle := m.styles.Cyan.Render(" Terminal xStocks")
	badges := lipgloss.JoinHorizontal(lipgloss.Center, netBadge, "   ", modeBadge)

	headerLine := spaceBetween(title+subtitle, badges, innerW)
	rule := m.styles.Dim.Render(strings.Repeat("─", innerW))

	// Center the screen's content as a block within the panel, instead of
	// letting it hug the left edge when the panel is wider than the content.
	centeredBody := lipgloss.PlaceHorizontal(innerW, lipgloss.Center, body)

	panel := m.styles.Border.Width(boxW - 2).Render(
		lipgloss.JoinVertical(lipgloss.Left, headerLine, rule, "", centeredBody),
	)

	status := m.styles.Status.Render(m.status)
	if m.errMsg != "" {
		status = m.styles.Error.Render("ERR: " + m.errMsg)
	}
	footer := lipgloss.PlaceHorizontal(boxW, lipgloss.Center, status)

	stacked := lipgloss.JoinVertical(lipgloss.Center, panel, "", footer)

	return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, stacked)
}

// sparkline renders a compact price-trend bar (▁▂▃▄▅▆▇█) from recent history.
func sparkline(prices []float64, cols int) string {
	if len(prices) == 0 {
		return strings.Repeat(" ", cols)
	}
	blocks := []rune("▁▂▃▄▅▆▇█")
	buckets := make([]float64, cols)
	counts := make([]int, cols)
	for i, p := range prices {
		idx := i * cols / len(prices)
		buckets[idx] += p
		counts[idx]++
	}
	min, max := prices[0], prices[0]
	for _, p := range prices {
		if p < min {
			min = p
		}
		if p > max {
			max = p
		}
	}
	span := max - min
	var out strings.Builder
	for c := 0; c < cols; c++ {
		avg := float64(0)
		if counts[c] > 0 {
			avg = buckets[c] / float64(counts[c])
		}
		lvl := 0
		if span > 0 {
			lvl = int((avg-min)/span*7 + 0.5)
			if lvl < 0 {
				lvl = 0
			}
			if lvl > 7 {
				lvl = 7
			}
		} else {
			lvl = 3 // flat line
		}
		out.WriteRune(blocks[lvl])
	}
	return out.String()
}

func (m Model) viewSplash() string {
	var b strings.Builder

	logo := []string{
		"████ ████  ██  ████ █  █      ████ █  █",
		"█     ██  █  █ █    █ █       █    █  █",
		"████  ██  █  █ █    ██        ████ ████",
		"   █  ██  █  █ █    █ █          █ █  █",
		"████  ██   ██  ████ █  █ █    ████ █  █",
	}
	for _, line := range logo {
		b.WriteString(m.styles.Cyan.Render(line))
		b.WriteString("\n")
	}
	b.WriteString("\n")
	b.WriteString("  " + m.styles.Green.Bold(true).Render("Terminal xStocks on Solana.") + "\n")
	b.WriteString("  " + m.styles.Dim.Render("Trade tokenized stock tokens 24/7 from your terminal.") + "\n")
	b.WriteString("\n")
	b.WriteString("  " + m.styles.Yellow.Render("Powered by Jupiter") + m.styles.Dim.Render("   ·   ") + m.styles.Cyan.Render("Built for Stocklana") + "\n")
	b.WriteString("\n")
	b.WriteString("  " + m.styles.Dim.Render("◆ ") + m.styles.Normal.Render(m.walletStatus) + "\n")
	b.WriteString("\n")
	b.WriteString("  " + m.styles.Dim.Render("[ press any key to continue ]") + "\n")
	return b.String()
}

func (m Model) viewTickers() string {
	var b strings.Builder
	sideLabel := m.styles.Green.Bold(true).Render("BUY")
	if m.side == "sell" {
		sideLabel = m.styles.Yellow.Bold(true).Render("SELL")
	}
	b.WriteString(m.styles.Header.Render(fmt.Sprintf("TICKERS                                    side: %s", sideLabel)))
	b.WriteString("\n\n")
	b.WriteString(m.styles.Dim.Render(fmt.Sprintf("  %-10s  %-12s  %-9s  %-7s  %-10s", "SYMBOL", "PRICE", "24h", "TREND", "SIZE")))
	b.WriteString("\n")
	b.WriteString(m.styles.Dim.Render("  " + strings.Repeat("-", 58)))
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
		spark := sparkline(t.History, 8)
		row := fmt.Sprintf("  %-10s  %-12s  %-9s  %s  %.0f USDC", t.Symbol, price, chg, spark, m.amountUSDC)
		if i == m.cursor {
			b.WriteString(m.styles.Selected.Render(row))
		} else {
			prefix := fmt.Sprintf("  %-10s  %-12s  ", t.Symbol, price)
			b.WriteString(m.styles.Normal.Render(prefix))
			if strings.HasPrefix(chg, "+") {
				b.WriteString(m.styles.Green.Render(fmt.Sprintf("%-9s", chg)))
			} else if strings.HasPrefix(chg, "-") {
				b.WriteString(m.styles.Error.Render(fmt.Sprintf("%-9s", chg)))
			} else {
				b.WriteString(m.styles.Dim.Render(fmt.Sprintf("%-9s", chg)))
			}
			b.WriteString(m.styles.Normal.Render("  "))
			b.WriteString(m.styles.Cyan.Render(spark))
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
	b.WriteString(m.styles.Dim.Render("  Live prices via Jupiter  -  auto-refresh every 5s"))
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
		b.WriteString(m.styles.Yellow.Render("  -> Dry-run: tx prepared, paper position recorded"))
		b.WriteString("\n")
		b.WriteString(m.styles.Dim.Render("     Watch it move in the portfolio (p)"))
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
		for _, line := range wrap(m.lastSig, max(m.innerWidth()-2, 32)) {
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
		b.WriteString("\n")
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
		b.WriteString(m.renderPositions())
		b.WriteString("\n")
		if m.dryRun {
			b.WriteString(m.styles.Dim.Render("  Paper portfolio (dry-run) - trades are simulated"))
			b.WriteString("\n")
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

type position struct {
	Symbol  string
	Qty     float64
	Price   float64
	HasPx   bool
	Value   float64
	AvgCost float64
	PnL     float64
	PnLPct  float64
	HasPnL  bool
	AllocPct float64
}

func (m Model) renderPositions() string {
	var b strings.Builder

	// price lookup: symbol -> (price, 24h change)
	type q struct{ price, chg float64 }
	px := make(map[string]q)
	for _, t := range m.tickers {
		px[t.Symbol] = q{t.PriceV, t.ChgV}
	}

	var positions []position
	var totalValue, totalPnL float64

	collect := func(symbol string, qty float64) {
		if qty <= 0 {
			return
		}
		p := position{Symbol: symbol, Qty: qty}
		if qi, ok := px[symbol]; ok && qi.price > 0 {
			p.Price, p.HasPx = qi.price, true
			p.Value = qty * qi.price
		}
		totalValue += p.Value

		h := m.led.Holding(symbol, p.Price)
		if h.HasBasis && p.Value > 0 {
			p.AvgCost = h.AvgCost
			p.PnL = h.PnL
			p.PnLPct = h.PnLPct
			p.HasPnL = true
			totalPnL += h.PnL
		}
		positions = append(positions, p)
	}

	if m.dryRun {
		// Paper mode: positions come from the ledger.
		for _, t := range m.tickers {
			collect(t.Symbol, m.led.NetHolding(t.Symbol))
		}
	} else {
		// Live mode: real on-chain balances.
		for _, tb := range m.tokenBalances {
			collect(tb.Symbol, tb.Amount)
		}
	}

	// Allocation percentages.
	for i := range positions {
		if totalValue > 0 {
			positions[i].AllocPct = positions[i].Value / totalValue * 100
		}
	}

	b.WriteString(m.styles.Header.Render("  xStocks holdings"))
	b.WriteString("\n\n")
	if len(positions) == 0 {
		b.WriteString(m.styles.Dim.Render("  (empty - buy some on the ticker screen)"))
		b.WriteString("\n")
		if m.network == "devnet" && !m.dryRun {
			b.WriteString(m.styles.Dim.Render("  note: xStocks liquidity is mainnet-only"))
			b.WriteString("\n")
		}
		return b.String()
	}

	// Table header.
	head := fmt.Sprintf("  %-7s %9s %-9s %-10s %-9s %-6s %s",
		"SYMBOL", "QTY", "PRICE", "VALUE", "P&L", "P&L%", "ALLOC")
	b.WriteString(m.styles.Dim.Render(head))
	b.WriteString("\n")
	b.WriteString(m.styles.Dim.Render("  " + strings.Repeat("─", 56)))
	b.WriteString("\n")

	for _, p := range positions {
		priceS := "–"
		if p.HasPx {
			priceS = fmt.Sprintf("$%.2f", p.Price)
		}
		valS := "–"
		if p.Value > 0 {
			valS = fmt.Sprintf("$%.2f", p.Value)
		}
		pnlS := m.styles.Dim.Render("–")
		if p.HasPnL {
			if p.PnL >= 0 {
				pnlS = m.styles.Success.Render(fmt.Sprintf("+$%.2f", p.PnL))
			} else {
				pnlS = m.styles.Error.Render(fmt.Sprintf("-$%.2f", -p.PnL))
			}
		}
		pctS := m.styles.Dim.Render("–")
		if p.HasPnL {
			if p.PnL >= 0 {
				pctS = m.styles.Success.Render(fmt.Sprintf("+%.1f%%", p.PnLPct))
			} else {
				pctS = m.styles.Error.Render(fmt.Sprintf("%.1f%%", p.PnLPct))
			}
		}

		row := fmt.Sprintf("  %-7s %9.5f  %-9s %-10s %-9s %-6s ",
			p.Symbol, p.Qty, priceS, valS, pnlS, pctS)
		b.WriteString(row)
		if p.AllocPct >= 0 {
			blocks := int(p.AllocPct/100*10 + 0.5)
			if blocks > 10 {
				blocks = 10
			}
			b.WriteString(m.styles.Cyan.Render(strings.Repeat("█", blocks)))
			b.WriteString(m.styles.Dim.Render(strings.Repeat("░", 10-blocks)))
			b.WriteString(fmt.Sprintf(" %2.0f%%", p.AllocPct))
		}
		b.WriteString("\n")
	}

	// Totals.
	avgTotalCost := totalValue - totalPnL
	totalPct := 0.0
	if avgTotalCost > 0 {
		totalPct = totalPnL / avgTotalCost * 100
	}
	b.WriteString(m.styles.Dim.Render("  " + strings.Repeat("─", 56)))
	b.WriteString("\n")
	totalLine := fmt.Sprintf("  %-7s %9s  %-9s %-10s ", "TOTAL", "", "", fmt.Sprintf("$%.2f", totalValue))
	if totalPnL >= 0 {
		totalLine += m.styles.Success.Render(fmt.Sprintf("+$%.2f", totalPnL))
	} else {
		totalLine += m.styles.Error.Render(fmt.Sprintf("-$%.2f", -totalPnL))
	}
	totalLine += " " + fmt.Sprintf("%+6.1f%%", totalPct)
	b.WriteString(totalLine)
	b.WriteString("\n")

	// 24h portfolio change.
	var delta24 float64
	for _, p := range positions {
		if qi, ok := px[p.Symbol]; ok {
			delta24 += p.Value * qi.chg / 100
		}
	}
	b.WriteString("\n")
	if delta24 >= 0 {
		b.WriteString("  " + m.styles.Green.Render(fmt.Sprintf("Portfolio 24h: +$%.2f", delta24)))
	} else {
		b.WriteString("  " + m.styles.Error.Render(fmt.Sprintf("Portfolio 24h: -$%.2f", -delta24)))
	}
	b.WriteString("\n")

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
