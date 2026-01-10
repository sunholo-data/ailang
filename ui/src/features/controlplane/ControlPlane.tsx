/**
 * AILANG Control Plane v4
 * AI Operations Mission Control Interface
 *
 * Aesthetic: Aerospace Mission Control meets Bloomberg Terminal
 */
import React, { useState, useEffect, useCallback, useMemo } from 'react';
import styles from './ControlPlane.module.css';
import {
  useHeatmapData,
  useTopologyData,
  useControlPlaneStats,
  useEventQueue,
  useTraceData,
  useBreakdownData,
} from './hooks';
import {
  ControlPlaneFilters,
  StatusFilter,
  hasActiveFilters,
  getFilterDescription,
  mergeFilters,
} from './types';

// Import extracted components
import {
  CommandBar,
  GlobalStats,
  AggregationNav,
  ActivityHeatmap,
  ExecHierarchy,
  MessageQueue,
  TraceWaterfall,
  DetailPanel,
  EventDetail,
  defaultTrustCapabilities,
} from './components';
import type {
  Agent,
  DateRange,
  DetailPanelState,
  HeatmapCell,
  EventMessage,
  TrustCapability,
  TopologyEdge,
  Span,
} from './components';
import type { EventType } from './components/MessageQueue';

/**
 * Find the agent path from an event through the topology.
 * Uses both source (from_agent) and target (to_inbox) to find best match.
 * Returns a Set of node IDs that should be highlighted.
 */
const findAgentPath = (
  sourceAgent: string,
  topologyEdges: TopologyEdge[],
  allNodeIds: string[],
  targetAgent?: string
): Set<string> => {
  const path = new Set<string>();
  if (allNodeIds.length === 0) return path;

  // Find a matching node - use exact match first, then partial
  let startNodeId: string | null = null;
  const candidates = [targetAgent, sourceAgent].filter(Boolean);

  for (const candidate of candidates) {
    if (!candidate) continue;
    const candidateLower = candidate.toLowerCase();

    // Exact match
    const exactMatch = allNodeIds.find(id => id === candidate || id.toLowerCase() === candidateLower);
    if (exactMatch) {
      startNodeId = exactMatch;
      break;
    }

    // Partial match
    const partialMatch = allNodeIds.find(id =>
      id.toLowerCase().includes(candidateLower) ||
      candidateLower.includes(id.toLowerCase())
    );
    if (partialMatch) {
      startNodeId = partialMatch;
      break;
    }
  }

  // If no match found, use the first node as a starting point
  if (!startNodeId && allNodeIds.length > 0) {
    startNodeId = allNodeIds[0];
  }

  if (!startNodeId) return path;

  // BFS from start node to find all connected nodes
  const visited = new Set<string>();
  const queue = [startNodeId];

  while (queue.length > 0) {
    const current = queue.shift()!;
    if (visited.has(current)) continue;
    visited.add(current);
    path.add(current);

    // Find connected nodes (both directions)
    topologyEdges.forEach(edge => {
      if (edge.source === current && !visited.has(edge.target)) {
        queue.push(edge.target);
      }
      if (edge.target === current && !visited.has(edge.source)) {
        queue.push(edge.source);
      }
    });
  }

  return path;
};

// Parse selectedLevel to ControlPlaneFilters
const parseSelectedLevelToFilters = (selectedLevel: string): ControlPlaneFilters => {
  if (!selectedLevel || selectedLevel === 'global') {
    return {};
  }
  // Format: "source-eval", "provider-claude", "model-claude-sonnet-4-5", "workspace-abc123"
  const parts = selectedLevel.split('-');
  if (parts.length < 2) return {};

  const filterType = parts[0];
  const filterValue = parts.slice(1).join('-'); // Handle model names with dashes

  switch (filterType) {
    case 'source':
      return { source_type: filterValue };
    case 'provider':
      return { provider: filterValue };
    case 'model':
      return { model: filterValue };
    case 'workspace':
      return { workspace: filterValue };
    default:
      return {};
  }
};

interface ControlPlaneProps {
  onSwitchToOldDashboard?: () => void;
}

