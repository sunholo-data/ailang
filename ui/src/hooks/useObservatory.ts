import { useEffect, useState, useCallback, useRef } from 'react';

// Observatory types matching Go backend models
export interface Workspace {
  id: string;
  name: string;
  path: string;
  created_at: string;
  updated_at: string;
}

export interface Task {
  id: string;
  workspace_id: string;
  parent_task_id?: string;  // Links to parent task for handoff chains
  title: string;
  description: string;
  status: 'pending' | 'running' | 'completed' | 'failed' | 'cancelled';
  source_type: string;
  source_ref?: string;
  priority: string;
  created_at: string;
  started_at?: string;
  completed_at?: string;
  // Aggregated metrics (matching Go backend)
  total_duration_ms: number;
  total_tokens_in: number;
  total_tokens_out: number;
  total_cost_usd: number;
  agent_count: number;
  span_count: number;
  error_count: number;
}

export interface Span {
  id: string;
  trace_id: string;
  parent_span_id?: string;
  task_id?: string;
  agent_assignment_id?: string;
  name: string;
  kind: string;
  status: string;
  status_message?: string;
  start_time: string;
  end_time?: string;
  duration_ms: number;
  // Normalized metrics (matching Go backend field names)
  tokens_in: number;
  tokens_out: number;
  cost_usd: number;
  model?: string;
  provider?: string;
  attributes: Record<string, any>;
  resource_attributes?: Record<string, any>;
}

export interface TraceSummary {
  trace_id: string;
  root_span: string;  // Root span name
  span_count: number;
  duration_ms: number;
  start_time: string;
  status: string;
  task_id?: string;
  service_name?: string;  // e.g., "ailang-run", "ailang-eval", "claude-code"
}

export interface Trace {
  trace_id: string;
  spans: Span[];
  task_id?: string;
  start_time: string;
  end_time?: string;
  duration_ms?: number;
}

export interface MetricsSummary {
  total_spans: number;
  total_traces: number;
  total_tasks: number;
  total_workspaces: number;
  total_agents: number;
  total_cost_usd: number;
  total_tokens_in: number;
  total_tokens_out: number;
  avg_duration_ms: number;
  error_rate: number;
  success_rate: number;
  spans_by_status: Record<string, number>;
  spans_by_provider: Record<string, number>;
}

export interface WSEvent {
  type: string;
  timestamp: string;
  data: any;
}

// Base API URL
const API_BASE = '/api/observatory';

// Fetch helper
async function fetchAPI<T>(endpoint: string, options?: RequestInit): Promise<T> {
  const response = await fetch(`${API_BASE}${endpoint}`, {
    headers: {
      'Content-Type': 'application/json',
    },
    ...options,
  });
  if (!response.ok) {
    throw new Error(`API error: ${response.statusText}`);
  }
  return response.json();
}

