/**
 * ExecHierarchy - Main container for the 4-level exec hierarchy view
 * Provides toggle between tree and graph views
 * Shows: Messages -> Execs -> Turns -> Tool Uses
 *
 * Uses the SAME spans data as TraceWaterfall for consistency.
 *
 * PR 2 Refactoring (M-DASHBOARD-SIMPLIFICATION):
 * - Smart label logic extracted to utils/smartLabel.ts
 * - ID service utilities extracted to utils/idService.ts
 * - State management available via hooks/useExecHierarchyState.ts
 */
import React, { useState, useCallback, useMemo, useEffect } from 'react';
import type { HierarchyNode, ViewMode, ExecHierarchyProps, Span, CoordinatorViewMode, FilterCriteria, ExecHierarchyNode } from './types';
import { ExecHierarchyTree } from './ExecHierarchyTree';
import { ExecHierarchyGraph } from './ExecHierarchyGraph';
import { TaskHierarchyGraph } from './TaskHierarchyGraph';
import { ChatHistory } from './ChatHistory';
import { EvolutionTree } from './EvolutionTree';
import { CliCommandHint } from '../CliCommandHint';
import { TraceWaterfall } from '../TraceWaterfall';
import { useObservatoryWs, Span as ObsSpan } from '../../../../hooks/useObservatory';
import styles from './ExecHierarchy.module.css';

// Import extracted components (PR 3 - M-DASHBOARD-SIMPLIFICATION)
import { ExecHierarchyToolbar } from './ExecHierarchyToolbar';
import { ExecHierarchyPopover } from './ExecHierarchyPopover';

// Import extracted utilities (PR 2 - M-DASHBOARD-SIMPLIFICATION)
// Note: getNodeIcon and formatDuration are now used by ExecHierarchyPopover directly
import {
  getNodeType,
  getSemanticType,
  getNodeStatus,
  getSmartLabel,
  extractMetrics,
  getTurnNumber,
} from '../../utils/smartLabel';
import {
  extractCoordinatorContext,
  collectSpanTypes as collectUniqueSpanTypes,
  collectTaskIds as collectSpanTaskIds,
} from '../../utils/idService';

// Popover max dimensions - compact design
const POPOVER_MAX_WIDTH = 420;
const POPOVER_MAX_HEIGHT = 480;
const POPOVER_MARGIN = 16;

// Transform Span[] to HierarchyNode[] (recursive)
function spanToHierarchyNode(span: Span, siblingIndex?: number, depth: number = 0): HierarchyNode {
  const nodeType = getNodeType(span.name);
  const isTurn = nodeType === 'turn';
  const turnNumber = isTurn ? getTurnNumber(span, siblingIndex) : undefined;
  const semanticType = getSemanticType(span.name);

  // Use smart label extraction
  const label = getSmartLabel(span);

  // Extract metrics
  const metrics = extractMetrics(span);

  // Extract coordinator context
  const coordContext = extractCoordinatorContext(span);

  // Transform children first so we can count them
  const children = span.children?.map((child, idx) => spanToHierarchyNode(child, idx, depth + 1));
  const hasChildren = children && children.length > 0;

  const node: HierarchyNode = {
    id: span.id,
    type: nodeType,
    label,
    status: getNodeStatus(span),
    durationMs: span.durationMs,
    startTime: span.startMs ? new Date(span.startMs).toISOString() : undefined,
    turnNumber,
    _span: span,  // Preserve original span for popover
    children,
    // Collapsibility
    isCollapsible: hasChildren,
    childCount: hasChildren ? countDescendants({ children } as HierarchyNode) + children.length : 0,
    // Metrics
    cost: metrics.cost,
    tokensIn: metrics.tokensIn,
    tokensOut: metrics.tokensOut,
    provider: metrics.provider,
    semanticType,
    // Coordinator context
    taskId: coordContext.taskId,
    parentTaskId: coordContext.parentTaskId,
    agentId: coordContext.agentId,
    approvalStatus: coordContext.approvalStatus,
    approvalId: coordContext.approvalId,
  };

  return node;
}

// Transform spans to hierarchy nodes
function transformSpansToNodes(spans: Span[]): HierarchyNode[] {
  return spans.map((span, idx) => spanToHierarchyNode(span, idx));
}

