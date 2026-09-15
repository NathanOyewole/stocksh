package ui

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"stocksh/jupiter"
	"stocksh/ledger"
)

func TestPadAddsSpace(t *testing.T) {
	got := pad("abc", 6)
	if l := len([]rune(got)); l != 6 {
		t.Fatalf("pad width = %d, want 6", l)
	}
	if !strings.HasPrefix(got, "abc") {
		t.Fatalf("pad lost original text")
	}
}

func TestPadNoTruncate(t *testing.T) {
	got := pad("abcdef", 3)
	if got != "abcdef" {
		t.Fatalf("pad should not truncate: got %q", got)
	}
}

func TestTickerColsWide(t *testing.T) {
	sym, price, chg, trend, liq, mark, pos, posVal, pnl, size := tickerCols(110)
	want := []int{8, 10, 7, 13, 10, 9, 7, 11, 8, 9}
	got := []int{sym, price, chg, trend, liq, mark, pos, posVal, pnl, size}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("wide columns = %v, want %v", got, want)
		}
	}
}

func TestTickerColsFits(t *testing.T) {
	for _, inner := range []int{30, 38, 46, 60, 72, 86, 100, 140} {
		sym, price, chg, trend, liq, mark, pos, posVal, pnl, size := tickerCols(inner)
		if trend < 1 {
			t.Fatalf("inner=%d: trend = %d, want >= 1", inner, trend)
		}
		total := sym + price + chg + trend + liq + mark + pos + posVal + pnl + size + 18
		if total > inner {
			t.Fatalf("inner=%d: total %d > available", inner, total)
		}
	}
}

func TestViewTickersRenders(t *testing.T) {
	m := InitialModel()
	m.width = 120
	m.height = 30
	m.dryRun = true
	m.amountUSDC = 500
	m.tickers = []Ticker{
		{Symbol: "NVDAx", Price: "$10.00", Change: "+1.2%", PriceV: 10, ChgV: 1.2,
			History: []float64{8, 9, 10}, Watch: true},
		{Symbol: "AAPLx", Price: "$12.00", Change: "-0.5%", PriceV: 12, ChgV: -0.5,
			History: []float64{12.5, 12}},
	}
	m.cursor = 0
	out := m.viewTickers()
	if !strings.Contains(out, "TICKERS") {
		t.Fatal("missing TICKERS header")
	}
	if !strings.Contains(out, "PRICE") {
		t.Fatal("missing column header")
	}
	if !strings.Contains(out, "NVDA") {
		t.Fatal("missing NVDAx ticker row")
	}
	if !strings.Contains(out, "AAPL") {
		t.Fatal("missing AAPLx ticker row")
	}
	if !strings.Contains(out, "+1.2") {
		t.Fatal("missing change value")
	}
	if !strings.Contains(out, "★") {
		t.Fatal("star not rendered for watched symbol")
	}
}

func TestViewHelpRenders(t *testing.T) {
	m := InitialModel()
	m.width = 86
	m.height = 40
	out := m.viewHelp()
	if !strings.Contains(out, "HELP") {
		t.Fatal("missing HELP header")
	}
	if !strings.Contains(out, "NAVIGATE") || !strings.Contains(out, "TRADE") || !strings.Contains(out, "SCREENS") {
		t.Fatalf("missing section headers: %s", out)
	}
	if !strings.Contains(out, "watchlist") || !strings.Contains(out, "airdrop") || !strings.Contains(out, "portfolio") {
		t.Fatalf("missing expected screen descriptions: %s", out)
	}
	// each action's description must appear exactly once (no duplicated entries)
	count := func(s string) int { return strings.Count(out, s) }
	if count("airdrop") != 1 {
		t.Fatalf("airdrop should be listed exactly once, got %d:\n%s", count("airdrop"), out)
	}
	for _, line := range strings.Split(out, "\n") {
		if lipgloss.Width(line) > m.innerWidth() {
			t.Fatalf("help line overflows panel width (%d > %d):\n%q", lipgloss.Width(line), m.innerWidth(), line)
		}
	}
	// help must also fit the framed viewport exactly at several sizes
	for _, sz := range [][2]int{{100, 40}, {70, 30}, {120, 44}} {
		m.width, m.height = sz[0], sz[1]
		m.mode = viewHelp
		view := m.View()
		if n := strings.Count(view, "\n") + 1; n != m.height {
			t.Fatalf("help %dx%d emitted %d lines, want %d", m.width, m.height, n, m.height)
		}
		for i, line := range strings.Split(view, "\n") {
			if w := lipgloss.Width(strings.TrimRight(line, " ")); w > m.width {
				t.Fatalf("help %dx%d: line %d is %d cells wide (viewport %d):\n%q", m.width, m.height, i, w, m.width, line)
			}
		}
	}
}

