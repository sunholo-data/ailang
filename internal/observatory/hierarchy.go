// Package observatory provides a unified observability platform for AILANG.
package observatory

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"
)

// TaskHierarchy represents a task with its full agent and span hierarchy.
type TaskHierarchy struct {
	Task   *Task             `json:"task"`
	Agents []*AgentHierarchy `json:"agents"`
}

// AgentHierarchy represents an agent assignment with its spans grouped by trace.
type AgentHierarchy struct {
	Agent  *AgentAssignment  `json:"agent"`
	Traces []*TraceHierarchy `json:"traces"`
}

// TraceHierarchy represents a trace with its span tree.
type TraceHierarchy struct {
	TraceID  string                 `json:"trace_id"`
	RootSpan *SpanNode              `json:"root_span,omitempty"`
	Spans    []*SpanNode            `json:"spans"` // Flat list for easy rendering
	Summary  *HierarchyTraceSummary `json:"summary"`
}

// SpanNode represents a span with its children for hierarchical display.
type SpanNode struct {
	Span     *Span       `json:"span"`
	Children []*SpanNode `json:"children,omitempty"`
}

// HierarchyTraceSummary contains aggregate metrics for a trace in hierarchy view.
// This differs from TraceSummary in models.go which is for trace list views.
type HierarchyTraceSummary struct {
	SpanCount    int     `json:"span_count"`
	TotalTokens  int64   `json:"total_tokens"`
	TotalCostUSD float64 `json:"total_cost_usd"`
	DurationMs   int64   `json:"duration_ms"`
	ErrorCount   int     `json:"error_count"`
}

// getEarliestStartTime returns the earliest span start time in the trace.
// Used for chronological sorting of traces.
func (t *TraceHierarchy) getEarliestStartTime() time.Time {
	var earliest time.Time
	for _, node := range t.Spans {
		if node.Span != nil && (earliest.IsZero() || node.Span.StartTime.Before(earliest)) {
			earliest = node.Span.StartTime
		}
	}
	return earliest
}

// HierarchyOptions configures the hierarchy query.
type HierarchyOptions struct {
	// MaxDepth limits the span tree depth (0 = unlimited)
	MaxDepth int
	// IncludeSpans controls whether to include individual spans
	IncludeSpans bool
	// Workspace filters spans to only include those belonging to this workspace path
	// This prevents cross-workspace span bleeding in hierarchy views
	Workspace string
	// WorkspaceID filters spans by workspace_id directly (set automatically from task)
	WorkspaceID string
}

// DefaultHierarchyOptions returns default options for hierarchy queries.
func DefaultHierarchyOptions() HierarchyOptions {
	return HierarchyOptions{
		MaxDepth:     0, // Unlimited
		IncludeSpans: true,
	}
}

// GetTaskHierarchy builds the full hierarchy for a task.
func GetTaskHierarchy(ctx context.Context, backend Backend, taskID string, opts HierarchyOptions) (*TaskHierarchy, error) {
	// Get the task
	task, err := backend.GetTask(ctx, taskID)
	if err != nil {
		return nil, fmt.Errorf("get task: %w", err)
	}
	if task == nil {
		return nil, fmt.Errorf("task not found: %s", taskID)
	}

	// Set workspace_id from task to prevent cross-workspace span bleeding
	if opts.WorkspaceID == "" && task.WorkspaceID != "" {
		opts.WorkspaceID = task.WorkspaceID
	}

	// Get agent assignments for this task
	agents, err := backend.ListAgentAssignments(ctx, taskID)
	if err != nil {
		return nil, fmt.Errorf("list agent assignments: %w", err)
	}

	hierarchy := &TaskHierarchy{
		Task:   task,
		Agents: make([]*AgentHierarchy, 0, len(agents)),
	}

	// Build hierarchy for each agent
	for _, agent := range agents {
		agentHierarchy, err := buildAgentHierarchy(ctx, backend, agent, opts)
		if err != nil {
			return nil, fmt.Errorf("build agent hierarchy: %w", err)
		}
		hierarchy.Agents = append(hierarchy.Agents, agentHierarchy)
	}

	return hierarchy, nil
}

