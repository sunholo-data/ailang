/**
 * ActivityHeatmap - GitHub-style activity heatmap for Control Plane
 *
 * Supports two modes:
 * 1. Grid mode (preferred): Uses pre-computed weeks/months from API
 * 2. Flat mode (legacy): Computes grid client-side from flat cell array
 */
import React, { useState, useMemo, useEffect, useRef } from 'react';
import type { HeatmapCell, DateRange } from './types';
import type { HeatmapGridData, HeatmapGridCell } from '../hooks/useHeatmapData';
import styles from '../ControlPlane.module.css';

type HeatmapRange = '3m' | '6m' | '1y';

export interface ActivityHeatmapProps {
  /** Flat cell array (legacy mode) */
  data: HeatmapCell[];
  /** Pre-computed grid data from API (preferred) */
  gridData?: HeatmapGridData | null;
  selectedRange: DateRange | null;
  onDateSelect: (range: DateRange) => void;
  onCellClick: (cell: HeatmapCell) => void;
  /** Whether the heatmap is in expanded mode (larger cells) */
  isExpanded?: boolean;
}

// Format duration for display
const formatDuration = (ms: number): string => {
  if (ms < 1000) return `${ms}ms`;
  if (ms < 60000) return `${(ms / 1000).toFixed(1)}s`;
  return `${Math.floor(ms / 60000)}m ${Math.floor((ms % 60000) / 1000)}s`;
};