func TestFitCapsToViewportHeight(t *testing.T) {
	m := InitialModel()
	m.height = 12 // panel body budget = 3
	body := ""
	for i := 0; i < 10; i++ {
		body += "line x\n"
	}
	got := m.fit(body)
	if n := strings.Count(got, "\n") + 1; n > m.bodyBudget() {
		t.Fatalf("fit produced %d lines, budget is %d:\n%s", n, m.bodyBudget(), got)
	}
	if !strings.Contains(got, "▾ more") {
		t.Fatalf("fit should mark the cut:\n%s", got)
	}
}

func TestViewNeverExceedsViewport(t *testing.T) {
	m := InitialModel()
	m.width = 60
	m.height = 10
	m.mode = viewTickers
	m.dryRun = true
	m.tickers = []Ticker{}
	for i := 0; i < 30; i++ {
		m.tickers = append(m.tickers, Ticker{Symbol: "SYM", Price: "$1.00", Change: "+1.0%"})
	}
	out := m.View()
	if n := strings.Count(out, "\n") + 1; n > m.height {
		t.Fatalf("View emitted %d lines, terminal has %d:\n%s", n, m.height, out)
	}
	for i, line := range strings.Split(out, "\n") {
		w := lipgloss.Width(strings.TrimRight(line, " "))
		if w > m.width {
			t.Fatalf("line %d is %d cells wide (viewport %d):\n%q", i, w, m.width, line)
		}
	}
}

func TestFrameFillsViewportExactly(t *testing.T) {
	m := InitialModel()
	m.width = 100
	m.height = 40
	m.mode = viewTickers
	m.dryRun = true
	m.tickers = []Ticker{}
	for i := 0; i < 60; i++ {
		m.tickers = append(m.tickers, Ticker{Symbol: "SYM", Price: "$1.00", Change: "+1.0%"})
	}
	out := m.View()
	if n := strings.Count(out, "\n") + 1; n != m.height {
		t.Fatalf("View emitted %d lines, want exactly %d:\n%s", n, m.height, out)
	}
	for i, line := range strings.Split(out, "\n") {
		w := lipgloss.Width(strings.TrimRight(line, " "))
		if w > m.width {
			t.Fatalf("line %d is %d cells wide (viewport %d):\n%q", i, w, m.width, line)
		}
	}
	// the border box must sit at the very top and the status row at the bottom
	lines := strings.Split(out, "\n")
	if !strings.HasPrefix(lines[0], "┏") {
		t.Fatalf("top border missing:\n%q", lines[0])
	}
	if !strings.HasPrefix(lines[m.height-2], "┗") {
		t.Fatalf("bottom border missing:\n%q", lines[m.height-2])
	}
	if strings.TrimRight(lines[m.height-1], " ") == "" {
		t.Fatalf("status/log line missing at bottom row:\n%q", lines[m.height-1])
	}
}

func TestViewTickersEmpty(t *testing.T) {
	m := InitialModel()
	m.width = 86
	m.tickers = nil
	out := m.viewTickers()
	if !strings.Contains(out, "Loading") {
		t.Fatal("empty tickers should show loading hint")
	}
}

func keyRune(r rune) tea.KeyMsg {
	return tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}}
}

func TestCustomAmountEntersAndSets(t *testing.T) {
	m := InitialModel()
	m.tickers = []Ticker{{Symbol: "NVDAx", Mint: "mint1", PriceV: 10}}
	m.amountUSDC = 10
	m.led = &ledger.Ledger{Path: t.TempDir() + "/t.json"}

	updated, _ := m.updateTickers(keyRune('c'))
	m2 := updated.(Model)
	if m2.mode != viewCustomAmount {
		t.Fatal("c should enter custom amount mode")
	}

	cur := m2
	for _, ch := range "525" {
		next, _ := cur.updateCustomAmount(keyRune(ch))
		cur = next.(Model)
	}
	if cur.customBuf != "525" {
		t.Fatalf("buffer = %q, want 525", cur.customBuf)
	}
	done, _ := cur.updateCustomAmount(tea.KeyMsg{Type: tea.KeyEnter})
	md := done.(Model)
	if md.orderUSDC != 525 {
		t.Fatalf("one-shot order = %v, want 525", md.orderUSDC)
	}
	if md.selected.Symbol != "NVDAx" {
		t.Fatalf("selected = %q, want NVDAx (custom size must quote the ticker under the cursor)", md.selected.Symbol)
	}
	if md.customBuf != "" {
		t.Fatalf("buffer not cleared after enter: %q", md.customBuf)
	}
	if md.amountUSDC != 10 {
		t.Fatalf("global size changed to %v, want 10", md.amountUSDC)
	}
	// feeding the quote drops the user straight into confirm
	qd, _ := md.Update(quoteMsg{quote: &jupiter.QuoteResponse{
		InAmount: "525000000", OutAmount: "250000", PriceImpactPct: "0.0",
	}, err: nil})
	qm := qd.(Model)
	if qm.mode != viewConfirm {
		t.Fatalf("quote should land on confirm, got mode=%v", qm.mode)
	}
}

