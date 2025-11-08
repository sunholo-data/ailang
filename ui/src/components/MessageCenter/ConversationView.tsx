import React, { useEffect, useRef } from 'react';
import { Message } from '../../types';

interface ConversationViewProps {
  threadId: string;
  messages: Message[];
  onSendMessage: (content: string, kind: string) => void;
}

export const ConversationView: React.FC<ConversationViewProps> = ({
  threadId,
  messages,
  onSendMessage,
}) => {
  const messagesEndRef = useRef<HTMLDivElement>(null);
  const [inputValue, setInputValue] = React.useState('');
  const [messageKind, setMessageKind] = React.useState<string>('directive');

  // Auto-scroll to bottom when new messages arrive
  useEffect(() => {
    messagesEndRef.current?.scrollIntoView({ behavior: 'smooth' });
  }, [messages]);

  const handleSend = () => {
    if (inputValue.trim()) {
      onSendMessage(inputValue, messageKind);
      setInputValue('');
    }
  };

  const handleKeyPress = (e: React.KeyboardEvent) => {
    if (e.key === 'Enter' && !e.shiftKey) {
      e.preventDefault();
      handleSend();
    }
  };

  const formatTimestamp = (timestamp: number) => {
    const date = new Date(timestamp);
    return date.toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' });
  };

  const getMessageIcon = (kind: string) => {
    switch (kind) {
      case 'directive':
        return '📋';
      case 'question':
        return '❓';
      case 'status':
        return '📊';
      case 'result':
        return '✅';
      case 'approval_request':
        return '🔒';
      default:
        return '💬';
    }
  };

  const getMessageColor = (fromType: string) => {
    return fromType === 'human' ? '#007bff' : '#28a745';
  };

  return (
    <div className="conversation-view">
      <div className="conversation-header">
        <h3>Thread: {threadId}</h3>
        <div className="header-actions">
          <button className="action-btn">⭐ Pin</button>
          <button className="action-btn">🔕 Mute</button>
          <button className="action-btn">📁 Archive</button>
        </div>
      </div>

      <div className="messages-container">
        {messages.length === 0 ? (
          <div className="empty-messages">
            <p>No messages yet</p>
            <p className="hint">Start the conversation by sending a message</p>
          </div>
        ) : (
          messages.map((message) => {
            const isHuman = message.from_type === 'human';

            return (
              <div
                key={message.id}
                className={`message ${isHuman ? 'human' : 'agent'}`}
              >
                <div className="message-header">
                  <span className="message-icon">{getMessageIcon(message.kind)}</span>
                  <span className="message-sender">
                    {isHuman ? '👤' : '🤖'} {message.from_id}
                  </span>
                  <span className="message-kind">{message.kind}</span>
                  <span className="message-seq">#{message.message_seq}</span>
                  <span className="message-time">
                    {formatTimestamp(message.created_at)}
                  </span>
                </div>

                <div className="message-content">{message.content}</div>

                {message.delivery_state !== 'acked' && (
                  <div className="message-status">
                    {message.delivery_state === 'pending' ? '📤' : '📭'}
                  </div>
                )}
              </div>
            );
          })
        )}
        <div ref={messagesEndRef} />
      </div>

      <div className="message-input">
        <div className="input-controls">
          <select
            value={messageKind}
            onChange={(e) => setMessageKind(e.target.value)}
            className="kind-selector"
          >
            <option value="directive">Directive</option>
            <option value="question">Question</option>
            <option value="status">Status</option>
            <option value="result">Result</option>
          </select>
        </div>

        <div className="input-area">
          <textarea
            value={inputValue}
            onChange={(e) => setInputValue(e.target.value)}
            onKeyPress={handleKeyPress}
            placeholder="Type a message... (Enter to send, Shift+Enter for new line)"
            rows={3}
          />
          <button onClick={handleSend} className="send-btn">
            Send
          </button>
        </div>
      </div>

      <style>{`
        .conversation-view {
          display: flex;
          flex-direction: column;
          height: 100%;
          background: white;
        }

        .conversation-header {
          display: flex;
          justify-content: space-between;
          align-items: center;
          padding: 1rem;
          border-bottom: 1px solid #e0e0e0;
        }

        .conversation-header h3 {
          margin: 0;
          font-size: 1.125rem;
          font-weight: 600;
        }

        .header-actions {
          display: flex;
          gap: 0.5rem;
        }

        .action-btn {
          padding: 0.25rem 0.75rem;
          background: #f5f5f5;
          border: 1px solid #e0e0e0;
          border-radius: 4px;
          cursor: pointer;
          font-size: 0.8125rem;
        }

        .action-btn:hover {
          background: #e0e0e0;
        }

        .messages-container {
          flex: 1;
          overflow-y: auto;
          padding: 1rem;
          background: #fafafa;
        }

        .empty-messages {
          text-align: center;
          padding: 2rem;
          color: #666;
        }

        .empty-messages .hint {
          font-size: 0.875rem;
          color: #999;
        }

        .message {
          margin-bottom: 1rem;
          padding: 0.75rem;
          border-radius: 8px;
          background: white;
          box-shadow: 0 1px 2px rgba(0, 0, 0, 0.05);
        }

        .message.human {
          border-left: 3px solid #007bff;
        }

        .message.agent {
          border-left: 3px solid #28a745;
        }

        .message-header {
          display: flex;
          gap: 0.5rem;
          align-items: center;
          margin-bottom: 0.5rem;
          font-size: 0.8125rem;
          color: #666;
        }

        .message-icon {
          font-size: 1rem;
        }

        .message-sender {
          font-weight: 500;
        }

        .message-kind {
          background: #e0e0e0;
          padding: 0.125rem 0.5rem;
          border-radius: 4px;
          font-size: 0.75rem;
        }

        .message-seq {
          color: #999;
          font-size: 0.75rem;
        }

        .message-time {
          margin-left: auto;
          color: #999;
        }

        .message-content {
          color: #333;
          line-height: 1.5;
          white-space: pre-wrap;
        }

        .message-status {
          margin-top: 0.5rem;
          text-align: right;
          font-size: 0.75rem;
        }

        .message-input {
          border-top: 1px solid #e0e0e0;
          padding: 1rem;
          background: white;
        }

        .input-controls {
          margin-bottom: 0.5rem;
        }

        .kind-selector {
          padding: 0.5rem;
          border: 1px solid #e0e0e0;
          border-radius: 4px;
          font-size: 0.875rem;
        }

        .input-area {
          display: flex;
          gap: 0.5rem;
        }

        .input-area textarea {
          flex: 1;
          padding: 0.75rem;
          border: 1px solid #e0e0e0;
          border-radius: 4px;
          font-family: inherit;
          font-size: 0.9375rem;
          resize: none;
        }

        .input-area textarea:focus {
          outline: none;
          border-color: #007bff;
        }

        .send-btn {
          padding: 0.75rem 1.5rem;
          background: #007bff;
          color: white;
          border: none;
          border-radius: 4px;
          cursor: pointer;
          font-weight: 500;
        }

        .send-btn:hover {
          background: #0056b3;
        }
      `}</style>
    </div>
  );
};
