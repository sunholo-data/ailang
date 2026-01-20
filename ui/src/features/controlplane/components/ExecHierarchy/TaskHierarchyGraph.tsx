/**
 * TaskHierarchyGraph - ReactFlow graph view for cross-task visualization
 * Shows: Tasks with handoff relationships, session continuity, and approval status
 * Uses dagre for automatic hierarchical layout with tasks as primary nodes
 */
import React, { useMemo, useCallback, useRef, useEffect, useState } from 'react';
import ReactFlow, {
  Node,
  Edge,
  Background,
  Controls,
  MiniMap,
  useNodesState,
  useEdgesState,
  MarkerType,
  Handle,
  Position,
  NodeProps,
  ReactFlowInstance,
  ReactFlowProvider,
} from 'reactflow';
import dagre from 'dagre';
import 'reactflow/dist/style.css';
import type { TaskHierarchyNode, TaskHierarchyEdge, TaskHierarchyResult, TaskSpanNode, Span, HierarchyNode } from './types';
import { useTaskHierarchy } from '../../hooks/useTaskHierarchy';
import { buildGraphFromSpans, buildGraphFromTurnGrouped, type HierarchySpan } from '../../utils/buildGraphFromSpans';
import styles from './ExecHierarchy.module.css';

// Get node color based on status and approval
function getNodeColor(status: string, approvalStatus?: string): string {
  if (approvalStatus === 'pending') return '#f59e0b'; // Amber for pending approval
  if (approvalStatus === 'rejected') return '#ef4444'; // Red for rejected
  switch (status) {
    case 'completed':
    case 'done':
      return '#25c2a0'; // Green
    case 'running':
    case 'busy':
      return '#3b82f6'; // Blue
    case 'failed':
    case 'error':
      return '#ef4444'; // Red
    case 'pending':
    case 'pending_approval':
      return '#f59e0b'; // Amber
    default:
      return '#64748b'; // Gray
  }
}

// Get span node color based on node_type
function getSpanNodeColor(nodeType?: string): string {
  switch (nodeType) {
    case 'coordinator':
      return '#8b5cf6'; // Purple
    case 'executor':
      return '#3b82f6'; // Blue
    case 'turn':
      return '#10b981'; // Green
    case 'tool':
      return '#f59e0b'; // Amber
    default:
      return '#64748b'; // Gray
  }
}

// Get span node icon
function getSpanNodeIcon(nodeType?: string): string {
  switch (nodeType) {
    case 'coordinator':
      return '\u2B21'; // Hexagon
    case 'executor':
      return '\u25CF'; // Circle
    case 'turn':
      return '\u25C9'; // Fish eye
    case 'tool':
      return '\u2699'; // Gear
    default:
      return '\u2022'; // Bullet
  }
}

// Get provider icon
function getProviderIcon(provider?: string): string {
  switch (provider) {
    case 'claude':
    case 'claude-code':
      return '\u{1F7E0}'; // Orange circle
    case 'gemini':
    case 'gemini-cli':
      return '\u{1F535}'; // Blue circle
    case 'script':
      return '\u{1F7E2}'; // Green circle
    default:
      return '';
  }
}

// Get approval status icon and color
function getApprovalBadge(status?: string): { icon: string; color: string } {
  switch (status) {
    case 'pending':
      return { icon: '\u23F3', color: '#f59e0b' }; // Hourglass, amber
    case 'approved':
      return { icon: '\u2713', color: '#25c2a0' }; // Check, green
    case 'rejected':
      return { icon: '\u2717', color: '#ef4444' }; // X, red
    default:
      return { icon: '', color: '#64748b' };
  }
}

// Format cost for display
function formatCost(cost?: number): string {
  if (!cost || cost === 0) return '';
  if (cost < 0.01) return `$${cost.toFixed(4)}`;
  if (cost < 1) return `$${cost.toFixed(3)}`;
  return `$${cost.toFixed(2)}`;
}

// Format token count for display
function formatTokens(count?: number): string {
  if (!count || count === 0) return '';
  if (count >= 1000) return `${(count / 1000).toFixed(1)}K`;
  return String(count);
}

// Format duration for display
function formatDuration(ms?: number): string {
  if (!ms) return '';
  if (ms < 1000) return `${ms}ms`;
  if (ms < 60000) return `${(ms / 1000).toFixed(1)}s`;
  return `${(ms / 60000).toFixed(1)}m`;
}

