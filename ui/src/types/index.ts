// Type definitions for the UI Collaboration Hub

export interface Thread {
  id: string;
  title: string;
  created_at: string | number;
  created_by_type: string;
  created_by_id: string;
  status: 'active' | 'paused' | 'resolved' | 'archived';
  context_json?: string;
  target_agent?: string;  // Which agent this conversation is with
  workspace?: string;     // Working directory for this thread (persisted)
  last_seq: number;
  updated_at: string | number;
}

export interface Message {
  id: string;
  thread_id: string;
  message_seq: number;
  created_at: string | number;
  from_type: string;
  from_id: string;
  to_type: string;
  to_id: string;
  kind: 'directive' | 'question' | 'proposal' | 'status' | 'result' | 'approval_request';
  subject?: string;
  content: string;
  metadata_json?: string;
  delivery_state: 'pending' | 'visible' | 'acked';
  business_state: 'open' | 'resolved' | 'archived';
}

export interface Approval {
  id: string;
  thread_id: string;
  thread_title?: string;  // Title of the associated thread
  instance_id: string;
  created_at: string | number;
  effect_delta_json: string;
  proposal: string;
  impact: 'low' | 'medium' | 'high';
  estimated_cost: number;
  status: 'pending' | 'approved' | 'rejected' | 'modified';
  reviewed_by?: string;
  reviewed_at?: string | number;
  review_notes?: string;
  capability_token?: string;
  token_expires_at?: number;
  // Multi-channel approval workflow fields (M-DASHBOARD-APPROVAL-INTEGRATION)
  iteration?: number;        // Current iteration (1-3), retrigger count
  channel?: string;          // Source: 'dashboard' | 'cli' | 'github'
  feedback?: string;         // Harvested feedback from GitHub comments
  feedback_author?: string;  // Author of the most recent feedback
}

export interface EffectDelta {
  cap_type: string;
  paths: string[];
  budget_delta: number;
}

// WebSocket Event Types
export type EventType =
  | 'subscribe'
  | 'ack'
  | 'message'
  | 'batch'
  | 'error'
  | 'ping'
  | 'pong'
  | 'thread_state';

export interface WSEvent {
  type: EventType;
  timestamp: number;
  data?: any;
}

export interface SubscribeEvent {
  thread_id: string;
  from_seq: number;
}

export interface AckEvent {
  thread_id: string;
  ack_seq: number;
}

export interface MessageEvent {
  id: string;
  thread_id: string;
  message_seq: number;
  created_at: string | number;
  from_type: string;
  from_id: string;
  to_type: string;
  to_id: string;
  kind: string;
  subject?: string;
  content: string;
  metadata_json?: string;
}

export interface BatchEvent {
  thread_id: string;
  messages: MessageEvent[];
  has_more: boolean;
}

export interface ErrorEvent {
  code: string;
  message: string;
}

export interface ThreadStateEvent {
  thread_id: string;
  status: string;
  last_seq: number;
  updated_at: string | number;
}

// Hierarchy Types for Agent/Thread Tree View

export interface Badge {
  type: 'unread' | 'pending' | 'running';
  count: number;
}

export interface HierarchyNode {
  type: 'root' | 'agent' | 'thread';
  id: string;
  label: string;
  status?: 'active' | 'idle' | 'pending';
  badges?: Badge[];
  children?: HierarchyNode[];
}

export interface ThreadStats {
  id: string;
  title: string;
  unread_count: number;
  pending_approvals: number;
  running_processes: number;
  last_message_at?: string;
}

export interface AgentStats {
  agent_id: string;
  status: 'active' | 'idle' | 'pending';
  thread_count: number;
  unread_messages: number;
  pending_approvals: number;
  running_processes: number;
  last_activity?: string;
  threads?: ThreadStats[];
}

export interface ExecutionStats {
  total_executions: number;
  successful_executions: number;
  failed_executions: number;
  total_duration_ms: number;
  total_cost: number;
  total_input_tokens: number;
  total_output_tokens: number;
  total_cache_read_tokens: number;
  total_cache_create_tokens: number;
  total_files_created: number;
}

