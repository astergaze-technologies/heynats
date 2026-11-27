import { useQuery } from '@tanstack/react-query';
import { natsApi, type ServerKind } from '@/lib/api';

export const serverKeys = {
  list: ['servers'] as const,
  raw: (id: string, kind: ServerKind) => ['servers', id, 'raw', kind] as const,
};

export function useServers(enabled = true) {
  return useQuery({
    queryKey: serverKeys.list,
    queryFn: natsApi.getServers,
    enabled,
    refetchInterval: 5000,
    retry: false,
  });
}

export function useServerRaw(id: string | undefined, kind: ServerKind) {
  return useQuery({
    queryKey: serverKeys.raw(id ?? '', kind),
    queryFn: () => natsApi.getServerRaw(id as string, kind),
    enabled: !!id,
    refetchInterval: 10_000,
    retry: false,
  });
}
