import { useEffect, useState, useCallback } from 'react';

// Control Plane API Base
const API_BASE = '/api/controlplane';

// Fetch helper
async function fetchAPI<T>(endpoint: string, options?: RequestInit): Promise<T> {
  const response = await fetch(`${API_BASE}${endpoint}`, {
    headers: {
      'Content-Type': 'application/json',
    },
    ...options,
  });
  if (!response.ok) {
    throw new Error(`API error: ${response.statusText}`);
  }
  return response.json();
}

// ===== Types =====

export interface BreakdownItem {
  id: string;
  label: string;
  span_count: number;
  task_count?: number;
  tokens_in: number;
  tokens_out: number;
  cost_usd: number;
  percentage: number;
}

export interface StatsBreakdown {
  by_provider: BreakdownItem[];
  by_source_type: BreakdownItem[];
  by_model: BreakdownItem[];
  by_workspace: BreakdownItem[];
  total_cost: number;
}

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

export interface CoordinatorStats {
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

export interface ControlPlaneStats {
  observatory: ObservatoryStats;
  coordinator: CoordinatorStats;
  sources: {
    observatory_db: string;
    coordinator_db: string;
    observatory_ok: boolean;
    coordinator_ok: boolean;
  };
}

export interface HeatmapDataPoint {
  day: number;      // 0-6 (Sunday-Saturday)
  hour: number;     // 0-23
  count: number;
  avg_duration_ms: number;
  total_cost: number;
}

export interface HeatmapData {
  data: HeatmapDataPoint[];
  min_count: number;
  max_count: number;
  total_spans: number;
}

export interface TopologyNode {
  id: string;
  name: string;
  type: 'service' | 'agent' | 'workspace';
  span_count: number;
  cost_usd: number;
}

export interface TopologyEdge {
  from: string;
  to: string;
  weight: number;
}

export interface TopologyData {
  nodes: TopologyNode[];
  edges: TopologyEdge[];
}

// ===== Hooks =====

// Hook for control plane stats
export function useControlPlaneStats() {
  const [stats, setStats] = useState<ControlPlaneStats | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  const refresh = useCallback(async () => {
    setLoading(true);
    try {
      const data = await fetchAPI<ControlPlaneStats>('/stats');
      setStats(data);
      setError(null);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to fetch stats');
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    refresh();
    // Auto-refresh every 30 seconds
    const interval = setInterval(refresh, 30000);
    return () => clearInterval(interval);
  }, [refresh]);

  return { stats, loading, error, refresh };
}

// Hook for stats breakdown
export interface UseBreakdownOptions {
  by?: 'provider' | 'model' | 'workspace' | 'source_type';
}

export function useStatsBreakdown(options: UseBreakdownOptions = {}) {
  const [breakdown, setBreakdown] = useState<StatsBreakdown | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  const refresh = useCallback(async () => {
    setLoading(true);
    try {
      const params = new URLSearchParams();
      if (options.by) params.set('by', options.by);

      const query = params.toString();
      const endpoint = query ? `/stats/breakdown?${query}` : '/stats/breakdown';
      const data = await fetchAPI<StatsBreakdown>(endpoint);
      setBreakdown(data);
      setError(null);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to fetch breakdown');
    } finally {
      setLoading(false);
    }
  }, [options.by]);

  useEffect(() => {
    refresh();
  }, [refresh]);

  return { breakdown, loading, error, refresh };
}

// Hook for heatmap data
export interface UseHeatmapOptions {
  days?: number;        // Number of days to include (default: 7)
  provider?: string;    // Filter by provider
  workspace?: string;   // Filter by workspace
}

export function useHeatmap(options: UseHeatmapOptions = {}) {
  const [heatmap, setHeatmap] = useState<HeatmapData | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  const refresh = useCallback(async () => {
    setLoading(true);
    try {
      const params = new URLSearchParams();
      if (options.days) params.set('days', options.days.toString());
      if (options.provider) params.set('provider', options.provider);
      if (options.workspace) params.set('workspace', options.workspace);

      const query = params.toString();
      const endpoint = query ? `/heatmap?${query}` : '/heatmap';
      const data = await fetchAPI<HeatmapData>(endpoint);
      setHeatmap(data);
      setError(null);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to fetch heatmap');
    } finally {
      setLoading(false);
    }
  }, [options.days, options.provider, options.workspace]);

  useEffect(() => {
    refresh();
  }, [refresh]);

  return { heatmap, loading, error, refresh };
}

// Hook for topology data
export function useTopology() {
  const [topology, setTopology] = useState<TopologyData | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  const refresh = useCallback(async () => {
    setLoading(true);
    try {
      const data = await fetchAPI<TopologyData>('/topology');
      setTopology(data);
      setError(null);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to fetch topology');
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    refresh();
  }, [refresh]);

  return { topology, loading, error, refresh };
}

// Utility: Format cost for display
export function formatCost(cost: number): string {
  if (cost < 0.01) {
    return '$' + cost.toFixed(4);
  } else if (cost < 1) {
    return '$' + cost.toFixed(3);
  } else if (cost < 100) {
    return '$' + cost.toFixed(2);
  } else {
    return '$' + cost.toLocaleString(undefined, { maximumFractionDigits: 2 });
  }
}

// Utility: Format token count
export function formatTokens(tokens: number): string {
  if (tokens < 1000) {
    return tokens.toString();
  } else if (tokens < 1000000) {
    return (tokens / 1000).toFixed(1) + 'K';
  } else {
    return (tokens / 1000000).toFixed(2) + 'M';
  }
}

// Utility: Format percentage
export function formatPercentage(value: number): string {
  return value.toFixed(1) + '%';
}
