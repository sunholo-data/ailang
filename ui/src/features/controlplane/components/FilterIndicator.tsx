/**
 * FilterIndicator - Displays active filters as removable pills
 *
 * Shows which filters are currently affecting the visualization,
 * allowing users to quickly see and clear individual filters.
 */
import React from 'react';
import type { ControlPlaneFilters } from '../types';
import styles from './FilterIndicator.module.css';

export interface FilterIndicatorProps {
  filters: ControlPlaneFilters;
  onClearFilter: (key: keyof ControlPlaneFilters) => void;
  onClearAll: () => void;
  /** Compact mode for smaller displays */
  compact?: boolean;
}

// Human-readable labels for filter keys
const filterLabels: Record<string, string> = {
  source_type: 'Source',
  provider: 'Provider',
  model: 'Model',
  workspace: 'Workspace',
  start_date: 'From',
  end_date: 'To',
  status: 'Status',
  search: 'Search',
};

// Human-readable values for source types
const sourceTypeLabels: Record<string, string> = {
  eval: 'Eval',
  coordinator: 'Coordinator',
  direct_api: 'Direct API',
  local: 'Local',
  other: 'Other',
};

// Format a filter value for display
const formatValue = (key: string, value: string): string => {
  if (key === 'source_type') {
    return sourceTypeLabels[value] || value;
  }
  if (key === 'start_date' || key === 'end_date') {
    // Format date as "Jan 5"
    const date = new Date(value + 'T00:00:00');
    return date.toLocaleDateString('en-US', { month: 'short', day: 'numeric' });
  }
  return value;
};

export const FilterIndicator: React.FC<FilterIndicatorProps> = ({
  filters,
  onClearFilter,
  onClearAll,
  compact = false,
}) => {
  // Get active filters (excluding sort/order which are not user-visible filters)
  const activeFilters = Object.entries(filters).filter(([key, value]) => {
    if (!value) return false;
    if (key === 'sort' || key === 'order') return false;
    if (key === 'status' && value === 'all') return false;
    return true;
  }) as Array<[keyof ControlPlaneFilters, string]>;

  if (activeFilters.length === 0) {
    return null;
  }

  // Combine start_date and end_date into a single "Date Range" pill if both present
  const hasDateRange = filters.start_date && filters.end_date;
  const displayFilters = activeFilters.filter(([key]) => {
    if (hasDateRange && (key === 'start_date' || key === 'end_date')) {
      return key === 'start_date'; // Only show one for the range
    }
    return true;
  });

  return (
    <div className={`${styles.container} ${compact ? styles.compact : ''}`}>
      <span className={styles.label}>Filters:</span>
      <div className={styles.pills}>
        {displayFilters.map(([key, value]) => {
          // Special handling for date range
          if (hasDateRange && key === 'start_date') {
            const startDate = formatValue('start_date', filters.start_date!);
            const endDate = formatValue('end_date', filters.end_date!);
            const rangeText = filters.start_date === filters.end_date
              ? startDate
              : `${startDate} → ${endDate}`;

            return (
              <span key="date_range" className={styles.pill}>
                <span className={styles.pillLabel}>Date</span>
                <span className={styles.pillValue}>{rangeText}</span>
                <button
                  className={styles.pillRemove}
                  onClick={() => {
                    onClearFilter('start_date');
                    onClearFilter('end_date');
                  }}
                  title="Clear date filter"
                >
                  ×
                </button>
              </span>
            );
          }

          return (
            <span key={key} className={styles.pill}>
              <span className={styles.pillLabel}>{filterLabels[key] || key}</span>
              <span className={styles.pillValue}>{formatValue(key, value)}</span>
              <button
                className={styles.pillRemove}
                onClick={() => onClearFilter(key)}
                title={`Clear ${filterLabels[key] || key} filter`}
              >
                ×
              </button>
            </span>
          );
        })}
      </div>
      {activeFilters.length > 1 && (
        <button className={styles.clearAll} onClick={onClearAll}>
          Clear All
        </button>
      )}
    </div>
  );
};

export default FilterIndicator;
