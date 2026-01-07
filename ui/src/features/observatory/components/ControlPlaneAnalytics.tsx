import React from 'react';
import { useControlPlaneStats, useStatsBreakdown } from '../../../hooks/useControlPlane';
import { ProviderBreakdown, SummaryCards } from './ProviderBreakdown';
import styles from './ControlPlaneAnalytics.module.css';

export function ControlPlaneAnalytics() {
  const { stats, loading: statsLoading, error: statsError, refresh } = useControlPlaneStats();
  const { breakdown, loading: breakdownLoading, error: breakdownError } = useStatsBreakdown();

  const loading = statsLoading || breakdownLoading;
  const error = statsError || breakdownError;

  if (loading && !stats && !breakdown) {
    return (
      <div className={styles.container}>
        <div className={styles.loading}>Loading analytics...</div>
      </div>
    );
  }

  if (error) {
    return (
      <div className={styles.container}>
        <div className={styles.error}>Error: {error}</div>
      </div>
    );
  }

  const observatory = stats?.observatory;
  const coordinator = stats?.coordinator;

  return (
    <div className={styles.container}>
      {/* Header */}
      <div className={styles.header}>
        <h2>Analytics</h2>
        <button onClick={refresh} className={styles.refreshButton}>
          Refresh
        </button>
      </div>

      {/* Summary Cards */}
      {observatory && (
        <SummaryCards
          totalSpans={observatory.total_spans}
          totalTasks={observatory.total_tasks}
          totalCost={observatory.total_cost_usd}
          totalTokens={observatory.total_tokens_in + observatory.total_tokens_out}
          successRate={observatory.success_rate}
        />
      )}

      {/* Coordinator Status */}
      {coordinator && (
        <div className={styles.coordinatorStatus}>
          <h3>Coordinator</h3>
          <div className={styles.coordinatorGrid}>
            <div className={styles.statusItem}>
              <span className={`${styles.statusDot} ${coordinator.running ? styles.statusRunning : styles.statusStopped}`} />
              <span>{coordinator.running ? 'Running' : 'Stopped'}</span>
            </div>
            <div className={styles.statItem}>
              <span className={styles.statValue}>{coordinator.pending_tasks}</span>
              <span className={styles.statLabel}>Pending</span>
            </div>
            <div className={styles.statItem}>
              <span className={styles.statValue}>{coordinator.running_tasks}</span>
              <span className={styles.statLabel}>Running</span>
            </div>
            <div className={styles.statItem}>
              <span className={styles.statValue}>{coordinator.completed_tasks}</span>
              <span className={styles.statLabel}>Completed</span>
            </div>
            <div className={styles.statItem}>
              <span className={styles.statValue}>{coordinator.failed_tasks}</span>
              <span className={styles.statLabel}>Failed</span>
            </div>
            <div className={styles.statItem}>
              <span className={styles.statValue}>{coordinator.pending_approvals}</span>
              <span className={styles.statLabel}>Approvals</span>
            </div>
          </div>
        </div>
      )}

      {/* Breakdown Sections */}
      {breakdown && (
        <div className={styles.breakdownGrid}>
          <ProviderBreakdown
            items={breakdown.by_provider}
            title="By Provider"
            colorScheme="provider"
          />
          <ProviderBreakdown
            items={breakdown.by_model}
            title="By Model"
            colorScheme="model"
          />
          <ProviderBreakdown
            items={breakdown.by_workspace}
            title="By Workspace"
            showTaskCount
            colorScheme="workspace"
          />
          <ProviderBreakdown
            items={breakdown.by_source_type}
            title="By Source Type"
          />
        </div>
      )}

      {/* Database Status */}
      {stats?.sources && (
        <div className={styles.sourcesStatus}>
          <h4>Data Sources</h4>
          <div className={styles.sourcesList}>
            <div className={styles.sourceItem}>
              <span className={`${styles.sourceStatus} ${stats.sources.observatory_ok ? styles.sourceOk : styles.sourceError}`}>
                {stats.sources.observatory_ok ? '✓' : '✗'}
              </span>
              <span className={styles.sourceLabel}>Observatory</span>
              <code className={styles.sourcePath}>{stats.sources.observatory_db}</code>
            </div>
            <div className={styles.sourceItem}>
              <span className={`${styles.sourceStatus} ${stats.sources.coordinator_ok ? styles.sourceOk : styles.sourceError}`}>
                {stats.sources.coordinator_ok ? '✓' : '✗'}
              </span>
              <span className={styles.sourceLabel}>Coordinator</span>
              <code className={styles.sourcePath}>{stats.sources.coordinator_db}</code>
            </div>
          </div>
        </div>
      )}
    </div>
  );
}

export default ControlPlaneAnalytics;
