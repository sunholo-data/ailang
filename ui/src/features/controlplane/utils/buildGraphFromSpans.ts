/**
 * buildGraphFromSpans - Transform spans into ReactFlow graph data
 *
 * Creates a TURN-BASED hierarchy for better visualization:
 *   Session/Executor
 *   ├── Turn 1 (sequential)
 *   │   ├── tool.Read
 *   │   └── tool.Edit
 *   ├── Turn 2 (sequential)
 *   │   └── tool.Bash
 *   └── Turn 3
 *       └── tool.Write
 *
 * Turns are connected sequentially (Turn 1 → Turn 2 → Turn 3).
 * This provides a clear temporal flow of the execution.
 */
import { Node, Edge, MarkerType } from 'reactflow';
import dagre from 'dagre';
import type { TurnGroupedHierarchy, TurnGroup, TurnTool } from '../components/ExecHierarchy/types';

// Span type from useObservatory
export interface ObsSpan {
  id: string;
  trace_id?: string;
  parent_span_id?: string;
  task_id?: string;
  name: string;
  kind?: string;
  status: string;
  start_time?: string;
  end_time?: string;
  duration_ms: number;
  tokens_in: number;
  tokens_out: number;
  cost_usd: number;
  model?: string;
  provider?: string;
  attributes?: Record<string, any>;
  children?: ObsSpan[];
}

// Simplified span type from ExecHierarchy (used by Tree/Timeline/Chat)
export interface HierarchySpan {
  id: string;
  name: string;
  display_name?: string;
  startMs: number;
  durationMs: number;
  children?: HierarchySpan[];
  status?: 'ok' | 'error';
  attributes?: Record<string, any>;
  // Extended fields that may be on enriched spans
  cost_usd?: number;
  tokens_in?: number;
  tokens_out?: number;
  provider?: string;
  model?: string;
  trace_id?: string;
  parent_span_id?: string;
  task_id?: string;
}

export interface GraphData {
  nodes: Node[];
  edges: Edge[];
  stats: {
    totalSpans: number;
    totalCost: number;
    totalTokens: number;
    totalDurationMs: number;
    totalTurns: number;
  };
  // Collapsibility info
  rootNodeIds: string[];       // IDs of root nodes (always visible)
  expandableNodeIds: string[]; // IDs of nodes that can be expanded
}

type AnySpan = HierarchySpan | ObsSpan;

// Node types for styling
type SpanNodeType = 'session' | 'coordinator' | 'executor' | 'turn' | 'tool' | 'compile' | 'api' | 'default';

// Detect node type from span name
function getNodeTypeFromSpan(name: string): SpanNodeType {
  if (name === 'claude_code.session') return 'session';
  if (name === 'coordinator.task.execute') return 'coordinator';
  if (name === 'claude.execute' || name === 'gemini.execute') return 'executor';
  if (name.startsWith('exec.turn') || name.includes('.turn') || name === 'api_request') return 'turn';
  if (name.startsWith('claude_code.tool.') || name === 'exec.tool_use' || name.includes('tool')) return 'tool';
  if (name.startsWith('compile.') || name.startsWith('eval.') || name.startsWith('ailang.')) return 'compile';
  if (name.includes('api') || name.includes('generate')) return 'api';
  return 'default';
}

// Check if span is a turn
function isTurnSpan(span: AnySpan): boolean {
  const name = span.name;
  return name.startsWith('exec.turn') || name.includes('.turn') || name === 'api_request';
}

// Check if span is a tool call
function isToolSpan(span: AnySpan): boolean {
  const name = span.name;
  return name.startsWith('claude_code.tool.') || name === 'exec.tool_use';
}

// Check if span is a session/executor (root-level)
function isRootSpan(span: AnySpan): boolean {
  const name = span.name;
  return name === 'claude_code.session' ||
         name === 'claude.execute' ||
         name === 'gemini.execute' ||
         name === 'coordinator.task.execute';
}

