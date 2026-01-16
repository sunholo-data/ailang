/**
 * UsageColumnChart - Bar chart showing aggregated usage over time
 *
 * Displays cost, tokens, turns, or spans bucketed by hour/day/week,
 * optionally split by provider, model, or workspace.
 */
import React, { useState, useMemo } from 'react';
import {
  BarChart,
  Bar,
  XAxis,
  YAxis,
  CartesianGrid,
  Tooltip,
  ResponsiveContainer,
} from 'recharts';
import type { UsageTimeSeriesPoint } from '../../hooks/useAnalytics';
import styles from './UsageColumnChart.module.css';

// Maximum number of legend items to show before collapsing
const MAX_LEGEND_ITEMS = 8;

// Type for dimension totals
interface DimensionTotal {
  name: string;
  total: number;
  color: string;
  taskCount?: number;
}

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
  // Track which legend item is hovered
  const [hoveredDimension, setHoveredDimension] = useState<string | null>(null);

  // Extract all unique dimensions for stacked bars
  // NOTE: Workspace normalization (Eval, Tasks, project names) is now done server-side
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

  // Calculate totals per dimension for sorting and display
  const dimensionTotals = useMemo((): DimensionTotal[] => {
    if (!splitBy || !dimensions.length) return [];

    const totals: Record<string, number> = {};
    dimensions.forEach((dim) => {
      totals[dim] = 0;
    });

    points.forEach((p) => {
      if (p.by_dimension) {
        Object.entries(p.by_dimension).forEach(([dim, value]) => {
          totals[dim] = (totals[dim] || 0) + value;
        });
      }
    });

    // Sort by total descending and assign colors
    return Object.entries(totals)
      .sort((a, b) => b[1] - a[1])
      .map(([name, total], index) => ({
        name,
        total,
        color: DIMENSION_COLORS[name.toLowerCase()] || DEFAULT_COLORS[index % DEFAULT_COLORS.length],
      }));
  }, [points, dimensions, splitBy]);

  // Transform data for Recharts
  const chartData = React.useMemo(() => {
    return points.map((p) => {
      const base: Record<string, any> = {
        bucket: p.bucket,
        bucketLabel: formatBucketLabel(p.bucket, interval),
      };

      if (splitBy && p.by_dimension) {
        // Use dimension values directly (already normalized server-side)
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

  // Truncate long dimension names for legend
  // NOTE: Workspace names are now normalized server-side (Eval, Tasks, project names)
  const truncateName = (name: string): string => {
    if (name.length > 20) {
      return name.substring(0, 17) + '...';
    }
    return name;
  };

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
        {payload.map((entry: any, index: number) => {
          // Display label - dataKey is already normalized server-side for workspaces
          const displayLabel = entry.dataKey === 'value' ? METRIC_LABELS[metric] : entry.dataKey;
          return (
            <div
              key={index}
              className={styles.tooltipRow}
              style={{ color: entry.color }}
            >
              <span className={styles.tooltipLabel}>{displayLabel}</span>
              <span className={styles.tooltipValue}>
                {formatMetricValue(entry.value, metric)}
              </span>
            </div>
          );
        })}
        {splitBy && payload.length > 1 && (
          <div className={styles.tooltipTotal}>
            <span>Total</span>
            <span>{formatMetricValue(total, metric)}</span>
          </div>
        )}
      </div>
    );
  };

  // Calculate grand total for percentages
  const grandTotal = useMemo(() => {
    return dimensionTotals.reduce((sum, d) => sum + d.total, 0);
  }, [dimensionTotals]);

  // Custom legend renderer with hover popups
  const renderCustomLegend = () => {
    if (!splitBy || dimensionTotals.length === 0) return null;

    const visibleItems = dimensionTotals.slice(0, MAX_LEGEND_ITEMS);
    const hiddenCount = dimensionTotals.length - MAX_LEGEND_ITEMS;
    const hiddenTotal = dimensionTotals
      .slice(MAX_LEGEND_ITEMS)
      .reduce((sum, d) => sum + d.total, 0);

    return (
      <div className={styles.legend}>
        {visibleItems.map((item) => {
          const percentage = grandTotal > 0 ? (item.total / grandTotal) * 100 : 0;
          return (
            <div
              key={item.name}
              className={styles.legendItem}
              onMouseEnter={() => setHoveredDimension(item.name)}
              onMouseLeave={() => setHoveredDimension(null)}
            >
              <span
                className={styles.legendColor}
                style={{ backgroundColor: item.color }}
              />
              <span className={styles.legendLabel} title={item.name}>
                {truncateName(item.name)}
              </span>
              {hoveredDimension === item.name && (
                <div className={styles.legendPopup}>
                  <div className={styles.legendPopupTitle}>{item.name}</div>
                  <div className={styles.legendPopupRow}>
                    <span>{METRIC_LABELS[metric]}:</span>
                    <span className={styles.legendPopupValue}>
                      {formatMetricValue(item.total, metric)}
                    </span>
                  </div>
                  <div className={styles.legendPopupRow}>
                    <span>Share:</span>
                    <span className={styles.legendPopupValue}>
                      {percentage.toFixed(1)}%
                    </span>
                  </div>
                  <div className={styles.legendPopupRow}>
                    <span>Type:</span>
                    <span className={styles.legendPopupValue}>
                      {splitBy}
                    </span>
                  </div>
                </div>
              )}
            </div>
          );
        })}
        {hiddenCount > 0 && (
          <span className={styles.legendMore} title={`${hiddenCount} more items totaling ${formatMetricValue(hiddenTotal, metric)}`}>
            +{hiddenCount} more
          </span>
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
      {renderCustomLegend()}
    </div>
  );
};

export default UsageColumnChart;
