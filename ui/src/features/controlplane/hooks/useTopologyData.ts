/**
 * Hook for fetching execution hierarchy data from Observatory
 *
 * Uses /api/controlplane/exec-hierarchy which returns the actual tree of
 * ailang exec/run/check commands with parent/child relationships from OTEL spans.
 *
 * This is the SINGLE SOURCE OF TRUTH for topology - no config files, no mock data.
 */
import { useState, useEffect, useCallback } from 'react';

export interface TopologyAgent {
  id: string;
  label: string;
  status: 'idle' | 'busy' | 'blocked' | 'error';
  trustScore: number;
  taskCount: number;
  cost: number;
  // Extended fields from ExecTaskNode
  command?: string;    // 'exec' | 'run' | 'check'
  provider?: string;   // claude, gemini, etc.
  workspace?: string;
  filePath?: string;
  startTime?: string;
  durationMs?: number;
}

export interface TopologyEdge {
  source: string;
  target: string;
  messageCount: number;
  lastActivity?: string;
  active?: boolean;
}

export interface TopologySink {
  id: string;
  label?: string;
  pendingCount?: number;
}

export interface TopologyData {
  agents: TopologyAgent[];
  edges: TopologyEdge[];
  sinks: TopologySink[];
  isEmpty?: boolean;
}

// ExecTaskNode from the backend API
interface ExecTaskNode {
  task_id: string;
  parent_task_id: string;
  command: string;      // exec, run, check
  provider: string;     // for exec: claude, gemini, etc.
  workspace: string;    // for exec
  file_path: string;    // for run, check
  status: string;
  start_time?: string;
  duration_ms?: number;
  children?: ExecTaskNode[];
}

interface ExecHierarchyResponse {
  hierarchy: ExecTaskNode[];
  count: number;
}

interface UseTopologyDataOptions {
  refreshInterval?: number; // ms, 0 to disable
  limit?: number;           // max nodes to fetch
}

/**
 * Flatten ExecTaskNode tree into agents and edges
 */
function flattenHierarchy(nodes: ExecTaskNode[]): { agents: TopologyAgent[]; edges: TopologyEdge[] } {
  const agents: TopologyAgent[] = [];
  const edges: TopologyEdge[] = [];
  const seen = new Set<string>();

  function processNode(node: ExecTaskNode, depth: number) {
    if (seen.has(node.task_id)) return;
    seen.add(node.task_id);

    // Create label based on command type
    let label = node.task_id;
    if (node.command === 'exec' && node.provider) {
      label = `${node.provider} exec`;
    } else if (node.command === 'run' && node.file_path) {
      // Extract filename from path
      const filename = node.file_path.split('/').pop() || node.file_path;
      label = `run ${filename}`;
    } else if (node.command === 'check' && node.file_path) {
      const filename = node.file_path.split('/').pop() || node.file_path;
      label = `check ${filename}`;
    } else if (node.command) {
      label = node.command;
    }

    // Map status to topology status
    let status: TopologyAgent['status'] = 'idle';
    if (node.status === 'error' || node.status === 'ERROR') {
      status = 'error';
    } else if (node.status === 'running' || node.status === 'RUNNING') {
      status = 'busy';
    } else if (node.status === 'ok' || node.status === 'OK' || node.status === 'completed') {
      status = 'idle';
    }

    agents.push({
      id: node.task_id,
      label,
      status,
      trustScore: 100, // Real data, full trust
      taskCount: (node.children?.length || 0) + 1,
      cost: 0, // Could add if available in spans
      command: node.command,
      provider: node.provider,
      workspace: node.workspace,
      filePath: node.file_path,
      startTime: node.start_time,
      durationMs: node.duration_ms,
    });

    // Create edge to parent
    if (node.parent_task_id) {
      edges.push({
        source: node.parent_task_id,
        target: node.task_id,
        messageCount: 1,
        active: status === 'busy',
      });
    }

    // Process children recursively
    if (node.children) {
      for (const child of node.children) {
        processNode(child, depth + 1);
      }
    }
  }

  for (const root of nodes) {
    processNode(root, 0);
  }

  return { agents, edges };
}

export function useTopologyData(options: UseTopologyDataOptions = {}) {
  const { refreshInterval = 5000, limit = 100 } = options;
  const [data, setData] = useState<TopologyData | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  const fetchData = useCallback(async () => {
    try {
      // Single source of truth: exec hierarchy from Observatory spans
      const response = await fetch(`/api/controlplane/exec-hierarchy?limit=${limit}`);
      if (!response.ok) {
        throw new Error(`HTTP error: ${response.status}`);
      }

      const result: ExecHierarchyResponse = await response.json();

      // Transform hierarchy tree into topology format
      const { agents, edges } = flattenHierarchy(result.hierarchy);

      setData({
        agents,
        edges,
        sinks: [], // No hardcoded sinks - data shows what's real
        isEmpty: result.count === 0,
      });

      setError(null);
    } catch (err) {
      // NO SILENT FALLBACKS - show the error
      setError(err instanceof Error ? err.message : 'Failed to fetch topology data');
      // Don't set fake data - leave as null to indicate failure
    } finally {
      setLoading(false);
    }
  }, [limit]);

  useEffect(() => {
    fetchData();

    if (refreshInterval > 0) {
      const interval = setInterval(fetchData, refreshInterval);
      return () => clearInterval(interval);
    }
  }, [fetchData, refreshInterval]);

  return { data, loading, error, refetch: fetchData };
}

