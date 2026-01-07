/**
 * EventDetail - Inline event detail view with trace waterfall
 * Replaces the overlay panel with an inline split view
 */
import React from 'react';
import type { EventMessage, Span } from './types';
import { formatDuration } from './utils';
import styles from '../ControlPlane.module.css';

export interface EventDetailProps {
  event: EventMessage | null;
  spans: Span[];
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
  spans,
  traceId,
  loading,
  onClose,
  onNavigate,
  currentIndex,
  totalEvents,
}) => {
  if (!event) return null;

  const metadata = event.metadata as Record<string, unknown> | undefined;
  const payload = metadata?.payload as string;

  // Calculate trace stats
  const totalDuration = React.useMemo(() => {
    let max = 0;
    const traverse = (span: Span) => {
      const end = span.startMs + span.durationMs;
      if (end > max) max = end;
      span.children?.forEach(traverse);
    };
    spans.forEach(traverse);
    return max;
  }, [spans]);

  const spanCount = React.useMemo(() => {
    let count = 0;
    const traverse = (span: Span) => {
      count++;
      span.children?.forEach(traverse);
    };
    spans.forEach(traverse);
    return count;
  }, [spans]);

  // Zoom state for trace waterfall (1x, 2x, 4x, 8x, 16x)
  const [zoomLevel, setZoomLevel] = React.useState(1);
  const zoomIn = () => setZoomLevel((z) => Math.min(z * 2, 16));
  const zoomOut = () => setZoomLevel((z) => Math.max(z / 2, 1));
  const resetZoom = () => setZoomLevel(1);

  const renderSpan = (span: Span, depth: number = 0): React.ReactNode => {
    const left = totalDuration > 0 ? (span.startMs / totalDuration) * 100 : 0;
    const width = totalDuration > 0 ? (span.durationMs / totalDuration) * 100 : 100;
    // Visual depth indicator: tree-style prefix showing nesting level
    const depthPrefix = depth === 0 ? '' : '├─'.repeat(Math.max(0, depth - 1)) + '└─ ';

    return (
      <React.Fragment key={span.id}>
        <div className={styles.waterfallRow} data-depth={depth}>
          <div className={styles.waterfallLabel} style={{ paddingLeft: `${depth * 20}px` }}>
            <span className={styles.waterfallName}>
              {depth > 0 && (
                <span style={{ color: 'var(--text-tertiary)', fontFamily: 'monospace', marginRight: '4px' }}>
                  {depthPrefix}
                </span>
              )}
              {span.name}
            </span>
            <span className={styles.waterfallDuration}>{formatDuration(span.durationMs)}</span>
          </div>
          <div className={styles.waterfallBar}>
            <div
              className={`${styles.waterfallSegment} ${span.status === 'error' ? styles.waterfallError : ''}`}
              style={{ left: `${left}%`, width: `${Math.max(width, 0.5)}%` }}
              data-depth={depth % 4}
            />
          </div>
        </div>
        {span.children?.map((child) => renderSpan(child, depth + 1))}
      </React.Fragment>
    );
  };

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

      {/* Split content: Info + Trace */}
      <div className={styles.eventDetailContent}>
        {/* Left: Event Info */}
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
                  // Flash feedback
                  const el = document.activeElement as HTMLElement;
                  el?.blur();
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
        </div>

        {/* Right: Trace Waterfall */}
        <div className={styles.eventDetailTrace}>
          <div className={styles.eventDetailTraceHeader}>
            <span className={styles.eventDetailTraceTitle}>Trace Hierarchy</span>
            {spanCount > 0 && (
              <span className={styles.eventDetailTraceStats}>
                {spanCount} spans · {formatDuration(totalDuration)}
              </span>
            )}
            {spanCount > 0 && (
              <div className={styles.zoomControls}>
                <button
                  className={styles.zoomBtn}
                  onClick={zoomOut}
                  disabled={zoomLevel <= 1}
                  title="Zoom out"
                >
                  −
                </button>
                <span className={styles.zoomLevel} onClick={resetZoom} title="Click to reset">
                  {zoomLevel}x
                </span>
                <button
                  className={styles.zoomBtn}
                  onClick={zoomIn}
                  disabled={zoomLevel >= 16}
                  title="Zoom in"
                >
                  +
                </button>
              </div>
            )}
          </div>
          {loading ? (
            <div className={styles.eventDetailTraceEmpty}>
              <span className={styles.eventDetailTraceEmptyIcon}>◎</span>
              <span className={styles.eventDetailTraceEmptyText}>Loading trace data...</span>
              <span className={styles.eventDetailTraceEmptyHint}>
                Searching by trace ID and task ID
              </span>
            </div>
          ) : spans.length === 0 ? (
            <div className={styles.eventDetailTraceEmpty}>
              <span className={styles.eventDetailTraceEmptyIcon}>◎</span>
              <span className={styles.eventDetailTraceEmptyText}>No trace data</span>
              <span className={styles.eventDetailTraceEmptyHint}>
                No spans found for ID: {traceId?.slice(0, 20)}...
              </span>
            </div>
          ) : (
            <div className={styles.waterfallScrollContainer} style={{ overflowX: zoomLevel > 1 ? 'auto' : 'hidden' }}>
              <div style={{ width: `${100 * zoomLevel}%`, minWidth: '100%' }}>
                <div className={styles.waterfallTimeline}>
                  {/* Generate timeline markers based on zoom level */}
                  {Array.from({ length: Math.min(zoomLevel * 2 + 1, 17) }).map((_, i, arr) => {
                    const pct = (i / (arr.length - 1)) * 100;
                    const time = (totalDuration * i) / (arr.length - 1);
                    return (
                      <div key={i} className={styles.timelineMarker} style={{ left: `${pct}%` }}>
                        {formatDuration(time)}
                      </div>
                    );
                  })}
                </div>
                <div className={styles.waterfallRows}>
                  {spans.map((span) => renderSpan(span))}
                </div>
              </div>
            </div>
          )}
        </div>
      </div>
    </div>
  );
};

export default EventDetail;
