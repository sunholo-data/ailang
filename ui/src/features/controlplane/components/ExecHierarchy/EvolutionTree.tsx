/**
 * EvolutionTree - Bioluminescent Data Organism Visualization
 *
 * An organic, tree-like visualization showing AI execution flow over time.
 * The stem curves based on token activity, turns appear as luminous nodes,
 * and tools branch off as smaller tendrils with leaf-like terminals.
 */
import React, { useMemo, useCallback, useRef, useEffect, useState } from 'react';
import { hierarchy, pack } from 'd3-hierarchy';
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
  fullName?: string;  // Full untruncated name for detailed view
  durationMs: number;
  status: 'ok' | 'error';
  cost?: number;
}

// Shared tool node - represents a unique tool name used across multiple turns
interface SharedToolNode {
  name: string;
  fullName?: string;  // Full untruncated name for detailed view
  displayName: string;
  toolType: string;
  color: string;
  usages: { turnIndex: number; turnId: string; tool: TreeTool }[];
  x: number;
  y: number;
  hasError: boolean;
}

// ============================================
// File-centric data structures
// ============================================

interface FileOperation {
  turnId: string;
  turnNumber: number;
  toolType: 'Read' | 'Edit' | 'Write';
  toolName: string;
  durationMs: number;
  status: 'ok' | 'error';
}

interface FileNode {
  filePath: string;       // Full path
  fileName: string;       // Just the filename
  fileType: string;       // Extension (.go, .tsx, etc.)
  directory: string;      // Parent directory path

  operations: FileOperation[];
  totalOps: number;
  readCount: number;
  editCount: number;
  writeCount: number;
  errorCount: number;

  // Which turns touched this file
  turnIds: Set<string>;
  firstTurn: number;
  lastTurn: number;

  // Positioned by circle packing
  x: number;
  y: number;
  radius: number;
}

interface DirectoryNode {
  path: string;           // "/internal/parser"
  name: string;           // "parser"
  files: FileNode[];
  totalOps: number;       // Aggregate of all files
  errorCount: number;

