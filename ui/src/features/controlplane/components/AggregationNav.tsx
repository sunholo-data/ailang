/**
 * AggregationNav - Right sidebar navigation for filtering by dimensions
 * Shows aggregations, breakdowns, and metrics summary
 */
import React, { useState, useMemo } from 'react';
import { BreakdownItem } from '../hooks';
import type { AggregationStats } from './types';
import type { ControlPlaneFilters } from '../types';
import { CliCommandHint } from './CliCommandHint';
import styles from '../ControlPlane.module.css';

interface FormattedBreakdownItem extends BreakdownItem {
  costFormatted: string;
  percentageFormatted: string;
  tokensFormatted: string;
}

export interface BreakdownData {
  byProvider: FormattedBreakdownItem[];
  bySourceType: FormattedBreakdownItem[];
  byModel: FormattedBreakdownItem[];
  byWorkspace: FormattedBreakdownItem[];
  totalCost: string;
}

export interface AggregationNavProps {
  selectedLevel: string;
  onSelectLevel: (level: string) => void;
  stats?: AggregationStats | null;
  breakdowns?: BreakdownData | null;
  loading?: boolean;
  filters?: ControlPlaneFilters;
}

// Icons for different breakdown categories
const sourceIcons: Record<string, string> = {
  eval: '⚗',
  coordinator: '◈',
  direct_api: '↗',
  local: '⌂',
  other: '○',
};

const providerIcons: Record<string, string> = {
  claude: '◉',
  anthropic: '◉',
  gemini: '◎',
  openai: '○',
  'gcp.vertex.agent': '◎',
};

// Format large numbers (tokens)
const formatTokens = (count: number): string => {
  if (!count || count === 0) return '0';
  if (count >= 1000000) return `${(count / 1000000).toFixed(1)}M`;
  if (count >= 1000) return `${(count / 1000).toFixed(1)}K`;
  return String(count);
};