func TestCustomAmountRejectsInvalid(t *testing.T) {
	m := InitialModel()
	m.tickers = []Ticker{{Symbol: "NVDAx", Mint: "mint1", PriceV: 10}}

	cur := m
	for _, ch := range "100000000" {
		next, _ := cur.updateCustomAmount(keyRune(ch))
		cur = next.(Model)
	}
	done, _ := cur.updateCustomAmount(tea.KeyMsg{Type: tea.KeyEnter})
	md := done.(Model)
	if md.errMsg == "" {
		t.Fatalf("amount over 50000 should be rejected, got amount=%v", md.amountUSDC)
	}
}

func TestCustomAmountTypesZerosWithoutSplash(t *testing.T) {
	m := InitialModel()
	m.tickers = []Ticker{{Symbol: "NVDAx", Mint: "mint1", PriceV: 10}}
	m.mode = viewTickers

	// enter custom mode via full Update so the global 0 handler is exercised
	updated, _ := m.Update(keyRune('0'))
	md := updated.(Model)
	if md.mode != viewSplash {
		t.Fatalf("0 on the ticker screen should return to splash (global shortcut), got mode=%v", md.mode)
	}

	updated, _ = md.Update(keyRune(' ')) // splash -> tickers
	m = updated.(Model)
	updated, _ = m.Update(keyRune('c'))
	m = updated.(Model)
	if m.mode != viewCustomAmount {
		t.Fatalf("c should enter custom amount mode, got mode=%v", m.mode)
	}

	cur := m
	for _, ch := range "450" {
		next, _ := cur.Update(keyRune(ch))
		cur = next.(Model)
		if cur.mode != viewCustomAmount {
			t.Fatalf("custom input was interrupted (mode=%v) while typing %q; 0 must stay a digit", cur.mode, ch)
		}
	}
	if cur.customBuf != "450" {
		t.Fatalf("buffer = %q, want 450", cur.customBuf)
	}
	if cur.mode != viewCustomAmount {
		t.Fatalf("0 during custom input must not go to splash, got mode=%v", cur.mode)
	}
}

func TestCustomAmountBackspaceAndEsc(t *testing.T) {
	m := InitialModel()
	m.tickers = []Ticker{{Symbol: "NVDAx", Mint: "mint1", PriceV: 10}}

	cur := m
	for _, ch := range "42" {
		next, _ := cur.updateCustomAmount(keyRune(ch))
		cur = next.(Model)
	}
	next, _ := cur.updateCustomAmount(tea.KeyMsg{Type: tea.KeyBackspace})
	cur = next.(Model)
	if cur.customBuf != "4" {
		t.Fatalf("backspace buffer = %q, want 4", cur.customBuf)
	}
	done, _ := cur.updateCustomAmount(tea.KeyMsg{Type: tea.KeyEsc})
	md := done.(Model)
	if md.mode != viewTickers {
		t.Fatal("Esc should cancel custom amount")
	}
	if md.customBuf != "" {
		t.Fatal("Esc should clear buffer")
	}
}

func TestZeroReturnsToSplash(t *testing.T) {
	m := InitialModel()
	m.tickers = []Ticker{{Symbol: "NVDAx", Mint: "mint1", PriceV: 10}}
	m.mode = viewTickers
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'0'}})
	m2 := updated.(Model)
	if m2.mode != viewSplash {
		t.Fatal("0 should return to splash")
	}
	// and any key from splash jumps back to tickers
	updated2, _ := m2.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{' '}})
	m3 := updated2.(Model)
	if m3.mode != viewTickers {
		t.Fatal("splash should advance to tickers on keypress")
	}
}

func TestViewTickersShowsCustomPrompt(t *testing.T) {
	m := InitialModel()
	m.width = 86
	m.mode = viewCustomAmount
	m.customBuf = "77.5"
	m.tickers = []Ticker{{Symbol: "NVDAx", Mint: "mint1", PriceV: 10, Price: "$10.00"}}
	out := m.viewTickers()
	if !strings.Contains(out, "Custom amount") {
		t.Fatalf("custom prompt missing: %s", out)
	}
	if !strings.Contains(out, "77.5") {
		t.Fatalf("typed value missing from prompt: %s", out)
	}
	if !strings.Contains(out, "Enter to quote") {
		t.Fatalf("quote hint missing: %s", out)
	}
}
