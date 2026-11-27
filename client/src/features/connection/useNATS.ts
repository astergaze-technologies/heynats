import {
  type QueryClient,
  useMutation,
  useQuery,
  useQueryClient,
} from '@tanstack/react-query';
import { type ConnectionCredentials, natsApi } from '@/lib/api';
import { showErrorToast, showSuccessToast } from '@/lib/error-utils';

// Query keys
export const queryKeys = {
  nats: {
    all: ['nats'] as const,
    status: () => [...queryKeys.nats.all, 'status'] as const,
    info: () => [...queryKeys.nats.all, 'info'] as const,
    account: () => [...queryKeys.nats.all, 'account'] as const,
    capabilities: () => [...queryKeys.nats.all, 'capabilities'] as const,
  },
};

// Connection status query
export function useConnectionStatus() {
  return useQuery({
    queryKey: queryKeys.nats.status(),
    queryFn: natsApi.getStatus,
    staleTime: 1000, // Consider data stale after 1 second
    refetchInterval: 5000, // Refetch every 5 seconds
    retry: false, // Don't retry on connection errors
  });
}

// NATS server info query (only when connected)
export function useNATSInfo(enabled = true) {
  return useQuery({
    queryKey: queryKeys.nats.info(),
    queryFn: natsApi.getInfo,
    enabled,
    staleTime: 2000,
    refetchInterval: 5000,
    retry: 1,
  });
}

// Account info query (only when connected)
export function useAccountInfo(enabled = true) {
  return useQuery({
    queryKey: queryKeys.nats.account(),
    queryFn: natsApi.getAccountInfo,
    enabled,
    staleTime: 2000,
    refetchInterval: 5000,
    retry: 1,
  });
}

// What this user and server support; refetched on connect with the other nats queries.
export function useCapabilities(enabled = true) {
  return useQuery({
    queryKey: queryKeys.nats.capabilities(),
    queryFn: natsApi.getCapabilities,
    enabled,
    staleTime: 30_000,
    refetchInterval: 30_000,
    retry: 1,
  });
}

// Streams/KV/etc. belong to the previous server; drop them when the connection changes.
const dropServerData = (queryClient: QueryClient) =>
  queryClient.removeQueries({
    predicate: (query) => query.queryKey[0] !== queryKeys.nats.all[0],
  });

// Connection mutation
export function useConnectToNATS() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (credentials: ConnectionCredentials) =>
      natsApi.connect(credentials),
    onSuccess: () => {
      dropServerData(queryClient);
      // Invalidate and refetch connection status
      queryClient.invalidateQueries({ queryKey: queryKeys.nats.status() });
      // Invalidate other NATS queries to trigger refetch when enabled
      queryClient.invalidateQueries({ queryKey: queryKeys.nats.all });
      showSuccessToast(
        'Connected to NATS server successfully!',
        'All NATS features are now available'
      );
    },
    // Errors are shown inline in the login form.
  });
}

// Disconnect mutation
export function useDisconnectFromNATS() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: natsApi.disconnect,
    onSuccess: () => {
      // Update connection status immediately
      queryClient.setQueryData(queryKeys.nats.status(), {
        connected: false,
        userInitiated: true,
      });
      dropServerData(queryClient);
      // Invalidate all NATS-related queries
      queryClient.invalidateQueries({ queryKey: queryKeys.nats.all });
      showSuccessToast(
        'Disconnected from NATS server',
        'Connection closed successfully'
      );
    },
    onError: (error) => {
      console.error('Disconnect failed:', error);
      showErrorToast(
        'disconnect from NATS server',
        error,
        'Failed to disconnect from NATS server'
      );
      // Even if disconnect API fails, update local state
      queryClient.setQueryData(queryKeys.nats.status(), {
        connected: false,
        userInitiated: true,
      });
      dropServerData(queryClient);
    },
  });
}