// Count spans recursively
function countSpans(spans?: { children?: unknown[] }[]): number {
  if (!spans) return 0;
  let count = spans.length;
  for (const span of spans) {
    if (span.children) {
      count += countSpans(span.children as { children?: unknown[] }[]);
    }
  }
  return count;
}

// Custom node component for tasks
const TaskNodeComponent: React.FC<NodeProps> = ({ data, selected }) => {
  const nodeColor = getNodeColor(data.status, data.approval_status);
  const approvalBadge = getApprovalBadge(data.approval_status);
  const hasMetrics = data.cost || data.tokens_in || data.tokens_out;
  const isHandoffTarget = data.isHandoffTarget;
  const iterationBadge = data.iteration && data.iteration > 1 ? `Iter ${data.iteration}` : '';
  const spanCount = countSpans(data.spans);
  const isExpanded = data.isExpanded;
  const onToggleExpand = data.onToggleExpand;

  const handleExpandClick = (e: React.MouseEvent) => {
    e.stopPropagation();
    if (onToggleExpand) {
      onToggleExpand(data.id);
    }
  };

  return (
    <div
      className={`${styles.rfNode} ${selected ? styles.rfNodeSelected : ''} ${isHandoffTarget ? styles.rfNodeHandoffTarget : ''}`}
      style={{
        borderLeftColor: nodeColor,
        minWidth: '220px',
      }}
    >
      <Handle type="target" position={Position.Top} className={styles.rfHandle} />

      {/* Header with provider icon, title, and approval badge */}
      <div className={styles.rfNodeHeader}>
        {data.provider && (
          <span className={styles.rfProviderIcon}>{getProviderIcon(data.provider)}</span>
        )}
        <span className={styles.rfNodeLabel} title={data.title}>
          {data.title && data.title.length > 35
            ? data.title.substring(0, 35) + '...'
            : data.title || data.id}
        </span>
        {approvalBadge.icon && (
          <span
            className={styles.rfApprovalBadge}
            style={{ backgroundColor: approvalBadge.color }}
            title={`Approval: ${data.approval_status}`}
          >
            {approvalBadge.icon}
          </span>
        )}
      </div>

      {/* Agent and status row */}
      <div className={styles.rfNodeMeta}>
        {data.agent_id && (
          <span className={styles.rfAgentLabel} title={`Agent: ${data.agent_id}`}>
            {data.agent_id}
          </span>
        )}
        <span style={{ color: nodeColor }}>{data.status}</span>
        {data.duration_ms > 0 && <span> • {formatDuration(data.duration_ms)}</span>}
        {iterationBadge && (
          <span
            className={styles.rfIterationBadge}
            style={{ backgroundColor: data.iteration >= 3 ? '#f59e0b' : '#3b82f6' }}
          >
            {iterationBadge}
          </span>
        )}
      </div>

      {/* Metrics row */}
      {hasMetrics && (
        <div className={styles.rfNodeMetrics}>
          {data.cost > 0 && <span className={styles.rfCost}>{formatCost(data.cost)}</span>}
          {data.tokens_in > 0 && <span>{formatTokens(data.tokens_in)} in</span>}
          {data.tokens_in > 0 && data.tokens_out > 0 && <span> / </span>}
          {data.tokens_out > 0 && <span>{formatTokens(data.tokens_out)} out</span>}
          {data.turns && data.turns > 0 && <span> • {data.turns} turns</span>}
        </div>
      )}

      {/* Spans expand/collapse button */}
      {spanCount > 0 && (
        <div className={styles.rfNodeSpans}>
          <button
            className={styles.rfSpanExpandBtn}
            onClick={handleExpandClick}
            title={isExpanded ? "Collapse spans" : "Expand spans"}
          >
            <span className={styles.rfExpandIcon}>{isExpanded ? '\u25BC' : '\u25B6'}</span>
            <span>{spanCount} spans</span>
          </button>
        </div>
      )}

      {/* Task ID hint */}
      <div className={styles.rfNodeId}>
        <span title={data.id}>{data.id}</span>
      </div>

      <Handle type="source" position={Position.Bottom} className={styles.rfHandle} />
    </div>
  );
};

