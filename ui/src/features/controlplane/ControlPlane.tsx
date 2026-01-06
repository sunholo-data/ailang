/**
 * AILANG Control Plane v4
 * AI Operations Mission Control Interface
 *
 * Aesthetic: Aerospace Mission Control meets Bloomberg Terminal
 */
import React, { useState, useEffect, useCallback, useMemo, useRef } from 'react';
import ReactFlow, {
  Node,
  Edge,
  Background,
  Controls,
  MiniMap,
  useNodesState,
  useEdgesState,
  MarkerType,
  Handle,
  Position,
  NodeProps,
  ReactFlowInstance,
} from 'reactflow';
import 'reactflow/dist/style.css';
import styles from './ControlPlane.module.css';
import {
  useHeatmapData,
  useTopologyData,
  useStatistics,
  useEventQueue,
  useTraceData,
} from './hooks';

// Types
interface Agent {
  id: string;
  label: string;
  status: 'idle' | 'busy' | 'blocked' | 'error';
  trustScore: number;
  taskCount: number;
  cost: number;
}

interface EventMessage {
  id: string;
  timestamp: string;
  type: 'task_start' | 'task_complete' | 'task_error' | 'handoff' | 'approval' | 'message';
  source: string;
  target?: string;
  content: string;
  metadata?: Record<string, unknown>;
}

interface DateRange {
  start: string;
  end: string;
}

interface DetailPanelState {
  type: 'agent' | 'trace' | 'event' | 'date' | null;
  id: string | null;
  data?: unknown;
}

interface HeatmapCell {
  date: string;
  taskCount: number;
  cost: number;
  successRate: number;
}

interface Span {
  id: string;
  name: string;
  startMs: number;
  durationMs: number;
  children?: Span[];
}

interface TopologyEdge {
  source: string;
  target: string;
  messageCount: number;
  active: boolean;
}

interface TrustCapability {
  name: string;
  score: number;
  icon: string;
}

// Mock data generators
const generateHeatmapData = (): HeatmapCell[] => {
  const cells: HeatmapCell[] = [];
  const now = new Date();
  for (let i = 89; i >= 0; i--) {
    const date = new Date(now);
    date.setDate(date.getDate() - i);
    const dayOfWeek = date.getDay();
    const isWeekend = dayOfWeek === 0 || dayOfWeek === 6;
    const baseActivity = isWeekend ? Math.random() * 20 : Math.random() * 100 + 20;
    cells.push({
      date: date.toISOString().split('T')[0],
      taskCount: Math.floor(baseActivity),
      cost: baseActivity * 0.003,
      successRate: 0.85 + Math.random() * 0.15,
    });
  }
  return cells;
};

const mockAgents: Agent[] = [
  { id: 'design-doc-creator', label: 'Doc Creator', status: 'idle', trustScore: 80, taskCount: 47, cost: 1.23 },
  { id: 'sprint-planner', label: 'Planner', status: 'busy', trustScore: 65, taskCount: 31, cost: 0.87 },
  { id: 'sprint-executor', label: 'Executor', status: 'idle', trustScore: 45, taskCount: 89, cost: 2.45 },
  { id: 'eval-analyzer', label: 'Analyzer', status: 'blocked', trustScore: 72, taskCount: 23, cost: 0.56 },
];

const mockEdges: TopologyEdge[] = [
  { source: 'github', target: 'design-doc-creator', messageCount: 12, active: false },
  { source: 'design-doc-creator', target: 'sprint-planner', messageCount: 31, active: true },
  { source: 'sprint-planner', target: 'sprint-executor', messageCount: 28, active: false },
  { source: 'sprint-executor', target: 'approval', messageCount: 89, active: false },
];

const mockSpans: Span[] = [
  {
    id: '1', name: 'coordinator.analyze', startMs: 0, durationMs: 2300,
    children: [
      { id: '1.1', name: 'task.classify', startMs: 100, durationMs: 800 },
      { id: '1.2', name: 'task.route', startMs: 1000, durationMs: 1200 },
    ]
  },
  { id: '2', name: 'executor.init', startMs: 2400, durationMs: 800 },
  {
    id: '3', name: 'executor.claude.execute', startMs: 3300, durationMs: 142000,
    children: [
      { id: '3.1', name: 'claude.read_files', startMs: 3400, durationMs: 3200 },
      {
        id: '3.2', name: 'claude.generate', startMs: 6700, durationMs: 98000,
        children: [
          { id: '3.2.1', name: 'anthropic.api', startMs: 6800, durationMs: 96000 },
        ]
      },
      { id: '3.3', name: 'claude.write_files', startMs: 105000, durationMs: 4100 },
    ]
  },
  { id: '4', name: 'coordinator.validate', startMs: 145500, durationMs: 3800 },
  { id: '5', name: 'coordinator.finalize', startMs: 149500, durationMs: 2100 },
];

const mockTrustCapabilities: TrustCapability[] = [
  { name: 'Read Files', score: 95, icon: '◉' },
  { name: 'Write Docs', score: 80, icon: '◎' },
  { name: 'Write Code', score: 45, icon: '⬡' },
  { name: 'Run Tests', score: 70, icon: '▣' },
  { name: 'Git Commit', score: 60, icon: '◈' },
  { name: 'Git Push', score: 30, icon: '◇' },
  { name: 'Release', score: 0, icon: '⬢' },
];

const mockEvents: EventMessage[] = [
  { id: 'e1', timestamp: new Date(Date.now() - 120000).toISOString(), type: 'task_start', source: 'coordinator', target: 'design-doc-creator', content: 'Starting task: Create design doc for Control Plane v4' },
  { id: 'e2', timestamp: new Date(Date.now() - 90000).toISOString(), type: 'message', source: 'design-doc-creator', content: 'Reading existing design docs...' },
  { id: 'e3', timestamp: new Date(Date.now() - 60000).toISOString(), type: 'handoff', source: 'design-doc-creator', target: 'sprint-planner', content: 'Design doc complete, handing off to sprint-planner' },
  { id: 'e4', timestamp: new Date(Date.now() - 45000).toISOString(), type: 'task_start', source: 'sprint-planner', content: 'Analyzing design doc, calculating velocity...' },
  { id: 'e5', timestamp: new Date(Date.now() - 30000).toISOString(), type: 'approval', source: 'sprint-planner', content: 'Sprint plan ready for approval: M-CONTROL-PLANE-V4' },
  { id: 'e6', timestamp: new Date(Date.now() - 15000).toISOString(), type: 'task_complete', source: 'sprint-executor', content: 'Milestone M1 complete: Component scaffolding' },
  { id: 'e7', timestamp: new Date(Date.now() - 5000).toISOString(), type: 'message', source: 'sprint-executor', content: 'Starting milestone M2: Heatmap implementation' },
];

// Utility functions
const getTrustLevel = (score: number): string => {
  if (score >= 85) return 'auto';
  if (score >= 60) return 'low-risk';
  if (score >= 25) return 'review';
  return 'manual';
};

