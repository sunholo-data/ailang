# Dashboard Simplification - Remaining Work (v0.7.0)

**Status**: PR 1 Complete, PRs 2-6 Remaining
**Timeline**: 5 PRs × 1-2 days each = 5-10 days
**Priority**: ExecHierarchy components (most central to dashboard)

---

## Executive Summary

The dashboard UI has grown organically with experimental layering, resulting in:
- **4 crisis files** totaling 8,145 lines (36% of UI codebase)
- **~1,000+ lines of duplicated logic** across components (170 lines eliminated in PR 1)
- **23 scattered state variables** in ControlPlane.tsx
- **AI coding difficulty: VERY HIGH** (context explosion in >1000 line files)

**Goal**: Reduce all files to <500 lines each for optimal AI coding performance.

---

## Current State (After PR 1)

### ✅ PR 1 COMPLETED - Shared Utilities + Tests

**Changes made:**
- Extended `ui/src/utils/formatters.ts` with optional-handling variants
- Created `ui/src/features/controlplane/utils/nodeStyles.ts` (180 lines, 32 tests)
- Created `ui/src/features/controlplane/utils/dagreLayout.ts` (120 lines, 11 tests)
- Updated ExecHierarchyGraph.tsx to use new utilities (~130 lines removed)
- Updated TaskHierarchyGraph.tsx to use new utilities (~120 lines removed)
- Removed 4 redundant memoization hooks in ControlPlane.tsx
- Added vitest to package.json

**Results:**
- 71 tests passing
- ~170 lines of duplication eliminated
- Build successful
- Foundation established for subsequent PRs

### Critical Files Remaining

| File | Lines | Issues | Target |
|------|-------|--------|--------|
| EvolutionTree.tsx | 3,850 | D3 viz + layout + file hierarchy | 400 |
| ExecHierarchy.tsx | 1,842 | 237-line getSmartLabel() + 11 useState | 400 |
| ChatHistory.tsx | 1,376 | 3 data sources, no normalization | 500 |
| TaskHierarchyGraph.tsx | 1,077 | Custom dagre layout | 600 |
| ControlPlane.tsx | 774 | 23 state variables | 400 |

---

## PR 2: ExecHierarchy Split (Part 1 - Hooks)

**Goal**: Extract reusable hooks from ExecHierarchy.tsx
**Expected reduction**: 1,842 → ~1,400 lines
**Estimated effort**: 1-2 days

### Tasks

#### 2.1 Create `hooks/useSmartLabel.ts` (~100 lines)

**Purpose**: Consolidate 237-line `getSmartLabel()` function (lines 64-301 in ExecHierarchy.tsx)

**Current implementation** (3 locations):
- ExecHierarchy.tsx: 237 lines (lines 64-301)
- ChatHistory.tsx: ~150 lines (similar label extraction)
- TraceWaterfall.tsx: ~80 lines (simpler version)

**New hook signature**:
```typescript
export function useSmartLabel(span: Span): {
  title: string;
  subtitle?: string;
  icon?: ReactNode;
  metadata?: Record<string, string>;
}
```

**Implementation details**:
- Extract semantic type detection (coordinator, executor, tool, turn)
- Extract provider detection (claude, gemini, ollama, script)
- Extract label formatting rules (tool names, file paths, arguments)
- Extract icon selection logic
- Handle all edge cases from current implementation

**Test coverage**:
- Create `hooks/useSmartLabel.test.ts`
- Test all semantic types
- Test all providers
- Test label truncation
- Test icon selection
- Target: 15+ test cases

#### 2.2 Create `hooks/useExecHierarchyState.ts` (~80 lines)

**Purpose**: Extract 11 useState hooks scattered throughout ExecHierarchy.tsx

**Current state variables** (lines 41-88):
```typescript
const [viewMode, setViewMode] = useState<'tree' | 'graph' | 'timeline'>('tree');
const [coordViewMode, setCoordViewMode] = useState<'simple' | 'full'>('simple');
const [expandedNodes, setExpandedNodes] = useState<Set<string>>(new Set());
const [showCoordinator, setShowCoordinator] = useState(true);
const [showExecutor, setShowExecutor] = useState(true);
const [showTools, setShowTools] = useState(true);
const [showTurns, setShowTurns] = useState(true);
const [displayLimit, setDisplayLimit] = useState(100);
const [selectedNode, setSelectedNode] = useState<HierarchyNode | null>(null);
const [popoverOpen, setPopoverOpen] = useState(false);
const [recenterTrigger, setRecenterTrigger] = useState(0);
```

