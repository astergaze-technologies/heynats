import { useParams, useSearchParams } from 'react-router-dom';
import {
  RANGE_SECONDS,
  TimeRangePicker,
  useTimeRange,
} from '@/components/charts/TimeRangePicker';
import { TimeSeriesChart } from '@/components/charts/TimeSeriesChart';
import { JsonViewer } from '@/components/JsonViewer';
import { EmptyState, LoadingState } from '@/components/PageStates';
import { StatsCard } from '@/components/StatsCard';
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs';
import { useMetrics } from '@/features/dashboard/useOverview';
import type { MetricsWindow, ServerKind, ServerRow } from '@/lib/api';
import {
  formatBytes,
  formatCount,
  formatRate,
  formatUptime,
} from '@/lib/format';
import { useServerRaw, useServers } from './useServers';

const TABS = ['overview', 'config', 'routes', 'raw'] as const;
type Tab = (typeof TABS)[number];

/** Keep only one member, so the charts show this server alone. */
function only(
  w: MetricsWindow | undefined,
  member: string
): MetricsWindow | undefined {
  if (!w) return w;
  const series = Object.fromEntries(
    Object.entries(w.series).map(([name, members]) => [
      name,
      member in members ? { [member]: members[member] } : {},
    ])
  );
  return { ...w, series };
}

const percent = (v: number) => `${v.toFixed(1)}%`;
const perSecond = (v: number) => formatRate(v);

function Overview({ server }: { server: ServerRow }) {
  const [range, setRange] = useTimeRange();
  const { data } = useMetrics(['msgs_in', 'msgs_out', 'cpu', 'mem'], range);
  const mine = only(data, server.name);
  const chart = (
    title: string,
    series: string,
    format: (v: number) => string
  ) => (
    <TimeSeriesChart
      title={title}
      data={mine}
      series={series}
      format={format}
      rangeSeconds={RANGE_SECONDS[range]}
    />
  );
  return (
    <div className="space-y-3">
      <div className="grid grid-cols-2 gap-3 lg:grid-cols-4">
        <StatsCard
          title="Uptime"
          value={server.start ? formatUptime(server.start) : '—'}
        />
        <StatsCard title="CPU" value={percent(server.cpu)} />
        <StatsCard title="Memory" value={formatBytes(server.mem)} />
        <StatsCard
          title="Connections"
          value={formatCount(server.connections)}
          subtitle={`${formatCount(server.subscriptions)} subscriptions`}
        />
      </div>
      <TimeRangePicker value={range} onChange={setRange} />
      <div className="grid gap-3 lg:grid-cols-2">
        {chart('Messages in', 'msgs_in', perSecond)}
        {chart('Messages out', 'msgs_out', perSecond)}
        {chart('CPU', 'cpu', percent)}
        {chart('Memory', 'mem', formatBytes)}
      </div>
    </div>
  );
}

type Raw = any;

function Fields({ rows }: { rows: [string, unknown][] }) {
  return (
    <dl className="grid gap-x-6 gap-y-3 rounded-lg border bg-card p-4 sm:grid-cols-2 lg:grid-cols-3">
      {rows.map(([label, value]) => (
        <div key={label} className="min-w-0">
          <dt className="text-xs text-muted-foreground">{label}</dt>
          <dd className="truncate font-mono text-sm">
            {value === undefined || value === null || value === ''
              ? '—'
              : String(value)}
          </dd>
        </div>
      ))}
    </dl>
  );
}

