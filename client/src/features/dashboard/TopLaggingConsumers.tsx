import { useId } from 'react';
import { Link } from 'react-router-dom';
import { useJetStreamHealth } from '@/features/health/jetstream';
import { formatCount } from '@/lib/format';

/** Overview card: consumers with the most messages waiting. */
export function TopLaggingConsumers() {
  const { data } = useJetStreamHealth();
  const headingId = useId();
  const top = (data?.consumers ?? []).filter((c) => c.pending > 0).slice(0, 5);
  if (!data) return null;
  return (
    <section
      className="rounded-lg border bg-card p-4"
      aria-labelledby={headingId}
    >
      <div className="flex items-center gap-2">
        <h2 id={headingId} className="font-semibold">
          Top lagging consumers
        </h2>
        <Link
          to="/dashboard/health?tab=jetstream"
          className="ml-auto text-sm text-primary hover:underline"
        >
          JetStream health
        </Link>
      </div>
      {top.length === 0 ? (
        <p className="mt-2 text-sm text-muted-foreground">
          Every consumer is caught up.
        </p>
      ) : (
        <ul className="mt-2 space-y-1.5 text-sm">
          {top.map((c) => (
            <li
              key={`${c.account}/${c.stream}/${c.name}`}
              className="flex gap-2"
            >
              <span className="font-medium">{c.name}</span>
              <span className="text-muted-foreground">on {c.stream}</span>
              <span className="ml-auto tabular-nums">
                {formatCount(c.pending)} pending
              </span>
            </li>
          ))}
        </ul>
      )}
    </section>
  );
}
