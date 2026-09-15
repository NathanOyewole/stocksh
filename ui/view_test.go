package ui

import (
	"strings"
	"testing"
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
