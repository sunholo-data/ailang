/**
 * Hook for lazy-loading spans for a specific chain stage (L2 tiered loading).
 *
 * Fetches SpanLite data (no attributes blob) via the new
 * GET /api/chains/{chainId}/stages/{stageId}/spans endpoint.
 * Only fetches when a stage is actively selected — not on page load.
 *
 * M-PERF-OBSERVATORY Phase 4.3: Lazy stage span loading
 */
import { useState, useEffect, useCallback, useRef } from 'react';
import type { Span } from '../components/ExecHierarchy/types';

export interface SpanLite {
  id: string;
  trace_id: string;
  parent_span_id?: string;
  chain_id?: string;
  stage_id?: string;
  name: string;
  kind: string;
  status: string;
  status_message?: string;
  start_time: string;
  end_time?: string;
  duration_ms: number;
  tokens_in?: number;
  tokens_out?: number;
  cost_usd?: number;
  model?: string;
  provider?: string;
}

export interface SpanLitePage {
  spans: SpanLite[];
  total: number;
  limit: number;
  offset: number;
}

export interface UseStageSpansOptions {
  chainId?: string | null;
  stageId?: string | null;
  limit?: number;
  offset?: number;
}

export interface UseStageSpansResult {
  spans: Span[];
  total: number;
  loading: boolean;
  error: string | null;
  refetch: () => void;
}

function spanLiteToSpan(lite: SpanLite): Span {
  return {
    id: lite.id,
    name: lite.name,
    display_name: lite.name,
    startMs: lite.start_time ? new Date(lite.start_time).getTime() : 0,
    durationMs: lite.duration_ms || 0,
    status: lite.status === 'error' || lite.status === 'ERROR' ? 'error' : 'ok',
    attributes: {},
    cost_usd: lite.cost_usd || 0,
    tokens_in: lite.tokens_in || 0,
    tokens_out: lite.tokens_out || 0,
    provider: lite.provider,
    children: [],
  };
}

export function useStageSpans(options: UseStageSpansOptions = {}): UseStageSpansResult {
  const { chainId, stageId, limit = 200, offset = 0 } = options;
  const [spans, setSpans] = useState<Span[]>([]);
  const [total, setTotal] = useState(0);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [fetchTrigger, setFetchTrigger] = useState(0);
  const cacheRef = useRef<Map<string, { spans: Span[]; total: number }>>(new Map());

  const refetch = useCallback(() => {
    setFetchTrigger(prev => prev + 1);
  }, []);

  useEffect(() => {
    if (!chainId || !stageId) {
      setSpans([]);
      setTotal(0);
      setLoading(false);
      return;
    }

    // Check cache first
    const cacheKey = `${chainId}-${stageId}-${limit}-${offset}`;
    if (fetchTrigger === 0) {
      const cached = cacheRef.current.get(cacheKey);
      if (cached) {
        setSpans(cached.spans);
        setTotal(cached.total);
        return;
      }
    }

    let cancelled = false;

    async function fetchSpans() {
      setLoading(true);
      setError(null);

      try {
        const params = new URLSearchParams({
          limit: String(limit),
          offset: String(offset),
        });

        const resp = await fetch(
          `/api/chains/${encodeURIComponent(chainId!)}/stages/${encodeURIComponent(stageId!)}/spans?${params}`
        );

        if (cancelled) return;

        if (!resp.ok) {
          throw new Error(`Failed to fetch stage spans: ${resp.status}`);
        }

        const page: SpanLitePage = await resp.json();
        const converted = (page.spans || []).map(spanLiteToSpan);

        // Cache the result
        cacheRef.current.set(cacheKey, { spans: converted, total: page.total });

        if (!cancelled) {
          setSpans(converted);
          setTotal(page.total);
        }
      } catch (err) {
        if (!cancelled) {
          setError(err instanceof Error ? err.message : 'Failed to fetch stage spans');
        }
      } finally {
        if (!cancelled) {
          setLoading(false);
        }
      }
    }

    fetchSpans();

    return () => {
      cancelled = true;
    };
  }, [chainId, stageId, limit, offset, fetchTrigger]);

  return { spans, total, loading, error, refetch };
}