  // Positioned by circle packing
  x: number;
  y: number;
  radius: number;
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

// Extract file path from tool name (e.g., "Read: /path/to/file.go" → "/path/to/file.go")
function extractFilePath(toolName: string): string | null {
  // Pattern 1: "ToolType: /path/to/file"
  const colonMatch = toolName.match(/^[A-Za-z]+:\s*(.+)$/);
  if (colonMatch) {
    const path = colonMatch[1].trim();
    // Verify it looks like a path (has extension or starts with / or contains /)
    if (path.includes('/') || path.includes('\\') || /\.\w+$/.test(path)) {
      return path;
    }
  }

  // Pattern 2: Direct path in name
  const pathMatch = toolName.match(/([/\\][\w./\\-]+\.\w+)/);
  if (pathMatch) {
    return pathMatch[1];
  }

  return null;
}

// Get file extension
function getFileExtension(filePath: string): string {
  const match = filePath.match(/\.(\w+)$/);
  return match ? match[1] : '';
}

// Get directory path from file path
function getDirectoryPath(filePath: string): string {
  const lastSlash = Math.max(filePath.lastIndexOf('/'), filePath.lastIndexOf('\\'));
  if (lastSlash === -1) return '/';
  return filePath.slice(0, lastSlash) || '/';
}

// Build file hierarchy from turns
function buildFileHierarchy(turns: TreeTurn[]): DirectoryNode[] {
  const fileMap = new Map<string, FileNode>();

  // Extract files from all turns - only file operations (Read, Edit, Write), NOT Bash
  turns.forEach((turn, turnIdx) => {
    turn.tools.forEach(tool => {
      const toolType = getToolType(tool.name);

      // Only include actual file operations, skip Bash commands
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

  // Group files by directory
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

  // Sort directories by total operations (most active first)
  let directories = Array.from(dirMap.values())
    .sort((a, b) => b.totalOps - a.totalOps);

  // Sort files within each directory
  directories.forEach(dir => {
    dir.files.sort((a, b) => b.totalOps - a.totalOps);
  });

  // Promote single-file directories to show files directly at root level
  // These files will appear without a directory wrapper (no extra click needed)
  const looseFiles: FileNode[] = [];
  directories = directories.filter(dir => {
    if (dir.files.length === 1) {
      looseFiles.push(dir.files[0]);
      return false; // Remove single-file directories
    }
    return true;
  });

  // Store loose files in a special marker directory that layout will handle differently
  // This marker directory will be "auto-expanded" to show files directly
  if (looseFiles.length > 0) {
    directories.push({
      path: '__loose_files__',
      name: '__loose_files__', // Special marker - layout will show files directly
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

// Circle packing layout for file hub
interface PackedNode {
  type: 'root' | 'directory' | 'file';
  data: DirectoryNode | FileNode | null;
  x: number;
  y: number;
  r: number;
  children?: PackedNode[];
}

function calculateFileHubLayout(
  directories: DirectoryNode[],
  expandedDirs: Set<string>,
  centerX: number,
  centerY: number,
  hubRadius: number
): PackedNode[] {
  if (directories.length === 0) return [];

  // Build hierarchy for d3
  interface HierarchyData {
    name: string;
    value?: number;
    type: 'root' | 'directory' | 'file';
    data: DirectoryNode | FileNode | null;
    children?: HierarchyData[];
  }

  // Build children: directories get wrapped, loose files appear directly at root
  const children: HierarchyData[] = [];

  directories.forEach(dir => {
    if (dir.path === '__loose_files__') {
      // Loose files appear directly at root level (no directory wrapper)
      dir.files.forEach(file => {
        children.push({
          name: file.fileName,
          type: 'file' as const,
          data: file,
          value: Math.max(file.totalOps, 1),
        });
      });
    } else {
      // Normal directory
      const isExpanded = expandedDirs.has(dir.path);
      children.push({
        name: dir.name,
        type: 'directory' as const,
        data: dir,
        // If expanded, show files as children; otherwise just the directory
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

  // Create d3 hierarchy and apply circle packing
  const root = hierarchy(rootData)
    .sum(d => d.value || 0)
    .sort((a, b) => (b.value || 0) - (a.value || 0));

  const packLayout = pack<HierarchyData>()
    .size([hubRadius * 2, hubRadius * 2])
    .padding(4);

  const packed = packLayout(root);

  // Convert to our PackedNode format, offset to center
  const result: PackedNode[] = [];

  packed.descendants().forEach(node => {
    if (node.data.type === 'root') return; // Skip root node

    const packedNode: PackedNode = {
      type: node.data.type,
      data: node.data.data,
      x: centerX - hubRadius + node.x,
      y: centerY - hubRadius + node.y,
      r: node.r,
    };

    // Update the original data objects with positions (for edges later)
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

export interface EvolutionTreeProps {
  spans?: Span[];
  nodes?: HierarchyNode[];
  selectedNodeId?: string | null;
  onNodeClick?: (node: HierarchyNode, event?: React.MouseEvent) => void;
  hiddenSpanTypes?: Set<string>;
  isExpanded?: boolean;
  // Theme from parent (syncs with app-level theme toggle)
  theme?: 'dark' | 'light';
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

  // Convert to array
  let sharedTools = Array.from(toolMap.values());

  // Exclude file-related tools (Read, Edit, Write) - these are shown in the file hub
  const FILE_TOOL_TYPES = new Set(['Read', 'Edit', 'Write']);
  sharedTools = sharedTools.filter(t => !FILE_TOOL_TYPES.has(t.toolType));

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

// Extract full tool name from span attributes (not truncated)
// Returns null if no full name can be extracted
function extractFullToolName(span: { name: string; attributes?: Record<string, unknown> }): string | null {
  if (!span.attributes) return null;

  // Get the tool type from span name
  const toolType = span.name.replace('claude_code.tool.', '').replace('exec.tool_use.', '').split('.')[0];

  // Try to get tool.input which contains the full command/parameters
  const toolInput = span.attributes['tool.input'] || span.attributes['tool_input'];
  if (!toolInput) return null;

  // Parse the tool input JSON
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

  // Build full name based on tool type
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
        const path = inputData.path && typeof inputData.path === 'string' ? ` in ${inputData.path}` : '';
        return `Grep: "${inputData.pattern}"${path}`;
      }
      break;
    case 'Glob':
      if (inputData.pattern && typeof inputData.pattern === 'string') {
        const path = inputData.path && typeof inputData.path === 'string' ? ` in ${inputData.path}` : '';
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
            // Try to extract full name from underlying span or attributes
            const spanData = child._span || { name: child.label, attributes: child.attributes };
            const fullName = extractFullToolName(spanData as { name: string; attributes?: Record<string, unknown> });
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
            const spanData = child._span || { name: child.label, attributes: child.attributes };
            const fullName = extractFullToolName(spanData as { name: string; attributes?: Record<string, unknown> });
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
            const displayName = child.display_name || child.name.replace('claude_code.tool.', '').replace('exec.tool_use.', '');
            const fullName = extractFullToolName(child as { name: string; attributes?: Record<string, unknown> });
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
            const displayName = child.display_name || child.name;
            const fullName = extractFullToolName(child as { name: string; attributes?: Record<string, unknown> });
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
  theme,
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

  // File hub state
  const [expandedDirs, setExpandedDirs] = useState<Set<string>>(new Set());
  const [selectedFile, setSelectedFile] = useState<FileNode | null>(null);
  const [hoveredFile, setHoveredFile] = useState<{
    file: FileNode;
    pos: { x: number; y: number };
  } | null>(null);
  const [hoveredDir, setHoveredDir] = useState<{
    dir: DirectoryNode;
    pos: { x: number; y: number };
  } | null>(null);

  // Time evolution playback state
  const [playback, setPlayback] = useState<{
    isPlaying: boolean;
    currentTimeMs: number;
    speed: number;
  }>({
    isPlaying: false,
    currentTimeMs: Infinity, // Start at end - show all by default
    speed: 1,
  });

  // Connection highlight decay rate (how many turns until fully faded)
  // 0 = instant fade, higher = longer glow persistence
  const [highlightDecay, setHighlightDecay] = useState(15);

  // Color scheme: use passed theme prop, fallback to system detection
  const [systemScheme, setSystemScheme] = useState<'light' | 'dark'>('dark');

  useEffect(() => {
    const mediaQuery = window.matchMedia('(prefers-color-scheme: light)');
    setSystemScheme(mediaQuery.matches ? 'light' : 'dark');

    const handler = (e: MediaQueryListEvent) => {
      setSystemScheme(e.matches ? 'light' : 'dark');
    };
    mediaQuery.addEventListener('change', handler);
    return () => mediaQuery.removeEventListener('change', handler);
  }, []);

  // Use passed theme prop if provided, otherwise fall back to system preference
  const colorScheme = theme ?? systemScheme;

  // Theme colors based on color scheme
  const themeColors = useMemo(() => ({
    // Background & foreground
    bg: colorScheme === 'light' ? '#f8fafc' : '#0f1419',
    bgSecondary: colorScheme === 'light' ? '#ffffff' : '#1a2332',
    text: colorScheme === 'light' ? '#1f2937' : '#e6edf3',
    textMuted: colorScheme === 'light' ? '#6b7280' : '#8b949e',
    textSubtle: colorScheme === 'light' ? '#9ca3af' : '#6b7280',

    // Accent colors (same for both, but can adjust intensity)
    emerald: colorScheme === 'light' ? '#059669' : '#10b981',
    emeraldLight: colorScheme === 'light' ? 'rgba(16, 185, 129, 0.1)' : 'rgba(16, 185, 129, 0.2)',
    cyan: '#06b6d4',
    error: colorScheme === 'light' ? '#dc2626' : '#ef4444',

    // SVG specific
    nodeText: colorScheme === 'light' ? '#1f2937' : '#e6edf3',
    nodeTextOnFill: colorScheme === 'light' ? '#ffffff' : '#0f1419',
    stemStroke: colorScheme === 'light' ? 'rgba(16, 185, 129, 0.4)' : 'rgba(16, 185, 129, 0.3)',
    stemGlow: colorScheme === 'light' ? 'rgba(16, 185, 129, 0.15)' : 'rgba(16, 185, 129, 0.15)',

    // File nodes
    fileCyan: colorScheme === 'light' ? '#0891b2' : '#06b6d4',
    fileRead: colorScheme === 'light' ? '#2563eb' : '#60a5fa',
    fileEdit: colorScheme === 'light' ? '#db2777' : '#f472b6',
    fileWrite: colorScheme === 'light' ? '#7c3aed' : '#a78bfa',
  }), [colorScheme]);

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

  // Calculate total duration for playback
  const totalDurationMs = useMemo(() => {
    const MIN_TURN_DURATION = 100; // Minimum visible time for very short turns
    return turns.reduce((sum, t) => sum + Math.max(t.durationMs || MIN_TURN_DURATION, MIN_TURN_DURATION), 0);
  }, [turns]);

  // Calculate visibility state based on playback time
  // Base visibility state (turns and tools) - computed without fileDirectories
  // Also tracks turn recency for decaying highlights
  const baseVisibilityState = useMemo(() => {
    // If not playing and at end, show everything
    if (!playback.isPlaying && playback.currentTimeMs >= totalDurationMs) {
      return {
        visibleTurnIndices: new Set(turns.map((_, i) => i)),
        visibleToolIds: new Set(turns.flatMap(t => t.tools.map(tool => tool.id))),
        partialTurnIndex: null as number | null,
        partialTurnProgress: 1,
        showAll: true,
        // Recency tracking: for showAll, all turns have max recency (no highlighting)
        turnRecency: new Map<number, number>(),
        currentTurnIndex: turns.length - 1,
      };
    }

    const MIN_TURN_DURATION = 100;
    let elapsed = 0;
    const visibleTurnIndices = new Set<number>();
    const visibleToolIds = new Set<string>();
    let partialTurnIndex: number | null = null;
    let partialTurnProgress = 0;
    let currentTurnIndex = -1;

    for (let i = 0; i < turns.length; i++) {
      const turn = turns[i];
      const turnDuration = Math.max(turn.durationMs || MIN_TURN_DURATION, MIN_TURN_DURATION);
      const turnStart = elapsed;
      const turnEnd = elapsed + turnDuration;

      if (playback.currentTimeMs >= turnEnd) {
        // Turn fully visible
        visibleTurnIndices.add(i);
        turn.tools.forEach(t => visibleToolIds.add(t.id));
        currentTurnIndex = i;
      } else if (playback.currentTimeMs >= turnStart) {
        // Turn partially visible (currently appearing)
        visibleTurnIndices.add(i);

        // Calculate tool cascade progress within turn
        const progress = (playback.currentTimeMs - turnStart) / turnDuration;
        const toolCount = turn.tools.length;
        const visibleToolCount = Math.floor(progress * (toolCount + 1)); // +1 so last tool appears before turn ends
        turn.tools.slice(0, visibleToolCount).forEach(t => visibleToolIds.add(t.id));

        partialTurnIndex = i;
        partialTurnProgress = progress;
        currentTurnIndex = i;
        break; // Future turns not visible
      }

      elapsed = turnEnd;
    }

    // Calculate recency for each visible turn
    // Current/appearing turn has recency 0, previous has 1, etc.
    const turnRecency = new Map<number, number>();
    visibleTurnIndices.forEach(turnIdx => {
      const recency = currentTurnIndex - turnIdx;
      turnRecency.set(turnIdx, recency);
    });

    return {
      visibleTurnIndices,
      visibleToolIds,
      partialTurnIndex,
      partialTurnProgress,
      showAll: false,
      turnRecency,
      currentTurnIndex,
    };
  }, [turns, playback.currentTimeMs, playback.isPlaying, totalDurationMs]);

  // Playback animation loop
  useEffect(() => {
    if (!playback.isPlaying) return;

    let lastTime = performance.now();
    let animationId: number;

    const tick = (now: number) => {
      const delta = (now - lastTime) * playback.speed;
      lastTime = now;

      setPlayback(prev => {
        const newTime = prev.currentTimeMs + delta;
        // Auto-pause at end
        if (newTime >= totalDurationMs) {
          return { ...prev, isPlaying: false, currentTimeMs: totalDurationMs };
        }
        return { ...prev, currentTimeMs: newTime };
      });

      animationId = requestAnimationFrame(tick);
    };

    animationId = requestAnimationFrame(tick);
    return () => cancelAnimationFrame(animationId);
  }, [playback.isPlaying, playback.speed, totalDurationMs]);

  // Playback control functions
  const togglePlayback = useCallback(() => {
    setPlayback(prev => {
      // If at end (or initial Infinity state) and pressing play, restart from beginning
      if (!prev.isPlaying && (prev.currentTimeMs >= totalDurationMs || prev.currentTimeMs === Infinity)) {
        return { ...prev, isPlaying: true, currentTimeMs: 0 };
      }
      return { ...prev, isPlaying: !prev.isPlaying };
    });
  }, [totalDurationMs]);

  const seekTo = useCallback((timeMs: number) => {
    setPlayback(prev => ({
      ...prev,
      currentTimeMs: Math.max(0, Math.min(timeMs, totalDurationMs)),
    }));
  }, [totalDurationMs]);

  const setPlaybackSpeed = useCallback((speed: number) => {
    setPlayback(prev => ({ ...prev, speed }));
  }, []);

  // Format time for display (mm:ss)
  const formatPlaybackTime = useCallback((ms: number, fallback?: number) => {
    // Handle Infinity (initial state) - use fallback or show as total
    const actualMs = ms === Infinity ? (fallback ?? ms) : ms;
    if (actualMs === Infinity) return '--:--';
    const totalSeconds = Math.floor(actualMs / 1000);
    const minutes = Math.floor(totalSeconds / 60);
    const seconds = totalSeconds % 60;
    return `${minutes}:${seconds.toString().padStart(2, '0')}`;
  }, []);

  // Calculate dynamic speed options based on total duration
  // Target: ~90 seconds playback by default (adjustable)
  const TARGET_PLAYBACK_SECONDS = 90;
  const speedOptions = useMemo(() => {
    const targetSpeed = totalDurationMs / (TARGET_PLAYBACK_SECONDS * 1000);

    // If session is short enough, use regular speeds
    if (targetSpeed <= 4) {
      return [0.5, 1, 2, 4];
    }

    // For longer sessions, calculate smart speed options
    // Round target to nice numbers
    const roundToNice = (n: number) => {
      if (n <= 5) return Math.ceil(n);
      if (n <= 10) return Math.ceil(n / 2) * 2;
      if (n <= 50) return Math.ceil(n / 5) * 5;
      if (n <= 100) return Math.ceil(n / 10) * 10;
      if (n <= 500) return Math.ceil(n / 25) * 25;
      return Math.ceil(n / 50) * 50;
    };

    const defaultSpeed = roundToNice(targetSpeed);
    const halfSpeed = roundToNice(targetSpeed / 2);
    const doubleSpeed = roundToNice(targetSpeed * 2);

    // Return speeds: half, default (for 90s), double, quadruple
    return [
      Math.max(1, halfSpeed),    // Slower (180s playback)
      defaultSpeed,              // Default (90s playback)
      doubleSpeed,               // Faster (45s playback)
      roundToNice(targetSpeed * 4), // Very fast (22s playback)
    ].filter((v, i, a) => a.indexOf(v) === i); // Remove duplicates
  }, [totalDurationMs]);

  // Default speed (targets 90 seconds playback)
  const defaultSpeed = useMemo(() => {
    return speedOptions.length > 1 ? speedOptions[1] : speedOptions[0];
  }, [speedOptions]);

  // Set default speed on initial render (only once)
  const [hasSetInitialSpeed, setHasSetInitialSpeed] = useState(false);
  useEffect(() => {
    if (!hasSetInitialSpeed && totalDurationMs > 0) {
      setPlayback(prev => ({ ...prev, speed: defaultSpeed }));
      setHasSetInitialSpeed(true);
    }
  }, [hasSetInitialSpeed, defaultSpeed, totalDurationMs]);

  // Generate spiral stem path (full path)
  const stemPath = useMemo(() =>
    generateSpiralPath(spiralPositions, CENTER_X, CENTER_Y),
    [spiralPositions]
  );

  // Generate visible stem path for playback (only through visible turns)
  const visibleStemPath = useMemo(() => {
    if (baseVisibilityState.showAll) return stemPath;

    // Find the last visible turn index
    let lastVisibleIdx = -1;
    for (let i = 0; i < spiralPositions.length; i++) {
      if (baseVisibilityState.visibleTurnIndices.has(i)) {
        lastVisibleIdx = i;
      }
    }

    if (lastVisibleIdx < 0) return '';

    // Generate path only through visible positions
    const visiblePositions = spiralPositions.slice(0, lastVisibleIdx + 1);
    return generateSpiralPath(visiblePositions, CENTER_X, CENTER_Y);
  }, [spiralPositions, stemPath, baseVisibilityState]);

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

  // Build file hierarchy from turns
  const fileDirectories = useMemo(() =>
    buildFileHierarchy(turns),
    [turns]
  );

  // Complete visibility state (adds file paths to base visibility)
  const visibilityState = useMemo(() => {
    const { visibleTurnIndices, visibleToolIds, partialTurnIndex, partialTurnProgress, showAll, turnRecency, currentTurnIndex } = baseVisibilityState;

    // If showing all, include all files
    if (showAll) {
      return {
        visibleTurnIndices,
        visibleToolIds,
        visibleFilePaths: new Set(fileDirectories.flatMap(d => d.files.map(f => f.filePath))),
        visibleDirectories: new Set(fileDirectories.map(d => d.path)),
        activeFilePaths: new Set<string>(),
        activeDirectories: new Set<string>(),
        partialTurnIndex,
        partialTurnProgress,
        showAll,
        turnRecency,
        currentTurnIndex,
      };
    }

    // Calculate which files are visible (file appears when its firstTurn becomes visible)
    const visibleFilePaths = new Set<string>();
    fileDirectories.forEach(dir => {
      dir.files.forEach(file => {
        if (visibleTurnIndices.has(file.firstTurn)) {
          visibleFilePaths.add(file.filePath);
        }
      });
    });

    // Calculate which directories are visible (directory appears when any file becomes visible)
    const visibleDirectories = new Set<string>();
    fileDirectories.forEach(dir => {
      const hasVisibleFile = dir.files.some(f => visibleFilePaths.has(f.filePath));
      if (hasVisibleFile) {
        visibleDirectories.add(dir.path);
      }
    });

    // Calculate active files/directories (being touched in current turn)
    // These get special highlighting during playback
    const activeFilePaths = new Set<string>();
    const activeDirectories = new Set<string>();

    if (partialTurnIndex !== null && currentTurnIndex >= 0) {
      const currentTurn = turns[currentTurnIndex];
      if (currentTurn) {
        // Get visible tools from current turn
        const visibleToolCount = Math.floor(partialTurnProgress * (currentTurn.tools.length + 1));
        const currentTools = currentTurn.tools.slice(0, visibleToolCount);

        currentTools.forEach(tool => {
          const filePath = extractFilePath(tool.name);
          if (filePath) {
            activeFilePaths.add(filePath);
            // Also mark the directory as active
            const dirPath = getDirectoryPath(filePath);
            activeDirectories.add(dirPath);
          }
        });
      }
    }

    return {
      visibleTurnIndices,
      visibleToolIds,
      visibleFilePaths,
      visibleDirectories,
      activeFilePaths,
      activeDirectories,
      partialTurnIndex,
      partialTurnProgress,
      showAll,
      turnRecency,
      currentTurnIndex,
    };
  }, [baseVisibilityState, fileDirectories, turns]);

  // Calculate file hub layout using circle packing
  // Hub fits inside the tool ring (center of the visualization)
  const FILE_HUB_RADIUS = Math.max(TOOL_RING_RADIUS - 15, 30);
  const fileHubNodes = useMemo(() =>
    calculateFileHubLayout(fileDirectories, expandedDirs, CENTER_X, CENTER_Y, FILE_HUB_RADIUS),
    [fileDirectories, expandedDirs, FILE_HUB_RADIUS]
  );

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

  // Helper to center the view on a specific position
  const centerOnPosition = useCallback((nodeX: number, nodeY: number) => {
    if (!containerRef.current) return;

    // Calculate the pan offset needed to center this position
    const containerWidth = containerRef.current.clientWidth;
    const containerHeight = containerRef.current.clientHeight;

    // Center of container in SVG coords (accounting for current zoom)
    const targetCenterX = containerWidth / (2 * zoom);
    const targetCenterY = containerHeight / (2 * zoom);

    // Calculate offset to center the node
    const newPanX = targetCenterX - nodeX;
    const newPanY = targetCenterY - nodeY;

    setPanOffset({ x: newPanX, y: newPanY });
  }, [zoom]);

  // Handle node click - show bioluminescent popup for turns
  const handleNodeClick = useCallback((turn: TreeTurn, event: React.MouseEvent, isAnomaly: boolean, activity: number, nodeX?: number, nodeY?: number) => {
    event.stopPropagation();

    // Close other popups - only one active at a time
    setToolPopup(null);
    setSelectedFile(null);

    // Position popup near the click, but constrained to viewport
    const x = Math.min(event.clientX + 20, window.innerWidth - 400);
    const y = Math.min(event.clientY - 50, window.innerHeight - 450);

    setTurnPopup({
      turn,
      pos: { x: Math.max(20, x), y: Math.max(20, y) },
      isAnomaly,
      activity,
    });

    // Auto-center on the clicked node
    if (nodeX !== undefined && nodeY !== undefined) {
      centerOnPosition(nodeX, nodeY);
    }
  }, [centerOnPosition]);

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

    // Close other popups - only one active at a time
    setTurnPopup(null);
    setSelectedFile(null);

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

    // Auto-center on the tool node
    centerOnPosition(toolNode.x, toolNode.y);
  }, [centerOnPosition]);

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

        {/* Main stem path - grows progressively during playback */}
        <path
          d={visibleStemPath}
          fill="none"
          stroke="url(#stemGradient)"
          strokeWidth="3"
          strokeLinecap="round"
          opacity="0.7"
          style={{ transition: 'd 0.1s linear' }}
        />

        {/* Animated "eating" pulses - simplified for performance (every 3rd segment) */}
        {(() => {
          const maxTokensIn = Math.max(...turns.map(t => t.tokensIn || 0), 1);

          // Only animate every 3rd segment to reduce DOM elements
          return spiralPositions.slice(1).filter((_, i) => i % 3 === 0).map((pos, filteredIdx) => {
            const actualIdx = filteredIdx * 3 + 1; // +1 because we sliced from index 1
            const i = filteredIdx * 3;

            // Only show pulses for visible turns
            const isTurnVisible = visibilityState.showAll || visibilityState.visibleTurnIndices.has(actualIdx);
            if (!isTurnVisible) return null;

            const prevPos = i === 0
              ? { x: CENTER_X, y: CENTER_Y }
              : spiralPositions[i];
            const tokensIn = pos.turn.tokensIn || 0;
            const tokensNorm = tokensIn / maxTokensIn;
            const pulseSpeed = Math.max(2, 5 - tokensNorm * 3);
            const bulgeSize = 4 + tokensNorm * 4;

            const segmentPath = i === 0
              ? `M ${CENTER_X} ${CENTER_Y} Q ${CENTER_X + (pos.x - CENTER_X) * 0.5} ${CENTER_Y + (pos.y - CENTER_Y) * 0.5}, ${pos.x} ${pos.y}`
              : `M ${prevPos.x} ${prevPos.y} Q ${prevPos.x + (pos.x - prevPos.x) * 0.5} ${prevPos.y + (pos.y - prevPos.y) * 0.5}, ${pos.x} ${pos.y}`;

            const pulseColor = pos.turn.status === 'error' ? '#ef4444' : pos.isAnomaly ? '#f59e0b' : '#10b981';

            return (
              <circle
                key={`eating-pulse-${i}`}
                r={bulgeSize}
                fill={pulseColor}
                opacity={0.7}
              >
                <animateMotion
                  dur={`${pulseSpeed}s`}
                  repeatCount="indefinite"
                  path={segmentPath}
                />
              </circle>
            );
          });
        })()}

        {/* File Hub - Circle-packed files organized by directory */}
        <g className={styles.fileHub}>
          {/* Hub background - fades in as files become visible */}
          <circle
            cx={CENTER_X}
            cy={CENTER_Y}
            r={FILE_HUB_RADIUS}
            fill="rgba(6, 182, 212, 0.05)"
            stroke="rgba(6, 182, 212, 0.2)"
            strokeWidth={1}
            style={{
              opacity: visibilityState.showAll || visibilityState.visibleFilePaths.size > 0 ? 1 : 0.2,
              transition: 'opacity 0.5s ease-out',
            }}
          />

          {/* Directory and file nodes */}
          {fileHubNodes.map((node, i) => {
            if (node.type === 'directory' && node.data) {
              const dir = node.data as DirectoryNode;
              const isExpanded = expandedDirs.has(dir.path);
              const isHovered = hoveredId === `dir-${dir.path}`;
              const hasErrors = dir.errorCount > 0;

              // Playback visibility - directory visible when any file in it is visible
              const isDirVisible = visibilityState.showAll || visibilityState.visibleDirectories.has(dir.path);

              // Active highlight - directory is being accessed in current turn
              const isDirActive = visibilityState.activeDirectories.has(dir.path);

              return (
                <g
                  key={`dir-${dir.path}`}
                  style={{
                    opacity: isDirVisible ? 1 : 0,
                    transform: isDirVisible ? 'scale(1)' : 'scale(0.5)',
                    transformOrigin: `${node.x}px ${node.y}px`,
                    transition: 'opacity 0.3s ease-out, transform 0.3s ease-out',
                  }}
                >
                  {/* Active glow ring for directory being accessed */}
                  {isDirActive && (
                    <circle
                      cx={node.x}
                      cy={node.y}
                      r={node.r + 6}
                      fill="none"
                      stroke="#3b82f6"
                      strokeWidth={3}
                      opacity={0.6}
                      className={styles.appearingGlow}
                    />
                  )}
                  {/* Directory circle */}
                  <circle
                    cx={node.x}
                    cy={node.y}
                    r={isHovered ? node.r + 2 : node.r}
                    fill={isDirActive ? 'rgba(59, 130, 246, 0.4)' : isExpanded ? 'rgba(59, 130, 246, 0.1)' : 'rgba(59, 130, 246, 0.3)'}
                    stroke={hasErrors ? '#ef4444' : isDirActive ? '#60a5fa' : '#3b82f6'}
                    strokeWidth={isDirActive ? 2.5 : isHovered ? 2 : 1}
                    style={{ cursor: 'pointer' }}
                    onClick={() => {
                      setExpandedDirs(prev => {
                        const next = new Set(prev);
                        if (next.has(dir.path)) next.delete(dir.path);
                        else next.add(dir.path);
                        return next;
                      });
                    }}
                    onMouseEnter={(e) => {
                      setHoveredId(`dir-${dir.path}`);
                      setHoveredDir({
                        dir,
                        pos: { x: e.clientX, y: e.clientY },
                      });
                    }}
                    onMouseLeave={() => {
                      setHoveredId(null);
                      setHoveredDir(null);
                    }}
                  />
                  {/* Directory name */}
                  {node.r > 15 && (
                    <text
                      x={node.x}
                      y={node.y + 3}
                      textAnchor="middle"
                      fontSize={Math.min(10, node.r / 3)}
                      fill={themeColors.text}
                      pointerEvents="none"
                    >
                      {isExpanded ? '📂' : '📁'} {dir.name.slice(0, 8)}
                    </text>
                  )}
                </g>
              );
            } else if (node.type === 'file' && node.data) {
              const file = node.data as FileNode;
              const isHovered = hoveredId === `file-${file.filePath}`;
              const isSelected = selectedFile?.filePath === file.filePath;
              const hasErrors = file.errorCount > 0;

              // Playback visibility - file visible when its firstTurn is visible
              const isFileVisible = visibilityState.showAll || visibilityState.visibleFilePaths.has(file.filePath);

              // Active highlight - file is being accessed in current turn
              const isFileActive = visibilityState.activeFilePaths.has(file.filePath);

              // File color based on type - emerald/cyan bioluminescent theme
              const fileColors: Record<string, string> = {
                go: themeColors.cyan,        // cyan for Go
                tsx: '#14b8a6',              // teal for React
                ts: '#0d9488',               // deeper teal for TypeScript
                js: '#34d399',               // light emerald for JS
                ail: themeColors.emerald,    // emerald for AILANG
                md: '#6ee7b7',               // pale emerald for markdown
                json: '#2dd4bf',             // bright teal for JSON
                yaml: '#5eead4',             // aqua for YAML
                css: '#22d3ee',              // sky cyan for CSS
              };
              const fileColor = fileColors[file.fileType] || themeColors.emerald;

              return (
                <g
                  key={`file-${file.filePath}`}
                  style={{
                    opacity: isFileVisible ? 1 : 0,
                    transform: isFileVisible ? 'scale(1)' : 'scale(0.5)',
                    transformOrigin: `${node.x}px ${node.y}px`,
                    transition: 'opacity 0.3s ease-out, transform 0.3s ease-out',
                  }}
                >
                  {/* Active glow ring for file being accessed */}
                  {isFileActive && (
                    <circle
                      cx={node.x}
                      cy={node.y}
                      r={node.r + 5}
                      fill="none"
                      stroke={fileColor}
                      strokeWidth={3}
                      opacity={0.7}
                      className={styles.appearingGlow}
                    />
                  )}
                  {/* File glow for selected */}
                  {isSelected && !isFileActive && (
                    <circle
                      cx={node.x}
                      cy={node.y}
                      r={node.r + 4}
                      fill="none"
                      stroke={fileColor}
                      strokeWidth={2}
                      opacity={0.5}
                      className={styles.fileSelectedGlow}
                    />
                  )}
                  {/* File circle */}
                  <circle
                    cx={node.x}
                    cy={node.y}
                    r={isHovered || isFileActive ? node.r + 2 : node.r}
                    fill={fileColor}
                    stroke={hasErrors ? '#ef4444' : isFileActive ? '#ffffff' : isSelected ? '#ffffff' : isHovered ? '#ffffff' : 'none'}
                    strokeWidth={hasErrors || isSelected || isFileActive ? 2 : isHovered ? 1.5 : 0}
                    opacity={isHovered || isFileActive ? 1 : 0.85}
                    style={{ cursor: 'pointer' }}
                    onClick={() => {
                      if (!isSelected) {
                        // Close other popups - only one active at a time
                        setToolPopup(null);
                        setTurnPopup(null);
                        setSelectedFile(file);
                        // Auto-center on the file node
                        centerOnPosition(node.x, node.y);
                      } else {
                        setSelectedFile(null);
                      }
                    }}
                    onMouseEnter={(e) => {
                      setHoveredId(`file-${file.filePath}`);
                      setHoveredFile({
                        file,
                        pos: { x: e.clientX, y: e.clientY },
                      });
                    }}
                    onMouseLeave={() => {
                      setHoveredId(null);
                      setHoveredFile(null);
                    }}
                  />
                  {/* File name - show basename, truncate only if very long */}
                  {node.r > 10 && (
                    <text
                      x={node.x}
                      y={node.y + 3}
                      textAnchor="middle"
                      fontSize={Math.min(8, node.r / 2)}
                      fill={themeColors.nodeTextOnFill}
                      fontWeight="500"
                      pointerEvents="none"
                    >
                      {file.fileName.length > 15 ? file.fileName.slice(0, 12) + '…' : file.fileName}
                    </text>
                  )}
                </g>
              );
            }
            return null;
          })}

          {/* Center origin point if no files */}
          {fileHubNodes.length === 0 && (
            <circle
              cx={CENTER_X}
              cy={CENTER_Y}
              r={10}
              fill={themeColors.cyan}
            />
          )}
        </g>

        {/* Edges from turns to selected file (when file selected) */}
        {selectedFile && (
          <g className={styles.fileConnections}>
            {spiralPositions
              .filter(pos => selectedFile.turnIds.has(pos.turn.id))
              .map((pos, idx) => {
                // Respect playback visibility - only show edges to visible turns
                const turnIdx = spiralPositions.indexOf(pos);
                const isTurnVisible = visibilityState.showAll || visibilityState.visibleTurnIndices.has(turnIdx);
                const isFileVisible = visibilityState.showAll || visibilityState.visibleFilePaths.has(selectedFile.filePath);
                const isVisible = isTurnVisible && isFileVisible;

                return (
                  <line
                    key={`file-edge-${pos.turn.id}`}
                    x1={pos.x}
                    y1={pos.y}
                    x2={selectedFile.x}
                    y2={selectedFile.y}
                    stroke={themeColors.emerald}
                    strokeWidth={2}
                    style={{
                      opacity: isVisible ? 0.6 : 0,
                      transition: 'opacity 0.3s ease-out',
                    }}
                  />
                );
              })}
          </g>
        )}

        {/* Edges from turns to shared tool nodes - drawn first (underneath) */}
        <g className={styles.toolEdges}>
          {spiralPositions.map((pos, turnIdx) => {
            const { x, y, turn } = pos;
            const isHovered = hoveredId === turn.id;
            const isTurnAppearing = visibilityState.partialTurnIndex === turnIdx;

            // Check if turn is visible in playback
            const isTurnVisible = visibilityState.showAll || visibilityState.visibleTurnIndices.has(turnIdx);
            if (!isTurnVisible) return null;

            // Get turn recency for decay calculation
            const recency = visibilityState.turnRecency.get(turnIdx) ?? 0;

            return turn.tools.map((tool) => {
              const sharedNode = toolNodeLookup.get(tool.name);
              if (!sharedNode) return null;

              // Check if tool is visible in playback
              const isToolVisible = visibilityState.showAll || visibilityState.visibleToolIds.has(tool.id);
              if (!isToolVisible) return null;

              // Highlight edges for hovered
              const isManuallyHighlighted = isHovered || hoveredId === sharedNode.name;

              // Calculate decay-based opacity for playback mode
              // recency 0 = current turn (full brightness), higher = older (fading)
              // highlightDecay controls how many turns until fully faded
              let decayOpacity = 0.15; // Base opacity for old edges
              let decayStrokeWidth = 0.5;

              if (!visibilityState.showAll && highlightDecay > 0 && recency <= highlightDecay) {
                // Within decay window - interpolate from full to base
                const decayProgress = recency / highlightDecay;
                decayOpacity = 0.9 - decayProgress * 0.75; // 0.9 → 0.15
                decayStrokeWidth = 2 - decayProgress * 1.5; // 2 → 0.5
              }

              // Manual hover always wins
              const finalOpacity = isManuallyHighlighted ? 0.8 : decayOpacity;
              const finalStrokeWidth = isManuallyHighlighted ? 1.5 : decayStrokeWidth;
              const showPulseAnimation = isTurnAppearing && recency === 0;

              return (
                <line
                  key={`${turn.id}-${tool.id}`}
                  x1={x} y1={y}
                  x2={sharedNode.x} y2={sharedNode.y}
                  stroke={sharedNode.color}
                  strokeWidth={finalStrokeWidth}
                  opacity={finalOpacity}
                  strokeDasharray={tool.status === 'error' ? '3,2' : 'none'}
                  className={showPulseAnimation ? styles.appearingEdge : ''}
                />
              );
            });
          })}
        </g>

        {/* Shared tool nodes in interior ring - bioluminescent style */}
        {sharedToolNodes.map((toolNode) => {
          const isToolHovered = hoveredId === toolNode.name;

          // Check if any usage of this tool is visible in playback
          const isAnyUsageVisible = visibilityState.showAll ||
            toolNode.usages.some(u => visibilityState.visibleToolIds.has(u.tool.id));
          if (!isAnyUsageVisible) return null;

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
          // Use tokensOut for size, fallback to duration if no token data
          const maxTokensOut = Math.max(...turns.map(t => t.tokensOut || 0), 1);
          const maxDuration = Math.max(...turns.map(t => t.durationMs || 0), 1);
          const hasTokenOutData = maxTokensOut > 10 && turns.some(t => (t.tokensOut || 0) !== maxTokensOut);

          return spiralPositions.map((pos, turnIdx) => {
            const { x, y, turn, isAnomaly, activity } = pos;
            const isHovered = hoveredId === turn.id;
            const isSelected = selectedNodeId === turn.id;
            const isError = turn.status === 'error';

            // Playback visibility
            const isVisible = visibilityState.showAll || visibilityState.visibleTurnIndices.has(turnIdx);
            const isAppearing = visibilityState.partialTurnIndex === turnIdx;
            const appearProgress = isAppearing ? visibilityState.partialTurnProgress : 1;

            // Size based on tokens OUT, or fallback to duration (3-18 range)
            const sizeMetric = hasTokenOutData
              ? (turn.tokensOut || 0) / maxTokensOut
              : (turn.durationMs || 0) / maxDuration;
            const baseSize = (3 + sizeMetric * 15) * nodeMultiplier;
            const nodeSize = isAnomaly ? baseSize + 3 : isError ? baseSize + 2 : baseSize;

            // Determine node color
            const nodeColor = isAnomaly ? '#f59e0b' : isError ? '#ef4444' : '#10b981';

            // Calculate opacity and scale for playback animation
            const playbackOpacity = isVisible ? (isAppearing ? appearProgress : 1) : 0;
            const playbackScale = isVisible ? (isAppearing ? 0.5 + appearProgress * 0.5 : 1) : 0.5;

            return (
              <g
                key={turn.id}
                className={`${isVisible ? styles.turnVisible : styles.turnHidden} ${isAppearing ? styles.turnAppearing : ''}`}
                style={{
                  opacity: playbackOpacity,
                  transform: `translate(${x}px, ${y}px) scale(${playbackScale})`,
                  transformOrigin: 'center',
                  pointerEvents: isVisible ? 'auto' : 'none',
                }}
              >
                {/* Appearing glow ring - shows when turn is entering */}
                {isAppearing && (
                  <circle
                    cx={0} cy={0}
                    r={nodeSize + 8}
                    fill="none"
                    stroke={nodeColor}
                    strokeWidth={3}
                    opacity={0.4 + appearProgress * 0.4}
                    className={styles.appearingGlow}
                  />
                )}

                {/* Main node - simplified, no animated pulse rings */}
                <circle
                  cx={0} cy={0}
                  r={isHovered ? nodeSize + 2 : nodeSize}
                  fill={nodeColor}
                  stroke={isHovered || isSelected || isAppearing ? '#fff' : 'none'}
                  strokeWidth={isHovered || isSelected ? 2 : isAppearing ? 1.5 : 0}
                  opacity={isAnomaly || isError ? 1 : 0.9}
                  onClick={(e) => handleNodeClick(turn, e, isAnomaly, activity, x, y)}
                  onMouseEnter={() => setHoveredId(turn.id)}
                  onMouseLeave={() => setHoveredId(null)}
                  style={{ cursor: 'pointer' }}
                />

                {/* Turn number - show on hover, anomalies, or when appearing */}
                {(isHovered || isAnomaly || isSelected || isAppearing) && (
                  <text
                    x={0} y={nodeSize + 12}
                    textAnchor="middle"
                    fontSize="8"
                    fill={themeColors.text}
                    fontWeight="600"
                  >
                    T{turn.turnNumber}
                  </text>
                )}

                {/* Tool count badge when appearing - shows what's being introduced */}
                {isAppearing && turn.tools.length > 0 && (
                  <g>
                    <circle
                      cx={nodeSize + 4} cy={-nodeSize - 4}
                      r={8}
                      fill={themeColors.cyan}
                    />
                    <text
                      x={nodeSize + 4} y={-nodeSize - 1}
                      textAnchor="middle"
                      fontSize="8"
                      fill="#0f1419"
                      fontWeight="700"
                    >
                      {turn.tools.length}
                    </text>
                  </g>
                )}
              </g>
            );
          });
        })()}
      </svg>

      {/* Playback Controls Container - stacked layout */}
      <div className={styles.playbackContainer}>
        {/* Now Playing Banner - shows recent activity history during playback */}
        {!visibilityState.showAll && visibilityState.currentTurnIndex >= 0 && (
          <div className={styles.nowPlayingBanner}>
            <div className={styles.nowPlayingHeader}>
              <span className={styles.nowPlayingTurnBadge}>
                Turn {turns[visibilityState.currentTurnIndex]?.turnNumber ?? '?'}
              </span>
              <span className={styles.nowPlayingToolCount}>
                {visibilityState.partialTurnIndex !== null
                  ? `${Math.floor(visibilityState.partialTurnProgress * (turns[visibilityState.partialTurnIndex]?.tools.length || 0))} / ${turns[visibilityState.partialTurnIndex]?.tools.length || 0} tools`
                  : ''}
              </span>
            </div>
            <div className={styles.nowPlayingHistory}>
              {(() => {
                // Collect recent tools across turns (last 4)
                const recentTools: { tool: TreeTool; turnNumber: number; isCurrent: boolean }[] = [];

                // Add tools from current partial turn
                if (visibilityState.partialTurnIndex !== null) {
                  const currentTurn = turns[visibilityState.partialTurnIndex];
                  const visibleCount = Math.floor(visibilityState.partialTurnProgress * (currentTurn.tools.length + 1));
                  currentTurn.tools.slice(0, visibleCount).forEach(tool => {
                    recentTools.push({ tool, turnNumber: currentTurn.turnNumber, isCurrent: true });
                  });
                }

                // Add tools from previous turns if we need more
                if (recentTools.length < 4 && visibilityState.partialTurnIndex !== null && visibilityState.partialTurnIndex > 0) {
                  for (let i = visibilityState.partialTurnIndex - 1; i >= 0 && recentTools.length < 4; i--) {
                    const turn = turns[i];
                    for (let j = turn.tools.length - 1; j >= 0 && recentTools.length < 4; j--) {
                      recentTools.unshift({ tool: turn.tools[j], turnNumber: turn.turnNumber, isCurrent: false });
                    }
                  }
                }

                // Take last 4 and reverse so newest is at bottom
                const displayTools = recentTools.slice(-4);

                if (displayTools.length === 0) return <div className={styles.nowPlayingEmpty}>Starting...</div>;

                return displayTools.map((item, idx) => {
                  const toolType = getToolType(item.tool.name);
                  const toolColor = getToolColor(item.tool.name);
                  const isNewest = idx === displayTools.length - 1;

                  return (
                    <div
                      key={`${item.tool.id}-${idx}`}
                      className={`${styles.nowPlayingItem} ${isNewest ? styles.nowPlayingItemCurrent : ''}`}
                      style={{ opacity: 0.4 + (idx / displayTools.length) * 0.6 }}
                    >
                      <span
                        className={styles.nowPlayingToolType}
                        style={{ backgroundColor: `${toolColor}20`, color: toolColor, borderColor: `${toolColor}40` }}
                      >
                        {toolType}
                      </span>
                      <span className={styles.nowPlayingToolName}>
                        {item.tool.fullName || item.tool.name}
                      </span>
                    </div>
                  );
                });
              })()}
            </div>
          </div>
        )}

        {/* Playback Controls Bar */}
        <div className={styles.playbackControls}>
          {/* Play/Pause */}
          <button
            className={styles.playbackButton}
            onClick={togglePlayback}
            title={playback.isPlaying ? 'Pause' : 'Play'}
          >
            {playback.isPlaying ? '⏸' : '▶'}
          </button>

          {/* Skip to start */}
          <button
            className={styles.playbackButton}
            onClick={() => seekTo(0)}
            title="Skip to start"
          >
            ⏮
          </button>

          {/* Skip to end */}
          <button
            className={styles.playbackButton}
            onClick={() => seekTo(totalDurationMs)}
            title="Skip to end"
          >
            ⏭
          </button>

          {/* Speed selector - dynamic speeds based on session duration */}
          <div className={styles.speedControls}>
            {speedOptions.map(s => (
              <button
                key={s}
                className={`${styles.speedButton} ${playback.speed === s ? styles.activeSpeed : ''}`}
                onClick={() => setPlaybackSpeed(s)}
                title={`${Math.round(totalDurationMs / s / 1000)}s playback`}
              >
                {s}x
              </button>
            ))}
          </div>

          {/* Decay control - how many turns until connections fade */}
          <div className={styles.decayControl} title={`Highlight decay: ${highlightDecay} turns (0=off)`}>
            <span className={styles.decayLabel}>Decay</span>
            <input
              type="range"
              className={styles.decayScrubber}
              min={0}
              max={30}
              value={highlightDecay}
              onChange={(e) => setHighlightDecay(Number(e.target.value))}
            />
            <span className={styles.decayValue}>{highlightDecay}</span>
          </div>

          {/* Timeline scrubber */}
          <input
            type="range"
            className={styles.playbackScrubber}
            min={0}
            max={totalDurationMs}
            value={playback.currentTimeMs}
            onChange={(e) => seekTo(Number(e.target.value))}
          />

          {/* Time display - shows actual time and estimated playback time */}
          <span
            className={styles.playbackTime}
            title={`Actual session: ${formatPlaybackTime(totalDurationMs)} | Playback at ${playback.speed}x: ${Math.round(totalDurationMs / playback.speed / 1000)}s`}
          >
            {formatPlaybackTime(playback.currentTimeMs, totalDurationMs)} / {formatPlaybackTime(totalDurationMs)}
            {playback.currentTimeMs !== Infinity && playback.currentTimeMs < totalDurationMs && (
              <span className={styles.playbackEstimate}>
                ({Math.round((totalDurationMs - playback.currentTimeMs) / playback.speed / 1000)}s left)
              </span>
            )}
          </span>
        </div>
      </div>

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

      {/* Tool Detail Slideover Panel */}
      {toolPopup && (
        <div
          className={styles.detailSlideover}
          onClick={(e) => e.stopPropagation()}
          onWheel={(e) => e.stopPropagation()}
        >
          {/* Header */}
          <div className={styles.slideoverHeader}>
            <div
              className={styles.slideoverIcon}
              style={{
                backgroundColor: toolPopup.node.color,
                boxShadow: `0 0 20px ${toolPopup.node.color}60`,
              }}
            />
            <div className={styles.slideoverTitle}>
              <span className={styles.slideoverType}>{toolPopup.node.toolType}</span>
              <span className={styles.slideoverName}>{toolPopup.node.fullName || toolPopup.node.name}</span>
            </div>
            <button
              className={styles.slideoverClose}
              onClick={() => setToolPopup(null)}
              aria-label="Close"
            >
              ×
            </button>
          </div>

          {/* Body content */}
          <div className={styles.slideoverBody}>

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
                {toolPopup.metrics.sortedUsages.slice(0, 20).map((usage, idx) => {
                  const turn = turns[usage.turnIndex];
                  return (
                    <div
                      key={`${usage.turnId}-${idx}`}
                      className={`${styles.bioTimelineItem} ${usage.tool.status === 'error' ? styles.bioTimelineItemError : ''}`}
                      style={{ cursor: 'pointer' }}
                      onClick={() => {
                        if (turn) {
                          // Find the spiral position for this turn
                          const pos = spiralPositions.find(p => p.turn.id === turn.id);
                          // Close tool popup and open turn popup
                          setToolPopup(null);
                          setTurnPopup({
                            turn,
                            pos: { x: 0, y: 0 },
                            isAnomaly: pos?.isAnomaly || false,
                            activity: pos?.activity || 0.5,
                          });
                          // Center on the turn node
                          if (pos) {
                            centerOnPosition(pos.x, pos.y);
                          }
                        }
                      }}
                      title={`Click to view Turn ${usage.turnIndex + 1}`}
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
                  );
                })}
                {toolPopup.metrics.sortedUsages.length > 20 && (
                  <div className={styles.bioTimelineMore}>
                    +{toolPopup.metrics.sortedUsages.length - 20} more usages
                  </div>
                )}
              </div>
            )}
          </div>

          {/* Mini subgraph - tool to turns (reuses main graph styling) */}
          <div className={styles.miniSubgraph}>
            <div className={styles.miniSubgraphTitle}>Connections</div>
            <svg width="100%" height="120" viewBox="0 0 340 120">
              <defs>
                {/* Same glow filter as main graph */}
                <filter id="miniGlow" x="-50%" y="-50%" width="200%" height="200%">
                  <feGaussianBlur stdDeviation="3" result="blur" />
                  <feMerge>
                    <feMergeNode in="blur" />
                    <feMergeNode in="SourceGraphic" />
                  </feMerge>
                </filter>
                <filter id="miniErrorGlow" x="-50%" y="-50%" width="200%" height="200%">
                  <feGaussianBlur stdDeviation="4" result="blur" />
                  <feFlood floodColor="#ef4444" floodOpacity="0.4" />
                  <feComposite in2="blur" operator="in" />
                  <feMerge>
                    <feMergeNode />
                    <feMergeNode in="SourceGraphic" />
                  </feMerge>
                </filter>
              </defs>

              {/* Edges first (underneath nodes) */}
              {toolPopup.metrics.sortedUsages.slice(0, 8).map((usage, i) => {
                const totalShown = Math.min(toolPopup.metrics.sortedUsages.length, 8);
                const angle = Math.PI + (Math.PI * (i + 0.5)) / totalShown;
                const radius = 45;
                const x = 170 + radius * Math.cos(angle);
                const y = 60 + radius * Math.sin(angle);
                const isError = usage.tool.status === 'error';
                return (
                  <line
                    key={`edge-${usage.turnId}-${i}`}
                    x1={170} y1={60}
                    x2={x} y2={y}
                    stroke={isError ? '#ef4444' : themeColors.emerald}
                    strokeWidth={2}
                    opacity={0.4}
                    strokeLinecap="round"
                  />
                );
              })}

              {/* Tool node in center - matching main graph styling */}
              <circle
                cx={170} cy={60} r={16}
                fill={toolPopup.node.color}
                filter="url(#miniGlow)"
              />
              <text x={170} y={64} textAnchor="middle" fontSize="10" fill={themeColors.nodeTextOnFill} fontWeight="600">
                {toolPopup.node.toolType.slice(0, 4)}
              </text>

              {/* Connected turns - matching main graph turn node styling */}
              {toolPopup.metrics.sortedUsages.slice(0, 8).map((usage, i) => {
                const totalShown = Math.min(toolPopup.metrics.sortedUsages.length, 8);
                const angle = Math.PI + (Math.PI * (i + 0.5)) / totalShown;
                const radius = 45;
                const x = 170 + radius * Math.cos(angle);
                const y = 60 + radius * Math.sin(angle);
                const isError = usage.tool.status === 'error';
                return (
                  <g key={`mini-${usage.turnId}-${i}`}>
                    <circle
                      cx={x} cy={y} r={10}
                      fill={isError ? '#ef4444' : themeColors.emerald}
                      filter={isError ? 'url(#miniErrorGlow)' : 'url(#miniGlow)'}
                    />
                    <text x={x} y={y + 3} textAnchor="middle" fontSize="8" fill={themeColors.nodeTextOnFill} fontWeight="600">
                      T{usage.turnIndex + 1}
                    </text>
                  </g>
                );
              })}

              {/* More indicator */}
              {toolPopup.metrics.sortedUsages.length > 8 && (
                <text x={170} y={115} textAnchor="middle" fontSize="10" fill={themeColors.textMuted}>
                  +{toolPopup.metrics.sortedUsages.length - 8} more
                </text>
              )}
            </svg>
          </div>
          </div>{/* End slideoverBody */}
        </div>
      )}

      {/* Turn Detail Slideover Panel */}
      {turnPopup && (
        <div
          className={styles.detailSlideover}
          onClick={(e) => e.stopPropagation()}
          onWheel={(e) => e.stopPropagation()}
        >
          {/* Header */}
          <div className={styles.slideoverHeader}>
            <div
              className={styles.slideoverIcon}
              style={{
                backgroundColor: turnPopup.isAnomaly ? '#f59e0b' : turnPopup.turn.status === 'error' ? '#ef4444' : '#10b981',
                boxShadow: `0 0 20px ${turnPopup.isAnomaly ? '#f59e0b' : turnPopup.turn.status === 'error' ? '#ef4444' : '#10b981'}60`,
              }}
            >
              <span style={{ fontSize: '14px', fontWeight: 700, color: themeColors.nodeTextOnFill }}>
                {turnPopup.turn.turnNumber}
              </span>
            </div>
            <div className={styles.slideoverTitle}>
              <span className={styles.slideoverType}>
                {turnPopup.isAnomaly ? 'ANOMALY TURN' : turnPopup.turn.status === 'error' ? 'ERROR TURN' : 'TURN'}
              </span>
              <span className={styles.slideoverName}>Turn {turnPopup.turn.turnNumber}</span>
            </div>
            <button
              className={styles.slideoverClose}
              onClick={() => setTurnPopup(null)}
              aria-label="Close"
            >
              ×
            </button>
          </div>

          {/* Body content */}
          <div className={styles.slideoverBody}>

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
                        style={{ borderLeftColor: tool.status === 'error' ? undefined : toolColor, cursor: 'pointer' }}
                        onClick={(e) => {
                          e.stopPropagation();
                          // Create an ad-hoc tool node for the popup (even if filtered)
                          const adHocNode: SharedToolNode = {
                            name: tool.name,
                            fullName: tool.fullName,
                            displayName: getToolDisplayName(tool.name),
                            toolType,
                            color: toolColor,
                            usages: [{ turnIndex: turnPopup.turn.turnNumber - 1, turnId: turnPopup.turn.id, tool }],
                            x: 0,
                            y: 0,
                            hasError: tool.status === 'error',
                          };
                          // Find other usages of this tool across all turns
                          turns.forEach((t, tIdx) => {
                            if (t.id === turnPopup.turn.id) return;
                            t.tools.forEach(tt => {
                              if (tt.name === tool.name) {
                                adHocNode.usages.push({ turnIndex: tIdx, turnId: t.id, tool: tt });
                              }
                            });
                          });
                          adHocNode.usages.sort((a, b) => a.turnIndex - b.turnIndex);
                          const totalDuration = adHocNode.usages.reduce((sum, u) => sum + u.tool.durationMs, 0);
                          const totalCost = adHocNode.usages.reduce((sum, u) => sum + (u.tool.cost || 0), 0);
                          const errorCount = adHocNode.usages.filter(u => u.tool.status === 'error').length;
                          setToolPopup({
                            node: adHocNode,
                            pos: { x: e.clientX + 20, y: e.clientY - 50 },
                            metrics: {
                              totalDuration,
                              avgDuration: totalDuration / adHocNode.usages.length,
                              minDuration: Math.min(...adHocNode.usages.map(u => u.tool.durationMs)),
                              maxDuration: Math.max(...adHocNode.usages.map(u => u.tool.durationMs)),
                              totalCost,
                              errorCount,
                              errorRate: errorCount / adHocNode.usages.length,
                              sortedUsages: adHocNode.usages,
                            },
                          });
                          setTurnPopup(null);
                        }}
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

          {/* Mini subgraph - turn to tools and files (reuses main graph styling) */}
          <div className={styles.miniSubgraph}>
            <div className={styles.miniSubgraphTitle}>Connections</div>
            <svg width="100%" height="140" viewBox="0 0 340 140">
              <defs>
                {/* Same glow filters as main graph */}
                <filter id="turnMiniGlow" x="-50%" y="-50%" width="200%" height="200%">
                  <feGaussianBlur stdDeviation="3" result="blur" />
                  <feMerge>
                    <feMergeNode in="blur" />
                    <feMergeNode in="SourceGraphic" />
                  </feMerge>
                </filter>
                <filter id="turnMiniErrorGlow" x="-50%" y="-50%" width="200%" height="200%">
                  <feGaussianBlur stdDeviation="4" result="blur" />
                  <feFlood floodColor="#ef4444" floodOpacity="0.4" />
                  <feComposite in2="blur" operator="in" />
                  <feMerge>
                    <feMergeNode />
                    <feMergeNode in="SourceGraphic" />
                  </feMerge>
                </filter>
                <filter id="turnMiniAnomalyGlow" x="-50%" y="-50%" width="200%" height="200%">
                  <feGaussianBlur stdDeviation="4" result="blur" />
                  <feFlood floodColor="#f59e0b" floodOpacity="0.4" />
                  <feComposite in2="blur" operator="in" />
                  <feMerge>
                    <feMergeNode />
                    <feMergeNode in="SourceGraphic" />
                  </feMerge>
                </filter>
              </defs>

              {/* Edges first (underneath nodes) */}
              {turnPopup.turn.tools.slice(0, 5).map((tool, i) => {
                const totalTools = Math.min(turnPopup.turn.tools.length, 5);
                const angle = Math.PI * 0.7 + (Math.PI * 0.6 * (i + 0.5)) / totalTools;
                const radius = 50;
                const x = 170 + radius * Math.cos(angle);
                const y = 70 + radius * Math.sin(angle);
                const toolColor = getToolColor(tool.name);
                const isError = tool.status === 'error';
                return (
                  <line
                    key={`edge-tool-${tool.id}`}
                    x1={170} y1={70}
                    x2={x} y2={y}
                    stroke={isError ? '#ef4444' : toolColor}
                    strokeWidth={2}
                    opacity={0.4}
                    strokeLinecap="round"
                  />
                );
              })}

              {/* File edges */}
              {(() => {
                const turnFiles = fileDirectories.flatMap(dir =>
                  dir.files.filter(f => f.turnIds.has(turnPopup.turn.id))
                ).slice(0, 5);
                return turnFiles.map((file, i) => {
                  const totalFiles = Math.min(turnFiles.length, 5);
                  const angle = -Math.PI * 0.3 + (Math.PI * 0.6 * (i + 0.5)) / totalFiles;
                  const radius = 50;
                  const x = 170 + radius * Math.cos(angle);
                  const y = 70 + radius * Math.sin(angle);
                  const fileColor = {
                    go: themeColors.cyan,
                    tsx: '#14b8a6',
                    ts: '#0d9488',
                    ail: themeColors.emerald,
                  }[file.fileType] || themeColors.emerald;
                  return (
                    <line
                      key={`edge-file-${file.filePath}`}
                      x1={170} y1={70}
                      x2={x} y2={y}
                      stroke={fileColor}
                      strokeWidth={2}
                      opacity={0.3}
                      strokeDasharray="4,3"
                      strokeLinecap="round"
                    />
                  );
                });
              })()}

              {/* Turn node in center - matching main graph styling */}
              <circle
                cx={170} cy={70} r={18}
                fill={turnPopup.isAnomaly ? '#f59e0b' : turnPopup.turn.status === 'error' ? '#ef4444' : themeColors.emerald}
                filter={turnPopup.isAnomaly ? 'url(#turnMiniAnomalyGlow)' : turnPopup.turn.status === 'error' ? 'url(#turnMiniErrorGlow)' : 'url(#turnMiniGlow)'}
              />
              <text x={170} y={74} textAnchor="middle" fontSize="11" fill={themeColors.nodeTextOnFill} fontWeight="700">
                T{turnPopup.turn.turnNumber}
              </text>

              {/* Tool nodes - matching main graph styling */}
              {turnPopup.turn.tools.slice(0, 5).map((tool, i) => {
                const totalTools = Math.min(turnPopup.turn.tools.length, 5);
                const angle = Math.PI * 0.7 + (Math.PI * 0.6 * (i + 0.5)) / totalTools;
                const radius = 50;
                const x = 170 + radius * Math.cos(angle);
                const y = 70 + radius * Math.sin(angle);
                const toolColor = getToolColor(tool.name);
                const isError = tool.status === 'error';
                return (
                  <g key={`turn-tool-${tool.id}`}>
                    <circle
                      cx={x} cy={y} r={10}
                      fill={isError ? '#ef4444' : toolColor}
                      filter={isError ? 'url(#turnMiniErrorGlow)' : 'url(#turnMiniGlow)'}
                    />
                    <text x={x} y={y + 3} textAnchor="middle" fontSize="7" fill={themeColors.nodeTextOnFill} fontWeight="600">
                      {getToolType(tool.name).slice(0, 4)}
                    </text>
                  </g>
                );
              })}

              {/* File nodes - matching main graph styling */}
              {(() => {
                const turnFiles = fileDirectories.flatMap(dir =>
                  dir.files.filter(f => f.turnIds.has(turnPopup.turn.id))
                ).slice(0, 5);
                return turnFiles.map((file, i) => {
                  const totalFiles = Math.min(turnFiles.length, 5);
                  const angle = -Math.PI * 0.3 + (Math.PI * 0.6 * (i + 0.5)) / totalFiles;
                  const radius = 50;
                  const x = 170 + radius * Math.cos(angle);
                  const y = 70 + radius * Math.sin(angle);
                  const fileColor = {
                    go: themeColors.cyan,
                    tsx: '#14b8a6',
                    ts: '#0d9488',
                    ail: themeColors.emerald,
                  }[file.fileType] || themeColors.emerald;
                  return (
                    <g key={`turn-file-${file.filePath}`}>
                      <circle
                        cx={x} cy={y} r={10}
                        fill={fileColor}
                        filter="url(#turnMiniGlow)"
                      />
                      <text x={x} y={y + 3} textAnchor="middle" fontSize="8" fill={themeColors.nodeTextOnFill}>
                        📄
                      </text>
                    </g>
                  );
                });
              })()}

              {/* Legend */}
              <text x={20} y={130} fontSize="9" fill={themeColors.textMuted}>Tools</text>
              <text x={280} y={130} fontSize="9" fill={themeColors.textMuted}>Files</text>
            </svg>
          </div>
          </div>{/* End slideoverBody */}
        </div>
      )}

      {/* File Hover Tooltip */}
      {hoveredFile && !selectedFile && (
        <div
          className={styles.fileTooltip}
          style={{
            left: hoveredFile.pos.x + 15,
            top: hoveredFile.pos.y - 10,
          }}
        >
          <div className={styles.fileTooltipName}>{hoveredFile.file.fileName}</div>
          <div className={styles.fileTooltipPath}>{hoveredFile.file.directory}</div>
          <div className={styles.fileTooltipOps}>
            {hoveredFile.file.readCount > 0 && <span>📖{hoveredFile.file.readCount}</span>}
            {hoveredFile.file.editCount > 0 && <span>✏️{hoveredFile.file.editCount}</span>}
            {hoveredFile.file.writeCount > 0 && <span>📝{hoveredFile.file.writeCount}</span>}
            {hoveredFile.file.errorCount > 0 && <span style={{ color: '#ef4444' }}>⚠️{hoveredFile.file.errorCount}</span>}
          </div>
        </div>
      )}

      {/* Directory Hover Tooltip */}
      {hoveredDir && (
        <div
          className={styles.fileTooltip}
          style={{
            left: hoveredDir.pos.x + 15,
            top: hoveredDir.pos.y - 10,
          }}
        >
          <div className={styles.fileTooltipName}>📁 {hoveredDir.dir.name}</div>
          <div className={styles.fileTooltipPath}>{hoveredDir.dir.path}</div>
          <div className={styles.fileTooltipOps}>
            <span>{hoveredDir.dir.files.length} files</span>
            <span>{hoveredDir.dir.totalOps} ops</span>
            {hoveredDir.dir.errorCount > 0 && <span style={{ color: '#ef4444' }}>⚠️{hoveredDir.dir.errorCount}</span>}
          </div>
          <div style={{ fontSize: '10px', color: '#6b7280', marginTop: '4px' }}>
            Click to {expandedDirs.has(hoveredDir.dir.path) ? 'collapse' : 'expand'}
          </div>
        </div>
      )}

      {/* File Detail Slideover Panel */}
      {selectedFile && (
        <div
          className={styles.detailSlideover}
          onClick={(e) => e.stopPropagation()}
          onWheel={(e) => e.stopPropagation()}
        >
          {/* Header */}
          <div className={styles.slideoverHeader}>
            <div
              className={styles.slideoverIcon}
              style={{
                backgroundColor: {
                  go: themeColors.cyan,
                  tsx: '#14b8a6',
                  ts: '#0d9488',
                  js: '#34d399',
                  ail: themeColors.emerald,
                  md: '#6ee7b7',
                  json: '#2dd4bf',
                  yaml: '#5eead4',
                  css: '#22d3ee',
                }[selectedFile.fileType] || themeColors.emerald,
              }}
            >
              📄
            </div>
            <div className={styles.slideoverTitle}>
              <span className={styles.slideoverType}>FILE</span>
              <span className={styles.slideoverName}>{selectedFile.fileName}</span>
            </div>
            <button
              className={styles.slideoverClose}
              onClick={() => setSelectedFile(null)}
              aria-label="Close"
            >
              ×
            </button>
          </div>

          {/* Body content */}
          <div className={styles.slideoverBody}>
            {/* Full path */}
            <div className={styles.filePath}>{selectedFile.filePath}</div>

            {/* Operation Summary - using same metrics grid as other slideovers */}
            <div className={styles.bioPopoverMetrics}>
              <div className={styles.bioMetric}>
                <span className={styles.bioMetricValue}>{selectedFile.readCount}</span>
                <span className={styles.bioMetricLabel}>reads</span>
              </div>
              <div className={styles.bioMetric}>
                <span className={styles.bioMetricValue}>{selectedFile.editCount}</span>
                <span className={styles.bioMetricLabel}>edits</span>
              </div>
              <div className={styles.bioMetric}>
                <span className={styles.bioMetricValue}>{selectedFile.writeCount}</span>
                <span className={styles.bioMetricLabel}>writes</span>
              </div>
              {selectedFile.errorCount > 0 && (
                <div className={`${styles.bioMetric} ${styles.bioMetricError}`}>
                  <span className={styles.bioMetricValue}>{selectedFile.errorCount}</span>
                  <span className={styles.bioMetricLabel}>errors</span>
                </div>
              )}
            </div>

            {/* Turn range */}
            <div className={styles.bioTurnTimeline}>
              <button
                className={styles.bioTimelineToggle}
                onClick={() => {/* Could add expand/collapse */}}
              >
                <span className={styles.bioTimelineToggleIcon}>▼</span>
                <span>Operations Timeline</span>
                <span className={styles.bioTimelineCount}>{selectedFile.operations.length} ops</span>
              </button>

              <div className={styles.bioTimelineContent}>
                {selectedFile.operations.map((op, i) => {
                  const opColor = op.toolType === 'Read' ? themeColors.fileRead :
                                  op.toolType === 'Edit' ? themeColors.fileEdit :
                                  themeColors.fileWrite;
                  return (
                    <div
                      key={i}
                      className={`${styles.bioTimelineItem} ${op.status === 'error' ? styles.bioTimelineItemError : ''}`}
                      style={{ cursor: 'pointer' }}
                      onClick={() => {
                        // Navigate to the turn that performed this operation
                        const turn = turns.find(t => t.id === op.turnId);
                        if (turn) {
                          const pos = spiralPositions.find(p => p.turn.id === op.turnId);
                          setSelectedFile(null);
                          setTurnPopup({
                            turn,
                            pos: { x: 0, y: 0 },
                            isAnomaly: pos?.isAnomaly || false,
                            activity: pos?.activity || 0.5,
                          });
                          if (pos) {
                            centerOnPosition(pos.x, pos.y);
                          }
                        }
                      }}
                      title={`Click to view Turn ${op.turnNumber}`}
                    >
                      <span className={styles.bioTimelineTurn}>T{op.turnNumber}</span>
                      <span
                        className={styles.bioTimelineDuration}
                        style={{ color: op.status === 'error' ? undefined : opColor }}
                      >
                        {op.toolType}
                      </span>
                      <span className={styles.bioTimelineCost}>{formatDuration(op.durationMs)}</span>
                      <span className={styles.bioTimelineStatus}>
                        {op.status === 'error' ? '✗' : '✓'}
                      </span>
                    </div>
                  );
                })}
              </div>
            </div>

            {/* Mini subgraph - file to turns (reuses main graph styling) */}
            <div className={styles.miniSubgraph}>
              <div className={styles.miniSubgraphTitle}>Connected Turns</div>
              <svg width="100%" height="120" viewBox="0 0 340 120">
                <defs>
                  {/* Same glow filters as main graph */}
                  <filter id="fileMiniGlow" x="-50%" y="-50%" width="200%" height="200%">
                    <feGaussianBlur stdDeviation="3" result="blur" />
                    <feMerge>
                      <feMergeNode in="blur" />
                      <feMergeNode in="SourceGraphic" />
                    </feMerge>
                  </filter>
                  <filter id="fileMiniErrorGlow" x="-50%" y="-50%" width="200%" height="200%">
                    <feGaussianBlur stdDeviation="4" result="blur" />
                    <feFlood floodColor="#ef4444" floodOpacity="0.4" />
                    <feComposite in2="blur" operator="in" />
                    <feMerge>
                      <feMergeNode />
                      <feMergeNode in="SourceGraphic" />
                    </feMerge>
                  </filter>
                </defs>

                {/* Edges first (underneath nodes) */}
                {(() => {
                  const uniqueTurnIds = Array.from(selectedFile.turnIds).slice(0, 8);
                  return uniqueTurnIds.map((turnId, i) => {
                    const totalShown = Math.min(uniqueTurnIds.length, 8);
                    const angle = Math.PI + (Math.PI * (i + 0.5)) / totalShown;
                    const radius = 45;
                    const x = 170 + radius * Math.cos(angle);
                    const y = 60 + radius * Math.sin(angle);
                    const turnOps = selectedFile.operations.filter(op => op.turnId === turnId);
                    const hasError = turnOps.some(op => op.status === 'error');
                    return (
                      <line
                        key={`edge-${turnId}`}
                        x1={170} y1={60}
                        x2={x} y2={y}
                        stroke={hasError ? '#ef4444' : themeColors.emerald}
                        strokeWidth={2}
                        opacity={0.4}
                        strokeLinecap="round"
                      />
                    );
                  });
                })()}

                {/* File node in center - matching main graph styling */}
                <circle
                  cx={170} cy={60} r={16}
                  fill={{
                    go: themeColors.cyan,
                    tsx: '#14b8a6',
                    ts: '#0d9488',
                    ail: themeColors.emerald,
                  }[selectedFile.fileType] || themeColors.emerald}
                  filter="url(#fileMiniGlow)"
                />
                <text x={170} y={64} textAnchor="middle" fontSize="11" fill={themeColors.nodeTextOnFill}>
                  📄
                </text>

                {/* Connected turns - matching main graph turn node styling */}
                {(() => {
                  const uniqueTurnIds = Array.from(selectedFile.turnIds).slice(0, 8);
                  return uniqueTurnIds.map((turnId, i) => {
                    const totalShown = Math.min(uniqueTurnIds.length, 8);
                    const angle = Math.PI + (Math.PI * (i + 0.5)) / totalShown;
                    const radius = 45;
                    const x = 170 + radius * Math.cos(angle);
                    const y = 60 + radius * Math.sin(angle);
                    const turnOps = selectedFile.operations.filter(op => op.turnId === turnId);
                    const hasError = turnOps.some(op => op.status === 'error');
                    const turnNum = turnOps[0]?.turnNumber || '?';
                    return (
                      <g key={`file-turn-${turnId}`}>
                        <circle
                          cx={x} cy={y} r={10}
                          fill={hasError ? '#ef4444' : themeColors.emerald}
                          filter={hasError ? 'url(#fileMiniErrorGlow)' : 'url(#fileMiniGlow)'}
                        />
                        <text x={x} y={y + 3} textAnchor="middle" fontSize="8" fill={themeColors.nodeTextOnFill} fontWeight="600">
                          T{turnNum}
                        </text>
                      </g>
                    );
                  });
                })()}

                {/* More indicator */}
                {selectedFile.turnIds.size > 8 && (
                  <text x={170} y={115} textAnchor="middle" fontSize="10" fill={themeColors.textMuted}>
                    +{selectedFile.turnIds.size - 8} more
                  </text>
                )}
              </svg>
            </div>
          </div>
        </div>
      )}
    </div>
  );
};

export default EvolutionTree;