const getStatusIcon = (status: Agent['status']): string => {
  switch (status) {
    case 'idle': return '○';
    case 'busy': return '●';
    case 'blocked': return '◐';
    case 'error': return '✕';
  }
};

const formatDuration = (ms: number): string => {
  if (ms < 1000) return `${ms}ms`;
  if (ms < 60000) return `${(ms / 1000).toFixed(1)}s`;
  return `${Math.floor(ms / 60000)}m ${Math.floor((ms % 60000) / 1000)}s`;
};

// Components
const CommandBar: React.FC<{
  searchQuery: string;
  onSearchChange: (q: string) => void;
  timeRange: string;
  onTimeRangeChange: (r: string) => void;
}> = ({ searchQuery, onSearchChange, timeRange, onTimeRangeChange }) => (
  <div className={styles.commandBar}>
    <div className={styles.searchContainer}>
      <span className={styles.searchIcon}>⌘</span>
      <input
        type="text"
        className={styles.searchInput}
        placeholder="Search traces, messages, tasks..."
        value={searchQuery}
        onChange={(e) => onSearchChange(e.target.value)}
      />
      <kbd className={styles.searchKbd}>K</kbd>
    </div>
    <div className={styles.commandActions}>
      <select
        className={styles.timeSelect}
        value={timeRange}
        onChange={(e) => onTimeRangeChange(e.target.value)}
      >
        <option value="1h">Last 1 hour</option>
        <option value="24h">Last 24 hours</option>
        <option value="7d">Last 7 days</option>
        <option value="30d">Last 30 days</option>
        <option value="90d">Last 90 days</option>
      </select>
      <div className={styles.filterChips}>
        <button className={`${styles.chip} ${styles.chipActive}`}>All</button>
        <button className={styles.chip}>Running</button>
        <button className={styles.chip}>Pending</button>
        <button className={styles.chip}>Failed</button>
      </div>
      <div className={styles.liveIndicator}>
        <span className={styles.liveDot} />
        <span>LIVE</span>
      </div>
    </div>
  </div>
);

const AggregationNav: React.FC<{
  selectedLevel: string;
  onSelectLevel: (level: string) => void;
}> = ({ selectedLevel, onSelectLevel }) => {
  const [expanded, setExpanded] = useState<Set<string>>(new Set(['global', 'workspaces']));

  const toggleExpand = (id: string) => {
    setExpanded(prev => {
      const next = new Set(prev);
      if (next.has(id)) next.delete(id);
      else next.add(id);
      return next;
    });
  };

  const NavItem: React.FC<{
    id: string;
    label: string;
    icon: string;
    depth: number;
    count?: number;
    cost?: number;
    children?: React.ReactNode;
  }> = ({ id, label, icon, depth, count, cost, children }) => {
    const isExpanded = expanded.has(id);
    const isSelected = selectedLevel === id;
    const hasChildren = React.Children.count(children) > 0;

    return (
      <div className={styles.navGroup}>
        <button
          className={`${styles.navItem} ${isSelected ? styles.navItemSelected : ''}`}
          style={{ paddingLeft: `${12 + depth * 16}px` }}
          onClick={() => {
            onSelectLevel(id);
            if (hasChildren) toggleExpand(id);
          }}
        >
          {hasChildren && (
            <span className={`${styles.navChevron} ${isExpanded ? styles.navChevronOpen : ''}`}>
              ▸
            </span>
          )}
          <span className={styles.navIcon}>{icon}</span>
          <span className={styles.navLabel}>{label}</span>
          {count !== undefined && (
            <span className={styles.navCount}>{count}</span>
          )}
          {cost !== undefined && (
            <span className={styles.navCost}>${cost.toFixed(2)}</span>
          )}
        </button>
        {hasChildren && isExpanded && (
          <div className={styles.navChildren}>{children}</div>
        )}
      </div>
    );
  };

  return (
    <nav className={styles.aggregationNav}>
      <div className={styles.navHeader}>
        <span className={styles.navTitle}>AGGREGATIONS</span>
      </div>
      <div className={styles.navTree}>
        <NavItem id="global" label="Global" icon="◎" depth={0} count={167} cost={4.20}>
          <NavItem id="workspaces" label="Workspaces" icon="▤" depth={1} count={3}>
            <NavItem id="ws-ailang" label="ailang" icon="▢" depth={2} count={89} cost={2.45} />
            <NavItem id="ws-stapledon" label="stapledon" icon="▢" depth={2} count={54} cost={1.12} />
            <NavItem id="ws-other" label="other" icon="▢" depth={2} count={24} cost={0.63} />
          </NavItem>
          <NavItem id="providers" label="Providers" icon="◇" depth={1}>
            <NavItem id="prov-claude" label="claude" icon="⬡" depth={2} count={142} cost={3.87} />
            <NavItem id="prov-gemini" label="gemini" icon="⬢" depth={2} count={25} cost={0.33} />
          </NavItem>
          <NavItem id="models" label="Models" icon="◈" depth={1}>
            <NavItem id="model-opus" label="opus-4.5" icon="●" depth={2} count={12} cost={1.24} />
            <NavItem id="model-sonnet" label="sonnet-4" icon="◐" depth={2} count={89} cost={2.13} />
            <NavItem id="model-haiku" label="haiku-4" icon="○" depth={2} count={41} cost={0.50} />
          </NavItem>
          <NavItem id="trust" label="Trust Level" icon="◉" depth={1}>
            <NavItem id="trust-auto" label="Full Auto" icon="●" depth={2} count={34} />
            <NavItem id="trust-low" label="Low-Risk" icon="◐" depth={2} count={67} />
            <NavItem id="trust-review" label="Review" icon="◔" depth={2} count={45} />
            <NavItem id="trust-manual" label="Manual" icon="○" depth={2} count={21} />
          </NavItem>
        </NavItem>
      </div>
      <div className={styles.navFooter}>
        <div className={styles.navStat}>
          <span className={styles.navStatLabel}>Active Agents</span>
          <span className={styles.navStatValue}>4</span>
        </div>
        <div className={styles.navStat}>
          <span className={styles.navStatLabel}>Pending Approvals</span>
          <span className={`${styles.navStatValue} ${styles.navStatWarning}`}>12</span>
        </div>
      </div>
    </nav>
  );
};

type HeatmapRange = '3m' | '6m' | '1y';

