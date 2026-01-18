/**
 * ExecHierarchyGraph - ReactFlow graph view for the exec hierarchy
 * Visualizes: Messages -> Execs -> Turns -> Tool Uses as a directed graph
 * Uses dagre for automatic hierarchical layout
 */
import React, { useMemo, useCallback, useRef, useEffect } from 'react';
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
import type { HierarchyNode, NodeStatus } from './types';
import styles from './ExecHierarchy.module.css';

// Get node color based on type
function getNodeColor(type: HierarchyNode['type']): string {
  switch (type) {
    case 'message':
      return '#f59e0b';
    case 'exec':
      return '#25c2a0';
    case 'turn':
      return '#3b82f6';
    case 'tool_use':
      return '#8b5cf6';
    case 'approval':
      return '#10b981'; // Emerald for human decisions
    default:
      return '#64748b';
  }
}

// Get status color
function getStatusColor(status: NodeStatus): string {
  switch (status) {
    case 'completed':
      return '#25c2a0';
    case 'busy':
      return '#3b82f6';
    case 'error':
      return '#ef4444';
    case 'pending':
      return '#f59e0b';
    default:
      return '#64748b';
  }
}

// Get icon for node type
function getNodeIcon(type: HierarchyNode['type']): string {
  switch (type) {
    case 'message':
      return '✉';
    case 'exec':
      return '⚡';
    case 'turn':
      return '↻';
    case 'tool_use':
      return '⚙';
    case 'approval':
      return '👤';
    default:
      return '●';
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

// Get provider icon
function getProviderIcon(provider?: string): string {
  switch (provider) {
    case 'claude': return '🟠';
    case 'gemini': return '🔵';
    case 'ollama': return '🟣';
    default: return '';
  }
}

// Get approval status icon
function getApprovalIcon(status?: string): string {
  switch (status) {
    case 'pending': return '⏳';
    case 'approved': return '✓';
    case 'rejected': return '✗';
    default: return '';
  }
}

// Get approval status color
function getApprovalColor(status?: string): string {
  switch (status) {
    case 'pending': return '#f59e0b';  // Amber
    case 'approved': return '#25c2a0';  // Green
    case 'rejected': return '#ef4444';  // Red
    default: return '#64748b';          // Gray
  }
}

// Custom node component
const HierarchyNodeComponent: React.FC<NodeProps> = ({ data, selected }) => {
  const nodeColor = getNodeColor(data.type);
  const statusColor = getStatusColor(data.status);
  const isCollapsible = data.isCollapsible;
  const isExpanded = data.isExpanded;
  const childCount = data.childCount || 0;
  const hasMetrics = data.cost || data.tokensIn || data.tokensOut;
  const hasApproval = data.approvalStatus && data.approvalStatus !== 'none';
  const isCoordinator = data.semanticType === 'coordinator';
  const isFiltered = data.isFiltered;

  const handleToggleClick = (e: React.MouseEvent) => {
    e.stopPropagation();
    if (data.onToggleExpand && isCollapsible) {
      data.onToggleExpand(data.id);
    }
  };

  return (
    <div
      className={`${styles.rfNode} ${selected ? styles.rfNodeSelected : ''} ${data.provider ? styles[`rfNodeProvider${data.provider.charAt(0).toUpperCase() + data.provider.slice(1)}`] : ''} ${isCoordinator ? styles.rfNodeCoordinator : ''} ${isFiltered ? styles.rfNodeFiltered : ''}`}
      style={{ borderLeftColor: nodeColor }}
      data-span-id={data._span?.id || data.id}
    >
      <Handle type="target" position={Position.Top} className={styles.rfHandle} />
      <div className={styles.rfNodeHeader}>
        {data.provider && (
          <span className={styles.rfProviderIcon}>{getProviderIcon(data.provider)}</span>
        )}
        <span className={styles.rfNodeIcon} style={{ color: nodeColor }}>
          {getNodeIcon(data.type)}
        </span>
        <span className={styles.rfNodeLabel}>{data.label}</span>
        {hasApproval && (
          <span
            className={styles.rfApprovalBadge}
            style={{ backgroundColor: getApprovalColor(data.approvalStatus) }}
            title={`Approval: ${data.approvalStatus}`}
          >
            {getApprovalIcon(data.approvalStatus)}
          </span>
        )}
        {isCollapsible && (
          <button
            className={styles.rfCollapseBtn}
            onClick={handleToggleClick}
            title={isExpanded ? 'Collapse' : 'Expand'}
          >
            {isExpanded ? '−' : '+'}
          </button>
        )}
      </div>
      <div className={styles.rfNodeMeta}>
        <span style={{ color: statusColor }}>{data.status}</span>
        {data.durationMs > 0 && <span> • {formatDuration(data.durationMs)}</span>}
        {data.cost > 0 && <span className={styles.rfCost}> • {formatCost(data.cost)}</span>}
        {!isExpanded && childCount > 0 && (
          <span className={styles.rfChildBadge}>{childCount}</span>
        )}
      </div>
      {/* Agent/Task context for coordinator nodes */}
      {isCoordinator && data.agentId && (
        <div className={styles.rfNodeContext}>
          <span className={styles.rfAgentLabel}>Agent: {data.agentId}</span>
        </div>
      )}
      {hasMetrics && (data.tokensIn > 0 || data.tokensOut > 0) && (
        <div className={styles.rfNodeMetrics}>
          {data.tokensIn > 0 && <span>{formatTokens(data.tokensIn)} in</span>}
          {data.tokensIn > 0 && data.tokensOut > 0 && <span> / </span>}
          {data.tokensOut > 0 && <span>{formatTokens(data.tokensOut)} out</span>}
        </div>
      )}
      <Handle type="source" position={Position.Bottom} className={styles.rfHandle} />
    </div>
  );
};

const nodeTypes = {
  hierarchy: HierarchyNodeComponent,
};

// Node dimensions for dagre layout
const NODE_WIDTH = 200;
const NODE_HEIGHT = 80;

// Apply dagre layout to nodes and edges
// direction: 'TB' = top-bottom, 'LR' = left-right
function applyDagreLayout(
  nodes: Node[],
  edges: Edge[],
  direction: 'TB' | 'LR' = 'TB'
): Node[] {
  const g = new dagre.graphlib.Graph();
  g.setGraph({
    rankdir: direction,
    nodesep: 60,       // Horizontal separation between nodes
    ranksep: 100,      // Vertical separation between ranks
    marginx: 20,
    marginy: 20,
  });
  g.setDefaultEdgeLabel(() => ({}));

  // Add nodes with dimensions
  nodes.forEach(node => {
    g.setNode(node.id, { width: NODE_WIDTH, height: NODE_HEIGHT });
  });

  // Add edges
  edges.forEach(edge => {
    g.setEdge(edge.source, edge.target);
  });

  // Run dagre layout
  dagre.layout(g);

  // Apply calculated positions to nodes
  return nodes.map(node => {
    const nodeWithPosition = g.node(node.id);
    return {
      ...node,
      position: {
        x: nodeWithPosition.x - NODE_WIDTH / 2,
        y: nodeWithPosition.y - NODE_HEIGHT / 2,
      },
    };
  });
}

// Flatten hierarchy nodes to ReactFlow nodes and edges
// Uses dagre for automatic hierarchical layout
function buildGraphData(
  nodes: HierarchyNode[],
  selectedId?: string | null,
  expandedNodes?: Set<string>,
  onToggleExpand?: (nodeId: string) => void
): { rfNodes: Node[]; rfEdges: Edge[] } {
  const rfNodes: Node[] = [];
  const rfEdges: Edge[] = [];
  const seen = new Set<string>();

  // Collect all visible nodes and edges
  function collectNodes(
    node: HierarchyNode,
    parentId: string | null,
    prevTurnId: string | null
  ): string | null {
    if (seen.has(node.id)) return prevTurnId;
    seen.add(node.id);

    const isExpanded = expandedNodes?.has(node.id) ?? false;
    const isTurn = node.type === 'turn';

    // Add node (position will be set by dagre)
    rfNodes.push({
      id: node.id,
      type: 'hierarchy',
      position: { x: 0, y: 0 }, // Will be calculated by dagre
      data: {
        ...node,
        selected: selectedId === node.id,
        isExpanded,
        onToggleExpand,
      },
      selected: selectedId === node.id,
    });

    // Create edge from parent
    if (parentId) {
      rfEdges.push({
        id: `${parentId}-${node.id}`,
        source: parentId,
        target: node.id,
        type: 'smoothstep',
        style: {
          stroke: getNodeColor(node.type),
          strokeWidth: 2,
        },
        markerEnd: {
          type: MarkerType.ArrowClosed,
          color: getNodeColor(node.type),
        },
      });
    }

    // Create sequential edge between turns (dashed)
    if (isTurn && prevTurnId) {
      rfEdges.push({
        id: `seq-${prevTurnId}-${node.id}`,
        source: prevTurnId,
        target: node.id,
        type: 'straight',
        style: {
          stroke: '#3b82f6',
          strokeWidth: 2,
          strokeDasharray: '5,5',
        },
        markerEnd: {
          type: MarkerType.ArrowClosed,
          color: '#3b82f6',
        },
      });
    }

    let lastTurnId = isTurn ? node.id : prevTurnId;

    // Process children if expanded
    if (isExpanded && node.children && node.children.length > 0) {
      // Separate turns for sequential edges
      const turns = node.children.filter(c => c.type === 'turn');
      const nonTurns = node.children.filter(c => c.type !== 'turn');

      // Process turns (maintain sequential edges)
      let currentTurnId: string | null = null;
      for (const turn of turns) {
        currentTurnId = collectNodes(turn, node.id, currentTurnId);
      }
      if (currentTurnId) lastTurnId = currentTurnId;

      // Process non-turn children
      for (const child of nonTurns) {
        collectNodes(child, node.id, null);
      }
    }

    return lastTurnId;
  }

  // Collect all root nodes
  nodes.forEach(node => {
    collectNodes(node, null, null);
  });

  // Apply dagre layout
  const layoutedNodes = applyDagreLayout(rfNodes, rfEdges, 'TB');

  return { rfNodes: layoutedNodes, rfEdges };
}

export interface ExecHierarchyGraphProps {
  nodes: HierarchyNode[];
  selectedNodeId?: string | null;
  onNodeClick?: (node: HierarchyNode) => void;
  loading?: boolean;
  error?: string | null;
  isExpanded?: boolean;
  // Collapsibility
  expandedNodes?: Set<string>;
  onToggleNodeExpand?: (nodeId: string) => void;
  // Recenter trigger - changes when we need to recenter
  recenterTrigger?: number;
}

// Inner component that uses ReactFlow hooks
const ExecHierarchyGraphInner: React.FC<ExecHierarchyGraphProps> = ({
  nodes,
  selectedNodeId,
  onNodeClick,
  loading,
  error,
  isExpanded,
  expandedNodes,
  onToggleNodeExpand,
  recenterTrigger,
}) => {
  const reactFlowRef = useRef<ReactFlowInstance | null>(null);

  // Build graph data with collapsibility
  const { rfNodes: initialNodes, rfEdges: initialEdges } = useMemo(
    () => buildGraphData(nodes, selectedNodeId, expandedNodes, onToggleNodeExpand),
    [nodes, selectedNodeId, expandedNodes, onToggleNodeExpand]
  );

  const [rfNodes, setNodes, onNodesChange] = useNodesState(initialNodes);
  const [rfEdges, setEdges, onEdgesChange] = useEdgesState(initialEdges);

  // Update when data changes (including expanded state)
  useEffect(() => {
    const { rfNodes, rfEdges } = buildGraphData(nodes, selectedNodeId, expandedNodes, onToggleNodeExpand);
    setNodes(rfNodes);
    setEdges(rfEdges);
  }, [nodes, selectedNodeId, expandedNodes, onToggleNodeExpand, setNodes, setEdges]);

  // Handle node click
  const handleNodeClick = useCallback(
    (_: React.MouseEvent, node: Node) => {
      if (onNodeClick && node.data) {
        onNodeClick(node.data as HierarchyNode);
      }
    },
    [onNodeClick]
  );

  // Fit view only on init (once) - NOT on data refresh to preserve user zoom/pan
  const onInit = useCallback((instance: ReactFlowInstance) => {
    reactFlowRef.current = instance;
    // Single fitView on init only
    setTimeout(() => instance.fitView({ padding: 0.2 }), 100);
  }, []);

  // Refit view when expand/collapse changes (recenterTrigger) or panel expand/collapse (isExpanded)
  useEffect(() => {
    if (reactFlowRef.current) {
      setTimeout(() => {
        reactFlowRef.current?.fitView({
          padding: 0.2,
          duration: 300,  // Smooth animation
          maxZoom: 1.5,   // Don't zoom in too much
        });
      }, 50);
    }
  }, [recenterTrigger, isExpanded]); // Recenter when triggered or panel resizes

  return (
    <ReactFlow
      nodes={rfNodes}
      edges={rfEdges}
      onNodesChange={onNodesChange}
      onEdgesChange={onEdgesChange}
      onNodeClick={handleNodeClick}
      onInit={onInit}
      nodeTypes={nodeTypes}
      // Removed fitView prop - using onInit fitView only to preserve user zoom/pan
      minZoom={0.3}
      maxZoom={2}
      proOptions={{ hideAttribution: true }}
    >
      <Background color="#1e293b" gap={24} size={1} />
      <Controls className={styles.graphControls} />
      {isExpanded && (
        <MiniMap
          nodeColor={(node) => getNodeColor(node.data?.type)}
          maskColor="rgba(13, 17, 23, 0.8)"
        />
      )}
    </ReactFlow>
  );
};

// Wrapper that provides ReactFlowProvider
export const ExecHierarchyGraph: React.FC<ExecHierarchyGraphProps> = (props) => {
  const { loading, error, nodes } = props;

  if (loading) {
    return (
      <div className={styles.treeLoading}>
        <div className={styles.loadingSpinner} />
        <span>Loading hierarchy...</span>
      </div>
    );
  }

  if (error) {
    return (
      <div className={styles.treeError}>
        <span className={styles.errorIcon}>⚠</span>
        <span>{error}</span>
      </div>
    );
  }

  if (!nodes || nodes.length === 0) {
    return (
      <div className={styles.treeEmpty}>
        <span className={styles.emptyIcon}>⬡</span>
        <div className={styles.emptyTitle}>No Executions Yet</div>
        <div className={styles.emptyText}>
          Execution nodes will appear here as ailang commands run.
        </div>
      </div>
    );
  }

  return (
    <div className={`${styles.graphContainer} ${props.isExpanded ? styles.graphContainerExpanded : ''}`}>
      <ReactFlowProvider>
        <ExecHierarchyGraphInner {...props} />
      </ReactFlowProvider>
    </div>
  );
};

export default ExecHierarchyGraph;
