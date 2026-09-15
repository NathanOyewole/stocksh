package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// innerWidth is the usable text width inside the full-screen frame: the
// terminal width minus the 1-cell thick border (2) and the 1-cell vertical
// padding around the frame interior (6). Everything in the UI derives from
// this, so resizing the terminal re-lays-out every screen automatically
// (tea.WindowSizeMsg is delivered on every SIGWINCH / PowerShell window
// resize).
func (m Model) innerWidth() int {
	w := m.width
	if w <= 0 {
		w = 84 // sane default before the first WindowSizeMsg lands
	}
	if iw := w - 8; iw > 0 {
		return iw
	}
	return 1
}

// frameHeight is the total height of the outer border box: the full viewport
// minus the single external status/log row at the bottom, so the status line
// never pushes the terminal into scrolling.
func (m Model) frameHeight() int {
	if m.height <= 0 {
		return 0
	}
	h := m.height - 1
	if h < 6 {
		h = 6
	}
	return h
}

// bodyBudget returns the number of content lines a view may emit inside the
// frame before the frame's own chrome is added: the border rows + padding (4),
// the header row + rule (2), and the keybinding footer on the trading screens
// (3: divider + two keybar rows). A body within this budget can never push
// the frame past the viewport or leave ghosted lines behind.
func (m Model) bodyBudget() int {
	if m.height <= 0 {
		return 1 << 20 // unknown viewport (pre-size frame / tests): never truncate
	}
	rows := m.frameHeight()
	rows -= 4 // top + bottom border rows + padding row above and below
	rows -= 2 // header row + rule
	if m.mode == viewTickers || m.mode == viewCustomAmount {
		rows -= 2 // footer divider + docked hint line
	}
	if rows < 1 {
		return 1
	}
	return rows
}

// fit caps an already-rendered body to the frame body height so no screen can
// exceed the terminal viewport. Trailing rows are dropped (a hint line marks
// the cut) rather than letting them leak past the frame border.
// tableRuleWidth returns a responsive width for the decorative rules in the
// auxiliary views (portfolio, history, watchlist), expanding with the
// terminal width but never overflowing it.
func (m Model) tableRuleWidth() int {
	w := m.innerWidth() - 2
	if w < 40 {
		w = 40
	}
	if w > 80 {
		w = 80
	}
	return w
}

func (m Model) fit(body string) string {
	budget := m.bodyBudget()
	if h := strings.Count(body, "\n") + 1; h <= budget {
		return body
	}
	lines := strings.Split(strings.TrimRight(body, "\n"), "\n")
	out := lines
	if budget > 1 {
		out = lines[:budget-1]
	}
	return strings.Join(out, "\n") + "\n" + m.styles.Dim.Render("  ▾ more entries — make the window taller")
}

// spaceBetween justifies left/right within width, padding with spaces. On
// widths too small to hold both sides it degrades to a clipped single line so
// the row never overflows the frame.
func spaceBetween(left, right string, width int) string {
	gap := width - lipgloss.Width(left) - lipgloss.Width(right)
	if gap >= 1 {
		return left + strings.Repeat(" ", gap) + right
	}
	return clip(left+" "+right, width)
}

// fitLine truncates a rendered line to at most width cells with an ellipsis,
// so footer/header rows can never wrap on narrow terminals.
func fitLine(s string, width int) string {
	if width < 1 {
		return ""
	}
	if lipgloss.Width(s) <= width {
		return pad(s, width)
	}
	return clip(s, width) + strings.Repeat(" ", max(0, width-lipgloss.Width(clip(s, width))))
}

// ralign right-aligns s within width cells (used for numeric columns).
func ralign(s string, width int) string {
	if width <= 0 {
		return ""
	}
	n := lipgloss.Width(s)
	if n >= width {
		return s
	}
	return strings.Repeat(" ", width-n) + s
}

// fmtUsd renders a large USDC figure in compact form ($1.79M, $8.4B).
func fmtUsd(v float64) string {
	if v >= 1e9 {
		return fmt.Sprintf("$%.2fB", v/1e9)
	}
	if v >= 1e6 {
		return fmt.Sprintf("$%.2fM", v/1e6)
	}
	if v >= 1e3 {
		return fmt.Sprintf("$%.1fK", v/1e3)
	}
	return fmt.Sprintf("$%.2f", v)
}

