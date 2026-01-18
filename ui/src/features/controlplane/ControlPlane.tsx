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
  useOutliersAnalysis,
} from './hooks';
import {
  ControlPlaneFilters,
  StatusFilter,
  SortField,
  SortOrder,
  mergeFilters,
} from './types';

// Import extracted components
import {
  AggregationNav,
  ActivityHeatmap,
  ExecHierarchy,
  MessageQueue,
  DetailPanel,
  EventDetail,
  VisualizationPanel,
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
import { useObservatoryWs, useApprovals } from '../../hooks/useObservatory';
import { useBudgetStatus } from '../../hooks/useBudgetStatus';
import { ApprovalDetailModal, ApprovalData } from '../approvals/ApprovalDetailModal';
import { HeaderStats } from '../../components/HeaderStats';
import type { Approval } from '../../types';

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

// Multi-filter type: maps dimension to selected value
// e.g., { workspace: 'ailang', source: 'eval', provider: 'claude' }
type SelectedFilters = Record<string, string>;

// Parse multi-filter object to ControlPlaneFilters
const parseSelectedFiltersToFilters = (selected: SelectedFilters): ControlPlaneFilters => {
  return {
    source_type: selected.source,
    provider: selected.provider,
    model: selected.model,
    workspace: selected.workspace,
  };
};

// Main Component
export const ControlPlane: React.FC = () => {
  const [searchQuery, setSearchQuery] = useState('');
  const [timeRange, setTimeRange] = useState('24h');
  const [statusFilter, setStatusFilter] = useState<StatusFilter>('all');
  const [selectedFilters, setSelectedFilters] = useState<SelectedFilters>({});
  const [trustCapabilities, setTrustCapabilities] = useState<TrustCapability[]>(defaultTrustCapabilities);
  const [theme, setTheme] = useState<'dark' | 'light'>('light');

  // Track time range selection from heatmap (separate from dimension filters)
  const [selectedDateRange, setSelectedDateRange] = useState<DateRange | null>(null);
  // Track event type filter (for MessageQueue)
  const [selectedEventTypes, setSelectedEventTypes] = useState<EventType[]>([]);
  // Track sort state for event queue
  const [sortBy, setSortBy] = useState<SortField>('timestamp');
  const [sortOrder, setSortOrder] = useState<SortOrder>('desc');
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

  // Event-to-aggregation highlighting state (when clicking an event, highlight related items in sidebar)
  const [highlightedAggItems, setHighlightedAggItems] = useState<{
    workspace?: string;
    provider?: string;
    model?: string;
    source_type?: string;
  } | null>(null);

  // Convert selectedFilters to ControlPlaneFilters
  const dimensionFilters = useMemo(() => parseSelectedFiltersToFilters(selectedFilters), [selectedFilters]);

  // Handler to toggle a filter dimension (for AggregationNav)
  const handleFilterToggle = useCallback((dimension: string, value: string) => {
    setSelectedFilters(prev => {
      const newFilters = { ...prev };
      if (newFilters[dimension] === value) {
        // Same value clicked - deselect this dimension
        delete newFilters[dimension];
      } else {
        // Different value - select this value for this dimension
        newFilters[dimension] = value;
      }
      return newFilters;
    });
  }, []);

  // Clear all filters (for "Global" click or "Clear All" button)
  const handleClearFilters = useCallback(() => {
    setSelectedFilters({});
  }, []);

  // Handle sort changes from MessageQueue
  const handleSortChange = useCallback((sort: SortField, order: SortOrder) => {
    setSortBy(sort);
    setSortOrder(order);
  }, []);

  // Merge dimension filters with time range from heatmap selection, status, search, and sort
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
    // Add sort parameters
    merged = mergeFilters(merged, { sort: sortBy, order: sortOrder });
    return merged;
  }, [dimensionFilters, selectedDateRange, statusFilter, searchQuery, sortBy, sortOrder]);

  // Fetch real data from APIs - pass merged filters to all applicable hooks
  // Use grid format for server-side date calculations (removes ~80 lines of client-side logic)
  const { gridData, data: heatmapResponse } = useHeatmapData({ days: 90, filters, format: 'grid' });
  const { data: topologyData } = useTopologyData({ refreshInterval: 5000 });
  const { stats, loading: statsLoading } = useControlPlaneStats({ refreshInterval: 10000, filters });
  const { breakdowns, loading: breakdownLoading } = useBreakdownData({ refreshInterval: 30000, filters });
  const { budget } = useBudgetStatus(30000, filters);
  const { events: liveEvents, loading: eventsLoading } = useEventQueue({ maxEvents: 50, filters });
  // WebSocket connection status for header indicator
  const { isConnected, connectionState, lastEventTime } = useObservatoryWs({});
  // Approvals data and modal state
  const { approvals, loading: approvalsLoading, refresh: refreshApprovals, approveApproval, rejectApproval } = useApprovals({ status: 'pending' });
  const [selectedApproval, setSelectedApproval] = useState<Approval | null>(null);
  const [approvalModalOpen, setApprovalModalOpen] = useState(false);
  const [approvalDropdownOpen, setApprovalDropdownOpen] = useState(false);
  // Track selected event for trace correlation
  const [selectedEventTraceId, setSelectedEventTraceId] = useState<string | null>(null);
  // Track highlighted span ID (for outlier click-to-highlight)
  const [highlightedSpanId, setHighlightedSpanId] = useState<string | null>(null);
  // Detail panel state - must be defined before memos that use it
  const [detailPanel, setDetailPanel] = useState<DetailPanelState>({ type: null, id: null });

  const { spans: traceSpans, spansLoading, fetchSpansForTrace } = useTraceData({
    limit: 100  // Don't auto-fetch, we'll call fetchSpansForTrace manually with auto mode
  });

  // Fetch outliers analysis for selected task (used in EventDetail)
  const { data: outliersData, loading: outliersLoading } = useOutliersAnalysis(
    selectedEventTraceId,
    { showRate: true, limit: 5 }
  );

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

  // Approval handlers
  const handleApprovalClick = useCallback((approval: Approval) => {
    setSelectedApproval(approval);
    setApprovalModalOpen(true);
  }, []);

  const handleApprovalModalClose = useCallback(() => {
    setApprovalModalOpen(false);
    setSelectedApproval(null);
  }, []);

  const handleApprovalApprove = useCallback(async (id: string) => {
    await approveApproval(id);
    handleApprovalModalClose();
    refreshApprovals();
  }, [approveApproval, refreshApprovals, handleApprovalModalClose]);

  const handleApprovalReject = useCallback(async (id: string, notes: string) => {
    await rejectApproval(id, notes);
    handleApprovalModalClose();
    refreshApprovals();
  }, [rejectApproval, refreshApprovals, handleApprovalModalClose]);

  const handleApprovalCancel = useCallback(async (id: string, notes?: string) => {
    await rejectApproval(id, notes || '', true); // permanent=true for cancel
    handleApprovalModalClose();
    refreshApprovals();
  }, [rejectApproval, refreshApprovals, handleApprovalModalClose]);

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

    // Highlight related aggregation items in sidebar based on event metadata
    setHighlightedAggItems({
      workspace: metadata?.workspace as string | undefined,
      provider: metadata?.provider as string | undefined,
      model: metadata?.model as string | undefined,
      source_type: metadata?.source_type as string | undefined,
    });
  }, [fetchSpansForTrace]);

  // Handler for task selection from Evolution chart
  const handleTaskSelect = useCallback((taskId: string) => {
    // Try to find matching event in the events list
    const matchingEvent = events.find((event) => {
      // Check direct task_id field
      if (event.task_id === taskId) return true;
      // Check metadata fields
      const metadata = event.metadata as Record<string, unknown> | undefined;
      if (metadata?.task_id === taskId) return true;
      if (metadata?.parent_task_id === taskId) return true;
      // Check if event.id matches the short task ID format
      if (taskId.includes('/') && event.id?.startsWith(taskId.split('/')[1])) return true;
      return false;
    });

    if (matchingEvent) {
      // Found matching event, use the full event click handler
      handleEventClick(matchingEvent);
    } else {
      // No matching event found, but still show spans for this task
      setSelectedEventTraceId(taskId);
      fetchSpansForTrace(taskId, 'auto');
      // Clear detail panel to show "Select an event" state
      setDetailPanel({ type: null, id: null });
    }
  }, [events, handleEventClick, fetchSpansForTrace]);

  const closeDetailPanel = useCallback(() => {
    setDetailPanel({ type: null, id: null });
    setSelectedEventTraceId(null);
    // Clear topology highlighting
    setHighlightedPath(new Set());
    setSelectedTopologyNode(null);
    // Clear aggregation highlighting
    setHighlightedAggItems(null);
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
          <img
            src="https://ailang.sunholo.com/img/logo.png"
            alt="Sunholo"
            className={styles.brandLogo}
          />
          <div className={styles.brandText}>
            <h1 className={styles.brandTitle}>Observatory</h1>
            <span className={styles.brandSub}>Control Plane</span>
          </div>
        </div>
        <div className={styles.headerActions}>
          {/* Dashboard Stats */}
          <HeaderStats />

          {/* Pending Approvals Dropdown */}
          <div className={styles.approvalsDropdownWrapper}>
            <button
              className={`${styles.approvalsBtn} ${approvals.length > 0 ? styles.approvalsBtnActive : ''}`}
              onClick={() => setApprovalDropdownOpen(!approvalDropdownOpen)}
              title={`${approvals.length} pending approval${approvals.length !== 1 ? 's' : ''}`}
            >
              <span className={styles.approvalsBtnIcon}>⏳</span>
              <span className={styles.approvalsBtnLabel}>Approvals</span>
              {approvals.length > 0 && (
                <span className={styles.approvalsBtnBadge}>{approvals.length}</span>
              )}
            </button>
            {approvalDropdownOpen && (
              <div className={styles.approvalsDropdown}>
                <div className={styles.approvalsDropdownHeader}>
                  <span>Pending Approvals ({approvals.length})</span>
                  <button
                    className={styles.approvalsDropdownClose}
                    onClick={() => setApprovalDropdownOpen(false)}
                  >×</button>
                </div>
                {approvals.length === 0 ? (
                  <div className={styles.approvalsDropdownEmpty}>No pending approvals</div>
                ) : (
                  <div className={styles.approvalsDropdownList}>
                    {approvals.map((approval) => (
                      <button
                        key={approval.id}
                        className={styles.approvalsDropdownItem}
                        onClick={() => {
                          handleApprovalClick(approval);
                          setApprovalDropdownOpen(false);
                        }}
                      >
                        <span className={`${styles.approvalsDropdownType} ${styles[`type${(approval.request_type || 'merge').charAt(0).toUpperCase() + (approval.request_type || 'merge').slice(1)}`]}`}>
                          {approval.request_type || 'merge'}
                        </span>
                        <div className={styles.approvalsDropdownInfo}>
                          <span className={styles.approvalsDropdownTitle}>{approval.thread_title || approval.summary || 'Approval Request'}</span>
                          <span className={styles.approvalsDropdownMeta}>
                            <span className={styles.approvalsDropdownTask}>{approval.task_id}</span>
                            <span className={styles.approvalsDropdownDate}>{new Date(approval.created_at).toLocaleDateString()} {new Date(approval.created_at).toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' })}</span>
                          </span>
                        </div>
                      </button>
                    ))}
                  </div>
                )}
              </div>
            )}
          </div>

          {/* WebSocket Connection Status */}
          <div
            className={`${styles.connectionStatus} ${isConnected ? styles.connectionStatusLive : styles.connectionStatusOffline}`}
            title={`WebSocket ${connectionState}${lastEventTime ? ` • Last event: ${lastEventTime.toLocaleTimeString()}` : ''}`}
          >
            <span className={styles.connectionDot} />
            <span className={styles.connectionLabel}>{isConnected ? 'LIVE' : 'OFFLINE'}</span>
          </div>

          <button className={styles.themeToggle} onClick={toggleTheme} title={`Switch to ${theme === 'dark' ? 'light' : 'dark'} theme`}>
            <span className={styles.themeToggleIcon}>{theme === 'dark' ? '☀️' : '🌙'}</span>
          </button>
        </div>
      </header>

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
            sortBy={sortBy}
            sortOrder={sortOrder}
            onSortChange={handleSortChange}
            filters={filters}
            selectedEventId={detailPanel.type === 'event' ? detailPanel.id : null}
          />
        </aside>

        {/* Main Canvas */}
        <main className={`${styles.mainCanvas} ${topologyExpanded ? styles.canvasWithExpanded : ''}`}>
          {/* Top Row: Visualization Panel + Event Detail (always side by side) */}
          <div className={`${styles.canvasRow} ${styles.canvasRowSplit}`}>
            <VisualizationPanel
              filters={filters}
              heatmapData={heatmapData}
              heatmapGridData={gridData}
              selectedDateRange={selectedDateRange}
              onDateSelect={handleDateSelect}
              onHeatmapCellClick={handleCellClick}
              onClearFilter={(key) => {
                // Handle different filter types
                if (key === 'start_date' || key === 'end_date') {
                  setSelectedDateRange(null);
                } else if (key === 'status') {
                  setStatusFilter('all');
                } else if (key === 'search') {
                  setSearchQuery('');
                } else {
                  // Dimension filters (provider, model, workspace, source_type)
                  // Note: selectedFilters uses 'source' but filters use 'source_type'
                  const internalKey = key === 'source_type' ? 'source' : key;
                  setSelectedFilters(prev => {
                    const newFilters = { ...prev };
                    delete newFilters[internalKey];
                    return newFilters;
                  });
                }
              }}
              onClearAllFilters={handleClearFilters}
              onSetFilter={handleFilterToggle}
              onTaskSelect={handleTaskSelect}
            />
            {/* Event Detail Panel - always visible, shows placeholder when no event selected */}
            {!topologyExpanded && (
              <EventDetail
                event={detailPanel.type === 'event' ? detailPanel.data as EventMessage : null}
                traceId={selectedEventTraceId}
                loading={spansLoading}
                onClose={closeDetailPanel}
                onNavigate={navigateEvent}
                currentIndex={events.findIndex(e => e.id === detailPanel.id)}
                totalEvents={events.length}
                outliers={outliersData}
                outliersLoading={outliersLoading}
                onOutlierClick={(spanId) => setHighlightedSpanId(spanId)}
              />
            )}
          </div>
          <div className={`${styles.canvasRow} ${topologyExpanded ? styles.canvasRowExpanded : ''}`}>
            <ExecHierarchy
              isExpanded={topologyExpanded}
              onToggleExpand={() => setTopologyExpanded(!topologyExpanded)}
              selectedNodeId={selectedEventTraceId}
              highlightedSpanId={highlightedSpanId}
              onClearHighlight={() => setHighlightedSpanId(null)}
              spans={spans}
              loading={spansLoading}
              filterCriteria={{
                dateRange: selectedDateRange,
                eventTypes: selectedEventTypes.length > 0 ? selectedEventTypes : undefined,
                provider: filters.provider,
                model: filters.model,
                workspace: filters.workspace,
                source_type: filters.source_type,
              }}
              hiddenSpanTypes={hiddenSpanTypes}
              onToggleSpanType={toggleHiddenSpanType}
              filters={filters}
            />
          </div>
        </main>

        {/* Right Sidebar - Aggregations (Analysis/Metrics) */}
        <aside className={styles.aggregationPanel}>
          <AggregationNav
            selectedFilters={selectedFilters}
            onFilterToggle={handleFilterToggle}
            onClearFilters={handleClearFilters}
            stats={stats ? {
              totalTasks: stats.totalTasks,
              totalCost: parseFloat(stats.totalCost.replace('$', '')),
              activeAgents: stats.activeAgents,
              pendingApprovals: stats.pendingApprovals,
            } : null}
            breakdowns={breakdowns}
            loading={statsLoading || breakdownLoading}
            filters={filters}
            highlightedItems={highlightedAggItems}
            burnRate={budget?.burnRate}
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
        <span className={styles.footerTime}>{new Date().toISOString()}</span>
      </footer>

      {/* Approval Detail Modal */}
      {approvalModalOpen && selectedApproval && (
        <ApprovalDetailModal
          isOpen={approvalModalOpen}
          approval={selectedApproval as ApprovalData}
          approvals={approvals as ApprovalData[]}
          onClose={handleApprovalModalClose}
          onApprove={handleApprovalApprove}
          onReject={handleApprovalReject}
          onCancel={handleApprovalCancel}
          onNavigate={(approval) => setSelectedApproval(approval as Approval)}
        />
      )}
    </div>
  );
};

export default ControlPlane;
