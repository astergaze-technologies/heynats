import { useQuery } from '@tanstack/react-query';
import { ChevronRight, Ear, HardDrive, X } from 'lucide-react';
import { useState } from 'react';
import { useSearchParams } from 'react-router-dom';
import { EmptyState, LoadingState } from '@/components/PageStates';
import { SourceBadge } from '@/components/SourceBadge';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { useConnectionStatus } from '@/features/connection/useNATS';
import { useJetStreamHealth } from '@/features/health/jetstream';
import { natsApi, type SubjectMap } from '@/lib/api';
import { formatCount } from '@/lib/format';
import { isWildcard, subjectMatches, subjectsOverlap } from '@/lib/subjects';
import { useSearchQuery } from '@/lib/useSearchQuery';
import { cn } from '@/lib/utils';
import {
  buildTree,
  find,
  type SubjectNode,
  sortedChildren,
  totals,
} from './tree';

function useSubjects() {
  const { data: status } = useConnectionStatus();
  return useQuery({
    queryKey: ['subjects'],
    queryFn: natsApi.getSubjects,
    enabled: !!status?.connected,
    refetchInterval: 15_000,
    retry: false,
  });
}

function Tester() {
  const [pattern, setPattern] = useState('orders.*.created');
  const [subject, setSubject] = useState('orders.eu.created');
  const ok = pattern && subject ? subjectMatches(pattern, subject) : null;
  return (
    <section className="flex flex-wrap items-center gap-2 rounded-lg border bg-card p-3 text-sm">
      <span className="font-medium">Wildcard tester</span>
      <Input
        aria-label="Pattern"
        value={pattern}
        onChange={(e) => setPattern(e.target.value)}
        className="h-8 w-48 font-mono"
      />
      <span className="text-muted-foreground">matches</span>
      <Input
        aria-label="Subject"
        value={subject}
        onChange={(e) => setSubject(e.target.value)}
        className="h-8 w-48 font-mono"
      />
      {ok != null && (
        <output
          className={cn(
            'rounded-full px-2 py-0.5 text-xs font-medium',
            ok ? 'bg-success/10 text-success' : 'bg-muted text-muted-foreground'
          )}
        >
          {ok ? 'yes, it matches' : "no, it doesn't match"}
        </output>
      )}
    </section>
  );
}

function Badges({ n }: { n: SubjectNode }) {
  const t = totals(n);
  return (
    <span className="ml-auto flex gap-2 text-xs text-muted-foreground">
      {t.listeners > 0 && (
        <span
          className="inline-flex items-center gap-1"
          title="Subscriptions here and below"
        >
          <Ear className="size-3.5" aria-hidden="true" />
          {t.listeners} listening
        </span>
      )}
      {(t.stored > 0 || t.streams > 0) && (
        <span
          className="inline-flex items-center gap-1"
          title="Stored messages here and below"
        >
          <HardDrive className="size-3.5" aria-hidden="true" />
          {t.stored > 0 ? `${formatCount(t.stored)} stored` : 'stored'}
        </span>
      )}
    </span>
  );
}

