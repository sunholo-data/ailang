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

// Display limit constants
const DEFAULT_DISPLAY_LIMIT = 100;
const LOAD_MORE_INCREMENT = 100;

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
  // ALL HOOKS MUST BE CALLED BEFORE ANY EARLY RETURNS (Rules of Hooks)

  // Display limit state for span pagination
  const [displayLimit, setDisplayLimit] = React.useState(DEFAULT_DISPLAY_LIMIT);

  // Flatten spans for counting and limiting
  const flattenedSpans = React.useMemo(() => {
    const result: Array<{ span: Span; depth: number }> = [];
    const flatten = (spanList: Span[], depth: number) => {
      for (const span of spanList) {
        result.push({ span, depth });
        if (span.children) {
          flatten(span.children, depth + 1);
        }
      }
    };
    flatten(spans, 0);
    return result;
  }, [spans]);

  // Total span count for display
  const totalSpanCount = flattenedSpans.length;

  // Limited spans for rendering
  const limitedSpans = React.useMemo(() => {
    return flattenedSpans.slice(0, displayLimit);
  }, [flattenedSpans, displayLimit]);

  const hasMore = totalSpanCount > displayLimit;
  const loadMore = () => setDisplayLimit((prev) => prev + LOAD_MORE_INCREMENT);
  const showAll = () => setDisplayLimit(totalSpanCount);

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

  // spanCount is now totalSpanCount from flattened spans above

  // Zoom state for trace waterfall (1x, 2x, 4x, 8x, 16x)
  const [zoomLevel, setZoomLevel] = React.useState(1);
  const zoomIn = () => setZoomLevel((z) => Math.min(z * 2, 16));
  const zoomOut = () => setZoomLevel((z) => Math.max(z / 2, 1));
  const resetZoom = () => setZoomLevel(1);

  // Early return AFTER all hooks
  if (!event) return null;

  const metadata = event.metadata as Record<string, unknown> | undefined;
  const payload = metadata?.payload as string;

  // Render a single span row (flat, not recursive)
  const renderSpanRow = (span: Span, depth: number, index: number): React.ReactNode => {
    const left = totalDuration > 0 ? (span.startMs / totalDuration) * 100 : 0;
    const width = totalDuration > 0 ? (span.durationMs / totalDuration) * 100 : 100;
    // Visual depth indicator: tree-style prefix showing nesting level
    const depthPrefix = depth === 0 ? '' : '├─'.repeat(Math.max(0, depth - 1)) + '└─ ';

    return (
      <div key={`${span.id}-${index}`} className={styles.waterfallRow} data-depth={depth}>
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
            {totalSpanCount > 0 && (
              <span className={styles.eventDetailTraceStats}>
                {hasMore ? `${displayLimit} of ${totalSpanCount}` : totalSpanCount} spans · {formatDuration(totalDuration)}
              </span>
            )}
            {totalSpanCount > 0 && (
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
                  {limitedSpans.map(({ span, depth }, index) => renderSpanRow(span, depth, index))}
                </div>
                {hasMore && (
                  <div className={styles.loadMoreContainer}>
                    <button className={styles.loadMoreBtn} onClick={loadMore}>
                      Load {Math.min(LOAD_MORE_INCREMENT, totalSpanCount - displayLimit)} more
                    </button>
                    <button className={styles.loadMoreBtn} onClick={showAll}>
                      Show all {totalSpanCount}
                    </button>
                  </div>
                )}
              </div>
            </div>
          )}
        </div>
      </div>
    </div>
  );
};

export default EventDetail;
