import { useQuery } from '@tanstack/react-query';
import { useConnectionStatus } from '@/features/connection/useNATS';
import { natsApi } from '@/lib/api';

export function useAccounts() {
  const { data: status } = useConnectionStatus();
  return useQuery({
    queryKey: ['accounts'],
    queryFn: natsApi.getAccounts,
    enabled: !!status?.connected,
    refetchInterval: 10_000,
    retry: false,
  });
}

export function useAccount(name: string, enabled = true) {
  return useQuery({
    queryKey: ['accounts', name],
    queryFn: () => natsApi.getAccount(name),
    enabled,
    retry: false,
  });
}