// Extract nested coordinator tasks for breakout view
// Returns all coordinator tasks as separate root nodes with parent links preserved
function breakoutCoordinatorTasks(nodes: HierarchyNode[]): HierarchyNode[] {
  const result: HierarchyNode[] = [];
  const coordinatorNodes: HierarchyNode[] = [];

  // Collect all coordinator nodes and their children
  const collectCoordinatorNodes = (nodeList: HierarchyNode[], parentTaskId?: string) => {
    for (const node of nodeList) {
      // Check if this is a coordinator task (semanticType or span name)
      const isCoordinator = node.semanticType === 'coordinator' ||
        node._span?.name === 'coordinator.task.execute';

      if (isCoordinator) {
        // Create a copy with parent reference preserved
        const coordNode: HierarchyNode = {
          ...node,
          parentTaskId: parentTaskId || node.parentTaskId,
        };

        // Recursively collect nested coordinators, then remove them from children
        if (node.children && node.children.length > 0) {
          const nonCoordChildren: HierarchyNode[] = [];
          for (const child of node.children) {
            const childIsCoord = child.semanticType === 'coordinator' ||
              child._span?.name === 'coordinator.task.execute';
            if (childIsCoord) {
              // This child is a coordinator - collect it separately
              collectCoordinatorNodes([child], coordNode.taskId || coordNode.id);
            } else {
              nonCoordChildren.push(child);
            }
          }
          coordNode.children = nonCoordChildren;
          coordNode.childCount = nonCoordChildren.length > 0 ?
            countDescendants(coordNode) + nonCoordChildren.length : 0;
        }

        coordinatorNodes.push(coordNode);
      } else {
        // Not a coordinator - keep in place but check children for nested coordinators
        const nodeCopy = { ...node };
        if (node.children && node.children.length > 0) {
          const nonCoordChildren: HierarchyNode[] = [];
          for (const child of node.children) {
            const childIsCoord = child.semanticType === 'coordinator' ||
              child._span?.name === 'coordinator.task.execute';
            if (childIsCoord) {
              collectCoordinatorNodes([child], parentTaskId);
            } else {
              nonCoordChildren.push(child);
            }
          }
          nodeCopy.children = nonCoordChildren;
        }
        result.push(nodeCopy);
      }
    }
  };

  collectCoordinatorNodes(nodes);

  // Combine: non-coordinator roots + all coordinator nodes as separate roots
  return [...result, ...coordinatorNodes];
}

// Count total descendants recursively
function countDescendants(node: HierarchyNode): number {
  if (!node.children || node.children.length === 0) return 0;
  return node.children.reduce((sum, child) => sum + 1 + countDescendants(child), 0);
}

// Determine default expanded state based on node type and depth
function getDefaultExpandedState(node: HierarchyNode, depth: number): boolean {
  // Always expand top-level coordinator/exec nodes
  if (depth === 0) return true;
  // Expand turns (they're the main navigation units)
  if (node.type === 'turn') return true;
  // Collapse tool_use and deep nodes by default (reduce noise)
  if (node.type === 'tool_use' || depth > 2) return false;
  return true;
}

