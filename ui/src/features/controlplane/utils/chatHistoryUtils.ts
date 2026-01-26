/**
 * ChatHistory utility functions
 * Extracted from ChatHistory.tsx (PR 4 - M-DASHBOARD-SIMPLIFICATION)
 *
 * Contains:
 * - Event consolidation functions
 * - Turn grouping functions
 * - Node/span lookup functions
 * - Formatting utilities
 */

import type { HierarchyNode, Span } from '../components/ExecHierarchy/types';

// ============================================================================
// Types (local to utilities, re-exported from ChatHistory)
// ============================================================================

export interface TaskEvent {
  id: string;
  task_id: string;
  stream_type: string;
  turn_num: number;
  text: string;
  tool_name?: string;
  tool_input?: string;
  tool_output?: string;
  error_msg?: string;
  tokens_in?: number;
  tokens_out?: number;
  cost?: number;
  created_at: string;
}

export interface Turn {
  turnNumber: number;
  events: TaskEvent[];
  startTime?: string;
}

export interface HierarchyTurn {
  turnNumber: number;
  node: HierarchyNode;
  toolUses: HierarchyNode[];
}

// ============================================================================
// Event Consolidation
// ============================================================================

/**
 * Consolidate consecutive text events into single message blocks.
 * Streaming text comes as many small chunks - we merge them for display.
 */
export function consolidateTextEvents(events: TaskEvent[]): TaskEvent[] {
  const result: TaskEvent[] = [];
  let currentTextBlock: TaskEvent | null = null;

  for (const event of events) {
    if (event.stream_type === 'text' && event.text) {
      if (currentTextBlock) {
        // Append to existing text block
        currentTextBlock = {
          ...currentTextBlock,
          text: currentTextBlock.text + event.text,
        };
      } else {
        // Start new text block
        currentTextBlock = { ...event };
      }
    } else {
      // Non-text event: flush any pending text block
      if (currentTextBlock) {
        result.push(currentTextBlock);
        currentTextBlock = null;
      }
      result.push(event);
    }
  }

  // Flush final text block
  if (currentTextBlock) {
    result.push(currentTextBlock);
  }

  return result;
}

// ============================================================================
// Turn Grouping
// ============================================================================

/**
 * Group events by turn number (for coordinator task events)
 */
export function groupEventsByTurn(events: TaskEvent[]): Turn[] {
  const turnMap = new Map<number, TaskEvent[]>();

  for (const event of events) {
    const turnNum = event.turn_num || 0;
    if (!turnMap.has(turnNum)) {
      turnMap.set(turnNum, []);
    }
    turnMap.get(turnNum)!.push(event);
  }

  const turns: Turn[] = [];
  const sortedKeys = Array.from(turnMap.keys()).sort((a, b) => a - b);

  for (const turnNum of sortedKeys) {
    const turnEvents = turnMap.get(turnNum)!;
    // Sort by time, then consolidate consecutive text events
    const sortedEvents = turnEvents.sort(
      (a, b) => new Date(a.created_at).getTime() - new Date(b.created_at).getTime()
    );
    const consolidatedEvents = consolidateTextEvents(sortedEvents);

    turns.push({
      turnNumber: turnNum,
      events: consolidatedEvents,
      startTime: turnEvents[0]?.created_at,
    });
  }

  return turns;
}

/**
 * Extract turns and tool uses from hierarchy node children (for OTEL spans)
 * Searches recursively since turns may be nested (e.g., coordinator → claude.execute → exec.turn)
 */
export function extractHierarchyTurns(node: HierarchyNode): HierarchyTurn[] {
  const turns: HierarchyTurn[] = [];

  // Recursively collect all turn nodes from the hierarchy
  const collectTurns = (n: HierarchyNode) => {
    if (!n.children) return;

    for (const child of n.children) {
      if (child.type === 'turn') {
        const toolUses = child.children?.filter(c => c.type === 'tool_use') || [];
        turns.push({
          turnNumber: child.turnNumber || turns.length + 1,
          node: child,
          toolUses,
        });
      } else {
        // Recursively search in non-turn children (e.g., exec nodes like claude.execute)
        collectTurns(child);
      }
    }
  };

  collectTurns(node);

  // Sort turns by turnNumber
  turns.sort((a, b) => a.turnNumber - b.turnNumber);

  // If no explicit turns found, treat all tool_use children as a single implicit turn
  if (turns.length === 0 && node.children) {
    const collectToolUses = (n: HierarchyNode): HierarchyNode[] => {
      const tools: HierarchyNode[] = [];
      if (!n.children) return tools;
      for (const child of n.children) {
        if (child.type === 'tool_use') {
          tools.push(child);
        } else {
          tools.push(...collectToolUses(child));
        }
      }
      return tools;
    };

    const toolUses = collectToolUses(node);
    if (toolUses.length > 0) {
      turns.push({
        turnNumber: 1,
        node: node,
        toolUses,
      });
    }
  }

  return turns;
}

