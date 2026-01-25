/**
 * EventDetail - Inline event detail view
 * Shows event metadata (Source, Target, Time, Task ID, Content)
 * Plus outliers summary for the selected task
 * Span visualization is now in the Execution Spans panel
 */
import React from 'react';
import type { EventMessage } from './types';
import type { OutliersResponse, SpanOutlier } from '../hooks/useOutliersAnalysis';
import styles from '../ControlPlane.module.css';

export interface EventDetailProps {
  event: EventMessage | null;
  traceId: string | null;
  loading?: boolean;
  onClose: () => void;
  onNavigate?: (direction: 'prev' | 'next') => void;
  currentIndex?: number;
  totalEvents?: number;
  /** Outliers analysis data for the selected task */
  outliers?: OutliersResponse | null;
  /** Loading state for outliers data */
  outliersLoading?: boolean;
  /** Callback when an outlier span is clicked (to highlight in ExecHierarchy) */
  onOutlierClick?: (spanId: string) => void;
}

const getEventTypeLabel = (type: EventMessage['type']): string => {
  switch (type) {
    case 'task_start': return 'Task Started';
    case 'task_complete': return 'Task Complete';
    case 'task_error': return 'Task Error';
    case 'handoff': return 'Handoff';
    case 'approval': return 'Approval Request';
    case 'message': return 'Message';
    case 'session': return 'Session';
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
    case 'session': return 'primary';
  }
};

// Format duration for display
const formatDuration = (ms: number): string => {
  if (ms < 1000) return `${ms}ms`;
  if (ms < 60000) return `${(ms / 1000).toFixed(1)}s`;
  return `${(ms / 60000).toFixed(1)}m`;
};

// Extract workspace name from path
const getWorkspaceName = (workspace: string): string => {
  const parts = workspace.split('/');
  return parts[parts.length - 1] || workspace;
};

// Format outlier metric value for display
const formatOutlierValue = (value: number, metric: string): string => {
  if (metric === 'cost_usd') return `$${value.toFixed(4)}`;
  if (metric === 'duration_ms') return `${value.toLocaleString()}ms`;
  if (metric === 'tokens') return value.toLocaleString();
  return value.toFixed(2);
};

// Format z-score with direction indicator
const formatZScore = (zScore: number): string => {
  const sign = zScore > 0 ? '+' : '';
  return `${sign}${zScore.toFixed(2)}σ`;
};

