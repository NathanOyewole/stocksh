package ui

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"stocksh/ledger"
)

func TestWatchlistEmpty(t *testing.T) {
	m := InitialModel()
	m.led = &ledger.Ledger{}
	out := m.viewWatchlist()
	if !strings.Contains(out, "Nothing tracked yet") {
		t.Fatal("empty watchlist should show message")
	}
}

func TestWatchlistToggleAndAlert(t *testing.T) {
	m := InitialModel()
	m.tickers = []Ticker{
		{Symbol: "NVDAx", PriceV: 390},
	}
	m.led = &ledger.Ledger{Path: t.TempDir() + "/t.json"}

	// * toggles the first ticker as watched
	updated, _ := m.updateTickers(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'*'}})
	m2 := updated.(Model)
	if !m2.tickers[0].Watch {
		t.Fatal("first * should watch")
	}

	// * again should unwatch
	updated2, _ := m2.updateTickers(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'*'}})
	m3 := updated2.(Model)
	if m3.tickers[0].Watch {
		t.Fatal("second * should unwatch")
	}
}

func TestViewWatchlistShowsWatched(t *testing.T) {
	m := InitialModel()
	m.tickers = []Ticker{
		{Symbol: "NVDAx", Watch: true, Alert: 400, PriceV: 390, Price: "$390.00", Change: "+1.0%", ChgV: 1.0},
		{Symbol: "TSLAx", Watch: false},
	}
	m.led = &ledger.Ledger{}
	out := m.viewWatchlist()
	if !strings.Contains(out, "NVDAx") {
		t.Fatal("missing watched symbol")
	}
	if strings.Contains(out, "TSLAx") {
		t.Fatal("unwatched symbol should not appear")
	}
	if !strings.Contains(out, "armed") {
		t.Fatal("should show alert status")
	}
}