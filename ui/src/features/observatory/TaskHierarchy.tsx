import React, { useState, useEffect } from 'react';
import { useTaskHierarchy, TaskHierarchy as TaskHierarchyType, AgentHierarchy, TraceHierarchy, SpanNode } from '../../hooks/useObservatory';
import { TaskTimeline } from './components/TaskTimeline';
import styles from './TaskHierarchy.module.css';

// Format duration in human-readable format
function formatDuration(ms: number | string): string {
  const val = Number(ms) || 0;
  if (val < 1000) return `${val.toFixed(0)}ms`;
  if (val < 60000) return `${(val / 1000).toFixed(2)}s`;
  return `${(val / 60000).toFixed(2)}m`;
}

// Format cost in USD
function formatCost(usd: number | string): string {
  const val = Number(usd) || 0;
  if (val < 0.01) return `$${val.toFixed(4)}`;
  return `$${val.toFixed(2)}`;
}

// Format large numbers
function formatNumber(n: number | string): string {
  const val = Number(n) || 0;
  if (val >= 1000000) return `${(val / 1000000).toFixed(1)}M`;
  if (val >= 1000) return `${(val / 1000).toFixed(1)}K`;
  return val.toString();
}

// Get status class
function getStatusClass(status: string): string {
  switch (status) {
    case 'completed':
    case 'ok':
    case 'OK':
      return styles.statusOk;
    case 'failed':
    case 'error':
    case 'ERROR':
      return styles.statusError;
    case 'running':
      return styles.statusRunning;
    case 'pending':
      return styles.statusPending;
    default:
      return styles.statusDefault;
  }
}

// Get provider class for color coding
function getProviderClass(provider: string): string {
  switch (provider) {
    case 'claude':
      return styles.providerClaude;
    case 'gemini':
      return styles.providerGemini;
    case 'openai':
      return styles.providerOpenai;
    default:
      return styles.providerOther;
  }
}

// Span node in the tree
function SpanNodeView({ node, depth = 0 }: { node: SpanNode; depth?: number }) {
  const [expanded, setExpanded] = useState(depth < 2);
  const hasChildren = node.children && node.children.length > 0;
  const span = node.span;

  return (
    <div className={styles.spanNode} style={{ marginLeft: `${depth * 20}px` }}>
      <div className={styles.spanHeader} onClick={() => hasChildren && setExpanded(!expanded)}>
        {hasChildren && (
          <span className={styles.expandIcon}>{expanded ? '▼' : '▶'}</span>
        )}
        {!hasChildren && <span className={styles.expandIcon}>•</span>}
        <span className={styles.spanName}>{span.name}</span>
        <span className={styles.spanDuration}>{formatDuration(span.duration_ms)}</span>
        <span className={`${styles.spanStatus} ${getStatusClass(span.status)}`}>
          {span.status}
        </span>
      </div>
      {span.tokens_in > 0 || span.tokens_out > 0 || span.cost_usd > 0 ? (
        <div className={styles.spanMetrics}>
          {span.tokens_in > 0 && (
            <span className={styles.metric}>↓{formatNumber(span.tokens_in)}</span>
          )}
          {span.tokens_out > 0 && (
            <span className={styles.metric}>↑{formatNumber(span.tokens_out)}</span>
          )}
          {span.cost_usd > 0 && (
            <span className={styles.cost}>{formatCost(span.cost_usd)}</span>
          )}
        </div>
      ) : null}
      {expanded && hasChildren && (
        <div className={styles.spanChildren}>
          {node.children!.map((child, idx) => (
            <SpanNodeView key={child.span.id || idx} node={child} depth={depth + 1} />
          ))}
        </div>
      )}
    </div>
  );
}

// Trace section within an agent
function TraceSection({ trace }: { trace: TraceHierarchy }) {
  const [expanded, setExpanded] = useState(false);

  return (
    <div className={styles.traceSection}>
      <div className={styles.traceHeader} onClick={() => setExpanded(!expanded)}>
        <span className={styles.expandIcon}>{expanded ? '▼' : '▶'}</span>
        <span className={styles.traceId}>Trace: {trace.trace_id.substring(0, 8)}...</span>
        <div className={styles.traceSummary}>
          <span className={styles.summaryItem}>{trace.summary.span_count} spans</span>
          <span className={styles.summaryItem}>{formatDuration(trace.summary.duration_ms)}</span>
          {trace.summary.total_tokens > 0 && (
            <span className={styles.summaryItem}>{formatNumber(trace.summary.total_tokens)} tokens</span>
          )}
          {trace.summary.total_cost_usd > 0 && (
            <span className={styles.cost}>{formatCost(trace.summary.total_cost_usd)}</span>
          )}
          {trace.summary.error_count > 0 && (
            <span className={styles.errorCount}>{trace.summary.error_count} errors</span>
          )}
        </div>
      </div>
      {expanded && (
        <div className={styles.spanTree}>
          {trace.root_span ? (
            <SpanNodeView node={trace.root_span} />
          ) : (
            trace.spans.map((node, idx) => (
              <SpanNodeView key={node.span.id || idx} node={node} />
            ))
          )}
        </div>
      )}
    </div>
  );
}

