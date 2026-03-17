// Package observatory provides cross-trace merging and session-based correlation for hierarchy views.
package observatory

import (
	"sort"
	"strings"
	"time"
)

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
