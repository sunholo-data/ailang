import React, { useState, useEffect, useCallback, useRef, useMemo } from 'react';
import { ThreadList } from './ThreadList';
import { ConversationView } from './ConversationView';
import { FilterBar } from '../../../components/badges/FilterBar';
import { useWebSocket } from '../../../hooks/useWebSocket';
import { Thread, Message, MessageEvent } from '../../../types';

interface RunningAgent {
  instance_id: string;
  pid: number;
  started_at: string;
}

interface MessageCenterProps {
  websocketUrl: string;
  instanceId: string;
  initialThreadId?: string | null;  // Thread to navigate to when mounting
  onThreadNavigated?: () => void;   // Callback when navigation is complete
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
  initialThreadId,
  onThreadNavigated,
}) => {
  const [threads, setThreads] = useState<Thread[]>([]);
  const [selectedThreadId, setSelectedThreadId] = useState<string | null>(null);
  const [messages, setMessages] = useState<Map<string, Message[]>>(new Map());
  const [unreadCounts, setUnreadCounts] = useState<Map<string, number>>(new Map());

  // Filter state with localStorage persistence
  const [selectedProvider, setSelectedProvider] = useState(
    () => localStorage.getItem('collab_filter_provider') || ''
  );
  const [selectedWorkspace, setSelectedWorkspace] = useState(
    () => localStorage.getItem('collab_filter_workspace') || ''
  );

  // Persist filter state to localStorage
  useEffect(() => {
    localStorage.setItem('collab_filter_provider', selectedProvider);
  }, [selectedProvider]);

  useEffect(() => {
    localStorage.setItem('collab_filter_workspace', selectedWorkspace);
  }, [selectedWorkspace]);

  // Extract unique providers and workspaces from threads
  const providers = useMemo(() =>
    [...new Set(threads.map(t => t.target_agent).filter(Boolean) as string[])],
    [threads]
  );

  const workspaces = useMemo(() =>
    [...new Set(threads.map(t => t.workspace).filter(Boolean) as string[])],
    [threads]
  );

  // Filter threads based on selected filters
  const filteredThreads = useMemo(() =>
    threads.filter(t => {
      if (selectedProvider && t.target_agent !== selectedProvider) return false;
      if (selectedWorkspace && t.workspace !== selectedWorkspace) return false;
      return true;
    }),
    [threads, selectedProvider, selectedWorkspace]
  );

  // Clear all filters
  const handleClearFilters = useCallback(() => {
    setSelectedProvider('');
    setSelectedWorkspace('');
  }, []);

  // Agent management
  const [runningAgents, setRunningAgents] = useState<RunningAgent[]>([]);
  const [showAgentModal, setShowAgentModal] = useState(false);
  const [newAgentId, setNewAgentId] = useState('');

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
        // Auto-select the first/most recent thread if none selected
        if (data.length > 0 && !selectedThreadId) {
          setSelectedThreadId(data[0].id);
        }
      } catch (error) {
        console.error('Error fetching threads:', error);
      }
    };

    fetchThreads();
  }, []);

  // Fetch messages when a thread is selected
  useEffect(() => {
    if (!selectedThreadId) return;

    // Capture threadId to avoid stale closure
    const threadId = selectedThreadId;

    const fetchMessages = async () => {
      try {
        const response = await fetch(`/api/messages?thread_id=${threadId}`);
        if (!response.ok) {
          console.error('Failed to fetch messages:', await response.text());
          return;
        }
        const data: Message[] = await response.json();

        // Always update messages for this thread (even if empty)
        setMessages((prev) => {
          const existing = prev.get(threadId) || [];
          const merged = data ? [...data] : [];

          // Merge with any messages that arrived via WebSocket
          for (const msg of existing) {
            if (!merged.find((m) => m.id === msg.id)) {
              merged.push(msg);
            }
          }
          // Sort by sequence number
          merged.sort((a, b) => a.message_seq - b.message_seq);
          return new Map(prev).set(threadId, merged);
        });
      } catch (error) {
        console.error('Error fetching messages:', error);
      }
    };

    fetchMessages();
  }, [selectedThreadId]);

  // Track which initialThreadId we've already navigated to
  const navigatedToRef = useRef<string | null>(null);

  // Handle external navigation to a specific thread (only when initialThreadId changes)
  useEffect(() => {
    // Only navigate if initialThreadId changed and we haven't already navigated to it
    if (initialThreadId && initialThreadId !== navigatedToRef.current && threads.length > 0) {
      const threadExists = threads.some(t => t.id === initialThreadId);
      if (threadExists) {
        navigatedToRef.current = initialThreadId;
        setSelectedThreadId(initialThreadId);
        // Clear unread count for this thread
        setUnreadCounts((prev) => {
          const updated = new Map(prev);
          updated.delete(initialThreadId);
          return updated;
        });
      }
      // Notify parent that navigation is complete
      if (onThreadNavigated) {
        onThreadNavigated();
      }
    }
  }, [initialThreadId, threads, onThreadNavigated]);

  const handleCreateThread = useCallback(async (title: string) => {
    try {
      const response = await fetch('/api/threads', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          title,
          created_by_type: 'human',
          created_by_id: 'user',
          target_agent: instanceId, // Associate thread with current target agent
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
  }, [instanceId]);

  // Fetch running agents
  const fetchAgents = useCallback(async () => {
    try {
      const response = await fetch('/api/agents');
      if (!response.ok) {
        console.error('Failed to fetch agents:', await response.text());
        return;
      }
      const data = await response.json();
      setRunningAgents(data.running || []);
    } catch (error) {
      console.error('Error fetching agents:', error);
    }
  }, []);

  // Fetch agents on mount and periodically
  useEffect(() => {
    fetchAgents();
    const interval = setInterval(fetchAgents, 5000); // Refresh every 5s
    return () => clearInterval(interval);
  }, [fetchAgents]);

  // Launch a new agent
  const handleLaunchAgent = useCallback(async () => {
    if (!newAgentId.trim()) return;

    try {
      const response = await fetch('/api/agents', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ instance_id: newAgentId.trim() }),
      });

      if (!response.ok) {
        const error = await response.text();
        console.error('Failed to launch agent:', error);
        alert(`Failed to launch agent: ${error}`);
        return;
      }

      const agent: RunningAgent = await response.json();
      setRunningAgents((prev) => [...prev, agent]);
      setNewAgentId('');
      setShowAgentModal(false);
    } catch (error) {
      console.error('Error launching agent:', error);
    }
  }, [newAgentId]);

  // Stop an agent
  const handleStopAgent = useCallback(async (agentId: string) => {
    try {
      const response = await fetch(`/api/agents/${agentId}`, {
        method: 'DELETE',
      });

      if (!response.ok) {
        console.error('Failed to stop agent:', await response.text());
        return;
      }

      setRunningAgents((prev) => prev.filter((a) => a.instance_id !== agentId));
    } catch (error) {
      console.error('Error stopping agent:', error);
    }
  }, []);

  // Update thread workspace
  const handleWorkspaceChange = useCallback(async (workspace: string) => {
    if (!selectedThreadId) return;

    try {
      const response = await fetch(`/api/threads/${selectedThreadId}`, {
        method: 'PUT',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ workspace }),
      });

      if (!response.ok) {
        console.error('Failed to update workspace:', await response.text());
        return;
      }

      // Update local thread state
      const updatedThread: Thread = await response.json();
      setThreads((prev) => prev.map((t) => (t.id === selectedThreadId ? updatedThread : t)));
    } catch (error) {
      console.error('Error updating workspace:', error);
    }
  }, [selectedThreadId]);

  // Delete a thread
  const handleDeleteThread = useCallback(async (threadId: string) => {
    try {
      const response = await fetch(`/api/threads/${threadId}`, {
        method: 'DELETE',
      });

      if (!response.ok) {
        console.error('Failed to delete thread:', await response.text());
        return;
      }

      // Remove from local state
      setThreads((prev) => prev.filter((t) => t.id !== threadId));

      // Clear messages for this thread
      setMessages((prev) => {
        const updated = new Map(prev);
        updated.delete(threadId);
        return updated;
      });

      // Clear unread count
      setUnreadCounts((prev) => {
        const updated = new Map(prev);
        updated.delete(threadId);
        return updated;
      });

      // If deleted thread was selected, clear selection
      if (selectedThreadId === threadId) {
        setSelectedThreadId(null);
      }
    } catch (error) {
      console.error('Error deleting thread:', error);
    }
  }, [selectedThreadId]);

  // Rename a thread
  const handleRenameThread = useCallback(async (threadId: string, newTitle: string) => {
    try {
      const response = await fetch(`/api/threads/${threadId}`, {
        method: 'PUT',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ title: newTitle }),
      });

      if (!response.ok) {
        console.error('Failed to rename thread:', await response.text());
        return;
      }

      // Update local thread state
      const updatedThread: Thread = await response.json();
      setThreads((prev) => prev.map((t) => (t.id === threadId ? updatedThread : t)));
    } catch (error) {
      console.error('Error renaming thread:', error);
    }
  }, []);

  // Approve an approval request
  const handleApproveRequest = useCallback(async (approvalId: string, notes: string) => {
    try {
      const response = await fetch(`/api/approvals/${approvalId}/approve`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          reviewed_by: 'user',
          review_notes: notes,
        }),
      });

      if (!response.ok) {
        const error = await response.text();
        console.error('Failed to approve request:', error);
        alert(`Failed to approve: ${error}`);
        return;
      }

      console.log('Approval approved successfully');
    } catch (error) {
      console.error('Error approving request:', error);
    }
  }, []);

  // Reject an approval request
  const handleRejectRequest = useCallback(async (approvalId: string, notes: string) => {
    try {
      const response = await fetch(`/api/approvals/${approvalId}/reject`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          reviewed_by: 'user',
          review_notes: notes,
        }),
      });

      if (!response.ok) {
        const error = await response.text();
        console.error('Failed to reject request:', error);
        alert(`Failed to reject: ${error}`);
        return;
      }

      console.log('Approval rejected successfully');
    } catch (error) {
      console.error('Error rejecting request:', error);
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
          <span className="thread-count">
            {filteredThreads.length === threads.length
              ? `${threads.length} threads`
              : `${filteredThreads.length}/${threads.length} threads`}
          </span>
          <span className="agent-count">{runningAgents.length} agents</span>
          <button className="launch-agent-btn" onClick={() => setShowAgentModal(true)}>
            + Agent
          </button>
        </div>
      </div>

      {/* Running Agents Bar */}
      {runningAgents.length > 0 && (
        <div className="agents-bar">
          {runningAgents.map((agent) => (
            <div key={agent.instance_id} className="agent-chip">
              <span className="agent-pulse" />
              <span className="agent-name">{agent.instance_id}</span>
              <span className="agent-pid">PID {agent.pid}</span>
              <button
                className="agent-stop-btn"
                onClick={() => handleStopAgent(agent.instance_id)}
                title="Stop agent"
              >
                ×
              </button>
            </div>
          ))}
        </div>
      )}

      {/* Launch Agent Modal */}
      {showAgentModal && (
        <div className="modal-overlay" onClick={() => setShowAgentModal(false)}>
          <div className="modal-content" onClick={(e) => e.stopPropagation()}>
            <h3>Launch New Agent</h3>
            <input
              type="text"
              value={newAgentId}
              onChange={(e) => setNewAgentId(e.target.value)}
              placeholder="Enter instance ID (e.g., agent-2)"
              autoFocus
              onKeyDown={(e) => {
                if (e.key === 'Enter') handleLaunchAgent();
                if (e.key === 'Escape') setShowAgentModal(false);
              }}
            />
            <div className="modal-actions">
              <button className="cancel-btn" onClick={() => setShowAgentModal(false)}>
                Cancel
              </button>
              <button className="launch-btn" onClick={handleLaunchAgent}>
                Launch
              </button>
            </div>
          </div>
        </div>
      )}

      {/* Main Layout */}
      <div className="center-layout">
        <aside className="threads-panel">
          {/* FilterBar - only show if there are threads to filter */}
          {threads.length > 0 && (providers.length > 0 || workspaces.length > 0) && (
            <FilterBar
              providers={providers}
              workspaces={workspaces}
              selectedProvider={selectedProvider}
              selectedWorkspace={selectedWorkspace}
              onProviderChange={setSelectedProvider}
              onWorkspaceChange={setSelectedWorkspace}
              onClearFilters={handleClearFilters}
            />
          )}
          <ThreadList
            threads={filteredThreads}
            selectedThreadId={selectedThreadId}
            onSelectThread={handleSelectThread}
            onCreateThread={handleCreateThread}
            onDeleteThread={handleDeleteThread}
            onRenameThread={handleRenameThread}
            unreadCounts={unreadCounts}
          />
        </aside>

        <main className="conversation-panel">
          {selectedThreadId ? (
            <ConversationView
              thread={threads.find(t => t.id === selectedThreadId)}
              messages={selectedMessages}
              onSendMessage={handleSendMessage}
              onWorkspaceChange={handleWorkspaceChange}
              onApproveRequest={handleApproveRequest}
              onRejectRequest={handleRejectRequest}
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

        .thread-count, .agent-count {
          font-size: var(--text-xs);
          color: var(--text-tertiary);
        }

        .launch-agent-btn {
          padding: var(--space-1) var(--space-2);
          font-size: var(--text-xs);
          font-weight: var(--font-medium);
          color: var(--color-primary);
          background: transparent;
          border: 1px solid var(--color-primary);
          border-radius: var(--radius-sm);
          cursor: pointer;
          transition: all var(--transition-fast);
        }

        .launch-agent-btn:hover {
          background: var(--color-primary);
          color: var(--text-inverse);
        }

        /* Running Agents Bar */
        .agents-bar {
          display: flex;
          flex-wrap: wrap;
          gap: var(--space-2);
          padding: var(--space-2) var(--space-4);
          background: var(--bg-elevated);
          border-bottom: 1px solid var(--border-subtle);
        }

        .agent-chip {
          display: flex;
          align-items: center;
          gap: var(--space-2);
          padding: var(--space-1) var(--space-2);
          background: var(--bg-surface);
          border: 1px solid var(--border-default);
          border-radius: var(--radius-md);
          font-size: var(--text-xs);
        }

        .agent-pulse {
          width: 8px;
          height: 8px;
          background: var(--color-success);
          border-radius: var(--radius-full);
          animation: pulse 2s ease-in-out infinite;
        }

        @keyframes pulse {
          0%, 100% { opacity: 1; transform: scale(1); }
          50% { opacity: 0.6; transform: scale(0.9); }
        }

        .agent-name {
          font-weight: var(--font-medium);
          color: var(--text-primary);
        }

        .agent-pid {
          color: var(--text-tertiary);
          font-family: var(--font-mono);
        }

        .agent-stop-btn {
          display: flex;
          align-items: center;
          justify-content: center;
          width: 16px;
          height: 16px;
          background: transparent;
          color: var(--text-tertiary);
          border: none;
          border-radius: var(--radius-sm);
          cursor: pointer;
          font-size: 14px;
          line-height: 1;
          transition: all var(--transition-fast);
        }

        .agent-stop-btn:hover {
          background: var(--color-danger);
          color: var(--text-inverse);
        }

        /* Modal */
        .modal-overlay {
          position: fixed;
          top: 0;
          left: 0;
          right: 0;
          bottom: 0;
          background: rgba(0, 0, 0, 0.5);
          display: flex;
          align-items: center;
          justify-content: center;
          z-index: 1000;
        }

        .modal-content {
          background: var(--bg-surface);
          border: 1px solid var(--border-default);
          border-radius: var(--radius-lg);
          padding: var(--space-6);
          width: 400px;
          max-width: 90vw;
        }

        .modal-content h3 {
          font-size: var(--text-lg);
          font-weight: var(--font-semibold);
          color: var(--text-primary);
          margin-bottom: var(--space-4);
        }

        .modal-content input {
          width: 100%;
          padding: var(--space-2) var(--space-3);
          background: var(--bg-base);
          color: var(--text-primary);
          font-size: var(--text-sm);
          border: 1px solid var(--border-default);
          border-radius: var(--radius-md);
          margin-bottom: var(--space-4);
        }

        .modal-content input:focus {
          outline: none;
          border-color: var(--color-primary);
          box-shadow: 0 0 0 2px rgba(37, 194, 160, 0.1);
        }

        .modal-actions {
          display: flex;
          justify-content: flex-end;
          gap: var(--space-2);
        }

        .modal-actions .cancel-btn {
          padding: var(--space-2) var(--space-4);
          font-size: var(--text-sm);
          font-weight: var(--font-medium);
          color: var(--text-secondary);
          background: transparent;
          border: 1px solid var(--border-default);
          border-radius: var(--radius-md);
          cursor: pointer;
          transition: all var(--transition-fast);
        }

        .modal-actions .cancel-btn:hover {
          background: var(--bg-hover);
        }

        .modal-actions .launch-btn {
          padding: var(--space-2) var(--space-4);
          font-size: var(--text-sm);
          font-weight: var(--font-medium);
          color: var(--text-inverse);
          background: var(--color-primary);
          border: none;
          border-radius: var(--radius-md);
          cursor: pointer;
          transition: all var(--transition-fast);
        }

        .modal-actions .launch-btn:hover {
          background: var(--color-primary-light);
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
