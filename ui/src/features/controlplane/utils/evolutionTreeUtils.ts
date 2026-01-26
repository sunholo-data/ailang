/**
 * EvolutionTree utility functions
 * Extracted from EvolutionTree.tsx (PR 5 - M-DASHBOARD-SIMPLIFICATION)
 *
 * Contains:
 * - Tool type detection and coloring
 * - File path extraction and manipulation
 * - Span type detection (turn, tool, session)
 * - Geometry helpers (polar coordinates, arc paths)
 */

import type { Span } from '../components/ExecHierarchy/types';

// ============================================================================
// Types
// ============================================================================

export interface TreeSession {
  id: string;
  name: string;
  durationMs: number;
  cost: number;
  tokensIn: number;
  tokensOut: number;
}

export interface TreeTurn {
  id: string;
  turnNumber: number;
  durationMs: number;
  cost: number;
  tokensIn: number;
  tokensOut: number;
  status: 'ok' | 'error' | 'pending';
  tools: TreeTool[];
}

export interface TreeTool {
  id: string;
  name: string;
  fullName?: string;
  durationMs: number;
  status: 'ok' | 'error';
  cost?: number;
}

export interface SharedToolNode {
  name: string;
  fullName?: string;
  displayName: string;
  toolType: string;
  color: string;
  usages: { turnIndex: number; turnId: string; tool: TreeTool }[];
  x: number;
  y: number;
  hasError: boolean;
}

export interface FileOperation {
  turnId: string;
  turnNumber: number;
  toolType: 'Read' | 'Edit' | 'Write';
  toolName: string;
  durationMs: number;
  status: 'ok' | 'error';
}

export interface FileNode {
  filePath: string;
  fileName: string;
  fileType: string;
  directory: string;
  operations: FileOperation[];
  totalOps: number;
  readCount: number;
  editCount: number;
  writeCount: number;
  errorCount: number;
  turnIds: Set<string>;
  firstTurn: number;
  lastTurn: number;
  x: number;
  y: number;
  radius: number;
}

export interface DirectoryNode {
  path: string;
  name: string;
  files: FileNode[];
  totalOps: number;
  errorCount: number;
  x: number;
  y: number;
  radius: number;
}

export interface SpiralPosition {
  x: number;
  y: number;
  angle: number;
  radius: number;
  nodeRadius: number;
  activity: number;
  isAnomaly: boolean;
  turn: TreeTurn;
}

// ============================================================================
// Constants
// ============================================================================