// Custom node component for spans (execution details)
const SpanNodeComponent: React.FC<NodeProps> = ({ data }) => {
  // Use nodeColor from data (set by buildGraphFromSpans) or fall back to node_type
  const nodeColor = data.nodeColor || getSpanNodeColor(data.node_type || data.nodeType);
  const icon = getSpanNodeIcon(data.node_type || data.nodeType);
  const hasMetrics = data.cost_usd || data.cost || data.tokens_in || data.tokensIn || data.tokens_out || data.tokensOut;

  // Build display label - use label from data if available
  let label = data.label || data.name;
  if (!data.label) {
    if (data.turn_number || data.turnNumber) {
      label = `Turn #${data.turn_number || data.turnNumber}`;
    } else if (data.tool_name) {
      label = `${data.tool_name}`;
    }
  }

  // Get metrics values (handle both naming conventions)
  const cost = data.cost_usd || data.cost || 0;
  const tokensIn = data.tokens_in || data.tokensIn || 0;
  const tokensOut = data.tokens_out || data.tokensOut || 0;
  const durationMs = data.duration_ms || data.durationMs || 0;

  // Expand/collapse state
  const isExpandable = data.isExpandable;
  const isExpanded = data.isExpanded;
  const childCount = data.childCount || 0;

  const handleExpandClick = (e: React.MouseEvent) => {
    e.stopPropagation();  // Don't trigger node click
    if (data.onToggleExpand) {
      data.onToggleExpand();
    }
  };

  return (
    <div
      className={styles.rfSpanNode}
      style={{
        borderLeftColor: nodeColor,
        marginLeft: `${(data.depth || 0) * 8}px`,
      }}
    >
      <Handle type="target" position={Position.Top} className={styles.rfHandle} />

      {/* Header with icon and name */}
      <div className={styles.rfSpanHeader}>
        <span className={styles.rfSpanIcon} style={{ color: nodeColor }}>{icon}</span>
        <span className={styles.rfSpanLabel} title={data.name}>
          {label.length > 30 ? label.substring(0, 30) + '...' : label}
        </span>
        {data.status === 'error' && (
          <span className={styles.rfSpanError}>!</span>
        )}
        {/* Expand/collapse button */}
        {isExpandable && (
          <button
            className={styles.rfExpandBtn}
            onClick={handleExpandClick}
            title={isExpanded ? `Collapse (${childCount} items)` : `Expand (${childCount} items)`}
          >
            {isExpanded ? '−' : '+'}
            {!isExpanded && childCount > 0 && (
              <span className={styles.rfExpandCount}>{childCount}</span>
            )}
          </button>
        )}
      </div>

      {/* Duration and metrics */}
      <div className={styles.rfSpanMeta}>
        <span>{formatDuration(durationMs)}</span>
        {hasMetrics && (
          <>
            {cost > 0 && <span> • {formatCost(cost)}</span>}
            {tokensIn > 0 && <span> [{formatTokens(tokensIn)}→{formatTokens(tokensOut)}]</span>}
          </>
        )}
      </div>

      <Handle type="source" position={Position.Bottom} className={styles.rfHandle} />
    </div>
  );
};

const nodeTypes = {
  task: TaskNodeComponent,
  span: SpanNodeComponent,
};

// Node dimensions for dagre layout
const TASK_NODE_WIDTH = 240;
const TASK_NODE_HEIGHT = 100;
const SPAN_NODE_WIDTH = 180;
const SPAN_NODE_HEIGHT = 50;

// Apply dagre layout to nodes and edges
function applyDagreLayout(
  nodes: Node[],
  edges: Edge[],
  direction: 'TB' | 'LR' = 'TB'
): Node[] {
  if (nodes.length === 0) return nodes;

  const g = new dagre.graphlib.Graph();
  g.setGraph({
    rankdir: direction,
    nodesep: 40,        // Horizontal separation between nodes
    ranksep: 60,        // Vertical separation between ranks
    marginx: 50,
    marginy: 50,
    align: 'UL',        // Align nodes to upper-left for consistent layout
  });
  g.setDefaultEdgeLabel(() => ({}));

  // Add nodes with dimensions based on type
  nodes.forEach(node => {
    const isSpan = node.type === 'span';
    g.setNode(node.id, {
      width: isSpan ? SPAN_NODE_WIDTH : TASK_NODE_WIDTH,
      height: isSpan ? SPAN_NODE_HEIGHT : TASK_NODE_HEIGHT,
    });
  });

  // Add edges - dagre uses these to determine hierarchy
  edges.forEach(edge => {
    g.setEdge(edge.source, edge.target);
  });

  // Run dagre layout
  dagre.layout(g);

  // Apply calculated positions to nodes
  return nodes.map(node => {
    const nodeWithPosition = g.node(node.id);
    if (!nodeWithPosition) return node;
    const isSpan = node.type === 'span';
    const width = isSpan ? SPAN_NODE_WIDTH : TASK_NODE_WIDTH;
    const height = isSpan ? SPAN_NODE_HEIGHT : TASK_NODE_HEIGHT;
    return {
      ...node,
      position: {
        x: nodeWithPosition.x - width / 2,
        y: nodeWithPosition.y - height / 2,
      },
    };
  });
}

