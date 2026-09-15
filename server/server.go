// Package server exposes STOCK.sh as an HTTP service: a live price/portfolio
// dashboard (Vite + React build embedded from webdist/) plus a small JSON API
// for quoting and executing (paper or live) swaps. It reuses the same Jupiter
// + ledger logic as the TUI.
package server

import (
	"embed"
	"encoding/json"
	"fmt"
	"io/fs"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"stocksh/jupiter"
	"stocksh/ledger"
	"stocksh/solana"
)

//go:embed all:webdist
var staticFS embed.FS

// TickerOrder keeps the display order stable across the TUI and the web UI.
var TickerOrder = []string{"NVDAx", "TSLAx", "AAPLx", "METAx", "SPYx", "QQQx", "MSTRx", "CRCLx"}

type Server struct {
	jup     *jupiter.Client
	led     *ledger.Ledger
	dryRun  bool
	network string

	mu     sync.RWMutex
	prices map[string]jupiter.PriceInfo
}

func New() *Server {
	dry := os.Getenv("STOCKSH_LIVE") != "1"
	s := &Server{
		jup:     jupiter.NewClient(),
		led:     ledger.Load(),
		dryRun:  dry,
		network: "devnet",
		prices:  map[string]jupiter.PriceInfo{},
	}
	if p := os.Getenv("STOCKSH_LEDGER"); p != "" {
		s.led = ledger.LoadPath(p)
	} else if os.Getenv("STOCKSH_DEMO") == "1" {
		s.led = ledger.LoadPath(".stocksh_demo.json")
	}
	if !solana.IsDevnet("") {
		s.network = "mainnet"
	}
	// Paper account baseline, exactly like the TUI: a fresh devnet dry-run
	// gets 100 paper SOL so the portfolio has something to trade against.
	if s.dryRun && s.network == "devnet" && !s.led.SolSeeded {
		_ = s.led.SeedSol(100)
	}
	return s
}

// priceMints returns every mint the dashboard tracks: the 8 xStocks plus SOL
// and USDC for rate/balance conversion.
func (s *Server) priceMints() []string {
	mints := make([]string, 0, len(jupiter.Stocks)+2)
	mints = append(mints, jupiter.SolMint, jupiter.USDC)
	for _, m := range jupiter.Stocks {
		mints = append(mints, m)
	}
	return mints
}

// refreshLoop repopulates the price cache every 5s.
func (s *Server) refreshLoop() {
	for {
		prices, err := s.jup.GetPrices(s.priceMints())
		if err == nil {
			s.mu.Lock()
			s.prices = prices
			s.mu.Unlock()
		}
		time.Sleep(5 * time.Second)
	}
}

func (s *Server) getPrices() map[string]jupiter.PriceInfo {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make(map[string]jupiter.PriceInfo, len(s.prices))
	for k, v := range s.prices {
		out[k] = v
	}
	return out
}

func (s *Server) solPrice(prices map[string]jupiter.PriceInfo) float64 {
	if p, ok := prices[jupiter.SolMint]; ok {
		return p.USDPrice
	}
	return 0
}

func (s *Server) stockPrice(prices map[string]jupiter.PriceInfo, mint string) float64 {
	if p, ok := prices[mint]; ok {
		if p.Stock != nil && p.Stock.Price > 0 {
			return p.Stock.Price
		}
		return p.USDPrice
	}
	return 0
}

func (s *Server) Run() error {
	mux := http.NewServeMux()
	mux.HandleFunc("/", s.handleIndex)
	mux.HandleFunc("/api/prices", s.handlePrices)
	mux.HandleFunc("/api/portfolio", s.handlePortfolio)
	mux.HandleFunc("/api/history", s.handleHistory)
	mux.HandleFunc("/api/activity", s.handleActivity)
	mux.HandleFunc("/api/quote", s.handleQuote)
	mux.HandleFunc("/api/execute", s.handleExecute)
	mux.HandleFunc("/healthz", s.handleHealth)

	go s.refreshLoop()

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	log.Printf("stocksh serve %s — paper=%v — listening on :%s", s.network, s.dryRun, port)
	return http.ListenAndServe(":"+port, mux)
}

