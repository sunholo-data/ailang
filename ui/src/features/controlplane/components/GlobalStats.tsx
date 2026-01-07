/**
 * GlobalStats - Header statistics display for Control Plane
 */
import React from 'react';
import styles from '../ControlPlane.module.css';

export interface GlobalStatsData {
  // Observatory stats (canonical telemetry)
  totalSpans: number;
  totalTasks: number;
  totalTokens: string;
  totalCost: string;
  successRate: string;
  // Coordinator runtime stats
  activeAgents: number;
  pendingApprovals: number;
  coordinatorTasks: number;
  coordinatorCost: string;
  // Data source status
  observatoryOK: boolean;
  coordinatorOK: boolean;
}

export interface GlobalStatsProps {
  stats?: GlobalStatsData | null;
  loading?: boolean;
  isFiltered?: boolean;
  filterDescription?: string;
  onClearFilter?: () => void;
}

export const GlobalStats: React.FC<GlobalStatsProps> = ({
  stats,
  loading,
  isFiltered,
  filterDescription,
  onClearFilter,
}) => {
  return (
    <div className={styles.globalStats}>
      {isFiltered && (
        <div className={styles.filterBadge}>
          <span className={styles.filterIcon}>⚡</span>
          <span className={styles.filterText}>{filterDescription}</span>
          <button className={styles.clearFilterBtn} onClick={onClearFilter} title="Clear filter">×</button>
        </div>
      )}
      <span className={styles.statsScope} title={isFiltered ? "Filtered telemetry" : "Observatory telemetry (all sources)"}>
        {isFiltered ? 'Filtered' : 'Observatory'}
      </span>
      <div className={styles.statCard} title="Total spans from all sources (evals, coordinator, local)">
        <span className={styles.statIcon}>▤</span>
        <div className={styles.statContent}>
          <span className={styles.statValue}>{loading ? '...' : stats?.totalSpans ?? '—'}</span>
          <span className={styles.statLabel}>Spans</span>
        </div>
      </div>
      <div className={styles.statCard} title="Total cost across all API calls">
        <span className={styles.statIcon}>$</span>
        <div className={styles.statContent}>
          <span className={styles.statValue}>{loading ? '...' : stats?.totalCost ?? '—'}</span>
          <span className={styles.statLabel}>Total Cost</span>
        </div>
      </div>
      <div className={styles.statCard} title="Total tokens (input + output)">
        <span className={styles.statIcon}>◎</span>
        <div className={styles.statContent}>
          <span className={styles.statValue}>{loading ? '...' : stats?.totalTokens ?? '—'}</span>
          <span className={styles.statLabel}>Tokens</span>
        </div>
      </div>
      <div className={styles.statCard} title="Success rate across all operations">
        <span className={`${styles.statIcon} ${styles.statSuccess}`}>✓</span>
        <div className={styles.statContent}>
          <span className={`${styles.statValue} ${styles.statSuccess}`}>
            {loading ? '...' : stats?.successRate ?? '—'}
          </span>
          <span className={styles.statLabel}>Success</span>
        </div>
      </div>
      <span className={styles.statsScope} title="Coordinator runtime (delegated tasks only)">
        Coordinator
      </span>
      <div className={styles.statCard} title="Agents currently running tasks">
        <span className={styles.statIcon}>◈</span>
        <div className={styles.statContent}>
          <span className={styles.statValue}>{loading ? '...' : stats?.activeAgents ?? '—'}</span>
          <span className={styles.statLabel}>Active</span>
        </div>
      </div>
      <div className={styles.statCard} title="Tasks awaiting human approval">
        <span className={`${styles.statIcon} ${styles.statWarning}`}>⏳</span>
        <div className={styles.statContent}>
          <span className={`${styles.statValue} ${stats?.pendingApprovals ? styles.statWarning : ''}`}>
            {loading ? '...' : stats?.pendingApprovals ?? '—'}
          </span>
          <span className={styles.statLabel}>Pending</span>
        </div>
      </div>
    </div>
  );
};

export default GlobalStats;
