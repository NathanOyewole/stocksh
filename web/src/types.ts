export interface SymbolQuote {
  symbol: string;
  price: number;
  change24h: number;
  mark: number;
  liquidity: number;
  loaded: boolean;
}

export interface PricesResponse {
  network: string;
  paper: boolean;
  sol: number;
  symbols: SymbolQuote[];
}

export interface Position {
  symbol: string;
  qty: number;
  price: number;
  hasPx: boolean;
  value: number;
  avgCost: number;
  pnl: number;
  pnlPct: number;
  hasPnL: boolean;
  allocPct: number;
}

export interface PortfolioResponse {
  paper: boolean;
  solBalance: number;
  solPrice: number;
  positions: Position[];
  totalValue: number;
  totalPnL: number;
}

export interface Trade {
  symbol: string;
  side: "buy" | "sell";
  qty: number;
  price: number;
  time: string;
}

export interface Activity {
  time: string;
  type: string;
  text: string;
}

export interface JupiterQuote {
  inAmount: string;
  outAmount: string;
  slippageBps: number;
  priceImpactPct: string;
  swapMode: string;
}

export interface QuoteResponse {
  symbol: string;
  side: string;
  price: number;
  solPrice: number;
  dryRun: boolean;
  network: string;
  quote: JupiterQuote;
}

export interface ExecuteResponse {
  status: string;
  message: string;
  symbol: string;
  side: string;
  quote?: JupiterQuote;
  signature?: string;
}

export interface QuoteRequest {
  symbol: string;
  side: string;
  usdc: number;
}