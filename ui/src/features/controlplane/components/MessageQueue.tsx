/**
 * MessageQueue - Real-time event feed for Control Plane
 * Now with filtering support for date range and event types
 * M-CHAT-HISTORY-DB Phase 3: Adds chat preview for Claude Code events
 */
import React, { useState, useMemo, useCallback, useEffect, useRef } from 'react';
import type { EventMessage, DateRange } from './types';
import type { ControlPlaneFilters, SortField, SortOrder } from '../types';
import { hasActiveFilters } from '../types';
import { CliCommandHint } from './CliCommandHint';
import styles from '../ControlPlane.module.css';

// Chat preview cache type
interface ChatPreview {
  sessionId: string;
  firstUserMessage: string;
  loading: boolean;
  error?: string;
}

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
  // Sort props
  sortBy?: SortField;
  sortOrder?: SortOrder;
  onSortChange?: (sort: SortField, order: SortOrder) => void;
  // Filters for CLI hint
  filters?: ControlPlaneFilters;
  // Selection highlighting
  selectedEventId?: string | null;
}

const getEventIcon = (type: EventMessage['type']): string => {
  switch (type) {
    case 'task_start': return '▶';
    case 'task_complete': return '✓';
    case 'task_error': return '✕';
    case 'handoff': return '→';
    case 'approval': return '⏳';
    case 'message': return '◉';
    case 'session': return '◈';
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
    case 'session': return 'primary';
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

const ALL_EVENT_TYPES: EventType[] = ['task_start', 'task_complete', 'task_error', 'handoff', 'approval', 'message', 'session'];

const formatDateRange = (range: DateRange): string => {
  const opts: Intl.DateTimeFormatOptions = { month: 'short', day: 'numeric' };
  const startDate = new Date(range.start + 'T00:00:00');
  const endDate = new Date(range.end + 'T00:00:00');
  const startStr = startDate.toLocaleDateString('en-US', opts);
  const endStr = endDate.toLocaleDateString('en-US', opts);
  if (startStr === endStr) return startStr;
  return `${startStr} - ${endStr}`;
};

// Helper to extract workspace name from full path
const getWorkspaceName = (workspace: string): string => {
  const parts = workspace.split('/');
  return parts[parts.length - 1] || workspace;
};

// Helper to format duration in ms to human-readable string
const formatDuration = (ms: number): string => {
  if (ms < 1000) return `${ms}ms`;
  if (ms < 60000) return `${(ms / 1000).toFixed(1)}s`;
  return `${(ms / 60000).toFixed(1)}m`;
};

// Helper to get payload preview when no directive is available
const getPayloadPreview = (event: EventMessage): string => {
  const payload = event.metadata?.payload as string;
  if (!payload) return 'No description';
  // Remove newlines and trim
  const clean = payload.replace(/\n+/g, ' ').trim();
  return clean.length > 100 ? clean.slice(0, 97) + '...' : clean;
};

// Sort options configuration
const SORT_OPTIONS: { value: SortField; label: string }[] = [
  { value: 'timestamp', label: 'Recent' },
  { value: 'cost', label: 'Cost' },
  { value: 'tokens', label: 'Tokens' },
  { value: 'turns', label: 'Turns' },
  { value: 'duration', label: 'Duration' },
];

export const MessageQueue: React.FC<MessageQueueProps> = ({
  events,
  onEventClick,
  loading,
  pageSize = 10,
  selectedDateRange,
  onDateRangeChange,
  selectedTypes,
  onTypeFilterChange,
  sortBy = 'timestamp',
  sortOrder = 'desc',
  onSortChange,
  filters,
  selectedEventId,
}) => {
  const [currentPage, setCurrentPage] = useState(0);
  const [showTypeFilter, setShowTypeFilter] = useState(false);
  const [showSortDropdown, setShowSortDropdown] = useState(false);
  const [expandedEventId, setExpandedEventId] = useState<string | null>(null);

  // M-CHAT-HISTORY-DB: Chat preview cache for Claude Code events
  const [chatPreviews, setChatPreviews] = useState<Record<string, ChatPreview>>({});
  const fetchedSessionsRef = useRef<Set<string>>(new Set());

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

  // M-CHAT-HISTORY-DB: Fetch chat previews for Claude Code events (session type)
  // For these events, task_id IS the session_id (see backend_controlplane.go:1112)
  const fetchChatPreview = useCallback(async (sessionId: string) => {
    // Skip if already fetched or fetching
    if (fetchedSessionsRef.current.has(sessionId)) return;
    fetchedSessionsRef.current.add(sessionId);

    // Mark as loading
    setChatPreviews((prev) => ({
      ...prev,
      [sessionId]: { sessionId, firstUserMessage: '', loading: true },
    }));

    try {
      // Fetch first message from chat history
      // Use the JSONL endpoint with limit=1 to get just the first user message
      const response = await fetch(`/api/claude-history/session/${sessionId}?limit=1`);
      if (!response.ok) {
        throw new Error(`HTTP ${response.status}`);
      }

      const data = await response.json();
      let firstUserMessage = '';

      // Extract first user message text
      if (data?.messages && data.messages.length > 0) {
        const firstMsg = data.messages.find((m: { type: string }) => m.type === 'user');
        if (firstMsg?.content) {
          // Content is array of content blocks - find text block
          for (const block of firstMsg.content) {
            if (block.type === 'text' && block.text) {
              firstUserMessage = block.text;
              // Truncate if too long
              if (firstUserMessage.length > 150) {
                firstUserMessage = firstUserMessage.slice(0, 147) + '...';
              }
              break;
            }
          }
        }
      }

      setChatPreviews((prev) => ({
        ...prev,
        [sessionId]: { sessionId, firstUserMessage, loading: false },
      }));
    } catch {
      setChatPreviews((prev) => ({
        ...prev,
        [sessionId]: { sessionId, firstUserMessage: '', loading: false, error: 'Failed to load' },
      }));
    }
  }, []);

  // Calculate pagination on filtered events
  const totalPages = useMemo(() => Math.ceil(filteredEvents.length / pageSize), [filteredEvents.length, pageSize]);
  const paginatedEvents = useMemo(() => {
    const start = currentPage * pageSize;
    return filteredEvents.slice(start, start + pageSize);
  }, [filteredEvents, currentPage, pageSize]);

  // Fetch chat previews for visible session events (must be after paginatedEvents is defined)
  useEffect(() => {
    const sessionEvents = paginatedEvents.filter(
      (e) => e.type === 'session' && e.task_id && !chatPreviews[e.task_id]
    );

    // Fetch in batches to avoid overwhelming the API
    sessionEvents.slice(0, 5).forEach((event) => {
      if (event.task_id) {
        fetchChatPreview(event.task_id);
      }
    });
  }, [paginatedEvents, chatPreviews, fetchChatPreview]);

  // Reset to first page when events change significantly
  React.useEffect(() => {
    if (currentPage >= totalPages && totalPages > 0) {
      setCurrentPage(totalPages - 1);
    }
  }, [totalPages, currentPage]);

  const goToPrevPage = () => setCurrentPage((p) => Math.max(0, p - 1));
  const goToNextPage = () => setCurrentPage((p) => Math.min(totalPages - 1, p + 1));

  // Exclusive selection: click type to select ONLY that type
  const handleTypeSelect = useCallback((type: EventType) => {
    if (!onTypeFilterChange) return;
    onTypeFilterChange([type]);  // Select ONLY this type
    setShowTypeFilter(false);    // Close dropdown after selection
  }, [onTypeFilterChange]);

  // Additive toggle: checkbox adds/removes from current selection
  const handleTypeToggle = useCallback((type: EventType, e: React.MouseEvent) => {
    e.stopPropagation();  // Prevent row click
    if (!onTypeFilterChange) return;
    const current = selectedTypes || [];

    // Empty array means "all selected" - toggling removes one type
    if (current.length === 0) {
      // "All" is selected, toggle OFF means "all except this one"
      onTypeFilterChange(ALL_EVENT_TYPES.filter(t => t !== type));
    } else if (current.includes(type)) {
      // Type is currently selected, remove it
      const newTypes = current.filter(t => t !== type);
      // If we'd have all types selected, use empty array (= show all)
      if (newTypes.length === ALL_EVENT_TYPES.length) {
        onTypeFilterChange([]);
      } else {
        onTypeFilterChange(newTypes);
      }
    } else {
      // Type is not selected, add it
      const newTypes = [...current, type];
      // If we now have all types, use empty array (= show all)
      if (newTypes.length === ALL_EVENT_TYPES.length) {
        onTypeFilterChange([]);
      } else {
        onTypeFilterChange(newTypes);
      }
    }
  }, [selectedTypes, onTypeFilterChange]);

  const handleClearDateFilter = useCallback(() => {
    onDateRangeChange?.(null);
  }, [onDateRangeChange]);

  // "All" means clear filter (show all types) - use empty array
  const handleSelectAllTypes = useCallback(() => {
    onTypeFilterChange?.([]);
  }, [onTypeFilterChange]);

  // "Clear" removes all type selections - filter to show nothing
  // We use a sentinel value to indicate "none selected"
  const handleClearTypeFilter = useCallback(() => {
    // Setting to a single impossible type would hide all events
    // But for UX, "Clear" should mean "reset to default" = show all
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
          {selectedDateRange ? formatDateRange(selectedDateRange) : 'Dates'}
          {selectedDateRange && <span className={styles.filterClear}>×</span>}
        </button>

        {/* Event type filter */}
        <div className={styles.filterDropdown}>
          <button
            className={`${styles.filterBtn} ${selectedTypes && selectedTypes.length > 0 ? styles.filterBtnActive : ''}`}
            onClick={() => setShowTypeFilter(!showTypeFilter)}
          >
            <span className={styles.filterIcon}>◉</span>
            {selectedTypes && selectedTypes.length > 0 ? `${selectedTypes.length} types` : 'types'}
            <span className={styles.filterChevron}>{showTypeFilter ? '▲' : '▼'}</span>
          </button>
          {showTypeFilter && (
            <div className={styles.filterMenu}>
              <div className={styles.filterMenuActions}>
                <button onClick={handleSelectAllTypes} className={styles.filterMenuAction}>All</button>
                <button onClick={handleClearTypeFilter} className={styles.filterMenuAction}>Clear</button>
              </div>
              {ALL_EVENT_TYPES.map((type) => (
                <div
                  key={type}
                  className={styles.filterOption}
                  onClick={() => handleTypeSelect(type)}
                  title={`Click to show only ${type.replace('_', ' ')} events`}
                >
                  <input
                    type="checkbox"
                    checked={!selectedTypes || selectedTypes.length === 0 || selectedTypes.includes(type)}
                    onChange={(e) => handleTypeToggle(type, e as unknown as React.MouseEvent)}
                    onClick={(e) => e.stopPropagation()}
                    title="Toggle in selection"
                  />
                  <span className={styles.filterOptionIcon} data-type={getEventColor(type)}>
                    {getEventIcon(type)}
                  </span>
                  <span className={styles.filterOptionLabel}>{type.replace('_', ' ')}</span>
                </div>
              ))}
            </div>
          )}
        </div>

        {/* Sort dropdown */}
        <div className={styles.filterDropdown}>
          <button
            className={`${styles.filterBtn} ${sortBy !== 'timestamp' ? styles.filterBtnActive : ''}`}
            onClick={() => setShowSortDropdown(!showSortDropdown)}
          >
            <span className={styles.filterIcon}>{sortOrder === 'desc' ? '↓' : '↑'}</span>
            {SORT_OPTIONS.find(o => o.value === sortBy)?.label || 'Sort'}
            <span className={styles.filterChevron}>{showSortDropdown ? '▲' : '▼'}</span>
          </button>
          {showSortDropdown && (
            <div className={styles.filterMenu}>
              {SORT_OPTIONS.map((option) => (
                <div
                  key={option.value}
                  className={`${styles.filterOption} ${sortBy === option.value ? styles.filterOptionSelected : ''}`}
                  onClick={() => {
                    if (onSortChange) {
                      // If clicking same sort, toggle order; otherwise select new sort with desc
                      if (sortBy === option.value) {
                        onSortChange(option.value, sortOrder === 'desc' ? 'asc' : 'desc');
                      } else {
                        onSortChange(option.value, 'desc');
                      }
                    }
                    setShowSortDropdown(false);
                  }}
                >
                  <span className={styles.filterOptionIcon}>
                    {sortBy === option.value ? (sortOrder === 'desc' ? '↓' : '↑') : '○'}
                  </span>
                  <span className={styles.filterOptionLabel}>{option.label}</span>
                </div>
              ))}
            </div>
          )}
        </div>
      </div>

      {/* Active dimension filters (workspace/provider/model from sidebar) */}
      {filters && hasActiveFilters(filters) && (
        <div className={styles.queueActiveFilters}>
          {filters.workspace && (
            <span className={styles.queueFilterChip}>
              <span className={styles.queueFilterIcon}>⬡</span>
              {filters.workspace}
            </span>
          )}
          {filters.provider && (
            <span className={styles.queueFilterChip}>
              <span className={styles.queueFilterIcon}>◈</span>
              {filters.provider}
            </span>
          )}
          {filters.model && (
            <span className={styles.queueFilterChip}>
              <span className={styles.queueFilterIcon}>◎</span>
              {filters.model}
            </span>
          )}
          {filters.source_type && (
            <span className={styles.queueFilterChip}>
              <span className={styles.queueFilterIcon}>▤</span>
              {filters.source_type}
            </span>
          )}
        </div>
      )}

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
            className={`${styles.queueItem} ${event.id === selectedEventId ? styles.queueItemSelected : ''}`}
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
                {/* Context badges for workspace and agent */}
                {event.workspace && (
                  <span className={styles.queueBadge} data-badge="workspace" title={event.workspace}>
                    {getWorkspaceName(event.workspace)}
                  </span>
                )}
                {event.agent_id && (
                  <span className={styles.queueBadge} data-badge="agent" title={`Agent: ${event.agent_id}`}>
                    {event.agent_id}
                  </span>
                )}
                {/* Metrics badges - show as compact inline badges */}
                {event.turn_count !== undefined && event.turn_count > 0 && (
                  <span className={styles.metricBadge} title="Number of turns">
                    {event.turn_count}⟳
                  </span>
                )}
                {event.cost_usd !== undefined && event.cost_usd > 0 && (
                  <span className={styles.metricBadge} title="AI cost">
                    ${event.cost_usd < 0.01 ? event.cost_usd.toFixed(4) : event.cost_usd.toFixed(2)}
                  </span>
                )}
                {event.duration_ms !== undefined && event.duration_ms > 0 && (
                  <span className={styles.metricBadge} title="Duration">
                    {formatDuration(event.duration_ms)}
                  </span>
                )}
              </div>
              {/* Show directive as primary content, fallback to payload preview */}
              <div className={styles.queueMessage}>
                {event.directive || event.content || getPayloadPreview(event)}
              </div>
              {/* M-CHAT-HISTORY-DB: Chat preview for session events */}
              {event.type === 'session' && event.task_id && (
                <div className={styles.queueChatPreview}>
                  {chatPreviews[event.task_id]?.loading && (
                    <span className={styles.queueChatLoading}>💬 Loading...</span>
                  )}
                  {chatPreviews[event.task_id]?.firstUserMessage && (
                    <span className={styles.queueChatText} title="Initial user prompt">
                      💬 "{chatPreviews[event.task_id].firstUserMessage}"
                    </span>
                  )}
                  {chatPreviews[event.task_id]?.error && (
                    <span className={styles.queueChatError}>💬 {chatPreviews[event.task_id].error}</span>
                  )}
                  {!chatPreviews[event.task_id] && (
                    <span className={styles.queueChatIndicator}>💬</span>
                  )}
                </div>
              )}
              {/* Expandable full directive */}
              {event.directive_full && event.directive_full.length > 200 && (
                <button
                  className={styles.queueExpandBtn}
                  onClick={(e) => {
                    e.stopPropagation();
                    setExpandedEventId(expandedEventId === event.id ? null : event.id);
                  }}
                  title={expandedEventId === event.id ? 'Collapse' : 'Expand full directive'}
                >
                  {expandedEventId === event.id ? '▼' : '▶'} {event.directive_full.length} chars
                </button>
              )}
              {expandedEventId === event.id && event.directive_full && (
                <div className={styles.queueExpandedContent}>
                  {event.directive_full}
                </div>
              )}
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
