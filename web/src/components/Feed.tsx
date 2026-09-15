import { useState } from "react";
import type { Activity, Trade } from "../types";
import { fmtTime, fmtUSD, typeColor } from "../format";

function ActivityRow({ a }: { a: Activity }) {
  return (
    <div className="feed-row">
      <i className="feed-dot" style={{ background: typeColor(a.type) }} />
      <span className="feed-text">
        <b className="feed-type">{a.type}</b> {a.text}
      </span>
      <span className="feed-time mono muted">{fmtTime(a.time)}</span>
    </div>
  );
}

function TradeRow({ t }: { t: Trade }) {
  const buy = t.side === "buy";
  return (
    <div className="feed-row">
      <i className="feed-dot" style={{ background: buy ? "var(--accent)" : "var(--danger)" }} />
      <span className="feed-text">
        <b className={`side ${buy ? "pos" : "neg"}`}>{buy ? "BUY" : "SELL"}</b>
        <span className="mono">
          {buy ? "+" : "-"}
          {Math.abs(t.qty).toLocaleString("en-US", { maximumFractionDigits: 4 })} {t.symbol}
        </span>
        <span className="muted"> @ {fmtUSD(t.price, 2)}</span>
      </span>
      <span className="feed-time mono muted">{fmtTime(t.time)}</span>
    </div>
  );
}

export function Feed({ activity, history }: { activity: Activity[]; history: Trade[] }) {
  const [tab, setTab] = useState<"activity" | "history">("activity");

  return (
    <section className="card feed" aria-label="Ledger feed">
      <header className="card-head">
        <h2>LEDGER</h2>
        <div className="tabs">
          <button className={tab === "activity" ? "tab on" : "tab"} onClick={() => setTab("activity")}>
            ACTIVITY
          </button>
          <button className={tab === "history" ? "tab on" : "tab"} onClick={() => setTab("history")}>
            HISTORY
          </button>
        </div>
      </header>
      <div className="feed-list">
        {tab === "activity"
          ? activity.length === 0
            ? <div className="empty">nothing on the tape yet — make your first paper trade</div>
            : [...activity].reverse().map((a, i) => <ActivityRow key={`${a.time}-${i}`} a={a} />)
          : history.length === 0
            ? <div className="empty">no trades on record</div>
            : [...history].reverse().map((t, i) => <TradeRow key={`${t.time}-${i}`} t={t} />)}
      </div>
    </section>
  );
}