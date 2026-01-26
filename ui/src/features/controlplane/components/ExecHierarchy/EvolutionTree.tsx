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

// Import types and utilities from extracted modules
import {
  TOOL_COLORS,
  getToolColor,
  getToolType,
  getToolDisplayName,
  extractFilePath,
  getDirectoryPath,
} from '../../utils/evolutionTreeUtils';
import {
  ToolSlideover,
  TurnSlideover,
  FileSlideover,
  FileTooltip,
  DirTooltip,
  formatDuration,
  formatCost,
  type ThemeColors,
  type ToolPopupState,
  type TurnPopupState,
  type FileHoverState,
  type DirHoverState,
} from './EvolutionTreeSlideovers';
import {
  type TreeSession,
  type TreeTurn,
  type TreeTool,
  type SharedToolNode,
  type FileNode,
  type DirectoryNode,
  type PackedNode,
  type SpiralPosition,
  TOOL_TYPE_RINGS,
  buildFileHierarchy,
  calculateFileHubLayout,
  detectAnomalies,
  calculateSpiralPositions,
  buildSharedToolNodes,
  polarToCartesian,
  describeArc,
  generateBranchPath,
  filterSpans,
  buildTreeData,
} from '../../utils/evolutionTreeBuilders';

// ============================================================================
// Props Interface
// ============================================================================

export interface EvolutionTreeProps {
  spans?: Span[];
  nodes?: HierarchyNode[];
  selectedNodeId?: string | null;
  onNodeClick?: (node: HierarchyNode, event?: React.MouseEvent) => void;
  hiddenSpanTypes?: Set<string>;
  isExpanded?: boolean;
  // Theme from parent (syncs with app-level theme toggle)
  theme?: 'dark' | 'light';
  // Callback when chat context is requested (M-CHAT-HISTORY-DB Phase 3)
  onChatContextClick?: () => void;
}

// ============================================================================
// Component
// ============================================================================

// NOTE: All builder functions extracted to evolutionTreeBuilders.ts
// generateSpiralPath is a local variant that uses smooth S curves
function generateSpiralPath(positions: SpiralPosition[], centerX: number, centerY: number): string {
  if (positions.length === 0) return '';
  let path = `M ${centerX} ${centerY}`;
  positions.forEach((pos, i) => {
    if (i === 0) {
      path += ` Q ${centerX + (pos.x - centerX) * 0.5} ${centerY + (pos.y - centerY) * 0.5}, ${pos.x} ${pos.y}`;
    } else {
      const prev = positions[i - 1];
      const cpX = prev.x + (pos.x - prev.x) * 0.5;
      const cpY = prev.y + (pos.y - prev.y) * 0.5;
      path += ` S ${cpX} ${cpY}, ${pos.x} ${pos.y}`;
    }
  });
  return path;
}

