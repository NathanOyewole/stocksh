package ui

import (
	"fmt"
	"os"
	"strconv"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"stocksh/jupiter"
	"stocksh/solana"
)

type viewMode int

const (
	viewTickers viewMode = iota
	viewConfirm
	viewResult
	viewPortfolio
	viewHelp
)

type Ticker struct {
	Symbol string
	Mint   string
	Price  string
	Change string
}

type Model struct {
	mode          viewMode
	tickers       []Ticker
	cursor        int
	width, height int

	selected   Ticker
	amountUSDC float64
	quote      *jupiter.QuoteResponse
	status     string
	lastSig    string
	errMsg     string
	dryRun     bool
	side       string

	solBalance    float64
	solLoaded     bool
	tokenBalances []solana.TokenBalance
	tokensLoaded  bool

	jup     *jupiter.Client
	wallet  *solana.Wallet
	network string

	styles Styles
}

func InitialModel() Model {
	stocks := make([]Ticker, 0, len(jupiter.Stocks))
	order := []string{"NVDAx", "TSLAx", "AAPLx", "METAx", "SPYx", "QQQx", "MSTRx", "CRCLx"}
	for _, sym := range order {
		if mint, ok := jupiter.Stocks[sym]; ok {
			stocks = append(stocks, Ticker{Symbol: sym, Mint: mint, Price: "—", Change: ""})
		}
	}

	dry := os.Getenv("STOCKSH_LIVE") != "1"

	net := "devnet"
	if !solana.IsDevnet("") {
		net = "mainnet"
	}

	return Model{
		mode:       viewTickers,
		tickers:    stocks,
		cursor:     0,
		amountUSDC: 10.0,
		jup:        jupiter.NewClient(),
		dryRun:     dry,
		network:    net,
		side:       "buy",
		status:     "Ready · ↑↓ select · Enter quote · p portfolio · ? help · q quit",
		styles:     NewStyles(),
	}
}
