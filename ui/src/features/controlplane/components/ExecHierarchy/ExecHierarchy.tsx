/**
 * ExecHierarchy - Main container for the 4-level exec hierarchy view
 * Provides toggle between tree and graph views
 * Shows: Messages -> Execs -> Turns -> Tool Uses
 *
 * Uses the SAME spans data as TraceWaterfall for consistency.
 */
import React, { useState, useCallback, useMemo, useEffect, useRef } from 'react';
import type { HierarchyNode, ViewMode, ExecHierarchyProps, Span, NodeStatus, CoordinatorViewMode, FilterCriteria, ControlPlaneFilters, ExecHierarchyNode } from './types';
import { ExecHierarchyTree } from './ExecHierarchyTree';
import { ExecHierarchyGraph } from './ExecHierarchyGraph';
import { ChatHistory } from './ChatHistory';
import { CliCommandHint } from '../CliCommandHint';
import { TraceWaterfall } from '../TraceWaterfall';
import { useObservatoryWs, Approval, Span as ObsSpan } from '../../../../hooks/useObservatory';
import styles from './ExecHierarchy.module.css';

// Popover max dimensions - compact design
const POPOVER_MAX_WIDTH = 420;
const POPOVER_MAX_HEIGHT = 480;
const POPOVER_MARGIN = 16;

// Extended node types for better visualization
type ExtendedNodeType = HierarchyNode['type'] | 'coordinator' | 'executor' | 'ailang';

// Determine node type from span name with semantic patterns
function getNodeType(name: string): HierarchyNode['type'] {
  // Approval spans (M-DASHBOARD-APPROVAL-INTEGRATION)
  if (name === 'approval.decision' || name === 'human.approval' || name === 'human.feedback') return 'approval';
  // Claude Code session (virtual root)
  if (name === 'claude_code.session') return 'exec';
  // Claude Code api_request (turn in a session)
  if (name === 'api_request') return 'turn';
  // Claude Code tool calls
  if (name.startsWith('claude_code.tool.')) return 'tool_use';
  // Coordinator task execution
  if (name === 'coordinator.task.execute') return 'exec';
  // Executor spans (Claude/Gemini)
  if (name === 'claude.execute' || name === 'gemini.execute') return 'exec';
  // Turn spans
  if (name.startsWith('exec.turn') || name.includes('.turn')) return 'turn';
  // Tool use spans
  if (name === 'exec.tool_use' || name.includes('tool')) return 'tool_use';
  // Eval event spans (Milestone 12: spans created alongside inbox messages)
  if (name.startsWith('eval.event.')) return 'message';
  // Message spans
  if (name.includes('message') || name.includes('msg')) return 'message';
  // Default to exec for ailang.*, compile.*, etc.
  return 'exec';
}

// Get semantic type for enhanced display (used for labels/icons)
function getSemanticType(name: string): ExtendedNodeType {
  if (name === 'coordinator.task.execute') return 'coordinator';
  if (name === 'claude.execute' || name === 'gemini.execute') return 'executor';
  if (name.startsWith('ailang.') || name.startsWith('compile.') || name.startsWith('eval.')) return 'ailang';
  return getNodeType(name);
}

// Extract smart label from span name and attributes
function getSmartLabel(span: Span): string {
  // Prefer backend-enriched display_name if available (from /api/observatory/spans/enriched)
  // This includes tool metadata like file paths, commands, patterns from Claude Code hooks
  if (span.display_name) {
    return span.display_name;
  }

  const name = span.name;
  const attrs = span.attributes || {};

  // Claude Code session: show session summary
  if (name === 'claude_code.session') {
    // Session has aggregated metrics
    const cost = span.cost_usd || 0;
    const tokensIn = span.tokens_in || 0;
    const tokensOut = span.tokens_out || 0;
    const durationMs = span.durationMs || 0;
    const durationMins = Math.round(durationMs / 60000);
    const children = (span as any).children?.length || 0;
    return `Claude Code Session (${children} turns, $${cost.toFixed(2)}, ${durationMins}m)`;
  }

  // Approval decision spans (M-DASHBOARD-APPROVAL-INTEGRATION)
  if (name === 'approval.decision') {
    const action = attrs['approval.action'] || attrs['action'];
    const by = attrs['approval.by'] || attrs['approved.by'] || attrs['rejected.by'] || 'user';
    const channel = attrs['approval.channel'] || '';
    const channelSuffix = channel ? ` via ${channel}` : '';
    if (action === 'approve') {
      return `✓ Approved by ${by}${channelSuffix}`;
    } else if (action === 'reject') {
      return `✗ Rejected by ${by}${channelSuffix}`;
    }
    return `Approval Decision by ${by}`;
  }

  // Human approval spans
  if (name === 'human.approval') {
    const by = attrs['approved.by'] || 'user';
    return `✓ Approved by ${by}`;
  }

  // Human feedback spans
  if (name === 'human.feedback') {
    const action = attrs['feedback.action'] || '';
    const by = attrs['feedback.user'] || 'user';
    if (action === 'reject') {
      return `✗ Feedback from ${by}`;
    }
    return `Feedback from ${by}`;
  }

  // Coordinator task: use task.title or extract from directive
  if (name === 'coordinator.task.execute') {
    const title = attrs['task.title'] || attrs['directive'];
    if (title) {
      return title.length > 40 ? title.substring(0, 40) + '...' : title;
    }
    return 'Coordinator Task';
  }

  // Claude/Gemini executor: show provider + directive prefix
  if (name === 'claude.execute') {
    const directive = attrs['directive'] || attrs['task.directive'] || '';
    if (directive) {
      const prefix = directive.length > 35 ? directive.substring(0, 35) + '...' : directive;
      return `Claude: ${prefix}`;
    }
    return 'Claude Execute';
  }
  if (name === 'gemini.execute') {
    const directive = attrs['directive'] || attrs['task.directive'] || '';
    if (directive) {
      const prefix = directive.length > 35 ? directive.substring(0, 35) + '...' : directive;
      return `Gemini: ${prefix}`;
    }
    return 'Gemini Execute';
  }

  // Turn: use turn.number
  if (name.startsWith('exec.turn') || name.includes('.turn')) {
    const turnNum = attrs['turn.number'] || attrs['exec.turn'] || attrs['turn_number'];
    if (turnNum) return `Turn ${turnNum}`;
    return name.replace('exec.', '');
  }

  // Tool use: show tool name and brief input
  if (name === 'exec.tool_use') {
    const toolName = attrs['tool.name'] || attrs['tool_name'] || 'Tool';
    const input = attrs['tool.input'] || attrs['input'] || '';
    if (input && typeof input === 'string') {
      // Extract first meaningful part of input
      const brief = input.split('\n')[0].substring(0, 30);
      return `${toolName}: ${brief}${input.length > 30 ? '...' : ''}`;
    }
    return toolName;
  }

  // Eval event spans (Milestone 12): show the event title from attributes
  if (name.startsWith('eval.event.')) {
    const eventTitle = attrs['event.title'];
    if (eventTitle) {
      return eventTitle.length > 45 ? eventTitle.substring(0, 45) + '...' : eventTitle;
    }
    // Fallback: clean up the event type
    const eventType = name.replace('eval.event.', '');
    return `Eval Event: ${eventType.replace(/_/g, ' ')}`;
  }

  // Message send operations: show destination inbox and category
  if (name === 'messages.send') {
    const toInbox = attrs['message.to_inbox'] || '';
    const category = attrs['message.category'] || '';
    const fromAgent = attrs['message.from_agent'] || '';
    if (toInbox && category) {
      return `Send → ${toInbox} (${category})`;
    }
    if (toInbox) {
      return `Send → ${toInbox}`;
    }
    if (fromAgent) {
      return `Send from ${fromAgent}`;
    }
    return 'Send Message';
  }

  // Claude Code tool calls: extract tool name and context
  if (name.startsWith('claude_code.tool.')) {
    const toolName = name.replace('claude_code.tool.', '');

    // Try individual attributes first (Claude Code sends these directly)
    // File-based tools: Read, Write, Edit, Glob
    const filePath = attrs['file_path'] || attrs['path'];
    if (filePath && typeof filePath === 'string') {
      const fileName = filePath.split('/').pop() || filePath;
      return `${toolName}: ${fileName}`;
    }

    // Bash tool: show command or description
    const command = attrs['command'] || attrs['bash_command'];
    if (command && typeof command === 'string') {
      const brief = command.length > 35 ? command.substring(0, 35) + '...' : command;
      return `${toolName}: ${brief}`;
    }

    const description = attrs['description'];
    if (description && typeof description === 'string') {
      const brief = description.length > 35 ? description.substring(0, 35) + '...' : description;
      return `${toolName}: ${brief}`;
    }

    // Grep/Search tools: show pattern or query
    const pattern = attrs['pattern'] || attrs['query'] || attrs['search'];
    if (pattern && typeof pattern === 'string') {
      const brief = pattern.length > 30 ? pattern.substring(0, 30) + '...' : pattern;
      return `${toolName}: ${brief}`;
    }

    // WebFetch: show URL hostname
    const url = attrs['url'];
    if (url && typeof url === 'string') {
      try {
        const hostname = new URL(url).hostname;
        return `${toolName}: ${hostname}`;
      } catch {
        const brief = url.length > 30 ? url.substring(0, 30) + '...' : url;
        return `${toolName}: ${brief}`;
      }
    }

    // Edit tool: show what was changed
    const oldString = attrs['old_string'];
    if (oldString && typeof oldString === 'string') {
      const brief = oldString.split('\n')[0].substring(0, 25);
      return `${toolName}: "${brief}..."`;
    }

    // Fallback: try tool_parameters JSON (legacy support)
    const params = attrs['tool_parameters'] || '';
    if (params && typeof params === 'string') {
      try {
        const parsed = JSON.parse(params);
        if (parsed.file_path) {
          const path = parsed.file_path.split('/').pop() || parsed.file_path;
          return `${toolName}: ${path}`;
        }
        if (parsed.description) {
          const desc = parsed.description;
          return `${toolName}: ${desc.length > 35 ? desc.substring(0, 35) + '...' : desc}`;
        }
        if (parsed.bash_command || parsed.command) {
          const cmd = parsed.bash_command || parsed.command;
          return `${toolName}: ${cmd.length > 30 ? cmd.substring(0, 30) + '...' : cmd}`;
        }
      } catch {
        // Ignore JSON parse errors
      }
    }

    return toolName;
  }

  // API requests (Claude Code turns): show model and cost
  if (name === 'api_request') {
    const model = attrs['model'] || '';
    const cost = parseFloat(attrs['cost_usd'] || '0');
    // Show shortened model name
    let modelShort = model.replace('claude-', '').replace('-20251101', '').replace('-20251001', '');
    if (modelShort.length > 15) modelShort = modelShort.substring(0, 15);
    if (model && cost > 0) {
      return `Turn (${modelShort}) $${cost < 0.01 ? cost.toFixed(4) : cost.toFixed(2)}`;
    }
    if (model) {
      return `Turn (${modelShort})`;
    }
    return 'Turn';
  }

  // AILANG operations: show clean operation name
  if (name.startsWith('ailang.')) {
    return name.replace('ailang.', '').replace(/\./g, ' → ');
  }
  if (name.startsWith('compile.')) {
    return 'Compile: ' + name.replace('compile.', '');
  }
  if (name.startsWith('eval.')) {
    return 'Eval: ' + name.replace('eval.', '');
  }

  // Other API requests: show model
  if (name.includes('generate')) {
    const model = attrs['model'] || attrs['gen_ai.request.model'] || '';
    if (model) return `API: ${model}`;
  }

  // Default: clean up the span name
  return name.replace(/\./g, ' ').replace(/_/g, ' ');
}