function Branch({
  n,
  depth,
  open,
  toggle,
  selected,
  select,
  query,
}: {
  n: SubjectNode;
  depth: number;
  open: Set<string>;
  toggle: (path: string) => void;
  selected: string | null;
  select: (path: string) => void;
  query: string;
}) {
  const kids = sortedChildren(n).filter(
    (c) => !query || c.path.startsWith(query) || query.startsWith(`${c.path}.`)
  );
  // While searching, branches leading to matches open by themselves.
  const expanded =
    open.has(n.path) ||
    (query !== '' && (query === n.path || query.startsWith(`${n.path}.`)));
  return (
    <li>
      <div
        className={cn(
          'flex items-center gap-1 rounded px-1 py-0.5 text-sm hover:bg-muted/60',
          selected === n.path && 'bg-muted'
        )}
        style={{ paddingLeft: depth * 16 }}
      >
        {n.children.size > 0 ? (
          <button
            type="button"
            aria-label={`${expanded ? 'Collapse' : 'Expand'} ${n.path}`}
            aria-expanded={expanded}
            onClick={() => toggle(n.path)}
            className="rounded p-0.5 text-muted-foreground hover:text-foreground"
          >
            <ChevronRight
              className={cn(
                'size-4 transition-transform',
                expanded && 'rotate-90'
              )}
            />
          </button>
        ) : (
          <span className="w-5" />
        )}
        <button
          type="button"
          onClick={() => select(n.path)}
          className="font-mono"
          title={n.path}
        >
          {n.token}
        </button>
        <Badges n={n} />
      </div>
      {expanded && kids.length > 0 && (
        <ul>
          {kids.map((c) => (
            <Branch
              key={c.path}
              n={c}
              depth={depth + 1}
              {...{ open, toggle, selected, select, query }}
            />
          ))}
        </ul>
      )}
    </li>
  );
}

function Inspector({
  map,
  path,
  onClose,
}: {
  map: SubjectMap;
  path: string;
  onClose: () => void;
}) {
  const { data: js } = useJetStreamHealth();
  const node = find(buildTree(map), path);
  const literal = !isWildcard(path);
  // A literal subject also reaches every wildcard subscription that matches it.
  const viaWildcards = literal
    ? Object.entries(map.listening).filter(
        ([s]) => s !== path && isWildcard(s) && subjectMatches(s, path)
      )
    : [];
  const capturing = map.streams.filter((s) =>
    (s.subjects ?? []).some((p) =>
      literal ? subjectMatches(p, path) : subjectsOverlap(p, path)
    )
  );
  const consumers = (js?.consumers ?? []).filter(
    (c) =>
      capturing.some(
        (s) => s.name === c.stream && (s.account ?? '') === c.account
      ) &&
      (!c.filter_subjects?.length ||
        c.filter_subjects.some((f) => subjectsOverlap(f, path)))
  );
  const stored = node?.stored ?? [];
  return (
    <aside
      aria-label="Subject details"
      className="space-y-4 rounded-lg border bg-card p-4 text-sm lg:sticky lg:top-4"
    >
      <div className="flex items-center gap-2">
        <h2 className="break-all font-mono font-semibold">{path}</h2>
        <Button
          variant="ghost"
          size="icon"
          className="ml-auto shrink-0"
          aria-label="Close details"
          onClick={onClose}
        >
          <X className="size-4" />
        </Button>
      </div>
      <section>
        <h3 className="mb-1 font-semibold">
          Subscribers ({node?.listeners.length ?? 0})
        </h3>
        {(node?.listeners ?? []).length === 0 && (
          <p className="text-muted-foreground">
            Nobody subscribes to exactly this subject.
          </p>
        )}
        <ul className="space-y-1">
          {(node?.listeners ?? []).map((s) => (
            <li key={`${s.server}/${s.cid}/${s.queue ?? ''}`}>
              {s.name || `cid ${s.cid}`}{' '}
              <span className="text-muted-foreground">on {s.server}</span>
              {s.queue && (
                <span className="ml-1 rounded bg-muted px-1.5 text-xs">
                  queue {s.queue}
                </span>
              )}
              {s.account && (
                <span className="ml-1 text-xs text-muted-foreground">
                  {s.account}
                </span>
              )}
            </li>
          ))}
        </ul>
        {viaWildcards.length > 0 && (
          <p className="mt-2 text-muted-foreground">
            Also delivered to wildcard subscriptions:{' '}
            {viaWildcards
              .map(([s, subs]) => `${s} (${subs.length})`)
              .join(', ')}
          </p>
        )}
      </section>
      <section>
        <h3 className="mb-1 font-semibold">
          Streams that capture it ({capturing.length})
        </h3>
        {capturing.length === 0 ? (
          <p className="text-muted-foreground">
            No stream stores messages on this subject.
          </p>
        ) : (
          <ul className="space-y-1">
            {capturing.map((s) => {
              const here = stored.find((x) => x.stream === s)?.count;
              return (
                <li key={`${s.account}/${s.name}`}>
                  {s.name}{' '}
                  <span className="font-mono text-xs text-muted-foreground">
                    ({(s.subjects ?? []).join(', ')})
                  </span>
                  {here != null && (
                    <span className="ml-1">
                      · {formatCount(here)} stored here
                    </span>
                  )}
                  {s.truncated && (
                    <span className="ml-1 text-xs text-warning">
                      (subject list truncated at 10,000)
                    </span>
                  )}
                </li>
              );
            })}
          </ul>
        )}
      </section>
      <section>
        <h3 className="mb-1 font-semibold">
          Consumers that receive it ({consumers.length})
        </h3>
        {consumers.length === 0 ? (
          <p className="text-muted-foreground">None.</p>
        ) : (
          <ul className="space-y-1">
            {consumers.map((c) => (
              <li key={`${c.account}/${c.stream}/${c.name}`}>
                {c.name}{' '}
                <span className="text-muted-foreground">on {c.stream}</span>
              </li>
            ))}
          </ul>
        )}
      </section>
    </aside>
  );
}

