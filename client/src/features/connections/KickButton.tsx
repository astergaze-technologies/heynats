import { Unplug } from 'lucide-react';
import { useState } from 'react';
import { ConfirmDialog } from '@/components/ConfirmDialog';
import { Gate } from '@/components/Gate';
import { Button } from '@/components/ui/button';
import { CopyCommand } from '@/features/health/FindingCard';
import type { ConnDetail } from '@/lib/api';
import { useKickConnection } from './useConnections';

/** Disconnect an open client, after saying exactly who and what happens. */
export function KickButton({ conn: c }: { conn: ConnDetail }) {
  const [open, setOpen] = useState(false);
  const kick = useKickConnection();
  const who = c.name || `connection ${c.cid}`;
  return (
    <Gate feature="connections.kick">
      <Button
        variant="outline"
        size="sm"
        disabled={kick.isPending}
        onClick={() => setOpen(true)}
      >
        <Unplug className="size-4" aria-hidden="true" />
        {kick.isPending ? 'Disconnecting…' : 'Disconnect client'}
      </Button>
      <ConfirmDialog
        open={open}
        onOpenChange={setOpen}
        title={`Disconnect ${who}?`}
        confirmLabel="Disconnect"
        destructive
        onConfirm={() => kick.mutate({ server: c.server, cid: c.cid })}
        description={
          <div className="space-y-3">
            <dl className="grid grid-cols-[auto_1fr] gap-x-3 gap-y-1 text-foreground">
              <dt className="text-muted-foreground">Client</dt>
              <dd>
                {who} (cid {c.cid}) at {c.ip}:{c.port}
              </dd>
              <dt className="text-muted-foreground">Account</dt>
              <dd>
                {c.account || '—'}
                {c.authorized_user && `, user ${c.authorized_user}`}
              </dd>
              <dt className="text-muted-foreground">Server</dt>
              <dd>{c.server}</dd>
            </dl>
            <p>
              The server closes this connection now, with reason "Kicked".
              Messages on their way to it are lost. Most clients reconnect by
              themselves within seconds, so this stops a stuck client but won't
              keep a misbehaving one out.
            </p>
            <div className="space-y-1">
              <p>Same thing from the CLI:</p>
              <CopyCommand
                cli={`nats server request kick ${c.cid} ${c.server_id}`}
              />
            </div>
          </div>
        }
      />
    </Gate>
  );
}
