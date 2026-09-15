import { useMemo, useState } from "react";
import type { SymbolQuote } from "../types";
import { clsFor, fmtCompact, fmtDelta, fmtPct, fmtUSD } from "../format";
import { useFlash } from "../hooks/useFlash";
import { Sparkline } from "./Sparkline";

type SortKey = "order" | "price" | "change";

function Row({ s, samples }: { s: SymbolQuote; samples: number[] }) {
  const dir = useFlash(s.mark || s.price);
  const px = s.loaded ? s.mark || s.price : 0;

  return (
    <tr>
      <td className="cell-sym">
        <span className="sym-x">{s.symbol.slice(0, -1)}</span>
        <span className="sym-suffix">x</span>
      </td>
      <td className="cell-spark">
        <Sparkline data={samples} />
      </td>
      <td className={`cell-px mono ${dir ? `flash-${dir}` : ""}`} data-flash={dir ?? ""}>
        {s.loaded ? fmtUSD(px, 2) : "—"}
      </td>
      <td className="cell-chg">
        <span className={`pill ${clsFor(s.change24h)}`}>
          {fmtDelta(s.change24h, 2)}
          <em>{fmtPct(s.change24h / 100, 2)}</em>
        </span>
      </td>
      <td className="cell-liq mono muted">{fmtCompact(s.liquidity)}</td>
    </tr>
  );
}

export function MarketTable({
  items,
  samples,
  onTrade,
}: {
  items: SymbolQuote[];
  samples: Record<string, number[]>;
  onTrade: (symbol: string) => void;
}) {
  const [sort, setSort] = useState<{ key: SortKey; dir: 1 | -1 }>({
    key: "order",
    dir: 1,
  });

  const rows = useMemo(() => {
    if (sort.key === "order") return items;
    return [...items].sort((a, b) => {
      const va = sort.key === "price" ? a.mark || a.price : a.change24h;
      const vb = sort.key === "price" ? b.mark || b.price : b.change24h;
      return (va - vb) * sort.dir;
    });
  }, [items, sort]);

  const setSortKey = (key: SortKey) =>
    setSort((prev) =>
      prev.key === key ? { key, dir: (prev.dir * -1) as 1 | -1 } : { key, dir: 1 },
    );

  const arrow = (key: SortKey) =>
    sort.key === key ? (sort.dir === 1 ? "▲" : "▼") : "";

  return (
    <section className="card board" aria-label="Market board">
      <header className="card-head">
        <h2>
          <span className="dot-live" /> MARKET BOARD
        </h2>
        <span className="mid muted mono">NASDAQ × SOLANA</span>
      </header>
      <div className="table-wrap">
        <table>
          <thead>
            <tr>
              <th className="sort" onClick={() => setSortKey("order")}>
                TICKER
              </th>
              <th className="right muted">60s</th>
              <th className="right sort" onClick={() => setSortKey("price")}>
                PRICE {arrow("price")}
              </th>
              <th className="right sort" onClick={() => setSortKey("change")}>
                24H {arrow("change")}
              </th>
              <th className="right muted">LIQ</th>
            </tr>
          </thead>
          <tbody>
            {rows.map((s) => (
              <Row key={s.symbol} s={s} samples={samples[s.symbol] ?? []} />
            ))}
          </tbody>
        </table>
      </div>
      <footer className="card-foot mono muted">
        <span>live×8 · jupiter aggregator</span>
        <button className="link-btn" onClick={() => onTrade("")}>
          open trade desk →
        </button>
      </footer>
    </section>
  );
}