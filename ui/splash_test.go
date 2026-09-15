package ui

import (
	"strings"
	"testing"
)

func TestSplashRenders(t *testing.T) {
	m := InitialModel()
	m.width = 100
	m.height = 30
	m.walletStatus = "Wallet connected - DRY-RUN"
	out := m.View()
	if out == "" {
		t.Fatal("empty splash")
	}
	if !strings.Contains(out, "Terminal xStocks") {
		t.Fatal("splash missing title")
	}
	if m.mode != viewSplash {
		t.Fatal("initial mode should be splash")
	}
}

func TestSplashAutoAdvances(t *testing.T) {
	m := InitialModel()
	updated, _ := m.Update(splashMsg{n: 1})
	m2 := updated.(Model)
	if m2.mode != viewSplash {
		t.Fatal("one tick should not advance yet")
	}
	updated2, _ := m2.Update(splashMsg{n: 2})
	m3 := updated2.(Model)
	if m3.mode != viewTickers {
		t.Fatal("splash should advance to tickers after enough ticks")
	}
}