const ActivityHeatmap: React.FC<{
  data: HeatmapCell[];
  selectedRange: DateRange | null;
  onDateSelect: (range: DateRange) => void;
  onCellClick: (cell: HeatmapCell) => void;
}> = ({ data, selectedRange, onDateSelect, onCellClick }) => {
  const [hoveredCell, setHoveredCell] = useState<HeatmapCell | null>(null);
  const [tooltipPos, setTooltipPos] = useState({ x: 0, y: 0 });
  const [selectionStart, setSelectionStart] = useState<string | null>(null);
  const [isDragging, setIsDragging] = useState(false);
  const [displayRange, setDisplayRange] = useState<HeatmapRange>('3m');
  const gridRef = useRef<HTMLDivElement>(null);

  const getIntensity = (count: number): number => {
    if (count === 0) return 0;
    if (count < 10) return 1;
    if (count < 30) return 2;
    if (count < 60) return 3;
    return 4;
  };

  // Filter data based on display range
  const filteredData = useMemo(() => {
    const now = new Date();
    let daysToShow: number;
    switch (displayRange) {
      case '3m': daysToShow = 90; break;
      case '6m': daysToShow = 180; break;
      case '1y': daysToShow = 365; break;
    }
    const cutoffDate = new Date(now.getTime() - daysToShow * 24 * 60 * 60 * 1000);
    const cutoffStr = cutoffDate.toISOString().split('T')[0];
    return data.filter(cell => cell.date >= cutoffStr);
  }, [data, displayRange]);

  const weeks = useMemo(() => {
    const result: HeatmapCell[][] = [];
    let currentWeek: HeatmapCell[] = [];

    if (filteredData.length === 0) return result;

    // Pad start to align with day of week (Monday = 0)
    const firstDate = new Date(filteredData[0].date);
    const dayOfWeek = firstDate.getDay();
    const startPadding = dayOfWeek === 0 ? 6 : dayOfWeek - 1;
    for (let i = 0; i < startPadding; i++) {
      currentWeek.push({ date: '', taskCount: -1, cost: 0, successRate: 0 });
    }

    filteredData.forEach((cell) => {
      currentWeek.push(cell);
      if (currentWeek.length === 7) {
        result.push(currentWeek);
        currentWeek = [];
      }
    });
    if (currentWeek.length > 0) {
      while (currentWeek.length < 7) {
        currentWeek.push({ date: '', taskCount: -1, cost: 0, successRate: 0 });
      }
      result.push(currentWeek);
    }
    return result;
  }, [filteredData]);

  // Get month labels with their positions
  const monthLabels = useMemo(() => {
    const labels: { label: string; weekIdx: number; span: number }[] = [];
    let currentMonth = -1;
    let currentLabel: { label: string; weekIdx: number; span: number } | null = null;

    weeks.forEach((week, weekIdx) => {
      const validCell = week.find(c => c.date);
      if (validCell) {
        const date = new Date(validCell.date);
        const month = date.getMonth();
        if (month !== currentMonth) {
          if (currentLabel) {
            labels.push(currentLabel);
          }
          currentLabel = {
            label: date.toLocaleDateString('en-US', { month: 'short' }),
            weekIdx,
            span: 1,
          };
          currentMonth = month;
        } else if (currentLabel) {
          currentLabel.span++;
        }
      }
    });
    if (currentLabel) {
      labels.push(currentLabel);
    }
    return labels;
  }, [weeks]);

  const handleCellHover = (cell: HeatmapCell, e: React.MouseEvent) => {
    if (cell.taskCount >= 0) {
      setHoveredCell(cell);
      setTooltipPos({ x: e.clientX, y: e.clientY });

      if (isDragging && selectionStart) {
        const startIdx = filteredData.findIndex(c => c.date === selectionStart);
        const endIdx = filteredData.findIndex(c => c.date === cell.date);
        const [start, end] = startIdx < endIdx
          ? [selectionStart, cell.date]
          : [cell.date, selectionStart];
        onDateSelect({ start, end });
      }
    }
  };

  const handleCellMouseDown = (cell: HeatmapCell) => {
    if (cell.taskCount < 0) return;
    setIsDragging(true);
    setSelectionStart(cell.date);
    onDateSelect({ start: cell.date, end: cell.date });
  };

  const handleCellMouseUp = (cell: HeatmapCell) => {
    if (cell.taskCount >= 0) {
      setIsDragging(false);
      onCellClick(cell);
    }
  };

  const handleMouseUp = () => {
    setIsDragging(false);
  };

  useEffect(() => {
    window.addEventListener('mouseup', handleMouseUp);
    return () => window.removeEventListener('mouseup', handleMouseUp);
  }, []);

  // Scroll to end (latest dates) on mount and when range changes
  useEffect(() => {
    if (gridRef.current) {
      gridRef.current.scrollLeft = gridRef.current.scrollWidth;
    }
  }, [displayRange, weeks.length]);

  const isInRange = (date: string): boolean => {
    if (!selectedRange || !selectedRange.start || !selectedRange.end) return false;
    return date >= selectedRange.start && date <= selectedRange.end;
  };

  const rangeStats = useMemo(() => {
    if (!selectedRange || !selectedRange.start) return null;
    const cells = filteredData.filter(c => c.date >= selectedRange.start && c.date <= selectedRange.end);
    if (cells.length === 0) return null;

    const totalTasks = cells.reduce((sum, c) => sum + c.taskCount, 0);
    const totalCost = cells.reduce((sum, c) => sum + c.cost, 0);
    const avgSuccess = cells.reduce((sum, c) => sum + c.successRate, 0) / cells.length;

    return { totalTasks, totalCost, avgSuccess, days: cells.length };
  }, [filteredData, selectedRange]);

  // Calculate totals for display
  const totals = useMemo(() => {
    const totalTasks = filteredData.reduce((sum, c) => sum + c.taskCount, 0);
    const totalCost = filteredData.reduce((sum, c) => sum + c.cost, 0);
    return { totalTasks, totalCost };
  }, [filteredData]);

  return (
    <div className={styles.heatmapContainer}>
      <div className={styles.heatmapHeader}>
        <h3 className={styles.panelTitle}>
          <span className={styles.panelIcon}>▤</span>
          Activity
        </h3>
        <div className={styles.heatmapTotals}>
          <span className={styles.heatmapTotal}>{totals.totalTasks} tasks</span>
          <span className={styles.heatmapTotal}>${totals.totalCost.toFixed(2)}</span>
        </div>
        <div className={styles.heatmapRangeSelector}>
          <button
            className={`${styles.rangeBtn} ${displayRange === '3m' ? styles.rangeBtnActive : ''}`}
            onClick={() => setDisplayRange('3m')}
          >
            3M
          </button>
          <button
            className={`${styles.rangeBtn} ${displayRange === '6m' ? styles.rangeBtnActive : ''}`}
            onClick={() => setDisplayRange('6m')}
          >
            6M
          </button>
          <button
            className={`${styles.rangeBtn} ${displayRange === '1y' ? styles.rangeBtnActive : ''}`}
            onClick={() => setDisplayRange('1y')}
          >
            1Y
          </button>
        </div>
        <div className={styles.heatmapLegend}>
          <span className={styles.legendLabel}>Less</span>
          <span className={`${styles.legendCell} ${styles.intensity0}`} />
          <span className={`${styles.legendCell} ${styles.intensity1}`} />
          <span className={`${styles.legendCell} ${styles.intensity2}`} />
          <span className={`${styles.legendCell} ${styles.intensity3}`} />
          <span className={`${styles.legendCell} ${styles.intensity4}`} />
          <span className={styles.legendLabel}>More</span>
        </div>
      </div>

      <div className={styles.heatmapBody}>
        <div className={styles.heatmapDayLabels}>
          <span></span>
          <span>Mon</span>
          <span></span>
          <span>Wed</span>
          <span></span>
          <span>Fri</span>
          <span></span>
        </div>

        <div className={styles.heatmapScrollArea} ref={gridRef}>
          <div className={styles.heatmapMonthRow}>
            {monthLabels.map((m, i) => (
              <span
                key={i}
                className={styles.monthLabel}
                style={{
                  gridColumn: `${m.weekIdx + 1} / span ${m.span}`,
                }}
              >
                {m.label}
              </span>
            ))}
          </div>

          <div className={styles.heatmapGrid} style={{ gridTemplateColumns: `repeat(${weeks.length}, 14px)` }}>
            {weeks.map((week, weekIdx) => (
              <div key={weekIdx} className={styles.heatmapWeek}>
                {week.map((cell, dayIdx) => (
                  <div
                    key={dayIdx}
                    className={`${styles.heatmapCell} ${
                      cell.taskCount < 0 ? styles.cellEmpty : styles[`intensity${getIntensity(cell.taskCount)}`]
                    } ${isInRange(cell.date) ? styles.cellSelected : ''}`}
                    onMouseEnter={(e) => handleCellHover(cell, e)}
                    onMouseLeave={() => !isDragging && setHoveredCell(null)}
                    onMouseDown={() => handleCellMouseDown(cell)}
                    onMouseUp={() => handleCellMouseUp(cell)}
                  />
                ))}
              </div>
            ))}
          </div>
        </div>
      </div>

      {selectedRange && selectedRange.start && rangeStats && (
        <div className={styles.heatmapSelection}>
          <div className={styles.selectionInfo}>
            <span className={styles.selectionDates}>
              {selectedRange.start === selectedRange.end
                ? selectedRange.start
                : `${selectedRange.start} → ${selectedRange.end}`}
            </span>
            <span className={styles.selectionDays}>({rangeStats.days} days)</span>
          </div>
          <div className={styles.selectionStats}>
            <span className={styles.selectionStat}>
              <strong>{rangeStats.totalTasks}</strong> tasks
            </span>
            <span className={styles.selectionStat}>
              <strong>${rangeStats.totalCost.toFixed(2)}</strong> cost
            </span>
            <span className={styles.selectionStat}>
              <strong>{(rangeStats.avgSuccess * 100).toFixed(0)}%</strong> success
            </span>
          </div>
          <button
            className={styles.selectionClear}
            onClick={() => onDateSelect({ start: '', end: '' })}
          >
            ✕
          </button>
        </div>
      )}

      {hoveredCell && (
        <div
          className={styles.heatmapTooltip}
          style={{ left: tooltipPos.x + 10, top: tooltipPos.y - 80 }}
        >
          <div className={styles.tooltipDate}>{hoveredCell.date}</div>
          <div className={styles.tooltipStat}>
            <span>{hoveredCell.taskCount} tasks</span>
            <span>${hoveredCell.cost.toFixed(3)}</span>
          </div>
          <div className={styles.tooltipSuccess}>
            {(hoveredCell.successRate * 100).toFixed(0)}% success
          </div>
          <div className={styles.tooltipHint}>Click & drag to select range</div>
        </div>
      )}
    </div>
  );
};

