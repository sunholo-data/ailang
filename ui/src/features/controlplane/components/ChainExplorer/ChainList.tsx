/**
 * ChainList - Browsable list of all execution chains with filtering.
 * Shows chain cards with source badges, agent flow pills, metrics, and status.
 */
import React, { useState, useMemo } from 'react';
import type { ChainSummary, ChainListFilters } from './types';
import { useChainList } from '../../hooks/useChainList';
import {
  formatCostOpt,
  formatTokensOpt,
  formatDurationMsOpt,
  formatRelativeTime,
  truncateId,
} from '../../../../utils/formatters';
import styles from './ChainExplorer.module.css';

// ============================================================================
// Source icons by type
// ============================================================================

const SOURCE_ICONS: Record<string, string> = {
  github_issue: '\u{1F4CC}', // pushpin (GitHub)
  message: '\u2709',         // envelope
  eval_suite: '\u2697',      // alembic
  manual: '\u{1F527}',       // wrench
};

const STATUS_LABELS: Record<string, { label: string; className: string }> = {
  active: { label: 'Active', className: 'statusActive' },
  pending_approval: { label: 'Pending', className: 'statusPending' },
  completed: { label: 'Done', className: 'statusCompleted' },
  failed: { label: 'Failed', className: 'statusFailed' },
};

// ============================================================================
// Sub-components
// ============================================================================

interface AgentFlowProps {
  flow: string;
}

const AgentFlow: React.FC<AgentFlowProps> = ({ flow }) => {
  if (!flow) return null;
  const agents = flow.split(' -> ').filter(Boolean);
  // Deduplicate consecutive duplicates (eval chains repeat same agent)
  const deduped: { name: string; count: number }[] = [];
  for (const agent of agents) {
    const last = deduped[deduped.length - 1];
    if (last && last.name === agent) {
      last.count++;
    } else {
      deduped.push({ name: agent, count: 1 });
    }
  }

  return (
    <div className={styles.agentFlow}>
      {deduped.map((entry, i) => (
        <React.Fragment key={i}>
          {i > 0 && <span className={styles.agentArrow}>&rarr;</span>}
          <span className={styles.agentPill}>
            {entry.name}
            {entry.count > 1 && (
              <span className={styles.agentCount}>&times;{entry.count}</span>
            )}
          </span>
        </React.Fragment>
      ))}
    </div>
  );
};

interface ProgressDotsProps {
  completed: number;
  total: number;
}

const ProgressDots: React.FC<ProgressDotsProps> = ({ completed, total }) => {
  if (total <= 0) return null;
  // Cap displayed dots at 10 to avoid overflow
  const displayTotal = Math.min(total, 10);
  const displayCompleted = Math.min(completed, displayTotal);

  return (
    <span className={styles.progressDots}>
      {Array.from({ length: displayTotal }, (_, i) => (
        <span
          key={i}
          className={i < displayCompleted ? styles.dotFilled : styles.dotEmpty}
        />
      ))}
      {total > 10 && <span className={styles.dotOverflow}>+{total - 10}</span>}
    </span>
  );
};

// ============================================================================
// Filter Bar
// ============================================================================

interface FilterBarProps {
  filters: ChainListFilters;
  onChange: (filters: ChainListFilters) => void;
  agents: string[];
}

const FilterBar: React.FC<FilterBarProps> = ({ filters, onChange, agents }) => {
  return (
    <div className={styles.filterBar}>
      <select
        className={styles.filterSelect}
        value={filters.status || ''}
        onChange={e => onChange({ ...filters, status: e.target.value || undefined })}
      >
        <option value="">All status</option>
        <option value="active">Active</option>
        <option value="pending_approval">Pending</option>
        <option value="completed">Completed</option>
        <option value="failed">Failed</option>
      </select>

      <select
        className={styles.filterSelect}
        value={filters.source_type || ''}
        onChange={e => onChange({ ...filters, source_type: e.target.value || undefined })}
      >
        <option value="">All sources</option>
        <option value="github_issue">GitHub Issue</option>
        <option value="message">Message</option>
        <option value="eval_suite">Eval Suite</option>
        <option value="manual">Manual</option>
      </select>

      {agents.length > 0 && (
        <select
          className={styles.filterSelect}
          value={filters.agent_id || ''}
          onChange={e => onChange({ ...filters, agent_id: e.target.value || undefined })}
        >
          <option value="">All agents</option>
          {agents.map(a => (
            <option key={a} value={a}>{a}</option>
          ))}
        </select>
      )}
    </div>
  );
};