export const AggregationNav: React.FC<AggregationNavProps> = ({
  selectedLevel,
  onSelectLevel,
  stats,
  breakdowns,
  loading,
  filters,
}) => {
  const [expanded, setExpanded] = useState<Set<string>>(new Set(['global', 'source-type']));

  // Calculate aggregate metrics from breakdowns
  const metrics = useMemo(() => {
    if (!breakdowns) return null;

    // Sum up tokens from provider breakdowns (they cover all data)
    const totals = breakdowns.byProvider.reduce(
      (acc, item) => {
        // Access the original BreakdownItem properties
        const original = item as FormattedBreakdownItem & { tokens_in?: number; tokens_out?: number; cost_usd?: number };
        return {
          tokensIn: acc.tokensIn + (original.tokens_in || 0),
          tokensOut: acc.tokensOut + (original.tokens_out || 0),
          cost: acc.cost + (original.cost_usd || 0),
          spans: acc.spans + (item.span_count || 0),
        };
      },
      { tokensIn: 0, tokensOut: 0, cost: 0, spans: 0 }
    );

    return {
      tokensIn: formatTokens(totals.tokensIn),
      tokensOut: formatTokens(totals.tokensOut),
      totalTokens: formatTokens(totals.tokensIn + totals.tokensOut),
      totalCost: breakdowns.totalCost,
      totalSpans: totals.spans,
    };
  }, [breakdowns]);

  const toggleExpand = (id: string) => {
    setExpanded(prev => {
      const next = new Set(prev);
      if (next.has(id)) next.delete(id);
      else next.add(id);
      return next;
    });
  };

  const NavItem: React.FC<{
    id: string;
    label: string;
    icon: string;
    depth: number;
    count?: number;
    cost?: string;
    percentage?: string;
    children?: React.ReactNode;
  }> = ({ id, label, icon, depth, count, cost, percentage, children }) => {
    const isExpanded = expanded.has(id);
    const isSelected = selectedLevel === id;
    const hasChildren = React.Children.count(children) > 0;

    return (
      <div className={styles.navGroup}>
        <button
          className={`${styles.navItem} ${isSelected ? styles.navItemSelected : ''}`}
          style={{ paddingLeft: `${12 + depth * 16}px` }}
          onClick={() => {
            onSelectLevel(id);
            if (hasChildren) toggleExpand(id);
          }}
        >
          {hasChildren && (
            <span className={`${styles.navChevron} ${isExpanded ? styles.navChevronOpen : ''}`}>
              ▸
            </span>
          )}
          <span className={styles.navIcon}>{icon}</span>
          <span className={styles.navLabel}>{label}</span>
          {count !== undefined && (
            <span className={styles.navCount}>{count.toLocaleString()}</span>
          )}
          {percentage && (
            <span className={styles.navPct}>{percentage}</span>
          )}
          {cost && (
            <span className={styles.navCost}>{cost}</span>
          )}
        </button>
        {hasChildren && isExpanded && (
          <div className={styles.navChildren}>{children}</div>
        )}
      </div>
    );
  };

  return (
    <nav className={styles.aggregationNav}>
      <div className={styles.navHeader}>
        <span className={styles.navTitle}>AGGREGATIONS</span>
      </div>
      <div className={styles.navTree}>
        <NavItem
          id="global"
          label="Global"
          icon="◎"
          depth={0}
          count={loading ? undefined : stats?.totalTasks}
          cost={loading ? undefined : breakdowns?.totalCost}
        >
          {/* Source Type breakdown */}
          <NavItem
            id="source-type"
            label="By Source"
            icon="▤"
            depth={1}
          >
            {breakdowns?.bySourceType.map(item => (
              <NavItem
                key={item.id}
                id={`source-${item.id}`}
                label={item.label}
                icon={sourceIcons[item.id] || '○'}
                depth={2}
                count={item.span_count}
                percentage={item.percentageFormatted}
                cost={item.costFormatted}
              />
            ))}
          </NavItem>

          {/* Provider breakdown */}
          <NavItem
            id="provider"
            label="By Provider"
            icon="◈"
            depth={1}
          >
            {breakdowns?.byProvider.map(item => (
              <NavItem
                key={item.id}
                id={`provider-${item.id}`}
                label={item.label}
                icon={providerIcons[item.id] || '○'}
                depth={2}
                count={item.span_count}
                percentage={item.percentageFormatted}
                cost={item.costFormatted}
              />
            ))}
          </NavItem>

          {/* Workspace breakdown - higher visibility for multi-project filtering */}
          {breakdowns?.byWorkspace && breakdowns.byWorkspace.length > 0 && (
            <NavItem
              id="workspace"
              label="By Workspace"
              icon="⬡"
              depth={1}
            >
              {breakdowns.byWorkspace.slice(0, 10).map(item => (
                <NavItem
                  key={item.id}
                  id={`workspace-${item.id}`}
                  label={item.label || item.id}
                  icon="·"
                  depth={2}
                  count={item.span_count}
                  percentage={item.percentageFormatted}
                  cost={item.costFormatted}
                />
              ))}
            </NavItem>
          )}

          {/* Model breakdown */}
          <NavItem
            id="model"
            label="By Model"
            icon="◎"
            depth={1}
          >
            {breakdowns?.byModel.slice(0, 10).map(item => (
              <NavItem
                key={item.id}
                id={`model-${item.id}`}
                label={item.label}
                icon="·"
                depth={2}
                count={item.span_count}
                percentage={item.percentageFormatted}
                cost={item.costFormatted}
              />
            ))}
          </NavItem>
        </NavItem>
      </div>

      {/* Metrics Summary Section */}
      <div className={styles.navMetrics}>
        <div className={styles.navMetricsHeader}>
          <span className={styles.navMetricsTitle}>METRICS</span>
        </div>
        <div className={styles.navMetricsGrid}>
          <div className={styles.metricCard}>
            <span className={styles.metricValue}>
              {loading ? '...' : metrics?.totalCost || '$0.00'}
            </span>
            <span className={styles.metricLabel}>Total Cost</span>
          </div>
          <div className={styles.metricCard}>
            <span className={styles.metricValue}>
              {loading ? '...' : metrics?.totalTokens || '0'}
            </span>
            <span className={styles.metricLabel}>Total Tokens</span>
          </div>
          <div className={styles.metricCard}>
            <span className={styles.metricValue}>
              {loading ? '...' : metrics?.tokensIn || '0'}
            </span>
            <span className={styles.metricLabel}>Tokens In</span>
          </div>
          <div className={styles.metricCard}>
            <span className={styles.metricValue}>
              {loading ? '...' : metrics?.tokensOut || '0'}
            </span>
            <span className={styles.metricLabel}>Tokens Out</span>
          </div>
        </div>
      </div>

      <div className={styles.navFooter}>
        <div className={styles.navStat}>
          <span className={styles.navStatLabel}>Active Agents</span>
          <span className={styles.navStatValue}>
            {loading ? '...' : stats?.activeAgents ?? 0}
          </span>
        </div>
        <div className={styles.navStat}>
          <span className={styles.navStatLabel}>Pending Approvals</span>
          <span className={`${styles.navStatValue} ${stats?.pendingApprovals ? styles.navStatWarning : ''}`}>
            {loading ? '...' : stats?.pendingApprovals ?? 0}
          </span>
        </div>
      </div>

      {/* CLI command hint */}
      <CliCommandHint
        commandType="stats"
        filters={filters}
        compact
      />
    </nav>
  );
};

export default AggregationNav;
