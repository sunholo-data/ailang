/**
 * Hook for fetching statistics from Control Plane API
 */
import { useState, useEffect, useCallback } from 'react';

export interface CoordinatorStats {
  total_tasks: number;
  pending_tasks: number;
  running_tasks: number;
  completed_tasks: number;
  failed_tasks: number;
  total_cost: number;
  total_tokens: number;
  active_agents: number;
  pending_approvals: number;
  success_rate: number;
}

export interface StatisticsData {
  threads: {
    total: number;
    by_status: Record<string, number>;
    by_workspace: Record<string, number>;
  };
  coordinator?: CoordinatorStats;
}

interface UseStatisticsOptions {
  refreshInterval?: number; // ms, 0 to disable
}

export function useStatistics(options: UseStatisticsOptions = {}) {
  const { refreshInterval = 10000 } = options;
  const [data, setData] = useState<StatisticsData | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  const fetchData = useCallback(async () => {
    try {
      const response = await fetch('/api/statistics');
      if (!response.ok) {
        throw new Error(`HTTP error: ${response.status}`);
      }
      const result = await response.json();
      setData(result);
      setError(null);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to fetch statistics');
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    fetchData();

    if (refreshInterval > 0) {
      const interval = setInterval(fetchData, refreshInterval);
      return () => clearInterval(interval);
    }
  }, [fetchData, refreshInterval]);

  // Derive formatted stats for display
  const stats = data?.coordinator
    ? {
        activeAgents: data.coordinator.active_agents,
        pendingApprovals: data.coordinator.pending_approvals,
        taskSuccess: `${(data.coordinator.success_rate * 100).toFixed(1)}%`,
        totalCost: `$${data.coordinator.total_cost.toFixed(2)}`,
        completedTasks: data.coordinator.completed_tasks,
        failedTasks: data.coordinator.failed_tasks,
        totalTokens: data.coordinator.total_tokens,
      }
    : null;

  return { data, stats, loading, error, refetch: fetchData };
}