// Recursively add span nodes and edges
function addSpanNodes(
  parentId: string,
  spans: TaskSpanNode[],
  nodes: Node[],
  edges: Edge[],
  depth: number = 0
): void {
  spans.forEach((span, index) => {
    const nodeId = `span-${span.id}`;

    // Add span node
    nodes.push({
      id: nodeId,
      type: 'span',
      position: { x: 0, y: 0 }, // Will be calculated by dagre
      data: {
        ...span,
        depth,
      },
    });

    // Add edge from parent to this span
    edges.push({
      id: `e-${parentId}-${nodeId}`,
      source: parentId,
      target: nodeId,
      type: 'default',
      style: {
        stroke: '#4b5563',
        strokeWidth: 1,
      },
      markerEnd: {
        type: MarkerType.ArrowClosed,
        color: '#4b5563',
        width: 12,
        height: 12,
      },
    });

    // Recursively add children
    if (span.children && span.children.length > 0) {
      addSpanNodes(nodeId, span.children, nodes, edges, depth + 1);
    }
  });
}

// Build ReactFlow graph data from task hierarchy
function buildGraphData(
  result: TaskHierarchyResult | null,
  selectedId?: string | null,
  onTaskClick?: (task: TaskHierarchyNode) => void,
  expandedTasks?: Set<string>,
  onToggleExpand?: (taskId: string) => void
): { rfNodes: Node[]; rfEdges: Edge[] } {
  if (!result || !result.tasks || result.tasks.length === 0) {
    return { rfNodes: [], rfEdges: [] };
  }

  const rfNodes: Node[] = [];
  const rfEdges: Edge[] = [];
  const handoffTargets = new Set<string>();

  // Identify handoff targets from edges
  result.edges.forEach(edge => {
    if (edge.type === 'handoff') {
      handoffTargets.add(edge.target);
    }
  });

  // Create nodes from tasks
  result.tasks.forEach(task => {
    const isExpanded = expandedTasks?.has(task.id) || false;

    rfNodes.push({
      id: task.id,
      type: 'task',
      position: { x: 0, y: 0 }, // Will be calculated by dagre
      data: {
        ...task,
        isHandoffTarget: handoffTargets.has(task.id),
        isExpanded,
        onToggleExpand,
        onTaskClick,
      },
      selected: selectedId === task.id,
    });

    // If task is expanded and has spans, add span nodes
    if (isExpanded && task.spans && task.spans.length > 0) {
      addSpanNodes(task.id, task.spans, rfNodes, rfEdges, 0);
    }
  });

  // Create edges from relationships - ONLY if both source and target nodes exist
  const nodeIds = new Set(rfNodes.map(n => n.id));
  result.edges.forEach(edge => {
    // Skip edges where source or target don't exist in our node set
    if (!nodeIds.has(edge.source) || !nodeIds.has(edge.target)) {
      return;
    }

    const isHandoff = edge.type === 'handoff';
    const isSession = edge.type === 'session';

    rfEdges.push({
      id: `e-${edge.source}-${edge.target}-${edge.type}`,
      source: edge.source,
      target: edge.target,
      type: 'default', // Use default edge type for reliability
      animated: isSession,
      style: {
        stroke: isHandoff ? '#f59e0b' : '#3b82f6',
        strokeWidth: isHandoff ? 3 : 2,
        strokeDasharray: isSession ? '5,5' : undefined,
      },
      label: isHandoff ? 'handoff' : 'session',
      labelStyle: {
        fill: '#94a3b8',
        fontSize: 10,
        fontWeight: 500,
      },
      labelBgStyle: {
        fill: '#0d1117',
        fillOpacity: 0.8,
      },
      markerEnd: {
        type: MarkerType.ArrowClosed,
        color: isHandoff ? '#f59e0b' : '#3b82f6',
      },
    });
  });

  // Apply dagre layout
  const layoutedNodes = applyDagreLayout(rfNodes, rfEdges, 'TB');

  return { rfNodes: layoutedNodes, rfEdges };
}