function Config({ id }: { id: string }) {
  const { data: v, isLoading, error } = useServerRaw(id, 'VARZ');
  if (isLoading) return <LoadingState label="Loading configuration…" />;
  if (error || !v)
    return (
      <EmptyState
        tone="error"
        title="Couldn't load VARZ"
        description={error?.message}
      />
    );
  const js: Raw = v.jetstream?.config;
  return (
    <div className="space-y-3">
      <h3 className="text-sm font-semibold">Server</h3>
      <Fields
        rows={[
          ['Version', v.version],
          ['Go', v.go],
          ['Git commit', v.git_commit],
          ['Client address', `${v.host}:${v.port}`],
          ['Monitoring port', v.http_port],
          ['Auth required', v.auth_required ? 'yes' : 'no'],
          ['TLS required', v.tls_required ? 'yes' : 'no'],
          ['System account', v.system_account],
        ]}
      />
      <h3 className="text-sm font-semibold">Limits</h3>
      <Fields
        rows={[
          ['Max connections', formatCount(v.max_connections)],
          ['Max payload', formatBytes(v.max_payload)],
          ['Max pending', formatBytes(v.max_pending)],
          ['Max control line', formatBytes(v.max_control_line)],
          ['Ping interval', v.ping_interval && `${v.ping_interval / 1e9}s`],
          ['Max pings out', v.ping_max],
          ['Write deadline', v.write_deadline && `${v.write_deadline / 1e9}s`],
        ]}
      />
      <h3 className="text-sm font-semibold">JetStream</h3>
      {js ? (
        <Fields
          rows={[
            ['Max memory', formatBytes(js.max_memory)],
            ['Max storage', formatBytes(js.max_storage)],
            ['Store directory', js.store_dir],
            ['Sync interval', js.sync_interval && `${js.sync_interval / 1e9}s`],
          ]}
        />
      ) : (
        <p className="text-sm text-muted-foreground">
          JetStream is off on this server.
        </p>
      )}
      <h3 className="text-sm font-semibold">Cluster</h3>
      <Fields
        rows={[
          ['Name', v.cluster?.name],
          ['Route port', v.cluster?.port],
          ['Configured routes', (v.cluster?.urls ?? []).join(', ')],
          ['Gateway', v.gateway?.name],
          ['Leaf node port', v.leaf?.port],
        ]}
      />
    </div>
  );
}

