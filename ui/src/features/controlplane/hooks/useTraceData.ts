/**
 * Hook for fetching trace/span data from Observatory API
 */
import { useState, useEffect, useCallback } from 'react';

// Chat context embedded in spans (when include_chat=true on API)
export interface ChatContext {
  user_prompt?: string;
  assistant_response?: string;
  has_thinking?: boolean;
  turn_number?: number;
  full_chat_url?: string;
}

export interface Span {
  id: string;
  name: string;
  display_name?: string; // Enriched name with tool metadata (file paths, patterns, etc.)
  startMs: number;
  durationMs: number;
  children?: Span[];
  status?: 'ok' | 'error';
  attributes?: Record<string, string>;
  // Cost and metrics fields (from backend)
  cost_usd?: number;
  tokens_in?: number;
  tokens_out?: number;
  provider?: string;
  // Chat context (from backend with include_chat=true)
  chat_context?: ChatContext;
}

export interface Trace {
  trace_id: string;
  root_span_id: string;
  service_name: string;
  duration_ms: number;
  span_count: number;
  status: string;
  timestamp: string;
}

// Raw span from API response (matches actual /api/observatory/spans response)
interface RawSpan {
  id: string;
  parent_span_id?: string;
  name: string;
  display_name?: string; // Enriched name with tool metadata
  start_time: string;
  duration_ms: number;
  status?: string;
  attributes?: Record<string, string>;
  // Cost and metrics fields
  cost_usd?: number;
  tokens_in?: number;
  tokens_out?: number;
  provider?: string;
  // Chat context (from backend with include_chat=true)
  chat_context?: ChatContext;
}

interface UseTraceDataOptions {
  traceId?: string;
  taskId?: string;
  limit?: number;
  refreshInterval?: number;
  workspace?: string;  // Filter by workspace to prevent cross-workspace span bleeding
}

