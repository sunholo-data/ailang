/**
 * EvolutionTree builder functions
 * Extracted from EvolutionTree.tsx (PR 5 - M-DASHBOARD-SIMPLIFICATION)
 *
 * Contains:
 * - File hierarchy building
 * - Circle packing layout calculations
 * - Spiral position calculations
 * - Shared tool node building
 * - Span filtering and type detection
 * - Tree data building from nodes/spans
 */

import { hierarchy, pack } from 'd3-hierarchy';
import type { HierarchyNode, Span } from '../components/ExecHierarchy/types';
import {
  type TreeSession,
  type TreeTurn,
  type TreeTool,
  type SharedToolNode,
  type FileNode,
  type DirectoryNode,
  TOOL_COLORS,
  getToolType,
  getToolColor,
  getToolDisplayName,
  extractFilePath,
  getFileExtension,
  getDirectoryPath,
} from './evolutionTreeUtils';

// Re-export types needed by component
export type { TreeSession, TreeTurn, TreeTool, SharedToolNode, FileNode, DirectoryNode };

// ============================================================================
// Types
// ============================================================================

export interface PackedNode {
  type: 'root' | 'directory' | 'file';
  data: DirectoryNode | FileNode | null;
  x: number;
  y: number;
  r: number;
  children?: PackedNode[];
}

export interface SpiralPosition {
  x: number;
  y: number;
  angle: number;
  radius: number;
  turn: TreeTurn;
  isAnomaly: boolean;
  spiralDirection: number;
  activity: number;
}

// Tool type priority for ring assignment (inner = more important/frequent types)
export const TOOL_TYPE_RINGS: Record<string, number> = {
  Read: 0,
  Edit: 0,
  Write: 0,
  Bash: 1,
  Grep: 1,
  Glob: 1,
  Task: 2,
  WebFetch: 2,
  WebSearch: 2,
  default: 1,
};

// ============================================================================
// File Hierarchy Building
// ============================================================================

/**
 * Build file hierarchy from turns
 */
export function buildFileHierarchy(turns: TreeTurn[]): DirectoryNode[] {
  const fileMap = new Map<string, FileNode>();

  turns.forEach((turn, turnIdx) => {
    turn.tools.forEach(tool => {
      const toolType = getToolType(tool.name);
      if (!['Read', 'Edit', 'Write'].includes(toolType)) return;

      const filePath = extractFilePath(tool.name);
      if (!filePath) return;

      const opType = toolType as 'Read' | 'Edit' | 'Write';

      if (!fileMap.has(filePath)) {
        fileMap.set(filePath, {
          filePath,
          fileName: filePath.split(/[/\\]/).pop() || filePath,
          fileType: getFileExtension(filePath),
          directory: getDirectoryPath(filePath),
          operations: [],
          totalOps: 0,
          readCount: 0,
          editCount: 0,
          writeCount: 0,
          errorCount: 0,
          turnIds: new Set(),
          firstTurn: turnIdx,
          lastTurn: turnIdx,
          x: 0,
          y: 0,
          radius: 0,
        });
      }

      const file = fileMap.get(filePath)!;
      file.operations.push({
        turnId: turn.id,
        turnNumber: turn.turnNumber,
        toolType: opType,
        toolName: tool.name,
        durationMs: tool.durationMs,
        status: tool.status,
      });

      file.totalOps++;
      file.turnIds.add(turn.id);
      file.lastTurn = Math.max(file.lastTurn, turnIdx);

      if (opType === 'Read') file.readCount++;
      else if (opType === 'Edit') file.editCount++;
      else if (opType === 'Write') file.writeCount++;
      if (tool.status === 'error') file.errorCount++;
    });
  });

  const dirMap = new Map<string, DirectoryNode>();
  fileMap.forEach(file => {
    const dirPath = file.directory;
    if (!dirMap.has(dirPath)) {
      dirMap.set(dirPath, {
        path: dirPath,
        name: dirPath.split(/[/\\]/).pop() || dirPath,
        files: [],
        totalOps: 0,
        errorCount: 0,
        x: 0,
        y: 0,
        radius: 0,
      });
    }
    const dir = dirMap.get(dirPath)!;
    dir.files.push(file);
    dir.totalOps += file.totalOps;
    dir.errorCount += file.errorCount;
  });

  let directories = Array.from(dirMap.values()).sort((a, b) => b.totalOps - a.totalOps);
  directories.forEach(dir => {
    dir.files.sort((a, b) => b.totalOps - a.totalOps);
  });

  const looseFiles: FileNode[] = [];
  directories = directories.filter(dir => {
    if (dir.files.length === 1) {
      looseFiles.push(dir.files[0]);
      return false;
    }
    return true;
  });

  if (looseFiles.length > 0) {
    directories.push({
      path: '__loose_files__',
      name: '__loose_files__',
      files: looseFiles,
      totalOps: looseFiles.reduce((sum, f) => sum + f.totalOps, 0),
      errorCount: looseFiles.reduce((sum, f) => sum + f.errorCount, 0),
      x: 0,
      y: 0,
      radius: 0,
    });
  }

  return directories;
}

