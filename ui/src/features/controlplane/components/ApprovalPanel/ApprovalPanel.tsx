/**
 * ApprovalPanel - Dedicated side panel for approval workflow
 * Shows pending approvals with batch approve/reject actions
 */
import React, { useState, useCallback } from 'react';
import { useApprovals, Approval } from '../../../../hooks/useObservatory';
import { ApprovalDetailModal, ApprovalData } from '../../../approvals/ApprovalDetailModal';
import styles from './ApprovalPanel.module.css';

// Format time remaining until expiry
function formatTimeRemaining(expiresAt?: string): string {
  if (!expiresAt) return '';
  const now = new Date();
  const expiry = new Date(expiresAt);
  const diffMs = expiry.getTime() - now.getTime();

  if (diffMs <= 0) return 'Expired';
  if (diffMs < 60000) return '<1m';
  if (diffMs < 3600000) return `${Math.floor(diffMs / 60000)}m`;
  if (diffMs < 86400000) return `${Math.floor(diffMs / 3600000)}h`;
  return `${Math.floor(diffMs / 86400000)}d`;
}

// Format date for display
function formatDate(dateStr?: string): string {
  if (!dateStr) return '-';
  const date = new Date(dateStr);
  return date.toLocaleString(undefined, {
    month: 'short',
    day: 'numeric',
    hour: '2-digit',
    minute: '2-digit',
  });
}

// Get status badge style
function getStatusClass(status: string): string {
  switch (status) {
    case 'pending': return styles.statusPending;
    case 'approved': return styles.statusApproved;
    case 'rejected': return styles.statusRejected;
    default: return styles.statusDefault;
  }
}

export interface ApprovalPanelProps {
  isOpen: boolean;
  onClose: () => void;
  onApprovalClick?: (approval: Approval) => void;
  selectedTaskId?: string | null;
}

