package ui

import (
	"fmt"
	"os"
	"strconv"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"stocksh/jupiter"
	"stocksh/ledger"
	"stocksh/solana"
)

type viewMode int

const (
	viewSplash viewMode = iota
	viewTickers
	viewCustomAmount
	viewConfirm
	viewResult
	viewPortfolio
	viewHistory
	viewWatchlist
	viewHelp
)

type Ticker struct {
	Symbol  string
	Mint    string
	Price   string
	PriceV  float64
	Change  string
	ChgV    float64
	History []float64
	Watch   bool
	Alert   float64 // 0 = no alert; otherwise target price in USD
	alertUp bool    // last known relationship of price to target
}

type Model struct {
	mode          viewMode
	tickers       []Ticker
	cursor        int
	width, height int
	selected      Ticker
	amountUSDC    float64
	customBuf     string
	quote         *jupiter.QuoteResponse
	status        string
	lastSig       string
	errMsg        string
	dryRun        bool
	side          string
	solBalance    float64
	solLoaded     bool
	tokenBalances []solana.TokenBalance
	tokensLoaded  bool
	jup           *jupiter.Client
	wallet        *solana.Wallet
	network       string
	styles        Styles
	splashTicks   int
	walletStatus  string
	led           *ledger.Ledger
	watchCursor   int
}

func InitialModel() Model {
	stocks := make([]Ticker, 0, len(jupiter.Stocks))
	order := []string{"NVDAx", "TSLAx", "AAPLx", "METAx", "SPYx", "QQQx", "MSTRx", "CRCLx"}
	for _, sym := range order {
		if mint, ok := jupiter.Stocks[sym]; ok {
			stocks = append(stocks, Ticker{Symbol: sym, Mint: mint, Price: "-", Change: ""})
		}
	}
	dry := os.Getenv("STOCKSH_LIVE") != "1"
	net := "devnet"
	if !solana.IsDevnet("") {
		net = "mainnet"
	}
	return Model{
		mode: viewSplash, tickers: stocks, cursor: 0, amountUSDC: 10.0,
		jup: jupiter.NewClient(), dryRun: dry, network: net, side: "buy",
		status:    "Ready - up/down select - Enter quote - p portfolio - ? help - q quit",
		styles:    NewStyles(),
		walletStatus: "Connecting to Solana...",
		led: ledger.Load(),
	}
}

func (m Model) Init() tea.Cmd {
	return tea.Batch(tickCmd(), m.loadWallet(), m.fetchAllPrices(), splashCmd())
}

type splashMsg struct{ n int }

func splashCmd() tea.Cmd {
	return tea.Tick(time.Second, func(t time.Time) tea.Msg { return splashMsg{1} })
}

type tickMsg time.Time
type walletLoadedMsg struct {
	wallet *solana.Wallet
	err    error
}
type quoteMsg struct {
	quote *jupiter.QuoteResponse
	err   error
}
type swapResultMsg struct {
	sig string
	err error
}
type balanceMsg struct {
	sol float64
	err error
}
type tokensMsg struct {
	balances []solana.TokenBalance
	err      error
}
type pricesMsg struct {
	prices map[string]jupiter.PriceInfo
	err    error
}

func tickCmd() tea.Cmd {
	return tea.Tick(time.Second*5, func(t time.Time) tea.Msg { return tickMsg(t) })
}

func (m Model) loadWallet() tea.Cmd {
	return func() tea.Msg {
		w, err := solana.LoadWallet()
		return walletLoadedMsg{wallet: w, err: err}
	}
}

func (m Model) fetchBalance() tea.Cmd {
	return func() tea.Msg {
		if m.wallet == nil {
			return balanceMsg{err: fmt.Errorf("no wallet")}
		}
		client := solana.NewRPC("")
		bal, err := solana.GetBalance(client, m.wallet.PubKey)
		return balanceMsg{sol: bal, err: err}
	}
}