// Convert span status to node status
function getNodeStatus(span: Span): NodeStatus {
  if (span.status === 'error') return 'error';
  if (span.status === 'ok') return 'completed';
  // Check if still running (no end time, or endMs === startMs for in-progress)
  if (span.durationMs === 0) return 'busy';
  return 'completed';
}

// Extract turn number from span attributes or sibling index
function getTurnNumber(span: Span, siblingIndex?: number): number | undefined {
  // Try to get from span attributes
  const fromAttr = span.attributes?.['turn.number']
    || span.attributes?.['exec.turn']
    || span.attributes?.['turn_number'];

  if (fromAttr) {
    const num = parseInt(String(fromAttr), 10);
    if (!isNaN(num)) return num;
  }

  // Fall back to sibling index (1-based)
  if (siblingIndex !== undefined) return siblingIndex + 1;

  return undefined;
}

// Extract metrics from span attributes
function extractMetrics(span: Span): { cost?: number; tokensIn?: number; tokensOut?: number; provider?: string } {
  const attrs = span.attributes || {};

  // Try various attribute naming conventions
  const cost = parseFloat(attrs['cost_usd'] || attrs['cost'] || attrs['total_cost'] || '0') || undefined;
  const tokensIn = parseInt(attrs['tokens_in'] || attrs['input_tokens'] || attrs['gen_ai.usage.prompt_tokens'] || '0', 10) || undefined;
  const tokensOut = parseInt(attrs['tokens_out'] || attrs['output_tokens'] || attrs['gen_ai.usage.completion_tokens'] || '0', 10) || undefined;

  // Detect provider from span name or attributes
  let provider: string | undefined;
  if (span.name.includes('claude') || attrs['provider'] === 'claude') {
    provider = 'claude';
  } else if (span.name.includes('gemini') || attrs['provider'] === 'gemini') {
    provider = 'gemini';
  } else if (span.name.includes('ollama') || attrs['provider'] === 'ollama') {
    provider = 'ollama';
  } else if (attrs['provider']) {
    provider = attrs['provider'];
  }

  return { cost, tokensIn, tokensOut, provider };
}

// Extract coordinator context from span attributes
function extractCoordinatorContext(span: Span): {
  taskId?: string;
  parentTaskId?: string;
  agentId?: string;
  approvalStatus?: 'pending' | 'approved' | 'rejected' | 'none';
  approvalId?: string;
} {
  const attrs = span.attributes || {};

  return {
    taskId: attrs['task.id'] || attrs['task_id'] || attrs['ailang.task_id'] || undefined,
    parentTaskId: attrs['task.parent_id'] || attrs['parent_task_id'] || attrs['ailang.parent_task_id'] || undefined,
    agentId: attrs['agent.id'] || attrs['agent_id'] || undefined,
    approvalStatus: attrs['approval.status'] || attrs['approval_status'] || undefined,
    approvalId: attrs['approval.id'] || attrs['approval_id'] || undefined,
  };
}

// Check if span is a coordinator task
function isCoordinatorTask(span: Span): boolean {
  return span.name === 'coordinator.task.execute';
}

// Transform Span[] to HierarchyNode[] (recursive)
function spanToHierarchyNode(span: Span, siblingIndex?: number, depth: number = 0): HierarchyNode {
  const nodeType = getNodeType(span.name);
  const isTurn = nodeType === 'turn';
  const turnNumber = isTurn ? getTurnNumber(span, siblingIndex) : undefined;
  const semanticType = getSemanticType(span.name);

  // Use smart label extraction
  const label = getSmartLabel(span);

  // Extract metrics
  const metrics = extractMetrics(span);

  // Extract coordinator context
  const coordContext = extractCoordinatorContext(span);

  // Transform children first so we can count them
  const children = span.children?.map((child, idx) => spanToHierarchyNode(child, idx, depth + 1));
  const hasChildren = children && children.length > 0;

  const node: HierarchyNode = {
    id: span.id,
    type: nodeType,
    label,
    status: getNodeStatus(span),
    durationMs: span.durationMs,
    turnNumber,
    _span: span,  // Preserve original span for popover
    children,
    // Collapsibility
    isCollapsible: hasChildren,
    childCount: hasChildren ? countDescendants({ children } as HierarchyNode) + children.length : 0,
    // Metrics
    cost: metrics.cost,
    tokensIn: metrics.tokensIn,
    tokensOut: metrics.tokensOut,
    provider: metrics.provider,
    semanticType,
    // Coordinator context
    taskId: coordContext.taskId,
    parentTaskId: coordContext.parentTaskId,
    agentId: coordContext.agentId,
    approvalStatus: coordContext.approvalStatus,
    approvalId: coordContext.approvalId,
  };

  return node;
}

