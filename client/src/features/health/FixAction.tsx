import { useMutation, useQueryClient } from '@tanstack/react-query';
import { Play } from 'lucide-react';
import { useState } from 'react';
import { ConfirmDialog } from '@/components/ConfirmDialog';
import { Gate } from '@/components/Gate';
import { Button } from '@/components/ui/button';
import { type Finding, natsApi } from '@/lib/api';
import { showErrorToast, showSuccessToast } from '@/lib/error-utils';

type Fix = NonNullable<Finding['fixes']>[number];

/** One-click fix from a finding, after saying what it will do. */
export function FixAction({ fix }: { fix: Fix }) {
  const [open, setOpen] = useState(false);
  const queryClient = useQueryClient();
  const stream = fix.params?.stream ?? '';
  const consumer = fix.params?.consumer ?? '';
  const resume = useMutation({
    mutationFn: () => natsApi.resumeConsumer(stream, consumer),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['diagnostics'] });
      queryClient.invalidateQueries({ queryKey: ['jetstream-health'] });
      showSuccessToast(
        'Consumer resumed',
        `${consumer} on ${stream} is delivering again.`
      );
    },
    onError: (error) => showErrorToast('resume consumer', error),
  });
  if (fix.action !== 'consumer.resume') return null;
  return (
    <Gate feature="consumers.pause">
      <Button
        size="sm"
        variant="outline"
        disabled={resume.isPending}
        onClick={() => setOpen(true)}
      >
        <Play className="size-4" aria-hidden="true" />
        {resume.isPending ? 'Resuming…' : fix.label}
      </Button>
      <ConfirmDialog
        open={open}
        onOpenChange={setOpen}
        title={`Resume ${consumer}?`}
        confirmLabel="Resume"
        onConfirm={() => resume.mutate()}
        description={
          <p>
            Consumer <strong>{consumer}</strong> on stream{' '}
            <strong>{stream}</strong> starts delivering again right away,
            including every message that piled up while it was paused.
          </p>
        }
      />
    </Gate>
  );
}