// Check if a node matches the filter criteria
function nodeMatchesFilter(node: HierarchyNode, criteria: FilterCriteria | undefined): boolean {
  if (!criteria) return true;

  // Check date range filter - convert string dates to Date objects for reliable comparison
  if (criteria.dateRange && node.startTime) {
    const nodeDate = new Date(node.startTime);
    const startDate = new Date(criteria.dateRange.start + 'T00:00:00');
    const endDate = new Date(criteria.dateRange.end + 'T23:59:59');
    if (nodeDate < startDate || nodeDate > endDate) {
      return false;
    }
  }

  // Check event type filter (map node.type to event types)
  if (criteria.eventTypes && criteria.eventTypes.length > 0) {
    // Map node types to event types
    const typeMapping: Record<string, string[]> = {
      'exec': ['task_start', 'task_complete', 'task_error'],
      'turn': ['task_start', 'task_complete'],
      'tool_use': ['task_start', 'task_complete'],
      'message': ['message', 'handoff', 'approval'],
    };
    const mappedTypes = typeMapping[node.type] || [];
    const hasMatchingType = mappedTypes.some(t => criteria.eventTypes!.includes(t));
    if (!hasMatchingType) {
      return false;
    }
  }

  // Check provider filter
  if (criteria.provider && node.provider) {
    if (node.provider !== criteria.provider) {
      return false;
    }
  }

  // Check model filter (check node.provider for model or span attributes)
  if (criteria.model) {
    // Model might be stored in span attributes or could be derived from provider
    const nodeModel = node._span?.attributes?.model || node._span?.attributes?.['llm.model'];
    if (nodeModel && nodeModel !== criteria.model) {
      return false;
    }
  }

  // Check workspace filter
  if (criteria.workspace) {
    // Workspace might be on the node directly or in span attributes
    const nodeWorkspace = (node as ExecHierarchyNode).workspace ||
                          node._span?.attributes?.workspace ||
                          node._span?.attributes?.['ailang.workspace'];
    if (nodeWorkspace && nodeWorkspace !== criteria.workspace) {
      return false;
    }
  }

  // Check source_type filter
  if (criteria.source_type) {
    const nodeSourceType = node._span?.attributes?.source_type ||
                           node._span?.attributes?.['ailang.source_type'];
    if (nodeSourceType && nodeSourceType !== criteria.source_type) {
      return false;
    }
  }

  return true;
}

// Apply filter criteria to nodes recursively (marks nodes as isFiltered, does NOT hide them)
function applyFilterToNodes(nodes: HierarchyNode[], criteria: FilterCriteria | undefined): HierarchyNode[] {
  const hasFilters = criteria && (
    criteria.dateRange ||
    (criteria.eventTypes && criteria.eventTypes.length > 0) ||
    criteria.provider ||
    criteria.model ||
    criteria.workspace ||
    criteria.source_type
  );
  if (!hasFilters) {
    // No filtering active - return nodes as-is
    return nodes;
  }

  const applyFilter = (node: HierarchyNode): HierarchyNode => {
    const isFiltered = !nodeMatchesFilter(node, criteria);
    const filteredChildren = node.children?.map(applyFilter);

    return {
      ...node,
      isFiltered,
      children: filteredChildren,
    };
  };

  return nodes.map(applyFilter);
}