function LinkTable({
  title,
  rows,
  empty,
}: {
  title: string;
  rows: {
    name: string;
    address: string;
    rtt?: string;
    inMsgs: number;
    outMsgs: number;
    subs: number;
  }[];
  empty: string;
}) {
  return (
    <section className="space-y-2">
      <h3 className="text-sm font-semibold">
        {title}{' '}
        <span className="font-normal text-muted-foreground">
          ({rows.length})
        </span>
      </h3>
      {rows.length === 0 ? (
        <p className="text-sm text-muted-foreground">{empty}</p>
      ) : (
        <div className="overflow-x-auto rounded-lg border bg-card">
          <table className="w-full text-sm">
            <thead className="border-b bg-muted/50 text-xs text-muted-foreground">
              <tr>
                <th scope="col" className="px-4 py-2 text-left font-medium">
                  Remote
                </th>
                <th scope="col" className="px-4 py-2 text-left font-medium">
                  Address
                </th>
                <th scope="col" className="px-4 py-2 text-right font-medium">
                  RTT
                </th>
                <th scope="col" className="px-4 py-2 text-right font-medium">
                  Msgs in
                </th>
                <th scope="col" className="px-4 py-2 text-right font-medium">
                  Msgs out
                </th>
                <th scope="col" className="px-4 py-2 text-right font-medium">
                  Subscriptions
                </th>
              </tr>
            </thead>
            <tbody className="divide-y tabular-nums">
              {rows.map((r, i) => (
                <tr key={`${r.name}-${i}`}>
                  <td className="px-4 py-2">{r.name}</td>
                  <td className="px-4 py-2 font-mono text-xs">{r.address}</td>
                  <td className="px-4 py-2 text-right">{r.rtt ?? '—'}</td>
                  <td className="px-4 py-2 text-right">
                    {formatCount(r.inMsgs)}
                  </td>
                  <td className="px-4 py-2 text-right">
                    {formatCount(r.outMsgs)}
                  </td>
                  <td className="px-4 py-2 text-right">
                    {formatCount(r.subs)}
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}
    </section>
  );
}

function Routes({ id }: { id: string }) {
  const routez = useServerRaw(id, 'ROUTEZ').data;
  const gatewayz = useServerRaw(id, 'GATEWAYZ').data;
  const leafz = useServerRaw(id, 'LEAFZ').data;
  const link = (r: Raw, name: string) => ({
    name,
    address: `${r.ip}:${r.port}`,
    rtt: r.rtt,
    inMsgs: r.in_msgs ?? 0,
    outMsgs: r.out_msgs ?? 0,
    subs: r.subscriptions ?? 0,
  });
  const gateways = [
    ...Object.entries(gatewayz?.outbound_gateways ?? {}).map(
      ([n, g]: [string, Raw]) => link(g.connection ?? {}, `${n} (out)`)
    ),
    ...Object.entries(gatewayz?.inbound_gateways ?? {}).flatMap(
      ([n, gs]: [string, Raw]) =>
        (gs as Raw[]).map((g) => link(g.connection ?? {}, `${n} (in)`))
    ),
  ];
  return (
    <div className="space-y-5">
      <LinkTable
        title="Routes"
        rows={(routez?.routes ?? []).map((r: Raw) =>
          link(r, r.remote_name || r.remote_id)
        )}
        empty="No routes: this server isn't clustered."
      />
      <LinkTable
        title="Gateways"
        rows={gateways}
        empty="No gateways configured."
      />
      <LinkTable
        title="Leaf nodes"
        rows={(leafz?.leafs ?? []).map((l: Raw) =>
          link(l, l.name || l.account)
        )}
        empty="No leaf nodes connected."
      />
    </div>
  );
}

const RAW_KINDS: ServerKind[] = [
  'VARZ',
  'HEALTHZ',
  'ROUTEZ',
  'GATEWAYZ',
  'LEAFZ',
  'JSZ',
  'SUBSZ',
  'CONNZ',
  'ACCOUNTZ',
];

function RawTab({ id }: { id: string }) {
  const [params, setParams] = useSearchParams();
  const kind = (RAW_KINDS.find((k) => k === params.get('kind')) ??
    'VARZ') as ServerKind;
  const { data, error } = useServerRaw(id, kind);
  return (
    <div className="space-y-3">
      <label className="flex items-center gap-2 text-sm">
        <span className="text-muted-foreground">Kind</span>
        <select
          value={kind}
          onChange={(e) =>
            setParams(
              (p) => {
                const next = new URLSearchParams(p);
                next.set('kind', e.target.value);
                return next;
              },
              { replace: true }
            )
          }
          className="rounded-md border bg-card px-2 py-1"
        >
          {RAW_KINDS.map((k) => (
            <option key={k}>{k}</option>
          ))}
        </select>
      </label>
      {error ? (
        <EmptyState
          tone="error"
          title={`Couldn't load ${kind}`}
          description={error.message}
        />
      ) : (
        <div className="rounded-lg border bg-card p-4">
          {data ? (
            <JsonViewer data={data} />
          ) : (
            <LoadingState label={`Loading ${kind}…`} />
          )}
        </div>
      )}
    </div>
  );
}

export function ServerDetailPage() {
  const { serverName = '' } = useParams();
  const [params, setParams] = useSearchParams();
  const tab = (TABS.find((t) => t === params.get('tab')) ?? 'overview') as Tab;
  const { data, isLoading, error } = useServers();

  if (isLoading) return <LoadingState label="Loading server…" />;
  const server = data?.servers.find((s) => s.name === serverName);
  if (error || !server) {
    return (
      <div className="p-4 sm:p-6">
        <EmptyState
          tone="error"
          title={
            error ? "Couldn't load servers" : `No server named ${serverName}`
          }
          description={
            error?.message ??
            'It may have left the cluster. Go back to the servers list.'
          }
        />
      </div>
    );
  }

  const setTab = (t: string) =>
    setParams(
      (p) => {
        const next = new URLSearchParams(p);
        next.set('tab', t);
        next.delete('kind');
        return next;
      },
      { replace: true }
    );

  return (
    <div className="space-y-4 p-4 sm:p-6">
      {!server.responding && (
        <p
          role="status"
          className="rounded-lg border border-destructive/30 bg-destructive/10 p-3 text-sm text-destructive"
        >
          {server.name} is not responding, so it has no live values. Its charts
          still show history up to when it stopped answering.
        </p>
      )}
      <Tabs defaultValue="overview" value={tab} onValueChange={setTab}>
        <TabsList>
          <TabsTrigger value="overview">Overview</TabsTrigger>
          <TabsTrigger value="config">Config</TabsTrigger>
          <TabsTrigger value="routes">Routes / Gateways / Leafs</TabsTrigger>
          <TabsTrigger value="raw">Raw</TabsTrigger>
        </TabsList>
        <TabsContent value="overview" className="mt-4">
          <Overview server={server} />
        </TabsContent>
        <TabsContent value="config" className="mt-4">
          <Config id={server.id} />
        </TabsContent>
        <TabsContent value="routes" className="mt-4">
          <Routes id={server.id} />
        </TabsContent>
        <TabsContent value="raw" className="mt-4">
          <RawTab id={server.id} />
        </TabsContent>
      </Tabs>
    </div>
  );
}
