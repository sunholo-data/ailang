import React from 'react';
import { BreakdownItem, formatCost, formatTokens, formatPercentage } from '../../../hooks/useControlPlane';
import styles from './ProviderBreakdown.module.css';

interface ProviderBreakdownProps {
  items: BreakdownItem[];
  title: string;
  showTokens?: boolean;
  showTaskCount?: boolean;
  colorScheme?: 'provider' | 'model' | 'workspace' | 'default';
}

// Provider/Model colors
const colorMap: Record<string, string> = {
  // Providers
  claude: '#c4a35a',
  gemini: '#4285f4',
  ollama: '#7c3aed',
  openai: '#10a37f',
  // Models
  'claude-opus-4': '#c4a35a',
  'claude-sonnet-4-5': '#d4b86a',
  'claude-haiku-4-5': '#e4c87a',
  'gemini-2-5-pro': '#4285f4',
  'gemini-2-5-flash': '#6ca5f9',
  'gpt5': '#10a37f',
  'gpt5-mini': '#30c39f',
  // Workspaces
  'ailang-dev': '#22c55e',
  'ailang-staging': '#f59e0b',
  'ailang-prod': '#ef4444',
  // Source types
  exec: '#8b5cf6',
  eval: '#06b6d4',
  local: '#10b981',
  direct_api: '#f97316',
  other: '#6b7280',
};

function getColor(id: string): string {
  return colorMap[id] || colorMap[id.toLowerCase()] || '#6b7280';
}

export function ProviderBreakdown({ items, title, showTokens = true, showTaskCount = false, colorScheme = 'default' }: ProviderBreakdownProps) {
  if (!items || items.length === 0) {
    return (
      <div className={styles.container}>
        <h3 className={styles.title}>{title}</h3>
        <div className={styles.empty}>No data available</div>
      </div>
    );
  }

  // Find max percentage for bar scaling
  const maxPercentage = Math.max(...items.map(i => i.percentage));

  return (
    <div className={styles.container}>
      <h3 className={styles.title}>{title}</h3>
      <div className={styles.list}>
        {items.map((item) => (
          <div key={item.id} className={styles.item}>
            <div className={styles.header}>
              <div className={styles.labelContainer}>
                <span
                  className={styles.colorDot}
                  style={{ backgroundColor: getColor(item.id) }}
                />
                <span className={styles.label}>{item.label}</span>
              </div>
              <span className={styles.percentage}>{formatPercentage(item.percentage)}</span>
            </div>
            <div className={styles.barContainer}>
              <div
                className={styles.bar}
                style={{
                  width: `${(item.percentage / maxPercentage) * 100}%`,
                  backgroundColor: getColor(item.id),
                }}
              />
            </div>
            <div className={styles.stats}>
              <span className={styles.stat}>
                <span className={styles.statLabel}>Spans</span>
                <span className={styles.statValue}>{item.span_count.toLocaleString()}</span>
              </span>
              {showTaskCount && item.task_count !== undefined && (
                <span className={styles.stat}>
                  <span className={styles.statLabel}>Tasks</span>
                  <span className={styles.statValue}>{item.task_count}</span>
                </span>
              )}
              {showTokens && (
                <span className={styles.stat}>
                  <span className={styles.statLabel}>Tokens</span>
                  <span className={styles.statValue}>{formatTokens(item.tokens_in + item.tokens_out)}</span>
                </span>
              )}
              <span className={styles.stat}>
                <span className={styles.statLabel}>Cost</span>
                <span className={styles.statValue}>{formatCost(item.cost_usd)}</span>
              </span>
            </div>
          </div>
        ))}
      </div>
    </div>
  );
}

// Summary card showing key metrics
interface SummaryCardsProps {
  totalSpans: number;
  totalTasks: number;
  totalCost: number;
  totalTokens: number;
  successRate: number;
}

export function SummaryCards({ totalSpans, totalTasks, totalCost, totalTokens, successRate }: SummaryCardsProps) {
  return (
    <div className={styles.summaryGrid}>
      <div className={styles.summaryCard}>
        <div className={styles.summaryValue}>{totalSpans.toLocaleString()}</div>
        <div className={styles.summaryLabel}>Total Spans</div>
      </div>
      <div className={styles.summaryCard}>
        <div className={styles.summaryValue}>{totalTasks}</div>
        <div className={styles.summaryLabel}>Tasks</div>
      </div>
      <div className={styles.summaryCard}>
        <div className={styles.summaryValue}>{formatTokens(totalTokens)}</div>
        <div className={styles.summaryLabel}>Tokens</div>
      </div>
      <div className={styles.summaryCard}>
        <div className={styles.summaryValue}>{formatCost(totalCost)}</div>
        <div className={styles.summaryLabel}>Total Cost</div>
      </div>
      <div className={styles.summaryCard}>
        <div className={styles.summaryValue}>{successRate.toFixed(1)}%</div>
        <div className={styles.summaryLabel}>Success Rate</div>
      </div>
    </div>
  );
}

export default ProviderBreakdown;
