import type { DataSource } from '@/lib/api';
import { cn } from '@/lib/utils';

const LABELS: Record<DataSource, string> = {
  system: 'via system account',
  account: 'account scope',
};

/** Where a number comes from, and whether some servers didn't answer. */
export function SourceBadge({
  source,
  partial,
  servers,
  expected,
}: {
  source: DataSource;
  partial?: boolean;
  servers?: number;
  expected?: number;
}) {
  return (
    <span
      className={cn(
        'rounded-full border px-2 py-0.5 text-xs',
        partial
          ? 'border-warning/30 bg-warning/10 text-warning'
          : 'border-border text-muted-foreground'
      )}
    >
      {LABELS[source]}
      {partial && ` · partial: ${servers} of ${expected} servers`}
    </span>
  );
}
