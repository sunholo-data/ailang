import React, { useMemo, useState } from 'react';
import { useTaskTimeline, TaskTimelineItem } from '../../../hooks/useObservatory';
import styles from './TaskTimeline.module.css';

interface TaskTimelineProps {
  taskId: string;
}

// Format duration in human-readable format
function formatDuration(ms: number | undefined): string {
  if (!ms) return '-';
  if (ms < 1000) return `${ms.toFixed(0)}ms`;
  if (ms < 60000) return `${(ms / 1000).toFixed(2)}s`;
  return `${(ms / 60000).toFixed(2)}m`;
}

// Format cost in USD
function formatCost(usd: number | undefined): string {
  if (!usd) return '-';
  if (usd < 0.01) return `$${usd.toFixed(4)}`;
  return `$${usd.toFixed(2)}`;
}

// Format large numbers
function formatNumber(n: number | undefined): string {
  if (!n) return '0';
  if (n >= 1000000) return `${(n / 1000000).toFixed(1)}M`;
  if (n >= 1000) return `${(n / 1000).toFixed(1)}K`;
  return n.toString();
}

// Get status class
function getStatusClass(status: string | undefined): string {
  switch (status) {
    case 'ok':
    case 'OK':
      return styles.statusOk;
    case 'error':
    case 'ERROR':
      return styles.statusError;
    case 'running':
      return styles.statusRunning;
    default:
      return styles.statusUnset;
  }
}

// Get provider class for color coding
function getProviderClass(provider: string | undefined): string {
  switch (provider) {
    case 'claude':
      return styles.providerClaude;
    case 'gemini':
      return styles.providerGemini;
    case 'openai':
      return styles.providerOpenai;
    default:
      return styles.providerOther;
  }
}

// Calculate timeline bounds and scale
function useTimelineBounds(items: TaskTimelineItem[]) {
  return useMemo(() => {
    if (items.length === 0) return { minTime: 0, maxTime: 0, totalDuration: 0 };

    const times = items
      .filter(item => item.start_time)
      .map(item => ({
        start: new Date(item.start_time!).getTime(),
        end: item.end_time ? new Date(item.end_time).getTime() : Date.now(),
      }));

    if (times.length === 0) return { minTime: 0, maxTime: 0, totalDuration: 0 };

    const minTime = Math.min(...times.map(t => t.start));
    const maxTime = Math.max(...times.map(t => t.end));
    const totalDuration = maxTime - minTime;

    return { minTime, maxTime, totalDuration };
  }, [items]);
}

// Single timeline bar
function TimelineBar({ item, bounds }: { item: TaskTimelineItem; bounds: ReturnType<typeof useTimelineBounds> }) {
  const [isHovered, setIsHovered] = useState(false);

  if (!item.span_id || !item.start_time || bounds.totalDuration === 0) {
    return null;
  }

  const startTime = new Date(item.start_time).getTime();
  const endTime = item.end_time ? new Date(item.end_time).getTime() : Date.now();
  const duration = item.duration_ms || (endTime - startTime);

  const leftPercent = ((startTime - bounds.minTime) / bounds.totalDuration) * 100;
  const widthPercent = Math.max(1, (duration / bounds.totalDuration) * 100);

  return (
    <div className={styles.timelineRow}>
      <div className={styles.spanLabel}>
        <span className={`${styles.spanName} ${getProviderClass(item.provider)}`}>
          {item.span_name || item.span_id.substring(0, 8)}
        </span>
        <span className={`${styles.spanStatus} ${getStatusClass(item.span_status)}`}>
          {item.span_status || 'unset'}
        </span>
      </div>
      <div className={styles.timelineTrack}>
        <div
          className={`${styles.timelineBar} ${getStatusClass(item.span_status)} ${getProviderClass(item.provider)}`}
          style={{
            left: `${leftPercent}%`,
            width: `${widthPercent}%`,
          }}
          onMouseEnter={() => setIsHovered(true)}
          onMouseLeave={() => setIsHovered(false)}
        />
        {isHovered && (
          <div
            className={styles.tooltip}
            style={{
              left: `${leftPercent + widthPercent / 2}%`,
            }}
          >
            <div className={styles.tooltipTitle}>{item.span_name}</div>
            <div className={styles.tooltipRow}>
              <span>Duration:</span>
              <strong>{formatDuration(item.duration_ms)}</strong>
            </div>
            {item.tokens_in !== undefined && item.tokens_in > 0 && (
              <div className={styles.tooltipRow}>
                <span>Tokens In:</span>
                <strong>{formatNumber(item.tokens_in)}</strong>
              </div>
            )}
            {item.tokens_out !== undefined && item.tokens_out > 0 && (
              <div className={styles.tooltipRow}>
                <span>Tokens Out:</span>
                <strong>{formatNumber(item.tokens_out)}</strong>
              </div>
            )}
            {item.cost_usd !== undefined && item.cost_usd > 0 && (
              <div className={styles.tooltipRow}>
                <span>Cost:</span>
                <strong>{formatCost(item.cost_usd)}</strong>
              </div>
            )}
            {item.provider && (
              <div className={styles.tooltipRow}>
                <span>Provider:</span>
                <strong>{item.provider}</strong>
              </div>
            )}
          </div>
        )}
      </div>
      <div className={styles.spanMetrics}>
        <span>{formatDuration(item.duration_ms)}</span>
        {item.cost_usd !== undefined && item.cost_usd > 0 && (
          <span className={styles.cost}>{formatCost(item.cost_usd)}</span>
        )}
      </div>
    </div>
  );
}