// Custom React Flow Node Components
const AgentNode: React.FC<NodeProps> = ({ data }) => {
  const statusColors: Record<string, string> = {
    idle: '#6b7280',
    busy: '#25c2a0',
    blocked: '#f59e0b',
    error: '#ef4444',
  };

  return (
    <div
      className={styles.rfAgentNode}
      data-status={data.status}
      onClick={() => data.onClick?.(data)}
    >
      <Handle type="target" position={Position.Top} className={styles.rfHandle} />
      <div className={styles.rfNodeStatus} style={{ backgroundColor: statusColors[data.status] }}>
        {getStatusIcon(data.status)}
      </div>
      <div className={styles.rfNodeContent}>
        <div className={styles.rfNodeLabel}>{data.label}</div>
        <div className={styles.rfNodeMeta}>
          <span className={styles.rfNodeTrust}>Trust: {data.trustScore}%</span>
          <span className={styles.rfNodeTasks}>{data.taskCount} tasks</span>
        </div>
      </div>
      <Handle type="source" position={Position.Bottom} className={styles.rfHandle} />
    </div>
  );
};

const SourceNode: React.FC<NodeProps> = ({ data }) => (
  <div className={styles.rfSourceNode}>
    <div className={styles.rfSourceIcon}>{data.icon}</div>
    <div className={styles.rfSourceLabel}>{data.label}</div>
    <Handle type="source" position={Position.Bottom} className={styles.rfHandle} />
  </div>
);

const SinkNode: React.FC<NodeProps> = ({ data }) => (
  <div className={styles.rfSinkNode} data-type={data.type}>
    <Handle type="target" position={Position.Top} className={styles.rfHandle} />
    <div className={styles.rfSinkContent}>
      <div className={styles.rfSinkLabel}>{data.label}</div>
      {data.badge && <div className={styles.rfSinkBadge}>{data.badge}</div>}
    </div>
    {data.hasOutput && <Handle type="source" position={Position.Bottom} className={styles.rfHandle} />}
  </div>
);

const nodeTypes = {
  agent: AgentNode,
  source: SourceNode,
  sink: SinkNode,
};

