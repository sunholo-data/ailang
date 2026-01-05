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

	// Get all spans for this agent assignment
	spans, err := backend.ListSpans(ctx, SpanListOptions{
		AgentAssignmentID: agent.ID,
		Limit:             1000, // Reasonable limit
	})
	if err != nil {
		return nil, fmt.Errorf("list spans: %w", err)
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

	// Build flat list for easy rendering, sorted by start time
	flatSpans := make([]*SpanNode, 0, len(spans))
	for _, span := range spans {
		flatSpans = append(flatSpans, spanIndex[span.ID])
	}
	sort.Slice(flatSpans, func(i, j int) bool {
		return flatSpans[i].Span.StartTime.Before(flatSpans[j].Span.StartTime)
	})

	// Sort children of each node by start time
	for _, node := range flatSpans {
		sortChildrenByTime(node)
	}

	traceHierarchy := &TraceHierarchy{
		TraceID: traceID,
		Spans:   flatSpans,
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