**New hook signature**:
```typescript
export function useExecHierarchyState() {
  return {
    state: {
      viewMode: 'tree' | 'graph' | 'timeline',
      coordViewMode: 'simple' | 'full',
      expandedNodes: Set<string>,
      filters: {
        showCoordinator: boolean,
        showExecutor: boolean,
        showTools: boolean,
        showTurns: boolean,
      },
      displayLimit: number,
      selectedNode: HierarchyNode | null,
      popoverOpen: boolean,
      recenterTrigger: number,
    },
    actions: {
      setViewMode: (mode: 'tree' | 'graph' | 'timeline') => void,
      setCoordViewMode: (mode: 'simple' | 'full') => void,
      toggleNodeExpand: (nodeId: string) => void,
      expandAll: () => void,
      collapseAll: () => void,
      toggleFilter: (filter: keyof Filters) => void,
      setDisplayLimit: (limit: number) => void,
      selectNode: (node: HierarchyNode | null) => void,
      openPopover: () => void,
      closePopover: () => void,
      triggerRecenter: () => void,
    }
  };
}
```

**Implementation details**:
- Group related state (filters into single object)
- Provide semantic action names
- Handle state dependencies (e.g., selectNode → openPopover)
- Persist view preferences to localStorage

**Test coverage**:
- Create `hooks/useExecHierarchyState.test.ts`
- Test all actions
- Test state transitions
- Test localStorage persistence
- Target: 20+ test cases

#### 2.3 Create `hooks/useIdService.ts` (~40 lines)

**Purpose**: Centralize scattered ID transformation logic

**Current usage** (scattered across ExecHierarchy.tsx):
- Normalize IDs (strip prefixes, handle UUIDs)
- Extract event IDs from messages
- Detect task IDs vs trace IDs
- Convert between formats

**New hook signature**:
```typescript
export function useIdService(): {
  normalize: (id: string) => string;
  fromEvent: (event: EventMessage) => string;
  isTaskId: (id: string) => boolean;
  isTraceId: (id: string) => boolean;
  extractShort: (id: string) => string; // First 8 chars
}
```

**Implementation details**:
- Handle `task-` prefix stripping
- Handle UUID format detection
- Handle `eval-` prefix for eval runs
- Consistent short ID extraction

**Test coverage**:
- Create `hooks/useIdService.test.ts`
- Test all ID formats (task-, eval-, UUID)
- Test prefix stripping
- Test format detection
- Target: 12+ test cases

#### 2.4 Update ExecHierarchy.tsx

**Changes**:
1. Replace inline `getSmartLabel()` with `useSmartLabel()` hook
2. Replace 11 useState hooks with `useExecHierarchyState()` hook
3. Replace scattered ID logic with `useIdService()` hook
4. Update all usages throughout component

**Expected result**:
- Lines: 1,842 → ~1,400 (442 lines removed)
- Cleaner component structure
- All tests passing

---

## PR 3: ExecHierarchy Split (Part 2 - Components)

**Goal**: Extract popover and toolbar components
**Expected reduction**: 1,400 → ~400 lines
**Estimated effort**: 1-2 days

### Tasks

#### 3.1 Extract `ExecHierarchyPopover.tsx` (~500 lines)

**Purpose**: Move entire popover rendering (lines ~1100-1600 in current file)

**Component structure**:
```typescript
export interface ExecHierarchyPopoverProps {
  node: HierarchyNode | null;
  open: boolean;
  onClose: () => void;
}

export const ExecHierarchyPopover: React.FC<ExecHierarchyPopoverProps> = ({
  node,
  open,
  onClose,
}) => {
  // Node details display
  // Attributes/metrics sections
  // Tool details expansion
  // JSON viewer
}
```

**Sections to include**:
- Node header (icon, label, status)
- Metadata (duration, cost, tokens)
- Attributes table (key-value pairs)
- Tool details (if applicable)
- JSON viewer (raw span data)
- Approval status (if applicable)

**Styling**:
- Keep existing CSS classes
- Extract relevant styles from ExecHierarchy.module.css
- Consider creating ExecHierarchyPopover.module.css

#### 3.2 Extract `ExecHierarchyToolbar.tsx` (~100 lines)

**Purpose**: Move header controls (top of ExecHierarchy.tsx)

**Component structure**:
```typescript
export interface ExecHierarchyToolbarProps {
  viewMode: 'tree' | 'graph' | 'timeline';
  coordViewMode: 'simple' | 'full';
  filters: {
    showCoordinator: boolean;
    showExecutor: boolean;
    showTools: boolean;
    showTurns: boolean;
  };
  displayLimit: number;
  onViewModeChange: (mode: 'tree' | 'graph' | 'timeline') => void;
  onCoordViewModeChange: (mode: 'simple' | 'full') => void;
  onFilterToggle: (filter: keyof Filters) => void;
  onDisplayLimitChange: (limit: number) => void;
  onExpandAll: () => void;
  onCollapseAll: () => void;
}
```

