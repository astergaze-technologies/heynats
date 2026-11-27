import {
  ArrowDown,
  ArrowUp,
  Database,
  MessageSquare,
  Users,
  Zap,
} from 'lucide-react';
import type { ReactNode } from 'react';
import { Sparkline } from '@/components/charts/Sparkline';
import { cn, formatBytes } from '@/lib/utils';

interface StatsCardProps {
  title: string;
  value: string | number;
  subtitle?: string;
  icon?: ReactNode;
  /** Change vs a named period, e.g. { text: "+12% vs 1h ago", up: true, good: true }. */
  delta?: { text: string; up: boolean; good: boolean };
  trend?: (number | null)[];
}

export function StatsCard({
  title,
  value,
  subtitle,
  icon,
  delta,
  trend,
}: StatsCardProps) {
  return (
    <div className="rounded-lg border bg-card p-4">
      <div className="flex items-center justify-between gap-2 text-muted-foreground">
        <h3 className="truncate text-xs font-medium">{title}</h3>
        {icon && (
          <span className="shrink-0 [&_svg]:size-4" aria-hidden="true">
            {icon}
          </span>
        )}
      </div>
      <p className="mt-1.5 truncate text-2xl font-semibold tracking-tight">
        {typeof value === 'number' ? value.toLocaleString() : value}
      </p>
      {delta && (
        <p
          className={cn(
            'mt-0.5 flex items-center gap-1 text-xs',
            delta.good ? 'text-success' : 'text-destructive'
          )}
        >
          {delta.up ? (
            <ArrowUp className="size-3" aria-hidden="true" />
          ) : (
            <ArrowDown className="size-3" aria-hidden="true" />
          )}
          {delta.text}
        </p>
      )}
      {subtitle && (
        <p className="mt-0.5 truncate text-xs text-muted-foreground">
          {subtitle}
        </p>
      )}
      {trend && (
        <div className="mt-2">
          <Sparkline values={trend} />
        </div>
      )}
    </div>
  );
}

interface ConnectionStatsProps {
  totalConnections: number;
  activeConnections: number;
  totalMessages: number;
  totalBytes: number;
}

export function ConnectionStats({
  totalConnections,
  activeConnections,
  totalMessages,
  totalBytes,
}: ConnectionStatsProps) {
  const activePercentage =
    totalConnections > 0
      ? ((activeConnections / totalConnections) * 100).toFixed(1)
      : '0';

  return (
    <div className="grid grid-cols-2 gap-3 lg:grid-cols-4">
      <StatsCard
        title="Connections"
        value={totalConnections}
        subtitle="Clients on this server"
        icon={<Users />}
      />
      <StatsCard
        title="Active"
        value={activeConnections}
        subtitle={`${activePercentage}% of total`}
        icon={<Zap />}
      />
      <StatsCard
        title="Messages"
        value={totalMessages}
        subtitle="Processed by these clients"
        icon={<MessageSquare />}
      />
      <StatsCard
        title="Data transferred"
        value={formatBytes(totalBytes)}
        subtitle="In and out"
        icon={<Database />}
      />
    </div>
  );
}