func (m Model) View() string {
	if m.width == 0 {
		return "Loading STOCK.sh..."
	}

	if m.mode == viewSplash {
		return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, m.viewSplash())
	}

	var body, footer string
	switch m.mode {
	case viewTickers, viewCustomAmount:
		body = m.viewTickers()
		footer = m.viewFooter()
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
	case viewActivity:
		body = m.viewActivity()
	case viewHelp:
		body = m.viewHelp()
	}

	return m.frame(body, footer)
}

// frame builds the full-screen dashboard: a header row (brand left, network /
// mode badges right), a rule, the view body, an optional keybinding footer,
// all inside the full-width outer border, with the live status/log line pinned
// to its own final row below the frame. The frame is explicitly padded to the
// viewport so every cell is overwritten every frame and nothing ghosts, and
// every inner line is clamped to the exact content width so the border never
// overruns the terminal.
func (m Model) frame(body, footer string) string {
	inner := m.innerWidth()
	rule := m.styles.Dim.Render(strings.Repeat("─", inner))

	// Content area height: the frame box minus its 2 border rows and its 1-row
	// top/bottom padding. Everything above the border must fit exactly there.
	contentH := m.frameHeight() - 4

	parts := []string{m.headerLine(), rule, body}
	if footer != "" {
		parts = append(parts, footer)
	}
	content := strings.Join(parts, "\n")
	if n := strings.Count(content, "\n") + 1; n > contentH {
		// The keybinding footer is the last to give way on cramped terminals.
		if footer != "" {
			parts = parts[:3]
			content = strings.Join(parts, "\n")
		}
		if n := strings.Count(content, "\n") + 1; n > contentH {
			lines := strings.Split(content, "\n")
			content = strings.Join(lines[:contentH], "\n")
		}
	}
	if gap := contentH - (strings.Count(content, "\n") + 1); gap > 0 {
		content += strings.Repeat("\n", gap)
	}

	panel := m.styles.Border.Render(content)
	stacked := panel + "\n" + m.statusBar()
	return lipgloss.Place(m.width, m.height, lipgloss.Top, lipgloss.Left, stacked)
}

// headerLine anchors "[ STOCK.sh ] Terminal xStocks" top-left and the
// network/mode badges flush to the right edge, spaced to the exact frame width.
func (m Model) headerLine() string {
	left := m.styles.Green.Bold(true).Render("[ ") +
		m.styles.Title.Render("STOCK.sh") +
		m.styles.Green.Bold(true).Render(" ]") +
		m.styles.Cyan.Render("  Terminal xStocks")
	return spaceBetween(left, m.badges(), m.innerWidth())
}

func (m Model) badges() string {
	netBadge := m.styles.Yellow.Bold(true).Render("[ DEVNET ]")
	if m.network == "mainnet" {
		netBadge = m.styles.Green.Bold(true).Render("[ MAINNET ]")
	}
	modeBadge := m.styles.Cyan.Bold(true).Render("[ DRY-RUN ]")
	if !m.dryRun {
		modeBadge = m.styles.Error.Render("[ LIVE ]")
	}
	return lipgloss.JoinHorizontal(lipgloss.Center, netBadge, "  ", modeBadge)
}

// statusBar is the single bottom row reserved for the status/log line. It
// always spans the full width so stale cells are overwritten.
func (m Model) statusBar() string {
	status := m.styles.Status.Render(m.status)
	if m.errMsg != "" {
		status = m.styles.Error.Render("ERR: " + m.errMsg)
	}
	return pad(clip(status, m.width), m.width)
}