export const EvolutionTree: React.FC<EvolutionTreeProps> = ({
  spans,
  nodes,
  selectedNodeId,
  onNodeClick,
  hiddenSpanTypes,
  isExpanded,
  theme,
  onChatContextClick,
}) => {
  const containerRef = useRef<HTMLDivElement>(null);
  const svgRef = useRef<SVGSVGElement>(null);
  const [hoveredId, setHoveredId] = useState<string | null>(null);
  const [showLegend, setShowLegend] = useState(false);
  const [showControls, setShowControls] = useState(false);

  // Bioluminescent popup state for tools
  const [toolPopup, setToolPopup] = useState<ToolPopupState | null>(null);
  const [usageExpanded, setUsageExpanded] = useState(true);

  // Bioluminescent popup state for turns
  const [turnPopup, setTurnPopup] = useState<TurnPopupState | null>(null);
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
  const [hoveredFile, setHoveredFile] = useState<FileHoverState | null>(null);
  const [hoveredDir, setHoveredDir] = useState<DirHoverState | null>(null);

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

  // Use theme prop when provided (from app toggle), fall back to system preference
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

  // Format helpers imported from EvolutionTreeSlideovers

  // Handler for opening turn popup (used by tool slideover)
  const handleTurnClick = useCallback((turn: TreeTurn, isAnomaly: boolean, activity: number) => {
    setToolPopup(null);
    setSelectedFile(null);
    setTurnPopup({
      turn,
      pos: { x: 0, y: 0 },
      isAnomaly,
      activity,
    });
  }, []);

  // Handler for opening tool popup (used by turn slideover)
  const handleToolClickFromTurn = useCallback((tool: TreeTool, e: React.MouseEvent) => {
    if (!turnPopup) return;
    e.stopPropagation();

    const toolType = getToolType(tool.name);
    const toolColor = getToolColor(tool.name);

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
  }, [turnPopup, turns]);

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
      className={`${styles.container} ${colorScheme === 'light' ? styles.lightMode : ''}`}
      onWheel={handleWheel}
      onMouseDown={handleMouseDown}
      onMouseMove={handleMouseMove}
      onMouseUp={handleMouseUp}
      onMouseLeave={handleMouseUp}
      onClick={handleContainerClick}
      style={{ cursor: isPanning ? 'grabbing' : 'grab', background: themeColors.bg }}
    >
      {/* Background ambient glow */}
      <div className={styles.ambientGlow} />

      {/* Session header with zoom controls */}
      <div
        className={styles.sessionHeader}
        style={{
          background: colorScheme === 'light'
            ? 'linear-gradient(180deg, rgba(248, 250, 252, 0.98) 0%, rgba(248, 250, 252, 0.9) 80%, transparent 100%)'
            : 'linear-gradient(180deg, rgba(15, 20, 25, 0.98) 0%, rgba(15, 20, 25, 0.9) 80%, transparent 100%)',
        }}
      >
        <div className={styles.sessionIcon} style={{ color: '#10b981' }}>◉</div>
        <div className={styles.sessionInfo}>
          <span
            className={styles.sessionName}
            style={{ color: colorScheme === 'light' ? '#1f2937' : '#e6edf3' }}
          >
            {session.name}
          </span>
          <span
            className={styles.sessionMeta}
            style={{ color: colorScheme === 'light' ? '#6b7280' : '#8b949e' }}
          >
            {turns.length} turns • {formatDuration(session.durationMs)}
            {session.cost > 0 && ` • ${formatCost(session.cost)}`}
          </span>
        </div>
        {/* Zoom controls */}
        <div
          className={styles.zoomControls}
          style={{
            background: colorScheme === 'light' ? 'rgba(16, 185, 129, 0.08)' : 'rgba(16, 185, 129, 0.15)',
            borderColor: colorScheme === 'light' ? 'rgba(16, 185, 129, 0.25)' : 'rgba(16, 185, 129, 0.3)',
          }}
        >
          <button
            onClick={handleZoomOut}
            title="Zoom out"
            style={{
              background: colorScheme === 'light' ? 'rgba(16, 185, 129, 0.1)' : 'rgba(16, 185, 129, 0.2)',
              color: colorScheme === 'light' ? '#059669' : '#10b981',
            }}
          >
            −
          </button>
          <span
            className={styles.zoomLevel}
            style={{ color: colorScheme === 'light' ? '#059669' : '#10b981' }}
          >
            {Math.round(zoom * 100)}%
          </span>
          <button
            onClick={handleZoomIn}
            title="Zoom in"
            style={{
              background: colorScheme === 'light' ? 'rgba(16, 185, 129, 0.1)' : 'rgba(16, 185, 129, 0.2)',
              color: colorScheme === 'light' ? '#059669' : '#10b981',
            }}
          >
            +
          </button>
          <button
            onClick={handleZoomFit}
            title="Fit to view"
            style={{
              background: colorScheme === 'light' ? 'rgba(16, 185, 129, 0.1)' : 'rgba(16, 185, 129, 0.2)',
              color: colorScheme === 'light' ? '#059669' : '#10b981',
            }}
          >
            ⊡
          </button>
          <button
            onClick={() => setShowControls(!showControls)}
            title="Layout controls"
            style={{
              background: colorScheme === 'light' ? 'rgba(16, 185, 129, 0.1)' : 'rgba(16, 185, 129, 0.2)',
              color: colorScheme === 'light' ? '#059669' : '#10b981',
            }}
          >
            ⚙
          </button>
          <button
            onClick={() => setShowLegend(!showLegend)}
            title="Show legend"
            style={{
              background: colorScheme === 'light' ? 'rgba(16, 185, 129, 0.1)' : 'rgba(16, 185, 129, 0.2)',
              color: colorScheme === 'light' ? '#059669' : '#10b981',
            }}
          >
            ?
          </button>
          {onChatContextClick && (
            <button
              onClick={onChatContextClick}
              title="View chat context"
              style={{
                background: colorScheme === 'light' ? 'rgba(59, 130, 246, 0.1)' : 'rgba(59, 130, 246, 0.2)',
                color: colorScheme === 'light' ? '#2563eb' : '#3b82f6',
              }}
            >
              💬
            </button>
          )}
        </div>
      </div>

      {/* Legend panel */}
      {showLegend && (
        <div
          className={styles.legendPanel}
          style={{
            background: colorScheme === 'light' ? 'rgba(255, 255, 255, 0.95)' : 'rgba(15, 20, 25, 0.95)',
            borderColor: colorScheme === 'light' ? 'rgba(16, 185, 129, 0.25)' : 'rgba(16, 185, 129, 0.3)',
          }}
        >
          <div className={styles.legendTitle} style={{ color: colorScheme === 'light' ? '#059669' : '#10b981' }}>Turns (Outer Spiral)</div>
          <div className={styles.legendItem} style={{ color: colorScheme === 'light' ? '#374151' : '#e6edf3' }}>
            <span className={styles.legendDot} style={{ background: '#10b981' }} />
            <span>Normal turn</span>
          </div>
          <div className={styles.legendItem} style={{ color: colorScheme === 'light' ? '#374151' : '#e6edf3' }}>
            <span className={styles.legendDot} style={{ background: '#f59e0b' }} />
            <span>Anomaly (&gt;2σ)</span>
          </div>
          <div className={styles.legendItem} style={{ color: colorScheme === 'light' ? '#374151' : '#e6edf3' }}>
            <span className={styles.legendDot} style={{ background: '#ef4444' }} />
            <span>Error</span>
          </div>
          <div className={styles.legendDivider} style={{ borderColor: colorScheme === 'light' ? 'rgba(16, 185, 129, 0.2)' : 'rgba(16, 185, 129, 0.2)' }} />
          <div className={styles.legendTitle} style={{ color: colorScheme === 'light' ? '#059669' : '#10b981' }}>Tools (Inner Rings)</div>
          <div className={styles.legendHint} style={{ marginBottom: '6px', color: colorScheme === 'light' ? '#6b7280' : '#8b949e' }}>
            Grouped by type into concentric rings
          </div>
          <div className={styles.legendItem} style={{ color: colorScheme === 'light' ? '#374151' : '#e6edf3' }}>
            <span className={styles.legendDot} style={{ background: '#60a5fa' }} />
            <span>Read/Edit/Write (innermost)</span>
          </div>
          <div className={styles.legendItem} style={{ color: colorScheme === 'light' ? '#374151' : '#e6edf3' }}>
            <span className={styles.legendDot} style={{ background: '#fbbf24' }} />
            <span>Bash/Grep/Glob (middle)</span>
          </div>
          <div className={styles.legendItem} style={{ color: colorScheme === 'light' ? '#374151' : '#e6edf3' }}>
            <span className={styles.legendDot} style={{ background: '#fb923c' }} />
            <span>Task/Web (outer)</span>
          </div>
          <div className={styles.legendDivider} style={{ borderColor: colorScheme === 'light' ? 'rgba(16, 185, 129, 0.2)' : 'rgba(16, 185, 129, 0.2)' }} />
          <div className={styles.legendHint} style={{ color: colorScheme === 'light' ? '#6b7280' : '#8b949e' }}>
            Only tools used 2+ times shown by default
          </div>
          <div className={styles.legendHint} style={{ color: colorScheme === 'light' ? '#6b7280' : '#8b949e' }}>
            Use ⚙ controls to show all tools
          </div>
        </div>
      )}

      {/* Controls panel */}
      {showControls && (
        <div
          className={styles.controlsPanel}
          style={{
            background: colorScheme === 'light' ? 'rgba(255, 255, 255, 0.95)' : 'rgba(15, 20, 25, 0.95)',
            borderColor: colorScheme === 'light' ? 'rgba(16, 185, 129, 0.25)' : 'rgba(16, 185, 129, 0.3)',
          }}
        >
          <div className={styles.legendTitle} style={{ color: colorScheme === 'light' ? '#059669' : '#10b981' }}>Layout Controls</div>

          <div className={styles.controlItem} style={{ color: colorScheme === 'light' ? '#374151' : '#e6edf3' }}>
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

          <div className={styles.controlItem} style={{ color: colorScheme === 'light' ? '#374151' : '#e6edf3' }}>
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

          <div className={styles.controlItem} style={{ color: colorScheme === 'light' ? '#374151' : '#e6edf3' }}>
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

          <div className={styles.legendDivider} style={{ borderColor: colorScheme === 'light' ? 'rgba(16, 185, 129, 0.2)' : 'rgba(16, 185, 129, 0.2)' }} />

          <div className={styles.controlItem} style={{ color: colorScheme === 'light' ? '#374151' : '#e6edf3' }}>
            <label style={{ display: 'flex', alignItems: 'center', gap: '8px', cursor: 'pointer' }}>
              <input
                type="checkbox"
                checked={showAllTools}
                onChange={(e) => setShowAllTools(e.target.checked)}
                style={{ cursor: 'pointer' }}
              />
              Show all tools ({totalUniqueTools})
            </label>
            <span style={{ fontSize: '9px', color: colorScheme === 'light' ? '#6b7280' : '#8b949e' }}>
              {showAllTools ? 'Showing all' : `Showing ${sharedToolNodes.length} (used 2+ times)`}
            </span>
          </div>

          <div className={styles.legendDivider} style={{ borderColor: colorScheme === 'light' ? 'rgba(16, 185, 129, 0.2)' : 'rgba(16, 185, 129, 0.2)' }} />
          <div className={styles.legendHint} style={{ color: colorScheme === 'light' ? '#6b7280' : '#8b949e' }}>
            Turns/rotation: {turnsPerRotation.toFixed(1)}
          </div>
          <div className={styles.legendHint} style={{ color: colorScheme === 'light' ? '#6b7280' : '#8b949e' }}>
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
          <div
            className={styles.nowPlayingBanner}
            style={{
              background: colorScheme === 'light'
                ? 'linear-gradient(135deg, rgba(255, 255, 255, 0.98) 0%, rgba(248, 250, 252, 0.95) 100%)'
                : 'linear-gradient(135deg, rgba(15, 20, 25, 0.98) 0%, rgba(10, 15, 20, 0.95) 100%)',
              borderColor: colorScheme === 'light' ? 'rgba(16, 185, 129, 0.4)' : 'rgba(16, 185, 129, 0.4)',
            }}
          >
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
                      <span className={styles.nowPlayingToolName} style={{ color: colorScheme === 'light' ? '#1f2937' : '#e6edf3' }}>
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
        <div
          className={styles.playbackControls}
          style={{
            background: colorScheme === 'light' ? 'rgba(255, 255, 255, 0.95)' : 'rgba(15, 20, 25, 0.95)',
            borderColor: colorScheme === 'light' ? 'rgba(16, 185, 129, 0.35)' : 'rgba(16, 185, 129, 0.3)',
          }}
        >
          {/* Play/Pause */}
          <button
            className={styles.playbackButton}
            onClick={togglePlayback}
            title={playback.isPlaying ? 'Pause' : 'Play'}
            style={{
              background: colorScheme === 'light' ? 'rgba(16, 185, 129, 0.12)' : 'rgba(16, 185, 129, 0.15)',
              color: colorScheme === 'light' ? '#059669' : '#10b981',
            }}
          >
            {playback.isPlaying ? '⏸' : '▶'}
          </button>

          {/* Skip to start */}
          <button
            className={styles.playbackButton}
            onClick={() => seekTo(0)}
            title="Skip to start"
            style={{
              background: colorScheme === 'light' ? 'rgba(16, 185, 129, 0.12)' : 'rgba(16, 185, 129, 0.15)',
              color: colorScheme === 'light' ? '#059669' : '#10b981',
            }}
          >
            ⏮
          </button>

          {/* Skip to end */}
          <button
            className={styles.playbackButton}
            onClick={() => seekTo(totalDurationMs)}
            title="Skip to end"
            style={{
              background: colorScheme === 'light' ? 'rgba(16, 185, 129, 0.12)' : 'rgba(16, 185, 129, 0.15)',
              color: colorScheme === 'light' ? '#059669' : '#10b981',
            }}
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
                style={playback.speed !== s ? {
                  background: colorScheme === 'light' ? 'rgba(16, 185, 129, 0.08)' : 'rgba(16, 185, 129, 0.1)',
                  color: colorScheme === 'light' ? '#6b7280' : '#8b949e',
                } : undefined}
              >
                {s}x
              </button>
            ))}
          </div>

          {/* Decay control - how many turns until connections fade */}
          <div className={styles.decayControl} title={`Highlight decay: ${highlightDecay} turns (0=off)`}>
            <span className={styles.decayLabel} style={{ color: colorScheme === 'light' ? '#6b7280' : '#6e7681' }}>Decay</span>
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
            style={{ color: colorScheme === 'light' ? '#6b7280' : '#8b949e' }}
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
        <ToolSlideover
          toolPopup={toolPopup}
          turns={turns}
          spiralPositions={spiralPositions}
          themeColors={themeColors as ThemeColors}
          usageExpanded={usageExpanded}
          onClose={() => setToolPopup(null)}
          onUsageExpandedChange={setUsageExpanded}
          onTurnClick={handleTurnClick}
          onCenterOnPosition={centerOnPosition}
        />
      )}

      {/* Turn Detail Slideover Panel */}
      {turnPopup && (
        <TurnSlideover
          turnPopup={turnPopup}
          turns={turns}
          spiralPositions={spiralPositions}
          fileDirectories={fileDirectories}
          themeColors={themeColors as ThemeColors}
          turnToolsExpanded={turnToolsExpanded}
          onClose={() => setTurnPopup(null)}
          onTurnToolsExpandedChange={setTurnToolsExpanded}
          onToolClick={handleToolClickFromTurn}
        />
      )}

      {/* File Hover Tooltip */}
      {hoveredFile && !selectedFile && (
        <FileTooltip hoveredFile={hoveredFile} />
      )}

      {/* Directory Hover Tooltip */}
      {hoveredDir && (
        <DirTooltip hoveredDir={hoveredDir} expandedDirs={expandedDirs} />
      )}

      {/* File Detail Slideover Panel */}
      {selectedFile && (
        <FileSlideover
          selectedFile={selectedFile}
          turns={turns}
          spiralPositions={spiralPositions}
          themeColors={themeColors as ThemeColors}
          onClose={() => setSelectedFile(null)}
          onTurnClick={(turn, isAnomaly, activity) => {
            setSelectedFile(null);
            handleTurnClick(turn, isAnomaly, activity);
          }}
          onCenterOnPosition={centerOnPosition}
        />
      )}
    </div>
  );
};

export default EvolutionTree;