export const EventDetail: React.FC<EventDetailProps> = ({
  event,
  traceId,
  loading,
  onClose,
  onNavigate,
  currentIndex,
  totalEvents,
  outliers,
  outliersLoading,
  onOutlierClick,
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
          {/* PRIMARY: Directive/Task Description - most important info */}
          {(event.directive || event.directive_full) && (
            <div className={styles.eventDetailDirective}>
              <div className={styles.eventDetailDirectiveText}>
                {event.directive_full || event.directive}
              </div>
            </div>
          )}

          {/* CONTEXT: Source → Target flow with badges */}
          <div className={styles.eventDetailContext}>
            <span className={styles.eventDetailContextFlow}>
              <span className={styles.eventDetailContextSource}>{event.source}</span>
              {event.target && (
                <>
                  <span className={styles.eventDetailContextArrow}>→</span>
                  <span className={styles.eventDetailContextTarget}>{event.target}</span>
                </>
              )}
            </span>
            {event.workspace && (
              <span className={styles.eventDetailContextBadge} title={event.workspace}>
                {getWorkspaceName(event.workspace)}
              </span>
            )}
            <span className={styles.eventDetailContextTime}>
              {new Date(event.timestamp).toLocaleString()}
            </span>
          </div>

          {/* METRICS: Badges for sessions (turns, cost, duration) */}
          {(event.turn_count || event.cost_usd || event.duration_ms) && (
            <div className={styles.eventDetailMetrics}>
              {event.turn_count !== undefined && event.turn_count > 0 && (
                <span className={styles.eventDetailMetricBadge} title="Number of turns">
                  {event.turn_count} turns
                </span>
              )}
              {event.cost_usd !== undefined && event.cost_usd > 0 && (
                <span className={styles.eventDetailMetricBadge} title="AI cost">
                  ${event.cost_usd < 0.01 ? event.cost_usd.toFixed(4) : event.cost_usd.toFixed(2)}
                </span>
              )}
              {event.duration_ms !== undefined && event.duration_ms > 0 && (
                <span className={styles.eventDetailMetricBadge} title="Duration">
                  {formatDuration(event.duration_ms)}
                </span>
              )}
              {event.tokens_in !== undefined && event.tokens_out !== undefined && (
                <span className={styles.eventDetailMetricBadge} title="Tokens (in/out)">
                  {((event.tokens_in + event.tokens_out) / 1000).toFixed(1)}k tokens
                </span>
              )}
            </div>
          )}

          {/* SECONDARY: Content/Title (if different from directive) */}
          {event.content && event.content !== event.directive && (
            <div className={styles.eventDetailMessage}>
              <span className={styles.eventDetailRowLabel}>Content</span>
              <div className={styles.eventDetailMessageContent}>{event.content}</div>
            </div>
          )}

          {/* TERTIARY: Technical details (collapsible feel) */}
          <div className={styles.eventDetailTechnical}>
            {traceId && (
              <div className={styles.eventDetailTechnicalRow}>
                <span className={styles.eventDetailTechnicalLabel}>Task ID</span>
                <span
                  className={styles.eventDetailTechnicalValue}
                  title={`Click to copy: ${traceId}`}
                  onClick={() => navigator.clipboard.writeText(traceId)}
                >
                  {traceId}
                </span>
              </div>
            )}
          </div>

          {/* PAYLOAD: Full message payload (expandable) */}
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

          {/* Outliers Summary Section */}
          {traceId && (outliersLoading || outliers) && (
            <div className={styles.eventDetailOutliers}>
              <div className={styles.eventDetailOutliersHeader}>
                <span className={styles.eventDetailRowLabel}>
                  Outliers {outliers?.outliers?.length ? `(${outliers.outliers.length})` : ''}
                </span>
                {outliersLoading && <span className={styles.eventDetailOutliersLoading}>...</span>}
              </div>
              {outliers?.outliers && outliers.outliers.length > 0 ? (
                <div className={styles.eventDetailOutliersList}>
                  {outliers.outliers.slice(0, 5).map((outlier, idx) => (
                    <button
                      key={`${outlier.span.id}-${outlier.metric}-${idx}`}
                      className={styles.eventDetailOutlierItem}
                      onClick={() => onOutlierClick?.(outlier.span.id)}
                      title={`Click to highlight in Execution Spans\n${outlier.span.name}\n${formatOutlierValue(outlier.value, outlier.metric)} (${formatZScore(outlier.z_score)})`}
                    >
                      <span className={styles.eventDetailOutlierName}>
                        {outlier.span.name.length > 25
                          ? outlier.span.name.substring(0, 22) + '...'
                          : outlier.span.name}
                      </span>
                      <span className={styles.eventDetailOutlierMetric}>
                        {outlier.metric.replace('_usd', '').replace('_ms', '')}
                      </span>
                      <span
                        className={styles.eventDetailOutlierZScore}
                        data-severity={Math.abs(outlier.z_score) > 3 ? 'high' : 'medium'}
                      >
                        {formatZScore(outlier.z_score)}
                      </span>
                    </button>
                  ))}
                </div>
              ) : !outliersLoading && outliers ? (
                <div className={styles.eventDetailOutliersEmpty}>
                  No statistical outliers detected (threshold: {outliers.threshold}σ)
                </div>
              ) : null}
            </div>
          )}
        </div>
      </div>
    </div>
  );
};

export default EventDetail;
