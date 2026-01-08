/**
 * AgentTopology - ReactFlow-based agent network visualization
 * Data-driven: derives nodes and edges from actual message flow data
 */
import React, { useMemo, useEffect, useCallback, useRef } from 'react';
import ReactFlow, {
  Node,
  Edge,
  Background,
  Controls,
  MiniMap,
  Panel,
  useNodesState,
  useEdgesState,
  MarkerType,
  Handle,
  Position,
  NodeProps,
  ReactFlowInstance,
} from 'reactflow';
import 'reactflow/dist/style.css';
import type { Agent, TopologyEdge } from './types';
import { getStatusIcon } from './utils';
import styles from '../ControlPlane.module.css';

export interface AgentTopologyProps {
  agents: Agent[];
  edges: TopologyEdge[];
  isExpanded: boolean;
  onToggleExpand: () => void;
  onAgentClick: (agent: Agent) => void;
  selectedNodeId?: string | null;
  highlightedPath?: Set<string>;
  onNodeSelect?: (nodeId: string | null) => void;
  isEmpty?: boolean;
  /** Display mode: 'hierarchy' for exec task hierarchy (default), 'messages' for message flow */
  mode?: 'hierarchy' | 'messages';
}

// Workflow groups for visual organization
type WorkflowGroup = 'input' | 'processing' | 'output';

// Custom React Flow Node Components with selection/highlight support
const AgentNode: React.FC<NodeProps> = ({ data }) => {
  const statusColors: Record<string, string> = {
    idle: '#6b7280',
    busy: '#25c2a0',
    blocked: '#f59e0b',
    error: '#ef4444',
  };

  const classNames = [
    styles.rfAgentNode,
    data.isSelected && styles.rfNodeSelected,
    data.isDimmed && styles.rfNodeDimmed,
    data.isHighlighted && styles.rfNodeHighlighted,
  ].filter(Boolean).join(' ');

  return (
    <div
      className={classNames}
      data-status={data.status}
      onClick={() => data.onClick?.(data)}
    >
      <Handle type="target" position={Position.Top} className={styles.rfHandle} />
      <div className={styles.rfNodeStatus} style={{ backgroundColor: statusColors[data.status] || statusColors.idle }}>
        {getStatusIcon(data.status)}
      </div>
      <div className={styles.rfNodeContent}>
        <div className={styles.rfNodeLabel}>{data.label}</div>
        <div className={styles.rfNodeMeta}>
          <span className={styles.rfNodeTrust}>Trust: {data.trustScore || 0}%</span>
          <span className={styles.rfNodeTasks}>{data.taskCount || 0} msgs</span>
        </div>
      </div>
      <Handle type="source" position={Position.Bottom} className={styles.rfHandle} />
    </div>
  );
};

const SourceNode: React.FC<NodeProps> = ({ data }) => {
  const classNames = [
    styles.rfSourceNode,
    data.isSelected && styles.rfNodeSelected,
    data.isDimmed && styles.rfNodeDimmed,
    data.isHighlighted && styles.rfNodeHighlighted,
  ].filter(Boolean).join(' ');

  return (
    <div className={classNames} onClick={() => data.onClick?.({ id: data.id, label: data.label })}>
      <div className={styles.rfSourceIcon}>{data.icon || '◉'}</div>
      <div className={styles.rfSourceLabel}>{data.label}</div>
      <Handle type="source" position={Position.Bottom} className={styles.rfHandle} />
    </div>
  );
};

const SinkNode: React.FC<NodeProps> = ({ data }) => {
  const classNames = [
    styles.rfSinkNode,
    data.isSelected && styles.rfNodeSelected,
    data.isDimmed && styles.rfNodeDimmed,
    data.isHighlighted && styles.rfNodeHighlighted,
  ].filter(Boolean).join(' ');

  return (
    <div className={classNames} data-type={data.type} onClick={() => data.onClick?.({ id: data.id, label: data.label })}>
      <Handle type="target" position={Position.Top} className={styles.rfHandle} />
      <div className={styles.rfSinkContent}>
        <div className={styles.rfSinkLabel}>{data.label}</div>
        {data.badge && <div className={styles.rfSinkBadge}>{data.badge}</div>}
      </div>
      {data.hasOutput && <Handle type="source" position={Position.Bottom} className={styles.rfHandle} />}
    </div>
  );
};

