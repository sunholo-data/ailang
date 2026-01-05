// Package observatory provides a unified observability platform for AILANG.
package observatory

import (
	"context"
	"fmt"
	"log"
)

// CompositeBackend implements Backend with write-local, read-remote pattern.
// - Writes always go to local SQLite for durability
// - Reads can query remote backends (GCP Trace, Jaeger) for historical traces
// - Merges results from multiple sources for comprehensive views
type CompositeBackend struct {
	local   Backend   // Primary backend for writes (SQLite)
	remotes []Backend // Remote backends for trace reads
}

// CompositeConfig contains configuration for composite backend.
type CompositeConfig struct {
	Local   Backend   // Required: local storage backend
	Remotes []Backend // Optional: remote backends for trace queries
}

// NewCompositeBackend creates a new composite backend.
func NewCompositeBackend(config CompositeConfig) (*CompositeBackend, error) {
	if config.Local == nil {
		return nil, fmt.Errorf("local backend is required")
	}
	return &CompositeBackend{
		local:   config.Local,
		remotes: config.Remotes,
	}, nil
}

// Close closes all backends.
func (b *CompositeBackend) Close() error {
	var errs []error
	if err := b.local.Close(); err != nil {
		errs = append(errs, fmt.Errorf("local: %w", err))
	}
	for i, remote := range b.remotes {
		if err := remote.Close(); err != nil {
			errs = append(errs, fmt.Errorf("remote[%d]: %w", i, err))
		}
	}
	if len(errs) > 0 {
		return fmt.Errorf("close errors: %v", errs)
	}
	return nil
}

// ===== Workspace Operations (local only) =====

func (b *CompositeBackend) CreateWorkspace(ctx context.Context, w *Workspace) error {
	return b.local.CreateWorkspace(ctx, w)
}

func (b *CompositeBackend) GetWorkspace(ctx context.Context, id string) (*Workspace, error) {
	return b.local.GetWorkspace(ctx, id)
}

func (b *CompositeBackend) ListWorkspaces(ctx context.Context) ([]*Workspace, error) {
	return b.local.ListWorkspaces(ctx)
}

func (b *CompositeBackend) UpdateWorkspace(ctx context.Context, w *Workspace) error {
	return b.local.UpdateWorkspace(ctx, w)
}

func (b *CompositeBackend) DeleteWorkspace(ctx context.Context, id string) error {
	return b.local.DeleteWorkspace(ctx, id)
}

func (b *CompositeBackend) GetWorkspaceStats(ctx context.Context, id string) (*WorkspaceStats, error) {
	return b.local.GetWorkspaceStats(ctx, id)
}

// ===== Task Operations (local only) =====

func (b *CompositeBackend) CreateTask(ctx context.Context, t *Task) error {
	return b.local.CreateTask(ctx, t)
}

func (b *CompositeBackend) GetTask(ctx context.Context, id string) (*Task, error) {
	return b.local.GetTask(ctx, id)
}

func (b *CompositeBackend) ListTasks(ctx context.Context, opts TaskListOptions) ([]*Task, error) {
	return b.local.ListTasks(ctx, opts)
}

func (b *CompositeBackend) UpdateTask(ctx context.Context, t *Task) error {
	return b.local.UpdateTask(ctx, t)
}

func (b *CompositeBackend) DeleteTask(ctx context.Context, id string) error {
	return b.local.DeleteTask(ctx, id)
}

// ===== Agent Assignment Operations (local only) =====

func (b *CompositeBackend) CreateAgentAssignment(ctx context.Context, a *AgentAssignment) error {
	return b.local.CreateAgentAssignment(ctx, a)
}

func (b *CompositeBackend) GetAgentAssignment(ctx context.Context, id string) (*AgentAssignment, error) {
	return b.local.GetAgentAssignment(ctx, id)
}

func (b *CompositeBackend) ListAgentAssignments(ctx context.Context, taskID string) ([]*AgentAssignment, error) {
	return b.local.ListAgentAssignments(ctx, taskID)
}

func (b *CompositeBackend) UpdateAgentAssignment(ctx context.Context, a *AgentAssignment) error {
	return b.local.UpdateAgentAssignment(ctx, a)
}

func (b *CompositeBackend) DeleteAgentAssignment(ctx context.Context, id string) error {
	return b.local.DeleteAgentAssignment(ctx, id)
}

func (b *CompositeBackend) GetAgentStats(ctx context.Context, agentID string) (*AgentStats, error) {
	return b.local.GetAgentStats(ctx, agentID)
}

// ===== Span Operations (write local, read merged) =====

func (b *CompositeBackend) CreateSpan(ctx context.Context, span *Span) error {
	// Always write to local
	return b.local.CreateSpan(ctx, span)
}

func (b *CompositeBackend) GetSpan(ctx context.Context, id string) (*Span, error) {
	// Try local first
	span, err := b.local.GetSpan(ctx, id)
	if err == nil && span != nil {
		return span, nil
	}

	// Try remotes
	for _, remote := range b.remotes {
		span, err := remote.GetSpan(ctx, id)
		if err == nil && span != nil {
			return span, nil
		}
	}

	return nil, fmt.Errorf("span not found: %s", id)
}

