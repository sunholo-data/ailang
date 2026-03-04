/**
 * StageDetail - Tabbed detail view for a single chain stage.
 * Tabs: Summary | Chat | Tools | Files
 * Summary tab includes per-turn breakdown matching `ailang chains tree --detailed`.
 * Reuses ChatHistory for the Chat tab.
 */
import React, { useState, useMemo } from 'react';
import type { ChainStageData, Span, HierarchyNode } from '../ExecHierarchy/types';
import type { StageDetailTab, ToolUsageEntry, FileTouchEntry, TurnBreakdown } from './types';
import { ChatHistory } from '../ExecHierarchy/ChatHistory';
import {
  formatCostOpt,
  formatTokensOpt,
  formatDurationMsOpt,
} from '../../../../utils/formatters';
import styles from './ChainExplorer.module.css';

// ============================================================================
// Span detection helpers
// ============================================================================

/** Check if a span represents a tool use (multiple naming patterns) */
function isToolSpan(name: string): boolean {
  return (
    name.startsWith('claude_code.tool.') ||
    name === 'exec.tool_use' ||
    name.includes('.tool.') ||
    name.includes('tool_use')
  );
}

/** Extract tool name from a tool span */
function getToolName(span: Span): string {
  if (span.name.startsWith('claude_code.tool.')) {
    return span.name.replace('claude_code.tool.', '');
  }
  if (span.name === 'exec.tool_use') {
    const attrs = span.attributes || {};
    return attrs['tool.name'] || attrs['tool_name'] || 'Unknown';
  }
  const match = span.name.match(/\.tool\.(\w+)$/);
  if (match) return match[1];
  return span.display_name || span.name;
}

/** Check if a span represents a turn */
function isTurnSpan(name: string): boolean {
  return (
    name === 'api_request' ||
    name.startsWith('exec.turn') ||
    name.includes('.turn')
  );
}

// ============================================================================
// Span extraction helpers
// ============================================================================