// Transform spans to hierarchy nodes
function transformSpansToNodes(spans: Span[]): HierarchyNode[] {
  return spans.map((span, idx) => spanToHierarchyNode(span, idx));
}

// Extract nested coordinator tasks for breakout view
// Returns all coordinator tasks as separate root nodes with parent links preserved
function breakoutCoordinatorTasks(nodes: HierarchyNode[]): HierarchyNode[] {
  const result: HierarchyNode[] = [];
  const coordinatorNodes: HierarchyNode[] = [];

  // Collect all coordinator nodes and their children
  const collectCoordinatorNodes = (nodeList: HierarchyNode[], parentTaskId?: string) => {
    for (const node of nodeList) {
      // Check if this is a coordinator task (semanticType or span name)
      const isCoordinator = node.semanticType === 'coordinator' ||
        node._span?.name === 'coordinator.task.execute';

      if (isCoordinator) {
        // Create a copy with parent reference preserved
        const coordNode: HierarchyNode = {
          ...node,
          parentTaskId: parentTaskId || node.parentTaskId,
        };

        // Recursively collect nested coordinators, then remove them from children
        if (node.children && node.children.length > 0) {
          const nonCoordChildren: HierarchyNode[] = [];
          for (const child of node.children) {
            const childIsCoord = child.semanticType === 'coordinator' ||
              child._span?.name === 'coordinator.task.execute';
            if (childIsCoord) {
              // This child is a coordinator - collect it separately
              collectCoordinatorNodes([child], coordNode.taskId || coordNode.id);
            } else {
              nonCoordChildren.push(child);
            }
          }
          coordNode.children = nonCoordChildren;
          coordNode.childCount = nonCoordChildren.length > 0 ?
            countDescendants(coordNode) + nonCoordChildren.length : 0;
        }

        coordinatorNodes.push(coordNode);
      } else {
        // Not a coordinator - keep in place but check children for nested coordinators
        const nodeCopy = { ...node };
        if (node.children && node.children.length > 0) {
          const nonCoordChildren: HierarchyNode[] = [];
          for (const child of node.children) {
            const childIsCoord = child.semanticType === 'coordinator' ||
              child._span?.name === 'coordinator.task.execute';
            if (childIsCoord) {
              collectCoordinatorNodes([child], parentTaskId);
            } else {
              nonCoordChildren.push(child);
            }
          }
          nodeCopy.children = nonCoordChildren;
        }
        result.push(nodeCopy);
      }
    }
  };

  collectCoordinatorNodes(nodes);

  // Combine: non-coordinator roots + all coordinator nodes as separate roots
  return [...result, ...coordinatorNodes];
}

// Format duration for display
function formatDuration(ms?: number): string {
  if (!ms) return '-';
  if (ms < 1000) return `${ms.toFixed(0)}ms`;
  if (ms < 60000) return `${(ms / 1000).toFixed(1)}s`;
  return `${(ms / 60000).toFixed(1)}m`;
}

// Get icon for node type
function getNodeIcon(type: HierarchyNode['type']): string {
  switch (type) {
    case 'message': return '✉';
    case 'exec': return '⚡';
    case 'turn': return '↻';
    case 'tool_use': return '⚙';
    case 'approval': return '👤';
    default: return '●';
  }
}

// Count total descendants recursively
function countDescendants(node: HierarchyNode): number {
  if (!node.children || node.children.length === 0) return 0;
  return node.children.reduce((sum, child) => sum + 1 + countDescendants(child), 0);
}

// Determine default expanded state based on node type and depth
function getDefaultExpandedState(node: HierarchyNode, depth: number): boolean {
  // Always expand top-level coordinator/exec nodes
  if (depth === 0) return true;
  // Expand turns (they're the main navigation units)
  if (node.type === 'turn') return true;
  // Collapse tool_use and deep nodes by default (reduce noise)
  if (node.type === 'tool_use' || depth > 2) return false;
  return true;
}

// Check if a node matches the filter criteria
function nodeMatchesFilter(node: HierarchyNode, criteria: FilterCriteria | undefined): boolean {
  if (!criteria) return true;

  // Check date range filter - convert string dates to Date objects for reliable comparison
  if (criteria.dateRange && node.startTime) {
    const nodeDate = new Date(node.startTime);
    const startDate = new Date(criteria.dateRange.start + 'T00:00:00');
    const endDate = new Date(criteria.dateRange.end + 'T23:59:59');
    if (nodeDate < startDate || nodeDate > endDate) {
      return false;
    }
  }

  // Check event type filter (map node.type to event types)
  if (criteria.eventTypes && criteria.eventTypes.length > 0) {
    // Map node types to event types
    const typeMapping: Record<string, string[]> = {
      'exec': ['task_start', 'task_complete', 'task_error'],
      'turn': ['task_start', 'task_complete'],
      'tool_use': ['task_start', 'task_complete'],
      'message': ['message', 'handoff', 'approval'],
    };
    const mappedTypes = typeMapping[node.type] || [];
    const hasMatchingType = mappedTypes.some(t => criteria.eventTypes!.includes(t));
    if (!hasMatchingType) {
      return false;
    }
  }

  // Check provider filter
  if (criteria.provider && node.provider) {
    if (node.provider !== criteria.provider) {
      return false;
    }
  }

  // Check model filter (check node.provider for model or span attributes)
  if (criteria.model) {
    // Model might be stored in span attributes or could be derived from provider
    const nodeModel = node._span?.attributes?.model || node._span?.attributes?.['llm.model'];
    if (nodeModel && nodeModel !== criteria.model) {
      return false;
    }
  }

  // Check workspace filter
  if (criteria.workspace) {
    // Workspace might be on the node directly or in span attributes
    const nodeWorkspace = (node as ExecHierarchyNode).workspace ||
                          node._span?.attributes?.workspace ||
                          node._span?.attributes?.['ailang.workspace'];
    if (nodeWorkspace && nodeWorkspace !== criteria.workspace) {
      return false;
    }
  }

  // Check source_type filter
  if (criteria.source_type) {
    const nodeSourceType = node._span?.attributes?.source_type ||
                           node._span?.attributes?.['ailang.source_type'];
    if (nodeSourceType && nodeSourceType !== criteria.source_type) {
      return false;
    }
  }

  return true;
}

// Apply filter criteria to nodes recursively (marks nodes as isFiltered, does NOT hide them)
function applyFilterToNodes(nodes: HierarchyNode[], criteria: FilterCriteria | undefined): HierarchyNode[] {
  const hasFilters = criteria && (
    criteria.dateRange ||
    (criteria.eventTypes && criteria.eventTypes.length > 0) ||
    criteria.provider ||
    criteria.model ||
    criteria.workspace ||
    criteria.source_type
  );
  if (!hasFilters) {
    // No filtering active - return nodes as-is
    return nodes;
  }

  const applyFilter = (node: HierarchyNode): HierarchyNode => {
    const isFiltered = !nodeMatchesFilter(node, criteria);
    const filteredChildren = node.children?.map(applyFilter);

    return {
      ...node,
      isFiltered,
      children: filteredChildren,
    };
  };

  return nodes.map(applyFilter);
}