**Controls to include**:
- View mode switcher (tree/graph/timeline)
- Coordinator view toggle (simple/full)
- Filter checkboxes (coordinator, executor, tools, turns)
- Display limit selector
- Expand/collapse all buttons

#### 3.3 Simplify ExecHierarchy.tsx

**Remaining responsibilities**:
- State management (via hooks)
- View mode dispatch (tree/graph/timeline rendering)
- Data transformation (spans → hierarchy)
- Pass props to toolbar and popover

**Expected structure**:
```typescript
export const ExecHierarchy: React.FC<Props> = ({ spans, ... }) => {
  const { state, actions } = useExecHierarchyState();
  const idService = useIdService();

  // Transform spans to hierarchy
  const hierarchy = useMemo(() =>
    buildHierarchy(spans, state.filters, idService),
    [spans, state.filters, idService]
  );

  return (
    <div>
      <ExecHierarchyToolbar
        viewMode={state.viewMode}
        filters={state.filters}
        onViewModeChange={actions.setViewMode}
        onFilterToggle={actions.toggleFilter}
        {...}
      />

      {state.viewMode === 'tree' && <TreeView hierarchy={hierarchy} />}
      {state.viewMode === 'graph' && <GraphView hierarchy={hierarchy} />}
      {state.viewMode === 'timeline' && <TimelineView hierarchy={hierarchy} />}

      <ExecHierarchyPopover
        node={state.selectedNode}
        open={state.popoverOpen}
        onClose={actions.closePopover}
      />
    </div>
  );
}
```

**Expected result**:
- Lines: 1,400 → ~400 (1,000 lines removed)
- Single-responsibility component
- All tests passing

---

## PR 4: ChatHistory Refactor

**Goal**: Split data sources, normalize attributes
**Expected reduction**: 1,376 → ~500 lines
**Estimated effort**: 1-2 days

### Tasks

#### 4.1 Create `hooks/useChatClaudeHistory.ts` (~150 lines)

**Purpose**: Fetch and parse Claude Code JSONL sessions

**Current implementation**: Lines ~200-450 in ChatHistory.tsx

**Hook signature**:
```typescript
export function useChatClaudeHistory(sessionId?: string) {
  return {
    messages: NormalizedMessage[],
    loading: boolean,
    error: string | null,
    refresh: () => void,
  };
}
```

**Implementation details**:
- Fetch from `/api/claude-history/:sessionId`
- Parse JSONL format
- Extract turns (user messages, assistant responses, tool uses)
- Normalize to common message format
- Handle loading/error states

#### 4.2 Create `hooks/useChatCoordinatorEvents.ts` (~100 lines)

**Purpose**: Fetch coordinator task events

**Current implementation**: Lines ~450-600 in ChatHistory.tsx

**Hook signature**:
```typescript
export function useChatCoordinatorEvents(taskId?: string) {
  return {
    events: NormalizedMessage[],
    loading: boolean,
    error: string | null,
    refresh: () => void,
  };
}
```

**Implementation details**:
- Fetch from `/api/coordinator/tasks/:taskId/events`
- Parse event types (task_created, agent_started, tool_executed, etc.)
- Convert events to chat message format
- Normalize to common format

#### 4.3 Create `hooks/useChatOtelHierarchy.ts` (~80 lines)

**Purpose**: Fetch OTEL hierarchy as fallback

**Current implementation**: Lines ~600-750 in ChatHistory.tsx

**Hook signature**:
```typescript
export function useChatOtelHierarchy(traceId?: string) {
  return {
    messages: NormalizedMessage[],
    loading: boolean,
    error: string | null,
    refresh: () => void,
  };
}
```

**Implementation details**:
- Fetch from `/api/topology/hierarchy?trace_id=...`
- Extract turn/tool spans
- Convert to chat message format
- Normalize to common format

#### 4.4 Create `utils/chatNormalization.ts` (~80 lines)

**Purpose**: Centralize attribute name variations

**Current problem**: Different data sources use different field names:
- `task_id` vs `taskId` vs `task.id`
- `session_id` vs `sessionId` vs `session.id`
- `agent_id` vs `agentId` vs `agent`