/** Walk span tree and collect tool usage entries */
function extractToolUsage(spans: Span[]): ToolUsageEntry[] {
  const toolMap = new Map<string, ToolUsageEntry>();

  function walk(spanList: Span[]) {
    for (const span of spanList) {
      if (isToolSpan(span.name)) {
        const toolName = getToolName(span);
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

/** Extract a file path from a tool span, checking multiple sources */
function extractFilePath(span: Span): string | null {
  const attrs = span.attributes || {};

  // 1. Direct attribute keys (various naming conventions)
  const direct = attrs['file_path'] || attrs['path'] || attrs['file']
    || attrs['tool.file_path'] || attrs['file.path'];
  if (direct && !direct.includes('*')) return direct;

  // 2. Parse tool.input JSON (OTEL stores tool params as JSON string)
  const toolInput = attrs['tool.input'];
  if (toolInput) {
    try {
      const parsed = typeof toolInput === 'string' ? JSON.parse(toolInput) : toolInput;
      const fromInput = parsed?.file_path || parsed?.path || parsed?.file
        || parsed?.notebook_path || parsed?.directory;
      if (fromInput && typeof fromInput === 'string' && !fromInput.includes('*')) return fromInput;
    } catch { /* not JSON */ }
  }

  // 3. Parse display_name (enriched: "Read: /path/to/file.go")
  if (span.display_name) {
    const match = span.display_name.match(/:\s*(\/[^\s]+)/);
    if (match) return match[1];
  }

  return null;
}

/** Walk span tree and extract file paths from tool attributes */
function extractFiles(spans: Span[]): FileTouchEntry[] {
  const fileMap = new Map<string, FileTouchEntry>();

  function walk(spanList: Span[]) {
    for (const span of spanList) {
      if (isToolSpan(span.name)) {
        const toolName = getToolName(span).toLowerCase();
        const filePath = extractFilePath(span);
        if (filePath) {
          const opMap: Record<string, string> = {
            read: 'read', edit: 'edit', write: 'write', glob: 'glob', grep: 'grep',
            notebookedit: 'edit', bash: 'exec',
          };
          const op = opMap[toolName] || toolName;
          const existing = fileMap.get(filePath) || { path: filePath, operations: [], lastTouched: 0 };
          if (!existing.operations.includes(op)) existing.operations.push(op);
          const timestamp = span.startMs || 0;
          if (timestamp > existing.lastTouched) existing.lastTouched = timestamp;
          fileMap.set(filePath, existing);
        }
      }
      if (span.children) walk(span.children);
    }
  }

  walk(spans);
  return Array.from(fileMap.values()).sort((a, b) => b.lastTouched - a.lastTouched);
}

/** Walk span tree and extract per-turn breakdown with tool names.
 *  Mirrors `ailang chains tree --detailed` CLI output. */
function extractTurnBreakdown(spans: Span[]): TurnBreakdown[] {
  const turns: TurnBreakdown[] = [];
  let counter = 0;

  function collectTurns(spanList: Span[]) {
    for (const span of spanList) {
      if (isTurnSpan(span.name)) {
        counter++;
        const attrs = span.attributes || {};
        // Try multiple attribute paths for turn number
        const attrTurnNum = attrs['turn.number'] || attrs['exec.turn'] || attrs['turn_number'];
        // Also check chat_context for Claude Code sessions
        const chatTurnNum = (span as any).chat_context?.turn_number;
        const rawNum = attrTurnNum ? parseInt(String(attrTurnNum), 10) : (chatTurnNum || counter);
        const turnNum = isNaN(rawNum) ? counter : rawNum;

        const cost = parseFloat(attrs['cost_usd'] || '0') || 0;

        // Collect tool names from direct children
        const tools: { name: string; detail?: string }[] = [];
        if (span.children) {
          for (const child of span.children) {
            if (isToolSpan(child.name)) {
              const name = getToolName(child);
              const detail = child.display_name
                ? child.display_name.replace(`${name}: `, '').substring(0, 60)
                : undefined;
              tools.push({ name, detail });
            }
          }
        }

        turns.push({
          turnNumber: turnNum,
          spanId: span.id,
          durationMs: span.durationMs || 0,
          cost,
          tools,
        });
      } else if (span.children) {
        // Recurse into non-turn spans (session, executor wrapper spans)
        collectTurns(span.children);
      }
    }
  }

  collectTurns(spans);
  turns.sort((a, b) => a.turnNumber - b.turnNumber);
  return turns;
}

// ============================================================================
// Tab Components
// ============================================================================

interface SummaryTabProps {
  stage: ChainStageData;
  turns: TurnBreakdown[];
}

const TURNS_PAGE_SIZE = 10;

const SummaryTab: React.FC<SummaryTabProps> = ({ stage, turns }) => {
  const [showAllTurns, setShowAllTurns] = useState(false);
  const durationMs = stage.duration_ms || 0;
  // Use stage.tool_calls metadata as the authoritative count
  const toolCalls = stage.tool_calls || 0;

  const visibleTurns = showAllTurns ? turns : turns.slice(0, TURNS_PAGE_SIZE);
  const hiddenCount = turns.length - TURNS_PAGE_SIZE;

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
          <span className={styles.metricValue}>{toolCalls}</span>
        </div>
        <div className={styles.metricCard}>
          <span className={styles.metricLabel}>Duration</span>
          <span className={styles.metricValue}>{formatDurationMsOpt(durationMs, '--')}</span>
        </div>
      </div>

      {/* Per-turn breakdown (matching ailang chains tree --detailed) */}
      {turns.length > 0 && (
        <div className={styles.turnsSection}>
          <span className={styles.turnsSectionTitle}>
            Turns ({turns.length})
          </span>
          <div className={styles.turnsList}>
            {visibleTurns.map(turn => (
              <div key={turn.spanId} className={styles.turnRow}>
                <span className={styles.turnNumber}>Turn {turn.turnNumber}</span>
                {turn.tools.length > 0 && (
                  <span className={styles.turnTools}>
                    {turn.tools.map((t, i) => (
                      <span key={i} className={styles.turnToolPill} title={t.detail}>
                        {t.name}
                      </span>
                    ))}
                  </span>
                )}
                {turn.cost > 0 && (
                  <span className={styles.turnCost}>
                    {formatCostOpt(turn.cost)}
                  </span>
                )}
              </div>
            ))}
          </div>
          {!showAllTurns && hiddenCount > 0 && (
            <button
              className={styles.loadMoreButton}
              onClick={() => setShowAllTurns(true)}
            >
              Show {hiddenCount} more turn{hiddenCount !== 1 ? 's' : ''}
            </button>
          )}
          {showAllTurns && turns.length > TURNS_PAGE_SIZE && (
            <button
              className={styles.loadMoreButton}
              onClick={() => setShowAllTurns(false)}
            >
              Show fewer
            </button>
          )}
        </div>
      )}

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
  spans: externalSpans,
  hiddenSpanTypes,
  theme,
}) => {
  const [activeTab, setActiveTab] = useState<StageDetailTab>('summary');

  // Use externally-loaded spans (from useStageSpans) when available, fall back to inline
  const stageSpans = externalSpans ?? stage.spans ?? [];
  const tools = useMemo(() => extractToolUsage(stageSpans), [stageSpans]);
  const files = useMemo(() => extractFiles(stageSpans), [stageSpans]);
  const turns = useMemo(() => extractTurnBreakdown(stageSpans), [stageSpans]);

  // Use extracted tool count, falling back to stage metadata
  const toolCount = tools.length > 0 ? tools.reduce((sum, t) => sum + t.count, 0) : (stage.tool_calls || 0);

  // Synthesize a HierarchyNode from stage metadata for ChatHistory context.
  // Include _span with session.id so ChatHistory can find the Claude Code session,
  // and taskId for coordinator event lookups.
  const syntheticNode = useMemo<HierarchyNode>(() => ({
    id: stage.id,
    type: 'exec',
    label: stage.agent_id,
    status: stage.status === 'completed' ? 'completed' : stage.status === 'failed' ? 'error' : 'busy',
    taskId: stage.task_id,
    agentId: stage.agent_id,
    durationMs: stage.duration_ms,
    // Attach _span with session.id so extractClaudeSessionId finds it
    _span: (stage.session_id || (stage.task_id && /^[0-9a-f]{8}-/.test(stage.task_id))) ? {
      id: stageSpans[0]?.id || stage.id,
      name: 'stage',
      startMs: 0,
      durationMs: stage.duration_ms || 0,
      attributes: { 'session.id': stage.session_id || stage.task_id! },
    } : undefined,
  }), [stage, stageSpans]);

  const tabs: { id: StageDetailTab; label: string; count?: number }[] = [
    { id: 'summary', label: 'Summary' },
    { id: 'chat', label: 'Chat' },
    { id: 'tools', label: 'Tools', count: toolCount },
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
        {activeTab === 'summary' && <SummaryTab stage={stage} turns={turns} />}
        {activeTab === 'chat' && (
          <div className={styles.chatTabWrapper}>
            <ChatHistory
              nodes={[syntheticNode]}
              selectedNodeId={stage.task_id || stage.id}
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