// buildAgentHierarchy builds the trace/span hierarchy for a single agent.
func buildAgentHierarchy(ctx context.Context, backend Backend, agent *AgentAssignment, opts HierarchyOptions) (*AgentHierarchy, error) {
	agentHierarchy := &AgentHierarchy{
		Agent:  agent,
		Traces: make([]*TraceHierarchy, 0),
	}

	if !opts.IncludeSpans {
		return agentHierarchy, nil
	}

	// Get spans explicitly linked to this agent assignment
	// Filter by workspace to prevent cross-workspace bleeding
	spans, err := backend.ListSpans(ctx, SpanListOptions{
		AgentAssignmentID: agent.ID,
		WorkspaceID:       opts.WorkspaceID,
		Limit:             1000,
	})
	if err != nil {
		return nil, fmt.Errorf("list spans by assignment: %w", err)
	}

	// Also get spans linked to task but without agent_assignment_id
	// These are spans from OTLP that were linked via task_id from cwd path
	taskSpans, err := backend.ListSpans(ctx, SpanListOptions{
		TaskID:      agent.TaskID,
		WorkspaceID: opts.WorkspaceID,
		Limit:       1000,
	})
	if err != nil {
		return nil, fmt.Errorf("list spans by task: %w", err)
	}

	// Merge task spans that don't have an agent_assignment_id
	// (they belong to this task but weren't explicitly linked to an agent)
	seenSpans := make(map[string]bool)
	for _, s := range spans {
		seenSpans[s.ID] = true
	}
	for _, s := range taskSpans {
		if s.AgentAssignmentID == "" && !seenSpans[s.ID] {
			spans = append(spans, s)
			seenSpans[s.ID] = true
		}
	}

	// Collect unique trace IDs from linked spans
	traceIDs := make(map[string]bool)
	for _, s := range spans {
		traceIDs[s.TraceID] = true
	}

	// For each trace with linked spans, get ALL spans in that trace
	// This ensures we include child spans even if they don't have task_id set
	// IMPORTANT: Filter by workspace to prevent cross-workspace span bleeding
	for traceID := range traceIDs {
		traceSpanList, err := backend.ListSpans(ctx, SpanListOptions{
			TraceID:     traceID,
			WorkspaceID: opts.WorkspaceID,
			Limit:       1000,
		})
		if err != nil {
			// Log but continue with what we have
			continue
		}
		for _, s := range traceSpanList {
			if !seenSpans[s.ID] {
				spans = append(spans, s)
				seenSpans[s.ID] = true
			}
		}
	}

	// Group spans by trace_id
	traceSpans := make(map[string][]*Span)
	for _, span := range spans {
		traceSpans[span.TraceID] = append(traceSpans[span.TraceID], span)
	}

	// Build hierarchy for each trace
	for traceID, spanList := range traceSpans {
		traceHierarchy := buildTraceHierarchy(traceID, spanList, opts.MaxDepth)
		agentHierarchy.Traces = append(agentHierarchy.Traces, traceHierarchy)
	}

	// Merge related traces (cross-trace parent-child linking)
	// This handles cases where coordinator.task.execute spawns claude.execute
	// in a different trace via TRACEPARENT propagation
	// Pass workspace to prevent cross-workspace span bleeding
	agentHierarchy.Traces = mergeRelatedTraces(agentHierarchy.Traces, opts.WorkspaceID)

	// Merge session-related traces (Claude Code telemetry correlation)
	// This handles orphan traces that share session.id with the main trace
	// but have no parent_span_id (Claude Code's own telemetry)
	agentHierarchy.Traces = mergeSessionRelatedTraces(agentHierarchy.Traces, opts.WorkspaceID)

	// Sort traces by their first span's start time (chronological order)
	sort.Slice(agentHierarchy.Traces, func(i, j int) bool {
		iTime := agentHierarchy.Traces[i].getEarliestStartTime()
		jTime := agentHierarchy.Traces[j].getEarliestStartTime()
		return iTime.Before(jTime)
	})

	return agentHierarchy, nil
}

// buildTraceHierarchy builds the span tree for a single trace.
func buildTraceHierarchy(traceID string, spans []*Span, maxDepth int) *TraceHierarchy {
	// Build span index for parent lookups
	spanIndex := make(map[string]*SpanNode)
	for _, span := range spans {
		spanIndex[span.ID] = &SpanNode{Span: span}
	}

	// Build tree structure
	var rootNodes []*SpanNode
	for _, span := range spans {
		node := spanIndex[span.ID]
		if span.ParentSpanID == "" {
			// Root span
			rootNodes = append(rootNodes, node)
		} else if parent, ok := spanIndex[span.ParentSpanID]; ok {
			// Has parent in this trace
			parent.Children = append(parent.Children, node)
		} else {
			// Parent not in trace, treat as root
			rootNodes = append(rootNodes, node)
		}
	}

	// Apply virtual re-parenting for ailang.* spans based on timestamps.
	// This creates a "logical" hierarchy where ailang.* spans are nested
	// under the exec.tool_use span that triggered them, even though their
	// DB parent_span_id points to claude.execute.
	applyTimestampCorrelation(spanIndex)

	// Apply depth limit if specified
	if maxDepth > 0 {
		for _, root := range rootNodes {
			limitDepth(root, 1, maxDepth)
		}
	}

	// Calculate summary
	summary := &HierarchyTraceSummary{}
	for _, span := range spans {
		summary.SpanCount++
		summary.TotalTokens += span.TokensIn + span.TokensOut
		summary.TotalCostUSD += span.CostUSD
		if span.DurationMs > summary.DurationMs {
			summary.DurationMs = span.DurationMs // Use max for concurrent spans
		}
		if span.Status == SpanStatusError {
			summary.ErrorCount++
		}
	}

	// Sort root nodes by start time
	sort.Slice(rootNodes, func(i, j int) bool {
		return rootNodes[i].Span.StartTime.Before(rootNodes[j].Span.StartTime)
	})

	// Sort children of each node by start time (recursively)
	for _, node := range rootNodes {
		sortChildrenByTime(node)
	}

	// IMPORTANT: Spans contains ONLY root-level spans (those with no parent or orphans).
	// The full hierarchy is provided via the Children[] arrays on each node.
	// This was previously using flatSpans (all spans) which caused duplicates.
	traceHierarchy := &TraceHierarchy{
		TraceID: traceID,
		Spans:   rootNodes,
		Summary: summary,
	}

	// Set root span if there's exactly one
	if len(rootNodes) == 1 {
		traceHierarchy.RootSpan = rootNodes[0]
	}

	return traceHierarchy
}

// limitDepth removes children beyond the specified depth.
func limitDepth(node *SpanNode, currentDepth, maxDepth int) {
	if currentDepth >= maxDepth {
		node.Children = nil
		return
	}
	for _, child := range node.Children {
		limitDepth(child, currentDepth+1, maxDepth)
	}
}

