/**
 * MessageQueue - Real-time event feed for Control Plane
 * Now with filtering support for date range and event types
 */
import React, { useState, useMemo, useCallback } from 'react';
import type { EventMessage, DateRange } from './types';
import type { ControlPlaneFilters } from '../types';
import { CliCommandHint } from './CliCommandHint';
import styles from '../ControlPlane.module.css';

export type EventType = EventMessage['type'];

export interface MessageQueueProps {
  events: EventMessage[];
  onEventClick: (event: EventMessage) => void;
  loading?: boolean;
  pageSize?: number;
  // Filter props
  selectedDateRange?: DateRange | null;
  onDateRangeChange?: (range: DateRange | null) => void;
  selectedTypes?: EventType[];
  onTypeFilterChange?: (types: EventType[]) => void;
  // Filters for CLI hint
  filters?: ControlPlaneFilters;
}

const getEventIcon = (type: EventMessage['type']): string => {
  switch (type) {
    case 'task_start': return '▶';
    case 'task_complete': return '✓';
    case 'task_error': return '✕';
    case 'handoff': return '→';
    case 'approval': return '⏳';
    case 'message': return '◉';
  }
};

const getEventColor = (type: EventMessage['type']): string => {
  switch (type) {
    case 'task_start': return 'primary';
    case 'task_complete': return 'success';
    case 'task_error': return 'error';
    case 'handoff': return 'amber';
    case 'approval': return 'warning';
    case 'message': return 'muted';
  }
};

const formatTime = (timestamp: string): string => {
  const date = new Date(timestamp);
  return date.toLocaleTimeString('en-US', { hour12: false });
};

const formatRelativeTime = (timestamp: string): string => {
  const diff = Date.now() - new Date(timestamp).getTime();
  if (diff < 60000) return `${Math.floor(diff / 1000)}s ago`;
  if (diff < 3600000) return `${Math.floor(diff / 60000)}m ago`;
  return `${Math.floor(diff / 3600000)}h ago`;
};

const ALL_EVENT_TYPES: EventType[] = ['task_start', 'task_complete', 'task_error', 'handoff', 'approval', 'message'];

const formatDateRange = (range: DateRange): string => {
  const opts: Intl.DateTimeFormatOptions = { month: 'short', day: 'numeric' };
  const startDate = new Date(range.start + 'T00:00:00');
  const endDate = new Date(range.end + 'T00:00:00');
  const startStr = startDate.toLocaleDateString('en-US', opts);
  const endStr = endDate.toLocaleDateString('en-US', opts);
  if (startStr === endStr) return startStr;
  return `${startStr} - ${endStr}`;
};