export interface TaskHierarchyGraphProps {
  selectedNodeId?: string | null;
  onNodeClick?: (node: HierarchyNode, event?: React.MouseEvent) => void;  // Changed to HierarchyNode for popover
  isExpanded?: boolean;
  // Recenter trigger - changes when we need to recenter
  recenterTrigger?: number;
  // Filters (from parent ControlPlane filters)
  workspace?: string;
  provider?: string;
  // Filter to specific task ID (from selection in Event Queue)
  filterTaskId?: string | null;
  // Filter to multiple task IDs (extracted from loaded spans)
  spanTaskIds?: string[];
  // Filter to tasks with spans in this trace (from selection in Event Queue)
  filterTraceId?: string | null;
  // NEW: Spans prop - when provided, renders from spans (same data source as Tree/Timeline/Chat)
  // This is the FIX for filtering - uses already-loaded spans instead of separate API calls
  spans?: Span[];
  // Use API turn grouping (via group_by=turns) instead of client-side grouping
  // This provides consistent results between CLI and dashboard
  useTurnGrouping?: boolean;
  // Task ID to fetch with turn grouping (when useTurnGrouping is true)
  taskIdForTurnGrouping?: string;
  // Span type filtering - Set of span names to hide (e.g., 'api_request')
  // Same as other views (Tree, Timeline, Chat, Waterfall)
  hiddenSpanTypes?: Set<string>;
}

// Inner component that uses ReactFlow hooks
const TaskHierarchyGraphInner: React.FC<TaskHierarchyGraphProps & {
  data: TaskHierarchyResult | null;
  loading: boolean;
  error: string | null;
}> = ({
  data,
  loading,
  error,
  selectedNodeId,
  onNodeClick,
  isExpanded,
  recenterTrigger,
}) => {
  const reactFlowRef = useRef<ReactFlowInstance | null>(null);

  // Track which tasks have their spans expanded
  const [expandedTasks, setExpandedTasks] = useState<Set<string>>(new Set());

  // Toggle span expansion for a task
  const handleToggleExpand = useCallback((taskId: string) => {
    setExpandedTasks(prev => {
      const next = new Set(prev);
      if (next.has(taskId)) {
        next.delete(taskId);
      } else {
        next.add(taskId);
      }
      return next;
    });
  }, []);

  // Build graph data
  const { rfNodes: initialNodes, rfEdges: initialEdges } = useMemo(
    () => buildGraphData(data, selectedNodeId, onNodeClick, expandedTasks, handleToggleExpand),
    [data, selectedNodeId, onNodeClick, expandedTasks, handleToggleExpand]
  );

  const [rfNodes, setNodes, onNodesChange] = useNodesState(initialNodes);
  const [rfEdges, setEdges, onEdgesChange] = useEdgesState(initialEdges);

  // Update when data or expanded state changes
  useEffect(() => {
    const { rfNodes, rfEdges } = buildGraphData(data, selectedNodeId, onNodeClick, expandedTasks, handleToggleExpand);
    setNodes(rfNodes);
    setEdges(rfEdges);
  }, [data, selectedNodeId, onNodeClick, expandedTasks, handleToggleExpand, setNodes, setEdges]);

  // Handle node click
  const handleNodeClick = useCallback(
    (_: React.MouseEvent, node: Node) => {
      if (onNodeClick && node.data) {
        onNodeClick(node.data as TaskHierarchyNode);
      }
    },
    [onNodeClick]
  );

  // Fit view on init
  const onInit = useCallback((instance: ReactFlowInstance) => {
    reactFlowRef.current = instance;
    setTimeout(() => instance.fitView({ padding: 0.2 }), 100);
  }, []);

  // Refit view when expand/collapse changes or panel resizes
  useEffect(() => {
    if (reactFlowRef.current) {
      setTimeout(() => {
        reactFlowRef.current?.fitView({
          padding: 0.2,
          duration: 300,
          maxZoom: 1.5,
        });
      }, 50);
    }
  }, [recenterTrigger, isExpanded, expandedTasks]);

  if (loading) {
    return (
      <div className={styles.treeLoading}>
        <div className={styles.loadingSpinner} />
        <span>Loading task hierarchy...</span>
      </div>
    );
  }

  if (error) {
    return (
      <div className={styles.treeError}>
        <span className={styles.errorIcon}>\u26A0</span>
        <span>{error}</span>
      </div>
    );
  }

  if (!data || data.tasks.length === 0) {
    return (
      <div className={styles.treeEmpty}>
        <span className={styles.emptyIcon}>\u2B21</span>
        <div className={styles.emptyTitle}>No Tasks Yet</div>
        <div className={styles.emptyText}>
          Tasks will appear here as coordinator executes work.
        </div>
      </div>
    );
  }

  return (
    <ReactFlow
      nodes={rfNodes}
      edges={rfEdges}
      onNodesChange={onNodesChange}
      onEdgesChange={onEdgesChange}
      onNodeClick={handleNodeClick}
      onInit={onInit}
      nodeTypes={nodeTypes}
      minZoom={0.2}
      maxZoom={2}
      proOptions={{ hideAttribution: true }}
    >
      <Background color="#1e293b" gap={24} size={1} />
      <Controls className={styles.graphControls} />
      {isExpanded && (
        <MiniMap
          nodeColor={(node) => getNodeColor(node.data?.status, node.data?.approval_status)}
          maskColor="rgba(13, 17, 23, 0.8)"
        />
      )}
    </ReactFlow>
  );
};

