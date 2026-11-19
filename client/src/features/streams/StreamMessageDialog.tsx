import { useState } from 'react';
import { CodeDisplay, JsonViewer } from '@/components/JsonViewer';
import { Button } from '@/components/ui/button';
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogTitle,
} from '@/components/ui/dialog';
import type { StreamMessage } from '@/lib/api';
import { formatJSON, isValidJSON, parseJSON } from '@/lib/json';
import { formatTimestamp } from '@/lib/utils';

// Message Detail Modal
interface StreamMessageDialogProps {
  message: StreamMessage | null;
  isOpen: boolean;
  onClose: () => void;
}

export function StreamMessageDialog({
  message,
  isOpen,
  onClose,
}: StreamMessageDialogProps) {
  const [viewMode, setViewMode] = useState<'formatted' | 'raw'>('formatted');

  if (!message) return null;

  const isMessageJSON = isValidJSON(message.data);

  return (
    <Dialog open={isOpen} onOpenChange={(open) => !open && onClose()}>
      <DialogContent
        showCloseButton={false}
        className="flex max-h-[90vh] flex-col gap-0 overflow-hidden p-0 sm:max-w-4xl"
      >
        {/* Header */}
        <div className="flex items-center justify-between p-6 border-b border-border sticky top-0 bg-card z-10">
          <div>
            <DialogTitle className="text-2xl font-bold text-foreground">
              Message Details - Sequence #{message.sequence}
            </DialogTitle>
            <DialogDescription className="mt-1">
              {message.subject}
            </DialogDescription>
          </div>
          <button
            type="button"
            onClick={onClose}
            className="text-muted-foreground hover:text-foreground/80 transition-colors"
            aria-label="Close modal"
          >
            <svg
              className="w-6 h-6"
              fill="none"
              stroke="currentColor"
              viewBox="0 0 24 24"
              aria-hidden="true"
            >
              <path
                strokeLinecap="round"
                strokeLinejoin="round"
                strokeWidth={2}
                d="M6 18L18 6M6 6l12 12"
              />
            </svg>
          </button>
        </div>

        {/* Content */}
        <div className="flex-1 overflow-auto p-6 space-y-6">
          {/* Subject */}
          <div>
            <label
              htmlFor="detail-subject"
              className="block text-sm font-medium text-foreground/80 mb-2"
            >
              Subject
            </label>
            <div
              id="detail-subject"
              className="p-3 bg-muted rounded border border-border font-mono text-sm break-all"
            >
              {message.subject}
            </div>
          </div>

          {/* Data Payload */}
          <div>
            <div className="flex items-center justify-between mb-2">
              <label
                htmlFor="detail-data"
                className="block text-sm font-medium text-foreground/80"
              >
                Data Payload
                {isMessageJSON && (
                  <span className="ml-2 text-xs font-normal text-success">
                    (JSON)
                  </span>
                )}
              </label>
              {isMessageJSON && (
                <div className="flex gap-2">
                  <button
                    onClick={() => setViewMode('formatted')}
                    className={`px-3 py-1 rounded text-xs font-medium transition-colors ${
                      viewMode === 'formatted'
                        ? 'bg-primary/10 text-primary'
                        : 'bg-muted text-foreground/80 hover:bg-muted'
                    }`}
                    type="button"
                  >
                    Formatted
                  </button>
                  <button
                    onClick={() => setViewMode('raw')}
                    className={`px-3 py-1 rounded text-xs font-medium transition-colors ${
                      viewMode === 'raw'
                        ? 'bg-primary/10 text-primary'
                        : 'bg-muted text-foreground/80 hover:bg-muted'
                    }`}
                    type="button"
                  >
                    Raw
                  </button>
                </div>
              )}
            </div>

            <div id="detail-data">
              {viewMode === 'formatted' && isMessageJSON ? (
                <div className="p-4 bg-muted rounded border border-border overflow-auto max-h-96">
                  <JsonViewer data={parseJSON(message.data)} />
                </div>
              ) : (
                <CodeDisplay
                  code={isMessageJSON ? formatJSON(message.data) : message.data}
                  maxHeight="max-h-96"
                />
              )}
            </div>
          </div>

          {/* Metadata Grid */}
          <div className="grid grid-cols-3 gap-4">
            <div>
              <label
                htmlFor="detail-sequence"
                className="block text-sm font-medium text-foreground/80 mb-2"
              >
                Sequence
              </label>
              <div
                id="detail-sequence"
                className="p-3 bg-muted rounded border border-border text-sm font-mono"
              >
                {message.sequence}
              </div>
            </div>
            <div>
              <label
                htmlFor="detail-size"
                className="block text-sm font-medium text-foreground/80 mb-2"
              >
                Size
              </label>
              <div
                id="detail-size"
                className="p-3 bg-muted rounded border border-border text-sm font-mono"
              >
                {message.size} bytes
              </div>
            </div>
            <div>
              <label
                htmlFor="detail-timestamp"
                className="block text-sm font-medium text-foreground/80 mb-2"
              >
                Timestamp
              </label>
              <div
                id="detail-timestamp"
                className="p-3 bg-muted rounded border border-border text-sm font-mono"
              >
                {formatTimestamp(message.timestamp)}
              </div>
            </div>
          </div>

          {/* Headers */}
          {message.headers && Object.keys(message.headers).length > 0 && (
            <div>
              <label
                htmlFor="detail-headers"
                className="block text-sm font-medium text-foreground/80 mb-2"
              >
                Headers ({Object.keys(message.headers).length})
              </label>
              <div
                id="detail-headers"
                className="p-3 bg-muted rounded border border-border space-y-2 max-h-48 overflow-auto"
              >
                {Object.entries(message.headers).map(([key, value]) => (
                  <div
                    key={key}
                    className="text-sm border-b border-border pb-2 last:border-b-0"
                  >
                    <span className="font-medium text-foreground/80">
                      {key}
                    </span>
                    <span className="text-muted-foreground mx-2">:</span>
                    <span className="text-muted-foreground break-all">
                      {Array.isArray(value) ? value.join(', ') : String(value)}
                    </span>
                  </div>
                ))}
              </div>
            </div>
          )}
        </div>

        {/* Footer */}
        <div className="border-t border-border p-4 bg-muted flex justify-end">
          <Button variant="outline" onClick={onClose}>
            Close
          </Button>
        </div>
      </DialogContent>
    </Dialog>
  );
}