// viewFooter is the minimal docked hint pinned inside the bottom of the frame:
// the original `[?] Help · [q] Quit` plus the live-refresh note. Full-screen
// views keep their own on-screen hints; this stays out of the way.
func (m Model) viewFooter() string {
	inner := m.innerWidth()
	hint := m.styles.Green.Bold(true).Render("[?] Help") +
		m.styles.Dim.Render("  ") +
		m.styles.Cyan.Bold(true).Render("[q] Quit") +
		m.styles.Dim.Render("  │  Live prices via Jupiter · auto-refresh every 5s")
	rule := m.styles.Dim.Render(strings.Repeat("─", inner))
	return strings.Join([]string{rule, fitLine(pad(hint, inner), inner)}, "\n")
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
	symW, priceW, chgW, trendW, liqW, markW, posW, posValW, pnlW, sizeW := tickerCols(inner)

	sideLabel := m.styles.Green.Bold(true).Render("BUY")
	if m.side == "sell" {
		sideLabel = m.styles.Yellow.Bold(true).Render("SELL")
	}
	headRight := m.styles.Dim.Render("side ") + sideLabel +
		m.styles.Dim.Render("  ·  size ") + m.styles.Cyan.Render(fmt.Sprintf("%.0f USDC", m.amountUSDC))
	if m.solLoaded {
		headRight += m.styles.Dim.Render("  ·  SOL ") +
			m.styles.Green.Bold(true).Render(fmt.Sprintf("%.2f", m.solBalance))
	}

	var b strings.Builder
	b.WriteString(spaceBetween(m.styles.Header.Render("TICKERS"), headRight, inner))
	b.WriteString("\n")

	if len(m.tickers) == 0 {
		b.WriteString("  " + m.styles.Dim.Render("Loading market data..."))
		b.WriteString("\n")
		return b.String()
	}

	// Column header aligned to the same column template as the rows.
	headCells := []string{
		pad(clip("SYMBOL", symW), symW),
		pad(clip("PRICE", priceW), priceW),
		pad(clip("24H", chgW), chgW),
		pad(clip("TREND", trendW), trendW),
		pad(clip("LIQ", liqW), liqW),
		pad(clip("MARK", markW), markW),
		pad(clip("POS", posW), posW),
		pad(clip("POS VAL", posValW), posValW),
		pad(clip("P&L", pnlW), pnlW),
		pad(clip("SIZE", sizeW), sizeW),
	}
	b.WriteString(m.styles.Dim.Render(pad(strings.Join(headCells, "  "), inner)))
	b.WriteString("\n\n")

	// Row window: rows that fit between the section header and the frame footer.
	chrome := 3 // section line + col header + blank
	if m.mode == viewCustomAmount {
		chrome = 5 // ... + custom prompt + hint line
	}
	window := m.bodyBudget() - chrome
	if window < 1 {
		window = 1
	}
	total := len(m.tickers)
	if total > window {
		window-- // reserve a line for the "▾ more" indicator
		if window < 1 {
			window = 1
		}
	}
	off := 0
	if m.cursor >= window {
		off = m.cursor - window + 1
	}
	end := off + window
	if end > total {
		end = total
	}
	more := total - end

	for i := off; i < end; i++ {
		t := m.tickers[i]
		price := t.Price
		if price == "" {
			price = "-"
		}
		chg := t.Change
		if chg == "" {
			chg = "-"
		}

		posQty := 0.0
		posHas := false
		pnlStr := "–"
		if m.led != nil {
			h := m.led.Holding(t.Symbol, t.PriceV)
			posQty, posHas = h.Qty, h.Qty > 0
			if h.HasBasis && h.Qty > 0 {
				if h.PnLPct >= 0 {
					pnlStr = fmt.Sprintf("+%.1f%%", h.PnLPct)
				} else {
					pnlStr = fmt.Sprintf("%.1f%%", h.PnLPct)
				}
			}
		}
		posS := "–"
		posValS := "–"
		if posHas {
			posS = fmt.Sprintf("%.2f", posQty)
			if t.PriceV > 0 {
				posValS = fmt.Sprintf("$%.2f", posQty*t.PriceV)
			}
		}
		liqS := fmtUsd(t.Liquidity)
		if t.Liquidity <= 0 {
			liqS = "–"
		}
		markS := fmt.Sprintf("$%.2f", t.Mark)
		if t.Mark <= 0 {
			markS = "–"
		}
		sizeS := fmt.Sprintf("%.0f", m.amountUSDC)
		spk := sparkline(t.History, trendW)

		// non-watch rows keep a blank 2-cell gutter so the symbol column stays
		// aligned with watched rows; the watched star lives inside the cell.
		symPlain := "  "
		if t.Watch {
			symPlain = "★ "
		}
		symColored := "  "
		if t.Watch {
			symColored = m.styles.Yellow.Render("★ ")
		}

		// raw cells (used for the full-width selected highlight).
		plain := strings.Join([]string{
			pad(clip(symPlain+t.Symbol, symW), symW),
			ralign(clip(price, priceW), priceW),
			ralign(clip(chg, chgW), chgW),
			pad(clip(spk, trendW), trendW),
			ralign(clip(liqS, liqW), liqW),
			ralign(clip(markS, markW), markW),
			ralign(clip(posS, posW), posW),
			ralign(clip(posValS, posValW), posValW),
			ralign(clip(pnlStr, pnlW), pnlW),
			ralign(clip(sizeS, sizeW), sizeW),
		}, "  ")

		// colored cells for the non-selected rows
		chgCol := pad(clip(chg, chgW), chgW)
		if strings.HasPrefix(chg, "+") {
			chgCol = m.styles.Green.Render(pad(clip(chg, chgW), chgW))
		} else if strings.HasPrefix(chg, "-") {
			chgCol = m.styles.Error.Render(pad(clip(chg, chgW), chgW))
		} else {
			chgCol = m.styles.Dim.Render(pad(clip(chg, chgW), chgW))
		}
		pnlCol := ralign(clip(pnlStr, pnlW), pnlW)
		if strings.HasPrefix(pnlStr, "+") {
			pnlCol = m.styles.Green.Render(ralign(clip(pnlStr, pnlW), pnlW))
		} else if strings.HasPrefix(pnlStr, "-") {
			pnlCol = m.styles.Error.Render(ralign(clip(pnlStr, pnlW), pnlW))
		}
		colored := strings.Join([]string{
			pad(clip(symColored+t.Symbol, symW), symW),
			ralign(clip(price, priceW), priceW),
			chgCol,
			m.styles.Cyan.Render(pad(clip(spk, trendW), trendW)),
			m.styles.Dim.Render(ralign(clip(liqS, liqW), liqW)),
			m.styles.Green.Render(ralign(clip(markS, markW), markW)),
			m.styles.Dim.Render(ralign(clip(posS, posW), posW)),
			m.styles.Dim.Render(ralign(clip(posValS, posValW), posValW)),
			pnlCol,
			m.styles.Dim.Render(ralign(clip(sizeS, sizeW), sizeW)),
		}, "  ")

		if i == m.cursor {
			// the row highlight spans the full table width
			b.WriteString(m.styles.Selected.Render(pad(plain, inner)))
			b.WriteString("\n")
			continue
		}
		b.WriteString(pad(colored, inner))
		b.WriteString("\n")
	}

	if more > 0 {
		b.WriteString(m.styles.Dim.Render(fmt.Sprintf("  ▾ %d more symbols below", more)))
		b.WriteString("\n")
	}

	if m.mode == viewCustomAmount {
		b.WriteString("\n")
		b.WriteString("  " + m.styles.Header.Render("Custom amount (USDC): $"+m.customBuf+"_"))
		b.WriteString("\n" + m.styles.Dim.Render("  Enter to quote   ·   Esc to cancel   ·   backspace to delete"))
	}
	// final clamp: the frame needs the body within bodyBudget; if a tiny
	// viewport somehow exceeds it, truncate rather than push past the border.
	return m.fit(b.String())
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
	b.WriteString(m.styles.Dim.Render("  p portfolio    Enter / Esc   back to tickers"))
	return b.String()
}

