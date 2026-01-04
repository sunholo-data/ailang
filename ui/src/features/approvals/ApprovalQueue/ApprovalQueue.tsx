import React, { useState } from 'react';
import { Approval, EffectDelta } from '../../../types';
import { Icons } from '../../../components/common/Icons';
import { formatTimestamp } from '../../../utils/formatters';
import './ApprovalQueue.css';

interface ApprovalQueueProps {
  approvals: Approval[];
  history?: Approval[];
  onApprove: (approvalId: string, notes: string) => void;
  onReject: (approvalId: string, notes: string) => void;
  onNavigateToThread?: (threadId: string) => void;
}

export const ApprovalQueue: React.FC<ApprovalQueueProps> = ({
  approvals,
  history = [],
  onApprove,
  onReject,
  onNavigateToThread,
}) => {
  const [showHistory, setShowHistory] = useState(true);
  const [expandedId, setExpandedId] = useState<string | null>(null);
  const [reviewNotes, setReviewNotes] = useState<Map<string, string>>(new Map());

  const parseEffectDelta = (json: string): EffectDelta | null => {
    try {
      return JSON.parse(json) as EffectDelta;
    } catch {
      return null;
    }
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
      {/* Header */}
      <div className="queue-header">
        <div className="header-title">
          <h2>Approval Queue</h2>
          <span className="pending-count">
            {pendingApprovals.length} pending
          </span>
        </div>
      </div>

      {/* Approvals List */}
      <div className="approvals-container">
        {pendingApprovals.length === 0 ? (
          <div className="empty-state">
            <div className="empty-icon">{Icons.sparkles}</div>
            <h3>All caught up!</h3>
            <p>No pending approvals to review</p>
          </div>
        ) : (
          <div className="approvals-list">
            {pendingApprovals.map((approval) => {
              const effectDelta = parseEffectDelta(approval.effect_delta_json);
              const isExpanded = expandedId === approval.id;

              return (
                <div
                  key={approval.id}
                  className={`approval-card impact-${approval.impact}`}
                >
                  {/* Card Header */}
                  <div
                    className="card-header"
                    onClick={() => setExpandedId(isExpanded ? null : approval.id)}
                  >
                    <div className="header-left">
                      <div className={`impact-indicator ${approval.impact}`} />
                      <div className="proposal-info">
                        <span className="proposal-text">{approval.proposal}</span>
                        <div className="proposal-meta">
                          {approval.thread_title && (
                            <span
                              className="meta-item thread-link"
                              onClick={(e) => {
                                e.stopPropagation();
                                onNavigateToThread?.(approval.thread_id);
                              }}
                              title="Go to thread"
                            >
                              {Icons.message}
                              {approval.thread_title}
                            </span>
                          )}
                          <span className="meta-item">
                            {Icons.bot}
                            {approval.instance_id}
                          </span>
                          <span className="meta-item">
                            {Icons.clock}
                            {formatTimestamp(approval.created_at)}
                          </span>
                        </div>
                      </div>
                    </div>

                    <div className="header-right">
                      <span className="cost-badge">
                        {Icons.dollar}
                        ${approval.estimated_cost.toFixed(2)}
                      </span>
                      <span className={`impact-badge ${approval.impact}`}>
                        {approval.impact}
                      </span>
                      <button className="expand-btn">
                        {isExpanded ? Icons.chevronUp : Icons.chevronDown}
                      </button>
                    </div>
                  </div>

                  {/* Expanded Details */}
                  {isExpanded && (
                    <div className="card-details">
                      {/* Effect Details */}
                      {effectDelta && (
                        <div className="detail-section">
                          <h4>Effect Details</h4>
                          <div className="detail-grid">
                            <div className="detail-item">
                              <span className="detail-label">Capability</span>
                              <span className="detail-value code">{effectDelta.cap_type}</span>
                            </div>
                            <div className="detail-item">
                              <span className="detail-label">Budget Delta</span>
                              <span className="detail-value">${effectDelta.budget_delta.toFixed(2)}</span>
                            </div>
                            {effectDelta.paths.length > 0 && (
                              <div className="detail-item full-width">
                                <span className="detail-label">Paths</span>
                                <div className="paths-list">
                                  {effectDelta.paths.map((path, idx) => (
                                    <span key={idx} className="path-tag">
                                      {Icons.folder}
                                      {path}
                                    </span>
                                  ))}
                                </div>
                              </div>
                            )}
                          </div>
                        </div>
                      )}

                      {/* Request Info */}
                      <div className="detail-section">
                        <h4>Request Info</h4>
                        <div className="detail-grid">
                          <div className="detail-item">
                            <span className="detail-label">Thread</span>
                            <span className="detail-value code">{approval.thread_id}</span>
                          </div>
                          <div className="detail-item">
                            <span className="detail-label">Impact Level</span>
                            <span className={`detail-value impact-text ${approval.impact}`}>
                              {approval.impact.toUpperCase()}
                            </span>
                          </div>
                        </div>
                      </div>

                      {/* Review Section */}
                      <div className="review-section">
                        <h4>Review Notes</h4>
                        <textarea
                          value={reviewNotes.get(approval.id) || ''}
                          onChange={(e) => updateNotes(approval.id, e.target.value)}
                          placeholder="Add notes about your decision (required for rejection)..."
                          rows={3}
                        />

                        <div className="action-buttons">
                          <button
                            className="reject-btn"
                            onClick={() => handleReject(approval.id)}
                          >
                            {Icons.x}
                            Reject
                          </button>
                          <button
                            className="approve-btn"
                            onClick={() => handleApprove(approval.id)}
                          >
                            {Icons.check}
                            Approve
                          </button>
                        </div>
                      </div>
                    </div>
                  )}
                </div>
              );
            })}
          </div>
        )}

        {/* History Section */}
        {history.length > 0 && (
          <div className="history-section">
            <div
              className="history-header"
              onClick={() => setShowHistory(!showHistory)}
            >
              <h3>
                {showHistory ? Icons.chevronDown : Icons.chevronUp}
                Review History
              </h3>
              <span className="history-count">{history.length} decisions</span>
            </div>

            {showHistory && (
              <div className="history-list">
                {history.map((approval) => {
                  const isExpanded = expandedId === `history-${approval.id}`;

                  return (
                    <div
                      key={`history-${approval.id}`}
                      className={`history-card ${approval.status}`}
                      onClick={() => setExpandedId(isExpanded ? null : `history-${approval.id}`)}
                    >
                      <div className="history-card-header">
                        <div className="history-status">
                          <span className={`status-icon ${approval.status}`}>
                            {approval.status === 'approved' ? Icons.check : Icons.x}
                          </span>
                          <div className="history-info">
                            <span className="history-proposal">{approval.proposal}</span>
                            {approval.thread_title && (
                              <span
                                className="history-thread"
                                onClick={(e) => {
                                  e.stopPropagation();
                                  onNavigateToThread?.(approval.thread_id);
                                }}
                                title="Go to thread"
                              >
                                {Icons.message}
                                {approval.thread_title}
                              </span>
                            )}
                          </div>
                        </div>
                        <div className="history-meta">
                          <span className="history-agent">{approval.instance_id}</span>
                          <span className={`history-badge ${approval.status}`}>
                            {approval.status}
                          </span>
                          <span className="history-time">
                            {approval.reviewed_at ? formatTimestamp(approval.reviewed_at) : formatTimestamp(approval.created_at)}
                          </span>
                        </div>
                      </div>

                      {isExpanded && (
                        <div className="history-details">
                          <div className="detail-row">
                            <span className="detail-label">Reviewed by</span>
                            <span className="detail-value">{approval.reviewed_by || 'Unknown'}</span>
                          </div>
                          <div className="detail-row">
                            <span className="detail-label">Cost</span>
                            <span className="detail-value">${approval.estimated_cost.toFixed(2)}</span>
                          </div>
                          <div className="detail-row">
                            <span className="detail-label">Impact</span>
                            <span className={`detail-value impact-text ${approval.impact}`}>
                              {approval.impact.toUpperCase()}
                            </span>
                          </div>
                          {approval.review_notes && (
                            <div className="detail-row full-width">
                              <span className="detail-label">Notes</span>
                              <span className="detail-value notes">{approval.review_notes}</span>
                            </div>
                          )}
                        </div>
                      )}
                    </div>
                  );
                })}
              </div>
            )}
          </div>
        )}
      </div>
    </div>
  );
};
