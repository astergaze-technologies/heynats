import { useState } from 'react';
import { Button } from '@/components/ui/button';
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog';
import { TagInput } from '@/components/ui/tag-input';
import type { StreamConfig } from '@/lib/api';

interface CreateStreamModalProps {
  isOpen: boolean;
  onClose: () => void;
  onSubmit: (config: Partial<StreamConfig>) => Promise<void>;
  isLoading?: boolean;
}

export function CreateStreamModal({
  isOpen,
  onClose,
  onSubmit,
  isLoading = false,
}: CreateStreamModalProps) {
  const [formData, setFormData] = useState({
    name: '',
    subjects: [] as string[],
    storage: 'file' as 'file' | 'memory',
    retention: 'limits' as 'limits' | 'interest' | 'workqueue',
    max_msgs: '-1',
    max_bytes: '-1',
    max_age: '-1',
    max_consumers: '-1',
    max_msgs_per_subject: '-1',
    max_msg_size: '-1',
    duplicate_window: '',
    compression: 'none' as 'none' | 's2',
    num_replicas: 1,
    discard: 'old' as 'old' | 'new',
    allow_direct: true,
    allow_msg_ttl: false,
  });

  const [errors, setErrors] = useState<Record<string, string>>({});

  const handleSubjectsChange = (subjects: string[]) => {
    setFormData({
      ...formData,
      subjects,
    });
    // Clear subject error if it exists
    if (errors.subjects) {
      const { subjects: _, ...restErrors } = errors;
      setErrors(restErrors);
    }
  };

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();

    // Validation
    const newErrors: Record<string, string> = {};

    if (!formData.name.trim()) {
      newErrors.name = 'Stream name is required';
    } else if (!/^[a-zA-Z0-9_-]+$/.test(formData.name)) {
      newErrors.name =
        'Stream name can only contain letters, numbers, underscores, and hyphens';
    }

    if (formData.subjects.length === 0) {
      newErrors.subjects = 'At least one subject is required';
    }

    if (Object.keys(newErrors).length > 0) {
      setErrors(newErrors);
      return;
    }

    // Prepare config
    const config: Partial<StreamConfig> = {
      name: formData.name,
      subjects: formData.subjects,
      storage: formData.storage,
      retention: formData.retention,
      discard: formData.discard,
      num_replicas: formData.num_replicas,
      allow_direct: formData.allow_direct,
      allow_msg_ttl: formData.allow_msg_ttl,
      compression: formData.compression,
    };

    // Add limits if specified
    if (formData.max_msgs) {
      config.max_msgs = Number.parseInt(formData.max_msgs) || -1;
    }
    if (formData.max_bytes) {
      config.max_bytes = Number.parseInt(formData.max_bytes) || -1;
    }
    if (formData.max_age) {
      config.max_age = Number.parseInt(formData.max_age) * 1000000000 || 0; // Convert seconds to nanoseconds
    }
    if (formData.max_consumers) {
      config.max_consumers = Number.parseInt(formData.max_consumers) || -1;
    }
    if (formData.max_msgs_per_subject) {
      config.max_msgs_per_subject =
        Number.parseInt(formData.max_msgs_per_subject, 10) || -1;
    }
    if (formData.max_msg_size) {
      config.max_msg_size = Number.parseInt(formData.max_msg_size, 10) || -1;
    }
    if (formData.duplicate_window) {
      config.duplicate_window =
        Number.parseInt(formData.duplicate_window, 10) * 1_000_000_000 || 0;
    }

    try {
      await onSubmit(config);
      handleClose();
    } catch (error) {
      console.error('Failed to create stream:', error);
      // Error toast is handled by the parent component's mutation
    }
  };

  const handleClose = () => {
    setFormData({
      name: '',
      subjects: [],
      storage: 'file',
      retention: 'limits',
      max_msgs: '',
      max_bytes: '',
      max_age: '',
      max_consumers: '',
      max_msgs_per_subject: '',
      max_msg_size: '',
      duplicate_window: '',
      compression: 'none',
      num_replicas: 1,
      discard: 'old',
      allow_direct: true,
      allow_msg_ttl: false,
    });
    setErrors({});
    onClose();
  };

  return (
    <Dialog
      open={isOpen}
      onOpenChange={(open) => !open && !isLoading && handleClose()}
    >
      <DialogContent className="max-h-[90vh] overflow-y-auto sm:max-w-2xl">
        <DialogHeader>
          <DialogTitle className="text-2xl">Create New Stream</DialogTitle>
          <DialogDescription>
            Configure a new JetStream stream
          </DialogDescription>
        </DialogHeader>

        <form onSubmit={handleSubmit} className="space-y-6">
          {/* Basic Configuration */}
          <div className="space-y-4">
            <h3 className="text-lg font-semibold text-foreground">
              Basic Configuration
            </h3>

            <div>
              <label
                htmlFor="stream-stream-name"
                className="block text-sm font-medium text-foreground/80 mb-2"
              >
                Stream Name *
              </label>
              <input
                id="stream-stream-name"
                type="text"
                value={formData.name}
                onChange={(e) =>
                  setFormData({ ...formData, name: e.target.value })
                }
                className={`w-full px-3 py-2 border rounded-md shadow-sm focus:outline-none focus:ring-2 focus:ring-ring ${
                  errors.name ? 'border-destructive/30' : 'border-border'
                }`}
                placeholder="my_stream"
                disabled={isLoading}
              />
              {errors.name && (
                <p className="text-destructive text-sm mt-1">{errors.name}</p>
              )}
            </div>

            <div>
              <label
                htmlFor="stream-subjects"
                className="block text-sm font-medium text-foreground/80 mb-2"
              >
                Subjects *
              </label>
              <TagInput
                id="stream-subjects"
                value={formData.subjects}
                onChange={handleSubjectsChange}
                placeholder="Enter subject (e.g., orders.created, orders.*)"
                disabled={isLoading}
                error={errors.subjects}
              />
              <p className="text-muted-foreground text-sm mt-1">
                Type subjects and press Enter to add them as chips. Use commas
                to separate multiple subjects at once.
              </p>
            </div>

            <div className="grid grid-cols-2 gap-4">
              <div>
                <label
                  htmlFor="stream-storage-type"
                  className="block text-sm font-medium text-foreground/80 mb-2"
                >
                  Storage Type
                </label>
                <select
                  id="stream-storage-type"
                  value={formData.storage}
                  onChange={(e) =>
                    setFormData({
                      ...formData,
                      storage: e.target.value as 'file' | 'memory',
                    })
                  }
                  className="w-full px-3 py-2 border border-border rounded-md shadow-sm focus:outline-none focus:ring-2 focus:ring-ring"
                  disabled={isLoading}
                >
                  <option value="file">File</option>
                  <option value="memory">Memory</option>
                </select>
              </div>

              <div>
                <label
                  htmlFor="stream-retention-policy"
                  className="block text-sm font-medium text-foreground/80 mb-2"
                >
                  Retention Policy
                </label>
                <select
                  id="stream-retention-policy"
                  value={formData.retention}
                  onChange={(e) =>
                    setFormData({
                      ...formData,
                      retention: e.target.value as
                        | 'limits'
                        | 'interest'
                        | 'workqueue',
                    })
                  }
                  className="w-full px-3 py-2 border border-border rounded-md shadow-sm focus:outline-none focus:ring-2 focus:ring-ring"
                  disabled={isLoading}
                >
                  <option value="limits">Limits</option>
                  <option value="interest">Interest</option>
                  <option value="workqueue">Work Queue</option>
                </select>
              </div>
            </div>
          </div>

          {/* Limits */}
          <div className="space-y-4">
            <h3 className="text-lg font-semibold text-foreground">Limits</h3>

            <div className="grid grid-cols-2 gap-4">
              <div>
                <label
                  htmlFor="stream-max-messages"
                  className="block text-sm font-medium text-foreground/80 mb-2"
                >
                  Max Messages
                </label>
                <input
                  id="stream-max-messages"
                  type="number"
                  value={formData.max_msgs}
                  onChange={(e) =>
                    setFormData({ ...formData, max_msgs: e.target.value })
                  }
                  className="w-full px-3 py-2 border border-border rounded-md shadow-sm focus:outline-none focus:ring-2 focus:ring-ring"
                  placeholder="No limit"
                  min="-1"
                  disabled={isLoading}
                />
              </div>

              <div>
                <label
                  htmlFor="stream-max-bytes"
                  className="block text-sm font-medium text-foreground/80 mb-2"
                >
                  Max Bytes
                </label>
                <input
                  id="stream-max-bytes"
                  type="number"
                  value={formData.max_bytes}
                  onChange={(e) =>
                    setFormData({ ...formData, max_bytes: e.target.value })
                  }
                  className="w-full px-3 py-2 border border-border rounded-md shadow-sm focus:outline-none focus:ring-2 focus:ring-ring"
                  placeholder="No limit"
                  min="-1"
                  disabled={isLoading}
                />
              </div>

              <div>
                <label
                  htmlFor="stream-max-age-seconds"
                  className="block text-sm font-medium text-foreground/80 mb-2"
                >
                  Max Age (seconds)
                </label>
                <input
                  id="stream-max-age-seconds"
                  type="number"
                  value={formData.max_age}
                  onChange={(e) =>
                    setFormData({ ...formData, max_age: e.target.value })
                  }
                  className="w-full px-3 py-2 border border-border rounded-md shadow-sm focus:outline-none focus:ring-2 focus:ring-ring"
                  placeholder="No limit"
                  min="-1"
                  disabled={isLoading}
                />
              </div>

              <div>
                <label
                  htmlFor="stream-max-consumers"
                  className="block text-sm font-medium text-foreground/80 mb-2"
                >
                  Max Consumers
                </label>
                <input
                  id="stream-max-consumers"
                  type="number"
                  value={formData.max_consumers}
                  onChange={(e) =>
                    setFormData({ ...formData, max_consumers: e.target.value })
                  }
                  className="w-full px-3 py-2 border border-border rounded-md shadow-sm focus:outline-none focus:ring-2 focus:ring-ring"
                  placeholder="No limit"
                  min="-1"
                  disabled={isLoading}
                />
              </div>
              <div>
                <label
                  htmlFor="stream-max-msgs-per-subject"
                  className="block text-sm font-medium text-foreground/80 mb-2"
                >
                  Max Messages per Subject
                </label>
                <input
                  id="stream-max-msgs-per-subject"
                  type="number"
                  value={formData.max_msgs_per_subject}
                  onChange={(e) =>
                    setFormData({
                      ...formData,
                      max_msgs_per_subject: e.target.value,
                    })
                  }
                  className="w-full px-3 py-2 border border-border rounded-md shadow-sm focus:outline-none focus:ring-2 focus:ring-ring"
                  placeholder="No limit"
                  min="-1"
                  disabled={isLoading}
                />
              </div>
              <div>
                <label
                  htmlFor="stream-max-msg-size"
                  className="block text-sm font-medium text-foreground/80 mb-2"
                >
                  Max Message Size (bytes)
                </label>
                <input
                  id="stream-max-msg-size"
                  type="number"
                  value={formData.max_msg_size}
                  onChange={(e) =>
                    setFormData({ ...formData, max_msg_size: e.target.value })
                  }
                  className="w-full px-3 py-2 border border-border rounded-md shadow-sm focus:outline-none focus:ring-2 focus:ring-ring"
                  placeholder="No limit"
                  min="-1"
                  disabled={isLoading}
                />
              </div>
            </div>
          </div>

          {/* Advanced Options */}
          <div className="space-y-4">
            <h3 className="text-lg font-semibold text-foreground">
              Advanced Options
            </h3>

            <div className="grid grid-cols-2 gap-4">
              <div>
                <label
                  htmlFor="stream-replicas"
                  className="block text-sm font-medium text-foreground/80 mb-2"
                >
                  Replicas
                </label>
                <input
                  id="stream-replicas"
                  type="number"
                  value={formData.num_replicas}
                  onChange={(e) =>
                    setFormData({
                      ...formData,
                      num_replicas: Number.parseInt(e.target.value) || 1,
                    })
                  }
                  className="w-full px-3 py-2 border border-border rounded-md shadow-sm focus:outline-none focus:ring-2 focus:ring-ring"
                  min="1"
                  max="5"
                  disabled={isLoading}
                />
              </div>

              <div>
                <label
                  htmlFor="stream-discard-policy"
                  className="block text-sm font-medium text-foreground/80 mb-2"
                >
                  Discard Policy
                </label>
                <select
                  id="stream-discard-policy"
                  value={formData.discard}
                  onChange={(e) =>
                    setFormData({
                      ...formData,
                      discard: e.target.value as 'old' | 'new',
                    })
                  }
                  className="w-full px-3 py-2 border border-border rounded-md shadow-sm focus:outline-none focus:ring-2 focus:ring-ring"
                  disabled={isLoading}
                >
                  <option value="old">Old</option>
                  <option value="new">New</option>
                </select>
              </div>
              <div>
                <label
                  htmlFor="stream-duplicate-window"
                  className="block text-sm font-medium text-foreground/80 mb-2"
                >
                  Duplicate Window (seconds)
                </label>
                <input
                  id="stream-duplicate-window"
                  type="number"
                  value={formData.duplicate_window}
                  onChange={(e) =>
                    setFormData({
                      ...formData,
                      duplicate_window: e.target.value,
                    })
                  }
                  className="w-full px-3 py-2 border border-border rounded-md shadow-sm focus:outline-none focus:ring-2 focus:ring-ring"
                  placeholder="Server default (2 min)"
                  min="0"
                  disabled={isLoading}
                />
              </div>

              <div>
                <label
                  htmlFor="stream-compression"
                  className="block text-sm font-medium text-foreground/80 mb-2"
                >
                  Compression
                </label>
                <select
                  id="stream-compression"
                  value={formData.compression}
                  onChange={(e) =>
                    setFormData({
                      ...formData,
                      compression: e.target.value as 'none' | 's2',
                    })
                  }
                  className="w-full px-3 py-2 border border-border rounded-md shadow-sm focus:outline-none focus:ring-2 focus:ring-ring"
                  disabled={isLoading}
                >
                  <option value="none">None</option>
                  <option value="s2">S2</option>
                </select>
              </div>
            </div>

            <div className="space-y-3">
              <label className="flex items-center">
                <input
                  type="checkbox"
                  checked={formData.allow_direct}
                  onChange={(e) =>
                    setFormData({ ...formData, allow_direct: e.target.checked })
                  }
                  className="rounded border-border text-primary focus:ring-ring"
                  disabled={isLoading}
                />
                <span className="ml-2 text-sm text-foreground/80">
                  Allow Direct Access
                </span>
              </label>

              <label className="flex items-center">
                <input
                  type="checkbox"
                  checked={formData.allow_msg_ttl}
                  onChange={(e) =>
                    setFormData({
                      ...formData,
                      allow_msg_ttl: e.target.checked,
                    })
                  }
                  className="rounded border-border text-primary focus:ring-ring"
                  disabled={isLoading}
                />
                <span className="ml-2 text-sm text-foreground/80">
                  Allow Message TTL
                </span>
              </label>
            </div>
          </div>

          {/* Actions */}
          <div className="flex justify-end gap-3 pt-6 border-t border-border">
            <Button
              type="button"
              variant="outline"
              onClick={handleClose}
              disabled={isLoading}
            >
              Cancel
            </Button>
            <Button type="submit" disabled={isLoading}>
              {isLoading ? 'Creating...' : 'Create Stream'}
            </Button>
          </div>
        </form>
      </DialogContent>
    </Dialog>
  );
}
