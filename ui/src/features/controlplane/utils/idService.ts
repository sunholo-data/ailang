/**
 * ID transformation and extraction utilities
 * Centralizes scattered ID logic from ExecHierarchy.tsx
 *
 * Handles various ID formats:
 * - task-<8char> (coordinator tasks)
 * - eval-<timestamp> (eval runs)
 * - UUID format (Claude Code sessions)
 */

import type { Span } from '../components/ExecHierarchy/types';

// ============================================================================
// ID Format Detection
// ============================================================================

/**
 * Check if an ID is a coordinator task ID (task-<8char> format)
 */
export function isTaskId(id: string): boolean {
  return id.startsWith('task-');
}

/**
 * Check if an ID is an eval run ID (eval-<timestamp> format)
 */
export function isEvalId(id: string): boolean {
  return id.startsWith('eval-');
}

/**
 * Check if an ID is a UUID format
 */
export function isUuidId(id: string): boolean {
  const uuidRegex = /^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$/i;
  return uuidRegex.test(id);
}

/**
 * Check if an ID is a trace ID (32-char hex)
 */
export function isTraceId(id: string): boolean {
  return /^[0-9a-f]{32}$/i.test(id);
}

/**
 * Check if an ID is a span ID (16-char hex)
 */
export function isSpanId(id: string): boolean {
  return /^[0-9a-f]{16}$/i.test(id);
}

// ============================================================================
// ID Normalization
// ============================================================================

/**
 * Normalize an ID by stripping common prefixes
 * Returns the raw ID portion for consistent comparison
 */
export function normalizeId(id: string): string {
  if (id.startsWith('task-')) {
    return id.substring(5);
  }
  if (id.startsWith('eval-')) {
    return id.substring(5);
  }
  return id;
}

/**
 * Extract a short ID for display (first 8 characters)
 */
export function extractShortId(id: string, length: number = 8): string {
  const normalized = normalizeId(id);
  if (normalized.length <= length) return normalized;
  return normalized.substring(0, length);
}

/**
 * Format ID for display with ellipsis if too long
 */
export function formatIdForDisplay(id: string, maxLength: number = 12): string {
  if (id.length <= maxLength) return id;
  return `${id.substring(0, maxLength)}...`;
}

// ============================================================================
// ID Extraction from Spans/Attributes
// ============================================================================

/**
 * Extract task ID from span attributes
 * Checks multiple possible attribute names
 */
export function extractTaskId(attrs: Record<string, string> | undefined): string | undefined {
  if (!attrs) return undefined;
  return attrs['task.id'] || attrs['task_id'] || attrs['ailang.task_id'];
}

/**
 * Extract parent task ID from span attributes
 */
export function extractParentTaskId(attrs: Record<string, string> | undefined): string | undefined {
  if (!attrs) return undefined;
  return attrs['task.parent_id'] || attrs['parent_task_id'] || attrs['ailang.parent_task_id'];
}

/**
 * Extract session ID from span attributes
 */
export function extractSessionId(attrs: Record<string, string> | undefined): string | undefined {
  if (!attrs) return undefined;
  return attrs['session.id'] || attrs['session_id'];
}

/**
 * Extract agent ID from span attributes
 */
export function extractAgentId(attrs: Record<string, string> | undefined): string | undefined {
  if (!attrs) return undefined;
  return attrs['agent.id'] || attrs['agent_id'];
}

/**
 * Extract approval status from span attributes
 */
export function extractApprovalStatus(
  attrs: Record<string, string> | undefined
): 'pending' | 'approved' | 'rejected' | 'none' | undefined {
  if (!attrs) return undefined;
  const status = attrs['approval.status'] || attrs['approval_status'];
  if (status === 'pending' || status === 'approved' || status === 'rejected' || status === 'none') {
    return status;
  }
  return undefined;
}

/**
 * Extract approval ID from span attributes
 */
export function extractApprovalId(attrs: Record<string, string> | undefined): string | undefined {
  if (!attrs) return undefined;
  return attrs['approval.id'] || attrs['approval_id'];
}

// ============================================================================
// Coordinator Context Extraction
// ============================================================================

/**
 * Extract full coordinator context from span attributes
 */
export interface CoordinatorContext {
  taskId?: string;
  parentTaskId?: string;
  agentId?: string;
  approvalStatus?: 'pending' | 'approved' | 'rejected' | 'none';
  approvalId?: string;
}

/**
 * Extract coordinator context from span attributes
 */
export function extractCoordinatorContext(span: Span): CoordinatorContext {
  const attrs = span.attributes;
  return {
    taskId: extractTaskId(attrs),
    parentTaskId: extractParentTaskId(attrs),
    agentId: extractAgentId(attrs),
    approvalStatus: extractApprovalStatus(attrs),
    approvalId: extractApprovalId(attrs),
  };
}

/**
 * Check if span is a coordinator task
 */
export function isCoordinatorTask(span: Span): boolean {
  return span.name === 'coordinator.task.execute';
}

// ============================================================================
// ID Collection from Span Trees
// ============================================================================

/**
 * Collect all task IDs from a span tree
 * Recursively traverses children
 */
export function collectTaskIds(spans: Span[]): string[] {
  const taskIds = new Set<string>();

  const collect = (spanList: Span[]) => {
    for (const span of spanList) {
      const taskId = extractTaskId(span.attributes);
      if (taskId && taskId.startsWith('task-')) {
        taskIds.add(taskId);
      }
      if (span.children) {
        collect(span.children);
      }
    }
  };

  collect(spans);
  return Array.from(taskIds);
}

/**
 * Collect all unique span names from a span tree
 * Useful for building filter dropdown options
 */
export function collectSpanTypes(spans: Span[]): string[] {
  const types = new Set<string>();

  const collect = (spanList: Span[]) => {
    for (const span of spanList) {
      if (span.name) types.add(span.name);
      if (span.children) collect(span.children);
    }
  };

  collect(spans);
  return Array.from(types).sort((a, b) => a.localeCompare(b));
}
