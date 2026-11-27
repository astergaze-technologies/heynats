import { SeverityBadge } from '@/features/health/FindingCard';
import { useJetStreamHealth } from '@/features/health/jetstream';
import { useDiagnostics } from '@/features/health/useDiagnostics';
import { formatCount } from '@/lib/format';

/** This stream's consumers with their lag and any open findings about them. */
export function StreamConsumers({ stream }: { stream: string }) {
  const { data: health } = useJetStreamHealth();
  const { data: report } = useDiagnostics();
  const consumers = (health?.consumers ?? []).filter(
    (c) => c.stream === stream
  );
  if (!health) return null;
  const findingsFor = (name: string) =>
    (report?.findings ?? []).filter(
      (f) =>
        f.resource.endsWith(`${stream}/${name}`) &&
        f.resource.startsWith('consumer:')
    );
  return (
    <section className="mb-4 shrink-0 rounded-lg border bg-card p-4">
      <h2 className="mb-3 text-base font-semibold">Consumers</h2>
      {consumers.length === 0 ? (
        <p className="text-sm text-muted-foreground">
          No consumers on this stream.
        </p>
      ) : (
        <ul className="divide-y text-sm">
          {consumers.map((c) => {
            const findings = findingsFor(c.name);
            return (
              <li key={c.name} className="py-2">
                <div className="flex flex-wrap items-center gap-x-4 gap-y-1">
                  <span className="font-medium">{c.name}</span>
                  <span className="text-muted-foreground">
                    {c.deliver_subject ? 'push' : 'pull'}
                  </span>
                  <span className="tabular-nums">
                    {formatCount(c.pending)} pending
                  </span>
                  <span className="tabular-nums text-muted-foreground">
                    {formatCount(c.ack_pending)} waiting for ack
                  </span>
                  {findings.length === 0 && (
                    <span className="text-success">no issues found</span>
                  )}
                </div>
                {findings.map((f) => (
                  <p key={f.rule_id} className="mt-1 flex items-center gap-2">
                    <SeverityBadge severity={f.severity} />
                    {f.title}
                  </p>
                ))}
              </li>
            );
          })}
        </ul>
      )}
    </section>
  );
}
