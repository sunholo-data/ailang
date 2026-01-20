/**
 * EvolutionTree - Bioluminescent Data Organism Visualization
 *
 * An organic, tree-like visualization showing AI execution flow over time.
 * The stem curves based on token activity, turns appear as luminous nodes,
 * and tools branch off as smaller tendrils with leaf-like terminals.
 */
import React, { useMemo, useCallback, useRef, useEffect, useState } from 'react';
import type { HierarchyNode, Span } from './types';
import styles from './EvolutionTree.module.css';

// Types for our tree structure
interface TreeSession {
  id: string;
  name: string;
  durationMs: number;
  cost: number;
  tokensIn: number;
  tokensOut: number;
}

interface TreeTurn {
  id: string;
  turnNumber: number;
  durationMs: number;
  cost: number;
  tokensIn: number;
  tokensOut: number;
  status: 'ok' | 'error' | 'pending';
  tools: TreeTool[];
}

interface TreeTool {
  id: string;
  name: string;
  durationMs: number;
  status: 'ok' | 'error';
  cost?: number;
}

// Shared tool node - represents a unique tool name used across multiple turns
interface SharedToolNode {
  name: string;
  displayName: string;
  toolType: string;
  color: string;
  usages: { turnIndex: number; turnId: string; tool: TreeTool }[];
  x: number;
  y: number;
  hasError: boolean;
}

// Tool type colors
const TOOL_COLORS: Record<string, string> = {
  Read: '#60a5fa',      // Blue - reading files
  Write: '#a78bfa',     // Purple - writing files
  Edit: '#f472b6',      // Pink - editing files
  Bash: '#fbbf24',      // Amber - shell commands
  Grep: '#34d399',      // Green - searching
  Glob: '#2dd4bf',      // Teal - file patterns
  Task: '#fb923c',      // Orange - agent tasks
  WebFetch: '#38bdf8',  // Sky - web requests
  WebSearch: '#818cf8', // Indigo - web search
  default: '#94a3b8',   // Gray - other
};

// Extract tool type from name
function getToolType(name: string): string {
  // Common tool prefixes
  const prefixes = ['Read', 'Write', 'Edit', 'Bash', 'Grep', 'Glob', 'Task', 'WebFetch', 'WebSearch'];
  for (const prefix of prefixes) {
    if (name.startsWith(prefix) || name.toLowerCase().includes(prefix.toLowerCase())) {
      return prefix;
    }
  }
  return 'default';
}

// Get color for tool
function getToolColor(name: string): string {
  const toolType = getToolType(name);
  return TOOL_COLORS[toolType] || TOOL_COLORS.default;
}

// Create display name (truncate file paths)
function getToolDisplayName(name: string): string {
  // Extract filename from paths
  const pathMatch = name.match(/[/\\]([^/\\]+)$/);
  if (pathMatch) {
    return pathMatch[1].length > 20 ? pathMatch[1].slice(0, 17) + '...' : pathMatch[1];
  }
  return name.length > 20 ? name.slice(0, 17) + '...' : name;
}

export interface EvolutionTreeProps {
  spans?: Span[];
  nodes?: HierarchyNode[];
  selectedNodeId?: string | null;
  onNodeClick?: (node: HierarchyNode, event?: React.MouseEvent) => void;
  hiddenSpanTypes?: Set<string>;
  isExpanded?: boolean;
}

// Detect anomalies in turn metrics
function detectAnomalies(turns: TreeTurn[]): Set<number> {
  if (turns.length < 3) return new Set();

  // Calculate mean and stdDev for cost, tokens, and duration
  const costs = turns.map(t => t.cost || 0);
  const tokens = turns.map(t => (t.tokensIn || 0) + (t.tokensOut || 0));
  const durations = turns.map(t => t.durationMs || 0);

  const mean = (arr: number[]) => arr.reduce((a, b) => a + b, 0) / arr.length;
  const stdDev = (arr: number[], m: number) => Math.sqrt(arr.reduce((sum, x) => sum + (x - m) ** 2, 0) / arr.length);

  const costMean = mean(costs);
  const costStd = stdDev(costs, costMean) || 1;
  const tokenMean = mean(tokens);
  const tokenStd = stdDev(tokens, tokenMean) || 1;
  const durationMean = mean(durations);
  const durationStd = stdDev(durations, durationMean) || 1;

  const anomalies = new Set<number>();
  turns.forEach((turn, i) => {
    const costZ = Math.abs((turn.cost || 0) - costMean) / costStd;
    const tokenZ = Math.abs(((turn.tokensIn || 0) + (turn.tokensOut || 0)) - tokenMean) / tokenStd;
    const durationZ = Math.abs((turn.durationMs || 0) - durationMean) / durationStd;

    // Mark as anomaly if any metric is > 2 std devs from mean
    if (costZ > 2 || tokenZ > 2 || durationZ > 2) {
      anomalies.add(i);
    }
  });

  return anomalies;
}

// Calculate spiral positions for turns
interface SpiralPosition {
  x: number;
  y: number;
  angle: number;
  radius: number;
  turn: TreeTurn;
  isAnomaly: boolean;
  spiralDirection: number; // 1 or -1
  activity: number; // 0-1, used for node sizing
}

function calculateSpiralPositions(
  turns: TreeTurn[],
  centerX: number,
  centerY: number,
  minRadius: number,
  maxRadius: number,
  turnsPerRotation: number = 12
): SpiralPosition[] {
  if (turns.length === 0) return [];

  const anomalies = detectAnomalies(turns);
  const n = turns.length;

  // Calculate max metrics for activity visualization (used for node sizing, not spacing)
  const maxTokens = Math.max(...turns.map(t => (t.tokensIn || 0) + (t.tokensOut || 0)), 1);
  const maxCost = Math.max(...turns.map(t => t.cost || 0), 0.001);

  // Calculate duration-proportional angle steps
  // Total angle budget for all turns (based on turnsPerRotation)
  const totalAngleBudget = (2 * Math.PI * n) / turnsPerRotation;

  // Calculate total duration (use minimum of 1ms to avoid division by zero)
  const totalDuration = Math.max(
    turns.reduce((sum, t) => sum + (t.durationMs || 0), 0),
    1
  );

  // Pre-calculate proportional angle for each turn
  // The gap AFTER a turn is proportional to that turn's duration
  // Use a minimum proportion to ensure very fast turns are still visible
  const minProportion = 0.3 / n; // Each turn gets at least 30% of uniform spacing
  const remainingProportion = 1 - (minProportion * n);

  const angleSteps = turns.map(turn => {
    const duration = turn.durationMs || 0;
    const durationProportion = remainingProportion * (duration / totalDuration);
    const totalProportion = minProportion + durationProportion;
    return totalAngleBudget * totalProportion;
  });

  const positions: SpiralPosition[] = [];
  let spiralDirection = 1; // 1 = clockwise, -1 = counter-clockwise
  let currentAngle = 0;
  let currentRadius = minRadius;
  const radiusRange = maxRadius - minRadius;

  turns.forEach((turn, i) => {
    const progress = i / Math.max(n - 1, 1);
    const isAnomaly = anomalies.has(i);

    // Activity score for node sizing
    const tokenActivity = ((turn.tokensIn || 0) + (turn.tokensOut || 0)) / maxTokens;
    const costActivity = (turn.cost || 0) / maxCost;
    const activity = tokenActivity * 0.5 + costActivity * 0.5;

    // When anomaly detected: flip direction AND jump radius outward to avoid crossings
    if (isAnomaly && i > 0) {
      spiralDirection *= -1;
      // Jump radius by ~15% to create visual separation and avoid line crossings
      currentRadius = Math.min(currentRadius + radiusRange * 0.15, maxRadius);
    } else {
      // Normal radius progression
      currentRadius = minRadius + radiusRange * progress;
    }

    // Advance angle using duration-proportional step
    currentAngle += angleSteps[i] * spiralDirection;

    // Calculate position
    const x = centerX + currentRadius * Math.cos(currentAngle);
    const y = centerY + currentRadius * Math.sin(currentAngle);

    positions.push({
      x,
      y,
      angle: currentAngle,
      radius: currentRadius,
      turn,
      isAnomaly,
      spiralDirection,
      activity,
    });
  });

  return positions;
}