**Utility functions**:
```typescript
export interface NormalizedMessage {
  id: string;
  type: 'user' | 'assistant' | 'tool_use' | 'tool_result';
  timestamp: string;
  content: string;
  metadata: {
    taskId?: string;
    sessionId?: string;
    agentId?: string;
    toolName?: string;
    cost?: number;
    tokens?: { input: number; output: number };
  };
}

export function normalizeMessage(raw: unknown): NormalizedMessage;
export function extractTaskId(attrs: Record<string, unknown>): string | undefined;
export function extractSessionId(attrs: Record<string, unknown>): string | undefined;
export function extractAgentId(attrs: Record<string, unknown>): string | undefined;
```

**Test coverage**:
- Create `utils/chatNormalization.test.ts`
- Test all attribute variations
- Test missing fields
- Test invalid inputs
- Target: 15+ test cases

#### 4.5 Simplify ChatHistory.tsx

**Remaining responsibilities**:
- Select appropriate data source hook based on props
- Merge messages from multiple sources (if needed)
- Render chat UI
- Handle message grouping/threading

**Expected structure**:
```typescript
export const ChatHistory: React.FC<Props> = ({ taskId, sessionId, traceId }) => {
  const claudeHistory = useChatClaudeHistory(sessionId);
  const coordEvents = useChatCoordinatorEvents(taskId);
  const otelHierarchy = useChatOtelHierarchy(traceId);

  // Select primary data source
  const messages = useMemo(() => {
    if (sessionId && claudeHistory.messages.length > 0) return claudeHistory.messages;
    if (taskId && coordEvents.events.length > 0) return coordEvents.events;
    return otelHierarchy.messages;
  }, [sessionId, taskId, claudeHistory, coordEvents, otelHierarchy]);

  return (
    <ChatContainer>
      {messages.map(msg => <ChatMessage key={msg.id} message={msg} />)}
    </ChatContainer>
  );
}
```

**Expected result**:
- Lines: 1,376 → ~500 (876 lines removed)
- Clear data source separation
- All tests passing

---

## PR 5: EvolutionTree Modularization

**Goal**: Split massive D3 component into focused modules
**Expected reduction**: 3,850 → ~400 lines
**Estimated effort**: 2-3 days
**Note**: Most complex refactor due to D3 visualization complexity

### Directory Structure

Create `EvolutionTree/` subdirectory:
```
EvolutionTree/
├── index.tsx                    # Main component (~400 lines)
├── layouts/
│   ├── spiral.ts                # Spiral positioning (~150 lines)
│   └── circlePacking.ts         # Circle packing (~200 lines)
├── hooks/
│   └── useFileHierarchy.ts      # File/dir aggregation (~200 lines)
├── components/
│   ├── TreeCanvas.tsx           # D3 rendering (~400 lines)
│   └── TreeTooltip.tsx          # Tooltip rendering (~200 lines)
└── utils/
    └── anomalyDetection.ts      # Anomaly detection (~100 lines)
```

### Tasks

#### 5.1 Extract Layout Algorithms

**File**: `layouts/spiral.ts` (~150 lines)

**Purpose**: Spiral positioning algorithm (lines ~800-950 in current file)

**Exports**:
```typescript
export interface SpiralLayoutOptions {
  width: number;
  height: number;
  padding: number;
  angleIncrement: number;
}

export function spiralLayout(
  nodes: FileNode[],
  options: SpiralLayoutOptions
): PositionedNode[];
```

**File**: `layouts/circlePacking.ts` (~200 lines)

**Purpose**: D3 circle packing layout (lines ~950-1150)

**Exports**:
```typescript
export interface CirclePackingOptions {
  width: number;
  height: number;
  padding: number;
}

export function circlePackingLayout(
  hierarchy: HierarchyNode,
  options: CirclePackingOptions
): PositionedNode[];
```

#### 5.2 Extract File Hierarchy Hook

**File**: `hooks/useFileHierarchy.ts` (~200 lines)

**Purpose**: Aggregate files by directory, compute metrics (lines ~200-400)

**Hook signature**:
```typescript
export function useFileHierarchy(spans: Span[]) {
  return {
    hierarchy: FileHierarchyNode,
    metrics: {
      totalFiles: number,
      totalSize: number,
      maxDepth: number,
    },
    loading: boolean,
  };
}
```

**Implementation details**:
- Extract file paths from spans
- Build directory tree
- Compute size metrics (line counts, token counts)
- Detect anomalies (outliers)
- Memoize results

#### 5.3 Extract D3 Canvas Component

**File**: `components/TreeCanvas.tsx` (~400 lines)

**Purpose**: D3 SVG rendering and interactions (lines ~1150-1550)

