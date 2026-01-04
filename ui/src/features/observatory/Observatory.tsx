import React, { useState } from 'react';
import { useTraces, useMetrics, useObservatoryWs, TraceSummary, MetricsSummary } from '../../hooks/useObservatory';
import styles from './Observatory.module.css';

// Format duration in human-readable format
function formatDuration(ms: number): string {
  if (ms < 1000) return `${ms.toFixed(0)}ms`;
  if (ms < 60000) return `${(ms / 1000).toFixed(2)}s`;
  return `${(ms / 60000).toFixed(2)}m`;
}

// Format cost in USD
function formatCost(usd: number): string {
  if (usd < 0.01) return `$${usd.toFixed(4)}`;
  return `$${usd.toFixed(2)}`;
}

// Format large numbers
function formatNumber(n: number): string {
  if (n >= 1000000) return `${(n / 1000000).toFixed(1)}M`;
  if (n >= 1000) return `${(n / 1000).toFixed(1)}K`;
  return n.toString();
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

// Trace list item
function TraceRow({ trace, onClick }: { trace: TraceSummary; onClick: () => void }) {
  const statusClass = trace.status === 'OK' ? styles.statusOk :
                      trace.status === 'ERROR' ? styles.statusError : styles.statusUnset;

  return (
    <tr className={styles.traceRow} onClick={onClick}>
      <td className={styles.traceId}>{trace.trace_id.substring(0, 8)}...</td>
      <td className={styles.traceName}>{trace.root_span_name}</td>
      <td className={styles.traceSpans}>{trace.span_count}</td>
      <td className={styles.traceDuration}>{formatDuration(trace.duration_ms)}</td>
      <td className={statusClass}>{trace.status}</td>
      <td className={styles.traceTime}>{new Date(trace.start_time).toLocaleString()}</td>
    </tr>
  );
}

// Trace list component
function TraceList({ traces, onSelectTrace }: { traces: TraceSummary[]; onSelectTrace: (id: string) => void }) {
  if (traces.length === 0) {
    return (
      <div className={styles.emptyState}>
        <p>No traces yet. Traces will appear here as AI operations complete.</p>
      </div>
    );
  }

  return (
    <table className={styles.traceTable}>
      <thead>
        <tr>
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
          <TraceRow
            key={trace.trace_id}
            trace={trace}
            onClick={() => onSelectTrace(trace.trace_id)}
          />
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

// Main Observatory component
export function Observatory() {
  const { traces, loading: tracesLoading, error: tracesError, refresh: refreshTraces } = useTraces({ limit: 50 });
  const { metrics, loading: metricsLoading, error: metricsError, refresh: refreshMetrics } = useMetrics();
  const [selectedTraceId, setSelectedTraceId] = useState<string | null>(null);

  // Real-time updates
  const { isConnected } = useObservatoryWs({
    onSpanCreated: () => {
      refreshTraces();
      refreshMetrics();
    },
    onTaskCompleted: () => {
      refreshTraces();
      refreshMetrics();
    },
    onMetricsUpdated: (newMetrics) => {
      // Could update metrics directly here instead of refetching
      refreshMetrics();
    },
  });

  return (
    <div className={styles.observatory}>
      <header className={styles.header}>
        <h1>Observatory</h1>
        <ConnectionStatus isConnected={isConnected} />
      </header>

      <section className={styles.overview}>
        <MetricsCard metrics={metrics} />
      </section>

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
          <TraceList traces={traces} onSelectTrace={setSelectedTraceId} />
        )}
      </section>

      {selectedTraceId && (
        <section className={styles.traceDetail}>
          <div className={styles.sectionHeader}>
            <h2>Trace Detail: {selectedTraceId.substring(0, 16)}...</h2>
            <button onClick={() => setSelectedTraceId(null)} className={styles.closeButton}>
              Close
            </button>
          </div>
          <TraceDetailView traceId={selectedTraceId} />
        </section>
      )}
    </div>
  );
}

// Trace detail view (placeholder - can be expanded)
function TraceDetailView({ traceId }: { traceId: string }) {
  const { trace, loading, error } = require('../../hooks/useObservatory').useTrace(traceId);

  if (loading) return <div className={styles.loading}>Loading trace details...</div>;
  if (error) return <div className={styles.error}>Error: {error}</div>;
  if (!trace) return <div className={styles.emptyState}>Trace not found</div>;

  return (
    <div className={styles.spanTree}>
      {trace.spans.map((span: any) => (
        <div
          key={span.id}
          className={styles.spanNode}
          style={{ marginLeft: span.parent_span_id ? '24px' : '0' }}
        >
          <div className={styles.spanHeader}>
            <span className={styles.spanName}>{span.name}</span>
            <span className={styles.spanDuration}>{formatDuration(span.duration_ms || 0)}</span>
          </div>
          <div className={styles.spanMeta}>
            <span className={styles.spanProvider}>{span.provider}</span>
            {span.model && <span className={styles.spanModel}>{span.model}</span>}
            {span.cost_usd > 0 && <span className={styles.spanCost}>{formatCost(span.cost_usd)}</span>}
          </div>
        </div>
      ))}
    </div>
  );
}

export default Observatory;
