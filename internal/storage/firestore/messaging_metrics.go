package firestore

import (
	"context"
	"fmt"
	"time"

	"cloud.google.com/go/firestore"
	"google.golang.org/api/iterator"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/sunholo/ailang/internal/messaging"
)

// --- Metrics & Analytics ---

func (s *MessagingStore) RecordMetrics(threadID, agentID string, stats *messaging.MessageExecutionStats) error {
	ctx := context.Background()
	id := fmt.Sprintf("metric_%d_%s", time.Now().UnixMilli(), generateShortID())
	_, err := s.client.Doc(collMetrics, id).Set(ctx, metricsToMap(threadID, agentID, stats))
	return err
}

func (s *MessagingStore) GetMetrics(scopeType, scopeID string) (*messaging.AggregatedMetrics, error) {
	ctx := context.Background()
	q := s.client.Collection(collMetrics).
		Where(scopeType+"_id", "==", scopeID)

	return s.aggregateMetrics(ctx, q, scopeType, scopeID)
}

func (s *MessagingStore) GetGlobalMetrics() (*messaging.AggregatedMetrics, error) {
	ctx := context.Background()
	q := s.client.Collection(collMetrics).Query
	return s.aggregateMetrics(ctx, q, "global", "all")
}

func (s *MessagingStore) GetAgentMetrics(agentID string) (*messaging.AggregatedMetrics, error) {
	return s.GetMetrics("agent", agentID)
}

func (s *MessagingStore) GetThreadMetrics(threadID string) (*messaging.AggregatedMetrics, error) {
	return s.GetMetrics("thread", threadID)
}

func (s *MessagingStore) aggregateMetrics(ctx context.Context, q firestore.Query, scopeType, scopeID string) (*messaging.AggregatedMetrics, error) {
	iter := q.Documents(ctx)
	defer iter.Stop()

	agg := &messaging.AggregatedMetrics{
		ScopeType: scopeType,
		ScopeID:   scopeID,
	}

	for {
		doc, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, err
		}
		data := doc.Data()
		agg.TotalRuns++
		agg.TotalTokens += getInt(data, "input_tokens") + getInt(data, "output_tokens")
		agg.TotalCost += float64(getInt(data, "cost_cents")) / 100.0
		agg.TotalDuration += getInt(data, "duration_ms")
	}

	if agg.TotalRuns > 0 {
		agg.AvgTokens = float64(agg.TotalTokens) / float64(agg.TotalRuns)
		agg.AvgCost = agg.TotalCost / float64(agg.TotalRuns)
		agg.AvgDuration = float64(agg.TotalDuration) / float64(agg.TotalRuns)
	}

	return agg, nil
}

func (s *MessagingStore) GetMetricsTrends(scopeType, scopeID, period string, limit int) ([]map[string]interface{}, error) {
	ctx := context.Background()
	q := s.client.Collection(collMetrics).Query

	if scopeType != "global" && scopeID != "" {
		q = q.Where(scopeType+"_id", "==", scopeID)
	}
	q = q.OrderBy("created_at", firestore.Desc)
	if limit > 0 {
		q = q.Limit(limit)
	}

	iter := q.Documents(ctx)
	defer iter.Stop()

	var trends []map[string]interface{}
	for {
		doc, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, err
		}
		trends = append(trends, doc.Data())
	}
	return trends, nil
}

// --- Execution Hierarchy ---

func (s *MessagingStore) GetAggregatedExecutionStats() (*messaging.ExecutionStats, error) {
	ctx := context.Background()
	iter := s.client.Collection(collMetrics).Documents(ctx)
	defer iter.Stop()

	stats := &messaging.ExecutionStats{}
	for {
		doc, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, err
		}
		data := doc.Data()
		stats.TotalExecutions++
		stats.TotalDurationMS += int64(getInt(data, "duration_ms"))
		stats.TotalCost += float64(getInt(data, "cost_cents")) / 100.0
		stats.TotalInputTokens += getInt(data, "input_tokens")
		stats.TotalOutputTokens += getInt(data, "output_tokens")
	}
	// Note: Firestore doesn't store success/failure status in metrics;
	// this would need correlation with task records.
	stats.SuccessfulExecutions = stats.TotalExecutions
	return stats, nil
}

