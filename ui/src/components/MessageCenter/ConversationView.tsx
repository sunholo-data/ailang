import React, { useEffect, useRef, useState, useMemo } from 'react';
import ReactMarkdown from 'react-markdown';
import { Message, Thread } from '../../types';

// Maximum characters to display before truncating (10KB)
const MAX_DISPLAY_LENGTH = 10 * 1024;
// Maximum lines to show before truncating
const MAX_DISPLAY_LINES = 200;

interface ConversationViewProps {
  thread?: Thread;
  messages: Message[];
  onSendMessage: (content: string, kind: string, workspace?: string) => void;
  onWorkspaceChange?: (workspace: string) => void;  // Callback to save workspace to thread
  onApproveRequest?: (approvalId: string, notes: string) => void;
  onRejectRequest?: (approvalId: string, notes: string) => void;
}

// Icons
const Icons = {
  send: (
    <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
      <line x1="22" y1="2" x2="11" y2="13" />
      <polygon points="22 2 15 22 11 13 2 9 22 2" />
    </svg>
  ),
  directive: (
    <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
      <path d="M14 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V8z" />
      <polyline points="14 2 14 8 20 8" />
      <line x1="16" y1="13" x2="8" y2="13" />
      <line x1="16" y1="17" x2="8" y2="17" />
    </svg>
  ),
  question: (
    <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
      <circle cx="12" cy="12" r="10" />
      <path d="M9.09 9a3 3 0 0 1 5.83 1c0 2-3 3-3 3" />
      <line x1="12" y1="17" x2="12.01" y2="17" />
    </svg>
  ),
  status: (
    <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
      <path d="M22 12h-4l-3 9L9 3l-3 9H2" />
    </svg>
  ),
  result: (
    <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
      <path d="M22 11.08V12a10 10 0 1 1-5.93-9.14" />
      <polyline points="22 4 12 14.01 9 11.01" />
    </svg>
  ),
  lock: (
    <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
      <rect x="3" y="11" width="18" height="11" rx="2" ry="2" />
      <path d="M7 11V7a5 5 0 0 1 10 0v4" />
    </svg>
  ),
  user: (
    <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
      <path d="M20 21v-2a4 4 0 0 0-4-4H8a4 4 0 0 0-4 4v2" />
      <circle cx="12" cy="7" r="4" />
    </svg>
  ),
  bot: (
    <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
      <rect x="3" y="11" width="18" height="10" rx="2" />
      <circle cx="12" cy="5" r="2" />
      <path d="M12 7v4" />
    </svg>
  ),
  check: (
    <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
      <polyline points="20 6 9 17 4 12" />
    </svg>
  ),
  x: (
    <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
      <line x1="18" y1="6" x2="6" y2="18" />
      <line x1="6" y1="6" x2="18" y2="18" />
    </svg>
  ),
};

const getKindIcon = (kind: string) => {
  switch (kind) {
    case 'directive': return Icons.directive;
    case 'question': return Icons.question;
    case 'status': return Icons.status;
    case 'result': return Icons.result;
    case 'approval_request': return Icons.lock;
    default: return Icons.directive;
  }
};