const nodeTypes = {
  agent: AgentNode,
  source: SourceNode,
  sink: SinkNode,
};

// Detect workflow group from edge topology (not by name matching)
function detectWorkflowGroups(
  agents: Agent[],
  edges: TopologyEdge[]
): Map<string, WorkflowGroup> {
  const groups = new Map<string, WorkflowGroup>();
  const hasIncoming = new Set<string>();
  const hasOutgoing = new Set<string>();

  // Build edge sets
  for (const edge of edges) {
    hasIncoming.add(edge.target);
    hasOutgoing.add(edge.source);
  }

  // Classify agents based on their connectivity
  for (const agent of agents) {
    const incoming = hasIncoming.has(agent.id);
    const outgoing = hasOutgoing.has(agent.id);

    if (!incoming && outgoing) {
      groups.set(agent.id, 'input');
    } else if (incoming && !outgoing) {
      groups.set(agent.id, 'output');
    } else {
      groups.set(agent.id, 'processing');
    }
  }

  return groups;
}

// Calculate dynamic node positions based on workflow groups
function calculateNodePositions(
  agents: Agent[],
  groups: Map<string, WorkflowGroup>
): Record<string, { x: number; y: number }> {
  const positions: Record<string, { x: number; y: number }> = {};

  // Group agents by workflow stage
  const inputAgents = agents.filter(a => groups.get(a.id) === 'input');
  const processingAgents = agents.filter(a => groups.get(a.id) === 'processing');
  const outputAgents = agents.filter(a => groups.get(a.id) === 'output');

  // Layout parameters
  const nodeWidth = 160;
  const nodeHeight = 80;
  const horizontalSpacing = 200;
  const verticalSpacing = 140;
  const centerX = 400;

  // Position input nodes at the top
  const inputStartX = centerX - ((inputAgents.length - 1) * horizontalSpacing) / 2;
  inputAgents.forEach((agent, idx) => {
    positions[agent.id] = {
      x: inputStartX + idx * horizontalSpacing,
      y: 30,
    };
  });

  // Position processing nodes in the middle (grid layout)
  const cols = Math.min(processingAgents.length, 4);
  const processingStartX = centerX - ((Math.min(cols, processingAgents.length) - 1) * horizontalSpacing) / 2;
  processingAgents.forEach((agent, idx) => {
    const col = idx % cols;
    const row = Math.floor(idx / cols);
    positions[agent.id] = {
      x: processingStartX + col * horizontalSpacing,
      y: 180 + row * verticalSpacing,
    };
  });

  // Position output nodes at the bottom
  const maxProcessingRow = processingAgents.length > 0 ? Math.floor((processingAgents.length - 1) / cols) : 0;
  const outputY = 180 + (maxProcessingRow + 1) * verticalSpacing + 50;
  const outputStartX = centerX - ((outputAgents.length - 1) * horizontalSpacing) / 2;
  outputAgents.forEach((agent, idx) => {
    positions[agent.id] = {
      x: outputStartX + idx * horizontalSpacing,
      y: outputY,
    };
  });

  return positions;
}

