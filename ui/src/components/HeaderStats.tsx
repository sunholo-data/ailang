/**
 * HeaderStats - Inline stats display for header
 * Includes budget status indicator (AILANG dogfooding: contracts + effect budgets)
 */
import React, { useState } from 'react';
import { useControlPlaneStats } from '../features/controlplane/hooks/useControlPlaneStats';
import { useBudgetStatus } from '../hooks/useBudgetStatus';
import './HeaderStats.css';

export const HeaderStats: React.FC = () => {
  const { stats, loading } = useControlPlaneStats({ refreshInterval: 60000 });
  const { budget, loading: budgetLoading } = useBudgetStatus(120000);
  const [showProviders, setShowProviders] = useState(false);

  // Calculate budget percentage and warning state
  const budgetPercent = budget?.usage?.usagePercent ?? 0;
  const budgetLevel = budget?.status?.warningLevel ?? 'ok';
  const budgetClass = budgetLevel === 'ok' ? '' :
                      budgetLevel === 'warning' ? 'header-stat-warning' :
                      'header-stat-critical';

  // Get provider data
  const providers = budget?.byProvider ? Object.entries(budget.byProvider) : [];

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

      {/* Budget Status - AILANG dogfooding with per-provider breakdown */}
      <div
        className={`header-stat-item header-stat-budget ${budgetClass}`}
        onMouseEnter={() => setShowProviders(true)}
        onMouseLeave={() => setShowProviders(false)}
      >
        <span className="header-stat-icon">⚡</span>
        <div className="header-budget-bar">
          <div
            className="header-budget-fill"
            style={{ width: `${Math.min(budgetPercent, 100)}%` }}
          />
        </div>
        <span className="header-stat-value">
          {budgetLoading ? '...' : `${budgetPercent.toFixed(0)}%`}
        </span>
        <span className="header-stat-label">Budget</span>
        {budget?.usingAilang && (
          <span className="header-ailang-badge" title="Validated by AILANG contracts">AIL</span>
        )}

        {/* Per-Provider Dropdown */}
        {showProviders && providers.length > 0 && (
          <div className="header-provider-dropdown">
            <div className="header-provider-title">Per-Provider Usage</div>
            {providers.map(([provider, usage]) => (
              <div key={provider} className={`header-provider-row ${usage.warningLevel !== 'ok' ? 'header-provider-warn' : ''}`}>
                <span className="header-provider-name">
                  {provider}
                  {usage.hardLimit && <span className="header-provider-hard">H</span>}
                </span>
                <div className="header-provider-bar-container">
                  <div className="header-provider-bar">
                    <div
                      className={`header-provider-fill ${usage.warningLevel}`}
                      style={{ width: `${Math.min(usage.usagePercent, 100)}%` }}
                    />
                  </div>
                </div>
                <span className="header-provider-value">
                  ${usage.spend.toFixed(2)} / ${usage.budget.toFixed(0)}
                </span>
              </div>
            ))}
          </div>
        )}
      </div>
    </div>
  );
};

export default HeaderStats;
