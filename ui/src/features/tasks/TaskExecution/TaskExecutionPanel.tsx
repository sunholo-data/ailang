import React, { useState, useCallback, useEffect } from 'react';
import {
  TaskStreamEvent,
  TaskResourceMetrics,
  PendingApprovalRequest,
} from '../../../types';
import { StreamingLog } from './StreamingLog';
import { ResourceMetrics } from './ResourceMetrics';
import { useTaskStream } from '../../../hooks/useTaskStream';
import { DiffViewer } from '../../../components/DiffViewer';
import { ApprovalDetailModal, ApprovalData } from '../../approvals/ApprovalDetailModal';
import styles from './TaskExecution.module.css';

interface TaskExecutionPanelProps {
  taskId: string;
  threadId?: string;
  // Optional overrides - if not provided, uses useTaskStream hook
  events?: TaskStreamEvent[];
  metrics?: TaskResourceMetrics | null;
  pendingApproval?: PendingApprovalRequest | null;
  status?: 'pending' | 'running' | 'completed' | 'failed' | 'approval_pending';
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
  events: propEvents,
  metrics: propMetrics,
  pendingApproval: propPendingApproval,
  status: propStatus,
  onApprove,
  onReject,
  onCancel,
}) => {
  // Use the task stream hook for real-time WebSocket updates
  const taskStream = useTaskStream({ taskId });

  // Use props if provided, otherwise use hook data
  const events = propEvents ?? taskStream.events;
  const metrics = propMetrics ?? taskStream.metrics;
  const pendingApproval = propPendingApproval ?? taskStream.pendingApproval;
  const status = propStatus ?? (taskStream.status as 'pending' | 'running' | 'completed' | 'failed' | 'approval_pending');

  const [isApproving, setIsApproving] = useState(false);
  const [showDiff, setShowDiff] = useState(false);
  const [isModalOpen, setIsModalOpen] = useState(false);
  const [fullDiff, setFullDiff] = useState<string>('');
  const statusBadge = getStatusBadge(status);

  // Fetch full diff when modal opens
  const handleOpenModal = useCallback(async () => {
    if (pendingApproval?.task_id) {
      try {
        const res = await fetch(`/api/coordinator/tasks/${pendingApproval.task_id}/diff`);
        if (res.ok) {
          const data = await res.json();
          setFullDiff(data.diff || '');
        }
      } catch {
        // Use diff_summary as fallback
        setFullDiff(pendingApproval.diff_summary || '');
      }
    }
    setIsModalOpen(true);
  }, [pendingApproval]);

  const handleApprove = useCallback(async () => {
    if (!pendingApproval) return;
    setIsApproving(true);
    try {
      if (onApprove) {
        await onApprove(pendingApproval.id);
      } else {
        await taskStream.approve();
      }
    } finally {
      setIsApproving(false);
    }
  }, [pendingApproval, onApprove, taskStream]);

  const handleReject = useCallback(async () => {
    if (!pendingApproval) return;
    setIsApproving(true);
    try {
      if (onReject) {
        await onReject(pendingApproval.id);
      } else {
        await taskStream.reject();
      }
    } finally {
      setIsApproving(false);
    }
  }, [pendingApproval, onReject, taskStream]);

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

            {/* File changes toggle and diff preview */}
            {(pendingApproval.files_changed?.length || pendingApproval.diff_summary) && (
              <div className={styles.filesChanged}>
                <div className={styles.diffControls}>
                  <button
                    className={styles.toggleButton}
                    onClick={() => setShowDiff(!showDiff)}
                  >
                    {showDiff ? 'Hide' : 'Show'} Changes
                    {pendingApproval.files_changed?.length && ` (${pendingApproval.files_changed.length} files)`}
                  </button>
                  <button
                    className={styles.fullReviewButton}
                    onClick={handleOpenModal}
                  >
                    Full Review
                  </button>
                </div>
                {showDiff && pendingApproval.diff_summary && (
                  <div className={styles.diffPreview}>
                    <DiffViewer
                      diff={pendingApproval.diff_summary}
                      viewMode="unified"
                      compact
                      maxHeight="300px"
                    />
                  </div>
                )}
              </div>
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

      {/* Full Review Modal */}
      {pendingApproval && isModalOpen && (
        <ApprovalDetailModal
          approval={pendingApproval as ApprovalData}
          isOpen={isModalOpen}
          onClose={() => setIsModalOpen(false)}
          onApprove={async (id) => {
            await handleApprove();
          }}
          onReject={async (id, notes) => {
            await handleReject();
          }}
          diff={fullDiff}
          events={events}
        />
      )}
    </div>
  );
};

export default TaskExecutionPanel;