export const AgentTopology: React.FC<AgentTopologyProps> = ({
  agents,
  edges: topologyEdges,
  isExpanded,
  onToggleExpand,
  onAgentClick,
  selectedNodeId,
  highlightedPath,
  onNodeSelect,
  isEmpty = false,
  mode = 'hierarchy',
}) => {
  const reactFlowRef = useRef<ReactFlowInstance | null>(null);

  // Check if any highlighting is active
  const hasHighlight = highlightedPath && highlightedPath.size > 0;

  // Detect workflow groups from edge topology
  const workflowGroups = useMemo(
    () => detectWorkflowGroups(agents, topologyEdges),
    [agents, topologyEdges]
  );

  // Calculate dynamic node positions
  const nodePositions = useMemo(
    () => calculateNodePositions(agents, workflowGroups),
    [agents, workflowGroups]
  );

  // Convert agents to React Flow nodes
  const initialNodes: Node[] = useMemo(() => {
    if (isEmpty || agents.length === 0) {
      return [];
    }

    const isNodeHighlighted = (nodeId: string) => highlightedPath?.has(nodeId) ?? false;
    const isNodeDimmed = (nodeId: string) => hasHighlight && !isNodeHighlighted(nodeId);

    return agents.map((agent) => {
      const group = workflowGroups.get(agent.id) || 'processing';
      const nodeType = group === 'input' ? 'source' : (group === 'output' ? 'sink' : 'agent');

      return {
        id: agent.id,
        type: nodeType,
        position: nodePositions[agent.id] || { x: 400, y: 200 },
        data: {
          ...agent,
          id: agent.id,
          icon: group === 'input' ? '◉' : undefined,
          type: group === 'output' ? 'output' : undefined,
          hasOutput: false,
          onClick: onAgentClick,
          isSelected: selectedNodeId === agent.id,
          isHighlighted: isNodeHighlighted(agent.id),
          isDimmed: isNodeDimmed(agent.id),
        },
      };
    });
  }, [agents, workflowGroups, nodePositions, onAgentClick, selectedNodeId, highlightedPath, hasHighlight, isEmpty]);

  // Convert edges to React Flow edges
  const initialEdges: Edge[] = useMemo(() => {
    if (isEmpty || topologyEdges.length === 0) {
      return [];
    }

    return topologyEdges.map((edge, idx) => {
      const edgeId = `edge-${idx}`;
      const isEdgeHighlighted = highlightedPath?.has(edgeId) ||
        (highlightedPath?.has(edge.source) && highlightedPath?.has(edge.target));
      const isEdgeDimmed = hasHighlight && !isEdgeHighlighted;

      const isActive = edge.active || isEdgeHighlighted;
      const strokeColor = isEdgeHighlighted ? '#25c2a0' : (isEdgeDimmed ? '#374151' : (edge.active ? '#25c2a0' : '#64748b'));
      const strokeWidth = isEdgeHighlighted ? 4 : (edge.active ? 3 : 2);
      const opacity = isEdgeDimmed ? 0.3 : 1;

      return {
        id: edgeId,
        source: edge.source,
        target: edge.target,
        type: 'smoothstep',
        animated: isActive,
        className: isActive ? styles.rfEdgeActive : '',
        label: edge.messageCount > 0 ? edge.messageCount.toString() : undefined,
        labelStyle: {
          fill: isEdgeHighlighted ? '#25c2a0' : (isEdgeDimmed ? '#4b5563' : (edge.active ? '#25c2a0' : '#94a3b8')),
          fontSize: 12,
          fontFamily: 'monospace',
          fontWeight: edge.messageCount > 0 ? 600 : 400,
          opacity,
        },
        labelBgStyle: { fill: '#1a1f2e', fillOpacity: 0.95 },
        labelBgPadding: [6, 4] as [number, number],
        labelBgBorderRadius: 4,
        style: {
          stroke: strokeColor,
          strokeWidth,
          opacity,
          transition: 'all 0.3s ease-in-out',
        },
        markerEnd: {
          type: MarkerType.ArrowClosed,
          color: strokeColor,
          width: 24,
          height: 24,
        },
      };
    });
  }, [topologyEdges, highlightedPath, hasHighlight, isEmpty]);

  const [nodes, setNodes, onNodesChange] = useNodesState(initialNodes);
  const [flowEdges, setEdges, onEdgesChange] = useEdgesState(initialEdges);

  // Update nodes when selection/highlight changes
  useEffect(() => {
    setNodes(initialNodes);
  }, [initialNodes, setNodes]);

  // Update edges when highlight changes
  useEffect(() => {
    setEdges(initialEdges);
  }, [initialEdges, setEdges]);

  // Trigger fitView on init
  const onInit = useCallback((reactFlowInstance: ReactFlowInstance) => {
    reactFlowRef.current = reactFlowInstance;
    const fitWithDelay = (delay: number) => {
      setTimeout(() => {
        reactFlowInstance.fitView({ padding: 0.2 });
      }, delay);
    };
    fitWithDelay(50);
    fitWithDelay(200);
    fitWithDelay(500);
  }, []);

  // Re-fit when expanded state changes
  useEffect(() => {
    if (reactFlowRef.current) {
      setTimeout(() => {
        reactFlowRef.current?.fitView({ padding: 0.2 });
      }, 100);
    }
  }, [isExpanded]);

  // Auto-zoom to highlighted nodes when path changes
  useEffect(() => {
    if (reactFlowRef.current && highlightedPath && highlightedPath.size > 0) {
      const highlightedNodes = nodes.filter((n) => highlightedPath.has(n.id));
      if (highlightedNodes.length > 0) {
        setTimeout(() => {
          reactFlowRef.current?.fitView({
            padding: 0.3,
            nodes: highlightedNodes,
            duration: 500,
          });
        }, 100);
      }
    }
  }, [highlightedPath, nodes]);

  // Empty state component
  const EmptyState = () => (
    <div className={styles.topologyEmpty}>
      <div className={styles.topologyEmptyIcon}>⬡</div>
      <div className={styles.topologyEmptyTitle}>No Executions Yet</div>
      <div className={styles.topologyEmptyText}>
        Execution nodes will appear here as ailang commands run.
        Try running `ailang exec`, `ailang run`, or `ailang check`.
      </div>
    </div>
  );

  return (
    <div className={`${styles.topologyContainer} ${isExpanded ? styles.topologyExpanded : ''}`}>
      <div className={styles.topologyHeader}>
        <h3 className={styles.panelTitle}>
          <span className={styles.panelIcon}>◎</span>
          {mode === 'hierarchy' ? 'Exec Hierarchy' : 'Agent Topology'}
          {!isEmpty && agents.length > 0 && (
            <span className={styles.panelBadge}>
              {agents.length} {mode === 'hierarchy' ? 'tasks' : 'agents'}
            </span>
          )}
        </h3>
        <div className={styles.topologyControls}>
          {hasHighlight && (
            <button
              className={styles.showAllBtn}
              onClick={() => onNodeSelect?.(null)}
              title="Show all tasks"
            >
              Show All
            </button>
          )}
          <button className={styles.expandBtn} onClick={onToggleExpand}>
            {isExpanded ? '⤡' : '⤢'}
          </button>
        </div>
        <div className={styles.topologyLegend}>
          <span className={styles.legendItem}><span className={styles.statusIdle}>○</span> idle</span>
          <span className={styles.legendItem}><span className={styles.statusBusy}>●</span> busy</span>
          <span className={styles.legendItem}><span className={styles.statusBlocked}>◐</span> blocked</span>
          <span className={styles.legendItem}><span className={styles.statusError}>✕</span> error</span>
        </div>
      </div>
      <div className={styles.topologyViewport}>
        {isEmpty || agents.length === 0 ? (
          <EmptyState />
        ) : (
          <ReactFlow
            nodes={nodes}
            edges={flowEdges}
            onNodesChange={onNodesChange}
            onEdgesChange={onEdgesChange}
            onInit={onInit}
            onPaneClick={() => onNodeSelect?.(null)}
            nodeTypes={nodeTypes}
            fitView
            fitViewOptions={{ padding: 0.2 }}
            minZoom={0.3}
            maxZoom={2}
            defaultEdgeOptions={{
              type: 'smoothstep',
            }}
            proOptions={{ hideAttribution: true }}
          >
            <Background color="#1e293b" gap={24} size={1} />
            <Controls className={styles.rfControls} />
            {isExpanded && <MiniMap className={styles.rfMinimap} nodeColor="#374151" maskColor="rgba(13, 17, 23, 0.8)" />}
            <Panel position="bottom-center" className={styles.workflowStagesPanel}>
              <div className={styles.workflowStages}>
                <span className={`${styles.workflowStage} ${styles.workflowInput}`}>▼ Input</span>
                <span className={styles.workflowArrow}>→</span>
                <span className={`${styles.workflowStage} ${styles.workflowProcessing}`}>⚙ Processing</span>
                <span className={styles.workflowArrow}>→</span>
                <span className={`${styles.workflowStage} ${styles.workflowOutput}`}>▲ Output</span>
              </div>
            </Panel>
          </ReactFlow>
        )}
      </div>
    </div>
  );
};

export default AgentTopology;
