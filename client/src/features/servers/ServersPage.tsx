import { Server } from 'lucide-react';
import { Link } from 'react-router-dom';
import { EmptyState, LoadingState } from '@/components/PageStates';
import { SourceBadge } from '@/components/SourceBadge';
import {
  useCapabilities,
  useConnectionStatus,
} from '@/features/connection/useNATS';
import type { ServerRow } from '@/lib/api';
import {
  formatBytes,
  formatCount,
  formatRate,
  formatUptime,
} from '@/lib/format';
import { cn } from '@/lib/utils';
import { useServers } from './useServers';

/** The version most servers run; others are flagged so upgrades don't stall half-way. */
function commonVersion(rows: ServerRow[]) {
  const counts = new Map<string, number>();
  for (const r of rows)
    if (r.version) counts.set(r.version, (counts.get(r.version) ?? 0) + 1);
  return [...counts].sort((a, b) => b[1] - a[1])[0]?.[0];
}

function Health({ row }: { row: ServerRow }) {
  const [label, tone] = !row.responding
    ? ['Not responding', 'bg-destructive/10 text-destructive']
    : row.health === 'ok'
      ? ['Healthy', 'bg-success/10 text-success']
      : [row.health || 'Unknown', 'bg-warning/10 text-warning'];
  return (
    <span className={cn('rounded-full px-2 py-0.5 text-xs font-medium', tone)}>
      {label}
    </span>
  );
}

const num = 'px-4 py-2.5 text-right tabular-nums';

export function ServersPage() {
  const { data: status } = useConnectionStatus();
  const { data: caps } = useCapabilities(status?.connected);
  const allowed = caps?.features.servers.ok;
  const { data, isLoading, error } = useServers(!!allowed);

  if (caps && !allowed) {
    return (
      <div className="p-4 sm:p-6">
        <EmptyState
          icon={<Server />}
          title="Servers need the system account"
          description={`${caps.features.servers.reason}. Connect with a user from the system account to see every server in the cluster.`}
        />
      </div>
    );
  }
  if (!caps || isLoading) return <LoadingState label="Loading servers…" />;
  if (error || !data) {
    return (
      <div className="p-4 sm:p-6">
        <EmptyState
          tone="error"
          title="Couldn't load servers"
          description={error?.message}
        />
      </div>
    );
  }

  const common = commonVersion(data.servers);
  const responding = data.servers.filter((s) => s.responding).length;

  return (
    <div className="space-y-3 p-4 sm:p-6">
      <div className="flex flex-wrap items-center gap-3 text-sm text-muted-foreground">
        <span>
          {responding} of {data.servers.length} servers responding
        </span>
        <SourceBadge
          source="system"
          partial={data.partial}
          servers={responding}
          expected={data.expected}
        />
      </div>
      <div className="overflow-x-auto rounded-lg border bg-card">
        <table className="w-full text-sm">
          <thead className="border-b bg-muted/50 text-xs text-muted-foreground">
            <tr>
              {['Name', 'Version', 'Health', 'Uptime'].map((h) => (
                <th
                  key={h}
                  scope="col"
                  className="px-4 py-2.5 text-left font-medium"
                >
                  {h}
                </th>
              ))}
              {[
                'CPU',
                'Memory',
                'Connections',
                'Subscriptions',
                'Slow consumers',
                'Msgs in',
                'Msgs out',
              ].map((h) => (
                <th
                  key={h}
                  scope="col"
                  className="px-4 py-2.5 text-right font-medium"
                >
                  {h}
                </th>
              ))}
              <th scope="col" className="px-4 py-2.5 text-left font-medium">
                JetStream
              </th>
            </tr>
          </thead>
          <tbody className="divide-y">
            {data.servers.map((s) => {
              const differs = s.version && common && s.version !== common;
              const live = s.responding;
              const dash = (v: string) => (live ? v : '—');
              return (
                <tr
                  key={s.name}
                  className={cn(
                    'hover:bg-muted/50',
                    !live && 'text-muted-foreground'
                  )}
                >
                  <td className="px-4 py-2.5">
                    <Link
                      to={`/dashboard/servers/${encodeURIComponent(s.name)}`}
                      className="font-medium text-foreground hover:text-primary hover:underline"
                    >
                      {s.name}
                    </Link>
                    {s.cluster && (
                      <div className="text-xs text-muted-foreground">
                        {s.cluster}
                      </div>
                    )}
                  </td>
                  <td className="px-4 py-2.5 tabular-nums">
                    {s.version}
                    {differs && (
                      <span className="ml-2 rounded bg-warning/10 px-1.5 py-0.5 text-xs text-warning">
                        differs from {common}
                      </span>
                    )}
                  </td>
                  <td className="px-4 py-2.5">
                    <Health row={s} />
                  </td>
                  <td className="px-4 py-2.5 tabular-nums">
                    {s.start && live ? formatUptime(s.start) : '—'}
                  </td>
                  <td className={num}>{dash(`${s.cpu.toFixed(1)}%`)}</td>
                  <td className={num}>{dash(formatBytes(s.mem))}</td>
                  <td className={num}>{dash(formatCount(s.connections))}</td>
                  <td className={num}>{dash(formatCount(s.subscriptions))}</td>
                  <td
                    className={cn(
                      num,
                      live && s.slow_consumers > 0 && 'text-warning'
                    )}
                  >
                    {dash(formatCount(s.slow_consumers))}
                  </td>
                  <td className={num}>
                    {s.msgs_in == null ? '—' : formatRate(s.msgs_in)}
                  </td>
                  <td className={num}>
                    {s.msgs_out == null ? '—' : formatRate(s.msgs_out)}
                  </td>
                  <td className="px-4 py-2.5">
                    {dash(s.jetstream ? 'On' : 'Off')}
                  </td>
                </tr>
              );
            })}
          </tbody>
        </table>
      </div>
    </div>
  );
}