func (s *Server) handleIndex(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path == "/healthz" || strings.HasPrefix(r.URL.Path, "/api/") {
		http.NotFound(w, r)
		return
	}
	w.Header().Set("Cache-Control", "no-store")
	content, err := fs.Sub(staticFS, "webdist")
	if err != nil {
		http.Error(w, "template error", http.StatusInternalServerError)
		return
	}
	http.FileServer(http.FS(content)).ServeHTTP(w, r)
}

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	fmt.Fprint(w, `{"ok":true}`)
}

// writeJSON is the small helper for all API responses.
func (s *Server) writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(v); err != nil {
		log.Printf("json encode: %v", err)
	}
}

func (s *Server) handlePrices(w http.ResponseWriter, r *http.Request) {
	prices := s.getPrices()
	symbols := make([]map[string]any, 0, len(TickerOrder))
	for _, sym := range TickerOrder {
		mint := jupiter.Stocks[sym]
		p := prices[mint]
		mark := p.USDPrice
		if p.Stock != nil && p.Stock.Price > 0 {
			mark = p.Stock.Price
		}
		symbols = append(symbols, map[string]any{
			"symbol":     sym,
			"price":      p.USDPrice,
			"change24h":  p.PriceChange24h,
			"mark":       mark,
			"liquidity":  p.Liquidity,
			"loaded":     p.USDPrice > 0,
		})
	}
	s.writeJSON(w, http.StatusOK, map[string]any{
		"network": s.network,
		"paper":   s.dryRun,
		"sol":     prices[jupiter.SolMint].USDPrice,
		"symbols": symbols,
	})
}

type apiPosition struct {
	Symbol    string  `json:"symbol"`
	Qty       float64 `json:"qty"`
	Price     float64 `json:"price"`
	HasPx     bool    `json:"hasPx"`
	Value     float64 `json:"value"`
	AvgCost   float64 `json:"avgCost"`
	PnL       float64 `json:"pnl"`
	PnLPct    float64 `json:"pnlPct"`
	HasPnL    bool    `json:"hasPnL"`
	AllocPct  float64 `json:"allocPct"`
}

func (s *Server) handlePortfolio(w http.ResponseWriter, r *http.Request) {
	prices := s.getPrices()
	positions := []apiPosition{}
	totalValue, totalPnL := 0.0, 0.0
	for _, sym := range TickerOrder {
		qty := s.led.NetHolding(sym)
		if qty <= 0 {
			continue
		}
		price := s.stockPrice(prices, jupiter.Stocks[sym])
		p := apiPosition{Symbol: sym, Qty: qty, Price: price, HasPx: price > 0}
		p.Value = qty * price
		totalValue += p.Value
		if h := s.led.Holding(sym, price); h.HasBasis {
			p.AvgCost, p.PnL, p.PnLPct, p.HasPnL = h.AvgCost, h.PnL, h.PnLPct, true
			totalPnL += h.PnL
		}
		positions = append(positions, p)
	}
	for i := range positions {
		if totalValue > 0 {
			positions[i].AllocPct = positions[i].Value / totalValue * 100
		}
	}
	addr, _ := solana.LoadWallet()
	address := ""
	mode := "paper"
	if addr != nil {
		address = addr.PubKey.String()
	}
	if !s.dryRun {
		mode = "live"
	}
	s.writeJSON(w, http.StatusOK, map[string]any{
		"paper":      s.dryRun,
		"network":    s.network,
		"mode":       mode,
		"address":    address,
		"tradeCount": len(s.led.Trades),
		"solBalance": s.led.SolBalance,
		"solPrice":   s.solPrice(prices),
		"positions":  positions,
		"totalValue": totalValue,
		"totalPnL":   totalPnL,
	})
}

