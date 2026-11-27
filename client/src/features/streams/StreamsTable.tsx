import {
  ArrowDown,
  ArrowUp,
  ChevronsUpDown,
  Inbox,
  Trash2,
} from 'lucide-react';
import { useMemo, useState } from 'react';
import { Link } from 'react-router-dom';
import { Button } from '@/components/ui/button';
import {
  StatusBadge,
  streamStatus,
  useJetStreamHealth,
} from '@/features/health/jetstream';
import type { Stream, StreamHealth } from '@/lib/api';
import { cn, formatBytes, formatTimestamp } from '@/lib/utils';

type SortKey = 'name' | 'messages' | 'bytes' | 'consumers';

const sortValue: Record<SortKey, (s: Stream) => string | number> = {
  name: (s) => s.config?.name ?? '',
  messages: (s) => s.state?.messages ?? 0,
  bytes: (s) => s.state?.bytes ?? 0,
  consumers: (s) => s.state?.consumer_count ?? 0,
};

interface StreamsTableProps {
  streams: Stream[];
  onBrowse: (name: string) => void;
  onDelete: (name: string) => void;
}

export function StreamsTable({
  streams,
  onBrowse,
  onDelete,
}: StreamsTableProps) {
  // Replica health by stream name (the list shows the user's own account).
  const { data: js } = useJetStreamHealth();
  const health = new Map((js?.streams ?? []).map((s) => [s.name, s]));
  const [sort, setSort] = useState<{ key: SortKey; desc: boolean }>({
    key: 'name',
    desc: false,
  });

  const rows = useMemo(() => {
    const get = sortValue[sort.key];
    return [...streams].sort((a, b) => {
      const [x, y] = [get(a), get(b)];
      const cmp =
        typeof x === 'number' && typeof y === 'number'
          ? x - y
          : String(x).localeCompare(String(y));
      return sort.desc ? -cmp : cmp;
    });
  }, [streams, sort]);

  const header = (key: SortKey, label: string, align = 'text-left') => {
    const active = sort.key === key;
    const Icon = !active ? ChevronsUpDown : sort.desc ? ArrowDown : ArrowUp;
    return (
      <th
        scope="col"
        aria-sort={active ? (sort.desc ? 'descending' : 'ascending') : 'none'}
        className={cn('px-4 py-2.5 font-medium', align)}
      >
        <button
          type="button"
          onClick={() =>
            setSort((s) => ({
              key,
              desc: s.key === key ? !s.desc : key !== 'name',
            }))
          }
          className={cn(
            'inline-flex items-center gap-1 rounded hover:text-foreground',
            active && 'text-foreground'
          )}
        >
          {label}
          <Icon className="size-3.5" aria-hidden="true" />
        </button>
      </th>
    );
  };

  return (
    <div className="relative overflow-x-auto rounded-lg border bg-card">
      <table className="w-full text-sm">
        <thead className="border-b bg-muted/50 text-xs text-muted-foreground">
          <tr>
            {header('name', 'Name')}
            <th scope="col" className="px-4 py-2.5 text-left font-medium">
              Health
            </th>
            <th scope="col" className="px-4 py-2.5 text-left font-medium">
              Subjects
            </th>
            <th
              scope="col"
              className="hidden px-4 py-2.5 text-left font-medium xl:table-cell"
            >
              Storage
            </th>
            {header('messages', 'Messages', 'text-right')}
            {header('bytes', 'Size', 'text-right')}
            {header('consumers', 'Consumers', 'text-right')}
            <th
              scope="col"
              className="hidden px-4 py-2.5 text-left font-medium xl:table-cell"
            >
              Created
            </th>
            <th scope="col" className="sticky right-0 bg-muted px-4 py-2.5">
              <span className="sr-only">Actions</span>
            </th>
          </tr>
        </thead>
        <tbody className="divide-y">
          {rows.map((stream) => {
            const name = stream.config?.name ?? '';
            const subjects = stream.config?.subjects ?? [];
            return (
              <tr key={name} className="transition-colors hover:bg-muted/50">
                <td className="max-w-64 px-4 py-2.5">
                  <div className="flex items-center gap-2">
                    <Link
                      to={`/dashboard/streams/${encodeURIComponent(name)}`}
                      className="truncate font-medium text-foreground hover:text-primary hover:underline"
                      title={name}
                    >
                      {name}
                    </Link>
                    {stream.config?.retention && (
                      <span className="shrink-0 rounded bg-muted px-1.5 py-0.5 text-xs text-muted-foreground">
                        {stream.config.retention}
                      </span>
                    )}
                  </div>
                </td>
                <td className="px-4 py-2.5">
                  {health.get(name) ? (
                    <StatusBadge
                      status={streamStatus(health.get(name) as StreamHealth)}
                    />
                  ) : (
                    <span className="text-muted-foreground">—</span>
                  )}
                </td>
                <td className="max-w-56 px-4 py-2.5">
                  <div className="flex items-center gap-1 overflow-hidden">
                    {subjects.slice(0, 2).map((s) => (
                      <code
                        key={s}
                        className="truncate rounded bg-muted px-1.5 py-0.5 font-mono text-xs"
                        title={s}
                      >
                        {s}
                      </code>
                    ))}
                    {subjects.length > 2 && (
                      <span
                        className="shrink-0 text-xs text-muted-foreground"
                        title={subjects.slice(2).join(', ')}
                      >
                        +{subjects.length - 2}
                      </span>
                    )}
                  </div>
                </td>
                <td className="hidden px-4 py-2.5 text-muted-foreground xl:table-cell">
                  {stream.config?.storage}
                </td>
                <td className="whitespace-nowrap px-4 py-2.5 text-right">
                  {(stream.state?.messages ?? 0).toLocaleString()}
                </td>
                <td className="whitespace-nowrap px-4 py-2.5 text-right">
                  {formatBytes(stream.state?.bytes ?? 0)}
                </td>
                <td className="whitespace-nowrap px-4 py-2.5 text-right">
                  {stream.state?.consumer_count ?? 0}
                </td>
                <td className="hidden whitespace-nowrap px-4 py-2.5 text-muted-foreground xl:table-cell">
                  {stream.created ? formatTimestamp(stream.created) : '—'}
                </td>
                <td className="sticky right-0 bg-card px-2 py-1.5 shadow-[-8px_0_8px_-8px_rgb(0_0_0/0.15)]">
                  <div className="flex justify-end gap-1">
                    <Button
                      variant="ghost"
                      size="icon"
                      aria-label={`Browse messages in ${name}`}
                      title="Browse messages"
                      onClick={() => onBrowse(name)}
                    >
                      <Inbox />
                    </Button>
                    <Button
                      variant="ghost"
                      size="icon"
                      aria-label={`Delete ${name}`}
                      title="Delete stream"
                      onClick={() => onDelete(name)}
                      className="text-muted-foreground hover:bg-destructive/10 hover:text-destructive"
                    >
                      <Trash2 />
                    </Button>
                  </div>
                </td>
              </tr>
            );
          })}
        </tbody>
      </table>
    </div>
  );
}