export const ExecHierarchy: React.FC<ExecHierarchyProps> = ({
  isExpanded,
  onToggleExpand,
  onNodeClick,
  selectedNodeId,
  isEmpty: propsIsEmpty,
  spans,
  loading,
  filterCriteria,
  hiddenSpanTypes: propsHiddenSpanTypes,
  onToggleSpanType,
  filters,
  highlightedSpanId,
  onClearHighlight,
  theme,
}) => {
  // Default to graph view (ReactFlow)
  const [viewMode, setViewMode] = useState<ViewMode>('graph');

  // Coordinator view mode: nested (default) or breakout
  const [coordViewMode, setCoordViewMode] = useState<CoordinatorViewMode>('nested');

  // Span type filtering (Milestone 14) - generic filter for any span type
  // Use props if provided (lifted state), otherwise use internal state
  const [internalHiddenTypes, setInternalHiddenTypes] = useState<Set<string>>(new Set(['api_request']));
  const hiddenSpanTypes = propsHiddenSpanTypes !== undefined ? propsHiddenSpanTypes : internalHiddenTypes;
  const toggleSpanType = onToggleSpanType || ((spanType: string) => {
    setInternalHiddenTypes(prev => {
      const next = new Set(prev);
      if (next.has(spanType)) {
        next.delete(spanType);
      } else {
        next.add(spanType);
      }
      return next;
    });
  });

  // Span type filter dropdown visibility
  const [showSpanTypeFilter, setShowSpanTypeFilter] = useState(false);

  // Collapsibility state: Set of node IDs that are expanded
  // START COLLAPSED - empty set means only root nodes visible
  const [expandedNodes, setExpandedNodes] = useState<Set<string>>(new Set());

  // Track expand/collapse changes for recentering
  const [expandChangeCounter, setExpandChangeCounter] = useState(0);

  // Display limit state - start with 100 nodes, can load more
  const DEFAULT_DISPLAY_LIMIT = 100;
  const LOAD_MORE_INCREMENT = 100;
  const [displayLimit, setDisplayLimit] = useState(DEFAULT_DISPLAY_LIMIT);

  // Reverse order toggle - show newest first when true
  const [reverseOrder, setReverseOrder] = useState(false);

  // Real-time updates via WebSocket
  const [lastUpdate, setLastUpdate] = useState<Date | null>(null);
  const { isConnected, connectionState, lastEventTime } = useObservatoryWs({
    onSpanCreated: useCallback((span: ObsSpan) => {
      // Update last event time for visual feedback
      setLastUpdate(new Date());
    }, []),
    onSpanUpdated: useCallback((span: ObsSpan) => {
      setLastUpdate(new Date());
    }, []),
  });

  // Popover state (internal state now managed by ExecHierarchyPopover component)
  const [popoverNode, setPopoverNode] = useState<HierarchyNode | null>(null);
  const [popoverPos, setPopoverPos] = useState({ x: 0, y: 0 });

  // Transform spans to hierarchy nodes (same data as TraceWaterfall)
  const rawNodes = useMemo(() => {
    if (!spans || spans.length === 0) return [];
    return transformSpansToNodes(spans);
  }, [spans]);

  // Extract unique span types from all spans (for filter dropdown)
  // Uses collectUniqueSpanTypes from utils/idService.ts
  const uniqueSpanTypes = useMemo(() => {
    if (!spans) return [];
    return collectUniqueSpanTypes(spans);
  }, [spans]);

  // Count only hidden types that exist in current data (avoids negative counts when switching traces)
  const effectiveHiddenCount = useMemo(() => {
    return uniqueSpanTypes.filter(type => hiddenSpanTypes.has(type)).length;
  }, [uniqueSpanTypes, hiddenSpanTypes]);

  // Extract unique task_ids from spans (for filtering TaskHierarchyGraph)
  // Uses collectSpanTaskIds from utils/idService.ts
  const spanTaskIds = useMemo(() => {
    if (!spans) return [];
    return collectSpanTaskIds(spans);
  }, [spans]);

  // Show All / Hide All handlers for span type filter
  const handleShowAllSpanTypes = useCallback(() => {
    uniqueSpanTypes.forEach(type => {
      if (hiddenSpanTypes.has(type)) toggleSpanType(type);
    });
  }, [uniqueSpanTypes, hiddenSpanTypes, toggleSpanType]);

  const handleHideAllSpanTypes = useCallback(() => {
    uniqueSpanTypes.forEach(type => {
      if (!hiddenSpanTypes.has(type)) toggleSpanType(type);
    });
  }, [uniqueSpanTypes, hiddenSpanTypes, toggleSpanType]);

  // Apply coordinator view mode transformation
  const transformedNodes = useMemo(() => {
    if (coordViewMode === 'breakout') {
      return breakoutCoordinatorTasks(rawNodes);
    }
    return rawNodes;
  }, [rawNodes, coordViewMode]);

  // Apply span type filtering (Milestone 14): hide spans by type, promote their children
  // When a span type is in hiddenSpanTypes, it's removed and its children are promoted to parent
  const spanFilteredNodes = useMemo(() => {
    if (hiddenSpanTypes.size === 0) return transformedNodes; // No filtering

    // Helper to recursively filter nodes, promoting children of hidden nodes
    const filterNodes = (nodeList: HierarchyNode[]): HierarchyNode[] => {
      const result: HierarchyNode[] = [];

      for (const node of nodeList) {
        const spanName = node._span?.name || '';
        const isHidden = hiddenSpanTypes.has(spanName);

        if (isHidden) {
          // This node is hidden - promote its children to this level
          if (node.children && node.children.length > 0) {
            // Recursively filter children first
            const filteredChildren = filterNodes(node.children);
            result.push(...filteredChildren);
          }
          // Node itself is not added to result
        } else {
          // This node is visible - keep it, but filter its children
          if (node.children && node.children.length > 0) {
            const filteredChildren = filterNodes(node.children);
            result.push({
              ...node,
              children: filteredChildren,
              childCount: filteredChildren.length,
              isCollapsible: filteredChildren.length > 0,
            });
          } else {
            result.push(node);
          }
        }
      }

      // Sort by start time to preserve execution order after promotion
      // When reverseOrder is true, show newest first (descending)
      result.sort((a, b) => {
        const aStart = a._span?.startMs || 0;
        const bStart = b._span?.startMs || 0;
        return reverseOrder ? bStart - aStart : aStart - bStart;
      });

      return result;
    };

    return filterNodes(transformedNodes);
  }, [transformedNodes, hiddenSpanTypes, reverseOrder]);

  // Apply filter criteria (marks nodes as isFiltered, does NOT hide them)
  const nodes = useMemo(() => {
    return applyFilterToNodes(spanFilteredNodes, filterCriteria);
  }, [spanFilteredNodes, filterCriteria]);

  // Count total nodes recursively (for display limit)
  const totalNodeCount = useMemo(() => {
    let count = 0;
    const countNodes = (nodeList: HierarchyNode[]) => {
      for (const node of nodeList) {
        count++;
        if (node.children) countNodes(node.children);
      }
    };
    countNodes(nodes);
    return count;
  }, [nodes]);

  // Apply display limit to nodes
  const limitedNodes = useMemo(() => {
    if (totalNodeCount <= displayLimit) return nodes;

    // Flatten, limit, and rebuild tree structure
    let remaining = displayLimit;
    const limitTree = (nodeList: HierarchyNode[]): HierarchyNode[] => {
      const result: HierarchyNode[] = [];
      for (const node of nodeList) {
        if (remaining <= 0) break;
        remaining--;
        const limitedChildren = node.children ? limitTree(node.children) : undefined;
        result.push({
          ...node,
          children: limitedChildren,
          childCount: limitedChildren?.length,
        });
      }
      return result;
    };
    return limitTree(nodes);
  }, [nodes, displayLimit, totalNodeCount]);

  const hasMoreNodes = totalNodeCount > displayLimit;
  const loadMoreNodes = () => setDisplayLimit(prev => prev + LOAD_MORE_INCREMENT);
  const showAllNodes = () => setDisplayLimit(totalNodeCount);

  // Handle highlighted span (from outliers click) - auto-expand path and scroll to node
  useEffect(() => {
    if (!highlightedSpanId || !spans || spans.length === 0) return;

    // Find the path to the highlighted span
    const findPathToNode = (nodeList: HierarchyNode[], targetId: string, path: string[] = []): string[] | null => {
      for (const node of nodeList) {
        // Check if this node matches the highlighted span
        if (node._span?.id === targetId || node.id === targetId) {
          return path;
        }
        if (node.children && node.children.length > 0) {
          const foundPath = findPathToNode(node.children, targetId, [...path, node.id]);
          if (foundPath) return foundPath;
        }
      }
      return null;
    };

    // Delay to ensure nodes are rendered
    const timer = setTimeout(() => {
      const pathToExpand = findPathToNode(nodes, highlightedSpanId);
      if (pathToExpand) {
        // Expand all nodes in the path
        setExpandedNodes(prev => {
          const next = new Set(prev);
          pathToExpand.forEach(id => next.add(id));
          return next;
        });
        setExpandChangeCounter(c => c + 1);

        // Scroll to the highlighted element after expansion
        setTimeout(() => {
          const element = document.querySelector(`[data-span-id="${highlightedSpanId}"]`);
          if (element) {
            element.scrollIntoView({ behavior: 'smooth', block: 'center' });
            // Add pulse animation class
            element.classList.add(styles.highlightedSpan);
            // Remove after animation
            setTimeout(() => {
              element.classList.remove(styles.highlightedSpan);
              onClearHighlight?.();
            }, 3000);
          }
        }, 100);
      }
    }, 50);

    return () => clearTimeout(timer);
  }, [highlightedSpanId, spans, nodes, onClearHighlight]);

  // Expand ONE LEVEL: expand children of currently expanded nodes (or roots if nothing expanded)
  const expandOneLevel = useCallback(() => {
    setExpandedNodes(prev => {
      const next = new Set(prev);

      // Helper to find all nodes at given IDs
      const findNode = (id: string, nodeList: HierarchyNode[]): HierarchyNode | undefined => {
        for (const node of nodeList) {
          if (node.id === id) return node;
          if (node.children) {
            const found = findNode(id, node.children);
            if (found) return found;
          }
        }
        return undefined;
      };

      // If nothing expanded, expand root nodes
      if (prev.size === 0) {
        nodes.forEach(n => {
          if (n.isCollapsible) {
            next.add(n.id);
          }
        });
        return next;
      }

      // Find children of currently expanded nodes and expand them
      const expandChildren = (nodeList: HierarchyNode[]) => {
        for (const node of nodeList) {
          if (prev.has(node.id) && node.children) {
            // This node is expanded, expand its children
            node.children.forEach(child => {
              if (child.isCollapsible) {
                next.add(child.id);
              }
            });
          }
          // Recurse to check nested expanded nodes
          if (node.children) {
            expandChildren(node.children);
          }
        }
      };

      expandChildren(nodes);
      return next;
    });
    setExpandChangeCounter(c => c + 1);
  }, [nodes]);

  // Collapse ONE LEVEL: collapse the deepest expanded nodes
  const collapseOneLevel = useCallback(() => {
    setExpandedNodes(prev => {
      if (prev.size === 0) return prev;

      // Find the depth of each expanded node
      const getDepth = (id: string, nodeList: HierarchyNode[], depth: number): number => {
        for (const node of nodeList) {
          if (node.id === id) return depth;
          if (node.children) {
            const found = getDepth(id, node.children, depth + 1);
            if (found >= 0) return found;
          }
        }
        return -1;
      };

      // Get depths of all expanded nodes
      const depths = new Map<string, number>();
      prev.forEach(id => {
        const depth = getDepth(id, nodes, 0);
        if (depth >= 0) depths.set(id, depth);
      });

      // Find max depth
      let maxDepth = -1;
      depths.forEach(d => {
        if (d > maxDepth) maxDepth = d;
      });

      // Remove nodes at max depth
      const next = new Set<string>();
      prev.forEach(id => {
        const depth = depths.get(id);
        if (depth !== undefined && depth < maxDepth) {
          next.add(id);
        }
      });

      return next;
    });
    setExpandChangeCounter(c => c + 1);
  }, [nodes]);

  // Toggle expand/collapse for a node (individual click)
  const handleToggleExpand = useCallback((nodeId: string) => {
    setExpandedNodes(prev => {
      const next = new Set(prev);
      if (next.has(nodeId)) {
        next.delete(nodeId);
      } else {
        next.add(nodeId);
      }
      return next;
    });
    setExpandChangeCounter(c => c + 1);
  }, []);

  // Expand all nodes
  const handleExpandAll = useCallback(() => {
    const allIds = new Set<string>();
    const collect = (nodeList: HierarchyNode[]) => {
      for (const node of nodeList) {
        if (node.isCollapsible) {
          allIds.add(node.id);
        }
        if (node.children) {
          collect(node.children);
        }
      }
    };
    collect(nodes);
    setExpandedNodes(allIds);
    setExpandChangeCounter(c => c + 1);
  }, [nodes]);

  // Collapse all nodes
  const handleCollapseAll = useCallback(() => {
    setExpandedNodes(new Set());
    setExpandChangeCounter(c => c + 1);
  }, []);

  const isEmpty = propsIsEmpty ?? nodes.length === 0;

  // Calculate viewport-aware position for popover
  const calculatePopoverPosition = useCallback((clickX: number, clickY: number) => {
    const viewportWidth = window.innerWidth;
    const viewportHeight = window.innerHeight;

    let x = clickX + 10;
    let y = clickY + 10;

    // Keep popover within right edge
    if (x + POPOVER_MAX_WIDTH + POPOVER_MARGIN > viewportWidth) {
      x = clickX - POPOVER_MAX_WIDTH - 10;
    }
    // Keep within left edge
    if (x < POPOVER_MARGIN) {
      x = POPOVER_MARGIN;
    }

    // Keep popover within bottom edge
    if (y + POPOVER_MAX_HEIGHT + POPOVER_MARGIN > viewportHeight) {
      y = viewportHeight - POPOVER_MAX_HEIGHT - POPOVER_MARGIN;
    }
    // Keep within top edge
    if (y < POPOVER_MARGIN) {
      y = POPOVER_MARGIN;
    }

    return { x, y };
  }, []);

  // Handle node click - show popover
  const handleNodeClick = useCallback(
    (node: HierarchyNode, event?: React.MouseEvent) => {
      onNodeClick?.(node);

      // Toggle popover - close if clicking same node
      if (popoverNode?.id === node.id) {
        setPopoverNode(null);
      } else {
        setPopoverNode(node);
        // Position with viewport awareness
        if (event) {
          const pos = calculatePopoverPosition(event.clientX, event.clientY);
          setPopoverPos(pos);
        } else {
          // Fallback to center of viewport
          setPopoverPos({ x: window.innerWidth / 2 - POPOVER_MAX_WIDTH / 2, y: 100 });
        }
      }
    },
    [onNodeClick, popoverNode, calculatePopoverPosition]
  );

  // Close popover when clicking outside
  const handleContainerClick = useCallback((e: React.MouseEvent) => {
    // Only close if clicking directly on container (not on children)
    if (e.target === e.currentTarget) {
      setPopoverNode(null);
    }
  }, []);

  return (
    <div className={`${styles.container} ${isExpanded ? styles.containerExpanded : ''}`}>
      {/* Header - Mission Control Telemetry Bar (Extracted PR 3 - M-DASHBOARD-SIMPLIFICATION) */}
      <ExecHierarchyToolbar
        isEmpty={isEmpty}
        totalNodeCount={totalNodeCount}
        topLevelNodeCount={nodes.length}
        viewMode={viewMode}
        coordViewMode={coordViewMode}
        reverseOrder={reverseOrder}
        isExpanded={isExpanded}
        onViewModeChange={setViewMode}
        onCoordViewModeChange={setCoordViewMode}
        onToggleReverseOrder={() => setReverseOrder(!reverseOrder)}
        onToggleExpand={onToggleExpand}
        uniqueSpanTypes={uniqueSpanTypes}
        hiddenSpanTypes={hiddenSpanTypes}
        effectiveHiddenCount={effectiveHiddenCount}
        showSpanTypeFilter={showSpanTypeFilter}
        onToggleSpanType={toggleSpanType}
        onToggleSpanTypeFilter={() => setShowSpanTypeFilter(!showSpanTypeFilter)}
        onShowAllSpanTypes={handleShowAllSpanTypes}
        onHideAllSpanTypes={handleHideAllSpanTypes}
        onExpandAll={handleExpandAll}
        onCollapseAll={handleCollapseAll}
        onExpandOneLevel={expandOneLevel}
        onCollapseOneLevel={collapseOneLevel}
      />

      {/* Selected Span Metadata */}
      {popoverNode && popoverNode._span && (
        <div className={styles.selectedSpanMeta}>
          <div className={styles.metaRow}>
            <span className={styles.metaLabel}>Source</span>
            <span className={styles.metaValue}>
              {popoverNode._span.attributes?.['source'] || popoverNode.provider || 'claude-code'}
            </span>
          </div>
          <div className={styles.metaRow}>
            <span className={styles.metaLabel}>Target</span>
            <span className={styles.metaValue}>
              {popoverNode._span.attributes?.['target'] || 'user'}
            </span>
          </div>
          <div className={styles.metaRow}>
            <span className={styles.metaLabel}>Time</span>
            <span className={styles.metaValue}>
              {popoverNode._span.startMs ? new Date(popoverNode._span.startMs).toLocaleString() : '—'}
            </span>
          </div>
          <div className={styles.metaRow}>
            <span className={styles.metaLabel}>Task ID</span>
            <span className={styles.metaValue} title={popoverNode.taskId || popoverNode._span.id}>
              {(popoverNode.taskId || popoverNode._span.id || '—').slice(0, 36)}
            </span>
          </div>
          <div className={styles.metaRow}>
            <span className={styles.metaLabel}>Content</span>
            <span className={styles.metaValue}>{popoverNode.label}</span>
          </div>
          <button
            className={styles.metaClose}
            onClick={() => setPopoverNode(null)}
            title="Clear selection"
          >
            ×
          </button>
        </div>
      )}

      {/* Viewport */}
      <div
        className={`${styles.viewport} ${isExpanded ? styles.viewportExpanded : ''}`}
        onClick={handleContainerClick}
      >
        {viewMode === 'tree' && (
          <ExecHierarchyTree
            nodes={limitedNodes}
            selectedNodeId={selectedNodeId}
            onNodeClick={handleNodeClick}
            loading={loading}
            error={null}
            expandedNodeIds={expandedNodes}
            onToggleExpand={handleToggleExpand}
            highlightedSpanId={highlightedSpanId}
            onChatContextClick={() => {
              // Switch to chat view when 💬 button is clicked
              setViewMode('chat');
            }}
          />
        )}
        {viewMode === 'graph' && (
          <TaskHierarchyGraph
            selectedNodeId={selectedNodeId}
            // FIX: Pass spans directly - same data source as Tree/Timeline/Chat views
            // This ensures filtering works correctly when events are selected
            spans={spans}
            // Keep legacy filter props for fallback when no spans loaded
            filterTaskId={selectedNodeId?.startsWith('task-') ? selectedNodeId : undefined}
            spanTaskIds={spanTaskIds.length > 0 ? spanTaskIds : undefined}
            filterTraceId={selectedNodeId && !selectedNodeId.startsWith('task-') && spanTaskIds.length === 0 ? selectedNodeId : undefined}
            // Use handleNodeClick for popover (same as Tree view)
            onNodeClick={handleNodeClick}
            isExpanded={isExpanded}
            recenterTrigger={expandChangeCounter}
            workspace={filters?.workspace}
            provider={filters?.provider}
            // Span type filtering - same as other views
            hiddenSpanTypes={hiddenSpanTypes}
            // Chat context - switch to chat view when clicked
            onChatContextClick={() => {
              setViewMode('chat');
            }}
          />
        )}
        {viewMode === 'timeline' && (
          <TraceWaterfall
            spans={spans || []}
            loading={loading}
            hiddenSpanTypes={hiddenSpanTypes}
            onToggleSpanType={toggleSpanType}
            onChatContextClick={(span) => {
              // Switch to chat view when 💬 button is clicked
              // ChatHistory will display messages for the current spans/session
              setViewMode('chat');
            }}
          />
        )}
        {viewMode === 'chat' && (
          <ChatHistory
            nodes={limitedNodes}
            selectedNodeId={selectedNodeId}
            onNodeClick={handleNodeClick}
            loading={loading}
            spans={spans}
          />
        )}
        {viewMode === 'evolution' && (
          <EvolutionTree
            spans={spans}
            nodes={transformedNodes}
            selectedNodeId={selectedNodeId}
            onNodeClick={handleNodeClick}
            hiddenSpanTypes={hiddenSpanTypes}
            isExpanded={isExpanded}
            theme={theme}
            onChatContextClick={() => {
              // Switch to chat view when chat button is clicked
              setViewMode('chat');
            }}
          />
        )}

        {/* Load more buttons when nodes are limited */}
        {hasMoreNodes && (
          <div className={styles.loadMoreContainer}>
            <span className={styles.loadMoreInfo}>
              Showing {displayLimit} of {totalNodeCount} nodes
            </span>
            <button className={styles.loadMoreBtn} onClick={loadMoreNodes}>
              Load {Math.min(LOAD_MORE_INCREMENT, totalNodeCount - displayLimit)} more
            </button>
            <button className={styles.loadMoreBtn} onClick={showAllNodes}>
              Show all {totalNodeCount}
            </button>
          </div>
        )}
      </div>

      {/* Node Popover (Extracted PR 3 - M-DASHBOARD-SIMPLIFICATION) */}
      {popoverNode && (
        <ExecHierarchyPopover
          node={popoverNode}
          position={popoverPos}
          onClose={() => setPopoverNode(null)}
          hiddenSpanTypes={hiddenSpanTypes}
          onToggleSpanType={toggleSpanType}
        />
      )}

      {/* CLI command hint */}
      <CliCommandHint
        commandType="hierarchy"
        filters={filters}
        compact
      />
    </div>
  );
};

export default ExecHierarchy;
