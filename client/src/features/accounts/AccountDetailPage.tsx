import { Link, useParams, useSearchParams } from 'react-router-dom';
import { UsageBar } from '@/components/charts/UsageBar';
import { JsonViewer } from '@/components/JsonViewer';
import { EmptyState, LoadingState } from '@/components/PageStates';
import { StatsCard } from '@/components/StatsCard';
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs';
import type { AccountDetail, AccountImport, AccountRow } from '@/lib/api';
import { formatBytes, formatCount } from '@/lib/format';
import { cn } from '@/lib/utils';
import { useAccount, useAccounts } from './useAccounts';

const TABS = ['usage', 'imports', 'mappings', 'jwt', 'raw'] as const;

// Stays on Imports / Exports, so following an import lands on the matching export.
const accountLink = (name: string) => (
  <Link
    to={`/dashboard/accounts/${encodeURIComponent(name)}?tab=imports`}
    className="text-primary hover:underline"
  >
    {name}
  </Link>
);

function Usage({ a }: { a: AccountRow }) {
  return (
    <div className="space-y-3">
      <div className="grid grid-cols-2 gap-3 lg:grid-cols-4">
        <StatsCard
          title="Connections"
          value={formatCount(a.connections)}
          subtitle={`${formatCount(a.leafnodes)} leaf nodes`}
        />
        <StatsCard title="Subscriptions" value={formatCount(a.subscriptions)} />
        <StatsCard
          title="Messages sent / received"
          value={`${formatCount(a.msgs_sent)} / ${formatCount(a.msgs_received)}`}
          subtitle={`${formatBytes(a.bytes_sent)} / ${formatBytes(a.bytes_received)}`}
        />
        <StatsCard
          title="Slow consumers"
          value={formatCount(a.slow_consumers)}
        />
      </div>
      <section className="space-y-3 rounded-lg border bg-card p-4">
        <h3 className="font-semibold">JetStream</h3>
        {!a.jetstream ? (
          <p className="text-sm text-muted-foreground">
            JetStream is not enabled for this account.
          </p>
        ) : (
          (
            [
              ['Storage', a.js_storage, a.js_max_storage],
              ['Memory', a.js_memory, a.js_max_memory],
            ] as const
          ).map(([label, used, max]) =>
            max > 0 ? (
              <UsageBar
                key={label}
                label={label}
                value={used}
                max={max}
                format={formatBytes}
              />
            ) : (
              <p key={label} className="text-sm">
                <span className="text-muted-foreground">{label}:</span>{' '}
                {formatBytes(used)}{' '}
                <span className="text-xs text-muted-foreground">
                  · no limit set
                </span>
              </p>
            )
          )
        )}
      </section>
    </div>
  );
}

// Every account gets these from the system account; they aren't from your config.
const builtIn = (i: AccountImport, systemAccount?: string) =>
  i.account === systemAccount;

