import { useQuery } from '@tanstack/react-query';
import { useConnectionStatus } from '@/features/connection/useNATS';
import {
  type ConsumerHealth,
  natsApi,
  type Peer,
  type StreamHealth,
} from '@/lib/api';
import { cn } from '@/lib/utils';

export function useJetStreamHealth() {
  const { data: status } = useConnectionStatus();
  return useQuery({
    queryKey: ['jetstream-health'],
    queryFn: natsApi.getJetStreamHealth,
    enabled: !!status?.connected,
    refetchInterval: 10_000,
    retry: false,
  });
}

/** The name metrics and findings use: account/stream/consumer for system users. */
export const consumerKey = (c: ConsumerHealth) =>
  [c.account, c.stream, c.name].filter(Boolean).join('/');

type Status = { label: string; tone: 'ok' | 'warning' | 'critical' };

export function streamStatus(s: StreamHealth): Status {
  const peers = s.replicas ?? [];
  if (s.clustered && !s.leader) return { label: 'No leader', tone: 'critical' };
  const offline = peers.filter((p) => p.offline).length;
  if (offline)
    return {
      label: `${offline} replica${offline > 1 ? 's' : ''} offline`,
      tone: 'warning',
    };
  if (peers.some((p) => !p.current))
    return { label: 'Catching up', tone: 'warning' };
  return { label: 'Healthy', tone: 'ok' };
}

const TONE = {
  ok: 'bg-success/10 text-success',
  warning: 'bg-warning/10 text-warning',
  critical: 'bg-destructive/10 text-destructive',
};

export function StatusBadge({ status }: { status: Status }) {
  return (
    <span
      className={cn(
        'rounded-full px-2 py-0.5 text-xs font-medium',
        TONE[status.tone]
      )}
    >
      {status.label}
    </span>
  );
}

/** One chip per replica: name plus offline / behind by N / current. */
export function Replicas({
  leader,
  peers,
}: {
  leader: string;
  peers: Peer[] | null;
}) {
  return (
    <span className="flex flex-wrap gap-1">
      {leader && (
        <span className="rounded bg-primary/10 px-1.5 py-0.5 text-xs text-primary">
          {leader} · leader
        </span>
      )}
      {(peers ?? []).map((p) => (
        <span
          key={p.name}
          className={cn(
            'rounded px-1.5 py-0.5 text-xs',
            p.offline
              ? TONE.critical
              : p.current
                ? 'bg-muted text-muted-foreground'
                : TONE.warning
          )}
        >
          {p.name} ·{' '}
          {p.offline
            ? 'offline'
            : p.current
              ? 'current'
              : `${p.lag ?? 0} behind`}
        </span>
      ))}
    </span>
  );
}
