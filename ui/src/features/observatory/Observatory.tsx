import React, { useState } from 'react';
import { useTraces, useTrace, useMetrics, useTasks, useObservatoryWs, useTelemetryConfig, TraceSummary, MetricsSummary, Task } from '../../hooks/useObservatory';
import { TaskHierarchyView } from './TaskHierarchy';
import styles from './Observatory.module.css';

type TabType = 'traces' | 'tasks';

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

// Metrics overview card
function MetricsCard({ metrics }: { metrics: MetricsSummary | null }) {
  if (!metrics) {
    return (
      <div className={styles.metricsCard}>
        <div className={styles.loading}>Loading metrics...</div>
      </div>
    );
  }

  return (
    <div className={styles.metricsCard}>
      <h2>Overview</h2>
      <div className={styles.metricsGrid}>
        <div className={styles.metric}>
          <div className={styles.metricValue}>{formatNumber(metrics.total_traces || 0)}</div>
          <div className={styles.metricLabel}>Traces</div>
        </div>
        <div className={styles.metric}>
          <div className={styles.metricValue}>{formatNumber(metrics.total_spans || 0)}</div>
          <div className={styles.metricLabel}>Spans</div>
        </div>
        <div className={styles.metric}>
          <div className={styles.metricValue}>{formatNumber(metrics.total_tasks || 0)}</div>
          <div className={styles.metricLabel}>Tasks</div>
        </div>
        <div className={styles.metric}>
          <div className={styles.metricValue}>{formatCost(metrics.total_cost_usd || 0)}</div>
          <div className={styles.metricLabel}>Total Cost</div>
        </div>
        <div className={styles.metric}>
          <div className={styles.metricValue}>{formatNumber((metrics.total_tokens_in || 0) + (metrics.total_tokens_out || 0))}</div>
          <div className={styles.metricLabel}>Tokens</div>
        </div>
        <div className={styles.metric}>
          <div className={styles.metricValue}>{formatDuration(metrics.avg_duration_ms || 0)}</div>
          <div className={styles.metricLabel}>Avg Duration</div>
        </div>
        <div className={styles.metric}>
          <div className={styles.metricValue}>{((metrics.success_rate || 0) * 100).toFixed(1)}%</div>
          <div className={styles.metricLabel}>Success Rate</div>
        </div>
      </div>
    </div>
  );
}

// Format service name for display (shorter, friendlier labels)
function formatServiceName(serviceName: string | undefined): string {
  if (!serviceName) return '-';
  // Map common service names to shorter labels
  const serviceMap: Record<string, string> = {
    'ailang-run': 'run',
    'ailang-eval': 'eval',
    'ailang-check': 'check',
    'ailang-messages': 'msg',
    'ailang-server': 'server',
    'ailang-coordinator': 'coord',
    'claude-code': 'claude',
    'claude-code-vscode': 'claude',
    'claude-code-test': 'test',
    'gemini-cli': 'gemini',
  };
  return serviceMap[serviceName] || serviceName;
}

// Get color class for service name
function getServiceClass(serviceName: string | undefined): string {
  if (!serviceName) return '';
  if (serviceName.startsWith('ailang-')) return styles.serviceAilang;
  if (serviceName === 'claude-code') return styles.serviceClaude;
  if (serviceName === 'gemini-cli') return styles.serviceGemini;
  if (serviceName === 'coordinator') return styles.serviceCoordinator;
  return styles.serviceOther;
}

// Trace list item with inline expandable details
function TraceRowWithDetail({ trace, isExpanded, onToggle }: {
  trace: TraceSummary;
  isExpanded: boolean;
  onToggle: () => void;
}) {
  const statusClass = trace.status === 'OK' ? styles.statusOk :
                      trace.status === 'ERROR' ? styles.statusError : styles.statusUnset;

  return (
    <>
      <tr className={`${styles.traceRow} ${isExpanded ? styles.traceRowExpanded : ''}`} onClick={onToggle}>
        <td className={styles.traceExpander}>{isExpanded ? '▼' : '▶'}</td>
        <td className={`${styles.traceSource} ${getServiceClass(trace.service_name)}`}>
          {formatServiceName(trace.service_name)}
        </td>
        <td className={styles.traceId}>{trace.trace_id.substring(0, 8)}...</td>
        <td className={styles.traceName}>{trace.root_span}</td>
        <td className={styles.traceSpans}>{trace.span_count}</td>
        <td className={styles.traceDuration}>{formatDuration(trace.duration_ms)}</td>
        <td className={statusClass}>{trace.status}</td>
        <td className={styles.traceTime}>{new Date(trace.start_time).toLocaleString()}</td>
      </tr>
      {isExpanded && (
        <tr className={styles.traceDetailRow}>
          <td colSpan={8}>
            <TraceDetailView traceId={trace.trace_id} />
          </td>
        </tr>
      )}
    </>
  );
}

