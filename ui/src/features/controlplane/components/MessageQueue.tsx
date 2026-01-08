/**
 * MessageQueue - Real-time event feed for Control Plane
 */
import React, { useState, useMemo } from 'react';
import type { EventMessage } from './types';
import styles from '../ControlPlane.module.css';

export interface MessageQueueProps {
  events: EventMessage[];
  onEventClick: (event: EventMessage) => void;
  loading?: boolean;
  pageSize?: number;
}

const getEventIcon = (type: EventMessage['type']): string => {
  switch (type) {
    case 'task_start': return '▶';
    case 'task_complete': return '✓';
    case 'task_error': return '✕';
    case 'handoff': return '→';
    case 'approval': return '⏳';
    case 'message': return '◉';
  }
};

const getEventColor = (type: EventMessage['type']): string => {
  switch (type) {
    case 'task_start': return 'primary';
    case 'task_complete': return 'success';
    case 'task_error': return 'error';
    case 'handoff': return 'amber';
    case 'approval': return 'warning';
    case 'message': return 'muted';
  }
};

const formatTime = (timestamp: string): string => {
  const date = new Date(timestamp);
  return date.toLocaleTimeString('en-US', { hour12: false });
};

const formatRelativeTime = (timestamp: string): string => {
  const diff = Date.now() - new Date(timestamp).getTime();
  if (diff < 60000) return `${Math.floor(diff / 1000)}s ago`;
  if (diff < 3600000) return `${Math.floor(diff / 60000)}m ago`;
  return `${Math.floor(diff / 3600000)}h ago`;
};

export const MessageQueue: React.FC<MessageQueueProps> = ({ events, onEventClick, loading, pageSize = 10 }) => {
  const [currentPage, setCurrentPage] = useState(0);

  // Calculate pagination
  const totalPages = useMemo(() => Math.ceil(events.length / pageSize), [events.length, pageSize]);
  const paginatedEvents = useMemo(() => {
    const start = currentPage * pageSize;
    return events.slice(start, start + pageSize);
  }, [events, currentPage, pageSize]);

  // Reset to first page when events change significantly
  React.useEffect(() => {
    if (currentPage >= totalPages && totalPages > 0) {
      setCurrentPage(totalPages - 1);
    }
  }, [totalPages, currentPage]);

  const goToPrevPage = () => setCurrentPage((p) => Math.max(0, p - 1));
  const goToNextPage = () => setCurrentPage((p) => Math.min(totalPages - 1, p + 1));

  return (
    <div className={styles.messageQueue}>
      <div className={styles.queueHeader}>
        <h3 className={styles.panelTitle}>
          <span className={styles.panelIcon}>▥</span>
          Event Queue
        </h3>
        <span className={styles.queueCount}>
          {loading ? '...' : `${events.length} events`}
        </span>
      </div>
      <div className={styles.queueList}>
        {loading && (
          <div className={styles.queueEmpty}>
            <span className={styles.queueEmptyIcon}>◎</span>
            <span className={styles.queueEmptyText}>Loading events...</span>
          </div>
        )}
        {!loading && events.length === 0 && (
          <div className={styles.queueEmpty}>
            <span className={styles.queueEmptyIcon}>◎</span>
            <span className={styles.queueEmptyText}>No recent events</span>
            <span className={styles.queueEmptyHint}>
              Events appear when tasks run or agents communicate
            </span>
          </div>
        )}
        {!loading && paginatedEvents.map((event) => (
          <div
            key={event.id}
            className={styles.queueItem}
            onClick={() => onEventClick(event)}
            data-type={getEventColor(event.type)}
          >
            <span className={styles.queueIcon} data-type={getEventColor(event.type)}>
              {getEventIcon(event.type)}
            </span>
            <div className={styles.queueContent}>
              <div className={styles.queueMeta}>
                <span className={styles.queueSource}>{event.source}</span>
                {event.target && (
                  <>
                    <span className={styles.queueArrow}>→</span>
                    <span className={styles.queueTarget}>{event.target}</span>
                  </>
                )}
              </div>
              <div className={styles.queueMessage}>{event.content}</div>
            </div>
            <div className={styles.queueTime}>
              <span className={styles.queueRelative}>{formatRelativeTime(event.timestamp)}</span>
              <span className={styles.queueAbsolute}>{formatTime(event.timestamp)}</span>
            </div>
          </div>
        ))}
      </div>
      {/* Pagination controls */}
      {!loading && totalPages > 1 && (
        <div className={styles.queuePagination}>
          <button
            className={styles.queuePageBtn}
            onClick={goToPrevPage}
            disabled={currentPage === 0}
            title="Previous page"
          >
            ←
          </button>
          <span className={styles.queuePageInfo}>
            {currentPage + 1} / {totalPages}
          </span>
          <button
            className={styles.queuePageBtn}
            onClick={goToNextPage}
            disabled={currentPage >= totalPages - 1}
            title="Next page"
          >
            →
          </button>
        </div>
      )}
    </div>
  );
};

export default MessageQueue;
