/**
 * ChatPreview - Reusable compact chat preview component
 * Shows recent messages for a span/session with option to expand to full chat view
 */
import React, { useState, useEffect, useCallback } from 'react';
import styles from './ExecHierarchy.module.css';

export interface ChatPreviewProps {
  spanId?: string;
  sessionId?: string;
  maxMessages?: number;  // Default 3 (most recent)
  onClose?: () => void;
  onExpandToChat?: () => void;
  compact?: boolean;  // Smaller styling for inline use
}

interface ChatMessage {
  role: 'user' | 'assistant';
  content: string;
  timestamp?: string;
}

interface FetchState {
  loading: boolean;
  error?: string;
  messages: ChatMessage[];
}

export const ChatPreview: React.FC<ChatPreviewProps> = ({
  spanId,
  sessionId,
  maxMessages = 3,
  onClose,
  onExpandToChat,
  compact = false,
}) => {
  const [state, setState] = useState<FetchState>({
    loading: true,
    messages: [],
  });

  // Process messages from API response - defined before fetchMessages since it's used by it
  const processMessages = useCallback((data: any) => {
    const messages: ChatMessage[] = [];

    // Handle different response formats
    const rawMessages = data.messages || data.Messages || data;
    if (!Array.isArray(rawMessages)) {
      setState({ loading: false, error: 'Invalid response format', messages: [] });
      return;
    }

    for (const msg of rawMessages) {
      const role = msg.role || msg.Role;
      if (role !== 'user' && role !== 'assistant') continue;

      // Extract text content from various formats
      let content = '';
      const rawContent = msg.content || msg.Content || msg.message;

      if (typeof rawContent === 'string') {
        content = rawContent;
      } else if (Array.isArray(rawContent)) {
        // Claude API format: [{type: 'text', text: '...'}]
        for (const block of rawContent) {
          if (block.type === 'text' && block.text) {
            content += block.text + '\n';
          }
        }
        content = content.trim();
      }

      if (content) {
        messages.push({
          role,
          content,
          timestamp: msg.timestamp || msg.Timestamp,
        });
      }
    }

    // Take last N messages (most recent)
    const recentMessages = messages.slice(-maxMessages);

    setState({
      loading: false,
      messages: recentMessages,
    });
  }, [maxMessages]);

  const fetchMessages = useCallback(async () => {
    if (!spanId && !sessionId) {
      setState({ loading: false, error: 'No span or session ID provided', messages: [] });
      return;
    }

    setState(prev => ({ ...prev, loading: true, error: undefined }));

    try {
      // Try DB endpoint first (faster, indexed), fall back to JSONL endpoint
      let url: string;
      if (sessionId) {
        // Fetch from DB by session - get last N messages
        url = `/api/claude-history/db/session/${sessionId}?limit=${maxMessages * 2}`;
      } else if (spanId) {
        // Fetch by span - time-filtered messages
        url = `/api/claude-history/by-span/${spanId}`;
      } else {
        throw new Error('No ID provided');
      }

      const response = await fetch(url);
      if (!response.ok) {
        // Try fallback to JSONL endpoint if DB fails
        if (sessionId) {
          const fallbackUrl = `/api/claude-history/session/${sessionId}?limit=${maxMessages * 2}`;
          const fallbackResponse = await fetch(fallbackUrl);
          if (!fallbackResponse.ok) {
            throw new Error(`Failed to fetch: ${fallbackResponse.status}`);
          }
          const data = await fallbackResponse.json();
          processMessages(data);
          return;
        }
        throw new Error(`Failed to fetch: ${response.status}`);
      }

      const data = await response.json();
      processMessages(data);
    } catch (err) {
      setState({
        loading: false,
        error: err instanceof Error ? err.message : 'Failed to fetch messages',
        messages: [],
      });
    }
  }, [spanId, sessionId, maxMessages, processMessages]);

  useEffect(() => {
    fetchMessages();
  }, [fetchMessages]);

  // Truncate long messages
  const truncateMessage = (text: string, maxLen: number = 150): string => {
    if (text.length <= maxLen) return text;
    return text.substring(0, maxLen).trim() + '...';
  };

  if (state.loading) {
    return (
      <div className={`${styles.chatPreview} ${compact ? styles.chatPreviewCompact : ''}`}>
        <div className={styles.chatPreviewLoading}>
          <span className={styles.chatPreviewSpinner}>◎</span>
          Loading chat...
        </div>
      </div>
    );
  }

  if (state.error) {
    return (
      <div className={`${styles.chatPreview} ${compact ? styles.chatPreviewCompact : ''}`}>
        <div className={styles.chatPreviewError}>
          <span className={styles.chatPreviewErrorIcon}>⚠</span>
          {state.error}
        </div>
        {onClose && (
          <button className={styles.chatPreviewClose} onClick={onClose} title="Close">
            ×
          </button>
        )}
      </div>
    );
  }

  if (state.messages.length === 0) {
    return (
      <div className={`${styles.chatPreview} ${compact ? styles.chatPreviewCompact : ''}`}>
        <div className={styles.chatPreviewEmpty}>
          <span className={styles.chatPreviewEmptyIcon}>💬</span>
          No chat messages found
        </div>
        {onClose && (
          <button className={styles.chatPreviewClose} onClick={onClose} title="Close">
            ×
          </button>
        )}
      </div>
    );
  }

  return (
    <div className={`${styles.chatPreview} ${compact ? styles.chatPreviewCompact : ''}`}>
      <div className={styles.chatPreviewHeader}>
        <span className={styles.chatPreviewTitle}>
          💬 Chat Preview
        </span>
        <div className={styles.chatPreviewActions}>
          {onExpandToChat && (
            <button
              className={styles.chatPreviewExpandBtn}
              onClick={onExpandToChat}
              title="View full conversation"
            >
              ↗
            </button>
          )}
          {onClose && (
            <button className={styles.chatPreviewClose} onClick={onClose} title="Close">
              ×
            </button>
          )}
        </div>
      </div>

      <div className={styles.chatPreviewMessages}>
        {state.messages.map((msg, idx) => (
          <div
            key={idx}
            className={`${styles.chatPreviewMessage} ${
              msg.role === 'user' ? styles.chatPreviewUser : styles.chatPreviewAssistant
            }`}
          >
            <span className={styles.chatPreviewRole}>
              {msg.role === 'user' ? '👤' : '🤖'}
            </span>
            <span className={styles.chatPreviewContent}>
              {truncateMessage(msg.content)}
            </span>
          </div>
        ))}
      </div>

      {onExpandToChat && (
        <button
          className={styles.chatPreviewViewFull}
          onClick={onExpandToChat}
        >
          View full conversation →
        </button>
      )}
    </div>
  );
};

export default ChatPreview;
