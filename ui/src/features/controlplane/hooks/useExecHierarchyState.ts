/**
 * State management hook for ExecHierarchy component
 * Consolidates 14 useState hooks into organized state groups
 *
 * Groups:
 * - View: viewMode, coordViewMode
 * - Expansion: expandedNodes, expandChangeCounter
 * - Filter: hiddenSpanTypes, showSpanTypeFilter
 * - Display: displayLimit, reverseOrder
 * - Popover: popoverNode, popoverPos, section expansion states
 */

import { useState, useCallback, useMemo } from 'react';
import type { HierarchyNode, ViewMode, CoordinatorViewMode } from '../components/ExecHierarchy/types';

// ============================================================================
// Types
// ============================================================================

export interface PopoverPosition {
  x: number;
  y: number;
}

export interface ExecHierarchyViewState {
  viewMode: ViewMode;
  coordViewMode: CoordinatorViewMode;
}

export interface ExecHierarchyExpansionState {
  expandedNodes: Set<string>;
  expandChangeCounter: number;
}

export interface ExecHierarchyFilterState {
  hiddenSpanTypes: Set<string>;
  showSpanTypeFilter: boolean;
}

export interface ExecHierarchyDisplayState {
  displayLimit: number;
  reverseOrder: boolean;
}

export interface ExecHierarchyPopoverState {
  popoverNode: HierarchyNode | null;
  popoverPos: PopoverPosition;
  attributesExpanded: boolean;
  metricsExpanded: boolean;
  toolDetailsExpanded: boolean;
}

export interface ExecHierarchyState {
  view: ExecHierarchyViewState;
  expansion: ExecHierarchyExpansionState;
  filter: ExecHierarchyFilterState;
  display: ExecHierarchyDisplayState;
  popover: ExecHierarchyPopoverState;
}

export interface ExecHierarchyActions {
  // View actions
  setViewMode: (mode: ViewMode) => void;
  setCoordViewMode: (mode: CoordinatorViewMode) => void;

  // Expansion actions
  toggleNodeExpand: (nodeId: string) => void;
  expandAll: (nodes: HierarchyNode[]) => void;
  collapseAll: () => void;
  expandOneLevel: (nodes: HierarchyNode[]) => void;
  collapseOneLevel: (nodes: HierarchyNode[]) => void;

  // Filter actions
  toggleSpanType: (spanType: string) => void;
  showAllSpanTypes: (allTypes: string[]) => void;
  hideAllSpanTypes: (allTypes: string[]) => void;
  toggleSpanTypeFilter: () => void;

  // Display actions
  setDisplayLimit: (limit: number) => void;
  loadMoreNodes: (increment: number) => void;
  showAllNodes: (totalCount: number) => void;
  toggleReverseOrder: () => void;

  // Popover actions
  openPopover: (node: HierarchyNode, pos: PopoverPosition) => void;
  closePopover: () => void;
  togglePopover: (node: HierarchyNode, pos: PopoverPosition) => void;
  toggleAttributesExpanded: () => void;
  toggleMetricsExpanded: () => void;
  toggleToolDetailsExpanded: () => void;
}

export interface UseExecHierarchyStateResult {
  state: ExecHierarchyState;
  actions: ExecHierarchyActions;
}

// ============================================================================
// Constants
// ============================================================================

const DEFAULT_DISPLAY_LIMIT = 100;
const DEFAULT_HIDDEN_SPAN_TYPES = new Set(['api_request']);

// ============================================================================
// Hook Implementation
// ============================================================================

export interface UseExecHierarchyStateOptions {
  /** Initial view mode */
  initialViewMode?: ViewMode;
  /** External hidden span types (for lifted state) */
  externalHiddenTypes?: Set<string>;
  /** External toggle handler (for lifted state) */
  onToggleSpanType?: (spanType: string) => void;
}

