import { cn } from '@/lib/utils';

const WARN = 0.8;
const CRITICAL = 0.95;

/** Used vs limit, with marks at 80% and 95%; colour and text both carry severity. */
export function UsageBar({
  label,
  value,
  max,
  format = String,
}: {
  label: string;
  value: number;
  max: number;
  format?: (n: number) => string;
}) {
  const ratio = max > 0 ? Math.min(value / max, 1) : 0;
  const level =
    ratio >= CRITICAL ? 'critical' : ratio >= WARN ? 'warning' : 'ok';
  const fill = {
    ok: 'bg-primary',
    warning: 'bg-warning',
    critical: 'bg-destructive',
  }[level];
  return (
    <div>
      <div className="flex justify-between gap-2 text-xs">
        <span className="text-muted-foreground">{label}</span>
        <span className="tabular-nums">
          {format(value)} of {format(max)} · {Math.round(ratio * 100)}%
          {level !== 'ok' && ` · ${level === 'critical' ? 'critical' : 'high'}`}
        </span>
      </div>
      <meter
        className="sr-only"
        aria-label={label}
        min={0}
        max={max}
        value={value}
      />
      <div
        aria-hidden="true"
        className="relative mt-1 h-2 rounded-full bg-muted"
      >
        <div
          className={cn('h-full rounded-full', fill)}
          style={{ width: `${ratio * 100}%` }}
        />
        {[WARN, CRITICAL].map((m) => (
          <span
            key={m}
            aria-hidden="true"
            className="absolute -top-0.5 h-3 w-px bg-foreground/40"
            style={{ left: `${m * 100}%` }}
          />
        ))}
      </div>
    </div>
  );
}
