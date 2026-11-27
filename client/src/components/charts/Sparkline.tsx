/** Trend line for a stat tile; decorative, the tile states the value. */
export function Sparkline({ values }: { values: (number | null)[] }) {
  const points = values.flatMap((v, i) => (v == null ? [] : [[i, v] as const]));
  if (points.length < 2) return null;
  const ys = points.map(([, v]) => v);
  const min = Math.min(...ys);
  const span = Math.max(...ys) - min || 1;
  const last = values.length - 1 || 1;
  const xy = ([i, v]: readonly [number, number]) =>
    `${(i / last) * 100},${28 - ((v - min) / span) * 24}`;
  const [lx, ly] = xy(points[points.length - 1]).split(',');
  return (
    <svg
      viewBox="0 0 100 32"
      preserveAspectRatio="none"
      className="h-8 w-full overflow-visible"
      aria-hidden="true"
    >
      <polyline
        points={points.map(xy).join(' ')}
        fill="none"
        strokeWidth={2}
        strokeLinejoin="round"
        strokeLinecap="round"
        vectorEffect="non-scaling-stroke"
        className="stroke-muted-foreground/50"
      />
      <circle cx={lx} cy={ly} r={3} className="fill-primary" />
    </svg>
  );
}