export function useTraceData(options: UseTraceDataOptions = {}) {
  const { traceId, taskId, limit = 10, refreshInterval = 0, workspace } = options;
  const [traces, setTraces] = useState<Trace[]>([]);
  const [spans, setSpans] = useState<Span[]>([]);
  const [loading, setLoading] = useState(true);
  const [spansLoading, setSpansLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  // Fetch trace list
  const fetchTraces = useCallback(async () => {
    try {
      const response = await fetch(`/api/observatory/traces?limit=${limit}`);
      if (!response.ok) {
        throw new Error(`HTTP error: ${response.status}`);
      }
      const result = await response.json();
      setTraces(result || []);
      setError(null);
    } catch (err) {
      // Silently handle - observatory might not be configured
      setTraces([]);
    } finally {
      setLoading(false);
    }
  }, [limit]);

  // Convert HierarchicalSpan from API to local Span type
  // Must be defined before fetchByTraceId to avoid use-before-define linter error
  const convertHierarchicalSpan = (hs: any): Span => {
    const span: Span = {
      id: hs.id,
      name: hs.name,
      display_name: hs.display_name,
      startMs: new Date(hs.start_time).getTime(),
      durationMs: hs.duration_ms,
      status: hs.status === 'error' || hs.status === 'ERROR' ? 'error' : 'ok',
      attributes: hs.attributes,
      cost_usd: hs.cost_usd,
      tokens_in: hs.tokens_in,
      tokens_out: hs.tokens_out,
      provider: hs.provider,
      chat_context: hs.chat_context, // Embedded conversation content
      children: hs.children?.map(convertHierarchicalSpan) || [],
    };
    return span;
  };

  // Fetch spans by trace_id using enriched endpoint with hierarchical format
  // Returns Span[] with children already populated (server-side hierarchy building)
  const fetchByTraceId = async (tid: string): Promise<Span[]> => {
    // Use enriched endpoint with hierarchical=true for server-side hierarchy building
    // This removes ~50 lines of client-side buildSpanHierarchy logic
    // include_chat=true embeds conversation context from chat_messages table
    const response = await fetch(`/api/observatory/spans/enriched?trace_id=${tid}&limit=100&hierarchical=true&include_chat=true`);
    if (!response.ok) {
      // Fallback to regular endpoint with client-side hierarchy (also request chat context)
      const fallbackResponse = await fetch(`/api/observatory/spans?trace_id=${tid}&limit=100&include_chat=true`);
      if (!fallbackResponse.ok) return [];
      const rawSpans = await fallbackResponse.json() || [];
      return buildSpanHierarchy(rawSpans);
    }
    const result = await response.json();

    // If server returned hierarchical format, spans already have children
    if (result.hierarchical) {
      return (result.spans || []).map(convertHierarchicalSpan);
    }

    // Fallback: server returned flat format, build hierarchy client-side
    return buildSpanHierarchy(result.spans || []);
  };

  // Fetch spans by task_id using the hierarchy endpoint
  // Preserves the hierarchical structure from the backend (children[] arrays)
  const fetchByTaskId = async (tid: string): Promise<Span[]> => {
    // Use hierarchy endpoint which does proper trace expansion
    // Include workspace param to prevent cross-workspace span bleeding
    const params = new URLSearchParams();
    if (workspace) params.set('workspace', workspace);
    params.set('include_chat', 'true');
    const url = `/api/observatory/tasks/${tid}/hierarchy?${params.toString()}`;
    const response = await fetch(url);
    if (!response.ok) {
      // Fallback to direct spans query if hierarchy fails (also request chat context)
      const fallbackUrl = workspace
        ? `/api/observatory/spans?task_id=${tid}&limit=100&workspace=${encodeURIComponent(workspace)}&include_chat=true`
        : `/api/observatory/spans?task_id=${tid}&limit=100&include_chat=true`;
      const fallbackResponse = await fetch(fallbackUrl);
      if (!fallbackResponse.ok) return [];
      const rawSpans = await fallbackResponse.json() || [];
      return buildSpanHierarchy(rawSpans);  // Use old approach for fallback
    }

    const hierarchy = await response.json();

    // Transform SpanNodes to Spans, preserving children hierarchy
    const transformSpanNode = (spanNode: any, minStart: number): Span | null => {
      if (!spanNode?.span) return null;

      const raw = spanNode.span;
      const startMs = new Date(raw.start_time).getTime() - minStart;

      const span: Span = {
        id: raw.id,
        name: raw.name,
        display_name: raw.display_name, // Enriched name from session_tools
        startMs,
        durationMs: raw.duration_ms,
        status: raw.status === 'error' || raw.status === 'ERROR' ? 'error' : 'ok',
        attributes: raw.attributes,
        // Copy cost and metrics fields from backend
        cost_usd: raw.cost_usd,
        tokens_in: raw.tokens_in,
        tokens_out: raw.tokens_out,
        provider: raw.provider,
        chat_context: raw.chat_context, // Embedded conversation content
        children: [], // Will populate below
      };

      // Recursively transform children (preserving backend hierarchy!)
      if (spanNode.children && Array.isArray(spanNode.children)) {
        span.children = spanNode.children
          .map((child: any) => transformSpanNode(child, minStart))
          .filter((s: Span | null): s is Span => s !== null)
          .sort((a: Span, b: Span) => a.startMs - b.startMs);
      }

      return span;
    };

    // Find minimum start time across all spans (traverse full hierarchy)
    const findMinStart = (nodes: any[]): number => {
      let min = Infinity;
      const traverse = (node: any) => {
        if (node?.span?.start_time) {
          const t = new Date(node.span.start_time).getTime();
          if (t < min) min = t;
        }
        node?.children?.forEach(traverse);
      };
      nodes.forEach(traverse);
      return min === Infinity ? 0 : min;
    };

    // Extract root spans from hierarchy response, preserving children
    const rootSpans: Span[] = [];

    if (hierarchy?.agents) {
      for (const agent of hierarchy.agents) {
        if (agent?.traces) {
          for (const trace of agent.traces) {
            if (trace?.spans) {
              const minStart = findMinStart(trace.spans);
              for (const spanNode of trace.spans) {
                const span = transformSpanNode(spanNode, minStart);
                if (span) rootSpans.push(span);
              }
            }
          }
        }
      }
    }

    // Sort root spans by start time
    rootSpans.sort((a, b) => a.startMs - b.startMs);

    return rootSpans;
  };

  // Fetch spans for a specific trace or task - tries multiple approaches
  // Both fetchByTraceId and fetchByTaskId now return hierarchical Span[] directly
  const fetchSpans = useCallback(async (id: string, idType?: 'trace' | 'task' | 'auto') => {
    if (!id) return;

    setSpansLoading(true);
    try {
      let hierarchicalSpans: Span[] = [];
      const type = idType || 'auto';

      if (type === 'trace') {
        // Trace ID path: server returns hierarchical format
        hierarchicalSpans = await fetchByTraceId(id);
      } else if (type === 'task') {
        // Task ID path: returns Span[] with hierarchy already preserved
        hierarchicalSpans = await fetchByTaskId(id);
      } else {
        // Auto mode: try trace_id first, then task_id
        hierarchicalSpans = await fetchByTraceId(id);
        if (hierarchicalSpans.length === 0) {
          // Try as task_id
          hierarchicalSpans = await fetchByTaskId(id);
          if (hierarchicalSpans.length === 0) {
            // Try with "task-" prefix if not already present
            if (!id.startsWith('task-')) {
              hierarchicalSpans = await fetchByTaskId(`task-${id}`);
            }
          }
        }
      }

      setSpans(hierarchicalSpans);
      setError(null);
    } catch (err) {
      setSpans([]);
    } finally {
      setSpansLoading(false);
    }
  }, []);

  useEffect(() => {
    fetchTraces();

    if (refreshInterval > 0) {
      const interval = setInterval(fetchTraces, refreshInterval);
      return () => clearInterval(interval);
    }
  }, [fetchTraces, refreshInterval]);

  useEffect(() => {
    if (traceId) {
      fetchSpans(traceId, 'trace');
    } else if (taskId) {
      fetchSpans(taskId, 'task');
    }
  }, [traceId, taskId, fetchSpans]);

  return {
    traces,
    spans,
    loading,
    spansLoading,
    error,
    refetchTraces: fetchTraces,
    fetchSpansForTrace: fetchSpans,
  };
}

// Transform flat span list to hierarchical structure
function buildSpanHierarchy(rawSpans: RawSpan[]): Span[] {
  if (!rawSpans.length) return [];

  // Find the minimum start time
  const minStart = Math.min(...rawSpans.map((s) => new Date(s.start_time).getTime()));

  // Convert to map for parent lookup
  const spanMap = new Map<string, Span>();
  const childMap = new Map<string, string[]>();

  rawSpans.forEach((raw) => {
    const span: Span = {
      id: raw.id,
      name: raw.name,
      display_name: raw.display_name, // Enriched name with tool metadata
      startMs: new Date(raw.start_time).getTime() - minStart,
      durationMs: raw.duration_ms,
      status: raw.status === 'error' || raw.status === 'ERROR' ? 'error' : 'ok',
      attributes: raw.attributes,
      // Copy cost and metrics fields
      cost_usd: raw.cost_usd,
      tokens_in: raw.tokens_in,
      tokens_out: raw.tokens_out,
      provider: raw.provider,
      chat_context: raw.chat_context, // Embedded conversation content
      children: [],
    };
    spanMap.set(span.id, span);

    if (raw.parent_span_id) {
      const children = childMap.get(raw.parent_span_id) || [];
      children.push(raw.id);
      childMap.set(raw.parent_span_id, children);
    }
  });

  // Build hierarchy
  const rootSpans: Span[] = [];

  rawSpans.forEach((raw) => {
    const span = spanMap.get(raw.id)!;
    const childIds = childMap.get(raw.id) || [];
    span.children = childIds.map((id) => spanMap.get(id)!).filter(Boolean);

    if (!raw.parent_span_id) {
      rootSpans.push(span);
    }
  });

  // Sort root spans and their children by start time
  rootSpans.sort((a, b) => a.startMs - b.startMs);
  rootSpans.forEach((root) => {
    root.children?.sort((a, b) => a.startMs - b.startMs);
  });

  return rootSpans;
}
