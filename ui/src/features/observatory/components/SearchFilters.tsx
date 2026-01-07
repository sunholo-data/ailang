import React from 'react';
import styles from './SearchFilters.module.css';

// Status filter options
export type StatusFilter = 'all' | 'pending' | 'running' | 'completed' | 'failed';

// Time range options
export type TimeRange = 'all' | '1h' | '24h' | '7d' | '30d';

export interface FilterState {
  search: string;
  status: StatusFilter;
  timeRange: TimeRange;
  provider?: string;
}

export interface SearchFiltersProps {
  filters: FilterState;
  onChange: (filters: FilterState) => void;
  showStatusFilter?: boolean;
  showProviderFilter?: boolean;
  statusOptions?: StatusFilter[];
  placeholder?: string;
}

const defaultStatusOptions: StatusFilter[] = ['all', 'pending', 'running', 'completed', 'failed'];

const timeRangeLabels: Record<TimeRange, string> = {
  'all': 'All time',
  '1h': 'Last hour',
  '24h': 'Last 24h',
  '7d': 'Last 7 days',
  '30d': 'Last 30 days',
};

const statusLabels: Record<StatusFilter, string> = {
  'all': 'All',
  'pending': 'Pending',
  'running': 'Running',
  'completed': 'Completed',
  'failed': 'Failed',
};

export function SearchFilters({
  filters,
  onChange,
  showStatusFilter = true,
  showProviderFilter = false,
  statusOptions = defaultStatusOptions,
  placeholder = 'Search...',
}: SearchFiltersProps) {
  const handleSearchChange = (e: React.ChangeEvent<HTMLInputElement>) => {
    onChange({ ...filters, search: e.target.value });
  };

  const handleStatusChange = (status: StatusFilter) => {
    onChange({ ...filters, status });
  };

  const handleTimeRangeChange = (e: React.ChangeEvent<HTMLSelectElement>) => {
    onChange({ ...filters, timeRange: e.target.value as TimeRange });
  };

  const handleProviderChange = (e: React.ChangeEvent<HTMLSelectElement>) => {
    onChange({ ...filters, provider: e.target.value || undefined });
  };

  const clearFilters = () => {
    onChange({
      search: '',
      status: 'all',
      timeRange: 'all',
      provider: undefined,
    });
  };

  const hasActiveFilters = filters.search || filters.status !== 'all' || filters.timeRange !== 'all' || filters.provider;

  return (
    <div className={styles.container}>
      {/* Search Input */}
      <div className={styles.searchBox}>
        <span className={styles.searchIcon}>🔍</span>
        <input
          type="text"
          className={styles.searchInput}
          placeholder={placeholder}
          value={filters.search}
          onChange={handleSearchChange}
        />
        {filters.search && (
          <button
            className={styles.clearSearch}
            onClick={() => onChange({ ...filters, search: '' })}
            title="Clear search"
          >
            ×
          </button>
        )}
      </div>

      {/* Status Filter Chips */}
      {showStatusFilter && (
        <div className={styles.statusChips}>
          {statusOptions.map((status) => (
            <button
              key={status}
              className={`${styles.statusChip} ${filters.status === status ? styles.statusChipActive : ''} ${styles[`chip${status.charAt(0).toUpperCase() + status.slice(1)}`]}`}
              onClick={() => handleStatusChange(status)}
            >
              {statusLabels[status]}
            </button>
          ))}
        </div>
      )}

      {/* Time Range Dropdown */}
      <select
        className={styles.dropdown}
        value={filters.timeRange}
        onChange={handleTimeRangeChange}
      >
        {Object.entries(timeRangeLabels).map(([value, label]) => (
          <option key={value} value={value}>{label}</option>
        ))}
      </select>

      {/* Provider Filter */}
      {showProviderFilter && (
        <select
          className={styles.dropdown}
          value={filters.provider || ''}
          onChange={handleProviderChange}
        >
          <option value="">All providers</option>
          <option value="claude">Claude</option>
          <option value="gemini">Gemini</option>
          <option value="openai">OpenAI</option>
        </select>
      )}

      {/* Clear All Button */}
      {hasActiveFilters && (
        <button className={styles.clearAll} onClick={clearFilters}>
          Clear filters
        </button>
      )}
    </div>
  );
}

// Helper function to filter tasks based on FilterState
export function filterTasks<T extends {
  id: string;
  title?: string;
  status: string;
  created_at: string;
}>(tasks: T[], filters: FilterState): T[] {
  return tasks.filter((task) => {
    // Search filter
    if (filters.search) {
      const searchLower = filters.search.toLowerCase();
      const titleMatch = task.title?.toLowerCase().includes(searchLower);
      const idMatch = task.id.toLowerCase().includes(searchLower);
      if (!titleMatch && !idMatch) return false;
    }

    // Status filter
    if (filters.status !== 'all' && task.status !== filters.status) {
      return false;
    }

    // Time range filter
    if (filters.timeRange !== 'all') {
      const now = Date.now();
      const createdAt = new Date(task.created_at).getTime();
      const ranges: Record<TimeRange, number> = {
        'all': Infinity,
        '1h': 60 * 60 * 1000,
        '24h': 24 * 60 * 60 * 1000,
        '7d': 7 * 24 * 60 * 60 * 1000,
        '30d': 30 * 24 * 60 * 60 * 1000,
      };
      if (now - createdAt > ranges[filters.timeRange]) return false;
    }

    return true;
  });
}

// Helper function to filter traces based on FilterState
export function filterTraces<T extends {
  trace_id: string;
  root_span?: string;
  status: string;
  start_time: string;
  service_name?: string;
}>(traces: T[], filters: FilterState): T[] {
  return traces.filter((trace) => {
    // Search filter
    if (filters.search) {
      const searchLower = filters.search.toLowerCase();
      const spanMatch = trace.root_span?.toLowerCase().includes(searchLower);
      const idMatch = trace.trace_id.toLowerCase().includes(searchLower);
      const serviceMatch = trace.service_name?.toLowerCase().includes(searchLower);
      if (!spanMatch && !idMatch && !serviceMatch) return false;
    }

    // Status filter (map task status to trace status)
    if (filters.status !== 'all') {
      const statusMap: Record<string, string[]> = {
        'completed': ['OK', 'ok'],
        'failed': ['ERROR', 'error'],
        'running': ['RUNNING', 'running'],
        'pending': ['UNSET', 'unset', ''],
      };
      const allowedStatuses = statusMap[filters.status] || [];
      if (!allowedStatuses.includes(trace.status)) return false;
    }

    // Time range filter
    if (filters.timeRange !== 'all') {
      const now = Date.now();
      const startTime = new Date(trace.start_time).getTime();
      const ranges: Record<TimeRange, number> = {
        'all': Infinity,
        '1h': 60 * 60 * 1000,
        '24h': 24 * 60 * 60 * 1000,
        '7d': 7 * 24 * 60 * 60 * 1000,
        '30d': 30 * 24 * 60 * 60 * 1000,
      };
      if (now - startTime > ranges[filters.timeRange]) return false;
    }

    // Provider filter (if applicable)
    if (filters.provider && trace.service_name) {
      const serviceLower = trace.service_name.toLowerCase();
      if (!serviceLower.includes(filters.provider.toLowerCase())) {
        return false;
      }
    }

    return true;
  });
}

export default SearchFilters;
