/**
 * Hook for fetching unified stats from Control Plane API
 * Combines Observatory telemetry with Coordinator runtime state
 */
import { useState, useEffect, useCallback } from 'react';
import { ControlPlaneFilters, buildFilterQueryString } from '../types';

// Observatory metrics (canonical source of truth for telemetry)
export interface ObservatoryStats {
  total_spans: number;
  total_tasks: number;
  total_workspaces: number;
  total_agents: number;
  total_tokens_in: number;
  total_tokens_out: number;
  total_cost_usd: number;
  success_rate: number;
}

// Coordinator runtime state (subset - delegated tasks only)
export interface CoordinatorRuntimeStats {
  running: boolean;
  completed_tasks: number;
  pending_tasks: number;
  running_tasks: number;
  failed_tasks: number;
  pending_approvals: number;
  active_agents: number;
  total_cost: number;
  total_tokens: number;
}

// Data source metadata
export interface DataSources {
  observatory_db: string;
  coordinator_db: string;
  observatory_ok: boolean;
  coordinator_ok: boolean;
}

// Full unified stats response
export interface UnifiedStatsData {
  observatory?: ObservatoryStats;
  coordinator?: CoordinatorRuntimeStats;
  sources: DataSources;
}

interface UseControlPlaneStatsOptions {
  refreshInterval?: number; // ms, 0 to disable
  filters?: ControlPlaneFilters; // Filter params for interactive filtering
}

export function useControlPlaneStats(options: UseControlPlaneStatsOptions = {}) {
  const { refreshInterval = 10000, filters = {} } = options;
  const [data, setData] = useState<UnifiedStatsData | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  // Memoize filter string to detect changes
  const filterString = buildFilterQueryString(filters);

  const fetchData = useCallback(async () => {
    try {
      const response = await fetch(`/api/controlplane/stats${filterString}`);
      if (!response.ok) {
        throw new Error(`HTTP error: ${response.status}`);
      }
      const result = await response.json();
      setData(result);
      setError(null);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to fetch control plane stats');
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

  // Format numbers for display
  const formatNumber = (n: number): string => {
    if (n >= 1_000_000) return `${(n / 1_000_000).toFixed(1)}M`;
    if (n >= 1_000) return `${(n / 1_000).toFixed(1)}K`;
    return n.toString();
  };

  // Derive formatted stats for header display
  // Priority: Observatory (canonical) with Coordinator (runtime state)
  const stats = data
    ? {
        // From Observatory (canonical telemetry)
        totalSpans: data.observatory?.total_spans ?? 0,
        totalTasks: data.observatory?.total_tasks ?? 0,
        totalTokens: formatNumber(
          (data.observatory?.total_tokens_in ?? 0) + (data.observatory?.total_tokens_out ?? 0)
        ),
        totalTokensIn: formatNumber(data.observatory?.total_tokens_in ?? 0),
        totalTokensOut: formatNumber(data.observatory?.total_tokens_out ?? 0),
        totalCost: `$${(data.observatory?.total_cost_usd ?? 0).toFixed(2)}`,
        successRate: `${((data.observatory?.success_rate ?? 0) * 100).toFixed(1)}%`,

        // From Coordinator (runtime state)
        activeAgents: data.coordinator?.active_agents ?? 0,
        pendingApprovals: data.coordinator?.pending_approvals ?? 0,
        coordinatorRunning: data.coordinator?.running ?? false,
        coordinatorTasks: data.coordinator?.completed_tasks ?? 0,
        coordinatorCost: `$${(data.coordinator?.total_cost ?? 0).toFixed(2)}`,

        // Data source status
        observatoryOK: data.sources.observatory_ok,
        coordinatorOK: data.sources.coordinator_ok,
      }
    : null;

  return { data, stats, loading, error, refetch: fetchData };
}