export const MessageQueue: React.FC<MessageQueueProps> = ({
  events,
  onEventClick,
  loading,
  pageSize = 10,
  selectedDateRange,
  onDateRangeChange,
  selectedTypes,
  onTypeFilterChange,
  filters,
}) => {
  const [currentPage, setCurrentPage] = useState(0);
  const [showTypeFilter, setShowTypeFilter] = useState(false);

  // Filter events by date range and types
  const filteredEvents = useMemo(() => {
    let result = events;

    // Filter by date range - convert strings to Date objects for reliable comparison
    if (selectedDateRange) {
      const startDate = new Date(selectedDateRange.start + 'T00:00:00');
      const endDate = new Date(selectedDateRange.end + 'T23:59:59');
      result = result.filter((event) => {
        const eventDate = new Date(event.timestamp);
        return eventDate >= startDate && eventDate <= endDate;
      });
    }

    // Filter by event types
    if (selectedTypes && selectedTypes.length > 0) {
      result = result.filter((event) => selectedTypes.includes(event.type));
    }

    return result;
  }, [events, selectedDateRange, selectedTypes]);

  // Calculate pagination on filtered events
  const totalPages = useMemo(() => Math.ceil(filteredEvents.length / pageSize), [filteredEvents.length, pageSize]);
  const paginatedEvents = useMemo(() => {
    const start = currentPage * pageSize;
    return filteredEvents.slice(start, start + pageSize);
  }, [filteredEvents, currentPage, pageSize]);

  // Reset to first page when events change significantly
  React.useEffect(() => {
    if (currentPage >= totalPages && totalPages > 0) {
      setCurrentPage(totalPages - 1);
    }
  }, [totalPages, currentPage]);

  const goToPrevPage = () => setCurrentPage((p) => Math.max(0, p - 1));
  const goToNextPage = () => setCurrentPage((p) => Math.min(totalPages - 1, p + 1));

  const handleTypeToggle = useCallback((type: EventType) => {
    if (!onTypeFilterChange) return;
    const current = selectedTypes || [];
    if (current.includes(type)) {
      onTypeFilterChange(current.filter(t => t !== type));
    } else {
      onTypeFilterChange([...current, type]);
    }
  }, [selectedTypes, onTypeFilterChange]);

  const handleClearDateFilter = useCallback(() => {
    onDateRangeChange?.(null);
  }, [onDateRangeChange]);

  const handleSelectAllTypes = useCallback(() => {
    onTypeFilterChange?.(ALL_EVENT_TYPES);
  }, [onTypeFilterChange]);

  const handleClearTypeFilter = useCallback(() => {
    onTypeFilterChange?.([]);
  }, [onTypeFilterChange]);

  const hasFilters = !!(selectedDateRange || (selectedTypes && selectedTypes.length > 0));

  return (
    <div className={styles.messageQueue}>
      <div className={styles.queueHeader}>
        <h3 className={styles.panelTitle}>
          <span className={styles.panelIcon}>▥</span>
          Event Queue
        </h3>
        <span className={styles.queueCount}>
          {loading ? '...' : hasFilters ? `${filteredEvents.length}/${events.length}` : `${events.length}`}
        </span>
      </div>

      {/* Filter toolbar */}
      <div className={styles.queueFilters}>
        {/* Date range filter */}
        <button
          className={`${styles.filterBtn} ${selectedDateRange ? styles.filterBtnActive : ''}`}
          onClick={handleClearDateFilter}
          disabled={!selectedDateRange}
          title={selectedDateRange ? 'Clear date filter' : 'Select dates from heatmap'}
        >
          <span className={styles.filterIcon}>📅</span>
          {selectedDateRange ? formatDateRange(selectedDateRange) : 'All dates'}
          {selectedDateRange && <span className={styles.filterClear}>×</span>}
        </button>

        {/* Event type filter */}
        <div className={styles.filterDropdown}>
          <button
            className={`${styles.filterBtn} ${selectedTypes && selectedTypes.length > 0 ? styles.filterBtnActive : ''}`}
            onClick={() => setShowTypeFilter(!showTypeFilter)}
          >
            <span className={styles.filterIcon}>◉</span>
            {selectedTypes && selectedTypes.length > 0 ? `${selectedTypes.length} types` : 'All types'}
            <span className={styles.filterChevron}>{showTypeFilter ? '▲' : '▼'}</span>
          </button>
          {showTypeFilter && (
            <div className={styles.filterMenu}>
              <div className={styles.filterMenuActions}>
                <button onClick={handleSelectAllTypes} className={styles.filterMenuAction}>All</button>
                <button onClick={handleClearTypeFilter} className={styles.filterMenuAction}>None</button>
              </div>
              {ALL_EVENT_TYPES.map((type) => (
                <label key={type} className={styles.filterOption}>
                  <input
                    type="checkbox"
                    checked={!selectedTypes || selectedTypes.length === 0 || selectedTypes.includes(type)}
                    onChange={() => handleTypeToggle(type)}
                  />
                  <span className={styles.filterOptionIcon} data-type={getEventColor(type)}>
                    {getEventIcon(type)}
                  </span>
                  <span className={styles.filterOptionLabel}>{type.replace('_', ' ')}</span>
                </label>
              ))}
            </div>
          )}
        </div>
      </div>
      <div className={styles.queueList}>
        {loading && (
          <div className={styles.queueEmpty}>
            <span className={styles.queueEmptyIcon}>◎</span>
            <span className={styles.queueEmptyText}>Loading events...</span>
          </div>
        )}
        {!loading && events.length === 0 && (
          <div className={styles.queueEmpty}>
            <span className={styles.queueEmptyIcon}>◎</span>
            <span className={styles.queueEmptyText}>No recent events</span>
            <span className={styles.queueEmptyHint}>
              Events appear when tasks run or agents communicate
            </span>
          </div>
        )}
        {!loading && paginatedEvents.map((event) => (
          <div
            key={event.id}
            className={styles.queueItem}
            onClick={() => onEventClick(event)}
            data-type={getEventColor(event.type)}
          >
            <span className={styles.queueIcon} data-type={getEventColor(event.type)}>
              {getEventIcon(event.type)}
            </span>
            <div className={styles.queueContent}>
              <div className={styles.queueMeta}>
                <span className={styles.queueSource}>{event.source}</span>
                {event.target && (
                  <>
                    <span className={styles.queueArrow}>→</span>
                    <span className={styles.queueTarget}>{event.target}</span>
                  </>
                )}
              </div>
              <div className={styles.queueMessage}>{event.content}</div>
            </div>
            <div className={styles.queueTime}>
              <span className={styles.queueRelative}>{formatRelativeTime(event.timestamp)}</span>
              <span className={styles.queueAbsolute}>{formatTime(event.timestamp)}</span>
            </div>
          </div>
        ))}
      </div>
      {/* Pagination controls */}
      {!loading && totalPages > 1 && (
        <div className={styles.queuePagination}>
          <button
            className={styles.queuePageBtn}
            onClick={goToPrevPage}
            disabled={currentPage === 0}
            title="Previous page"
          >
            ←
          </button>
          <span className={styles.queuePageInfo}>
            {currentPage + 1} / {totalPages}
          </span>
          <button
            className={styles.queuePageBtn}
            onClick={goToNextPage}
            disabled={currentPage >= totalPages - 1}
            title="Next page"
          >
            →
          </button>
        </div>
      )}

      {/* CLI command hint */}
      <CliCommandHint
        commandType="inbox"
        filters={filters}
        limit={pageSize}
        compact
      />
    </div>
  );
};

export default MessageQueue;
