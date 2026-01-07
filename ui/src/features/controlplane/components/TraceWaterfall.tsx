/**
 * TraceWaterfall - Span timeline visualization
 */
import React, { useMemo } from 'react';
import type { Span } from './types';
import { formatDuration } from './utils';
import styles from '../ControlPlane.module.css';

export interface TraceWaterfallProps {
  spans: Span[];
}

export const TraceWaterfall: React.FC<TraceWaterfallProps> = ({ spans }) => {
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

  const renderSpan = (span: Span, depth: number = 0): React.ReactNode => {
    const left = (span.startMs / totalDuration) * 100;
    const width = (span.durationMs / totalDuration) * 100;

    return (
      <div key={span.id} className={styles.waterfallRow}>
        <div className={styles.waterfallLabel} style={{ paddingLeft: `${12 + depth * 16}px` }}>
          <span className={styles.waterfallName}>{span.name}</span>
          <span className={styles.waterfallDuration}>{formatDuration(span.durationMs)}</span>
        </div>
        <div className={styles.waterfallBar}>
          <div
            className={styles.waterfallSegment}
            style={{ left: `${left}%`, width: `${Math.max(width, 0.5)}%` }}
            data-depth={depth % 4}
          />
        </div>
        {span.children?.map((child) => renderSpan(child, depth + 1))}
      </div>
    );
  };

  return (
    <div className={styles.waterfallContainer}>
      <div className={styles.waterfallHeader}>
        <h3 className={styles.panelTitle}>
          <span className={styles.panelIcon}>▥</span>
          Trace Waterfall
        </h3>
        <div className={styles.waterfallMeta}>
          <span className={styles.metaItem}>
            <span className={styles.metaLabel}>Duration</span>
            <span className={styles.metaValue}>{formatDuration(totalDuration)}</span>
          </span>
          <span className={styles.metaItem}>
            <span className={styles.metaLabel}>Status</span>
            <span className={`${styles.metaValue} ${styles.metaSuccess}`}>✓ Complete</span>
          </span>
          <span className={styles.metaItem}>
            <span className={styles.metaLabel}>Cost</span>
            <span className={styles.metaValue}>$0.0847</span>
          </span>
        </div>
      </div>
      <div className={styles.waterfallTimeline}>
        <div className={styles.timelineMarker} style={{ left: '0%' }}>0s</div>
        <div className={styles.timelineMarker} style={{ left: '25%' }}>{formatDuration(totalDuration * 0.25)}</div>
        <div className={styles.timelineMarker} style={{ left: '50%' }}>{formatDuration(totalDuration * 0.5)}</div>
        <div className={styles.timelineMarker} style={{ left: '75%' }}>{formatDuration(totalDuration * 0.75)}</div>
        <div className={styles.timelineMarker} style={{ left: '100%' }}>{formatDuration(totalDuration)}</div>
      </div>
      <div className={styles.waterfallRows}>
        {spans.map((span) => renderSpan(span))}
      </div>
    </div>
  );
};

export default TraceWaterfall;
