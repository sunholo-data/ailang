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
  command: string;      // exec, run, check, turn, tool_use
  provider: string;     // for exec: claude, gemini, etc.
  workspace: string;    // for exec
  file_path: string;    // for run, check
  status: string;
  start_time?: string;
  duration_ms?: number;
  children?: ExecTaskNode[];
  // Turn/tool specific
  turn_number?: number;
  tool_name?: string;
  tool_input?: string;
  tool_output?: string;
  display_name?: string; // enriched name from session_tools
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
