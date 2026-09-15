package ui

import (
	"strings"
	"testing"

	"stocksh/jupiter"
	"stocksh/ledger"
)

func TestRecordBuyDebitsSolBalance(t *testing.T) {
	m := InitialModel()
	m.led = &ledger.Ledger{Path: t.TempDir() + "/t.json"}
	_ = m.led.SeedSol(100)
	m.solPrice = 200 // $200 per SOL
	m.side = "buy"
	m.selected = Ticker{Symbol: "NVDAx"}
	m.quote = &jupiter.QuoteResponse{
		InAmount:  "500000000", // spend 500 USDC
		OutAmount: "2500000",   // get 2.5 NVDAx
	}
	m.recordTrade()
	if got := m.led.SolBalance; got != 97.5 {
		t.Fatalf("balance after buy = %v, want 97.5", got)
	}
	if m.solBalance != 97.5 {
		t.Fatalf("model solBalance = %v, want 97.5", m.solBalance)
	}
	if len(m.led.Activities) == 0 || m.led.Activities[len(m.led.Activities)-1].Type != "trade" {
		t.Fatal("buy should append a trade activity")
	}
}

func TestRecordSellCreditsSolBalance(t *testing.T) {
	m := InitialModel()
	m.led = &ledger.Ledger{Path: t.TempDir() + "/t.json"}
	_ = m.led.SeedSol(100)
	m.solPrice = 200
	m.side = "sell"
	m.selected = Ticker{Symbol: "NVDAx"}
	m.quote = &jupiter.QuoteResponse{
		InAmount:  "1000000", // 1 NVDAx
		OutAmount: "4200000", // receive 4.2 USDC
	}
	m.recordTrade()
	if want := 100 + 4.2/200; m.led.SolBalance != want {
		t.Fatalf("balance after sell = %v, want %v", m.led.SolBalance, want)
	}
	if got := m.led.NetHolding("NVDAx"); got != -1 {
		t.Fatalf("net = %v, want -1", got)
	}
}

func TestSwapResultConsumesOneShotOrder(t *testing.T) {
	m := InitialModel()
	m.led = &ledger.Ledger{Path: t.TempDir() + "/t.json"}
	_ = m.led.SeedSol(100)
	m.orderUSDC = 525
	m.quote = &jupiter.QuoteResponse{
		InAmount:  "525000000",
		OutAmount: "2500000",
	}
	m.selected = Ticker{Symbol: "NVDAx"}
	updated, _ := m.Update(swapResultMsg{sig: "sig", err: nil})
	md := updated.(Model)
	if md.orderUSDC != 0 {
		t.Fatalf("one-shot order = %v, want 0 after trade", md.orderUSDC)
	}
	// the global default size is untouched by a one-shot order
	if md.amountUSDC != m.amountUSDC {
		t.Fatalf("global size changed: %v -> %v", m.amountUSDC, md.amountUSDC)
	}
}

func TestViewActivityShowsFeed(t *testing.T) {
	m := InitialModel()
	m.width = 86
	m.height = 40
	m.led = &ledger.Ledger{Path: t.TempDir() + "/t.json"}
	m.solLoaded = true
	m.solBalance = 97.5
	_ = m.led.RecordActivity("trade", "BUY 2.5000 NVDAx @ $200.00 (SOL 2.5000)")
	_ = m.led.RecordActivity("airdrop", "Devnet faucet +1 SOL (balance 98.5000)")

	out := m.viewActivity()
	if !strings.Contains(out, "ACTIVITY") {
		t.Fatalf("missing ACTIVITY header: %s", out)
	}
	if !strings.Contains(out, "97.5000") {
		t.Fatalf("missing SOL balance: %s", out)
	}
	if !strings.Contains(out, "NVDAx") {
		t.Fatalf("missing trade text: %s", out)
	}
	if !strings.Contains(out, "AIRDROP") {
		t.Fatalf("missing airdrop tag: %s", out)
	}
}