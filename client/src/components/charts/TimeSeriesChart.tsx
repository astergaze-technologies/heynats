import { useEffect, useMemo, useRef, useState } from 'react';
import type uPlot from 'uplot';
import type { MetricsWindow } from '@/lib/api';
import { formatClock } from '@/lib/format';
import { cn } from '@/lib/utils';

// Literal class names so Tailwind generates them; order matches --chart-N.
const SWATCHES = [
  'bg-chart-1',
  'bg-chart-2',
  'bg-chart-3',
  'bg-chart-4',
  'bg-chart-5',
  'bg-chart-6',
  'bg-chart-7',
  'bg-chart-8',
];
const cssVar = (name: string) =>
  getComputedStyle(document.documentElement).getPropertyValue(name).trim();

/** Re-renders when the theme class on <html> flips, so canvas colours follow. */
function useDarkMode() {
  const [dark, setDark] = useState(() =>
    document.documentElement.classList.contains('dark')
  );
  useEffect(() => {
    const el = document.documentElement;
    const obs = new MutationObserver(() =>
      setDark(el.classList.contains('dark'))
    );
    obs.observe(el, { attributes: true, attributeFilter: ['class'] });
    return () => obs.disconnect();
  }, []);
  return dark;
}

/**
 * One metric over time, one line per member (server, account, consumer).
 * Charts with the same syncKey share a crosshair.
 */
export function TimeSeriesChart({
  title,
  data,
  series,
  format,
  rangeSeconds,
  syncKey = 'metrics',
  height = 180,
}: {
  title: string;
  data?: MetricsWindow;
  series: string;
  format: (v: number) => string;
  rangeSeconds: number;
  syncKey?: string;
  height?: number;
}) {
  const plotRef = useRef<HTMLDivElement>(null);
  const plot = useRef<uPlot | null>(null);
  const [hover, setHover] = useState<number | null>(null);
  const [showTable, setShowTable] = useState(false);
  const dark = useDarkMode();

  const values = data?.series[series] ?? {};
  // Sorted by name so a member keeps its colour when others come and go.
  // shortcut: past 8 members the rest are dropped; fold into "Other" when a
  // chart can have more (consumers).
  const members = Object.keys(values).sort().slice(0, SWATCHES.length);
  const t = data?.t ?? [];
  const empty = t.length === 0 || members.length === 0;
  const memberKey = members.join('\n');
  // New only when the data does: hovering re-renders, and must not reset the plot.
  const aligned = useMemo(
    () => [t, ...members.map((m) => values[m])] as uPlot.AlignedData,
    [data, series, memberKey]
  );
  const latest = useRef(aligned);
  latest.current = aligned;

  // Build once per member set, theme and size; data refreshes go through setData.
  useEffect(() => {
    const el = plotRef.current;
    if (!el || !memberKey || showTable) return;
    let gone = false;
    const resize = new ResizeObserver(() =>
      plot.current?.setSize({ width: el.clientWidth, height })
    );
    Promise.all([import('uplot'), import('uplot/dist/uPlot.min.css')]).then(
      ([{ default: UPlot }]) => {
        if (gone) return;
        const muted = cssVar('--muted-foreground');
        const grid = { stroke: cssVar('--border'), width: 1 };
        const axis = { stroke: muted, grid, ticks: { show: false } };
        plot.current = new UPlot(
          {
            width: el.clientWidth,
            height,
            legend: { show: false },
            cursor: {
              sync: { key: syncKey },
              drag: { x: false, y: false },
              points: { size: 8 },
            },
            scales: { x: { time: true } },
            axes: [
              axis,
              {
                ...axis,
                size: 64,
                values: (_u, ticks) => ticks.map(format),
              },
            ],
            series: [
              {},
              ...members.map((m, i) => ({
                label: m,
                stroke: cssVar(`--chart-${i + 1}`),
                width: 2,
                points: { show: false },
              })),
            ],
            hooks: { setCursor: [(u) => setHover(u.cursor.idx ?? null)] },
          },
          latest.current,
          el
        );
        resize.observe(el);
      }
    );
    return () => {
      gone = true;
      resize.disconnect();
      plot.current?.destroy();
      plot.current = null;
    };
  }, [memberKey, series, height, syncKey, format, dark, showTable]);

  useEffect(() => {
    plot.current?.setData(aligned);
  }, [aligned]);

  const at = hover ?? t.length - 1;
  const since = data ? Date.parse(data.history_since) / 1000 : 0;
  const partialHistory = since > Date.now() / 1000 - rangeSeconds;

  return (
    <figure className="rounded-lg border bg-card p-4">
      <figcaption className="flex flex-wrap items-baseline gap-x-3 gap-y-1">
        <span className="font-semibold">{title}</span>
        {data && partialHistory && (
          <span className="text-xs text-muted-foreground">
            History since {formatClock(since)}
          </span>
        )}
        {!empty && (
          <button
            type="button"
            className="ml-auto text-xs text-muted-foreground underline-offset-2 hover:underline"
            aria-pressed={showTable}
            onClick={() => setShowTable((s) => !s)}
          >
            {showTable ? 'Show chart' : 'Show table'}
          </button>
        )}
      </figcaption>

      {empty ? (
        <p
          className="grid place-items-center text-sm text-muted-foreground"
          style={{ height }}
        >
          {data
            ? 'Collecting data… the first points appear within a few seconds.'
            : 'Loading…'}
        </p>
      ) : showTable ? (
        <div className="mt-3 max-h-72 overflow-auto">
          <table className="w-full text-sm tabular-nums">
            <thead className="sticky top-0 bg-card text-xs text-muted-foreground">
              <tr>
                <th className="py-1 text-left font-medium">Time</th>
                {members.map((m) => (
                  <th key={m} className="py-1 text-right font-medium">
                    {m}
                  </th>
                ))}
              </tr>
            </thead>
            <tbody>
              {t.map((ts, i) => (
                <tr key={ts} className="border-t">
                  <td className="py-1">{formatClock(ts)}</td>
                  {members.map((m) => {
                    const v = values[m][i];
                    return (
                      <td key={m} className="py-1 text-right">
                        {v == null ? '—' : format(v)}
                      </td>
                    );
                  })}
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      ) : (
        <>
          {/* Legend and readout in one: latest values, or the hovered time. */}
          <dl
            className="mt-2 flex flex-wrap gap-x-4 gap-y-1 text-xs"
            aria-live="off"
          >
            <div className="text-muted-foreground tabular-nums">
              {formatClock(t[at])}
            </div>
            {members.map((m, i) => {
              const v = values[m][at];
              return (
                <div key={m} className="flex items-center gap-1.5">
                  {members.length > 1 && (
                    <span
                      aria-hidden="true"
                      className={cn('h-0.5 w-3 rounded-full', SWATCHES[i])}
                    />
                  )}
                  <dt className="text-muted-foreground">{m}</dt>
                  <dd className="font-medium tabular-nums">
                    {v == null ? '—' : format(v)}
                  </dd>
                </div>
              );
            })}
          </dl>
          <div
            ref={plotRef}
            className="mt-2"
            style={{ height }}
            role="img"
            aria-label={`${title}, ${members.length} series. Use "Show table" for the values.`}
          />
        </>
      )}
    </figure>
  );
}
