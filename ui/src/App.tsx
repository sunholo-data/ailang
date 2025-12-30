import React, { useState, useEffect } from 'react';
import { HierarchyTree } from './components/HierarchyTree';
import { AllAgentsOverview } from './components/Overview';
import { Breadcrumb } from './components/common';
import { MessageCenter } from './components/MessageCenter/MessageCenter';
import { ApprovalQueue } from './components/ApprovalQueue/ApprovalQueue';
import { ConnectionStatus } from './components/ConnectionStatus';
import { MetricsCard } from './components/MetricsCard';
import { TrendsChart } from './components/TrendsChart';
import { StatsPanel } from './components/StatsPanel';
import { Selection, HierarchyResponse, Approval } from './types';
import { TaskExecutionPanel } from './components/TaskExecution';
import './App.module.css';

// Logo - use same image as ailang.sunholo.com
const LogoImage = <img src="/logo.png" alt="AILANG" width="28" height="28" />;

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

    if (selection.type === 'task' && selection.taskId) {
      if (selection.agentId) {
        items.push({
          label: selection.agentId,
          onClick: () => setSelection({ type: 'agent', agentId: selection.agentId }),
        });
      }
      if (selection.threadId) {
        const agent = hierarchy?.root.children?.find(a => a.id === selection.agentId);
        const thread = agent?.children?.find(t => t.id === selection.threadId);
        items.push({
          label: thread?.label || 'Thread',
          onClick: () => setSelection({ type: 'thread', agentId: selection.agentId, threadId: selection.threadId }),
        });
      }
      items.push({ label: `Task ${selection.taskId.slice(0, 8)}...` });
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
        <div className="overview-container">
          <div className="overview-main">
            <AllAgentsOverview
              aggregate={hierarchy.aggregate}
              agents={hierarchy.root.children || []}
              onSelectAgent={(agentId) => setSelection({ type: 'agent', agentId })}
            />
          </div>
          <aside className="overview-sidebar">
            <StatsPanel />
          </aside>
        </div>
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

    if (selection.type === 'task' && selection.taskId) {
      return (
        <div className="task-view">
          <TaskExecutionPanel
            taskId={selection.taskId}
            threadId={selection.threadId}
            onCancel={() => {
              // Navigate back to overview or previous view
              if (selection.threadId) {
                setSelection({ type: 'thread', agentId: selection.agentId, threadId: selection.threadId });
              } else {
                setSelection({ type: 'overview' });
              }
            }}
          />
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
          <div className="brand-logo">{LogoImage}</div>
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
          <a
            href="https://ailang.sunholo.com"
            target="_blank"
            rel="noopener noreferrer"
            className="docs-link"
            title="View documentation"
          >
            <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
              <path d="M18 13v6a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2V8a2 2 0 0 1 2-2h6" />
              <polyline points="15 3 21 3 21 9" />
              <line x1="10" y1="14" x2="21" y2="3" />
            </svg>
            Docs
          </a>
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
    </div>
  );
};
