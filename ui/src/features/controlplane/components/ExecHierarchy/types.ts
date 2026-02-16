/**
 * Types for the Exec Hierarchy component
 * 4-level hierarchy: Messages -> Execs -> Turns -> Tool Uses
 */
import type React from 'react';

// Node types in the hierarchy
export type HierarchyNodeType = 'message' | 'exec' | 'turn' | 'tool_use' | 'approval';

// View mode toggle
export type ViewMode = 'tree' | 'graph' | 'timeline' | 'chat' | 'evolution';

// Coordinator task view mode
export type CoordinatorViewMode = 'nested' | 'breakout';

// Status for all node types
export type NodeStatus = 'idle' | 'busy' | 'error' | 'completed' | 'pending' | 'unknown';

// Base node interface
export interface HierarchyNode {
  id: string;
  type: HierarchyNodeType;
  label: string;
  status: NodeStatus;
  startTime?: string;
  durationMs?: number;
  children?: HierarchyNode[];
  // Optional fields for enhanced display
  turnNumber?: number;      // For turn nodes - sequential turn number
  _span?: Span;             // Original span data for popover display
  // Collapsibility (Milestone 2)
  isCollapsible?: boolean;  // Has children that can be collapsed
  isExpanded?: boolean;     // Current expanded state (controlled by parent)
  childCount?: number;      // Total descendant count for collapsed badge
  // Metrics (Milestone 4)
  cost?: number;            // Cost in USD
  tokensIn?: number;        // Input tokens
  tokensOut?: number;       // Output tokens
  provider?: 'claude' | 'gemini' | 'ollama' | string;  // AI provider
  semanticType?: string;    // Extended type for styling (coordinator, executor, ailang)
  // Coordinator context (Milestone 5)
  taskId?: string;          // Coordinator task ID
  parentTaskId?: string;    // Parent coordinator task ID (for nesting)
  approvalStatus?: 'pending' | 'approved' | 'rejected' | 'none';  // Approval workflow status
  approvalId?: string;      // Approval request ID
  agentId?: string;         // Agent that executed this task
  // Filtering (Milestone 10)
  isFiltered?: boolean;     // True if node doesn't match current filter criteria (greyed out)
  // Custom attributes (for nodes without _span, e.g., shared tool aggregations from Evolution Tree)
  attributes?: Record<string, unknown>;
}

// Message node (Level 1) - triggers coordinator tasks
export interface MessageHierarchyNode extends HierarchyNode {
  type: 'message';
  messageId: string;
  fromAgent: string;
  toInbox: string;
  messageType: string;
  title: string;
}

// Exec node (Level 2) - ailang exec/run/check commands
export interface ExecHierarchyNode extends HierarchyNode {
  type: 'exec';
  taskId: string;
  parentTaskId?: string;
  provider?: string; // claude, gemini, etc.
  workspace?: string;
  filePath?: string;
  command: 'exec' | 'run' | 'check';
}

// Turn node (Level 3) - conversation turns within an exec
export interface TurnHierarchyNode extends HierarchyNode {
  type: 'turn';
  turnNumber: number;
}

// Tool use node (Level 4) - tool invocations within turns
export interface ToolUseHierarchyNode extends HierarchyNode {
  type: 'tool_use';
  toolName: string;
  toolInput?: string;
  toolOutput?: string;
}

// Union type for all hierarchy nodes
export type HierarchyNodeUnion =
  | MessageHierarchyNode
  | ExecHierarchyNode
  | TurnHierarchyNode
  | ToolUseHierarchyNode;

// API response types (matches Go backend)
export interface MessageNode {
  message_id: string;
  title: string;
  from_agent: string;
  to_inbox: string;
  message_type: string;
  status: string;
  created_at?: string;
  execs?: ExecTaskNode[];
}

export interface ExecTaskNode {
  task_id: string;
  parent_task_id: string;
  command: string;      // exec, run, check, turn, tool_use
  provider: string;     // claude, gemini, etc.
  workspace: string;
  file_path: string;
  status: string;
  start_time?: string;
  duration_ms?: number;
  children?: ExecTaskNode[];
  // Turn/tool specific
  turn_number?: number;
  tool_name?: string;
  tool_input?: string;
  tool_output?: string;
  display_name?: string; // enriched name from session_tools (e.g., "Read: /path/file.go")
}

export interface ExecHierarchyWithMessages {
  messages?: MessageNode[];
  orphan?: ExecTaskNode[];
  count: number;
}

