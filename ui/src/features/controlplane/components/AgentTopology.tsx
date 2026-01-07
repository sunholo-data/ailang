/**
 * AgentTopology - ReactFlow-based agent network visualization
 */
import React, { useMemo, useEffect, useCallback, useRef } from 'react';
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
}

// Custom React Flow Node Components
const AgentNode: React.FC<NodeProps> = ({ data }) => {
  const statusColors: Record<string, string> = {
    idle: '#6b7280',
    busy: '#25c2a0',
    blocked: '#f59e0b',
    error: '#ef4444',
  };

  return (
    <div
      className={styles.rfAgentNode}
      data-status={data.status}
      onClick={() => data.onClick?.(data)}
    >
      <Handle type="target" position={Position.Top} className={styles.rfHandle} />
      <div className={styles.rfNodeStatus} style={{ backgroundColor: statusColors[data.status] }}>
        {getStatusIcon(data.status)}
      </div>
      <div className={styles.rfNodeContent}>
        <div className={styles.rfNodeLabel}>{data.label}</div>
        <div className={styles.rfNodeMeta}>
          <span className={styles.rfNodeTrust}>Trust: {data.trustScore}%</span>
          <span className={styles.rfNodeTasks}>{data.taskCount} tasks</span>
        </div>
      </div>
      <Handle type="source" position={Position.Bottom} className={styles.rfHandle} />
    </div>
  );
};

const SourceNode: React.FC<NodeProps> = ({ data }) => (
  <div className={styles.rfSourceNode}>
    <div className={styles.rfSourceIcon}>{data.icon}</div>
    <div className={styles.rfSourceLabel}>{data.label}</div>
    <Handle type="source" position={Position.Bottom} className={styles.rfHandle} />
  </div>
);

const SinkNode: React.FC<NodeProps> = ({ data }) => (
  <div className={styles.rfSinkNode} data-type={data.type}>
    <Handle type="target" position={Position.Top} className={styles.rfHandle} />
    <div className={styles.rfSinkContent}>
      <div className={styles.rfSinkLabel}>{data.label}</div>
      {data.badge && <div className={styles.rfSinkBadge}>{data.badge}</div>}
    </div>
    {data.hasOutput && <Handle type="source" position={Position.Bottom} className={styles.rfHandle} />}
  </div>
);

const nodeTypes = {
  agent: AgentNode,
  source: SourceNode,
  sink: SinkNode,
};

export const AgentTopology: React.FC<AgentTopologyProps> = ({
  agents,
  edges: topologyEdges,
  isExpanded,
  onToggleExpand,
  onAgentClick,
}) => {
  // Convert agents to React Flow nodes
  const initialNodes: Node[] = useMemo(() => {
    const nodePositions: Record<string, { x: number; y: number }> = {
      'github': { x: 350, y: 0 },
      'design-doc-creator': { x: 100, y: 150 },
      'sprint-planner': { x: 350, y: 150 },
      'sprint-executor': { x: 600, y: 150 },
      'eval-analyzer': { x: 100, y: 300 },
      'approval': { x: 350, y: 300 },
      'main': { x: 350, y: 450 },
    };

    const nodes: Node[] = [
      {
        id: 'github',
        type: 'source',
        position: nodePositions.github,
        data: { label: 'GitHub Issues', icon: '⬡' },
      },
      ...agents.map((agent) => ({
        id: agent.id,
        type: 'agent',
        position: nodePositions[agent.id] || { x: 350, y: 150 },
        data: {
          ...agent,
          onClick: onAgentClick,
        },
      })),
      {
        id: 'approval',
        type: 'sink',
        position: nodePositions.approval,
        data: { label: 'Approval Queue', badge: '12 ⏳', type: 'approval', hasOutput: true },
      },
      {
        id: 'main',
        type: 'sink',
        position: nodePositions.main,
        data: { label: 'Main Branch', badge: '✓ 29', type: 'success', hasOutput: false },
      },
    ];

    return nodes;
  }, [agents, onAgentClick]);

  // Convert edges to React Flow edges - always visible with clear styling
  const initialEdges: Edge[] = useMemo(() => {
    return topologyEdges.map((edge, idx) => ({
      id: `edge-${idx}`,
      source: edge.source,
      target: edge.target,
      type: 'smoothstep',
      animated: edge.active,
      // Show message count or "→" if 0
      label: edge.messageCount > 0 ? edge.messageCount.toString() : '→',
      labelStyle: {
        fill: edge.active ? '#25c2a0' : '#94a3b8',
        fontSize: 12,
        fontFamily: 'monospace',
        fontWeight: edge.messageCount > 0 ? 600 : 400,
      },
      labelBgStyle: { fill: '#1a1f2e', fillOpacity: 0.95 },
      labelBgPadding: [6, 4] as [number, number],
      labelBgBorderRadius: 4,
      style: {
        // Always use at least strokeWidth 2 for visibility
        stroke: edge.active ? '#25c2a0' : '#64748b',
        strokeWidth: edge.active ? 3 : 2,
      },
      markerEnd: {
        type: MarkerType.ArrowClosed,
        color: edge.active ? '#25c2a0' : '#64748b',
        width: 24,
        height: 24,
      },
    }));
  }, [topologyEdges]);

  const [nodes, , onNodesChange] = useNodesState(initialNodes);
  const [flowEdges, , onEdgesChange] = useEdgesState(initialEdges);
  const reactFlowRef = useRef<ReactFlowInstance | null>(null);

  // Trigger fitView on init and store instance for later use
  const onInit = useCallback((reactFlowInstance: ReactFlowInstance) => {
    reactFlowRef.current = reactFlowInstance;
    // Multiple attempts to ensure container has proper dimensions
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

  return (
    <div className={`${styles.topologyContainer} ${isExpanded ? styles.topologyExpanded : ''}`}>
      <div className={styles.topologyHeader}>
        <h3 className={styles.panelTitle}>
          <span className={styles.panelIcon}>◎</span>
          Agent Topology
        </h3>
        <div className={styles.topologyControls}>
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
        <ReactFlow
          nodes={nodes}
          edges={flowEdges}
          onNodesChange={onNodesChange}
          onEdgesChange={onEdgesChange}
          onInit={onInit}
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
        </ReactFlow>
      </div>
    </div>
  );
};

export default AgentTopology;
