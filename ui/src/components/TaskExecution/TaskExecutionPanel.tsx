import React, { useState, useCallback, useEffect } from 'react';
import {
  TaskStreamEvent,
  TaskResourceMetrics,
  PendingApprovalRequest,
} from '../../types';
import { StreamingLog } from './StreamingLog';
import { ResourceMetrics } from './ResourceMetrics';
import styles from './TaskExecution.module.css';

interface TaskExecutionPanelProps {
  taskId: string;
  threadId?: string;
  events: TaskStreamEvent[];
  metrics: TaskResourceMetrics | null;
  pendingApproval: PendingApprovalRequest | null;
  status: 'pending' | 'running' | 'completed' | 'failed' | 'approval_pending';
  onApprove?: (requestId: string) => void;
  onReject?: (requestId: string) => void;
  onCancel?: () => void;
}

const getStatusBadge = (status: string): { label: string; className: string } => {
  switch (status) {
    case 'pending':
      return { label: 'Pending', className: styles.statusPending };
    case 'running':
      return { label: 'Running', className: styles.statusRunning };
    case 'completed':
      return { label: 'Completed', className: styles.statusCompleted };
    case 'failed':
      return { label: 'Failed', className: styles.statusFailed };
    case 'approval_pending':
      return { label: 'Awaiting Approval', className: styles.statusApproval };
    default:
      return { label: status, className: '' };
  }
};

export const TaskExecutionPanel: React.FC<TaskExecutionPanelProps> = ({
  taskId,
  threadId,
  events,
  metrics,
  pendingApproval,
  status,
  onApprove,
  onReject,
  onCancel,
}) => {
  const [isApproving, setIsApproving] = useState(false);
  const [showDiff, setShowDiff] = useState(false);
  const statusBadge = getStatusBadge(status);

  const handleApprove = useCallback(async () => {
    if (!pendingApproval || !onApprove) return;
    setIsApproving(true);
    try {
      await onApprove(pendingApproval.id);
    } finally {
      setIsApproving(false);
    }
  }, [pendingApproval, onApprove]);

  const handleReject = useCallback(async () => {
    if (!pendingApproval || !onReject) return;
    setIsApproving(true);
    try {
      await onReject(pendingApproval.id);
    } finally {
      setIsApproving(false);
    }
  }, [pendingApproval, onReject]);

  // Sound notification for pending approvals
  useEffect(() => {
    if (pendingApproval && status === 'approval_pending') {
      // Play a subtle notification sound
      try {
        const audio = new Audio('/notification.mp3');
        audio.volume = 0.3;
        audio.play().catch(() => {
          // Audio play may fail if not allowed by browser
        });
      } catch {
        // Ignore audio errors
      }
    }
  }, [pendingApproval, status]);

  return (
    <div className={styles.taskExecutionPanel}>
      {/* Header */}
      <div className={styles.panelHeader}>
        <div className={styles.headerLeft}>
          <h3 className={styles.taskTitle}>Task: {taskId}</h3>
          {threadId && (
            <span className={styles.threadId}>Thread: {threadId}</span>
          )}
        </div>
        <div className={styles.headerRight}>
          <span className={`${styles.statusBadge} ${statusBadge.className}`}>
            {statusBadge.label}
          </span>
          {status === 'running' && onCancel && (
            <button className={styles.cancelButton} onClick={onCancel}>
              Cancel
            </button>
          )}
        </div>
      </div>

      {/* Resource Metrics */}
      <div className={styles.metricsSection}>
        <ResourceMetrics metrics={metrics} />
      </div>

      {/* Approval Request */}
      {pendingApproval && (
        <div className={styles.approvalSection}>
          <div className={styles.approvalHeader}>
            <span className={styles.approvalIcon}>Approval Required</span>
            <span className={styles.approvalType}>{pendingApproval.type}</span>
          </div>
          <div className={styles.approvalContent}>
            <p className={styles.approvalDescription}>{pendingApproval.description}</p>

            {pendingApproval.files_changed && pendingApproval.files_changed.length > 0 && (
              <div className={styles.filesChanged}>
                <button
                  className={styles.toggleButton}
                  onClick={() => setShowDiff(!showDiff)}
                >
                  {showDiff ? 'Hide' : 'Show'} Changed Files ({pendingApproval.files_changed.length})
                </button>
                {showDiff && (
                  <ul className={styles.fileList}>
                    {pendingApproval.files_changed.map((file, i) => (
                      <li key={i}>{file}</li>
                    ))}
                  </ul>
                )}
              </div>
            )}

            {showDiff && pendingApproval.diff_summary && (
              <pre className={styles.diffSummary}>{pendingApproval.diff_summary}</pre>
            )}

            <div className={styles.approvalActions}>
              <button
                className={styles.approveButton}
                onClick={handleApprove}
                disabled={isApproving}
              >
                {isApproving ? 'Processing...' : 'Approve'}
              </button>
              <button
                className={styles.rejectButton}
                onClick={handleReject}
                disabled={isApproving}
              >
                {isApproving ? 'Processing...' : 'Reject'}
              </button>
            </div>

            {pendingApproval.timeout_at && (
              <div className={styles.timeout}>
                Expires: {new Date(pendingApproval.timeout_at).toLocaleTimeString()}
              </div>
            )}
          </div>
        </div>
      )}

      {/* Streaming Log */}
      <div className={styles.logSection}>
        <div className={styles.logHeader}>
          <span>Live Output</span>
          <span className={styles.eventCount}>{events.length} events</span>
        </div>
        <StreamingLog events={events} />
      </div>
    </div>
  );
};

export default TaskExecutionPanel;
