/**
 * AggregationNav - Left sidebar navigation for filtering by dimensions
 */
import React, { useState } from 'react';
import { BreakdownItem } from '../hooks';
import type { AggregationStats } from './types';
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

export const AggregationNav: React.FC<AggregationNavProps> = ({
  selectedLevel,
  onSelectLevel,
  stats,
  breakdowns,
  loading,
}) => {
  const [expanded, setExpanded] = useState<Set<string>>(new Set(['global', 'source-type']));

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

          {/* Workspace breakdown */}
          {breakdowns?.byWorkspace && breakdowns.byWorkspace.length > 0 && (
            <NavItem
              id="workspace"
              label="By Workspace"
              icon="⬡"
              depth={1}
            >
              {breakdowns.byWorkspace.map(item => (
                <NavItem
                  key={item.id}
                  id={`workspace-${item.id}`}
                  label={item.label || item.id}
                  icon="·"
                  depth={2}
                  count={item.task_count}
                  cost={item.costFormatted}
                />
              ))}
            </NavItem>
          )}
        </NavItem>
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
    </nav>
  );
};

export default AggregationNav;
