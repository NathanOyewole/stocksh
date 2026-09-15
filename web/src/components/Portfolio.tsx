import type { PortfolioResponse } from "../types";
import { clsFor, fmtNum, fmtPct, fmtUSD } from "../format";

function Weights({ portfolio }: { portfolio: PortfolioResponse }) {
  const solUsd = portfolio.solBalance * portfolio.solPrice;
  const total = portfolio.totalValue + solUsd;
  const solW = total > 0 ? (solUsd / total) * 100 : 0;
  const segments = portfolio.positions.map((p) => ({
    symbol: p.symbol,
    w: p.allocPct,
  }));

  return (
    <div className="weights">
      <div
        className="weight-bar"
        style={{
          background: `linear-gradient(90deg, var(--sol) 0 ${solW.toFixed(2)}%, transparent ${solW.toFixed(2)}%)`,
        }}
      />
      <div className="weight-legend mono muted">
        <span>
          <i className="chip sol" /> SOL {fmtNum(solW, 0)}%
        </span>
        {segments.map((seg) => (
          <span key={seg.symbol}>
            <i className="chip asset" /> {seg.symbol} {fmtNum(seg.w, 0)}%
          </span>
        ))}
      </div>
    </div>
  );
}

function PositionRow({ p }: { p: PortfolioResponse["positions"][number] }) {
  const cls = clsFor(p.pnl);
  return (
    <div className="pos-row">
      <div className="pos-main">
        <div className="pos-name">
          <span className="sym-x">{p.symbol.slice(0, -1)}</span>
          <span className="sym-suffix">x</span>
          <span className="pos-qty mono muted">· {fmtNum(p.qty, 2)}</span>
        </div>
        <div className="pos-bar">
          <span style={{ width: `${Math.min(100, p.allocPct)}%` }} />
        </div>
      </div>
      <div className="pos-nums mono right">
        <div>{p.hasPx ? fmtUSD(p.value, 2) : "—"}</div>
        <div className={`pnl ${cls}`}>
          {p.hasPnL ? `${fmtUSD(p.pnl, 2)} (${fmtPct(p.pnlPct, 2)})` : "—"}
        </div>
      </div>
    </div>
  );
}

export function Portfolio({
  portfolio,
}: {
  portfolio: PortfolioResponse | null;
}) {
  if (!portfolio) {
    return (
      <section className="card col">
        <header className="card-head">
          <h2>PORTFOLIO</h2>
        </header>
        <div className="empty">no data yet</div>
      </section>
    );
  }

  const solUsd = portfolio.solBalance * portfolio.solPrice;
  const total = portfolio.totalValue + solUsd;
  const cls = clsFor(portfolio.totalPnL);
  const pos = [...portfolio.positions].sort((a, b) => b.value - a.value);

  return (
    <section className="card col" aria-label="Portfolio">
      <header className="card-head">
        <h2>PORTFOLIO</h2>
        <span className={`badge ${portfolio.paper ? "paper" : "live"}`}>
          {portfolio.paper ? "PAPER" : "LIVE"} {portfolio.paper ? "DRY-RUN" : "MAINNET"}
        </span>
      </header>

      <div className="kpi-row">
        <div className="kpi">
          <span className="kpi-label mono muted">TOTAL VALUE</span>
          <span className="kpi-value mono">{fmtUSD(total, 2)}</span>
        </div>
        <div className="kpi">
          <span className="kpi-label mono muted">REALIZED P&L</span>
          <span className={`kpi-value mono ${cls}`}>
            {portfolio.totalPnL >= 0 ? "+" : ""}
            {fmtUSD(portfolio.totalPnL, 2)}
          </span>
        </div>
      </div>

      <div className="kpi-row sub">
        <div className="kpi inline">
          <span className="kpi-label mono muted">SOL</span>
          <span className="kpi-value mono sol">
            {fmtNum(portfolio.solBalance, 2)}
            <span className="muted"> ≈ {fmtUSD(solUsd, 0)}</span>
          </span>
        </div>
        <div className="kpi inline">
          <span className="kpi-label mono muted">ALLOCATION</span>
          <Weights portfolio={portfolio} />
        </div>
      </div>

      <div className="pos-list">
        {pos.length === 0 ? (
          <div className="empty">
            no positions yet — open the trade desk and start building your book
          </div>
        ) : (
          pos.map((p) => <PositionRow key={p.symbol} p={p} />)
        )}
      </div>
    </section>
  );
}