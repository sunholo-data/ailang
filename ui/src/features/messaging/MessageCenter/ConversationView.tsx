/**
 * Conversation view - displays messages and input for a thread
 */
import React, { useEffect, useRef, useState } from 'react';
import { Message, Thread } from '../../../types';
import { Icons } from '../../../components/common/Icons';
import { truncateId } from '../../../utils/formatters';
import { MessageItem } from './MessageItem';
import { MessageInput } from './MessageInput';
import './ConversationView.module.css';

interface ConversationViewProps {
  thread?: Thread;
  messages: Message[];
  onSendMessage: (content: string, kind: string, workspace?: string) => void;
  onWorkspaceChange?: (workspace: string) => void;
  onApproveRequest?: (approvalId: string, notes: string) => void;
  onRejectRequest?: (approvalId: string, notes: string) => void;
}

export const ConversationView: React.FC<ConversationViewProps> = ({
  thread,
  messages,
  onSendMessage,
  onWorkspaceChange,
  onApproveRequest,
  onRejectRequest,
}) => {
  const messagesEndRef = useRef<HTMLDivElement>(null);
  const [workspace, setWorkspace] = useState<string>('');
  const [approvalNotes, setApprovalNotes] = useState<Map<string, string>>(new Map());
  const [handledApprovals, setHandledApprovals] = useState<Set<string>>(new Set());
  const [expandedMessages, setExpandedMessages] = useState<Set<string>>(new Set());
  const [expandedFiles, setExpandedFiles] = useState<Set<string>>(new Set());

  // Load workspace from thread when thread changes
  useEffect(() => {
    if (thread?.workspace) {
      setWorkspace(thread.workspace);
    } else {
      setWorkspace('');
    }
  }, [thread?.id, thread?.workspace]);

  // Auto-scroll to bottom on new messages
  useEffect(() => {
    messagesEndRef.current?.scrollIntoView({ behavior: 'smooth' });
  }, [messages]);

  const handleWorkspaceChange = (newWorkspace: string) => {
    setWorkspace(newWorkspace);
    if (onWorkspaceChange) {
      onWorkspaceChange(newWorkspace);
    }
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

  const toggleFiles = (messageId: string) => {
    setExpandedFiles(prev => {
      const next = new Set(prev);
      if (next.has(messageId)) {
        next.delete(messageId);
      } else {
        next.add(messageId);
      }
      return next;
    });
  };

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
            const showAvatar = index === 0 || messages[index - 1].from_type !== message.from_type;
            return (
              <MessageItem
                key={message.id}
                message={message}
                showAvatar={showAvatar}
                isExpanded={expandedMessages.has(message.id)}
                onToggleExpanded={() => toggleMessageExpanded(message.id)}
                expandedFiles={expandedFiles}
                onToggleFiles={toggleFiles}
                approvalNotes={approvalNotes}
                handledApprovals={handledApprovals}
                onApprovalNotesChange={updateApprovalNotes}
                onApprove={handleApprove}
                onReject={handleReject}
              />
            );
          })
        )}
        <div ref={messagesEndRef} />
      </div>

      {/* Input Area */}
      <MessageInput
        workspace={workspace}
        onWorkspaceChange={handleWorkspaceChange}
        onSendMessage={onSendMessage}
      />
    </div>
  );
};
