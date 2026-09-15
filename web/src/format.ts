export function fmtUSD(n: number, dp = 2): string {
  if (!isFinite(n)) return "—";
  return n.toLocaleString("en-US", {
    style: "currency",
    currency: "USD",
    minimumFractionDigits: dp,
    maximumFractionDigits: dp,
  });
}

export function fmtNum(n: number, dp = 4): string {
  if (!isFinite(n)) return "—";
  return n.toLocaleString("en-US", {
    minimumFractionDigits: dp,
    maximumFractionDigits: dp,
  });
}

export function fmtDelta(n: number, dp = 2): string {
  if (!isFinite(n)) return "—";
  const s = Math.abs(n).toLocaleString("en-US", {
    minimumFractionDigits: dp,
    maximumFractionDigits: dp,
  });
  return n >= 0 ? `+${s}` : `-${s}`;
}

export function fmtPct(n: number, dp = 2): string {
  if (!isFinite(n)) return "—";
  const s = Math.abs(n).toLocaleString("en-US", {
    minimumFractionDigits: dp,
    maximumFractionDigits: dp,
  });
  return `${n >= 0 ? "+" : "-"}${s}%`;
}

export function fmtCompact(n: number): string {
  if (!isFinite(n) || n <= 0) return "—";
  const abs = Math.abs(n);
  if (abs >= 1e9) return `${(n / 1e9).toFixed(1)}B`;
  if (abs >= 1e6) return `${(n / 1e6).toFixed(2)}M`;
  if (abs >= 1e3) return `${(n / 1e3).toFixed(1)}K`;
  return n.toFixed(2);
}

export function clsFor(n: number): string {
  if (!isFinite(n) || n === 0) return "flat";
  return n > 0 ? "pos" : "neg";
}

export function fmtTime(iso: string): string {
  const d = new Date(iso);
  if (isNaN(d.getTime())) return "—";
  return d.toLocaleTimeString("en-US", {
    hour12: false,
    hour: "2-digit",
    minute: "2-digit",
    second: "2-digit",
  });
}

export function typeColor(type: string): string {
  switch (type) {
    case "trade":
      return "var(--accent)";
    case "airdrop":
      return "var(--sol)";
    case "alert":
      return "var(--warn)";
    case "watch":
      return "var(--info)";
    default:
      return "#7e8ba0";
  }
}