const AgentTopology: React.FC<{
  agents: Agent[];
  edges: TopologyEdge[];
  isExpanded: boolean;
  onToggleExpand: () => void;
  onAgentClick: (agent: Agent) => void;
}> = ({ agents, edges: topologyEdges, isExpanded, onToggleExpand, onAgentClick }) => {
  // Convert agents to React Flow nodes
  const initialNodes: Node[] = useMemo(() => {
    const nodePositions: Record<string, { x: number; y: number }> = {
      'github': { x: 350, y: 0 },
      'design-doc-creator': { x: 100, y: 150 },
      'sprint-planner': { x: 350, y: 150 },
      'sprint-executor': { x: 600, y: 150 },
      'eval-analyzer': { x: 100, y: 300 },
      'approval': { x: 350, y: 300 },
      'main': { x: 350, y: 450 },
    };

    const nodes: Node[] = [
      {
        id: 'github',
        type: 'source',
        position: nodePositions.github,
        data: { label: 'GitHub Issues', icon: '⬡' },
      },
      ...agents.map((agent) => ({
        id: agent.id,
        type: 'agent',
        position: nodePositions[agent.id] || { x: 350, y: 150 },
        data: {
          ...agent,
          onClick: onAgentClick,
        },
      })),
      {
        id: 'approval',
        type: 'sink',
        position: nodePositions.approval,
        data: { label: 'Approval Queue', badge: '12 ⏳', type: 'approval', hasOutput: true },
      },
      {
        id: 'main',
        type: 'sink',
        position: nodePositions.main,
        data: { label: 'Main Branch', badge: '✓ 29', type: 'success', hasOutput: false },
      },
    ];

    return nodes;
  }, [agents, onAgentClick]);

  // Convert edges to React Flow edges
  const initialEdges: Edge[] = useMemo(() => {
    return topologyEdges.map((edge, idx) => ({
      id: `edge-${idx}`,
      source: edge.source,
      target: edge.target,
      type: 'smoothstep',
      animated: edge.active,
      label: edge.messageCount.toString(),
      labelStyle: { fill: '#94a3b8', fontSize: 11, fontFamily: 'monospace' },
      labelBgStyle: { fill: '#1a1f2e', fillOpacity: 0.9 },
      labelBgPadding: [4, 2] as [number, number],
      style: {
        stroke: edge.active ? '#25c2a0' : '#374151',
        strokeWidth: edge.active ? 2 : 1,
      },
      markerEnd: {
        type: MarkerType.ArrowClosed,
        color: edge.active ? '#25c2a0' : '#374151',
        width: 20,
        height: 20,
      },
    }));
  }, [topologyEdges]);

  const [nodes, , onNodesChange] = useNodesState(initialNodes);
  const [flowEdges, , onEdgesChange] = useEdgesState(initialEdges);
  const reactFlowRef = useRef<ReactFlowInstance | null>(null);

  // Trigger fitView on init and store instance for later use
  const onInit = useCallback((reactFlowInstance: ReactFlowInstance) => {
    reactFlowRef.current = reactFlowInstance;
    // Multiple attempts to ensure container has proper dimensions
    const fitWithDelay = (delay: number) => {
      setTimeout(() => {
        reactFlowInstance.fitView({ padding: 0.2 });
      }, delay);
    };
    fitWithDelay(50);
    fitWithDelay(200);
    fitWithDelay(500);
  }, []);

  // Re-fit when expanded state changes
  useEffect(() => {
    if (reactFlowRef.current) {
      setTimeout(() => {
        reactFlowRef.current?.fitView({ padding: 0.2 });
      }, 100);
    }
  }, [isExpanded]);

  return (
    <div className={`${styles.topologyContainer} ${isExpanded ? styles.topologyExpanded : ''}`}>
      <div className={styles.topologyHeader}>
        <h3 className={styles.panelTitle}>
          <span className={styles.panelIcon}>◎</span>
          Agent Topology
        </h3>
        <div className={styles.topologyControls}>
          <button className={styles.expandBtn} onClick={onToggleExpand}>
            {isExpanded ? '⤡' : '⤢'}
          </button>
        </div>
        <div className={styles.topologyLegend}>
          <span className={styles.legendItem}><span className={styles.statusIdle}>○</span> idle</span>
          <span className={styles.legendItem}><span className={styles.statusBusy}>●</span> busy</span>
          <span className={styles.legendItem}><span className={styles.statusBlocked}>◐</span> blocked</span>
          <span className={styles.legendItem}><span className={styles.statusError}>✕</span> error</span>
        </div>
      </div>
      <div className={styles.topologyViewport}>
        <ReactFlow
          nodes={nodes}
          edges={flowEdges}
          onNodesChange={onNodesChange}
          onEdgesChange={onEdgesChange}
          onInit={onInit}
          nodeTypes={nodeTypes}
          fitView
          fitViewOptions={{ padding: 0.2 }}
          minZoom={0.3}
          maxZoom={2}
          defaultEdgeOptions={{
            type: 'smoothstep',
          }}
          proOptions={{ hideAttribution: true }}
        >
          <Background color="#1e293b" gap={24} size={1} />
          <Controls className={styles.rfControls} />
          {isExpanded && <MiniMap className={styles.rfMinimap} nodeColor="#374151" maskColor="rgba(13, 17, 23, 0.8)" />}
        </ReactFlow>
      </div>
    </div>
  );
};

// Message Queue Component
const MessageQueue: React.FC<{
  events: EventMessage[];
  onEventClick: (event: EventMessage) => void;
}> = ({ events, onEventClick }) => {
  const getEventIcon = (type: EventMessage['type']): string => {
    switch (type) {
      case 'task_start': return '▶';
      case 'task_complete': return '✓';
      case 'task_error': return '✕';
      case 'handoff': return '→';
      case 'approval': return '⏳';
      case 'message': return '◉';
    }
  };

  const getEventColor = (type: EventMessage['type']): string => {
    switch (type) {
      case 'task_start': return 'primary';
      case 'task_complete': return 'success';
      case 'task_error': return 'error';
      case 'handoff': return 'amber';
      case 'approval': return 'warning';
      case 'message': return 'muted';
    }
  };

  const formatTime = (timestamp: string): string => {
    const date = new Date(timestamp);
    return date.toLocaleTimeString('en-US', { hour12: false });
  };

  const formatRelativeTime = (timestamp: string): string => {
    const diff = Date.now() - new Date(timestamp).getTime();
    if (diff < 60000) return `${Math.floor(diff / 1000)}s ago`;
    if (diff < 3600000) return `${Math.floor(diff / 60000)}m ago`;
    return `${Math.floor(diff / 3600000)}h ago`;
  };

  return (
    <div className={styles.messageQueue}>
      <div className={styles.queueHeader}>
        <h3 className={styles.panelTitle}>
          <span className={styles.panelIcon}>▥</span>
          Event Queue
        </h3>
        <span className={styles.queueCount}>{events.length} events</span>
      </div>
      <div className={styles.queueList}>
        {events.map((event) => (
          <div
            key={event.id}
            className={styles.queueItem}
            onClick={() => onEventClick(event)}
            data-type={getEventColor(event.type)}
          >
            <span className={styles.queueIcon} data-type={getEventColor(event.type)}>
              {getEventIcon(event.type)}
            </span>
            <div className={styles.queueContent}>
              <div className={styles.queueMeta}>
                <span className={styles.queueSource}>{event.source}</span>
                {event.target && (
                  <>
                    <span className={styles.queueArrow}>→</span>
                    <span className={styles.queueTarget}>{event.target}</span>
                  </>
                )}
              </div>
              <div className={styles.queueMessage}>{event.content}</div>
            </div>
            <div className={styles.queueTime}>
              <span className={styles.queueRelative}>{formatRelativeTime(event.timestamp)}</span>
              <span className={styles.queueAbsolute}>{formatTime(event.timestamp)}</span>
            </div>
          </div>
        ))}
      </div>
    </div>
  );
};

