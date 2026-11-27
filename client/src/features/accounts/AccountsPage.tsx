import { Link } from 'react-router-dom';
import { UsageBar } from '@/components/charts/UsageBar';
import { EmptyState, LoadingState } from '@/components/PageStates';
import { SourceBadge } from '@/components/SourceBadge';
import type { AccountRow } from '@/lib/api';
import { formatBytes, formatCount } from '@/lib/format';
import { useAccounts } from './useAccounts';

export function JetStreamUsage({ a }: { a: AccountRow }) {
  if (!a.jetstream) return <span className="text-muted-foreground">Off</span>;
  if (a.js_max_storage > 0) {
    return (
      <UsageBar
        label="Storage"
        value={a.js_storage}
        max={a.js_max_storage}
        format={formatBytes}
      />
    );
  }
  return (
    <span>
      {formatBytes(a.js_storage)}{' '}
      <span className="text-xs text-muted-foreground">· no limit set</span>
    </span>
  );
}

const num = 'px-4 py-2.5 text-right tabular-nums';

export function AccountsPage() {
  const { data, isLoading, error } = useAccounts();
  if (isLoading) return <LoadingState label="Loading accounts…" />;
  if (error || !data) {
    return (
      <div className="p-4 sm:p-6">
        <EmptyState
          tone="error"
          title="Couldn't load accounts"
          description={error?.message}
        />
      </div>
    );
  }
  return (
    <div className="space-y-3 p-4 sm:p-6">
      <div className="flex flex-wrap items-center gap-3 text-sm text-muted-foreground">
        <span>
          {data.accounts.length} account{data.accounts.length === 1 ? '' : 's'}
          {data.source === 'account' &&
            ' (yours only: connect with a system account user to see all)'}
        </span>
        <SourceBadge {...data} />
      </div>
      <div className="overflow-x-auto rounded-lg border bg-card">
        <table className="w-full text-sm">
          <thead className="border-b bg-muted/50 text-xs text-muted-foreground">
            <tr>
              <th scope="col" className="px-4 py-2.5 text-left font-medium">
                Account
              </th>
              {[
                'Connections',
                'Leaf nodes',
                'Subscriptions',
                'Msgs sent / received',
                'Bytes sent / received',
                'Slow consumers',
              ].map((h) => (
                <th
                  key={h}
                  scope="col"
                  className="px-4 py-2.5 text-right font-medium"
                >
                  {h}
                </th>
              ))}
              <th
                scope="col"
                className="w-56 px-4 py-2.5 text-left font-medium"
              >
                JetStream
              </th>
            </tr>
          </thead>
          <tbody className="divide-y">
            {data.accounts.map((a) => (
              <tr key={a.name} className="hover:bg-muted/50">
                <td className="px-4 py-2.5">
                  <Link
                    to={`/dashboard/accounts/${encodeURIComponent(a.name)}`}
                    className="font-medium text-foreground hover:text-primary hover:underline"
                  >
                    {a.name}
                  </Link>
                  {a.system && (
                    <span className="ml-2 rounded bg-muted px-1.5 py-0.5 text-xs text-muted-foreground">
                      system
                    </span>
                  )}
                </td>
                <td className={num}>{formatCount(a.connections)}</td>
                <td className={num}>{formatCount(a.leafnodes)}</td>
                <td className={num}>{formatCount(a.subscriptions)}</td>
                <td className={num}>
                  {formatCount(a.msgs_sent)} / {formatCount(a.msgs_received)}
                </td>
                <td className={num}>
                  {formatBytes(a.bytes_sent)} / {formatBytes(a.bytes_received)}
                </td>
                <td className={num}>{formatCount(a.slow_consumers)}</td>
                <td className="px-4 py-2.5">
                  <JetStreamUsage a={a} />
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>
    </div>
  );
}
