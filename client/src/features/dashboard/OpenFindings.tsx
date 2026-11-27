import { useId } from 'react';
import { Link } from 'react-router-dom';
import { SeverityBadge } from '@/features/health/FindingCard';
import { useDiagnostics } from '@/features/health/useDiagnostics';

/** Overview card: the worst open findings, linking to the Health page. */
export function OpenFindings() {
  const { data: r } = useDiagnostics();
  const headingId = useId();
  if (!r) return null;
  return (
    <section
      className="rounded-lg border bg-card p-4"
      aria-labelledby={headingId}
    >
      <div className="flex items-center gap-2">
        <h2 id={headingId} className="font-semibold">
          Open findings
        </h2>
        <span className="text-sm text-muted-foreground">
          ({r.findings.length})
        </span>
        <Link
          to="/dashboard/health"
          className="ml-auto text-sm text-primary hover:underline"
        >
          Open health
        </Link>
      </div>
      {r.findings.length === 0 ? (
        <p className="mt-2 text-sm text-muted-foreground">
          All checks passing.
        </p>
      ) : (
        <ul className="mt-2 space-y-1.5">
          {r.findings.slice(0, 3).map((f) => (
            <li
              key={`${f.rule_id}|${f.resource}`}
              className="flex items-center gap-2 text-sm"
            >
              <SeverityBadge severity={f.severity} />
              {f.title}
            </li>
          ))}
        </ul>
      )}
    </section>
  );
}
