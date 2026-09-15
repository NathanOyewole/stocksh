package ui

import (
	"errors"
	"strings"
	"testing"

	"stocksh/jupiter"
	"stocksh/ledger"
)

// baseModel returns a Model set up for paper trading with NVDAx/SOL/USDC prices.
func baseModel(t *testing.T) Model {
	t.Helper()
	m := InitialModel()
	m.led = &ledger.Ledger{Path: t.TempDir() + "/t.json"}
	m.width = 120
	m.height = 40
	m.mode = viewTickers

	updated, _ := m.Update(walletLoadedMsg{err: errors.New("no wallet")})
	m = updated.(Model)

	prices := map[string]jupiter.PriceInfo{
		jupiter.Stocks["NVDAx"]: {USDPrice: 200, PriceChange24h: 1.5},
		jupiter.USDC:            {USDPrice: 1, PriceChange24h: 0},
		jupiter.SolMint:         {USDPrice: 100, PriceChange24h: 0},
	}
	updated, _ = m.Update(pricesMsg{prices: prices, err: nil})
	return updated.(Model)
}

func TestPortfolioShowsPositionAfterBuy(t *testing.T) {
	m := baseModel(t)

	// Buy 0.5 NVDAx via mock quote (0.5 tokens at $200 = $100)
	m.cursor = 0
	updated, _ := m.Update(key("enter"))
	m = updated.(Model)
	updated, _ = m.Update(quoteMsg{quote: &jupiter.QuoteResponse{
		InAmount: "100000000", OutAmount: "500000", PriceImpactPct: "0.0",
	}, err: nil})
	m = updated.(Model)
	updated, _ = m.Update(swapResultMsg{sig: "DRY-RUN OK", err: nil})
	m = updated.(Model)

	if m.led.NetHolding("NVDAx") != 0.5 {
		t.Fatalf("expected 0.5 NVDAx, got %v", m.led.NetHolding("NVDAx"))
	}

	// Esc back to tickers → p to portfolio
	updated, _ = m.Update(key("esc"))
	m = updated.(Model)
	if m.mode != viewTickers {
		t.Fatalf("esc should return to tickers, got mode=%v", m.mode)
	}
	updated, _ = m.Update(key("p"))
	m = updated.(Model)

	if m.mode != viewPortfolio {
		t.Fatalf("p should go to portfolio, got mode=%v", m.mode)
	}
	if !strings.Contains(m.View(), "NVDAx") {
		t.Fatalf("portfolio missing NVDAx position after buy")
	}
}

func TestGlobalNavFromResultScreen(t *testing.T) {
	m := baseModel(t)

	// Buy via mock quote
	m.cursor = 0
	updated, _ := m.Update(key("enter"))
	m = updated.(Model)
	updated, _ = m.Update(quoteMsg{quote: &jupiter.QuoteResponse{
		InAmount: "100000000", OutAmount: "500000", PriceImpactPct: "0.0",
	}, err: nil})
	m = updated.(Model)
	updated, _ = m.Update(swapResultMsg{sig: "DRY-RUN OK", err: nil})
	m = updated.(Model)
	if m.mode != viewResult {
		t.Fatalf("expected viewResult, got mode=%v", m.mode)
	}

	// Press p directly from result — global nav should work
	updated, _ = m.Update(key("p"))
	m = updated.(Model)
	if m.mode != viewPortfolio {
		t.Fatalf("p from result should go to portfolio, got mode=%v", m.mode)
	}
	if !strings.Contains(m.View(), "NVDAx") {
		t.Fatalf("portfolio missing NVDAx position via global nav from result")
	}

	// Press t from portfolio → history
	updated, _ = m.Update(key("t"))
	m = updated.(Model)
	if m.mode != viewHistory {
		t.Fatalf("t from portfolio should go to history, got mode=%v", m.mode)
	}

	// Press 0 → splash
	updated, _ = m.Update(key("0"))
	m = updated.(Model)
	if m.mode != viewSplash {
		t.Fatalf("0 should go to splash, got mode=%v", m.mode)
	}
}

