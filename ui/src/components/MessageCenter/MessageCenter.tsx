import React, { useState, useEffect, useCallback } from 'react';
import { ThreadList } from './ThreadList';
import { ConversationView } from './ConversationView';
import { useWebSocket } from '../../hooks/useWebSocket';
import { Thread, Message, MessageEvent } from '../../types';

interface MessageCenterProps {
  websocketUrl: string;
  instanceId: string;
}

// Connection status icon
const StatusIcon = ({ connected }: { connected: boolean }) => (
  <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2">
    {connected ? (
      <>
        <path d="M22 11.08V12a10 10 0 1 1-5.93-9.14" />
        <polyline points="22 4 12 14.01 9 11.01" />
      </>
    ) : (
      <>
        <circle cx="12" cy="12" r="10" />
        <line x1="15" y1="9" x2="9" y2="15" />
        <line x1="9" y1="9" x2="15" y2="15" />
      </>
    )}
  </svg>
);

export const MessageCenter: React.FC<MessageCenterProps> = ({
  websocketUrl,
  instanceId,
}) => {
  const [threads, setThreads] = useState<Thread[]>([]);
  const [selectedThreadId, setSelectedThreadId] = useState<string | null>(null);
  const [messages, setMessages] = useState<Map<string, Message[]>>(new Map());
  const [unreadCounts, setUnreadCounts] = useState<Map<string, number>>(new Map());

  // WebSocket connection
  const { isConnected, subscribe, acknowledge } = useWebSocket({
    url: websocketUrl,
    instanceId,
    onMessage: handleNewMessage,
    onBatch: handleBatch,
  });

  // Handle incoming message
  function handleNewMessage(msgEvent: MessageEvent) {
    const msg: Message = {
      id: msgEvent.id,
      thread_id: msgEvent.thread_id,
      message_seq: msgEvent.message_seq,
      created_at: msgEvent.created_at,
      from_type: msgEvent.from_type,
      from_id: msgEvent.from_id,
      to_type: msgEvent.to_type,
      to_id: msgEvent.to_id,
      kind: msgEvent.kind as any,
      subject: msgEvent.subject,
      content: msgEvent.content,
      metadata_json: msgEvent.metadata_json,
      delivery_state: 'visible',
      business_state: 'open',
    };

    setMessages((prev) => {
      const threadMessages = prev.get(msg.thread_id) || [];
      if (!threadMessages.find((m) => m.id === msg.id)) {
        return new Map(prev).set(
          msg.thread_id,
          [...threadMessages, msg].sort((a, b) => a.message_seq - b.message_seq)
        );
      }
      return prev;
    });

    if (msg.thread_id !== selectedThreadId) {
      setUnreadCounts((prev) => {
        const current = prev.get(msg.thread_id) || 0;
        return new Map(prev).set(msg.thread_id, current + 1);
      });
    }

    acknowledge(msg.thread_id, msg.message_seq);
  }

  function handleBatch(batch: any) {
    batch.messages.forEach((msgEvent: MessageEvent) => {
      handleNewMessage(msgEvent);
    });
  }

  const handleSelectThread = useCallback(
    (threadId: string) => {
      setSelectedThreadId(threadId);

      setUnreadCounts((prev) => {
        const updated = new Map(prev);
        updated.delete(threadId);
        return updated;
      });

      if (isConnected) {
        const threadMessages = messages.get(threadId) || [];
        const lastSeq =
          threadMessages.length > 0
            ? Math.max(...threadMessages.map((m) => m.message_seq))
            : 0;
        subscribe(threadId, lastSeq);
      }
    },
    [isConnected, subscribe, messages]
  );

  const handleSendMessage = useCallback(
    async (content: string, kind: string, workspace?: string) => {
      if (!selectedThreadId) return;

      // Build metadata if workspace is specified
      const metadata = workspace ? JSON.stringify({ workspace }) : undefined;

      try {
        const response = await fetch('/api/messages', {
          method: 'POST',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify({
            thread_id: selectedThreadId,
            from_type: 'human',
            from_id: 'user',
            to_type: 'ailang_instance',
            to_id: instanceId,
            kind,
            content,
            metadata_json: metadata,
          }),
        });

        if (!response.ok) {
          console.error('Failed to send message:', await response.text());
          return;
        }

        const sentMessage: Message = await response.json();

        setMessages((prev) => {
          const threadMessages = prev.get(selectedThreadId) || [];
          if (!threadMessages.find((m) => m.id === sentMessage.id)) {
            return new Map(prev).set(selectedThreadId, [...threadMessages, sentMessage]);
          }
          return prev;
        });
      } catch (error) {
        console.error('Error sending message:', error);
      }
    },
    [selectedThreadId, instanceId]
  );

  useEffect(() => {
    const fetchThreads = async () => {
      try {
        const response = await fetch('/api/threads');
        if (!response.ok) {
          console.error('Failed to fetch threads:', await response.text());
          return;
        }
        const data: Thread[] = await response.json();
        setThreads(data);
      } catch (error) {
        console.error('Error fetching threads:', error);
      }
    };

    fetchThreads();
  }, []);

  const handleCreateThread = useCallback(async (title: string) => {
    try {
      const response = await fetch('/api/threads', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          title,
          created_by_type: 'human',
          created_by_id: 'user',
        }),
      });

      if (!response.ok) {
        console.error('Failed to create thread:', await response.text());
        return;
      }

      const newThread: Thread = await response.json();
      setThreads((prev) => [newThread, ...prev]);
      setSelectedThreadId(newThread.id);
    } catch (error) {
      console.error('Error creating thread:', error);
    }
  }, []);

  const selectedMessages = selectedThreadId ? messages.get(selectedThreadId) || [] : [];

  return (
    <div className="message-center">
      {/* Status Bar */}
      <div className="status-bar">
        <div className={`status-indicator ${isConnected ? 'connected' : 'disconnected'}`}>
          <StatusIcon connected={isConnected} />
          <span>{isConnected ? 'Connected' : 'Disconnected'}</span>
        </div>
        <div className="status-meta">
          <span className="thread-count">{threads.length} threads</span>
        </div>
      </div>

      {/* Main Layout */}
      <div className="center-layout">
        <aside className="threads-panel">
          <ThreadList
            threads={threads}
            selectedThreadId={selectedThreadId}
            onSelectThread={handleSelectThread}
            onCreateThread={handleCreateThread}
            unreadCounts={unreadCounts}
          />
        </aside>

        <main className="conversation-panel">
          {selectedThreadId ? (
            <ConversationView
              threadId={selectedThreadId}
              messages={selectedMessages}
              onSendMessage={handleSendMessage}
            />
          ) : (
            <div className="empty-state">
              <div className="empty-icon">
                <svg width="48" height="48" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.5">
                  <path d="M21 15a2 2 0 0 1-2 2H7l-4 4V5a2 2 0 0 1 2-2h14a2 2 0 0 1 2 2z" />
                </svg>
              </div>
              <h3>Select a conversation</h3>
              <p>Choose a thread from the sidebar or create a new one to get started</p>
            </div>
          )}
        </main>
      </div>

      <style>{`
        .message-center {
          display: flex;
          flex-direction: column;
          height: 100%;
          background: var(--bg-base);
        }

        /* Status Bar */
        .status-bar {
          display: flex;
          align-items: center;
          justify-content: space-between;
          padding: var(--space-2) var(--space-4);
          background: var(--bg-surface);
          border-bottom: 1px solid var(--border-subtle);
        }

        .status-indicator {
          display: flex;
          align-items: center;
          gap: var(--space-2);
          font-size: var(--text-xs);
          font-weight: var(--font-medium);
        }

        .status-indicator.connected {
          color: var(--color-success);
        }

        .status-indicator.connected svg {
          filter: drop-shadow(0 0 4px var(--color-success));
        }

        .status-indicator.disconnected {
          color: var(--color-danger);
        }

        .status-indicator.disconnected svg {
          filter: drop-shadow(0 0 4px var(--color-danger));
        }

        .status-meta {
          display: flex;
          align-items: center;
          gap: var(--space-4);
        }

        .thread-count {
          font-size: var(--text-xs);
          color: var(--text-tertiary);
        }

        /* Layout */
        .center-layout {
          flex: 1;
          display: flex;
          overflow: hidden;
        }

        .threads-panel {
          width: 320px;
          border-right: 1px solid var(--border-subtle);
          flex-shrink: 0;
        }

        .conversation-panel {
          flex: 1;
          display: flex;
          flex-direction: column;
          overflow: hidden;
        }

        /* Empty State */
        .empty-state {
          display: flex;
          flex-direction: column;
          align-items: center;
          justify-content: center;
          height: 100%;
          padding: var(--space-8);
          text-align: center;
        }

        .empty-icon {
          display: flex;
          align-items: center;
          justify-content: center;
          width: 80px;
          height: 80px;
          background: var(--bg-surface);
          border-radius: var(--radius-xl);
          margin-bottom: var(--space-4);
          color: var(--text-tertiary);
        }

        .empty-state h3 {
          font-size: var(--text-lg);
          font-weight: var(--font-semibold);
          color: var(--text-primary);
          margin-bottom: var(--space-2);
        }

        .empty-state p {
          font-size: var(--text-sm);
          color: var(--text-tertiary);
          max-width: 300px;
        }

        /* Responsive */
        @media (max-width: 768px) {
          .threads-panel {
            width: 280px;
          }
        }

        @media (max-width: 640px) {
          .center-layout {
            flex-direction: column;
          }

          .threads-panel {
            width: 100%;
            height: 200px;
            border-right: none;
            border-bottom: 1px solid var(--border-subtle);
          }
        }
      `}</style>
    </div>
  );
};
