/**
 * ChainDetail - Detail view for a single execution chain.
 * Shows header with metrics, stage pipeline, and stage detail panel.
 */
import React, { useState, useEffect, useMemo } from 'react';
import type { ChainData, ChainStageData } from '../ExecHierarchy/types';
import { StagePipeline } from './StagePipeline';
import { StageDetail } from './StageDetail';
import { JourneySummary } from './JourneySummary';
import { useStageSpans } from '../../hooks';
import {
  formatCost,
  formatTokens,
  formatDurationMs,
  formatRelativeTime,
  truncateId,
} from '../../../../utils/formatters';
import styles from './ChainExplorer.module.css';

// ============================================================================
// Source configuration
// ============================================================================

const SOURCE_CONFIG: Record<string, { icon: string; label: string }> = {
  github_issue: { icon: '\u{1F4CC}', label: 'GitHub Issue' },
  message: { icon: '\u2709', label: 'Message' },
  eval_suite: { icon: '\u2697', label: 'Eval Suite' },
  manual: { icon: '\u{1F527}', label: 'Manual' },
  session: { icon: '\u{1F4AC}', label: 'Session' },
  cli: { icon: '\u{1F4BB}', label: 'CLI' },
};

const STATUS_CONFIG: Record<string, { label: string; className: string }> = {
  active: { label: 'Active', className: 'statusActive' },
  pending_approval: { label: 'Pending Approval', className: 'statusPending' },
  completed: { label: 'Completed', className: 'statusCompleted' },
  failed: { label: 'Failed', className: 'statusFailed' },
};

// ============================================================================
// Main ChainDetail
// ============================================================================

export interface ChainDetailProps {
  chain: ChainData;
  onBrowseAll?: () => void;
  hiddenSpanTypes?: Set<string>;
  theme?: 'dark' | 'light';
}

export const ChainDetail: React.FC<ChainDetailProps> = ({
  chain,
  onBrowseAll,
  hiddenSpanTypes,
  theme,
}) => {
  const [selectedStageId, setSelectedStageId] = useState<string | null>(null);

  // Auto-select the first/current stage
  useEffect(() => {
    if (chain.stages && chain.stages.length > 0) {
      // Find the current stage or default to first
      const currentStage = chain.stages.find(
        s => s.stage_number === chain.current_stage
      );
      setSelectedStageId(currentStage?.id || chain.stages[0].id);
    }
  }, [chain.id, chain.stages, chain.current_stage]);

  const selectedStage = useMemo<ChainStageData | null>(
    () => chain.stages?.find(s => s.id === selectedStageId) || null,
    [chain.stages, selectedStageId]
  );

  const isVirtual = chain.id.startsWith('virtual-');

  // Lazy-load spans for the selected stage (L2 tiered loading)
  // Skip for virtual chains (their spans are already inline from synthesis)
  const isVirtualStage = selectedStageId?.startsWith('virtual-') ?? false;
  const { spans: lazySpans, loading: spansLoading } = useStageSpans({
    chainId: isVirtual ? null : chain.id,
    stageId: isVirtualStage ? null : selectedStageId,
  });

  // Use inline spans if available (virtual chains), otherwise lazy-loaded spans
  const stageSpansForDetail = selectedStage?.spans?.length
    ? selectedStage.spans
    : lazySpans;

  // Compute chain duration
  const durationMs = chain.completed_at
    ? new Date(chain.completed_at).getTime() - new Date(chain.created_at).getTime()
    : Date.now() - new Date(chain.created_at).getTime();

  const source = SOURCE_CONFIG[chain.source_type] || SOURCE_CONFIG.session;
  const status = STATUS_CONFIG[chain.status] || STATUS_CONFIG.completed;
  const title = chain.source_ref || truncateId(chain.id, 12);

  // GitHub link
  const githubUrl = chain.github_repo && chain.github_issue_number
    ? `https://github.com/${chain.github_repo}/issues/${chain.github_issue_number}`
    : null;

  return (
    <div className={styles.chainDetailContainer}>
      {/* Header */}
      <div className={styles.detailHeader}>
        <div className={styles.detailHeaderTop}>
          <div className={styles.detailTitle}>
            <span className={styles.detailSourceIcon}>{source.icon}</span>
            {githubUrl ? (
              <a
                href={githubUrl}
                target="_blank"
                rel="noopener noreferrer"
                className={styles.detailTitleLink}
              >
                {title}
              </a>
            ) : (
              <span className={styles.detailTitleText}>{title}</span>
            )}
            <span className={`${styles.statusBadge} ${styles[status.className]}`}>
              {status.label}
            </span>
            {isVirtual && (
              <span className={styles.virtualBadge}>synthesized</span>
            )}
          </div>
          {onBrowseAll && (
            <button className={styles.browseAllButton} onClick={onBrowseAll}>
              Browse All Chains
            </button>
          )}
        </div>

        <div className={styles.detailMetrics}>
          <div className={styles.detailMetricBox}>
            <span className={styles.detailMetricLabel}>Cost</span>
            <span className={styles.detailMetricValue}>{formatCost(chain.total_cost)}</span>
          </div>
          <div className={styles.detailMetricBox}>
            <span className={styles.detailMetricLabel}>Tokens</span>
            <span className={styles.detailMetricValue}>{formatTokens(chain.total_tokens)}</span>
          </div>
          <div className={styles.detailMetricBox}>
            <span className={styles.detailMetricLabel}>Turns</span>
            <span className={styles.detailMetricValue}>{chain.total_turns}</span>
          </div>
          <div className={styles.detailMetricBox}>
            <span className={styles.detailMetricLabel}>Duration</span>
            <span className={styles.detailMetricValue}>{formatDurationMs(durationMs)}</span>
          </div>
          <div className={styles.detailMetricBox}>
            <span className={styles.detailMetricLabel}>Stages</span>
            <span className={styles.detailMetricValue}>
              {chain.stages_completed}/{chain.stages?.length || 0}
            </span>
          </div>
          <div className={styles.detailMetricBox}>
            <span className={styles.detailMetricLabel}>Created</span>
            <span className={styles.detailMetricValue}>
              {formatRelativeTime(chain.created_at)}
            </span>
          </div>
        </div>
      </div>

      {/* Journey Narrative */}
      {!isVirtual && chain.stages && chain.stages.length > 1 ? (
        <JourneySummary chainId={chain.id} />
      ) : chain.stages && chain.stages.length > 0 ? (
        <div className={styles.journeyContainer}>
          <div className={styles.journeySummaryLine}>
            {chain.stages[0].agent_id || source.label} session
            {chain.total_turns ? ` \u2014 ${chain.total_turns} turns` : ''}
            {chain.stages[0].tool_calls ? `, ${chain.stages[0].tool_calls} tool calls` : ''}
            {chain.total_cost ? `, ${formatCost(chain.total_cost)}` : ''}
          </div>
        </div>
      ) : null}

      {/* Stage Pipeline */}
      {chain.stages && chain.stages.length > 0 && (
        <StagePipeline
          stages={chain.stages}
          selectedStageId={selectedStageId}
          onSelectStage={setSelectedStageId}
        />
      )}

      {/* Stage Detail */}
      {selectedStage ? (
        <StageDetail
          stage={selectedStage}
          spans={stageSpansForDetail}
          hiddenSpanTypes={hiddenSpanTypes}
          theme={theme}
        />
      ) : (
        <div className={styles.noStageSelected}>
          {chain.stages && chain.stages.length > 0
            ? 'Select a stage above to see details'
            : 'No stages available'}
        </div>
      )}
    </div>
  );
};

export default ChainDetail;
