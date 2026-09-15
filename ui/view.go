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
	case viewTickers, viewCustomAmount:
		body = m.viewTickers()
	case viewConfirm:
		body = m.viewConfirm()
	case viewResult:
		body = m.viewResult()
	case viewPortfolio:
		body = m.viewPortfolio()
	case viewHistory:
		body = m.viewHistory()
	case viewWatchlist:
		body = m.viewWatchlist()
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
	inner := m.innerWidth()
	symW, priceW, chgW, sparkW, sizeW := tickerCols(inner)

	sideLabel := m.styles.Green.Bold(true).Render("BUY")
	if m.side == "sell" {
		sideLabel = m.styles.Yellow.Bold(true).Render("SELL")
	}
	headRight := m.styles.Dim.Render("side ") + sideLabel +
		m.styles.Dim.Render("  ·  size ") + m.styles.Cyan.Render(fmt.Sprintf("%.0f USDC", m.amountUSDC))

	var b strings.Builder
	b.WriteString(spaceBetween(m.styles.Header.Render("TICKERS"), headRight, inner))
	b.WriteString("\n")
	b.WriteString(m.styles.Dim.Render("  " + strings.Repeat("─", inner-2)))
	b.WriteString("\n")

	if len(m.tickers) == 0 {
		b.WriteString("\n  " + m.styles.Dim.Render("Loading market data..."))
		b.WriteString("\n")
		return b.String()
	}

	// Column header aligned to the same column template as the rows.
	b.WriteString(m.styles.Dim.Render(fmt.Sprintf("  %s  %-*s  %-*s  %-*s  %-*s  %-*s",
		"  ", symW, "SYMBOL", priceW, "PRICE", chgW, "24H", sparkW, "TREND", sizeW, "SIZE")))
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
		size := fmt.Sprintf("%.0f USDC", m.amountUSDC)
		spk := sparkline(t.History, sparkW)

		raw := "  " +
			pad(t.Symbol, symW) + "  " +
			pad(price, priceW) + "  " +
			pad(chg, chgW) + "  " +
			pad(spk, sparkW) + "  " +
			pad(size, sizeW)

		star := "  "
		if t.Watch {
			star = m.styles.Yellow.Render("★ ")
		}
		if i == m.cursor {
			b.WriteString(star)
			b.WriteString(m.styles.Selected.Render(raw))
			b.WriteString("\n")
			continue
		}

		b.WriteString(star)
		b.WriteString("  ")
		b.WriteString(pad(t.Symbol, symW))
		b.WriteString("  ")
		b.WriteString(pad(price, priceW))
		b.WriteString("  ")
		if strings.HasPrefix(chg, "+") {
			b.WriteString(m.styles.Green.Render(pad(chg, chgW)))
		} else if strings.HasPrefix(chg, "-") {
			b.WriteString(m.styles.Error.Render(pad(chg, chgW)))
		} else {
			b.WriteString(m.styles.Dim.Render(pad(chg, chgW)))
		}
		b.WriteString("  ")
		b.WriteString(m.styles.Cyan.Render(pad(spk, sparkW)))
		b.WriteString("  ")
		b.WriteString(pad(size, sizeW))
		b.WriteString("\n")
	}

	b.WriteString("\n")
	if m.mode == viewCustomAmount {
		b.WriteString("  " + m.styles.Header.Render("Custom amount (USDC): $"+m.customBuf+"_"))
		b.WriteString("\n" + m.styles.Dim.Render("  Enter to confirm   ·   Esc to cancel   ·   backspace to delete"))
	} else {
		b.WriteString(m.hintBar())
		b.WriteString("\n")
		b.WriteString(m.styles.Dim.Render("  Live prices via Jupiter  ·  auto-refresh every 5s"))
	}
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
		if m.dryRun {
			// Paper trading works wallet-free: positions come from the ledger.
			b.WriteString(m.styles.Dim.Render("  (no wallet - paper positions only)"))
			b.WriteString("\n\n")
			b.WriteString(m.renderPositions())
			b.WriteString("\n")
			b.WriteString(m.styles.Dim.Render("  Paper portfolio (dry-run) - trades are simulated"))
			b.WriteString("\n")
		} else {
			b.WriteString(m.styles.Dim.Render("  No wallet loaded."))
			b.WriteString("\n\n")
			b.WriteString(m.styles.Dim.Render("  Set SOLANA_PRIVATE_KEY in .env"))
			b.WriteString("\n")
			b.WriteString(m.styles.Dim.Render("  or place keypair at ~/.config/solana/id.json"))
			b.WriteString("\n")
		}
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

