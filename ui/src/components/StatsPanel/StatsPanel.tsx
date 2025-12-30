import React, { useState, useEffect } from 'react';
import styles from './StatsPanel.module.css';

interface ThreadStatistics {
  total: number;
  by_status: Record<string, number>;
  by_workspace: Record<string, number>;
}

interface CoordinatorSummary {
  total_tasks: number;
  pending_tasks: number;
  running_tasks: number;
  completed_tasks: number;
  failed_tasks: number;
  by_provider?: Record<string, number>;
  by_workspace?: Record<string, number>;
  total_cost: number;
  total_tokens: number;
}

interface StatisticsResponse {
  threads: ThreadStatistics;
  coordinator?: CoordinatorSummary;
}

interface StatsPanelProps {
  refreshTrigger?: number;
}

export function StatsPanel({ refreshTrigger }: StatsPanelProps) {
  const [stats, setStats] = useState<StatisticsResponse | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    fetchStats();
  }, [refreshTrigger]);

  const fetchStats = async () => {
    setLoading(true);
    setError(null);
    try {
      const response = await fetch('/api/statistics');
      if (!response.ok) {
        throw new Error('Failed to fetch statistics');
      }
      const data = await response.json();
      setStats(data);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Unknown error');
    } finally {
      setLoading(false);
    }
  };

  if (loading && !stats) {
    return <div className={styles.loading}>Loading statistics...</div>;
  }

  if (error) {
    return <div className={styles.error}>Error: {error}</div>;
  }

  if (!stats) {
    return null;
  }

  // Get folder name from full path
  const getFolderName = (path: string): string => {
    if (path === '(no workspace)') return path;
    const parts = path.split('/');
    return parts[parts.length - 1] || path;
  };

  return (
    <div className={styles.container}>
      <div className={styles.header}>
        <h3>Statistics</h3>
        <button onClick={fetchStats} className={styles.refreshButton} disabled={loading}>
          {loading ? 'Refreshing...' : 'Refresh'}
        </button>
      </div>

      {/* Thread Statistics */}
      <div className={styles.section}>
        <h4>Threads</h4>
        <div className={styles.statRow}>
          <span className={styles.label}>Total:</span>
          <span className={styles.value}>{stats.threads.total}</span>
        </div>

        {/* By Status */}
        <div className={styles.subSection}>
          <h5>By Status</h5>
          {Object.entries(stats.threads.by_status).map(([status, count]) => (
            <div key={status} className={styles.statRow}>
              <span className={styles.statusBadge} data-status={status}>{status}</span>
              <span className={styles.value}>{count}</span>
            </div>
          ))}
        </div>

        {/* By Workspace */}
        <div className={styles.subSection}>
          <h5>By Workspace</h5>
          {Object.entries(stats.threads.by_workspace)
            .sort(([, a], [, b]) => b - a)
            .slice(0, 10)
            .map(([workspace, count]) => (
            <div key={workspace} className={styles.statRow}>
              <span className={styles.workspaceName} title={workspace}>
                {getFolderName(workspace)}
              </span>
              <span className={styles.value}>{count}</span>
            </div>
          ))}
        </div>
      </div>

      {/* Coordinator Statistics */}
      {stats.coordinator && (
        <div className={styles.section}>
          <h4>Coordinator Tasks</h4>
          <div className={styles.statRow}>
            <span className={styles.label}>Total:</span>
            <span className={styles.value}>{stats.coordinator.total_tasks}</span>
          </div>
          <div className={styles.statRow}>
            <span className={styles.label}>Pending:</span>
            <span className={styles.value}>{stats.coordinator.pending_tasks}</span>
          </div>
          <div className={styles.statRow}>
            <span className={styles.label}>Running:</span>
            <span className={styles.value}>{stats.coordinator.running_tasks}</span>
          </div>
          <div className={styles.statRow}>
            <span className={styles.label}>Completed:</span>
            <span className={styles.value}>{stats.coordinator.completed_tasks}</span>
          </div>
          <div className={styles.statRow}>
            <span className={styles.label}>Failed:</span>
            <span className={styles.value}>{stats.coordinator.failed_tasks}</span>
          </div>
          {stats.coordinator.total_cost > 0 && (
            <div className={styles.statRow}>
              <span className={styles.label}>Total Cost:</span>
              <span className={styles.value}>${stats.coordinator.total_cost.toFixed(4)}</span>
            </div>
          )}
          {stats.coordinator.total_tokens > 0 && (
            <div className={styles.statRow}>
              <span className={styles.label}>Total Tokens:</span>
              <span className={styles.value}>{stats.coordinator.total_tokens.toLocaleString()}</span>
            </div>
          )}

          {/* By Provider */}
          {stats.coordinator.by_provider && Object.keys(stats.coordinator.by_provider).length > 0 && (
            <div className={styles.subSection}>
              <h5>By Provider</h5>
              {Object.entries(stats.coordinator.by_provider).map(([provider, count]) => (
                <div key={provider} className={styles.statRow}>
                  <span className={styles.providerBadge} data-provider={provider}>{provider}</span>
                  <span className={styles.value}>{count}</span>
                </div>
              ))}
            </div>
          )}
        </div>
      )}
    </div>
  );
}