// Legacy response (backward compatible)
export interface ExecHierarchyResponse {
  hierarchy: ExecTaskNode[];
  count: number;
}

// Chat context embedded in spans (when include_chat=true on API)
// Contains conversation content for spans with chat history.
export interface ChatContext {
  user_prompt?: string;        // First 500 chars of user prompt
  assistant_response?: string; // First 500 chars of assistant response
  has_thinking?: boolean;      // True if thinking blocks are present
  turn_number?: number;        // Conversation turn number
  full_chat_url?: string;      // Link to full conversation endpoint
}

// Span type from shared types (for same data as TraceWaterfall)
export interface Span {
  id: string;
  name: string;
  display_name?: string;  // Enriched name with tool metadata (file paths, patterns, commands)
  startMs: number;
  durationMs: number;
  children?: Span[];
  status?: 'ok' | 'error';
  attributes?: Record<string, string>;
  chat_context?: ChatContext;  // Embedded chat content (populated with include_chat=true)
}

// Filter criteria for greying out (not hiding) nodes
export interface FilterCriteria {
  dateRange?: { start: string; end: string } | null;  // ISO date strings (YYYY-MM-DD)
  eventTypes?: string[];  // If empty, show all
  provider?: string;      // Filter by AI provider (claude, gemini, openai, etc.)
  model?: string;         // Filter by specific model name
  workspace?: string;     // Filter by workspace/project
  source_type?: string;   // Filter by source type (eval, coordinator, direct_api, etc.)
}

// ControlPlaneFilters type (for CLI hint)
export interface ControlPlaneFilters {
  source_type?: string;
  provider?: string;
  model?: string;
  workspace?: string;
  start_date?: string;
  end_date?: string;
  status?: string;
  search?: string;
}

// Props for the main ExecHierarchy component
export interface ExecHierarchyProps {
  isExpanded: boolean;
  onToggleExpand: () => void;
  onNodeClick?: (node: HierarchyNode) => void;
  selectedNodeId?: string | null;
  isEmpty?: boolean;
  // Theme from parent (syncs with app-level theme toggle)
  theme?: 'dark' | 'light';
  // Same data source as TraceWaterfall
  spans?: Span[];
  loading?: boolean;
  // Filter criteria - filtered nodes are greyed out, not hidden
  filterCriteria?: FilterCriteria;
  // Span type filtering (Milestone 14) - generic filter for any span type
  // hiddenSpanTypes: Set of span names to hide (e.g., 'api_request')
  // If provided, uses parent state; otherwise uses internal state with ['api_request'] default
  hiddenSpanTypes?: Set<string>;
  onToggleSpanType?: (spanType: string) => void;
  // Filters for CLI hint
  filters?: ControlPlaneFilters;
  // Span highlighting for outliers (click from EventDetail outliers list)
  highlightedSpanId?: string | null;
  onClearHighlight?: () => void;
  // Chain data (M-CHAIN-DASHBOARD) - structured chain+stages for chain-aware views
  // When present, views render stages as top-level groups; when null, falls back to raw spans
  chainData?: ChainData | null;
}

// ============================================================================
// Cross-Task Hierarchy Types (M-EXEC-HIERARCHY-REFACTOR)
// For visualizing task relationships across the system
// ============================================================================

// Span node nested within a task (from observatory.db)
export interface TaskSpanNode {
  id: string;
  name: string;
  node_type: 'coordinator' | 'executor' | 'turn' | 'tool' | 'other';
  duration_ms: number;
  tokens_in?: number;
  tokens_out?: number;
  cost_usd?: number;
  turn_number?: number;
  tool_name?: string;
  status: string;
  children?: TaskSpanNode[];
}

// Task node for cross-task visualization
export interface TaskHierarchyNode {
  id: string;
  title: string;
  agent_id?: string;
  parent_task_id?: string;      // Links to parent task (creates handoff edge)
  session_id?: string;          // Links tasks with shared context
  status: string;
  approval_status?: 'pending' | 'approved' | 'rejected' | '';
  approval_type?: string;       // "merge", "merge_handoff", etc.
  iteration?: number;
  cost: number;
  tokens_in: number;
  tokens_out: number;
  turns?: number;
  duration_ms: number;
  created_at: string;
  provider?: string;
  workspace?: string;
  children?: TaskHierarchyNode[];  // Child tasks (via parent_task_id)
  // Execution spans nested within this task
  spans?: TaskSpanNode[];
  // Turn-grouped hierarchy (when group_by=turns is requested)
  turn_grouped?: TurnGroupedHierarchy;
}

