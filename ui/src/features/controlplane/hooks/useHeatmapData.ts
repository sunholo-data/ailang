/**
 * Hook for fetching heatmap data from Control Plane API
 */
import { useState, useEffect, useCallback } from 'react';

export interface HeatmapCell {
  date: string;
  taskCount: number;
  cost: number;
  successRate: number;
}

export interface HeatmapData {
  cells: HeatmapCell[];
  totals: {
    tasks: number;
    cost: number;
  };
}

interface UseHeatmapDataOptions {
  days?: number;
  refreshInterval?: number; // ms, 0 to disable
}

export function useHeatmapData(options: UseHeatmapDataOptions = {}) {
  const { days = 90, refreshInterval = 30000 } = options;
  const [data, setData] = useState<HeatmapData | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  const fetchData = useCallback(async () => {
    try {
      const response = await fetch(`/api/controlplane/heatmap?days=${days}`);
      if (!response.ok) {
        throw new Error(`HTTP error: ${response.status}`);
      }
      const result = await response.json();
      setData(result);
      setError(null);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to fetch heatmap data');
    } finally {
      setLoading(false);
    }
  }, [days]);

  useEffect(() => {
    fetchData();

    if (refreshInterval > 0) {
      const interval = setInterval(fetchData, refreshInterval);
      return () => clearInterval(interval);
    }
  }, [fetchData, refreshInterval]);

  return { data, loading, error, refetch: fetchData };
}
