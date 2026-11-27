import { BarChart3, HardDrive, Hash, Play, Square, Users } from 'lucide-react';
import { useEffect, useState } from 'react';
import { useNavigate, useParams } from 'react-router-dom';
import { EmptyState, LoadingState } from '@/components/PageStates';
import { StatsCard } from '@/components/StatsCard';
import { Button } from '@/components/ui/button';
import { appendCapped, useEventSources } from '@/lib/useEventSources';
import { formatBytes } from '@/lib/utils';
import { LiveMessageDialog } from './LiveMessageDialog';
import { LiveMessagesPanel } from './LiveMessagesPanel';
import { StreamConsumers } from './StreamConsumers';
import { SubjectList } from './SubjectList';
import type { MessageEvent, SubjectSubscription } from './types';
import { useStream } from './useStreams';

export function StreamDetailPage() {
  const { streamName } = useParams<{ streamName: string }>();
  const navigate = useNavigate();
  const { open: openSource, close: closeSource } = useEventSources();
  const [subscriptions, setSubscriptions] = useState<
    Record<string, SubjectSubscription>
  >({});
  const [selectedSubjects, setSelectedSubjects] = useState<Set<string>>(
    new Set()
  );
  const [selectedMessage, setSelectedMessage] = useState<MessageEvent | null>(
    null
  );
  const [isModalOpen, setIsModalOpen] = useState(false);

  // New state for improved UX
  const [activeTab, setActiveTab] = useState<string>('');

  // Fetch stream details
  const { data: stream, isLoading, error } = useStream(streamName);

  // Get subjects and active messages for virtualization (must be called before any early returns)
  const subjects = stream?.config?.subjects || [];

  // Note: Removed message virtualization to fix scrolling issues
  // For better UX, we'll render the last 200 messages directly

  // Initial scroll to bottom is handled by the auto-scroll effect

  const handleSubjectToggle = (subject: string) => {
    setSelectedSubjects((prev) => {
      const newSet = new Set(prev);
      if (newSet.has(subject)) {
        newSet.delete(subject);
      } else {
        newSet.add(subject);
      }
      return newSet;
    });
  };

  const startSubscription = (subject: string) => {
    if (subscriptions[subject]?.isActive) return;

    const url = `/api/nats/streams/${encodeURIComponent(streamName ?? '')}/subjects/${encodeURIComponent(subject)}/subscribe`;
    openSource(subject, url, {
      onOpen: () =>
        setSubscriptions((prev) => ({
          ...prev,
          [subject]: {
            subject,
            isActive: true,
            messages: prev[subject]?.messages || [],
          },
        })),
      onMessage: (messageData: MessageEvent) =>
        setSubscriptions((prev) => {
          const currentSub = prev[subject] || {
            subject,
            isActive: true,
            messages: [],
          };

          // Handle connection/status messages (messages with 'type' field)
          if (messageData.type) {
            return {
              ...prev,
              [subject]: {
                ...currentSub,
                connectionStatus: messageData.type as
                  | 'connected'
                  | 'disconnected'
                  | 'connecting',
                lastStatusUpdate: messageData.timestamp,
              },
            };
          }

          // Handle data messages (messages with 'data' field)
          if (messageData.data !== undefined) {
            return {
              ...prev,
              [subject]: {
                ...currentSub,
                messages: appendCapped(currentSub.messages, messageData),
              },
            };
          }

          // If neither type nor data, just update the subscription without adding to messages
          return prev;
        }),
      onError: () => {
        console.error('SSE error for subject:', subject);
        markStopped(subject);
      },
    });
  };

  const markStopped = (subject: string) =>
    setSubscriptions((prev) => ({
      ...prev,
      [subject]: { ...prev[subject], isActive: false },
    }));

  const stopSubscription = (subject: string) => {
    closeSource(subject);
    markStopped(subject);
  };

  const startSelectedSubscriptions = () => {
    selectedSubjects.forEach((subject) => {
      startSubscription(subject);
    });
  };

  const stopAllSubscriptions = () => {
    Object.keys(subscriptions).forEach((subject) => {
      stopSubscription(subject);
    });
  };

  const exportMessages = (subject: string) => {
    const messages = subscriptions[subject]?.messages || [];
    const dataStr = JSON.stringify(messages, null, 2);
    const dataBlob = new Blob([dataStr], { type: 'application/json' });
    const url = URL.createObjectURL(dataBlob);
    const link = document.createElement('a');
    link.href = url;
    link.download = `${subject}-messages.json`;
    link.click();
    URL.revokeObjectURL(url);
  };

  const openMessageModal = (message: MessageEvent) => {
    setSelectedMessage(message);
    setIsModalOpen(true);
  };

  const closeMessageModal = () => {
    setIsModalOpen(false);
    setSelectedMessage(null);
  };

  // Initialize active tab when subscriptions change
  useEffect(() => {
    const activeSubjects = Object.entries(subscriptions)
      .filter(([_, sub]) => sub.isActive && sub.messages.length > 0)
      .map(([subject]) => subject);

    if (activeSubjects.length > 0 && !activeTab) {
      setActiveTab(activeSubjects[0]); // Set first subject as active tab
    }
  }, [subscriptions, activeTab]);

  if (isLoading) {
    return <LoadingState label="Loading stream…" />;
  }

  if (error || !stream) {
    return (
      <div className="p-4 sm:p-6">
        <EmptyState
          tone="error"
          title="Couldn't load this stream"
          description="It may have been deleted, or the name in the URL is wrong."
          action={
            <Button
              variant="outline"
              onClick={() => navigate('/dashboard/streams')}
            >
              Back to streams
            </Button>
          }
        />
      </div>
    );
  }

  const hasActiveSubscriptions = Object.values(subscriptions).some(
    (sub) => sub.isActive
  );

  return (
    <div className="flex h-full flex-col p-4 sm:p-6">
      <div className="flex min-h-0 flex-1 flex-col gap-4 lg:flex-row">
        {/* Left Side - Main Content */}
        <div className="flex min-h-0 flex-1 flex-col overflow-hidden lg:w-1/2">
          <div className="mb-4 flex shrink-0 flex-wrap items-center justify-between gap-2">
            <p className="text-sm text-muted-foreground">
              Select subjects, then start a live subscription.
            </p>
            <div className="flex gap-2">
              <Button
                onClick={startSelectedSubscriptions}
                disabled={selectedSubjects.size === 0}
                size="sm"
              >
                <Play />
                Start ({selectedSubjects.size})
              </Button>
              <Button
                onClick={stopAllSubscriptions}
                disabled={!hasActiveSubscriptions}
                variant="outline"
                size="sm"
              >
                <Square />
                Stop all
              </Button>
            </div>
          </div>

          {/* Stream Stats */}
          <div className="grid grid-cols-2 lg:grid-cols-4 gap-3 mb-4 shrink-0">
            <StatsCard
              title="Messages"
              value={(stream.state?.messages || 0).toLocaleString()}
              icon={<BarChart3 className="w-4 h-4 text-primary" />}
            />
            <StatsCard
              title="Bytes"
              value={formatBytes(stream.state?.bytes || 0)}
              icon={<HardDrive className="w-4 h-4 text-success" />}
            />
            <StatsCard
              title="Consumers"
              value={(stream.state?.consumer_count || 0).toString()}
              icon={<Users className="w-4 h-4 text-primary" />}
            />
            <StatsCard
              title="Subjects"
              value={subjects.length.toString()}
              icon={<Hash className="w-4 h-4 text-warning" />}
            />
          </div>

          <StreamConsumers stream={streamName ?? ''} />

          {/* Stream Configuration */}
          <div className="bg-card rounded-lg border border-border p-4 mb-4 shrink-0">
            <h2 className="text-base font-semibold mb-3">
              Stream Configuration
            </h2>
            <div className="grid grid-cols-2 md:grid-cols-3 lg:grid-cols-6 gap-3 text-xs">
              <div>
                <span className="font-medium">Retention:</span>{' '}
                {stream.config?.retention || 'Unknown'}
              </div>
              <div>
                <span className="font-medium">Storage:</span>{' '}
                {stream.config?.storage || 'Unknown'}
              </div>
              <div>
                <span className="font-medium">Replicas:</span>{' '}
                {stream.config?.num_replicas || 0}
              </div>
              <div>
                <span className="font-medium">Max Messages:</span>{' '}
                {(stream.config?.max_msgs || 0).toLocaleString()}
              </div>
              <div>
                <span className="font-medium">Max Bytes:</span>{' '}
                {((stream.config?.max_bytes || 0) / 1024 / 1024).toFixed(2)} MB
              </div>
              <div>
                <span className="font-medium">Max Age:</span>{' '}
                {stream.config?.max_age
                  ? `${stream.config.max_age}s`
                  : 'No limit'}
              </div>
            </div>
          </div>

          <SubjectList
            subjects={subjects}
            subscriptions={subscriptions}
            selectedSubjects={selectedSubjects}
            handleSubjectToggle={handleSubjectToggle}
            startSubscription={startSubscription}
            stopSubscription={stopSubscription}
            exportMessages={exportMessages}
          />
        </div>

        <LiveMessagesPanel
          subscriptions={subscriptions}
          activeTab={activeTab}
          setActiveTab={setActiveTab}
          openMessageModal={openMessageModal}
        />

        <LiveMessageDialog
          message={selectedMessage}
          open={isModalOpen}
          onClose={closeMessageModal}
        />
      </div>
    </div>
  );
}
