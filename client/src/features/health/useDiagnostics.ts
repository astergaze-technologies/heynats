import { useQuery } from '@tanstack/react-query';
import { useConnectionStatus } from '@/features/connection/useNATS';
import { natsApi } from '@/lib/api';

export function useDiagnostics() {
  const { data: status } = useConnectionStatus();
  return useQuery({
    queryKey: ['diagnostics'],
    queryFn: natsApi.getDiagnostics,
    enabled: !!status?.connected,
    refetchInterval: 5000,
    retry: false,
  });
}
