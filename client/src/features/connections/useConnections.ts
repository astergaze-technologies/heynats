import {
  keepPreviousData,
  useInfiniteQuery,
  useQuery,
} from '@tanstack/react-query';
import { useConnectionStatus } from '@/features/connection/useNATS';
import { natsApi } from '@/lib/api';

export const PAGE_SIZE = 500;

/** Pages of the merged list; params are state, sort and the URL filters. */
export function useConnections(params: Record<string, string>) {
  const { data: status } = useConnectionStatus();
  return useInfiniteQuery({
    queryKey: ['connections', params],
    queryFn: ({ pageParam }) =>
      natsApi.getConnections({
        ...params,
        offset: String(pageParam),
        limit: String(PAGE_SIZE),
      }),
    initialPageParam: 0,
    getNextPageParam: (last, pages) => {
      const loaded = pages.reduce((n, p) => n + p.connections.length, 0);
      return loaded < last.matched && last.connections.length === PAGE_SIZE
        ? loaded
        : undefined;
    },
    enabled: !!status?.connected,
    refetchInterval: 10_000,
    placeholderData: keepPreviousData,
    retry: false,
  });
}

export function useConnection(server: string | null, cid: number | null) {
  return useQuery({
    queryKey: ['connections', 'detail', server, cid],
    queryFn: () => natsApi.getConnection(server as string, cid as number),
    enabled: !!server && cid != null,
    retry: false,
  });
}
