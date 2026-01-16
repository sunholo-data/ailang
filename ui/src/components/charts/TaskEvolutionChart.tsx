/**
 * TaskEvolutionChart - Line chart showing task metrics over time
 *
 * Displays cost, tokens, turns, or spans accumulation over the course
 * of task execution. Each task starts at 0 for easy comparison.
 */
import React, { useMemo } from 'react';
import {
  LineChart,
  Line,
  XAxis,
  YAxis,
  CartesianGrid,
  Tooltip,
  Legend,
  ResponsiveContainer,
} from 'recharts';
import type { TaskEvolutionData } from '../../hooks/useAnalytics';
import styles from './TaskEvolutionChart.module.css';

export interface TaskEvolutionChartProps {
  tasks: TaskEvolutionData[];
  metric: 'cost' | 'tokens' | 'turns' | 'spans';
  height?: number;
  onTaskClick?: (taskId: string) => void;
}

// Color palette for multiple task lines
const COLORS = [
  '#8884d8', // purple
  '#82ca9d', // green
  '#ffc658', // yellow
  '#ff8042', // orange
  '#00C49F', // teal
  '#FFBB28', // gold
  '#FF8042', // coral
  '#0088FE', // blue
  '#00C49F', // mint
  '#FF00FF', // magenta
];

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
  if (metric === 'tokens' && value > 1000) {
    return `${(value / 1000).toFixed(1)}K`;
  }
  return value.toLocaleString();
};

export const TaskEvolutionChart: React.FC<TaskEvolutionChartProps> = ({
  tasks,
  metric,
  height = 300,
  onTaskClick,
}) => {
  // Transform data for Recharts
  // We need to align all tasks on a common X axis (turn number)
  const chartData = useMemo(() => {
    if (!tasks || tasks.length === 0) return [];

    // Find max turns across all tasks
    const maxTurns = Math.max(...tasks.map((t) => t.points?.length || 0));

    // Create data points for each turn index
    const data: Array<Record<string, number | string>> = [];

    for (let i = 0; i < maxTurns; i++) {
      const point: Record<string, number | string> = { x: i };

      tasks.forEach((task) => {
        if (task.points && task.points[i]) {
          const p = task.points[i];
          switch (metric) {
            case 'cost':
              point[task.task_id] = p.cost;
              break;
            case 'tokens':
              point[task.task_id] = p.tokens;
              break;
            case 'turns':
              point[task.task_id] = p.turns;
              break;
            case 'spans':
              point[task.task_id] = p.spans;
              break;
          }
        }
      });

      data.push(point);
    }

    return data;
  }, [tasks, metric]);

  // Custom tooltip
  const CustomTooltip = ({ active, payload, label }: any) => {
    if (!active || !payload || payload.length === 0) return null;

    return (
      <div className={styles.tooltip}>
        <div className={styles.tooltipHeader}>Turn {label}</div>
        {payload.map((entry: any, index: number) => {
          const task = tasks.find((t) => t.task_id === entry.dataKey);
          return (
            <div
              key={index}
              className={styles.tooltipRow}
              style={{ color: entry.color }}
            >
              <span className={styles.tooltipLabel}>
                {task?.title?.substring(0, 20) || entry.dataKey}
              </span>
              <span className={styles.tooltipValue}>
                {formatMetricValue(entry.value, metric)}
              </span>
            </div>
          );
        })}
      </div>
    );
  };

  if (!tasks || tasks.length === 0) {
    return (
      <div className={styles.empty}>
        <span>No task evolution data available</span>
      </div>
    );
  }

  return (
    <div className={styles.container}>
      <ResponsiveContainer width="100%" height={height}>
        <LineChart
          data={chartData}
          margin={{ top: 5, right: 30, left: 20, bottom: 5 }}
        >
          <CartesianGrid strokeDasharray="3 3" stroke="var(--border-subtle)" />
          <XAxis
            dataKey="x"
            label={{
              value: 'Turn',
              position: 'insideBottomRight',
              offset: -5,
              style: { fill: 'var(--text-muted)' },
            }}
            tick={{ fill: 'var(--text-secondary)' }}
            axisLine={{ stroke: 'var(--border-default)' }}
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
          <Tooltip content={<CustomTooltip />} />
          <Legend
            formatter={(value) => {
              const task = tasks.find((t) => t.task_id === value);
              const label = task?.title || value;
              return label.length > 25 ? label.substring(0, 22) + '...' : label;
            }}
            onClick={(e) => onTaskClick?.(String(e.dataKey))}
            wrapperStyle={{ cursor: 'pointer' }}
          />
          {tasks.map((task, index) => (
            <Line
              key={task.task_id}
              type="monotone"
              dataKey={task.task_id}
              stroke={COLORS[index % COLORS.length]}
              strokeWidth={2}
              dot={false}
              activeDot={{ r: 6, cursor: 'pointer' }}
              name={task.task_id}
            />
          ))}
        </LineChart>
      </ResponsiveContainer>
    </div>
  );
};

export default TaskEvolutionChart;
