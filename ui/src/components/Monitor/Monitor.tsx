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
  summary: MonitorSummary;
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

export const Monitor: React.FC = () => {
  const [data, setData] = useState<MonitorResponse | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [lastUpdate, setLastUpdate] = useState<Date | null>(null);

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
        <div className="summary-item">
          <span className="summary-icon">{Icons.dollar}</span>
          <span className="summary-value">{formatCost(data?.summary.total_cost || 0)}</span>
          <span className="summary-label">Cost</span>
        </div>
        {(data?.summary.warning_count || 0) > 0 && (
          <div className="summary-item warning">
            <span className="summary-icon">{Icons.warning}</span>
            <span className="summary-value">{data?.summary.warning_count}</span>
            <span className="summary-label">Warnings</span>
          </div>
        )}
        <div className="summary-spacer" />
        <div className="summary-update">
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

        {data?.processes.map((proc) => (
          <div
            key={proc.instance_id}
            className={`process-card ${isWarning(proc) ? 'warning' : ''}`}
          >
            <div className="process-header">
              <div className="process-status">
                <span
                  className="status-dot"
                  style={{ background: getStatusColor(proc.status) }}
                />
                <span className="process-name">{proc.instance_id}</span>
              </div>
              {proc.status === 'running' && (
                <button
                  className="stop-btn"
                  onClick={() => handleStopProcess(proc.instance_id)}
                  title="Stop process"
                >
                  {Icons.stop}
                </button>
              )}
              {proc.status === 'completed' && (
                <span className="status-badge completed">{Icons.check} Done</span>
              )}
            </div>

            <div className="process-metrics">
              <div className="metric">
                <span className="metric-icon">{Icons.cpu}</span>
                <span className={`metric-value ${proc.cpu_percent > 80 ? 'high' : ''}`}>
                  {proc.cpu_percent.toFixed(1)}%
                </span>
                <span className="metric-label">CPU</span>
              </div>
              <div className="metric">
                <span className="metric-icon">{Icons.memory}</span>
                <span className="metric-value">{proc.memory_mb.toFixed(0)} MB</span>
                <span className="metric-label">Memory</span>
              </div>
              <div className="metric">
                <span className="metric-icon">{Icons.clock}</span>
                <span className={`metric-value ${proc.duration_sec > 300 ? 'high' : ''}`}>
                  {formatDuration(proc.duration_sec)}
                </span>
                <span className="metric-label">Duration</span>
              </div>
            </div>

            <div className="process-footer">
              <span className="process-pid">PID: {proc.pid}</span>
              {proc.turns && (
                <span className="process-turns">{proc.turns} turns</span>
              )}
              {proc.cost !== undefined && proc.cost > 0 && (
                <span className="process-cost">{formatCost(proc.cost)}</span>
              )}
            </div>
          </div>
        ))}
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
        }

        .summary-spacer {
          flex: 1;
        }

        .summary-update {
          font-size: var(--text-xs);
          color: var(--text-tertiary);
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
