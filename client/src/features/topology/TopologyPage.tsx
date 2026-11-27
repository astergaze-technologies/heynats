import { Network, X } from 'lucide-react';
import { useState } from 'react';
import { Link, useSearchParams } from 'react-router-dom';
import { EmptyState, LoadingState } from '@/components/PageStates';
import { SourceBadge } from '@/components/SourceBadge';
import { Button } from '@/components/ui/button';
import {
  useCapabilities,
  useConnectionStatus,
} from '@/features/connection/useNATS';
import { useServers } from '@/features/servers/useServers';
import type { TopoEdge, TopoNode } from '@/lib/api';
import { formatBytes, formatRate, formatUptime } from '@/lib/format';
import { cn } from '@/lib/utils';
import { HEIGHT, layout, WIDTH } from './layout';
import { useTopology } from './useTopology';

const EDGE_STYLE: Record<TopoEdge['kind'], string> = {
  route: 'stroke-muted-foreground/50',
  gateway: 'stroke-primary',
  leaf: 'stroke-chart-3',
};
const KIND_LABEL: Record<TopoNode['kind'], string> = {
  server: 'Server',
  gateway: 'Remote cluster (gateway)',
  leaf: 'Leaf node',
};

function Inspector({ node, onClose }: { node: TopoNode; onClose: () => void }) {
  const { data } = useServers(node.kind === 'server');
  const s = data?.servers.find((x) => x.name === node.id);
  return (
    <aside
      aria-label="Node details"
      className="rounded-lg border bg-card p-4 text-sm"
    >
      <div className="mb-2 flex items-center gap-2">
        <h2 className="font-semibold">{node.label}</h2>
        <span className="text-muted-foreground">{KIND_LABEL[node.kind]}</span>
        <Button
          variant="ghost"
          size="icon"
          className="ml-auto"
          aria-label="Close details"
          onClick={onClose}
        >
          <X className="size-4" />
        </Button>
      </div>
      {!node.responding && (
        <p className="mb-2 text-destructive">Not responding</p>
      )}
      {s?.responding && (
        <dl className="grid grid-cols-[auto_1fr] gap-x-3 gap-y-1">
          <dt className="text-muted-foreground">Cluster</dt>
          <dd>{s.cluster || '—'}</dd>
          <dt className="text-muted-foreground">Version</dt>
          <dd>{s.version}</dd>
          <dt className="text-muted-foreground">Uptime</dt>
          <dd>{s.start ? formatUptime(s.start) : '—'}</dd>
          <dt className="text-muted-foreground">CPU / memory</dt>
          <dd>
            {s.cpu.toFixed(1)}% / {formatBytes(s.mem)}
          </dd>
          <dt className="text-muted-foreground">Connections</dt>
          <dd>{s.connections}</dd>
        </dl>
      )}
      {node.kind === 'server' && (
        <Link
          to={`/dashboard/servers/${encodeURIComponent(node.id)}`}
          className="mt-3 inline-block text-primary hover:underline"
        >
          Open server
        </Link>
      )}
    </aside>
  );
}

