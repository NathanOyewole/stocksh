interface Props {
  data: number[];
}

export function Sparkline({ data }: Props) {
  if (data.length < 2) {
    return <span className="spark-empty">·</span>;
  }
  const min = Math.min(...data);
  const max = Math.max(...data);
  const span = max - min || 1;
  const step = data.length > 1 ? 100 / (data.length - 1) : 100;
  const pts = data.map((v, i) => {
    const x = (i * step).toFixed(2);
    const y = (100 - ((v - min) / span) * 100).toFixed(2);
    return `${x},${y}`;
  });
  const up = data[data.length - 1] >= data[0];
  const area = `0,100 ${pts.join(" ")} 100,100`;

  return (
    <svg
      viewBox="0 0 100 30"
      preserveAspectRatio="none"
      className={`spark ${up ? "up" : "down"}`}
      aria-hidden="true"
    >
      <polygon points={area} className="spark-area" />
      <polyline points={pts.join(" ")} className="spark-line" fill="none" />
    </svg>
  );
}