/**
 * HeaderStats - Inline stats display for header
 */
import React from 'react';
import { useControlPlaneStats } from '../features/controlplane/hooks/useControlPlaneStats';
import './HeaderStats.css';

export const HeaderStats: React.FC = () => {
  const { stats, loading } = useControlPlaneStats({ refreshInterval: 10000 });

  return (
    <div className="header-stats-inline">
      <div className="header-stat-item">
        <span className="header-stat-icon">$</span>
        <span className="header-stat-value">{loading ? '...' : stats?.totalCost ?? '$0.00'}</span>
        <span className="header-stat-label">Cost</span>
      </div>

      <div className="header-stat-item">
        <span className="header-stat-icon">◎</span>
        <span className="header-stat-value">{loading ? '...' : stats?.totalTokens ?? '0'}</span>
        <span className="header-stat-label">Tokens</span>
      </div>

      <div className="header-stat-item">
        <span className="header-stat-icon">▤</span>
        <span className="header-stat-value">{loading ? '...' : stats?.totalSpans ?? 0}</span>
        <span className="header-stat-label">Spans</span>
      </div>
    </div>
  );
};

export default HeaderStats;
