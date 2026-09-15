package ui

import (
	"strings"
	"testing"

	"stocksh/ledger"
)

func TestViewHistory(t *testing.T) {
	m := InitialModel()
	m.led = &ledger.Ledger{Path: t.TempDir() + "/t.json"}
	_ = m.led.Record("NVDAx", "buy", 4, 370)
	_ = m.led.Record("TSLAx", "sell", -2, 240)

	out := m.viewHistory()
	for _, want := range []string{"NVDAx", "TSLAx", "BUY", "SELL", "$370.00"} {
		if !strings.Contains(out, want) {
			t.Fatalf("history missing %q:\n%s", want, out)
		}
	}
}

func TestViewHistoryEmpty(t *testing.T) {
	m := InitialModel()
	m.led = &ledger.Ledger{Path: t.TempDir() + "/t.json"}
	out := m.viewHistory()
	if !strings.Contains(out, "No trades yet") {
		t.Fatalf("empty history should say 'No trades yet':\n%s", out)
	}
}