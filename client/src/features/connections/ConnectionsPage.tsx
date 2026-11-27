import { useVirtualizer } from '@tanstack/react-virtual';
import { Unplug, X } from 'lucide-react';
import { useEffect, useRef, useState } from 'react';
import { useSearchParams } from 'react-router-dom';
import { EmptyState, LoadingState } from '@/components/PageStates';
import { SourceBadge } from '@/components/SourceBadge';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Tabs, TabsList, TabsTrigger } from '@/components/ui/tabs';
import type { Conn, ConnState } from '@/lib/api';
import { formatBytes, formatCount } from '@/lib/format';
import { cn, formatRTT, formatTimestamp } from '@/lib/utils';
import { useConnection, useConnections } from './useConnections';

const FILTERS = ['account', 'user', 'server', 'name', 'ip', 'lang'] as const;
const STATES: ConnState[] = ['open', 'closed', 'slow'];
const SORTS: Record<string, string> = {
  cid: 'CID',
  pending: 'Pending bytes',
  msgs_from: 'Messages in',
  msgs_to: 'Messages out',
  bytes_from: 'Bytes in',
  bytes_to: 'Bytes out',
  subs: 'Subscriptions',
  idle: 'Longest idle',
  uptime: 'Longest connected',
  last: 'Last activity',
};
// Open connections holding this much unsent data are close to being cut off.
const AT_RISK_BYTES = 64 * 1024;
const ROW = 44;

/** One URL param, typed into locally and written after a short pause. */
function useUrlParam(key: string) {
  const [params, setParams] = useSearchParams();
  const [value, setValue] = useState(params.get(key) ?? '');
  useEffect(() => {
    const t = setTimeout(
      () =>
        setParams(
          (p) => {
            const next = new URLSearchParams(p);
            if (value) next.set(key, value);
            else next.delete(key);
            return next;
          },
          { replace: true }
        ),
      300
    );
    return () => clearTimeout(t);
  }, [key, value, setParams]);
  return [value, setValue] as const;
}

function FilterInput({ name }: { name: (typeof FILTERS)[number] }) {
  const [value, setValue] = useUrlParam(name);
  return (
    <Input
      aria-label={`Filter by ${name}`}
      placeholder={name === 'ip' ? 'IP' : name[0].toUpperCase() + name.slice(1)}
      value={value}
      onChange={(e) => setValue(e.target.value)}
      className="h-8 w-32"
    />
  );
}

const COLUMNS =
  'grid grid-cols-[4rem_4rem_minmax(8rem,1fr)_6rem_6rem_7rem_9rem_5rem_5rem_5rem_4rem_6rem_7rem_7rem] gap-x-3';

function Header({ closed }: { closed: boolean }) {
  const cells = [
    'CID',
    'Server',
    'Name',
    'Account',
    'User',
    'Lang',
    'Address',
    closed ? 'Stopped' : 'Uptime',
    closed ? 'Reason' : 'Idle',
    'RTT',
    'Subs',
    'Pending',
    'Msgs in / out',
    'Bytes in / out',
  ];
  return (
    <thead className="sticky top-0 z-10 grid bg-muted text-xs text-muted-foreground">
      <tr className={cn(COLUMNS, 'border-b px-4 py-2')}>
        {cells.map((c) => (
          <th key={c} scope="col" className="text-left font-medium">
            {c}
          </th>
        ))}
      </tr>
    </thead>
  );
}

function Row({
  c,
  closed,
  onOpen,
  selected,
  start,
}: {
  c: Conn;
  closed: boolean;
  onOpen: () => void;
  selected: boolean;
  start: number;
}) {
  return (
    <tr
      aria-selected={selected}
      style={{ transform: `translateY(${start}px)`, height: ROW }}
      className={cn(
        COLUMNS,
        'absolute inset-x-0 top-0 items-center border-b px-4 text-sm tabular-nums hover:bg-muted/50',
        selected && 'bg-muted'
      )}
    >
      <td>{c.cid}</td>
      <td>{c.server}</td>
      <td className="truncate">
        <button
          type="button"
          onClick={onOpen}
          title={c.name}
          className="max-w-full truncate font-medium hover:text-primary hover:underline"
        >
          {c.name || `cid ${c.cid}`}
        </button>
      </td>
      <td className="truncate">{c.account || '—'}</td>
      <td className="truncate">{c.authorized_user || '—'}</td>
      <td className="truncate text-muted-foreground">
        {c.lang ? `${c.lang} ${c.version ?? ''}` : '—'}
      </td>
      <td className="truncate font-mono text-xs">{`${c.ip}:${c.port}`}</td>
      <td>{closed ? (c.stop ? formatTimestamp(c.stop) : '—') : c.uptime}</td>
      <td className="truncate" title={closed ? c.reason : undefined}>
        {closed ? c.reason || '—' : c.idle}
      </td>
      <td>{c.rtt ? formatRTT(c.rtt) : '—'}</td>
      <td>{formatCount(c.subscriptions)}</td>
      <td
        className={cn(
          c.pending_bytes >= AT_RISK_BYTES && 'font-medium text-warning'
        )}
      >
        {formatBytes(c.pending_bytes)}
      </td>
      <td>
        {formatCount(c.in_msgs)} / {formatCount(c.out_msgs)}
      </td>
      <td>
        {formatBytes(c.in_bytes)} / {formatBytes(c.out_bytes)}
      </td>
    </tr>
  );
}

