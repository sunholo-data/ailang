import React, { useState, useEffect, useCallback } from 'react';

// Process statistics from the API
interface ProcessStats {
  instance_id: string;
  pid: number;
  started_at: string;
  duration_sec: number;
  cpu_percent: number;
  memory_mb: number;
  status: string;
  source?: string;      // "ui", "eval", "cli", "agent"
  command?: string;     // Short command description
  full_cmd?: string;    // Full command line (for debugging)
  stopped_at?: string;  // When the process stopped
  turns?: number;
  tokens_in?: number;
  tokens_out?: number;
  cost?: number;
}

interface MonitorSummary {
  total_processes: number;
  total_cpu_percent: number;
  total_memory_mb: number;
  total_cost: number;
  warning_count: number;
}

interface MonitorResponse {
  timestamp: string;
  processes: ProcessStats[];
  history?: ProcessStats[];  // Recently completed/failed processes
  summary: MonitorSummary;
}

// Telemetry event from WebSocket
interface TelemetryEvent {
  instance_id: string;
  pid: number;
  turns: number;
  tokens_in: number;
  tokens_out: number;
  cost: number;
  status: string;
  duration_sec: number;
}

// Icons
const Icons = {
  cpu: (
    <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
      <rect x="4" y="4" width="16" height="16" rx="2" />
      <rect x="9" y="9" width="6" height="6" />
      <line x1="9" y1="1" x2="9" y2="4" />
      <line x1="15" y1="1" x2="15" y2="4" />
      <line x1="9" y1="20" x2="9" y2="23" />
      <line x1="15" y1="20" x2="15" y2="23" />
      <line x1="20" y1="9" x2="23" y2="9" />
      <line x1="20" y1="14" x2="23" y2="14" />
      <line x1="1" y1="9" x2="4" y2="9" />
      <line x1="1" y1="14" x2="4" y2="14" />
    </svg>
  ),
  memory: (
    <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
      <rect x="2" y="6" width="20" height="12" rx="2" />
      <line x1="6" y1="10" x2="6" y2="14" />
      <line x1="10" y1="10" x2="10" y2="14" />
      <line x1="14" y1="10" x2="14" y2="14" />
      <line x1="18" y1="10" x2="18" y2="14" />
    </svg>
  ),
  clock: (
    <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
      <circle cx="12" cy="12" r="10" />
      <polyline points="12 6 12 12 16 14" />
    </svg>
  ),
  dollar: (
    <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
      <line x1="12" y1="1" x2="12" y2="23" />
      <path d="M17 5H9.5a3.5 3.5 0 0 0 0 7h5a3.5 3.5 0 0 1 0 7H6" />
    </svg>
  ),
  activity: (
    <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
      <polyline points="22 12 18 12 15 21 9 3 6 12 2 12" />
    </svg>
  ),
  tokens: (
    <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
      <path d="M14.5 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V7.5L14.5 2z" />
      <polyline points="14 2 14 8 20 8" />
      <line x1="16" y1="13" x2="8" y2="13" />
      <line x1="16" y1="17" x2="8" y2="17" />
      <line x1="10" y1="9" x2="8" y2="9" />
    </svg>
  ),
  message: (
    <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
      <path d="M21 15a2 2 0 0 1-2 2H7l-4 4V5a2 2 0 0 1 2-2h14a2 2 0 0 1 2 2z" />
    </svg>
  ),
  stop: (
    <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
      <rect x="3" y="3" width="18" height="18" rx="2" />
    </svg>
  ),
  warning: (
    <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
      <path d="M10.29 3.86L1.82 18a2 2 0 0 0 1.71 3h16.94a2 2 0 0 0 1.71-3L13.71 3.86a2 2 0 0 0-3.42 0z" />
      <line x1="12" y1="9" x2="12" y2="13" />
      <line x1="12" y1="17" x2="12.01" y2="17" />
    </svg>
  ),
  check: (
    <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
      <polyline points="20 6 9 17 4 12" />
    </svg>
  ),
};

// Track previous values for trend indicators
interface TrendData {
  prevCost: number;
  prevTokensIn: number;
  prevTokensOut: number;
  timestamp: number;
}

