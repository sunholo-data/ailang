/**
 * TaskEvolutionChart - Line chart showing task metrics over time
 *
 * Displays cost, tokens, turns, or spans accumulation over the course
 * of task execution. Each task starts at 0 for easy comparison.
 */
import React, { useMemo, useState } from 'react';
import {
  LineChart,
  Line,
  XAxis,
  YAxis,
  CartesianGrid,
  Tooltip,
  ResponsiveContainer,
} from 'recharts';
import type { TaskEvolutionData } from '../../hooks/useAnalytics';
import styles from './TaskEvolutionChart.module.css';

export interface TaskEvolutionChartProps {
  tasks: TaskEvolutionData[];
  metric: 'cost' | 'tokens' | 'turns' | 'spans';
  height?: number;
  logScale?: boolean;
  onTaskClick?: (taskId: string) => void;
}

// Maximum legend items before collapsing
const MAX_LEGEND_ITEMS = 8;

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
  logScale = false,
  onTaskClick,
}) => {
  // Track which legend item is hovered
  const [hoveredTaskId, setHoveredTaskId] = useState<string | null>(null);

  // Calculate totals for each task (for sorting legend)
  const taskTotals = useMemo(() => {
    return tasks.map((task) => {
      const lastPoint = task.points?.[task.points.length - 1];
      let total = 0;
      if (lastPoint) {
        switch (metric) {
          case 'cost': total = lastPoint.cost; break;
          case 'tokens': total = lastPoint.tokens; break;
          case 'turns': total = lastPoint.turns; break;
          case 'spans': total = lastPoint.spans; break;
        }
      }
      return { task, total };
    }).sort((a, b) => b.total - a.total);
  }, [tasks, metric]);

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

  // Custom tooltip with task details
  const CustomTooltip = ({ active, payload, label }: any) => {
    if (!active || !payload || payload.length === 0) return null;

    return (
      <div className={styles.tooltip}>
        <div className={styles.tooltipHeader}>Turn {label}</div>
        {payload.map((entry: any, index: number) => {
          const task = tasks.find((t) => t.task_id === entry.dataKey);
          return (
            <div key={index}>
              <div
                className={styles.tooltipRow}
                style={{ color: entry.color }}
              >
                <span className={styles.tooltipLabel}>
                  {task?.title?.substring(0, 30) || entry.dataKey.substring(0, 8)}
                </span>
                <span className={styles.tooltipValue}>
                  {formatMetricValue(entry.value, metric)}
                </span>
              </div>
              {task && (
                <div className={styles.tooltipDetails}>
                  <div className={styles.tooltipDetail}>
                    <span>Provider:</span>
                    <span>{task.provider}</span>
                  </div>
                  <div className={styles.tooltipDetail}>
                    <span>Status:</span>
                    <span>{task.status}</span>
                  </div>
                </div>
              )}
            </div>
          );
        })}
      </div>
    );
  };

  // Truncate title for legend
  const truncateTitle = (title: string, maxLen: number = 18): string => {
    if (!title) return 'Untitled';
    if (title.length <= maxLen) return title;
    return title.substring(0, maxLen - 3) + '...';
  };

  // Custom legend renderer with hover popups
  const renderCustomLegend = () => {
    if (!tasks || tasks.length === 0) return null;

    const visibleTasks = taskTotals.slice(0, MAX_LEGEND_ITEMS);
    const hiddenCount = taskTotals.length - MAX_LEGEND_ITEMS;

    return (
      <div className={styles.legend}>
        {visibleTasks.map(({ task, total }, index) => {
          const color = COLORS[tasks.indexOf(task) % COLORS.length];
          const lastPoint = task.points?.[task.points.length - 1];

          return (
            <div
              key={task.task_id}
              className={styles.legendItem}
              onMouseEnter={() => setHoveredTaskId(task.task_id)}
              onMouseLeave={() => setHoveredTaskId(null)}
              onClick={() => onTaskClick?.(task.task_id)}
            >
              <span
                className={styles.legendColor}
                style={{ backgroundColor: color }}
              />
              <span className={styles.legendLabel}>
                {truncateTitle(task.title)}
              </span>
              {hoveredTaskId === task.task_id && (
                <div className={styles.legendPopup}>
                  <div className={styles.legendPopupTitle}>{task.title || 'Untitled Task'}</div>
                  <div className={styles.legendPopupMeta}>
                    <span className={`${styles.legendPopupTag} ${styles.provider}`}>
                      {task.provider}
                    </span>
                    <span className={`${styles.legendPopupTag} ${task.status === 'failed' ? styles.statusFailed : styles.status}`}>
                      {task.status}
                    </span>
                  </div>
                  <div className={styles.legendPopupRow}>
                    <span>Total {METRIC_LABELS[metric]}:</span>
                    <span className={styles.legendPopupValue}>
                      {formatMetricValue(total, metric)}
                    </span>
                  </div>
                  {lastPoint && (
                    <>
                      <div className={styles.legendPopupRow}>
                        <span>Total Cost:</span>
                        <span className={styles.legendPopupValue}>
                          ${lastPoint.cost.toFixed(2)}
                        </span>
                      </div>
                      <div className={styles.legendPopupRow}>
                        <span>Turns:</span>
                        <span className={styles.legendPopupValue}>
                          {lastPoint.turns?.toLocaleString() || 0}
                        </span>
                      </div>
                      <div className={styles.legendPopupRow}>
                        <span>Total Tokens:</span>
                        <span className={styles.legendPopupValue}>
                          {lastPoint.tokens.toLocaleString()}
                        </span>
                      </div>
                      <div className={styles.legendPopupRow}>
                        <span>Spans:</span>
                        <span className={styles.legendPopupValue}>
                          {lastPoint.spans?.toLocaleString() || task.points?.length || 0}
                        </span>
                      </div>
                    </>
                  )}
                  <div className={styles.legendPopupId}>
                    ID: {task.task_id}
                  </div>
                </div>
              )}
            </div>
          );
        })}
        {hiddenCount > 0 && (
          <span className={styles.legendMore}>
            +{hiddenCount} more
          </span>
        )}
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
            scale={logScale ? 'log' : 'auto'}
            domain={logScale ? ['auto', 'auto'] : [0, 'auto']}
            allowDataOverflow={logScale}
            label={{
              value: `${METRIC_LABELS[metric]}${logScale ? ' (log)' : ''}`,
              angle: -90,
              position: 'insideLeft',
              style: { fill: 'var(--text-muted)' },
            }}
            tick={{ fill: 'var(--text-secondary)' }}
            axisLine={{ stroke: 'var(--border-default)' }}
            tickFormatter={(value) => formatMetricValue(value, metric)}
          />
          <Tooltip content={<CustomTooltip />} />
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
      {renderCustomLegend()}
    </div>
  );
};

export default TaskEvolutionChart;