function Inspector({
  server,
  cid,
  onClose,
}: {
  server: string;
  cid: number;
  onClose: () => void;
}) {
  const { data: c, error, isLoading } = useConnection(server, cid);
  const fields: [string, string | number | undefined][] = c
    ? [
        ['Server', c.server],
        ['Account', c.account],
        ['User', c.authorized_user],
        ['Client', c.lang && `${c.lang} ${c.version ?? ''}`],
        ['Address', `${c.ip}:${c.port}`],
        ['Connected', formatTimestamp(c.start)],
        ['Last activity', formatTimestamp(c.last_activity)],
        ['RTT', c.rtt],
        ['TLS', c.tls_version],
        ['Pending', formatBytes(c.pending_bytes)],
        [
          'Messages in / out',
          `${formatCount(c.in_msgs)} / ${formatCount(c.out_msgs)}`,
        ],
        [
          'Closed',
          c.reason && `${c.reason}, ${c.stop ? formatTimestamp(c.stop) : ''}`,
        ],
      ]
    : [];
  return (
    <aside
      aria-label="Connection details"
      className="rounded-lg border bg-card p-4 lg:sticky lg:top-4"
    >
      <div className="mb-3 flex items-center gap-2">
        <h2 className="font-semibold">
          {c?.name || `Connection ${cid}`}{' '}
          <span className="font-normal text-muted-foreground">
            · {server}/{cid}
          </span>
        </h2>
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
      {isLoading && <LoadingState label="Loading connection…" />}
      {error && <p className="text-sm text-destructive">{error.message}</p>}
      {c && (
        <>
          <dl className="grid grid-cols-[auto_1fr] gap-x-4 gap-y-1 text-sm">
            {fields
              .filter(([, v]) => v)
              .map(([k, v]) => (
                <div key={k} className="contents">
                  <dt className="text-muted-foreground">{k}</dt>
                  <dd className="truncate">{v}</dd>
                </div>
              ))}
          </dl>
          <h3 className="mt-4 mb-2 text-sm font-semibold">
            Subscriptions ({c.subscriptions_list_detail?.length ?? 0})
          </h3>
          {c.subscriptions_list_detail?.length ? (
            <ul className="max-h-80 space-y-1 overflow-auto text-sm">
              {c.subscriptions_list_detail.map((s) => (
                <li key={s.sid} className="flex gap-2">
                  <code className="truncate">{s.subject}</code>
                  {s.qgroup && (
                    <span className="text-muted-foreground">
                      queue {s.qgroup}
                    </span>
                  )}
                  <span className="ml-auto tabular-nums text-muted-foreground">
                    {formatCount(s.msgs)} msgs
                  </span>
                </li>
              ))}
            </ul>
          ) : (
            <p className="text-sm text-muted-foreground">No subscriptions.</p>
          )}
        </>
      )}
    </aside>
  );
}

function AtRisk({ onOpen }: { onOpen: (c: Conn) => void }) {
  const { data } = useConnections({ state: 'open', sort: 'pending' });
  const risky = (data?.pages[0]?.connections ?? [])
    .filter((c) => c.pending_bytes >= AT_RISK_BYTES)
    .slice(0, 20);
  return (
    <section className="rounded-lg border bg-card p-4 text-sm">
      <h2 className="font-semibold">At risk now</h2>
      <p className="text-muted-foreground">
        Open connections with {formatBytes(AT_RISK_BYTES)} or more waiting to be
        sent; they get cut off as slow consumers if it keeps growing.
      </p>
      {risky.length === 0 ? (
        <p className="mt-2">None right now.</p>
      ) : (
        <ul className="mt-2 space-y-1">
          {risky.map((c) => (
            <li key={`${c.server}/${c.cid}`}>
              <button
                type="button"
                className="hover:underline"
                onClick={() => onOpen(c)}
              >
                {c.name || `cid ${c.cid}`} on {c.server}
              </button>{' '}
              ·{' '}
              <span className="font-medium text-warning">
                {formatBytes(c.pending_bytes)}
              </span>{' '}
              pending
            </li>
          ))}
        </ul>
      )}
    </section>
  );
}