export const ActivityHeatmap: React.FC<ActivityHeatmapProps> = ({
  data,
  gridData,
  selectedRange,
  onDateSelect,
  onCellClick,
  isExpanded = false,
}) => {
  // Grid sizing based on expanded mode
  const cellSize = isExpanded ? 20 : 12;
  const columnWidth = isExpanded ? 22 : 14;
  const gap = 2;
  const [hoveredCell, setHoveredCell] = useState<HeatmapCell | null>(null);
  const [tooltipPos, setTooltipPos] = useState({ x: 0, y: 0 });
  const [selectionStart, setSelectionStart] = useState<string | null>(null);
  const [isDragging, setIsDragging] = useState(false);
  const [displayRange, setDisplayRange] = useState<HeatmapRange>('3m');
  const gridRef = useRef<HTMLDivElement>(null);

  // Filter data based on display range
  const filteredData = useMemo(() => {
    const now = new Date();
    let daysToShow: number;
    switch (displayRange) {
      case '3m': daysToShow = 90; break;
      case '6m': daysToShow = 180; break;
      case '1y': daysToShow = 365; break;
    }
    const cutoffDate = new Date(now.getTime() - daysToShow * 24 * 60 * 60 * 1000);
    const cutoffStr = cutoffDate.toISOString().split('T')[0];
    return data.filter(cell => cell.date >= cutoffStr);
  }, [data, displayRange]);

  // Calculate max task count for relative intensity scaling
  const maxCount = useMemo(() => {
    const counts = filteredData.map(c => c.taskCount).filter(c => c > 0);
    return counts.length > 0 ? Math.max(...counts) : 1;
  }, [filteredData]);

  // Get intensity level (0-4) for a cell
  // Uses pre-computed intensity from API when available (grid mode)
  const getIntensity = (cell: HeatmapCell & { _intensity?: number }): number => {
    // Use pre-computed intensity from grid mode (0-1 range → 0-4 levels)
    if (typeof cell._intensity === 'number') {
      if (cell._intensity === 0) return 0;
      if (cell._intensity <= 0.25) return 1;
      if (cell._intensity <= 0.50) return 2;
      if (cell._intensity <= 0.75) return 3;
      return 4;
    }
    // Legacy fallback: compute from count
    if (cell.taskCount === 0) return 0;
    const ratio = cell.taskCount / maxCount;
    if (ratio <= 0.25) return 1;  // 0-25% of max
    if (ratio <= 0.50) return 2;  // 25-50% of max
    if (ratio <= 0.75) return 3;  // 50-75% of max
    return 4;                      // 75-100% of max
  };

  // Use server-computed grid data if available, otherwise compute client-side
  const weeks = useMemo(() => {
    // Grid mode: Use pre-computed weeks from API (removes ~50 lines of client-side calculation)
    if (gridData?.weeks) {
      // Convert HeatmapGridCell to HeatmapCell for compatibility with tooltips/selection
      return gridData.weeks.map(week =>
        week.map(cell => ({
          date: cell.date,
          taskCount: cell.count,
          cost: cell.cost,
          successRate: cell.successRate,
          // Store pre-computed intensity for getIntensity optimization
          _intensity: cell.intensity,
        } as HeatmapCell & { _intensity?: number }))
      );
    }

    // Legacy mode: Compute grid client-side from flat data
    const result: HeatmapCell[][] = [];
    let currentWeek: HeatmapCell[] = [];

    // Determine days to show based on display range
    let daysToShow: number;
    switch (displayRange) {
      case '3m': daysToShow = 90; break;
      case '6m': daysToShow = 180; break;
      case '1y': daysToShow = 365; break;
    }

    // Build lookup map from existing data for O(1) access
    const dataMap = new Map(filteredData.map(c => [c.date, c]));

    // Generate ALL dates in range (oldest to newest)
    const now = new Date();
    const allDates: HeatmapCell[] = [];
    for (let i = daysToShow - 1; i >= 0; i--) {
      const date = new Date(now.getTime() - i * 24 * 60 * 60 * 1000);
      const dateStr = date.toISOString().split('T')[0];
      const existing = dataMap.get(dateStr);
      allDates.push(existing || { date: dateStr, taskCount: 0, cost: 0, successRate: 0 });
    }

    if (allDates.length === 0) return result;

    // Pad start to align with day of week (Monday = 0)
    const firstDate = new Date(allDates[0].date);
    const dayOfWeek = firstDate.getDay();
    const startPadding = dayOfWeek === 0 ? 6 : dayOfWeek - 1;
    for (let i = 0; i < startPadding; i++) {
      currentWeek.push({ date: '', taskCount: -1, cost: 0, successRate: 0 });
    }

    allDates.forEach((cell) => {
      currentWeek.push(cell);
      if (currentWeek.length === 7) {
        result.push(currentWeek);
        currentWeek = [];
      }
    });
    if (currentWeek.length > 0) {
      while (currentWeek.length < 7) {
        currentWeek.push({ date: '', taskCount: -1, cost: 0, successRate: 0 });
      }
      result.push(currentWeek);
    }
    return result;
  }, [gridData, filteredData, displayRange]);

  // Get month labels with their positions
  // Use server-computed labels if available (removes ~30 lines of client-side calculation)
  const monthLabels = useMemo(() => {
    // Grid mode: Use pre-computed month labels from API
    if (gridData?.monthLabels) {
      return gridData.monthLabels.map(m => ({
        label: m.name,
        weekIdx: m.weekIndex,
        span: 1, // API doesn't provide span, but CSS handles it
      }));
    }

    // Legacy mode: Compute month labels client-side
    const labels: { label: string; weekIdx: number; span: number }[] = [];
    let currentMonth = -1;
    let currentLabel: { label: string; weekIdx: number; span: number } | null = null;

    weeks.forEach((week, weekIdx) => {
      const validCell = week.find(c => c.date);
      if (validCell) {
        const date = new Date(validCell.date);
        const month = date.getMonth();
        if (month !== currentMonth) {
          if (currentLabel) {
            labels.push(currentLabel);
          }
          currentLabel = {
            label: date.toLocaleDateString('en-US', { month: 'short' }),
            weekIdx,
            span: 1,
          };
          currentMonth = month;
        } else if (currentLabel) {
          currentLabel.span++;
        }
      }
    });
    if (currentLabel) {
      labels.push(currentLabel);
    }
    return labels;
  }, [gridData, weeks]);

  const handleCellHover = (cell: HeatmapCell, e: React.MouseEvent) => {
    if (cell.taskCount >= 0) {
      setHoveredCell(cell);
      setTooltipPos({ x: e.clientX, y: e.clientY });

      if (isDragging && selectionStart) {
        const startIdx = filteredData.findIndex(c => c.date === selectionStart);
        const endIdx = filteredData.findIndex(c => c.date === cell.date);
        const [start, end] = startIdx < endIdx
          ? [selectionStart, cell.date]
          : [cell.date, selectionStart];
        onDateSelect({ start, end });
      }
    }
  };

  const handleCellMouseDown = (cell: HeatmapCell) => {
    if (cell.taskCount < 0) return;
    setIsDragging(true);
    setSelectionStart(cell.date);
    onDateSelect({ start: cell.date, end: cell.date });
  };

  const handleCellMouseUp = (cell: HeatmapCell) => {
    if (cell.taskCount >= 0) {
      setIsDragging(false);
      onCellClick(cell);
    }
  };

  const handleMouseUp = () => {
    setIsDragging(false);
  };

  useEffect(() => {
    window.addEventListener('mouseup', handleMouseUp);
    return () => window.removeEventListener('mouseup', handleMouseUp);
  }, []);

  // Scroll to end (latest dates) on mount and when range changes
  useEffect(() => {
    if (gridRef.current) {
      gridRef.current.scrollLeft = gridRef.current.scrollWidth;
    }
  }, [displayRange, weeks.length]);

  const isInRange = (date: string): boolean => {
    if (!selectedRange || !selectedRange.start || !selectedRange.end) return false;
    return date >= selectedRange.start && date <= selectedRange.end;
  };

  const rangeStats = useMemo(() => {
    if (!selectedRange || !selectedRange.start) return null;
    const cells = filteredData.filter(c => c.date >= selectedRange.start && c.date <= selectedRange.end);
    if (cells.length === 0) return null;

    const totalTasks = cells.reduce((sum, c) => sum + c.taskCount, 0);
    const totalCost = cells.reduce((sum, c) => sum + c.cost, 0);
    const avgSuccess = cells.reduce((sum, c) => sum + c.successRate, 0) / cells.length;

    return { totalTasks, totalCost, avgSuccess, days: cells.length };
  }, [filteredData, selectedRange]);

  // Calculate totals for display (use server totals when available)
  const totals = useMemo(() => {
    if (gridData?.totals) {
      return { totalTasks: gridData.totals.tasks, totalCost: gridData.totals.cost };
    }
    const totalTasks = filteredData.reduce((sum, c) => sum + c.taskCount, 0);
    const totalCost = filteredData.reduce((sum, c) => sum + c.cost, 0);
    return { totalTasks, totalCost };
  }, [gridData, filteredData]);

  return (
    <div className={`${styles.heatmapContainer} ${isExpanded ? styles.heatmapExpanded : ''}`}>
      <div className={styles.heatmapHeader}>
        <h3 className={styles.panelTitle}>
          <span className={styles.panelIcon}>▤</span>
          Activity
        </h3>
        <div className={styles.heatmapTotals}>
          <span className={styles.heatmapTotal}>{totals.totalTasks} tasks</span>
          <span className={styles.heatmapTotal}>${totals.totalCost.toFixed(2)}</span>
        </div>
        <div className={styles.heatmapRangeSelector}>
          <button
            className={`${styles.rangeBtn} ${displayRange === '3m' ? styles.rangeBtnActive : ''}`}
            onClick={() => setDisplayRange('3m')}
          >
            3M
          </button>
          <button
            className={`${styles.rangeBtn} ${displayRange === '6m' ? styles.rangeBtnActive : ''}`}
            onClick={() => setDisplayRange('6m')}
          >
            6M
          </button>
          <button
            className={`${styles.rangeBtn} ${displayRange === '1y' ? styles.rangeBtnActive : ''}`}
            onClick={() => setDisplayRange('1y')}
          >
            1Y
          </button>
        </div>
        <div className={styles.heatmapLegend}>
          <span className={styles.legendLabel}>Less</span>
          <span className={`${styles.legendCell} ${styles.intensity0}`} />
          <span className={`${styles.legendCell} ${styles.intensity1}`} />
          <span className={`${styles.legendCell} ${styles.intensity2}`} />
          <span className={`${styles.legendCell} ${styles.intensity3}`} />
          <span className={`${styles.legendCell} ${styles.intensity4}`} />
          <span className={styles.legendLabel}>More</span>
        </div>
      </div>

      <div className={styles.heatmapBody}>
        <div className={styles.heatmapDayLabels}>
          <span></span>
          <span>Mon</span>
          <span></span>
          <span>Wed</span>
          <span></span>
          <span>Fri</span>
          <span></span>
        </div>

        <div className={styles.heatmapScrollArea} ref={gridRef}>
          <div className={styles.heatmapMonthRow}>
            {monthLabels.map((m, i) => (
              <span
                key={i}
                className={styles.monthLabel}
                style={{
                  gridColumn: `${m.weekIdx + 1} / span ${m.span}`,
                }}
              >
                {m.label}
              </span>
            ))}
          </div>

          <div className={styles.heatmapGrid} style={{ gridTemplateColumns: `repeat(${weeks.length}, ${columnWidth}px)` }}>
            {weeks.map((week, weekIdx) => (
              <div key={weekIdx} className={styles.heatmapWeek}>
                {week.map((cell, dayIdx) => (
                  <div
                    key={dayIdx}
                    className={`${styles.heatmapCell} ${
                      cell.taskCount < 0 ? styles.cellEmpty : styles[`intensity${getIntensity(cell)}`]
                    } ${isInRange(cell.date) ? styles.cellSelected : ''}`}
                    onMouseEnter={(e) => handleCellHover(cell, e)}
                    onMouseLeave={() => !isDragging && setHoveredCell(null)}
                    onMouseDown={() => handleCellMouseDown(cell)}
                    onMouseUp={() => handleCellMouseUp(cell)}
                  />
                ))}
              </div>
            ))}
          </div>
        </div>
      </div>

      {selectedRange && selectedRange.start && rangeStats && (
        <div className={styles.heatmapSelection}>
          <div className={styles.selectionInfo}>
            <span className={styles.selectionDates}>
              {selectedRange.start === selectedRange.end
                ? selectedRange.start
                : `${selectedRange.start} → ${selectedRange.end}`}
            </span>
            <span className={styles.selectionDays}>({rangeStats.days} days)</span>
          </div>
          <div className={styles.selectionStats}>
            <span className={styles.selectionStat}>
              <strong>{rangeStats.totalTasks}</strong> tasks
            </span>
            <span className={styles.selectionStat}>
              <strong>${rangeStats.totalCost.toFixed(2)}</strong> cost
            </span>
            <span className={styles.selectionStat}>
              <strong>{(rangeStats.avgSuccess * 100).toFixed(0)}%</strong> success
            </span>
          </div>
          <button
            className={styles.selectionClear}
            onClick={() => onDateSelect({ start: '', end: '' })}
          >
            ✕
          </button>
        </div>
      )}

      {hoveredCell && (
        <div
          className={styles.heatmapTooltip}
          style={{ left: tooltipPos.x + 10, top: tooltipPos.y - 80 }}
        >
          <div className={styles.tooltipDate}>{hoveredCell.date}</div>
          <div className={styles.tooltipStat}>
            <span>{hoveredCell.taskCount} tasks</span>
            <span>${hoveredCell.cost.toFixed(3)}</span>
          </div>
          <div className={styles.tooltipSuccess}>
            {(hoveredCell.successRate * 100).toFixed(0)}% success
          </div>
          <div className={styles.tooltipHint}>Click & drag to select range</div>
        </div>
      )}
    </div>
  );
};

export default ActivityHeatmap;
