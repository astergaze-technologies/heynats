import { useSearchParams } from 'react-router-dom';
import { cn } from '@/lib/utils';

export const RANGES = ['5m', '15m', '1h', '6h', '24h'] as const;
export type Range = (typeof RANGES)[number];

/** The page's time range, kept in `?range=` so refresh and shared links keep it. */
export function useTimeRange(fallback: Range = '1h') {
  const [params, setParams] = useSearchParams();
  const raw = params.get('range');
  const range = RANGES.includes(raw as Range) ? (raw as Range) : fallback;
  const setRange = (next: Range) =>
    setParams(
      (prev) => {
        const p = new URLSearchParams(prev);
        p.set('range', next);
        return p;
      },
      { replace: true }
    );
  return [range, setRange] as const;
}

/** One row of presets above the charts it scopes. */
export function TimeRangePicker({
  value,
  onChange,
}: {
  value: Range;
  onChange: (r: Range) => void;
}) {
  return (
    <fieldset className="inline-flex rounded-md border bg-card p-0.5">
      <legend className="sr-only">Time range</legend>
      {RANGES.map((r) => (
        <button
          key={r}
          type="button"
          aria-pressed={r === value}
          onClick={() => onChange(r)}
          className={cn(
            'rounded px-2.5 py-1 text-xs font-medium tabular-nums text-muted-foreground hover:bg-muted',
            r === value && 'bg-primary text-primary-foreground hover:bg-primary'
          )}
        >
          {r}
        </button>
      ))}
    </fieldset>
  );
}
