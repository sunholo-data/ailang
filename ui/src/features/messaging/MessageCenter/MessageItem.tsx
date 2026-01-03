/**
 * Individual message item component
 */
import React, { useState } from 'react';
import ReactMarkdown from 'react-markdown';
import { Message } from '../../../types';
import { Icons, getKindIcon } from '../../../components/common/Icons';
import { formatTime } from '../../../utils/formatters';
import { getTruncatedContent, isRunningStatus, parseExecutionMetadata, getApprovalId } from './helpers';
import { InlineApprovalPanel } from './InlineApprovalPanel';

interface MessageItemProps {
  message: Message;
  showAvatar: boolean;
  isExpanded: boolean;
  onToggleExpanded: () => void;
  expandedFiles: Set<string>;
  onToggleFiles: (messageId: string) => void;
  // Approval handling
  approvalNotes: Map<string, string>;
  handledApprovals: Set<string>;
  onApprovalNotesChange: (approvalId: string, notes: string) => void;
  onApprove: (approvalId: string) => void;
  onReject: (approvalId: string) => void;
}

export const MessageItem: React.FC<MessageItemProps> = ({
  message,
  showAvatar,
  isExpanded,
  onToggleExpanded,
  expandedFiles,
  onToggleFiles,
  approvalNotes,
  handledApprovals,
  onApprovalNotesChange,
  onApprove,
  onReject,
}) => {
  const isHuman = message.from_type === 'human';
  const isRunning = isRunningStatus(message);
  const { needsTruncation, truncated, fullLength, lineCount } = getTruncatedContent(message.content);
  const displayContent = isExpanded ? message.content : truncated;
  const isFilesExpanded = expandedFiles.has(message.id);

  return (
    <div className={`message ${isHuman ? 'human' : 'agent'}${isRunning ? ' running-status' : ''}`}>
      {/* Avatar */}
      <div className={`message-avatar ${showAvatar ? 'visible' : ''}`}>
        {showAvatar && (isHuman ? Icons.user : Icons.bot)}
      </div>

      {/* Content */}
      <div className="message-body">
        {showAvatar && (
          <div className="message-meta">
            <span className="sender-name">{message.from_id}</span>
            <span className={`kind-badge${isRunning ? ' running' : ''}`}>
              {isRunning ? Icons.spinner : getKindIcon(message.kind)} {message.kind}
            </span>
            <span className="message-time">{formatTime(message.created_at)}</span>
          </div>
        )}
        <div className="message-content">
          {message.kind === 'result' || !isHuman ? (
            <ReactMarkdown
              components={{
                // Custom link renderer - convert local paths to file:// URLs
                a: ({ href, children }) => {
                  let finalHref = href;
                  if (href && href.startsWith('/') && !href.startsWith('//')) {
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
              <button className="expand-btn" onClick={onToggleExpanded}>
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
            if (!approvalId) return null;

            const isHandled = handledApprovals.has(approvalId);
            const notes = approvalNotes.get(approvalId) || '';

            return (
              <InlineApprovalPanel
                approvalId={approvalId}
                isHandled={isHandled}
                notes={notes}
                onNotesChange={(n) => onApprovalNotesChange(approvalId, n)}
                onApprove={() => onApprove(approvalId)}
                onReject={() => onReject(approvalId)}
              />
            );
          })()}

          {/* Files Created Section for result messages */}
          {message.kind === 'result' && (() => {
            const execMeta = parseExecutionMetadata(message.metadata_json);
            if (!execMeta || !execMeta.files_created || execMeta.files_created.length === 0) {
              return null;
            }

            return (
              <div className="files-created-section">
                <button
                  className={`files-toggle-btn ${isFilesExpanded ? 'expanded' : ''}`}
                  onClick={() => onToggleFiles(message.id)}
                >
                  {Icons.file}
                  <span>Files Created ({execMeta.files_created.length})</span>
                  {execMeta.workspace && (
                    <span className="workspace-badge" title={execMeta.workspace}>
                      {Icons.folder}
                      {execMeta.workspace.split('/').pop()}
                    </span>
                  )}
                  <span className="toggle-chevron">{isFilesExpanded ? '▼' : '▶'}</span>
                </button>
                {isFilesExpanded && (
                  <ul className="files-list">
                    {execMeta.files_created.map((file, idx) => (
                      <li key={idx} className="file-item">
                        <a
                          href={`file://${execMeta.workspace ? execMeta.workspace + '/' : ''}${file}`}
                          target="_blank"
                          rel="noopener noreferrer"
                          title={file}
                        >
                          {file}
                        </a>
                      </li>
                    ))}
                  </ul>
                )}
              </div>
            );
          })()}
        </div>
        <div className="message-footer">
          <span className="message-seq">#{message.message_seq}</span>
        </div>
      </div>
    </div>
  );
};