func (s *Server) handleHistory(w http.ResponseWriter, r *http.Request) {
	if s.led.Trades == nil {
		s.led.Trades = []ledger.Trade{}
	}
	s.writeJSON(w, http.StatusOK, s.led.Trades)
}

func (s *Server) handleActivity(w http.ResponseWriter, r *http.Request) {
	if s.led.Activities == nil {
		s.led.Activities = []ledger.Activity{}
	}
	s.writeJSON(w, http.StatusOK, s.led.Activities)
}

// quoteRequest is the JSON body for /api/quote and /api/execute.
type quoteRequest struct {
	Symbol string `json:"symbol"`
	Side   string `json:"side"` // buy (default) or sell
	USDC   int64  `json:"usdc"` // order size in whole-USDC terms
}

// lookupStock matches a symbol case-insensitively and returns the canonical
// spelling (NVDAx, TSLAx, ...) plus its mint. The display names end in a
// lowercase 'x', so naive ToUpper would miss them.
func lookupStock(symbol string) (canonical, mint string, ok bool) {
	for sym, m := range jupiter.Stocks {
		if strings.EqualFold(sym, symbol) {
			return sym, m, true
		}
	}
	return "", "", false
}

func (rr quoteRequest) validate() (canonical, mint string, side string, err error) {
	canonical, mint, ok := lookupStock(rr.Symbol)
	if !ok {
		return "", "", "", fmt.Errorf("unknown symbol %q", rr.Symbol)
	}
	side = strings.ToLower(rr.Side)
	if side == "" {
		side = "buy"
	}
	if side != "buy" && side != "sell" {
		return "", "", "", fmt.Errorf("side must be buy or sell, got %q", rr.Side)
	}
	if rr.USDC < 1 {
		return "", "", "", fmt.Errorf("usdc must be >= 1")
	}
	return canonical, mint, side, nil
}

// buildQuote mirrors the TUI: buy sizes in USDC lamports, sell sizes convert
// to token quantity capped at the current holding.
func (s *Server) buildQuote(rr quoteRequest) (*jupiter.QuoteResponse, string, float64, float64, error) {
	canonical, mint, side, err := rr.validate()
	if err != nil {
		return nil, "", 0, 0, err
	}
	prices := s.getPrices()
	price := s.stockPrice(prices, mint)
	if price <= 0 {
		return nil, "", 0, 0, fmt.Errorf("no price for %s yet — try again in a few seconds", canonical)
	}
	var input, output string
	var amount uint64
	if side == "buy" {
		input, output = jupiter.USDC, mint
		amount = uint64(rr.USDC) * 1_000_000
	} else {
		input, output = mint, jupiter.USDC
		held := s.led.NetHolding(canonical)
		if held <= 0 {
			return nil, "", 0, 0, fmt.Errorf("nothing to sell — you hold 0 %s", canonical)
		}
		ammt := float64(rr.USDC)
		if ammt/price > held {
			ammt = held * price
		}
		amount = uint64(ammt/price*1_000_000 + 0.5)
		if amount == 0 {
			return nil, "", 0, 0, fmt.Errorf("sale too small")
		}
	}
	quote, err := s.jup.GetQuote(input, output, amount, 50)
	return quote, canonical, price, s.solPrice(prices), err
}

func (s *Server) handleQuote(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		s.writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "POST only"})
		return
	}
	var rr quoteRequest
	if err := json.NewDecoder(r.Body).Decode(&rr); err != nil {
		s.writeJSON(w, http.StatusBadRequest, map[string]string{"error": "bad json: " + err.Error()})
		return
	}
	quote, canonical, price, solPrice, err := s.buildQuote(rr)
	if err != nil {
		s.writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	s.writeJSON(w, http.StatusOK, map[string]any{
		"symbol":   canonical,
		"side":     strings.ToLower(rr.Side),
		"price":    price,
		"solPrice": solPrice,
		"quote":    quote,
		"dryRun":   s.dryRun,
		"network":  s.network,
	})
}