**Component structure**:
```typescript
export interface TreeCanvasProps {
  nodes: PositionedNode[];
  layoutMode: 'spiral' | 'circlePacking';
  onNodeClick: (node: FileNode) => void;
  onNodeHover: (node: FileNode | null) => void;
}

export const TreeCanvas: React.FC<TreeCanvasProps> = ({ ... }) => {
  const svgRef = useRef<SVGSVGElement>(null);

  useEffect(() => {
    const svg = d3.select(svgRef.current);
    // D3 rendering logic
  }, [nodes, layoutMode]);

  return <svg ref={svgRef} />;
}
```

**Responsibilities**:
- Create D3 selections
- Bind data to SVG elements
- Handle zoom/pan
- Handle node clicks/hovers
- Animate transitions

#### 5.4 Extract Tooltip Component

**File**: `components/TreeTooltip.tsx` (~200 lines)

**Purpose**: Tooltip rendering (lines ~1550-1750)

**Component structure**:
```typescript
export interface TreeTooltipProps {
  node: FileNode | null;
  position: { x: number; y: number };
  visible: boolean;
}

export const TreeTooltip: React.FC<TreeTooltipProps> = ({ node, position, visible }) => {
  if (!visible || !node) return null;

  return (
    <div
      className={styles.tooltip}
      style={{ left: position.x, top: position.y }}
    >
      <div className={styles.tooltipTitle}>{node.path}</div>
      <div className={styles.tooltipMetrics}>
        <span>Lines: {node.lineCount}</span>
        <span>Size: {formatBytes(node.size)}</span>
      </div>
    </div>
  );
}
```

#### 5.5 Extract Anomaly Detection

**File**: `utils/anomalyDetection.ts` (~100 lines)

**Purpose**: Detect outlier files (lines ~1750-1850)

**Exports**:
```typescript
export interface AnomalyDetectionOptions {
  threshold: number; // Standard deviations
  minSamples: number;
}

export function detectAnomalies(
  nodes: FileNode[],
  metric: 'size' | 'lineCount' | 'complexity',
  options: AnomalyDetectionOptions
): FileNode[];
```

**Implementation details**:
- Compute mean and standard deviation
- Flag nodes beyond threshold
- Return sorted by severity

#### 5.6 Create Main Index Component

**File**: `EvolutionTree/index.tsx` (~400 lines)

**Purpose**: Orchestrate all modules

**Component structure**:
```typescript
export const EvolutionTree: React.FC<Props> = ({ spans }) => {
  const { hierarchy, metrics } = useFileHierarchy(spans);
  const [layoutMode, setLayoutMode] = useState<'spiral' | 'circlePacking'>('spiral');
  const [hoveredNode, setHoveredNode] = useState<FileNode | null>(null);
  const [tooltipPos, setTooltipPos] = useState({ x: 0, y: 0 });

  const layout = useMemo(() => {
    if (layoutMode === 'spiral') {
      return spiralLayout(hierarchy.children, { ... });
    } else {
      return circlePackingLayout(hierarchy, { ... });
    }
  }, [hierarchy, layoutMode]);

  return (
    <div>
      <LayoutModeSelector mode={layoutMode} onChange={setLayoutMode} />
      <TreeCanvas
        nodes={layout}
        layoutMode={layoutMode}
        onNodeClick={handleNodeClick}
        onNodeHover={(node, pos) => {
          setHoveredNode(node);
          setTooltipPos(pos);
        }}
      />
      <TreeTooltip
        node={hoveredNode}
        position={tooltipPos}
        visible={!!hoveredNode}
      />
    </div>
  );
}
```

**Expected result**:
- Lines: 3,850 → ~400 (3,450 lines removed)
- Clean separation of concerns
- Reusable layout algorithms
- All tests passing

---

## PR 6: ControlPlane State Consolidation

**Goal**: Reduce 23 state variables to 3 consolidated objects
**Expected reduction**: 774 → ~400 lines
**Estimated effort**: 1-2 days

### Tasks

#### 6.1 Create `hooks/useFilterState.ts` (~60 lines)

**Purpose**: Consolidate filter state (currently 3 separate variables)

**Current state variables** (scattered in ControlPlane.tsx):
```typescript
const [selectedFilters, setSelectedFilters] = useState<string[]>([]);
const [dimensionFilters, setDimensionFilters] = useState<Record<string, string>>({});
const [filters, setFilters] = useState<Filters>({});
```

