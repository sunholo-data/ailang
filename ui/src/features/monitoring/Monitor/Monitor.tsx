import React, { useState, useEffect, useCallback } from 'react';
import { Icons, getStatusColor } from '../../../components/common/Icons';
import { formatDuration, formatCost } from '../../../utils/formatters';
import './Monitor.css';

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
    </div>
  );
};
