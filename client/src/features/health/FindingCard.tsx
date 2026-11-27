import { AlertTriangle, Check, Copy, Info, XCircle } from 'lucide-react';
import { useState } from 'react';
import { Link } from 'react-router-dom';
import type { Finding, Severity } from '@/lib/api';
import { cn, formatTimestamp } from '@/lib/utils';

export const SEVERITY: Record<
  Severity,
  { label: string; icon: typeof Info; tone: string }
> = {
  critical: {
    label: 'Critical',
    icon: XCircle,
    tone: 'bg-destructive/10 text-destructive',
  },
  warning: {
    label: 'Warning',
    icon: AlertTriangle,
    tone: 'bg-warning/10 text-warning',
  },
  info: { label: 'Info', icon: Info, tone: 'bg-muted text-muted-foreground' },
};

export function SeverityBadge({ severity }: { severity: Severity }) {
  const { label, icon: Icon, tone } = SEVERITY[severity];
  return (
    <span
      className={cn(
        'inline-flex items-center gap-1 rounded-full px-2 py-0.5 text-xs font-medium',
        tone
      )}
    >
      <Icon className="size-3.5" aria-hidden="true" />
      {label}
    </span>
  );
}

export function CopyCommand({ cli }: { cli: string }) {
  const [copied, setCopied] = useState(false);
  return (
    <span className="inline-flex items-center gap-1 rounded-md border bg-muted px-2 py-1 font-mono text-xs">
      <code>{cli}</code>
      <button
        type="button"
        aria-label={`Copy ${cli}`}
        className="text-muted-foreground hover:text-foreground"
        onClick={() =>
          navigator.clipboard.writeText(cli).then(() => {
            setCopied(true);
            setTimeout(() => setCopied(false), 1500);
          })
        }
      >
        {copied ? (
          <Check className="size-3.5" />
        ) : (
          <Copy className="size-3.5" />
        )}
      </button>
    </span>
  );
}

export function FindingCard({ finding: f }: { finding: Finding }) {
  return (
    <article className="rounded-lg border bg-card p-4">
      <div className="flex flex-wrap items-center gap-2">
        <SeverityBadge severity={f.severity} />
        <h3 className="font-semibold">{f.title}</h3>
        <span className="ml-auto text-xs text-muted-foreground">
          since {formatTimestamp(f.first_seen)}
        </span>
      </div>
      <p className="mt-1 font-mono text-xs text-muted-foreground">
        {f.resource} · {f.rule_id}
      </p>
      <p className="mt-2 text-sm">{f.detail}</p>
      {f.evidence.length > 0 && (
        <dl className="mt-3 grid gap-x-6 gap-y-1 text-sm sm:grid-cols-2">
          {f.evidence.map((e) => (
            <div key={e.label} className="flex gap-2">
              <dt className="text-muted-foreground">{e.label}</dt>
              <dd className="font-medium tabular-nums">{e.value}</dd>
              <dd className="text-xs text-muted-foreground">({e.source})</dd>
            </div>
          ))}
        </dl>
      )}
      {f.fixes && f.fixes.length > 0 && (
        <div className="mt-3 flex flex-wrap items-center gap-2 text-sm">
          {f.fixes.map((x) =>
            x.link ? (
              <Link
                key={x.label}
                to={x.link}
                className="text-primary hover:underline"
              >
                {x.label}
              </Link>
            ) : x.cli ? (
              <span key={x.label} className="inline-flex items-center gap-2">
                <span className="text-muted-foreground">{x.label}:</span>
                <CopyCommand cli={x.cli} />
              </span>
            ) : null
          )}
        </div>
      )}
    </article>
  );
}