func (m Model) viewPortfolio() string {
	var b strings.Builder
	b.WriteString(m.styles.Header.Render("PORTFOLIO"))
	b.WriteString("\n\n")
	if m.wallet == nil {
		if m.solLoaded {
			// No wallet: show the persisted paper SOL account (devnet seed).
			b.WriteString(fmt.Sprintf("  %-11s %s\n",
				"Paper SOL", m.styles.Green.Bold(true).Render(fmt.Sprintf("%.4f", m.solBalance))))
			b.WriteString("\n")
		}
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
			b.WriteString(m.renderPositions())
			b.WriteString("\n")
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
			b.WriteString(fmt.Sprintf("  %-11s %s\n",
				"SOL", m.styles.Green.Bold(true).Render(fmt.Sprintf("%.4f", m.solBalance))))
		} else {
			b.WriteString("  SOL         loading...\n")
		}
		b.WriteString("\n")
		b.WriteString(m.renderPositions())
		b.WriteString("\n")
		if m.dryRun {
			b.WriteString(m.styles.Dim.Render("  Paper swap account - SOL debits/credits are simulated"))
			b.WriteString("\n")
		}
	}
	b.WriteString("\n\n")
	b.WriteString(m.styles.Dim.Render("  r     refresh balances   ·   i     activity feed"))
	b.WriteString("\n")
	b.WriteString(m.styles.Dim.Render("  Esc / p     back to tickers"))
	return m.fit(b.String())
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
	b.WriteString(m.styles.Dim.Render("  " + strings.Repeat("─", m.tableRuleWidth())))
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
	b.WriteString(m.styles.Dim.Render("  " + strings.Repeat("─", m.tableRuleWidth())))
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
	b.WriteString(m.styles.Dim.Render("  " + strings.Repeat("─", m.tableRuleWidth())))
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
	return m.fit(b.String())
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
	b.WriteString(m.styles.Dim.Render("  " + strings.Repeat("─", m.tableRuleWidth())))
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
	return m.fit(b.String())
}

