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
import { ActivityHeatmap, HeatmapMetric } from './ActivityHeatmap';
import { FilterIndicator } from './FilterIndicator';
import { CliCommandHint, CommandType } from './CliCommandHint';
import { TaskEvolutionChart } from '../../../components/charts/TaskEvolutionChart';
import { UsageColumnChart } from '../../../components/charts/UsageColumnChart';
import { CostBreakdownChart, BreakdownDimension } from '../../../components/charts/CostBreakdownChart';
import { useTaskEvolution, useUsageTimeSeries } from '../../../hooks/useAnalytics';
import { useBreakdownData } from '../hooks/useBreakdownData';
import styles from './VisualizationPanel.module.css';

export type ChartType = 'heatmap' | 'evolution' | 'usage' | 'breakdown';

export interface VisualizationPanelProps {
  /** Current active filters */
  filters: ControlPlaneFilters;
  /** Handler to clear a specific filter */
  onClearFilter: (key: keyof ControlPlaneFilters) => void;
  /** Handler to clear all filters */
  onClearAllFilters: () => void;
  /** Handler to set/add a filter (for interactive exploration) */
  onSetFilter?: (dimension: string, value: string) => void;
  /** Handler when a task is selected in evolution chart */
  onTaskSelect?: (taskId: string) => void;

  // Heatmap props
  heatmapData: HeatmapCell[];
  heatmapGridData?: HeatmapGridData | null;
  selectedDateRange: DateRange | null;
  onDateSelect: (range: DateRange) => void;
  onHeatmapCellClick: (cell: HeatmapCell) => void;
}

// Metric type for charts
export type MetricType = 'cost' | 'tokens' | 'turns' | 'spans' | 'duration';
export type IntervalType = 'hour' | 'day' | 'week';
export type SplitByType = 'provider' | 'model' | 'workspace' | '';

// Chart tab type
interface ChartTab {
  type: ChartType;
  label: string;
  icon: string;
  commandType: CommandType;
  available: boolean;
}

// Static chart tabs (always visible)
const staticChartTabs: ChartTab[] = [
  { type: 'heatmap', label: 'Activity', icon: '▤', commandType: 'stats', available: true },
  { type: 'evolution', label: 'Evolution', icon: '📈', commandType: 'traces', available: true },
  { type: 'usage', label: 'Usage', icon: '📊', commandType: 'stats', available: true },
  { type: 'breakdown', label: 'Breakdown', icon: '◐', commandType: 'stats', available: true },
];

