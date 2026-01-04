// Package observatory provides a unified observability platform for AILANG.
package observatory

import (
	"context"
	"fmt"
)

// GCPTraceBackend implements Backend using Google Cloud Trace.
// This is a read-only backend for querying traces stored in GCP.
type GCPTraceBackend struct {
	projectID string
	// client will be added when GCP client is integrated
}

// GCPConfig contains configuration for GCP Trace backend.
type GCPConfig struct {
	ProjectID       string
	CredentialsPath string
}

// NewGCPTraceBackend creates a new GCP Trace backend.
func NewGCPTraceBackend(config GCPConfig) (*GCPTraceBackend, error) {
	if config.ProjectID == "" {
		return nil, fmt.Errorf("project ID is required")
	}
	return &GCPTraceBackend{
		projectID: config.ProjectID,
	}, nil
}

// Close closes the GCP client.
func (b *GCPTraceBackend) Close() error {
	// Close GCP client when implemented
	return nil
}

// errNotSupported returns an error for operations not supported by this backend.
func errNotSupported(op string) error {
	return fmt.Errorf("operation %s not supported by GCP Trace backend (read-only)", op)
}

// ===== Workspace Operations (not supported) =====

func (b *GCPTraceBackend) CreateWorkspace(ctx context.Context, w *Workspace) error {
	return errNotSupported("CreateWorkspace")
}

func (b *GCPTraceBackend) GetWorkspace(ctx context.Context, id string) (*Workspace, error) {
	return nil, errNotSupported("GetWorkspace")
}

func (b *GCPTraceBackend) ListWorkspaces(ctx context.Context) ([]*Workspace, error) {
	return nil, errNotSupported("ListWorkspaces")
}

func (b *GCPTraceBackend) UpdateWorkspace(ctx context.Context, w *Workspace) error {
	return errNotSupported("UpdateWorkspace")
}

func (b *GCPTraceBackend) DeleteWorkspace(ctx context.Context, id string) error {
	return errNotSupported("DeleteWorkspace")
}

func (b *GCPTraceBackend) GetWorkspaceStats(ctx context.Context, id string) (*WorkspaceStats, error) {
	return nil, errNotSupported("GetWorkspaceStats")
}

// ===== Task Operations (not supported) =====

func (b *GCPTraceBackend) CreateTask(ctx context.Context, t *Task) error {
	return errNotSupported("CreateTask")
}

func (b *GCPTraceBackend) GetTask(ctx context.Context, id string) (*Task, error) {
	return nil, errNotSupported("GetTask")
}

func (b *GCPTraceBackend) ListTasks(ctx context.Context, opts TaskListOptions) ([]*Task, error) {
	return nil, errNotSupported("ListTasks")
}

func (b *GCPTraceBackend) UpdateTask(ctx context.Context, t *Task) error {
	return errNotSupported("UpdateTask")
}

func (b *GCPTraceBackend) DeleteTask(ctx context.Context, id string) error {
	return errNotSupported("DeleteTask")
}

// ===== Agent Assignment Operations (not supported) =====

func (b *GCPTraceBackend) CreateAgentAssignment(ctx context.Context, a *AgentAssignment) error {
	return errNotSupported("CreateAgentAssignment")
}

func (b *GCPTraceBackend) GetAgentAssignment(ctx context.Context, id string) (*AgentAssignment, error) {
	return nil, errNotSupported("GetAgentAssignment")
}

func (b *GCPTraceBackend) ListAgentAssignments(ctx context.Context, taskID string) ([]*AgentAssignment, error) {
	return nil, errNotSupported("ListAgentAssignments")
}

func (b *GCPTraceBackend) UpdateAgentAssignment(ctx context.Context, a *AgentAssignment) error {
	return errNotSupported("UpdateAgentAssignment")
}

func (b *GCPTraceBackend) DeleteAgentAssignment(ctx context.Context, id string) error {
	return errNotSupported("DeleteAgentAssignment")
}