export function SubjectsPage() {
  const { data, isLoading, error } = useSubjects();
  const [query, setQuery] = useSearchQuery();
  const [params, setParams] = useSearchParams();
  const [open, setOpen] = useState<Set<string>>(new Set());
  const selected = params.get('subject');
  const select = (path: string | null) =>
    setParams(
      (p) => {
        const next = new URLSearchParams(p);
        if (path) next.set('subject', path);
        else next.delete('subject');
        return next;
      },
      { replace: true }
    );
  const toggle = (path: string) =>
    setOpen((o) => {
      const next = new Set(o);
      if (next.has(path)) next.delete(path);
      else next.add(path);
      return next;
    });

  if (isLoading) return <LoadingState label="Loading subjects…" />;
  if (error || !data) {
    return (
      <div className="p-4 sm:p-6">
        <EmptyState
          tone="error"
          title="Couldn't load subjects"
          description={error?.message}
        />
      </div>
    );
  }
  const root = buildTree(data);
  const q = query.trim();
  const tops = sortedChildren(root).filter(
    (c) => !q || c.path.startsWith(q) || q.startsWith(`${c.path}.`)
  );
  const noCounts = data.source === 'system';

  return (
    <div className="space-y-3 p-4 sm:p-6">
      <Tester />
      <div className="flex flex-wrap items-center gap-3">
        <Input
          type="search"
          aria-label="Search subjects by prefix"
          placeholder="Prefix, e.g. orders.eu"
          value={query}
          onChange={(e) => setQuery(e.target.value)}
          className="h-8 w-64 font-mono"
        />
        <SourceBadge source={data.source} partial={data.partial} />
        <span className="text-xs text-muted-foreground">
          {data.hidden_inboxes > 0 &&
            `${formatCount(data.hidden_inboxes)} reply-inbox subscriptions hidden. `}
          {noCounts &&
            'Stored counts per subject need the owning account; showing stream patterns.'}
        </span>
      </div>
      <div className={cn('grid gap-3', selected && 'lg:grid-cols-[1fr_24rem]')}>
        <div className="rounded-lg border bg-card p-2">
          {tops.length === 0 ? (
            <EmptyState
              title={q ? `No subject starts with ${q}` : 'No subjects yet'}
              description={
                q
                  ? undefined
                  : 'Nothing subscribes and no stream stores messages.'
              }
            />
          ) : (
            <ul aria-label="Subjects">
              {tops.map((n) => (
                <Branch
                  key={n.path}
                  n={n}
                  depth={0}
                  open={open}
                  toggle={toggle}
                  selected={selected}
                  select={select}
                  query={q}
                />
              ))}
            </ul>
          )}
        </div>
        {selected && (
          <Inspector map={data} path={selected} onClose={() => select(null)} />
        )}
      </div>
    </div>
  );
}
