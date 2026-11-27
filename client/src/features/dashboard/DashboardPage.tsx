import { EmptyState, LoadingState } from '@/components/PageStates';
import {
  useAccountInfo,
  useConnectionStatus,
  useNATSInfo,
} from '@/features/connection/useNATS';
import { formatRTT } from '@/lib/utils';
import { OpenFindings } from './OpenFindings';
import { OverviewStats } from './OverviewStats';
import { PartialBanner } from './PartialBanner';
import { TopLaggingConsumers } from './TopLaggingConsumers';
import { TrafficCharts } from './TrafficCharts';

function Field({
  label,
  value,
  sub,
}: {
  label: string;
  value: React.ReactNode;
  sub?: React.ReactNode;
}) {
  return (
    <div className="min-w-0">
      <dt className="text-xs text-muted-foreground">{label}</dt>
      <dd
        className="mt-0.5 truncate font-mono text-sm"
        title={typeof value === 'string' ? value : undefined}
      >
        {value}
      </dd>
      {sub && <dd className="truncate text-xs text-muted-foreground">{sub}</dd>}
    </div>
  );
}

export function DashboardPage() {
  const { data: status } = useConnectionStatus();
  const {
    data: natsInfo,
    isLoading: infoLoading,
    error: infoError,
  } = useNATSInfo(status?.connected);
  const { data: accountInfo, isLoading: accountLoading } = useAccountInfo(
    status?.connected
  );

  if (infoLoading || accountLoading) {
    return <LoadingState label="Loading server information…" />;
  }

  if (infoError) {
    return (
      <div className="p-4 sm:p-6">
        <EmptyState
          tone="error"
          title="Couldn't reach the server"
          description={`${infoError.message}. Try disconnecting and connecting again.`}
        />
      </div>
    );
  }

  const info = accountInfo?.account_information;

  return (
    <div className="space-y-4 p-4 sm:p-6">
      <PartialBanner />
      <OverviewStats />
      <div className="grid gap-3 lg:grid-cols-2">
        <OpenFindings />
        <TopLaggingConsumers />
      </div>
      <TrafficCharts />

      {info && (
        <section
          className="rounded-lg border bg-card p-4"
          aria-labelledby="server-heading"
        >
          <div className="mb-3 flex items-center justify-between">
            <h2 id="server-heading" className="font-semibold">
              Server
            </h2>
            {natsInfo && (
              <span className="inline-flex items-center gap-1.5 rounded-full bg-success/10 px-2 py-0.5 text-xs font-medium text-success">
                <span
                  className="size-1.5 rounded-full bg-success"
                  aria-hidden="true"
                />
                {natsInfo.is_connected ? 'Connected' : 'Disconnected'}
              </span>
            )}
          </div>
          <dl className="grid grid-cols-2 gap-4 sm:grid-cols-3 lg:grid-cols-5">
            <Field
              label="Name"
              value={info.server_name}
              sub={`v${info.server_version}`}
            />
            <Field label="URL" value={info.connected_url} />
            <Field
              label="Client ID"
              value={info.client_id}
              sub={`${info.client_ip} → ${info.local_ip}`}
            />
            <Field
              label="RTT"
              value={formatRTT(info.rtt)}
              sub={`max payload ${info.max_payload}`}
            />
            <Field
              label="User"
              value={info.user || '—'}
              sub={
                info.header_supported
                  ? 'headers supported'
                  : 'no header support'
              }
            />
          </dl>
        </section>
      )}
    </div>
  );
}