export function useExecHierarchyState(
  options: UseExecHierarchyStateOptions = {}
): UseExecHierarchyStateResult {
  const {
    initialViewMode = 'graph',
    externalHiddenTypes,
    onToggleSpanType: externalToggle,
  } = options;

  // ============================================================================
  // View State
  // ============================================================================

  const [viewMode, setViewMode] = useState<ViewMode>(initialViewMode);
  const [coordViewMode, setCoordViewMode] = useState<CoordinatorViewMode>('nested');

  // ============================================================================
  // Expansion State
  // ============================================================================

  const [expandedNodes, setExpandedNodes] = useState<Set<string>>(new Set());
  const [expandChangeCounter, setExpandChangeCounter] = useState(0);

  const toggleNodeExpand = useCallback((nodeId: string) => {
    setExpandedNodes((prev) => {
      const next = new Set(prev);
      if (next.has(nodeId)) {
        next.delete(nodeId);
      } else {
        next.add(nodeId);
      }
      return next;
    });
    setExpandChangeCounter((c) => c + 1);
  }, []);

  const expandAll = useCallback((nodes: HierarchyNode[]) => {
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
    setExpandChangeCounter((c) => c + 1);
  }, []);

  const collapseAll = useCallback(() => {
    setExpandedNodes(new Set());
    setExpandChangeCounter((c) => c + 1);
  }, []);

  const expandOneLevel = useCallback((nodes: HierarchyNode[]) => {
    setExpandedNodes((prev) => {
      const next = new Set(prev);

      // If nothing expanded, expand root nodes
      if (prev.size === 0) {
        nodes.forEach((n) => {
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
            node.children.forEach((child) => {
              if (child.isCollapsible) {
                next.add(child.id);
              }
            });
          }
          if (node.children) {
            expandChildren(node.children);
          }
        }
      };

      expandChildren(nodes);
      return next;
    });
    setExpandChangeCounter((c) => c + 1);
  }, []);

  const collapseOneLevel = useCallback((nodes: HierarchyNode[]) => {
    setExpandedNodes((prev) => {
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
      prev.forEach((id) => {
        const depth = getDepth(id, nodes, 0);
        if (depth >= 0) depths.set(id, depth);
      });

      // Find max depth
      let maxDepth = -1;
      depths.forEach((d) => {
        if (d > maxDepth) maxDepth = d;
      });

      // Remove nodes at max depth
      const next = new Set<string>();
      prev.forEach((id) => {
        const depth = depths.get(id);
        if (depth !== undefined && depth < maxDepth) {
          next.add(id);
        }
      });

      return next;
    });
    setExpandChangeCounter((c) => c + 1);
  }, []);

  // ============================================================================
  // Filter State
  // ============================================================================

  const [internalHiddenTypes, setInternalHiddenTypes] = useState<Set<string>>(DEFAULT_HIDDEN_SPAN_TYPES);
  const [showSpanTypeFilter, setShowSpanTypeFilter] = useState(false);

  // Use external state if provided, otherwise use internal
  const hiddenSpanTypes = externalHiddenTypes ?? internalHiddenTypes;

  const toggleSpanType = useCallback(
    (spanType: string) => {
      if (externalToggle) {
        externalToggle(spanType);
      } else {
        setInternalHiddenTypes((prev) => {
          const next = new Set(prev);
          if (next.has(spanType)) {
            next.delete(spanType);
          } else {
            next.add(spanType);
          }
          return next;
        });
      }
    },
    [externalToggle]
  );

  const showAllSpanTypes = useCallback(
    (allTypes: string[]) => {
      allTypes.forEach((type) => {
        if (hiddenSpanTypes.has(type)) {
          toggleSpanType(type);
        }
      });
    },
    [hiddenSpanTypes, toggleSpanType]
  );

  const hideAllSpanTypes = useCallback(
    (allTypes: string[]) => {
      allTypes.forEach((type) => {
        if (!hiddenSpanTypes.has(type)) {
          toggleSpanType(type);
        }
      });
    },
    [hiddenSpanTypes, toggleSpanType]
  );

  const toggleSpanTypeFilterFn = useCallback(() => {
    setShowSpanTypeFilter((prev) => !prev);
  }, []);

  // ============================================================================
  // Display State
  // ============================================================================

  const [displayLimit, setDisplayLimit] = useState(DEFAULT_DISPLAY_LIMIT);
  const [reverseOrder, setReverseOrder] = useState(false);

  const loadMoreNodes = useCallback((increment: number) => {
    setDisplayLimit((prev) => prev + increment);
  }, []);

  const showAllNodes = useCallback((totalCount: number) => {
    setDisplayLimit(totalCount);
  }, []);

  const toggleReverseOrder = useCallback(() => {
    setReverseOrder((prev) => !prev);
  }, []);

  // ============================================================================
  // Popover State
  // ============================================================================

  const [popoverNode, setPopoverNode] = useState<HierarchyNode | null>(null);
  const [popoverPos, setPopoverPos] = useState<PopoverPosition>({ x: 0, y: 0 });
  const [attributesExpanded, setAttributesExpanded] = useState(false);
  const [metricsExpanded, setMetricsExpanded] = useState(false);
  const [toolDetailsExpanded, setToolDetailsExpanded] = useState(false);

  const openPopover = useCallback((node: HierarchyNode, pos: PopoverPosition) => {
    setPopoverNode(node);
    setPopoverPos(pos);
    // Reset all collapsible sections to collapsed by default
    setAttributesExpanded(false);
    setMetricsExpanded(false);
    setToolDetailsExpanded(false);
  }, []);

  const closePopover = useCallback(() => {
    setPopoverNode(null);
  }, []);

  const togglePopover = useCallback(
    (node: HierarchyNode, pos: PopoverPosition) => {
      if (popoverNode?.id === node.id) {
        closePopover();
      } else {
        openPopover(node, pos);
      }
    },
    [popoverNode, openPopover, closePopover]
  );

  const toggleAttributesExpanded = useCallback(() => {
    setAttributesExpanded((prev) => !prev);
  }, []);

  const toggleMetricsExpanded = useCallback(() => {
    setMetricsExpanded((prev) => !prev);
  }, []);

  const toggleToolDetailsExpanded = useCallback(() => {
    setToolDetailsExpanded((prev) => !prev);
  }, []);

  // ============================================================================
  // Combine State and Actions
  // ============================================================================

  const state: ExecHierarchyState = useMemo(
    () => ({
      view: {
        viewMode,
        coordViewMode,
      },
      expansion: {
        expandedNodes,
        expandChangeCounter,
      },
      filter: {
        hiddenSpanTypes,
        showSpanTypeFilter,
      },
      display: {
        displayLimit,
        reverseOrder,
      },
      popover: {
        popoverNode,
        popoverPos,
        attributesExpanded,
        metricsExpanded,
        toolDetailsExpanded,
      },
    }),
    [
      viewMode,
      coordViewMode,
      expandedNodes,
      expandChangeCounter,
      hiddenSpanTypes,
      showSpanTypeFilter,
      displayLimit,
      reverseOrder,
      popoverNode,
      popoverPos,
      attributesExpanded,
      metricsExpanded,
      toolDetailsExpanded,
    ]
  );

  const actions: ExecHierarchyActions = useMemo(
    () => ({
      // View
      setViewMode,
      setCoordViewMode,
      // Expansion
      toggleNodeExpand,
      expandAll,
      collapseAll,
      expandOneLevel,
      collapseOneLevel,
      // Filter
      toggleSpanType,
      showAllSpanTypes,
      hideAllSpanTypes,
      toggleSpanTypeFilter: toggleSpanTypeFilterFn,
      // Display
      setDisplayLimit,
      loadMoreNodes,
      showAllNodes,
      toggleReverseOrder,
      // Popover
      openPopover,
      closePopover,
      togglePopover,
      toggleAttributesExpanded,
      toggleMetricsExpanded,
      toggleToolDetailsExpanded,
    }),
    [
      toggleNodeExpand,
      expandAll,
      collapseAll,
      expandOneLevel,
      collapseOneLevel,
      toggleSpanType,
      showAllSpanTypes,
      hideAllSpanTypes,
      toggleSpanTypeFilterFn,
      loadMoreNodes,
      showAllNodes,
      toggleReverseOrder,
      openPopover,
      closePopover,
      togglePopover,
      toggleAttributesExpanded,
      toggleMetricsExpanded,
      toggleToolDetailsExpanded,
    ]
  );

  return { state, actions };
}