func (s *MessagingStore) GetExecutionStatsByThread(threadID string) (*messaging.ExecutionStats, error) {
	ctx := context.Background()
	iter := s.client.Collection(collMetrics).
		Where("thread_id", "==", threadID).
		Documents(ctx)
	defer iter.Stop()

	stats := &messaging.ExecutionStats{}
	for {
		doc, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, err
		}
		data := doc.Data()
		stats.TotalExecutions++
		stats.TotalDurationMS += int64(getInt(data, "duration_ms"))
		stats.TotalCost += float64(getInt(data, "cost_cents")) / 100.0
		stats.TotalInputTokens += getInt(data, "input_tokens")
		stats.TotalOutputTokens += getInt(data, "output_tokens")
	}
	stats.SuccessfulExecutions = stats.TotalExecutions
	return stats, nil
}

func (s *MessagingStore) GetHierarchy() (*messaging.HierarchyResponse, error) {
	ctx := context.Background()

	// Build root node from agents
	agentIter := s.client.Collection(collAgents).Documents(ctx)
	defer agentIter.Stop()

	var agentNodes []messaging.HierarchyNode
	var totalAgents, activeAgents int

	for {
		doc, err := agentIter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, err
		}
		data := doc.Data()
		totalAgents++
		agentStatus := getString(data, "status")
		if agentStatus == "active" {
			activeAgents++
		}

		agentNodes = append(agentNodes, messaging.HierarchyNode{
			Type:   "agent",
			ID:     getString(data, "agent_id"),
			Label:  getString(data, "label"),
			Status: agentStatus,
		})
	}

	root := messaging.HierarchyNode{
		Type:     "root",
		ID:       "root",
		Label:    "AILANG System",
		Children: agentNodes,
	}

	return &messaging.HierarchyResponse{
		Root: root,
		Aggregate: messaging.AggregateStats{
			TotalAgents:  totalAgents,
			ActiveAgents: activeAgents,
			IdleAgents:   totalAgents - activeAgents,
		},
	}, nil
}

func (s *MessagingStore) GetAgentStats(agentID string) (*messaging.AgentStats, error) {
	doc, err := s.client.Doc(collAgents, agentID).Get(context.Background())
	if err != nil {
		if status.Code(err) == codes.NotFound {
			return &messaging.AgentStats{AgentID: agentID, Status: "unknown"}, nil
		}
		return nil, err
	}
	data := doc.Data()
	return &messaging.AgentStats{
		AgentID: agentID,
		Status:  getString(data, "status"),
	}, nil
}

func (s *MessagingStore) GetKnownAgents() ([]messaging.AgentInfo, error) {
	ctx := context.Background()
	iter := s.client.Collection(collAgents).Documents(ctx)
	defer iter.Stop()

	var agents []messaging.AgentInfo
	for {
		doc, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, err
		}
		agents = append(agents, mapToAgentInfo(doc.Data()))
	}
	return agents, nil
}

// --- Agent Registration ---

func (s *MessagingStore) RegisterAgent(agentID, label, agentStatus string) error {
	ctx := context.Background()
	_, err := s.client.Doc(collAgents, agentID).Set(ctx, agentInfoToMap(agentID, label, agentStatus))
	return err
}

func (s *MessagingStore) UpdateAgentStatus(agentID, agentStatus string) error {
	_, err := s.client.Doc(collAgents, agentID).Update(context.Background(), []firestore.Update{
		{Path: "status", Value: agentStatus},
		{Path: "updated_at", Value: time.Now()},
	})
	return err
}

func (s *MessagingStore) RecordAgentInstance(agentID, instanceID string) error {
	_, err := s.client.Doc(collAgents, agentID).Update(context.Background(), []firestore.Update{
		{Path: "last_instance_id", Value: instanceID},
		{Path: "updated_at", Value: time.Now()},
	})
	return err
}