// Detail Panel Component (slides in from right)
const DetailPanel: React.FC<{
  state: DetailPanelState;
  onClose: () => void;
  agents: Agent[];
  trustCapabilities?: TrustCapability[];
  onTrustChange?: (name: string, score: number) => void;
}> = ({ state, onClose, agents, trustCapabilities, onTrustChange }) => {
  if (!state.type || !state.id) return null;

  const renderContent = () => {
    switch (state.type) {
      case 'agent': {
        const agent = agents.find(a => a.id === state.id);
        if (!agent) return <div>Agent not found</div>;
        return (
          <div className={styles.detailContent}>
            <div className={styles.detailSection}>
              <h4>Status</h4>
              <span className={`${styles.detailStatus} ${styles[`status${agent.status.charAt(0).toUpperCase() + agent.status.slice(1)}`]}`}>
                {agent.status.toUpperCase()}
              </span>
            </div>
            <div className={styles.detailSection}>
              <h4>Statistics</h4>
              <div className={styles.detailStats}>
                <div className={styles.detailStat}>
                  <span className={styles.detailStatLabel}>Tasks</span>
                  <span className={styles.detailStatValue}>{agent.taskCount}</span>
                </div>
                <div className={styles.detailStat}>
                  <span className={styles.detailStatLabel}>Trust Score</span>
                  <span className={styles.detailStatValue}>{agent.trustScore}%</span>
                </div>
                <div className={styles.detailStat}>
                  <span className={styles.detailStatLabel}>Cost</span>
                  <span className={styles.detailStatValue}>${agent.cost.toFixed(2)}</span>
                </div>
              </div>
            </div>
            {trustCapabilities && onTrustChange && (
              <div className={styles.detailSection}>
                <h4>Trust Configuration</h4>
                <div className={styles.trustCapabilities}>
                  {trustCapabilities.map((cap) => {
                    const level = getTrustLevel(cap.score);
                    // Convert 'low-risk' to 'LowRisk' for CSS class
                    const levelClass = level.split('-').map(s => s.charAt(0).toUpperCase() + s.slice(1)).join('');
                    return (
                      <div key={cap.name} className={styles.trustRow}>
                        <div className={styles.trustCapLabel}>
                          <span className={styles.trustCapIcon}>{cap.icon}</span>
                          <span className={styles.trustCapName}>{cap.name}</span>
                          <span className={`${styles.trustValue} ${styles[`trust${levelClass}`]}`}>
                            {cap.score}%
                          </span>
                        </div>
                        <div className={styles.trustSliderContainer}>
                          <input
                            type="range"
                            min="0"
                            max="100"
                            value={cap.score}
                            onChange={(e) => onTrustChange(cap.name, parseInt(e.target.value))}
                            className={styles.trustSlider}
                          />
                          <div
                            className={`${styles.trustFill} ${styles[`trust${levelClass}`]}`}
                            style={{ width: `${cap.score}%` }}
                          />
                        </div>
                      </div>
                    );
                  })}
                </div>
              </div>
            )}
            <div className={styles.detailSection}>
              <h4>Actions</h4>
              <div className={styles.detailActions}>
                <button className={styles.actionBtn}>View Messages</button>
                <button className={styles.actionBtn}>View Traces</button>
                <button className={`${styles.actionBtn} ${styles.actionDanger}`}>Pause Agent</button>
              </div>
            </div>
          </div>
        );
      }
      case 'date': {
        const cell = state.data as HeatmapCell;
        return (
          <div className={styles.detailContent}>
            <div className={styles.detailSection}>
              <h4>Activity Summary</h4>
              <div className={styles.detailStats}>
                <div className={styles.detailStat}>
                  <span className={styles.detailStatLabel}>Tasks</span>
                  <span className={styles.detailStatValue}>{cell.taskCount}</span>
                </div>
                <div className={styles.detailStat}>
                  <span className={styles.detailStatLabel}>Cost</span>
                  <span className={styles.detailStatValue}>${cell.cost.toFixed(3)}</span>
                </div>
                <div className={styles.detailStat}>
                  <span className={styles.detailStatLabel}>Success</span>
                  <span className={styles.detailStatValue}>{(cell.successRate * 100).toFixed(0)}%</span>
                </div>
              </div>
            </div>
            <div className={styles.detailSection}>
              <h4>Actions</h4>
              <div className={styles.detailActions}>
                <button className={styles.actionBtn}>View Tasks</button>
                <button className={styles.actionBtn}>View Traces</button>
                <button className={styles.actionBtn}>Export Report</button>
              </div>
            </div>
          </div>
        );
      }
      case 'event': {
        const event = state.data as EventMessage;
        return (
          <div className={styles.detailContent}>
            <div className={styles.detailSection}>
              <h4>Event Details</h4>
              <div className={styles.eventDetail}>
                <div className={styles.eventRow}>
                  <span className={styles.eventLabel}>Type</span>
                  <span className={styles.eventValue}>{event.type}</span>
                </div>
                <div className={styles.eventRow}>
                  <span className={styles.eventLabel}>Source</span>
                  <span className={styles.eventValue}>{event.source}</span>
                </div>
                {event.target && (
                  <div className={styles.eventRow}>
                    <span className={styles.eventLabel}>Target</span>
                    <span className={styles.eventValue}>{event.target}</span>
                  </div>
                )}
                <div className={styles.eventRow}>
                  <span className={styles.eventLabel}>Time</span>
                  <span className={styles.eventValue}>{new Date(event.timestamp).toLocaleString()}</span>
                </div>
              </div>
            </div>
            <div className={styles.detailSection}>
              <h4>Content</h4>
              <pre className={styles.eventContent}>{event.content}</pre>
            </div>
          </div>
        );
      }
      default:
        return <div>Unknown detail type</div>;
    }
  };

  return (
    <div className={styles.detailPanel}>
      <div className={styles.detailHeader}>
        <h3 className={styles.detailTitle}>
          {state.type === 'agent' && `Agent: ${state.id}`}
          {state.type === 'date' && `Date: ${state.id}`}
          {state.type === 'event' && `Event: ${state.id.slice(0, 8)}`}
        </h3>
        <button className={styles.detailClose} onClick={onClose}>✕</button>
      </div>
      {renderContent()}
    </div>
  );
};

