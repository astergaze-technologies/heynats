import { Search, Unplug } from 'lucide-react';
import { useState } from 'react';
import { EmptyState, LoadingState } from '@/components/PageStates';
import { SourceBadge } from '@/components/SourceBadge';
import { Input } from '@/components/ui/input';
import {
  useAccountInfo,
  useConnectionStatus,
  useNATSInfo,
} from '@/features/connection/useNATS';
import { formatRTT, formatTimestamp } from '@/lib/utils';
import { ConnectionsTable } from './ConnectionsTable';
import { OverviewStats } from './OverviewStats';
import { PartialBanner } from './PartialBanner';
import { TrafficCharts } from './TrafficCharts';

function Field({
  label,
  value,
  sub,
}: {
  label: string;
  value: React.ReactNode;
  sub?: React.ReactNode;
}) {
  return (
    <div className="min-w-0">
      <dt className="text-xs text-muted-foreground">{label}</dt>
      <dd
        className="mt-0.5 truncate font-mono text-sm"
        title={typeof value === 'string' ? value : undefined}
      >
        {value}
      </dd>
      {sub && <dd className="truncate text-xs text-muted-foreground">{sub}</dd>}
    </div>
  );
}

export function DashboardPage() {
  const [searchQuery, setSearchQuery] = useState('');
  const { data: status } = useConnectionStatus();
  const {
    data: natsInfo,
    isLoading: infoLoading,
    error: infoError,
  } = useNATSInfo(status?.connected);
  const { data: accountInfo, isLoading: accountLoading } = useAccountInfo(
    status?.connected
  );

  if (infoLoading || accountLoading) {
    return <LoadingState label="Loading server information…" />;
  }

  if (infoError) {
    return (
      <div className="p-4 sm:p-6">
        <EmptyState
          tone="error"
          title="Couldn't reach the server"
          description={`${infoError.message}. Try disconnecting and connecting again.`}
        />
      </div>
    );
  }

  const info = accountInfo?.account_information;
  const connections = accountInfo?.connection_limits?.connections || [];

  const query = searchQuery.trim().toLowerCase();
  const filteredConnections = connections.filter(
    (c) =>
      !query ||
      [c.name, c.lang, String(c.cid), `${c.ip}:${c.port}`].some((v) =>
        v?.toLowerCase().includes(query)
      )
  );

  return (
    <div className="space-y-4 p-4 sm:p-6">
      <PartialBanner />
      <OverviewStats />
      <TrafficCharts />

      {info && (
        <section
          className="rounded-lg border bg-card p-4"
          aria-labelledby="server-heading"
        >
          <div className="mb-3 flex items-center justify-between">
            <h2 id="server-heading" className="font-semibold">
              Server
            </h2>
            {natsInfo && (
              <span className="inline-flex items-center gap-1.5 rounded-full bg-success/10 px-2 py-0.5 text-xs font-medium text-success">
                <span
                  className="size-1.5 rounded-full bg-success"
                  aria-hidden="true"
                />
                {natsInfo.is_connected ? 'Connected' : 'Disconnected'}
              </span>
            )}
          </div>
          <dl className="grid grid-cols-2 gap-4 sm:grid-cols-3 lg:grid-cols-5">
            <Field
              label="Name"
              value={info.server_name}
              sub={`v${info.server_version}`}
            />
            <Field label="URL" value={info.connected_url} />
            <Field
              label="Client ID"
              value={info.client_id}
              sub={`${info.client_ip} → ${info.local_ip}`}
            />
            <Field
              label="RTT"
              value={formatRTT(info.rtt)}
              sub={`max payload ${info.max_payload}`}
            />
            <Field
              label="User"
              value={info.user || '—'}
              sub={
                info.header_supported
                  ? 'headers supported'
                  : 'no header support'
              }
            />
          </dl>
        </section>
      )}

      <section
        className="rounded-lg border bg-card"
        aria-labelledby="connections-heading"
      >
        <div className="flex flex-wrap items-center gap-3 p-4">
          <h2 id="connections-heading" className="font-semibold">
            Client connections
          </h2>
          {accountInfo && (
            <span className="text-sm text-muted-foreground">
              {query ? `${filteredConnections.length} of ` : ''}
              {accountInfo.connection_limits.num_connections} · limit{' '}
              {accountInfo.connection_limits.limit}
            </span>
          )}
          {accountInfo && (
            <SourceBadge
              source={accountInfo.connection_limits.source}
              partial={accountInfo.connection_limits.partial}
              servers={accountInfo.connection_limits.servers}
              expected={accountInfo.connection_limits.expected}
            />
          )}
          <span className="ml-auto text-xs text-muted-foreground">
            Updated{' '}
            {accountInfo?.connection_limits.now
              ? formatTimestamp(accountInfo.connection_limits.now)
              : '—'}
          </span>
          <div className="relative w-full sm:w-72">
            <Search
              className="pointer-events-none absolute top-1/2 left-3 size-4 -translate-y-1/2 text-muted-foreground"
              aria-hidden="true"
            />
            <Input
              type="search"
              aria-label="Search connections"
              placeholder="Name, IP, language or CID…"
              value={searchQuery}
              onChange={(e) => setSearchQuery(e.target.value)}
              className="pl-9"
            />
          </div>
        </div>

        {filteredConnections.length > 0 ? (
          <ConnectionsTable connections={filteredConnections} />
        ) : (
          <EmptyState
            className="m-4 mt-0"
            icon={query ? <Search /> : <Unplug />}
            title={
              query
                ? 'No connections match your search'
                : 'No client connections'
            }
            description={
              query ? `Nothing matches “${searchQuery}”.` : undefined
            }
          />
        )}
      </section>
    </div>
  );
}
