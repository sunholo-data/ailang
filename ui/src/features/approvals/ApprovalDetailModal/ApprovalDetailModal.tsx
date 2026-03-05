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
import { DiffViewer, ViewMode, parseDiff, extractFileChanges, findMarkdownFiles, extractNewFileContent, FileDiff } from '../../../components/DiffViewer';
import { FileTree } from '../../../components/FileTree';
import { TaskStreamEvent } from '../../../types';
import { IterationBadge, FeedbackInput, ChannelBadge } from '../components';
import styles from './ApprovalDetailModal.module.css';

type TabType = 'description' | 'files' | 'logs';

// Task event from coordinator API
interface TaskEvent {
  id: string;
  task_id: string;
  stream_type: string;  // text, tool_use, tool_result, error, status, turn_start, turn_end
  turn_num: number;
  text?: string;
  tool_name?: string;
  tool_input?: string;
  tool_output?: string;
  error_msg?: string;
  created_at: string;
}

interface Turn {
  turnNumber: number;
  events: TaskEvent[];
  startTime?: string;
}

/**
 * Consolidate consecutive text events into single message blocks.
 * Streaming text comes as many small chunks - we merge them for display.
 */
function consolidateTextEvents(events: TaskEvent[]): TaskEvent[] {
  const result: TaskEvent[] = [];
  let currentTextBlock: TaskEvent | null = null;

  for (const event of events) {
    if (event.stream_type === 'text' && event.text) {
      if (currentTextBlock) {
        // Append to existing text block
        currentTextBlock = {
          ...currentTextBlock,
          text: (currentTextBlock.text || '') + event.text,
        };
      } else {
        // Start new text block
        currentTextBlock = { ...event };
      }
    } else {
      // Non-text event: flush any pending text block
      if (currentTextBlock) {
        result.push(currentTextBlock);
        currentTextBlock = null;
      }
      result.push(event);
    }
  }

  // Flush final text block
  if (currentTextBlock) {
    result.push(currentTextBlock);
  }

  return result;
}

/**
 * Group events by turn number
 */
function groupEventsByTurn(events: TaskEvent[]): Turn[] {
  const turnMap = new Map<number, TaskEvent[]>();

  for (const event of events) {
    const turnNum = event.turn_num || 0;
    if (!turnMap.has(turnNum)) {
      turnMap.set(turnNum, []);
    }
    turnMap.get(turnNum)!.push(event);
  }

  const turns: Turn[] = [];
  const sortedKeys = Array.from(turnMap.keys()).sort((a, b) => a - b);

  for (const turnNum of sortedKeys) {
    const turnEvents = turnMap.get(turnNum)!;
    // Sort by time, then consolidate consecutive text events
    const sortedEvents = turnEvents.sort((a, b) =>
      new Date(a.created_at).getTime() - new Date(b.created_at).getTime()
    );
    const consolidatedEvents = consolidateTextEvents(sortedEvents);

    turns.push({
      turnNumber: turnNum,
      events: consolidatedEvents,
      startTime: turnEvents[0]?.created_at,
    });
  }

  return turns;
}

/**
 * Format timestamp for display
 */
function formatTime(timestamp: string | undefined): string {
  if (!timestamp) return '';
  try {
    return new Date(timestamp).toLocaleTimeString();
  } catch {
    return '';
  }
}

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
  // Multi-channel approval workflow fields (M-DASHBOARD-APPROVAL-INTEGRATION)
  iteration?: number;        // Current iteration (1-3), retrigger count
  channel?: string;          // Source: 'dashboard' | 'cli' | 'github'
  feedback?: string;         // Harvested feedback from GitHub comments
  feedback_author?: string;  // Author of the most recent feedback
}

