/**
 * StagePipeline - Horizontal CSS-only pipeline showing chain stages.
 * Each stage is a card connected by arrows with optional approval gates.
 * Pure CSS layout (no ReactFlow/dagre needed - chains are always linear).
 */
import React from 'react';
import type { ChainStageData } from '../ExecHierarchy/types';
import {
  formatCostOpt,
  formatDurationMsOpt,
} from '../../../../utils/formatters';
import styles from './ChainExplorer.module.css';

// ============================================================================
// Constants
// ============================================================================

const PROVIDER_ICONS: Record<string, string> = {
  claude: '\u{1F4AC}',  // speech balloon
  gemini: '\u2728',      // sparkles
  ollama: '\u{1F4E6}',  // package
};

const STATUS_CONFIG: Record<string, { dot: string; className: string }> = {
  pending: { dot: '\u25CB', className: 'stagePending' },     // empty circle
  running: { dot: '\u25CF', className: 'stageRunning' },     // filled circle (pulsing)
  awaiting_approval: { dot: '\u25D4', className: 'stageAwaiting' }, // half-filled
  completed: { dot: '\u2713', className: 'stageCompleted' }, // check
  failed: { dot: '\u2717', className: 'stageFailed' },       // cross
};

const APPROVAL_CONFIG: Record<string, { icon: string; className: string }> = {
  pending: { icon: '\u23F3', className: 'approvalPending' },   // hourglass
  approved: { icon: '\u2705', className: 'approvalApproved' }, // green check
  rejected: { icon: '\u274C', className: 'approvalRejected' }, // red cross
};

// ============================================================================
// Stage Card
// ============================================================================

interface StageCardProps {
  stage: ChainStageData;
  isSelected: boolean;
  onClick: () => void;
}

const StageCard: React.FC<StageCardProps> = ({ stage, isSelected, onClick }) => {
  const statusCfg = STATUS_CONFIG[stage.status] || STATUS_CONFIG.pending;
  const providerIcon = PROVIDER_ICONS[stage.provider || ''] || '';
  const isFailed = stage.status === 'failed';

  return (
    <div
      className={`${styles.stageCard} ${styles[statusCfg.className]} ${isSelected ? styles.stageCardSelected : ''} ${isFailed ? styles.stageCardFailed : ''}`}
      onClick={onClick}
      role="button"
      tabIndex={0}
      onKeyDown={e => e.key === 'Enter' && onClick()}
      title={isFailed && stage.error_message ? stage.error_message : undefined}
    >
      <div className={styles.stageCardHeader}>
        <span className={styles.stageStatusDot}>{statusCfg.dot}</span>
        <span className={styles.stageAgentName}>{stage.agent_id}</span>
        {providerIcon && <span className={styles.stageProvider}>{providerIcon}</span>}
      </div>

      {stage.iteration > 1 && (
        <span className={styles.iterationBadge}>Iter {stage.iteration}</span>
      )}

      <div className={styles.stageCardMetrics}>
        {stage.cost > 0 && (
          <span className={styles.stageMetric}>{formatCostOpt(stage.cost)}</span>
        )}
        {stage.turns > 0 && (
          <span className={styles.stageMetric}>{stage.turns} turns</span>
        )}
        {stage.tool_calls > 0 && (
          <span className={styles.stageMetric}>{stage.tool_calls} tools</span>
        )}
        {stage.duration_ms > 0 && (
          <span className={styles.stageMetric}>{formatDurationMsOpt(stage.duration_ms)}</span>
        )}
      </div>

      {isFailed && stage.error_message && (
        <div className={styles.stageErrorExcerpt}>
          {stage.error_message.slice(0, 60)}
          {stage.error_message.length > 60 ? '...' : ''}
        </div>
      )}
    </div>
  );
};

// ============================================================================
// Approval Gate
// ============================================================================

interface ApprovalGateProps {
  approvalStatus?: string;
  approvalType?: string;
  handoffTo?: string;
}

const ApprovalGate: React.FC<ApprovalGateProps> = ({ approvalStatus, approvalType, handoffTo }) => {
  const cfg = APPROVAL_CONFIG[approvalStatus || ''] || APPROVAL_CONFIG.pending;
  const typeLabel = approvalType === 'merge_handoff'
    ? 'merge+handoff'
    : approvalType || 'gate';

  return (
    <div className={`${styles.approvalGate} ${styles[cfg.className]}`}>
      <span className={styles.approvalIcon}>{cfg.icon}</span>
      <span className={styles.approvalLabel}>{typeLabel}</span>
      {handoffTo && (
        <span className={styles.approvalTarget}>&rarr; {handoffTo}</span>
      )}
    </div>
  );
};

// ============================================================================
// Connector Arrow
// ============================================================================

const Connector: React.FC = () => (
  <div className={styles.pipelineConnector}>
    <div className={styles.connectorLine} />
    <div className={styles.connectorArrow} />
  </div>
);

// ============================================================================
// Main StagePipeline
// ============================================================================

export interface StagePipelineProps {
  stages: ChainStageData[];
  selectedStageId?: string | null;
  onSelectStage: (stageId: string) => void;
}

export const StagePipeline: React.FC<StagePipelineProps> = ({
  stages,
  selectedStageId,
  onSelectStage,
}) => {
  if (!stages || stages.length === 0) {
    return <div className={styles.pipelineEmpty}>No stages</div>;
  }

  // Sort by stage_number
  const sorted = [...stages].sort((a, b) => a.stage_number - b.stage_number);

  return (
    <div className={styles.pipelineContainer}>
      <div className={styles.pipelineScroll}>
        {sorted.map((stage, i) => {
          const prevStage = i > 0 ? sorted[i - 1] : null;
          const hasApproval = prevStage && prevStage.approval_status;

          return (
            <React.Fragment key={stage.id}>
              {i > 0 && (
                <>
                  {hasApproval ? (
                    <ApprovalGate
                      approvalStatus={prevStage.approval_status}
                      approvalType={prevStage.approval_type}
                      handoffTo={prevStage.handoff_to}
                    />
                  ) : (
                    <Connector />
                  )}
                </>
              )}
              <StageCard
                stage={stage}
                isSelected={selectedStageId === stage.id}
                onClick={() => onSelectStage(stage.id)}
              />
            </React.Fragment>
          );
        })}
      </div>
    </div>
  );
};

export default StagePipeline;
