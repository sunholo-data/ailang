/**
 * AggregationNav - Right sidebar navigation for filtering by dimensions
 * Shows aggregations, breakdowns, and metrics summary
 */
import React, { useState, useMemo } from 'react';
import { BreakdownItem } from '../hooks';
import type { AggregationStats } from './types';
import type { ControlPlaneFilters } from '../types';
import { CliCommandHint } from './CliCommandHint';
import type { BurnRateInfo } from '../../../hooks/useBudgetStatus';
import { useWorkspaceAccess } from '../../../hooks/useWorkspaceAccess';
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

// Highlighted items from event selection (for visual feedback)
export interface HighlightedAggItems {
  workspace?: string;
  provider?: string;
  model?: string;
  source_type?: string;
}

// Multi-filter type: maps dimension to selected value
// e.g., { workspace: 'ailang', source: 'eval', provider: 'claude' }
export type SelectedFilters = Record<string, string>;

export interface AggregationNavProps {
  selectedFilters: SelectedFilters;
  onFilterToggle: (dimension: string, value: string) => void;
  onClearFilters: () => void;
  stats?: AggregationStats | null;
  breakdowns?: BreakdownData | null;
  loading?: boolean;
  filters?: ControlPlaneFilters;
  highlightedItems?: HighlightedAggItems | null;
  burnRate?: BurnRateInfo | null;
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

// Format burn rate for display
const formatBurnRate = (burnRate: BurnRateInfo | null | undefined): string => {
  if (!burnRate || burnRate.costPerHour <= 0) return '--';
  return `$${burnRate.costPerHour.toFixed(2)}/hr`;
};

// Format hours until exhaustion for display
const formatExhaustion = (burnRate: BurnRateInfo | null | undefined): string => {
  if (!burnRate || burnRate.costPerHour <= 0) return '--';
  if (burnRate.hoursUntilExhaustion < 0) return 'Safe';
  if (burnRate.hoursUntilExhaustion === 0) return 'Now!';
  return `${burnRate.hoursUntilExhaustion}h`;
};

// Get exhaustion status class
const getExhaustionClass = (burnRate: BurnRateInfo | null | undefined): string => {
  if (!burnRate || burnRate.costPerHour <= 0) return '';
  if (burnRate.hoursUntilExhaustion >= 0 && burnRate.hoursUntilExhaustion < 4) return styles.metricCritical;
  if (burnRate.hoursUntilExhaustion >= 0 && burnRate.hoursUntilExhaustion < 12) return styles.metricWarning;
  return '';
};

export const AggregationNav: React.FC<AggregationNavProps> = ({
  selectedFilters,
  onFilterToggle,
  onClearFilters,
  stats,
  breakdowns,
  loading,
  filters,
  highlightedItems,
  burnRate,
}) => {
  const [expanded, setExpanded] = useState<Set<string>>(new Set(['global']));

  // Get accessible workspaces with roles
  const { getRole, workspaces } = useWorkspaceAccess();

  // Calculate aggregate metrics from breakdowns
  const metrics = useMemo(() => {
    if (!breakdowns) return null;

    // Sum up tokens from provider breakdowns (includes cache tokens)
    const providerTotals = breakdowns.byProvider.reduce(
      (acc, item) => {
        const original = item as FormattedBreakdownItem & {
          tokens_in?: number;
          tokens_out?: number;
          cost_usd?: number;
          cache_read_tokens?: number;
          cache_creation_tokens?: number;
          cache_savings_usd?: number;
        };
        return {
          tokensIn: acc.tokensIn + (original.tokens_in || 0),
          tokensOut: acc.tokensOut + (original.tokens_out || 0),
          cost: acc.cost + (original.cost_usd || 0),
          cacheRead: acc.cacheRead + (original.cache_read_tokens || 0),
          cacheCreate: acc.cacheCreate + (original.cache_creation_tokens || 0),
          cacheSavings: acc.cacheSavings + (original.cache_savings_usd || 0),
        };
      },
      { tokensIn: 0, tokensOut: 0, cost: 0, cacheRead: 0, cacheCreate: 0, cacheSavings: 0 }
    );

    // Use workspace breakdown for total spans (most complete view)
    const totalSpans = breakdowns.byWorkspace?.reduce(
      (sum, item) => sum + (item.span_count || 0),
      0
    ) || 0;

    return {
      tokensIn: formatTokens(providerTotals.tokensIn),
      tokensOut: formatTokens(providerTotals.tokensOut),
      totalTokens: formatTokens(providerTotals.tokensIn + providerTotals.tokensOut),
      totalCost: breakdowns.totalCost,
      totalSpans,
      cacheRead: formatTokens(providerTotals.cacheRead),
      cacheCreate: formatTokens(providerTotals.cacheCreate),
      cacheSavings: providerTotals.cacheSavings > 0 ? `$${providerTotals.cacheSavings.toFixed(2)}` : '$0.00',
      hasCacheData: providerTotals.cacheRead > 0 || providerTotals.cacheCreate > 0,
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
    dimension?: string;  // Filter dimension: 'source', 'provider', 'model', 'workspace'
    value?: string;      // Filter value within dimension
    count?: number;
    cost?: string;
    percentage?: string;
    children?: React.ReactNode;
    isHighlighted?: boolean;
    badge?: { icon: string; type: 'approver' | 'viewer' | 'public'; title: string };
  }> = ({ id, label, icon, depth, dimension, value, count, cost, percentage, children, isHighlighted, badge }) => {
    const isExpanded = expanded.has(id);
    // For leaf items (with dimension/value), check if this dimension is selected
    // For "global", selected when no filters are active
    const isSelected = dimension && value
      ? selectedFilters[dimension] === value
      : id === 'global' && Object.keys(selectedFilters).length === 0;
    const hasChildren = React.Children.count(children) > 0;

    const handleClick = () => {
      if (dimension && value) {
        // Leaf item: toggle this filter
        onFilterToggle(dimension, value);
      } else if (id === 'global') {
        // Global: clear all filters
        onClearFilters();
      }
      // Category headers just expand/collapse
      if (hasChildren) toggleExpand(id);
    };

    return (
      <div className={styles.navGroup}>
        <button
          className={`${styles.navItem} ${isSelected ? styles.navItemSelected : ''} ${isHighlighted ? styles.navItemHighlighted : ''}`}
          style={{ paddingLeft: `${12 + depth * 16}px` }}
          onClick={handleClick}
        >
          {hasChildren && (
            <span className={`${styles.navChevron} ${isExpanded ? styles.navChevronOpen : ''}`}>
              ▸
            </span>
          )}
          <span className={styles.navIcon}>{icon}</span>
          <span className={styles.navLabel} title={label}>{label}</span>
          {badge && (
            <span
              className={`${styles.navBadge} ${styles[`navBadge${badge.type.charAt(0).toUpperCase() + badge.type.slice(1)}`]}`}
              title={badge.title}
            >
              {badge.icon}
            </span>
          )}
          {/* Show count inline only for global (no percentage), otherwise in tooltip */}
          {count !== undefined && !percentage && (
            <span className={styles.navCount} title="Total spans">
              {count.toLocaleString()}
            </span>
          )}
          {percentage && (
            <span
              className={styles.navPct}
              title={count !== undefined ? `${count.toLocaleString()} spans (${percentage} of total)` : `${percentage} of total`}
            >
              {percentage}
            </span>
          )}
          {cost && (
            <span className={styles.navCost} title="Estimated cost (USD)">
              {cost}
            </span>
          )}
          {/* Show remove indicator for selected leaf items */}
          {isSelected && dimension && (
            <span className={styles.navRemove} title="Click to remove filter">×</span>
          )}
        </button>
        {hasChildren && isExpanded && (
          <div className={styles.navChildren}>{children}</div>
        )}
      </div>
    );
  };

  // Count active filters for display
  const activeFilterCount = Object.keys(selectedFilters).length;

  return (
    <nav className={styles.aggregationNav}>
      {/* Compact stats row above aggregations */}
      <div className={styles.statsBar}>
        <div className={styles.statsBarItem} title="Agents currently running tasks">
          <span className={styles.statsBarValue}>
            {loading ? '—' : stats?.activeAgents ?? 0}
          </span>
          <span className={styles.statsBarLabel}>Active Agents</span>
        </div>
        <div className={styles.statsBarDivider} />
        <div className={`${styles.statsBarItem} ${stats?.pendingApprovals ? styles.statsBarWarning : ''}`} title="Tasks awaiting human approval">
          <span className={styles.statsBarValue}>
            {loading ? '—' : stats?.pendingApprovals ?? 0}
          </span>
          <span className={styles.statsBarLabel}>Pending</span>
        </div>
      </div>

      <div className={styles.navHeader}>
        <span className={styles.navTitle}>AGGREGATIONS</span>
        {activeFilterCount > 0 && (
          <button
            className={styles.clearFiltersBtn}
            onClick={onClearFilters}
            title="Clear all filters"
          >
            Clear ({activeFilterCount})
          </button>
        )}
      </div>
      <div className={styles.navTree}>
        <NavItem
          id="global"
          label="Global"
          icon="◎"
          depth={0}
          count={loading ? undefined : metrics?.totalSpans}
          cost={loading ? undefined : breakdowns?.totalCost}
        >
          {/* Workspace breakdown - PRIMARY filter for multi-project users */}
          {breakdowns?.byWorkspace && breakdowns.byWorkspace.length > 0 && (
            <NavItem
              id="workspace"
              label="By Workspace"
              icon="⬡"
              depth={1}
            >
              {breakdowns.byWorkspace.slice(0, 10).map(item => {
                const role = getRole(item.id);
                // Check if workspace is public (no explicit role grant)
                const wsInfo = workspaces.find(w => w.id === item.id);
                const isPublic = !role && wsInfo?.is_public;

                // Determine badge based on role or public status
                const badge = role
                  ? {
                      icon: role === 'Approver' ? '✓' : '👁',
                      type: (role === 'Approver' ? 'approver' : 'viewer') as 'approver' | 'viewer',
                      title: role,
                    }
                  : isPublic
                  ? { icon: '🌐', type: 'public' as const, title: 'Public workspace' }
                  : undefined;

                return (
                  <NavItem
                    key={item.id}
                    id={`workspace-${item.id}`}
                    label={item.label || item.id}
                    icon="·"
                    depth={2}
                    dimension="workspace"
                    value={item.id}
                    count={item.span_count}
                    percentage={item.percentageFormatted}
                    cost={item.costFormatted}
                    isHighlighted={highlightedItems?.workspace === item.id}
                    badge={badge}
                  />
                );
              })}
            </NavItem>
          )}

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
                dimension="source"
                value={item.id}
                count={item.span_count}
                percentage={item.percentageFormatted}
                cost={item.costFormatted}
                isHighlighted={highlightedItems?.source_type === item.id}
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
                dimension="provider"
                value={item.id}
                count={item.span_count}
                percentage={item.percentageFormatted}
                cost={item.costFormatted}
                isHighlighted={highlightedItems?.provider === item.id}
              />
            ))}
          </NavItem>

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
                dimension="model"
                value={item.id}
                count={item.span_count}
                percentage={item.percentageFormatted}
                cost={item.costFormatted}
                isHighlighted={highlightedItems?.model === item.id}
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
          <div className={styles.metricCard} title="Sum of all AI API costs (USD) based on model-specific pricing from models.yml. Calculated from input/output tokens × rate per 1K tokens.">
            <span className={styles.metricValue}>
              {loading ? '...' : metrics?.totalCost || '$0.00'}
            </span>
            <span className={styles.metricLabel}>Total Cost</span>
          </div>
          <div className={styles.metricCard} title="Total tokens processed (input + output). Each API call logs token counts from provider responses.">
            <span className={styles.metricValue}>
              {loading ? '...' : metrics?.totalTokens || '0'}
            </span>
            <span className={styles.metricLabel}>Total Tokens</span>
          </div>
          <div className={styles.metricCard} title="Input/prompt tokens sent to AI models. Includes system prompts, user messages, and context.">
            <span className={styles.metricValue}>
              {loading ? '...' : metrics?.tokensIn || '0'}
            </span>
            <span className={styles.metricLabel}>Tokens In</span>
          </div>
          <div className={styles.metricCard} title="Output/completion tokens generated by AI models. The AI's responses and tool calls.">
            <span className={styles.metricValue}>
              {loading ? '...' : metrics?.tokensOut || '0'}
            </span>
            <span className={styles.metricLabel}>Tokens Out</span>
          </div>
          <div className={styles.metricCard} title={burnRate && burnRate.costPerHour > 0
            ? `Cost per hour = Total cost in window ÷ ${burnRate.windowHours}h. Based on ${burnRate.windowHours}-hour rolling window of AI spend.`
            : 'No AI activity in the monitoring window. Burn rate appears when costs are recorded.'}>
            <span className={styles.metricValue}>
              {loading ? '...' : formatBurnRate(burnRate)}
            </span>
            <span className={styles.metricLabel}>Burn Rate</span>
          </div>
          <div className={`${styles.metricCard} ${getExhaustionClass(burnRate)}`} title={burnRate && burnRate.costPerHour > 0
            ? `Hours until budget exhausted = (Daily budget − Today's spend) ÷ Burn rate. Yellow <12h, Red <4h.`
            : 'Budget ETA calculated when burn rate is active. Shows "Safe" if no spending detected.'}>
            <span className={styles.metricValue}>
              {loading ? '...' : formatExhaustion(burnRate)}
            </span>
            <span className={styles.metricLabel}>Budget ETA</span>
          </div>
          {/* Cache metrics - only show when there's cache data */}
          {metrics?.hasCacheData && (
            <>
              <div className={styles.metricCard} title="Tokens served from prompt cache (90% cost discount). Cached context re-used across API calls.">
                <span className={styles.metricValue}>
                  {loading ? '...' : metrics?.cacheRead || '0'}
                </span>
                <span className={styles.metricLabel}>📦 Cache Read</span>
              </div>
              <div className={`${styles.metricCard} ${styles.metricSuccess}`} title="Estimated savings from prompt caching. Cache reads cost 90% less than regular input tokens.">
                <span className={styles.metricValue}>
                  {loading ? '...' : metrics?.cacheSavings || '$0.00'}
                </span>
                <span className={styles.metricLabel}>💰 Cache Savings</span>
              </div>
            </>
          )}
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
