import { useState, useEffect, useCallback } from 'react';
import type { ControlPlaneFilters } from '../features/controlplane/types';

const API_BASE = window.location.origin;

// Task Evolution types
export interface TaskEvolutionPoint {
  x: number;
  timestamp: string;
  cost: number;
  delta_cost: number;
  tokens: number;
  tokens_in: number;
  tokens_out: number;
  turns: number;
  spans: number;
  duration_ms: number;       // Cumulative execution time in ms
  delta_duration_ms: number; // Duration of this span in ms
  elapsed_ms: number;        // Wall clock time since task start in ms
}

export interface TaskEvolutionData {
  task_id: string;
  title: string;
  provider: string;
  status: string;
  points: TaskEvolutionPoint[];
}

export interface TaskEvolutionResponse {
  tasks: TaskEvolutionData[];
  metric: string;
  cli_command: string;
}

// Usage Time Series types
export interface UsageTimeSeriesPoint {
  bucket: string;
  bucket_end: string;
  cost: number;
  tokens: number;
  tokens_in: number;
  tokens_out: number;
  turns: number;
  spans: number;
  task_count: number;
  duration_ms: number; // Total duration in bucket (ms)
  by_dimension?: Record<string, number>;
}

export interface UsageTimeSeriesResponse {
  points: UsageTimeSeriesPoint[];
  metric: string;
  interval: string;
  split_by?: string;
  cli_command: string;
}

// Token Distribution types
export interface TokenBucket {
  label: string;
  min: number;
  max: number;
  count: number;
}

export interface TokenDistributionResponse {
  buckets: TokenBucket[];
  cli_command: string;
}

// Build query string from filters
function buildQueryString(
  params: Record<string, string | number | undefined>
): string {
  const query = new URLSearchParams();
  for (const [key, value] of Object.entries(params)) {
    if (value !== undefined && value !== '' && value !== null) {
      query.set(key, String(value));
    }
  }
  return query.toString();
}

// Hook for Task Evolution data
export function useTaskEvolution(
  filters: ControlPlaneFilters,
  metric: 'cost' | 'tokens' | 'turns' | 'spans' | 'duration' = 'cost',
  limit: number = 10
) {
  const [data, setData] = useState<TaskEvolutionResponse | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<Error | null>(null);

  const fetchData = useCallback(async () => {
    setLoading(true);
    setError(null);

    try {
      const params = {
        metric,
        limit,
        provider: filters.provider,
        model: filters.model,
        workspace: filters.workspace,
        start_date: filters.start_date,
        end_date: filters.end_date,
      };

      const queryString = buildQueryString(params);
      const url = `${API_BASE}/api/controlplane/task-evolution?${queryString}`;

      const response = await fetch(url);
      if (!response.ok) {
        throw new Error(`HTTP ${response.status}: ${response.statusText}`);
      }

      const result: TaskEvolutionResponse = await response.json();
      setData(result);
    } catch (err) {
      setError(err instanceof Error ? err : new Error(String(err)));
    } finally {
      setLoading(false);
    }
  }, [filters, metric, limit]);

  useEffect(() => {
    fetchData();
  }, [fetchData]);

  return { data, loading, error, refetch: fetchData };
}

// Hook for Usage Time Series data
export function useUsageTimeSeries(
  filters: ControlPlaneFilters,
  metric: 'cost' | 'tokens' | 'turns' | 'spans' | 'duration' = 'cost',
  interval: 'hour' | 'day' | 'week' = 'day',
  splitBy?: 'provider' | 'model' | 'workspace'
) {
  const [data, setData] = useState<UsageTimeSeriesResponse | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<Error | null>(null);

  const fetchData = useCallback(async () => {
    setLoading(true);
    setError(null);

    try {
      const params = {
        metric,
        interval,
        split_by: splitBy,
        provider: filters.provider,
        model: filters.model,
        workspace: filters.workspace,
        start_date: filters.start_date,
        end_date: filters.end_date,
      };

      const queryString = buildQueryString(params);
      const url = `${API_BASE}/api/controlplane/usage-timeseries?${queryString}`;

      const response = await fetch(url);
      if (!response.ok) {
        throw new Error(`HTTP ${response.status}: ${response.statusText}`);
      }

      const result: UsageTimeSeriesResponse = await response.json();
      setData(result);
    } catch (err) {
      setError(err instanceof Error ? err : new Error(String(err)));
    } finally {
      setLoading(false);
    }
  }, [filters, metric, interval, splitBy]);

  useEffect(() => {
    fetchData();
  }, [fetchData]);

  return { data, loading, error, refetch: fetchData };
}

// Hook for Token Distribution data
export function useTokenDistribution(filters: ControlPlaneFilters) {
  const [data, setData] = useState<TokenDistributionResponse | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<Error | null>(null);

  const fetchData = useCallback(async () => {
    setLoading(true);
    setError(null);

    try {
      const params = {
        provider: filters.provider,
        model: filters.model,
        workspace: filters.workspace,
        start_date: filters.start_date,
        end_date: filters.end_date,
      };

      const queryString = buildQueryString(params);
      const url = `${API_BASE}/api/controlplane/token-distribution?${queryString}`;

      const response = await fetch(url);
      if (!response.ok) {
        throw new Error(`HTTP ${response.status}: ${response.statusText}`);
      }

      const result: TokenDistributionResponse = await response.json();
      setData(result);
    } catch (err) {
      setError(err instanceof Error ? err : new Error(String(err)));
    } finally {
      setLoading(false);
    }
  }, [filters]);

  useEffect(() => {
    fetchData();
  }, [fetchData]);

  return { data, loading, error, refetch: fetchData };
}