// Inner component for rendering from spans (the fixed path)
// This renders directly from buildGraphFromSpans output without going through coordinator API
interface SpanGraphInnerProps {
  graphData: ReturnType<typeof buildGraphFromSpans>;
  selectedNodeId?: string | null;
  onNodeClick?: (node: HierarchyNode, event?: React.MouseEvent) => void;  // Changed to HierarchyNode for popover
  isExpanded?: boolean;
  recenterTrigger?: number;
}

const SpanGraphInner: React.FC<SpanGraphInnerProps> = ({
  graphData,
  selectedNodeId,
  onNodeClick,
  isExpanded,
  recenterTrigger,
}) => {
  const reactFlowRef = useRef<ReactFlowInstance | null>(null);

  // Track which nodes are expanded (start collapsed - only root nodes visible)
  const [expandedNodeIds, setExpandedNodeIds] = useState<Set<string>>(() => {
    // Start with root nodes expanded to show turns
    return new Set(graphData.rootNodeIds || []);
  });

  // Toggle expand/collapse for a node (defined early for use in useMemo)
  const handleToggleExpand = useCallback((nodeId: string) => {
    setExpandedNodeIds(prev => {
      const next = new Set(prev);
      if (next.has(nodeId)) {
        next.delete(nodeId);
      } else {
        next.add(nodeId);
      }
      return next;
    });
  }, []);

  // Filter nodes and edges based on expansion state
  const { visibleNodes, visibleEdges } = useMemo(() => {
    const expandedSet = expandedNodeIds;
    const rootSet = new Set(graphData.rootNodeIds || []);

    // A node is visible if:
    // 1. It's a root node, OR
    // 2. Its parent is expanded
    const visibleNodeIds = new Set<string>();

    for (const node of graphData.nodes) {
      const isRoot = rootSet.has(node.id);
      const parentId = node.data?.parentId;
      const parentExpanded = parentId ? expandedSet.has(parentId) : true;

      if (isRoot || parentExpanded) {
        visibleNodeIds.add(node.id);
      }
    }

    const visibleNodes = graphData.nodes
      .filter(node => visibleNodeIds.has(node.id))
      .map(node => ({
        ...node,
        data: {
          ...node.data,
          // Add expansion state to node data for rendering expand button
          isExpanded: expandedSet.has(node.id),
          onToggleExpand: () => handleToggleExpand(node.id),
        },
      }));

    // Only show edges where both source and target are visible
    const visibleEdges = graphData.edges.filter(
      edge => visibleNodeIds.has(edge.source) && visibleNodeIds.has(edge.target)
    );

    return { visibleNodes, visibleEdges };
  }, [graphData, expandedNodeIds]);

  // Apply layout to visible nodes
  const layoutedData = useMemo(() => {
    if (visibleNodes.length === 0) return { nodes: [], edges: visibleEdges };

    // Re-apply dagre layout to visible nodes
    const g = new dagre.graphlib.Graph();
    g.setGraph({ rankdir: 'TB', nodesep: 25, ranksep: 40, marginx: 30, marginy: 30 });
    g.setDefaultEdgeLabel(() => ({}));

    visibleNodes.forEach(node => {
      const isTurn = node.data?.nodeType === 'turn';
      g.setNode(node.id, { width: isTurn ? 220 : 180, height: isTurn ? 70 : 60 });
    });

    visibleEdges.forEach(edge => {
      g.setEdge(edge.source, edge.target);
    });

    dagre.layout(g);

    const layoutedNodes = visibleNodes.map(node => {
      const pos = g.node(node.id);
      if (!pos) return node;
      const isTurn = node.data?.nodeType === 'turn';
      const width = isTurn ? 220 : 180;
      const height = isTurn ? 70 : 60;
      return {
        ...node,
        position: { x: pos.x - width / 2, y: pos.y - height / 2 },
      };
    });

    return { nodes: layoutedNodes, edges: visibleEdges };
  }, [visibleNodes, visibleEdges]);

  const [rfNodes, setNodes, onNodesChange] = useNodesState(layoutedData.nodes);
  const [rfEdges, setEdges, onEdgesChange] = useEdgesState(layoutedData.edges);

  // Update when layout data changes
  useEffect(() => {
    setNodes(layoutedData.nodes);
    setEdges(layoutedData.edges);
  }, [layoutedData, setNodes, setEdges]);

  // Handle node click - convert to HierarchyNode format for popover
  const handleNodeClick = useCallback(
    (event: React.MouseEvent, node: Node) => {
      if (onNodeClick && node.data) {
        // Build HierarchyNode with _span for popover compatibility
        const hierarchyNode: HierarchyNode = {
          id: node.id,
          type: node.data.nodeType === 'turn' ? 'turn' : node.data.nodeType === 'tool' ? 'tool_use' : 'exec',
          label: node.data.label || node.data.name,
          status: (node.data.status === 'ok' || node.data.status === 'completed') ? 'completed' :
                  node.data.status === 'error' ? 'error' : 'unknown',
          durationMs: node.data.durationMs || 0,
          cost: node.data.cost || 0,
          tokensIn: node.data.tokensIn || 0,
          tokensOut: node.data.tokensOut || 0,
          provider: node.data.provider,
          turnNumber: node.data.turnNumber,
          // Include _span for popover detail display
          _span: node.data._span,
        };
        // Pass both node and event for popover positioning
        onNodeClick(hierarchyNode, event);
      }
    },
    [onNodeClick]
  );

  // Handle double-click to expand/collapse
  const handleNodeDoubleClick = useCallback(
    (_: React.MouseEvent, node: Node) => {
      if (node.data?.isExpandable) {
        handleToggleExpand(node.id);
      }
    },
    [handleToggleExpand]
  );

  // Fit view on init
  const onInit = useCallback((instance: ReactFlowInstance) => {
    reactFlowRef.current = instance;
    setTimeout(() => instance.fitView({ padding: 0.2 }), 100);
  }, []);

  // Refit view when panel resizes or expansion changes
  useEffect(() => {
    if (reactFlowRef.current) {
      setTimeout(() => {
        reactFlowRef.current?.fitView({
          padding: 0.2,
          duration: 300,
          maxZoom: 1.5,
        });
      }, 50);
    }
  }, [recenterTrigger, isExpanded, expandedNodeIds]);

  if (graphData.nodes.length === 0) {
    return (
      <div className={styles.treeEmpty}>
        <span className={styles.emptyIcon}>&#x2B21;</span>
        <div className={styles.emptyTitle}>Select an Event</div>
        <div className={styles.emptyText}>
          Click an event in the queue above to view its execution graph.
        </div>
      </div>
    );
  }

  return (
    <ReactFlow
      nodes={rfNodes}
      edges={rfEdges}
      onNodesChange={onNodesChange}
      onEdgesChange={onEdgesChange}
      onNodeClick={handleNodeClick}
      onNodeDoubleClick={handleNodeDoubleClick}
      onInit={onInit}
      nodeTypes={nodeTypes}
      minZoom={0.2}
      maxZoom={2}
      proOptions={{ hideAttribution: true }}
    >
      <Background color="#1e293b" gap={24} size={1} />
      <Controls className={styles.graphControls} />
      {isExpanded && (
        <MiniMap
          nodeColor={(node) => node.data?.nodeColor || '#64748b'}
          maskColor="rgba(13, 17, 23, 0.8)"
        />
      )}
    </ReactFlow>
  );
};