// Main Component
export const ControlPlane: React.FC<ControlPlaneProps> = ({ onSwitchToOldDashboard }) => {
  const [searchQuery, setSearchQuery] = useState('');
  const [timeRange, setTimeRange] = useState('24h');
  const [statusFilter, setStatusFilter] = useState<StatusFilter>('all');
  const [selectedLevel, setSelectedLevel] = useState('global');
  const [trustCapabilities, setTrustCapabilities] = useState<TrustCapability[]>(defaultTrustCapabilities);
  const [theme, setTheme] = useState<'dark' | 'light'>('light');

  // Track time range selection from heatmap (separate from dimension filters)
  const [selectedDateRange, setSelectedDateRange] = useState<DateRange | null>(null);
  // Track event type filter (for MessageQueue)
  const [selectedEventTypes, setSelectedEventTypes] = useState<EventType[]>([]);
  // Span type filtering (Milestone 14) - generic filter for any span type
  // Default: hide api_request spans (LLM turns are noisy, tool calls are useful)
  const [hiddenSpanTypes, setHiddenSpanTypes] = useState<Set<string>>(new Set(['api_request']));

  // Toggle a span type in/out of the hidden set
  const toggleHiddenSpanType = useCallback((spanType: string) => {
    setHiddenSpanTypes(prev => {
      const next = new Set(prev);
      if (next.has(spanType)) {
        next.delete(spanType);
      } else {
        next.add(spanType);
      }
      return next;
    });
  }, []);

  // Agent Topology selection and highlighting state
  const [selectedTopologyNode, setSelectedTopologyNode] = useState<string | null>(null);
  const [highlightedPath, setHighlightedPath] = useState<Set<string>>(new Set());

  // Convert selectedLevel to dimension filters
  const dimensionFilters = useMemo(() => parseSelectedLevelToFilters(selectedLevel), [selectedLevel]);

  // Merge dimension filters with time range from heatmap selection, status, and search
  const filters = useMemo((): ControlPlaneFilters => {
    let merged = { ...dimensionFilters };
    if (selectedDateRange && selectedDateRange.start && selectedDateRange.end) {
      merged = mergeFilters(merged, {
        start_date: selectedDateRange.start,
        end_date: selectedDateRange.end,
      });
    }
    // Add status filter if not 'all'
    if (statusFilter && statusFilter !== 'all') {
      merged = mergeFilters(merged, { status: statusFilter });
    }
    // Add search filter if not empty
    if (searchQuery.trim()) {
      merged = mergeFilters(merged, { search: searchQuery.trim() });
    }
    return merged;
  }, [dimensionFilters, selectedDateRange, statusFilter, searchQuery]);

  const isFiltered = hasActiveFilters(filters);
  const filterDescription = getFilterDescription(filters);

  // Fetch real data from APIs - pass merged filters to all applicable hooks
  const { data: heatmapResponse } = useHeatmapData({ days: 90, filters });
  const { data: topologyData } = useTopologyData({ refreshInterval: 5000 });
  const { stats, loading: statsLoading } = useControlPlaneStats({ refreshInterval: 10000, filters });
  const { breakdowns, loading: breakdownLoading } = useBreakdownData({ refreshInterval: 30000, filters });
  const { events: liveEvents, loading: eventsLoading } = useEventQueue({ maxEvents: 50, filters });
  // Track selected event for trace correlation
  const [selectedEventTraceId, setSelectedEventTraceId] = useState<string | null>(null);
  // Detail panel state - must be defined before memos that use it
  const [detailPanel, setDetailPanel] = useState<DetailPanelState>({ type: null, id: null });

  const { spans: traceSpans, spansLoading, fetchSpansForTrace } = useTraceData({
    limit: 100  // Don't auto-fetch, we'll call fetchSpansForTrace manually with auto mode
  });

  // Transform data for components - NO MOCK FALLBACKS
  const heatmapData = useMemo(() => {
    return heatmapResponse?.cells || [];
  }, [heatmapResponse]);

  // Full topology data (unfiltered) - from message-based observed topology
  const messageBasedAgents = useMemo(() => {
    return topologyData?.agents || [];
  }, [topologyData]);

  // Edges with active status based on agent state
  const allEdges = useMemo(() => {
    if (!topologyData?.edges) return [];
    return topologyData.edges.map((e) => ({
      ...e,
      active: messageBasedAgents.some((a) => a.id === e.source && a.status === 'busy'),
    }));
  }, [topologyData, messageBasedAgents]);

  // All node IDs for path finding
  const allNodeIds = useMemo(() => {
    return messageBasedAgents.map(a => a.id);
  }, [messageBasedAgents]);

  // Agents - filter by highlighted path if a node is selected
  const agents = useMemo(() => {
    if (highlightedPath.size === 0) {
      return messageBasedAgents;
    }
    return messageBasedAgents.filter(a => highlightedPath.has(a.id));
  }, [messageBasedAgents, highlightedPath]);

  // Edges - filter by highlighted path if a node is selected
  const edges = useMemo(() => {
    if (highlightedPath.size === 0) {
      return allEdges;
    }
    return allEdges.filter(e => highlightedPath.has(e.source) && highlightedPath.has(e.target));
  }, [allEdges, highlightedPath]);

  const events = useMemo(() => {
    return liveEvents;
  }, [liveEvents]);

  const spans = useMemo(() => {
    return traceSpans;
  }, [traceSpans]);

  // Interactive state
  const [topologyExpanded, setTopologyExpanded] = useState(false);

  const handleTrustChange = useCallback((name: string, score: number) => {
    setTrustCapabilities(prev =>
      prev.map(cap => cap.name === name ? { ...cap, score } : cap)
    );
  }, []);

  const handleDateSelect = useCallback((range: DateRange) => {
    if (range.start === '' && range.end === '') {
      setSelectedDateRange(null);
    } else {
      setSelectedDateRange(range);
    }
  }, []);

  const handleCellClick = useCallback((cell: HeatmapCell) => {
    // Date selection now acts as a filter - no detail panel needed
    // The date range is already set by handleDateSelect on mouseDown
    // This callback is kept for potential future use (e.g., double-click behavior)
  }, []);

  const handleAgentClick = useCallback((agent: Agent) => {
    setDetailPanel({ type: 'agent', id: agent.id, data: agent });
    // Highlight the clicked agent and its path in topology (use full graph for path finding)
    const agentPath = findAgentPath(agent.id, allEdges, allNodeIds);
    setHighlightedPath(agentPath);
    setSelectedTopologyNode(agent.id);
  }, [allEdges, allNodeIds]);

  // Handle node selection in topology (can be used to clear selection)
  const handleNodeSelect = useCallback((nodeId: string | null) => {
    if (nodeId === null) {
      // Clicking background or "Back" button clears selection
      setHighlightedPath(new Set());
      setSelectedTopologyNode(null);
      setSelectedEventTraceId(null);
      // Close detail panel if showing an event
      if (detailPanel.type === 'event') {
        setDetailPanel({ type: null, id: null });
      }
    } else {
      // Clicking a node selects it and highlights path
      setSelectedTopologyNode(nodeId);
      const agentPath = findAgentPath(nodeId, allEdges, allNodeIds);
      setHighlightedPath(agentPath);
    }
  }, [allEdges, allNodeIds, detailPanel.type]);

  const handleEventClick = useCallback((event: EventMessage) => {
    setDetailPanel({ type: 'event', id: event.id, data: event });
    // Extract IDs from event metadata - try multiple approaches
    const metadata = event.metadata as Record<string, unknown> | undefined;

    // Priority order for finding trace/task ID:
    // 1. event.task_id (direct field - Claude Code events use span_id as task_id)
    // 2. metadata.task_id (direct span attribute - already in task-XXX format)
    // 3. metadata.parent_task_id (from coordinator - already in task-XXX format)
    // 4. metadata.correlation_id (message correlation)
    // 5. Construct task-{first8chars} from event.id
    let lookupId = event.task_id as string
      || metadata?.task_id as string
      || metadata?.parent_task_id as string
      || metadata?.correlation_id as string;

    // If no explicit task_id, construct from message ID
    // Coordinator creates tasks with ID format: task-{first8chars of message_id}
    if (!lookupId && event.id) {
      lookupId = `task-${event.id.substring(0, 8)}`;
    }

    setSelectedEventTraceId(lookupId);
    // Use 'auto' mode to try trace_id, then task_id, then task-prefixed
    fetchSpansForTrace(lookupId, 'auto');

    // Clear highlighting - the span-based topology will be shown automatically
    // once spans are loaded (via isShowingSpanTopology check in the memos)
    setHighlightedPath(new Set());
    setSelectedTopologyNode(null);
  }, [fetchSpansForTrace]);

  const closeDetailPanel = useCallback(() => {
    setDetailPanel({ type: null, id: null });
    setSelectedEventTraceId(null);
    // Clear topology highlighting
    setHighlightedPath(new Set());
    setSelectedTopologyNode(null);
  }, []);

  // Navigate to previous/next event
  const navigateEvent = useCallback((direction: 'prev' | 'next') => {
    if (detailPanel.type !== 'event' || !detailPanel.id || events.length === 0) return;

    const currentIndex = events.findIndex(e => e.id === detailPanel.id);
    if (currentIndex === -1) return;

    let newIndex: number;
    if (direction === 'prev') {
      newIndex = currentIndex > 0 ? currentIndex - 1 : events.length - 1;
    } else {
      newIndex = currentIndex < events.length - 1 ? currentIndex + 1 : 0;
    }

    const newEvent = events[newIndex];
    if (newEvent) {
      handleEventClick(newEvent);
    }
  }, [detailPanel.type, detailPanel.id, events, handleEventClick]);

  // Keyboard shortcuts
  useEffect(() => {
    const handleKeyDown = (e: KeyboardEvent) => {
      // Cmd/Ctrl+K for search
      if ((e.metaKey || e.ctrlKey) && e.key === 'k') {
        e.preventDefault();
        document.querySelector<HTMLInputElement>(`.${styles.searchInput}`)?.focus();
      }
      // Escape to close detail panel
      if (e.key === 'Escape') {
        closeDetailPanel();
        if (topologyExpanded) setTopologyExpanded(false);
      }
      // Left/Right arrows to navigate events (when viewing an event)
      if (e.key === 'ArrowLeft') {
        e.preventDefault();
        navigateEvent('prev');
      }
      if (e.key === 'ArrowRight') {
        e.preventDefault();
        navigateEvent('next');
      }
    };
    window.addEventListener('keydown', handleKeyDown);
    return () => window.removeEventListener('keydown', handleKeyDown);
  }, [closeDetailPanel, topologyExpanded, navigateEvent]);

  const toggleTheme = useCallback(() => {
    setTheme((prev) => (prev === 'dark' ? 'light' : 'dark'));
  }, []);

  return (
    <div className={`${styles.controlPlane} ${theme === 'light' ? styles.controlPlaneLight : ''}`}>
      {/* Background effects */}
      <div className={styles.bgContours} />
      <div className={styles.bgScanlines} />

      {/* Header */}
      <header className={styles.header}>
        <div className={styles.headerBrand}>
          <span className={styles.brandIcon}>◎</span>
          <h1 className={styles.brandTitle}>AILANG</h1>
          <span className={styles.brandSub}>Control Plane</span>
          <button className={styles.themeToggle} onClick={toggleTheme} title={`Switch to ${theme === 'dark' ? 'light' : 'dark'} theme`}>
            <span className={styles.themeToggleIcon}>{theme === 'dark' ? '☀️' : '🌙'}</span>
            <span className={styles.themeToggleLabel}>{theme === 'dark' ? 'Light' : 'Dark'}</span>
          </button>
        </div>
        <GlobalStats
          stats={stats}
          loading={statsLoading}
          isFiltered={isFiltered}
          filterDescription={filterDescription}
          onClearFilter={() => {
            // Clear all filters
            setSelectedLevel('global');
            setSelectedDateRange(null);
            setStatusFilter('all');
            setSearchQuery('');
          }}
        />
      </header>

      {/* Command Bar */}
      <CommandBar
        searchQuery={searchQuery}
        onSearchChange={setSearchQuery}
        timeRange={timeRange}
        onTimeRangeChange={setTimeRange}
        statusFilter={statusFilter}
        onStatusChange={setStatusFilter}
      />

      {/* Main Layout - Event Queue LEFT, Aggregations RIGHT */}
      <div className={styles.mainLayout}>
        {/* Left Sidebar - Event Queue (Input Feed) */}
        <aside className={styles.eventPanel}>
          <MessageQueue
            events={events}
            onEventClick={handleEventClick}
            loading={eventsLoading}
            selectedDateRange={selectedDateRange}
            onDateRangeChange={setSelectedDateRange}
            selectedTypes={selectedEventTypes}
            onTypeFilterChange={setSelectedEventTypes}
          />
        </aside>

        {/* Main Canvas */}
        <main className={`${styles.mainCanvas} ${topologyExpanded ? styles.canvasWithExpanded : ''}`}>
          <div className={styles.canvasRow}>
            <ActivityHeatmap
              data={heatmapData}
              selectedRange={selectedDateRange}
              onDateSelect={handleDateSelect}
              onCellClick={handleCellClick}
            />
          </div>
          <div className={`${styles.canvasRow} ${topologyExpanded ? styles.canvasRowExpanded : ''}`}>
            <ExecHierarchy
              isExpanded={topologyExpanded}
              onToggleExpand={() => setTopologyExpanded(!topologyExpanded)}
              selectedNodeId={selectedEventTraceId}
              spans={spans}
              loading={spansLoading}
              filterCriteria={{
                dateRange: selectedDateRange,
                eventTypes: selectedEventTypes.length > 0 ? selectedEventTypes : undefined,
              }}
              hiddenSpanTypes={hiddenSpanTypes}
              onToggleSpanType={toggleHiddenSpanType}
            />
          </div>
          {!topologyExpanded && (
            <div className={styles.canvasRow}>
              {detailPanel.type === 'event' && detailPanel.data ? (
                <EventDetail
                  event={detailPanel.data as EventMessage}
                  spans={spans}
                  traceId={selectedEventTraceId}
                  loading={spansLoading}
                  onClose={closeDetailPanel}
                  onNavigate={navigateEvent}
                  currentIndex={events.findIndex(e => e.id === detailPanel.id)}
                  totalEvents={events.length}
                />
              ) : (
                <TraceWaterfall
                  spans={spans}
                  selectedTraceId={selectedEventTraceId}
                  loading={spansLoading}
                  hiddenSpanTypes={hiddenSpanTypes}
                  onToggleSpanType={toggleHiddenSpanType}
                />
              )}
            </div>
          )}
        </main>

        {/* Right Sidebar - Aggregations (Analysis/Metrics) */}
        <aside className={styles.aggregationPanel}>
          <AggregationNav
            selectedLevel={selectedLevel}
            onSelectLevel={setSelectedLevel}
            stats={stats ? {
              totalTasks: stats.totalTasks,
              totalCost: parseFloat(stats.totalCost.replace('$', '')),
              activeAgents: stats.activeAgents,
              pendingApprovals: stats.pendingApprovals,
            } : null}
            breakdowns={breakdowns}
            loading={statsLoading || breakdownLoading}
          />
        </aside>
      </div>

      {/* Detail Panel Overlay - only for agent and date, events show inline */}
      {detailPanel.type && detailPanel.type !== 'event' && (
        <>
          <div className={styles.detailOverlay} onClick={closeDetailPanel} />
          <DetailPanel
            state={detailPanel}
            onClose={closeDetailPanel}
            agents={agents}
            trustCapabilities={trustCapabilities}
            onTrustChange={handleTrustChange}
          />
        </>
      )}

      {/* Footer */}
      <footer className={styles.footer}>
        <span className={styles.footerVersion}>v0.6.4</span>
        <span className={styles.footerStatus}>
          <span className={styles.footerDot} />
          Coordinator Active
        </span>
        {onSwitchToOldDashboard && (
          <button className={styles.oldDashboardLink} onClick={onSwitchToOldDashboard}>
            View Old Dashboard →
          </button>
        )}
        <span className={styles.footerTime}>{new Date().toISOString()}</span>
      </footer>
    </div>
  );
};

export default ControlPlane;
