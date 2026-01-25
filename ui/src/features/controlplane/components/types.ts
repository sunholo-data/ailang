/**
 * Shared types for Control Plane components
 */

// Agent representation in topology
export interface Agent {
  id: string;
  label: string;
  status: 'idle' | 'busy' | 'blocked' | 'error';
  trustScore: number;
  taskCount: number;
  cost: number;
}

// Event message for the queue
export interface EventMessage {
  id: string;
  timestamp: string;
  type: 'task_start' | 'task_complete' | 'task_error' | 'handoff' | 'approval' | 'message' | 'session';
  source: string;
  target?: string;
  content: string;
  // Task context fields
  directive?: string;         // Initial user prompt (truncated preview)
  directive_full?: string;    // Full directive (for detail views)
  workspace?: string;         // Working directory
  agent_id?: string;          // Agent identifier
  // Metrics (for Claude Code sessions)
  turn_count?: number;
  cost_usd?: number;
  tokens_in?: number;
  tokens_out?: number;
  duration_ms?: number;
  metrics_summary?: string;   // "3 turns • $0.42 • 12.5s"
  metadata?: Record<string, unknown>;
}

// Date range selection
export interface DateRange {
  start: string;
  end: string;
}

// Detail panel state
export interface DetailPanelState {
  type: 'agent' | 'trace' | 'event' | 'date' | null;
  id: string | null;
  data?: unknown;
}

// Heatmap cell data
export interface HeatmapCell {
  date: string;
  taskCount: number;
  cost: number;
  successRate: number;
}

// Span for trace waterfall
export interface Span {
  id: string;
  name: string;
  display_name?: string; // Enriched name with tool metadata (file paths, patterns, commands)
  startMs: number;
  durationMs: number;
  children?: Span[];
  status?: 'ok' | 'error';
  attributes?: Record<string, string>;
}

// Topology edge
export interface TopologyEdge {
  source: string;
  target: string;
  messageCount: number;
  active: boolean;
}

// Trust capability
export interface TrustCapability {
  name: string;
  score: number;
  icon: string;
}

// Aggregation stats
export interface AggregationStats {
  totalTasks: number;
  totalCost: number;
  activeAgents: number;
  pendingApprovals: number;
  workspaces?: Record<string, number>;
}