export function TopologyPage() {
  const { data: status } = useConnectionStatus();
  const { data: caps } = useCapabilities(status?.connected);
  const allowed = !!caps?.features.servers.ok;
  const { data: topo, isLoading, error, rateOf } = useTopology(allowed);
  const [params, setParams] = useSearchParams();
  const [asList, setAsList] = useState(false);
  const selected = params.get('node');
  const select = (id: string | null) =>
    setParams(
      (p) => {
        const next = new URLSearchParams(p);
        if (id) next.set('node', id);
        else next.delete('node');
        return next;
      },
      { replace: true }
    );

  if (caps && !allowed) {
    return (
      <div className="p-4 sm:p-6">
        <EmptyState
          icon={<Network />}
          title="Topology needs the system account"
          description={`${caps.features.servers.reason}. Connect with a system account user to see how servers, clusters and leaf nodes connect.`}
        />
      </div>
    );
  }
  if (!caps || isLoading) return <LoadingState label="Loading topology…" />;
  if (error || !topo) {
    return (
      <div className="p-4 sm:p-6">
        <EmptyState
          tone="error"
          title="Couldn't load topology"
          description={error?.message}
        />
      </div>
    );
  }

  const pos = layout(topo.nodes, topo.edges);
  const byId = new Map(topo.nodes.map((n) => [n.id, n]));
  const node = selected ? byId.get(selected) : undefined;
  const rate = (e: TopoEdge) => {
    const r = rateOf(e);
    return r == null ? '…' : formatRate(r);
  };
  const responding = topo.nodes.filter(
    (n) => n.kind === 'server' && n.responding
  ).length;

  return (
    <div className="space-y-3 p-4 sm:p-6">
      <div className="flex flex-wrap items-center gap-3 text-sm text-muted-foreground">
        <span>
          {responding} of {topo.nodes.filter((n) => n.kind === 'server').length}{' '}
          servers responding · {topo.edges.length} links
        </span>
        <SourceBadge
          source="system"
          partial={topo.partial}
          servers={responding}
          expected={topo.expected}
        />
        <span className="flex items-center gap-3 text-xs">
          {(['route', 'gateway', 'leaf'] as const).map((k) => (
            <span key={k} className="flex items-center gap-1">
              <svg width="18" height="4" aria-hidden="true">
                <line
                  x1="0"
                  y1="2"
                  x2="18"
                  y2="2"
                  strokeWidth="2"
                  className={EDGE_STYLE[k]}
                />
              </svg>
              {k === 'route'
                ? 'route'
                : k === 'gateway'
                  ? 'gateway'
                  : 'leaf node'}
            </span>
          ))}
        </span>
        <Button
          variant="ghost"
          size="sm"
          className="ml-auto"
          aria-pressed={asList}
          onClick={() => setAsList((v) => !v)}
        >
          {asList ? 'Show graph' : 'Show as list'}
        </Button>
      </div>

      <div className={cn('grid gap-3', node && 'lg:grid-cols-[1fr_18rem]')}>
        {asList ? (
          <div className="overflow-x-auto rounded-lg border bg-card">
            <table className="w-full text-sm">
              <thead className="border-b bg-muted/50 text-xs text-muted-foreground">
                <tr>
                  {['From', 'To', 'Kind', 'Connections', 'Messages'].map(
                    (h) => (
                      <th
                        key={h}
                        scope="col"
                        className="px-4 py-2 text-left font-medium"
                      >
                        {h}
                      </th>
                    )
                  )}
                </tr>
              </thead>
              <tbody className="divide-y">
                {topo.edges.map((e) => (
                  <tr key={`${e.kind}|${e.from}|${e.to}`}>
                    <td className="px-4 py-2">{byId.get(e.from)?.label}</td>
                    <td className="px-4 py-2">{byId.get(e.to)?.label}</td>
                    <td className="px-4 py-2">{e.kind}</td>
                    <td className="px-4 py-2 tabular-nums">{e.links}</td>
                    <td className="px-4 py-2 tabular-nums">{rate(e)}</td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        ) : (
          <div
            className="relative rounded-lg border bg-card"
            style={{ aspectRatio: `${WIDTH} / ${HEIGHT}` }}
          >
            <svg
              viewBox={`0 0 ${WIDTH} ${HEIGHT}`}
              className="absolute inset-0 size-full"
              aria-hidden="true"
            >
              {topo.edges.map((e) => {
                const a = pos.get(e.from);
                const b = pos.get(e.to);
                if (!a || !b) return null;
                return (
                  <g key={`${e.kind}|${e.from}|${e.to}`}>
                    <line
                      x1={a.x}
                      y1={a.y}
                      x2={b.x}
                      y2={b.y}
                      strokeWidth={2}
                      className={EDGE_STYLE[e.kind]}
                    />
                    <text
                      x={(a.x + b.x) / 2}
                      y={(a.y + b.y) / 2 - 6}
                      textAnchor="middle"
                      className="fill-muted-foreground text-[13px] tabular-nums"
                    >
                      {rate(e)}
                    </text>
                  </g>
                );
              })}
            </svg>
            {topo.nodes.map((n) => {
              const p = pos.get(n.id);
              if (!p) return null;
              return (
                <button
                  key={n.id}
                  type="button"
                  aria-pressed={n.id === selected}
                  aria-label={`${KIND_LABEL[n.kind]} ${n.label}${n.responding ? '' : ', not responding'}`}
                  onClick={() => select(n.id === selected ? null : n.id)}
                  style={{
                    left: `${(p.x / WIDTH) * 100}%`,
                    top: `${(p.y / HEIGHT) * 100}%`,
                  }}
                  className={cn(
                    'absolute -translate-x-1/2 -translate-y-1/2 rounded-full border-2 bg-card px-3 py-1.5 text-sm font-medium shadow-sm hover:border-primary',
                    n.kind === 'server' && 'border-foreground/60',
                    n.kind === 'gateway' && 'rounded-md border-primary',
                    n.kind === 'leaf' && 'rounded-md border-chart-3 text-xs',
                    !n.responding &&
                      'border-dashed border-destructive/60 text-muted-foreground opacity-70',
                    n.id === selected && 'ring-2 ring-ring'
                  )}
                >
                  {n.label}
                  {!n.responding && (
                    <span className="block text-[11px] font-normal">
                      not responding
                    </span>
                  )}
                </button>
              );
            })}
          </div>
        )}
        {node && <Inspector node={node} onClose={() => select(null)} />}
      </div>
    </div>
  );
}
