import { Sparkline } from '@/components/charts/Sparkline';
import { EmptyState, LoadingState } from '@/components/PageStates';
import { SourceBadge } from '@/components/SourceBadge';
import { useMetrics } from '@/features/dashboard/useOverview';
import { formatBytes, formatCount } from '@/lib/format';
import {
  consumerKey,
  Replicas,
  StatusBadge,
  streamStatus,
  useJetStreamHealth,
} from './jetstream';

const th = 'px-4 py-2 text-left font-medium';

export function JetStreamTab() {
  const { data: h, isLoading, error } = useJetStreamHealth();
  const { data: lag } = useMetrics(['consumer_pending'], '1h');
  if (isLoading) return <LoadingState label="Loading JetStream health…" />;
  if (error || !h)
    return (
      <EmptyState
        tone="error"
        title="Couldn't read JetStream"
        description={error?.message}
      />
    );

  const pending = lag?.series.consumer_pending ?? {};
  return (
    <div className="space-y-5">
      <div className="flex items-center gap-2 text-sm text-muted-foreground">
        <span>
          {h.streams.length} streams · {h.consumers.length} consumers
        </span>
        <SourceBadge source={h.source} partial={h.partial} />
      </div>

      {h.meta ? (
        <section className="rounded-lg border bg-card p-4 text-sm">
          <h3 className="mb-2 font-semibold">Meta group</h3>
          <p className="mb-2 text-muted-foreground">
            The servers that agree on which streams and consumers exist.{' '}
            {h.meta.cluster_size} members.
          </p>
          {h.meta.leader ? (
            <Replicas leader={h.meta.leader} peers={h.meta.peers} />
          ) : (
            <StatusBadge
              status={{
                label: 'No leader: streams and consumers cannot change',
                tone: 'critical',
              }}
            />
          )}
        </section>
      ) : (
        <p className="text-sm text-muted-foreground">
          The meta group is visible to system account users only.
        </p>
      )}

      <section className="space-y-2">
        <h3 className="text-sm font-semibold">Streams</h3>
        {h.streams.length === 0 ? (
          <p className="text-sm text-muted-foreground">No streams.</p>
        ) : (
          <div className="overflow-x-auto rounded-lg border bg-card">
            <table className="w-full text-sm">
              <thead className="border-b bg-muted/50 text-xs text-muted-foreground">
                <tr>
                  <th scope="col" className={th}>
                    Stream
                  </th>
                  <th scope="col" className={th}>
                    Health
                  </th>
                  <th scope="col" className={th}>
                    Replicas
                  </th>
                  <th scope="col" className={`${th} text-right`}>
                    Messages
                  </th>
                  <th scope="col" className={`${th} text-right`}>
                    Size
                  </th>
                </tr>
              </thead>
              <tbody className="divide-y">
                {h.streams.map((s) => (
                  <tr key={`${s.account}/${s.name}`}>
                    <td className="px-4 py-2">
                      <span className="font-medium">{s.name}</span>
                      {s.account && (
                        <span className="ml-2 text-xs text-muted-foreground">
                          {s.account}
                        </span>
                      )}
                    </td>
                    <td className="px-4 py-2">
                      <StatusBadge status={streamStatus(s)} />
                    </td>
                    <td className="px-4 py-2">
                      {s.clustered ? (
                        <Replicas leader={s.leader} peers={s.replicas} />
                      ) : (
                        <span className="text-muted-foreground">
                          single copy
                        </span>
                      )}
                    </td>
                    <td className="px-4 py-2 text-right tabular-nums">
                      {formatCount(s.messages)}
                    </td>
                    <td className="px-4 py-2 text-right tabular-nums">
                      {formatBytes(s.bytes)}
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        )}
      </section>

      <section className="space-y-2">
        <h3 className="text-sm font-semibold">
          Consumers (biggest backlog first)
        </h3>
        {h.consumers.length === 0 ? (
          <p className="text-sm text-muted-foreground">No consumers.</p>
        ) : (
          <div className="overflow-x-auto rounded-lg border bg-card">
            <table className="w-full text-sm">
              <thead className="border-b bg-muted/50 text-xs text-muted-foreground">
                <tr>
                  <th scope="col" className={th}>
                    Consumer
                  </th>
                  <th scope="col" className={th}>
                    Leader
                  </th>
                  <th scope="col" className={`${th} text-right`}>
                    Pending
                  </th>
                  <th scope="col" className={`${th} text-right`}>
                    Ack pending
                  </th>
                  <th scope="col" className={`${th} text-right`}>
                    Redelivered
                  </th>
                  <th scope="col" className={`${th} w-40`}>
                    Pending, last hour
                  </th>
                </tr>
              </thead>
              <tbody className="divide-y">
                {h.consumers.map((c) => {
                  const key = consumerKey(c);
                  return (
                    <tr key={key}>
                      <td className="px-4 py-2">
                        <span className="font-medium">{c.name}</span>
                        <span className="ml-2 text-xs text-muted-foreground">
                          on {c.stream}
                          {c.account && ` · ${c.account}`}
                        </span>
                      </td>
                      <td className="px-4 py-2">{c.leader || '—'}</td>
                      <td className="px-4 py-2 text-right tabular-nums">
                        {formatCount(c.pending)}
                      </td>
                      <td className="px-4 py-2 text-right tabular-nums">
                        {formatCount(c.ack_pending)}
                      </td>
                      <td className="px-4 py-2 text-right tabular-nums">
                        {formatCount(c.redelivered)}
                      </td>
                      <td className="px-4 py-2">
                        {pending[key] ? (
                          <Sparkline values={pending[key]} />
                        ) : (
                          <span className="text-xs text-muted-foreground">
                            no history yet
                          </span>
                        )}
                      </td>
                    </tr>
                  );
                })}
              </tbody>
            </table>
          </div>
        )}
      </section>
    </div>
  );
}
