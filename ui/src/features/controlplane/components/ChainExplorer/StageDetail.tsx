/**
 * StageDetail - Tabbed detail view for a single chain stage.
 * Tabs: Summary | Chat | Tools | Files
 * Reuses ChatHistory for the Chat tab.
 */
import React, { useState, useMemo } from 'react';
import type { ChainStageData, Span, HierarchyNode } from '../ExecHierarchy/types';
import type { StageDetailTab, ToolUsageEntry, FileTouchEntry } from './types';
import { ChatHistory } from '../ExecHierarchy/ChatHistory';
import {
  formatCostOpt,
  formatTokensOpt,
  formatDurationMsOpt,
} from '../../../../utils/formatters';
import styles from './ChainExplorer.module.css';

// ============================================================================
// Span extraction helpers
// ============================================================================

/** Walk span tree and collect tool usage entries */
function extractToolUsage(spans: Span[]): ToolUsageEntry[] {
  const toolMap = new Map<string, ToolUsageEntry>();

  function walk(spanList: Span[]) {
    for (const span of spanList) {
      // Match tool spans: "claude_code.tool.Read", "claude_code.tool.Bash", etc.
      const toolMatch = span.name.match(/\.tool\.(\w+)$/);
      if (toolMatch) {
        const toolName = toolMatch[1];
        const existing = toolMap.get(toolName) || {
          toolName,
          count: 0,
          totalDurationMs: 0,
          errors: 0,
        };
        existing.count++;
        existing.totalDurationMs += span.durationMs || 0;
        if (span.status === 'error') existing.errors++;
        toolMap.set(toolName, existing);
      }
      if (span.children) walk(span.children);
    }
  }

  walk(spans);
  return Array.from(toolMap.values()).sort((a, b) => b.count - a.count);
}

/** Walk span tree and extract file paths from tool attributes */
function extractFiles(spans: Span[]): FileTouchEntry[] {
  const fileMap = new Map<string, FileTouchEntry>();

  function walk(spanList: Span[]) {
    for (const span of spanList) {
      const toolMatch = span.name.match(/\.tool\.(\w+)$/);
      if (toolMatch && span.attributes) {
        const toolName = toolMatch[1].toLowerCase();
        const attrs = span.attributes;

        // Try to extract file path from various attribute keys
        const filePath = attrs['file_path'] || attrs['path'] || attrs['file'] || '';
        if (filePath && !filePath.includes('*')) {
          const opMap: Record<string, string> = {
            read: 'read',
            edit: 'edit',
            write: 'write',
            glob: 'glob',
            grep: 'grep',
          };
          const op = opMap[toolName] || toolName;
          const existing = fileMap.get(filePath) || {
            path: filePath,
            operations: [],
            lastTouched: 0,
          };
          if (!existing.operations.includes(op)) {
            existing.operations.push(op);
          }
          const timestamp = span.startMs || 0;
          if (timestamp > existing.lastTouched) {
            existing.lastTouched = timestamp;
          }
          fileMap.set(filePath, existing);
        }
      }
      if (span.children) walk(span.children);
    }
  }

  walk(spans);
  return Array.from(fileMap.values()).sort((a, b) => b.lastTouched - a.lastTouched);
}

// ============================================================================
// Tab Components
// ============================================================================

interface SummaryTabProps {
  stage: ChainStageData;
}

