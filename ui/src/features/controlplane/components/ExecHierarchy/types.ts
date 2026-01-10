/**
 * Types for the Exec Hierarchy component
 * 4-level hierarchy: Messages -> Execs -> Turns -> Tool Uses
 */

// Node types in the hierarchy
export type HierarchyNodeType = 'message' | 'exec' | 'turn' | 'tool_use';

// View mode toggle
export type ViewMode = 'tree' | 'graph';

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

// Span type from shared types (for same data as TraceWaterfall)
export interface Span {
  id: string;
  name: string;
  startMs: number;
  durationMs: number;
  children?: Span[];
  status?: 'ok' | 'error';
  attributes?: Record<string, string>;
}

// Filter criteria for greying out (not hiding) nodes
export interface FilterCriteria {
  dateRange?: { start: Date; end: Date } | null;
  eventTypes?: string[];  // If empty, show all
}

// Props for the main ExecHierarchy component
export interface ExecHierarchyProps {
  isExpanded: boolean;
  onToggleExpand: () => void;
  onNodeClick?: (node: HierarchyNode) => void;
  selectedNodeId?: string | null;
  isEmpty?: boolean;
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
}
