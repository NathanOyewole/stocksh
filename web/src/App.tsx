import { useCallback, useEffect, useState } from "react";
import { useStockData, updatedAgo } from "./hooks/useStockData";
import { TickerTape } from "./components/TickerTape";
import { MarketTable } from "./components/MarketTable";
import { Portfolio } from "./components/Portfolio";
import { TradePanel } from "./components/TradePanel";
import { Feed } from "./components/Feed";
import { ToastStack, type ToastItem } from "./components/Toasts";
import { fmtUSD } from "./format";

let nextToast = 1;

function Clock() {
  const [now, setNow] = useState(() => new Date());
  useEffect(() => {
    const id = window.setInterval(() => setNow(new Date()), 1000);
    return () => window.clearInterval(id);
  }, []);
  return <span className="clock mono">{now.toUTCString().slice(17, 25)}z</span>;
}

export function App() {
  const { prices, portfolio, history, activity, samples, lastUpdated, error, refresh } = useStockData();
  const [now, setNow] = useState(() => Date.now());
  const [toasts, setToasts] = useState<ToastItem[]>([]);
  const [tradeTarget, setTradeTarget] = useState("");

  useEffect(() => {
    const id = window.setInterval(() => setNow(Date.now()), 1000);
    return () => window.clearInterval(id);
  }, []);

  const onToast = useCallback((msg: string, kind = "paper") => {
    const id = nextToast++;
    setToasts((prev) => [...prev.slice(-2), { id, msg, kind }]);
  }, []);

  const dismiss = useCallback((id: number) => {
    setToasts((prev) => prev.filter((t) => t.id !== id));
  }, []);

  const openTrade = useCallback((symbol: string) => {
    setTradeTarget(symbol || "");
    document.getElementById("trade")?.scrollIntoView({ behavior: "smooth", block: "center" });
  }, []);

  const pricesData = prices;
  const net = pricesData?.network ?? "devnet";
  const paper = pricesData?.paper ?? true;
  const statusClass = pricesData?.symbols.some((s) => s.loaded) ? "ok" : "down";

  return (
    <div className="app">
      <header className="topbar">
        <div className="brand">
          <span className="logo">
            <svg viewBox="0 0 32 32" aria-hidden="true">
              <rect width="32" height="32" rx="7" fill="var(--panel2)" />
              <path d="M7 22 L13 13 L17 17 L25 8" stroke="var(--accent)" strokeWidth="3" fill="none" strokeLinecap="round" strokeLinejoin="round" />
            </svg>
          </span>
          <span className="brand-name">
            STOCK<span className="brand-dot">.</span>sh
          </span>
          <span className="brand-sub mono muted">TOKENIZED STOCKS ON SOLANA</span>
        </div>
        <div className="topchips">
          <span className="chip-clk">
            <Clock />
          </span>
          <span className="chip">SOL {pricesData ? fmtUSD(pricesData.sol, 2) : "…"}</span>
          <span className="chip">{net.toUpperCase()}</span>
          <span className={`chip ${paper ? "paper" : "live"}`}>{paper ? "PAPER" : "LIVE"}</span>
        </div>
      </header>

      <TickerTape items={pricesData?.symbols ?? []} />

      <main className="grid">
        <div className="col-left">
          <MarketTable items={pricesData?.symbols ?? []} samples={samples} onTrade={openTrade} />
          <Feed activity={activity} history={history} />
        </div>
        <div className="col-side">
          <Portfolio portfolio={portfolio} />
          <div id="trade">
            <TradePanel
              prices={pricesData?.symbols ?? []}
              portfolio={portfolio}
              initialSymbol={tradeTarget}
              onToast={onToast}
              onExecuted={refresh}
            />
          </div>
        </div>
      </main>

      <footer className="botbar mono">
        <span>
          <i className={`u-dot ${statusClass}`} /> jupiter · {net} · 5s refresh
        </span>
        <span className="muted">
          {statusClass === "down" && error ? `api: ${error}` : updatedAgo(lastUpdated, now)}
        </span>
      </footer>

      <ToastStack toasts={toasts} onDismiss={dismiss} />
    </div>
  );
}