**New hook signature**:
```typescript
export function useFilterState() {
  return {
    filters: {
      selected: string[],
      dimensions: Record<string, string>,
      custom: Record<string, unknown>,
    },
    actions: {
      setFilter: (key: string, value: unknown) => void,
      clearFilter: (key: string) => void,
      clearAllFilters: () => void,
      toggleSelection: (item: string) => void,
    }
  };
}
```

**Implementation details**:
- Single state object for all filters
- Persistent to localStorage
- Provide convenience methods for common operations

**Test coverage**:
- Create `hooks/useFilterState.test.ts`
- Test filter operations
- Test localStorage persistence
- Target: 12+ test cases

#### 6.2 Create `hooks/useSelectionState.ts` (~50 lines)

**Purpose**: Consolidate selection state (currently 5 separate variables)

**Current state variables**:
```typescript
const [selectedTopologyNode, setSelectedTopologyNode] = useState<Node | null>(null);
const [selectedEventTraceId, setSelectedEventTraceId] = useState<string | null>(null);
const [detailPanel, setDetailPanel] = useState<'topology' | 'events' | null>(null);
const [highlightedPath, setHighlightedPath] = useState<string[]>([]);
const [highlightedSpanId, setHighlightedSpanId] = useState<string | null>(null);
```

**New hook signature**:
```typescript
export function useSelectionState() {
  return {
    selection: {
      topologyNode: Node | null,
      eventTraceId: string | null,
      detailPanel: 'topology' | 'events' | null,
      highlightedPath: string[],
      highlightedSpanId: string | null,
    },
    actions: {
      setSelection: (type: 'topology' | 'event', id: string) => void,
      clearSelection: () => void,
      setHighlight: (spanId: string | null, path?: string[]) => void,
      clearHighlight: () => void,
      openDetailPanel: (panel: 'topology' | 'events') => void,
      closeDetailPanel: () => void,
    }
  };
}
```

**Implementation details**:
- Single state object
- Handle selection dependencies (e.g., opening panel when selecting)
- Provide semantic actions

**Test coverage**:
- Create `hooks/useSelectionState.test.ts`
- Test selection operations
- Test state dependencies
- Target: 15+ test cases

#### 6.3 Create `controlPlaneReducer.ts` (~80 lines)

**Purpose**: Consolidate 13 separate callbacks into single dispatcher

**Current callbacks** (scattered in ControlPlane.tsx):
```typescript
const handleEventClick = (event: Event) => { ... };
const handleAgentClick = (agent: Agent) => { ... };
const handleNodeSelect = (node: Node) => { ... };
const handleTopologySelect = (node: Node) => { ... };
const handleSpanClick = (span: Span) => { ... };
const handleFilterChange = (filters: Filters) => { ... };
const handlePanelClose = () => { ... };
// ... 6 more
```

**New reducer**:
```typescript
export type ControlPlaneAction =
  | { type: 'SELECT_EVENT'; payload: { event: Event } }
  | { type: 'SELECT_AGENT'; payload: { agent: Agent } }
  | { type: 'SELECT_NODE'; payload: { node: Node } }
  | { type: 'SELECT_TOPOLOGY'; payload: { node: Node } }
  | { type: 'SELECT_SPAN'; payload: { span: Span } }
  | { type: 'UPDATE_FILTERS'; payload: { filters: Filters } }
  | { type: 'CLOSE_PANEL' }
  | { type: 'CLEAR_SELECTION' };

export interface ControlPlaneState {
  filters: FilterState;
  selection: SelectionState;
  ui: {
    activePanelTab: string;
    isExpanded: boolean;
  };
}

export function controlPlaneReducer(
  state: ControlPlaneState,
  action: ControlPlaneAction
): ControlPlaneState;
```

**Implementation details**:
- Handle all UI actions
- Coordinate state updates (filters + selection)
- Provide type-safe dispatch

#### 6.4 Refactor ControlPlane.tsx

**Changes**:
1. Replace filter state with `useFilterState()` hook
2. Replace selection state with `useSelectionState()` hook
3. Replace callbacks with `useReducer(controlPlaneReducer)`
4. Simplify component logic

**Expected structure**:
```typescript
export const ControlPlane: React.FC = () => {
  const { filters, actions: filterActions } = useFilterState();
  const { selection, actions: selectionActions } = useSelectionState();
  const [uiState, dispatch] = useReducer(controlPlaneReducer, initialState);

  // Data fetching hooks
  const { events } = useLiveEvents(filters);
  const { topology } = useTopologyData(selection);
  const { spans } = useSpanData(selection);

  return (
    <div>
      <ControlPlaneHeader filters={filters} onFilterChange={filterActions.setFilter} />
      <ControlPlaneContent
        events={events}
        topology={topology}
        spans={spans}
        selection={selection}
        onEvent={(e) => dispatch({ type: 'SELECT_EVENT', payload: { event: e } })}
      />
    </div>
  );
}
```