export const Monitor: React.FC = () => {
  const [data, setData] = useState<MonitorResponse | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [lastUpdate, setLastUpdate] = useState<Date | null>(null);
  const [telemetry, setTelemetry] = useState<Map<string, TelemetryEvent>>(new Map());
  const [trends, setTrends] = useState<TrendData>({ prevCost: 0, prevTokensIn: 0, prevTokensOut: 0, timestamp: 0 });

  const fetchMonitorData = useCallback(async () => {
    try {
      const response = await fetch('/api/monitor');
      if (!response.ok) {
        throw new Error(`Failed to fetch: ${response.statusText}`);
      }
      const result: MonitorResponse = await response.json();
      setData(result);
      setLastUpdate(new Date());
      setError(null);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Unknown error');
    }
  }, []);

  // Poll every 2 seconds
  useEffect(() => {
    fetchMonitorData();
    const interval = setInterval(fetchMonitorData, 2000);
    return () => clearInterval(interval);
  }, [fetchMonitorData]);

  // WebSocket connection for real-time telemetry
  useEffect(() => {
    const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
    const wsUrl = `${protocol}//${window.location.host}/ws`;

    let ws: WebSocket | null = null;
    let reconnectTimer: ReturnType<typeof setTimeout> | null = null;

    const connect = () => {
      ws = new WebSocket(wsUrl);

      ws.onmessage = (event) => {
        try {
          const msg = JSON.parse(event.data);
          if (msg.type === 'telemetry') {
            const telemData = msg.data as TelemetryEvent;
            setTelemetry(prev => {
              const next = new Map(prev);
              next.set(telemData.instance_id, telemData);
              return next;
            });
          }
        } catch {
          // Ignore parse errors
        }
      };

      ws.onclose = () => {
        // Reconnect after 3 seconds
        reconnectTimer = setTimeout(connect, 3000);
      };

      ws.onerror = () => {
        ws?.close();
      };
    };

    connect();

    return () => {
      if (reconnectTimer) clearTimeout(reconnectTimer);
      ws?.close();
    };
  }, []);

  const handleStopProcess = async (instanceId: string) => {
    try {
      await fetch(`/api/agents/${instanceId}`, { method: 'DELETE' });
      fetchMonitorData();
    } catch (err) {
      console.error('Failed to stop process:', err);
    }
  };

  const formatDuration = (seconds: number): string => {
    if (seconds < 0) return 'Unknown';
    if (seconds < 60) return `${seconds}s`;
    if (seconds < 3600) {
      const mins = Math.floor(seconds / 60);
      const secs = seconds % 60;
      return `${mins}m ${secs}s`;
    }
    const hours = Math.floor(seconds / 3600);
    const mins = Math.floor((seconds % 3600) / 60);
    return `${hours}h ${mins}m`;
  };

  const formatCost = (cost: number): string => {
    if (cost === 0) return '$0.00';
    if (cost < 0.01) return `$${cost.toFixed(4)}`;
    return `$${cost.toFixed(2)}`;
  };

  const getStatusColor = (status: string): string => {
    switch (status) {
      case 'running': return 'var(--color-success)';
      case 'completed': return 'var(--color-primary)';
      case 'failed': return 'var(--color-danger)';
      case 'orphan': return 'var(--color-warning)';
      default: return 'var(--text-tertiary)';
    }
  };

  const isWarning = (proc: ProcessStats): boolean => {
    return proc.cpu_percent > 80 || proc.duration_sec > 300;
  };

  // Merge process stats with real-time telemetry data
  const getProcessWithTelemetry = (proc: ProcessStats): ProcessStats & { hasLiveTelemetry: boolean } => {
    const telem = telemetry.get(proc.instance_id);
    if (telem) {
      return {
        ...proc,
        turns: telem.turns,
        tokens_in: telem.tokens_in,
        tokens_out: telem.tokens_out,
        cost: telem.cost,
        hasLiveTelemetry: true,
      };
    }
    return { ...proc, hasLiveTelemetry: false };
  };

  // Calculate totals from telemetry (live data)
  const telemetryTotals = Array.from(telemetry.values()).reduce(
    (acc, t) => ({
      tokens_in: acc.tokens_in + t.tokens_in,
      tokens_out: acc.tokens_out + t.tokens_out,
      cost: acc.cost + t.cost,
      turns: acc.turns + t.turns,
    }),
    { tokens_in: 0, tokens_out: 0, cost: 0, turns: 0 }
  );

  // Use telemetry cost if available, otherwise fall back to API summary
  const totalCost = telemetryTotals.cost > 0 ? telemetryTotals.cost : (data?.summary.total_cost || 0);
  const totalTokens = { in: telemetryTotals.tokens_in, out: telemetryTotals.tokens_out };
  const hasLiveTelemetry = telemetry.size > 0;

  // Update trends every 2 seconds
  useEffect(() => {
    if (hasLiveTelemetry) {
      const now = Date.now();
      if (now - trends.timestamp > 2000) {
        setTrends({
          prevCost: totalCost,
          prevTokensIn: totalTokens.in,
          prevTokensOut: totalTokens.out,
          timestamp: now,
        });
      }
    }
  }, [totalCost, totalTokens.in, totalTokens.out, hasLiveTelemetry, trends.timestamp]);

  // Calculate if values are increasing
  const costTrend = totalCost > trends.prevCost && trends.prevCost > 0 ? 'up' : null;
  const tokensTrend = (totalTokens.in + totalTokens.out) > (trends.prevTokensIn + trends.prevTokensOut) && (trends.prevTokensIn + trends.prevTokensOut) > 0 ? 'up' : null;

  const formatTokens = (count: number): string => {
    if (count >= 1000000) return `${(count / 1000000).toFixed(1)}M`;
    if (count >= 1000) return `${(count / 1000).toFixed(1)}K`;
    return count.toString();
  };

  return (
    <div className="monitor">
      {/* Summary Bar */}
      <div className="monitor-summary">
        <div className="summary-item">
          <span className="summary-icon">{Icons.activity}</span>
          <span className="summary-value">{data?.summary.total_processes || 0}</span>
          <span className="summary-label">Running</span>
        </div>
        <div className="summary-item">
          <span className="summary-icon">{Icons.cpu}</span>
          <span className="summary-value">{data?.summary.total_cpu_percent.toFixed(1) || '0.0'}%</span>
          <span className="summary-label">CPU</span>
        </div>
        <div className="summary-item">
          <span className="summary-icon">{Icons.memory}</span>
          <span className="summary-value">{data?.summary.total_memory_mb.toFixed(0) || '0'} MB</span>
          <span className="summary-label">Memory</span>
        </div>
        <div className={`summary-item ${hasLiveTelemetry ? 'live' : ''}`}>
          <span className="summary-icon">{Icons.dollar}</span>
          <span className="summary-value">
            {formatCost(totalCost)}
            {costTrend === 'up' && <span className="trend-up">▲</span>}
          </span>
          <span className="summary-label">Cost {hasLiveTelemetry && <span className="live-indicator">●</span>}</span>
        </div>
        <div className={`summary-item ${hasLiveTelemetry ? 'live' : ''}`}>
          <span className="summary-icon">{Icons.tokens}</span>
          <span className="summary-value">
            {formatTokens(totalTokens.in)}↓ / {formatTokens(totalTokens.out)}↑
            {tokensTrend === 'up' && <span className="trend-up">▲</span>}
          </span>
          <span className="summary-label">Tokens {hasLiveTelemetry && <span className="live-indicator">●</span>}</span>
        </div>
        {telemetryTotals.turns > 0 && (
          <div className="summary-item live">
            <span className="summary-icon">{Icons.message}</span>
            <span className="summary-value">{telemetryTotals.turns}</span>
            <span className="summary-label">Turns <span className="live-indicator">●</span></span>
          </div>
        )}
        {(data?.summary.warning_count || 0) > 0 && (
          <div className="summary-item warning">
            <span className="summary-icon">{Icons.warning}</span>
            <span className="summary-value">{data?.summary.warning_count}</span>
            <span className="summary-label">Warnings</span>
          </div>
        )}
        <div className="summary-spacer" />
        <div className="summary-update">
          {hasLiveTelemetry && <span className="live-badge-summary">LIVE</span>}
          Last update: {lastUpdate ? lastUpdate.toLocaleTimeString() : 'Never'}
        </div>
      </div>

      {/* Process Grid */}
      <div className="process-grid">
        {error && (
          <div className="error-card">
            <span className="error-icon">{Icons.warning}</span>
            <span>{error}</span>
          </div>
        )}

        {(!data?.processes || data.processes.length === 0) && !error && (
          <div className="empty-state">
            <span className="empty-icon">{Icons.activity}</span>
            <h3>No Active Processes</h3>
            <p>Spawn an agent from the Messages tab to see it here.</p>
          </div>
        )}

        {data?.processes.map((proc) => {
          const p = getProcessWithTelemetry(proc);
          return (
            <div
              key={p.instance_id}
              className={`process-card ${isWarning(p) ? 'warning' : ''} ${p.hasLiveTelemetry ? 'live' : ''}`}
            >
              <div className="process-header">
                <div className="process-status">
                  <span
                    className="status-dot"
                    style={{ background: getStatusColor(p.status) }}
                  />
                  <span className="process-name">{p.instance_id}</span>
                  {p.hasLiveTelemetry && (
                    <span className="live-badge">LIVE</span>
                  )}
                </div>
                {p.status === 'running' && (
                  <button
                    className="stop-btn"
                    onClick={() => handleStopProcess(p.instance_id)}
                    title="Stop process"
                  >
                    {Icons.stop}
                  </button>
                )}
                {p.status === 'completed' && (
                  <span className="status-badge completed">{Icons.check} Done</span>
                )}
              </div>

              <div className="process-metrics">
                <div className="metric">
                  <span className="metric-icon">{Icons.cpu}</span>
                  <span className={`metric-value ${p.cpu_percent > 80 ? 'high' : ''}`}>
                    {p.cpu_percent.toFixed(1)}%
                  </span>
                  <span className="metric-label">CPU</span>
                </div>
                <div className="metric">
                  <span className="metric-icon">{Icons.memory}</span>
                  <span className="metric-value">{p.memory_mb.toFixed(0)} MB</span>
                  <span className="metric-label">Memory</span>
                </div>
                <div className="metric">
                  <span className="metric-icon">{Icons.clock}</span>
                  <span className={`metric-value ${p.duration_sec > 300 ? 'high' : ''}`}>
                    {formatDuration(p.duration_sec)}
                  </span>
                  <span className="metric-label">Duration</span>
                </div>
              </div>

              {/* Telemetry row - only show if we have live data */}
              {p.hasLiveTelemetry && (
                <div className="process-telemetry">
                  <div className="telemetry-item">
                    <span className="telemetry-icon">{Icons.message}</span>
                    <span className="telemetry-value">{p.turns || 0}</span>
                    <span className="telemetry-label">Turns</span>
                  </div>
                  <div className="telemetry-item">
                    <span className="telemetry-icon">{Icons.tokens}</span>
                    <span className="telemetry-value">{formatTokens(p.tokens_in || 0)}</span>
                    <span className="telemetry-label">In</span>
                  </div>
                  <div className="telemetry-item">
                    <span className="telemetry-icon">{Icons.tokens}</span>
                    <span className="telemetry-value">{formatTokens(p.tokens_out || 0)}</span>
                    <span className="telemetry-label">Out</span>
                  </div>
                  <div className="telemetry-item">
                    <span className="telemetry-icon">{Icons.dollar}</span>
                    <span className="telemetry-value cost">{formatCost(p.cost || 0)}</span>
                    <span className="telemetry-label">Cost</span>
                  </div>
                </div>
              )}

              <div className="process-footer">
                <span className="process-pid">PID: {p.pid}</span>
                {p.source && (
                  <span className={`source-badge ${p.source}`}>{p.source}</span>
                )}
                {p.command && (
                  <span className="process-command" title={p.full_cmd}>{p.command}</span>
                )}
                {!p.hasLiveTelemetry && p.turns && (
                  <span className="process-turns">{p.turns} turns</span>
                )}
                {!p.hasLiveTelemetry && p.cost !== undefined && p.cost > 0 && (
                  <span className="process-cost">{formatCost(p.cost)}</span>
                )}
              </div>
            </div>
          );
        })}

        {/* History Section - Recently completed/failed processes */}
        {data?.history && data.history.length > 0 && (
          <>
            <div className="history-divider">
              <span>Recent History</span>
            </div>
            {data.history.map((proc) => (
              <div
                key={`history-${proc.instance_id}-${proc.stopped_at}`}
                className={`process-card history ${proc.status === 'failed' ? 'failed' : ''}`}
              >
                <div className="process-header">
                  <div className="process-status">
                    <span
                      className="status-dot"
                      style={{ background: getStatusColor(proc.status) }}
                    />
                    <span className="process-name">{proc.instance_id}</span>
                    <span className={`status-badge ${proc.status}`}>
                      {proc.status === 'completed' ? Icons.check : Icons.warning}
                      {proc.status}
                    </span>
                  </div>
                </div>
                <div className="process-footer">
                  <span className="process-pid">PID: {proc.pid}</span>
                  {proc.source && (
                    <span className={`source-badge ${proc.source}`}>{proc.source}</span>
                  )}
                  {proc.command && (
                    <span className="process-command" title={proc.full_cmd}>{proc.command}</span>
                  )}
                  <span className="process-duration">{formatDuration(proc.duration_sec)}</span>
                  {proc.cost !== undefined && proc.cost > 0 && (
                    <span className="process-cost">{formatCost(proc.cost)}</span>
                  )}
                </div>
              </div>
            ))}
          </>
        )}
      </div>

      <style>{`
        .monitor {
          display: flex;
          flex-direction: column;
          height: 100%;
          background: var(--bg-base);
        }

        /* Summary Bar */
        .monitor-summary {
          display: flex;
          align-items: center;
          gap: var(--space-6);
          padding: var(--space-4) var(--space-6);
          background: var(--bg-surface);
          border-bottom: 1px solid var(--border-subtle);
        }

        .summary-item {
          display: flex;
          align-items: center;
          gap: var(--space-2);
        }

        .summary-item.warning {
          color: var(--color-warning);
        }

        .summary-icon {
          color: var(--text-tertiary);
          display: flex;
        }

        .summary-item.warning .summary-icon {
          color: var(--color-warning);
        }

        .summary-value {
          font-size: var(--text-lg);
          font-weight: var(--font-semibold);
          font-family: var(--font-mono);
          color: var(--text-primary);
        }

        .summary-label {
          font-size: var(--text-xs);
          color: var(--text-tertiary);
          display: flex;
          align-items: center;
          gap: 4px;
        }

        .summary-item.live .summary-value {
          color: var(--color-primary);
        }

        .trend-up {
          color: var(--color-success);
          font-size: 10px;
          margin-left: 4px;
          animation: trend-flash 0.5s ease-out;
        }

        @keyframes trend-flash {
          0% { opacity: 0; transform: scale(1.5); }
          100% { opacity: 1; transform: scale(1); }
        }

        .live-indicator {
          color: var(--color-primary);
          font-size: 8px;
          animation: live-blink 1s ease-in-out infinite;
        }

        @keyframes live-blink {
          0%, 100% { opacity: 1; }
          50% { opacity: 0.3; }
        }

        .live-badge-summary {
          display: inline-block;
          font-size: 9px;
          font-weight: var(--font-bold);
          color: var(--color-primary);
          background: rgba(37, 194, 160, 0.15);
          padding: 2px 6px;
          border-radius: var(--radius-sm);
          margin-right: var(--space-2);
          animation: live-blink 1s ease-in-out infinite;
        }

        .summary-spacer {
          flex: 1;
        }

        .summary-update {
          font-size: var(--text-xs);
          color: var(--text-tertiary);
          display: flex;
          align-items: center;
        }

        /* Process Grid */
        .process-grid {
          flex: 1;
          overflow-y: auto;
          padding: var(--space-6);
          display: grid;
          grid-template-columns: repeat(auto-fill, minmax(320px, 1fr));
          gap: var(--space-4);
          align-content: start;
        }

        .error-card {
          grid-column: 1 / -1;
          display: flex;
          align-items: center;
          gap: var(--space-3);
          padding: var(--space-4);
          background: rgba(248, 81, 73, 0.1);
          border: 1px solid var(--color-danger);
          border-radius: var(--radius-md);
          color: var(--color-danger);
        }

        .error-icon {
          display: flex;
        }

        .empty-state {
          grid-column: 1 / -1;
          display: flex;
          flex-direction: column;
          align-items: center;
          justify-content: center;
          padding: var(--space-12);
          text-align: center;
          color: var(--text-tertiary);
        }

        .empty-icon {
          display: flex;
          align-items: center;
          justify-content: center;
          width: 64px;
          height: 64px;
          background: var(--bg-elevated);
          border-radius: var(--radius-lg);
          margin-bottom: var(--space-4);
        }

        .empty-icon svg {
          width: 32px;
          height: 32px;
        }

        .empty-state h3 {
          font-size: var(--text-lg);
          font-weight: var(--font-semibold);
          color: var(--text-primary);
          margin-bottom: var(--space-2);
        }

        .empty-state p {
          font-size: var(--text-sm);
          color: var(--text-tertiary);
        }

        /* Process Card */
        .process-card {
          background: var(--bg-surface);
          border: 1px solid var(--border-subtle);
          border-radius: var(--radius-lg);
          padding: var(--space-4);
          transition: all var(--transition-fast);
        }

        .process-card:hover {
          border-color: var(--border-default);
          box-shadow: var(--shadow-md);
        }

        .process-card.warning {
          border-color: var(--color-warning);
          background: rgba(210, 153, 34, 0.05);
        }

        .process-card.live {
          border-color: var(--color-primary);
        }

        .process-header {
          display: flex;
          align-items: center;
          justify-content: space-between;
          margin-bottom: var(--space-4);
        }

        .process-status {
          display: flex;
          align-items: center;
          gap: var(--space-2);
        }

        .status-dot {
          width: 8px;
          height: 8px;
          border-radius: var(--radius-full);
          animation: pulse 2s ease-in-out infinite;
        }

        @keyframes pulse {
          0%, 100% { opacity: 1; }
          50% { opacity: 0.5; }
        }

        .process-name {
          font-size: var(--text-sm);
          font-weight: var(--font-semibold);
          color: var(--text-primary);
          font-family: var(--font-mono);
        }

        .live-badge {
          font-size: var(--text-xs);
          font-weight: var(--font-semibold);
          color: var(--color-primary);
          background: rgba(37, 194, 160, 0.15);
          padding: 2px 6px;
          border-radius: var(--radius-sm);
          animation: live-pulse 1.5s ease-in-out infinite;
        }

        @keyframes live-pulse {
          0%, 100% { opacity: 1; }
          50% { opacity: 0.6; }
        }

        .stop-btn {
          display: flex;
          align-items: center;
          justify-content: center;
          width: 28px;
          height: 28px;
          background: transparent;
          color: var(--text-tertiary);
          border: 1px solid var(--border-default);
          border-radius: var(--radius-md);
          cursor: pointer;
          transition: all var(--transition-fast);
        }

        .stop-btn:hover {
          background: var(--color-danger);
          color: white;
          border-color: var(--color-danger);
        }

        .status-badge {
          display: flex;
          align-items: center;
          gap: var(--space-1);
          font-size: var(--text-xs);
          padding: var(--space-1) var(--space-2);
          border-radius: var(--radius-sm);
        }

        .status-badge.completed {
          background: rgba(37, 194, 160, 0.1);
          color: var(--color-primary);
        }

        /* Metrics */
        .process-metrics {
          display: flex;
          gap: var(--space-4);
          margin-bottom: var(--space-4);
        }

        .metric {
          display: flex;
          flex-direction: column;
          align-items: center;
          flex: 1;
          padding: var(--space-2);
          background: var(--bg-base);
          border-radius: var(--radius-md);
        }

        .metric-icon {
          color: var(--text-tertiary);
          margin-bottom: var(--space-1);
        }

        .metric-value {
          font-size: var(--text-base);
          font-weight: var(--font-semibold);
          font-family: var(--font-mono);
          color: var(--text-primary);
        }

        .metric-value.high {
          color: var(--color-warning);
        }

        .metric-label {
          font-size: var(--text-xs);
          color: var(--text-tertiary);
        }

        /* Telemetry Row */
        .process-telemetry {
          display: flex;
          gap: var(--space-3);
          padding: var(--space-3);
          background: rgba(37, 194, 160, 0.05);
          border-radius: var(--radius-md);
          margin-bottom: var(--space-4);
        }

        .telemetry-item {
          display: flex;
          flex-direction: column;
          align-items: center;
          flex: 1;
        }

        .telemetry-icon {
          color: var(--color-primary);
          margin-bottom: var(--space-1);
          opacity: 0.7;
        }

        .telemetry-value {
          font-size: var(--text-sm);
          font-weight: var(--font-semibold);
          font-family: var(--font-mono);
          color: var(--text-primary);
        }

        .telemetry-value.cost {
          color: var(--color-primary);
        }

        .telemetry-label {
          font-size: var(--text-xs);
          color: var(--text-tertiary);
        }

        /* Footer */
        .process-footer {
          display: flex;
          align-items: center;
          gap: var(--space-3);
          padding-top: var(--space-3);
          border-top: 1px solid var(--border-subtle);
        }

        .process-pid {
          font-size: var(--text-xs);
          font-family: var(--font-mono);
          color: var(--text-tertiary);
        }

        .process-turns {
          font-size: var(--text-xs);
          color: var(--text-secondary);
        }

        .process-cost {
          font-size: var(--text-xs);
          font-family: var(--font-mono);
          color: var(--color-primary);
          margin-left: auto;
        }

        /* Source Badge */
        .source-badge {
          font-size: var(--text-xs);
          font-weight: var(--font-medium);
          padding: 2px 6px;
          border-radius: var(--radius-sm);
          text-transform: uppercase;
        }

        .source-badge.ui {
          background: rgba(59, 130, 246, 0.15);
          color: #3b82f6;
        }

        .source-badge.eval {
          background: rgba(168, 85, 247, 0.15);
          color: #a855f7;
        }

        .source-badge.cli {
          background: rgba(100, 116, 139, 0.15);
          color: var(--text-secondary);
        }

        .source-badge.agent {
          background: rgba(37, 194, 160, 0.15);
          color: var(--color-primary);
        }

        .process-command {
          font-size: var(--text-xs);
          font-family: var(--font-mono);
          color: var(--text-secondary);
          max-width: 150px;
          overflow: hidden;
          text-overflow: ellipsis;
          white-space: nowrap;
        }

        .process-duration {
          font-size: var(--text-xs);
          font-family: var(--font-mono);
          color: var(--text-tertiary);
        }

        /* History Section */
        .history-divider {
          grid-column: 1 / -1;
          display: flex;
          align-items: center;
          gap: var(--space-3);
          margin: var(--space-4) 0;
        }

        .history-divider::before,
        .history-divider::after {
          content: '';
          flex: 1;
          height: 1px;
          background: var(--border-subtle);
        }

        .history-divider span {
          font-size: var(--text-xs);
          color: var(--text-tertiary);
          text-transform: uppercase;
          letter-spacing: 0.05em;
        }

        .process-card.history {
          opacity: 0.7;
          background: var(--bg-base);
        }

        .process-card.history:hover {
          opacity: 1;
        }

        .process-card.history.failed {
          border-color: var(--color-danger);
          background: rgba(248, 81, 73, 0.05);
        }

        .status-badge.failed {
          background: rgba(248, 81, 73, 0.1);
          color: var(--color-danger);
        }

        /* Responsive */
        @media (max-width: 768px) {
          .monitor-summary {
            flex-wrap: wrap;
            gap: var(--space-3);
          }

          .process-grid {
            padding: var(--space-4);
            grid-template-columns: 1fr;
          }
        }
      `}</style>
    </div>
  );
};
