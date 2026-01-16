/**
 * UsageColumnChart - Bar chart showing aggregated usage over time
 *
 * Displays cost, tokens, turns, or spans bucketed by hour/day/week,
 * optionally split by provider, model, or workspace.
 */
import React from 'react';
import {
  BarChart,
  Bar,
  XAxis,
  YAxis,
  CartesianGrid,
  Tooltip,
  Legend,
  ResponsiveContainer,
} from 'recharts';
import type { UsageTimeSeriesPoint } from '../../hooks/useAnalytics';
import styles from './UsageColumnChart.module.css';

export interface UsageColumnChartProps {
  points: UsageTimeSeriesPoint[];
  metric: 'cost' | 'tokens' | 'turns' | 'spans';
  splitBy?: string;
  interval: 'hour' | 'day' | 'week';
  height?: number;
  onBarClick?: (bucket: string) => void;
}

// Color palette for stacked bars
const DIMENSION_COLORS: Record<string, string> = {
  claude: '#8884d8',
  gemini: '#82ca9d',
  openai: '#ffc658',
  anthropic: '#8884d8',
  google: '#82ca9d',
  other: '#999999',
};

const DEFAULT_COLORS = ['#8884d8', '#82ca9d', '#ffc658', '#ff8042', '#00C49F'];

const METRIC_LABELS: Record<string, string> = {
  cost: 'Cost ($)',
  tokens: 'Tokens',
  turns: 'Turns',
  spans: 'Spans',
};

const formatMetricValue = (value: number, metric: string): string => {
  if (metric === 'cost') {
    return `$${value.toFixed(2)}`;
  }
  if (metric === 'tokens') {
    if (value > 1000000) return `${(value / 1000000).toFixed(1)}M`;
    if (value > 1000) return `${(value / 1000).toFixed(1)}K`;
  }
  return value.toLocaleString();
};

const formatBucketLabel = (bucket: string, interval: string): string => {
  if (interval === 'hour') {
    // Format: "2026-01-15 14:00" -> "14:00"
    const parts = bucket.split(' ');
    return parts[1] || bucket;
  }
  if (interval === 'week') {
    // Format: "2026-W03" -> "W03"
    return bucket.replace(/^\d{4}-/, '');
  }
  // Day format: "2026-01-15" -> "Jan 15"
  try {
    const date = new Date(bucket + 'T00:00:00');
    return date.toLocaleDateString('en-US', { month: 'short', day: 'numeric' });
  } catch {
    return bucket;
  }
};

export const UsageColumnChart: React.FC<UsageColumnChartProps> = ({
  points,
  metric,
  splitBy,
  interval,
  height = 300,
  onBarClick,
}) => {
  // Extract all unique dimensions for stacked bars
  const dimensions = React.useMemo(() => {
    if (!splitBy || !points.length) return [];
    const dims = new Set<string>();
    points.forEach((p) => {
      if (p.by_dimension) {
        Object.keys(p.by_dimension).forEach((d) => dims.add(d));
      }
    });
    return Array.from(dims);
  }, [points, splitBy]);

  // Transform data for Recharts
  const chartData = React.useMemo(() => {
    return points.map((p) => {
      const base: Record<string, any> = {
        bucket: p.bucket,
        bucketLabel: formatBucketLabel(p.bucket, interval),
      };

      if (splitBy && p.by_dimension) {
        // Use dimension values
        dimensions.forEach((dim) => {
          base[dim] = p.by_dimension?.[dim] || 0;
        });
      } else {
        // Use total value
        switch (metric) {
          case 'cost':
            base.value = p.cost;
            break;
          case 'tokens':
            base.value = p.tokens;
            break;
          case 'turns':
            base.value = p.turns;
            break;
          case 'spans':
            base.value = p.spans;
            break;
        }
      }

      return base;
    });
  }, [points, metric, splitBy, dimensions, interval]);

  // Custom tooltip
  const CustomTooltip = ({ active, payload, label }: any) => {
    if (!active || !payload || payload.length === 0) return null;

    const dataPoint = points.find(
      (p) => formatBucketLabel(p.bucket, interval) === label
    );
    const total = payload.reduce(
      (sum: number, entry: any) => sum + (entry.value || 0),
      0
    );

    return (
      <div className={styles.tooltip}>
        <div className={styles.tooltipHeader}>{dataPoint?.bucket || label}</div>
        {payload.map((entry: any, index: number) => (
          <div
            key={index}
            className={styles.tooltipRow}
            style={{ color: entry.color }}
          >
            <span className={styles.tooltipLabel}>
              {entry.dataKey === 'value' ? METRIC_LABELS[metric] : entry.dataKey}
            </span>
            <span className={styles.tooltipValue}>
              {formatMetricValue(entry.value, metric)}
            </span>
          </div>
        ))}
        {splitBy && payload.length > 1 && (
          <div className={styles.tooltipTotal}>
            <span>Total</span>
            <span>{formatMetricValue(total, metric)}</span>
          </div>
        )}
      </div>
    );
  };

  if (!points || points.length === 0) {
    return (
      <div className={styles.empty}>
        <span>No usage data available</span>
      </div>
    );
  }

  return (
    <div className={styles.container}>
      <ResponsiveContainer width="100%" height={height}>
        <BarChart
          data={chartData}
          margin={{ top: 5, right: 30, left: 20, bottom: 5 }}
          onClick={(e) => {
            if (e?.activePayload?.[0]?.payload?.bucket) {
              onBarClick?.(e.activePayload[0].payload.bucket);
            }
          }}
        >
          <CartesianGrid strokeDasharray="3 3" stroke="var(--border-subtle)" />
          <XAxis
            dataKey="bucketLabel"
            tick={{ fill: 'var(--text-secondary)', fontSize: 11 }}
            axisLine={{ stroke: 'var(--border-default)' }}
            interval="preserveStartEnd"
          />
          <YAxis
            label={{
              value: METRIC_LABELS[metric],
              angle: -90,
              position: 'insideLeft',
              style: { fill: 'var(--text-muted)' },
            }}
            tick={{ fill: 'var(--text-secondary)' }}
            axisLine={{ stroke: 'var(--border-default)' }}
            tickFormatter={(value) => formatMetricValue(value, metric)}
          />
          <Tooltip content={<CustomTooltip />} cursor={{ fill: 'var(--bg-hover)' }} />
          {splitBy && dimensions.length > 0 && <Legend />}
          {splitBy && dimensions.length > 0 ? (
            dimensions.map((dim, index) => (
              <Bar
                key={dim}
                dataKey={dim}
                stackId="stack"
                fill={DIMENSION_COLORS[dim.toLowerCase()] || DEFAULT_COLORS[index % DEFAULT_COLORS.length]}
                cursor="pointer"
              />
            ))
          ) : (
            <Bar dataKey="value" fill="#8884d8" cursor="pointer" />
          )}
        </BarChart>
      </ResponsiveContainer>
    </div>
  );
};

export default UsageColumnChart;
