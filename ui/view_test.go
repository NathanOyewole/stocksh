package ui

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
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
	sym, price, chg, spark, size := tickerCols(80)
	if sym != 9 || price != 12 || chg != 9 || spark != 8 || size != 10 {
		t.Fatalf("full columns: %v", []int{sym, price, chg, spark, size})
	}
}

func TestTickerColsNarrow(t *testing.T) {
	sym, price, chg, spark, size := tickerCols(46)
	if spark >= 8 {
		t.Fatalf("spark should shrink for narrow width, got %d", spark)
	}
	if price > 12 {
		t.Fatalf("price should shrink for narrow width, got %d", price)
	}
	total := sym + price + chg + spark + size + 12
	if total > 46 {
		t.Fatalf("total %d > available %d", total, 46)
	}
}

func TestViewTickersRenders(t *testing.T) {
	m := InitialModel()
	m.width = 86
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
	if !strings.Contains(out, "SYMBOL") {
		t.Fatal("missing column header")
	}
	if !strings.Contains(out, "NVDAx") {
		t.Fatal("missing ticker row")
	}
	if !strings.Contains(out, "AAPLx") {
		t.Fatal("missing second ticker row")
	}
	if !strings.Contains(out, "+1.2%") {
		t.Fatal("missing change value")
	}
	if !strings.Contains(out, "★") {
		t.Fatal("star not rendered for watched symbol")
	}
}

func TestViewHelpRenders(t *testing.T) {
	m := InitialModel()
	m.width = 86
	m.height = 30
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
	if md.mode != viewTickers {
		t.Fatal("Enter should return to tickers")
	}
	if md.amountUSDC != 525 {
		t.Fatalf("amount = %v, want 525", md.amountUSDC)
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
	updated, _ := m.updateTickers(keyRune('0'))
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
	if !strings.Contains(out, "Enter to confirm") {
		t.Fatalf("confirm hint missing: %s", out)
	}
}
