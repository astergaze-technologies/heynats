import { AlertTriangle } from 'lucide-react';
import { useId } from 'react';
import {
  RANGE_SECONDS,
  TimeRangePicker,
  useTimeRange,
} from '@/components/charts/TimeRangePicker';
import { TimeSeriesChart } from '@/components/charts/TimeSeriesChart';
import { useConnectionStatus } from '@/features/connection/useNATS';
import { formatRate } from '@/lib/format';
import { useMetrics } from './useOverview';

const perSecond = (v: number) => formatRate(v);

export function TrafficCharts() {
  const { data: status } = useConnectionStatus();
  const [range, setRange] = useTimeRange();
  const headingId = useId();
  const { data, error } = useMetrics(
    ['msgs_in', 'msgs_out'],
    range,
    !!status?.connected
  );

  return (
    <section aria-labelledby={headingId} className="space-y-3">
      <div className="flex flex-wrap items-center gap-3">
        <h2 id={headingId} className="font-semibold">
          Traffic
        </h2>
        <TimeRangePicker value={range} onChange={setRange} />
      </div>
      {error ? (
        <p className="flex items-center gap-2 rounded-lg border bg-card p-4 text-sm text-muted-foreground">
          <AlertTriangle className="size-4" aria-hidden="true" />
          {error.message}
        </p>
      ) : (
        <div className="grid gap-3 lg:grid-cols-2">
          <TimeSeriesChart
            title="Messages in per server"
            data={data}
            series="msgs_in"
            format={perSecond}
            rangeSeconds={RANGE_SECONDS[range]}
          />
          <TimeSeriesChart
            title="Messages out per server"
            data={data}
            series="msgs_out"
            format={perSecond}
            rangeSeconds={RANGE_SECONDS[range]}
          />
        </div>
      )}
    </section>
  );
}