// ============================================================================
// Circle Packing Layout
// ============================================================================

interface HierarchyData {
  name: string;
  value?: number;
  type: 'root' | 'directory' | 'file';
  data: DirectoryNode | FileNode | null;
  children?: HierarchyData[];
}

/**
 * Calculate circle packing layout for file hub
 */
export function calculateFileHubLayout(
  directories: DirectoryNode[],
  expandedDirs: Set<string>,
  centerX: number,
  centerY: number,
  hubRadius: number
): PackedNode[] {
  if (directories.length === 0) return [];

  const children: HierarchyData[] = [];

  directories.forEach(dir => {
    if (dir.path === '__loose_files__') {
      dir.files.forEach(file => {
        children.push({
          name: file.fileName,
          type: 'file' as const,
          data: file,
          value: Math.max(file.totalOps, 1),
        });
      });
    } else {
      const isExpanded = expandedDirs.has(dir.path);
      children.push({
        name: dir.name,
        type: 'directory' as const,
        data: dir,
        value: isExpanded ? undefined : Math.max(dir.totalOps, 1),
        children: isExpanded
          ? dir.files.map(file => ({
              name: file.fileName,
              type: 'file' as const,
              data: file,
              value: Math.max(file.totalOps, 1),
            }))
          : undefined,
      });
    }
  });

  const rootData: HierarchyData = {
    name: 'root',
    type: 'root',
    data: null,
    children,
  };

  const root = hierarchy(rootData)
    .sum(d => d.value || 0)
    .sort((a, b) => (b.value || 0) - (a.value || 0));

  const packLayout = pack<HierarchyData>()
    .size([hubRadius * 2, hubRadius * 2])
    .padding(4);

  const packed = packLayout(root);
  const result: PackedNode[] = [];

  packed.descendants().forEach(node => {
    if (node.data.type === 'root') return;

    const packedNode: PackedNode = {
      type: node.data.type,
      data: node.data.data,
      x: centerX - hubRadius + node.x,
      y: centerY - hubRadius + node.y,
      r: node.r,
    };

    if (node.data.type === 'directory' && node.data.data) {
      const dir = node.data.data as DirectoryNode;
      dir.x = packedNode.x;
      dir.y = packedNode.y;
      dir.radius = packedNode.r;
    } else if (node.data.type === 'file' && node.data.data) {
      const file = node.data.data as FileNode;
      file.x = packedNode.x;
      file.y = packedNode.y;
      file.radius = packedNode.r;
    }

    result.push(packedNode);
  });

  return result;
}

// ============================================================================
// Anomaly Detection
// ============================================================================

/**
 * Detect anomalies in turn metrics
 */
export function detectAnomalies(turns: TreeTurn[]): Set<number> {
  if (turns.length < 3) return new Set();

  const costs = turns.map(t => t.cost || 0);
  const tokens = turns.map(t => (t.tokensIn || 0) + (t.tokensOut || 0));
  const durations = turns.map(t => t.durationMs || 0);

  const mean = (arr: number[]) => arr.reduce((a, b) => a + b, 0) / arr.length;
  const stdDev = (arr: number[], m: number) =>
    Math.sqrt(arr.reduce((sum, x) => sum + (x - m) ** 2, 0) / arr.length);

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

    if (costZ > 2 || tokenZ > 2 || durationZ > 2) {
      anomalies.add(i);
    }
  });

  return anomalies;
}

// ============================================================================
// Spiral Positions
// ============================================================================

/**
 * Calculate spiral positions for turns
 */
