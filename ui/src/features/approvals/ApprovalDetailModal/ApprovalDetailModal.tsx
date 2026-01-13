/**
 * ApprovalDetailModal.tsx - Full-screen approval review modal
 *
 * Provides comprehensive review experience with:
 * - File tree sidebar
 * - Diff viewer (unified/split toggle)
 * - Description tab with markdown rendering
 * - Logs tab for execution history
 */
/* eslint-disable @typescript-eslint/no-use-before-define */

import React, { useState, useEffect, useCallback, useMemo } from 'react';
import ReactMarkdown from 'react-markdown';
import { DiffViewer, ViewMode, parseDiff, extractFileChanges } from '../../../components/DiffViewer';
import { FileTree } from '../../../components/FileTree';
import { TaskStreamEvent } from '../../../types';
import styles from './ApprovalDetailModal.module.css';

type TabType = 'files' | 'description' | 'logs';

// Union type to support both PendingApprovalRequest and the Approval from useObservatory
export interface ApprovalData {
  id: string;
  task_id: string;
  type?: string;           // From PendingApprovalRequest
  request_type?: string;   // From useObservatory Approval
  description?: string;    // From PendingApprovalRequest
  summary?: string;        // From useObservatory Approval
  status: string;
  created_at: string;
  timeout_at?: string;     // From PendingApprovalRequest
  expires_at?: string;     // From useObservatory Approval
  context_json?: string;
  files_changed?: string[];
  diff_summary?: string;
  diff_preview?: string;
  branch_name?: string;
  worktree_path?: string;
}

interface ApprovalDetailModalProps {
  approval: ApprovalData;
  isOpen: boolean;
  onClose: () => void;
  onApprove: (id: string, notes?: string) => Promise<void>;
  onReject: (id: string, notes: string) => Promise<void>;
  // Optional: pre-loaded diff data
  diff?: string;
  // Optional: task events for logs tab
  events?: TaskStreamEvent[];
}

