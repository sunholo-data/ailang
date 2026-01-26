/**
 * EvolutionTree Slideover Panels
 * Extracted from EvolutionTree.tsx (PR 5 - M-DASHBOARD-SIMPLIFICATION)
 *
 * Contains:
 * - Tool detail slideover panel
 * - Turn detail slideover panel
 * - File detail slideover panel
 * - File/Directory hover tooltips
 */
import React from 'react';
import type {
  TreeTurn,
  TreeTool,
  SharedToolNode,
  FileNode,
  DirectoryNode,
  SpiralPosition,
} from '../../utils/evolutionTreeBuilders';
import { getToolType, getToolColor, getToolDisplayName } from '../../utils/evolutionTreeUtils';
import styles from './EvolutionTree.module.css';

// ============================================================================
// Types
// ============================================================================

export interface ThemeColors {
  bg: string;
  bgSecondary: string;
  text: string;
  textMuted: string;
  textSubtle: string;
  emerald: string;
  emeraldLight: string;
  cyan: string;
  error: string;
  nodeText: string;
  nodeTextOnFill: string;
  stemStroke: string;
  stemGlow: string;
  fileCyan: string;
  fileRead: string;
  fileEdit: string;
  fileWrite: string;
}

export interface ToolPopupState {
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
}

export interface TurnPopupState {
  turn: TreeTurn;
  pos: { x: number; y: number };
  isAnomaly: boolean;
  activity: number;
}

export interface FileHoverState {
  file: FileNode;
  pos: { x: number; y: number };
}

export interface DirHoverState {
  dir: DirectoryNode;
  pos: { x: number; y: number };
}

// ============================================================================
// Formatting Helpers
// ============================================================================

export function formatDuration(ms: number): string {
  if (ms < 1000) return `${ms}ms`;
  if (ms < 60000) return `${(ms / 1000).toFixed(1)}s`;
  return `${(ms / 60000).toFixed(1)}m`;
}

export function formatCost(cost: number): string {
  if (cost === 0) return '';
  if (cost < 0.01) return `$${cost.toFixed(4)}`;
  return `$${cost.toFixed(3)}`;
}

// ============================================================================
// Tool Slideover Panel
// ============================================================================

interface ToolSlideoverProps {
  toolPopup: ToolPopupState;
  turns: TreeTurn[];
  spiralPositions: SpiralPosition[];
  themeColors: ThemeColors;
  usageExpanded: boolean;
  onClose: () => void;
  onUsageExpandedChange: (expanded: boolean) => void;
  onTurnClick: (turn: TreeTurn, isAnomaly: boolean, activity: number) => void;
  onCenterOnPosition: (x: number, y: number) => void;
}

