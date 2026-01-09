/**
 * ExecHierarchyGraph - ReactFlow graph view for the exec hierarchy
 * Visualizes: Messages -> Execs -> Turns -> Tool Uses as a directed graph
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
    default:
      return '●';
  }
}

// Custom node component
const HierarchyNodeComponent: React.FC<NodeProps> = ({ data, selected }) => {
  const nodeColor = getNodeColor(data.type);
  const statusColor = getStatusColor(data.status);

  return (
    <div
      className={`${styles.rfNode} ${selected ? styles.rfNodeSelected : ''}`}
      style={{ borderLeftColor: nodeColor }}
    >
      <Handle type="target" position={Position.Top} className={styles.rfHandle} />
      <div className={styles.rfNodeHeader}>
        <span className={styles.rfNodeIcon} style={{ color: nodeColor }}>
          {getNodeIcon(data.type)}
        </span>
        <span className={styles.rfNodeLabel}>{data.label}</span>
      </div>
      <div className={styles.rfNodeMeta}>
        <span style={{ color: statusColor }}>{data.status}</span>
        {data.durationMs && <span> • {data.durationMs}ms</span>}
      </div>
      <Handle type="source" position={Position.Bottom} className={styles.rfHandle} />
    </div>
  );
};

const nodeTypes = {
  hierarchy: HierarchyNodeComponent,
};

// Flatten hierarchy nodes to ReactFlow nodes and edges
function buildGraphData(
  nodes: HierarchyNode[],
  selectedId?: string | null
): { rfNodes: Node[]; rfEdges: Edge[] } {
  const rfNodes: Node[] = [];
  const rfEdges: Edge[] = [];
  const seen = new Set<string>();

  // Layout parameters
  const xSpacing = 200;
  const ySpacing = 100;

  function processNode(
    node: HierarchyNode,
    depth: number,
    parentId: string | null,
    siblingIndex: number,
    siblingCount: number
  ) {
    if (seen.has(node.id)) return;
    seen.add(node.id);

    // Calculate position
    const x = depth * xSpacing + 50;
    const totalHeight = siblingCount * ySpacing;
    const startY = -totalHeight / 2;
    const y = startY + siblingIndex * ySpacing + ySpacing / 2;

    rfNodes.push({
      id: node.id,
      type: 'hierarchy',
      position: { x, y },
      data: {
        ...node,
        selected: selectedId === node.id,
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

    // Process children
    if (node.children && node.children.length > 0) {
      node.children.forEach((child, idx) => {
        processNode(child, depth + 1, node.id, idx, node.children!.length);
      });
    }
  }

  // Process all root nodes
  nodes.forEach((node, idx) => {
    processNode(node, 0, null, idx, nodes.length);
  });

  return { rfNodes, rfEdges };
}

export interface ExecHierarchyGraphProps {
  nodes: HierarchyNode[];
  selectedNodeId?: string | null;
  onNodeClick?: (node: HierarchyNode) => void;
  loading?: boolean;
  error?: string | null;
  isExpanded?: boolean;
}

// Inner component that uses ReactFlow hooks
const ExecHierarchyGraphInner: React.FC<ExecHierarchyGraphProps> = ({
  nodes,
  selectedNodeId,
  onNodeClick,
  loading,
  error,
  isExpanded,
}) => {
  const reactFlowRef = useRef<ReactFlowInstance | null>(null);

  // Build graph data
  const { rfNodes: initialNodes, rfEdges: initialEdges } = useMemo(
    () => buildGraphData(nodes, selectedNodeId),
    [nodes, selectedNodeId]
  );

  const [rfNodes, setNodes, onNodesChange] = useNodesState(initialNodes);
  const [rfEdges, setEdges, onEdgesChange] = useEdgesState(initialEdges);

  // Update when data changes
  useEffect(() => {
    const { rfNodes, rfEdges } = buildGraphData(nodes, selectedNodeId);
    setNodes(rfNodes);
    setEdges(rfEdges);
  }, [nodes, selectedNodeId, setNodes, setEdges]);

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

  // Only refit on expand/collapse toggle, NOT on data changes
  useEffect(() => {
    if (reactFlowRef.current) {
      setTimeout(() => reactFlowRef.current?.fitView({ padding: 0.2 }), 100);
    }
  }, [isExpanded]); // Removed rfNodes - don't reset zoom on data refresh!

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