export const ApprovalDetailModal: React.FC<ApprovalDetailModalProps> = ({
  approval,
  isOpen,
  onClose,
  onApprove,
  onReject,
  diff: propDiff,
  events = [],
}) => {
  const [activeTab, setActiveTab] = useState<TabType>('files');
  const [viewMode, setViewMode] = useState<ViewMode>('unified');
  const [selectedFile, setSelectedFile] = useState<string | undefined>();
  const [diff, setDiff] = useState<string>(propDiff || '');
  const [isLoading, setIsLoading] = useState(!propDiff);
  const [isSubmitting, setIsSubmitting] = useState(false);
  const [rejectionNotes, setRejectionNotes] = useState('');
  const [showRejectForm, setShowRejectForm] = useState(false);
  const [error, setError] = useState<string | null>(null);

  // Fetch diff if not provided
  useEffect(() => {
    if (!propDiff && isOpen && approval.task_id) {
      setIsLoading(true);
      fetch(`/api/coordinator/tasks/${approval.task_id}/diff`)
        .then((res) => {
          if (!res.ok) throw new Error(`Failed to fetch diff: ${res.status}`);
          return res.json();
        })
        .then((data) => {
          setDiff(data.diff || '');
          setIsLoading(false);
        })
        .catch((err) => {
          console.error('Failed to fetch diff:', err);
          setError(err.message);
          setIsLoading(false);
        });
    }
  }, [propDiff, isOpen, approval.task_id]);

  // Parse diff for file tree
  const parsedDiff = useMemo(() => parseDiff(diff), [diff]);
  const fileChanges = useMemo(() => extractFileChanges(parsedDiff), [parsedDiff]);

  // Keyboard shortcuts
  useEffect(() => {
    if (!isOpen) return;

    const handleKeyDown = (e: KeyboardEvent) => {
      // ESC to close
      if (e.key === 'Escape') {
        if (showRejectForm) {
          setShowRejectForm(false);
        } else {
          onClose();
        }
        return;
      }

      // j/k for file navigation
      if (e.key === 'j' || e.key === 'k') {
        if (fileChanges.length === 0) return;
        const currentIndex = selectedFile
          ? fileChanges.findIndex((f) => f.path === selectedFile)
          : -1;
        const newIndex =
          e.key === 'j'
            ? Math.min(currentIndex + 1, fileChanges.length - 1)
            : Math.max(currentIndex - 1, 0);
        setSelectedFile(fileChanges[newIndex]?.path);
      }

      // 1/2/3 for tab switching
      if (e.key === '1') setActiveTab('files');
      if (e.key === '2') setActiveTab('description');
      if (e.key === '3') setActiveTab('logs');
    };

    window.addEventListener('keydown', handleKeyDown);
    return () => window.removeEventListener('keydown', handleKeyDown);
  }, [isOpen, onClose, fileChanges, selectedFile, showRejectForm]);

  const handleApprove = useCallback(async () => {
    setIsSubmitting(true);
    setError(null);
    try {
      await onApprove(approval.id);
      onClose();
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to approve');
    } finally {
      setIsSubmitting(false);
    }
  }, [approval.id, onApprove, onClose]);

  const handleReject = useCallback(async () => {
    if (!rejectionNotes.trim()) {
      setError('Please provide a reason for rejection');
      return;
    }
    setIsSubmitting(true);
    setError(null);
    try {
      await onReject(approval.id, rejectionNotes);
      onClose();
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to reject');
    } finally {
      setIsSubmitting(false);
    }
  }, [approval.id, rejectionNotes, onReject, onClose]);

  if (!isOpen) return null;

  return (
    <div className={styles.overlay} onClick={onClose}>
      <div className={styles.modal} onClick={(e) => e.stopPropagation()}>
        {/* Header */}
        <div className={styles.header}>
          <div className={styles.headerLeft}>
            <button className={styles.closeButton} onClick={onClose} title="Close (Esc)">
              &times;
            </button>
            <div className={styles.titleSection}>
              <h2 className={styles.title}>Review: {approval.description || approval.summary || 'Task Approval'}</h2>
              <span className={styles.taskId}>Task: {approval.task_id}</span>
              {approval.branch_name && (
                <span className={styles.branchName}>Branch: {approval.branch_name}</span>
              )}
            </div>
          </div>
          <div className={styles.headerRight}>
            <span className={`${styles.typeBadge} ${styles[`type${capitalize(approval.type || approval.request_type || 'merge')}`]}`}>
              {approval.type || approval.request_type || 'merge'}
            </span>
            {!showRejectForm && (
              <>
                <button
                  className={styles.approveButton}
                  onClick={handleApprove}
                  disabled={isSubmitting}
                >
                  {isSubmitting ? 'Processing...' : 'Approve'}
                </button>
                <button
                  className={styles.rejectButton}
                  onClick={() => setShowRejectForm(true)}
                  disabled={isSubmitting}
                >
                  Reject
                </button>
              </>
            )}
          </div>
        </div>

        {/* Rejection form */}
        {showRejectForm && (
          <div className={styles.rejectForm}>
            <label className={styles.rejectLabel}>Reason for rejection:</label>
            <textarea
              className={styles.rejectTextarea}
              value={rejectionNotes}
              onChange={(e) => setRejectionNotes(e.target.value)}
              placeholder="Please explain why this change is being rejected..."
              autoFocus
            />
            <div className={styles.rejectActions}>
              <button
                className={styles.rejectConfirmButton}
                onClick={handleReject}
                disabled={isSubmitting || !rejectionNotes.trim()}
              >
                {isSubmitting ? 'Rejecting...' : 'Confirm Rejection'}
              </button>
              <button
                className={styles.rejectCancelButton}
                onClick={() => setShowRejectForm(false)}
              >
                Cancel
              </button>
            </div>
          </div>
        )}

        {/* Error message */}
        {error && (
          <div className={styles.error}>
            {error}
            <button onClick={() => setError(null)}>&times;</button>
          </div>
        )}

        {/* Tabs */}
        <div className={styles.tabs}>
          <div className={styles.tabList}>
            <button
              className={`${styles.tab} ${activeTab === 'files' ? styles.active : ''}`}
              onClick={() => setActiveTab('files')}
            >
              Files ({fileChanges.length})
            </button>
            <button
              className={`${styles.tab} ${activeTab === 'description' ? styles.active : ''}`}
              onClick={() => setActiveTab('description')}
            >
              Description
            </button>
            <button
              className={`${styles.tab} ${activeTab === 'logs' ? styles.active : ''}`}
              onClick={() => setActiveTab('logs')}
            >
              Logs ({events.length})
            </button>
          </div>

          {activeTab === 'files' && (
            <div className={styles.viewModeToggle}>
              <button
                className={`${styles.viewModeButton} ${viewMode === 'unified' ? styles.active : ''}`}
                onClick={() => setViewMode('unified')}
              >
                Unified
              </button>
              <button
                className={`${styles.viewModeButton} ${viewMode === 'split' ? styles.active : ''}`}
                onClick={() => setViewMode('split')}
              >
                Split
              </button>
            </div>
          )}
        </div>

        {/* Content */}
        <div className={styles.content}>
          {isLoading ? (
            <div className={styles.loading}>Loading diff...</div>
          ) : (
            <>
              {activeTab === 'files' && (
                <div className={styles.filesView}>
                  {/* File Tree Sidebar */}
                  <div className={styles.sidebar}>
                    <FileTree
                      files={fileChanges}
                      selectedFile={selectedFile}
                      onSelectFile={setSelectedFile}
                    />
                  </div>

                  {/* Diff Viewer */}
                  <div className={styles.diffArea}>
                    <DiffViewer
                      diff={diff}
                      viewMode={viewMode}
                      selectedFile={selectedFile}
                      onFileClick={setSelectedFile}
                    />
                  </div>
                </div>
              )}

              {activeTab === 'description' && (
                <div className={styles.descriptionView}>
                  <div className={styles.markdownContent}>
                    <ReactMarkdown>{approval.description || approval.summary || 'No description available'}</ReactMarkdown>
                  </div>
                  {approval.context_json && (
                    <div className={styles.contextSection}>
                      <h3>Context</h3>
                      <pre className={styles.contextJson}>
                        {JSON.stringify(JSON.parse(approval.context_json), null, 2)}
                      </pre>
                    </div>
                  )}
                  {approval.worktree_path && (
                    <div className={styles.contextSection}>
                      <h3>Worktree</h3>
                      <code className={styles.worktreePath}>{approval.worktree_path}</code>
                    </div>
                  )}
                </div>
              )}

              {activeTab === 'logs' && (
                <div className={styles.logsView}>
                  {events.length === 0 ? (
                    <div className={styles.emptyLogs}>No execution logs available</div>
                  ) : (
                    <div className={styles.eventList}>
                      {events.map((event, index) => (
                        <EventItem key={index} event={event} />
                      ))}
                    </div>
                  )}
                </div>
              )}
            </>
          )}
        </div>

        {/* Footer with shortcuts hint */}
        <div className={styles.footer}>
          <span className={styles.shortcuts}>
            <kbd>Esc</kbd> Close
            <kbd>j</kbd>/<kbd>k</kbd> Navigate files
            <kbd>1</kbd>-<kbd>3</kbd> Switch tabs
          </span>
          {(approval.timeout_at || approval.expires_at) && (
            <span className={styles.timeout}>
              Expires: {new Date(approval.timeout_at || approval.expires_at!).toLocaleString()}
            </span>
          )}
        </div>
      </div>
    </div>
  );
};

interface EventItemProps {
  event: TaskStreamEvent;
}

const EventItem: React.FC<EventItemProps> = ({ event }) => {
  const typeClass = styles[`event${capitalize(event.stream_type)}`] || '';

  return (
    <div className={`${styles.eventItem} ${typeClass}`}>
      <span className={styles.eventType}>{event.stream_type}</span>
      {event.turn_num !== undefined && (
        <span className={styles.eventTurn}>Turn {event.turn_num}</span>
      )}
      {event.tool_name && (
        <span className={styles.eventTool}>{event.tool_name}</span>
      )}
      {event.text && (
        <div className={styles.eventText}>{event.text}</div>
      )}
      {event.error_msg && (
        <div className={styles.eventError}>{event.error_msg}</div>
      )}
      {event.timestamp && (
        <span className={styles.eventTime}>
          {new Date(event.timestamp).toLocaleTimeString()}
        </span>
      )}
    </div>
  );
};

function capitalize(str: string): string {
  return str.charAt(0).toUpperCase() + str.slice(1);
}

export default ApprovalDetailModal;