// Trace list component with inline expansion
function TraceList({ traces }: { traces: TraceSummary[] }) {
  const [expandedTraceId, setExpandedTraceId] = useState<string | null>(null);

  if (traces.length === 0) {
    return (
      <div className={styles.emptyState}>
        <p>No traces yet. Traces will appear here as AI operations complete.</p>
      </div>
    );
  }

  const toggleTrace = (traceId: string) => {
    setExpandedTraceId(prev => prev === traceId ? null : traceId);
  };

  return (
    <table className={styles.traceTable}>
      <thead>
        <tr>
          <th></th>
          <th>Source</th>
          <th>Trace ID</th>
          <th>Root Span</th>
          <th>Spans</th>
          <th>Duration</th>
          <th>Status</th>
          <th>Time</th>
        </tr>
      </thead>
      <tbody>
        {traces.map(trace => (
          <TraceRowWithDetail
            key={trace.trace_id}
            trace={trace}
            isExpanded={expandedTraceId === trace.trace_id}
            onToggle={() => toggleTrace(trace.trace_id)}
          />
        ))}
      </tbody>
    </table>
  );
}

// Task list item
function TaskRow({ task, onSelect }: { task: Task; onSelect: () => void }) {
  const statusClass = task.status === 'completed' ? styles.statusOk :
                      task.status === 'failed' ? styles.statusError :
                      task.status === 'running' ? styles.statusRunning : styles.statusPending;

  return (
    <tr className={styles.traceRow} onClick={onSelect}>
      <td className={styles.traceExpander}>▶</td>
      <td className={styles.taskTitle}>{task.title || task.id.substring(0, 12)}</td>
      <td className={statusClass}>{task.status}</td>
      <td className={styles.traceSpans}>{formatNumber(task.span_count || 0)}</td>
      <td className={styles.traceDuration}>{formatNumber((task.total_tokens_in || 0) + (task.total_tokens_out || 0))}</td>
      <td className={styles.traceDuration}>{formatCost(task.total_cost_usd || 0)}</td>
      <td className={styles.traceTime}>{new Date(task.created_at).toLocaleString()}</td>
    </tr>
  );
}

// Task list component
function TaskList({ tasks, onSelectTask }: { tasks: Task[]; onSelectTask: (taskId: string) => void }) {
  if (tasks.length === 0) {
    return (
      <div className={styles.emptyState}>
        <p>No tasks yet. Tasks will appear here when coordinator processes work.</p>
      </div>
    );
  }

  return (
    <table className={styles.traceTable}>
      <thead>
        <tr>
          <th></th>
          <th>Task</th>
          <th>Status</th>
          <th>Spans</th>
          <th>Tokens</th>
          <th>Cost</th>
          <th>Created</th>
        </tr>
      </thead>
      <tbody>
        {tasks.map(task => (
          <TaskRow key={task.id} task={task} onSelect={() => onSelectTask(task.id)} />
        ))}
      </tbody>
    </table>
  );
}

// Connection status indicator
function ConnectionStatus({ isConnected }: { isConnected: boolean }) {
  return (
    <div className={`${styles.connectionStatus} ${isConnected ? styles.connected : styles.disconnected}`}>
      <span className={styles.statusDot}></span>
      {isConnected ? 'Live' : 'Disconnected'}
    </div>
  );
}

// GCP Cloud Trace link
function GCPTraceLink({ url }: { url: string }) {
  return (
    <a
      href={url}
      target="_blank"
      rel="noopener noreferrer"
      className={styles.gcpLink}
      title="View traces in Google Cloud Console"
    >
      <span className={styles.gcpIcon}>☁</span>
      GCP Trace
    </a>
  );
}