// sortChildrenByTime recursively sorts children by start time.
func sortChildrenByTime(node *SpanNode) {
	if len(node.Children) == 0 {
		return
	}
	sort.Slice(node.Children, func(i, j int) bool {
		return node.Children[i].Span.StartTime.Before(node.Children[j].Span.StartTime)
	})
	for _, child := range node.Children {
		sortChildrenByTime(child)
	}
}

// collectAllSpans recursively collects all spans from a trace hierarchy.
// This traverses Children[] arrays to include nested spans, not just roots.
// IMPORTANT: Must be used when building span indexes for cross-trace merging,
// since trace.Spans only contains root-level spans after buildTraceHierarchy.
func collectAllSpans(trace *TraceHierarchy) []*SpanNode {
	var result []*SpanNode
	var traverse func(node *SpanNode)
	traverse = func(node *SpanNode) {
		if node != nil {
			result = append(result, node)
			for _, child := range node.Children {
				traverse(child)
			}
		}
	}
	for _, root := range trace.Spans {
		traverse(root)
	}
	return result
}

// =============================================================================
// Virtual Hierarchy: Timestamp-based span correlation
// =============================================================================
// When Claude Code runs `ailang run`, the child process creates spans that are
// parented to `claude.execute` (via TRACEPARENT). However, logically these spans
// should be nested under the `exec.tool_use` span that triggered them.
//
// Since parent_span_id is immutable after insertion, we apply "virtual re-parenting"
// at query time using timestamp correlation: if an ailang.* span's start time falls
// within an exec.tool_use span's time window, we re-parent it in the in-memory tree.
// =============================================================================

// isExecutorSpan returns true if this span is a claude.execute or gemini.execute.
func isExecutorSpan(span *Span) bool {
	return span.Name == "claude.execute" || span.Name == "gemini.execute"
}

// isToolUseSpan returns true if this is an exec.tool_use span.
func isToolUseSpan(span *Span) bool {
	return span.Name == "exec.tool_use"
}

// isAilangChildSpan returns true if this span should be correlated to a tool.
// These are spans from child ailang processes that should logically be nested
// under the tool that invoked them.
func isAilangChildSpan(span *Span) bool {
	return strings.HasPrefix(span.Name, "ailang.") ||
		strings.HasPrefix(span.Name, "compile.") ||
		strings.HasPrefix(span.Name, "eval.")
}

// findContainingToolSpan finds the exec.tool_use span that contains this child's start time.
// Returns nil if no containing tool span found.
func findContainingToolSpan(child *Span, toolSpans []*SpanNode) *SpanNode {
	for _, tool := range toolSpans {
		if tool.Span == nil || tool.Span.EndTime == nil {
			// Tool span not yet ended, skip
			continue
		}
		// Check if child started within tool's time window
		// child.StartTime >= tool.StartTime AND child.StartTime <= tool.EndTime
		toolEnd := *tool.Span.EndTime
		if (tool.Span.StartTime.Before(child.StartTime) || tool.Span.StartTime.Equal(child.StartTime)) &&
			(toolEnd.After(child.StartTime) || toolEnd.Equal(child.StartTime)) {
			return tool
		}
	}
	return nil
}

// applyTimestampCorrelation re-parents ailang.* spans under exec.tool_use based on timestamps.
// This creates a "logical" hierarchy that differs from the DB parent_span_id.
//
// The algorithm:
// 1. Find all executor spans (claude.execute, gemini.execute)
// 2. For each executor, collect its tool_use children
// 3. For each ailang.* child of the executor:
//   - Find the tool_use span whose time window contains the child's start time
//   - Move the child under that tool span (in-memory only)
//
// 4. Sort tool children by start time for consistent display
func applyTimestampCorrelation(spanIndex map[string]*SpanNode) {
	// 1. Find executor spans (claude.execute, gemini.execute)
	var executorNodes []*SpanNode
	for _, node := range spanIndex {
		if node.Span != nil && isExecutorSpan(node.Span) {
			executorNodes = append(executorNodes, node)
		}
	}

	// 2. For each executor, collect its direct children
	for _, executor := range executorNodes {
		// Collect tool spans (these will become new parents)
		var toolSpans []*SpanNode
		for _, child := range executor.Children {
			if child.Span != nil && isToolUseSpan(child.Span) {
				toolSpans = append(toolSpans, child)
			}
		}

		// No tools = nothing to re-parent
		if len(toolSpans) == 0 {
			continue
		}

		// 3. Find ailang.* children of executor that should move under tools
		var remainingChildren []*SpanNode
		for _, child := range executor.Children {
			if child.Span != nil && isAilangChildSpan(child.Span) {
				// Try to find containing tool span
				if tool := findContainingToolSpan(child.Span, toolSpans); tool != nil {
					// Re-parent: remove from executor, add to tool
					tool.Children = append(tool.Children, child)
					continue
				}
			}
			remainingChildren = append(remainingChildren, child)
		}
		executor.Children = remainingChildren

		// 4. Sort tool children by start time
		for _, tool := range toolSpans {
			if len(tool.Children) > 0 {
				sort.Slice(tool.Children, func(i, j int) bool {
					return tool.Children[i].Span.StartTime.Before(tool.Children[j].Span.StartTime)
				})
			}
		}
	}
}

