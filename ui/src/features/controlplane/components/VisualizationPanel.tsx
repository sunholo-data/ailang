/**
 * VisualizationPanel - Container for dashboard analytics visualizations
 *
 * Manages multiple visualization types (heatmap, evolution chart, usage chart)
 * with tabs, filter indicator, CLI command display, and expand/collapse.
 */
import React, { useState, useCallback, useMemo, useEffect } from 'react';
import type { ControlPlaneFilters } from '../types';
import type { HeatmapCell, DateRange } from './types';
import type { HeatmapGridData } from '../hooks/useHeatmapData';
import { ActivityHeatmap } from './ActivityHeatmap';
import { FilterIndicator } from './FilterIndicator';
import { CliCommandHint, CommandType } from './CliCommandHint';
import styles from './VisualizationPanel.module.css';

export type ChartType = 'heatmap' | 'evolution' | 'usage' | 'breakdown';

export interface VisualizationPanelProps {
  /** Current active filters */
  filters: ControlPlaneFilters;
  /** Handler to clear a specific filter */
  onClearFilter: (key: keyof ControlPlaneFilters) => void;
  /** Handler to clear all filters */
  onClearAllFilters: () => void;

  // Heatmap props
  heatmapData: HeatmapCell[];
  heatmapGridData?: HeatmapGridData | null;
  selectedDateRange: DateRange | null;
  onDateSelect: (range: DateRange) => void;
  onHeatmapCellClick: (cell: HeatmapCell) => void;

  // Evolution chart props (will be passed when implemented)
  // evolutionData?: TaskEvolutionData[];

  // Usage chart props (will be passed when implemented)
  // usageData?: UsageTimeSeriesData;

  // Breakdown data (for donut chart)
  // breakdownData?: BreakdownData;
}

// Chart type configuration
const chartTabs: Array<{
  type: ChartType;
  label: string;
  icon: string;
  commandType: CommandType;
  available: boolean;
}> = [
  { type: 'heatmap', label: 'Activity', icon: '▤', commandType: 'stats', available: true },
  { type: 'evolution', label: 'Evolution', icon: '📈', commandType: 'traces', available: false },
  { type: 'usage', label: 'Usage', icon: '📊', commandType: 'stats', available: false },
  { type: 'breakdown', label: 'Breakdown', icon: '◐', commandType: 'stats', available: false },
];

export const VisualizationPanel: React.FC<VisualizationPanelProps> = ({
  filters,
  onClearFilter,
  onClearAllFilters,
  heatmapData,
  heatmapGridData,
  selectedDateRange,
  onDateSelect,
  onHeatmapCellClick,
}) => {
  const [activeChart, setActiveChart] = useState<ChartType>('heatmap');
  const [isExpanded, setIsExpanded] = useState(false);

  const handleChartChange = useCallback((chartType: ChartType) => {
    const tab = chartTabs.find(t => t.type === chartType);
    if (tab?.available) {
      setActiveChart(chartType);
    }
  }, []);

  const toggleExpand = useCallback(() => {
    setIsExpanded(prev => !prev);
  }, []);

  // Close on ESC key
  useEffect(() => {
    if (!isExpanded) return;

    const handleKeyDown = (e: KeyboardEvent) => {
      if (e.key === 'Escape') {
        setIsExpanded(false);
      }
    };

    document.addEventListener('keydown', handleKeyDown);
    return () => document.removeEventListener('keydown', handleKeyDown);
  }, [isExpanded]);

  // Get the current chart's command type
  const currentCommandType = useMemo(() => {
    return chartTabs.find(t => t.type === activeChart)?.commandType || 'stats';
  }, [activeChart]);

  // Check if there are any active filters to show
  const hasFilters = useMemo(() => {
    return Object.entries(filters).some(([key, value]) => {
      if (!value) return false;
      if (key === 'sort' || key === 'order') return false;
      if (key === 'status' && value === 'all') return false;
      return true;
    });
  }, [filters]);

  return (
    <>
      {/* Backdrop for expanded mode */}
      {isExpanded && (
        <div
          className={styles.backdrop}
          onClick={() => setIsExpanded(false)}
        />
      )}

      <div className={`${styles.container} ${isExpanded ? styles.expanded : ''}`}>
        {/* Header with tabs and controls */}
        <div className={styles.header}>
        <div className={styles.tabs}>
          {chartTabs.map((tab) => (
            <button
              key={tab.type}
              className={`${styles.tab} ${activeChart === tab.type ? styles.tabActive : ''} ${
                !tab.available ? styles.tabDisabled : ''
              }`}
              onClick={() => handleChartChange(tab.type)}
              disabled={!tab.available}
              title={tab.available ? tab.label : `${tab.label} (Coming Soon)`}
            >
              <span className={styles.tabIcon}>{tab.icon}</span>
              <span className={styles.tabLabel}>{tab.label}</span>
              {!tab.available && <span className={styles.tabBadge}>Soon</span>}
            </button>
          ))}
        </div>

        <div className={styles.controls}>
          <button
            className={styles.expandBtn}
            onClick={toggleExpand}
            title={isExpanded ? 'Collapse (Esc)' : 'Expand'}
            aria-label={isExpanded ? 'Collapse' : 'Expand'}
          >
            {isExpanded ? '×' : '⊞'}
          </button>
        </div>
      </div>

      {/* Filter indicator */}
      {hasFilters && (
        <div className={styles.filterRow}>
          <FilterIndicator
            filters={filters}
            onClearFilter={onClearFilter}
            onClearAll={onClearAllFilters}
            compact={!isExpanded}
          />
        </div>
      )}

      {/* Chart content */}
      <div className={styles.content}>
        {activeChart === 'heatmap' && (
          <ActivityHeatmap
            data={heatmapData}
            gridData={heatmapGridData}
            selectedRange={selectedDateRange}
            onDateSelect={onDateSelect}
            onCellClick={onHeatmapCellClick}
          />
        )}

        {activeChart === 'evolution' && (
          <div className={styles.placeholder}>
            <span className={styles.placeholderIcon}>📈</span>
            <span className={styles.placeholderText}>Task Evolution Chart</span>
            <span className={styles.placeholderSub}>Coming in Phase 2</span>
          </div>
        )}

        {activeChart === 'usage' && (
          <div className={styles.placeholder}>
            <span className={styles.placeholderIcon}>📊</span>
            <span className={styles.placeholderText}>Usage Over Time</span>
            <span className={styles.placeholderSub}>Coming in Phase 3</span>
          </div>
        )}

        {activeChart === 'breakdown' && (
          <div className={styles.placeholder}>
            <span className={styles.placeholderIcon}>◐</span>
            <span className={styles.placeholderText}>Cost Breakdown</span>
            <span className={styles.placeholderSub}>Coming in Phase 4</span>
          </div>
        )}
      </div>

      {/* CLI Command hint */}
      <div className={styles.cliRow}>
        <CliCommandHint
          commandType={currentCommandType}
          filters={filters}
          compact={!isExpanded}
        />
      </div>
      </div>
    </>
  );
};

export default VisualizationPanel;