export const VisualizationPanel: React.FC<VisualizationPanelProps> = ({
  filters,
  onClearFilter,
  onClearAllFilters,
  onSetFilter,
  onTaskSelect,
  heatmapData,
  heatmapGridData,
  selectedDateRange,
  onDateSelect,
  onHeatmapCellClick,
}) => {
  const [activeChart, setActiveChart] = useState<ChartType>('heatmap');
  const [isExpanded, setIsExpanded] = useState(false);
  const [showFiltersPanel, setShowFiltersPanel] = useState(false);

  // Chart configuration state
  const [metric, setMetric] = useState<MetricType>('cost');
  const [interval, setInterval] = useState<IntervalType>('day');
  const [splitBy, setSplitBy] = useState<SplitByType>('');
  const [breakdownDimension, setBreakdownDimension] = useState<BreakdownDimension>('provider');
  const [logScale, setLogScale] = useState(false);

  // Fetch evolution data when on evolution tab
  const { data: evolutionData, loading: evolutionLoading } = useTaskEvolution(
    filters,
    metric,
    10 // limit to 10 tasks
  );

  // Fetch usage data when on usage tab
  const { data: usageData, loading: usageLoading } = useUsageTimeSeries(
    filters,
    metric,
    interval,
    splitBy || undefined
  );

  // Fetch breakdown data
  const { data: breakdownData, loading: breakdownLoading } = useBreakdownData({
    filters,
    refreshInterval: 30000,
  });

  // Handle task click from evolution chart - just trigger selection, don't switch tabs
  const handleTaskClick = useCallback((taskId: string) => {
    onTaskSelect?.(taskId);
  }, [onTaskSelect]);

  const handleChartChange = useCallback((chartType: ChartType) => {
    const tab = staticChartTabs.find(t => t.type === chartType);
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
    return staticChartTabs.find(t => t.type === activeChart)?.commandType || 'stats';
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
            {staticChartTabs.map((tab) => (
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

          {/* Global metric selector */}
          <div className={styles.globalMetric}>
            <select
              value={metric}
              onChange={(e) => setMetric(e.target.value as MetricType)}
              className={styles.metricSelect}
              title="Select metric for all visualizations"
            >
              <option value="cost">Cost</option>
              <option value="tokens">Tokens</option>
              <option value="turns">Tasks</option>
              <option value="spans">Spans</option>
              <option value="duration">Duration</option>
            </select>
          </div>

          <div className={styles.controls}>
            {isExpanded && onSetFilter && (
              <button
                className={`${styles.expandBtn} ${showFiltersPanel ? styles.filtersBtnActive : ''}`}
                onClick={() => setShowFiltersPanel(prev => !prev)}
                title={showFiltersPanel ? 'Hide Filters' : 'Show Filters'}
                aria-label={showFiltersPanel ? 'Hide Filters' : 'Show Filters'}
              >
                ⊡
              </button>
            )}
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

      {/* Quick Filters Panel (expanded mode only) */}
      {isExpanded && showFiltersPanel && onSetFilter && breakdownData && (
        <div className={styles.filtersPanel}>
          <div className={styles.filterSection}>
            <h4 className={styles.filterSectionTitle}>Provider</h4>
            <div className={styles.filterOptions}>
              {(breakdownData.by_provider || []).map((item) => (
                <button
                  key={item.id}
                  className={`${styles.filterOption} ${filters.provider === item.id ? styles.filterOptionActive : ''}`}
                  onClick={() => onSetFilter('provider', item.id)}
                >
                  <span className={styles.filterOptionLabel}>{item.label}</span>
                  <span className={styles.filterOptionCost}>${item.cost_usd.toFixed(2)}</span>
                </button>
              ))}
            </div>
          </div>
          <div className={styles.filterSection}>
            <h4 className={styles.filterSectionTitle}>Model</h4>
            <div className={styles.filterOptions}>
              {(breakdownData.by_model || []).slice(0, 8).map((item) => (
                <button
                  key={item.id}
                  className={`${styles.filterOption} ${filters.model === item.id ? styles.filterOptionActive : ''}`}
                  onClick={() => onSetFilter('model', item.id)}
                >
                  <span className={styles.filterOptionLabel}>{item.label}</span>
                  <span className={styles.filterOptionCost}>${item.cost_usd.toFixed(2)}</span>
                </button>
              ))}
            </div>
          </div>
          <div className={styles.filterSection}>
            <h4 className={styles.filterSectionTitle}>Source</h4>
            <div className={styles.filterOptions}>
              {(breakdownData.by_source_type || []).map((item) => (
                <button
                  key={item.id}
                  className={`${styles.filterOption} ${filters.source_type === item.id ? styles.filterOptionActive : ''}`}
                  onClick={() => onSetFilter('source', item.id)}
                >
                  <span className={styles.filterOptionLabel}>{item.label}</span>
                  <span className={styles.filterOptionCost}>${item.cost_usd.toFixed(2)}</span>
                </button>
              ))}
            </div>
          </div>
          <div className={styles.filterSection}>
            <h4 className={styles.filterSectionTitle}>Workspace</h4>
            <div className={styles.filterOptions}>
              {(breakdownData.by_workspace || []).slice(0, 6).map((item) => (
                <button
                  key={item.id}
                  className={`${styles.filterOption} ${filters.workspace === item.id ? styles.filterOptionActive : ''}`}
                  onClick={() => onSetFilter('workspace', item.id)}
                >
                  <span className={styles.filterOptionLabel}>{item.label}</span>
                  <span className={styles.filterOptionCost}>${item.cost_usd.toFixed(2)}</span>
                </button>
              ))}
            </div>
          </div>
        </div>
      )}

      {/* Chart content */}
      <div className={styles.content}>
        {activeChart === 'heatmap' && (
          <div className={styles.chartContainer}>
            <ActivityHeatmap
              data={heatmapData}
              gridData={heatmapGridData}
              selectedRange={selectedDateRange}
              onDateSelect={onDateSelect}
              onCellClick={onHeatmapCellClick}
              isExpanded={isExpanded}
              metric={metric === 'cost' ? 'cost' : 'tasks'}
            />
          </div>
        )}

        {activeChart === 'evolution' && (
          <div className={styles.chartContainer}>
            <div className={styles.chartControls}>
              <label className={styles.controlCheckbox}>
                <input
                  type="checkbox"
                  checked={logScale}
                  onChange={(e) => setLogScale(e.target.checked)}
                />
                Log scale
              </label>
            </div>
            {evolutionLoading ? (
              <div className={styles.loading}>Loading evolution data...</div>
            ) : (
              <TaskEvolutionChart
                tasks={evolutionData?.tasks || []}
                metric={metric}
                logScale={logScale}
                height={isExpanded ? 500 : 280}
                onTaskClick={handleTaskClick}
              />
            )}
          </div>
        )}

        {activeChart === 'usage' && (
          <div className={styles.chartContainer}>
            <div className={styles.chartControls}>
              <label className={styles.controlLabel}>
                Interval:
                <select
                  value={interval}
                  onChange={(e) => setInterval(e.target.value as IntervalType)}
                  className={styles.controlSelect}
                >
                  <option value="hour">Hourly</option>
                  <option value="day">Daily</option>
                  <option value="week">Weekly</option>
                </select>
              </label>
              <label className={styles.controlLabel}>
                Split by:
                <select
                  value={splitBy}
                  onChange={(e) => setSplitBy(e.target.value as SplitByType)}
                  className={styles.controlSelect}
                >
                  <option value="">None</option>
                  <option value="provider">Provider</option>
                  <option value="model">Model</option>
                  <option value="workspace">Workspace</option>
                </select>
              </label>
            </div>
            {usageLoading ? (
              <div className={styles.loading}>Loading usage data...</div>
            ) : (
              <UsageColumnChart
                points={usageData?.points || []}
                metric={metric}
                interval={interval}
                splitBy={splitBy || undefined}
                height={isExpanded ? 500 : 280}
              />
            )}
          </div>
        )}

        {activeChart === 'breakdown' && (
          <div className={styles.chartContainer}>
            <div className={styles.chartControls}>
              <label className={styles.controlLabel}>
                Group by:
                <select
                  value={breakdownDimension}
                  onChange={(e) => setBreakdownDimension(e.target.value as BreakdownDimension)}
                  className={styles.controlSelect}
                >
                  <option value="provider">Provider</option>
                  <option value="model">Model</option>
                  <option value="workspace">Workspace</option>
                  <option value="source_type">Source</option>
                </select>
              </label>
            </div>
            {breakdownLoading ? (
              <div className={styles.loading}>Loading breakdown data...</div>
            ) : breakdownData ? (
              <CostBreakdownChart
                items={
                  breakdownDimension === 'provider' ? breakdownData.by_provider :
                  breakdownDimension === 'model' ? breakdownData.by_model :
                  breakdownDimension === 'workspace' ? breakdownData.by_workspace :
                  breakdownData.by_source_type
                }
                dimension={breakdownDimension}
                totalCost={breakdownData.total_cost}
                metric={metric}
                height={isExpanded ? 450 : 260}
              />
            ) : (
              <div className={styles.loading}>No breakdown data available</div>
            )}
          </div>
        )}
      </div>

      {/* CLI Command hint - centralized component */}
      <div className={styles.cliRow}>
        <CliCommandHint
          command={
            activeChart === 'evolution' ? evolutionData?.cli_command :
            activeChart === 'usage' ? usageData?.cli_command :
            undefined
          }
          commandType={activeChart !== 'evolution' && activeChart !== 'usage' ? currentCommandType : undefined}
          filters={filters}
          compact={!isExpanded}
        />
      </div>
      </div>
    </>
  );
};

export default VisualizationPanel;