const TraceWaterfall: React.FC<{ spans: Span[] }> = ({ spans }) => {
  const totalDuration = useMemo(() => {
    let max = 0;
    const traverse = (span: Span) => {
      const end = span.startMs + span.durationMs;
      if (end > max) max = end;
      span.children?.forEach(traverse);
    };
    spans.forEach(traverse);
    return max;
  }, [spans]);

  const renderSpan = (span: Span, depth: number = 0): React.ReactNode => {
    const left = (span.startMs / totalDuration) * 100;
    const width = (span.durationMs / totalDuration) * 100;

    return (
      <div key={span.id} className={styles.waterfallRow}>
        <div className={styles.waterfallLabel} style={{ paddingLeft: `${12 + depth * 16}px` }}>
          <span className={styles.waterfallName}>{span.name}</span>
          <span className={styles.waterfallDuration}>{formatDuration(span.durationMs)}</span>
        </div>
        <div className={styles.waterfallBar}>
          <div
            className={styles.waterfallSegment}
            style={{ left: `${left}%`, width: `${Math.max(width, 0.5)}%` }}
            data-depth={depth % 4}
          />
        </div>
        {span.children?.map((child) => renderSpan(child, depth + 1))}
      </div>
    );
  };

  return (
    <div className={styles.waterfallContainer}>
      <div className={styles.waterfallHeader}>
        <h3 className={styles.panelTitle}>
          <span className={styles.panelIcon}>▥</span>
          Trace Waterfall
        </h3>
        <div className={styles.waterfallMeta}>
          <span className={styles.metaItem}>
            <span className={styles.metaLabel}>Duration</span>
            <span className={styles.metaValue}>{formatDuration(totalDuration)}</span>
          </span>
          <span className={styles.metaItem}>
            <span className={styles.metaLabel}>Status</span>
            <span className={`${styles.metaValue} ${styles.metaSuccess}`}>✓ Complete</span>
          </span>
          <span className={styles.metaItem}>
            <span className={styles.metaLabel}>Cost</span>
            <span className={styles.metaValue}>$0.0847</span>
          </span>
        </div>
      </div>
      <div className={styles.waterfallTimeline}>
        <div className={styles.timelineMarker} style={{ left: '0%' }}>0s</div>
        <div className={styles.timelineMarker} style={{ left: '25%' }}>{formatDuration(totalDuration * 0.25)}</div>
        <div className={styles.timelineMarker} style={{ left: '50%' }}>{formatDuration(totalDuration * 0.5)}</div>
        <div className={styles.timelineMarker} style={{ left: '75%' }}>{formatDuration(totalDuration * 0.75)}</div>
        <div className={styles.timelineMarker} style={{ left: '100%' }}>{formatDuration(totalDuration)}</div>
      </div>
      <div className={styles.waterfallRows}>
        {spans.map((span) => renderSpan(span))}
      </div>
    </div>
  );
};

const TrustConfigPanel: React.FC<{
  agentId: string;
  capabilities: TrustCapability[];
  onScoreChange: (name: string, score: number) => void;
}> = ({ agentId, capabilities, onScoreChange }) => {
  return (
    <div className={styles.trustContainer}>
      <div className={styles.trustHeader}>
        <h3 className={styles.panelTitle}>
          <span className={styles.panelIcon}>◉</span>
          Trust Configuration
        </h3>
        <span className={styles.trustAgent}>{agentId}</span>
      </div>
      <div className={styles.trustCapabilities}>
        {capabilities.map((cap) => {
          const level = getTrustLevel(cap.score);
          return (
            <div key={cap.name} className={styles.trustRow}>
              <div className={styles.trustCapLabel}>
                <span className={styles.trustCapIcon}>{cap.icon}</span>
                <span className={styles.trustCapName}>{cap.name}</span>
              </div>
              <div className={styles.trustSliderContainer}>
                <input
                  type="range"
                  min="0"
                  max="100"
                  value={cap.score}
                  onChange={(e) => onScoreChange(cap.name, parseInt(e.target.value))}
                  className={styles.trustSlider}
                  data-level={level}
                />
                <div
                  className={styles.trustSliderFill}
                  style={{ width: `${cap.score}%` }}
                  data-level={level}
                />
              </div>
              <div className={styles.trustScore}>
                <span className={styles.trustScoreValue}>{cap.score}</span>
                <span className={`${styles.trustLevel} ${styles[`trust${level.charAt(0).toUpperCase() + level.slice(1).replace('-', '')}`]}`}>
                  {level}
                </span>
              </div>
            </div>
          );
        })}
      </div>
      <div className={styles.trustLegend}>
        <div className={styles.trustLegendItem}>
          <span className={styles.trustLegendDot} data-level="manual" />
          <span>0-25: Manual</span>
        </div>
        <div className={styles.trustLegendItem}>
          <span className={styles.trustLegendDot} data-level="review" />
          <span>25-60: Review</span>
        </div>
        <div className={styles.trustLegendItem}>
          <span className={styles.trustLegendDot} data-level="low-risk" />
          <span>60-85: Low-Risk</span>
        </div>
        <div className={styles.trustLegendItem}>
          <span className={styles.trustLegendDot} data-level="auto" />
          <span>85-100: Auto</span>
        </div>
      </div>
    </div>
  );
};

interface GlobalStatsProps {
  stats?: {
    completedTasks: number;
    pendingApprovals: number;
    totalCost: string;
    totalTokens: number;
    taskSuccess: string;
  } | null;
  loading?: boolean;
}

const GlobalStats: React.FC<GlobalStatsProps> = ({ stats, loading }) => {
  const formatTokens = (n: number) => {
    if (n >= 1000000) return `${(n / 1000000).toFixed(1)}M`;
    if (n >= 1000) return `${(n / 1000).toFixed(1)}K`;
    return n.toString();
  };

  return (
    <div className={styles.globalStats}>
      <div className={styles.statCard}>
        <span className={styles.statIcon}>◈</span>
        <div className={styles.statContent}>
          <span className={styles.statValue}>{loading ? '...' : stats?.completedTasks ?? '—'}</span>
          <span className={styles.statLabel}>Completed</span>
        </div>
      </div>
      <div className={styles.statCard}>
        <span className={`${styles.statIcon} ${styles.statWarning}`}>⏳</span>
        <div className={styles.statContent}>
          <span className={`${styles.statValue} ${styles.statWarning}`}>
            {loading ? '...' : stats?.pendingApprovals ?? '—'}
          </span>
          <span className={styles.statLabel}>Pending</span>
        </div>
      </div>
      <div className={styles.statCard}>
        <span className={styles.statIcon}>$</span>
        <div className={styles.statContent}>
          <span className={styles.statValue}>{loading ? '...' : stats?.totalCost ?? '—'}</span>
          <span className={styles.statLabel}>Total Cost</span>
        </div>
      </div>
      <div className={styles.statCard}>
        <span className={styles.statIcon}>◎</span>
        <div className={styles.statContent}>
          <span className={styles.statValue}>
            {loading ? '...' : stats ? formatTokens(stats.totalTokens) : '—'}
          </span>
          <span className={styles.statLabel}>Tokens</span>
        </div>
      </div>
      <div className={styles.statCard}>
        <span className={`${styles.statIcon} ${styles.statSuccess}`}>✓</span>
        <div className={styles.statContent}>
          <span className={`${styles.statValue} ${styles.statSuccess}`}>
            {loading ? '...' : stats?.taskSuccess ?? '—'}
          </span>
          <span className={styles.statLabel}>Success Rate</span>
        </div>
      </div>
    </div>
  );
};

interface ControlPlaneProps {
  onSwitchToOldDashboard?: () => void;
}

