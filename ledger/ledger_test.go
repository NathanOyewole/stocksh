package ledger

import (
	"encoding/json"
	"os"
	"testing"
)

func TestNetHoldingAndBasis(t *testing.T) {
	l := &Ledger{Path: ""}
	_ = l.Record("NVDAx", "buy", 2, 100)
	_ = l.Record("NVDAx", "buy", 2, 150)
	_ = l.Record("NVDAx", "sell", -1, 200)

	if got := l.NetHolding("NVDAx"); got != 3 {
		t.Fatalf("net holding = %v, want 3", got)
	}
	h := l.Holding("NVDAx", 125)
	if !h.HasBasis {
		t.Fatal("should have basis")
	}
	if h.AvgCost != 125 {
		t.Fatalf("avg cost = %v, want 125", h.AvgCost)
	}
	if h.ValueUSDC != 375 {
		t.Fatalf("value = %v, want 375", h.ValueUSDC)
	}
	if h.PnL != 0 {
		t.Fatalf("pnl = %v, want 0", h.PnL)
	}
}

func TestNoTradesNoBasis(t *testing.T) {
	l := &Ledger{}
	if got := l.NetHolding("TSLAx"); got != 0 {
		t.Fatalf("net = %v, want 0", got)
	}
	h := l.Holding("TSLAx", 200)
	if h.HasBasis {
		t.Fatal("no basis expected")
	}
	if h.ValueUSDC != 0 {
		t.Fatalf("value = %v, want 0", h.ValueUSDC)
	}
}

func TestActivityAndBalancePersist(t *testing.T) {
	path := t.TempDir() + "/trades.json"
	l := &Ledger{Path: path}
	if err := l.SeedSol(100); err != nil {
		t.Fatal(err)
	}
	if err := l.AdjSol(-2.5); err != nil {
		t.Fatal(err)
	}
	if err := l.RecordActivity("trade", "BUY 2.5000 NVDAx @ $200.00"); err != nil {
		t.Fatal(err)
	}
	loaded := &Ledger{Path: path}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(data, loaded); err != nil {
		t.Fatal(err)
	}
	if loaded.SolBalance != 97.5 {
		t.Fatalf("persisted balance = %v, want 97.5", loaded.SolBalance)
	}
	if !loaded.SolSeeded {
		t.Fatal("SolSeeded should persist")
	}
	if len(loaded.Activities) != 1 || loaded.Activities[0].Text == "" {
		t.Fatal("activities should persist")
	}
}

func TestSeedIsIdempotent(t *testing.T) {
	l := &Ledger{Path: t.TempDir() + "/trades.json"}
	if err := l.SeedSol(100); err != nil {
		t.Fatal(err)
	}
	if err := l.SeedSol(999); err != nil {
		t.Fatal(err)
	}
	if l.SolBalance != 100 {
		t.Fatalf("seed overwrote baseline: %v", l.SolBalance)
	}
}
func TestRecordPersists(t *testing.T) {
	path := t.TempDir() + "/trades.json"
	l := &Ledger{Path: path}
	if err := l.Record("AAPLx", "buy", 1, 42); err != nil {
		t.Fatal(err)
	}
	loaded := &Ledger{Path: path}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(data, loaded); err != nil {
		t.Fatal(err)
	}
	if got := loaded.NetHolding("AAPLx"); got != 1 {
		t.Fatalf("loaded net = %v, want 1", got)
	}
}