func TestSellCapsToHolding(t *testing.T) {
	m := baseModel(t)

	// Buy 0.5 NVDAx
	m.cursor = 0
	updated, _ := m.Update(key("enter"))
	m = updated.(Model)
	updated, _ = m.Update(quoteMsg{quote: &jupiter.QuoteResponse{
		InAmount: "100000000", OutAmount: "500000", PriceImpactPct: "0.0",
	}, err: nil})
	m = updated.(Model)
	updated, _ = m.Update(swapResultMsg{sig: "DRY-RUN OK", err: nil})
	m = updated.(Model)

	held := m.led.NetHolding("NVDAx")
	if held != 0.5 {
		t.Fatalf("expected 0.5 NVDAx after buy, got %v", held)
	}

	// Back to tickers, flip to sell, request $1000 worth (far more than the
	// 0.5 tokens are worth at $200) — fetchQuote must cap it to the holding.
	updated, _ = m.Update(key("esc"))
	m = updated.(Model)
	if m.mode != viewTickers {
		t.Fatalf("esc should return to tickers, got mode=%v", m.mode)
	}
	updated, _ = m.Update(key("s"))
	m = updated.(Model)
	if m.side != "sell" {
		t.Fatalf("s should toggle side to sell, got side=%v", m.side)
	}
	m.orderUSDC = 1000

	updated, _ = m.Update(key("enter"))
	m = updated.(Model)
	updated, _ = m.Update(quoteMsg{quote: &jupiter.QuoteResponse{
		InAmount: "500000", OutAmount: "100000000", PriceImpactPct: "0.0",
	}, err: nil})
	m = updated.(Model)
	if m.side != "sell" {
		t.Fatalf("side should stay sell through confirm, got side=%v", m.side)
	}
	updated, _ = m.Update(key("y"))
	m = updated.(Model)
	updated, _ = m.Update(swapResultMsg{sig: "DRY-RUN SELL", err: nil})
	m = updated.(Model)

	// After selling 0.5 tokens, net holding should be exactly 0 (not negative)
	finalHeld := m.led.NetHolding("NVDAx")
	if finalHeld != 0 {
		t.Fatalf("expected 0 NVDAx after full sell, got %v", finalHeld)
	}

	// Portfolio should show no NVDAx row
	updated, _ = m.Update(key("p"))
	m = updated.(Model)
	if strings.Contains(m.View(), "NVDAx") {
		t.Fatalf("portfolio should not show NVDAx after full sell")
	}
}

func TestSellPartialThenCheck(t *testing.T) {
	m := baseModel(t)

	// Buy 0.5 NVDAx
	m.cursor = 0
	updated, _ := m.Update(key("enter"))
	m = updated.(Model)
	updated, _ = m.Update(quoteMsg{quote: &jupiter.QuoteResponse{
		InAmount: "100000000", OutAmount: "500000", PriceImpactPct: "0.0",
	}, err: nil})
	m = updated.(Model)
	updated, _ = m.Update(swapResultMsg{sig: "DRY-RUN BUY", err: nil})
	m = updated.(Model)

	// Back to tickers, flip to sell $50 worth (0.25 tokens at $200)
	updated, _ = m.Update(key("esc"))
	m = updated.(Model)
	updated, _ = m.Update(key("s"))
	m = updated.(Model)
	m.orderUSDC = 50

	updated, _ = m.Update(key("enter"))
	m = updated.(Model)
	updated, _ = m.Update(quoteMsg{quote: &jupiter.QuoteResponse{
		InAmount: "250000", OutAmount: "50000000", PriceImpactPct: "0.0",
	}, err: nil})
	m = updated.(Model)
	updated, _ = m.Update(key("y"))
	m = updated.(Model)
	updated, _ = m.Update(swapResultMsg{sig: "DRY-RUN SELL", err: nil})
	m = updated.(Model)

	remaining := m.led.NetHolding("NVDAx")
	if remaining != 0.25 {
		t.Fatalf("expected 0.25 NVDAx after partial sell, got %v", remaining)
	}

	// Portfolio should still show NVDAx with reduced position
	updated, _ = m.Update(key("p"))
	m = updated.(Model)
	out := m.View()
	if !strings.Contains(out, "NVDAx") {
		t.Fatalf("portfolio should show NVDAx after partial sell, got:\n%s", out)
	}
}
