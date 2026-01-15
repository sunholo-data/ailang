/**
 * Hook for fetching heatmap data from Control Plane API
 * Now uses grid format by default for server-side date calculations
 */
import { useState, useEffect, useCallback, useMemo } from 'react';
import { ControlPlaneFilters, buildFilterQueryString } from '../types';

export interface HeatmapCell {
  date: string;
  taskCount: number;
  cost: number;
  successRate: number;
}

// Grid cell from API (includes intensity pre-calculated)
export interface HeatmapGridCell {
  date: string;
  count: number;
  cost: number;
  successRate: number;
  intensity: number; // 0-1
  dayOfWeek: number;
}

// Month label from API
export interface HeatmapMonthLabel {
  name: string;
  weekIndex: number;
}

// Grid format response from API
export interface HeatmapGridData {
  weeks: HeatmapGridCell[][];
  monthLabels: HeatmapMonthLabel[];
  totals: {
    tasks: number;
    cost: number;
  };
  dateRange: {
    start: string;
    end: string;
  };
}

// Legacy flat format for backwards compatibility
export interface HeatmapFlatData {
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
  format?: 'grid' | 'flat'; // API response format
}

export function useHeatmapData(options: UseHeatmapDataOptions = {}) {
  const { days = 90, refreshInterval = 30000, filters = {}, format = 'grid' } = options;
  const [gridData, setGridData] = useState<HeatmapGridData | null>(null);
  const [flatData, setFlatData] = useState<HeatmapFlatData | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  // Build query string with days + filters + format
  const filterString = buildFilterQueryString(filters);
  const baseQuery = `?days=${days}&format=${format}`;
  const queryString = filterString ? `${baseQuery}&${filterString.slice(1)}` : baseQuery;

  const fetchData = useCallback(async () => {
    try {
      const response = await fetch(`/api/controlplane/heatmap${queryString}`);
      if (!response.ok) {
        throw new Error(`HTTP error: ${response.status}`);
      }
      const result = await response.json();
      if (format === 'grid') {
        setGridData(result);
        setFlatData(null);
      } else {
        setFlatData(result);
        setGridData(null);
      }
      setError(null);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to fetch heatmap data');
    } finally {
      setLoading(false);
    }
  }, [queryString, format]);

  useEffect(() => {
    fetchData();

    if (refreshInterval > 0) {
      const interval = setInterval(fetchData, refreshInterval);
      return () => clearInterval(interval);
    }
  }, [fetchData, refreshInterval]);

  // Convert grid data to flat cells for components that still use flat format
  const cells = useMemo(() => {
    if (flatData) return flatData.cells;
    if (!gridData) return [];

    const result: HeatmapCell[] = [];
    for (const week of gridData.weeks) {
      for (const cell of week) {
        if (cell.date) {
          result.push({
            date: cell.date,
            taskCount: cell.count,
            cost: cell.cost,
            successRate: cell.successRate,
          });
        }
      }
    }
    return result;
  }, [gridData, flatData]);

  const totals = gridData?.totals ?? flatData?.totals ?? { tasks: 0, cost: 0 };

  return {
    // Grid format (new)
    gridData,
    // Flat format (legacy)
    data: flatData ?? { cells, totals },
    cells,
    totals,
    loading,
    error,
    refetch: fetchData
  };
}
