import { useState, useEffect, useCallback } from 'react';
import { AggregatedMetrics } from '../../types';
import styles from './MetricsCard.module.css';

interface MetricsCardProps {
  scopeType: 'global' | 'agent' | 'thread';
  scopeId?: string;
  title?: string;
  compact?: boolean;
}

// Format milliseconds to human readable duration
function formatDuration(ms: number): string {
  if (ms < 1000) return `${ms}ms`;
  if (ms < 60000) return `${(ms / 1000).toFixed(1)}s`;
  if (ms < 3600000) return `${(ms / 60000).toFixed(1)}m`;
  return `${(ms / 3600000).toFixed(1)}h`;
}

// Format number with commas
function formatNumber(n: number): string {
  return n.toLocaleString();
}

// Format cost with appropriate precision
function formatCost(cost: number): string {
  if (cost === 0) return '$0.00';
  if (cost < 0.01) return `$${cost.toFixed(4)}`;
  return `$${cost.toFixed(2)}`;
}

export function MetricsCard({ scopeType, scopeId = '', title, compact = false }: MetricsCardProps) {
  const [metrics, setMetrics] = useState<AggregatedMetrics | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  const fetchMetrics = useCallback(async () => {
    try {
      let url = '/api/metrics';
      if (scopeType !== 'global') {
        url = `/api/metrics/${scopeType}/${scopeId}`;
      }

      const response = await fetch(url);
      if (!response.ok) {
        throw new Error(`Failed to fetch metrics: ${response.status}`);
      }

      const data = await response.json();
      setMetrics(data);
      setError(null);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to load metrics');
    } finally {
      setLoading(false);
    }
  }, [scopeType, scopeId]);

  useEffect(() => {
    fetchMetrics();
    // Refresh metrics every 30 seconds
    const interval = setInterval(fetchMetrics, 30000);
    return () => clearInterval(interval);
  }, [fetchMetrics]);

  if (loading) {
    return (
      <div className={`${styles.card} ${compact ? styles.compact : ''}`}>
        <div className={styles.loading}>Loading metrics...</div>
      </div>
    );
  }

  if (error) {
    return (
      <div className={`${styles.card} ${compact ? styles.compact : ''} ${styles.error}`}>
        <div className={styles.errorText}>{error}</div>
      </div>
    );
  }

  if (!metrics) {
    return null;
  }

  const displayTitle = title || (
    scopeType === 'global' ? 'Global Metrics' :
    scopeType === 'agent' ? `Agent: ${scopeId}` :
    `Thread: ${scopeId.slice(0, 12)}...`
  );

  if (compact) {
    return (
      <div className={`${styles.card} ${styles.compact}`}>
        <div className={styles.compactRow}>
          <span className={styles.compactLabel}>Runs:</span>
          <span className={styles.compactValue}>{formatNumber(metrics.total_runs)}</span>
          <span className={styles.compactLabel}>Tokens:</span>
          <span className={styles.compactValue}>{formatNumber(metrics.total_tokens)}</span>
          <span className={styles.compactLabel}>Cost:</span>
          <span className={styles.compactValue}>{formatCost(metrics.total_cost)}</span>
        </div>
      </div>
    );
  }

  return (
    <div className={styles.card}>
      <h3 className={styles.title}>{displayTitle}</h3>

      <div className={styles.metricsGrid}>
        <div className={styles.metricItem}>
          <span className={styles.metricLabel}>Total Runs</span>
          <span className={styles.metricValue}>{formatNumber(metrics.total_runs)}</span>
        </div>

        <div className={styles.metricItem}>
          <span className={styles.metricLabel}>Total Tokens</span>
          <span className={styles.metricValue}>{formatNumber(metrics.total_tokens)}</span>
        </div>

        <div className={styles.metricItem}>
          <span className={styles.metricLabel}>Total Cost</span>
          <span className={`${styles.metricValue} ${styles.cost}`}>
            {formatCost(metrics.total_cost)}
          </span>
        </div>

        <div className={styles.metricItem}>
          <span className={styles.metricLabel}>Total Duration</span>
          <span className={styles.metricValue}>{formatDuration(metrics.total_duration_ms)}</span>
        </div>

        <div className={styles.metricItem}>
          <span className={styles.metricLabel}>Files Modified</span>
          <span className={styles.metricValue}>{formatNumber(metrics.total_files_modified)}</span>
        </div>
      </div>

      {metrics.total_runs > 0 && (
        <div className={styles.averages}>
          <span className={styles.averagesLabel}>Averages per run:</span>
          <span className={styles.avgItem}>
            {formatNumber(Math.round(metrics.avg_tokens_per_run))} tokens
          </span>
          <span className={styles.avgItem}>
            {formatCost(metrics.avg_cost_per_run)}
          </span>
          <span className={styles.avgItem}>
            {formatDuration(Math.round(metrics.avg_duration_per_run))}
          </span>
        </div>
      )}
    </div>
  );
}