const SummaryTab: React.FC<SummaryTabProps> = ({ stage }) => {
  const durationMs = stage.duration_ms || 0;

  return (
    <div className={styles.summaryTab}>
      <div className={styles.metricsGrid}>
        <div className={styles.metricCard}>
          <span className={styles.metricLabel}>Cost</span>
          <span className={styles.metricValue}>{formatCostOpt(stage.cost, '$0.00')}</span>
        </div>
        <div className={styles.metricCard}>
          <span className={styles.metricLabel}>Tokens In</span>
          <span className={styles.metricValue}>{formatTokensOpt(stage.tokens_in, '0')}</span>
        </div>
        <div className={styles.metricCard}>
          <span className={styles.metricLabel}>Tokens Out</span>
          <span className={styles.metricValue}>{formatTokensOpt(stage.tokens_out, '0')}</span>
        </div>
        <div className={styles.metricCard}>
          <span className={styles.metricLabel}>Turns</span>
          <span className={styles.metricValue}>{stage.turns || 0}</span>
        </div>
        <div className={styles.metricCard}>
          <span className={styles.metricLabel}>Tool Calls</span>
          <span className={styles.metricValue}>{stage.tool_calls || 0}</span>
        </div>
        <div className={styles.metricCard}>
          <span className={styles.metricLabel}>Duration</span>
          <span className={styles.metricValue}>{formatDurationMsOpt(durationMs, '--')}</span>
        </div>
      </div>

      {/* Stage metadata */}
      <div className={styles.metadataSection}>
        <div className={styles.metadataRow}>
          <span className={styles.metadataKey}>Agent</span>
          <span className={styles.metadataVal}>{stage.agent_id}</span>
        </div>
        {stage.provider && (
          <div className={styles.metadataRow}>
            <span className={styles.metadataKey}>Provider</span>
            <span className={styles.metadataVal}>{stage.provider}</span>
          </div>
        )}
        {stage.iteration > 1 && (
          <div className={styles.metadataRow}>
            <span className={styles.metadataKey}>Iteration</span>
            <span className={styles.metadataVal}>{stage.iteration}</span>
          </div>
        )}
        {stage.approval_status && (
          <div className={styles.metadataRow}>
            <span className={styles.metadataKey}>Approval</span>
            <span className={styles.metadataVal}>
              {stage.approval_status} ({stage.approval_type || 'merge'})
            </span>
          </div>
        )}
      </div>

      {/* Error display */}
      {stage.error_message && (
        <div className={styles.errorSection}>
          <span className={styles.errorLabel}>Error ({stage.error_count || 1})</span>
          <pre className={styles.errorContent}>{stage.error_message}</pre>
        </div>
      )}

      {/* Eval assessment */}
      {stage.eval_assessment && (
        <div className={styles.evalSection}>
          <span className={styles.evalTitle}>Eval Assessment</span>
          <div className={styles.metricsGrid}>
            <div className={styles.metricCard}>
              <span className={styles.metricLabel}>Benchmark</span>
              <span className={styles.metricValue}>{stage.eval_assessment.benchmark_id}</span>
            </div>
            <div className={styles.metricCard}>
              <span className={styles.metricLabel}>Model</span>
              <span className={styles.metricValue}>{stage.eval_assessment.model}</span>
            </div>
            <div className={styles.metricCard}>
              <span className={styles.metricLabel}>Compile</span>
              <span className={`${styles.metricValue} ${stage.eval_assessment.compile_ok ? styles.passValue : styles.failValue}`}>
                {stage.eval_assessment.compile_ok ? 'PASS' : 'FAIL'}
              </span>
            </div>
            <div className={styles.metricCard}>
              <span className={styles.metricLabel}>Runtime</span>
              <span className={`${styles.metricValue} ${stage.eval_assessment.runtime_ok ? styles.passValue : styles.failValue}`}>
                {stage.eval_assessment.runtime_ok ? 'PASS' : 'FAIL'}
              </span>
            </div>
            <div className={styles.metricCard}>
              <span className={styles.metricLabel}>Output</span>
              <span className={`${styles.metricValue} ${stage.eval_assessment.stdout_ok ? styles.passValue : styles.failValue}`}>
                {stage.eval_assessment.stdout_ok ? 'PASS' : 'FAIL'}
              </span>
            </div>
            {stage.eval_assessment.error_category && (
              <div className={styles.metricCard}>
                <span className={styles.metricLabel}>Error Category</span>
                <span className={styles.metricValue}>{stage.eval_assessment.error_category}</span>
              </div>
            )}
          </div>
        </div>
      )}

      {/* Human feedback */}
      {stage.human_feedback && (
        <div className={styles.feedbackSection}>
          <span className={styles.feedbackLabel}>Human Feedback</span>
          <pre className={styles.feedbackContent}>{stage.human_feedback}</pre>
        </div>
      )}
    </div>
  );
};

interface ToolsTabProps {
  tools: ToolUsageEntry[];
}