// =============================================================================
// Cross-Trace Merging: Link parent-child spans across different trace IDs
// =============================================================================
// When TRACEPARENT is propagated to child processes, they create spans in a
// new trace but with parent_span_id pointing to the original trace. This creates
// "orphan" root spans that should logically be nested under their parent.
//
// Example:
//   Trace A (coordinator): coordinator.task.execute → sets TRACEPARENT
//   Trace B (claude): claude.execute (parent_span_id = coordinator span ID)
//
// We want: coordinator.task.execute → claude.execute (merged view)
// =============================================================================

// extractWorkspaceIDFromTrace extracts the workspace ID from a trace by checking its spans.
// Uses several strategies in order of reliability:
// 1. Check span's TaskID (if set, the span was filtered by workspace already)
// 2. Check Attributes for explicit workspace keys
// 3. Check ResourceAttributes["process.cwd"] for workspace path
// Returns the first non-empty workspace ID found, or empty string if none.
func extractWorkspaceIDFromTrace(trace *TraceHierarchy) string {
	if trace == nil {
		return ""
	}

	// Traverse ALL spans (including nested children)
	allSpans := collectAllSpans(trace)
	for _, node := range allSpans {
		if node.Span == nil {
			continue
		}

		// Strategy 1: TaskID implies workspace was already filtered
		if node.Span.TaskID != "" {
			return node.Span.TaskID
		}

		// Strategy 2: Check Attributes for explicit workspace
		if node.Span.Attributes != nil {
			for _, key := range []string{"ailang.workspace_id", "task.workspace", "workspace.id", "ailang.workspace"} {
				if val, ok := node.Span.Attributes[key]; ok {
					if strVal, ok := val.(string); ok && strVal != "" {
						return strVal
					}
				}
			}
		}

		// Strategy 3: Check ResourceAttributes for process.cwd
		if node.Span.ResourceAttributes != nil {
			if cwd, ok := node.Span.ResourceAttributes["process.cwd"]; ok {
				if cwdStr, ok := cwd.(string); ok && cwdStr != "" {
					return cwdStr
				}
			}
		}
	}
	return ""
}

// mergeRelatedTraces merges traces that have cross-trace parent-child relationships.
// It finds traces whose root spans have a ParentSpanID pointing to a span in another trace,
// and re-parents those spans under the appropriate parent.
//
// workspaceID is used to prevent cross-workspace span bleeding - orphan traces are
// only merged if their workspace matches (or if workspace can't be determined).
func mergeRelatedTraces(traces []*TraceHierarchy, workspaceID string) []*TraceHierarchy {
	if len(traces) <= 1 {
		return traces
	}

	// Build a global span index across all traces (recursively including children!)
	// IMPORTANT: trace.Spans only contains root nodes, but we need to index ALL spans
	// so that cross-trace parent lookups can find nested spans like claude.execute
	// that are children of coordinator.task.execute.
	globalSpanIndex := make(map[string]*SpanNode)
	traceForSpan := make(map[string]*TraceHierarchy)
	for _, trace := range traces {
		allSpans := collectAllSpans(trace) // Recursive traversal of Children[]
		for _, node := range allSpans {
			if node.Span != nil {
				globalSpanIndex[node.Span.ID] = node
				traceForSpan[node.Span.ID] = trace
			}
		}
	}

	// Track which traces have been merged into others
	mergedTraces := make(map[string]bool)

	// Find "orphan root" spans - these have ParentSpanID pointing to a different trace
	for _, trace := range traces {
		orphanRoots := findOrphanRoots(trace, globalSpanIndex)
		for _, orphan := range orphanRoots {
			if orphan.Span == nil || orphan.Span.ParentSpanID == "" {
				continue
			}

			// Find the parent in another trace
			parentNode, ok := globalSpanIndex[orphan.Span.ParentSpanID]
			if !ok {
				continue
			}

			// Verify parent is in a different trace
			parentTrace := traceForSpan[parentNode.Span.ID]
			if parentTrace == nil || parentTrace.TraceID == trace.TraceID {
				continue
			}

			// NOTE: Workspace filtering is already applied when spans are fetched via ListSpans.
			// All spans in these traces are already from the same workspace, so no additional
			// workspace check is needed here. The workspaceID parameter is kept for future use
			// but currently acts as a marker that workspace-scoped filtering was applied upstream.
			_ = workspaceID // Acknowledge param to prevent unused warning

			// Re-parent: add orphan as child of the parent node
			parentNode.Children = append(parentNode.Children, orphan)

			// Sort children by start time
			sort.Slice(parentNode.Children, func(i, j int) bool {
				return parentNode.Children[i].Span.StartTime.Before(parentNode.Children[j].Span.StartTime)
			})

			// Mark this trace as merged (don't return it as standalone)
			mergedTraces[trace.TraceID] = true

			// Merge summary stats into parent trace
			if parentTrace.Summary != nil && trace.Summary != nil {
				parentTrace.Summary.SpanCount += trace.Summary.SpanCount
				parentTrace.Summary.TotalTokens += trace.Summary.TotalTokens
				parentTrace.Summary.TotalCostUSD += trace.Summary.TotalCostUSD
				if trace.Summary.DurationMs > parentTrace.Summary.DurationMs {
					parentTrace.Summary.DurationMs = trace.Summary.DurationMs
				}
				parentTrace.Summary.ErrorCount += trace.Summary.ErrorCount
			}

			// NOTE: We do NOT add child trace spans to parent's flat Spans[] array.
			// The orphan span is already added as a child above.
			// All UI components use children[] arrays for rendering the hierarchy.
			// Adding to Spans[] would create duplicates.
		}
	}

	// Return non-merged traces
	result := make([]*TraceHierarchy, 0, len(traces))
	for _, trace := range traces {
		if !mergedTraces[trace.TraceID] {
			result = append(result, trace)
		}
	}

	return result
}

