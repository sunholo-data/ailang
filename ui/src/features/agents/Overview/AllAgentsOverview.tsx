import React from 'react';
import { AggregateStats, HierarchyNode, Selection } from '../../../types';
import { MetricsCard } from '../../../components/metrics';
import { TrendsChart } from '../../../components/metrics';
import { getAgentLabel } from '../../../utils/displayNames';
import './AllAgentsOverview.css';

interface AllAgentsOverviewProps {
  aggregate: AggregateStats;
  agents: HierarchyNode[];
  onSelectAgent: (agentId: string) => void;
}

interface StatCardProps {
  title: string;
  value: number | string;
  color?: 'default' | 'green' | 'orange' | 'blue' | 'purple';
  small?: boolean;
}

const StatCard: React.FC<StatCardProps> = ({ title, value, color = 'default', small }) => (
  <div className={`stat-card stat-${color} ${small ? 'stat-small' : ''}`}>
    <div className="stat-value">{value}</div>
    <div className="stat-title">{title}</div>
  </div>
);

// Helper to format duration
const formatDuration = (ms: number): string => {
  if (ms < 1000) return `${ms}ms`;
  const seconds = ms / 1000;
  if (seconds < 60) return `${seconds.toFixed(1)}s`;
  const minutes = Math.floor(seconds / 60);
  const remainingSeconds = (seconds % 60).toFixed(0);
  return `${minutes}m ${remainingSeconds}s`;
};

// Helper to format cost
const formatCost = (cost: number): string => {
  if (cost < 0.01) return `$${cost.toFixed(4)}`;
  return `$${cost.toFixed(2)}`;
};

// Helper to format token count
const formatTokens = (tokens: number): string => {
  if (tokens >= 1000000) return `${(tokens / 1000000).toFixed(1)}M`;
  if (tokens >= 1000) return `${(tokens / 1000).toFixed(1)}k`;
  return tokens.toString();
};

interface AgentStatusCardProps {
  agent: HierarchyNode;
  onClick: () => void;
}

const AgentStatusCard: React.FC<AgentStatusCardProps> = ({ agent, onClick }) => {
  const threadCount = agent.children?.length || 0;
  const pendingCount = agent.badges?.find(b => b.type === 'pending')?.count || 0;
  const runningCount = agent.badges?.find(b => b.type === 'running')?.count || 0;

  const statusColors: Record<string, string> = {
    active: '#22c55e',
    running: '#22c55e',  // Same as active - green
    pending: '#f59e0b',
    idle: '#6b7280',
    error: '#ef4444',    // Red for error state
  };

  return (
    <div className="agent-card" onClick={onClick}>
      <div className="agent-card-header">
        <span
          className="agent-status-dot"
          style={{ backgroundColor: statusColors[agent.status || 'idle'] }}
        />
        <span className="agent-name">{getAgentLabel(agent.id)}</span>
      </div>
      <div className="agent-card-stats">
        <span className="agent-stat">
          <span className="agent-stat-value">{threadCount}</span>
          <span className="agent-stat-label">threads</span>
        </span>
        {pendingCount > 0 && (
          <span className="agent-stat pending">
            <span className="agent-stat-value">{pendingCount}</span>
            <span className="agent-stat-label">pending</span>
          </span>
        )}
        {runningCount > 0 && (
          <span className="agent-stat running">
            <span className="agent-stat-value">{runningCount}</span>
            <span className="agent-stat-label">running</span>
          </span>
        )}
      </div>
    </div>
  );
};

export const AllAgentsOverview: React.FC<AllAgentsOverviewProps> = ({
  aggregate,
  agents,
  onSelectAgent,
}) => {
  const exec = aggregate.execution;
  const hasExecStats = exec && exec.total_executions > 0;
  const successRate = hasExecStats
    ? Math.round((exec.successful_executions / exec.total_executions) * 100)
    : 0;

  return (
    <div className="all-agents-overview">
      <div className="overview-header">
        <h2>All Agents Overview</h2>
      </div>

      <div className="stats-row">
        <StatCard title="Total Agents" value={aggregate.total_agents} />
        <StatCard title="Active" value={aggregate.active_agents} color="green" />
        <StatCard title="Pending Approvals" value={aggregate.pending_approvals} color="orange" />
        <StatCard title="Total Threads" value={aggregate.total_threads} color="blue" />
      </div>

      <div className="metrics-section">
        <h3>Usage Metrics (Today)</h3>
        <MetricsCard scopeType="global" title="Global Metrics" />
      </div>

      <div className="trends-section">
        <h3>Usage Trends (Last 24 Hours)</h3>
        <div className="trends-grid">
          <TrendsChart
            scopeType="global"
            scopeId=""
            period="hour"
            limit={24}
            metric="cost"
            title="Cost"
          />
          <TrendsChart
            scopeType="global"
            scopeId=""
            period="hour"
            limit={24}
            metric="tokens"
            title="Tokens"
          />
        </div>
      </div>

      {hasExecStats && (
        <div className="execution-stats-section">
          <h3>Execution Statistics</h3>
          <div className="stats-row">
            <StatCard
              title="Total Executions"
              value={exec.total_executions}
              color="purple"
            />
            <StatCard
              title="Success Rate"
              value={`${successRate}%`}
              color="green"
            />
            <StatCard
              title="Total Duration"
              value={formatDuration(exec.total_duration_ms)}
            />
            <StatCard
              title="Total Cost"
              value={formatCost(exec.total_cost)}
              color="orange"
            />
          </div>
          <div className="stats-row token-stats">
            <StatCard
              title="Input Tokens"
              value={formatTokens(exec.total_input_tokens)}
              small
            />
            <StatCard
              title="Output Tokens"
              value={formatTokens(exec.total_output_tokens)}
              small
            />
            <StatCard
              title="Cache Read"
              value={formatTokens(exec.total_cache_read_tokens)}
              small
            />
            <StatCard
              title="Cache Created"
              value={formatTokens(exec.total_cache_create_tokens)}
              small
            />
            <StatCard
              title="Files Created"
              value={exec.total_files_created}
              small
            />
          </div>
        </div>
      )}

      <div className="agents-section">
        <h3>Agents</h3>
        <div className="agent-cards-grid">
          {agents.map(agent => (
            <AgentStatusCard
              key={agent.id}
              agent={agent}
              onClick={() => onSelectAgent(agent.id)}
            />
          ))}
          {agents.length === 0 && (
            <div className="no-agents">
              No agents found. Start an agent to see it here.
            </div>
          )}
        </div>
      </div>
    </div>
  );
};

export default AllAgentsOverview;
