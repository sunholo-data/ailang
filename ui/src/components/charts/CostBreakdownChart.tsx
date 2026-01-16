/**
 * CostBreakdownChart - Donut chart showing cost breakdown by dimension
 *
 * Displays cost distribution by provider, model, workspace, or source type
 * using a Recharts PieChart with a donut hole and interactive legend.
 */
import React, { useMemo } from 'react';
import {
  PieChart,
  Pie,
  Cell,
  ResponsiveContainer,
  Tooltip,
  Legend,
} from 'recharts';
import type { BreakdownItem } from '../../features/controlplane/hooks/useBreakdownData';
import styles from './CostBreakdownChart.module.css';

export type BreakdownDimension = 'provider' | 'model' | 'workspace' | 'source_type';

export interface CostBreakdownChartProps {
  items: BreakdownItem[];
  dimension: BreakdownDimension;
  totalCost: number;
  height?: number;
  onSegmentClick?: (item: BreakdownItem) => void;
}

// Color palette for pie segments
const COLORS = [
  '#8884d8', // purple
  '#82ca9d', // green
  '#ffc658', // yellow
  '#ff8042', // orange
  '#00C49F', // teal
  '#FFBB28', // gold
  '#0088FE', // blue
  '#FF00FF', // magenta
];

const DIMENSION_LABELS: Record<BreakdownDimension, string> = {
  provider: 'Provider',
  model: 'Model',
  workspace: 'Workspace',
  source_type: 'Source',
};

const formatCost = (cost: number): string => {
  if (cost >= 1000) return `$${(cost / 1000).toFixed(1)}K`;
  if (cost >= 1) return `$${cost.toFixed(2)}`;
  if (cost >= 0.01) return `$${cost.toFixed(2)}`;
  return `$${cost.toFixed(4)}`;
};

const formatPercentage = (pct: number): string => {
  if (pct >= 10) return `${pct.toFixed(0)}%`;
  if (pct >= 1) return `${pct.toFixed(1)}%`;
  return `${pct.toFixed(2)}%`;
};

export const CostBreakdownChart: React.FC<CostBreakdownChartProps> = ({
  items,
  dimension,
  totalCost,
  height = 300,
  onSegmentClick,
}) => {
  // Transform items for Recharts
  const chartData = useMemo(() => {
    return items.map((item, index) => ({
      ...item,
      name: item.label,
      value: item.cost_usd,
      color: COLORS[index % COLORS.length],
    }));
  }, [items]);

  // Custom tooltip
  const CustomTooltip = ({ active, payload }: any) => {
    if (!active || !payload || payload.length === 0) return null;

    const data = payload[0].payload;
    const percentage = totalCost > 0 ? (data.cost_usd / totalCost) * 100 : 0;

    return (
      <div className={styles.tooltip}>
        <div className={styles.tooltipHeader}>{data.label}</div>
        <div className={styles.tooltipRow}>
          <span className={styles.tooltipLabel}>Cost:</span>
          <span className={styles.tooltipValue}>{formatCost(data.cost_usd)}</span>
        </div>
        <div className={styles.tooltipRow}>
          <span className={styles.tooltipLabel}>Percentage:</span>
          <span className={styles.tooltipValue}>{formatPercentage(percentage)}</span>
        </div>
        <div className={styles.tooltipRow}>
          <span className={styles.tooltipLabel}>Tasks:</span>
          <span className={styles.tooltipValue}>{data.task_count || 0}</span>
        </div>
        <div className={styles.tooltipRow}>
          <span className={styles.tooltipLabel}>Tokens:</span>
          <span className={styles.tooltipValue}>
            {(data.tokens_in + data.tokens_out).toLocaleString()}
          </span>
        </div>
      </div>
    );
  };

  // Custom legend renderer
  const renderLegend = (props: any) => {
    const { payload } = props;
    return (
      <div className={styles.legend}>
        {payload.map((entry: any, index: number) => {
          const item = chartData[index];
          const percentage = totalCost > 0 ? (item.cost_usd / totalCost) * 100 : 0;
          return (
            <div
              key={`legend-${index}`}
              className={styles.legendItem}
              onClick={() => onSegmentClick?.(item)}
            >
              <span
                className={styles.legendColor}
                style={{ backgroundColor: entry.color }}
              />
              <span className={styles.legendLabel}>{entry.value}</span>
              <span className={styles.legendValue}>{formatCost(item.cost_usd)}</span>
              <span className={styles.legendPct}>{formatPercentage(percentage)}</span>
            </div>
          );
        })}
      </div>
    );
  };

  if (!items || items.length === 0) {
    return (
      <div className={styles.empty}>
        <span>No breakdown data available</span>
      </div>
    );
  }

  return (
    <div className={styles.container}>
      <div className={styles.header}>
        <span className={styles.title}>Cost by {DIMENSION_LABELS[dimension]}</span>
        <span className={styles.total}>Total: {formatCost(totalCost)}</span>
      </div>
      <ResponsiveContainer width="100%" height={height}>
        <PieChart>
          <Pie
            data={chartData}
            cx="50%"
            cy="50%"
            innerRadius="50%"
            outerRadius="80%"
            paddingAngle={2}
            dataKey="value"
            onClick={(data) => onSegmentClick?.(data)}
            cursor="pointer"
          >
            {chartData.map((entry, index) => (
              <Cell
                key={`cell-${index}`}
                fill={entry.color}
                stroke="var(--bg-base)"
                strokeWidth={2}
              />
            ))}
          </Pie>
          <Tooltip content={<CustomTooltip />} />
          <Legend content={renderLegend} />
        </PieChart>
      </ResponsiveContainer>
    </div>
  );
};

export default CostBreakdownChart;