func (m Model) viewActivity() string {
	var b strings.Builder
	b.WriteString(m.styles.Header.Render("ACTIVITY"))
	b.WriteString("\n")
	if m.solLoaded {
		b.WriteString(m.styles.Dim.Render(fmt.Sprintf("  SOL balance: %.4f   (persisted to disk, survives quit)", m.solBalance)))
	} else {
		b.WriteString(m.styles.Dim.Render("  SOL balance: loading..."))
	}
	b.WriteString("\n\n")

	acts := m.led.Activities
	if len(acts) == 0 {
		b.WriteString(m.styles.Dim.Render("  No activity yet \u2014 trades, airdrops, alerts and sizes show up here."))
		b.WriteString("\n\n")
	} else {
		start := 0
		if len(acts) > 80 {
			start = len(acts) - 80
		}
		for i := len(acts) - 1; i >= start; i-- {
			a := acts[i]
			tag := m.styles.Dim.Render(strings.ToUpper(a.Type))
			switch a.Type {
			case "trade":
				tag = m.styles.Cyan.Bold(true).Render("TRADE")
			case "airdrop":
				tag = m.styles.Green.Bold(true).Render("AIRDROP")
			case "alert":
				tag = m.styles.Error.Bold(true).Render("ALERT")
			case "watch":
				tag = m.styles.Yellow.Bold(true).Render("WATCH")
			case "size":
				tag = m.styles.Cyan.Render("SIZE")
			}
			b.WriteString(fmt.Sprintf("  %s %s %s\n",
				m.styles.Dim.Render(a.Time.Format("15:04:05")),
				tag,
				clip(a.Text, m.innerWidth()-24)))
		}
		b.WriteString("\n")
	}
	b.WriteString(m.styles.Dim.Render("  i / Esc  back to tickers   \u00b7   t  trade history"))
	return m.fit(b.String())
}

func (m Model) viewHelp() string {
	inner := m.innerWidth()

	// key column width: fit the longest key, but never so wide the description
	// column would spill past the right edge of the panel.
	keyW := 14
	for _, g := range keyBindings() {
		for _, r := range g.rows {
			if w := lipgloss.Width(r.keys); w > keyW {
				keyW = w
			}
		}
	}
	keyW = min(keyW, max(inner-26, 10))

	var b strings.Builder
	b.WriteString(lipgloss.PlaceHorizontal(inner, lipgloss.Center, m.styles.Header.Render("HELP")))
	b.WriteString("\n")
	b.WriteString(m.styles.Dim.Render("  " + strings.Repeat("─", inner-2)))
	b.WriteString("\n\n")
	for _, g := range keyBindings() {
		b.WriteString(m.helpGroup(g, keyW))
	}
	b.WriteString(m.styles.Dim.Render("  Esc / ? / h    back to tickers"))
	return m.fit(b.String())
}

type keyDesc struct {
	keys, desc string
}

type helpGroup struct {
	title string
	rows  []keyDesc
}