func (m Model) fetchTokens() tea.Cmd {
	return func() tea.Msg {
		if m.wallet == nil {
			return tokensMsg{err: fmt.Errorf("no wallet")}
		}
		mintToSym := make(map[string]string, len(jupiter.Stocks))
		for sym, mint := range jupiter.Stocks {
			mintToSym[mint] = sym
		}
		client := solana.NewRPC("")
		bals, err := solana.GetTokenBalances(client, m.wallet.PubKey, mintToSym)
		return tokensMsg{balances: bals, err: err}
	}
}

func (m Model) fetchAllPrices() tea.Cmd {
	return func() tea.Msg {
		mints := make([]string, 0, len(m.tickers))
		for _, t := range m.tickers {
			mints = append(mints, t.Mint)
		}
		prices, err := m.jup.GetPrices(mints)
		return pricesMsg{prices: prices, err: err}
	}
}

func (m Model) fetchQuote() tea.Cmd {
	return func() tea.Msg {
		amount := uint64(m.amountUSDC * 1_000_000)
		var inputMint, outputMint string
		if m.side == "buy" {
			inputMint, outputMint = jupiter.USDC, m.selected.Mint
		} else {
			inputMint, outputMint = m.selected.Mint, jupiter.USDC
		}
		q, err := m.jup.GetQuote(inputMint, outputMint, amount, 50)
		return quoteMsg{quote: q, err: err}
	}
}

func (m Model) executeSwap() tea.Cmd {
	return func() tea.Msg {
		if m.quote == nil {
			return swapResultMsg{err: fmt.Errorf("missing quote")}
		}
		if m.wallet == nil {
			return swapResultMsg{err: fmt.Errorf("no wallet loaded")}
		}
		swapResp, err := m.jup.GetSwapTransaction(m.quote, m.wallet.PubKey.String())
		if err != nil {
			return swapResultMsg{err: err}
		}
		if m.dryRun || m.network == "devnet" {
			return swapResultMsg{
				sig: fmt.Sprintf("DRY-RUN OK - tx prepared (%d bytes)", len(swapResp.SwapTransaction)),
				err: nil,
			}
		}
		rpcClient := solana.NewRPC("")
		sig, err := solana.SignAndSend(rpcClient, m.wallet, swapResp.SwapTransaction)
		if err != nil {
			return swapResultMsg{err: err}
		}
		return swapResultMsg{sig: sig, err: nil}
	}
}

func formatUSDC(raw string) string {
	v, err := strconv.ParseUint(raw, 10, 64)
	if err != nil {
		return raw
	}
	return fmt.Sprintf("%.2f", float64(v)/1_000_000)
}

func formatStock(raw string) string {
	v, err := strconv.ParseUint(raw, 10, 64)
	if err != nil {
		return raw
	}
	return fmt.Sprintf("%.6f", float64(v)/1_000_000)
}

type Styles struct {
	Title, Selected, Normal, Status, Error, Success, Border, Dim, Header, Green, Cyan, Yellow lipgloss.Style
}

func NewStyles() Styles {
	return Styles{
		Title: lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#0B0F0C")).
			Background(lipgloss.Color("#00FF9F")).
			Padding(0, 2),
		Selected: lipgloss.NewStyle().
			Foreground(lipgloss.Color("#000000")).
			Background(lipgloss.Color("#00FF9F")).
			Bold(true),
		Normal: lipgloss.NewStyle().
			Foreground(lipgloss.Color("#F0F0F0")).
			Padding(0, 2),
		Status: lipgloss.NewStyle().
			Foreground(lipgloss.Color("#AAAAAA")).
			Padding(0, 1),
		Error: lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FF5555")).
			Bold(true),
		Success: lipgloss.NewStyle().
			Foreground(lipgloss.Color("#00FF9F")).
			Bold(true),
		Border: lipgloss.NewStyle().
			Border(lipgloss.ThickBorder()).
			BorderForeground(lipgloss.Color("#00FF9F")).
			Padding(1, 3),
		Dim: lipgloss.NewStyle().
			Foreground(lipgloss.Color("#777777")),
		Header: lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#00D4FF")).
			Padding(0, 1),
		Green: lipgloss.NewStyle().
			Foreground(lipgloss.Color("#00FF9F")),
		Cyan: lipgloss.NewStyle().
			Foreground(lipgloss.Color("#00D4FF")),
		Yellow: lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FFD700")),
	}
}
