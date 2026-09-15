// Package ledger persists trade history to disk and provides the position
// math (net holdings, average cost basis, unrealized P&L) used by the
// portfolio view. The same ledger backs both paper-trading (dry-run) and
// live positions, so the portfolio always has a cost basis to work with.
package ledger

import (
	"encoding/json"
	"os"
	"path/filepath"
	"time"
)

type Trade struct {
	Symbol    string    `json:"symbol"`
	Side      string    `json:"side"`
	Qty       float64   `json:"qty"`   // signed: +buy, -sell
	PriceUSDC float64   `json:"price"` // price per token at execution
	Time      time.Time `json:"time"`
}

// Holding is a computed position for one symbol shown in the portfolio.
type Holding struct {
	Symbol      string
	Qty         float64
	AvgCost     float64 // USDC per token (nil cost basis => 0)
	TotalCost   float64
	ValueUSDC   float64
	PnL         float64
	PnLPct      float64
	HasBasis    bool
}

type Ledger struct {
	Trades []Trade `json:"trades"`
	Path   string  `json:"-"`
}

func DefaultPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return ".stocksh_trades.json", nil
	}
	return filepath.Join(home, ".stocksh", "trades.json"), nil
}

// Load reads the ledger from the default location. A missing file yields an
// empty ledger (no error) so first-run is a no-op.
func Load() *Ledger {
	path, err := DefaultPath()
	if err != nil {
		path = ".stocksh_trades.json"
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return &Ledger{Path: path}
	}
	var l Ledger
	if err := json.Unmarshal(data, &l); err != nil {
		return &Ledger{Path: path}
	}
	l.Path = path
	return &l
}

// Record appends a signed trade and persists it. Trades are kept on disk so
// cost basis survives restarts.
func (l *Ledger) Record(symbol, side string, signedQty, priceUSDC float64) error {
	l.Trades = append(l.Trades, Trade{
		Symbol: symbol, Side: side, Qty: signedQty, PriceUSDC: priceUSDC, Time: time.Now(),
	})
	return l.save()
}

func (l *Ledger) save() error {
	if l.Path == "" {
		l.Path, _ = DefaultPath()
	}
	if dir := filepath.Dir(l.Path); dir != "." {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return err
		}
	}
	data, err := json.MarshalIndent(l, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(l.Path, data, 0o644)
}

// NetHolding returns the signed quantity a wallet holds for a symbol.
func (l *Ledger) NetHolding(symbol string) float64 {
	var q float64
	for _, t := range l.Trades {
		if t.Symbol == symbol {
			q += t.Qty
		}
	}
	return q
}

// Holding computes the full position (basis + P&L) for a symbol at a price.
// Sells reduce the position at the average buy cost.
func (l *Ledger) Holding(symbol string, priceUSDC float64) Holding {
	var buyQty, buyCost float64
	for _, t := range l.Trades {
		if t.Symbol != symbol || t.Qty <= 0 {
			continue
		}
		buyQty += t.Qty
		buyCost += t.Qty * t.PriceUSDC
	}
	net := l.NetHolding(symbol)
	h := Holding{Symbol: symbol, Qty: net}
	if buyQty > 0 && net > 0 {
		h.AvgCost = buyCost / buyQty
		h.TotalCost = net * h.AvgCost
		h.ValueUSDC = net * priceUSDC
		h.PnL = h.ValueUSDC - h.TotalCost
		if h.TotalCost > 0 {
			h.PnLPct = (h.PnL / h.TotalCost) * 100
		}
		h.HasBasis = true
	} else if net > 0 {
		h.ValueUSDC = net * priceUSDC
	}
	return h
}