// Package observatory provides a unified observability platform for AILANG.
package observatory

import (
	"context"
	"fmt"
	"sort"
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