export function calculateSpiralPositions(
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

  const maxTokens = Math.max(...turns.map(t => (t.tokensIn || 0) + (t.tokensOut || 0)), 1);
  const maxCost = Math.max(...turns.map(t => t.cost || 0), 0.001);

  const totalAngleBudget = (2 * Math.PI * n) / turnsPerRotation;
  const totalDuration = Math.max(
    turns.reduce((sum, t) => sum + (t.durationMs || 0), 0),
    1
  );

  const minProportion = 0.3 / n;
  const remainingProportion = 1 - minProportion * n;

  const angleSteps = turns.map(turn => {
    const duration = turn.durationMs || 0;
    const durationProportion = remainingProportion * (duration / totalDuration);
    const totalProportion = minProportion + durationProportion;
    return totalAngleBudget * totalProportion;
  });

  const positions: SpiralPosition[] = [];
  let spiralDirection = 1;
  let currentAngle = 0;
  let currentRadius = minRadius;
  const radiusRange = maxRadius - minRadius;

  turns.forEach((turn, i) => {
    const progress = i / Math.max(n - 1, 1);
    const isAnomaly = anomalies.has(i);

    const tokenActivity = ((turn.tokensIn || 0) + (turn.tokensOut || 0)) / maxTokens;
    const costActivity = (turn.cost || 0) / maxCost;
    const activity = tokenActivity * 0.5 + costActivity * 0.5;

    if (isAnomaly && i > 0) {
      spiralDirection *= -1;
      currentRadius = Math.min(currentRadius + radiusRange * 0.15, maxRadius);
    } else {
      currentRadius = minRadius + radiusRange * progress;
    }

    currentAngle += angleSteps[i] * spiralDirection;

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

// ============================================================================
// Shared Tool Nodes
// ============================================================================

/**
 * Build shared tool nodes from turns and spiral positions
 */
export function buildSharedToolNodes(
  turns: TreeTurn[],
  spiralPositions: SpiralPosition[],
  centerX: number,
  centerY: number,
  baseToolRingRadius: number,
  showAllTools: boolean = false
): SharedToolNode[] {
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
          fullName: tool.fullName,
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

  // Filter to show only tools used 2+ times, unless showAllTools is true
  const significantTools = Array.from(toolMap.values()).filter(
    t => showAllTools || t.usages.length >= 2
  );

  // Group by tool type and assign ring
  const toolsByType = new Map<string, SharedToolNode[]>();
  significantTools.forEach(tool => {
    const ring = TOOL_TYPE_RINGS[tool.toolType] ?? TOOL_TYPE_RINGS.default;
    const key = `${ring}`;
    if (!toolsByType.has(key)) {
      toolsByType.set(key, []);
    }
    toolsByType.get(key)!.push(tool);
  });

  // Position tools in rings around center
  const ringGap = 60;
  toolsByType.forEach((tools, ringKey) => {
    const ringNum = parseInt(ringKey, 10);
    const ringRadius = baseToolRingRadius + ringNum * ringGap;

    tools.forEach((tool, i) => {
      const angle = (2 * Math.PI * i) / tools.length - Math.PI / 2;
      tool.x = centerX + ringRadius * Math.cos(angle);
      tool.y = centerY + ringRadius * Math.sin(angle);
    });
  });

  return significantTools;
}

// ============================================================================
// Geometry Helpers
// ============================================================================

/**
 * Convert polar to cartesian coordinates
 */
export function polarToCartesian(
  cx: number,
  cy: number,
  radius: number,
  angleInDegrees: number
): { x: number; y: number } {
  const angleInRadians = ((angleInDegrees - 90) * Math.PI) / 180;
  return {
    x: cx + radius * Math.cos(angleInRadians),
    y: cy + radius * Math.sin(angleInRadians),
  };
}

/**
 * Describe an arc path for SVG
 */
export function describeArc(
  x: number,
  y: number,
  radius: number,
  startAngle: number,
  endAngle: number
): string {
  const start = polarToCartesian(x, y, radius, endAngle);
  const end = polarToCartesian(x, y, radius, startAngle);
  const largeArcFlag = endAngle - startAngle <= 180 ? '0' : '1';
  return `M ${start.x} ${start.y} A ${radius} ${radius} 0 ${largeArcFlag} 0 ${end.x} ${end.y}`;
}

/**
 * Generate branch path from stem to tool
 */
export function generateBranchPath(
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

// ============================================================================
// Span Filtering and Type Detection
// ============================================================================

/**
 * Filter spans based on hidden span types
 */
export function filterSpans(spans: Span[], hiddenSpanTypes?: Set<string>): Span[] {
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

/**
 * Check if a span is a turn
 */
export function isTurnSpan(name: string): boolean {
  return (
    name === 'api_request' ||
    name.startsWith('exec.turn') ||
    name.includes('.turn') ||
    name.includes('turn.')
  );
}

/**
 * Check if a span is a tool call
 */
export function isToolSpan(name: string): boolean {
  return (
    name.startsWith('claude_code.tool.') ||
    name === 'exec.tool_use' ||
    name.includes('.tool') ||
    name.includes('tool.')
  );
}

/**
 * Check if a span is a root/session span
 */
export function isSessionSpan(name: string): boolean {
  return (
    name === 'claude_code.session' ||
    name === 'claude.execute' ||
    name === 'gemini.execute' ||
    name === 'coordinator.task.execute' ||
    name.startsWith('ailang.') ||
    name.startsWith('eval.')
  );
}

/**
 * Extract full tool name from span attributes
 */
export function extractFullToolName(span: {
  name: string;
  attributes?: Record<string, unknown>;
}): string | null {
  if (!span.attributes) return null;

  const toolType = span.name
    .replace('claude_code.tool.', '')
    .replace('exec.tool_use.', '')
    .split('.')[0];

  const toolInput = span.attributes['tool.input'] || span.attributes['tool_input'];
  if (!toolInput) return null;

  let inputData: Record<string, unknown>;
  try {
    if (typeof toolInput === 'string') {
      inputData = JSON.parse(toolInput);
    } else if (typeof toolInput === 'object') {
      inputData = toolInput as Record<string, unknown>;
    } else {
      return null;
    }
  } catch {
    return null;
  }

  switch (toolType) {
    case 'Bash':
      if (inputData.command && typeof inputData.command === 'string') {
        return `Bash: ${inputData.command}`;
      }
      break;
    case 'Read':
    case 'Write':
    case 'Edit':
      if (inputData.file_path && typeof inputData.file_path === 'string') {
        return `${toolType}: ${inputData.file_path}`;
      }
      break;
    case 'Grep':
      if (inputData.pattern && typeof inputData.pattern === 'string') {
        const path =
          inputData.path && typeof inputData.path === 'string' ? ` in ${inputData.path}` : '';
        return `Grep: "${inputData.pattern}"${path}`;
      }
      break;
    case 'Glob':
      if (inputData.pattern && typeof inputData.pattern === 'string') {
        const path =
          inputData.path && typeof inputData.path === 'string' ? ` in ${inputData.path}` : '';
        return `Glob: ${inputData.pattern}${path}`;
      }
      break;
    case 'Task':
      if (inputData.description && typeof inputData.description === 'string') {
        return `Task: ${inputData.description}`;
      }
      if (inputData.prompt && typeof inputData.prompt === 'string') {
        return `Task: ${inputData.prompt}`;
      }
      break;
    case 'WebFetch':
      if (inputData.url && typeof inputData.url === 'string') {
        return `WebFetch: ${inputData.url}`;
      }
      break;
    case 'WebSearch':
      if (inputData.query && typeof inputData.query === 'string') {
        return `WebSearch: "${inputData.query}"`;
      }
      break;
  }

  return null;
}

// ============================================================================
// Tree Data Building
// ============================================================================

/**
 * Build tree from HierarchyNode array
 */
export function buildTreeFromNodes(nodes: HierarchyNode[]): {
  session: TreeSession | null;
  turns: TreeTurn[];
} {
  if (!nodes || nodes.length === 0) {
    return { session: null, turns: [] };
  }

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

  function collectFromNode(node: HierarchyNode) {
    if (node.type === 'turn') {
      turnCounter++;
      const turnNum = node.turnNumber || turnCounter;

      const tools: TreeTool[] = [];
      if (node.children) {
        node.children.forEach(child => {
          if (child.type === 'tool_use') {
            const spanData = child._span || { name: child.label, attributes: child.attributes };
            const fullName = extractFullToolName(
              spanData as { name: string; attributes?: Record<string, unknown> }
            );
            tools.push({
              id: child.id,
              name: child.label,
              fullName: fullName || undefined,
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

    if (node.children) {
      node.children.forEach(child => collectFromNode(child));
    }
  }

  nodes.forEach(node => collectFromNode(node));

  if (turnsList.length === 0) {
    nodes.forEach((node, idx) => {
      if (idx > 0 || nodes.length === 1) {
        const tools: TreeTool[] = [];
        if (node.children) {
          node.children.forEach(child => {
            const spanData = child._span || { name: child.label, attributes: child.attributes };
            const fullName = extractFullToolName(
              spanData as { name: string; attributes?: Record<string, unknown> }
            );
            tools.push({
              id: child.id,
              name: child.label,
              fullName: fullName || undefined,
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

/**
 * Build tree from Span array
 */
export function buildTreeFromSpans(
  spans?: Span[],
  _hiddenSpanTypes?: Set<string>
): { session: TreeSession | null; turns: TreeTurn[] } {
  if (!spans || spans.length === 0) {
    return { session: null, turns: [] };
  }

  const allSpans = spans;
  const sessionSpan = allSpans.find(s => isSessionSpan(s.name)) || allSpans[0];

  const session: TreeSession = {
    id: sessionSpan.id,
    name: sessionSpan.display_name || sessionSpan.name,
    durationMs: sessionSpan.durationMs || 0,
    cost: (sessionSpan as any).cost_usd || 0,
    tokensIn: (sessionSpan as any).tokens_in || 0,
    tokensOut: (sessionSpan as any).tokens_out || 0,
  };

  const turnsList: TreeTurn[] = [];
  let turnCounter = 0;

  function collectTurns(span: Span, _depth: number = 0) {
    const isTurn = isTurnSpan(span.name);

    if (isTurn) {
      turnCounter++;
      const attrTurnNum =
        span.attributes?.['turn.number'] ||
        span.attributes?.['turn_number'] ||
        span.attributes?.['exec.turn'];
      const match = span.name.match(/turn[._]?(\d+)/i);
      const turnNum = attrTurnNum
        ? parseInt(String(attrTurnNum), 10)
        : match
          ? parseInt(match[1], 10)
          : turnCounter;

      const tools: TreeTool[] = [];

      if (span.children) {
        span.children.forEach(child => {
          if (isToolSpan(child.name)) {
            const displayName =
              child.display_name ||
              child.name.replace('claude_code.tool.', '').replace('exec.tool_use.', '');
            const fullName = extractFullToolName(
              child as { name: string; attributes?: Record<string, unknown> }
            );
            tools.push({
              id: child.id,
              name: displayName,
              fullName: fullName || undefined,
              durationMs: child.durationMs || 0,
              status: child.status === 'error' ? 'error' : 'ok',
              cost: (child as any).cost_usd,
            });
          }
        });
      }

      const cost =
        (span as any).cost_usd || parseFloat(span.attributes?.['cost_usd'] || '0') || 0;
      const tokensIn =
        (span as any).tokens_in ||
        parseInt(span.attributes?.['tokens_in'] || span.attributes?.['input_tokens'] || '0', 10) ||
        0;
      const tokensOut =
        (span as any).tokens_out ||
        parseInt(
          span.attributes?.['tokens_out'] || span.attributes?.['output_tokens'] || '0',
          10
        ) ||
        0;

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

    if (span.children) {
      span.children.forEach(child => collectTurns(child, _depth + 1));
    }
  }

  allSpans.forEach(span => collectTurns(span, 0));

  if (turnsList.length === 0 && allSpans.length > 1) {
    allSpans.forEach((span, idx) => {
      if (span.id !== sessionSpan.id) {
        const tools: TreeTool[] = [];
        if (span.children) {
          span.children.forEach(child => {
            const displayName = child.display_name || child.name;
            const fullName = extractFullToolName(
              child as { name: string; attributes?: Record<string, unknown> }
            );
            tools.push({
              id: child.id,
              name: displayName,
              fullName: fullName || undefined,
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

  const turns = turnsList.sort((a, b) => a.turnNumber - b.turnNumber);
  return { session, turns };
}

/**
 * Main entry point - build tree data from nodes or spans
 */
export function buildTreeData(
  spans?: Span[],
  nodes?: HierarchyNode[],
  _hiddenSpanTypes?: Set<string>
): { session: TreeSession | null; turns: TreeTurn[] } {
  if (nodes && nodes.length > 0) {
    const fromNodes = buildTreeFromNodes(nodes);
    const hasTools = fromNodes.turns.some(t => t.tools.length > 0);
    if (fromNodes.turns.length > 0 && hasTools) {
      return fromNodes;
    }
  }

  if (spans && spans.length > 0) {
    const fromSpans = buildTreeFromSpans(spans, undefined);
    if (fromSpans.turns.length > 0) {
      return fromSpans;
    }
  }

  if (nodes && nodes.length > 0) {
    return buildTreeFromNodes(nodes);
  }

  return { session: null, turns: [] };
}