// ============================================================================
// Node/Span Lookup
// ============================================================================

/**
 * Find node by ID in hierarchy
 */
export function findNodeById(nodes: HierarchyNode[], id: string): HierarchyNode | null {
  for (const node of nodes) {
    if (node.id === id) return node;
    if (node.children) {
      const found = findNodeById(node.children, id);
      if (found) return found;
    }
  }
  return null;
}

/**
 * Find span by ID in spans array (recursive)
 */
export function findSpanById(spans: Span[], id: string): Span | null {
  for (const span of spans) {
    if (span.id === id) return span;
    if (span.children) {
      const found = findSpanById(span.children, id);
      if (found) return found;
    }
  }
  return null;
}

// ============================================================================
// Formatting Utilities
// ============================================================================

/**
 * Format JSON for display - parses and pretty-prints if valid JSON
 */
export function formatJsonString(input: string | undefined): string {
  if (!input) return '';

  try {
    const parsed = JSON.parse(input);
    return JSON.stringify(parsed, null, 2);
  } catch {
    return input;
  }
}

/**
 * Format timestamp for display
 */
export function formatTime(timestamp: string | number | undefined): string {
  if (!timestamp) return '—';
  try {
    if (typeof timestamp === 'number') {
      return new Date(timestamp).toLocaleTimeString();
    }
    return new Date(timestamp).toLocaleTimeString();
  } catch {
    return String(timestamp);
  }
}

/**
 * Format duration for display
 */
export function formatChatDuration(ms: number | undefined): string {
  if (!ms) return '—';
  if (ms < 1000) return `${ms}ms`;
  if (ms < 60000) return `${(ms / 1000).toFixed(1)}s`;
  return `${(ms / 60000).toFixed(1)}m`;
}

// ============================================================================
// ID Extraction Helpers
// ============================================================================

/**
 * Extract task ID from node, span, or raw ID
 */
export function extractTaskIdFromContext(
  selectedNode: HierarchyNode | null,
  selectedSpan: Span | null,
  selectedNodeId: string | null | undefined
): string | undefined {
  if (selectedNode?.taskId) return selectedNode.taskId;

  // Check span attributes - different executors use different attribute names!
  if (selectedSpan?.attributes) {
    const attrs = selectedSpan.attributes;
    // Coordinator uses "task.id" (DOT notation)
    if (attrs['task.id']) return attrs['task.id'];
    // Gemini executor uses "exec.task_id"
    if (attrs['exec.task_id']) return attrs['exec.task_id'];
    // Fallback: underscore notation (legacy)
    if (attrs['task_id']) return attrs['task_id'];
  }

  // If selectedNodeId looks like a task_id, use it directly
  if (selectedNodeId && selectedNodeId.startsWith('task-')) return selectedNodeId;
  return undefined;
}

/**
 * Extract Claude session ID from span/node or raw ID
 */
export function extractClaudeSessionId(
  selectedSpan: Span | null,
  selectedNode: HierarchyNode | null,
  selectedNodeId: string | null | undefined
): string | undefined {
  // First, check if selectedNodeId itself looks like a UUID (session.id format)
  // Event Queue often passes the session UUID as the selectedNodeId
  if (
    selectedNodeId &&
    /^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$/i.test(selectedNodeId)
  ) {
    return selectedNodeId;
  }

  if (selectedSpan?.attributes) {
    const attrs = selectedSpan.attributes;
    // Claude Code uses "session.id"
    if (attrs['session.id']) return attrs['session.id'];
  }
  // Also check node's span attributes
  if (selectedNode?._span?.attributes) {
    const attrs = selectedNode._span.attributes;
    if (attrs['session.id']) return attrs['session.id'];
  }
  return undefined;
}

/**
 * Find selected node in hierarchy with fallback logic
 */
export function findSelectedNode(
  nodes: HierarchyNode[],
  selectedNodeId: string | null | undefined
): HierarchyNode | null {
  if (!selectedNodeId) return null;

  // Try exact match first
  const exactMatch = findNodeById(nodes, selectedNodeId);
  if (exactMatch) return exactMatch;

  // If selectedNodeId looks like a task_id but nodes use span_ids,
  // check if any node's taskId matches
  const nodeWithTaskId = nodes.find(
    n => n.taskId === selectedNodeId || n._span?.attributes?.['task_id'] === selectedNodeId
  );
  if (nodeWithTaskId) return nodeWithTaskId;

  // Final fallback: use first root node if we have nodes but ID doesn't match
  if (nodes.length > 0) return nodes[0];

  return null;
}

/**
 * Find selected span with fallback logic
 */
export function findSelectedSpan(
  spans: Span[] | undefined,
  selectedNodeId: string | null | undefined
): Span | null {
  if (!selectedNodeId || !spans) return null;

  // Try exact match first
  const exactMatch = findSpanById(spans, selectedNodeId);
  if (exactMatch) return exactMatch;

  // If no match but we have spans, use first root span
  if (spans.length > 0) return spans[0];

  return null;
}
