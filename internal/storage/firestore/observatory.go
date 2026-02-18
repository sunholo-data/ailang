package firestore

import (
	"context"
	"fmt"
	"time"

	"google.golang.org/api/iterator"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	obs "github.com/sunholo/ailang/internal/observatory"
)

// Compile-time check that ObservatoryStore implements observatory.Backend.
var _ obs.Backend = (*ObservatoryStore)(nil)

// Observatory Firestore collection names (prefixed with obs_ to avoid collisions).
const (
	collObsWorkspaces       = "obs_workspaces"
	collObsTasks            = "obs_tasks"
	collObsAgentAssignments = "obs_agent_assignments"
	collObsSpans            = "obs_spans"
	collObsSpanEvents       = "obs_span_events"
	collObsMessages         = "obs_messages"
	collObsMetrics          = "obs_metrics"
	collObsSessions         = "obs_sessions"
	collObsSessionTools     = "obs_session_tools"
	collObsChatMessages     = "obs_chat_messages"
	collObsChains           = "obs_chains"
	collObsChainStages      = "obs_chain_stages"
)

// ObservatoryStore implements observatory.Backend backed by Firestore.
type ObservatoryStore struct {
	client *Client
}

// NewObservatoryStore creates a new Firestore-backed observatory store.
func NewObservatoryStore(client *Client) *ObservatoryStore {
	return &ObservatoryStore{client: client}
}

func (s *ObservatoryStore) Close() error {
	return s.client.Close()
}

// --- Workspace operations ---

func (s *ObservatoryStore) CreateWorkspace(ctx context.Context, w *obs.Workspace) error {
	if w.CreatedAt.IsZero() {
		w.CreatedAt = time.Now()
	}
	w.UpdatedAt = w.CreatedAt
	_, err := s.client.Doc(collObsWorkspaces, w.ID).Set(ctx, workspaceToMap(w))
	return err
}

func (s *ObservatoryStore) GetWorkspace(ctx context.Context, id string) (*obs.Workspace, error) {
	doc, err := s.client.Doc(collObsWorkspaces, id).Get(ctx)
	if err != nil {
		if status.Code(err) == codes.NotFound {
			return nil, fmt.Errorf("workspace not found: %s", id)
		}
		return nil, err
	}
	return mapToWorkspace(doc.Data()), nil
}

func (s *ObservatoryStore) ListWorkspaces(ctx context.Context) ([]*obs.Workspace, error) {
	iter := s.client.Collection(collObsWorkspaces).Documents(ctx)
	defer iter.Stop()

	var result []*obs.Workspace
	for {
		doc, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, err
		}
		result = append(result, mapToWorkspace(doc.Data()))
	}
	return result, nil
}

func (s *ObservatoryStore) UpdateWorkspace(ctx context.Context, w *obs.Workspace) error {
	w.UpdatedAt = time.Now()
	_, err := s.client.Doc(collObsWorkspaces, w.ID).Set(ctx, workspaceToMap(w))
	return err
}

func (s *ObservatoryStore) DeleteWorkspace(ctx context.Context, id string) error {
	_, err := s.client.Doc(collObsWorkspaces, id).Delete(ctx)
	return err
}

func (s *ObservatoryStore) GetWorkspaceStats(ctx context.Context, id string) (*obs.WorkspaceStats, error) {
	ws, err := s.GetWorkspace(ctx, id)
	if err != nil {
		return nil, err
	}

	// Count tasks in this workspace
	taskIter := s.client.Collection(collObsTasks).
		Where("workspace_id", "==", id).
		Documents(ctx)
	defer taskIter.Stop()

	stats := &obs.WorkspaceStats{
		ID:   ws.ID,
		Name: ws.Name,
		Path: ws.Path,
	}

	agents := make(map[string]bool)
	for {
		doc, err := taskIter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, err
		}
		data := doc.Data()
		stats.TaskCount++
		stats.TotalCost += getFloat64(data, "total_cost_usd")
		stats.TotalTokens += getInt64(data, "total_tokens_in") + getInt64(data, "total_tokens_out")
		if getString(data, "status") == "completed" {
			// Track for success rate
		}
	}

	// Count unique agents via assignments
	aaIter := s.client.Collection(collObsAgentAssignments).Documents(ctx)
	defer aaIter.Stop()
	for {
		doc, err := aaIter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			break
		}
		agents[getString(doc.Data(), "agent_id")] = true
	}
	stats.UniqueAgents = len(agents)

	return stats, nil
}

// --- Workspace conversion helpers ---

func workspaceToMap(w *obs.Workspace) map[string]interface{} {
	return map[string]interface{}{
		"id":         w.ID,
		"name":       w.Name,
		"path":       w.Path,
		"git_remote": w.GitRemote,
		"created_at": timeToFirestore(w.CreatedAt),
		"updated_at": timeToFirestore(w.UpdatedAt),
	}
}

func mapToWorkspace(data map[string]interface{}) *obs.Workspace {
	return &obs.Workspace{
		ID:        getString(data, "id"),
		Name:      getString(data, "name"),
		Path:      getString(data, "path"),
		GitRemote: getString(data, "git_remote"),
		CreatedAt: snapshotToTime(data, "created_at"),
		UpdatedAt: snapshotToTime(data, "updated_at"),
	}
}