// Main Observatory component
export function Observatory() {
  const [activeTab, setActiveTab] = useState<TabType>('traces');
  const [selectedTaskId, setSelectedTaskId] = useState<string | null>(null);

  const { traces, loading: tracesLoading, error: tracesError, refresh: refreshTraces } = useTraces({ limit: 50 });
  const { tasks, loading: tasksLoading, error: tasksError, refresh: refreshTasks } = useTasks({ limit: 50 });
  const { metrics, refresh: refreshMetrics } = useMetrics();
  const { config: telemetryConfig } = useTelemetryConfig();

  // Real-time updates
  const { isConnected } = useObservatoryWs({
    onSpanCreated: () => {
      refreshTraces();
      refreshTasks();
      refreshMetrics();
    },
    onTaskCompleted: () => {
      refreshTraces();
      refreshTasks();
      refreshMetrics();
    },
    onMetricsUpdated: () => {
      refreshMetrics();
    },
  });

  const handleSelectTask = (taskId: string) => {
    setSelectedTaskId(taskId);
  };

  const handleCloseTaskDetail = () => {
    setSelectedTaskId(null);
  };

  return (
    <div className={styles.observatory}>
      <header className={styles.header}>
        <h1>Observatory</h1>
        <div className={styles.headerActions}>
          {telemetryConfig?.gcp_enabled && telemetryConfig?.gcp_trace_url && (
            <GCPTraceLink url={telemetryConfig.gcp_trace_url} />
          )}
          <ConnectionStatus isConnected={isConnected} />
        </div>
      </header>

      <section className={styles.overview}>
        <MetricsCard metrics={metrics} />
      </section>

      {/* Tab Navigation */}
      <div className={styles.tabNav}>
        <button
          className={`${styles.tabButton} ${activeTab === 'traces' ? styles.tabButtonActive : ''}`}
          onClick={() => { setActiveTab('traces'); setSelectedTaskId(null); }}
        >
          Traces
        </button>
        <button
          className={`${styles.tabButton} ${activeTab === 'tasks' ? styles.tabButtonActive : ''}`}
          onClick={() => { setActiveTab('tasks'); setSelectedTaskId(null); }}
        >
          Tasks
        </button>
      </div>

      {/* Task Detail View */}
      {selectedTaskId && (
        <section className={styles.taskDetail}>
          <TaskHierarchyView taskId={selectedTaskId} onClose={handleCloseTaskDetail} />
        </section>
      )}

      {/* Traces Tab Content */}
      {activeTab === 'traces' && !selectedTaskId && (
        <section className={styles.traces}>
          <div className={styles.sectionHeader}>
            <h2>Recent Traces</h2>
            <button onClick={refreshTraces} className={styles.refreshButton}>
              Refresh
            </button>
          </div>

          {tracesLoading && <div className={styles.loading}>Loading traces...</div>}
          {tracesError && <div className={styles.error}>Error: {tracesError}</div>}
          {!tracesLoading && !tracesError && (
            <TraceList traces={traces} />
          )}
        </section>
      )}

      {/* Tasks Tab Content */}
      {activeTab === 'tasks' && !selectedTaskId && (
        <section className={styles.traces}>
          <div className={styles.sectionHeader}>
            <h2>Recent Tasks</h2>
            <button onClick={refreshTasks} className={styles.refreshButton}>
              Refresh
            </button>
          </div>

          {tasksLoading && <div className={styles.loading}>Loading tasks...</div>}
          {tasksError && <div className={styles.error}>Error: {tasksError}</div>}
          {!tasksLoading && !tasksError && (
            <TaskList tasks={tasks} onSelectTask={handleSelectTask} />
          )}
        </section>
      )}
    </div>
  );
}