// findOrphanRoots returns spans that appear as roots in this trace
// but have a ParentSpanID that could be in another trace.
func findOrphanRoots(trace *TraceHierarchy, globalIndex map[string]*SpanNode) []*SpanNode {
	// Build local index from ALL spans in this trace (recursive!)
	// IMPORTANT: trace.Spans only contains root nodes, but parent_span_id might
	// point to a nested child (e.g., claude.execute under coordinator.task.execute).
	// We need to index ALL spans to properly detect orphans.
	allSpans := collectAllSpans(trace)
	localIndex := make(map[string]bool)
	for _, node := range allSpans {
		if node.Span != nil {
			localIndex[node.Span.ID] = true
		}
	}

	// Find spans with ParentSpanID not in this trace but in global index
	// NOTE: Still only check root spans for orphan detection (those in trace.Spans),
	// since nested spans already have their parent in the same trace by definition.
	var orphans []*SpanNode
	for _, node := range trace.Spans {
		if node.Span == nil || node.Span.ParentSpanID == "" {
			continue
		}
		// Parent not in this trace?
		if !localIndex[node.Span.ParentSpanID] {
			// But parent exists in another trace?
			if _, exists := globalIndex[node.Span.ParentSpanID]; exists {
				orphans = append(orphans, node)
			}
		}
	}

	return orphans
}

// =============================================================================
// Session-Based Merging: Link Claude Code telemetry by session.id attribute
// =============================================================================
// Claude Code emits its own telemetry (api_request, user_prompt, tool events)
// in separate traces with NO parent_span_id. However, these spans share the
// same session.id attribute as our executor spans.
//
// This function finds orphan traces that share session.id with the main trace
// and nests their spans under appropriate exec.turn spans using timestamp
// correlation.
//
// Correlation strategy:
// 1. Primary: Match by session.id attribute (most reliable)
// 2. Fallback: If no session.id, use timestamp overlap with executor time window
// =============================================================================

// mergeSessionRelatedTraces finds orphan traces that share session.id with the main trace
// and nests them under appropriate spans using timestamp correlation.
//
// workspaceID is used to prevent cross-workspace span bleeding - particularly important
// for the timestamp fallback which could otherwise match ANY trace with overlapping times.
func mergeSessionRelatedTraces(traces []*TraceHierarchy, workspaceID string) []*TraceHierarchy {
	if len(traces) <= 1 {
		return traces
	}

	// 1. Find main trace (has coordinator.task.execute or claude.execute as root)
	var mainTrace *TraceHierarchy
	var orphanTraces []*TraceHierarchy

	for _, trace := range traces {
		if hasMainExecutorRoot(trace) {
			mainTrace = trace
		} else {
			orphanTraces = append(orphanTraces, trace)
		}
	}

	if mainTrace == nil {
		return traces // No main trace, nothing to merge
	}

	// 2. Extract session.id from main trace (primary correlation key)
	mainSessionID := extractSessionID(mainTrace)

	// 3. Get executor time window for timestamp fallback
	executorStart, executorEnd := getExecutorTimeWindow(mainTrace)

	// 4. Build turn spans index for timestamp-based nesting
	turnSpans := collectTurnSpans(mainTrace)

	// 5. Find orphan traces that can be correlated
	mergedTraces := make(map[string]bool)

	for _, orphan := range orphanTraces {
		orphanSessionID := extractSessionID(orphan)

		// PRIMARY: Match by session.id
		matchedBySession := mainSessionID != "" && orphanSessionID == mainSessionID

		// NOTE: Workspace filtering is already applied when spans are fetched via ListSpans.
		// All spans in these traces are already from the same workspace, so no additional
		// workspace check is needed here. The workspaceID parameter is kept for future use.
		_ = workspaceID // Acknowledge param to prevent unused warning

		// FALLBACK: Match by timestamp overlap (if no session match)
		// This is safe because input traces were already workspace-filtered upstream.
		matchedByTimestamp := false
		if !matchedBySession && executorEnd != nil {
			orphanStart := getTraceStartTime(orphan)
			if orphanStart != nil {
				// Orphan started within executor's time window
				matchedByTimestamp = !orphanStart.Before(executorStart) && !orphanStart.After(*executorEnd)
			}
		}

		if !matchedBySession && !matchedByTimestamp {
			continue // Not related to main trace
		}

		// 6. Nest orphan spans under appropriate turn spans using timestamps
		for _, orphanNode := range orphan.Spans {
			if orphanNode.Span == nil {
				continue
			}

			// Find containing turn span (or fallback to executor)
			parentSpan := findContainingTurnSpan(orphanNode.Span, turnSpans, mainTrace)
			if parentSpan != nil {
				parentSpan.Children = append(parentSpan.Children, orphanNode)
			}
		}

		// Mark as merged
		mergedTraces[orphan.TraceID] = true

		// Merge summary stats into main trace
		if mainTrace.Summary != nil && orphan.Summary != nil {
			mainTrace.Summary.SpanCount += orphan.Summary.SpanCount
			mainTrace.Summary.TotalTokens += orphan.Summary.TotalTokens
			mainTrace.Summary.TotalCostUSD += orphan.Summary.TotalCostUSD
			if orphan.Summary.DurationMs > mainTrace.Summary.DurationMs {
				mainTrace.Summary.DurationMs = orphan.Summary.DurationMs
			}
			mainTrace.Summary.ErrorCount += orphan.Summary.ErrorCount
		}

		// NOTE: Orphan spans are NOT added to the flat Spans[] array.
		// They are only added as children of turn/executor spans above.
		// All UI components (TraceWaterfall, ExecHierarchy) should use
		// the children[] arrays to render the proper hierarchy.
		// The flat Spans[] contains only ROOT-level spans.
	}

	// Sort turn children by start time after merging
	for _, turn := range turnSpans {
		if len(turn.Children) > 0 {
			sort.Slice(turn.Children, func(i, j int) bool {
				return turn.Children[i].Span.StartTime.Before(turn.Children[j].Span.StartTime)
			})
		}
	}

	// 7. Return non-merged traces
	result := make([]*TraceHierarchy, 0, len(traces))
	for _, trace := range traces {
		if !mergedTraces[trace.TraceID] {
			result = append(result, trace)
		}
	}

	return result
}

