/**
 * Hook for fetching Claude Code session data from Observatory API
 * Provides session list and tool details from hook-captured telemetry
 */
import { useState, useEffect, useCallback } from 'react';

// Session from /api/observatory/sessions
export interface Session {
  session_id: string;
  workspace: string;
  started_at: string;
  ended_at?: string;
  tool_count: number;
}

// Tool call from /api/observatory/sessions/{id}/tools
export interface SessionTool {
  tool_use_id: string;
  tool_name: string;
  start_time: string;
  end_time?: string;
  success?: boolean;
  metadata?: Record<string, unknown>; // file_path, pattern, command, etc.
}

// Tool summary from /api/observatory/sessions/{id}/tools/summary
export interface ToolSummaryItem {
  tool_name: string;
  count: number;
  success_count: number;
  details: string[]; // file paths, patterns, commands
}

export interface SessionToolsSummary {
  session_id: string;
  tools: ToolSummaryItem[];
}

interface UseSessionsDataOptions {
  limit?: number;
  refreshInterval?: number; // ms, 0 to disable
}

export function useSessionsData(options: UseSessionsDataOptions = {}) {
  const { limit = 10, refreshInterval = 30000 } = options;
  const [sessions, setSessions] = useState<Session[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  const fetchData = useCallback(async () => {
    try {
      const response = await fetch(`/api/observatory/sessions?limit=${limit}`);
      if (!response.ok) {
        throw new Error(`HTTP error: ${response.status}`);
      }
      const result = await response.json();
      setSessions(result.sessions || []);
      setError(null);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to fetch sessions');
    } finally {
      setLoading(false);
    }
  }, [limit]);

  useEffect(() => {
    fetchData();

    if (refreshInterval > 0) {
      const interval = setInterval(fetchData, refreshInterval);
      return () => clearInterval(interval);
    }
  }, [fetchData, refreshInterval]);

  return { sessions, loading, error, refetch: fetchData };
}

interface UseSessionToolsOptions {
  summary?: boolean;
  workspace?: string;  // Filter by workspace path
}

export function useSessionTools(sessionId: string | null, options: UseSessionToolsOptions = {}) {
  const { summary = true, workspace } = options;
  const [tools, setTools] = useState<ToolSummaryItem[] | SessionTool[]>([]);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    if (!sessionId) {
      setTools([]);
      return;
    }

    const fetchTools = async () => {
      setLoading(true);
      try {
        let endpoint = summary
          ? `/api/observatory/sessions/${sessionId}/tools/summary`
          : `/api/observatory/sessions/${sessionId}/tools`;
        // Add workspace filter if provided
        if (workspace) {
          endpoint += `?workspace=${encodeURIComponent(workspace)}`;
        }
        const response = await fetch(endpoint);
        if (!response.ok) {
          throw new Error(`HTTP error: ${response.status}`);
        }
        const result = await response.json();
        setTools(summary ? (result.tools || []) : (result.tools || []));
        setError(null);
      } catch (err) {
        setError(err instanceof Error ? err.message : 'Failed to fetch tools');
      } finally {
        setLoading(false);
      }
    };

    fetchTools();
  }, [sessionId, summary, workspace]);

  return { tools, loading, error };
}

// Helper to format relative time
export function formatRelativeTime(dateStr: string): string {
  const date = new Date(dateStr);
  const now = new Date();
  const diffMs = now.getTime() - date.getTime();
  const diffMins = Math.floor(diffMs / 60000);
  const diffHours = Math.floor(diffMs / 3600000);
  const diffDays = Math.floor(diffMs / 86400000);

  if (diffMins < 1) return 'just now';
  if (diffMins < 60) return `${diffMins}m ago`;
  if (diffHours < 24) return `${diffHours}h ago`;
  if (diffDays < 7) return `${diffDays}d ago`;
  return date.toLocaleDateString();
}

// Helper to extract workspace name from path
export function formatWorkspace(path: string): string {
  if (!path) return 'Unknown';
  const parts = path.split('/');
  return parts[parts.length - 1] || parts[parts.length - 2] || path;
}