// ============================================================================
// Turn-Based Grouping Types (from API group_by=turns)
// Structures spans by conversation turns for intuitive visualization
// ============================================================================

// Turn-grouped hierarchy response from API
export interface TurnGroupedHierarchy {
  session?: TurnGroupSession;
  turns: TurnGroup[];
  stats: TurnGroupStats;
}

// Top-level session/executor span
export interface TurnGroupSession {
  id: string;
  name: string;
  duration_ms: number;
  cost: number;
  tokens_in: number;
  tokens_out: number;
  provider?: string;
  model?: string;
}

// Single conversation turn with its tools
export interface TurnGroup {
  turn_number: number;
  span_id: string;
  duration_ms: number;
  cost: number;
  tokens_in: number;
  tokens_out: number;
  tools?: TurnTool[];
}

// Tool call within a turn
export interface TurnTool {
  id: string;
  name: string;
  tool_name?: string;  // Extracted tool name (e.g., "Read", "Bash")
  duration_ms: number;
  cost?: number;
  status: string;
}

// Aggregate statistics for turn-grouped view
export interface TurnGroupStats {
  total_turns: number;
  total_tools: number;
  total_cost: number;
  total_tokens: number;
  duration_ms: number;
}

// Edge between tasks
export interface TaskHierarchyEdge {
  source: string;
  target: string;
  type: 'handoff' | 'session';  // handoff = parent_task_id, session = shared session_id
}

// Stats for the cross-task hierarchy
export interface TaskHierarchyStats {
  total_tasks: number;
  total_spans: number;
  pending_approvals: number;
  total_cost: number;
}

// API response from /api/controlplane/task-hierarchy
export interface TaskHierarchyResult {
  tasks: TaskHierarchyNode[];
  edges: TaskHierarchyEdge[];
  stats: TaskHierarchyStats;
}

// Graph node for ReactFlow visualization
export interface TaskGraphNode {
  id: string;
  type: 'task';
  position: { x: number; y: number };
  data: TaskHierarchyNode;
}

// Graph edge for ReactFlow visualization
export interface TaskGraphEdge {
  id: string;
  source: string;
  target: string;
  type: 'handoff' | 'session';
  animated?: boolean;
  style?: React.CSSProperties;
}

// ============================================================================
// Chain Data Types (M-CHAIN-DASHBOARD: mirrors observatory.ExecutionChain)
// Used by useChainData hook to fetch chain context for selected events
// ============================================================================

// Execution chain - top-level workflow from trigger to completion
export interface ChainData {
  id: string;
  source_type: string;       // 'github_issue' | 'message' | 'manual' | 'eval_suite'
  source_ref?: string;
  github_repo?: string;
  github_issue_number?: number;
  status: 'active' | 'pending_approval' | 'completed' | 'failed';
  current_stage: number;
  workspace_id?: string;
  workspace_path?: string;
  created_at: string;
  updated_at?: string;
  completed_at?: string;
  total_cost: number;
  total_tokens: number;
  total_turns: number;
  stages_completed: number;
  stages?: ChainStageData[];
}

// Single agent execution within a chain
export interface ChainStageData {
  id: string;
  chain_id: string;
  stage_number: number;
  agent_id: string;
  provider?: string;
  message_id?: string;
  task_id?: string;
  session_id?: string;
  status: string;             // 'pending' | 'running' | 'awaiting_approval' | 'completed' | 'failed'
  approval_status?: string;   // 'pending' | 'approved' | 'rejected'
  approval_type?: string;     // 'merge' | 'handoff' | 'merge_handoff'
  handoff_to?: string;
  iteration: number;
  human_feedback?: string;
  started_at?: string;
  completed_at?: string;
  cost: number;
  tokens_in: number;
  tokens_out: number;
  turns: number;
  tool_calls: number;
  duration_ms: number;
  error_message?: string;
  error_count?: number;
  eval_assessment?: EvalAssessmentData;
  spans?: Span[];             // Populated when include_spans=true
}

// Evaluation results for agent benchmark stages
export interface EvalAssessmentData {
  benchmark_id: string;
  model: string;
  language: string;
  condition?: string;
  eval_mode: string;
  executor?: string;
  compile_ok: boolean;
  runtime_ok: boolean;
  stdout_ok: boolean;
  error_category: string;
  first_attempt_ok?: boolean;
  repair_used?: boolean;
  repair_ok?: boolean;
}
