/**
 * BudgetStatus - Displays budget status with AILANG indicator
 * Demonstrates AILANG dogfooding with contracts and effect budgets
 */
import React from 'react';
import { useBudgetStatus } from '../../hooks/useBudgetStatus';
import './BudgetStatus.css';

export const BudgetStatus: React.FC = () => {
  const { budget, loading, error } = useBudgetStatus();

  if (loading) {
    return <div className="budget-status loading">Loading budget...</div>;
  }

  if (error) {
    return <div className="budget-status error">Failed to load budget</div>;
  }

  if (!budget) {
    return null;
  }

  const { status, usage, burnRate, usingAilang } = budget;
  const warningClass = status.warningLevel === 'ok' ? 'ok'
    : status.warningLevel === 'warning' ? 'warning'
    : status.warningLevel === 'critical' ? 'critical'
    : 'exceeded';

  // Format burn rate display
  const formatBurnRate = () => {
    if (!burnRate || burnRate.costPerHour <= 0) {
      return 'No recent activity';
    }
    return `$${burnRate.costPerHour.toFixed(2)}/hr`;
  };

  // Format exhaustion forecast
  const formatForecast = () => {
    if (!burnRate || burnRate.costPerHour <= 0) {
      return null;
    }
    if (burnRate.hoursUntilExhaustion < 0) {
      return 'Budget safe';
    }
    if (burnRate.hoursUntilExhaustion === 0) {
      return 'Exhausted';
    }
    if (burnRate.hoursUntilExhaustion < 4) {
      return `${burnRate.hoursUntilExhaustion}h left`;
    }
    return `~${burnRate.hoursUntilExhaustion}h left`;
  };

  const forecastClass = burnRate?.hoursUntilExhaustion >= 0 && burnRate.hoursUntilExhaustion < 4
    ? 'forecast-critical'
    : burnRate?.hoursUntilExhaustion >= 0 && burnRate.hoursUntilExhaustion < 12
    ? 'forecast-warning'
    : '';

  return (
    <div className={`budget-status ${warningClass}`}>
      <div className="budget-header">
        <span className="budget-title">Budget Status</span>
        {usingAilang && <span className="ailang-badge">AILANG</span>}
      </div>

      <div className="budget-bar">
        <div
          className="budget-fill"
          style={{ width: `${Math.min(usage.usagePercent, 100)}%` }}
        />
      </div>

      <div className="budget-details">
        <span className="budget-spent">${usage.dailySpend.toFixed(2)}</span>
        <span className="budget-separator">/</span>
        <span className="budget-limit">${budget.config.dailyBudget.toFixed(2)}</span>
        <span className="budget-label">daily</span>
      </div>

      <div className="burn-rate-info">
        <span className="burn-rate-label">Burn:</span>
        <span className="burn-rate-value">{formatBurnRate()}</span>
        {formatForecast() && (
          <span className={`burn-rate-forecast ${forecastClass}`}>
            {formatForecast()}
          </span>
        )}
      </div>

      {status.warningLevel !== 'ok' && (
        <div className="budget-warning">
          {status.message}
        </div>
      )}
    </div>
  );
};

export default BudgetStatus;
