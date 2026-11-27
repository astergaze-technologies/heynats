import { useQuery } from '@tanstack/react-query';
import { useRef } from 'react';
import { useConnectionStatus } from '@/features/connection/useNATS';
import { natsApi, type Topology } from '@/lib/api';

const edgeKey = (e: { kind: string; from: string; to: string }) =>
  `${e.kind}|${e.from}|${e.to}`;

/** Polls the graph and turns counter deltas between polls into msgs/s per edge. */
export function useTopology(enabled: boolean) {
  const { data: status } = useConnectionStatus();
  const prev = useRef<{ at: number; topo: Topology } | null>(null);
  const rates = useRef(new Map<string, number>());
  const query = useQuery({
    queryKey: ['topology'],
    queryFn: async () => {
      const topo = await natsApi.getTopology();
      const at = Date.now();
      const last = prev.current;
      if (last) {
        const before = new Map(last.topo.edges.map((e) => [edgeKey(e), e]));
        const seconds = (at - last.at) / 1000;
        const next = new Map<string, number>();
        for (const e of topo.edges) {
          const b = before.get(edgeKey(e));
          const delta = b
            ? e.in_msgs + e.out_msgs - b.in_msgs - b.out_msgs
            : -1;
          // A drop means a link restarted: wait for the next poll.
          if (delta >= 0 && seconds > 0) next.set(edgeKey(e), delta / seconds);
        }
        rates.current = next;
      }
      prev.current = { at, topo };
      return topo;
    },
    enabled: enabled && !!status?.connected,
    refetchInterval: 5000,
    retry: false,
  });
  return {
    ...query,
    rateOf: (e: Topology['edges'][number]) => rates.current.get(edgeKey(e)),
  };
}