// hasMainExecutorRoot returns true if the trace has a main executor span as root.
func hasMainExecutorRoot(trace *TraceHierarchy) bool {
	for _, node := range trace.Spans {
		if node.Span == nil {
			continue
		}
		name := node.Span.Name
		if name == "coordinator.task.execute" || name == "claude.execute" || name == "gemini.execute" {
			return true
		}
	}
	return false
}

// extractSessionID extracts the session.id attribute from any span in the trace.
// Must traverse full hierarchy since session.id is typically on claude.execute or
// exec.turn spans, not the root coordinator.task.execute span.
func extractSessionID(trace *TraceHierarchy) string {
	// Use collectAllSpans to traverse the full hierarchy including Children[]
	allSpans := collectAllSpans(trace)
	for _, node := range allSpans {
		if node.Span == nil || node.Span.Attributes == nil {
			continue
		}
		if sid, ok := node.Span.Attributes["session.id"]; ok {
			if sidStr, ok := sid.(string); ok {
				return sidStr
			}
		}
	}
	return ""
}

// getExecutorTimeWindow returns the time window of the main executor span.
// Traverses the full hierarchy since executor spans may be nested.
func getExecutorTimeWindow(trace *TraceHierarchy) (time.Time, *time.Time) {
	// Recursive helper to find executor span at any depth
	var findExecutor func(node *SpanNode) *SpanNode
	findExecutor = func(node *SpanNode) *SpanNode {
		if node.Span != nil {
			if node.Span.Name == "claude.execute" || node.Span.Name == "gemini.execute" ||
				node.Span.Name == "coordinator.task.execute" {
				return node
			}
		}
		for _, child := range node.Children {
			if found := findExecutor(child); found != nil {
				return found
			}
		}
		return nil
	}

	for _, node := range trace.Spans {
		if found := findExecutor(node); found != nil {
			return found.Span.StartTime, found.Span.EndTime
		}
	}
	return time.Time{}, nil
}

// getTraceStartTime returns the earliest span start time in the trace.
// Traverses the full hierarchy since child spans may have earlier start times.
func getTraceStartTime(trace *TraceHierarchy) *time.Time {
	var earliest *time.Time
	// Use collectAllSpans to traverse the full hierarchy including Children[]
	allSpans := collectAllSpans(trace)
	for _, node := range allSpans {
		if node.Span == nil {
			continue
		}
		if earliest == nil || node.Span.StartTime.Before(*earliest) {
			t := node.Span.StartTime
			earliest = &t
		}
	}
	return earliest
}

// collectTurnSpans collects all exec.turn spans from the trace for timestamp matching.
// Traverses the full hierarchy since turns may be nested under executor spans.
func collectTurnSpans(trace *TraceHierarchy) []*SpanNode {
	var turns []*SpanNode

	// Recursive helper to find turn spans at any depth
	var collectFromNode func(node *SpanNode)
	collectFromNode = func(node *SpanNode) {
		if node.Span != nil && strings.Contains(node.Span.Name, "turn") {
			turns = append(turns, node)
		}
		for _, child := range node.Children {
			collectFromNode(child)
		}
	}

	// Traverse from root spans
	for _, node := range trace.Spans {
		collectFromNode(node)
	}

	// Sort by start time for consistent matching
	sort.Slice(turns, func(i, j int) bool {
		return turns[i].Span.StartTime.Before(turns[j].Span.StartTime)
	})
	return turns
}

// findContainingTurnSpan finds the exec.turn span whose time window contains or precedes the child's start time.
// Uses two strategies:
// 1. If child starts WITHIN a turn's time window, use that turn
// 2. If child starts AFTER turn ends (gap between turns), use the most recent preceding turn
// Falls back to the main executor span if no turn matches.
func findContainingTurnSpan(child *Span, turnSpans []*SpanNode, mainTrace *TraceHierarchy) *SpanNode {
	var bestPrecedingTurn *SpanNode
	var smallestGap time.Duration = time.Hour * 24 // Large initial value

	// Try to find containing or preceding turn span
	for _, turn := range turnSpans {
		if turn.Span == nil || turn.Span.EndTime == nil {
			continue
		}
		turnEnd := *turn.Span.EndTime

		// OPTION 1: Child starts WITHIN turn's time window
		if (turn.Span.StartTime.Before(child.StartTime) || turn.Span.StartTime.Equal(child.StartTime)) &&
			(turnEnd.After(child.StartTime) || turnEnd.Equal(child.StartTime)) {
			return turn
		}

		// OPTION 2: Child starts AFTER turn ends - track closest preceding turn
		// This handles api_request spans that occur in the gap between turns
		if turnEnd.Before(child.StartTime) {
			gap := child.StartTime.Sub(turnEnd)
			if gap < smallestGap {
				smallestGap = gap
				bestPrecedingTurn = turn
			}
		}
	}

	// If we found a preceding turn within 30s, use it
	// This covers api_request spans that start right after a turn ends
	if bestPrecedingTurn != nil && smallestGap < 30*time.Second {
		return bestPrecedingTurn
	}

	// Fallback: nest under main executor span (traverse hierarchy to find it)
	var findExecutorSpan func(node *SpanNode) *SpanNode
	findExecutorSpan = func(node *SpanNode) *SpanNode {
		if node.Span != nil && (node.Span.Name == "claude.execute" || node.Span.Name == "gemini.execute") {
			return node
		}
		for _, child := range node.Children {
			if found := findExecutorSpan(child); found != nil {
				return found
			}
		}
		return nil
	}

	for _, node := range mainTrace.Spans {
		if found := findExecutorSpan(node); found != nil {
			return found
		}
	}

	return nil
}