// Span detail card
function SpanDetail({ span }: { span: any }) {
  const [showAttributes, setShowAttributes] = useState(false);

  // Extract key metrics from attributes
  const attrs = span.attributes || {};
  const inputTokens = span.tokens_in || attrs.input_tokens;
  const outputTokens = span.tokens_out || attrs.output_tokens;
  const cacheReadTokens = attrs.cache_read_tokens;
  const cacheCreationTokens = attrs.cache_creation_tokens;
  const durationMs = attrs.duration_ms || span.duration_ms;
  const sessionId = attrs['session.id']?.substring(0, 8);

  return (
    <div
      className={styles.spanNode}
      style={{ marginLeft: span.parent_span_id ? '24px' : '0' }}
    >
      <div className={styles.spanHeader}>
        <span className={styles.spanName}>{span.name}</span>
        <span className={styles.spanDuration}>{formatDuration(durationMs || 0)}</span>
        <span className={`${styles.spanStatus} ${span.status === 'ok' ? styles.statusOk : span.status === 'error' ? styles.statusError : styles.statusUnset}`}>
          {span.status}
        </span>
      </div>

      <div className={styles.spanMeta}>
        {span.provider && <span className={styles.spanProvider}>{span.provider}</span>}
        {span.model && <span className={styles.spanModel}>{span.model}</span>}
        {sessionId && <span className={styles.spanSession}>session: {sessionId}...</span>}
      </div>

      {/* Token & Cost breakdown */}
      {(inputTokens || outputTokens || span.cost_usd) && (
        <div className={styles.spanMetrics}>
          {inputTokens > 0 && (
            <span className={styles.spanMetric}>
              <span className={styles.metricIcon}>↓</span>
              {formatNumber(inputTokens)} in
            </span>
          )}
          {outputTokens > 0 && (
            <span className={styles.spanMetric}>
              <span className={styles.metricIcon}>↑</span>
              {formatNumber(outputTokens)} out
            </span>
          )}
          {cacheReadTokens > 0 && (
            <span className={styles.spanMetric}>
              <span className={styles.metricIcon}>
                <svg width="12" height="12" viewBox="0 0 24 24" fill="currentColor">
                  <path d="M13 2L3 14h9l-1 8 10-12h-9l1-8z" />
                </svg>
              </span>
              {formatNumber(cacheReadTokens)} cached
            </span>
          )}
          {span.cost_usd > 0 && (
            <span className={styles.spanCost}>{formatCost(span.cost_usd)}</span>
          )}
        </div>
      )}

      {/* Expandable attributes */}
      {Object.keys(attrs).length > 0 && (
        <div className={styles.spanAttributes}>
          <button
            className={styles.attributeToggle}
            onClick={() => setShowAttributes(!showAttributes)}
          >
            {showAttributes ? '▼' : '▶'} Attributes ({Object.keys(attrs).length})
          </button>
          {showAttributes && (
            <div className={styles.attributeList}>
              {Object.entries(attrs).map(([key, value]) => (
                <div key={key} className={styles.attributeItem}>
                  <span className={styles.attributeKey}>{key}:</span>
                  <span className={styles.attributeValue}>
                    {typeof value === 'object' ? JSON.stringify(value) : String(value)}
                  </span>
                </div>
              ))}
            </div>
          )}
        </div>
      )}
    </div>
  );
}

// Trace detail view
function TraceDetailView({ traceId }: { traceId: string }) {
  const { trace, loading, error } = useTrace(traceId);

  if (loading) return <div className={styles.loading}>Loading trace details...</div>;
  if (error) return <div className={styles.error}>Error: {error}</div>;
  if (!trace) return <div className={styles.emptyState}>Trace not found</div>;

  // Calculate totals
  const totalTokensIn = trace.spans.reduce((sum: number, s: any) => sum + (s.tokens_in || 0), 0);
  const totalTokensOut = trace.spans.reduce((sum: number, s: any) => sum + (s.tokens_out || 0), 0);
  const totalCost = trace.spans.reduce((sum: number, s: any) => sum + (s.cost_usd || 0), 0);

  return (
    <div className={styles.traceDetailContent}>
      {/* Trace summary */}
      <div className={styles.traceSummary}>
        <div className={styles.traceSummaryItem}>
          <span className={styles.summaryLabel}>Spans:</span>
          <span className={styles.summaryValue}>{trace.spans.length}</span>
        </div>
        {totalTokensIn > 0 && (
          <div className={styles.traceSummaryItem}>
            <span className={styles.summaryLabel}>Total Tokens:</span>
            <span className={styles.summaryValue}>{formatNumber(totalTokensIn)} in / {formatNumber(totalTokensOut)} out</span>
          </div>
        )}
        {totalCost > 0 && (
          <div className={styles.traceSummaryItem}>
            <span className={styles.summaryLabel}>Total Cost:</span>
            <span className={styles.summaryValue}>{formatCost(totalCost)}</span>
          </div>
        )}
        <div className={styles.traceSummaryItem}>
          <span className={styles.summaryLabel}>Duration:</span>
          <span className={styles.summaryValue}>{formatDuration(trace.duration_ms || 0)}</span>
        </div>
      </div>

      {/* Span tree */}
      <div className={styles.spanTree}>
        {trace.spans.map((span: any) => (
          <SpanDetail key={span.id} span={span} />
        ))}
      </div>
    </div>
  );
}

export default Observatory;