function ImportsExports({
  d,
  systemAccount,
}: {
  d: AccountDetail;
  systemAccount?: string;
}) {
  const exports = d.detail.exports ?? [];
  const imports = [...(d.detail.imports ?? [])].sort(
    (x, y) =>
      Number(builtIn(x, systemAccount)) - Number(builtIn(y, systemAccount))
  );
  const th = 'px-4 py-2 text-left font-medium';
  return (
    <div className="space-y-5">
      <section className="space-y-2">
        <h3 className="text-sm font-semibold">Exports ({exports.length})</h3>
        {exports.length === 0 ? (
          <p className="text-sm text-muted-foreground">
            This account shares nothing with other accounts.
          </p>
        ) : (
          <div className="overflow-x-auto rounded-lg border bg-card">
            <table className="w-full text-sm">
              <thead className="border-b bg-muted/50 text-xs text-muted-foreground">
                <tr>
                  <th scope="col" className={th}>
                    Subject
                  </th>
                  <th scope="col" className={th}>
                    Type
                  </th>
                  <th scope="col" className={th}>
                    Who may import
                  </th>
                </tr>
              </thead>
              <tbody className="divide-y">
                {exports.map((e) => (
                  <tr key={`${e.type}:${e.subject}`}>
                    <td className="px-4 py-2 font-mono text-xs">{e.subject}</td>
                    <td className="px-4 py-2">{e.type}</td>
                    <td className="px-4 py-2">
                      {e.approved_accounts?.length
                        ? e.approved_accounts.map((a, i) => (
                            <span key={a}>
                              {i > 0 && ', '}
                              {accountLink(a)}
                            </span>
                          ))
                        : e.token_req
                          ? 'Accounts with an activation token'
                          : 'Any account'}
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        )}
      </section>
      <section className="space-y-2">
        <h3 className="text-sm font-semibold">Imports ({imports.length})</h3>
        {imports.length === 0 ? (
          <p className="text-sm text-muted-foreground">
            This account uses nothing from other accounts.
          </p>
        ) : (
          <div className="overflow-x-auto rounded-lg border bg-card">
            <table className="w-full text-sm">
              <thead className="border-b bg-muted/50 text-xs text-muted-foreground">
                <tr>
                  <th scope="col" className={th}>
                    Subject
                  </th>
                  <th scope="col" className={th}>
                    Type
                  </th>
                  <th scope="col" className={th}>
                    From account
                  </th>
                  <th scope="col" className={th}>
                    Seen here as
                  </th>
                </tr>
              </thead>
              <tbody className="divide-y">
                {imports.map((i) => {
                  const system = builtIn(i, systemAccount);
                  return (
                    <tr
                      key={`${i.account}:${i.subject}:${i.local_subject}`}
                      className={cn(system && 'text-muted-foreground')}
                    >
                      <td className="px-4 py-2 font-mono text-xs">
                        {i.subject}
                        {i.invalid && (
                          <span className="ml-2 rounded bg-destructive/10 px-1.5 text-destructive">
                            invalid
                          </span>
                        )}
                      </td>
                      <td className="px-4 py-2">{i.type}</td>
                      <td className="px-4 py-2">
                        {accountLink(i.account)}
                        {system && (
                          <span className="ml-2 rounded bg-muted px-1.5 py-0.5 text-xs">
                            built-in
                          </span>
                        )}
                      </td>
                      <td className="px-4 py-2 font-mono text-xs">
                        {i.local_subject || i.subject}
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

function Mappings({ d }: { d: AccountDetail }) {
  const rows = Object.entries(d.detail.mappings ?? {}).flatMap(
    ([from, dests]) => dests.map((to) => ({ from, ...to }))
  );
  if (rows.length === 0) {
    return (
      <p className="text-sm text-muted-foreground">
        No subject mappings in this account.
      </p>
    );
  }
  return (
    <div className="overflow-x-auto rounded-lg border bg-card">
      <table className="w-full text-sm">
        <thead className="border-b bg-muted/50 text-xs text-muted-foreground">
          <tr>
            <th scope="col" className="px-4 py-2 text-left font-medium">
              Published to
            </th>
            <th scope="col" className="px-4 py-2 text-left font-medium">
              Delivered to
            </th>
            <th scope="col" className="px-4 py-2 text-right font-medium">
              Weight
            </th>
            <th scope="col" className="px-4 py-2 text-left font-medium">
              Cluster
            </th>
          </tr>
        </thead>
        <tbody className="divide-y">
          {rows.map((r) => (
            <tr key={`${r.from}->${r.subject}:${r.cluster ?? ''}`}>
              <td className="px-4 py-2 font-mono text-xs">{r.from}</td>
              <td className="px-4 py-2 font-mono text-xs">{r.subject}</td>
              <td className="px-4 py-2 text-right tabular-nums">{r.weight}%</td>
              <td className="px-4 py-2">{r.cluster || 'any'}</td>
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  );
}

export function AccountDetailPage() {
  const { accountName = '' } = useParams();
  const [params, setParams] = useSearchParams();
  const tab = TABS.find((t) => t === params.get('tab')) ?? 'usage';
  const list = useAccounts();
  const row = list.data?.accounts.find((a) => a.name === accountName);
  const system = list.data?.source === 'system';
  const detail = useAccount(accountName, system && tab !== 'usage');
  const systemAccount = list.data?.accounts.find((a) => a.system)?.name;

  if (list.isLoading) return <LoadingState label="Loading account…" />;
  if (!row) {
    return (
      <div className="p-4 sm:p-6">
        <EmptyState
          tone="error"
          title={`No account named ${accountName}`}
          description={
            system
              ? 'Go back to the accounts list.'
              : 'Without the system account you can only see your own account.'
          }
        />
      </div>
    );
  }

  const needsSystem = (
    <EmptyState
      title="Needs the system account"
      description="Imports, exports, mappings and the JWT come from ACCOUNTZ, which only system account users can read."
    />
  );
  const withDetail = (render: (d: AccountDetail) => React.ReactNode) =>
    !system ? (
      needsSystem
    ) : detail.isLoading ? (
      <LoadingState label="Loading account details…" />
    ) : detail.error || !detail.data ? (
      <EmptyState
        tone="error"
        title="Couldn't load account details"
        description={detail.error?.message}
      />
    ) : (
      render(detail.data)
    );

  return (
    <div className="space-y-4 p-4 sm:p-6">
      <Tabs
        defaultValue="usage"
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
          <TabsTrigger value="usage">Usage</TabsTrigger>
          <TabsTrigger value="imports">Imports / Exports</TabsTrigger>
          <TabsTrigger value="mappings">Mappings</TabsTrigger>
          <TabsTrigger value="jwt">JWT</TabsTrigger>
          <TabsTrigger value="raw">Raw</TabsTrigger>
        </TabsList>
        <TabsContent value="usage" className="mt-4">
          <Usage a={row} />
        </TabsContent>
        <TabsContent value="imports" className="mt-4">
          {withDetail((d) => (
            <ImportsExports d={d} systemAccount={systemAccount} />
          ))}
        </TabsContent>
        <TabsContent value="mappings" className="mt-4">
          {withDetail((d) => (
            <Mappings d={d} />
          ))}
        </TabsContent>
        <TabsContent value="jwt" className="mt-4">
          {withDetail((d) =>
            d.detail.decoded_jwt ? (
              <div className="rounded-lg border bg-card p-4">
                <JsonViewer data={d.detail.decoded_jwt} />
              </div>
            ) : (
              <EmptyState
                title="No JWT for this account"
                description="The server isn't in operator mode: accounts come from its config file, so there are no signed claims to show."
              />
            )
          )}
        </TabsContent>
        <TabsContent value="raw" className="mt-4">
          {withDetail((d) => (
            <div className="rounded-lg border bg-card p-4">
              <JsonViewer data={d.detail} />
            </div>
          ))}
        </TabsContent>
      </Tabs>
    </div>
  );
}
