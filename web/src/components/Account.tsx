import type { PortfolioResponse } from "../types";
import { clsFor, fmtNum, fmtPct, fmtUSD } from "../format";

function Weights({ portfolio }: { portfolio: PortfolioResponse }) {
  const solUsd = portfolio.solBalance * portfolio.solPrice;
  const total = portfolio.totalValue + solUsd;
  const solW = total > 0 ? (solUsd / total) * 100 : 0;

  return (
    <div className="weights">
      <div
        className="weight-bar"
        style={{
          background: `linear-gradient(90deg, var(--sol) 0 ${solW.toFixed(2)}%, var(--accent) ${solW.toFixed(2)}%)`,
        }}
      />
      <div className="weight-legend mono muted">
        <span>
          <i className="chip-sq sol" /> SOL {fmtNum(solW, 0)}% · $
          {fmtUSD(solUsd, 0)}
        </span>
        <span>
          <i className="chip-sq asset" /> POSITIONS {fmtNum(100 - solW, 0)}%
        </span>
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
          {p.hasPnL && (
            <span className={`pos-cost mono muted`}>
              avg {fmtUSD(p.avgCost, 2)}
            </span>
          )}
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

export function Account({ portfolio }: { portfolio: PortfolioResponse | null }) {
  if (!portfolio) {
    return (
      <section className="card col account">
        <header className="card-head">
          <h2>MY ACCOUNT</h2>
        </header>
        <div className="empty">connecting — pulling your paper balance…</div>
      </section>
    );
  }

  const { paper, mode, network, address, tradeCount } = portfolio;
  const solUsd = portfolio.solBalance * portfolio.solPrice;
  const equity = portfolio.totalValue + solUsd;
  const cls = clsFor(portfolio.totalPnL);
  const pos = [...portfolio.positions].sort((a, b) => b.value - a.value);
  const id = address ? `${address.slice(0, 4)}…${address.slice(-8)}` : "PAPER-ACCOUNT";
  const live = !paper;

  return (
    <section className={`card col account ${live ? "live" : ""}`} aria-label="My account">
      <header className="card-head">
        <h2>MY ACCOUNT</h2>
        <span className={`badge ${paper ? "paper" : "live"}`}>
          {mode.toUpperCase()} · {network.toUpperCase()}
        </span>
      </header>

      <div className="identity">
        <span className="identity-avatar">
          <svg viewBox="0 0 32 32" aria-hidden="true">
            <rect width="32" height="32" rx="9" fill="var(--panel2)" stroke="var(--line-2)" />
            <path d="M7 22 L13 13 L17 17 L25 8" stroke="var(--accent)" strokeWidth="2.6" fill="none" strokeLinecap="round" strokeLinejoin="round" />
          </svg>
        </span>
        <div className="identity-meta">
          <span className="identity-name mono">{id}</span>
          <span className="identity-sub mono muted">
            {live && address
              ? `SOLANA WALLET · ${network} · LIVE MODE`
              : address
                ? `PAPER ACCOUNT · ${network} DRY-RUN · WALLET ${id} · ${tradeCount} TRADE${tradeCount === 1 ? "" : "S"}`
                : `PAPER ACCOUNT · ${network} DRY-RUN · NO WALLET · ${tradeCount} TRADE${tradeCount === 1 ? "" : "S"}`}
          </span>
        </div>
      </div>

      <div className="kpi-row">
        <div className="kpi">
          <span className="kpi-label mono muted">TOTAL EQUITY</span>
          <span className="kpi-value mono">{fmtUSD(equity, 2)}</span>
        </div>
        <div className="kpi">
          <span className="kpi-label mono muted">REALIZED P&L</span>
          <span className={`kpi-value mono ${cls}`}>
            {portfolio.totalPnL >= 0 ? "+" : ""}
            {fmtUSD(portfolio.totalPnL, 2)}
          </span>
        </div>
      </div>

      <div className="balances">
        <div className="balance">
          <span className="balance-label mono muted">CASH (SOL)</span>
          <span className="balance-value mono sol">
            {fmtNum(portfolio.solBalance, 2)} SOL
            <span className="muted"> ≈ {fmtUSD(solUsd, 0)}</span>
          </span>
        </div>
        <div className="balance">
          <span className="balance-label mono muted">POSITIONS</span>
          <span className="balance-value mono">
            {pos.length} HOLDING
            <span className="muted"> ≈ {fmtUSD(portfolio.totalValue, 0)}</span>
          </span>
        </div>
      </div>

      <Weights portfolio={portfolio} />

      <div className="pos-list">
        {pos.length === 0 ? (
          <div className="empty">
            no positions yet — go to the trade desk and start building your book
          </div>
        ) : (
          pos.map((p) => <PositionRow key={p.symbol} p={p} />)
        )}
      </div>

      <footer className="card-foot mono muted">
        <span>SOL {fmtUSD(portfolio.solPrice, 2)}</span>
        <span>{paper ? "paper · no wallet touched" : "linked wallet"}</span>
      </footer>
    </section>
  );
}

export default Account;