// Tool type priority for ring assignment (inner = more important/frequent types)
const TOOL_TYPE_RINGS: Record<string, number> = {
  Read: 0,      // Inner ring - file reads
  Edit: 0,      // Inner ring - file edits
  Write: 0,     // Inner ring - file writes
  Bash: 1,      // Middle ring - shell commands
  Grep: 1,      // Middle ring - searches
  Glob: 1,      // Middle ring - file patterns
  Task: 2,      // Outer ring - agent tasks
  WebFetch: 2,  // Outer ring - web
  WebSearch: 2, // Outer ring - web
  default: 1,   // Middle ring - other
};

// Build shared tool nodes from turns and spiral positions
function buildSharedToolNodes(
  turns: TreeTurn[],
  spiralPositions: SpiralPosition[],
  centerX: number,
  centerY: number,
  baseToolRingRadius: number,
  showAllTools: boolean = false
): SharedToolNode[] {
  // Collect all unique tool names and their usages
  const toolMap = new Map<string, SharedToolNode>();

  turns.forEach((turn, turnIndex) => {
    turn.tools.forEach(tool => {
      const existing = toolMap.get(tool.name);
      if (existing) {
        existing.usages.push({ turnIndex, turnId: turn.id, tool });
        if (tool.status === 'error') existing.hasError = true;
      } else {
        toolMap.set(tool.name, {
          name: tool.name,
          displayName: getToolDisplayName(tool.name),
          toolType: getToolType(tool.name),
          color: getToolColor(tool.name),
          usages: [{ turnIndex, turnId: turn.id, tool }],
          x: 0,
          y: 0,
          hasError: tool.status === 'error',
        });
      }
    });
  });

  // Convert to array
  let sharedTools = Array.from(toolMap.values());

  // Filter to only show tools used more than once (unless showAllTools is true)
  // This dramatically reduces clutter for large sessions
  if (!showAllTools && sharedTools.length > 30) {
    sharedTools = sharedTools.filter(t => t.usages.length >= 2);
  }

  // Group tools by type for ring assignment
  const toolsByRing: Map<number, SharedToolNode[]> = new Map();
  sharedTools.forEach(tool => {
    const ringIndex = TOOL_TYPE_RINGS[tool.toolType] ?? TOOL_TYPE_RINGS.default;
    if (!toolsByRing.has(ringIndex)) {
      toolsByRing.set(ringIndex, []);
    }
    toolsByRing.get(ringIndex)!.push(tool);
  });

  // Sort each ring's tools by usage count (most used = first)
  toolsByRing.forEach(tools => {
    tools.sort((a, b) => b.usages.length - a.usages.length);
  });

  // Calculate dynamic ring radii based on tool count per ring
  // More tools = larger ring to fit them with spacing
  const ringRadii: number[] = [];
  const ringGap = 25; // Gap between rings
  let currentRadius = baseToolRingRadius;

  for (let ring = 0; ring <= 2; ring++) {
    const toolsInRing = toolsByRing.get(ring) || [];
    // Minimum circumference to fit tools with ~15px spacing
    const minCircumference = toolsInRing.length * 18;
    const minRadiusForTools = minCircumference / (2 * Math.PI);
    // Use whichever is larger: base radius or calculated minimum
    ringRadii[ring] = Math.max(currentRadius, minRadiusForTools);
    currentRadius = ringRadii[ring] + ringGap;
  }

  // Position tools in their assigned rings
  toolsByRing.forEach((tools, ringIndex) => {
    const radius = ringRadii[ringIndex];
    const angleStep = (2 * Math.PI) / Math.max(tools.length, 1);

    tools.forEach((tool, i) => {
      // Start angle offset per ring to stagger tools
      const startAngle = (ringIndex * Math.PI) / 6;
      const angle = startAngle + i * angleStep;

      tool.x = centerX + radius * Math.cos(angle);
      tool.y = centerY + radius * Math.sin(angle);
    });
  });

  return sharedTools;
}

// Generate spiral path through turn positions
function generateSpiralPath(positions: SpiralPosition[], centerX: number, centerY: number): string {
  if (positions.length === 0) return '';

  // Start from center
  let path = `M ${centerX} ${centerY}`;

  // Smooth curve through all positions
  positions.forEach((pos, i) => {
    if (i === 0) {
      // First segment - quadratic curve from center
      path += ` Q ${centerX + (pos.x - centerX) * 0.5} ${centerY + (pos.y - centerY) * 0.5}, ${pos.x} ${pos.y}`;
    } else {
      // Subsequent segments - smooth curves
      const prev = positions[i - 1];
      const cpX = prev.x + (pos.x - prev.x) * 0.5;
      const cpY = prev.y + (pos.y - prev.y) * 0.5;
      path += ` S ${cpX} ${cpY}, ${pos.x} ${pos.y}`;
    }
  });

  return path;
}

// Helper to convert polar to cartesian coordinates
function polarToCartesian(cx: number, cy: number, radius: number, angleInDegrees: number) {
  const angleInRadians = (angleInDegrees - 90) * Math.PI / 180.0;
  return {
    x: cx + radius * Math.cos(angleInRadians),
    y: cy + radius * Math.sin(angleInRadians),
  };
}

// Helper to draw arc paths for turn spread visualization
function describeArc(x: number, y: number, radius: number, startAngle: number, endAngle: number): string {
  const start = polarToCartesian(x, y, radius, endAngle);
  const end = polarToCartesian(x, y, radius, startAngle);
  const largeArcFlag = endAngle - startAngle <= 180 ? "0" : "1";
  return `M ${start.x} ${start.y} A ${radius} ${radius} 0 ${largeArcFlag} 0 ${end.x} ${end.y}`;
}

// Generate branch path from stem to tool
function generateBranchPath(
  stemX: number,
  stemY: number,
  leafX: number,
  leafY: number,
  side: 'left' | 'right'
): string {
  const controlOffset = side === 'left' ? -40 : 40;
  const cp1x = stemX + controlOffset * 0.3;
  const cp1y = stemY;
  const cp2x = leafX - controlOffset * 0.5;
  const cp2y = leafY;

  return `M ${stemX} ${stemY} C ${cp1x} ${cp1y}, ${cp2x} ${cp2y}, ${leafX} ${leafY}`;
}

// Filter spans based on hiddenSpanTypes
function filterSpans(spans: Span[], hiddenSpanTypes?: Set<string>): Span[] {
  if (!hiddenSpanTypes || hiddenSpanTypes.size === 0) return spans;

  const filtered: Span[] = [];
  for (const span of spans) {
    if (hiddenSpanTypes.has(span.name)) continue;

    if (span.children && span.children.length > 0) {
      const filteredChildren = filterSpans(span.children, hiddenSpanTypes);
      filtered.push({ ...span, children: filteredChildren });
    } else {
      filtered.push(span);
    }
  }
  return filtered;
}

// Check if a span is a turn
function isTurnSpan(name: string): boolean {
  return name === 'api_request' ||
    name.startsWith('exec.turn') ||
    name.includes('.turn') ||
    name.includes('turn.');
}

// Check if a span is a tool call
function isToolSpan(name: string): boolean {
  return name.startsWith('claude_code.tool.') ||
    name === 'exec.tool_use' ||
    name.includes('.tool') ||
    name.includes('tool.');
}

// Check if a span is a root/session span
function isSessionSpan(name: string): boolean {
  return name === 'claude_code.session' ||
    name === 'claude.execute' ||
    name === 'gemini.execute' ||
    name === 'coordinator.task.execute' ||
    name.startsWith('ailang.') ||
    name.startsWith('eval.');
}