interface ApprovalDetailModalProps {
  approval: ApprovalData;
  isOpen: boolean;
  onClose: () => void;
  onApprove: (id: string, notes?: string) => Promise<void>;
  onReject: (id: string, notes: string) => Promise<void>;
  onCancel?: (id: string, notes?: string) => Promise<void>; // Permanent rejection, no retry
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
  onCancel,
  diff: propDiff,
  events = [],
}) => {
  const [activeTab, setActiveTab] = useState<TabType>('description');
  const [viewMode, setViewMode] = useState<ViewMode>('unified');
  const [selectedFile, setSelectedFile] = useState<string | undefined>();
  const [diff, setDiff] = useState<string>(propDiff || '');
  const [isLoading, setIsLoading] = useState(!propDiff);
  const [isSubmitting, setIsSubmitting] = useState(false);
  const [feedbackNotes, setFeedbackNotes] = useState('');
  const [showFeedbackForm, setShowFeedbackForm] = useState<'approve' | 'reject' | 'cancel' | null>(null);
  const [error, setError] = useState<string | null>(null);

  // State for fetched task events (for Logs tab)
  const [taskEvents, setTaskEvents] = useState<TaskEvent[]>([]);
  const [eventsLoading, setEventsLoading] = useState(false);
  const [expandedTools, setExpandedTools] = useState<Set<string>>(new Set());

  // Fetch events when modal opens (for Logs tab)
  useEffect(() => {
    if (isOpen && approval.task_id) {
      setEventsLoading(true);
      fetch(`/api/coordinator/tasks/${approval.task_id}/events?limit=500`)
        .then((res) => {
          if (!res.ok) throw new Error(`Failed to fetch events: ${res.status}`);
          return res.json();
        })
        .then((data) => {
          setTaskEvents(data.events || []);
          setEventsLoading(false);
        })
        .catch((err) => {
          console.error('Failed to fetch task events:', err);
          setEventsLoading(false);
        });
    }
  }, [isOpen, approval.task_id]);

  // Group events by turn for display
  const turns = useMemo(() => groupEventsByTurn(taskEvents), [taskEvents]);

  // Toggle tool expansion
  const toggleToolExpanded = useCallback((eventId: string) => {
    setExpandedTools(prev => {
      const next = new Set(prev);
      if (next.has(eventId)) {
        next.delete(eventId);
      } else {
        next.add(eventId);
      }
      return next;
    });
  }, []);

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

  // Find markdown files for the description tab
  const markdownFiles = useMemo(() => findMarkdownFiles(parsedDiff), [parsedDiff]);

  // State for markdown file viewer
  const [selectedMarkdownFile, setSelectedMarkdownFile] = useState<FileDiff | null>(null);
  const [markdownViewMode, setMarkdownViewMode] = useState<'rendered' | 'diff'>('rendered');

  // Auto-select first markdown file when available
  useEffect(() => {
    if (markdownFiles.length > 0 && !selectedMarkdownFile) {
      setSelectedMarkdownFile(markdownFiles[0]);
    }
  }, [markdownFiles, selectedMarkdownFile]);

  // Keyboard shortcuts
  useEffect(() => {
    if (!isOpen) return;

    const handleKeyDown = (e: KeyboardEvent) => {
      // ESC to close
      if (e.key === 'Escape') {
        if (showFeedbackForm) {
          setShowFeedbackForm(null);
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
      if (e.key === '1') setActiveTab('description');
      if (e.key === '2') setActiveTab('files');
      if (e.key === '3') setActiveTab('logs');
    };

    window.addEventListener('keydown', handleKeyDown);
    return () => window.removeEventListener('keydown', handleKeyDown);
  }, [isOpen, onClose, fileChanges, selectedFile, showFeedbackForm]);

  const handleApprove = useCallback(async () => {
    setIsSubmitting(true);
    setError(null);
    try {
      await onApprove(approval.id, feedbackNotes.trim() || undefined);
      onClose();
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to approve');
    } finally {
      setIsSubmitting(false);
    }
  }, [approval.id, feedbackNotes, onApprove, onClose]);

  const handleReject = useCallback(async () => {
    if (!feedbackNotes.trim()) {
      setError('Please provide a reason for rejection');
      return;
    }
    setIsSubmitting(true);
    setError(null);
    try {
      await onReject(approval.id, feedbackNotes);
      onClose();
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to reject');
    } finally {
      setIsSubmitting(false);
    }
  }, [approval.id, feedbackNotes, onReject, onClose]);

  const handleCancel = useCallback(async () => {
    if (!onCancel) return;
    setIsSubmitting(true);
    setError(null);
    try {
      await onCancel(approval.id, feedbackNotes || undefined);
      onClose();
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to cancel task');
    } finally {
      setIsSubmitting(false);
    }
  }, [approval.id, feedbackNotes, onCancel, onClose]);

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
            {approval.iteration && approval.iteration > 1 && (
              <IterationBadge iteration={approval.iteration} />
            )}
            {approval.channel && (
              <ChannelBadge channel={approval.channel} />
            )}
            <span className={`${styles.typeBadge} ${styles[`type${capitalize(approval.type || approval.request_type || 'merge')}`]}`}>
              {approval.type || approval.request_type || 'merge'}
            </span>
            {!showFeedbackForm && (
              <>
                <button
                  className={styles.approveButton}
                  onClick={() => setShowFeedbackForm('approve')}
                  disabled={isSubmitting}
                >
                  Approve
                </button>
                <button
                  className={styles.rejectButton}
                  onClick={() => setShowFeedbackForm('reject')}
                  disabled={isSubmitting}
                  title="Reject with feedback - agent will retry"
                >
                  Reject
                </button>
                {onCancel && (
                  <button
                    className={styles.cancelTaskButton}
                    onClick={() => setShowFeedbackForm('cancel')}
                    disabled={isSubmitting}
                    title="Cancel permanently - no retry"
                  >
                    Cancel Task
                  </button>
                )}
              </>
            )}
          </div>
        </div>

        {/* Feedback form - used for both approve and reject */}
        {showFeedbackForm && (
          <div className={styles.feedbackForm}>
            {/* Show harvested GitHub feedback if available */}
            {approval.feedback && (
              <div className={styles.harvestedFeedback}>
                <h4>Previous feedback from GitHub</h4>
                <div className={styles.feedbackContent}>
                  <span className={styles.feedbackAuthor}>
                    {approval.feedback_author || 'Unknown'} via GitHub:
                  </span>
                  <p>{approval.feedback}</p>
                </div>
              </div>
            )}
            <FeedbackInput
              value={feedbackNotes}
              onChange={setFeedbackNotes}
              label={
                showFeedbackForm === 'approve' ? 'Approval feedback (optional)' :
                showFeedbackForm === 'cancel' ? 'Cancellation reason (optional)' :
                'Rejection feedback (required)'
              }
              placeholder={
                showFeedbackForm === 'approve'
                  ? 'Add any notes or feedback for the next iteration (optional)...'
                  : showFeedbackForm === 'cancel'
                  ? 'Why are you cancelling this task? (optional)...'
                  : 'Please explain why this change is being rejected and what needs to be fixed...'
              }
              maxLength={1000}
              required={showFeedbackForm === 'reject'}
              autoFocus
              rows={4}
              error={error && error.includes('reason') ? error : undefined}
            />
            <div className={styles.feedbackActions}>
              {showFeedbackForm === 'approve' ? (
                <button
                  className={styles.approveConfirmButton}
                  onClick={handleApprove}
                  disabled={isSubmitting || feedbackNotes.length > 1000}
                >
                  {isSubmitting ? 'Approving...' : 'Confirm Approval'}
                </button>
              ) : showFeedbackForm === 'cancel' ? (
                <button
                  className={styles.cancelTaskConfirmButton}
                  onClick={handleCancel}
                  disabled={isSubmitting || feedbackNotes.length > 1000}
                >
                  {isSubmitting ? 'Cancelling...' : 'Confirm Cancellation'}
                </button>
              ) : (
                <button
                  className={styles.rejectConfirmButton}
                  onClick={handleReject}
                  disabled={isSubmitting || !feedbackNotes.trim() || feedbackNotes.length > 1000}
                >
                  {isSubmitting ? 'Rejecting...' : 'Confirm Rejection'}
                </button>
              )}
              <button
                className={styles.feedbackCancelButton}
                onClick={() => {
                  setShowFeedbackForm(null);
                  setFeedbackNotes('');
                }}
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
              className={`${styles.tab} ${activeTab === 'description' ? styles.active : ''}`}
              onClick={() => setActiveTab('description')}
            >
              Description
            </button>
            <button
              className={`${styles.tab} ${activeTab === 'files' ? styles.active : ''}`}
              onClick={() => setActiveTab('files')}
            >
              Files ({fileChanges.length})
            </button>
            <button
              className={`${styles.tab} ${activeTab === 'logs' ? styles.active : ''}`}
              onClick={() => setActiveTab('logs')}
            >
              Logs ({taskEvents.length > 0 ? turns.length + ' turns' : events.length || '—'})
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
                  {/* Description/Summary */}
                  <div className={styles.markdownContent}>
                    <ReactMarkdown>{approval.description || approval.summary || 'No description available'}</ReactMarkdown>
                  </div>

                  {/* Markdown Files Viewer */}
                  {markdownFiles.length > 0 && (
                    <div className={styles.markdownFilesSection}>
                      <div className={styles.markdownFilesHeader}>
                        <h3>Documentation Changes</h3>
                        <div className={styles.markdownControls}>
                          {/* File selector if multiple markdown files */}
                          {markdownFiles.length > 1 && (
                            <select
                              className={styles.markdownFileSelect}
                              value={selectedMarkdownFile?.newPath || ''}
                              onChange={(e) => {
                                const file = markdownFiles.find(f => f.newPath === e.target.value);
                                setSelectedMarkdownFile(file || null);
                              }}
                            >
                              {markdownFiles.map(file => (
                                <option key={file.newPath} value={file.newPath}>
                                  {file.newPath}
                                </option>
                              ))}
                            </select>
                          )}
                          {/* View mode toggle */}
                          <div className={styles.markdownViewToggle}>
                            <button
                              className={`${styles.viewToggleButton} ${markdownViewMode === 'rendered' ? styles.active : ''}`}
                              onClick={() => setMarkdownViewMode('rendered')}
                            >
                              Rendered
                            </button>
                            <button
                              className={`${styles.viewToggleButton} ${markdownViewMode === 'diff' ? styles.active : ''}`}
                              onClick={() => setMarkdownViewMode('diff')}
                            >
                              Diff
                            </button>
                          </div>
                        </div>
                      </div>

                      {selectedMarkdownFile && (
                        <div className={styles.markdownFileContent}>
                          <div className={styles.markdownFileName}>
                            <span className={`${styles.fileStatus} ${styles[selectedMarkdownFile.status]}`}>
                              {selectedMarkdownFile.status === 'added' ? '+' :
                               selectedMarkdownFile.status === 'deleted' ? '-' :
                               selectedMarkdownFile.status === 'renamed' ? '→' : '~'}
                            </span>
                            {markdownFiles.length === 1 && selectedMarkdownFile.newPath}
                            <span className={styles.fileStats}>
                              <span className={styles.additions}>+{selectedMarkdownFile.additions}</span>
                              <span className={styles.deletions}>-{selectedMarkdownFile.deletions}</span>
                            </span>
                          </div>

                          {markdownViewMode === 'rendered' ? (
                            <div className={styles.renderedMarkdown}>
                              <ReactMarkdown>
                                {extractNewFileContent(selectedMarkdownFile)}
                              </ReactMarkdown>
                            </div>
                          ) : (
                            <div className={styles.markdownDiff}>
                              <DiffViewer
                                diff={`diff --git a/${selectedMarkdownFile.oldPath} b/${selectedMarkdownFile.newPath}\n${selectedMarkdownFile.hunks.map(h => h.lines.map(l =>
                                  l.type === 'hunk' ? l.content :
                                  l.type === 'add' ? '+' + l.content :
                                  l.type === 'delete' ? '-' + l.content :
                                  ' ' + l.content
                                ).join('\n')).join('\n')}`}
                                viewMode="unified"
                                compact
                              />
                            </div>
                          )}
                        </div>
                      )}
                    </div>
                  )}

                  {/* Budget widget for cost-type approvals */}
                  {(approval.type === 'cost' || approval.request_type === 'cost') && approval.context_json && (() => {
                    try {
                      const ctx = JSON.parse(approval.context_json);
                      const usagePercent = ctx.daily_limit > 0
                        ? Math.min((ctx.current_spend / ctx.daily_limit) * 100, 100)
                        : 0;
                      const isOverBudget = ctx.current_spend >= ctx.daily_limit;

                      return (
                        <div className={styles.budgetSection}>
                          <h3>💰 Budget Status</h3>
                          <div className={styles.budgetProvider}>
                            Provider: <strong>{ctx.provider}</strong>
                          </div>
                          <div className={styles.budgetMeter}>
                            <div className={styles.budgetLabels}>
                              <span className={styles.budgetSpend}>
                                ${ctx.current_spend?.toFixed(2) || '0.00'}
                              </span>
                              <span className={styles.budgetLimit}>
                                / ${ctx.daily_limit?.toFixed(2) || '0.00'} daily limit
                              </span>
                            </div>
                            <div className={styles.budgetBarContainer}>
                              <div
                                className={`${styles.budgetBar} ${isOverBudget ? styles.budgetBarExceeded : ''}`}
                                style={{ width: `${usagePercent}%` }}
                              />
                            </div>
                            <div className={styles.budgetPercent}>
                              {usagePercent.toFixed(1)}% used
                              {isOverBudget && <span className={styles.budgetExceeded}> — EXCEEDED</span>}
                            </div>
                          </div>
                          <div className={styles.budgetReason}>
                            <strong>Reason:</strong> {ctx.reason}
                          </div>
                          <div className={styles.budgetWarning}>
                            ⚠️ Approving this task will allow it to proceed despite exceeding the budget limit.
                          </div>
                        </div>
                      );
                    } catch {
                      // Fall back to raw JSON if parsing fails
                      return (
                        <div className={styles.contextSection}>
                          <h3>Context</h3>
                          <pre className={styles.contextJson}>
                            {(() => { try { return JSON.stringify(JSON.parse(approval.context_json), null, 2); } catch { return approval.context_json; } })()}
                          </pre>
                        </div>
                      );
                    }
                  })()}

                  {/* Generic context for non-cost approvals */}
                  {approval.type !== 'cost' && approval.request_type !== 'cost' && approval.context_json && (
                    <div className={styles.contextSection}>
                      <h3>Context</h3>
                      <pre className={styles.contextJson}>
                        {(() => { try { return JSON.stringify(JSON.parse(approval.context_json), null, 2); } catch { return approval.context_json; } })()}
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
                  {eventsLoading ? (
                    <div className={styles.loading}>Loading conversation...</div>
                  ) : taskEvents.length === 0 && events.length === 0 ? (
                    <div className={styles.emptyLogs}>
                      <div>No execution logs available</div>
                      <div className={styles.logsHint}>
                        CLI: <code>ailang coordinator logs {approval.task_id}</code>
                      </div>
                    </div>
                  ) : taskEvents.length > 0 ? (
                    <div className={styles.conversationView}>
                      {turns.map((turn) => (
                        <div key={turn.turnNumber} className={styles.turn}>
                          <div className={styles.turnHeader}>
                            <span className={styles.turnNumber}>Turn {turn.turnNumber}</span>
                            {turn.startTime && (
                              <span className={styles.turnTime}>{formatTime(turn.startTime)}</span>
                            )}
                          </div>
                          <div className={styles.turnEvents}>
                            {turn.events.map((event, idx) => (
                              <ConversationEvent
                                key={event.id || idx}
                                event={event}
                                isExpanded={expandedTools.has(event.id)}
                                onToggle={() => toggleToolExpanded(event.id)}
                              />
                            ))}
                          </div>
                        </div>
                      ))}
                    </div>
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

/**
 * ConversationEvent - Renders a single event in conversation style
 */
interface ConversationEventProps {
  event: TaskEvent;
  isExpanded: boolean;
  onToggle: () => void;
}

const ConversationEvent: React.FC<ConversationEventProps> = ({ event, isExpanded, onToggle }) => {
  // Skip turn_start/turn_end markers - they're just structural
  if (event.stream_type === 'turn_start' || event.stream_type === 'turn_end') {
    return null;
  }

  // Text block - Claude's response
  if (event.stream_type === 'text' && event.text) {
    return (
      <div className={styles.textBlock}>
        <pre className={styles.textContent}>{event.text}</pre>
      </div>
    );
  }

  // Tool use - expandable
  if (event.stream_type === 'tool_use' && event.tool_name) {
    return (
      <div className={styles.toolBlock}>
        <div className={styles.toolHeader} onClick={onToggle}>
          <span className={styles.toolName}>
            <span className={styles.toolIcon}>🔧</span>
            {event.tool_name}
          </span>
          <span className={styles.toolExpand}>{isExpanded ? '▼' : '▶'}</span>
        </div>
        {isExpanded && (
          <div className={styles.toolContent}>
            {event.tool_input && (
              <div className={styles.toolSection}>
                <div className={styles.toolSectionLabel}>Input:</div>
                <pre className={styles.toolJson}>{formatToolContent(event.tool_input)}</pre>
              </div>
            )}
            {event.tool_output && (
              <div className={styles.toolSection}>
                <div className={styles.toolSectionLabel}>Output:</div>
                <pre className={styles.toolJson}>{formatToolContent(event.tool_output)}</pre>
              </div>
            )}
          </div>
        )}
      </div>
    );
  }

  // Error event
  if (event.stream_type === 'error' && event.error_msg) {
    return (
      <div className={styles.errorBlock}>
        <span className={styles.errorIcon}>❌</span>
        <span className={styles.errorText}>{event.error_msg}</span>
      </div>
    );
  }

  // Status event
  if (event.stream_type === 'status') {
    return (
      <div className={styles.statusBlock}>
        <span className={styles.statusIcon}>ℹ️</span>
        <span className={styles.statusText}>{event.text || 'Status update'}</span>
      </div>
    );
  }

  // Fallback for other event types
  return null;
};

/**
 * Format tool input/output for display
 */
function formatToolContent(content: string): string {
  try {
    const parsed = JSON.parse(content);
    return JSON.stringify(parsed, null, 2);
  } catch {
    return content;
  }
}

function capitalize(str: string): string {
  return str.charAt(0).toUpperCase() + str.slice(1);
}

export default ApprovalDetailModal;
