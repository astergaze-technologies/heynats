import type { ReactNode } from 'react';
import {
  useCapabilities,
  useConnectionStatus,
} from '@/features/connection/useNATS';
import type { FeatureName } from '@/lib/api';

/** Dims children and says why when the server or user can't do `feature`. */
export function Gate({
  feature,
  children,
}: {
  feature: FeatureName;
  children: ReactNode;
}) {
  const { data: status } = useConnectionStatus();
  const { data: caps } = useCapabilities(status?.connected);
  const f = caps?.features[feature];
  // Unknown yet: show it; the server still refuses what it can't do.
  if (!f || f.ok) return children;
  return (
    <div>
      <div inert className="opacity-50">
        {children}
      </div>
      <p className="mt-1 text-xs text-muted-foreground">{f.reason}</p>
    </div>
  );
}