// Get node color based on type
function getNodeColor(nodeType: SpanNodeType): string {
  switch (nodeType) {
    case 'session': return '#8b5cf6';     // Purple
    case 'coordinator': return '#6366f1'; // Indigo
    case 'executor': return '#3b82f6';    // Blue
    case 'turn': return '#10b981';        // Green
    case 'tool': return '#f59e0b';        // Amber
    case 'compile': return '#ec4899';     // Pink
    case 'api': return '#06b6d4';         // Cyan
    default: return '#64748b';            // Gray
  }
}

// Format duration for display
function formatDuration(ms?: number): string {
  if (!ms || ms === 0) return '';
  if (ms < 1000) return `${ms}ms`;
  if (ms < 60000) return `${(ms / 1000).toFixed(1)}s`;
  return `${(ms / 60000).toFixed(1)}m`;
}

// Format cost for display
function formatCost(cost?: number): string {
  if (!cost || cost === 0) return '';
  if (cost < 0.01) return `$${cost.toFixed(4)}`;
  if (cost < 1) return `$${cost.toFixed(3)}`;
  return `$${cost.toFixed(2)}`;
}

// Format tokens for display
function formatTokens(count?: number): string {
  if (!count || count === 0) return '';
  if (count >= 1000) return `${(count / 1000).toFixed(1)}K`;
  return String(count);
}

// Get turn number from span (returns 0 if not found, will be auto-numbered later)
function getTurnNumber(span: AnySpan): number {
  const attrs = span.attributes || {};
  const turnNum = attrs['turn.number'] || attrs['turn_number'];
  if (turnNum) return Number(turnNum);

  // Try to parse from name like "exec.turn.3"
  const match = span.name.match(/turn[._]?(\d+)/i);
  if (match) return Number(match[1]);

  return 0;
}

// Get start time from span for sorting
function getSpanStartTime(span: AnySpan): number {
  // Try various fields
  const ms = (span as any).startMs || (span as any).start_ms;
  if (ms) return ms;

  const startTime = (span as any).start_time;
  if (startTime) return new Date(startTime).getTime();

  return 0;
}

// Get smart label from span
function getSmartLabel(span: AnySpan): string {
  if (span.display_name) return span.display_name;

  const name = span.name;
  const attrs = span.attributes || {};

  // Claude Code tool calls
  if (name.startsWith('claude_code.tool.')) {
    const toolName = name.replace('claude_code.tool.', '');
    const filePath = attrs['file_path'] || attrs['path'];
    if (filePath && typeof filePath === 'string') {
      const fileName = filePath.split('/').pop() || filePath;
      return `${toolName}: ${fileName}`;
    }
    const command = attrs['command'] || attrs['description'];
    if (command && typeof command === 'string') {
      const brief = command.length > 25 ? command.substring(0, 25) + '...' : command;
      return `${toolName}: ${brief}`;
    }
    return toolName;
  }

  // Turn spans
  if (isTurnSpan(span)) {
    const turnNum = getTurnNumber(span);
    return turnNum > 0 ? `Turn ${turnNum}` : 'Turn';
  }

  // Session spans
  if (name === 'claude_code.session') return 'Session';
  if (name === 'claude.execute') return 'Claude Execute';
  if (name === 'gemini.execute') return 'Gemini Execute';
  if (name === 'coordinator.task.execute') {
    const title = attrs['task.title'] || attrs['directive'];
    if (title) return title.length > 30 ? title.substring(0, 30) + '...' : title;
    return 'Coordinator Task';
  }

  // Clean up the name
  return name.replace(/\./g, ' ').replace(/_/g, ' ');
}

// Get metrics from span
function getSpanMetrics(span: AnySpan) {
  return {
    durationMs: (span as any).durationMs || (span as any).duration_ms || 0,
    cost: (span as any).cost_usd || 0,
    tokensIn: (span as any).tokens_in || 0,
    tokensOut: (span as any).tokens_out || 0,
  };
}

// Collected turn info for building the graph
interface TurnInfo {
  span: AnySpan;
  turnNumber: number;
  tools: AnySpan[];
}