// recordTrade writes the swap into the ledger, including the persistent SOL
// debit/credit, using the exact same math as the TUI.
func (s *Server) recordTrade(symbol, side string, quote *jupiter.QuoteResponse, solPrice float64) error {
	in, err := strconv.ParseUint(quote.InAmount, 10, 64)
	if err != nil || in == 0 {
		return fmt.Errorf("bad inAmount %q", quote.InAmount)
	}
	out, err := strconv.ParseUint(quote.OutAmount, 10, 64)
	if err != nil || out == 0 {
		return fmt.Errorf("bad outAmount %q", quote.OutAmount)
	}
	var signedQty, price float64
	if side == "buy" {
		price = float64(in) / float64(out)
		signedQty = float64(out) / 1_000_000
	} else {
		price = float64(out) / float64(in)
		signedQty = -float64(in) / 1_000_000
	}
	if err := s.led.Record(symbol, side, signedQty, price); err != nil {
		return err
	}
	var solDelta float64
	if side == "buy" {
		spend := float64(in) / 1_000_000
		if solPrice > 0 {
			solDelta = spend / solPrice
		} else {
			solDelta = spend
		}
		solDelta = -solDelta
	} else {
		spend := float64(out) / 1_000_000
		if solPrice > 0 {
			solDelta = spend / solPrice
		} else {
			solDelta = spend
		}
	}
	_ = s.led.AdjSol(solDelta)
	label := "BUY"
	if side == "sell" {
		label = "SELL"
	}
	qty := signedQty
	if qty < 0 {
		qty = -qty
	}
	_ = s.led.RecordActivity("trade", fmt.Sprintf("%s %.4f %s @ $%.2f (SOL %.4f)", label, qty, symbol, price, solDelta))
	return nil
}

func (s *Server) handleExecute(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		s.writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "POST only"})
		return
	}
	var rr quoteRequest
	if err := json.NewDecoder(r.Body).Decode(&rr); err != nil {
		s.writeJSON(w, http.StatusBadRequest, map[string]string{"error": "bad json: " + err.Error()})
		return
	}
	quote, canonical, _, solPrice, err := s.buildQuote(rr)
	if err != nil {
		s.writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}

	if s.dryRun || s.network == "devnet" {
		if err := s.recordTrade(canonical, strings.ToLower(rr.Side), quote, solPrice); err != nil {
			s.writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
			return
		}
		s.writeJSON(w, http.StatusOK, map[string]any{
			"status":  "dry-run",
			"message": fmt.Sprintf("DRY-RUN OK — paper position recorded for %s", canonical),
			"symbol":  canonical,
			"side":    strings.ToLower(rr.Side),
			"quote":   quote,
		})
		return
	}

	// Live: prepare the swap transaction and broadcast it with the wallet.
	wallet, err := solana.LoadWallet()
	if err != nil {
		s.writeJSON(w, http.StatusBadRequest, map[string]string{"error": "live mode needs SOLANA_PRIVATE_KEY: " + err.Error()})
		return
	}
	swapResp, err := s.jup.GetSwapTransaction(quote, wallet.PubKey.String())
	if err != nil {
		s.writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	sig, err := solana.SignAndSend(solana.NewRPC(""), wallet, swapResp.SwapTransaction)
	if err != nil {
		s.writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	s.writeJSON(w, http.StatusOK, map[string]any{
		"status":       "live",
		"message":      "swap broadcast",
		"signature":    sig,
		"symbol":       canonical,
		"side":         strings.ToLower(rr.Side),
	})
}

// Main is the entry point for the `stocksh serve` subcommand.
func Main() int {
	return run()
}

func run() int {
	if err := New().Run(); err != nil {
		log.Printf("serve: %v", err)
		return 1
	}
	return 0
}