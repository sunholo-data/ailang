/**
 * TraceWaterfall - Span timeline visualization
 */
import React, { useMemo, useState } from 'react';
import type { Span } from './types';
import { formatDuration } from './utils';
import styles from '../ControlPlane.module.css';

export interface TraceWaterfallProps {
  spans: Span[];
  selectedTraceId?: string | null;
  loading?: boolean;
}

export const TraceWaterfall: React.FC<TraceWaterfallProps> = ({ spans, selectedTraceId, loading }) => {
  const totalDuration = useMemo(() => {
    let max = 0;
    const traverse = (span: Span) => {
      const end = span.startMs + span.durationMs;
      if (end > max) max = end;
      span.children?.forEach(traverse);
    };
    spans.forEach(traverse);
    return max;
  }, [spans]);

  // Count total spans including children
  const spanCount = useMemo(() => {
    let count = 0;
    const traverse = (span: Span) => {
      count++;
      span.children?.forEach(traverse);
    };
    spans.forEach(traverse);
    return count;
  }, [spans]);

  // Calculate overall status
  const hasError = useMemo(() => {
    let error = false;
    const traverse = (span: Span) => {
      if (span.status === 'error') error = true;
      span.children?.forEach(traverse);
    };
    spans.forEach(traverse);
    return error;
  }, [spans]);

  // Zoom state for trace waterfall (1x, 2x, 4x, 8x, 16x)
  const [zoomLevel, setZoomLevel] = useState(1);
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

  const isEmpty = spans.length === 0;

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
              <span className={styles.metaValue}>{spanCount}</span>
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
              {spans.map((span) => renderSpan(span))}
            </div>
          </div>
        </div>
      )}
    </div>
  );
};

export default TraceWaterfall;
