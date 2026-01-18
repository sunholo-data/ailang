/**
 * OutliersChart - Display statistical outliers detected in task spans
 *
 * Shows metric statistics and a ranked list of outlier spans with z-scores
 */
import React from 'react';
import type {
  OutliersResponse,
  SpanOutlier,
  TaskMetricStats,
} from '../../features/controlplane/hooks/useOutliersAnalysis';
import styles from './OutliersChart.module.css';

export interface OutliersChartProps {
  data: OutliersResponse | null;
  loading?: boolean;
  error?: string | null;
  onSpanClick?: (outlier: SpanOutlier) => void;
  onBack?: () => void;
}

// Format helpers
const formatCost = (cost: number): string => {
  if (cost >= 1) return `$${cost.toFixed(2)}`;
  if (cost >= 0.01) return `$${cost.toFixed(2)}`;
  return `$${cost.toFixed(4)}`;
};

const formatDuration = (ms: number): string => {
  if (ms >= 60000) return `${(ms / 60000).toFixed(1)}m`;
  if (ms >= 1000) return `${(ms / 1000).toFixed(1)}s`;
  return `${ms.toFixed(0)}ms`;
};

const formatTokens = (tokens: number): string => {
  if (tokens >= 1000000) return `${(tokens / 1000000).toFixed(1)}M`;
  if (tokens >= 1000) return `${(tokens / 1000).toFixed(1)}K`;
  return tokens.toFixed(0);
};

const formatMetricValue = (value: number, metric: string): string => {
  switch (metric) {
    case 'cost_usd':
      return formatCost(value);
    case 'duration_ms':
      return formatDuration(value);
    case 'tokens':
      return formatTokens(value);
    default:
      return value.toFixed(2);
  }
};

const METRIC_LABELS: Record<string, string> = {
  cost_usd: 'Cost',
  duration_ms: 'Duration',
  tokens: 'Tokens',
};

const StatsTable: React.FC<{ stats: TaskMetricStats[] }> = ({ stats }) => (
  <div className={styles.statsTable}>
    <div className={styles.statsHeader}>
      <span className={styles.statsCol}>Metric</span>
      <span className={styles.statsCol}>Count</span>
      <span className={styles.statsCol}>Sum</span>
      <span className={styles.statsCol}>Mean</span>
      <span className={styles.statsCol}>StdDev</span>
    </div>
    {stats.map((stat) => (
      <div key={stat.metric} className={styles.statsRow}>
        <span className={styles.statsCol}>{METRIC_LABELS[stat.metric] || stat.metric}</span>
        <span className={styles.statsCol}>{stat.count}</span>
        <span className={styles.statsCol}>{formatMetricValue(stat.sum, stat.metric)}</span>
        <span className={styles.statsCol}>{formatMetricValue(stat.mean, stat.metric)}</span>
        <span className={styles.statsCol}>{formatMetricValue(stat.std_dev, stat.metric)}</span>
      </div>
    ))}
  </div>
);

const OutliersList: React.FC<{
  outliers: SpanOutlier[];
  onSpanClick?: (outlier: SpanOutlier) => void;
}> = ({ outliers, onSpanClick }) => (
  <div className={styles.outliersList}>
    {outliers.map((outlier, index) => (
      <div
        key={`${outlier.span.id}-${outlier.metric}`}
        className={styles.outlierItem}
        onClick={() => onSpanClick?.(outlier)}
      >
        <div className={styles.outlierRank}>#{index + 1}</div>
        <div className={styles.outlierContent}>
          <div className={styles.outlierName}>{outlier.span.name}</div>
          <div className={styles.outlierDetails}>
            <span className={styles.outlierMetric}>
              {METRIC_LABELS[outlier.metric] || outlier.metric}:
            </span>
            <span className={styles.outlierValue}>
              {formatMetricValue(outlier.value, outlier.metric)}
            </span>
            <span className={styles.outlierZScore}>
              z={outlier.z_score.toFixed(2)}
            </span>
            <span className={styles.outlierPercent}>
              {outlier.percent_of_total.toFixed(1)}% of total
            </span>
          </div>
          {outlier.span.model && (
            <div className={styles.outlierMeta}>
              Model: {outlier.span.model}
              {outlier.span.provider && ` | Provider: ${outlier.span.provider}`}
            </div>
          )}
        </div>
      </div>
    ))}
  </div>
);

export const OutliersChart: React.FC<OutliersChartProps> = ({
  data,
  loading,
  error,
  onSpanClick,
  onBack,
}) => {
  if (loading) {
    return (
      <div className={styles.container}>
        <div className={styles.loading}>Analyzing spans...</div>
      </div>
    );
  }

  if (error) {
    return (
      <div className={styles.container}>
        <div className={styles.error}>
          <span className={styles.errorIcon}>!</span>
          {error}
        </div>
      </div>
    );
  }

  if (!data) {
    return (
      <div className={styles.container}>
        <div className={styles.empty}>
          <span>Select a task from the Evolution chart to analyze outliers</span>
        </div>
      </div>
    );
  }

  return (
    <div className={styles.container}>
      <div className={styles.header}>
        {onBack && (
          <button className={styles.backButton} onClick={onBack}>
            Back
          </button>
        )}
        <div className={styles.taskInfo}>
          <span className={styles.taskTitle}>{data.task_title || data.task_id}</span>
          <span className={styles.taskMeta}>
            {data.span_count} spans | Threshold: {data.threshold.toFixed(1)}
          </span>
        </div>
      </div>

      <div className={styles.section}>
        <div className={styles.sectionTitle}>Metric Statistics</div>
        <StatsTable stats={data.stats} />
      </div>

      <div className={styles.section}>
        <div className={styles.sectionTitle}>
          Outliers Detected ({data.outliers.length})
        </div>
        {data.outliers.length === 0 ? (
          <div className={styles.noOutliers}>
            No outliers detected (all spans within threshold)
          </div>
        ) : (
          <OutliersList outliers={data.outliers} onSpanClick={onSpanClick} />
        )}
      </div>

      <div className={styles.cliHint}>
        <span className={styles.cliLabel}>CLI:</span>
        <code className={styles.cliCommand}>{data.cli_command}</code>
      </div>
    </div>
  );
};

export default OutliersChart;