// Time axis
function TimeAxis({ bounds }: { bounds: ReturnType<typeof useTimelineBounds> }) {
  if (bounds.totalDuration === 0) return null;

  // Calculate sensible tick marks
  const tickCount = 5;
  const ticks = [];
  for (let i = 0; i <= tickCount; i++) {
    const time = bounds.minTime + (bounds.totalDuration * i) / tickCount;
    const leftPercent = (i / tickCount) * 100;
    const label = i === 0 ? '0s' : formatDuration(bounds.totalDuration * i / tickCount);
    ticks.push({ time, leftPercent, label });
  }

  return (
    <div className={styles.timeAxis}>
      <div className={styles.axisLabel}>Time</div>
      <div className={styles.axisTrack}>
        {ticks.map((tick, idx) => (
          <div
            key={idx}
            className={styles.axisTick}
            style={{ left: `${tick.leftPercent}%` }}
          >
            <div className={styles.tickMark} />
            <span className={styles.tickLabel}>{tick.label}</span>
          </div>
        ))}
      </div>
    </div>
  );
}

export function TaskTimeline({ taskId }: TaskTimelineProps) {
  const { timeline, loading, error, refresh } = useTaskTimeline(taskId);
  const bounds = useTimelineBounds(timeline);

  // Filter out task-level entry (no span_id) to show only spans
  const spans = useMemo(() =>
    timeline.filter(item => item.span_id),
    [timeline]
  );

  // Calculate totals
  const totals = useMemo(() => {
    return spans.reduce(
      (acc, item) => ({
        duration: acc.duration + (item.duration_ms || 0),
        tokens_in: acc.tokens_in + (item.tokens_in || 0),
        tokens_out: acc.tokens_out + (item.tokens_out || 0),
        cost: acc.cost + (item.cost_usd || 0),
      }),
      { duration: 0, tokens_in: 0, tokens_out: 0, cost: 0 }
    );
  }, [spans]);

  if (loading) {
    return <div className={styles.loading}>Loading timeline...</div>;
  }

  if (error) {
    return <div className={styles.error}>Error: {error}</div>;
  }

  if (spans.length === 0) {
    return (
      <div className={styles.container}>
        <div className={styles.header}>
          <h3>Timeline</h3>
          <button onClick={refresh} className={styles.refreshButton}>
            Refresh
          </button>
        </div>
        <div className={styles.empty}>No span timeline data available</div>
      </div>
    );
  }

  return (
    <div className={styles.container}>
      <div className={styles.header}>
        <h3>Timeline ({spans.length} spans)</h3>
        <button onClick={refresh} className={styles.refreshButton}>
          Refresh
        </button>
      </div>

      {/* Summary stats */}
      <div className={styles.summary}>
        <div className={styles.summaryItem}>
          <span className={styles.summaryLabel}>Total Duration</span>
          <span className={styles.summaryValue}>{formatDuration(bounds.totalDuration)}</span>
        </div>
        <div className={styles.summaryItem}>
          <span className={styles.summaryLabel}>Total Tokens</span>
          <span className={styles.summaryValue}>
            {formatNumber(totals.tokens_in)} in / {formatNumber(totals.tokens_out)} out
          </span>
        </div>
        <div className={styles.summaryItem}>
          <span className={styles.summaryLabel}>Total Cost</span>
          <span className={styles.summaryValue}>{formatCost(totals.cost)}</span>
        </div>
      </div>

      {/* Timeline visualization */}
      <div className={styles.timeline}>
        <TimeAxis bounds={bounds} />
        <div className={styles.timelineBody}>
          {spans.map((item, idx) => (
            <TimelineBar key={item.span_id || idx} item={item} bounds={bounds} />
          ))}
        </div>
      </div>
    </div>
  );
}

export default TaskTimeline;
