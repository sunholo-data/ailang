/**
 * Hook for fetching the list of execution chains from the Chains API.
 * Used by ChainList to browse all chains with filtering and pagination.
 */
import { useState, useEffect, useCallback, useRef } from 'react';
import type { ChainSummary, ChainListFilters } from '../components/ChainExplorer/types';

export interface UseChainListOptions {
  filters?: ChainListFilters;
  limit?: number;
  refreshInterval?: number; // ms, 0 = disabled
}

export interface UseChainListResult {
  chains: ChainSummary[];
  loading: boolean;
  error: string | null;
  total: number;
  hasMore: boolean;
  loadMore: () => void;
  refetch: () => void;
}

export function useChainList(options: UseChainListOptions = {}): UseChainListResult {
  const { filters, limit = 50, refreshInterval = 0 } = options;
  const [chains, setChains] = useState<ChainSummary[]>([]);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [offset, setOffset] = useState(0);
  const [hasMore, setHasMore] = useState(false);
  const [fetchTrigger, setFetchTrigger] = useState(0);
  const lastFiltersRef = useRef<string>('');

  const refetch = useCallback(() => {
    setOffset(0);
    setFetchTrigger(prev => prev + 1);
  }, []);

  const loadMore = useCallback(() => {
    if (hasMore && !loading) {
      setOffset(prev => prev + limit);
    }
  }, [hasMore, loading, limit]);

  useEffect(() => {
    // Reset pagination when filters change
    const filterKey = JSON.stringify(filters || {});
    if (filterKey !== lastFiltersRef.current) {
      lastFiltersRef.current = filterKey;
      setOffset(0);
      setChains([]);
    }

    let cancelled = false;

    async function fetchChains() {
      setLoading(true);
      setError(null);

      try {
        const params = new URLSearchParams();
        params.set('limit', String(limit));
        params.set('offset', String(offset));
        if (filters?.status) params.set('status', filters.status);
        if (filters?.source_type) params.set('source_type', filters.source_type);
        if (filters?.agent_id) params.set('agent_id', filters.agent_id);
        if (filters?.since) params.set('since', String(filters.since));

        const resp = await fetch(`/api/chains?${params.toString()}`);
        if (!resp.ok) throw new Error(`Failed to fetch chains: ${resp.status}`);
        if (cancelled) return;

        const data: ChainSummary[] = await resp.json();

        if (!cancelled) {
          if (offset === 0) {
            setChains(data);
          } else {
            setChains(prev => [...prev, ...data]);
          }
          setHasMore(data.length >= limit);
        }
      } catch (err) {
        if (!cancelled) {
          setError(err instanceof Error ? err.message : 'Failed to fetch chains');
        }
      } finally {
        if (!cancelled) setLoading(false);
      }
    }

    fetchChains();

    // Optional polling
    let intervalId: ReturnType<typeof setInterval> | null = null;
    if (refreshInterval > 0) {
      intervalId = setInterval(fetchChains, refreshInterval);
    }

    return () => {
      cancelled = true;
      if (intervalId) clearInterval(intervalId);
    };
  }, [filters, limit, offset, fetchTrigger, refreshInterval]);

  return {
    chains,
    loading,
    error,
    total: chains.length,
    hasMore,
    loadMore,
    refetch,
  };
}
