package firestore

import (
	"context"
	"fmt"

	"cloud.google.com/go/firestore"
	"google.golang.org/api/iterator"

	obs "github.com/sunholo-data/ailang/internal/observatory"
)

// ListChains filters before selecting a page. Agent membership is held by stages,
// so that path scans candidates in canonical order and stops at a full page.
func (s *ObservatoryStore) ListChains(ctx context.Context, opts obs.ChainListOptions) ([]*obs.ChainSummary, error) {
	if opts.Offset < 0 {
		return nil, fmt.Errorf("chain offset must be non-negative")
	}
	if opts.Limit <= 0 {
		opts.Limit = 50
	}
	var agentChains map[string]bool
	if opts.AgentID != "" {
		var err error
		agentChains, err = s.chainIDsForAgent(ctx, opts.AgentID)
		if err != nil {
			return nil, err
		}
		if len(agentChains) == 0 {
			return []*obs.ChainSummary{}, nil
		}
	}
	q := s.client.Collection(collObsChains).Query
	if opts.Status != "" {
		q = q.Where("status", "==", string(opts.Status))
	}
	if opts.SourceType != "" {
		q = q.Where("source_type", "==", opts.SourceType)
	}
	if opts.WorkspaceID != "" {
		q = q.Where("workspace_id", "==", opts.WorkspaceID)
	}
	if opts.GitHubRepo != "" {
		q = q.Where("github_repo", "==", opts.GitHubRepo)
	}
	if opts.CreatedAfter != nil {
		q = q.Where("created_at", ">", timeToFirestore(*opts.CreatedAfter))
	}
	q = q.OrderBy("created_at", firestore.Desc).OrderBy(firestore.DocumentID, firestore.Desc)
	if opts.AgentID == "" {
		q = q.Offset(opts.Offset).Limit(opts.Limit)
	}

	iter := q.Documents(ctx)
	defer iter.Stop()

	result := make([]*obs.ChainSummary, 0)
	matched := 0
	for {
		doc, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, err
		}
		data := doc.Data()
		if agentChains != nil {
			if !agentChains[getString(data, "id")] {
				continue
			}
			matched++
			if matched <= opts.Offset {
				continue
			}
		}
		result = append(result, &obs.ChainSummary{
			ID:                getString(data, "id"),
			SourceType:        getString(data, "source_type"),
			SourceRef:         getString(data, "source_ref"),
			GitHubRepo:        getString(data, "github_repo"),
			GitHubIssueNumber: getInt(data, "github_issue_number"),
			Status:            obs.ChainStatus(getString(data, "status")),
			CurrentStage:      getInt(data, "current_stage"),
			TotalCost:         getFloat64(data, "total_cost"),
			TotalTokens:       getInt(data, "total_tokens"),
			TotalTurns:        getInt(data, "total_turns"),
			StagesCompleted:   getInt(data, "stages_completed"),
			CreatedAt:         snapshotToTime(data, "created_at"),
			CompletedAt:       snapshotToTimePtr(data, "completed_at"),
		})
		if len(result) == opts.Limit {
			break
		}
	}

	return result, nil
}