**Expected result**:
- Lines: 774 → ~400 (374 lines removed)
- 23 state variables → 3 consolidated objects
- 13 callbacks → 1 dispatcher
- All tests passing

---

## Testing Strategy

### Unit Tests

**Target coverage**: >80% for all new files

**Test files to create**:
- PR 2: `useSmartLabel.test.ts`, `useExecHierarchyState.test.ts`, `useIdService.test.ts`
- PR 3: `ExecHierarchyPopover.test.tsx`, `ExecHierarchyToolbar.test.tsx`
- PR 4: `useChatClaudeHistory.test.ts`, `useChatCoordinatorEvents.test.ts`, `chatNormalization.test.ts`
- PR 5: `spiral.test.ts`, `circlePacking.test.ts`, `useFileHierarchy.test.ts`, `anomalyDetection.test.ts`
- PR 6: `useFilterState.test.ts`, `useSelectionState.test.ts`, `controlPlaneReducer.test.ts`

**Total new test files**: 17

### Integration Tests

**Test scenarios**:
- End-to-end view switching (tree → graph → timeline)
- Filter application across all components
- Selection propagation (click event → update all views)
- Data source switching (Claude history → coordinator events → OTEL)

### Manual Testing Checklist

For each PR:
- [ ] Build succeeds (`npm run build`)
- [ ] All tests pass (`npm test`)
- [ ] Linting passes (`npm run lint`)
- [ ] UI renders correctly
- [ ] No console errors
- [ ] Interactions work as expected
- [ ] No performance regressions (check React DevTools Profiler)

---

## Expected Outcomes

### File Size Improvements

| File | Before | After | Reduction |
|------|--------|-------|-----------|
| EvolutionTree.tsx | 3,850 | 400 | -90% (3,450 lines) |
| ExecHierarchy.tsx | 1,842 | 400 | -78% (1,442 lines) |
| ChatHistory.tsx | 1,376 | 500 | -64% (876 lines) |
| TaskHierarchyGraph.tsx | 1,077 | 600 | -44% (477 lines) |
| ControlPlane.tsx | 774 | 400 | -48% (374 lines) |

**Total reduction**: 6,619 lines removed from 5 files

### New Files Created

**Utilities** (3 files, ~400 lines):
- ✅ `utils/formatters.ts` extensions (PR 1)
- ✅ `utils/nodeStyles.ts` (PR 1)
- ✅ `utils/dagreLayout.ts` (PR 1)
- `utils/chatNormalization.ts` (PR 4)

**Hooks** (8 files, ~720 lines):
- `hooks/useSmartLabel.ts` (PR 2)
- `hooks/useExecHierarchyState.ts` (PR 2)
- `hooks/useIdService.ts` (PR 2)
- `hooks/useChatClaudeHistory.ts` (PR 4)
- `hooks/useChatCoordinatorEvents.ts` (PR 4)
- `hooks/useChatOtelHierarchy.ts` (PR 4)
- `hooks/useFileHierarchy.ts` (PR 5)
- `hooks/useFilterState.ts` (PR 6)
- `hooks/useSelectionState.ts` (PR 6)

**Components** (4 files, ~1,200 lines):
- `ExecHierarchyPopover.tsx` (PR 3)
- `ExecHierarchyToolbar.tsx` (PR 3)
- `EvolutionTree/components/TreeCanvas.tsx` (PR 5)
- `EvolutionTree/components/TreeTooltip.tsx` (PR 5)

**Layout Algorithms** (2 files, ~350 lines):
- `EvolutionTree/layouts/spiral.ts` (PR 5)
- `EvolutionTree/layouts/circlePacking.ts` (PR 5)

**Other** (2 files, ~180 lines):
- `EvolutionTree/utils/anomalyDetection.ts` (PR 5)
- `controlPlaneReducer.ts` (PR 6)

**Test files**: 17 files, ~1,500 lines

### Benefits

1. **AI Coding Performance**
   - All files <500 lines → context fits in working memory
   - Claude/GPT can understand full file context
   - Faster iteration on changes
   - Fewer mistakes due to missing context

2. **Maintainability**
   - Single responsibility per file
   - Clear separation of concerns
   - Easier to find relevant code
   - Reduced cognitive load

3. **Testability**
   - Hooks and utilities can be unit tested
   - Components can be tested in isolation
   - Test coverage >80% for new code