// Hook for workspaces
export function useWorkspaces() {
  const [workspaces, setWorkspaces] = useState<Workspace[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  const refresh = useCallback(async () => {
    setLoading(true);
    try {
      const data = await fetchAPI<Workspace[]>('/workspaces');
      setWorkspaces(data || []);
      setError(null);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to fetch workspaces');
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    refresh();
  }, [refresh]);

  return { workspaces, loading, error, refresh };
}

// Hook for tasks
interface UseTasksOptions {
  workspaceId?: string;
  status?: string;
  limit?: number;
}

export function useTasks(options: UseTasksOptions = {}) {
  const [tasks, setTasks] = useState<Task[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  const refresh = useCallback(async () => {
    setLoading(true);
    try {
      const params = new URLSearchParams();
      if (options.workspaceId) params.set('workspace_id', options.workspaceId);
      if (options.status) params.set('status', options.status);
      if (options.limit) params.set('limit', options.limit.toString());

      const query = params.toString();
      const endpoint = query ? `/tasks?${query}` : '/tasks';
      const data = await fetchAPI<Task[]>(endpoint);
      setTasks(data || []);
      setError(null);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to fetch tasks');
    } finally {
      setLoading(false);
    }
  }, [options.workspaceId, options.status, options.limit]);

  useEffect(() => {
    refresh();
  }, [refresh]);

  return { tasks, loading, error, refresh };
}

// Hook for spans
interface UseSpansOptions {
  traceId?: string;
  taskId?: string;
  status?: string;
  limit?: number;
}

export function useSpans(options: UseSpansOptions = {}) {
  const [spans, setSpans] = useState<Span[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  const refresh = useCallback(async () => {
    setLoading(true);
    try {
      const params = new URLSearchParams();
      if (options.traceId) params.set('trace_id', options.traceId);
      if (options.taskId) params.set('task_id', options.taskId);
      if (options.status) params.set('status', options.status);
      if (options.limit) params.set('limit', options.limit.toString());

      const query = params.toString();
      const endpoint = query ? `/spans?${query}` : '/spans';
      const data = await fetchAPI<Span[]>(endpoint);
      setSpans(data || []);
      setError(null);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to fetch spans');
    } finally {
      setLoading(false);
    }
  }, [options.traceId, options.taskId, options.status, options.limit]);

  useEffect(() => {
    refresh();
  }, [refresh]);

  return { spans, loading, error, refresh };
}

// Hook for traces
interface UseTracesOptions {
  taskId?: string;
  status?: string;
  limit?: number;
}

export function useTraces(options: UseTracesOptions = {}) {
  const [traces, setTraces] = useState<TraceSummary[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  const refresh = useCallback(async () => {
    setLoading(true);
    try {
      const params = new URLSearchParams();
      if (options.taskId) params.set('task_id', options.taskId);
      if (options.status) params.set('status', options.status);
      if (options.limit) params.set('limit', options.limit.toString());

      const query = params.toString();
      const endpoint = query ? `/traces?${query}` : '/traces';
      const data = await fetchAPI<TraceSummary[]>(endpoint);
      setTraces(data || []);
      setError(null);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to fetch traces');
    } finally {
      setLoading(false);
    }
  }, [options.taskId, options.status, options.limit]);

  useEffect(() => {
    refresh();
  }, [refresh]);

  return { traces, loading, error, refresh };
}

// Hook for single trace with full span tree
export function useTrace(traceId: string | null) {
  const [trace, setTrace] = useState<Trace | null>(null);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    if (!traceId) {
      setTrace(null);
      return;
    }

    setLoading(true);
    fetchAPI<Trace>(`/traces/${traceId}`)
      .then(data => {
        setTrace(data);
        setError(null);
      })
      .catch(err => {
        setError(err instanceof Error ? err.message : 'Failed to fetch trace');
      })
      .finally(() => {
        setLoading(false);
      });
  }, [traceId]);

  return { trace, loading, error };
}

// Hook for metrics summary
export function useMetrics() {
  const [metrics, setMetrics] = useState<MetricsSummary | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  const refresh = useCallback(async () => {
    setLoading(true);
    try {
      const data = await fetchAPI<MetricsSummary>('/metrics/summary');
      setMetrics(data);
      setError(null);
    } catch (err) {
      // Don't show error if just no data yet - show empty metrics
      setMetrics({
        total_spans: 0,
        total_traces: 0,
        total_tasks: 0,
        total_workspaces: 0,
        total_agents: 0,
        total_cost_usd: 0,
        total_tokens_in: 0,
        total_tokens_out: 0,
        avg_duration_ms: 0,
        error_rate: 0,
        success_rate: 0,
        spans_by_status: {},
        spans_by_provider: {},
      });
      setError(null);
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    refresh();
  }, [refresh]);

  return { metrics, loading, error, refresh };
}

// Hook for observatory WebSocket real-time updates
interface UseObservatoryWsOptions {
  onSpanCreated?: (span: Span) => void;
  onSpanUpdated?: (span: Span) => void;
  onTaskCreated?: (task: Task) => void;
  onTaskUpdated?: (task: Task) => void;
  onTaskCompleted?: (task: Task) => void;
  onMetricsUpdated?: (metrics: MetricsSummary) => void;
  workspaceId?: string;
  taskId?: string;
}

// Telemetry configuration from backend
export interface TelemetryConfig {
  gcp_enabled: boolean;
  gcp_project: string;
  gcp_trace_url?: string;
  otlp_enabled: boolean;
}

// Hook for telemetry configuration
export function useTelemetryConfig() {
  const [config, setConfig] = useState<TelemetryConfig | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    fetch('/api/telemetry/config')
      .then(response => {
        if (!response.ok) {
          throw new Error(`HTTP error: ${response.statusText}`);
        }
        return response.json();
      })
      .then(data => {
        setConfig(data);
        setError(null);
      })
      .catch(err => {
        setError(err instanceof Error ? err.message : 'Failed to fetch telemetry config');
      })
      .finally(() => {
        setLoading(false);
      });
  }, []);

  return { config, loading, error };
}

// Connection state enum
export type ConnectionState = 'connecting' | 'connected' | 'disconnected' | 'error';

// Return type for WebSocket hook
export interface UseObservatoryWsReturn {
  connectionState: ConnectionState;
  isConnected: boolean;
  lastEventTime: Date | null;
  reconnectAttempts: number;
  manualReconnect: () => void;
}

export function useObservatoryWs(options: UseObservatoryWsOptions = {}): UseObservatoryWsReturn {
  const [connectionState, setConnectionState] = useState<ConnectionState>('disconnected');
  const [lastEventTime, setLastEventTime] = useState<Date | null>(null);
  const [reconnectAttempts, setReconnectAttempts] = useState(0);
  const wsRef = useRef<WebSocket | null>(null);
  const optionsRef = useRef(options);
  const reconnectTimeoutRef = useRef<number | null>(null);

  // Update options ref
  useEffect(() => {
    optionsRef.current = options;
  }, [options]);

  const connect = useCallback(() => {
    // Clear any pending reconnect
    if (reconnectTimeoutRef.current) {
      clearTimeout(reconnectTimeoutRef.current);
      reconnectTimeoutRef.current = null;
    }

    const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
    const wsUrl = `${protocol}//${window.location.host}/ws/observatory`;

    setConnectionState('connecting');
    const ws = new WebSocket(wsUrl);
    wsRef.current = ws;

    ws.onopen = () => {
      setConnectionState('connected');
      setReconnectAttempts(0);
      // Send subscription if filters provided
      const sub: any = {};
      if (optionsRef.current.workspaceId) sub.workspace_id = optionsRef.current.workspaceId;
      if (optionsRef.current.taskId) sub.task_id = optionsRef.current.taskId;
      if (Object.keys(sub).length > 0) {
        ws.send(JSON.stringify(sub));
      }
    };

    ws.onclose = () => {
      setConnectionState('disconnected');
      // Exponential backoff: 1s, 2s, 4s, 8s, max 30s
      const delay = Math.min(1000 * Math.pow(2, reconnectAttempts), 30000);
      setReconnectAttempts(prev => prev + 1);
      reconnectTimeoutRef.current = window.setTimeout(connect, delay);
    };

    ws.onerror = () => {
      setConnectionState('error');
    };

    ws.onmessage = (event) => {
      try {
        const wsEvent: WSEvent = JSON.parse(event.data);
        setLastEventTime(new Date());
        const opts = optionsRef.current;

        switch (wsEvent.type) {
          case 'span.created':
            opts.onSpanCreated?.(wsEvent.data);
            break;
          case 'span.updated':
            opts.onSpanUpdated?.(wsEvent.data);
            break;
          case 'task.created':
            opts.onTaskCreated?.(wsEvent.data);
            break;
          case 'task.updated':
            opts.onTaskUpdated?.(wsEvent.data);
            break;
          case 'task.completed':
            opts.onTaskCompleted?.(wsEvent.data);
            break;
          case 'metrics.updated':
            opts.onMetricsUpdated?.(wsEvent.data);
            break;
        }
      } catch (err) {
        console.error('Failed to parse WebSocket message:', err);
      }
    };
  }, [reconnectAttempts]);

  const manualReconnect = useCallback(() => {
    wsRef.current?.close();
    setReconnectAttempts(0);
    connect();
  }, [connect]);

  useEffect(() => {
    connect();

    return () => {
      if (reconnectTimeoutRef.current) {
        clearTimeout(reconnectTimeoutRef.current);
      }
      wsRef.current?.close();
    };
  }, []);

  return {
    connectionState,
    isConnected: connectionState === 'connected',
    lastEventTime,
    reconnectAttempts,
    manualReconnect,
  };
}

// ===== Task Hierarchy Types (M-TASK-HIERARCHY) =====

export interface AgentAssignment {
  id: string;
  task_id: string;
  agent_id: string;
  provider: string;
  status: 'pending' | 'running' | 'completed' | 'failed' | 'cancelled';
  assigned_at: string;
  started_at?: string;
  completed_at?: string;
  duration_ms?: number;
  tokens_in?: number;
  tokens_out?: number;
  cost_usd?: number;
  tool_calls?: number;
}

export interface SpanNode {
  span: Span;
  children?: SpanNode[];
}

export interface HierarchyTraceSummary {
  span_count: number;
  total_tokens: number;
  total_cost_usd: number;
  duration_ms: number;
  error_count: number;
}

export interface TraceHierarchy {
  trace_id: string;
  root_span?: SpanNode;
  spans: SpanNode[];
  summary: HierarchyTraceSummary;
}

export interface AgentHierarchy {
  agent: AgentAssignment;
  traces: TraceHierarchy[];
}

export interface TaskHierarchy {
  task: Task;
  agents: AgentHierarchy[];
}

// Hook for task hierarchy
interface UseTaskHierarchyOptions {
  depth?: number;
  includeSpans?: boolean;
  workspace?: string;  // Filter by workspace to prevent cross-workspace span bleeding
}

export function useTaskHierarchy(taskId: string | null, options: UseTaskHierarchyOptions = {}) {
  const [hierarchy, setHierarchy] = useState<TaskHierarchy | null>(null);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const refresh = useCallback(async () => {
    if (!taskId) {
      setHierarchy(null);
      return;
    }

    setLoading(true);
    try {
      const params = new URLSearchParams();
      if (options.depth !== undefined) params.set('depth', options.depth.toString());
      if (options.includeSpans === false) params.set('include_spans', 'false');
      if (options.workspace) params.set('workspace', options.workspace);

      const query = params.toString();
      const endpoint = query ? `/tasks/${taskId}/hierarchy?${query}` : `/tasks/${taskId}/hierarchy`;
      const data = await fetchAPI<TaskHierarchy>(endpoint);
      setHierarchy(data);
      setError(null);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to fetch task hierarchy');
      setHierarchy(null);
    } finally {
      setLoading(false);
    }
  }, [taskId, options.depth, options.includeSpans, options.workspace]);

  useEffect(() => {
    refresh();
  }, [refresh]);

  return { hierarchy, loading, error, refresh };
}

// TaskTimeline type matching Go backend model
export interface TaskTimelineItem {
  task_id: string;
  title: string;
  status: string;
  span_id?: string;
  span_name?: string;
  start_time?: string;
  end_time?: string;
  duration_ms?: number;
  span_status?: string;
  tokens_in?: number;
  tokens_out?: number;
  cost_usd?: number;
  provider?: string;
}

// Hook for task timeline (Gantt-style span data)
export function useTaskTimeline(taskId: string | null) {
  const [timeline, setTimeline] = useState<TaskTimelineItem[]>([]);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const refresh = useCallback(async () => {
    if (!taskId) {
      setTimeline([]);
      return;
    }

    setLoading(true);
    try {
      const data = await fetchAPI<TaskTimelineItem[]>(`/tasks/${taskId}/timeline`);
      setTimeline(data || []);
      setError(null);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to fetch task timeline');
      setTimeline([]);
    } finally {
      setLoading(false);
    }
  }, [taskId]);

  useEffect(() => {
    refresh();
  }, [refresh]);

  return { timeline, loading, error, refresh };
}

// ===== Approval Types (M-TASK-HIERARCHY) =====

export interface Approval {
  id: string;
  task_id: string;
  thread_id?: string;
  request_type: string;
  status: 'pending' | 'approved' | 'rejected' | 'expired';
  branch_name?: string;
  worktree_path?: string;
  workspace?: string;        // Source workspace path (e.g., /Users/.../stapledons_voyage)
  summary?: string;
  diff_preview?: string;
  created_at: string;
  expires_at?: string;
  reviewed_at?: string;
  reviewed_by?: string;
  review_notes?: string;
  // Multi-channel approval workflow fields (M-DASHBOARD-APPROVAL-INTEGRATION)
  iteration?: number;        // Current iteration (1-3), retrigger count
  channel?: string;          // Source: 'dashboard' | 'cli' | 'github'
  feedback?: string;         // Harvested feedback from GitHub comments
  feedback_author?: string;  // Author of the most recent feedback
}

// Hook for approvals
interface UseApprovalsOptions {
  status?: string;
  limit?: number;
}

export function useApprovals(options: UseApprovalsOptions = {}) {
  const [approvals, setApprovals] = useState<Approval[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  const refresh = useCallback(async () => {
    setLoading(true);
    try {
      const status = options.status || 'pending';
      const data = await fetch(`/api/approvals?status=${status}`);
      if (!data.ok) {
        throw new Error(`API error: ${data.statusText}`);
      }
      const result = await data.json();
      setApprovals(result || []);
      setError(null);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to fetch approvals');
      setApprovals([]);
    } finally {
      setLoading(false);
    }
  }, [options.status]);

  useEffect(() => {
    refresh();
  }, [refresh]);

  const approveApproval = useCallback(async (approvalId: string, notes?: string) => {
    const response = await fetch(`/api/approvals/${approvalId}/approve`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ notes: notes || '' }),
    });
    if (!response.ok) {
      throw new Error('Failed to approve');
    }
    await refresh();
  }, [refresh]);

  const rejectApproval = useCallback(async (approvalId: string, notes?: string, permanent?: boolean) => {
    const response = await fetch(`/api/approvals/${approvalId}/reject`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ notes: notes || '', permanent: permanent || false }),
    });
    if (!response.ok) {
      throw new Error('Failed to reject');
    }
    await refresh();
  }, [refresh]);

  return { approvals, loading, error, refresh, approveApproval, rejectApproval };
}
