/**
 * Hook for fetching breakdown data from Control Plane API
 * Provides drill-down data by provider, source type, model, and workspace
 */
import { useState, useEffect, useCallback } from 'react';
import { ControlPlaneFilters, buildFilterQueryString } from '../types';

// Breakdown item type matches backend BreakdownItem
export interface BreakdownItem {
  id: string;
  label: string;
  span_count: number;
  task_count?: number;
  tokens_in: number;
  tokens_out: number;
  cost_usd: number;
  duration_ms: number;  // Total execution time in ms
  percentage?: number;

  // Cache metrics
  cache_read_tokens: number;
  cache_creation_tokens: number;
  cache_savings_usd: number;
}

export interface BreakdownData {
  by_provider: BreakdownItem[];
  by_source_type: BreakdownItem[];
  by_model: BreakdownItem[];
  by_workspace: BreakdownItem[];
  total_cost: number;
}

interface UseBreakdownDataOptions {
  refreshInterval?: number; // ms, 0 to disable
  filters?: ControlPlaneFilters; // Filter params for interactive filtering
}

export function useBreakdownData(options: UseBreakdownDataOptions = {}) {
  const { refreshInterval = 120000, filters = {} } = options; // Default 120s for breakdown (cost optimization M-COST2)
  const [data, setData] = useState<BreakdownData | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  // Memoize filter string to detect changes
  const filterString = buildFilterQueryString(filters);

  const fetchData = useCallback(async () => {
    try {
      const response = await fetch(`/api/controlplane/stats/breakdown${filterString}`);
      if (!response.ok) {
        throw new Error(`HTTP error: ${response.status}`);
      }
      const result = await response.json();
      setData(result);
      setError(null);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to fetch breakdown data');
    } finally {
      setLoading(false);
    }
  }, [filterString]);

  useEffect(() => {
    fetchData();

    if (refreshInterval > 0) {
      const interval = setInterval(fetchData, refreshInterval);
      return () => clearInterval(interval);
    }
  }, [fetchData, refreshInterval]);

  // Format cost for display
  const formatCost = (cost: number): string => {
    if (cost >= 1000) return `$${(cost / 1000).toFixed(1)}K`;
    if (cost >= 1) return `$${cost.toFixed(2)}`;
    if (cost >= 0.01) return `$${cost.toFixed(2)}`;
    return `$${cost.toFixed(4)}`;
  };

  // Format percentage for display
  const formatPercentage = (pct?: number): string => {
    if (pct === undefined) return '';
    if (pct >= 10) return `${pct.toFixed(0)}%`;
    if (pct >= 1) return `${pct.toFixed(1)}%`;
    return `${pct.toFixed(2)}%`;
  };

  // Transform breakdown items for display with formatted values
  const formatBreakdownItems = (items: BreakdownItem[]) => {
    return items.map(item => ({
      ...item,
      costFormatted: formatCost(item.cost_usd),
      percentageFormatted: formatPercentage(item.percentage),
      tokensFormatted: `${(item.tokens_in + item.tokens_out).toLocaleString()}`,
    }));
  };

  // Get formatted breakdowns (guard against null sub-arrays from API)
  const breakdowns = data ? {
    byProvider: formatBreakdownItems(data.by_provider || []),
    bySourceType: formatBreakdownItems(data.by_source_type || []),
    byModel: formatBreakdownItems(data.by_model || []),
    byWorkspace: formatBreakdownItems(data.by_workspace || []),
    totalCost: formatCost(data.total_cost || 0),
  } : null;

  return { data, breakdowns, loading, error, refetch: fetchData };
}