4. **Reusability**
   - Extracted hooks can be used elsewhere
   - Layout algorithms can be applied to other visualizations
   - Utilities reduce future duplication

5. **Performance**
   - Consolidated state reduces re-renders
   - Better memoization opportunities
   - Smaller bundle size (tree shaking)

---

## Dependencies Between PRs

```
PR 1 (Utilities) ✅ COMPLETE
    ↓
PR 2 (ExecHierarchy Hooks)
    ↓
PR 3 (ExecHierarchy Components) ← depends on PR 2
    ↓
PR 4 (ChatHistory) ← independent, can run parallel to PR 5
    ↓
PR 5 (EvolutionTree) ← independent, can run parallel to PR 4
    ↓
PR 6 (ControlPlane) ← depends on all previous PRs
```

**Parallel opportunities**:
- PR 4 and PR 5 can be done in parallel
- PR 6 should wait until all others are complete

---

## Risk Assessment

### High Risk

**EvolutionTree.tsx refactor (PR 5)**
- Complex D3 visualization logic
- Many interdependencies
- Easy to break rendering
- **Mitigation**: Extensive manual testing, visual regression testing

### Medium Risk

**ExecHierarchy hooks extraction (PR 2)**
- 11 state variables → complex state management
- Many usages throughout component
- **Mitigation**: Incremental testing, careful prop threading

**ControlPlane reducer (PR 6)**
- 13 callbacks → single dispatcher
- Coordinates multiple state slices
- **Mitigation**: Comprehensive test coverage, type safety

### Low Risk

**Utility extraction (PR 1)** ✅ COMPLETE
**ChatHistory refactor (PR 4)**
**Component extraction (PR 3)**

---

## Completion Criteria

Each PR must meet:
- [ ] All tests passing (unit + integration)
- [ ] Linting passes
- [ ] Build succeeds
- [ ] No console errors
- [ ] Manual testing checklist complete
- [ ] Code review approved
- [ ] Documentation updated (if needed)

---

## Timeline Estimate

| PR | Effort | Duration |
|----|--------|----------|
| PR 1 | ✅ Done | Completed |
| PR 2 | 1-2 days | 1-2 days |
| PR 3 | 1-2 days | 1-2 days |
| PR 4 | 1-2 days | 1-2 days |
| PR 5 | 2-3 days | 2-3 days |
| PR 6 | 1-2 days | 1-2 days |

**Total**: 6-11 days (sequential) or 4-7 days (with parallelization)

---

## Next Steps

1. Review and approve this design document
2. Create GitHub issues for PRs 2-6
3. Begin PR 2 implementation (ExecHierarchy hooks)
4. Iterate through remaining PRs

---

## Appendix: Code Locations

### ExecHierarchy.tsx Key Sections

- **getSmartLabel()**: Lines 64-301 (237 lines) → Extract to `useSmartLabel()`
- **useState hooks**: Lines 41-88 (11 hooks) → Extract to `useExecHierarchyState()`
- **Popover rendering**: Lines ~1100-1600 (500 lines) → Extract to `ExecHierarchyPopover.tsx`
- **Toolbar controls**: Lines ~100-200 (100 lines) → Extract to `ExecHierarchyToolbar.tsx`

### ChatHistory.tsx Key Sections

- **Claude history fetching**: Lines ~200-450 (250 lines) → Extract to `useChatClaudeHistory()`
- **Coordinator events**: Lines ~450-600 (150 lines) → Extract to `useChatCoordinatorEvents()`
- **OTEL hierarchy**: Lines ~600-750 (150 lines) → Extract to `useChatOtelHierarchy()`

### EvolutionTree.tsx Key Sections

- **File hierarchy**: Lines ~200-400 (200 lines) → Extract to `useFileHierarchy()`
- **Spiral layout**: Lines ~800-950 (150 lines) → Extract to `layouts/spiral.ts`
- **Circle packing**: Lines ~950-1150 (200 lines) → Extract to `layouts/circlePacking.ts`
- **D3 rendering**: Lines ~1150-1550 (400 lines) → Extract to `components/TreeCanvas.tsx`
- **Tooltip**: Lines ~1550-1750 (200 lines) → Extract to `components/TreeTooltip.tsx`
- **Anomaly detection**: Lines ~1750-1850 (100 lines) → Extract to `utils/anomalyDetection.ts`

### ControlPlane.tsx Key Sections

- **Filter state**: 3 separate useState → Consolidate to `useFilterState()`
- **Selection state**: 5 separate useState → Consolidate to `useSelectionState()`
- **Callbacks**: 13 separate functions → Consolidate to `controlPlaneReducer`