// Agent section within a task
function AgentSection({ agent }: { agent: AgentHierarchy }) {
  const [expanded, setExpanded] = useState(true);
  const assignment = agent.agent;

  // Calculate totals across all traces
  const totalSpans = agent.traces.reduce((sum, t) => sum + t.summary.span_count, 0);
  const totalTokens = agent.traces.reduce((sum, t) => sum + t.summary.total_tokens, 0);
  const totalCost = agent.traces.reduce((sum, t) => sum + t.summary.total_cost_usd, 0);
  const totalErrors = agent.traces.reduce((sum, t) => sum + t.summary.error_count, 0);

  return (
    <div className={styles.agentSection}>
      <div className={styles.agentHeader} onClick={() => setExpanded(!expanded)}>
        <span className={styles.expandIcon}>{expanded ? '▼' : '▶'}</span>
        <span className={`${styles.agentId} ${getProviderClass(assignment.provider)}`}>
          {assignment.agent_id}
        </span>
        <span className={`${styles.agentStatus} ${getStatusClass(assignment.status)}`}>
          {assignment.status}
        </span>
        <div className={styles.agentSummary}>
          <span className={styles.summaryItem}>{agent.traces.length} traces</span>
          <span className={styles.summaryItem}>{totalSpans} spans</span>
          {totalTokens > 0 && (
            <span className={styles.summaryItem}>{formatNumber(totalTokens)} tokens</span>
          )}
          {totalCost > 0 && (
            <span className={styles.cost}>{formatCost(totalCost)}</span>
          )}
          {totalErrors > 0 && (
            <span className={styles.errorCount}>{totalErrors} errors</span>
          )}
        </div>
      </div>
      {expanded && agent.traces.length > 0 && (
        <div className={styles.tracesContainer}>
          {agent.traces.map((trace, idx) => (
            <TraceSection key={trace.trace_id || idx} trace={trace} />
          ))}
        </div>
      )}
      {expanded && agent.traces.length === 0 && (
        <div className={styles.emptyTraces}>No traces recorded yet</div>
      )}
    </div>
  );
}

// Task summary card
function TaskSummaryCard({ hierarchy }: { hierarchy: TaskHierarchyType }) {
  const task = hierarchy.task;

  return (
    <div className={styles.taskSummary}>
      <div className={styles.taskHeader}>
        <h3 className={styles.taskTitle}>{task.title || task.id}</h3>
        <span className={`${styles.taskStatus} ${getStatusClass(task.status)}`}>
          {task.status}
        </span>
      </div>
      <div className={styles.taskMetrics}>
        <div className={styles.taskMetric}>
          <div className={styles.metricValue}>{hierarchy.agents.length}</div>
          <div className={styles.metricLabel}>Agents</div>
        </div>
        <div className={styles.taskMetric}>
          <div className={styles.metricValue}>{formatNumber(task.span_count || 0)}</div>
          <div className={styles.metricLabel}>Spans</div>
        </div>
        <div className={styles.taskMetric}>
          <div className={styles.metricValue}>{formatNumber((task.total_tokens_in || 0) + (task.total_tokens_out || 0))}</div>
          <div className={styles.metricLabel}>Tokens</div>
        </div>
        <div className={styles.taskMetric}>
          <div className={styles.metricValue}>{formatCost(task.total_cost_usd || 0)}</div>
          <div className={styles.metricLabel}>Cost</div>
        </div>
        <div className={styles.taskMetric}>
          <div className={styles.metricValue}>{formatDuration(task.total_duration_ms || 0)}</div>
          <div className={styles.metricLabel}>Duration</div>
        </div>
        {task.error_count > 0 && (
          <div className={styles.taskMetric}>
            <div className={`${styles.metricValue} ${styles.errorValue}`}>{task.error_count}</div>
            <div className={styles.metricLabel}>Errors</div>
          </div>
        )}
      </div>
    </div>
  );
}

// Main TaskHierarchy component
export interface TaskHierarchyViewProps {
  taskId: string;
  onClose?: () => void;
}

export function TaskHierarchyView({ taskId, onClose }: TaskHierarchyViewProps) {
  const { hierarchy, loading, error, refresh } = useTaskHierarchy(taskId, {
    includeSpans: true,
  });

  // Auto-refresh every 5 seconds if task is running
  useEffect(() => {
    if (!hierarchy?.task || hierarchy.task.status !== 'running') return;

    const interval = setInterval(refresh, 5000);
    return () => clearInterval(interval);
  }, [hierarchy?.task?.status, refresh]);

  if (loading && !hierarchy) {
    return (
      <div className={styles.container}>
        <div className={styles.loading}>Loading task hierarchy...</div>
      </div>
    );
  }

  if (error) {
    return (
      <div className={styles.container}>
        <div className={styles.error}>Error: {error}</div>
      </div>
    );
  }

  if (!hierarchy) {
    return (
      <div className={styles.container}>
        <div className={styles.emptyState}>Task not found</div>
      </div>
    );
  }

  return (
    <div className={styles.container}>
      <div className={styles.header}>
        <h2>Task Hierarchy</h2>
        <div className={styles.headerActions}>
          <button onClick={refresh} className={styles.refreshButton}>
            Refresh
          </button>
          {onClose && (
            <button onClick={onClose} className={styles.closeButton}>
              Close
            </button>
          )}
        </div>
      </div>

      <TaskSummaryCard hierarchy={hierarchy} />

      {/* Visual Timeline */}
      <TaskTimeline taskId={taskId} />

      <div className={styles.agentsContainer}>
        <h3 className={styles.sectionTitle}>Agent Assignments</h3>
        {hierarchy.agents.length > 0 ? (
          hierarchy.agents.map((agent, idx) => (
            <AgentSection key={agent.agent.id || idx} agent={agent} />
          ))
        ) : (
          <div className={styles.emptyState}>No agents assigned yet</div>
        )}
      </div>
    </div>
  );
}

export default TaskHierarchyView;
