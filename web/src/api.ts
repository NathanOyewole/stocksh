import type {
  Activity,
  ExecuteResponse,
  PortfolioResponse,
  PricesResponse,
  QuoteRequest,
  QuoteResponse,
  Trade,
} from "./types";

async function read<T>(res: Response): Promise<T> {
  const body = (await res.json().catch(() => ({}))) as { error?: string } & T;
  if (!res.ok) throw new Error(body?.error ?? `HTTP ${res.status}`);
  return body;
}

async function post<T>(url: string, body: unknown): Promise<T> {
  const res = await fetch(url, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(body),
  });
  return read<T>(res);
}

export const api = {
  prices: () => fetch("/api/prices").then((r) => read<PricesResponse>(r)),
  portfolio: () => fetch("/api/portfolio").then((r) => read<PortfolioResponse>(r)),
  history: () => fetch("/api/history").then((r) => read<Trade[]>(r)),
  activity: () => fetch("/api/activity").then((r) => read<Activity[]>(r)),
  quote: (q: QuoteRequest) => post<QuoteResponse>("/api/quote", q),
  execute: (q: QuoteRequest) => post<ExecuteResponse>("/api/execute", q),
};