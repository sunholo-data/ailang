/**
 * Hook for fetching trace/span data from Observatory API
 */
import { useState, useEffect, useCallback } from 'react';

export interface Span {
  id: string;
  name: string;
  startMs: number;
  durationMs: number;
  children?: Span[];
  status?: 'ok' | 'error';
  attributes?: Record<string, string>;
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

interface UseTraceDataOptions {
  traceId?: string;
  limit?: number;
  refreshInterval?: number;
}

export function useTraceData(options: UseTraceDataOptions = {}) {
  const { traceId, limit = 10, refreshInterval = 0 } = options;
  const [traces, setTraces] = useState<Trace[]>([]);
  const [spans, setSpans] = useState<Span[]>([]);
  const [loading, setLoading] = useState(true);
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

  // Fetch spans for a specific trace
  const fetchSpans = useCallback(async (tid: string) => {
    if (!tid) return;

    try {
      const response = await fetch(`/api/observatory/spans?trace_id=${tid}&limit=100`);
      if (!response.ok) {
        throw new Error(`HTTP error: ${response.status}`);
      }
      const result = await response.json();

      // Transform flat spans to hierarchical structure
      const hierarchicalSpans = buildSpanHierarchy(result || []);
      setSpans(hierarchicalSpans);
      setError(null);
    } catch (err) {
      setSpans([]);
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
      fetchSpans(traceId);
    }
  }, [traceId, fetchSpans]);

  return {
    traces,
    spans,
    loading,
    error,
    refetchTraces: fetchTraces,
    fetchSpansForTrace: fetchSpans,
  };
}

// Transform flat span list to hierarchical structure
interface RawSpan {
  span_id: string;
  parent_span_id?: string;
  operation_name: string;
  start_time: string;
  duration_ms: number;
  status_code?: string;
  attributes?: Record<string, string>;
}

function buildSpanHierarchy(rawSpans: RawSpan[]): Span[] {
  if (!rawSpans.length) return [];

  // Find the minimum start time
  const minStart = Math.min(...rawSpans.map((s) => new Date(s.start_time).getTime()));

  // Convert to map for parent lookup
  const spanMap = new Map<string, Span>();
  const childMap = new Map<string, string[]>();

  rawSpans.forEach((raw) => {
    const span: Span = {
      id: raw.span_id,
      name: raw.operation_name,
      startMs: new Date(raw.start_time).getTime() - minStart,
      durationMs: raw.duration_ms,
      status: raw.status_code === 'ERROR' ? 'error' : 'ok',
      attributes: raw.attributes,
      children: [],
    };
    spanMap.set(span.id, span);

    if (raw.parent_span_id) {
      const children = childMap.get(raw.parent_span_id) || [];
      children.push(raw.span_id);
      childMap.set(raw.parent_span_id, children);
    }
  });

  // Build hierarchy
  const rootSpans: Span[] = [];

  rawSpans.forEach((raw) => {
    const span = spanMap.get(raw.span_id)!;
    const childIds = childMap.get(raw.span_id) || [];
    span.children = childIds.map((id) => spanMap.get(id)!).filter(Boolean);

    if (!raw.parent_span_id) {
      rootSpans.push(span);
    }
  });

  // Sort by start time
  rootSpans.sort((a, b) => a.startMs - b.startMs);

  return rootSpans;
}
