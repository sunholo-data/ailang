import { useState, useEffect } from 'react';
import { MetricsTrendPoint } from '../../types';
import styles from './TrendsChart.module.css';

interface TrendsChartProps {
  scopeType: 'global' | 'agent' | 'thread';
  scopeId: string;
  period?: 'minute' | 'hour' | 'day';
  limit?: number;
  metric?: 'cost' | 'tokens' | 'duration' | 'runs';
  title?: string;
}

export function TrendsChart({
  scopeType,
  scopeId,
  period = 'hour',
  limit = 24,
  metric = 'cost',
  title,
}: TrendsChartProps) {
  const [data, setData] = useState<MetricsTrendPoint[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    const fetchTrends = async () => {
      try {
        setLoading(true);
        const response = await fetch(
          `/api/metrics/trends/${scopeType}/${scopeId}?period=${period}&limit=${limit}`
        );
        if (!response.ok) {
          throw new Error('Failed to fetch trends');
        }
        const trends = await response.json();
        setData(trends || []);
        setError(null);
      } catch (err) {
        setError(err instanceof Error ? err.message : 'Unknown error');
      } finally {
        setLoading(false);
      }
    };

    fetchTrends();
    const interval = setInterval(fetchTrends, 60000); // Refresh every minute
    return () => clearInterval(interval);
  }, [scopeType, scopeId, period, limit]);

  const getValue = (point: MetricsTrendPoint): number => {
    switch (metric) {
      case 'cost':
        return point.cost;
      case 'tokens':
        return point.tokens;
      case 'duration':
        return point.duration_ms / 1000; // Convert to seconds
      case 'runs':
        return point.runs;
      default:
        return point.cost;
    }
  };

  const formatValue = (value: number): string => {
    switch (metric) {
      case 'cost':
        return `$${value.toFixed(2)}`;
      case 'tokens':
        return value >= 1000 ? `${(value / 1000).toFixed(1)}k` : value.toString();
      case 'duration':
        return `${value.toFixed(1)}s`;
      case 'runs':
        return value.toString();
      default:
        return value.toFixed(2);
    }
  };

  const formatTime = (timestamp: number): string => {
    const date = new Date(timestamp);
    if (period === 'minute') {
      return date.toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' });
    } else if (period === 'hour') {
      return date.toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' });
    } else {
      return date.toLocaleDateString([], { month: 'short', day: 'numeric' });
    }
  };

  const getMetricLabel = (): string => {
    switch (metric) {
      case 'cost':
        return 'Cost ($)';
      case 'tokens':
        return 'Tokens';
      case 'duration':
        return 'Duration (s)';
      case 'runs':
        return 'Runs';
      default:
        return '';
    }
  };

  if (loading && data.length === 0) {
    return (
      <div className={styles.container}>
        <div className={styles.loading}>Loading trends...</div>
      </div>
    );
  }

  if (error) {
    return (
      <div className={styles.container}>
        <div className={styles.error}>{error}</div>
      </div>
    );
  }

  if (data.length === 0) {
    return (
      <div className={styles.container}>
        <div className={styles.empty}>No data available</div>
      </div>
    );
  }

  const values = data.map(getValue);
  const maxValue = Math.max(...values, 1);
  const total = values.reduce((sum, v) => sum + v, 0);

  return (
    <div className={styles.container}>
      {title && <div className={styles.title}>{title}</div>}
      <div className={styles.header}>
        <span className={styles.metricLabel}>{getMetricLabel()}</span>
        <span className={styles.total}>Total: {formatValue(total)}</span>
      </div>
      <div className={styles.chart}>
        {data.map((point, i) => {
          const value = getValue(point);
          const height = (value / maxValue) * 100;
          return (
            <div key={point.period_start} className={styles.barContainer}>
              <div className={styles.barWrapper}>
                <div
                  className={styles.bar}
                  style={{ height: `${Math.max(height, 2)}%` }}
                  title={`${formatTime(point.period_start)}: ${formatValue(value)}`}
                >
                  {height > 30 && (
                    <span className={styles.barValue}>{formatValue(value)}</span>
                  )}
                </div>
              </div>
              {i % Math.ceil(data.length / 6) === 0 && (
                <span className={styles.label}>{formatTime(point.period_start)}</span>
              )}
            </div>
          );
        })}
      </div>
    </div>
  );
}