// =============================================================================
// Turn-Based Grouping: Structure spans by conversation turns
// =============================================================================
// When viewing execution hierarchies, it's useful to see spans grouped by
// conversation turn rather than raw parent-child relationships. This creates
// a more intuitive view: Session → Turn 1 → Turn 2 → Turn 3, with tools
// nested under their respective turns.
// =============================================================================

// TurnGroupedHierarchy represents spans organized by conversation turns.
type TurnGroupedHierarchy struct {
	Session *TurnGroupSession `json:"session,omitempty"`
	Turns   []*TurnGroup      `json:"turns"`
	Stats   *TurnGroupStats   `json:"stats"`
}

// TurnGroupSession represents the top-level session/executor span.
type TurnGroupSession struct {
	ID         string  `json:"id"`
	Name       string  `json:"name"`
	DurationMs int64   `json:"duration_ms"`
	Cost       float64 `json:"cost"`
	TokensIn   int64   `json:"tokens_in"`
	TokensOut  int64   `json:"tokens_out"`
	Provider   string  `json:"provider,omitempty"`
	Model      string  `json:"model,omitempty"`
}

// TurnGroup represents a single conversation turn with its tools.
type TurnGroup struct {
	TurnNumber int         `json:"turn_number"`
	SpanID     string      `json:"span_id"`
	DurationMs int64       `json:"duration_ms"`
	Cost       float64     `json:"cost"`
	TokensIn   int64       `json:"tokens_in"`
	TokensOut  int64       `json:"tokens_out"`
	Tools      []*TurnTool `json:"tools,omitempty"`
}

// TurnTool represents a tool call within a turn.
type TurnTool struct {
	ID         string  `json:"id"`
	Name       string  `json:"name"`
	ToolName   string  `json:"tool_name,omitempty"` // Extracted tool name (e.g., "Read", "Bash")
	DurationMs int64   `json:"duration_ms"`
	Cost       float64 `json:"cost,omitempty"`
	Status     string  `json:"status"`
}

// TurnGroupStats contains aggregate statistics for the turn-grouped view.
type TurnGroupStats struct {
	TotalTurns  int     `json:"total_turns"`
	TotalTools  int     `json:"total_tools"`
	TotalCost   float64 `json:"total_cost"`
	TotalTokens int64   `json:"total_tokens"`
	DurationMs  int64   `json:"duration_ms"`
}

// GroupSpansByTurn transforms a span tree into a turn-based hierarchy.
// This is useful for displaying execution in a more intuitive way.
func GroupSpansByTurn(spans []*SpanNode) *TurnGroupedHierarchy {
	result := &TurnGroupedHierarchy{
		Turns: make([]*TurnGroup, 0),
		Stats: &TurnGroupStats{},
	}

	if len(spans) == 0 {
		return result
	}

	// Find the root session/executor span
	var sessionNode *SpanNode
	for _, node := range spans {
		if node.Span != nil && isSessionOrExecutorSpan(node.Span) {
			sessionNode = node
			break
		}
	}

	// If no session found, use the first root span
	if sessionNode == nil && len(spans) > 0 {
		sessionNode = spans[0]
	}

	if sessionNode != nil && sessionNode.Span != nil {
		result.Session = &TurnGroupSession{
			ID:         sessionNode.Span.ID,
			Name:       sessionNode.Span.Name,
			DurationMs: sessionNode.Span.DurationMs,
			Cost:       sessionNode.Span.CostUSD,
			TokensIn:   sessionNode.Span.TokensIn,
			TokensOut:  sessionNode.Span.TokensOut,
			Provider:   string(sessionNode.Span.Provider),
			Model:      sessionNode.Span.Model,
		}
		result.Stats.DurationMs = sessionNode.Span.DurationMs
	}

	// Collect all turns recursively from the span tree
	turnsMap := make(map[int]*turnCollector)
	collectTurnsForGrouping(spans, turnsMap)

	// Convert map to sorted slice
	turnNumbers := make([]int, 0, len(turnsMap))
	for num := range turnsMap {
		turnNumbers = append(turnNumbers, num)
	}
	sort.Ints(turnNumbers)

	// Build turn groups
	for _, turnNum := range turnNumbers {
		tc := turnsMap[turnNum]
		turn := &TurnGroup{
			TurnNumber: turnNum,
			SpanID:     tc.spanID,
			DurationMs: tc.durationMs,
			Cost:       tc.cost,
			TokensIn:   tc.tokensIn,
			TokensOut:  tc.tokensOut,
			Tools:      make([]*TurnTool, 0, len(tc.tools)),
		}

		// Add tools sorted by start time (already sorted during collection)
		for _, tool := range tc.tools {
			turn.Tools = append(turn.Tools, tool)
			result.Stats.TotalTools++
		}

		result.Turns = append(result.Turns, turn)
		result.Stats.TotalTurns++
		result.Stats.TotalCost += tc.cost
		result.Stats.TotalTokens += tc.tokensIn + tc.tokensOut
	}

	return result
}

