import { keepPreviousData, useQuery } from '@tanstack/react-query';
import { natsApi } from '@/lib/api';

// Not under the 'nats' key: dropped on connect, like other per-server data.
export const overviewKeys = {
  jetstream: ['overview', 'jetstream'] as const,
  metrics: (series: string[], range: string) =>
    ['overview', 'metrics', ...series, range] as const,
};

export function useJetStreamSummary(enabled = true) {
  return useQuery({
    queryKey: overviewKeys.jetstream,
    queryFn: natsApi.getJetStream,
    enabled,
    refetchInterval: 15_000,
    retry: 1,
  });
}

/** Polls at the sampler's pace and keeps the last frame while refetching. */
export function useMetrics(series: string[], range: string, enabled = true) {
  return useQuery({
    queryKey: overviewKeys.metrics(series, range),
    queryFn: () => natsApi.getMetrics(series, range),
    enabled,
    refetchInterval: 5000,
    placeholderData: keepPreviousData,
    retry: false,
  });
}