// Recursively collect all turns and their tools from span tree
function collectTurns(span: AnySpan, turnsMap: Map<string, TurnInfo>) {
  if (isTurnSpan(span)) {
    const turnNum = getTurnNumber(span);
    const tools: AnySpan[] = [];

    // Collect tool children
    if (span.children) {
      for (const child of span.children) {
        if (isToolSpan(child)) {
          tools.push(child);
        }
        // Also check grandchildren for tools
        if (child.children) {
          for (const grandchild of child.children) {
            if (isToolSpan(grandchild)) {
              tools.push(grandchild);
            }
          }
        }
      }
    }

    turnsMap.set(span.id, {
      span,
      turnNumber: turnNum,
      tools,
    });
  }

  // Recurse into children
  if (span.children) {
    for (const child of span.children) {
      collectTurns(child, turnsMap);
    }
  }
}

// Node dimensions
const NODE_WIDTH = 180;
const NODE_HEIGHT = 60;
const TURN_NODE_WIDTH = 220;
const TURN_NODE_HEIGHT = 70;

// Apply dagre layout
function applyDagreLayout(nodes: Node[], edges: Edge[]): Node[] {
  if (nodes.length === 0) return nodes;

  const g = new dagre.graphlib.Graph();
  g.setGraph({
    rankdir: 'TB',
    nodesep: 25,
    ranksep: 40,
    marginx: 30,
    marginy: 30,
  });
  g.setDefaultEdgeLabel(() => ({}));

  nodes.forEach(node => {
    const isTurn = node.data?.nodeType === 'turn';
    g.setNode(node.id, {
      width: isTurn ? TURN_NODE_WIDTH : NODE_WIDTH,
      height: isTurn ? TURN_NODE_HEIGHT : NODE_HEIGHT,
    });
  });

  edges.forEach(edge => {
    g.setEdge(edge.source, edge.target);
  });

  dagre.layout(g);

  return nodes.map(node => {
    const pos = g.node(node.id);
    if (!pos) return node;
    const isTurn = node.data?.nodeType === 'turn';
    const width = isTurn ? TURN_NODE_WIDTH : NODE_WIDTH;
    const height = isTurn ? TURN_NODE_HEIGHT : NODE_HEIGHT;
    return {
      ...node,
      position: {
        x: pos.x - width / 2,
        y: pos.y - height / 2,
      },
    };
  });
}

/**
 * Check if a span should be hidden based on hiddenSpanTypes
 */
function shouldHideSpan(span: AnySpan, hiddenSpanTypes?: Set<string>): boolean {
  if (!hiddenSpanTypes || hiddenSpanTypes.size === 0) return false;
  return hiddenSpanTypes.has(span.name);
}

/**
 * Recursively filter spans, removing hidden ones and their descendants
 * Returns filtered span array with updated children
 */
function filterSpans(spans: AnySpan[], hiddenSpanTypes?: Set<string>): AnySpan[] {
  if (!hiddenSpanTypes || hiddenSpanTypes.size === 0) return spans;

  const filtered: AnySpan[] = [];
  for (const span of spans) {
    if (shouldHideSpan(span, hiddenSpanTypes)) continue;

    // Recursively filter children
    if (span.children && span.children.length > 0) {
      const filteredChildren = filterSpans(span.children, hiddenSpanTypes);
      filtered.push({ ...span, children: filteredChildren });
    } else {
      filtered.push(span);
    }
  }
  return filtered;
}

/**
 * Build ReactFlow graph with turn-based hierarchy
 */
