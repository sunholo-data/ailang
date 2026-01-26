/**
 * Shared node styling utilities for controlplane visualizations
 * Consolidated from ExecHierarchyGraph, TaskHierarchyGraph, and buildGraphFromSpans
 */

// Color constants
export const COLORS = {
  // Status colors
  completed: '#25c2a0',
  busy: '#3b82f6',
  error: '#ef4444',
  pending: '#f59e0b',
  gray: '#64748b',

  // Node type colors
  message: '#f59e0b',
  exec: '#25c2a0',
  turn: '#3b82f6',
  toolUse: '#8b5cf6',
  approval: '#10b981',

  // Span node colors
  coordinator: '#8b5cf6',
  executor: '#3b82f6',
  tool: '#f59e0b',
} as const;

// Node types
export type HierarchyNodeType = 'message' | 'exec' | 'turn' | 'tool_use' | 'approval';
export type NodeStatus = 'completed' | 'done' | 'busy' | 'running' | 'error' | 'failed' | 'pending' | 'pending_approval';
export type SpanNodeType = 'coordinator' | 'executor' | 'turn' | 'tool';
export type ApprovalStatus = 'pending' | 'approved' | 'rejected';

/**
 * Get color for hierarchy node type (message/exec/turn/tool_use/approval)
 */
export function getNodeTypeColor(type: HierarchyNodeType | string): string {
  switch (type) {
    case 'message':
      return COLORS.message;
    case 'exec':
      return COLORS.exec;
    case 'turn':
      return COLORS.turn;
    case 'tool_use':
      return COLORS.toolUse;
    case 'approval':
      return COLORS.approval;
    default:
      return COLORS.gray;
  }
}

/**
 * Get color for node status (completed/busy/error/pending)
 */
export function getStatusColor(status: NodeStatus | string): string {
  switch (status) {
    case 'completed':
    case 'done':
      return COLORS.completed;
    case 'busy':
    case 'running':
      return COLORS.busy;
    case 'error':
    case 'failed':
      return COLORS.error;
    case 'pending':
    case 'pending_approval':
      return COLORS.pending;
    default:
      return COLORS.gray;
  }
}

/**
 * Get color for task node based on status and optional approval status
 */
export function getTaskNodeColor(status: string, approvalStatus?: string): string {
  if (approvalStatus === 'pending') return COLORS.pending;
  if (approvalStatus === 'rejected') return COLORS.error;
  return getStatusColor(status);
}

/**
 * Get color for span node type (coordinator/executor/turn/tool)
 */
export function getSpanNodeColor(nodeType?: SpanNodeType | string): string {
  switch (nodeType) {
    case 'coordinator':
      return COLORS.coordinator;
    case 'executor':
      return COLORS.executor;
    case 'turn':
      return COLORS.completed;
    case 'tool':
      return COLORS.tool;
    default:
      return COLORS.gray;
  }
}

/**
 * Get icon for hierarchy node type
 */
export function getNodeIcon(type: HierarchyNodeType | string): string {
  switch (type) {
    case 'message':
      return '✉';
    case 'exec':
      return '⚡';
    case 'turn':
      return '↻';
    case 'tool_use':
      return '⚙';
    case 'approval':
      return '👤';
    default:
      return '●';
  }
}

/**
 * Get icon for span node type
 */
export function getSpanNodeIcon(nodeType?: SpanNodeType | string): string {
  switch (nodeType) {
    case 'coordinator':
      return '\u2B21'; // Hexagon
    case 'executor':
      return '\u25CF'; // Circle
    case 'turn':
      return '\u25C9'; // Fish eye
    case 'tool':
      return '\u2699'; // Gear
    default:
      return '\u2022'; // Bullet
  }
}

/**
 * Get icon for AI provider
 */
export function getProviderIcon(provider?: string): string {
  switch (provider) {
    case 'claude':
    case 'claude-code':
      return '\u{1F7E0}'; // 🟠 Orange circle
    case 'gemini':
    case 'gemini-cli':
      return '\u{1F535}'; // 🔵 Blue circle
    case 'ollama':
      return '\u{1F7E3}'; // 🟣 Purple circle
    case 'script':
      return '\u{1F7E2}'; // 🟢 Green circle
    default:
      return '';
  }
}

/**
 * Get icon for approval status
 */
export function getApprovalIcon(status?: ApprovalStatus | string): string {
  switch (status) {
    case 'pending':
      return '\u23F3'; // ⏳ Hourglass
    case 'approved':
      return '\u2713'; // ✓ Check
    case 'rejected':
      return '\u2717'; // ✗ X
    default:
      return '';
  }
}

/**
 * Get color for approval status
 */
export function getApprovalColor(status?: ApprovalStatus | string): string {
  switch (status) {
    case 'pending':
      return COLORS.pending;
    case 'approved':
      return COLORS.completed;
    case 'rejected':
      return COLORS.error;
    default:
      return COLORS.gray;
  }
}

/**
 * Get approval badge (icon + color) for approval status
 */
export function getApprovalBadge(status?: ApprovalStatus | string): { icon: string; color: string } {
  return {
    icon: getApprovalIcon(status),
    color: getApprovalColor(status),
  };
}

/**
 * Get status icon for agent status
 */
export function getAgentStatusIcon(status: 'idle' | 'busy' | 'blocked' | 'error'): string {
  switch (status) {
    case 'idle':
      return '○';
    case 'busy':
      return '●';
    case 'blocked':
      return '◐';
    case 'error':
      return '✕';
  }
}
