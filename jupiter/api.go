package jupiter

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

const (
	BaseURL  = "https://lite-api.jup.ag/swap/v1"
	PriceURL = "https://lite-api.jup.ag/price/v3"
	USDC     = "EPjFWdd5AufqSSqeM2qN1xzybapC8G4wEGGkZwyTDt1v"
)

var Stocks = map[string]string{
	"NVDAx": "Xsc9qvGR1efVDFGLrVsmkzv3qi45LTBjeUKSPmx9qEh",
	"TSLAx": "XsDoVfqeBukxuZHWhdvWHBhgEHjGNst4MLodqsJHzoB",
	"AAPLx": "XsbEhLAtcf6HdfpFZ5xEMdqW8nfAvcsP5bdudRLJzJp",
	"METAx": "Xsa62P5mvPszXL1krVUnU5ar38bBSVcWAB6fmPCo5Zu",
	"SPYx":  "XsoCS1TfEyfFhfvj8EtZ528L3CaKBDBRqRapnBbDF2W",
	"QQQx":  "Xs8S1uUs1zvS2p7iwtsG3b6fkhpvmwz4GYU3gWAmWHZ",
	"MSTRx": "XsP7xzNPvEHS1m6qfanPUGjNmdnmsLKEoNAnHjdxxyZ",
	"CRCLx": "XsueG8BtpquVJX9LVLLEGuViXUungE6WmK5YZ3p3bd1",
}

type QuoteResponse struct {
	InputMint            string  `json:"inputMint"`
	InAmount             string  `json:"inAmount"`
	OutputMint           string  `json:"outputMint"`
	OutAmount            string  `json:"outAmount"`
	OtherAmountThreshold string  `json:"otherAmountThreshold"`
	SwapMode             string  `json:"swapMode"`
	SlippageBps          int     `json:"slippageBps"`
	PriceImpactPct       string  `json:"priceImpactPct"`
	RoutePlan            []any   `json:"routePlan"`
	ContextSlot          uint64  `json:"contextSlot"`
	TimeTaken            float64 `json:"timeTaken"`
}

type SwapResponse struct {
	SwapTransaction      string `json:"swapTransaction"`
	LastValidBlockHeight uint64 `json:"lastValidBlockHeight"`
}

type PriceInfo struct {
	USDPrice       float64 `json:"usdPrice"`
	PriceChange24h float64 `json:"priceChange24h"`
	Decimals       int     `json:"decimals"`
}

type Client struct {
	httpClient *http.Client
}

func NewClient() *Client {
	return &Client{httpClient: &http.Client{Timeout: 12 * time.Second}}
}

func (c *Client) GetPrices(mints []string) (map[string]PriceInfo, error) {
	if len(mints) == 0 {
		return nil, nil
	}
	u := PriceURL + "?ids=" + url.QueryEscape(strings.Join(mints, ","))
	resp, err := c.httpClient.Get(u)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("price error %d: %s", resp.StatusCode, string(body))
	}
	var raw map[string]PriceInfo
	if err := json.Unmarshal(body, &raw); err != nil {
		return nil, err
	}
	return raw, nil
}

func (c *Client) GetQuote(inputMint, outputMint string, amount uint64, slippageBps int) (*QuoteResponse, error) {
	u, _ := url.Parse(BaseURL + "/quote")
	q := u.Query()
	q.Set("inputMint", inputMint)
	q.Set("outputMint", outputMint)
	q.Set("amount", strconv.FormatUint(amount, 10))
	q.Set("slippageBps", strconv.Itoa(slippageBps))
	q.Set("restrictIntermediateTokens", "true")
	u.RawQuery = q.Encode()
	resp, err := c.httpClient.Get(u.String())
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("quote error %d: %s", resp.StatusCode, string(body))
	}
	var quote QuoteResponse
	if err := json.Unmarshal(body, &quote); err != nil {
		return nil, err
	}
	return &quote, nil
}

func (c *Client) GetSwapTransaction(quote *QuoteResponse, userPublicKey string) (*SwapResponse, error) {
	payload := map[string]any{
		"quoteResponse":           quote,
		"userPublicKey":           userPublicKey,
		"dynamicComputeUnitLimit": true,
		"dynamicSlippage":         true,
	}
	body, _ := json.Marshal(payload)
	req, err := http.NewRequest("POST", BaseURL+"/swap", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	respBody, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("swap error %d: %s", resp.StatusCode, string(respBody))
	}
	var swapResp SwapResponse
	if err := json.Unmarshal(respBody, &swapResp); err != nil {
		return nil, err
	}
	return &swapResp, nil
}
