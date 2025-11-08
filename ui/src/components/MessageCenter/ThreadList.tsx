import React from 'react';
import { Thread } from '../../types';

interface ThreadListProps {
  threads: Thread[];
  selectedThreadId: string | null;
  onSelectThread: (threadId: string) => void;
  onCreateThread: (title: string) => void;
  unreadCounts: Map<string, number>;
}

export const ThreadList: React.FC<ThreadListProps> = ({
  threads,
  selectedThreadId,
  onSelectThread,
  onCreateThread,
  unreadCounts,
}) => {
  const handleCreateThread = () => {
    const title = prompt('Enter thread title:');
    if (title && title.trim()) {
      onCreateThread(title.trim());
    }
  };
  const formatTimestamp = (timestamp: number) => {
    const date = new Date(timestamp);
    const now = new Date();
    const diff = now.getTime() - date.getTime();
    const minutes = Math.floor(diff / 60000);
    const hours = Math.floor(diff / 3600000);
    const days = Math.floor(diff / 86400000);

    if (minutes < 1) return 'Just now';
    if (minutes < 60) return `${minutes}m ago`;
    if (hours < 24) return `${hours}h ago`;
    if (days < 7) return `${days}d ago`;
    return date.toLocaleDateString();
  };

  const getStatusIcon = (status: string) => {
    switch (status) {
      case 'active':
        return '🟢';
      case 'paused':
        return '🟡';
      case 'resolved':
        return '✅';
      case 'archived':
        return '📁';
      default:
        return '○';
    }
  };

  return (
    <div className="thread-list">
      <div className="thread-list-header">
        <h2>Threads</h2>
        <button className="new-thread-btn" onClick={handleCreateThread}>
          + New Thread
        </button>
      </div>

      <div className="thread-list-items">
        {threads.length === 0 ? (
          <div className="empty-state">
            <p>No threads yet</p>
            <p className="hint">Create a new thread to get started</p>
          </div>
        ) : (
          threads.map((thread) => {
            const unreadCount = unreadCounts.get(thread.id) || 0;
            const isSelected = thread.id === selectedThreadId;

            return (
              <div
                key={thread.id}
                className={`thread-item ${isSelected ? 'selected' : ''}`}
                onClick={() => onSelectThread(thread.id)}
              >
                <div className="thread-header">
                  <span className="thread-status">{getStatusIcon(thread.status)}</span>
                  <span className="thread-title">{thread.title}</span>
                  {unreadCount > 0 && (
                    <span className="unread-badge">{unreadCount}</span>
                  )}
                </div>

                <div className="thread-meta">
                  <span className="thread-creator">
                    {thread.created_by_type === 'human' ? '👤' : '🤖'}{' '}
                    {thread.created_by_id}
                  </span>
                  <span className="thread-updated">
                    {formatTimestamp(thread.updated_at)}
                  </span>
                </div>

                <div className="thread-preview">
                  Last message seq: {thread.last_seq}
                </div>
              </div>
            );
          })
        )}
      </div>

      <style>{`
        .thread-list {
          display: flex;
          flex-direction: column;
          height: 100%;
          background: #f5f5f5;
          border-right: 1px solid #e0e0e0;
        }

        .thread-list-header {
          display: flex;
          justify-content: space-between;
          align-items: center;
          padding: 1rem;
          border-bottom: 1px solid #e0e0e0;
          background: white;
        }

        .thread-list-header h2 {
          margin: 0;
          font-size: 1.25rem;
          font-weight: 600;
        }

        .new-thread-btn {
          padding: 0.5rem 1rem;
          background: #007bff;
          color: white;
          border: none;
          border-radius: 4px;
          cursor: pointer;
          font-size: 0.875rem;
        }

        .new-thread-btn:hover {
          background: #0056b3;
        }

        .thread-list-items {
          flex: 1;
          overflow-y: auto;
        }

        .empty-state {
          padding: 2rem;
          text-align: center;
          color: #666;
        }

        .empty-state p {
          margin: 0.5rem 0;
        }

        .empty-state .hint {
          font-size: 0.875rem;
          color: #999;
        }

        .thread-item {
          padding: 1rem;
          background: white;
          border-bottom: 1px solid #e0e0e0;
          cursor: pointer;
          transition: background 0.2s;
        }

        .thread-item:hover {
          background: #f8f9fa;
        }

        .thread-item.selected {
          background: #e7f3ff;
          border-left: 3px solid #007bff;
        }

        .thread-header {
          display: flex;
          align-items: center;
          gap: 0.5rem;
          margin-bottom: 0.5rem;
        }

        .thread-status {
          font-size: 0.875rem;
        }

        .thread-title {
          flex: 1;
          font-weight: 500;
          font-size: 0.9375rem;
        }

        .unread-badge {
          background: #007bff;
          color: white;
          padding: 0.125rem 0.5rem;
          border-radius: 10px;
          font-size: 0.75rem;
          font-weight: 600;
        }

        .thread-meta {
          display: flex;
          justify-content: space-between;
          font-size: 0.8125rem;
          color: #666;
          margin-bottom: 0.25rem;
        }

        .thread-preview {
          font-size: 0.75rem;
          color: #999;
        }
      `}</style>
    </div>
  );
};
