import React, { useState, useEffect, useCallback } from 'react';
import { ThreadList } from './ThreadList';
import { ConversationView } from './ConversationView';
import { useWebSocket } from '../../hooks/useWebSocket';
import { Thread, Message, MessageEvent } from '../../types';

interface MessageCenterProps {
  websocketUrl: string;
  instanceId: string;
}

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

    // Add to messages
    setMessages((prev) => {
      const threadMessages = prev.get(msg.thread_id) || [];
      // Avoid duplicates
      if (!threadMessages.find((m) => m.id === msg.id)) {
        return new Map(prev).set(msg.thread_id, [...threadMessages, msg].sort((a, b) => a.message_seq - b.message_seq));
      }
      return prev;
    });

    // Update unread count if not the selected thread
    if (msg.thread_id !== selectedThreadId) {
      setUnreadCounts((prev) => {
        const current = prev.get(msg.thread_id) || 0;
        return new Map(prev).set(msg.thread_id, current + 1);
      });
    }

    // Auto-acknowledge
    acknowledge(msg.thread_id, msg.message_seq);
  }

  // Handle batch of messages
  function handleBatch(batch: any) {
    batch.messages.forEach((msgEvent: MessageEvent) => {
      handleNewMessage(msgEvent);
    });
  }

  // Select a thread
  const handleSelectThread = useCallback(
    (threadId: string) => {
      setSelectedThreadId(threadId);

      // Clear unread count for this thread
      setUnreadCounts((prev) => {
        const updated = new Map(prev);
        updated.delete(threadId);
        return updated;
      });

      // Subscribe to thread if connected
      if (isConnected) {
        const threadMessages = messages.get(threadId) || [];
        const lastSeq = threadMessages.length > 0
          ? Math.max(...threadMessages.map((m) => m.message_seq))
          : 0;
        subscribe(threadId, lastSeq);
      }
    },
    [isConnected, subscribe, messages]
  );

  // Send a message
  const handleSendMessage = useCallback(
    async (content: string, kind: string) => {
      if (!selectedThreadId) return;

      try {
        const response = await fetch('http://localhost:8080/api/messages', {
          method: 'POST',
          headers: {
            'Content-Type': 'application/json',
          },
          body: JSON.stringify({
            thread_id: selectedThreadId,
            from_type: 'human',
            from_id: 'user',
            to_type: 'ailang_instance',
            to_id: instanceId,
            kind,
            content,
          }),
        });

        if (!response.ok) {
          console.error('Failed to send message:', await response.text());
          return;
        }

        const sentMessage: Message = await response.json();

        // Add to messages (optimistic update - will also arrive via WebSocket)
        setMessages((prev) => {
          const threadMessages = prev.get(selectedThreadId) || [];
          // Avoid duplicates
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

  // Load threads from API on mount
  useEffect(() => {
    const fetchThreads = async () => {
      try {
        const response = await fetch('http://localhost:8080/api/threads');
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

  // Create a new thread
  const handleCreateThread = useCallback(async (title: string) => {
    try {
      const response = await fetch('http://localhost:8080/api/threads', {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
        },
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
      <div className="connection-status">
        {isConnected ? (
          <span className="status-connected">● Connected</span>
        ) : (
          <span className="status-disconnected">○ Disconnected</span>
        )}
      </div>

      <div className="center-layout">
        <div className="threads-panel">
          <ThreadList
            threads={threads}
            selectedThreadId={selectedThreadId}
            onSelectThread={handleSelectThread}
            onCreateThread={handleCreateThread}
            unreadCounts={unreadCounts}
          />
        </div>

        <div className="conversation-panel">
          {selectedThreadId ? (
            <ConversationView
              threadId={selectedThreadId}
              messages={selectedMessages}
              onSendMessage={handleSendMessage}
            />
          ) : (
            <div className="no-selection">
              <p>Select a thread to view messages</p>
            </div>
          )}
        </div>
      </div>

      <style>{`
        .message-center {
          display: flex;
          flex-direction: column;
          height: 100vh;
          font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif;
        }

        .connection-status {
          padding: 0.5rem 1rem;
          background: #f8f9fa;
          border-bottom: 1px solid #e0e0e0;
          font-size: 0.875rem;
        }

        .status-connected {
          color: #28a745;
          font-weight: 500;
        }

        .status-disconnected {
          color: #dc3545;
          font-weight: 500;
        }

        .center-layout {
          flex: 1;
          display: flex;
          overflow: hidden;
        }

        .threads-panel {
          width: 320px;
          border-right: 1px solid #e0e0e0;
        }

        .conversation-panel {
          flex: 1;
        }

        .no-selection {
          display: flex;
          align-items: center;
          justify-content: center;
          height: 100%;
          color: #666;
          font-size: 1.125rem;
        }
      `}</style>
    </div>
  );
};
