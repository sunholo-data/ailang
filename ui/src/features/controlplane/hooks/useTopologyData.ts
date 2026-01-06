/**
 * Hook for fetching agent topology data from Control Plane API
 */
import { useState, useEffect, useCallback } from 'react';

export interface TopologyAgent {
  id: string;
  label: string;
  status: 'idle' | 'busy' | 'blocked' | 'error';
  trustScore: number;
  taskCount: number;
  cost: number;
}

export interface TopologyEdge {
  source: string;
  target: string;
  messageCount: number;
  lastActivity?: string;
}

export interface TopologySink {
  id: string;
  label?: string;
  pendingCount?: number;
}

export interface TopologyData {
  agents: TopologyAgent[];
  edges: TopologyEdge[];
  sinks: TopologySink[];
}

interface UseTopologyDataOptions {
  refreshInterval?: number; // ms, 0 to disable
}

export function useTopologyData(options: UseTopologyDataOptions = {}) {
  const { refreshInterval = 5000 } = options;
  const [data, setData] = useState<TopologyData | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  const fetchData = useCallback(async () => {
    try {
      const response = await fetch('/api/controlplane/topology');
      if (!response.ok) {
        throw new Error(`HTTP error: ${response.status}`);
      }
      const result = await response.json();
      setData(result);
      setError(null);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to fetch topology data');
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

  // Transform topology data to include active state for edges
  const transformedData = data
    ? {
        ...data,
        edges: data.edges.map((edge) => ({
          ...edge,
          active: data.agents.some(
            (a) => a.id === edge.source && a.status === 'busy'
          ),
        })),
      }
    : null;

  return { data: transformedData, loading, error, refetch: fetchData };
}
