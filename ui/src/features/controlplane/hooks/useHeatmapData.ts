/**
 * Hook for fetching heatmap data from Control Plane API
 */
import { useState, useEffect, useCallback } from 'react';
import { ControlPlaneFilters, buildFilterQueryString } from '../types';

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
  filters?: ControlPlaneFilters; // Filter params for interactive filtering
}

export function useHeatmapData(options: UseHeatmapDataOptions = {}) {
  const { days = 90, refreshInterval = 30000, filters = {} } = options;
  const [data, setData] = useState<HeatmapData | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  // Build query string with days + filters
  // Use format=flat until frontend is updated to use grid format
  const filterString = buildFilterQueryString(filters);
  const queryString = filterString ? `?days=${days}&format=flat&${filterString.slice(1)}` : `?days=${days}&format=flat`;

  const fetchData = useCallback(async () => {
    try {
      const response = await fetch(`/api/controlplane/heatmap${queryString}`);
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
  }, [queryString]);

  useEffect(() => {
    fetchData();

    if (refreshInterval > 0) {
      const interval = setInterval(fetchData, refreshInterval);
      return () => clearInterval(interval);
    }
  }, [fetchData, refreshInterval]);

  return { data, loading, error, refetch: fetchData };
}