export function buildGraphFromSpans(
  spans: AnySpan[] | undefined,
  selectedNodeId?: string | null,
  hiddenSpanTypes?: Set<string>
): GraphData {
  const emptyResult: GraphData = {
    nodes: [],
    edges: [],
    stats: { totalSpans: 0, totalCost: 0, totalTokens: 0, totalDurationMs: 0, totalTurns: 0 },
    rootNodeIds: [],
    expandableNodeIds: [],
  };

  if (!spans || spans.length === 0) {
    return emptyResult;
  }

  // Apply span type filtering
  const filteredSpans = filterSpans(spans, hiddenSpanTypes);
  if (filteredSpans.length === 0) {
    return emptyResult;
  }

  const nodes: Node[] = [];
  const edges: Edge[] = [];
  const stats = { totalSpans: 0, totalCost: 0, totalTokens: 0, totalDurationMs: 0, totalTurns: 0 };
  const rootNodeIds: string[] = [];
  const expandableNodeIds: string[] = [];

  // Find root span (session/executor)
  let rootSpan: AnySpan | null = null;
  for (const span of filteredSpans) {
    if (isRootSpan(span)) {
      rootSpan = span;
      break;
    }
  }

  // If no root span, use first span as root
  if (!rootSpan && filteredSpans.length > 0) {
    rootSpan = filteredSpans[0];
  }

  if (!rootSpan) {
    return emptyResult;
  }

  // Collect all turns from the span tree
  const turnsMap = new Map<string, TurnInfo>();
  collectTurns(rootSpan, turnsMap);

  // Sort turns by turn number, then by start time (for api_request spans without turn numbers)
  const sortedTurns = Array.from(turnsMap.values())
    .sort((a, b) => {
      // If both have turn numbers, sort by number
      if (a.turnNumber > 0 && b.turnNumber > 0) {
        return a.turnNumber - b.turnNumber;
      }
      // If neither has turn number, sort by start time
      if (a.turnNumber === 0 && b.turnNumber === 0) {
        return getSpanStartTime(a.span) - getSpanStartTime(b.span);
      }
      // Spans with numbers come before spans without
      return b.turnNumber - a.turnNumber;
    });

  // Auto-number turns that don't have explicit turn numbers
  sortedTurns.forEach((turn, index) => {
    if (turn.turnNumber === 0) {
      turn.turnNumber = index + 1;  // Assign 1-based index
    }
  });

  // Create root node
  const rootType = getNodeTypeFromSpan(rootSpan.name);
  const rootColor = getNodeColor(rootType);
  const rootMetrics = getSpanMetrics(rootSpan);

  stats.totalSpans++;
  stats.totalCost += rootMetrics.cost;
  stats.totalTokens += rootMetrics.tokensIn + rootMetrics.tokensOut;
  stats.totalDurationMs = rootMetrics.durationMs;

  // Track root node
  rootNodeIds.push(rootSpan.id);
  // Root is expandable if it has turns
  if (sortedTurns.length > 0) {
    expandableNodeIds.push(rootSpan.id);
  }

  nodes.push({
    id: rootSpan.id,
    type: 'span',
    position: { x: 0, y: 0 },
    data: {
      label: getSmartLabel(rootSpan),
      name: rootSpan.name,
      nodeType: rootType,
      nodeColor: rootColor,
      status: rootSpan.status,
      ...rootMetrics,
      metricsStr: [
        formatDuration(rootMetrics.durationMs),
        formatCost(rootMetrics.cost),
        rootMetrics.tokensIn || rootMetrics.tokensOut
          ? `[${formatTokens(rootMetrics.tokensIn)}→${formatTokens(rootMetrics.tokensOut)}]`
          : '',
      ].filter(Boolean).join(' '),
      // Original span for popover
      _span: rootSpan,
      // Collapsibility info
      childCount: sortedTurns.length,
      isExpandable: sortedTurns.length > 0,
      childType: 'turn',
    },
    selected: selectedNodeId === rootSpan.id,
    style: {
      borderLeftColor: rootColor,
      borderLeftWidth: 4,
      borderLeftStyle: 'solid',
    },
  });

  // Create turn nodes and connect them sequentially
  let prevTurnId: string | null = null;

  for (const turnInfo of sortedTurns) {
    const { span: turnSpan, tools } = turnInfo;
    // Force turn styling for all turn spans (including api_request)
    const turnColor = getNodeColor('turn');  // Green for turns
    const turnMetrics = getSpanMetrics(turnSpan);

    stats.totalSpans++;
    stats.totalTurns++;
    stats.totalCost += turnMetrics.cost;
    stats.totalTokens += turnMetrics.tokensIn + turnMetrics.tokensOut;

    // Turn is expandable if it has tools
    if (tools.length > 0) {
      expandableNodeIds.push(turnSpan.id);
    }

    // Create turn node - use auto-assigned turn number for label
    const turnLabel = turnInfo.turnNumber > 0 ? `Turn ${turnInfo.turnNumber}` : getSmartLabel(turnSpan);

    nodes.push({
      id: turnSpan.id,
      type: 'span',
      position: { x: 0, y: 0 },
      data: {
        label: turnLabel,
        name: turnSpan.name,
        nodeType: 'turn',  // Always mark as turn for proper styling
        nodeColor: turnColor,
        status: turnSpan.status,
        ...turnMetrics,
        metricsStr: [
          formatDuration(turnMetrics.durationMs),
          formatCost(turnMetrics.cost),
          turnMetrics.tokensIn || turnMetrics.tokensOut
            ? `[${formatTokens(turnMetrics.tokensIn)}→${formatTokens(turnMetrics.tokensOut)}]`
            : '',
        ].filter(Boolean).join(' '),
        toolCount: tools.length,
        // Original span for popover
        _span: turnSpan,
        // Collapsibility info
        childCount: tools.length,
        isExpandable: tools.length > 0,
        childType: 'tool',
        parentId: rootSpan.id,
        turnNumber: turnInfo.turnNumber,
      },
      selected: selectedNodeId === turnSpan.id,
      style: {
        borderLeftColor: turnColor,
        borderLeftWidth: 3,
        borderLeftStyle: 'solid',
      },
    });

    // Connect first turn to root, subsequent turns to previous turn (sequential flow)
    const sourceId = prevTurnId || rootSpan.id;
    edges.push({
      id: `e-${sourceId}-${turnSpan.id}`,
      source: sourceId,
      target: turnSpan.id,
      type: 'smoothstep',
      style: {
        stroke: prevTurnId ? '#10b981' : '#4b5563', // Green for turn-to-turn, gray for root-to-turn
        strokeWidth: 2,
      },
      markerEnd: {
        type: MarkerType.ArrowClosed,
        color: prevTurnId ? '#10b981' : '#4b5563',
        width: 12,
        height: 12,
      },
    });

    prevTurnId = turnSpan.id;

    // Create tool nodes as children of the turn
    for (const toolSpan of tools) {
      const toolType = getNodeTypeFromSpan(toolSpan.name);
      const toolColor = getNodeColor(toolType);
      const toolMetrics = getSpanMetrics(toolSpan);

      stats.totalSpans++;
      stats.totalCost += toolMetrics.cost;
      stats.totalTokens += toolMetrics.tokensIn + toolMetrics.tokensOut;

      nodes.push({
        id: toolSpan.id,
        type: 'span',
        position: { x: 0, y: 0 },
        data: {
          label: getSmartLabel(toolSpan),
          name: toolSpan.name,
          nodeType: toolType,
          nodeColor: toolColor,
          status: toolSpan.status,
          ...toolMetrics,
          metricsStr: formatDuration(toolMetrics.durationMs),
          // Original span for popover
          _span: toolSpan,
          // Tool hierarchy info
          parentId: turnSpan.id,
          isExpandable: false,
        },
        selected: selectedNodeId === toolSpan.id,
        style: {
          borderLeftColor: toolColor,
          borderLeftWidth: 2,
          borderLeftStyle: 'solid',
        },
      });

      // Connect tool to its turn
      edges.push({
        id: `e-${turnSpan.id}-${toolSpan.id}`,
        source: turnSpan.id,
        target: toolSpan.id,
        type: 'default',
        style: { stroke: '#4b5563', strokeWidth: 1 },
        markerEnd: {
          type: MarkerType.ArrowClosed,
          color: '#4b5563',
          width: 8,
          height: 8,
        },
      });
    }
  }

  // If no turns found, fall back to simpler display of direct children
  if (sortedTurns.length === 0 && rootSpan.children) {
    // Root is expandable if it has children
    if (rootSpan.children.length > 0) {
      expandableNodeIds.push(rootSpan.id);
    }

    for (const child of rootSpan.children) {
      const childType = getNodeTypeFromSpan(child.name);
      const childColor = getNodeColor(childType);
      const childMetrics = getSpanMetrics(child);

      stats.totalSpans++;
      stats.totalCost += childMetrics.cost;
      stats.totalTokens += childMetrics.tokensIn + childMetrics.tokensOut;

      nodes.push({
        id: child.id,
        type: 'span',
        position: { x: 0, y: 0 },
        data: {
          label: getSmartLabel(child),
          name: child.name,
          nodeType: childType,
          nodeColor: childColor,
          status: child.status,
          ...childMetrics,
          metricsStr: formatDuration(childMetrics.durationMs),
          // Original span for popover
          _span: child,
          parentId: rootSpan.id,
          isExpandable: false,
        },
        selected: selectedNodeId === child.id,
        style: {
          borderLeftColor: childColor,
          borderLeftWidth: 2,
          borderLeftStyle: 'solid',
        },
      });

      edges.push({
        id: `e-${rootSpan.id}-${child.id}`,
        source: rootSpan.id,
        target: child.id,
        type: 'default',
        style: { stroke: '#4b5563', strokeWidth: 1 },
        markerEnd: {
          type: MarkerType.ArrowClosed,
          color: '#4b5563',
          width: 8,
          height: 8,
        },
      });
    }
  }

  // Apply layout
  const layoutedNodes = applyDagreLayout(nodes, edges);

  return {
    nodes: layoutedNodes,
    edges,
    stats,
    rootNodeIds,
    expandableNodeIds,
  };
}