// Per-message execution metadata (extracted from metadata_json)
export interface ExecutionMetadata {
  success: boolean;
  duration_ms: number;
  num_turns: number;
  cost: number;
  session_id: string;
  input_tokens: number;
  output_tokens: number;
  cache_read_tokens: number;
  cache_creation_tokens: number;
  files_created_count: number;
  files_created: string[];
  workspace: string;
}

export interface AggregateStats {
  total_agents: number;
  active_agents: number;
  idle_agents: number;
  pending_approvals: number;
  running_processes: number;
  total_threads: number;
  execution: ExecutionStats;
}

export interface HierarchyResponse {
  root: HierarchyNode;
  aggregate: AggregateStats;
}

// Selection state for hierarchy navigation
export type SelectionType = 'overview' | 'agent' | 'thread' | 'task' | 'observatory' | 'controlplane';

export interface Selection {
  type: SelectionType;
  agentId?: string;
  threadId?: string;
  taskId?: string;
}

// Aggregated metrics from /api/metrics endpoint
export interface AggregatedMetrics {
  scope_type: 'global' | 'agent' | 'thread';
  scope_id: string;
  total_runs: number;
  total_tokens: number;
  total_cost: number;
  total_duration_ms: number;
  total_files_modified: number;
  avg_tokens_per_run: number;
  avg_cost_per_run: number;
  avg_duration_per_run: number;
  pending_tasks: number; // Number of currently running/pending tasks
}

// Metrics trend data point
export interface MetricsTrendPoint {
  period_start: number;
  runs: number;
  tokens: number;
  cost: number;
  duration_ms: number;
}

// Approval history entry
export interface ApprovalHistoryEntry {
  id: string;
  approval_id: string;
  thread_id: string;
  agent_id: string;
  action: 'created' | 'approved' | 'rejected' | 'expired';
  actor: string;
  proposal?: string;
  impact?: string;
  estimated_cost?: number;
  capability_token?: string;
  created_at: number;
}

// Instance history entry
export interface InstanceHistoryEntry {
  id: string;
  agent_id: string;
  instance_id: string;
  started_at: number;
  ended_at?: number;
  exit_code?: number;
  total_tokens: number;
  total_cost_cents: number;
  thread_count: number;
}

// Task Stream Events (for coordinator executor feedback loop)
// Must match Go's TaskStreamEventType in internal/websocket/events.go
export type TaskStreamEventType =
  | 'turn_start'    // Turn started
  | 'text'          // Text output from agent
  | 'tool_use'      // Tool invocation
  | 'tool_result'   // Tool result
  | 'turn_end'      // Turn ended
  | 'error'         // Error event
  | 'status';       // Status change with metrics

export interface TaskStreamEvent {
  type: 'task_stream';
  task_id: string;
  thread_id?: string;
  stream_type: TaskStreamEventType; // Maps to Go's StreamType
  turn_num?: number;         // Turn number
  timestamp?: number;
  text?: string;             // Text content from agent
  tool_name?: string;
  tool_input?: string;
  tool_output?: string;
  cost?: number;
  tokens_in?: number;
  tokens_out?: number;
  duration_sec?: number;
  status?: string;           // running, completed, failed
  error_msg?: string;
}

export interface TaskResourceMetrics {
  task_id: string;
  cpu_percent: number;
  memory_mb: number;
  tokens_in: number;
  tokens_out: number;
  cost: number;
  peak_cpu: number;
  peak_memory: number;
  updated_at: number;
}

export interface PendingApprovalRequest {
  id: string;
  task_id: string;
  type: 'merge' | 'destroy' | 'execute' | 'cost';
  description: string;
  context_json?: string;
  status: 'pending' | 'approved' | 'rejected' | 'timeout';
  created_at: string;
  timeout_at?: string;
  files_changed?: string[];
  diff_summary?: string;
}