export const ExecHierarchy: React.FC<ExecHierarchyProps> = ({
  isExpanded,
  onToggleExpand,
  onNodeClick,
  selectedNodeId,
  isEmpty: propsIsEmpty,
  spans,
  loading,
  filterCriteria,
  hiddenSpanTypes: propsHiddenSpanTypes,
  onToggleSpanType,
  filters,
  highlightedSpanId,
  onClearHighlight,
}) => {
  // Default to graph view (ReactFlow)
  const [viewMode, setViewMode] = useState<ViewMode>('graph');

  // Coordinator view mode: nested (default) or breakout
  const [coordViewMode, setCoordViewMode] = useState<CoordinatorViewMode>('nested');

  // Span type filtering (Milestone 14) - generic filter for any span type
  // Use props if provided (lifted state), otherwise use internal state
  const [internalHiddenTypes, setInternalHiddenTypes] = useState<Set<string>>(new Set(['api_request']));
  const hiddenSpanTypes = propsHiddenSpanTypes !== undefined ? propsHiddenSpanTypes : internalHiddenTypes;
  const toggleSpanType = onToggleSpanType || ((spanType: string) => {
    setInternalHiddenTypes(prev => {
      const next = new Set(prev);
      if (next.has(spanType)) {
        next.delete(spanType);
      } else {
        next.add(spanType);
      }
      return next;
    });
  });

  // Span type filter dropdown visibility
  const [showSpanTypeFilter, setShowSpanTypeFilter] = useState(false);

  // Collapsibility state: Set of node IDs that are expanded
  // START COLLAPSED - empty set means only root nodes visible
  const [expandedNodes, setExpandedNodes] = useState<Set<string>>(new Set());

  // Track expand/collapse changes for recentering
  const [expandChangeCounter, setExpandChangeCounter] = useState(0);

  // Display limit state - start with 100 nodes, can load more
  const DEFAULT_DISPLAY_LIMIT = 100;
  const LOAD_MORE_INCREMENT = 100;
  const [displayLimit, setDisplayLimit] = useState(DEFAULT_DISPLAY_LIMIT);

  // Reverse order toggle - show newest first when true
  const [reverseOrder, setReverseOrder] = useState(false);

  // Real-time updates via WebSocket
  const [lastUpdate, setLastUpdate] = useState<Date | null>(null);
  const { isConnected, connectionState, lastEventTime } = useObservatoryWs({
    onSpanCreated: useCallback((span: ObsSpan) => {
      // Update last event time for visual feedback
      setLastUpdate(new Date());
    }, []),
    onSpanUpdated: useCallback((span: ObsSpan) => {
      setLastUpdate(new Date());
    }, []),
  });

  // Popover state
  const [popoverNode, setPopoverNode] = useState<HierarchyNode | null>(null);
  const [popoverPos, setPopoverPos] = useState({ x: 0, y: 0 });
  const [attributesExpanded, setAttributesExpanded] = useState(false);
  const [metricsExpanded, setMetricsExpanded] = useState(false);
  const [toolDetailsExpanded, setToolDetailsExpanded] = useState(false);
  const popoverRef = useRef<HTMLDivElement>(null);

  // Transform spans to hierarchy nodes (same data as TraceWaterfall)
  const rawNodes = useMemo(() => {
    if (!spans || spans.length === 0) return [];
    return transformSpansToNodes(spans);
  }, [spans]);

  // Extract unique span types from all spans (for filter dropdown)
  const uniqueSpanTypes = useMemo(() => {
    const types = new Set<string>();
    const collect = (spanList: Span[]) => {
      for (const span of spanList) {
        if (span.name) types.add(span.name);
        if (span.children) collect(span.children);
      }
    };
    if (spans) collect(spans);
    return Array.from(types).sort();
  }, [spans]);

  // Count only hidden types that exist in current data (avoids negative counts when switching traces)
  const effectiveHiddenCount = useMemo(() => {
    return uniqueSpanTypes.filter(type => hiddenSpanTypes.has(type)).length;
  }, [uniqueSpanTypes, hiddenSpanTypes]);

  // Show All / Hide All handlers for span type filter
  const handleShowAllSpanTypes = useCallback(() => {
    uniqueSpanTypes.forEach(type => {
      if (hiddenSpanTypes.has(type)) toggleSpanType(type);
    });
  }, [uniqueSpanTypes, hiddenSpanTypes, toggleSpanType]);

  const handleHideAllSpanTypes = useCallback(() => {
    uniqueSpanTypes.forEach(type => {
      if (!hiddenSpanTypes.has(type)) toggleSpanType(type);
    });
  }, [uniqueSpanTypes, hiddenSpanTypes, toggleSpanType]);

  // Apply coordinator view mode transformation
  const transformedNodes = useMemo(() => {
    if (coordViewMode === 'breakout') {
      return breakoutCoordinatorTasks(rawNodes);
    }
    return rawNodes;
  }, [rawNodes, coordViewMode]);

  // Apply span type filtering (Milestone 14): hide spans by type, promote their children
  // When a span type is in hiddenSpanTypes, it's removed and its children are promoted to parent
  const spanFilteredNodes = useMemo(() => {
    if (hiddenSpanTypes.size === 0) return transformedNodes; // No filtering

    // Helper to recursively filter nodes, promoting children of hidden nodes
    const filterNodes = (nodeList: HierarchyNode[]): HierarchyNode[] => {
      const result: HierarchyNode[] = [];

      for (const node of nodeList) {
        const spanName = node._span?.name || '';
        const isHidden = hiddenSpanTypes.has(spanName);

        if (isHidden) {
          // This node is hidden - promote its children to this level
          if (node.children && node.children.length > 0) {
            // Recursively filter children first
            const filteredChildren = filterNodes(node.children);
            result.push(...filteredChildren);
          }
          // Node itself is not added to result
        } else {
          // This node is visible - keep it, but filter its children
          if (node.children && node.children.length > 0) {
            const filteredChildren = filterNodes(node.children);
            result.push({
              ...node,
              children: filteredChildren,
              childCount: filteredChildren.length,
              isCollapsible: filteredChildren.length > 0,
            });
          } else {
            result.push(node);
          }
        }
      }

      // Sort by start time to preserve execution order after promotion
      // When reverseOrder is true, show newest first (descending)
      result.sort((a, b) => {
        const aStart = a._span?.startMs || 0;
        const bStart = b._span?.startMs || 0;
        return reverseOrder ? bStart - aStart : aStart - bStart;
      });

      return result;
    };

    return filterNodes(transformedNodes);
  }, [transformedNodes, hiddenSpanTypes, reverseOrder]);

  // Apply filter criteria (marks nodes as isFiltered, does NOT hide them)
  const nodes = useMemo(() => {
    return applyFilterToNodes(spanFilteredNodes, filterCriteria);
  }, [spanFilteredNodes, filterCriteria]);

  // Count total nodes recursively (for display limit)
  const totalNodeCount = useMemo(() => {
    let count = 0;
    const countNodes = (nodeList: HierarchyNode[]) => {
      for (const node of nodeList) {
        count++;
        if (node.children) countNodes(node.children);
      }
    };
    countNodes(nodes);
    return count;
  }, [nodes]);

  // Apply display limit to nodes
  const limitedNodes = useMemo(() => {
    if (totalNodeCount <= displayLimit) return nodes;

    // Flatten, limit, and rebuild tree structure
    let remaining = displayLimit;
    const limitTree = (nodeList: HierarchyNode[]): HierarchyNode[] => {
      const result: HierarchyNode[] = [];
      for (const node of nodeList) {
        if (remaining <= 0) break;
        remaining--;
        const limitedChildren = node.children ? limitTree(node.children) : undefined;
        result.push({
          ...node,
          children: limitedChildren,
          childCount: limitedChildren?.length,
        });
      }
      return result;
    };
    return limitTree(nodes);
  }, [nodes, displayLimit, totalNodeCount]);

  const hasMoreNodes = totalNodeCount > displayLimit;
  const loadMoreNodes = () => setDisplayLimit(prev => prev + LOAD_MORE_INCREMENT);
  const showAllNodes = () => setDisplayLimit(totalNodeCount);

  // Handle highlighted span (from outliers click) - auto-expand path and scroll to node
  useEffect(() => {
    if (!highlightedSpanId || !spans || spans.length === 0) return;

    // Find the path to the highlighted span
    const findPathToNode = (nodeList: HierarchyNode[], targetId: string, path: string[] = []): string[] | null => {
      for (const node of nodeList) {
        // Check if this node matches the highlighted span
        if (node._span?.id === targetId || node.id === targetId) {
          return path;
        }
        if (node.children && node.children.length > 0) {
          const foundPath = findPathToNode(node.children, targetId, [...path, node.id]);
          if (foundPath) return foundPath;
        }
      }
      return null;
    };

    // Delay to ensure nodes are rendered
    const timer = setTimeout(() => {
      const pathToExpand = findPathToNode(nodes, highlightedSpanId);
      if (pathToExpand) {
        // Expand all nodes in the path
        setExpandedNodes(prev => {
          const next = new Set(prev);
          pathToExpand.forEach(id => next.add(id));
          return next;
        });
        setExpandChangeCounter(c => c + 1);

        // Scroll to the highlighted element after expansion
        setTimeout(() => {
          const element = document.querySelector(`[data-span-id="${highlightedSpanId}"]`);
          if (element) {
            element.scrollIntoView({ behavior: 'smooth', block: 'center' });
            // Add pulse animation class
            element.classList.add(styles.highlightedSpan);
            // Remove after animation
            setTimeout(() => {
              element.classList.remove(styles.highlightedSpan);
              onClearHighlight?.();
            }, 3000);
          }
        }, 100);
      }
    }, 50);

    return () => clearTimeout(timer);
  }, [highlightedSpanId, spans, nodes, onClearHighlight]);

  // Expand ONE LEVEL: expand children of currently expanded nodes (or roots if nothing expanded)
  const expandOneLevel = useCallback(() => {
    setExpandedNodes(prev => {
      const next = new Set(prev);

      // Helper to find all nodes at given IDs
      const findNode = (id: string, nodeList: HierarchyNode[]): HierarchyNode | undefined => {
        for (const node of nodeList) {
          if (node.id === id) return node;
          if (node.children) {
            const found = findNode(id, node.children);
            if (found) return found;
          }
        }
        return undefined;
      };

      // If nothing expanded, expand root nodes
      if (prev.size === 0) {
        nodes.forEach(n => {
          if (n.isCollapsible) {
            next.add(n.id);
          }
        });
        return next;
      }

      // Find children of currently expanded nodes and expand them
      const expandChildren = (nodeList: HierarchyNode[]) => {
        for (const node of nodeList) {
          if (prev.has(node.id) && node.children) {
            // This node is expanded, expand its children
            node.children.forEach(child => {
              if (child.isCollapsible) {
                next.add(child.id);
              }
            });
          }
          // Recurse to check nested expanded nodes
          if (node.children) {
            expandChildren(node.children);
          }
        }
      };

      expandChildren(nodes);
      return next;
    });
    setExpandChangeCounter(c => c + 1);
  }, [nodes]);

  // Collapse ONE LEVEL: collapse the deepest expanded nodes
  const collapseOneLevel = useCallback(() => {
    setExpandedNodes(prev => {
      if (prev.size === 0) return prev;

      // Find the depth of each expanded node
      const getDepth = (id: string, nodeList: HierarchyNode[], depth: number): number => {
        for (const node of nodeList) {
          if (node.id === id) return depth;
          if (node.children) {
            const found = getDepth(id, node.children, depth + 1);
            if (found >= 0) return found;
          }
        }
        return -1;
      };

      // Get depths of all expanded nodes
      const depths = new Map<string, number>();
      prev.forEach(id => {
        const depth = getDepth(id, nodes, 0);
        if (depth >= 0) depths.set(id, depth);
      });

      // Find max depth
      let maxDepth = -1;
      depths.forEach(d => {
        if (d > maxDepth) maxDepth = d;
      });

      // Remove nodes at max depth
      const next = new Set<string>();
      prev.forEach(id => {
        const depth = depths.get(id);
        if (depth !== undefined && depth < maxDepth) {
          next.add(id);
        }
      });

      return next;
    });
    setExpandChangeCounter(c => c + 1);
  }, [nodes]);

  // Toggle expand/collapse for a node (individual click)
  const handleToggleExpand = useCallback((nodeId: string) => {
    setExpandedNodes(prev => {
      const next = new Set(prev);
      if (next.has(nodeId)) {
        next.delete(nodeId);
      } else {
        next.add(nodeId);
      }
      return next;
    });
    setExpandChangeCounter(c => c + 1);
  }, []);

  // Expand all nodes
  const handleExpandAll = useCallback(() => {
    const allIds = new Set<string>();
    const collect = (nodeList: HierarchyNode[]) => {
      for (const node of nodeList) {
        if (node.isCollapsible) {
          allIds.add(node.id);
        }
        if (node.children) {
          collect(node.children);
        }
      }
    };
    collect(nodes);
    setExpandedNodes(allIds);
    setExpandChangeCounter(c => c + 1);
  }, [nodes]);

  // Collapse all nodes
  const handleCollapseAll = useCallback(() => {
    setExpandedNodes(new Set());
    setExpandChangeCounter(c => c + 1);
  }, []);

  const isEmpty = propsIsEmpty ?? nodes.length === 0;

  // Calculate viewport-aware position for popover
  const calculatePopoverPosition = useCallback((clickX: number, clickY: number) => {
    const viewportWidth = window.innerWidth;
    const viewportHeight = window.innerHeight;

    let x = clickX + 10;
    let y = clickY + 10;

    // Keep popover within right edge
    if (x + POPOVER_MAX_WIDTH + POPOVER_MARGIN > viewportWidth) {
      x = clickX - POPOVER_MAX_WIDTH - 10;
    }
    // Keep within left edge
    if (x < POPOVER_MARGIN) {
      x = POPOVER_MARGIN;
    }

    // Keep popover within bottom edge
    if (y + POPOVER_MAX_HEIGHT + POPOVER_MARGIN > viewportHeight) {
      y = viewportHeight - POPOVER_MAX_HEIGHT - POPOVER_MARGIN;
    }
    // Keep within top edge
    if (y < POPOVER_MARGIN) {
      y = POPOVER_MARGIN;
    }

    return { x, y };
  }, []);

  // Handle node click - show popover
  const handleNodeClick = useCallback(
    (node: HierarchyNode, event?: React.MouseEvent) => {
      onNodeClick?.(node);

      // Toggle popover - close if clicking same node
      if (popoverNode?.id === node.id) {
        setPopoverNode(null);
      } else {
        setPopoverNode(node);
        // Reset all collapsible sections to collapsed by default
        setAttributesExpanded(false);
        setMetricsExpanded(false);
        setToolDetailsExpanded(false);
        // Position with viewport awareness
        if (event) {
          const pos = calculatePopoverPosition(event.clientX, event.clientY);
          setPopoverPos(pos);
        } else {
          // Fallback to center of viewport
          setPopoverPos({ x: window.innerWidth / 2 - POPOVER_MAX_WIDTH / 2, y: 100 });
        }
      }
    },
    [onNodeClick, popoverNode, calculatePopoverPosition]
  );

  // Close popover when clicking outside
  const handleContainerClick = useCallback((e: React.MouseEvent) => {
    // Only close if clicking directly on container (not on children)
    if (e.target === e.currentTarget) {
      setPopoverNode(null);
    }
  }, []);

  return (
    <div className={`${styles.container} ${isExpanded ? styles.containerExpanded : ''}`}>
      {/* Header - Mission Control Telemetry Bar */}
      <div className={styles.header}>
        <div className={styles.headerLeft}>
          <div className={styles.headerTitle}>
            <span className={styles.headerIcon}>◎</span>
            <span className={styles.headerLabel}>Execution Spans</span>
          </div>

          {/* Span Count Readout */}
          {!isEmpty && totalNodeCount > 0 && (
            <div
              className={styles.telemetryItem}
              title={`Total execution spans in view (${nodes.length} top-level, ${totalNodeCount} total including nested)`}
            >
              <span className={styles.telemetryValue}>{totalNodeCount}</span>
              <span className={styles.telemetryLabel}>spans</span>
            </div>
          )}
        </div>

        <div className={styles.headerControls}>
          {/* Span Type Filter */}
          {uniqueSpanTypes.length > 0 && (
            <div className={styles.filterDropdown}>
              <button
                className={`${styles.filterBtn} ${effectiveHiddenCount > 0 ? styles.filterBtnActive : ''}`}
                onClick={() => setShowSpanTypeFilter(!showSpanTypeFilter)}
                title="Filter which span types to display"
              >
                <span className={styles.filterIcon}>⚙</span>
                <span className={styles.filterText}>
                  {effectiveHiddenCount > 0
                    ? `${uniqueSpanTypes.length - effectiveHiddenCount}/${uniqueSpanTypes.length}`
                    : 'Types'}
                </span>
                <span className={styles.filterChevron}>{showSpanTypeFilter ? '▴' : '▾'}</span>
              </button>

              {showSpanTypeFilter && (
                <div className={styles.filterMenu}>
                  <div className={styles.filterMenuHeader}>
                    <span className={styles.filterMenuTitle}>Span Types</span>
                    <div className={styles.filterMenuActions}>
                      <button onClick={handleShowAllSpanTypes} className={styles.filterMenuAction}>
                        All
                      </button>
                      <button onClick={handleHideAllSpanTypes} className={styles.filterMenuAction}>
                        None
                      </button>
                    </div>
                  </div>
                  <div className={styles.filterMenuList}>
                    {uniqueSpanTypes.map(spanType => (
                      <label key={spanType} className={styles.filterOption}>
                        <input
                          type="checkbox"
                          checked={!hiddenSpanTypes.has(spanType)}
                          onChange={() => toggleSpanType(spanType)}
                        />
                        <span className={styles.filterOptionCheck}>
                          {!hiddenSpanTypes.has(spanType) ? '✓' : ''}
                        </span>
                        <span className={styles.filterOptionLabel}>{spanType}</span>
                      </label>
                    ))}
                  </div>
                </div>
              )}
            </div>
          )}

          {/* Coordinator View Mode Toggle */}
          {!isEmpty && (
            <div className={styles.viewToggle}>
              <button
                className={`${styles.viewToggleBtn} ${coordViewMode === 'nested' ? styles.viewToggleBtnActive : ''}`}
                onClick={() => setCoordViewMode('nested')}
                title="Nested View (child tasks under parents)"
              >
                ⊏
              </button>
              <button
                className={`${styles.viewToggleBtn} ${coordViewMode === 'breakout' ? styles.viewToggleBtnActive : ''}`}
                onClick={() => setCoordViewMode('breakout')}
                title="Breakout View (each task as separate root)"
              >
                ⊔
              </button>
            </div>
          )}

          {/* Collapse/Expand Controls */}
          {!isEmpty && (
            <div className={styles.collapseControls}>
              <button
                className={styles.collapseBtn}
                onClick={handleCollapseAll}
                title="Collapse All"
              >
                ⊟
              </button>
              <button
                className={styles.collapseBtn}
                onClick={collapseOneLevel}
                title="Collapse One Level"
              >
                −
              </button>
              <button
                className={styles.collapseBtn}
                onClick={expandOneLevel}
                title="Expand One Level"
              >
                +
              </button>
              <button
                className={styles.collapseBtn}
                onClick={handleExpandAll}
                title="Expand All"
              >
                ⊞
              </button>
            </div>
          )}

          {/* View Toggle */}
          <div className={styles.viewToggle}>
            <button
              className={`${styles.viewToggleBtn} ${viewMode === 'tree' ? styles.viewToggleBtnActive : ''}`}
              onClick={() => setViewMode('tree')}
              title="Tree View"
            >
              ≡
            </button>
            <button
              className={`${styles.viewToggleBtn} ${viewMode === 'graph' ? styles.viewToggleBtnActive : ''}`}
              onClick={() => setViewMode('graph')}
              title="Graph View"
            >
              ⬡
            </button>
            <button
              className={`${styles.viewToggleBtn} ${viewMode === 'timeline' ? styles.viewToggleBtnActive : ''}`}
              onClick={() => setViewMode('timeline')}
              title="Timeline View"
            >
              ▥
            </button>
            <button
              className={`${styles.viewToggleBtn} ${viewMode === 'chat' ? styles.viewToggleBtnActive : ''}`}
              onClick={() => setViewMode('chat')}
              title="Chat History"
            >
              💬
            </button>
          </div>

          {/* Reverse Order Toggle */}
          <button
            className={`${styles.viewToggleBtn} ${reverseOrder ? styles.viewToggleBtnActive : ''}`}
            onClick={() => setReverseOrder(!reverseOrder)}
            title={reverseOrder ? "Show oldest first" : "Show newest first"}
          >
            {reverseOrder ? "↓" : "↑"}
          </button>

          {/* Expand Button */}
          <button
            className={styles.expandBtn}
            onClick={onToggleExpand}
            title={isExpanded ? 'Collapse' : 'Expand'}
          >
            {isExpanded ? '⤢' : '⤡'}
          </button>
        </div>
      </div>

      {/* Selected Span Metadata */}
      {popoverNode && popoverNode._span && (
        <div className={styles.selectedSpanMeta}>
          <div className={styles.metaRow}>
            <span className={styles.metaLabel}>Source</span>
            <span className={styles.metaValue}>
              {popoverNode._span.attributes?.['source'] || popoverNode.provider || 'claude-code'}
            </span>
          </div>
          <div className={styles.metaRow}>
            <span className={styles.metaLabel}>Target</span>
            <span className={styles.metaValue}>
              {popoverNode._span.attributes?.['target'] || 'user'}
            </span>
          </div>
          <div className={styles.metaRow}>
            <span className={styles.metaLabel}>Time</span>
            <span className={styles.metaValue}>
              {popoverNode._span.startMs ? new Date(popoverNode._span.startMs).toLocaleString() : '—'}
            </span>
          </div>
          <div className={styles.metaRow}>
            <span className={styles.metaLabel}>Task ID</span>
            <span className={styles.metaValue} title={popoverNode.taskId || popoverNode._span.id}>
              {(popoverNode.taskId || popoverNode._span.id || '—').slice(0, 36)}
            </span>
          </div>
          <div className={styles.metaRow}>
            <span className={styles.metaLabel}>Content</span>
            <span className={styles.metaValue}>{popoverNode.label}</span>
          </div>
          <button
            className={styles.metaClose}
            onClick={() => setPopoverNode(null)}
            title="Clear selection"
          >
            ×
          </button>
        </div>
      )}

      {/* Viewport */}
      <div
        className={`${styles.viewport} ${isExpanded ? styles.viewportExpanded : ''}`}
        onClick={handleContainerClick}
      >
        {viewMode === 'tree' && (
          <ExecHierarchyTree
            nodes={limitedNodes}
            selectedNodeId={selectedNodeId}
            onNodeClick={handleNodeClick}
            loading={loading}
            error={null}
            expandedNodeIds={expandedNodes}
            onToggleExpand={handleToggleExpand}
            highlightedSpanId={highlightedSpanId}
          />
        )}
        {viewMode === 'graph' && (
          <ExecHierarchyGraph
            nodes={limitedNodes}
            selectedNodeId={selectedNodeId}
            onNodeClick={handleNodeClick}
            loading={loading}
            error={null}
            isExpanded={isExpanded}
            expandedNodes={expandedNodes}
            onToggleNodeExpand={handleToggleExpand}
            recenterTrigger={expandChangeCounter}
          />
        )}
        {viewMode === 'timeline' && (
          <TraceWaterfall
            spans={spans || []}
            loading={loading}
            hiddenSpanTypes={hiddenSpanTypes}
            onToggleSpanType={toggleSpanType}
          />
        )}
        {viewMode === 'chat' && (
          <ChatHistory
            nodes={limitedNodes}
            selectedNodeId={selectedNodeId}
            onNodeClick={handleNodeClick}
            loading={loading}
            spans={spans}
          />
        )}

        {/* Load more buttons when nodes are limited */}
        {hasMoreNodes && (
          <div className={styles.loadMoreContainer}>
            <span className={styles.loadMoreInfo}>
              Showing {displayLimit} of {totalNodeCount} nodes
            </span>
            <button className={styles.loadMoreBtn} onClick={loadMoreNodes}>
              Load {Math.min(LOAD_MORE_INCREMENT, totalNodeCount - displayLimit)} more
            </button>
            <button className={styles.loadMoreBtn} onClick={showAllNodes}>
              Show all {totalNodeCount}
            </button>
          </div>
        )}
      </div>

      {/* Node Popover */}
      {popoverNode && (
        <div
          ref={popoverRef}
          className={styles.nodePopover}
          style={{ left: popoverPos.x, top: popoverPos.y }}
        >
          <div className={styles.popoverHeader}>
            <span className={styles.popoverIcon}>{getNodeIcon(popoverNode.type)}</span>
            <span className={styles.popoverTitle}>{popoverNode.label}</span>
            <div className={styles.popoverHeaderActions}>
              {/* Filter toggle button */}
              {popoverNode._span && (
                <button
                  className={`${styles.popoverFilterBtn} ${hiddenSpanTypes?.has(popoverNode._span.name) ? styles.popoverFilterActive : ''}`}
                  onClick={() => toggleSpanType(popoverNode._span?.name || '')}
                  title={hiddenSpanTypes?.has(popoverNode._span.name) ? 'Show this span type' : 'Hide this span type'}
                >
                  {hiddenSpanTypes?.has(popoverNode._span.name) ? '◯' : '◉'}
                </button>
              )}
              <button
                className={styles.popoverClose}
                onClick={() => setPopoverNode(null)}
                title="Close"
              >
                ×
              </button>
            </div>
          </div>
          <div className={styles.popoverBody}>
            {/* Hero Row - Duration + Status prominently displayed */}
            <div className={styles.popoverHero}>
              <div className={styles.popoverDuration}>
                <span className={styles.popoverDurationIcon}>⏱</span>
                <span className={styles.popoverDurationValue}>{formatDuration(popoverNode.durationMs)}</span>
              </div>
              <span className={`${styles.popoverStatusBadge} ${styles[`status${popoverNode.status.charAt(0).toUpperCase() + popoverNode.status.slice(1)}`]}`}>
                {popoverNode.status === 'ok' || popoverNode.status === 'complete' ? '✓' : popoverNode.status === 'error' ? '✕' : '◎'}
                {' '}{popoverNode.status}
              </span>
            </div>

            {/* Quick Info Row - Type, Provider, Turn */}
            <div className={styles.popoverQuickInfo}>
              <span className={styles.popoverQuickItem}>
                <span className={styles.popoverQuickLabel}>Type</span>
                <span className={styles.popoverQuickValue}>{popoverNode.type}</span>
              </span>
              {popoverNode.provider && (
                <span className={styles.popoverQuickItem}>
                  <span className={styles.popoverQuickLabel}>Provider</span>
                  <span className={styles.popoverQuickValue}>{popoverNode.provider}</span>
                </span>
              )}
              {popoverNode.turnNumber && (
                <span className={styles.popoverQuickItem}>
                  <span className={styles.popoverQuickLabel}>Turn</span>
                  <span className={styles.popoverQuickValue}>{popoverNode.turnNumber}</span>
                </span>
              )}
            </div>

            {/* Metrics Section - Collapsible */}
            {(popoverNode.cost || popoverNode.tokensIn || popoverNode.tokensOut) && (
              <div className={styles.popoverSection}>
                <button
                  className={styles.popoverSectionToggle}
                  onClick={() => setMetricsExpanded(!metricsExpanded)}
                >
                  <span className={styles.popoverToggleIcon}>
                    {metricsExpanded ? '▼' : '▶'}
                  </span>
                  <span className={styles.popoverSectionTitle}>
                    Metrics
                    {popoverNode.cost && popoverNode.cost > 0 && (
                      <span className={styles.popoverSectionBadge}>
                        ${popoverNode.cost < 0.01 ? popoverNode.cost.toFixed(4) : popoverNode.cost.toFixed(2)}
                      </span>
                    )}
                  </span>
                </button>
                {metricsExpanded && (
                  <div className={styles.popoverMetricsGrid}>
                    {popoverNode.cost !== undefined && popoverNode.cost > 0 && (
                      <div className={styles.popoverMetricItem}>
                        <span className={styles.popoverMetricLabel}>Cost</span>
                        <span className={styles.popoverMetricValue}>
                          ${popoverNode.cost < 0.01 ? popoverNode.cost.toFixed(4) : popoverNode.cost.toFixed(3)}
                        </span>
                      </div>
                    )}
                    {popoverNode.tokensIn !== undefined && popoverNode.tokensIn > 0 && (
                      <div className={styles.popoverMetricItem}>
                        <span className={styles.popoverMetricLabel}>Input</span>
                        <span className={styles.popoverMetricValue}>
                          {popoverNode.tokensIn >= 1000 ? `${(popoverNode.tokensIn / 1000).toFixed(1)}K` : popoverNode.tokensIn} tokens
                        </span>
                      </div>
                    )}
                    {popoverNode.tokensOut !== undefined && popoverNode.tokensOut > 0 && (
                      <div className={styles.popoverMetricItem}>
                        <span className={styles.popoverMetricLabel}>Output</span>
                        <span className={styles.popoverMetricValue}>
                          {popoverNode.tokensOut >= 1000 ? `${(popoverNode.tokensOut / 1000).toFixed(1)}K` : popoverNode.tokensOut} tokens
                        </span>
                      </div>
                    )}
                  </div>
                )}
              </div>
            )}

            {/* Agent ID - only show if present (provider already in quick info) */}
            {popoverNode.agentId && (
              <div className={styles.popoverCompactInfo}>
                <span className={styles.popoverCompactLabel}>Agent</span>
                <span className={styles.popoverCompactValue}>{popoverNode.agentId}</span>
              </div>
            )}

            {/* Coordinator Context */}
            {(popoverNode.taskId || popoverNode.parentTaskId) && (
              <div className={styles.popoverSection}>
                <div className={styles.popoverSectionTitle}>Task Context</div>
                <div className={styles.popoverInfoList}>
                  {popoverNode.taskId && (
                    <div className={styles.popoverInfoRow}>
                      <span className={styles.popoverInfoLabel}>Task ID</span>
                      <span className={styles.popoverInfoValue} style={{ fontFamily: 'var(--font-mono)' }}>
                        {popoverNode.taskId.substring(0, 16)}...
                      </span>
                    </div>
                  )}
                  {popoverNode.parentTaskId && (
                    <div className={styles.popoverInfoRow}>
                      <span className={styles.popoverInfoLabel}>Parent Task</span>
                      <span className={styles.popoverInfoValue} style={{ fontFamily: 'var(--font-mono)' }}>
                        {popoverNode.parentTaskId.substring(0, 16)}...
                      </span>
                    </div>
                  )}
                </div>
              </div>
            )}

            {/* Approval Status */}
            {popoverNode.approvalStatus && popoverNode.approvalStatus !== 'none' && (
              <div className={styles.popoverSection}>
                <div className={styles.popoverSectionTitle}>Approval</div>
                <div className={styles.popoverApproval}>
                  <span className={`${styles.popoverApprovalStatus} ${styles[`approval${popoverNode.approvalStatus.charAt(0).toUpperCase() + popoverNode.approvalStatus.slice(1)}`]}`}>
                    {popoverNode.approvalStatus === 'pending' ? '⏳' : popoverNode.approvalStatus === 'approved' ? '✓' : '✗'}
                    {' '}{popoverNode.approvalStatus}
                  </span>
                </div>
              </div>
            )}

            {popoverNode._span && (
              <>
                {/* Tool Details Section - Collapsible */}
                <div className={styles.popoverSection}>
                  <button
                    className={styles.popoverSectionToggle}
                    onClick={() => setToolDetailsExpanded(!toolDetailsExpanded)}
                  >
                    <span className={styles.popoverToggleIcon}>
                      {toolDetailsExpanded ? '▼' : '▶'}
                    </span>
                    <span className={styles.popoverSectionTitle}>
                      Tool Details
                      <span className={styles.popoverSectionBadge}>
                        {popoverNode._span.name}
                      </span>
                    </span>
                  </button>
                  {toolDetailsExpanded && (
                    <div className={styles.popoverInfoList}>
                      {/* Display Name (enriched) */}
                      <div className={styles.popoverInfoRow}>
                        <span className={styles.popoverInfoLabel}>Display Name</span>
                        <span className={styles.popoverInfoValue}>
                          {(popoverNode._span as any).display_name || popoverNode.label}
                        </span>
                      </div>
                      {/* Raw Span Name */}
                      <div className={styles.popoverInfoRow}>
                        <span className={styles.popoverInfoLabel}>Span Type</span>
                        <span className={styles.popoverInfoValue} style={{ fontFamily: 'var(--font-mono)', fontSize: '11px' }}>
                          {popoverNode._span.name}
                        </span>
                      </div>
                      {/* Timestamp */}
                      <div className={styles.popoverInfoRow}>
                        <span className={styles.popoverInfoLabel}>Start Time</span>
                        <span className={styles.popoverInfoValue}>
                          {popoverNode._span.startMs
                            ? new Date(popoverNode._span.startMs).toLocaleString()
                            : '-'}
                        </span>
                      </div>
                      {/* Tool Input - nested inside Tool Details */}
                      {(popoverNode._span as any).tool_input && (
                        <div className={styles.popoverNestedSection}>
                          <div className={styles.popoverSectionTitle}>
                            Tool Input
                            {(popoverNode._span as any).tool_success !== undefined && (
                              <span className={(popoverNode._span as any).tool_success ? styles.popoverToolSuccess : styles.popoverToolError}>
                                {(popoverNode._span as any).tool_success ? ' ✓' : ' ✗'}
                              </span>
                            )}
                          </div>
                          <pre className={styles.popoverCodeBlock}>
                            {(() => {
                              try {
                                const input = (popoverNode._span as any).tool_input;
                                const parsed = typeof input === 'string' ? JSON.parse(input) : input;
                                return JSON.stringify(parsed, null, 2);
                              } catch {
                                return String((popoverNode._span as any).tool_input);
                              }
                            })()}
                          </pre>
                        </div>
                      )}

                      {/* Tool Response - nested inside Tool Details */}
                      {(popoverNode._span as any).tool_response && (
                        <div className={styles.popoverNestedSection}>
                          <div className={styles.popoverSectionTitle}>Tool Response</div>
                          <pre className={styles.popoverCodeBlock}>
                            {(() => {
                              try {
                                const response = (popoverNode._span as any).tool_response;
                                const parsed = typeof response === 'string' ? JSON.parse(response) : response;
                                return JSON.stringify(parsed, null, 2);
                              } catch {
                                return String((popoverNode._span as any).tool_response);
                              }
                            })()}
                          </pre>
                        </div>
                      )}
                    </div>
                  )}
                </div>

                {/* Span ID */}
                <div className={styles.popoverSection}>
                  <div className={styles.popoverSectionTitle}>Span ID</div>
                  <div
                    className={styles.popoverIdValue}
                    onClick={() => {
                      navigator.clipboard.writeText(popoverNode._span?.id || '');
                    }}
                    title="Click to copy"
                    style={{ cursor: 'pointer' }}
                  >
                    {popoverNode._span.id}
                    <span className={styles.popoverCopyHint}>📋</span>
                  </div>
                </div>

                {/* CLI Command Hint */}
                <div className={styles.popoverCliHint}>
                  <div className={styles.popoverSectionTitle}>CLI Command</div>
                  <div
                    className={styles.popoverCliCommand}
                    onClick={() => {
                      const cmd = (() => {
                        const span = popoverNode._span;
                        if (!span) return '';
                        // For spans with trace_id, show trace view command
                        if (span.trace_id) {
                          return `ailang trace view ${span.trace_id}`;
                        }
                        // For spans with task_id attribute, show filtered spans
                        const taskId = span.attributes?.['task.id'] || span.attributes?.['task_id'] || span.attributes?.['ailang.task_id'];
                        if (taskId) {
                          return `ailang dashboard spans --task-id ${taskId} --enriched --json`;
                        }
                        // For spans with session.id, show session tools
                        const sessionId = span.attributes?.['session.id'];
                        if (sessionId) {
                          return `ailang dashboard tools ${sessionId} --json`;
                        }
                        // Fallback: generic dashboard spans query
                        return `ailang dashboard spans --enriched --limit 10 --json`;
                      })();
                      navigator.clipboard.writeText(cmd);
                    }}
                    title="Click to copy"
                  >
                    <code>
                      {(() => {
                        const span = popoverNode._span;
                        if (!span) return '';
                        if (span.trace_id) {
                          return `ailang trace view ${span.trace_id}`;
                        }
                        const taskId = span.attributes?.['task.id'] || span.attributes?.['task_id'] || span.attributes?.['ailang.task_id'];
                        if (taskId) {
                          return `ailang dashboard spans --task-id ${taskId.substring(0, 8)}...`;
                        }
                        const sessionId = span.attributes?.['session.id'];
                        if (sessionId) {
                          return `ailang dashboard tools ${sessionId.substring(0, 8)}...`;
                        }
                        return `ailang dashboard spans --enriched --limit 10`;
                      })()}
                    </code>
                    <span className={styles.popoverCopyHint}>📋</span>
                  </div>
                </div>

                {/* Attributes - Collapsible */}
                {popoverNode._span.attributes && Object.keys(popoverNode._span.attributes).length > 0 && (
                  <div className={styles.popoverSection}>
                    <button
                      className={styles.popoverSectionToggle}
                      onClick={() => setAttributesExpanded(!attributesExpanded)}
                    >
                      <span className={styles.popoverToggleIcon}>
                        {attributesExpanded ? '▼' : '▶'}
                      </span>
                      <span className={styles.popoverSectionTitle}>
                        Attributes ({Object.keys(popoverNode._span.attributes).length})
                      </span>
                    </button>
                    {attributesExpanded && (
                      <div className={styles.popoverAttributesList}>
                        {Object.entries(popoverNode._span.attributes).map(([key, value]) => (
                          <div key={key} className={styles.popoverAttrRow}>
                            <span className={styles.popoverAttrKey}>{key}</span>
                            <span className={styles.popoverAttrValue}>
                              {String(value).length > 100
                                ? `${String(value).substring(0, 100)}...`
                                : String(value)}
                            </span>
                          </div>
                        ))}
                      </div>
                    )}
                  </div>
                )}
              </>
            )}
          </div>
        </div>
      )}

      {/* CLI command hint */}
      <CliCommandHint
        commandType="hierarchy"
        filters={filters}
        compact
      />
    </div>
  );
};

export default ExecHierarchy;