/**
 * Build ReactFlow graph from API turn-grouped data
 * This uses the server's pre-computed turn grouping (via group_by=turns)
 * for consistent results between CLI and dashboard.
 */
export function buildGraphFromTurnGrouped(
  turnGrouped: TurnGroupedHierarchy,
  selectedNodeId?: string | null
): GraphData {
  const emptyResult: GraphData = {
    nodes: [],
    edges: [],
    stats: { totalSpans: 0, totalCost: 0, totalTokens: 0, totalDurationMs: 0, totalTurns: 0 },
    rootNodeIds: [],
    expandableNodeIds: [],
  };

  if (!turnGrouped || !turnGrouped.turns || turnGrouped.turns.length === 0) {
    return emptyResult;
  }

  const nodes: Node[] = [];
  const edges: Edge[] = [];
  const rootNodeIds: string[] = [];
  const expandableNodeIds: string[] = [];
  const stats = {
    totalSpans: 0,
    totalCost: turnGrouped.stats?.total_cost || 0,
    totalTokens: turnGrouped.stats?.total_tokens || 0,
    totalDurationMs: turnGrouped.stats?.duration_ms || 0,
    totalTurns: turnGrouped.stats?.total_turns || 0,
  };

  // Create session/root node if available
  let rootNodeId: string | null = null;
  if (turnGrouped.session) {
    rootNodeId = turnGrouped.session.id;
    const sessionColor = '#8b5cf6'; // Purple for session

    stats.totalSpans++;
    rootNodeIds.push(rootNodeId);
    // Session is expandable if it has turns
    if (turnGrouped.turns.length > 0) {
      expandableNodeIds.push(rootNodeId);
    }

    nodes.push({
      id: rootNodeId,
      type: 'span',
      position: { x: 0, y: 0 },
      data: {
        label: turnGrouped.session.name || 'Session',
        name: turnGrouped.session.name,
        nodeType: 'session',
        nodeColor: sessionColor,
        status: 'ok',
        durationMs: turnGrouped.session.duration_ms,
        cost: turnGrouped.session.cost,
        tokensIn: turnGrouped.session.tokens_in,
        tokensOut: turnGrouped.session.tokens_out,
        metricsStr: [
          formatDuration(turnGrouped.session.duration_ms),
          formatCost(turnGrouped.session.cost),
          turnGrouped.session.tokens_in || turnGrouped.session.tokens_out
            ? `[${formatTokens(turnGrouped.session.tokens_in)}→${formatTokens(turnGrouped.session.tokens_out)}]`
            : '',
        ].filter(Boolean).join(' '),
        // Collapsibility info
        childCount: turnGrouped.turns.length,
        isExpandable: turnGrouped.turns.length > 0,
        childType: 'turn',
      },
      selected: selectedNodeId === rootNodeId,
      style: {
        borderLeftColor: sessionColor,
        borderLeftWidth: 4,
        borderLeftStyle: 'solid',
      },
    });
  }

  // Create turn nodes and connect them sequentially
  let prevTurnId: string | null = null;

  for (const turn of turnGrouped.turns) {
    const turnNodeId = turn.span_id;
    const turnColor = '#10b981'; // Green for turns
    const toolCount = turn.tools?.length || 0;

    stats.totalSpans++;

    // Turn is expandable if it has tools
    if (toolCount > 0) {
      expandableNodeIds.push(turnNodeId);
    }

    // Create turn node
    nodes.push({
      id: turnNodeId,
      type: 'span',
      position: { x: 0, y: 0 },
      data: {
        label: `Turn ${turn.turn_number}`,
        name: `exec.turn.${turn.turn_number}`,
        nodeType: 'turn',
        nodeColor: turnColor,
        status: 'ok',
        durationMs: turn.duration_ms,
        cost: turn.cost,
        tokensIn: turn.tokens_in,
        tokensOut: turn.tokens_out,
        metricsStr: [
          formatDuration(turn.duration_ms),
          formatCost(turn.cost),
          turn.tokens_in || turn.tokens_out
            ? `[${formatTokens(turn.tokens_in)}→${formatTokens(turn.tokens_out)}]`
            : '',
        ].filter(Boolean).join(' '),
        toolCount,
        // Collapsibility info
        childCount: toolCount,
        isExpandable: toolCount > 0,
        childType: 'tool',
        parentId: rootNodeId,
        turnNumber: turn.turn_number,
      },
      selected: selectedNodeId === turnNodeId,
      style: {
        borderLeftColor: turnColor,
        borderLeftWidth: 3,
        borderLeftStyle: 'solid',
      },
    });

    // Connect: first turn to root (if exists), subsequent turns to previous turn
    const sourceId = prevTurnId || rootNodeId;
    if (sourceId) {
      edges.push({
        id: `e-${sourceId}-${turnNodeId}`,
        source: sourceId,
        target: turnNodeId,
        type: 'smoothstep',
        style: {
          stroke: prevTurnId ? '#10b981' : '#4b5563', // Green for turn-to-turn, gray for root-to-turn
          strokeWidth: 2,
        },
        markerEnd: {
          type: MarkerType.ArrowClosed,
          color: prevTurnId ? '#10b981' : '#4b5563',
          width: 12,
          height: 12,
        },
      });
    }

    prevTurnId = turnNodeId;

    // Create tool nodes as children of the turn
    if (turn.tools) {
      for (const tool of turn.tools) {
        const toolColor = '#f59e0b'; // Amber for tools

        stats.totalSpans++;

        // Build tool label
        let toolLabel = tool.tool_name || tool.name;
        if (toolLabel.startsWith('claude_code.tool.')) {
          toolLabel = toolLabel.replace('claude_code.tool.', '');
        }

        nodes.push({
          id: tool.id,
          type: 'span',
          position: { x: 0, y: 0 },
          data: {
            label: toolLabel,
            name: tool.name,
            nodeType: 'tool',
            nodeColor: toolColor,
            status: tool.status || 'ok',
            durationMs: tool.duration_ms,
            cost: tool.cost || 0,
            metricsStr: formatDuration(tool.duration_ms),
            // Tool hierarchy info
            parentId: turnNodeId,
            isExpandable: false,
          },
          selected: selectedNodeId === tool.id,
          style: {
            borderLeftColor: toolColor,
            borderLeftWidth: 2,
            borderLeftStyle: 'solid',
          },
        });

        // Connect tool to its turn
        edges.push({
          id: `e-${turnNodeId}-${tool.id}`,
          source: turnNodeId,
          target: tool.id,
          type: 'default',
          style: { stroke: '#4b5563', strokeWidth: 1 },
          markerEnd: {
            type: MarkerType.ArrowClosed,
            color: '#4b5563',
            width: 8,
            height: 8,
          },
        });
      }
    }
  }

  // Apply layout
  const layoutedNodes = applyDagreLayout(nodes, edges);

  return {
    nodes: layoutedNodes,
    edges,
    stats,
    rootNodeIds,
    expandableNodeIds,
  };
}

export default buildGraphFromSpans;
