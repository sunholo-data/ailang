/**
 * Hook for fetching cross-task hierarchy data for visualization
 * Returns tasks with parent_task_id relationships, session continuity, and approval status
 */
import { useState, useEffect, useCallback, useMemo } from 'react';
import type { Node, Edge } from 'reactflow';
import type {
  TaskHierarchyNode,
  TaskHierarchyEdge,
  TaskHierarchyResult,
  TaskGraphNode,
  TaskGraphEdge,
} from '../components/ExecHierarchy/types';

interface UseTaskHierarchyOptions {
  refreshInterval?: number; // ms, 0 to disable
  limit?: number;           // Max tasks to fetch (default: 50)
  status?: string[];        // Filter by status
  workspace?: string;       // Filter by workspace path
  provider?: string;        // Filter by provider
  taskId?: string;          // Filter to specific task and its chain
  taskIds?: string[];       // Filter to multiple specific task IDs
  traceId?: string;         // Filter to tasks with spans in this trace
  skip?: boolean;           // Skip fetching (used when spans are provided instead)
  groupBy?: 'turns';        // Group spans by conversation turn (Session → Turn 1 → Turn 2 → ...)
}

// Layout constants
const NODE_WIDTH = 220;
const NODE_HEIGHT = 100;
const HORIZONTAL_SPACING = 80;
const VERTICAL_SPACING = 120;

export function useTaskHierarchy(options: UseTaskHierarchyOptions = {}) {
  const { refreshInterval = 30000, limit = 50, status, workspace, provider, taskId, taskIds, traceId, skip = false, groupBy } = options;
  const [data, setData] = useState<TaskHierarchyResult | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  // Build query string
  const queryString = useMemo(() => {
    const params = new URLSearchParams();
    if (limit) params.set('limit', String(limit));
    if (status?.length) params.set('status', status.join(','));
    if (workspace) params.set('workspace', workspace);
    if (provider) params.set('provider', provider);
    if (taskId) params.set('task_id', taskId);
    if (taskIds?.length) params.set('task_ids', taskIds.join(','));
    if (traceId) params.set('trace_id', traceId);
    if (groupBy) params.set('group_by', groupBy);
    const qs = params.toString();
    return qs ? `?${qs}` : '';
  }, [limit, status, workspace, provider, taskId, taskIds, traceId, groupBy]);

  const fetchData = useCallback(async () => {
    try {
      const response = await fetch(`/api/controlplane/task-hierarchy${queryString}`);
      if (!response.ok) {
        throw new Error(`HTTP error: ${response.status}`);
      }
      const result: TaskHierarchyResult = await response.json();
      setData(result);
      setError(null);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to fetch task hierarchy');
    } finally {
      setLoading(false);
    }
  }, [queryString]);

  useEffect(() => {
    // Skip fetching when spans are provided directly (the fixed path)
    if (skip) {
      setLoading(false);
      return;
    }

    fetchData();

    if (refreshInterval > 0) {
      const interval = setInterval(fetchData, refreshInterval);
      return () => clearInterval(interval);
    }
  }, [fetchData, refreshInterval, skip]);

  // Transform data to ReactFlow nodes and edges with hierarchical layout
  const graphData = useMemo(() => {
    if (!data) return { nodes: [], edges: [] };

    const nodes: Node<TaskHierarchyNode>[] = [];
    const edges: Edge[] = [];

    // Build adjacency map for layout
    const childrenMap = new Map<string, string[]>();
    const rootTasks: TaskHierarchyNode[] = [];

    for (const task of data.tasks) {
      if (task.parent_task_id) {
        const children = childrenMap.get(task.parent_task_id) || [];
        children.push(task.id);
        childrenMap.set(task.parent_task_id, children);
      } else {
        rootTasks.push(task);
      }
    }

    // Build task lookup map
    const taskMap = new Map<string, TaskHierarchyNode>();
    for (const task of data.tasks) {
      taskMap.set(task.id, task);
    }

    // Recursive layout function
    let xOffset = 0;
    const layoutTask = (task: TaskHierarchyNode, depth: number, xPos: number): number => {
      const children = childrenMap.get(task.id) || [];

      if (children.length === 0) {
        // Leaf node - place at current position
        nodes.push({
          id: task.id,
          type: 'task',
          position: { x: xPos, y: depth * (NODE_HEIGHT + VERTICAL_SPACING) },
          data: task,
        });
        return NODE_WIDTH + HORIZONTAL_SPACING;
      }

      // Layout children first
      let childX = xPos;
      let totalWidth = 0;
      for (const childId of children) {
        const childTask = taskMap.get(childId);
        if (childTask) {
          const width = layoutTask(childTask, depth + 1, childX);
          childX += width;
          totalWidth += width;
        }
      }

      // Center parent above children
      const parentX = xPos + (totalWidth - NODE_WIDTH - HORIZONTAL_SPACING) / 2;
      nodes.push({
        id: task.id,
        type: 'task',
        position: { x: Math.max(xPos, parentX), y: depth * (NODE_HEIGHT + VERTICAL_SPACING) },
        data: task,
      });

      return Math.max(totalWidth, NODE_WIDTH + HORIZONTAL_SPACING);
    };

    // Layout all root tasks
    for (const root of rootTasks) {
      const width = layoutTask(root, 0, xOffset);
      xOffset += width;
    }

    // Convert edges
    for (const edge of data.edges) {
      edges.push({
        id: `${edge.source}-${edge.target}`,
        source: edge.source,
        target: edge.target,
        type: edge.type === 'handoff' ? 'smoothstep' : 'straight',
        animated: edge.type === 'session',
        style: {
          stroke: edge.type === 'handoff' ? '#6366f1' : '#3b82f6',
          strokeWidth: edge.type === 'handoff' ? 2 : 1,
          strokeDasharray: edge.type === 'session' ? '5,5' : undefined,
        },
        label: edge.type,
        labelStyle: { fill: '#8b949e', fontSize: 10 },
      });
    }

    return { nodes, edges };
  }, [data]);

  // Get root tasks (no parent)
  const rootTasks = useMemo(() => {
    if (!data) return [];
    return data.tasks.filter(t => !t.parent_task_id);
  }, [data]);

  // Get tasks with pending approvals
  const pendingApprovalTasks = useMemo(() => {
    if (!data) return [];
    return data.tasks.filter(t => t.approval_status === 'pending');
  }, [data]);

  return {
    data,
    loading,
    error,
    refetch: fetchData,
    // Derived data
    graphData,
    rootTasks,
    pendingApprovalTasks,
    stats: data?.stats ?? null,
  };
}
