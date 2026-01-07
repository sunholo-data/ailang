/**
 * CommandBar - Search and filter controls for Control Plane
 */
import React from 'react';
import { StatusFilter } from '../types';
import styles from '../ControlPlane.module.css';

// Status filter options with labels
const statusFilterOptions: { value: StatusFilter; label: string }[] = [
  { value: 'all', label: 'All' },
  { value: 'running', label: 'Running' },
  { value: 'pending', label: 'Pending' },
  { value: 'completed', label: 'Completed' },
  { value: 'failed', label: 'Failed' },
];

export interface CommandBarProps {
  searchQuery: string;
  onSearchChange: (q: string) => void;
  timeRange: string;
  onTimeRangeChange: (r: string) => void;
  statusFilter: StatusFilter;
  onStatusChange: (status: StatusFilter) => void;
}

export const CommandBar: React.FC<CommandBarProps> = ({
  searchQuery,
  onSearchChange,
  timeRange,
  onTimeRangeChange,
  statusFilter,
  onStatusChange,
}) => (
  <div className={styles.commandBar}>
    <div className={styles.searchContainer}>
      <span className={styles.searchIcon}>⌘</span>
      <input
        type="text"
        className={styles.searchInput}
        placeholder="Search traces, messages, tasks..."
        value={searchQuery}
        onChange={(e) => onSearchChange(e.target.value)}
      />
      {searchQuery && (
        <button
          className={styles.searchClear}
          onClick={() => onSearchChange('')}
          title="Clear search"
        >
          ×
        </button>
      )}
      <kbd className={styles.searchKbd}>K</kbd>
    </div>
    <div className={styles.commandActions}>
      <select
        className={styles.timeSelect}
        value={timeRange}
        onChange={(e) => onTimeRangeChange(e.target.value)}
      >
        <option value="1h">Last 1 hour</option>
        <option value="24h">Last 24 hours</option>
        <option value="7d">Last 7 days</option>
        <option value="30d">Last 30 days</option>
        <option value="90d">Last 90 days</option>
      </select>
      <div className={styles.filterChips}>
        {statusFilterOptions.map(({ value, label }) => (
          <button
            key={value}
            className={`${styles.chip} ${statusFilter === value ? styles.chipActive : ''} ${styles[`chip${value.charAt(0).toUpperCase() + value.slice(1)}`] || ''}`}
            onClick={() => onStatusChange(value)}
          >
            {label}
          </button>
        ))}
      </div>
      <div className={styles.liveIndicator}>
        <span className={styles.liveDot} />
        <span>LIVE</span>
      </div>
    </div>
  </div>
);

export default CommandBar;
