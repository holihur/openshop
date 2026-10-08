import type { DailyPoint } from "@lib/types";

/**
 * Dependency-free SVG charts. Recharts et al. would add a large bundle for two
 * small graphs, so these draw the path directly and stay accessible by exposing
 * the underlying numbers as text.
 */

function path(points: DailyPoint[], width: number, height: number, pad: number) {
  const max = Math.max(1, ...points.map((p) => p.value));
  const step = points.length > 1 ? (width - pad * 2) / (points.length - 1) : 0;
  const xy = points.map((p, i) => {
    const x = pad + i * step;
    const y = height - pad - (p.value / max) * (height - pad * 2);
    return [x, y] as const;
  });
  const line = xy.map(([x, y], i) => `${i === 0 ? "M" : "L"}${x.toFixed(1)},${y.toFixed(1)}`).join(" ");
  const area = `${line} L${(pad + (points.length - 1) * step).toFixed(1)},${height - pad} L${pad},${height - pad} Z`;
  return { line, area, max, xy };
}

/** A compact trend line for a module header. */
export function Sparkline({
  points,
  className,
  ariaLabel,
}: {
  points: DailyPoint[];
  className?: string;
  ariaLabel?: string;
}) {
  if (points.length < 2) return null;
  const width = 160;
  const height = 36;
  const { line, area } = path(points, width, height, 2);
  return (
    <svg
      viewBox={`0 0 ${width} ${height}`}
      className={className ?? "h-9 w-40"}
      role="img"
      aria-label={ariaLabel}
      preserveAspectRatio="none"
    >
      <path d={area} className="fill-primary/10" />
      <path d={line} className="stroke-primary" fill="none" strokeWidth="1.5" vectorEffect="non-scaling-stroke" />
    </svg>
  );
}

/** A labelled trend chart with axis hints, used on the dashboard. */
export function LineChart({
  points,
  format,
  ariaLabel,
}: {
  points: DailyPoint[];
  format?: (n: number) => string;
  ariaLabel?: string;
}) {
  if (points.length < 2) return null;
  const width = 720;
  const height = 180;
  const pad = 8;
  const { line, area, max } = path(points, width, height, pad);
  const fmt = format ?? ((n: number) => String(n));
  const first = points[0];
  const last = points[points.length - 1];

  return (
    <div className="space-y-1">
      <svg
        viewBox={`0 0 ${width} ${height}`}
        className="h-44 w-full"
        role="img"
        aria-label={ariaLabel}
        preserveAspectRatio="none"
      >
        {[0.25, 0.5, 0.75].map((f) => (
          <line
            key={f}
            x1={0}
            x2={width}
            y1={height * f}
            y2={height * f}
            className="stroke-border"
            strokeWidth="1"
            vectorEffect="non-scaling-stroke"
          />
        ))}
        <path d={area} className="fill-primary/10" />
        <path d={line} className="stroke-primary" fill="none" strokeWidth="2" vectorEffect="non-scaling-stroke" />
      </svg>
      <div className="text-muted-foreground flex justify-between text-xs">
        <span>{first.date}</span>
        <span>{fmt(max)}</span>
        <span>{last.date}</span>
      </div>
    </div>
  );
}
