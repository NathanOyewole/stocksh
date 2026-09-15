import type { SymbolQuote } from "../types";
import { clsFor, fmtCompact, fmtDelta, fmtUSD } from "../format";

interface Props {
  items: SymbolQuote[];
}

export function TickerTape({ items }: Props) {
  if (items.length === 0) {
    return <div className="tape tape-off">warmup — fetching live prices…</div>;
  }

  const track = [...items, ...items];

  return (
    <div className="tape">
      <div className="tape-track">
        {track.map((s, i) => (
          <span className="tape-item" key={`${s.symbol}-${i}`}>
            <span className="tape-sym">{s.symbol}</span>
            <span className="tape-px">{s.loaded ? fmtUSD(s.mark || s.price, 2) : "—"}</span>
            <span className={`tape-chg ${clsFor(s.change24h)}`}>
              {s.change24h >= 0 ? "▲" : "▼"} {fmtDelta(Math.abs(s.change24h), 2)}
            </span>
            <span className="tape-liq">{fmtCompact(s.liquidity)}</span>
          </span>
        ))}
      </div>
    </div>
  );
}