func (b *GCPTraceBackend) GetAgentStats(ctx context.Context, agentID string) (*AgentStats, error) {
	return nil, errNotSupported("GetAgentStats")
}

// ===== Span Operations (read-only supported) =====

func (b *GCPTraceBackend) CreateSpan(ctx context.Context, span *Span) error {
	return errNotSupported("CreateSpan")
}

func (b *GCPTraceBackend) GetSpan(ctx context.Context, id string) (*Span, error) {
	// TODO: Implement GCP Trace API query
	return nil, fmt.Errorf("GCP Trace GetSpan not yet implemented")
}

func (b *GCPTraceBackend) ListSpans(ctx context.Context, opts SpanListOptions) ([]*Span, error) {
	// TODO: Implement GCP Trace API query
	return nil, fmt.Errorf("GCP Trace ListSpans not yet implemented")
}

func (b *GCPTraceBackend) UpdateSpan(ctx context.Context, span *Span) error {
	return errNotSupported("UpdateSpan")
}

func (b *GCPTraceBackend) DeleteSpan(ctx context.Context, id string) error {
	return errNotSupported("DeleteSpan")
}

func (b *GCPTraceBackend) GetTrace(ctx context.Context, traceID string) (*Trace, error) {
	// TODO: Implement GCP Trace API query
	return nil, fmt.Errorf("GCP Trace GetTrace not yet implemented")
}

func (b *GCPTraceBackend) ListTraces(ctx context.Context, opts TraceQuery) ([]*TraceSummary, error) {
	// TODO: Implement GCP Trace API query
	return nil, fmt.Errorf("GCP Trace ListTraces not yet implemented")
}

// ===== Span Event Operations (not supported) =====

func (b *GCPTraceBackend) CreateSpanEvent(ctx context.Context, e *SpanEvent) error {
	return errNotSupported("CreateSpanEvent")
}

func (b *GCPTraceBackend) GetSpanEvents(ctx context.Context, spanID string) ([]SpanEvent, error) {
	return nil, errNotSupported("GetSpanEvents")
}

func (b *GCPTraceBackend) DeleteSpanEvent(ctx context.Context, id int64) error {
	return errNotSupported("DeleteSpanEvent")
}

// ===== Message Operations (not supported) =====

func (b *GCPTraceBackend) CreateMessage(ctx context.Context, m *Message) error {
	return errNotSupported("CreateMessage")
}

func (b *GCPTraceBackend) GetMessage(ctx context.Context, id string) (*Message, error) {
	return nil, errNotSupported("GetMessage")
}

func (b *GCPTraceBackend) ListMessages(ctx context.Context, opts MessageListOptions) ([]*Message, error) {
	return nil, errNotSupported("ListMessages")
}

func (b *GCPTraceBackend) UpdateMessage(ctx context.Context, m *Message) error {
	return errNotSupported("UpdateMessage")
}

func (b *GCPTraceBackend) DeleteMessage(ctx context.Context, id string) error {
	return errNotSupported("DeleteMessage")
}

func (b *GCPTraceBackend) MarkMessageRead(ctx context.Context, id string) error {
	return errNotSupported("MarkMessageRead")
}

func (b *GCPTraceBackend) MarkMessageArchived(ctx context.Context, id string) error {
	return errNotSupported("MarkMessageArchived")
}

// ===== Aggregate Operations (not supported) =====

func (b *GCPTraceBackend) GetMetricsSummary(ctx context.Context) (*MetricsSummary, error) {
	return nil, errNotSupported("GetMetricsSummary")
}

func (b *GCPTraceBackend) GetProviderComparison(ctx context.Context) ([]*ProviderComparison, error) {
	return nil, errNotSupported("GetProviderComparison")
}

func (b *GCPTraceBackend) GetTaskTimeline(ctx context.Context, taskID string) ([]*TaskTimeline, error) {
	return nil, errNotSupported("GetTaskTimeline")
}

// Ensure GCPTraceBackend implements Backend
var _ Backend = (*GCPTraceBackend)(nil)
