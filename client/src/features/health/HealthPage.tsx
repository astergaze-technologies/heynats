import { CheckCircle2, Download } from 'lucide-react';
import { useId } from 'react';
import { useSearchParams } from 'react-router-dom';
import { EmptyState, LoadingState } from '@/components/PageStates';
import { SourceBadge } from '@/components/SourceBadge';
import { Button } from '@/components/ui/button';
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs';
import type { DiagnosticsReport } from '@/lib/api';
import { cn, formatTimestamp } from '@/lib/utils';
import { FindingCard, SEVERITY } from './FindingCard';
import { JetStreamTab } from './JetStreamTab';
import { toMarkdown } from './markdown';
import { useDiagnostics } from './useDiagnostics';

const VERDICT = {
  healthy: {
    label: 'Healthy',
    icon: CheckCircle2,
    tone: 'border-success/30 bg-success/10 text-success',
  },
  degraded: {
    label: 'Degraded',
    icon: SEVERITY.warning.icon,
    tone: 'border-warning/30 bg-warning/10 text-warning',
  },
  critical: {
    label: 'Critical',
    icon: SEVERITY.critical.icon,
    tone: 'border-destructive/30 bg-destructive/10 text-destructive',
  },
} as const;

function download(r: DiagnosticsReport) {
  const url = URL.createObjectURL(
    new Blob([toMarkdown(r)], { type: 'text/markdown' })
  );
  const a = document.createElement('a');
  a.href = url;
  a.download = `nats-health-${r.checked_at.slice(0, 19).replace(/:/g, '')}.md`;
  a.click();
  URL.revokeObjectURL(url);
}

export function HealthPage() {
  const [params, setParams] = useSearchParams();
  const tab = params.get('tab') === 'jetstream' ? 'jetstream' : 'overview';
  return (
    <div className="p-4 sm:p-6">
      <Tabs
        defaultValue="overview"
        value={tab}
        onValueChange={(t) =>
          setParams(
            (p) => {
              const next = new URLSearchParams(p);
              next.set('tab', t);
              return next;
            },
            { replace: true }
          )
        }
      >
        <TabsList>
          <TabsTrigger value="overview">Overview</TabsTrigger>
          <TabsTrigger value="jetstream">JetStream</TabsTrigger>
        </TabsList>
        <TabsContent value="overview" className="mt-4">
          <HealthOverview />
        </TabsContent>
        <TabsContent value="jetstream" className="mt-4">
          <JetStreamTab />
        </TabsContent>
      </Tabs>
    </div>
  );
}

function HealthOverview() {
  const { data: r, isLoading, error } = useDiagnostics();
  const serversId = useId();
  const findingsId = useId();
  if (isLoading) return <LoadingState label="Running health checks…" />;
  if (error || !r) {
    return (
      <div className="p-4 sm:p-6">
        <EmptyState
          tone="error"
          title="Couldn't run health checks"
          description={error?.message}
        />
      </div>
    );
  }

  const v = VERDICT[r.verdict];
  const unavailable = r.rules.filter((x) => !x.available);

  return (
    <div className="space-y-4">
      <section
        className={cn(
          'flex flex-wrap items-center gap-4 rounded-lg border p-5',
          v.tone
        )}
        aria-live="polite"
      >
        <v.icon className="size-10 shrink-0" aria-hidden="true" />
        <div>
          <p className="text-3xl font-semibold">{v.label}</p>
          <p className="text-sm">
            {r.findings.length === 0
              ? 'All checks passing.'
              : `${r.findings.length} open finding${r.findings.length === 1 ? '' : 's'}.`}{' '}
            Checked {formatTimestamp(r.checked_at)}.
          </p>
        </div>
        <div className="ml-auto flex items-center gap-2">
          <SourceBadge source={r.source} />
          <Button variant="outline" size="sm" onClick={() => download(r)}>
            <Download className="size-4" aria-hidden="true" />
            Export as Markdown
          </Button>
        </div>
      </section>

      <section
        aria-labelledby={serversId}
        className="rounded-lg border bg-card p-4"
      >
        <h2 id={serversId} className="mb-2 font-semibold">
          Servers
        </h2>
        <ul className="flex flex-wrap gap-2">
          {r.servers.map((s) => {
            const fine = s.responding && (!s.health || s.health === 'ok');
            return (
              <li
                key={s.name}
                className={cn(
                  'rounded-full px-3 py-1 text-sm',
                  fine
                    ? 'bg-success/10 text-success'
                    : 'bg-warning/10 text-warning'
                )}
              >
                {s.name} ·{' '}
                {!s.responding ? 'not responding' : fine ? 'ok' : s.health}
              </li>
            );
          })}
        </ul>
      </section>

      <section aria-labelledby={findingsId} className="space-y-3">
        <h2 id={findingsId} className="font-semibold">
          Open findings
        </h2>
        {r.findings.length === 0 ? (
          <p className="text-sm text-muted-foreground">
            Nothing to look at right now.
          </p>
        ) : (
          r.findings.map((f) => (
            <FindingCard key={`${f.rule_id}|${f.resource}`} finding={f} />
          ))
        )}
      </section>

      <details className="rounded-lg border bg-card p-4 text-sm">
        <summary className="cursor-pointer font-semibold">
          What is checked ({r.rules.length - unavailable.length} of{' '}
          {r.rules.length} checks run)
        </summary>
        <ul className="mt-3 space-y-2">
          {r.rules.map((x) => (
            <li
              key={x.id}
              className={cn(!x.available && 'text-muted-foreground')}
            >
              <code className="text-xs">{x.id}</code> · {x.doc}
              {!x.available && ` Not run: ${x.reason}.`}
            </li>
          ))}
        </ul>
      </details>
    </div>
  );
}