func (m Model) viewHistory() string {
	var b strings.Builder
	b.WriteString(m.styles.Header.Render("TRADE HISTORY"))
	b.WriteString("\n\n")

	trades := m.led.Trades
	if len(trades) == 0 {
		b.WriteString(m.styles.Dim.Render("  No trades yet."))
		b.WriteString("\n")
		b.WriteString(m.styles.Dim.Render("  Prepare a buy on the ticker screen and watch it appear here."))
		b.WriteString("\n\n")
		b.WriteString(m.styles.Dim.Render("  Esc / t     back to tickers"))
		return b.String()
	}

	head := fmt.Sprintf("  %-8s %-7s %-4s %-12s %-10s %-10s", "TIME", "SYMBOL", "SIDE", "QTY", "PRICE", "TOTAL")
	b.WriteString(m.styles.Dim.Render(head))
	b.WriteString("\n")
	b.WriteString(m.styles.Dim.Render("  " + strings.Repeat("─", 56)))
	b.WriteString("\n")

	// newest first
	for i := len(trades) - 1; i >= 0; i-- {
		tr := trades[i]
		side := m.styles.Green.Bold(true).Render("BUY")
		qty := tr.Qty
		if tr.Side == "sell" {
			side = m.styles.Yellow.Bold(true).Render("SELL")
			qty = -tr.Qty
		}
		timeS := tr.Time.Format("15:04:05")
		row := fmt.Sprintf("  %-8s %-7s %-4s %-12.4f %-10s %-10s",
			timeS, tr.Symbol, side, qty,
			fmt.Sprintf("$%.2f", tr.PriceUSDC),
			fmt.Sprintf("$%.2f", qty*tr.PriceUSDC))
		b.WriteString(row)
		b.WriteString("\n")
	}
	b.WriteString("\n")
	b.WriteString(m.styles.Dim.Render("  Esc / t     back to tickers"))
	return b.String()
}

func (m Model) viewWatchlist() string {
	var b strings.Builder
	b.WriteString(m.styles.Header.Render("WATCHLIST"))
	b.WriteString("\n\n")

	var watched []int
	for i := range m.tickers {
		if m.tickers[i].Watch {
			watched = append(watched, i)
		}
	}
	if len(watched) == 0 {
		b.WriteString(m.styles.Dim.Render("  Nothing tracked yet."))
		b.WriteString("\n")
		b.WriteString(m.styles.Dim.Render("  On the ticker screen press * to star a symbol."))
		b.WriteString("\n\n")
		b.WriteString(m.styles.Dim.Render("  Esc / w     back to tickers"))
		return b.String()
	}

	head := fmt.Sprintf("  %-7s %-10s %-8s %-9s %-9s %s", "SYMBOL", "PRICE", "24h", "TREND", "TARGET", "ALERT")
	b.WriteString(m.styles.Dim.Render(head))
	b.WriteString("\n")
	b.WriteString(m.styles.Dim.Render("  " + strings.Repeat("─", 56)))
	b.WriteString("\n")

	for pos, ti := range watched {
		t := m.tickers[ti]
		price := t.Price
		if price == "" {
			price = "-"
		}
		chg := t.Change
		if chg == "" {
			chg = "-"
		}
		target := "–"
		if t.Alert > 0 {
			target = fmt.Sprintf("$%.2f", t.Alert)
		}
		alert := m.styles.Dim.Render("off")
		if t.Alert > 0 {
			if t.PriceV >= t.Alert {
				alert = m.styles.Success.Bold(true).Render("ALERTED")
			} else {
				alert = m.styles.Yellow.Render("armed")
			}
		}

		star := m.styles.Cyan.Render("★ ")
		row := fmt.Sprintf("  %s%-6s %-10s %-8s %-9s %-9s ",
			star, t.Symbol, price, chg, sparkline(t.History, 7), target)
		if pos == m.watchCursor {
			b.WriteString(m.styles.Selected.Render(row + alert))
			b.WriteString("\n")
		} else {
			b.WriteString(m.styles.Normal.Render(row))
			b.WriteString(alert)
			b.WriteString("\n")
		}
	}
	b.WriteString("\n")
	b.WriteString(m.styles.Dim.Render("  up/down select   +/- set target   x clear   * untrack   w back"))
	return b.String()
}