export const ConversationView: React.FC<ConversationViewProps> = ({
  thread,
  messages,
  onSendMessage,
  onWorkspaceChange,
  onApproveRequest,
  onRejectRequest,
}) => {
  const messagesEndRef = useRef<HTMLDivElement>(null);
  const [inputValue, setInputValue] = React.useState('');
  const [messageKind, setMessageKind] = React.useState<string>('directive');
  const [workspace, setWorkspace] = React.useState<string>('');
  const [showWorkspaceInput, setShowWorkspaceInput] = React.useState<boolean>(false);
  const [approvalNotes, setApprovalNotes] = React.useState<Map<string, string>>(new Map());
  const [handledApprovals, setHandledApprovals] = React.useState<Set<string>>(new Set());
  const [expandedMessages, setExpandedMessages] = useState<Set<string>>(new Set());

  // Helper to check if content needs truncation and get truncated version
  const getTruncatedContent = (content: string): { needsTruncation: boolean; truncated: string; fullLength: number; lineCount: number } => {
    const lineCount = (content.match(/\n/g) || []).length + 1;
    const needsTruncation = content.length > MAX_DISPLAY_LENGTH || lineCount > MAX_DISPLAY_LINES;

    if (!needsTruncation) {
      return { needsTruncation: false, truncated: content, fullLength: content.length, lineCount };
    }

    // Truncate by character limit first
    let truncated = content.slice(0, MAX_DISPLAY_LENGTH);

    // Then truncate by line limit if still too many lines
    const lines = truncated.split('\n');
    if (lines.length > MAX_DISPLAY_LINES) {
      truncated = lines.slice(0, MAX_DISPLAY_LINES).join('\n');
    }

    // Try to end at a newline for cleaner display
    const lastNewline = truncated.lastIndexOf('\n');
    if (lastNewline > truncated.length * 0.8) {
      truncated = truncated.slice(0, lastNewline);
    }

    return { needsTruncation: true, truncated, fullLength: content.length, lineCount };
  };

  const toggleMessageExpanded = (messageId: string) => {
    setExpandedMessages(prev => {
      const next = new Set(prev);
      if (next.has(messageId)) {
        next.delete(messageId);
      } else {
        next.add(messageId);
      }
      return next;
    });
  };

  // Load workspace from thread when thread changes
  useEffect(() => {
    if (thread?.workspace) {
      setWorkspace(thread.workspace);
    } else {
      setWorkspace('');
    }
  }, [thread?.id, thread?.workspace]);

  useEffect(() => {
    messagesEndRef.current?.scrollIntoView({ behavior: 'smooth' });
  }, [messages]);

  // Save workspace to thread when changed
  const handleWorkspaceChange = (newWorkspace: string) => {
    setWorkspace(newWorkspace);
    if (onWorkspaceChange) {
      onWorkspaceChange(newWorkspace);
    }
  };

  const handleSend = () => {
    if (inputValue.trim()) {
      onSendMessage(inputValue, messageKind, workspace || undefined);
      setInputValue('');
    }
  };

  const handleKeyPress = (e: React.KeyboardEvent) => {
    if (e.key === 'Enter' && !e.shiftKey) {
      e.preventDefault();
      handleSend();
    }
  };

  const formatTimestamp = (timestamp: number | string) => {
    const date = new Date(timestamp);
    return date.toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' });
  };

  const truncateId = (id: string) => {
    if (id.length > 12) {
      return `${id.slice(0, 8)}...`;
    }
    return id;
  };

  // Parse approval ID from message metadata
  const getApprovalId = (message: Message): string | null => {
    if (!message.metadata_json) return null;
    try {
      const metadata = JSON.parse(message.metadata_json);
      return metadata.approval_id || null;
    } catch {
      return null;
    }
  };

  // Handle approval action
  const handleApprove = (approvalId: string) => {
    const notes = approvalNotes.get(approvalId) || '';
    if (onApproveRequest) {
      onApproveRequest(approvalId, notes);
      setHandledApprovals(prev => new Set(prev).add(approvalId));
      setApprovalNotes(prev => {
        const newMap = new Map(prev);
        newMap.delete(approvalId);
        return newMap;
      });
    }
  };

  // Handle rejection action
  const handleReject = (approvalId: string) => {
    const notes = approvalNotes.get(approvalId) || '';
    if (!notes.trim()) {
      alert('Please provide a reason for rejection');
      return;
    }
    if (onRejectRequest) {
      onRejectRequest(approvalId, notes);
      setHandledApprovals(prev => new Set(prev).add(approvalId));
      setApprovalNotes(prev => {
        const newMap = new Map(prev);
        newMap.delete(approvalId);
        return newMap;
      });
    }
  };

  const updateApprovalNotes = (approvalId: string, notes: string) => {
    setApprovalNotes(prev => new Map(prev).set(approvalId, notes));
  };

  if (!thread) {
    return null;
  }

  return (
    <div className="conversation-view">
      {/* Header */}
      <div className="conversation-header">
        <div className="header-info">
          <h2 className="thread-title">{thread.title}</h2>
          {thread.target_agent && (
            <span className="thread-agent-badge">
              {Icons.bot}
              {thread.target_agent}
            </span>
          )}
        </div>
        <div className="header-stats">
          <span className="message-count">{messages.length} messages</span>
          <span className="thread-id" title={thread.id}>{truncateId(thread.id)}</span>
        </div>
      </div>

      {/* Messages */}
      <div className="messages-container">
        {messages.length === 0 ? (
          <div className="empty-messages">
            <div className="empty-icon">
              <svg width="40" height="40" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.5">
                <path d="M21 15a2 2 0 0 1-2 2H7l-4 4V5a2 2 0 0 1 2-2h14a2 2 0 0 1 2 2z" />
              </svg>
            </div>
            <p>No messages yet</p>
            <span className="hint">Send a message to start the conversation</span>
          </div>
        ) : (
          messages.map((message, index) => {
            const isHuman = message.from_type === 'human';
            const showAvatar = index === 0 || messages[index - 1].from_type !== message.from_type;
            const isExpanded = expandedMessages.has(message.id);
            const { needsTruncation, truncated, fullLength, lineCount } = getTruncatedContent(message.content);
            const displayContent = isExpanded ? message.content : truncated;

            return (
              <div
                key={message.id}
                className={`message ${isHuman ? 'human' : 'agent'}`}
              >
                {/* Avatar */}
                <div className={`message-avatar ${showAvatar ? 'visible' : ''}`}>
                  {showAvatar && (isHuman ? Icons.user : Icons.bot)}
                </div>

                {/* Content */}
                <div className="message-body">
                  {showAvatar && (
                    <div className="message-meta">
                      <span className="sender-name">{message.from_id}</span>
                      <span className="kind-badge">{getKindIcon(message.kind)} {message.kind}</span>
                      <span className="message-time">{formatTimestamp(message.created_at)}</span>
                    </div>
                  )}
                  <div className="message-content">
                    {message.kind === 'result' || !isHuman ? (
                      <ReactMarkdown
                        components={{
                          // Custom link renderer - convert local paths to file:// URLs
                          a: ({ href, children }) => {
                            // Convert absolute local paths to file:// protocol
                            let finalHref = href;
                            if (href && href.startsWith('/') && !href.startsWith('//')) {
                              // Looks like a local filesystem path, convert to file:// URL
                              finalHref = `file://${href}`;
                            }
                            return (
                              <a href={finalHref} target="_blank" rel="noopener noreferrer">
                                {children}
                              </a>
                            );
                          },
                          // Code blocks with syntax highlighting placeholder
                          code: ({ className, children, ...props }) => {
                            const isInline = !className;
                            return isInline ? (
                              <code className="inline-code" {...props}>{children}</code>
                            ) : (
                              <code className={className} {...props}>{children}</code>
                            );
                          },
                        }}
                      >
                        {displayContent}
                      </ReactMarkdown>
                    ) : (
                      displayContent
                    )}

                    {/* Truncation indicator and expand/collapse button */}
                    {needsTruncation && (
                      <div className="truncation-notice">
                        <button
                          className="expand-btn"
                          onClick={() => toggleMessageExpanded(message.id)}
                        >
                          {isExpanded ? (
                            <>Show less</>
                          ) : (
                            <>Show more ({Math.round(fullLength / 1024)}KB, {lineCount} lines)</>
                          )}
                        </button>
                      </div>
                    )}

                    {/* Inline Approval UI for approval_request messages */}
                    {message.kind === 'approval_request' && (() => {
                      const approvalId = getApprovalId(message);
                      const isHandled = approvalId && handledApprovals.has(approvalId);

                      if (!approvalId) return null;

                      return (
                        <div className="inline-approval">
                          {isHandled ? (
                            <div className="approval-handled">
                              {Icons.check}
                              <span>Action taken</span>
                            </div>
                          ) : (
                            <>
                              <input
                                type="text"
                                className="approval-notes-input"
                                placeholder="Notes (required for rejection)..."
                                value={approvalNotes.get(approvalId) || ''}
                                onChange={(e) => updateApprovalNotes(approvalId, e.target.value)}
                              />
                              <div className="approval-actions">
                                <button
                                  className="reject-btn"
                                  onClick={() => handleReject(approvalId)}
                                  title="Reject"
                                >
                                  {Icons.x}
                                  Reject
                                </button>
                                <button
                                  className="approve-btn"
                                  onClick={() => handleApprove(approvalId)}
                                  title="Approve"
                                >
                                  {Icons.check}
                                  Approve
                                </button>
                              </div>
                            </>
                          )}
                        </div>
                      );
                    })()}
                  </div>
                  <div className="message-footer">
                    <span className="message-seq">#{message.message_seq}</span>
                    {message.delivery_state !== 'acked' && (
                      <span className={`delivery-status ${message.delivery_state}`}>
                        {message.delivery_state === 'pending' ? 'sending...' : 'delivered'}
                      </span>
                    )}
                  </div>
                </div>
              </div>
            );
          })
        )}
        <div ref={messagesEndRef} />
      </div>

      {/* Input Area */}
      <div className="input-area">
        {/* Workspace expanded input row */}
        {showWorkspaceInput && (
          <div className="workspace-input-row">
            <input
              type="text"
              value={workspace}
              onChange={(e) => handleWorkspaceChange(e.target.value)}
              onBlur={() => {
                // Save on blur (when user finishes typing)
                if (onWorkspaceChange) {
                  onWorkspaceChange(workspace);
                }
              }}
              placeholder="/path/to/working/directory (leave empty for fresh workspace)"
              className="workspace-input"
            />
            <button
              onClick={async () => {
                try {
                  const response = await fetch('/api/select-folder');
                  const data = await response.json();
                  if (!data.cancelled && data.path) {
                    handleWorkspaceChange(data.path);
                  }
                } catch (err) {
                  console.error('Failed to open folder picker:', err);
                }
              }}
              className="workspace-browse"
              title="Browse for folder"
            >
              <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
                <path d="M22 19a2 2 0 0 1-2 2H4a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h5l2 3h9a2 2 0 0 1 2 2z" />
                <line x1="12" y1="11" x2="12" y2="17" />
                <line x1="9" y1="14" x2="15" y2="14" />
              </svg>
            </button>
            {workspace && (
              <button
                onClick={() => { handleWorkspaceChange(''); setShowWorkspaceInput(false); }}
                className="workspace-clear"
              >
                Clear
              </button>
            )}
          </div>
        )}

        <div className="input-wrapper">
          {/* Workspace button on the LEFT */}
          <button
            onClick={() => setShowWorkspaceInput(!showWorkspaceInput)}
            className={`workspace-toggle ${workspace ? 'has-workspace' : ''}`}
            title={workspace || 'Set working directory'}
          >
            <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
              <path d="M22 19a2 2 0 0 1-2 2H4a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h5l2 3h9a2 2 0 0 1 2 2z" />
            </svg>
          </button>
          <select
            value={messageKind}
            onChange={(e) => setMessageKind(e.target.value)}
            className="kind-selector"
          >
            <option value="directive">Directive</option>
            <option value="question">Question</option>
          </select>
          <textarea
            value={inputValue}
            onChange={(e) => setInputValue(e.target.value)}
            onKeyPress={handleKeyPress}
            placeholder={workspace ? `Message (workspace: ${workspace.split('/').pop()})` : "Type a message..."}
            rows={1}
          />
          <button
            onClick={handleSend}
            className="send-btn"
            disabled={!inputValue.trim()}
          >
            {Icons.send}
          </button>
        </div>
        <div className="input-hint">
          Press <kbd>Enter</kbd> to send, <kbd>Shift + Enter</kbd> for new line
        </div>
      </div>

      <style>{`
        .conversation-view {
          display: flex;
          flex-direction: column;
          height: 100%;
          background: var(--bg-base);
        }

        /* Header */
        .conversation-header {
          display: flex;
          align-items: center;
          justify-content: space-between;
          padding: var(--space-3) var(--space-4);
          background: var(--bg-surface);
          border-bottom: 1px solid var(--border-subtle);
        }

        .header-info {
          display: flex;
          align-items: center;
          gap: var(--space-3);
        }

        .thread-title {
          font-size: var(--text-base);
          font-weight: var(--font-semibold);
          color: var(--text-primary);
          margin: 0;
        }

        .thread-agent-badge {
          display: flex;
          align-items: center;
          gap: var(--space-1);
          font-size: var(--text-xs);
          font-weight: var(--font-medium);
          color: var(--color-primary);
          padding: 2px 8px;
          background: rgba(37, 194, 160, 0.1);
          border-radius: var(--radius-sm);
        }

        .thread-agent-badge svg {
          opacity: 0.8;
        }

        .thread-id {
          font-size: var(--text-xs);
          font-family: var(--font-mono);
          color: var(--text-tertiary);
          padding: var(--space-1) var(--space-2);
          background: var(--bg-elevated);
          border-radius: var(--radius-sm);
        }

        .header-stats {
          display: flex;
          align-items: center;
          gap: var(--space-3);
        }

        .message-count {
          font-size: var(--text-xs);
          color: var(--text-tertiary);
        }

        /* Messages Container */
        .messages-container {
          flex: 1;
          overflow-y: auto;
          padding: var(--space-4);
        }

        .empty-messages {
          display: flex;
          flex-direction: column;
          align-items: center;
          justify-content: center;
          height: 100%;
          text-align: center;
          color: var(--text-tertiary);
        }

        .empty-icon {
          display: flex;
          align-items: center;
          justify-content: center;
          width: 64px;
          height: 64px;
          background: var(--bg-surface);
          border-radius: var(--radius-lg);
          margin-bottom: var(--space-3);
        }

        .empty-messages p {
          font-size: var(--text-sm);
          margin-bottom: var(--space-1);
        }

        .empty-messages .hint {
          font-size: var(--text-xs);
          color: var(--text-tertiary);
        }

        /* Message */
        .message {
          display: flex;
          gap: var(--space-3);
          margin-bottom: var(--space-3);
        }

        .message-avatar {
          width: 32px;
          height: 32px;
          display: flex;
          align-items: center;
          justify-content: center;
          border-radius: var(--radius-full);
          flex-shrink: 0;
          visibility: hidden;
        }

        .message-avatar.visible {
          visibility: visible;
        }

        .message.human .message-avatar {
          background: var(--bg-elevated);
          color: var(--text-secondary);
        }

        .message.agent .message-avatar {
          background: rgba(37, 194, 160, 0.15);
          color: var(--color-primary);
        }

        .message-body {
          flex: 1;
          min-width: 0;
        }

        .message-meta {
          display: flex;
          align-items: center;
          gap: var(--space-2);
          margin-bottom: var(--space-1);
        }

        .sender-name {
          font-size: var(--text-sm);
          font-weight: var(--font-semibold);
          color: var(--text-primary);
        }

        .kind-badge {
          display: flex;
          align-items: center;
          gap: var(--space-1);
          font-size: var(--text-xs);
          color: var(--text-tertiary);
          padding: 2px var(--space-2);
          background: var(--bg-elevated);
          border-radius: var(--radius-sm);
        }

        .message-time {
          font-size: var(--text-xs);
          color: var(--text-tertiary);
          margin-left: auto;
        }

        .message-content {
          font-size: var(--text-sm);
          color: var(--text-primary);
          line-height: 1.6;
          word-break: break-word;
          padding: var(--space-3);
          background: var(--bg-surface);
          border-radius: var(--radius-lg);
          border: 1px solid var(--border-subtle);
        }

        /* Markdown styles */
        .message-content h2 {
          font-size: var(--text-lg);
          font-weight: var(--font-semibold);
          color: var(--text-primary);
          margin: 0 0 var(--space-3) 0;
          padding-bottom: var(--space-2);
          border-bottom: 1px solid var(--border-subtle);
        }

        .message-content h3 {
          font-size: var(--text-base);
          font-weight: var(--font-semibold);
          color: var(--text-primary);
          margin: var(--space-4) 0 var(--space-2) 0;
        }

        .message-content p {
          margin: 0 0 var(--space-2) 0;
        }

        .message-content p:last-child {
          margin-bottom: 0;
        }

        .message-content ul, .message-content ol {
          margin: var(--space-2) 0;
          padding-left: var(--space-5);
        }

        .message-content li {
          margin: var(--space-1) 0;
        }

        .message-content pre {
          background: var(--bg-base);
          border: 1px solid var(--border-subtle);
          border-radius: var(--radius-md);
          padding: var(--space-3);
          overflow-x: auto;
          margin: var(--space-2) 0;
        }

        .message-content pre code {
          background: none;
          padding: 0;
          font-family: var(--font-mono);
          font-size: var(--text-xs);
          color: var(--text-primary);
        }

        .message-content .inline-code {
          background: var(--bg-elevated);
          padding: 2px 6px;
          border-radius: var(--radius-sm);
          font-family: var(--font-mono);
          font-size: var(--text-xs);
          color: var(--color-primary);
        }

        .message-content a {
          color: var(--color-primary);
          text-decoration: none;
        }

        .message-content a:hover {
          text-decoration: underline;
        }

        .message-content details {
          margin: var(--space-3) 0;
          padding: var(--space-2);
          background: var(--bg-base);
          border: 1px solid var(--border-subtle);
          border-radius: var(--radius-md);
        }

        .message-content summary {
          cursor: pointer;
          font-weight: var(--font-medium);
          color: var(--text-secondary);
          padding: var(--space-1);
        }

        .message-content summary:hover {
          color: var(--text-primary);
        }

        .message-content strong {
          font-weight: var(--font-semibold);
          color: var(--text-primary);
        }

        .message-content hr {
          border: none;
          border-top: 1px solid var(--border-subtle);
          margin: var(--space-4) 0;
        }

        .message.human .message-content {
          border-left: 2px solid var(--color-info);
        }

        .message.agent .message-content {
          border-left: 2px solid var(--color-primary);
        }

        .message-footer {
          display: flex;
          align-items: center;
          gap: var(--space-3);
          margin-top: var(--space-1);
          padding-left: var(--space-3);
        }

        .message-seq {
          font-size: var(--text-xs);
          font-family: var(--font-mono);
          color: var(--text-tertiary);
        }

        .delivery-status {
          font-size: var(--text-xs);
          color: var(--text-tertiary);
        }

        .delivery-status.pending {
          color: var(--color-warning);
        }

        /* Input Area */
        .input-area {
          padding: var(--space-4);
          background: var(--bg-surface);
          border-top: 1px solid var(--border-subtle);
        }

        /* Workspace toggle button in input row */
        .workspace-toggle {
          display: flex;
          align-items: center;
          justify-content: center;
          width: 36px;
          height: 36px;
          padding: 0;
          background: var(--bg-elevated);
          color: var(--text-tertiary);
          border: 1px solid var(--border-subtle);
          border-radius: var(--radius-md);
          cursor: pointer;
          transition: all var(--transition-fast);
          flex-shrink: 0;
        }

        .workspace-toggle:hover {
          color: var(--text-secondary);
          border-color: var(--border-default);
          background: var(--bg-hover);
        }

        .workspace-toggle.has-workspace {
          color: var(--color-primary);
          border-color: var(--color-primary);
          background: rgba(37, 194, 160, 0.15);
        }

        .workspace-toggle.has-workspace:hover {
          background: rgba(37, 194, 160, 0.25);
        }

        .workspace-input-row {
          display: flex;
          align-items: center;
          gap: var(--space-2);
          margin-bottom: var(--space-2);
        }

        .workspace-input {
          flex: 1;
          padding: var(--space-2);
          background: var(--bg-base);
          color: var(--text-primary);
          font-family: var(--font-mono);
          font-size: var(--text-xs);
          border: 1px solid var(--border-default);
          border-radius: var(--radius-sm);
          transition: all var(--transition-fast);
        }

        .workspace-input:focus {
          outline: none;
          border-color: var(--color-primary);
          box-shadow: 0 0 0 2px rgba(37, 194, 160, 0.15);
        }

        .workspace-input::placeholder {
          color: var(--text-tertiary);
        }

        .workspace-browse {
          display: flex;
          align-items: center;
          justify-content: center;
          padding: var(--space-1) var(--space-2);
          background: var(--bg-elevated);
          color: var(--text-secondary);
          font-size: var(--text-xs);
          border: 1px solid var(--border-subtle);
          border-radius: var(--radius-sm);
          cursor: pointer;
          transition: all var(--transition-fast);
        }

        .workspace-browse:hover {
          color: var(--color-primary);
          border-color: var(--color-primary);
          background: rgba(37, 194, 160, 0.1);
        }

        .workspace-clear {
          padding: var(--space-1) var(--space-2);
          background: transparent;
          color: var(--text-tertiary);
          font-size: var(--text-xs);
          border: 1px solid var(--border-subtle);
          border-radius: var(--radius-sm);
          cursor: pointer;
          transition: all var(--transition-fast);
        }

        .workspace-clear:hover {
          color: var(--color-danger);
          border-color: var(--color-danger);
        }

        .input-wrapper {
          display: flex;
          align-items: flex-end;
          gap: var(--space-2);
          background: var(--bg-base);
          border: 1px solid var(--border-default);
          border-radius: var(--radius-lg);
          padding: var(--space-2);
          transition: border-color var(--transition-fast);
        }

        .input-wrapper:focus-within {
          border-color: var(--color-primary);
          box-shadow: 0 0 0 3px rgba(37, 194, 160, 0.1);
        }

        .kind-selector {
          padding: var(--space-2) var(--space-3);
          padding-right: var(--space-6);
          background: var(--bg-elevated);
          color: var(--text-secondary);
          font-size: var(--text-xs);
          border: none;
          border-radius: var(--radius-sm);
          cursor: pointer;
          appearance: none;
          background-image: url("data:image/svg+xml,%3Csvg xmlns='http://www.w3.org/2000/svg' width='10' height='10' viewBox='0 0 24 24' fill='none' stroke='%238b949e' stroke-width='2'%3E%3Cpolyline points='6 9 12 15 18 9'%3E%3C/polyline%3E%3C/svg%3E");
          background-repeat: no-repeat;
          background-position: right var(--space-2) center;
        }

        .kind-selector:focus {
          outline: none;
        }

        .input-wrapper textarea {
          flex: 1;
          min-height: 40px;
          max-height: 150px;
          padding: var(--space-2);
          background: transparent;
          color: var(--text-primary);
          font-family: var(--font-sans);
          font-size: var(--text-sm);
          line-height: 1.5;
          border: none;
          resize: none;
        }

        .input-wrapper textarea:focus {
          outline: none;
        }

        .input-wrapper textarea::placeholder {
          color: var(--text-tertiary);
        }

        .send-btn {
          display: flex;
          align-items: center;
          justify-content: center;
          width: 40px;
          height: 40px;
          background: var(--color-primary);
          color: var(--text-inverse);
          border: none;
          border-radius: var(--radius-md);
          cursor: pointer;
          transition: all var(--transition-fast);
          flex-shrink: 0;
        }

        .send-btn:hover:not(:disabled) {
          background: var(--color-primary-light);
          transform: translateY(-1px);
        }

        .send-btn:disabled {
          background: var(--bg-elevated);
          color: var(--text-tertiary);
          cursor: not-allowed;
        }

        .input-hint {
          margin-top: var(--space-2);
          font-size: var(--text-xs);
          color: var(--text-tertiary);
          text-align: center;
        }

        .input-hint kbd {
          padding: 2px 6px;
          background: var(--bg-elevated);
          border-radius: var(--radius-sm);
          font-family: var(--font-mono);
          font-size: 10px;
        }

        /* Inline Approval UI */
        .inline-approval {
          margin-top: var(--space-3);
          padding: var(--space-3);
          background: var(--bg-elevated);
          border: 1px solid var(--border-default);
          border-radius: var(--radius-md);
        }

        .approval-notes-input {
          width: 100%;
          padding: var(--space-2);
          background: var(--bg-base);
          color: var(--text-primary);
          font-size: var(--text-sm);
          border: 1px solid var(--border-subtle);
          border-radius: var(--radius-sm);
          margin-bottom: var(--space-2);
        }

        .approval-notes-input:focus {
          outline: none;
          border-color: var(--color-primary);
          box-shadow: 0 0 0 2px rgba(37, 194, 160, 0.15);
        }

        .approval-notes-input::placeholder {
          color: var(--text-tertiary);
        }

        .approval-actions {
          display: flex;
          gap: var(--space-2);
          justify-content: flex-end;
        }

        .approve-btn, .reject-btn {
          display: flex;
          align-items: center;
          gap: var(--space-1);
          padding: var(--space-2) var(--space-3);
          font-size: var(--text-sm);
          font-weight: var(--font-medium);
          border: none;
          border-radius: var(--radius-sm);
          cursor: pointer;
          transition: all var(--transition-fast);
        }

        .approve-btn {
          background: var(--color-success);
          color: var(--text-inverse);
        }

        .approve-btn:hover {
          filter: brightness(1.1);
          transform: translateY(-1px);
        }

        .reject-btn {
          background: var(--bg-surface);
          color: var(--color-danger);
          border: 1px solid var(--color-danger);
        }

        .reject-btn:hover {
          background: var(--color-danger);
          color: var(--text-inverse);
        }

        .approval-handled {
          display: flex;
          align-items: center;
          gap: var(--space-2);
          color: var(--text-tertiary);
          font-size: var(--text-sm);
        }

        .approval-handled svg {
          color: var(--color-success);
        }

        /* Truncation notice */
        .truncation-notice {
          margin-top: var(--space-2);
          padding-top: var(--space-2);
          border-top: 1px dashed var(--border-subtle);
        }

        .expand-btn {
          display: inline-flex;
          align-items: center;
          gap: var(--space-1);
          padding: var(--space-1) var(--space-2);
          font-size: var(--text-xs);
          font-weight: var(--font-medium);
          color: var(--color-primary);
          background: rgba(37, 194, 160, 0.1);
          border: 1px solid transparent;
          border-radius: var(--radius-sm);
          cursor: pointer;
          transition: all var(--transition-fast);
        }

        .expand-btn:hover {
          background: rgba(37, 194, 160, 0.2);
          border-color: var(--color-primary);
        }
      `}</style>
    </div>
  );
};
