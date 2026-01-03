/**
 * Agent detail view - shows threads, metrics, and approvals for a single agent
 */
import React, { useState } from 'react';
import { HierarchyNode, Approval } from '../../../types';
import { MetricsCard, TrendsChart } from '../../../components/metrics';
import { ApprovalQueue } from '../../approvals';
import './AgentView.module.css';

interface AgentViewProps {
  agentId: string;
  agent?: HierarchyNode;
  approvals: Approval[];
  onThreadSelect: (threadId: string) => void;
  onApprove: (approvalId: string, notes: string) => void;
  onReject: (approvalId: string, notes: string) => void;
  onNavigateToThread: (threadId: string) => void;
  onCreateThread: (title: string) => void;
}

export const AgentView: React.FC<AgentViewProps> = ({
  agentId,
  agent,
  approvals,
  onThreadSelect,
  onApprove,
  onReject,
  onNavigateToThread,
  onCreateThread,
}) => {
  const [isCreatingThread, setIsCreatingThread] = useState(false);
  const [newThreadTitle, setNewThreadTitle] = useState('');

  const handleCreateThread = () => {
    if (!newThreadTitle.trim()) return;
    onCreateThread(newThreadTitle.trim());
    setNewThreadTitle('');
    setIsCreatingThread(false);
  };

  return (
    <div className="agent-view">
      <div className="agent-view-header">
        <h2>{agentId}</h2>
        <span className="agent-thread-count">
          {agent?.children?.length || 0} threads
        </span>
      </div>

      <div className="agent-metrics-section">
        <h3>Agent Metrics</h3>
        <MetricsCard scopeType="agent" scopeId={agentId} title="" />
        <div className="agent-trends-grid">
          <TrendsChart
            scopeType="agent"
            scopeId={agentId}
            period="hour"
            limit={24}
            metric="cost"
            title="Cost (24h)"
          />
          <TrendsChart
            scopeType="agent"
            scopeId={agentId}
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
                  if (e.key === 'Enter') handleCreateThread();
                  if (e.key === 'Escape') {
                    setIsCreatingThread(false);
                    setNewThreadTitle('');
                  }
                }}
                placeholder="Thread title..."
                autoFocus
              />
              <div className="form-actions">
                <button onClick={() => { setIsCreatingThread(false); setNewThreadTitle(''); }}>
                  Cancel
                </button>
                <button className="create-btn" onClick={handleCreateThread}>
                  Create
                </button>
              </div>
            </div>
          )}

          {agent?.children?.map(thread => (
            <div
              key={thread.id}
              className="thread-card"
              onClick={() => onThreadSelect(thread.id)}
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

        {approvals.length > 0 && (
          <div className="agent-approvals">
            <h3>Pending Approvals</h3>
            <ApprovalQueue
              approvals={approvals}
              history={[]}
              onApprove={onApprove}
              onReject={onReject}
              onNavigateToThread={onNavigateToThread}
            />
          </div>
        )}
      </div>
    </div>
  );
};