// keyBindings is the single source of truth for what every key does. It feeds
// the help screen only, so each action is listed exactly once.
func keyBindings() []helpGroup {
	return []helpGroup{
		{"NAVIGATE", []keyDesc{
			{"up/down · j/k", "move the highlight"},
			{"Enter / Space", "see a live quote"},
			{"Esc", "go back / cancel"},
			{"q", "quit STOCK.sh"},
		}},
		{"TRADE", []keyDesc{
			{"+ / -", "order size, in USDC"},
			{"c", "type a custom USDC size"},
			{"s", "switch BUY ↔ SELL"},
			{"y / Enter", "confirm & prepare swap"},
		}},
		{"SCREENS", []keyDesc{
			{"p", "portfolio & P&L"},
			{"t", "trade history"},
			{"w", "watchlist & alerts"},
			{"*", "track / star a symbol"},
			{"i", "activity feed (trades + events)"},
			{"0", "back to splash (a digit in custom $ size)"},
			{"r", "refresh prices & balances"},
			{"a", "airdrop SOL (+1 paper / devnet faucet)"},
		}},
	}
}

// helpGroup renders one section as a strict two-column table. The key column
// is a fixed width shared by every section and descriptions start at the exact
// same offset, so nothing wraps back into the key margin. A trailing blank row
// separates logical groups for breathing room.
func (m Model) helpGroup(g helpGroup, keyW int) string {
	body := lipgloss.NewStyle().Foreground(lipgloss.Color("#EFEFEF"))
	descW := max(m.innerWidth()-keyW-6, 8)
	var b strings.Builder
	b.WriteString("  " + m.styles.Cyan.Bold(true).Render(g.title))
	b.WriteString("\n")
	for _, r := range g.rows {
		k := m.styles.Green.Bold(true).Render(pad(r.keys, keyW))
		b.WriteString("  " + k + "  " + body.Render(clip(r.desc, descW)))
		b.WriteString("\n")
	}
	b.WriteString("\n")
	return b.String()
}

// clip truncates s to at most width cells, appending a narrow ellipsis when it
// had to cut, so a single row can never exceed the panel width.
func clip(s string, width int) string {
	if width < 1 {
		return ""
	}
	if lipgloss.Width(s) <= width {
		return s
	}
	out := ""
	for _, r := range s {
		if lipgloss.Width(out)+1 >= width {
			break
		}
		out += string(r)
	}
	return out + "…"
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

// tickerCols returns responsive column widths for the 10-column trading table
// given the available frame width. Wide terminals get the full table plus a
// long trend/sparkline column; narrow ones shrink every column before the
// trend column, which is the last to give up space.
func tickerCols(inner int) (sym, price, chg, trend, liq, mark, pos, posVal, pnl, size int) {
	sym, price, chg, liq, mark, pos, posVal, pnl, size = 8, 10, 7, 10, 9, 7, 11, 8, 9
	fixed := func() int { return sym + price + chg + liq + mark + pos + posVal + pnl + size + 18 }
	const minT = 12

	trend = inner - fixed()
	if trend >= minT {
		return sym, price, chg, trend, liq, mark, pos, posVal, pnl, size
	}

	// Not enough room: shrink the fixed columns (largest / least essential
	// first) until the trend/sparkline achieves its minimum width. The price
	// and mark columns give up space before the symbol and POS columns.
	cols := []*int{&price, &mark, &liq, &posVal, &sym, &size, &pnl, &pos, &chg}
	floors := []int{6, 6, 7, 7, 5, 6, 6, 5, 5}
	for trend < minT {
		shrank := false
		for i := range cols {
			if *cols[i] > floors[i] {
				*cols[i]--
				shrank = true
				trend = inner - fixed()
				break
			}
		}
		if !shrank {
			break
		}
	}

	// Absolute guarantee: trend is never zero/negative and the returned table
	// is never wider than inner. Only at absurdly narrow widths does this trim
	// columns below their normal floors.
	if trend < 1 {
		trend = 1
	}
	i := 0
	all := []*int{&sym, &price, &chg, &liq, &mark, &pos, &posVal, &pnl, &size}
	for fixed()+trend > inner {
		if *all[i] > 1 {
			*all[i]--
		} else if trend > 1 {
			trend--
		}
		i = (i + 1) % len(all)
	}
	return sym, price, chg, trend, liq, mark, pos, posVal, pnl, size
}