export const TOOL_COLORS: Record<string, string> = {
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

// ============================================================================
// Tool Utilities
// ============================================================================

/**
 * Extract tool type from name
 */
export function getToolType(name: string): string {
  const prefixes = ['Read', 'Write', 'Edit', 'Bash', 'Grep', 'Glob', 'Task', 'WebFetch', 'WebSearch'];
  for (const prefix of prefixes) {
    if (name.startsWith(prefix) || name.toLowerCase().includes(prefix.toLowerCase())) {
      return prefix;
    }
  }
  return 'default';
}

/**
 * Get color for tool type
 */
export function getToolColor(name: string): string {
  const toolType = getToolType(name);
  return TOOL_COLORS[toolType] || TOOL_COLORS.default;
}

/**
 * Create display name (truncate file paths)
 */
export function getToolDisplayName(name: string): string {
  const pathMatch = name.match(/[/\\]([^/\\]+)$/);
  if (pathMatch) {
    return pathMatch[1].length > 20 ? pathMatch[1].slice(0, 17) + '...' : pathMatch[1];
  }
  return name.length > 20 ? name.slice(0, 17) + '...' : name;
}

// ============================================================================
// File Path Utilities
// ============================================================================

/**
 * Extract file path from tool name
 * e.g., "Read: /path/to/file.go" → "/path/to/file.go"
 */
export function extractFilePath(toolName: string): string | null {
  // Pattern 1: "ToolType: /path/to/file"
  const colonMatch = toolName.match(/^[A-Za-z]+:\s*(.+)$/);
  if (colonMatch) {
    const path = colonMatch[1].trim();
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

/**
 * Get file extension from path
 */
export function getFileExtension(filePath: string): string {
  const match = filePath.match(/\.(\w+)$/);
  return match ? match[1] : '';
}

/**
 * Get directory path from file path
 */
export function getDirectoryPath(filePath: string): string {
  const lastSlash = Math.max(filePath.lastIndexOf('/'), filePath.lastIndexOf('\\'));
  if (lastSlash === -1) return '/';
  return filePath.slice(0, lastSlash) || '/';
}

// ============================================================================
// Span Type Detection
// ============================================================================

/**
 * Check if span represents a turn
 */
export function isTurnSpan(name: string): boolean {
  return (
    name.includes('turn') ||
    name === 'api_request' ||
    name.startsWith('exec.turn')
  );
}

/**
 * Check if span represents a tool use
 */
export function isToolSpan(name: string): boolean {
  return (
    name.includes('tool') ||
    name.startsWith('claude_code.tool') ||
    name.startsWith('exec.tool_use')
  );
}

/**
 * Check if span represents a session/execution
 */
export function isSessionSpan(name: string): boolean {
  return (
    name === 'claude_code.session' ||
    name === 'coordinator.task.execute' ||
    name === 'claude.execute' ||
    name === 'gemini.execute' ||
    name.includes('.execute') ||
    name.includes('.session')
  );
}

/**
 * Extract full tool name from span attributes
 */
export function extractFullToolName(span: { name: string; attributes?: Record<string, unknown> }): string | null {
  const attrs = span.attributes;
  if (!attrs) return null;

  // Claude Code tools have specific attribute patterns
  // Try various known attribute keys
  const toolAttrKeys = [
    'tool.name',
    'tool_name',
    'claude_code.tool.name',
    'command',
    'file_path',
    'pattern',
  ];

  for (const key of toolAttrKeys) {
    if (attrs[key] && typeof attrs[key] === 'string') {
      const value = attrs[key] as string;
      // If we have a value, combine with tool type from span name
      const toolType = span.name.split('.').pop() || span.name;
      if (key === 'file_path') {
        return `${toolType}: ${value}`;
      }
      if (key === 'command') {
        // Truncate long commands
        const truncated = value.length > 50 ? value.slice(0, 47) + '...' : value;
        return `Bash: ${truncated}`;
      }
      if (key === 'pattern') {
        return `${toolType}: ${value}`;
      }
      return value;
    }
  }

  // Fallback: use display_name if available
  if (attrs['display_name'] && typeof attrs['display_name'] === 'string') {
    return attrs['display_name'] as string;
  }

  return null;
}

// ============================================================================
// Geometry Helpers
// ============================================================================

/**
 * Convert polar coordinates to Cartesian
 */
export function polarToCartesian(
  cx: number,
  cy: number,
  radius: number,
  angleInDegrees: number
): { x: number; y: number } {
  const angleInRadians = ((angleInDegrees - 90) * Math.PI) / 180.0;
  return {
    x: cx + radius * Math.cos(angleInRadians),
    y: cy + radius * Math.sin(angleInRadians),
  };
}

/**
 * Generate SVG arc path description
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
 * Generate branch path between two points with curve
 */
export function generateBranchPath(
  startX: number,
  startY: number,
  endX: number,
  endY: number
): string {
  const midX = (startX + endX) / 2;
  const midY = (startY + endY) / 2;
  // Curve toward the center slightly
  const controlX = midX * 0.9;
  const controlY = midY * 0.9;
  return `M ${startX} ${startY} Q ${controlX} ${controlY} ${endX} ${endY}`;
}

/**
 * Generate spiral path through positions
 */
export function generateSpiralPath(
  positions: SpiralPosition[],
  centerX: number,
  centerY: number
): string {
  if (positions.length === 0) return '';

  // Start from center with a gentle curve
  let path = `M ${centerX} ${centerY}`;

  // Use quadratic bezier curves for smooth transitions
  positions.forEach((pos, i) => {
    if (i === 0) {
      // First point: gentle curve from center
      const controlX = (centerX + pos.x) / 2;
      const controlY = (centerY + pos.y) / 2;
      path += ` Q ${controlX} ${controlY} ${pos.x} ${pos.y}`;
    } else {
      // Subsequent points: smooth curve through
      const prev = positions[i - 1];
      const controlX = (prev.x + pos.x) / 2;
      const controlY = (prev.y + pos.y) / 2;
      path += ` Q ${controlX} ${controlY} ${pos.x} ${pos.y}`;
    }
  });

  return path;
}

// ============================================================================
// Span Filtering
// ============================================================================

/**
 * Filter spans by hidden span types (recursive)
 */
export function filterSpans(spans: Span[], hiddenSpanTypes?: Set<string>): Span[] {
  if (!hiddenSpanTypes || hiddenSpanTypes.size === 0) return spans;

  const filter = (spanList: Span[]): Span[] => {
    const result: Span[] = [];
    for (const span of spanList) {
      if (hiddenSpanTypes.has(span.name)) {
        // Span is hidden - promote its children
        if (span.children && span.children.length > 0) {
          result.push(...filter(span.children));
        }
      } else {
        // Keep span, but filter its children
        result.push({
          ...span,
          children: span.children ? filter(span.children) : undefined,
        });
      }
    }
    return result;
  };

  return filter(spans);
}

// ============================================================================
// Anomaly Detection
// ============================================================================

/**
 * Detect anomalies in turn sequence based on statistical deviation
 */
export function detectAnomalies(turns: TreeTurn[]): Set<number> {
  if (turns.length < 5) return new Set();

  const anomalies = new Set<number>();

  // Calculate metrics for each turn
  const metrics = turns.map(turn => ({
    duration: turn.durationMs,
    cost: turn.cost || 0,
    tokensTotal: (turn.tokensIn || 0) + (turn.tokensOut || 0),
    toolCount: turn.tools.length,
    hasError: turn.status === 'error',
  }));

  // Calculate means and standard deviations
  const calcStats = (values: number[]) => {
    const mean = values.reduce((a, b) => a + b, 0) / values.length;
    const variance = values.reduce((sum, v) => sum + Math.pow(v - mean, 2), 0) / values.length;
    return { mean, stdDev: Math.sqrt(variance) };
  };

  const durationStats = calcStats(metrics.map(m => m.duration));
  const costStats = calcStats(metrics.map(m => m.cost));
  const tokenStats = calcStats(metrics.map(m => m.tokensTotal));
  const toolStats = calcStats(metrics.map(m => m.toolCount));

  // Mark anomalies (values > 2 standard deviations from mean)
  metrics.forEach((m, i) => {
    const threshold = 2;

    // Check for statistical outliers
    if (durationStats.stdDev > 0 && Math.abs(m.duration - durationStats.mean) > threshold * durationStats.stdDev) {
      anomalies.add(i);
    }
    if (costStats.stdDev > 0 && Math.abs(m.cost - costStats.mean) > threshold * costStats.stdDev) {
      anomalies.add(i);
    }
    if (tokenStats.stdDev > 0 && Math.abs(m.tokensTotal - tokenStats.mean) > threshold * tokenStats.stdDev) {
      anomalies.add(i);
    }
    if (toolStats.stdDev > 0 && Math.abs(m.toolCount - toolStats.mean) > threshold * toolStats.stdDev) {
      anomalies.add(i);
    }

    // Errors are always anomalies
    if (m.hasError) {
      anomalies.add(i);
    }
  });

  return anomalies;
}
