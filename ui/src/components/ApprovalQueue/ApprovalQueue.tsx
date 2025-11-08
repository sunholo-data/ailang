import React, { useState } from 'react';
import { Approval, EffectDelta } from '../../types';

interface ApprovalQueueProps {
  approvals: Approval[];
  onApprove: (approvalId: string, notes: string) => void;
  onReject: (approvalId: string, notes: string) => void;
}

export const ApprovalQueue: React.FC<ApprovalQueueProps> = ({
  approvals,
  onApprove,
  onReject,
}) => {
  const [expandedId, setExpandedId] = useState<string | null>(null);
  const [reviewNotes, setReviewNotes] = useState<Map<string, string>>(new Map());

  const parseEffectDelta = (json: string): EffectDelta | null => {
    try {
      return JSON.parse(json) as EffectDelta;
    } catch {
      return null;
    }
  };

  const getImpactColor = (impact: string) => {
    switch (impact) {
      case 'low':
        return '#28a745';
      case 'medium':
        return '#ffc107';
      case 'high':
        return '#dc3545';
      default:
        return '#6c757d';
    }
  };

  const getImpactIcon = (impact: string) => {
    switch (impact) {
      case 'low':
        return '🟢';
      case 'medium':
        return '🟡';
      case 'high':
        return '🔴';
      default:
        return '⚪';
    }
  };

  const formatTimestamp = (timestamp: number) => {
    const date = new Date(timestamp);
    return date.toLocaleString();
  };

  const handleApprove = (approvalId: string) => {
    const notes = reviewNotes.get(approvalId) || '';
    onApprove(approvalId, notes);
    setReviewNotes(new Map(reviewNotes.set(approvalId, '')));
  };

  const handleReject = (approvalId: string) => {
    const notes = reviewNotes.get(approvalId) || '';
    if (!notes.trim()) {
      alert('Please provide a reason for rejection');
      return;
    }
    onReject(approvalId, notes);
    setReviewNotes(new Map(reviewNotes.set(approvalId, '')));
  };

  const updateNotes = (approvalId: string, notes: string) => {
    setReviewNotes(new Map(reviewNotes.set(approvalId, notes)));
  };

  const pendingApprovals = approvals.filter((a) => a.status === 'pending');

  return (
    <div className="approval-queue">
      <div className="queue-header">
        <h2>Approval Queue</h2>
        <span className="count-badge">{pendingApprovals.length} pending</span>
      </div>

      <div className="approvals-list">
        {pendingApprovals.length === 0 ? (
          <div className="empty-state">
            <p>✨ No pending approvals</p>
            <p className="hint">All requests have been reviewed</p>
          </div>
        ) : (
          pendingApprovals.map((approval) => {
            const effectDelta = parseEffectDelta(approval.effect_delta_json);
            const isExpanded = expandedId === approval.id;

            return (
              <div key={approval.id} className="approval-card">
                <div
                  className="approval-header"
                  onClick={() => setExpandedId(isExpanded ? null : approval.id)}
                >
                  <div className="approval-title">
                    <span className="impact-icon">
                      {getImpactIcon(approval.impact)}
                    </span>
                    <span className="proposal">{approval.proposal}</span>
                  </div>

                  <div className="approval-meta">
                    <span className="instance">🤖 {approval.instance_id}</span>
                    <span className="cost">${approval.estimated_cost.toFixed(2)}</span>
                    <span className="expand-icon">{isExpanded ? '▼' : '▶'}</span>
                  </div>
                </div>

                {isExpanded && (
                  <div className="approval-details">
                    <div className="detail-section">
                      <h4>Effect Details</h4>
                      {effectDelta && (
                        <div className="effect-info">
                          <div className="info-row">
                            <span className="label">Capability:</span>
                            <span className="value">{effectDelta.cap_type}</span>
                          </div>
                          <div className="info-row">
                            <span className="label">Budget Delta:</span>
                            <span className="value">
                              ${effectDelta.budget_delta.toFixed(2)}
                            </span>
                          </div>
                          {effectDelta.paths.length > 0 && (
                            <div className="info-row">
                              <span className="label">Paths:</span>
                              <div className="paths">
                                {effectDelta.paths.map((path, idx) => (
                                  <span key={idx} className="path">
                                    {path}
                                  </span>
                                ))}
                              </div>
                            </div>
                          )}
                        </div>
                      )}
                    </div>

                    <div className="detail-section">
                      <h4>Request Info</h4>
                      <div className="info-row">
                        <span className="label">Thread:</span>
                        <span className="value">{approval.thread_id}</span>
                      </div>
                      <div className="info-row">
                        <span className="label">Requested:</span>
                        <span className="value">
                          {formatTimestamp(approval.created_at)}
                        </span>
                      </div>
                      <div className="info-row">
                        <span className="label">Impact:</span>
                        <span
                          className="value"
                          style={{ color: getImpactColor(approval.impact) }}
                        >
                          {approval.impact.toUpperCase()}
                        </span>
                      </div>
                    </div>

                    <div className="review-section">
                      <h4>Review Notes</h4>
                      <textarea
                        value={reviewNotes.get(approval.id) || ''}
                        onChange={(e) => updateNotes(approval.id, e.target.value)}
                        placeholder="Add notes about your decision..."
                        rows={3}
                      />

                      <div className="action-buttons">
                        <button
                          className="reject-btn"
                          onClick={() => handleReject(approval.id)}
                        >
                          ❌ Reject
                        </button>
                        <button
                          className="approve-btn"
                          onClick={() => handleApprove(approval.id)}
                        >
                          ✅ Approve
                        </button>
                      </div>
                    </div>
                  </div>
                )}
              </div>
            );
          })
        )}
      </div>

      <style jsx>{`
        .approval-queue {
          display: flex;
          flex-direction: column;
          height: 100%;
          background: #f5f5f5;
        }

        .queue-header {
          display: flex;
          justify-content: space-between;
          align-items: center;
          padding: 1rem;
          background: white;
          border-bottom: 1px solid #e0e0e0;
        }

        .queue-header h2 {
          margin: 0;
          font-size: 1.25rem;
          font-weight: 600;
        }

        .count-badge {
          background: #007bff;
          color: white;
          padding: 0.25rem 0.75rem;
          border-radius: 12px;
          font-size: 0.875rem;
          font-weight: 500;
        }

        .approvals-list {
          flex: 1;
          overflow-y: auto;
          padding: 1rem;
        }

        .empty-state {
          text-align: center;
          padding: 3rem 1rem;
          color: #666;
        }

        .empty-state p {
          margin: 0.5rem 0;
        }

        .empty-state .hint {
          font-size: 0.875rem;
          color: #999;
        }

        .approval-card {
          background: white;
          border-radius: 8px;
          box-shadow: 0 2px 4px rgba(0, 0, 0, 0.1);
          margin-bottom: 1rem;
          overflow: hidden;
        }

        .approval-header {
          padding: 1rem;
          cursor: pointer;
          transition: background 0.2s;
        }

        .approval-header:hover {
          background: #f8f9fa;
        }

        .approval-title {
          display: flex;
          align-items: center;
          gap: 0.5rem;
          margin-bottom: 0.5rem;
        }

        .impact-icon {
          font-size: 1.25rem;
        }

        .proposal {
          font-weight: 500;
          font-size: 1rem;
        }

        .approval-meta {
          display: flex;
          gap: 1rem;
          align-items: center;
          font-size: 0.875rem;
          color: #666;
        }

        .instance {
          flex: 1;
        }

        .cost {
          font-weight: 600;
          color: #333;
        }

        .expand-icon {
          color: #999;
          font-size: 0.75rem;
        }

        .approval-details {
          border-top: 1px solid #e0e0e0;
          padding: 1rem;
          background: #fafafa;
        }

        .detail-section {
          margin-bottom: 1rem;
        }

        .detail-section h4 {
          margin: 0 0 0.5rem 0;
          font-size: 0.875rem;
          font-weight: 600;
          color: #333;
        }

        .effect-info {
          background: white;
          padding: 0.75rem;
          border-radius: 4px;
        }

        .info-row {
          display: flex;
          gap: 0.5rem;
          margin-bottom: 0.5rem;
          font-size: 0.875rem;
        }

        .info-row:last-child {
          margin-bottom: 0;
        }

        .label {
          font-weight: 500;
          color: #666;
          min-width: 100px;
        }

        .value {
          color: #333;
        }

        .paths {
          display: flex;
          flex-wrap: wrap;
          gap: 0.25rem;
        }

        .path {
          background: #e0e0e0;
          padding: 0.125rem 0.5rem;
          border-radius: 4px;
          font-size: 0.75rem;
          font-family: monospace;
        }

        .review-section textarea {
          width: 100%;
          padding: 0.75rem;
          border: 1px solid #e0e0e0;
          border-radius: 4px;
          font-family: inherit;
          font-size: 0.875rem;
          resize: vertical;
          margin-bottom: 0.75rem;
        }

        .action-buttons {
          display: flex;
          gap: 0.5rem;
          justify-content: flex-end;
        }

        .reject-btn,
        .approve-btn {
          padding: 0.5rem 1.5rem;
          border: none;
          border-radius: 4px;
          font-weight: 500;
          cursor: pointer;
          font-size: 0.875rem;
        }

        .reject-btn {
          background: #dc3545;
          color: white;
        }

        .reject-btn:hover {
          background: #c82333;
        }

        .approve-btn {
          background: #28a745;
          color: white;
        }

        .approve-btn:hover {
          background: #218838;
        }
      `}</style>
    </div>
  );
};