export const ToolSlideover: React.FC<ToolSlideoverProps> = ({
  toolPopup,
  turns,
  spiralPositions,
  themeColors,
  usageExpanded,
  onClose,
  onUsageExpandedChange,
  onTurnClick,
  onCenterOnPosition,
}) => {
  return (
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
          onClick={onClose}
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
            onClick={() => onUsageExpandedChange(!usageExpanded)}
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
                        const pos = spiralPositions.find(p => p.turn.id === turn.id);
                        onTurnClick(turn, pos?.isAnomaly || false, pos?.activity || 0.5);
                        if (pos) {
                          onCenterOnPosition(pos.x, pos.y);
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

        {/* Mini subgraph - tool to turns */}
        <div className={styles.miniSubgraph}>
          <div className={styles.miniSubgraphTitle}>Connections</div>
          <svg width="100%" height="120" viewBox="0 0 340 120">
            <defs>
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

            {/* Tool node in center */}
            <circle
              cx={170} cy={60} r={16}
              fill={toolPopup.node.color}
              filter="url(#miniGlow)"
            />
            <text x={170} y={64} textAnchor="middle" fontSize="10" fill={themeColors.nodeTextOnFill} fontWeight="600">
              {toolPopup.node.toolType.slice(0, 4)}
            </text>

            {/* Connected turns */}
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
      </div>
    </div>
  );
};

// ============================================================================
// Turn Slideover Panel
// ============================================================================

interface TurnSlideoverProps {
  turnPopup: TurnPopupState;
  turns: TreeTurn[];
  spiralPositions: SpiralPosition[];
  fileDirectories: DirectoryNode[];
  themeColors: ThemeColors;
  turnToolsExpanded: boolean;
  onClose: () => void;
  onTurnToolsExpandedChange: (expanded: boolean) => void;
  onToolClick: (tool: TreeTool, event: React.MouseEvent) => void;
}

export const TurnSlideover: React.FC<TurnSlideoverProps> = ({
  turnPopup,
  turns,
  fileDirectories,
  themeColors,
  turnToolsExpanded,
  onClose,
  onTurnToolsExpandedChange,
  onToolClick,
}) => {
  return (
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
          onClick={onClose}
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
              onClick={() => onTurnToolsExpandedChange(!turnToolsExpanded)}
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
                      onClick={(e) => onToolClick(tool, e)}
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

        {/* Mini subgraph - turn to tools and files */}
        <div className={styles.miniSubgraph}>
          <div className={styles.miniSubgraphTitle}>Connections</div>
          <svg width="100%" height="140" viewBox="0 0 340 140">
            <defs>
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

            {/* Edges to tools */}
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

            {/* Turn node in center */}
            <circle
              cx={170} cy={70} r={18}
              fill={turnPopup.isAnomaly ? '#f59e0b' : turnPopup.turn.status === 'error' ? '#ef4444' : themeColors.emerald}
              filter={turnPopup.isAnomaly ? 'url(#turnMiniAnomalyGlow)' : turnPopup.turn.status === 'error' ? 'url(#turnMiniErrorGlow)' : 'url(#turnMiniGlow)'}
            />
            <text x={170} y={74} textAnchor="middle" fontSize="11" fill={themeColors.nodeTextOnFill} fontWeight="700">
              T{turnPopup.turn.turnNumber}
            </text>

            {/* Tool nodes */}
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

            {/* File nodes */}
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
      </div>
    </div>
  );
};

// ============================================================================
// File Slideover Panel
// ============================================================================

interface FileSlideoverProps {
  selectedFile: FileNode;
  turns: TreeTurn[];
  spiralPositions: SpiralPosition[];
  themeColors: ThemeColors;
  onClose: () => void;
  onTurnClick: (turn: TreeTurn, isAnomaly: boolean, activity: number) => void;
  onCenterOnPosition: (x: number, y: number) => void;
}

export const FileSlideover: React.FC<FileSlideoverProps> = ({
  selectedFile,
  turns,
  spiralPositions,
  themeColors,
  onClose,
  onTurnClick,
  onCenterOnPosition,
}) => {
  const fileTypeColor = {
    go: themeColors.cyan,
    tsx: '#14b8a6',
    ts: '#0d9488',
    js: '#34d399',
    ail: themeColors.emerald,
    md: '#6ee7b7',
    json: '#2dd4bf',
    yaml: '#5eead4',
    css: '#22d3ee',
  }[selectedFile.fileType] || themeColors.emerald;

  return (
    <div
      className={styles.detailSlideover}
      onClick={(e) => e.stopPropagation()}
      onWheel={(e) => e.stopPropagation()}
    >
      {/* Header */}
      <div className={styles.slideoverHeader}>
        <div
          className={styles.slideoverIcon}
          style={{ backgroundColor: fileTypeColor }}
        >
          📄
        </div>
        <div className={styles.slideoverTitle}>
          <span className={styles.slideoverType}>FILE</span>
          <span className={styles.slideoverName}>{selectedFile.fileName}</span>
        </div>
        <button
          className={styles.slideoverClose}
          onClick={onClose}
          aria-label="Close"
        >
          ×
        </button>
      </div>

      {/* Body content */}
      <div className={styles.slideoverBody}>
        {/* Full path */}
        <div className={styles.filePath}>{selectedFile.filePath}</div>

        {/* Operation Summary */}
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
          <button className={styles.bioTimelineToggle}>
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
                    const turn = turns.find(t => t.id === op.turnId);
                    if (turn) {
                      const pos = spiralPositions.find(p => p.turn.id === op.turnId);
                      onTurnClick(turn, pos?.isAnomaly || false, pos?.activity || 0.5);
                      if (pos) {
                        onCenterOnPosition(pos.x, pos.y);
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

        {/* Mini subgraph - file to turns */}
        <div className={styles.miniSubgraph}>
          <div className={styles.miniSubgraphTitle}>Connected Turns</div>
          <svg width="100%" height="120" viewBox="0 0 340 120">
            <defs>
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

            {/* Edges */}
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

            {/* File node in center */}
            <circle
              cx={170} cy={60} r={16}
              fill={fileTypeColor}
              filter="url(#fileMiniGlow)"
            />
            <text x={170} y={64} textAnchor="middle" fontSize="11" fill={themeColors.nodeTextOnFill}>
              📄
            </text>

            {/* Connected turns */}
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
  );
};

// ============================================================================
// Hover Tooltips
// ============================================================================

interface FileTooltipProps {
  hoveredFile: FileHoverState;
}

export const FileTooltip: React.FC<FileTooltipProps> = ({ hoveredFile }) => {
  return (
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
  );
};

interface DirTooltipProps {
  hoveredDir: DirHoverState;
  expandedDirs: Set<string>;
}

export const DirTooltip: React.FC<DirTooltipProps> = ({ hoveredDir, expandedDirs }) => {
  return (
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
  );
};