export function ConnectionsPage() {
  const [params, setParams] = useSearchParams();
  const tab = (STATES.find((s) => s === params.get('tab')) ??
    'open') as ConnState;
  const closed = tab !== 'open';
  const sortParam = params.get('sort') ?? (closed ? 'stop' : 'cid');
  const sort =
    sortParam in SORTS || (closed && sortParam === 'stop') ? sortParam : 'cid';
  const query: Record<string, string> = { state: tab, sort };
  for (const f of FILTERS) {
    const v = params.get(f);
    if (v) query[f] = v;
  }
  const {
    data,
    isLoading,
    error,
    fetchNextPage,
    hasNextPage,
    isFetchingNextPage,
  } = useConnections(query);
  const rows = data?.pages.flatMap((p) => p.connections) ?? [];
  const first = data?.pages[0];

  const selServer = params.get('server_detail');
  const selCid = params.get('cid');
  const set = (changes: Record<string, string | null>) =>
    setParams(
      (p) => {
        const next = new URLSearchParams(p);
        for (const [k, v] of Object.entries(changes)) {
          if (v == null) next.delete(k);
          else next.set(k, v);
        }
        return next;
      },
      { replace: true }
    );
  const open = (c: Conn) =>
    set({ server_detail: c.server, cid: String(c.cid) });

  const scrollRef = useRef<HTMLDivElement>(null);
  const virtual = useVirtualizer({
    count: rows.length,
    getScrollElement: () => scrollRef.current,
    estimateSize: () => ROW,
    overscan: 10,
    initialRect: { width: 1200, height: 600 },
  });

  return (
    <div className="space-y-3 p-4 sm:p-6">
      <div className="flex flex-wrap items-center gap-3">
        <Tabs
          value={tab}
          defaultValue="open"
          onValueChange={(t) => set({ tab: t, sort: null })}
        >
          <TabsList>
            <TabsTrigger value="open">Open</TabsTrigger>
            <TabsTrigger value="closed">Closed</TabsTrigger>
            <TabsTrigger value="slow">Slow consumers</TabsTrigger>
          </TabsList>
        </Tabs>
        <label className="flex items-center gap-2 text-sm">
          <span className="text-muted-foreground">Sort</span>
          <select
            value={sort}
            onChange={(e) => set({ sort: e.target.value })}
            className="rounded-md border bg-card px-2 py-1"
          >
            {closed && <option value="stop">Recently closed</option>}
            {Object.entries(SORTS).map(([k, label]) => (
              <option key={k} value={k}>
                {label}
              </option>
            ))}
          </select>
        </label>
        {first && (
          <div className="ml-auto flex items-center gap-2 text-sm text-muted-foreground">
            <span>
              {formatCount(first.matched)}{' '}
              {Object.keys(query).length > 2 ? 'matching' : ''} of{' '}
              {formatCount(first.total)}
            </span>
            <SourceBadge {...first} />
          </div>
        )}
      </div>
      <fieldset className="flex flex-wrap gap-2">
        <legend className="sr-only">Filters</legend>
        {FILTERS.map((f) => (
          <FilterInput key={f} name={f} />
        ))}
      </fieldset>

      {tab === 'slow' && <AtRisk onOpen={open} />}

      <div
        className={cn(
          'grid gap-3',
          selServer && selCid && 'lg:grid-cols-[1fr_24rem]'
        )}
      >
        <div className="min-w-0 rounded-lg border bg-card">
          {isLoading ? (
            <LoadingState label="Loading connections…" />
          ) : error ? (
            <EmptyState
              tone="error"
              title="Couldn't load connections"
              description={error.message}
              className="m-4"
            />
          ) : rows.length === 0 ? (
            <EmptyState
              icon={<Unplug />}
              title={
                tab === 'slow'
                  ? 'No slow consumers disconnected recently'
                  : 'No connections match'
              }
              className="m-4"
            />
          ) : (
            <div ref={scrollRef} className="h-[60vh] overflow-auto">
              <table
                aria-label="Connections"
                aria-rowcount={rows.length + 1}
                className="grid min-w-[78rem]"
              >
                <Header closed={closed} />
                <tbody
                  className="relative grid"
                  style={{ height: virtual.getTotalSize() }}
                >
                  {virtual.getVirtualItems().map((v) => {
                    const c = rows[v.index];
                    return (
                      <Row
                        key={`${c.server}/${c.cid}`}
                        c={c}
                        closed={closed}
                        start={v.start}
                        onOpen={() => open(c)}
                        selected={
                          c.server === selServer && String(c.cid) === selCid
                        }
                      />
                    );
                  })}
                </tbody>
              </table>
            </div>
          )}
          {hasNextPage && (
            <div className="border-t p-2 text-center">
              <Button
                variant="ghost"
                size="sm"
                disabled={isFetchingNextPage}
                onClick={() => fetchNextPage()}
              >
                {isFetchingNextPage ? 'Loading…' : 'Load more'}
              </Button>
            </div>
          )}
        </div>
        {selServer && selCid && (
          <Inspector
            server={selServer}
            cid={Number(selCid)}
            onClose={() => set({ server_detail: null, cid: null })}
          />
        )}
      </div>
    </div>
  );
}