// ============================================================================
// Chain Card
// ============================================================================

interface ChainCardProps {
  chain: ChainSummary;
  isSelected: boolean;
  onClick: () => void;
}

const ChainCard: React.FC<ChainCardProps> = ({ chain, isSelected, onClick }) => {
  const statusInfo = STATUS_LABELS[chain.status] || STATUS_LABELS.completed;
  const sourceIcon = SOURCE_ICONS[chain.source_type] || '\u{1F517}'; // link
  const title = chain.source_ref || truncateId(chain.id, 12);

  // Compute duration from created_at to completed_at or now
  const durationMs = chain.completed_at
    ? new Date(chain.completed_at).getTime() - new Date(chain.created_at).getTime()
    : Date.now() - new Date(chain.created_at).getTime();

  return (
    <div
      className={`${styles.chainCard} ${isSelected ? styles.chainCardSelected : ''}`}
      onClick={onClick}
      role="button"
      tabIndex={0}
      onKeyDown={e => e.key === 'Enter' && onClick()}
    >
      <div className={styles.chainCardHeader}>
        <span className={styles.sourceIcon}>{sourceIcon}</span>
        <span className={styles.chainTitle}>{title}</span>
        <span className={`${styles.statusBadge} ${styles[statusInfo.className]}`}>
          {statusInfo.label}
        </span>
      </div>

      <AgentFlow flow={chain.agent_flow} />

      <div className={styles.chainMetrics}>
        <span>{formatCostOpt(chain.total_cost, '--')}</span>
        <span>{formatTokensOpt(chain.total_tokens, '--')} tok</span>
        <span>{chain.total_turns} turns</span>
        <span>{formatDurationMsOpt(durationMs, '--')}</span>
      </div>

      <div className={styles.chainFooter}>
        <span className={styles.stageCount}>
          {chain.stages_completed}/{chain.stage_count} stages
        </span>
        <ProgressDots completed={chain.stages_completed} total={chain.stage_count} />
        <span className={styles.relativeTime}>
          {formatRelativeTime(chain.created_at)}
        </span>
      </div>
    </div>
  );
};

// ============================================================================
// Main ChainList
// ============================================================================

export interface ChainListProps {
  selectedChainId?: string | null;
  onSelectChain: (chainId: string) => void;
}

export const ChainList: React.FC<ChainListProps> = ({
  selectedChainId,
  onSelectChain,
}) => {
  const [filters, setFilters] = useState<ChainListFilters>({});
  const { chains, loading, error, hasMore, loadMore } = useChainList({
    filters,
    limit: 50,
    refreshInterval: 30000, // poll every 30s for active chains
  });

  // Extract unique agent names for filter dropdown
  const uniqueAgents = useMemo(() => {
    const agentSet = new Set<string>();
    for (const chain of chains) {
      if (chain.agent_flow) {
        chain.agent_flow.split(' -> ').forEach(a => agentSet.add(a));
      }
    }
    return Array.from(agentSet).sort((a, b) => a.localeCompare(b));
  }, [chains]);

  return (
    <div className={styles.chainListContainer}>
      <div className={styles.chainListHeader}>
        <span className={styles.chainListTitle}>Chains</span>
        <span className={styles.chainListCount}>{chains.length}</span>
      </div>

      <FilterBar filters={filters} onChange={setFilters} agents={uniqueAgents} />

      <div className={styles.chainListScroll}>
        {error && <div className={styles.errorMessage}>{error}</div>}
        {!loading && chains.length === 0 && !error && (
          <div className={styles.emptyState}>No chains found</div>
        )}

        {chains.map(chain => (
          <ChainCard
            key={chain.id}
            chain={chain}
            isSelected={selectedChainId === chain.id}
            onClick={() => onSelectChain(chain.id)}
          />
        ))}

        {loading && <div className={styles.loadingIndicator}>Loading...</div>}

        {hasMore && !loading && (
          <button className={styles.loadMoreButton} onClick={loadMore}>
            Load more
          </button>
        )}
      </div>
    </div>
  );
};

export default ChainList;