// Transform HierarchyNode[] to tree structure (uses pre-transformed data)
function buildTreeFromNodes(
  nodes: HierarchyNode[]
): { session: TreeSession | null; turns: TreeTurn[] } {
  if (!nodes || nodes.length === 0) {
    return { session: null, turns: [] };
  }

  // First node is typically the session/root
  const rootNode = nodes[0];

  const session: TreeSession = {
    id: rootNode.id,
    name: rootNode.label,
    durationMs: rootNode.durationMs || 0,
    cost: rootNode.cost || 0,
    tokensIn: rootNode.tokensIn || 0,
    tokensOut: rootNode.tokensOut || 0,
  };

  const turnsList: TreeTurn[] = [];
  let turnCounter = 0;

  // Recursively collect turns from hierarchy
  function collectFromNode(node: HierarchyNode) {
    if (node.type === 'turn') {
      turnCounter++;
      const turnNum = node.turnNumber || turnCounter;

      const tools: TreeTool[] = [];
      if (node.children) {
        node.children.forEach(child => {
          if (child.type === 'tool_use') {
            tools.push({
              id: child.id,
              name: child.label,
              durationMs: child.durationMs || 0,
              status: child.status === 'error' ? 'error' : 'ok',
              cost: child.cost,
            });
          }
        });
      }

      turnsList.push({
        id: node.id,
        turnNumber: turnNum,
        durationMs: node.durationMs || 0,
        cost: node.cost || 0,
        tokensIn: node.tokensIn || 0,
        tokensOut: node.tokensOut || 0,
        status: node.status === 'error' ? 'error' : 'ok',
        tools,
      });
    }

    // Recurse into children
    if (node.children) {
      node.children.forEach(child => collectFromNode(child));
    }
  }

  nodes.forEach(node => collectFromNode(node));

  // If no turns found, use all non-root nodes as virtual turns
  if (turnsList.length === 0) {
    nodes.forEach((node, idx) => {
      if (idx > 0 || nodes.length === 1) {
        const tools: TreeTool[] = [];
        if (node.children) {
          node.children.forEach(child => {
            tools.push({
              id: child.id,
              name: child.label,
              durationMs: child.durationMs || 0,
              status: child.status === 'error' ? 'error' : 'ok',
              cost: child.cost,
            });
          });
        }

        turnsList.push({
          id: node.id,
          turnNumber: idx + 1,
          durationMs: node.durationMs || 0,
          cost: node.cost || 0,
          tokensIn: node.tokensIn || 0,
          tokensOut: node.tokensOut || 0,
          status: node.status === 'error' ? 'error' : 'ok',
          tools,
        });
      }
    });
  }

  return { session, turns: turnsList.sort((a, b) => a.turnNumber - b.turnNumber) };
}

// Transform spans to tree structure (fallback when nodes not available)
function buildTreeFromSpans(
  spans?: Span[],
  hiddenSpanTypes?: Set<string>
): { session: TreeSession | null; turns: TreeTurn[] } {
  if (!spans || spans.length === 0) {
    return { session: null, turns: [] };
  }

  // DON'T filter spans when building tree - we need to count turns first
  // Filtering happens at display time, not structure time
  const allSpans = spans;

  // Find session/root span - look for known session types first, then use first span
  const sessionSpan = allSpans.find(s => isSessionSpan(s.name)) || allSpans[0];

  const session: TreeSession = {
    id: sessionSpan.id,
    name: sessionSpan.display_name || sessionSpan.name,
    durationMs: sessionSpan.durationMs || 0,
    cost: (sessionSpan as any).cost_usd || 0,
    tokensIn: (sessionSpan as any).tokens_in || 0,
    tokensOut: (sessionSpan as any).tokens_out || 0,
  };

  // Collect turns - walk the entire tree
  const turnsList: TreeTurn[] = [];
  let turnCounter = 0;

  function collectTurns(span: Span, depth: number = 0) {
    const isTurn = isTurnSpan(span.name);

    if (isTurn) {
      turnCounter++;
      // Try to extract turn number from attributes or name
      const attrTurnNum = span.attributes?.['turn.number'] ||
                          span.attributes?.['turn_number'] ||
                          span.attributes?.['exec.turn'];
      const match = span.name.match(/turn[._]?(\d+)/i);
      const turnNum = attrTurnNum ? parseInt(String(attrTurnNum), 10) :
                      match ? parseInt(match[1], 10) :
                      turnCounter;

      const tools: TreeTool[] = [];

      // Collect tool children (direct children only)
      if (span.children) {
        span.children.forEach(child => {
          if (isToolSpan(child.name)) {
            tools.push({
              id: child.id,
              name: child.display_name || child.name.replace('claude_code.tool.', '').replace('exec.tool_use.', ''),
              durationMs: child.durationMs || 0,
              status: child.status === 'error' ? 'error' : 'ok',
              cost: (child as any).cost_usd,
            });
          }
        });
      }

      // Get cost from attributes if not directly on span
      const cost = (span as any).cost_usd ||
                   parseFloat(span.attributes?.['cost_usd'] || '0') || 0;
      const tokensIn = (span as any).tokens_in ||
                       parseInt(span.attributes?.['tokens_in'] || span.attributes?.['input_tokens'] || '0', 10) || 0;
      const tokensOut = (span as any).tokens_out ||
                        parseInt(span.attributes?.['tokens_out'] || span.attributes?.['output_tokens'] || '0', 10) || 0;

      turnsList.push({
        id: span.id,
        turnNumber: turnNum,
        durationMs: span.durationMs || 0,
        cost,
        tokensIn,
        tokensOut,
        status: span.status === 'error' ? 'error' : 'ok',
        tools,
      });
    }

    // Recurse into children
    if (span.children) {
      span.children.forEach(child => collectTurns(child, depth + 1));
    }
  }

  // Collect from all spans (unfiltered to get turn counts right)
  allSpans.forEach(span => collectTurns(span, 0));

  // If no turns found, try to create virtual turns from top-level spans
  // This handles cases where the hierarchy is flat
  if (turnsList.length === 0 && allSpans.length > 1) {
    allSpans.forEach((span, idx) => {
      if (span.id !== sessionSpan.id) {
        const tools: TreeTool[] = [];
        if (span.children) {
          span.children.forEach(child => {
            tools.push({
              id: child.id,
              name: child.display_name || child.name,
              durationMs: child.durationMs || 0,
              status: child.status === 'error' ? 'error' : 'ok',
              cost: (child as any).cost_usd,
            });
          });
        }

        turnsList.push({
          id: span.id,
          turnNumber: idx,
          durationMs: span.durationMs || 0,
          cost: (span as any).cost_usd || 0,
          tokensIn: (span as any).tokens_in || 0,
          tokensOut: (span as any).tokens_out || 0,
          status: span.status === 'error' ? 'error' : 'ok',
          tools,
        });
      }
    });
  }

  // Sort by turn number
  const turns = turnsList.sort((a, b) => a.turnNumber - b.turnNumber);

  return { session, turns };
}

// Main entry point - try multiple sources to get the best data
// Priority: nodes (has hierarchical structure) > spans (may be flat)
function buildTreeData(
  spans?: Span[],
  nodes?: HierarchyNode[],
  _hiddenSpanTypes?: Set<string>
): { session: TreeSession | null; turns: TreeTurn[] } {
  // Try nodes first - they have the pre-built hierarchy with tools as children
  if (nodes && nodes.length > 0) {
    const fromNodes = buildTreeFromNodes(nodes);
    // Check if we got meaningful data (turns with tools)
    const hasTools = fromNodes.turns.some(t => t.tools.length > 0);
    if (fromNodes.turns.length > 0 && hasTools) {
      return fromNodes;
    }
  }

  // Fall back to spans - may not have children populated
  if (spans && spans.length > 0) {
    const fromSpans = buildTreeFromSpans(spans, undefined);
    if (fromSpans.turns.length > 0) {
      return fromSpans;
    }
  }

  // If nodes exist but had no typed turns, use them anyway (treat all as turns)
  if (nodes && nodes.length > 0) {
    return buildTreeFromNodes(nodes);
  }

  return { session: null, turns: [] };
}

