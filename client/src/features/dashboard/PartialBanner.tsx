import { AlertTriangle } from 'lucide-react';
import {
  useCapabilities,
  useConnectionStatus,
} from '@/features/connection/useNATS';

/** Says so when some servers didn't answer, so low totals aren't misread. */
export function PartialBanner() {
  const { data: status } = useConnectionStatus();
  const { data: caps } = useCapabilities(status?.connected);
  if (!caps?.partial) return null;
  return (
    <p
      role="status"
      className="flex items-center gap-2 rounded-lg border border-warning/30 bg-warning/10 p-3 text-sm text-warning"
    >
      <AlertTriangle className="size-4 shrink-0" aria-hidden="true" />
      Only {caps.servers} of {caps.expected} servers answered. Totals on this
      page may be low until the rest are back.
    </p>
  );
}
