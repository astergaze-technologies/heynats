import { UsageBar } from '@/components/charts/UsageBar';
import { SourceBadge } from '@/components/SourceBadge';
import { StatsCard } from '@/components/StatsCard';
import {
  useAccountInfo,
  useCapabilities,
  useConnectionStatus,
} from '@/features/connection/useNATS';
import type { MetricsWindow } from '@/lib/api';
import { formatBytes, formatCount, formatRate } from '@/lib/format';
import { useJetStreamSummary, useMetrics } from './useOverview';

/** Sum over members at each point; null where no member had a value. */
export function totals(w: MetricsWindow | undefined, name: string) {
  const members = Object.values(w?.series[name] ?? {});
  return (w?.t ?? []).map((_, i) => {
    const vals = members.map((m) => m[i]).filter((v) => v != null);
    return vals.length ? vals.reduce((a, b) => a + b, 0) : null;
  });
}

function lastValue(vals: (number | null)[]) {
  for (let i = vals.length - 1; i >= 0; i--)
    if (vals[i] != null) return vals[i];
  return null;
}

export function OverviewStats() {
  const { data: status } = useConnectionStatus();
  const on = !!status?.connected;
  const { data: caps } = useCapabilities(on);
  const { data: account } = useAccountInfo(on);
  const { data: js } = useJetStreamSummary(on);
  // Last 5 minutes is enough for the tile trend, whatever range the chart shows.
  const { data: rates } = useMetrics(['msgs_in', 'msgs_out'], '5m', on);

  const connz = account?.connection_limits;
  const msgsIn = totals(rates, 'msgs_in');
  const msgsOut = totals(rates, 'msgs_out');
  const rate = (vals: (number | null)[]) => {
    const v = lastValue(vals);
    return v == null ? '—' : formatRate(v);
  };
  const ratesBadge = rates?.source && <SourceBadge source={rates.source} />;

  return (
    <div className="grid grid-cols-2 gap-3 md:grid-cols-3 xl:grid-cols-6">
      <StatsCard
        title="Servers"
        value={
          caps
            ? caps.partial
              ? `${caps.servers} of ${caps.expected}`
              : caps.servers
            : '—'
        }
        subtitle={caps && `NATS ${caps.server_version}`}
        badge={caps && <SourceBadge {...caps} />}
      />
      <StatsCard
        title="Connections"
        value={connz ? connz.num_connections : '—'}
        subtitle={connz && `across ${connz.servers} servers`}
        badge={connz && <SourceBadge {...connz} />}
      />
      <StatsCard
        title="Messages in"
        value={rate(msgsIn)}
        trend={msgsIn.slice(-30)}
        badge={ratesBadge}
      />
      <StatsCard
        title="Messages out"
        value={rate(msgsOut)}
        trend={msgsOut.slice(-30)}
        badge={ratesBadge}
      />
      <StatsCard
        title="JetStream storage"
        value={js ? formatBytes(js.storage) : '—'}
        subtitle={js && js.max_storage < 0 ? 'no limit set' : undefined}
        badge={js && <SourceBadge {...js} />}
      >
        {js && js.max_storage > 0 && (
          <div className="mt-2">
            <UsageBar
              label="Used of limit"
              value={js.storage}
              max={js.max_storage}
              format={formatBytes}
            />
          </div>
        )}
      </StatsCard>
      <StatsCard
        title="Streams · consumers"
        value={
          js ? `${formatCount(js.streams)} · ${formatCount(js.consumers)}` : '—'
        }
        badge={js && <SourceBadge {...js} />}
      />
    </div>
  );
}