// Stats bar component
const StatsBar: React.FC<{ stats: TaskHierarchyResult['stats'] }> = ({ stats }) => (
  <div className={styles.taskStatsBar}>
    <div className={styles.taskStat}>
      <span className={styles.taskStatValue}>{stats.total_tasks}</span>
      <span className={styles.taskStatLabel}>Tasks</span>
    </div>
    {stats.total_spans > 0 && (
      <div className={styles.taskStat}>
        <span className={styles.taskStatValue}>{stats.total_spans}</span>
        <span className={styles.taskStatLabel}>Spans</span>
      </div>
    )}
    {stats.pending_approvals > 0 && (
      <div className={styles.taskStat} style={{ color: '#f59e0b' }}>
        <span className={styles.taskStatValue}>{stats.pending_approvals}</span>
        <span className={styles.taskStatLabel}>Pending</span>
      </div>
    )}
    {stats.total_cost > 0 && (
      <div className={styles.taskStat}>
        <span className={styles.taskStatValue}>{formatCost(stats.total_cost)}</span>
        <span className={styles.taskStatLabel}>Cost</span>
      </div>
    )}
  </div>
);

// Wrapper that provides ReactFlowProvider and fetches data
export const TaskHierarchyGraph: React.FC<TaskHierarchyGraphProps> = (props) => {
  const { spans, useTurnGrouping, taskIdForTurnGrouping } = props;

  // Fetch with API turn grouping when enabled and we have a task ID
  const { data: turnGroupedData, loading: turnGroupedLoading } = useTaskHierarchy({
    taskId: useTurnGrouping && taskIdForTurnGrouping ? taskIdForTurnGrouping : undefined,
    groupBy: useTurnGrouping ? 'turns' : undefined,
    skip: !useTurnGrouping || !taskIdForTurnGrouping,
    refreshInterval: 0, // Don't auto-refresh for single task view
  });

  // When spans are provided, build graph from them directly (same data source as Tree/Timeline/Chat)
  // This is the FIX: no separate API calls to coordinator.db that have ID mismatches
  const graphFromSpans = useMemo(() => {
    // Priority 1: Use API turn-grouped data if available (from useTaskHierarchy with group_by=turns)
    if (useTurnGrouping && turnGroupedData?.tasks?.length) {
      const task = turnGroupedData.tasks[0];
      if (task.turn_grouped) {
        const graphData = buildGraphFromTurnGrouped(task.turn_grouped, props.selectedNodeId);
        const result: TaskHierarchyResult = {
          tasks: turnGroupedData.tasks,
          edges: turnGroupedData.edges,
          stats: {
            total_tasks: turnGroupedData.stats.total_tasks,
            total_spans: graphData.stats.totalSpans,
            pending_approvals: turnGroupedData.stats.pending_approvals,
            total_cost: graphData.stats.totalCost,
          },
        };
        return { graphData, result };
      }
    }

    // Priority 2: Build from spans prop (client-side turn grouping)
    if (!spans || spans.length === 0) return null;

    // Convert Span[] to HierarchySpan[] for buildGraphFromSpans
    const hierarchySpans = spans as unknown as HierarchySpan[];
    const graphData = buildGraphFromSpans(hierarchySpans, props.selectedNodeId, props.hiddenSpanTypes);

    // Convert to TaskHierarchyResult format for stats display
    const result: TaskHierarchyResult = {
      tasks: [],
      edges: [],
      stats: {
        total_tasks: 0,
        total_spans: graphData.stats.totalSpans,
        pending_approvals: 0,
        total_cost: graphData.stats.totalCost,
      },
    };

    return { graphData, result };
  }, [spans, props.selectedNodeId, props.hiddenSpanTypes, useTurnGrouping, turnGroupedData]);

  // NO FALLBACK to useTaskHierarchy - if no spans, show empty state
  // This prevents the overwhelming "show everything" behavior
  const hasSpans = spans && spans.length > 0;
  const hasApiTurnGrouped = useTurnGrouping && turnGroupedData?.tasks?.length && turnGroupedData.tasks[0].turn_grouped;
  const hasData = hasSpans || hasApiTurnGrouped;

  return (
    <div className={`${styles.graphContainer} ${props.isExpanded ? styles.graphContainerExpanded : ''}`}>
      {/* Stats bar - only show when we have data */}
      {graphFromSpans?.result?.stats && (
        <StatsBar stats={graphFromSpans.result.stats} />
      )}

      <ReactFlowProvider>
        {turnGroupedLoading ? (
          // Loading state for API turn grouping
          <div className={styles.treeLoading}>
            <div className={styles.loadingSpinner} />
            <span>Loading turn hierarchy...</span>
          </div>
        ) : hasData && graphFromSpans ? (
          <SpanGraphInner
            graphData={graphFromSpans.graphData}
            selectedNodeId={props.selectedNodeId}
            onNodeClick={props.onNodeClick}
            isExpanded={props.isExpanded}
            recenterTrigger={props.recenterTrigger}
          />
        ) : (
          // Empty state when no event selected
          <div className={styles.treeEmpty}>
            <span className={styles.emptyIcon}>&#x2B21;</span>
            <div className={styles.emptyTitle}>Select an Event</div>
            <div className={styles.emptyText}>
              Click an event in the queue above to view its execution graph.
            </div>
          </div>
        )}
      </ReactFlowProvider>
    </div>
  );
};

export default TaskHierarchyGraph;
