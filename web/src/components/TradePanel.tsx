import { useEffect, useMemo, useRef, useState } from "react";
import { api } from "../api";
import type { ExecuteResponse, PortfolioResponse, QuoteResponse, SymbolQuote } from "../types";
import { clsFor, fmtNum, fmtPct, fmtUSD } from "../format";

interface Props {
  prices: SymbolQuote[];
  portfolio: PortfolioResponse | null;
  initialSymbol?: string;
  live: boolean;
  network: string;
  onToast: (msg: string, kind?: string) => void;
  onExecuted: () => void;
}

export function TradePanel({ prices, portfolio, initialSymbol, live, network, onToast, onExecuted }: Props) {
  const loaded = useMemo(() => prices.filter((p) => p.loaded), [prices]);
  const [symbol, setSymbol] = useState(initialSymbol && loaded.some((p) => p.symbol === initialSymbol)
      ? initialSymbol
      : loaded[0]?.symbol ?? "");
  const [side, setSide] = useState<"buy" | "sell">("buy");
  const [amount, setAmount] = useState("100");
  const [preview, setPreview] = useState<QuoteResponse | null>(null);
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const request = useRef(0);

  useEffect(() => {
    if (initialSymbol && loaded.some((p) => p.symbol === initialSymbol)) {
      setSymbol(initialSymbol);
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [initialSymbol]);

  const sym = loaded.find((s) => s.symbol === symbol);
  const holding = portfolio?.positions.find((p) => p.symbol === symbol);
  const usdc = Math.floor(Number(amount));
  const valid = usdc >= 1 && !!symbol && !!sym?.loaded;

  useEffect(() => {
    if (!valid) {
      setPreview(null);
      return;
    }
    const id = ++request.current;
    const t = window.setTimeout(() => {
      api
        .quote({ symbol, side, usdc })
        .then((q) => {
          if (request.current === id) {
            setPreview(q);
            setError(null);
          }
        })
        .catch((e: Error) => {
          if (request.current === id) {
            setPreview(null);
            setError(e.message);
          }
        });
    }, 700);
    return () => window.clearTimeout(t);
  }, [symbol, side, usdc, valid]);

  const inTok = preview?.quote?.inAmount ? Number(preview.quote.inAmount) : 0;
  const outTok = preview?.quote?.outAmount ? Number(preview.quote.outAmount) : 0;
  const qty = outTok / 1e6;
  const execPrice = inTok && outTok ? inTok / outTok : 0;
  const solCost = preview && preview.solPrice > 0 ? usdc / preview.solPrice : 0;

  const doExecute = async () => {
    if (!valid || busy) return;
    setBusy(true);
    try {
      const res: ExecuteResponse = await api.execute({ symbol, side, usdc });
      onToast(res.message || `${side === "buy" ? "Bought" : "Sold"} ${symbol}`, res.status);
      setError(null);
      onExecuted();
    } catch (e) {
      const msg = e instanceof Error ? e.message : "execute failed";
      onToast(msg, "error");
      setError(msg);
    } finally {
      setBusy(false);
    }
  };

  const chg = sym ? clsFor(sym.change24h) : "flat";

  return (
    <section className="card trade" aria-label="Trade desk">
      <header className="card-head">
        <h2>TRADE DESK</h2>
        <span className={`mid mono ${live ? "live-tag" : "muted"}`}>
          {live ? "LIVE · MAINNET" : "paper · devnet"}
        </span>
      </header>

      <div className="trade-sym">
        <span className="big-sym">
          {symbol ? (
            <>
              {symbol.slice(0, -1)}
              <i className="sym-suffix x2">x</i>
            </>
          ) : (
            "—"
          )}
        </span>
        {sym && (
          <span className="big-px mono">
            {fmtUSD(sym.mark || sym.price, 2)}{" "}
            <i className={`chg ${chg}`}>{sym.change24h >= 0 ? "▲" : "▼"} {fmtPct(sym.change24h / 100, 2)}</i>
          </span>
        )}
        {holding && holding.qty > 0 && (
          <span className="hold mono muted">you hold {fmtNum(holding.qty, 2)}</span>
        )}
      </div>

      <div className="seg">
        <button className={side === "buy" ? "seg-btn on buy" : "seg-btn"} onClick={() => setSide("buy")}>
          BUY
        </button>
        <button className={side === "sell" ? "seg-btn on sell" : "seg-btn"} onClick={() => setSide("sell")}>
          SELL
        </button>
      </div>

      <label className="field">
        <span className="field-label mono muted">TICKER</span>
        <select value={symbol} onChange={(e) => setSymbol(e.target.value)}>
          {loaded.map((s) => (
            <option key={s.symbol} value={s.symbol}>
              {s.symbol} — {fmtUSD(s.mark || s.price, 2)}
            </option>
          ))}
        </select>
      </label>

      <label className="field">
        <span className="field-label mono muted">SIZE (USDC)</span>
        <div className="amount-wrap">
          <span className="amount-prefix mono">$</span>
          <input
            inputMode="decimal"
            value={amount}
            onChange={(e) => setAmount(e.target.value.replace(/[^\d.]/g, ""))}
          />
        </div>
      </label>

      <div className={`preview ${error ? "has-error" : ""}`} aria-live="polite">
        {error ? (
          <span className="err-msg">{error}</span>
        ) : preview && valid ? (
          <div className="preview-grid mono">
            <span className="muted">est. qty</span>
            <span>{fmtNum(qty, 3)} {symbol}</span>
            <span className="muted">exec price</span>
            <span>{fmtUSD(execPrice, 2)}</span>
            <span className="muted">SOL impact</span>
            <span>{fmtNum(solCost, 4)} SOL</span>
          </div>
        ) : (
          <span className="muted">enter a size to see the Jupiter quote…</span>
        )}
      </div>

      <button className="exec" onClick={doExecute} disabled={!valid || busy}>
        {busy
          ? "WORKING…"
          : `${side === "buy" ? "BUY" : "SELL"} ${symbol || ""} · ${fmtUSD(usdc, 2)}`}
      </button>

      <footer className="card-foot mono muted">
        <span>
          {preview
            ? `slippage ${preview.quote.slippageBps / 100}%`
            : live
              ? `live · real ${network.toUpperCase()} swap`
              : "dry-run — no wallet touched"}
        </span>
        <span>
          {preview
            ? `impact ${fmtPct(Number(preview.quote.priceImpactPct) * 100, 3)}`
            : live
              ? "jupiter · mainnet"
              : "jupiter · devnet quotes"}
        </span>
      </footer>
    </section>
  );
}