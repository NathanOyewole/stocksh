package ui

import (
	"strings"
	"testing"

	"stocksh/jupiter"
	"stocksh/ledger"
)

func TestRecordTradeBuyAndSell(t *testing.T) {
	m := InitialModel()
	m.led = &ledger.Ledger{Path: t.TempDir() + "/t.json"}
	m.selected = Ticker{Symbol: "NVDAx"}
	m.side = "buy"
	m.quote = &jupiter.QuoteResponse{
		InAmount:  "10000000", // 10 USDC
		OutAmount: "2500000",  // 2.5 NVDAx
	}
	m.recordTrade()
	// in/out = 10/2.5 = 4 USDC per token
	if got := m.led.NetHolding("NVDAx"); got != 2.5 {
		t.Fatalf("net = %v, want 2.5", got)
	}
	h := m.led.Holding("NVDAx", 4)
	if h.AvgCost != 4 {
		t.Fatalf("avg cost = %v, want 4", h.AvgCost)
	}

	m.side = "sell"
	m.quote = &jupiter.QuoteResponse{
		InAmount:  "1000000", // 1 NVDAx
		OutAmount: "4200000", // 4.2 USDC
	}
	m.recordTrade()
	if got := m.led.NetHolding("NVDAx"); got != 1.5 {
		t.Fatalf("net after sell = %v, want 1.5", got)
	}
}

func TestRenderPositionsShowsPnL(t *testing.T) {
	m := InitialModel()
	m.dryRun = true
	m.led = &ledger.Ledger{Path: t.TempDir() + "/t.json"}
	_ = m.led.Record("NVDAx", "buy", 2, 100)

	m.tickers = []Ticker{
		{Symbol: "NVDAx", PriceV: 125, ChgV: 3.2},
	}
	out := m.renderPositions()
	if !strings.Contains(out, "NVDAx") {
		t.Fatalf("position missing: %s", out)
	}
	if !strings.Contains(out, "TOTAL") {
		t.Fatalf("total row missing: %s", out)
	}
	if !strings.Contains(out, "Portfolio 24h") {
		t.Fatalf("24h line missing: %s", out)
	}
}