// Main Component
export const ControlPlane: React.FC<ControlPlaneProps> = ({ onSwitchToOldDashboard }) => {
  const [searchQuery, setSearchQuery] = useState('');
  const [timeRange, setTimeRange] = useState('24h');
  const [selectedLevel, setSelectedLevel] = useState('global');
  const [trustCapabilities, setTrustCapabilities] = useState(mockTrustCapabilities);
  const [theme, setTheme] = useState<'dark' | 'light'>('dark');

  // Fetch real data from APIs
  const { data: heatmapResponse, loading: heatmapLoading } = useHeatmapData({ days: 90 });
  const { data: topologyData, loading: topologyLoading } = useTopologyData({ refreshInterval: 5000 });
  const { stats, loading: statsLoading } = useStatistics({ refreshInterval: 10000 });
  const { events: liveEvents, connected: wsConnected } = useEventQueue({ maxEvents: 50 });
  const { spans: traceSpans, traces } = useTraceData({ limit: 10 });

  // Transform data for components
  const heatmapData = useMemo(() => {
    if (heatmapResponse?.cells) return heatmapResponse.cells;
    return generateHeatmapData(); // Fallback to mock if loading
  }, [heatmapResponse]);

  const agents = useMemo(() => {
    if (topologyData?.agents) return topologyData.agents;
    return mockAgents; // Fallback to mock if loading
  }, [topologyData]);

  const edges = useMemo(() => {
    if (topologyData?.edges) {
      return topologyData.edges.map((e) => ({
        ...e,
        active: agents.some((a) => a.id === e.source && a.status === 'busy'),
      }));
    }
    return mockEdges; // Fallback to mock if loading
  }, [topologyData, agents]);

  const events = useMemo(() => {
    if (liveEvents.length > 0) return liveEvents;
    return mockEvents; // Fallback to mock if no live events
  }, [liveEvents]);

  const spans = useMemo(() => {
    if (traceSpans.length > 0) return traceSpans;
    return mockSpans; // Fallback to mock if no trace data
  }, [traceSpans]);

  // Interactive state
  const [selectedDateRange, setSelectedDateRange] = useState<DateRange | null>(null);
  const [topologyExpanded, setTopologyExpanded] = useState(false);
  const [detailPanel, setDetailPanel] = useState<DetailPanelState>({ type: null, id: null });
  const [selectedAgentId, setSelectedAgentId] = useState('sprint-executor');

  const handleTrustChange = useCallback((name: string, score: number) => {
    setTrustCapabilities(prev =>
      prev.map(cap => cap.name === name ? { ...cap, score } : cap)
    );
  }, []);

  const handleDateSelect = useCallback((range: DateRange) => {
    if (range.start === '' && range.end === '') {
      setSelectedDateRange(null);
    } else {
      setSelectedDateRange(range);
    }
  }, []);

  const handleCellClick = useCallback((cell: HeatmapCell) => {
    setDetailPanel({ type: 'date', id: cell.date, data: cell });
  }, []);

  const handleAgentClick = useCallback((agent: Agent) => {
    setDetailPanel({ type: 'agent', id: agent.id, data: agent });
    setSelectedAgentId(agent.id);
  }, []);

  const handleEventClick = useCallback((event: EventMessage) => {
    setDetailPanel({ type: 'event', id: event.id, data: event });
  }, []);

  const closeDetailPanel = useCallback(() => {
    setDetailPanel({ type: null, id: null });
  }, []);

  // Keyboard shortcut for search
  useEffect(() => {
    const handleKeyDown = (e: KeyboardEvent) => {
      if ((e.metaKey || e.ctrlKey) && e.key === 'k') {
        e.preventDefault();
        document.querySelector<HTMLInputElement>(`.${styles.searchInput}`)?.focus();
      }
      // Escape to close detail panel
      if (e.key === 'Escape') {
        closeDetailPanel();
        if (topologyExpanded) setTopologyExpanded(false);
      }
    };
    window.addEventListener('keydown', handleKeyDown);
    return () => window.removeEventListener('keydown', handleKeyDown);
  }, [closeDetailPanel, topologyExpanded]);

  const toggleTheme = useCallback(() => {
    setTheme((prev) => (prev === 'dark' ? 'light' : 'dark'));
  }, []);

  return (
    <div className={`${styles.controlPlane} ${theme === 'light' ? styles.controlPlaneLight : ''}`}>
      {/* Background effects */}
      <div className={styles.bgContours} />
      <div className={styles.bgScanlines} />

      {/* Header */}
      <header className={styles.header}>
        <div className={styles.headerBrand}>
          <span className={styles.brandIcon}>◎</span>
          <h1 className={styles.brandTitle}>AILANG</h1>
          <span className={styles.brandSub}>Control Plane</span>
          <button className={styles.themeToggle} onClick={toggleTheme} title={`Switch to ${theme === 'dark' ? 'light' : 'dark'} theme`}>
            <span className={styles.themeToggleIcon}>{theme === 'dark' ? '☀️' : '🌙'}</span>
            <span className={styles.themeToggleLabel}>{theme === 'dark' ? 'Light' : 'Dark'}</span>
          </button>
        </div>
        <GlobalStats stats={stats} loading={statsLoading} />
      </header>

      {/* Command Bar */}
      <CommandBar
        searchQuery={searchQuery}
        onSearchChange={setSearchQuery}
        timeRange={timeRange}
        onTimeRangeChange={setTimeRange}
      />

      {/* Main Layout */}
      <div className={styles.mainLayout}>
        {/* Left Sidebar */}
        <AggregationNav
          selectedLevel={selectedLevel}
          onSelectLevel={setSelectedLevel}
        />

        {/* Main Canvas */}
        <main className={`${styles.mainCanvas} ${topologyExpanded ? styles.canvasWithExpanded : ''}`}>
          <div className={styles.canvasRow}>
            <ActivityHeatmap
              data={heatmapData}
              selectedRange={selectedDateRange}
              onDateSelect={handleDateSelect}
              onCellClick={handleCellClick}
            />
          </div>
          <div className={`${styles.canvasRow} ${topologyExpanded ? styles.canvasRowExpanded : ''}`}>
            <AgentTopology
              agents={agents}
              edges={edges}
              isExpanded={topologyExpanded}
              onToggleExpand={() => setTopologyExpanded(!topologyExpanded)}
              onAgentClick={handleAgentClick}
            />
          </div>
          {!topologyExpanded && (
            <div className={styles.canvasRow}>
              <TraceWaterfall spans={spans} />
            </div>
          )}
        </main>

        {/* Right Panel - Event Queue */}
        <aside className={styles.contextPanel}>
          <MessageQueue
            events={events}
            onEventClick={handleEventClick}
          />
        </aside>
      </div>

      {/* Detail Panel Overlay */}
      {detailPanel.type && (
        <>
          <div className={styles.detailOverlay} onClick={closeDetailPanel} />
          <DetailPanel
            state={detailPanel}
            onClose={closeDetailPanel}
            agents={agents}
            trustCapabilities={trustCapabilities}
            onTrustChange={handleTrustChange}
          />
        </>
      )}

      {/* Footer */}
      <footer className={styles.footer}>
        <span className={styles.footerVersion}>v0.6.4</span>
        <span className={styles.footerStatus}>
          <span className={styles.footerDot} />
          Coordinator Active
        </span>
        {onSwitchToOldDashboard && (
          <button className={styles.oldDashboardLink} onClick={onSwitchToOldDashboard}>
            View Old Dashboard →
          </button>
        )}
        <span className={styles.footerTime}>{new Date().toISOString()}</span>
      </footer>
    </div>
  );
};

export default ControlPlane;