export const EvolutionTree: React.FC<EvolutionTreeProps> = ({
  spans,
  nodes,
  selectedNodeId,
  onNodeClick,
  hiddenSpanTypes,
  isExpanded,
}) => {
  const containerRef = useRef<HTMLDivElement>(null);
  const svgRef = useRef<SVGSVGElement>(null);
  const [hoveredId, setHoveredId] = useState<string | null>(null);
  const [showLegend, setShowLegend] = useState(false);
  const [showControls, setShowControls] = useState(false);

  // Bioluminescent popup state for tools
  const [toolPopup, setToolPopup] = useState<{
    node: SharedToolNode;
    pos: { x: number; y: number };
    metrics: {
      totalDuration: number;
      avgDuration: number;
      minDuration: number;
      maxDuration: number;
      totalCost: number;
      errorCount: number;
      errorRate: number;
      sortedUsages: { turnIndex: number; turnId: string; tool: TreeTool }[];
    };
  } | null>(null);
  const [usageExpanded, setUsageExpanded] = useState(true);

  // Bioluminescent popup state for turns
  const [turnPopup, setTurnPopup] = useState<{
    turn: TreeTurn;
    pos: { x: number; y: number };
    isAnomaly: boolean;
    activity: number;
  } | null>(null);
  const [turnToolsExpanded, setTurnToolsExpanded] = useState(true);

  // Layout tuning controls - auto-calculated defaults, user adjustable
  const [spiralTightness, setSpiralTightness] = useState(50); // 0-100, controls turns per rotation
  const [toolRingScale, setToolRingScale] = useState(50);     // 0-100, tool ring radius
  const [nodeScale, setNodeScale] = useState(50);             // 0-100, node sizes
  const [showAllTools, setShowAllTools] = useState(false);    // Show all tools or just frequent ones

  // Zoom and pan state
  const [zoom, setZoom] = useState(0.5); // Start zoomed out to fit
  const [panOffset, setPanOffset] = useState({ x: 0, y: 0 });
  const [isPanning, setIsPanning] = useState(false);
  const [panStart, setPanStart] = useState({ x: 0, y: 0 });

  // Build tree data - uses spans directly (ignoring display filters)
  const { session, turns } = useMemo(
    () => buildTreeData(spans, nodes, hiddenSpanTypes),
    [spans, nodes, hiddenSpanTypes]
  );

  // Layout constants - dynamically calculated based on turn count and controls
  const SVG_SIZE = 900;
  const CENTER_X = SVG_SIZE / 2;
  const CENTER_Y = SVG_SIZE / 2;

  // Calculate optimal layout based on number of turns
  const layoutParams = useMemo(() => {
    const turnCount = turns.length;
    const maxSvgRadius = SVG_SIZE / 2 - 50; // Leave margin

    // Auto-fit: adjust density based on turn count
    // Fewer turns = tighter spiral, more turns = wider spread
    const autoTurnsPerRotation = Math.max(6, Math.min(20, turnCount / 2.5));

    // Apply user control (0-100 maps to 0.5x - 2x of auto value)
    const tightnessMultiplier = 0.5 + (spiralTightness / 100) * 1.5;
    const turnsPerRotation = autoTurnsPerRotation * tightnessMultiplier;

    // Calculate radii to fit spiral nicely
    // More turns = larger spiral radius range
    const rotations = turnCount / turnsPerRotation;
    const idealMaxRadius = Math.min(maxSvgRadius, 100 + rotations * 80);

    // Tool ring scales from 40-120 based on control
    const toolRingRadius = 40 + (toolRingScale / 100) * 80;

    // Min radius starts just outside tool ring
    const minRadius = Math.max(toolRingRadius + 30, 70);

    // Max radius fills available space
    const maxRadius = Math.max(minRadius + 100, Math.min(idealMaxRadius, maxSvgRadius));

    // Node scale: 0.5x to 1.5x
    const nodeMultiplier = 0.5 + (nodeScale / 100);

    return {
      turnsPerRotation,
      toolRingRadius,
      minRadius,
      maxRadius,
      nodeMultiplier,
    };
  }, [turns.length, spiralTightness, toolRingScale, nodeScale]);

  const { turnsPerRotation, toolRingRadius: TOOL_RING_RADIUS, minRadius: MIN_RADIUS, maxRadius: MAX_RADIUS, nodeMultiplier } = layoutParams;

  // Calculate spiral positions for turns
  const spiralPositions = useMemo(() =>
    calculateSpiralPositions(turns, CENTER_X, CENTER_Y, MIN_RADIUS, MAX_RADIUS, turnsPerRotation),
    [turns, MIN_RADIUS, MAX_RADIUS, turnsPerRotation]
  );

  // Generate spiral stem path
  const stemPath = useMemo(() =>
    generateSpiralPath(spiralPositions, CENTER_X, CENTER_Y),
    [spiralPositions]
  );

  // Build shared tool nodes
  const sharedToolNodes = useMemo(() =>
    buildSharedToolNodes(turns, spiralPositions, CENTER_X, CENTER_Y, TOOL_RING_RADIUS, showAllTools),
    [turns, spiralPositions, TOOL_RING_RADIUS, showAllTools]
  );

  // Count total unique tools for display
  const totalUniqueTools = useMemo(() => {
    const toolNames = new Set<string>();
    turns.forEach(turn => turn.tools.forEach(tool => toolNames.add(tool.name)));
    return toolNames.size;
  }, [turns]);

  // Create lookup from tool name to shared node
  const toolNodeLookup = useMemo(() => {
    const lookup = new Map<string, SharedToolNode>();
    sharedToolNodes.forEach(node => lookup.set(node.name, node));
    return lookup;
  }, [sharedToolNodes]);

  // Zoom handlers
  const handleZoomIn = useCallback(() => {
    setZoom(z => Math.min(z * 1.3, 3));
  }, []);

  const handleZoomOut = useCallback(() => {
    setZoom(z => Math.max(z / 1.3, 0.1));
  }, []);

  const handleZoomFit = useCallback(() => {
    // Calculate zoom to fit square spiral in container
    if (!containerRef.current) return;
    const containerHeight = containerRef.current.clientHeight - 100; // Account for header/footer
    const containerWidth = containerRef.current.clientWidth;
    const minDim = Math.min(containerHeight, containerWidth);
    const fitZoom = minDim / SVG_SIZE;
    setZoom(Math.max(Math.min(fitZoom, 1.2), 0.3));
    setPanOffset({ x: 0, y: 0 });
  }, []);

  // Mouse wheel zoom
  const handleWheel = useCallback((e: React.WheelEvent) => {
    e.preventDefault();
    const delta = e.deltaY > 0 ? 0.9 : 1.1;
    setZoom(z => Math.min(Math.max(z * delta, 0.1), 3));
  }, []);

  // Pan handlers
  const handleMouseDown = useCallback((e: React.MouseEvent) => {
    if (e.button === 0) { // Left click
      setIsPanning(true);
      setPanStart({ x: e.clientX - panOffset.x, y: e.clientY - panOffset.y });
    }
  }, [panOffset]);

  const handleMouseMove = useCallback((e: React.MouseEvent) => {
    if (isPanning) {
      setPanOffset({
        x: e.clientX - panStart.x,
        y: e.clientY - panStart.y,
      });
    }
  }, [isPanning, panStart]);

  const handleMouseUp = useCallback(() => {
    setIsPanning(false);
  }, []);

  // Auto-fit zoom on initial load or when turns change significantly
  useEffect(() => {
    if (spiralPositions.length > 0) {
      // Delay to let container render
      const timer = setTimeout(() => {
        handleZoomFit();
      }, 200);
      return () => clearTimeout(timer);
    }
  }, [spiralPositions.length > 0 ? 'has-data' : 'no-data', handleZoomFit]);

  // Handle node click - show bioluminescent popup for turns
  const handleNodeClick = useCallback((turn: TreeTurn, event: React.MouseEvent, isAnomaly: boolean, activity: number) => {
    event.stopPropagation();

    // Position popup near the click, but constrained to viewport
    const x = Math.min(event.clientX + 20, window.innerWidth - 400);
    const y = Math.min(event.clientY - 50, window.innerHeight - 450);

    setTurnPopup({
      turn,
      pos: { x: Math.max(20, x), y: Math.max(20, y) },
      isAnomaly,
      activity,
    });
  }, []);

  const handleToolClick = useCallback((tool: TreeTool, event: React.MouseEvent) => {
    event.stopPropagation();
    if (onNodeClick) {
      const node: HierarchyNode = {
        id: tool.id,
        type: 'tool_use',
        label: tool.name,
        status: tool.status === 'ok' ? 'completed' : 'error',
        durationMs: tool.durationMs,
        cost: tool.cost,
      };
      onNodeClick(node, event);
    }
  }, [onNodeClick]);

  // Handle click on shared tool node - shows bioluminescent popup
  const handleSharedToolClick = useCallback((toolNode: SharedToolNode, event: React.MouseEvent) => {
    event.stopPropagation();

    // Calculate metrics for the popup
    const totalDuration = toolNode.usages.reduce((sum, u) => sum + u.tool.durationMs, 0);
    const totalCost = toolNode.usages.reduce((sum, u) => sum + (u.tool.cost || 0), 0);
    const avgDuration = totalDuration / toolNode.usages.length;
    const errorCount = toolNode.usages.filter(u => u.tool.status === 'error').length;
    const minDuration = Math.min(...toolNode.usages.map(u => u.tool.durationMs));
    const maxDuration = Math.max(...toolNode.usages.map(u => u.tool.durationMs));
    const errorRate = errorCount / toolNode.usages.length;

    // Sort usages by turn number for display
    const sortedUsages = [...toolNode.usages].sort((a, b) => a.turnIndex - b.turnIndex);

    // Position popup near the click, but constrained to viewport
    const x = Math.min(event.clientX + 20, window.innerWidth - 400);
    const y = Math.min(event.clientY - 50, window.innerHeight - 400);

    setToolPopup({
      node: toolNode,
      pos: { x: Math.max(20, x), y: Math.max(20, y) },
      metrics: {
        totalDuration,
        avgDuration,
        minDuration,
        maxDuration,
        totalCost,
        errorCount,
        errorRate,
        sortedUsages,
      },
    });
  }, []);

  // Close popups when clicking outside
  const handleContainerClick = useCallback((event: React.MouseEvent) => {
    // Only close if clicking on the container itself (not children)
    if (event.target === event.currentTarget || (event.target as HTMLElement).tagName === 'svg') {
      setToolPopup(null);
      setTurnPopup(null);
    }
  }, []);

  // Format helpers
  const formatDuration = (ms: number) => {
    if (ms < 1000) return `${ms}ms`;
    if (ms < 60000) return `${(ms / 1000).toFixed(1)}s`;
    return `${(ms / 60000).toFixed(1)}m`;
  };

  const formatCost = (cost: number) => {
    if (cost === 0) return '';
    if (cost < 0.01) return `$${cost.toFixed(4)}`;
    return `$${cost.toFixed(3)}`;
  };

  if (!session || turns.length === 0) {
    return (
      <div className={styles.emptyState}>
        <div className={styles.emptyIcon}>🌱</div>
        <div className={styles.emptyTitle}>Select an Event</div>
        <div className={styles.emptyText}>
          Watch the execution tree grow as you explore events
        </div>
      </div>
    );
  }

  return (
    <div
      ref={containerRef}
      className={styles.container}
      onWheel={handleWheel}
      onMouseDown={handleMouseDown}
      onMouseMove={handleMouseMove}
      onMouseUp={handleMouseUp}
      onMouseLeave={handleMouseUp}
      onClick={handleContainerClick}
      style={{ cursor: isPanning ? 'grabbing' : 'grab' }}
    >
      {/* Background ambient glow */}
      <div className={styles.ambientGlow} />

      {/* Session header with zoom controls */}
      <div className={styles.sessionHeader}>
        <div className={styles.sessionIcon}>◉</div>
        <div className={styles.sessionInfo}>
          <span className={styles.sessionName}>{session.name}</span>
          <span className={styles.sessionMeta}>
            {turns.length} turns • {formatDuration(session.durationMs)}
            {session.cost > 0 && ` • ${formatCost(session.cost)}`}
          </span>
        </div>
        {/* Zoom controls */}
        <div className={styles.zoomControls}>
          <button onClick={handleZoomOut} title="Zoom out">−</button>
          <span className={styles.zoomLevel}>{Math.round(zoom * 100)}%</span>
          <button onClick={handleZoomIn} title="Zoom in">+</button>
          <button onClick={handleZoomFit} title="Fit to view">⊡</button>
          <button onClick={() => setShowControls(!showControls)} title="Layout controls">⚙</button>
          <button onClick={() => setShowLegend(!showLegend)} title="Show legend">?</button>
        </div>
      </div>

      {/* Legend panel */}
      {showLegend && (
        <div className={styles.legendPanel}>
          <div className={styles.legendTitle}>Turns (Outer Spiral)</div>
          <div className={styles.legendItem}>
            <span className={styles.legendDot} style={{ background: '#10b981' }} />
            <span>Normal turn</span>
          </div>
          <div className={styles.legendItem}>
            <span className={styles.legendDot} style={{ background: '#f59e0b' }} />
            <span>Anomaly (&gt;2σ)</span>
          </div>
          <div className={styles.legendItem}>
            <span className={styles.legendDot} style={{ background: '#ef4444' }} />
            <span>Error</span>
          </div>
          <div className={styles.legendDivider} />
          <div className={styles.legendTitle}>Tools (Inner Rings)</div>
          <div className={styles.legendHint} style={{ marginBottom: '6px' }}>
            Grouped by type into concentric rings
          </div>
          <div className={styles.legendItem}>
            <span className={styles.legendDot} style={{ background: '#60a5fa' }} />
            <span>Read/Edit/Write (innermost)</span>
          </div>
          <div className={styles.legendItem}>
            <span className={styles.legendDot} style={{ background: '#fbbf24' }} />
            <span>Bash/Grep/Glob (middle)</span>
          </div>
          <div className={styles.legendItem}>
            <span className={styles.legendDot} style={{ background: '#fb923c' }} />
            <span>Task/Web (outer)</span>
          </div>
          <div className={styles.legendDivider} />
          <div className={styles.legendHint}>
            Only tools used 2+ times shown by default
          </div>
          <div className={styles.legendHint}>
            Use ⚙ controls to show all tools
          </div>
        </div>
      )}

      {/* Controls panel */}
      {showControls && (
        <div className={styles.controlsPanel}>
          <div className={styles.legendTitle}>Layout Controls</div>

          <div className={styles.controlItem}>
            <label>Spiral Tightness</label>
            <input
              type="range"
              min="0"
              max="100"
              value={spiralTightness}
              onChange={(e) => setSpiralTightness(parseInt(e.target.value))}
            />
            <span>{spiralTightness}</span>
          </div>

          <div className={styles.controlItem}>
            <label>Tool Ring Size</label>
            <input
              type="range"
              min="0"
              max="100"
              value={toolRingScale}
              onChange={(e) => setToolRingScale(parseInt(e.target.value))}
            />
            <span>{toolRingScale}</span>
          </div>

          <div className={styles.controlItem}>
            <label>Node Size</label>
            <input
              type="range"
              min="0"
              max="100"
              value={nodeScale}
              onChange={(e) => setNodeScale(parseInt(e.target.value))}
            />
            <span>{nodeScale}</span>
          </div>

          <div className={styles.legendDivider} />

          <div className={styles.controlItem}>
            <label style={{ display: 'flex', alignItems: 'center', gap: '8px', cursor: 'pointer' }}>
              <input
                type="checkbox"
                checked={showAllTools}
                onChange={(e) => setShowAllTools(e.target.checked)}
                style={{ cursor: 'pointer' }}
              />
              Show all tools ({totalUniqueTools})
            </label>
            <span style={{ fontSize: '9px', color: '#6b7280' }}>
              {showAllTools ? 'Showing all' : `Showing ${sharedToolNodes.length} (used 2+ times)`}
            </span>
          </div>

          <div className={styles.legendDivider} />
          <div className={styles.legendHint}>
            Turns/rotation: {turnsPerRotation.toFixed(1)}
          </div>
          <div className={styles.legendHint}>
            Radii: {MIN_RADIUS.toFixed(0)} - {MAX_RADIUS.toFixed(0)}
          </div>
        </div>
      )}

      {/* SVG Spiral with zoom/pan transform */}
      <svg
        ref={svgRef}
        className={styles.treeSvg}
        viewBox={`0 0 ${SVG_SIZE} ${SVG_SIZE}`}
        preserveAspectRatio="xMidYMid meet"
        style={{
          transform: `scale(${zoom}) translate(${panOffset.x / zoom}px, ${panOffset.y / zoom}px)`,
          transformOrigin: 'center center',
          maxHeight: '100%',
        }}
      >
        <defs>
          {/* Stem gradient */}
          <linearGradient id="stemGradient" x1="0%" y1="0%" x2="0%" y2="100%">
            <stop offset="0%" stopColor="#06b6d4" />
            <stop offset="50%" stopColor="#10b981" />
            <stop offset="100%" stopColor="#059669" />
          </linearGradient>

          {/* Glow filter */}
          <filter id="glow" x="-50%" y="-50%" width="200%" height="200%">
            <feGaussianBlur stdDeviation="4" result="blur" />
            <feMerge>
              <feMergeNode in="blur" />
              <feMergeNode in="SourceGraphic" />
            </feMerge>
          </filter>

          {/* Error glow */}
          <filter id="errorGlow" x="-50%" y="-50%" width="200%" height="200%">
            <feGaussianBlur stdDeviation="6" result="blur" />
            <feFlood floodColor="#ef4444" floodOpacity="0.5" />
            <feComposite in2="blur" operator="in" />
            <feMerge>
              <feMergeNode />
              <feMergeNode in="SourceGraphic" />
            </feMerge>
          </filter>

          {/* Tool bloom filter for hover state */}
          <filter id="tool-bloom" x="-50%" y="-50%" width="200%" height="200%">
            <feGaussianBlur in="SourceGraphic" stdDeviation="3" result="blur" />
            <feMerge>
              <feMergeNode in="blur" />
              <feMergeNode in="blur" />
              <feMergeNode in="SourceGraphic" />
            </feMerge>
          </filter>

          {/* Dynamic glow gradients for each tool */}
          {sharedToolNodes.map(tool => {
            const errorCount = tool.usages.filter(u => u.tool.status === 'error').length;
            const errorRate = errorCount / tool.usages.length;
            const baseHue = 160; // Emerald
            const errorHue = 20;  // Coral/red
            const hue = baseHue - errorRate * (baseHue - errorHue);
            return (
              <radialGradient
                key={tool.name}
                id={`glow-${tool.name.replace(/[^a-zA-Z0-9]/g, '')}`}
              >
                <stop offset="0%" stopColor={`hsl(${hue}, 85%, 55%)`} stopOpacity="0.5" />
                <stop offset="60%" stopColor={`hsl(${hue}, 85%, 55%)`} stopOpacity="0.15" />
                <stop offset="100%" stopColor={`hsl(${hue}, 85%, 55%)`} stopOpacity="0" />
              </radialGradient>
            );
          })}
        </defs>

        {/* Main stem path - simplified, no filter for performance */}
        <path
          d={stemPath}
          fill="none"
          stroke="url(#stemGradient)"
          strokeWidth="3"
          strokeLinecap="round"
          opacity="0.7"
        />

        {/* Center node (origin) - simplified */}
        <circle
          cx={CENTER_X}
          cy={CENTER_Y}
          r={10}
          fill="#06b6d4"
        />

        {/* Edges from turns to shared tool nodes - drawn first (underneath) */}
        <g className={styles.toolEdges}>
          {spiralPositions.map((pos) => {
            const { x, y, turn } = pos;
            const isHovered = hoveredId === turn.id;

            return turn.tools.map((tool) => {
              const sharedNode = toolNodeLookup.get(tool.name);
              if (!sharedNode) return null;

              const isEdgeHighlighted = isHovered || hoveredId === sharedNode.name;

              return (
                <line
                  key={`${turn.id}-${tool.id}`}
                  x1={x} y1={y}
                  x2={sharedNode.x} y2={sharedNode.y}
                  stroke={sharedNode.color}
                  strokeWidth={isEdgeHighlighted ? 1.5 : 0.5}
                  opacity={isEdgeHighlighted ? 0.8 : 0.15}
                  strokeDasharray={tool.status === 'error' ? '3,2' : 'none'}
                />
              );
            });
          })}
        </g>

        {/* Shared tool nodes in interior ring - bioluminescent style */}
        {sharedToolNodes.map((toolNode) => {
          const isToolHovered = hoveredId === toolNode.name;
          const usageCount = toolNode.usages.length;

          // Calculate metrics for bioluminescent encoding
          const totalDuration = toolNode.usages.reduce((sum, u) => sum + u.tool.durationMs, 0);
          const errorCount = toolNode.usages.filter(u => u.tool.status === 'error').length;
          const errorRate = errorCount / usageCount;
          const lastUsedTurn = Math.max(...toolNode.usages.map(u => u.turnIndex));
          const firstUsedTurn = Math.min(...toolNode.usages.map(u => u.turnIndex));
          const turnSpread = lastUsedTurn - firstUsedTurn + 1;

          // Normalize metrics
          const maxDuration = Math.max(...sharedToolNodes.map(t =>
            t.usages.reduce((sum, u) => sum + u.tool.durationMs, 0)
          ), 1);
          const normalizedDuration = totalDuration / maxDuration;

          // Glow intensity based on duration (0.3 - 1.0)
          const glowIntensity = 0.3 + normalizedDuration * 0.7;

          // Pulse speed based on recency (recent = faster, 2-6 seconds)
          const recency = turns.length > 0 ? 1 - (lastUsedTurn / (turns.length - 1 || 1)) : 0.5;
          const pulseSpeed = 6 - recency * 4;

          // Color temperature: emerald (0 errors) → amber → coral (many errors)
          const baseHue = 160; // Emerald
          const errorHue = 20;  // Coral/red
          const hue = baseHue - errorRate * (baseHue - errorHue);
          const glowColor = `hsla(${hue}, 85%, 55%, ${glowIntensity})`;
          const baseColor = `hsl(${hue}, 85%, 55%)`;

          // Orbital rings: 1-3 based on usage tiers
          const orbitalRings = usageCount > 10 ? 3 : usageCount > 4 ? 2 : 1;

          // Arc sweep based on turn spread (what % of session this tool spans)
          const arcSweep = turns.length > 1 ? Math.min((turnSpread / turns.length) * 360, 340) : 0;

          // Node size based on usage count
          const nodeSize = Math.min((5 + usageCount * 1.5) * nodeMultiplier, 14 * nodeMultiplier);

          return (
            <g
              key={toolNode.name}
              style={{
                '--pulse-speed': `${pulseSpeed}s`,
                '--glow-intensity': glowIntensity,
              } as React.CSSProperties}
            >
              {/* Outer glow aura - breathes */}
              <circle
                cx={toolNode.x}
                cy={toolNode.y}
                r={nodeSize + 10}
                fill={`url(#glow-${toolNode.name.replace(/[^a-zA-Z0-9]/g, '')})`}
                className={styles.toolAura}
                style={{ opacity: glowIntensity * 0.4 }}
              />

              {/* Orbital rings showing usage tiers */}
              {Array.from({ length: orbitalRings }).map((_, i) => (
                <circle
                  key={i}
                  cx={toolNode.x}
                  cy={toolNode.y}
                  r={nodeSize + 3 + i * 5}
                  fill="none"
                  stroke={baseColor}
                  strokeWidth={0.5}
                  strokeOpacity={0.3 - i * 0.08}
                  strokeDasharray="2,4"
                  className={styles.orbitalRing}
                  style={{ animationDelay: `${i * 0.3}s` }}
                />
              ))}

              {/* Turn spread arc - shows temporal footprint */}
              {arcSweep > 30 && (
                <path
                  d={describeArc(toolNode.x, toolNode.y, nodeSize + 6, -90, -90 + arcSweep)}
                  fill="none"
                  stroke={baseColor}
                  strokeWidth={2}
                  strokeOpacity={0.5}
                  strokeLinecap="round"
                  className={styles.spreadArc}
                />
              )}

              {/* Core node - the nucleus */}
              <circle
                cx={toolNode.x}
                cy={toolNode.y}
                r={isToolHovered ? nodeSize + 2 : nodeSize}
                fill={errorRate > 0.3 ? baseColor : toolNode.color}
                className={styles.toolCore}
                filter={isToolHovered ? 'url(#glow)' : undefined}
                onClick={(e) => handleSharedToolClick(toolNode, e)}
                onMouseEnter={() => setHoveredId(toolNode.name)}
                onMouseLeave={() => setHoveredId(null)}
                style={{ cursor: 'pointer' }}
              />

              {/* Inner pulse ring */}
              <circle
                cx={toolNode.x}
                cy={toolNode.y}
                r={nodeSize - 2}
                fill="none"
                stroke="rgba(255,255,255,0.3)"
                strokeWidth={0.75}
                className={styles.innerPulse}
              />

              {/* Error indicator - warm glow overlay */}
              {errorRate > 0 && (
                <circle
                  cx={toolNode.x}
                  cy={toolNode.y}
                  r={nodeSize + 2}
                  fill="none"
                  stroke="#ef4444"
                  strokeWidth={1.5}
                  strokeOpacity={errorRate * 0.8}
                  strokeDasharray="3,3"
                  className={styles.errorIndicator}
                />
              )}

              {/* Usage count - visible when count > 1 or on hover */}
              {(usageCount > 1 || isToolHovered) && (
                <text
                  x={toolNode.x}
                  y={toolNode.y + 3}
                  textAnchor="middle"
                  className={styles.toolUsageCount}
                  style={{ opacity: isToolHovered ? 1 : 0.9 }}
                >
                  {usageCount}
                </text>
              )}

              {/* Tool label - visible on hover or for high-usage tools */}
              {(isToolHovered || usageCount > 3) && (
                <text
                  x={toolNode.x}
                  y={toolNode.y + nodeSize + 12}
                  textAnchor="middle"
                  className={`${styles.toolHoverLabel} ${isToolHovered ? styles.visible : ''}`}
                  style={{
                    opacity: isToolHovered ? 1 : 0.6,
                    fill: baseColor,
                  }}
                >
                  {toolNode.displayName}
                </text>
              )}
            </g>
          );
        })}

        {/* Turn nodes along spiral */}
        {(() => {
          // Calculate max metrics for normalization
          // Use tokensOut for size, but fallback to duration if no token data
          const maxTokensOut = Math.max(...turns.map(t => t.tokensOut || 0), 1);
          const maxTokensIn = Math.max(...turns.map(t => t.tokensIn || 0), 1);
          const maxDuration = Math.max(...turns.map(t => t.durationMs || 0), 1);

          // Check if we have meaningful token data (at least some variance)
          const hasTokenOutData = maxTokensOut > 10 && turns.some(t => (t.tokensOut || 0) !== maxTokensOut);
          const hasTokenInData = maxTokensIn > 10 && turns.some(t => (t.tokensIn || 0) !== maxTokensIn);

          return spiralPositions.map((pos) => {
            const { x, y, turn, isAnomaly, activity } = pos;
            const isHovered = hoveredId === turn.id;
            const isSelected = selectedNodeId === turn.id;
            const isError = turn.status === 'error';

            // Size based on tokens OUT, or fallback to duration (3-18 range for more visible difference)
            const sizeMetric = hasTokenOutData
              ? (turn.tokensOut || 0) / maxTokensOut
              : (turn.durationMs || 0) / maxDuration;
            const baseSize = (3 + sizeMetric * 15) * nodeMultiplier;
            const nodeSize = isAnomaly ? baseSize + 3 : isError ? baseSize + 2 : baseSize;

            // Pulse speed based on tokens IN, or fallback to duration (more = faster pulse, 1-4 seconds)
            const pulseMetric = hasTokenInData
              ? (turn.tokensIn || 0) / maxTokensIn
              : (turn.durationMs || 0) / maxDuration;
            const pulseSpeed = 4 - pulseMetric * 3; // 4s for low input, 1s for high input
            const pulseIntensity = 0.3 + pulseMetric * 0.7; // 0.3-1.0 opacity

            // Determine node color
            const nodeColor = isAnomaly ? '#f59e0b' : isError ? '#ef4444' : '#10b981';

            return (
              <g
                key={turn.id}
                style={{
                  '--turn-pulse-speed': `${pulseSpeed}s`,
                  '--turn-pulse-intensity': pulseIntensity,
                } as React.CSSProperties}
              >
                {/* Outer pulse ring - breathing based on tokens in / duration */}
                {pulseMetric > 0.1 && (
                  <circle
                    cx={x} cy={y}
                    r={nodeSize + 6}
                    fill="none"
                    stroke={nodeColor}
                    strokeWidth={1.5}
                    className={styles.turnPulseRing}
                  />
                )}

                {/* Secondary pulse ring for high input / long duration */}
                {pulseMetric > 0.5 && (
                  <circle
                    cx={x} cy={y}
                    r={nodeSize + 10}
                    fill="none"
                    stroke={nodeColor}
                    strokeWidth={0.75}
                    className={styles.turnPulseRingOuter}
                  />
                )}

                {/* Main node */}
                <circle
                  cx={x} cy={y}
                  r={isHovered ? nodeSize + 2 : nodeSize}
                  fill={nodeColor}
                  stroke={isHovered || isSelected ? '#fff' : 'none'}
                  strokeWidth={isHovered || isSelected ? 2 : 0}
                  opacity={isAnomaly || isError ? 1 : 0.9}
                  onClick={(e) => handleNodeClick(turn, e, isAnomaly, activity)}
                  onMouseEnter={() => setHoveredId(turn.id)}
                  onMouseLeave={() => setHoveredId(null)}
                  style={{ cursor: 'pointer' }}
                />

                {/* Inner glow for high output / long duration turns */}
                {sizeMetric > 0.3 && (
                  <circle
                    cx={x} cy={y}
                    r={nodeSize - 2}
                    fill="none"
                    stroke="rgba(255,255,255,0.4)"
                    strokeWidth={0.75}
                    className={styles.turnInnerGlow}
                  />
                )}

                {/* Turn number - show on hover or for anomalies */}
                {(isHovered || isAnomaly || isSelected) && (
                  <text
                    x={x} y={y + nodeSize + 12}
                    textAnchor="middle"
                    fontSize="8"
                    fill="#e6edf3"
                    fontWeight="600"
                  >
                    T{turn.turnNumber}
                  </text>
                )}
              </g>
            );
          });
        })()}
      </svg>

      {/* Stats footer */}
      <div className={styles.statsFooter}>
        <div className={styles.stat}>
          <span className={styles.statValue}>{turns.length}</span>
          <span className={styles.statLabel}>turns</span>
        </div>
        <div className={styles.stat}>
          <span className={styles.statValue}>
            {sharedToolNodes.length}{totalUniqueTools > sharedToolNodes.length ? `/${totalUniqueTools}` : ''}
          </span>
          <span className={styles.statLabel}>tools shown</span>
        </div>
        <div className={styles.stat}>
          <span className={styles.statValue}>
            {turns.reduce((sum, t) => sum + t.tools.length, 0)}
          </span>
          <span className={styles.statLabel}>total calls</span>
        </div>
        {session.cost > 0 && (
          <div className={styles.stat}>
            <span className={styles.statValue}>{formatCost(session.cost)}</span>
            <span className={styles.statLabel}>cost</span>
          </div>
        )}
      </div>

      {/* Bioluminescent Tool Popup */}
      {toolPopup && (
        <div
          className={styles.bioPopover}
          style={{
            left: toolPopup.pos.x,
            top: toolPopup.pos.y,
          }}
          onClick={(e) => e.stopPropagation()}
        >
          {/* Ambient glow effect */}
          <div className={styles.bioPopoverGlow} />

          {/* Header with tool info */}
          <div className={styles.bioPopoverHeader}>
            <div
              className={styles.bioPopoverIcon}
              style={{
                backgroundColor: toolPopup.node.color,
                boxShadow: `0 0 20px ${toolPopup.node.color}60`,
              }}
            />
            <div className={styles.bioPopoverTitle}>
              <span className={styles.bioPopoverToolType}>{toolPopup.node.toolType}</span>
              <span className={styles.bioPopoverToolName}>{toolPopup.node.displayName}</span>
            </div>
            <button
              className={styles.bioPopoverClose}
              onClick={() => setToolPopup(null)}
              aria-label="Close"
            >
              ×
            </button>
          </div>

          {/* Full name if different from display name */}
          {toolPopup.node.name !== toolPopup.node.displayName && (
            <div className={styles.bioPopoverFullName}>
              {toolPopup.node.name}
            </div>
          )}

          {/* Metrics grid */}
          <div className={styles.bioPopoverMetrics}>
            {/* Usage count */}
            <div className={styles.bioMetric}>
              <span className={styles.bioMetricValue}>{toolPopup.node.usages.length}</span>
              <span className={styles.bioMetricLabel}>usages</span>
            </div>

            {/* Duration */}
            <div className={styles.bioMetric}>
              <span className={styles.bioMetricValue}>{formatDuration(toolPopup.metrics.totalDuration)}</span>
              <span className={styles.bioMetricLabel}>total time</span>
            </div>

            {/* Average duration */}
            <div className={styles.bioMetric}>
              <span className={styles.bioMetricValue}>{formatDuration(toolPopup.metrics.avgDuration)}</span>
              <span className={styles.bioMetricLabel}>avg</span>
            </div>

            {/* Cost if available */}
            {toolPopup.metrics.totalCost > 0 && (
              <div className={styles.bioMetric}>
                <span className={styles.bioMetricValue}>{formatCost(toolPopup.metrics.totalCost)}</span>
                <span className={styles.bioMetricLabel}>cost</span>
              </div>
            )}

            {/* Errors */}
            {toolPopup.metrics.errorCount > 0 && (
              <div className={`${styles.bioMetric} ${styles.bioMetricError}`}>
                <span className={styles.bioMetricValue}>{toolPopup.metrics.errorCount}</span>
                <span className={styles.bioMetricLabel}>errors</span>
              </div>
            )}
          </div>

          {/* Duration range bar */}
          <div className={styles.bioDurationBar}>
            <div className={styles.bioDurationBarLabel}>
              <span>{formatDuration(toolPopup.metrics.minDuration)}</span>
              <span className={styles.bioDurationBarTitle}>duration range</span>
              <span>{formatDuration(toolPopup.metrics.maxDuration)}</span>
            </div>
            <div className={styles.bioDurationBarTrack}>
              <div
                className={styles.bioDurationBarFill}
                style={{
                  width: toolPopup.metrics.maxDuration > 0
                    ? `${Math.min(100, (toolPopup.metrics.avgDuration / toolPopup.metrics.maxDuration) * 100)}%`
                    : '50%',
                  backgroundColor: toolPopup.node.color,
                }}
              />
            </div>
          </div>

          {/* Turn timeline */}
          <div className={styles.bioTurnTimeline}>
            <button
              className={styles.bioTimelineToggle}
              onClick={() => setUsageExpanded(!usageExpanded)}
            >
              <span className={styles.bioTimelineToggleIcon}>{usageExpanded ? '▼' : '▶'}</span>
              <span>Turn Usage Timeline</span>
              <span className={styles.bioTimelineCount}>{toolPopup.metrics.sortedUsages.length} calls</span>
            </button>

            {usageExpanded && (
              <div className={styles.bioTimelineContent}>
                {toolPopup.metrics.sortedUsages.slice(0, 20).map((usage, idx) => (
                  <div
                    key={`${usage.turnId}-${idx}`}
                    className={`${styles.bioTimelineItem} ${usage.tool.status === 'error' ? styles.bioTimelineItemError : ''}`}
                  >
                    <span className={styles.bioTimelineTurn}>T{usage.turnIndex + 1}</span>
                    <span className={styles.bioTimelineDuration}>{formatDuration(usage.tool.durationMs)}</span>
                    {usage.tool.cost && usage.tool.cost > 0 && (
                      <span className={styles.bioTimelineCost}>{formatCost(usage.tool.cost)}</span>
                    )}
                    <span className={styles.bioTimelineStatus}>
                      {usage.tool.status === 'error' ? '✗' : '✓'}
                    </span>
                  </div>
                ))}
                {toolPopup.metrics.sortedUsages.length > 20 && (
                  <div className={styles.bioTimelineMore}>
                    +{toolPopup.metrics.sortedUsages.length - 20} more usages
                  </div>
                )}
              </div>
            )}
          </div>
        </div>
      )}

      {/* Bioluminescent Turn Popup */}
      {turnPopup && (
        <div
          className={styles.bioPopover}
          style={{
            left: turnPopup.pos.x,
            top: turnPopup.pos.y,
          }}
          onClick={(e) => e.stopPropagation()}
        >
          {/* Ambient glow effect */}
          <div
            className={styles.bioPopoverGlow}
            style={{
              background: turnPopup.isAnomaly
                ? 'radial-gradient(ellipse at 50% 0%, rgba(245, 158, 11, 0.2) 0%, transparent 70%)'
                : turnPopup.turn.status === 'error'
                ? 'radial-gradient(ellipse at 50% 0%, rgba(239, 68, 68, 0.2) 0%, transparent 70%)'
                : undefined,
            }}
          />

          {/* Header with turn info */}
          <div className={styles.bioPopoverHeader}>
            <div
              className={styles.bioPopoverIcon}
              style={{
                backgroundColor: turnPopup.isAnomaly ? '#f59e0b' : turnPopup.turn.status === 'error' ? '#ef4444' : '#10b981',
                boxShadow: `0 0 20px ${turnPopup.isAnomaly ? '#f59e0b' : turnPopup.turn.status === 'error' ? '#ef4444' : '#10b981'}60`,
              }}
            >
              <span style={{ fontSize: '14px', fontWeight: 700, color: '#0f1419' }}>
                {turnPopup.turn.turnNumber}
              </span>
            </div>
            <div className={styles.bioPopoverTitle}>
              <span className={styles.bioPopoverToolType}>
                {turnPopup.isAnomaly ? 'ANOMALY TURN' : turnPopup.turn.status === 'error' ? 'ERROR TURN' : 'TURN'}
              </span>
              <span className={styles.bioPopoverToolName}>Turn {turnPopup.turn.turnNumber}</span>
            </div>
            <button
              className={styles.bioPopoverClose}
              onClick={() => setTurnPopup(null)}
              aria-label="Close"
            >
              ×
            </button>
          </div>

          {/* Status badge for anomaly/error */}
          {(turnPopup.isAnomaly || turnPopup.turn.status === 'error') && (
            <div className={styles.turnStatusBadge} style={{
              backgroundColor: turnPopup.isAnomaly ? 'rgba(245, 158, 11, 0.1)' : 'rgba(239, 68, 68, 0.1)',
              borderColor: turnPopup.isAnomaly ? 'rgba(245, 158, 11, 0.3)' : 'rgba(239, 68, 68, 0.3)',
              color: turnPopup.isAnomaly ? '#f59e0b' : '#ef4444',
            }}>
              {turnPopup.isAnomaly ? '⚠ Anomaly detected (>2σ from mean)' : '✗ Error occurred'}
            </div>
          )}

          {/* Metrics grid */}
          <div className={styles.bioPopoverMetrics}>
            {/* Duration */}
            <div className={styles.bioMetric}>
              <span className={styles.bioMetricValue}>{formatDuration(turnPopup.turn.durationMs)}</span>
              <span className={styles.bioMetricLabel}>duration</span>
            </div>

            {/* Tools */}
            <div className={styles.bioMetric}>
              <span className={styles.bioMetricValue}>{turnPopup.turn.tools.length}</span>
              <span className={styles.bioMetricLabel}>tools</span>
            </div>

            {/* Cost if available */}
            {turnPopup.turn.cost > 0 && (
              <div className={styles.bioMetric}>
                <span className={styles.bioMetricValue}>{formatCost(turnPopup.turn.cost)}</span>
                <span className={styles.bioMetricLabel}>cost</span>
              </div>
            )}

            {/* Tokens In */}
            {turnPopup.turn.tokensIn > 0 && (
              <div className={styles.bioMetric}>
                <span className={styles.bioMetricValue}>{turnPopup.turn.tokensIn.toLocaleString()}</span>
                <span className={styles.bioMetricLabel}>tokens in</span>
              </div>
            )}

            {/* Tokens Out */}
            {turnPopup.turn.tokensOut > 0 && (
              <div className={styles.bioMetric}>
                <span className={styles.bioMetricValue}>{turnPopup.turn.tokensOut.toLocaleString()}</span>
                <span className={styles.bioMetricLabel}>tokens out</span>
              </div>
            )}

            {/* Activity level */}
            <div className={styles.bioMetric}>
              <span className={styles.bioMetricValue}>{Math.round(turnPopup.activity * 100)}%</span>
              <span className={styles.bioMetricLabel}>activity</span>
            </div>
          </div>

          {/* Tools list */}
          {turnPopup.turn.tools.length > 0 && (
            <div className={styles.bioTurnTimeline}>
              <button
                className={styles.bioTimelineToggle}
                onClick={() => setTurnToolsExpanded(!turnToolsExpanded)}
              >
                <span className={styles.bioTimelineToggleIcon}>{turnToolsExpanded ? '▼' : '▶'}</span>
                <span>Tools Used</span>
                <span className={styles.bioTimelineCount}>{turnPopup.turn.tools.length} tools</span>
              </button>

              {turnToolsExpanded && (
                <div className={styles.bioTimelineContent}>
                  {turnPopup.turn.tools.map((tool, idx) => {
                    const toolType = getToolType(tool.name);
                    const toolColor = getToolColor(tool.name);
                    return (
                      <div
                        key={`${tool.id}-${idx}`}
                        className={`${styles.bioTimelineItem} ${tool.status === 'error' ? styles.bioTimelineItemError : ''}`}
                        style={{ borderLeftColor: tool.status === 'error' ? undefined : toolColor }}
                      >
                        <span
                          className={styles.bioTimelineTurn}
                          style={{ color: tool.status === 'error' ? undefined : toolColor }}
                        >
                          {toolType}
                        </span>
                        <span className={styles.bioTimelineDuration}>{formatDuration(tool.durationMs)}</span>
                        {tool.cost && tool.cost > 0 && (
                          <span className={styles.bioTimelineCost}>{formatCost(tool.cost)}</span>
                        )}
                        <span className={styles.bioTimelineStatus}>
                          {tool.status === 'error' ? '✗' : '✓'}
                        </span>
                      </div>
                    );
                  })}
                </div>
              )}
            </div>
          )}
        </div>
      )}
    </div>
  );
};

export default EvolutionTree;