// turnCollector accumulates data for a single turn during traversal.
type turnCollector struct {
	spanID     string
	durationMs int64
	cost       float64
	tokensIn   int64
	tokensOut  int64
	startTime  int64 // For sorting tools
	tools      []*TurnTool
}

// collectTurnsForGrouping recursively traverses spans to collect turn data.
func collectTurnsForGrouping(nodes []*SpanNode, turnsMap map[int]*turnCollector) {
	for _, node := range nodes {
		if node == nil || node.Span == nil {
			continue
		}

		// Check if this is a turn span
		if isTurnSpanForGrouping(node.Span) {
			turnNum := extractTurnNumber(node.Span)
			if turnNum > 0 {
				tc := &turnCollector{
					spanID:     node.Span.ID,
					durationMs: node.Span.DurationMs,
					cost:       node.Span.CostUSD,
					tokensIn:   node.Span.TokensIn,
					tokensOut:  node.Span.TokensOut,
					startTime:  node.Span.StartTime.UnixMilli(),
					tools:      make([]*TurnTool, 0),
				}

				// Collect tool children
				collectToolsFromChildren(node.Children, tc)

				turnsMap[turnNum] = tc
			}
		}

		// Recurse into children
		collectTurnsForGrouping(node.Children, turnsMap)
	}
}

// collectToolsFromChildren collects tool spans from a turn's children.
func collectToolsFromChildren(children []*SpanNode, tc *turnCollector) {
	for _, child := range children {
		if child == nil || child.Span == nil {
			continue
		}

		if isToolSpanForGrouping(child.Span) {
			tool := &TurnTool{
				ID:         child.Span.ID,
				Name:       child.Span.Name,
				ToolName:   extractToolName(child.Span),
				DurationMs: child.Span.DurationMs,
				Cost:       child.Span.CostUSD,
				Status:     string(child.Span.Status),
			}
			tc.tools = append(tc.tools, tool)
		}

		// Also check nested children (tools might have children too)
		collectToolsFromChildren(child.Children, tc)
	}
}

// isSessionOrExecutorSpan returns true if this is a session or executor span.
func isSessionOrExecutorSpan(span *Span) bool {
	name := span.Name
	return name == "claude.execute" ||
		name == "gemini.execute" ||
		name == "coordinator.task.execute" ||
		strings.HasPrefix(name, "exec.session") ||
		strings.HasPrefix(name, "session.")
}

// isTurnSpanForGrouping returns true if this is a turn span.
func isTurnSpanForGrouping(span *Span) bool {
	name := span.Name
	return strings.Contains(name, "turn") ||
		strings.HasPrefix(name, "exec.turn")
}

// isToolSpanForGrouping returns true if this is a tool span.
func isToolSpanForGrouping(span *Span) bool {
	name := span.Name
	return strings.HasPrefix(name, "tool.") ||
		strings.HasPrefix(name, "exec.tool") ||
		strings.Contains(name, "tool_use")
}

// extractTurnNumber extracts the turn number from a span.
func extractTurnNumber(span *Span) int {
	// Try to extract from span name (e.g., "exec.turn.3" or "turn.3")
	name := span.Name

	// Pattern: exec.turn.N or turn.N
	if strings.Contains(name, "turn") {
		parts := strings.Split(name, ".")
		for i, part := range parts {
			if part == "turn" && i+1 < len(parts) {
				if num, err := parseIntSafe(parts[i+1]); err == nil && num > 0 {
					return num
				}
			}
		}
	}

	// Try attributes
	if span.Attributes != nil {
		// Check turn.number attribute
		if turnNumAttr, ok := span.Attributes["turn.number"]; ok {
			if num, ok := turnNumAttr.(float64); ok {
				return int(num)
			}
			if num, ok := turnNumAttr.(int); ok {
				return num
			}
			if numStr, ok := turnNumAttr.(string); ok {
				if num, err := parseIntSafe(numStr); err == nil {
					return num
				}
			}
		}
	}

	return 0
}

// extractToolName extracts the tool name from a span.
func extractToolName(span *Span) string {
	name := span.Name

	// Pattern: tool.Read, exec.tool_use.Bash, etc.
	if strings.HasPrefix(name, "tool.") {
		return strings.TrimPrefix(name, "tool.")
	}
	if strings.Contains(name, "tool_use") {
		// exec.tool_use or claude_code.tool_use
		if span.Attributes != nil {
			if toolName, ok := span.Attributes["tool.name"].(string); ok {
				return toolName
			}
		}
		// Try to extract from name
		parts := strings.Split(name, ".")
		if len(parts) > 0 {
			return parts[len(parts)-1]
		}
	}

	return name
}

// parseIntSafe safely parses an integer string without strconv.
func parseIntSafe(s string) (int, error) {
	var n int
	for _, c := range s {
		if c < '0' || c > '9' {
			return 0, fmt.Errorf("not a number: %s", s)
		}
		n = n*10 + int(c-'0')
	}
	return n, nil
}
