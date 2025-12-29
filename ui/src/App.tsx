import React, { useState, useEffect } from 'react';
import { HierarchyTree } from './components/HierarchyTree';
import { AllAgentsOverview } from './components/Overview';
import { Breadcrumb } from './components/common';
import { MessageCenter } from './components/MessageCenter/MessageCenter';
import { ApprovalQueue } from './components/ApprovalQueue/ApprovalQueue';
import { ConnectionStatus } from './components/ConnectionStatus';
import { MetricsCard } from './components/MetricsCard';
import { TrendsChart } from './components/TrendsChart';
import { Selection, HierarchyResponse, Approval } from './types';

// Logo icon
const LogoIcon = (
  <svg width="28" height="28" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.5" strokeLinecap="round" strokeLinejoin="round">
    <circle cx="12" cy="12" r="10" />
    <path d="M12 6v12M6 12h12" />
    <circle cx="12" cy="12" r="3" fill="currentColor" />
  </svg>
);

export const App: React.FC = () => {
  const [selection, setSelection] = useState<Selection>({ type: 'overview' });
  const [hierarchy, setHierarchy] = useState<HierarchyResponse | null>(null);
  const [approvals, setApprovals] = useState<Approval[]>([]);
  const [approvalHistory, setApprovalHistory] = useState<Approval[]>([]);
  const [isCreatingThread, setIsCreatingThread] = useState(false);
  const [newThreadTitle, setNewThreadTitle] = useState('');
  const [version, setVersion] = useState<string>('...');

  // WebSocket URL - dynamically use current host/port
  const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
  const websocketUrl = `${protocol}//${window.location.host}/ws`;

  // Fetch version from server
  useEffect(() => {
    const fetchVersion = async () => {
      try {
        const response = await fetch('/api/version');
        if (response.ok) {
          const data = await response.json();
          setVersion(data.version || 'dev');
        }
      } catch (error) {
        console.error('Error fetching version:', error);
        setVersion('dev');
      }
    };
    fetchVersion();
  }, []);

  // Fetch hierarchy data
  useEffect(() => {
    const fetchHierarchy = async () => {
      try {
        const response = await fetch('/api/hierarchy');
        if (response.ok) {
          const data = await response.json();
          setHierarchy(data);
        }
      } catch (error) {
        console.error('Error fetching hierarchy:', error);
      }
    };
    fetchHierarchy();
    const interval = setInterval(fetchHierarchy, 5000);
    return () => clearInterval(interval);
  }, []);

  // Fetch approvals
  useEffect(() => {
    const fetchApprovals = async () => {
      try {
        const pendingResponse = await fetch('/api/approvals?status=pending');
        if (pendingResponse.ok) {
          const pendingData: Approval[] = await pendingResponse.json();
          setApprovals(pendingData);
        }

        const [approvedResponse, rejectedResponse] = await Promise.all([
          fetch('/api/approvals?status=approved'),
          fetch('/api/approvals?status=rejected'),
        ]);

        const history: Approval[] = [];
        if (approvedResponse.ok) {
          const approved: Approval[] = await approvedResponse.json();
          history.push(...approved);
        }
        if (rejectedResponse.ok) {
          const rejected: Approval[] = await rejectedResponse.json();
          history.push(...rejected);
        }

        history.sort((a, b) => {
          const aTime = a.reviewed_at ? new Date(a.reviewed_at).getTime() : 0;
          const bTime = b.reviewed_at ? new Date(b.reviewed_at).getTime() : 0;
          return bTime - aTime;
        });

        setApprovalHistory(history);
      } catch (error) {
        console.error('Error fetching approvals:', error);
      }
    };
    fetchApprovals();
    const interval = setInterval(fetchApprovals, 5000);
    return () => clearInterval(interval);
  }, []);

  const handleApprove = async (approvalId: string, notes: string) => {
    try {
      const response = await fetch(`/api/approvals/${approvalId}/approve`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ notes }),
      });

      if (!response.ok) {
        console.error('Failed to approve:', await response.text());
        return;
      }

      const approved = approvals.find((a) => a.id === approvalId);
      if (approved) {
        const updatedApproval = {
          ...approved,
          status: 'approved' as const,
          reviewed_by: 'user',
          review_notes: notes,
          reviewed_at: Date.now(),
        };
        setApprovalHistory((prev) => [updatedApproval, ...prev]);
      }
      setApprovals((prev) => prev.filter((a) => a.id !== approvalId));
    } catch (error) {
      console.error('Error approving:', error);
    }
  };

  const handleReject = async (approvalId: string, notes: string) => {
    try {
      const response = await fetch(`/api/approvals/${approvalId}/reject`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ notes }),
      });

      if (!response.ok) {
        console.error('Failed to reject:', await response.text());
        return;
      }

      const rejected = approvals.find((a) => a.id === approvalId);
      if (rejected) {
        const updatedApproval = {
          ...rejected,
          status: 'rejected' as const,
          reviewed_by: 'user',
          review_notes: notes,
          reviewed_at: Date.now(),
        };
        setApprovalHistory((prev) => [updatedApproval, ...prev]);
      }
      setApprovals((prev) => prev.filter((a) => a.id !== approvalId));
    } catch (error) {
      console.error('Error rejecting:', error);
    }
  };

  // Build breadcrumb items
  const getBreadcrumbItems = () => {
    const items = [
      { label: 'All Agents', onClick: () => setSelection({ type: 'overview' }) },
    ];

    if (selection.type === 'agent' && selection.agentId) {
      items.push({ label: selection.agentId });
    }

    if (selection.type === 'thread' && selection.threadId) {
      if (selection.agentId) {
        items.push({
          label: selection.agentId,
          onClick: () => setSelection({ type: 'agent', agentId: selection.agentId }),
        });
      }
      // Find thread title
      const agent = hierarchy?.root.children?.find(a => a.id === selection.agentId);
      const thread = agent?.children?.find(t => t.id === selection.threadId);
      items.push({ label: thread?.label || 'Thread' });
    }

    return items;
  };

  // Handle navigation to thread from approval
  const handleNavigateToThread = (threadId: string) => {
    // Find which agent owns this thread
    const agent = hierarchy?.root.children?.find(a => a.children?.some(t => t.id === threadId));
    setSelection({ type: 'thread', agentId: agent?.id, threadId });
  };

  // Create a new thread for a specific agent
  const handleCreateThreadForAgent = async (agentId: string) => {
    if (!newThreadTitle.trim()) return;

    try {
      const response = await fetch('/api/threads', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          title: newThreadTitle.trim(),
          created_by_type: 'human',
          created_by_id: 'user',
          target_agent: agentId,
        }),
      });

      if (!response.ok) {
        console.error('Failed to create thread:', await response.text());
        return;
      }

      const newThread = await response.json();
      setNewThreadTitle('');
      setIsCreatingThread(false);
      // Navigate to the new thread
      setSelection({ type: 'thread', agentId, threadId: newThread.id });
    } catch (error) {
      console.error('Error creating thread:', error);
    }
  };

  // Render main content based on selection
  const renderContent = () => {
    if (selection.type === 'overview' && hierarchy) {
      return (
        <AllAgentsOverview
          aggregate={hierarchy.aggregate}
          agents={hierarchy.root.children || []}
          onSelectAgent={(agentId) => setSelection({ type: 'agent', agentId })}
        />
      );
    }

    if (selection.type === 'agent' && selection.agentId) {
      // Find agent's threads and show them with approvals
      const agent = hierarchy?.root.children?.find(a => a.id === selection.agentId);
      const agentApprovals = approvals.filter(a => {
        // Check if approval belongs to a thread owned by this agent
        return agent?.children?.some(t => t.id === a.thread_id);
      });

      return (
        <div className="agent-view">
          <div className="agent-view-header">
            <h2>{selection.agentId}</h2>
            <span className="agent-thread-count">
              {agent?.children?.length || 0} threads
            </span>
          </div>

          <div className="agent-metrics-section">
            <h3>Agent Metrics</h3>
            <MetricsCard scopeType="agent" scopeId={selection.agentId} title="" />
            <div className="agent-trends-grid">
              <TrendsChart
                scopeType="agent"
                scopeId={selection.agentId}
                period="hour"
                limit={24}
                metric="cost"
                title="Cost (24h)"
              />
              <TrendsChart
                scopeType="agent"
                scopeId={selection.agentId}
                period="hour"
                limit={24}
                metric="tokens"
                title="Tokens (24h)"
              />
            </div>
          </div>

          <div className="agent-view-content">
            <div className="agent-threads">
              <div className="threads-header">
                <h3>Threads</h3>
                <button
                  className="new-thread-btn"
                  onClick={() => setIsCreatingThread(true)}
                  title="New thread"
                >
                  + New Thread
                </button>
              </div>

              {isCreatingThread && (
                <div className="new-thread-form">
                  <input
                    type="text"
                    value={newThreadTitle}
                    onChange={(e) => setNewThreadTitle(e.target.value)}
                    onKeyDown={(e) => {
                      if (e.key === 'Enter') handleCreateThreadForAgent(selection.agentId!);
                      if (e.key === 'Escape') { setIsCreatingThread(false); setNewThreadTitle(''); }
                    }}
                    placeholder="Thread title..."
                    autoFocus
                  />
                  <div className="form-actions">
                    <button onClick={() => { setIsCreatingThread(false); setNewThreadTitle(''); }}>
                      Cancel
                    </button>
                    <button
                      className="create-btn"
                      onClick={() => handleCreateThreadForAgent(selection.agentId!)}
                    >
                      Create
                    </button>
                  </div>
                </div>
              )}

              {agent?.children?.map(thread => (
                <div
                  key={thread.id}
                  className="thread-card"
                  onClick={() => setSelection({ type: 'thread', agentId: selection.agentId, threadId: thread.id })}
                >
                  <span className="thread-title">{thread.label}</span>
                  {thread.badges && thread.badges.length > 0 && (
                    <span className="thread-badges">
                      {thread.badges.map((badge, i) => (
                        <span key={i} className={`badge badge-${badge.type}`}>
                          {badge.count}
                        </span>
                      ))}
                    </span>
                  )}
                </div>
              ))}
              {(!agent?.children || agent.children.length === 0) && !isCreatingThread && (
                <div className="no-threads">
                  No threads yet
                  <button
                    className="start-thread-btn"
                    onClick={() => setIsCreatingThread(true)}
                  >
                    Start a conversation
                  </button>
                </div>
              )}
            </div>

            {agentApprovals.length > 0 && (
              <div className="agent-approvals">
                <h3>Pending Approvals</h3>
                <ApprovalQueue
                  approvals={agentApprovals}
                  history={[]}
                  onApprove={handleApprove}
                  onReject={handleReject}
                  onNavigateToThread={handleNavigateToThread}
                />
              </div>
            )}
          </div>
        </div>
      );
    }

    if (selection.type === 'thread' && selection.threadId) {
      return (
        <div className="thread-view">
          <div className="thread-metrics-bar">
            <MetricsCard
              scopeType="thread"
              scopeId={selection.threadId}
              title="Thread Metrics"
              compact
            />
          </div>
          <div className="thread-messages-container">
            <MessageCenter
              websocketUrl={websocketUrl}
              instanceId={selection.agentId || 'default'}
              initialThreadId={selection.threadId}
              onThreadNavigated={() => {}}
            />
          </div>
        </div>
      );
    }

    return (
      <div className="empty-state">
        <p>Select an agent or thread from the sidebar</p>
      </div>
    );
  };

  const pendingCount = approvals?.filter((a) => a.status === 'pending').length || 0;

  return (
    <div className="app">
      <header className="app-header">
        <div className="header-brand">
          <div className="brand-logo">{LogoIcon}</div>
          <div className="brand-text">
            <h1>AILANG</h1>
            <span className="brand-subtitle">Collaboration Hub</span>
          </div>
        </div>

        <div className="header-meta">
          <ConnectionStatus />
          {pendingCount > 0 && (
            <span className="pending-badge" title={`${pendingCount} pending approvals`}>
              {pendingCount} pending
            </span>
          )}
          <span className="version-tag">{version}</span>
        </div>
      </header>

      <div className="app-body">
        <aside className="app-sidebar">
          <HierarchyTree
            selection={selection}
            onSelect={setSelection}
          />
        </aside>

        <main className="app-main">
          {selection.type !== 'overview' && (
            <Breadcrumb items={getBreadcrumbItems()} />
          )}
          <div className="main-content">
            {renderContent()}
          </div>
        </main>
      </div>

      <style>{`
        .app {
          display: flex;
          flex-direction: column;
          height: 100vh;
          background: var(--bg-base);
          color: var(--text-primary);
        }

        /* Header */
        .app-header {
          display: flex;
          align-items: center;
          justify-content: space-between;
          height: 52px;
          padding: 0 var(--space-4);
          background: var(--bg-surface);
          border-bottom: 1px solid var(--border-subtle);
          flex-shrink: 0;
        }

        .header-brand {
          display: flex;
          align-items: center;
          gap: var(--space-3);
        }

        .brand-logo {
          display: flex;
          align-items: center;
          justify-content: center;
          width: 36px;
          height: 36px;
          background: linear-gradient(135deg, var(--color-primary-dark), var(--color-primary));
          border-radius: var(--radius-md);
          color: var(--text-inverse);
        }

        .brand-text h1 {
          font-size: var(--text-base);
          font-weight: var(--font-bold);
          letter-spacing: -0.02em;
          color: var(--text-primary);
          line-height: 1;
          margin-bottom: 2px;
        }

        .brand-subtitle {
          font-size: 10px;
          color: var(--text-tertiary);
          text-transform: uppercase;
          letter-spacing: 0.1em;
        }

        .header-meta {
          display: flex;
          align-items: center;
          gap: var(--space-3);
        }

        .pending-badge {
          padding: var(--space-1) var(--space-2);
          background: rgba(245, 158, 11, 0.15);
          color: #f59e0b;
          font-size: var(--text-xs);
          font-weight: var(--font-medium);
          border-radius: var(--radius-full);
        }

        .version-tag {
          padding: var(--space-1) var(--space-2);
          background: var(--bg-elevated);
          color: var(--text-tertiary);
          font-family: var(--font-mono);
          font-size: var(--text-xs);
          border-radius: var(--radius-sm);
          border: 1px solid var(--border-subtle);
        }

        /* Body Layout */
        .app-body {
          display: flex;
          flex: 1;
          overflow: hidden;
        }

        .app-sidebar {
          flex-shrink: 0;
          overflow: hidden;
        }

        .app-main {
          flex: 1;
          display: flex;
          flex-direction: column;
          overflow: hidden;
          background: var(--bg-base);
        }

        .main-content {
          flex: 1;
          overflow: auto;
        }

        /* Agent View */
        .agent-view {
          padding: 24px;
          height: 100%;
          overflow-y: auto;
        }

        .agent-view-header {
          display: flex;
          align-items: center;
          gap: 16px;
          margin-bottom: 24px;
        }

        .agent-view-header h2 {
          margin: 0;
          font-size: 24px;
          font-weight: 600;
          color: #cdd6f4;
        }

        .agent-thread-count {
          font-size: 14px;
          color: #6c7086;
        }

        .agent-view-content {
          display: flex;
          flex-direction: column;
          gap: 32px;
        }

        /* Agent Metrics Section */
        .agent-metrics-section {
          margin-bottom: 24px;
          padding: 20px;
          background: linear-gradient(135deg, rgba(59, 130, 246, 0.08), rgba(99, 102, 241, 0.04));
          border: 1px solid rgba(59, 130, 246, 0.2);
          border-radius: 12px;
        }

        .agent-metrics-section h3 {
          margin: 0 0 16px 0;
          font-size: 16px;
          font-weight: 600;
          color: #3b82f6;
        }

        .agent-trends-grid {
          display: grid;
          grid-template-columns: repeat(auto-fit, minmax(300px, 1fr));
          gap: 16px;
          margin-top: 16px;
        }

        /* Thread View */
        .thread-view {
          display: flex;
          flex-direction: column;
          height: 100%;
          overflow: hidden;
        }

        .thread-metrics-bar {
          flex-shrink: 0;
          padding: 12px 16px;
          background: linear-gradient(135deg, rgba(34, 197, 94, 0.08), rgba(16, 185, 129, 0.04));
          border-bottom: 1px solid rgba(34, 197, 94, 0.2);
        }

        .thread-messages-container {
          flex: 1;
          overflow: hidden;
        }

        .agent-threads h3,
        .agent-approvals h3 {
          margin: 0 0 16px 0;
          font-size: 16px;
          font-weight: 600;
          color: #cdd6f4;
        }

        .thread-card {
          display: flex;
          align-items: center;
          justify-content: space-between;
          padding: 12px 16px;
          background: #1e1e2e;
          border: 1px solid #313244;
          border-radius: 8px;
          margin-bottom: 8px;
          cursor: pointer;
          transition: all 0.15s;
        }

        .thread-card:hover {
          border-color: #45475a;
          background: #232336;
        }

        .thread-title {
          font-size: 14px;
          color: #cdd6f4;
        }

        .thread-badges {
          display: flex;
          gap: 6px;
        }

        .badge {
          padding: 2px 8px;
          font-size: 11px;
          border-radius: 10px;
        }

        .badge-pending {
          background: rgba(245, 158, 11, 0.2);
          color: #f59e0b;
        }

        .badge-unread {
          background: rgba(59, 130, 246, 0.2);
          color: #3b82f6;
        }

        .badge-running {
          background: rgba(34, 197, 94, 0.2);
          color: #22c55e;
        }

        .no-threads {
          padding: 20px;
          text-align: center;
          color: #6c7086;
          font-size: 14px;
          display: flex;
          flex-direction: column;
          align-items: center;
          gap: 12px;
        }

        .threads-header {
          display: flex;
          align-items: center;
          justify-content: space-between;
          margin-bottom: 16px;
        }

        .threads-header h3 {
          margin: 0;
        }

        .new-thread-btn {
          padding: 6px 12px;
          background: var(--color-primary);
          color: white;
          border: none;
          border-radius: 6px;
          font-size: 13px;
          cursor: pointer;
          transition: all 0.15s;
        }

        .new-thread-btn:hover {
          background: var(--color-primary-dark);
        }

        .start-thread-btn {
          padding: 8px 16px;
          background: var(--color-primary);
          color: white;
          border: none;
          border-radius: 6px;
          font-size: 13px;
          cursor: pointer;
          transition: all 0.15s;
        }

        .start-thread-btn:hover {
          background: var(--color-primary-dark);
        }

        .new-thread-form {
          padding: 16px;
          background: #1e1e2e;
          border: 1px solid #313244;
          border-radius: 8px;
          margin-bottom: 12px;
        }

        .new-thread-form input {
          width: 100%;
          padding: 10px 12px;
          background: #11111b;
          border: 1px solid #45475a;
          border-radius: 6px;
          color: #cdd6f4;
          font-size: 14px;
          margin-bottom: 12px;
        }

        .new-thread-form input:focus {
          outline: none;
          border-color: var(--color-primary);
        }

        .form-actions {
          display: flex;
          justify-content: flex-end;
          gap: 8px;
        }

        .form-actions button {
          padding: 6px 14px;
          border-radius: 6px;
          font-size: 13px;
          cursor: pointer;
          transition: all 0.15s;
        }

        .form-actions button:first-child {
          background: transparent;
          border: 1px solid #45475a;
          color: #6c7086;
        }

        .form-actions button:first-child:hover {
          background: #313244;
        }

        .form-actions .create-btn {
          background: var(--color-primary);
          border: none;
          color: white;
        }

        .form-actions .create-btn:hover {
          background: var(--color-primary-dark);
        }

        .empty-state {
          display: flex;
          align-items: center;
          justify-content: center;
          height: 100%;
          color: #6c7086;
          font-size: 14px;
        }

        /* Responsive */
        @media (max-width: 768px) {
          .brand-text {
            display: none;
          }

          .app-sidebar {
            width: 60px;
          }
        }
      `}</style>
    </div>
  );
};