const ToolsTab: React.FC<ToolsTabProps> = ({ tools }) => {
  if (tools.length === 0) {
    return <div className={styles.emptyTabState}>No tool usage recorded</div>;
  }

  return (
    <div className={styles.toolsTab}>
      <table className={styles.dataTable}>
        <thead>
          <tr>
            <th>Tool</th>
            <th>Count</th>
            <th>Duration</th>
            <th>Errors</th>
          </tr>
        </thead>
        <tbody>
          {tools.map(tool => (
            <tr key={tool.toolName}>
              <td className={styles.toolName}>{tool.toolName}</td>
              <td>{tool.count}</td>
              <td>{formatDurationMsOpt(tool.totalDurationMs, '--')}</td>
              <td className={tool.errors > 0 ? styles.failValue : ''}>
                {tool.errors || '--'}
              </td>
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  );
};

interface FilesTabProps {
  files: FileTouchEntry[];
}

const OP_COLORS: Record<string, string> = {
  read: '#60a5fa',
  edit: '#fbbf24',
  write: '#4ade80',
  glob: '#a78bfa',
  grep: '#f472b6',
};

const FilesTab: React.FC<FilesTabProps> = ({ files }) => {
  if (files.length === 0) {
    return <div className={styles.emptyTabState}>No file operations recorded</div>;
  }

  return (
    <div className={styles.filesTab}>
      <table className={styles.dataTable}>
        <thead>
          <tr>
            <th>File</th>
            <th>Operations</th>
          </tr>
        </thead>
        <tbody>
          {files.map(file => (
            <tr key={file.path}>
              <td className={styles.filePath} title={file.path}>
                {file.path.split('/').slice(-2).join('/')}
              </td>
              <td>
                <div className={styles.opBadges}>
                  {file.operations.map(op => (
                    <span
                      key={op}
                      className={styles.opBadge}
                      style={{ borderColor: OP_COLORS[op] || '#6e7681' }}
                    >
                      {op}
                    </span>
                  ))}
                </div>
              </td>
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  );
};

// ============================================================================
// Main StageDetail
// ============================================================================

export interface StageDetailProps {
  stage: ChainStageData;
  spans?: Span[];
  hiddenSpanTypes?: Set<string>;
  theme?: 'dark' | 'light';
}

export const StageDetail: React.FC<StageDetailProps> = ({
  stage,
  hiddenSpanTypes,
  theme,
}) => {
  const [activeTab, setActiveTab] = useState<StageDetailTab>('summary');

  // Extract data from stage spans
  const stageSpans = stage.spans || [];
  const tools = useMemo(() => extractToolUsage(stageSpans), [stageSpans]);
  const files = useMemo(() => extractFiles(stageSpans), [stageSpans]);

  // Synthesize a HierarchyNode from stage metadata for ChatHistory context
  const syntheticNode = useMemo<HierarchyNode>(() => ({
    id: stage.id,
    type: 'exec',
    label: stage.agent_id,
    status: stage.status === 'completed' ? 'completed' : stage.status === 'failed' ? 'error' : 'busy',
    taskId: stage.task_id,
    agentId: stage.agent_id,
    durationMs: stage.duration_ms,
  }), [stage]);

  const tabs: { id: StageDetailTab; label: string; count?: number }[] = [
    { id: 'summary', label: 'Summary' },
    { id: 'chat', label: 'Chat' },
    { id: 'tools', label: 'Tools', count: tools.length },
    { id: 'files', label: 'Files', count: files.length },
  ];

  return (
    <div className={styles.stageDetailContainer}>
      <div className={styles.tabBar}>
        {tabs.map(tab => (
          <button
            key={tab.id}
            className={`${styles.tab} ${activeTab === tab.id ? styles.tabActive : ''}`}
            onClick={() => setActiveTab(tab.id)}
          >
            {tab.label}
            {tab.count !== undefined && tab.count > 0 && (
              <span className={styles.tabCount}>{tab.count}</span>
            )}
          </button>
        ))}
      </div>

      <div className={styles.tabContent}>
        {activeTab === 'summary' && <SummaryTab stage={stage} />}
        {activeTab === 'chat' && (
          <div className={styles.chatTabWrapper}>
            <ChatHistory
              nodes={[syntheticNode]}
              selectedNodeId={stage.id}
              spans={stageSpans}
            />
          </div>
        )}
        {activeTab === 'tools' && <ToolsTab tools={tools} />}
        {activeTab === 'files' && <FilesTab files={files} />}
      </div>
    </div>
  );
};

export default StageDetail;
