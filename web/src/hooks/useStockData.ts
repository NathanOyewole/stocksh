import { useCallback, useEffect, useState } from "react";
import { api } from "../api";
import type { Activity, PortfolioResponse, PricesResponse, Trade } from "../types";

const REFRESH_MS = 5000;
const MAX_SAMPLES = 48;

export interface StockFeed {
  prices: PricesResponse | null;
  portfolio: PortfolioResponse | null;
  history: Trade[];
  activity: Activity[];
  samples: Record<string, number[]>;
  lastUpdated: number;
  error: string | null;
  refresh: () => void;
}

export function useStockData(): StockFeed {
  const [prices, setPrices] = useState<PricesResponse | null>(null);
  const [portfolio, setPortfolio] = useState<PortfolioResponse | null>(null);
  const [history, setHistory] = useState<Trade[]>([]);
  const [activity, setActivity] = useState<Activity[]>([]);
  const [samples, setSamples] = useState<Record<string, number[]>>({});
  const [lastUpdated, setLastUpdated] = useState(0);
  const [error, setError] = useState<string | null>(null);

  const tick = useCallback(async () => {
    const [pr, pf, h, a] = await Promise.allSettled([
      api.prices(),
      api.portfolio(),
      api.history(),
      api.activity(),
    ]);

    if (pr.status === "fulfilled") {
      setPrices(pr.value);
      setSamples((prev) => {
        const next: Record<string, number[]> = {};
        for (const s of pr.value.symbols) {
          const ring = prev[s.symbol] ?? [];
          next[s.symbol] = [...ring.slice(-(MAX_SAMPLES - 1)), s.loaded ? s.mark || s.price : 0];
        }
        return next;
      });
    } else setError(String((pr as PromiseRejectedResult).reason));

    if (pf.status === "fulfilled") setPortfolio(pf.value);
    if (h.status === "fulfilled") setHistory(h.value);
    if (a.status === "fulfilled") setActivity(a.value);

    setLastUpdated(Date.now());
  }, []);

  useEffect(() => {
    tick();
    const id = window.setInterval(tick, REFRESH_MS);
    return () => window.clearInterval(id);
  }, [tick]);

  const refresh = useCallback(() => {
    void tick();
  }, [tick]);

  return { prices, portfolio, history, activity, samples, lastUpdated, error, refresh };
}

export function updatedAgo(lastUpdated: number, now: number): string {
  if (!lastUpdated) return "connecting…";
  const s = Math.max(0, Math.round((now - lastUpdated) / 1000));
  if (s < 5) return "live now";
  return `${s}s ago`;
}