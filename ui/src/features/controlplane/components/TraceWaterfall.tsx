/**
 * TraceWaterfall - Span timeline visualization
 * Limited to 100 spans by default with "Show more" option
 */
import React, { useMemo, useState } from 'react';
import type { Span } from './types';
import { formatDuration } from './utils';
import styles from '../ControlPlane.module.css';

export interface TraceWaterfallProps {
  spans: Span[];
  selectedTraceId?: string | null;
  loading?: boolean;
  // Span type filtering (Milestone 14) - generic filter for any span type
  hiddenSpanTypes?: Set<string>;
  onToggleSpanType?: (spanType: string) => void;
  // Chat context indicator (M-CHAT-HISTORY-DB) - callback when user clicks chat icon
  onChatContextClick?: (span: Span) => void;
}

// Default display limit
const DEFAULT_DISPLAY_LIMIT = 100;
const LOAD_MORE_INCREMENT = 100;

export const TraceWaterfall: React.FC<TraceWaterfallProps> = ({
  spans,
  selectedTraceId,
  loading,
  hiddenSpanTypes,
  onChatContextClick,
}) => {
  // Display limit state - start with 100, can load more
  const [displayLimit, setDisplayLimit] = useState(DEFAULT_DISPLAY_LIMIT);

  // Filter spans by hidden types (recursive, promotes children)
  const filteredSpans = useMemo(() => {
    if (!hiddenSpanTypes || hiddenSpanTypes.size === 0) return spans;

    const filterSpans = (spanList: Span[]): Span[] => {
      const result: Span[] = [];
      for (const span of spanList) {
        const isHidden = hiddenSpanTypes.has(span.name);
        if (isHidden) {
          // Promote children to this level
          if (span.children && span.children.length > 0) {
            result.push(...filterSpans(span.children));
          }
        } else {
          // Keep span, filter its children
          result.push({
            ...span,
            children: span.children ? filterSpans(span.children) : undefined,
          });
        }
      }
      return result;
    };

    return filterSpans(spans);
  }, [spans, hiddenSpanTypes]);

  // Flatten spans for counting and limiting
  const flattenedSpans = useMemo(() => {
    const result: Array<{ span: Span; depth: number }> = [];
    const flatten = (spanList: Span[], depth: number) => {
      for (const span of spanList) {
        result.push({ span, depth });
        if (span.children) {
          flatten(span.children, depth + 1);
        }
      }
    };
    flatten(filteredSpans, 0);
    return result;
  }, [filteredSpans]);

  // Total count for display
  const totalSpanCount = flattenedSpans.length;

  // Limited spans for rendering
  const limitedSpans = useMemo(() => {
    return flattenedSpans.slice(0, displayLimit);
  }, [flattenedSpans, displayLimit]);

  const hasMore = totalSpanCount > displayLimit;
  const loadMore = () => setDisplayLimit(prev => prev + LOAD_MORE_INCREMENT);
  const showAll = () => setDisplayLimit(totalSpanCount);

  // Calculate total duration from all spans (not just limited)
  const totalDuration = useMemo(() => {
    let max = 0;
    for (const { span } of flattenedSpans) {
      const end = span.startMs + span.durationMs;
      if (end > max) max = end;
    }
    return max;
  }, [flattenedSpans]);

  // Calculate overall status from all spans
  const hasError = useMemo(() => {
    return flattenedSpans.some(({ span }) => span.status === 'error');
  }, [flattenedSpans]);

  // Zoom state for trace waterfall (1x, 2x, 4x, 8x, 16x)
  const [zoomLevel, setZoomLevel] = useState(1);
  const zoomIn = () => setZoomLevel((z) => Math.min(z * 2, 16));
  const zoomOut = () => setZoomLevel((z) => Math.max(z / 2, 1));
  const resetZoom = () => setZoomLevel(1);

  // Check if a span has chat context available (via session.id attribute)
  const hasChatContext = (span: Span): boolean => {
    return !!(span.attributes?.['session.id']);
  };

  // Render a single span row (flat, not recursive)
  const renderSpanRow = (span: Span, depth: number, index: number): React.ReactNode => {
    const left = totalDuration > 0 ? (span.startMs / totalDuration) * 100 : 0;
    const width = totalDuration > 0 ? (span.durationMs / totalDuration) * 100 : 100;
    // Visual depth indicator: tree-style prefix showing nesting level
    const depthPrefix = depth === 0 ? '' : '├─'.repeat(Math.max(0, depth - 1)) + '└─ ';
    const hasChat = hasChatContext(span);

    return (
      <div key={`${span.id}-${index}`} className={styles.waterfallRow} data-depth={depth} data-span-id={span.id}>
        <div className={styles.waterfallLabel} style={{ paddingLeft: `${depth * 20}px` }}>
          <span className={styles.waterfallName}>
            {depth > 0 && (
              <span style={{ color: 'var(--text-tertiary)', fontFamily: 'monospace', marginRight: '4px' }}>
                {depthPrefix}
              </span>
            )}
            {span.display_name || span.name}
          </span>
          {hasChat && onChatContextClick && (
            <button
              className={styles.waterfallChatBtn}
              onClick={(e) => {
                e.stopPropagation();
                onChatContextClick(span);
              }}
              title="View conversation context"
            >
              💬
            </button>
          )}
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

  const isEmpty = filteredSpans.length === 0;

  return (
    <div className={styles.waterfallContainer}>
      <div className={styles.waterfallHeader}>
        <h3 className={styles.panelTitle}>
          <span className={styles.panelIcon}>▥</span>
          Trace Waterfall
          {selectedTraceId && (
            <span className={styles.traceIdBadge} title={selectedTraceId}>
              {selectedTraceId.slice(0, 8)}...
            </span>
          )}
        </h3>
        {!isEmpty && (
          <div className={styles.waterfallMeta}>
            <span className={styles.metaItem}>
              <span className={styles.metaLabel}>Spans</span>
              <span className={styles.metaValue}>
                {hasMore ? `${displayLimit} of ${totalSpanCount}` : totalSpanCount}
              </span>
            </span>
            <span className={styles.metaItem}>
              <span className={styles.metaLabel}>Duration</span>
              <span className={styles.metaValue}>{formatDuration(totalDuration)}</span>
            </span>
            <span className={styles.metaItem}>
              <span className={styles.metaLabel}>Status</span>
              <span className={`${styles.metaValue} ${hasError ? styles.metaError : styles.metaSuccess}`}>
                {hasError ? '✕ Error' : '✓ Complete'}
              </span>
            </span>
            <div className={styles.zoomControls}>
              <button className={styles.zoomBtn} onClick={zoomOut} disabled={zoomLevel <= 1} title="Zoom out">−</button>
              <span className={styles.zoomLevel} onClick={resetZoom} title="Click to reset">{zoomLevel}x</span>
              <button className={styles.zoomBtn} onClick={zoomIn} disabled={zoomLevel >= 16} title="Zoom in">+</button>
            </div>
          </div>
        )}
      </div>

      {loading && (
        <div className={styles.waterfallEmpty}>
          <span className={styles.waterfallEmptyIcon}>◎</span>
          <span className={styles.waterfallEmptyText}>Loading trace...</span>
        </div>
      )}

      {!loading && isEmpty && (
        <div className={styles.waterfallEmpty}>
          <span className={styles.waterfallEmptyIcon}>◎</span>
          <span className={styles.waterfallEmptyText}>
            {selectedTraceId ? 'No spans found for this trace' : 'Select an event to view its trace'}
          </span>
          <span className={styles.waterfallEmptyHint}>
            Click on an event in the queue to see its execution hierarchy
          </span>
        </div>
      )}

      {!loading && !isEmpty && (
        <div className={styles.waterfallScrollContainer} style={{ overflowX: zoomLevel > 1 ? 'auto' : 'hidden' }}>
          <div style={{ width: `${100 * zoomLevel}%`, minWidth: '100%' }}>
            <div className={styles.waterfallTimeline}>
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
  );
};

export default TraceWaterfall;
