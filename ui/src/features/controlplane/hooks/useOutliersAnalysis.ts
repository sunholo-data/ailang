/**
 * Hook for fetching outlier analysis data from Control Plane API
 * Detects statistically anomalous spans within a task
 */
import { useState, useEffect, useCallback } from 'react';

// Span data (subset of full Span type)
export interface OutlierSpan {
  id: string;
  name: string;
  duration_ms: number;
  tokens_in: number;
  tokens_out: number;
  cost_usd: number;
  model?: string;
  provider?: string;
  status: string;
}

// Individual outlier with z-score
export interface SpanOutlier {
  span: OutlierSpan;
  metric: string; // "cost_usd", "duration_ms", "tokens"
  value: number;
  mean: number;
  std_dev: number;
  z_score: number;
  percent_of_total: number;
}

// Statistical summary per metric
export interface TaskMetricStats {
  metric: string;
  count: number;
  sum: number;
  mean: number;
  std_dev: number;
  min: number;
  max: number;
}

// Cumulative progression point
export interface CumulativePoint {
  span_index: number;
  span_name: string;
  timestamp: string;
  value: number;
  cumulative: number;
  delta_percent: number;
}

// Rate of change data
export interface RateAnalysis {
  cumulative_cost: CumulativePoint[];
  cumulative_tokens: CumulativePoint[];
  cumulative_duration: CumulativePoint[];
}

// Full response from API
export interface OutliersResponse {
  task_id: string;
  task_title: string;
  span_count: number;
  threshold: number;
  stats: TaskMetricStats[];
  outliers: SpanOutlier[];
  rate_of_change?: RateAnalysis;
  cli_command: string;
  analyzed_at: string;
}

interface UseOutliersAnalysisOptions {
  threshold?: number;
  metric?: string; // "cost", "duration", "tokens", or "" for all
  showRate?: boolean;
  limit?: number;
}

export function useOutliersAnalysis(
  taskId: string | null,
  options: UseOutliersAnalysisOptions = {}
) {
  const { threshold = 2.0, metric = '', showRate = true, limit = 10 } = options;
  const [data, setData] = useState<OutliersResponse | null>(null);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const fetchData = useCallback(async () => {
    if (!taskId) {
      setData(null);
      setLoading(false);
      return;
    }

    setLoading(true);
    try {
      const params = new URLSearchParams({
        task_id: taskId,
        threshold: threshold.toString(),
        limit: limit.toString(),
      });
      if (metric) params.set('metric', metric);
      if (showRate) params.set('rate', 'true');

      const response = await fetch(`/api/controlplane/outliers?${params}`);
      if (!response.ok) {
        if (response.status === 404) {
          throw new Error('Task not found in observatory. It may have been deleted or not yet recorded.');
        }
        throw new Error(`HTTP error: ${response.status}`);
      }
      const result = await response.json();
      setData(result);
      setError(null);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to fetch outliers data');
      setData(null);
    } finally {
      setLoading(false);
    }
  }, [taskId, threshold, metric, showRate, limit]);

  useEffect(() => {
    fetchData();
  }, [fetchData]);

  return {
    data,
    loading,
    error,
    refetch: fetchData,
  };
}