export const ApprovalPanel: React.FC<ApprovalPanelProps> = ({
  isOpen,
  onClose,
  onApprovalClick,
  selectedTaskId,
}) => {
  const { approvals, loading, error, refresh, approveApproval, rejectApproval } = useApprovals({ status: 'pending' });
  const [selectedIds, setSelectedIds] = useState<Set<string>>(new Set());
  const [filterAgent, setFilterAgent] = useState<string>('');
  const [processing, setProcessing] = useState(false);
  const [actionError, setActionError] = useState<string | null>(null);

  // Modal state for detailed review
  const [selectedApproval, setSelectedApproval] = useState<Approval | null>(null);
  const [isModalOpen, setIsModalOpen] = useState(false);

  // Handle approval click - open modal
  const handleApprovalClick = useCallback((approval: Approval) => {
    setSelectedApproval(approval);
    setIsModalOpen(true);
    onApprovalClick?.(approval);
  }, [onApprovalClick]);

  // Handle modal approve
  const handleModalApprove = useCallback(async (id: string) => {
    await approveApproval(id);
    setIsModalOpen(false);
    setSelectedApproval(null);
  }, [approveApproval]);

  // Handle modal reject
  const handleModalReject = useCallback(async (id: string, notes: string) => {
    await rejectApproval(id, notes);
    setIsModalOpen(false);
    setSelectedApproval(null);
  }, [rejectApproval]);

  // Handle modal cancel (permanent rejection)
  const handleModalCancel = useCallback(async (id: string, notes?: string) => {
    await rejectApproval(id, notes || '', true); // permanent=true
    setIsModalOpen(false);
    setSelectedApproval(null);
  }, [rejectApproval]);

  // Toggle selection
  const toggleSelection = useCallback((id: string) => {
    setSelectedIds(prev => {
      const next = new Set(prev);
      if (next.has(id)) {
        next.delete(id);
      } else {
        next.add(id);
      }
      return next;
    });
  }, []);

  // Toggle all
  const toggleAll = useCallback(() => {
    if (selectedIds.size === approvals.length) {
      setSelectedIds(new Set());
    } else {
      setSelectedIds(new Set(approvals.map(a => a.id)));
    }
  }, [approvals, selectedIds.size]);

  // Filter approvals
  const filteredApprovals = filterAgent
    ? approvals.filter(a => a.thread_id?.includes(filterAgent) || a.summary?.includes(filterAgent))
    : approvals;

  // Batch approve
  const handleBatchApprove = useCallback(async () => {
    if (selectedIds.size === 0) return;
    setProcessing(true);
    setActionError(null);
    try {
      for (const id of selectedIds) {
        await approveApproval(id);
      }
      setSelectedIds(new Set());
    } catch (err) {
      setActionError(err instanceof Error ? err.message : 'Failed to approve');
    } finally {
      setProcessing(false);
    }
  }, [selectedIds, approveApproval]);

  // Batch reject
  const handleBatchReject = useCallback(async () => {
    if (selectedIds.size === 0) return;
    setProcessing(true);
    setActionError(null);
    try {
      for (const id of selectedIds) {
        await rejectApproval(id);
      }
      setSelectedIds(new Set());
    } catch (err) {
      setActionError(err instanceof Error ? err.message : 'Failed to reject');
    } finally {
      setProcessing(false);
    }
  }, [selectedIds, rejectApproval]);

  if (!isOpen) return null;

  return (
    <div className={styles.panel}>
      {/* Header */}
      <div className={styles.header}>
        <div className={styles.headerTitle}>
          <span className={styles.headerIcon}>📋</span>
          Pending Approvals
          {approvals.length > 0 && (
            <span className={styles.headerBadge}>{approvals.length}</span>
          )}
        </div>
        <button className={styles.closeBtn} onClick={onClose} title="Close">
          ×
        </button>
      </div>

      {/* Toolbar */}
      <div className={styles.toolbar}>
        <label className={styles.selectAll}>
          <input
            type="checkbox"
            checked={selectedIds.size === approvals.length && approvals.length > 0}
            onChange={toggleAll}
            disabled={approvals.length === 0}
          />
          <span>All</span>
        </label>
        <input
          type="text"
          placeholder="Filter..."
          value={filterAgent}
          onChange={(e) => setFilterAgent(e.target.value)}
          className={styles.filterInput}
        />
        <button
          className={styles.refreshBtn}
          onClick={() => refresh()}
          title="Refresh"
          disabled={loading}
        >
          ↻
        </button>
      </div>

      {/* Error display */}
      {(error || actionError) && (
        <div className={styles.error}>
          {error || actionError}
        </div>
      )}

      {/* Approval list */}
      <div className={styles.list}>
        {loading ? (
          <div className={styles.loading}>
            <div className={styles.spinner} />
            <span>Loading approvals...</span>
          </div>
        ) : filteredApprovals.length === 0 ? (
          <div className={styles.empty}>
            <span className={styles.emptyIcon}>✓</span>
            <span className={styles.emptyText}>No pending approvals</span>
          </div>
        ) : (
          filteredApprovals.map((approval) => (
            <div
              key={approval.id}
              className={`${styles.item} ${selectedIds.has(approval.id) ? styles.itemSelected : ''} ${approval.task_id === selectedTaskId ? styles.itemHighlighted : ''}`}
              onClick={() => handleApprovalClick(approval)}
            >
              <div className={styles.itemHeader}>
                <input
                  type="checkbox"
                  checked={selectedIds.has(approval.id)}
                  onChange={(e) => {
                    e.stopPropagation();
                    toggleSelection(approval.id);
                  }}
                  onClick={(e) => e.stopPropagation()}
                />
                <span className={styles.itemId}>{approval.id.substring(0, 8)}...</span>
                <span className={getStatusClass(approval.status)}>{approval.status}</span>
              </div>
              <div className={styles.itemSummary}>
                {approval.summary || approval.request_type || 'Task completion'}
              </div>
              <div className={styles.itemMeta}>
                {approval.thread_id && (
                  <span className={styles.itemAgent}>{approval.thread_id}</span>
                )}
                {approval.expires_at && (
                  <span className={styles.itemExpiry}>
                    ⏱ {formatTimeRemaining(approval.expires_at)}
                  </span>
                )}
                <span className={styles.itemDate}>{formatDate(approval.created_at)}</span>
              </div>
              {approval.branch_name && (
                <div className={styles.itemBranch}>
                  <span className={styles.branchIcon}>⎇</span>
                  {approval.branch_name}
                </div>
              )}
            </div>
          ))
        )}
      </div>

      {/* Action bar */}
      <div className={styles.actions}>
        <button
          className={`${styles.actionBtn} ${styles.approveBtn}`}
          onClick={handleBatchApprove}
          disabled={selectedIds.size === 0 || processing}
        >
          {processing ? '...' : `Approve (${selectedIds.size})`}
        </button>
        <button
          className={`${styles.actionBtn} ${styles.rejectBtn}`}
          onClick={handleBatchReject}
          disabled={selectedIds.size === 0 || processing}
        >
          {processing ? '...' : `Reject (${selectedIds.size})`}
        </button>
      </div>

      {/* Detail Modal */}
      {selectedApproval && (
        <ApprovalDetailModal
          approval={selectedApproval as ApprovalData}
          isOpen={isModalOpen}
          onClose={() => {
            setIsModalOpen(false);
            setSelectedApproval(null);
          }}
          onApprove={handleModalApprove}
          onReject={handleModalReject}
          onCancel={handleModalCancel}
        />
      )}
    </div>
  );
};

export default ApprovalPanel;
