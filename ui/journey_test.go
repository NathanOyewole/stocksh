package ui

import (
	"errors"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"stocksh/jupiter"
	"stocksh/ledger"
)

// TestPaperJourneyNoWallet drives the exact wallet-less demo path: seed paper
// SOL, load prices, quote NVDAx, prepare the swap (paper pubkey), record the
// trade + SOL debit, airdrop +1, then render every screen. It guards against
// anything that would visibly "not work" on the judge's machine.
func TestPaperJourneyNoWallet(t *testing.T) {
	m := InitialModel()
	m.led = &ledger.Ledger{Path: t.TempDir() + "/t.json"}
	m.width = 86
	m.height = 40
	m.mode = viewTickers // skip the splash auto-advance

	// no wallet -> paper SOL seed of 100
	updated, _ := m.Update(walletLoadedMsg{err: errors.New("no wallet")})
	m = updated.(Model)
	if !m.led.SolSeeded || m.led.SolBalance != 100 {
		t.Fatalf("paper seed missing: seeded=%v bal=%v", m.led.SolSeeded, m.led.SolBalance)
	}

	// prices arrive, including the SOL rate
	prices := map[string]jupiter.PriceInfo{
		jupiter.Stocks["NVDAx"]: {USDPrice: 200, PriceChange24h: 1.5},
		jupiter.USDC:            {USDPrice: 1, PriceChange24h: 0},
		jupiter.SolMint:         {USDPrice: 200, PriceChange24h: -0.1},
	}
	updated, _ = m.Update(pricesMsg{prices: prices, err: nil})
	m = updated.(Model)
	if m.solPrice != 200 {
		t.Fatalf("solPrice = %v, want 200 (SOL rate must be captured)", m.solPrice)
	}
	if m.tickers[0].Price != "$200.00" {
		t.Fatalf("ticker price not applied: %q", m.tickers[0].Price)
	}
	if m.tickers[0].Change != "+1.5%" {
		t.Fatalf("ticker change not applied: %q", m.tickers[0].Change)
	}

	// select NVDAx, quote, land on the confirm screen
	m.cursor = 0
	updated, _ = m.Update(key("enter"))
	m = updated.(Model)
	updated, _ = m.Update(quoteMsg{quote: &jupiter.QuoteResponse{
		InAmount: "500000000", OutAmount: "2500000", PriceImpactPct: "0.0",
	}, err: nil})
	m = updated.(Model)
	if m.mode != viewConfirm {
		t.Fatalf("quote should open confirm, got mode=%v", m.mode)
	}

	// swap prepared in paper mode (the transaction itself is far upstream)
	updated, _ = m.Update(swapResultMsg{sig: "DRY-RUN OK - tx prepared (780 bytes)", err: nil})
	m = updated.(Model)
	if m.mode != viewResult {
		t.Fatalf("swap should land on result view, got mode=%v", m.mode)
	}
	if got := m.led.SolBalance; got != 97.5 {
		t.Fatalf("SOL after buy = %v, want 97.5", got)
	}
	if n := len(m.led.Trades); n != 1 {
		t.Fatalf("trades = %d, want 1", n)
	}

	// back to tickers, then airdrop with no wallet -> paper +1 SOL
	updated, _ = m.Update(key("esc"))
	m = updated.(Model)
	if m.mode != viewTickers {
		t.Fatalf("esc should return to tickers, got mode=%v", m.mode)
	}
	updated, _ = m.Update(key("a"))
	m = updated.(Model)
	if got := m.led.SolBalance; got != 98.5 {
		t.Fatalf("SOL after airdrop = %v, want 98.5", got)
	}

	// every screen renders without a panic or "no wallet loaded" guardrail
	updated, _ = m.Update(key("p"))
	m = updated.(Model)
	if out := m.View(); !strings.Contains(out, "98.5") {
		t.Fatalf("portfolio missing updated balance:\n%s", out)
	}
	updated, _ = m.Update(key("esc"))
	m = updated.(Model)
	updated, _ = m.Update(key("i"))
	m = updated.(Model)
	if out := m.View(); !strings.Contains(out, "ACTIVITY") || !strings.Contains(out, "AIRDROP") {
		t.Fatalf("activity missing entries:\n%s", out)
	}
	updated, _ = m.Update(key("esc"))
	m = updated.(Model)
	updated, _ = m.Update(key("t"))
	m = updated.(Model)
	if m.View() == "" {
		t.Fatal("empty history view")
	}
	updated, _ = m.Update(key("esc"))
	m = updated.(Model)
	updated, _ = m.Update(key("w"))
	m = updated.(Model)
	if m.View() == "" {
		t.Fatal("empty watchlist view")
	}
	updated, _ = m.Update(key("esc"))
	m = updated.(Model)
	updated, _ = m.Update(key("?"))
	m = updated.(Model)
	if m.View() == "" {
		t.Fatal("empty help view")
	}
}

func key(s string) tea.KeyMsg {
	switch s {
	case "enter":
		return tea.KeyMsg{Type: tea.KeyEnter}
	case "esc":
		return tea.KeyMsg{Type: tea.KeyEscape}
	case " ":
		return tea.KeyMsg{Type: tea.KeySpace}
	}
	return tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(s)}
}