func (m Model) viewHelp() string {
	const keyW = 16
	left := m.helpTable("NAVIGATE", []keyDesc{
		{"up/down · j/k", "move the highlight"},
		{"Enter / Space", "see a live quote"},
		{"Esc", "go back / cancel"},
		{"q", "quit STOCK.sh"},
	}, keyW)
	left += m.helpTable("TRADE", []keyDesc{
		{"+ / -", "order size, in USDC"},
		{"s", "switch BUY ↔ SELL"},
		{"y / Enter", "confirm & prepare swap"},
	}, keyW)

	right := m.helpTable("SCREENS", []keyDesc{
		{"p", "portfolio & P&L"},
		{"t", "trade history"},
		{"w", "watchlist & alerts"},
		{"*", "track / star a symbol"},
		{"c", "type a custom USDC size"},
		{"0", "back to the splash screen"},
		{"r", "refresh prices & balances"},
		{"a", "airdrop SOL (devnet only)"},
	}, keyW)

	var b strings.Builder
	b.WriteString(lipgloss.PlaceHorizontal(m.innerWidth(), lipgloss.Center,
		m.styles.Header.Render("HELP")))
	b.WriteString("\n")
	b.WriteString(m.styles.Dim.Render("  " + strings.Repeat("─", m.innerWidth()-2)))
	b.WriteString("\n")
	b.WriteString(lipgloss.PlaceHorizontal(m.innerWidth(), lipgloss.Center,
		m.styles.Dim.Render("Every screen works the same way. Here's the whole tour:")))
	b.WriteString("\n")
	b.WriteString(lipgloss.JoinHorizontal(lipgloss.Top, left, "  │  ", right))
	b.WriteString("\n\n")
	b.WriteString(m.styles.Dim.Render("  Tip:  ? or h  reopens this guide from anywhere."))
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

// pad pads s with trailing spaces so its rendered width is at least w.
func pad(s string, w int) string {
	if w <= 0 {
		return ""
	}
	if n := lipgloss.Width(s); n >= w {
		return s
	}
	return s + strings.Repeat(" ", w-lipgloss.Width(s))
}

// tickerCols returns responsive column widths for the ticker table given the
// available inner panel width. Wide terminals keep the full table; narrow ones
// progressively drop sparkline, numeric, and symbol column width so it fits.
func tickerCols(inner int) (sym, price, chg, spark, size int) {
	sym, price, chg, spark, size = 9, 12, 9, 8, 10
	const minSym, minPrice, minChg, minSpark, minSize = 6, 5, 5, 5, 8
	total := sym + price + chg + spark + size + 12 // star col + 5 gaps
	if inner >= total {
		return sym, price, chg, spark, size
	}
	for total > inner {
		switch {
		case spark > minSpark:
			spark--
		case price > minPrice:
			price--
		case chg > minChg:
			chg--
		case sym > minSym:
			sym--
		case size > minSize:
			size--
		default:
			return sym, price, chg, spark, size
		}
		total--
	}
	return sym, price, chg, spark, size
}

type hintPair struct {
	keys, desc string
}

// hintBar renders the key hints as a clean aligned grid instead of a ragged
// left-aligned list.
func (m Model) hintBar() string {
	pairs := []hintPair{
		{"up/down j/k", "move"}, {"Enter", "quote"}, {"+ / -", "size"},
		{"s", "buy/sell"}, {"y", "confirm"}, {"Esc", "back"},
		{"c", "custom size"}, {"0", "home / splash"}, {"*", "track"},
		{"p / t", "portf / hist"}, {"w", "watchlist"}, {"r", "refresh"},
		{"a", "airdrop"}, {"? / h", "help"}, {"q", "quit"},
	}
	var b strings.Builder
	for i, p := range pairs {
		if i > 0 && i%3 == 0 {
			b.WriteString("\n")
		}
		k := m.styles.Green.Bold(true).Render(pad(p.keys, 10))
		b.WriteString("  " + k + " " + m.styles.Dim.Render(pad(p.desc, 9)))
	}
	return b.String()
}

type keyDesc struct {
	keys, desc string
}

// helpTable renders one titled group of key/description rows with the keys
// column aligned, so the section reads as a neat grid rather than scattered.
func (m Model) helpTable(title string, rows []keyDesc, keyW int) string {
	var b strings.Builder
	b.WriteString("\n")
	b.WriteString("  " + m.styles.Cyan.Bold(true).Render(strings.ToUpper(title)))
	b.WriteString("\n")
	for _, r := range rows {
		k := m.styles.Green.Bold(true).Render(pad(r.keys, keyW))
		b.WriteString("  " + k + " " + lipgloss.NewStyle().Foreground(lipgloss.Color("#EFEFEF")).Render(r.desc))
		b.WriteString("\n")
	}
	return b.String()
}
