/**
 * EventDetail - Inline event detail view
 * Shows event metadata (Source, Target, Time, Task ID, Content)
 * Span visualization is now in the Execution Spans panel
 */
import React from 'react';
import type { EventMessage } from './types';
import styles from '../ControlPlane.module.css';

export interface EventDetailProps {
  event: EventMessage | null;
  traceId: string | null;
  loading?: boolean;
  onClose: () => void;
  onNavigate?: (direction: 'prev' | 'next') => void;
  currentIndex?: number;
  totalEvents?: number;
}

const getEventTypeLabel = (type: EventMessage['type']): string => {
  switch (type) {
    case 'task_start': return 'Task Started';
    case 'task_complete': return 'Task Complete';
    case 'task_error': return 'Task Error';
    case 'handoff': return 'Handoff';
    case 'approval': return 'Approval Request';
    case 'message': return 'Message';
  }
};

const getEventTypeColor = (type: EventMessage['type']): string => {
  switch (type) {
    case 'task_start': return 'primary';
    case 'task_complete': return 'success';
    case 'task_error': return 'error';
    case 'handoff': return 'amber';
    case 'approval': return 'warning';
    case 'message': return 'muted';
  }
};

export const EventDetail: React.FC<EventDetailProps> = ({
  event,
  traceId,
  loading,
  onClose,
  onNavigate,
  currentIndex,
  totalEvents,
}) => {
  // Empty state when no event selected
  if (!event) {
    return (
      <div className={styles.eventDetailInline}>
        <div className={styles.eventDetailHeader}>
          <div className={styles.eventDetailTitle}>
            <span className={styles.eventDetailIcon} data-type="muted">◎</span>
            <span className={styles.eventDetailLabel}>Event Details</span>
          </div>
        </div>
        <div className={styles.eventDetailEmpty}>
          <span className={styles.eventDetailEmptyIcon}>◇</span>
          <span className={styles.eventDetailEmptyText}>Select an event to view details</span>
        </div>
      </div>
    );
  }

  const metadata = event.metadata as Record<string, unknown> | undefined;
  const payload = metadata?.payload as string;

  return (
    <div className={styles.eventDetailInline}>
      {/* Header with navigation and close button */}
      <div className={styles.eventDetailHeader}>
        <div className={styles.eventDetailTitle}>
          <span className={styles.eventDetailIcon} data-type={getEventTypeColor(event.type)}>◉</span>
          <span className={styles.eventDetailLabel}>{getEventTypeLabel(event.type)}</span>
          <span className={styles.eventDetailId}>{event.id.slice(0, 8)}</span>
        </div>
        <div className={styles.eventDetailNav}>
          {onNavigate && (
            <>
              <button
                className={styles.eventDetailNavBtn}
                onClick={() => onNavigate('prev')}
                title="Previous event (←)"
              >
                ←
              </button>
              {currentIndex !== undefined && totalEvents !== undefined && (
                <span className={styles.eventDetailNavCount}>
                  {currentIndex + 1} / {totalEvents}
                </span>
              )}
              <button
                className={styles.eventDetailNavBtn}
                onClick={() => onNavigate('next')}
                title="Next event (→)"
              >
                →
              </button>
            </>
          )}
          <button className={styles.eventDetailClose} onClick={onClose}>✕</button>
        </div>
      </div>

      {/* Event Info */}
      <div className={styles.eventDetailContent}>
        <div className={styles.eventDetailInfo}>
          <div className={styles.eventDetailRow}>
            <span className={styles.eventDetailRowLabel}>Source</span>
            <span className={styles.eventDetailRowValue}>{event.source}</span>
          </div>
          {event.target && (
            <div className={styles.eventDetailRow}>
              <span className={styles.eventDetailRowLabel}>Target</span>
              <span className={styles.eventDetailRowValue}>{event.target}</span>
            </div>
          )}
          <div className={styles.eventDetailRow}>
            <span className={styles.eventDetailRowLabel}>Time</span>
            <span className={styles.eventDetailRowValue}>
              {new Date(event.timestamp).toLocaleString()}
            </span>
          </div>
          {traceId && (
            <div className={styles.eventDetailRow}>
              <span className={styles.eventDetailRowLabel}>Task ID</span>
              <span
                className={styles.eventDetailRowValue}
                title={`Click to copy: ${traceId}`}
                style={{ cursor: 'pointer', fontFamily: 'monospace' }}
                onClick={() => {
                  navigator.clipboard.writeText(traceId);
                }}
              >
                {traceId}
              </span>
            </div>
          )}
          <div className={styles.eventDetailMessage}>
            <span className={styles.eventDetailRowLabel}>Content</span>
            <div className={styles.eventDetailMessageContent}>{event.content}</div>
          </div>
          {payload && payload !== event.content && (
            <div className={styles.eventDetailPayload}>
              <span className={styles.eventDetailRowLabel}>Payload</span>
              <pre className={styles.eventDetailPayloadContent}>
                {typeof payload === 'string' && payload.startsWith('{')
                  ? JSON.stringify(JSON.parse(payload), null, 2)
                  : payload}
              </pre>
            </div>
          )}
          {loading && (
            <div className={styles.eventDetailRow}>
              <span className={styles.eventDetailRowLabel}>Status</span>
              <span className={styles.eventDetailRowValue}>Loading trace data...</span>
            </div>
          )}
        </div>
      </div>
    </div>
  );
};

export default EventDetail;