// ============================================================================
// New Exec Hierarchy Hook (4-level: Messages -> Execs -> Turns -> Tools)
// ============================================================================

import type {
  HierarchyNode,
  MessageNode as MessageNodeType,
  ExecHierarchyWithMessages,
  NodeStatus,
  ExecTaskNode as ExecTaskNodeType,
} from '../components/ExecHierarchy/types';

// Re-export types for consumers
export type { HierarchyNode, ViewMode, NodeStatus } from '../components/ExecHierarchy/types';

interface UseExecHierarchyOptions {
  refreshInterval?: number;
  limit?: number;
}

interface ExecHierarchyData {
  nodes: HierarchyNode[];
  isEmpty: boolean;
}

/**
 * Map backend status string to NodeStatus
 */
function mapStatus(status: string): NodeStatus {
  const s = status?.toLowerCase() || '';
  if (s === 'error') return 'error';
  if (s === 'running') return 'busy';
  if (s === 'ok' || s === 'completed') return 'completed';
  if (s === 'pending' || s === 'unread') return 'pending';
  return 'idle';
}

/**
 * Transform ExecTaskNode to HierarchyNode recursively
 */
function transformExecNode(node: ExecTaskNodeType): HierarchyNode {
  // Determine node type from command
  const nodeType = (node.command === 'turn' || node.command === 'tool_use')
    ? node.command
    : 'exec';

  // Create label based on command type
  let label = node.task_id;
  if (node.command === 'exec' && node.provider) {
    label = `${node.provider} exec`;
  } else if (node.command === 'run' && node.file_path) {
    const filename = node.file_path.split('/').pop() || node.file_path;
    label = `run ${filename}`;
  } else if (node.command === 'check' && node.file_path) {
    const filename = node.file_path.split('/').pop() || node.file_path;
    label = `check ${filename}`;
  } else if (node.command === 'turn') {
    label = `Turn ${node.turn_number || '?'}`;
  } else if (node.command === 'tool_use') {
    label = node.tool_name || 'Tool';
  }

  const result: HierarchyNode = {
    id: node.task_id,
    type: nodeType as HierarchyNode['type'],
    label,
    status: mapStatus(node.status),
    startTime: node.start_time,
    durationMs: node.duration_ms,
  };

  // Add type-specific fields
  if (nodeType === 'exec') {
    Object.assign(result, {
      taskId: node.task_id,
      parentTaskId: node.parent_task_id,
      provider: node.provider,
      workspace: node.workspace,
      filePath: node.file_path,
      command: node.command,
    });
  } else if (nodeType === 'turn') {
    Object.assign(result, {
      turnNumber: node.turn_number || 0,
    });
  } else if (nodeType === 'tool_use') {
    Object.assign(result, {
      toolName: node.tool_name || '',
      toolInput: node.tool_input,
      toolOutput: node.tool_output,
    });
  }

  // Transform children recursively
  if (node.children && node.children.length > 0) {
    result.children = node.children.map(transformExecNode);
  }

  return result;
}

/**
 * Transform MessageNode to HierarchyNode
 */
function transformMessageNode(msg: MessageNodeType): HierarchyNode {
  const result: HierarchyNode = {
    id: msg.message_id,
    type: 'message',
    label: msg.title || msg.message_id,
    status: mapStatus(msg.status),
    startTime: msg.created_at,
  };

  // Add message-specific fields
  Object.assign(result, {
    messageId: msg.message_id,
    fromAgent: msg.from_agent,
    toInbox: msg.to_inbox,
    messageType: msg.message_type,
    title: msg.title,
  });

  // Transform exec children
  if (msg.execs && msg.execs.length > 0) {
    result.children = msg.execs.map(transformExecNode);
  }

  return result;
}

/**
 * Hook for fetching exec hierarchy with messages (4-level view)
 */
export function useExecHierarchy(options: UseExecHierarchyOptions = {}) {
  const { refreshInterval = 5000, limit = 100 } = options;
  const [data, setData] = useState<ExecHierarchyData | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  const fetchData = useCallback(async () => {
    try {
      const response = await fetch(
        `/api/controlplane/exec-hierarchy?limit=${limit}&include_messages=true`
      );
      if (!response.ok) {
        throw new Error(`HTTP error: ${response.status}`);
      }

      const result: ExecHierarchyWithMessages = await response.json();

      // Transform to unified HierarchyNode format
      const nodes: HierarchyNode[] = [];

      // Add message nodes (with their exec children)
      if (result.messages) {
        for (const msg of result.messages) {
          nodes.push(transformMessageNode(msg));
        }
      }

      // Add orphan exec nodes (no triggering message)
      if (result.orphan) {
        for (const exec of result.orphan) {
          nodes.push(transformExecNode(exec));
        }
      }

      setData({
        nodes,
        isEmpty: result.count === 0,
      });
      setError(null);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to fetch exec hierarchy');
    } finally {
      setLoading(false);
    }
  }, [limit]);

  useEffect(() => {
    fetchData();

    if (refreshInterval > 0) {
      const interval = setInterval(fetchData, refreshInterval);
      return () => clearInterval(interval);
    }
  }, [fetchData, refreshInterval]);

  return { data, loading, error, refetch: fetchData };
}