func (b *CompositeBackend) ListSpans(ctx context.Context, opts SpanListOptions) ([]*Span, error) {
	// Get local spans
	spans, err := b.local.ListSpans(ctx, opts)
	if err != nil {
		return nil, fmt.Errorf("local ListSpans: %w", err)
	}

	// Merge with remote spans (if enabled)
	if len(b.remotes) > 0 {
		seen := make(map[string]bool)
		for _, s := range spans {
			seen[s.ID] = true
		}

		for _, remote := range b.remotes {
			remoteSpans, err := remote.ListSpans(ctx, opts)
			if err != nil {
				// Log but don't fail - remote may be unavailable
				continue
			}
			for _, s := range remoteSpans {
				if !seen[s.ID] {
					spans = append(spans, s)
					seen[s.ID] = true
				}
			}
		}
	}

	return spans, nil
}

func (b *CompositeBackend) UpdateSpan(ctx context.Context, span *Span) error {
	return b.local.UpdateSpan(ctx, span)
}

func (b *CompositeBackend) UpdateSpanLinks(ctx context.Context, spanID, taskID, assignmentID string) error {
	return b.local.UpdateSpanLinks(ctx, spanID, taskID, assignmentID)
}

func (b *CompositeBackend) DeleteSpan(ctx context.Context, id string) error {
	return b.local.DeleteSpan(ctx, id)
}

func (b *CompositeBackend) GetTrace(ctx context.Context, traceID string) (*Trace, error) {
	// Try local first
	trace, err := b.local.GetTrace(ctx, traceID)
	if err == nil && trace != nil {
		return trace, nil
	}

	// Try remotes
	for _, remote := range b.remotes {
		trace, err := remote.GetTrace(ctx, traceID)
		if err == nil && trace != nil {
			return trace, nil
		}
	}

	return nil, fmt.Errorf("trace not found: %s", traceID)
}

func (b *CompositeBackend) ListTraces(ctx context.Context, opts TraceQuery) ([]*TraceSummary, error) {
	// Get local traces
	traces, err := b.local.ListTraces(ctx, opts)
	if err != nil {
		return nil, fmt.Errorf("local ListTraces: %w", err)
	}

	// Merge with remote traces
	if len(b.remotes) > 0 {
		seen := make(map[string]bool)
		for _, t := range traces {
			seen[t.TraceID] = true
		}

		for i, remote := range b.remotes {
			remoteTraces, err := remote.ListTraces(ctx, opts)
			if err != nil {
				log.Printf("composite: remote[%d] ListTraces error: %v", i, err)
				continue
			}
			log.Printf("composite: remote[%d] returned %d traces", i, len(remoteTraces))
			for _, t := range remoteTraces {
				if !seen[t.TraceID] {
					traces = append(traces, t)
					seen[t.TraceID] = true
				}
			}
		}
	}

	return traces, nil
}

// ===== Span Event Operations (local only) =====

func (b *CompositeBackend) CreateSpanEvent(ctx context.Context, e *SpanEvent) error {
	return b.local.CreateSpanEvent(ctx, e)
}

func (b *CompositeBackend) GetSpanEvents(ctx context.Context, spanID string) ([]SpanEvent, error) {
	return b.local.GetSpanEvents(ctx, spanID)
}

func (b *CompositeBackend) DeleteSpanEvent(ctx context.Context, id int64) error {
	return b.local.DeleteSpanEvent(ctx, id)
}

// ===== Message Operations (local only) =====

func (b *CompositeBackend) CreateMessage(ctx context.Context, m *Message) error {
	return b.local.CreateMessage(ctx, m)
}

func (b *CompositeBackend) GetMessage(ctx context.Context, id string) (*Message, error) {
	return b.local.GetMessage(ctx, id)
}

func (b *CompositeBackend) ListMessages(ctx context.Context, opts MessageListOptions) ([]*Message, error) {
	return b.local.ListMessages(ctx, opts)
}

func (b *CompositeBackend) UpdateMessage(ctx context.Context, m *Message) error {
	return b.local.UpdateMessage(ctx, m)
}

func (b *CompositeBackend) DeleteMessage(ctx context.Context, id string) error {
	return b.local.DeleteMessage(ctx, id)
}

func (b *CompositeBackend) MarkMessageRead(ctx context.Context, id string) error {
	return b.local.MarkMessageRead(ctx, id)
}

func (b *CompositeBackend) MarkMessageArchived(ctx context.Context, id string) error {
	return b.local.MarkMessageArchived(ctx, id)
}

// ===== Aggregate Operations (local only) =====

func (b *CompositeBackend) GetMetricsSummary(ctx context.Context) (*MetricsSummary, error) {
	return b.local.GetMetricsSummary(ctx)
}

func (b *CompositeBackend) GetProviderComparison(ctx context.Context) ([]*ProviderComparison, error) {
	return b.local.GetProviderComparison(ctx)
}

func (b *CompositeBackend) GetTaskTimeline(ctx context.Context, taskID string) ([]*TaskTimeline, error) {
	return b.local.GetTaskTimeline(ctx, taskID)
}

// Ensure CompositeBackend implements Backend
var _ Backend = (*CompositeBackend)(nil)
