import React from 'react';
import { TaskResourceMetrics } from '../../../types';
import styles from './TaskExecution.module.css';

interface ResourceMetricsProps {
  metrics: TaskResourceMetrics | null;
  compact?: boolean;
}

const formatBytes = (mb: number): string => {
  if (mb >= 1024) {
    return `${(mb / 1024).toFixed(1)} GB`;
  }
  return `${mb.toFixed(0)} MB`;
};

const formatCost = (cost: number): string => {
  if (cost < 0.01) {
    return `$${cost.toFixed(4)}`;
  }
  return `$${cost.toFixed(2)}`;
};

const formatNumber = (n: number): string => {
  if (n >= 1000000) {
    return `${(n / 1000000).toFixed(1)}M`;
  }
  if (n >= 1000) {
    return `${(n / 1000).toFixed(1)}K`;
  }
  return n.toString();
};

export const ResourceMetrics: React.FC<ResourceMetricsProps> = ({
  metrics,
  compact = false,
}) => {
  if (!metrics) {
    return (
      <div className={styles.resourceMetrics}>
        <div className={styles.metricsPlaceholder}>
          No metrics available
        </div>
      </div>
    );
  }

  if (compact) {
    return (
      <div className={styles.resourceMetricsCompact}>
        <span className={styles.metricItem}>
          CPU: {metrics.cpu_percent.toFixed(0)}%
        </span>
        <span className={styles.metricItem}>
          RAM: {formatBytes(metrics.memory_mb)}
        </span>
        <span className={styles.metricItem}>
          Tokens: {formatNumber(metrics.tokens_in + metrics.tokens_out)}
        </span>
        <span className={styles.metricItem}>
          Cost: {formatCost(metrics.cost)}
        </span>
      </div>
    );
  }

  return (
    <div className={styles.resourceMetrics}>
      <div className={styles.metricsGrid}>
        {/* CPU */}
        <div className={styles.metricCard}>
          <div className={styles.metricLabel}>CPU</div>
          <div className={styles.metricValue}>{metrics.cpu_percent.toFixed(1)}%</div>
          <div className={styles.metricBar}>
            <div
              className={styles.metricBarFill}
              style={{ width: `${Math.min(100, metrics.cpu_percent)}%` }}
            />
          </div>
          <div className={styles.metricPeak}>Peak: {metrics.peak_cpu.toFixed(1)}%</div>
        </div>

        {/* Memory */}
        <div className={styles.metricCard}>
          <div className={styles.metricLabel}>Memory</div>
          <div className={styles.metricValue}>{formatBytes(metrics.memory_mb)}</div>
          <div className={styles.metricBar}>
            <div
              className={styles.metricBarFill}
              style={{ width: `${Math.min(100, (metrics.memory_mb / 8192) * 100)}%` }}
            />
          </div>
          <div className={styles.metricPeak}>Peak: {formatBytes(metrics.peak_memory)}</div>
        </div>

        {/* Tokens */}
        <div className={styles.metricCard}>
          <div className={styles.metricLabel}>Tokens</div>
          <div className={styles.metricValue}>
            {formatNumber(metrics.tokens_in + metrics.tokens_out)}
          </div>
          <div className={styles.metricDetail}>
            <span>In: {formatNumber(metrics.tokens_in)}</span>
            <span>Out: {formatNumber(metrics.tokens_out)}</span>
          </div>
        </div>

        {/* Cost */}
        <div className={styles.metricCard}>
          <div className={styles.metricLabel}>Cost</div>
          <div className={styles.metricValue}>{formatCost(metrics.cost)}</div>
          <div className={styles.metricDetail}>
            <span>Running total</span>
          </div>
        </div>
      </div>
    </div>
  );
};

export default ResourceMetrics;
