package observatory

import (
	"context"
	"time"
)

// Chain operations - GCP backend doesn't store chains (local-only feature)

func (b *GCPTraceBackend) CreateChain(ctx context.Context, req *ChainCreateRequest) (*ExecutionChain, error) {
	return nil, nil
}

func (b *GCPTraceBackend) GetChain(ctx context.Context, id string, opts ChainReadOptions) (*ExecutionChain, error) {
	return nil, nil
}

func (b *GCPTraceBackend) GetChainByMessageID(ctx context.Context, messageID string) (*ExecutionChain, error) {
	return nil, nil
}

func (b *GCPTraceBackend) GetChainByTaskID(ctx context.Context, taskID string) (*ExecutionChain, error) {
	return nil, nil
}

func (b *GCPTraceBackend) ListChains(ctx context.Context, opts ChainListOptions) ([]*ChainSummary, error) {
	return nil, nil
}

func (b *GCPTraceBackend) UpdateChainStatus(ctx context.Context, chainID string, status ChainStatus) error {
	return nil
}

func (b *GCPTraceBackend) CreateStage(ctx context.Context, req *StageCreateRequest) (*ChainStage, error) {
	return nil, nil
}

func (b *GCPTraceBackend) GetStage(ctx context.Context, id string) (*ChainStage, error) {
	return nil, nil
}

func (b *GCPTraceBackend) GetChainStages(ctx context.Context, chainID string, opts ChainReadOptions) ([]*ChainStage, error) {
	return nil, nil
}

func (b *GCPTraceBackend) UpdateStageStatus(ctx context.Context, stageID string, status ChainStageStatus) error {
	return nil
}

func (b *GCPTraceBackend) UpdateStageSession(ctx context.Context, stageID, sessionID string) error {
	return nil
}

func (b *GCPTraceBackend) UpdateStageMetrics(ctx context.Context, stageID string, cost float64, tokensIn, tokensOut, turns, toolCalls int, durationMs int64, costProvenance string) error {
	return nil
}

func (b *GCPTraceBackend) UpdateStageApproval(ctx context.Context, stageID string, status ApprovalStatus, approvalType ApprovalType, feedback string) error {
	return nil
}

func (b *GCPTraceBackend) UpdateStageError(ctx context.Context, stageID, errorMessage string) error {
	return nil
}

// UpdateStageEvalAssessment is a no-op here, like every other chain-stage write on
// this backend: GCP Cloud Trace holds spans, not the chain hierarchy.
func (b *GCPTraceBackend) UpdateStageEvalAssessment(ctx context.Context, stageID string, assessment *EvalAssessment) error {
	return nil
}

func (b *GCPTraceBackend) GetSpansByStageID(ctx context.Context, stageID string) ([]*Span, error) {
	return nil, nil
}

func (b *GCPTraceBackend) LinkSpanToChain(ctx context.Context, spanID, chainID, stageID string) error {
	return nil
}

func (b *GCPTraceBackend) ListPendingApprovals(ctx context.Context, limit int) ([]*PendingApprovalInfo, error) {
	return nil, nil
}

func (b *GCPTraceBackend) GetSession(ctx context.Context, sessionID string) (*Session, error) {
	return nil, nil
}

func (b *GCPTraceBackend) GetSessionTools(ctx context.Context, sessionID string) ([]SessionTool, error) {
	return nil, nil
}

func (b *GCPTraceBackend) GetChatMessagesByTaskID(ctx context.Context, taskID string) ([]*ChatMessage, error) {
	return nil, nil
}

func (b *GCPTraceBackend) GetChatMessagesBySession(ctx context.Context, sessionID string, startTime, endTime time.Time) ([]*ChatMessage, error) {
	return nil, nil
}

func (b *GCPTraceBackend) CountChatMessages(ctx context.Context, q ChatMessageQuery) (int, int, error) {
	return 0, 0, nil
}

func (b *GCPTraceBackend) GetChainByGitHubIssue(ctx context.Context, repo string, issueNumber int) (*ExecutionChain, error) {
	return nil, nil
}

func (b *GCPTraceBackend) UpdateChainMetrics(ctx context.Context, id string, cost float64, tokens, turns int) error {
	return nil
}

func (b *GCPTraceBackend) GetChainStats(ctx context.Context) (*ChainStats, error) {
	return nil, nil
}

func (b *GCPTraceBackend) GetChainStatusCounts(ctx context.Context, createdAfter *time.Time) (*ChainStatusCounts, error) {
	return &ChainStatusCounts{}, nil
}

func (b *GCPTraceBackend) GetChainStatsByAgent(ctx context.Context, createdAfter *time.Time) ([]*AgentStatsResult, error) {
	return nil, nil
}

func (b *GCPTraceBackend) GetCostRollup(ctx context.Context, createdAfter *time.Time, sourcePrefix string) (CostRollup, error) {
	return CostRollup{}, nil
}

func (b *GCPTraceBackend) GetMissionRollups(ctx context.Context, createdAfter *time.Time, sourcePrefix string, topN int) ([]MissionRollup, error) {
	return nil, nil
}

func (b *GCPTraceBackend) GetSpanLitesByStageID(ctx context.Context, stageID string, limit, offset int) (*SpanLitePage, error) {
	return &SpanLitePage{}, nil
}

// Ensure GCPTraceBackend implements Backend

// M-COMPLETION-PATH-PARITY M0b — finalisation writes.
//
// Unlike the Update* methods above, these do NOT return a silent nil. This
// backend cannot store the chain hierarchy, and a finalisation that quietly
// succeeds while writing nothing is the failure mode the milestone exists to
// remove.
func (b *GCPTraceBackend) SetStageStatus(ctx context.Context, stageID string, status ChainStageStatus) error {
	return ErrChainWritesUnsupported
}

func (b *GCPTraceBackend) SetStageMetrics(ctx context.Context, stageID string, cost float64, tokensIn, tokensOut, turns, toolCalls int, durationMs int64, costProvenance string) error {
	return ErrChainWritesUnsupported
}

func (b *GCPTraceBackend) SetStageError(ctx context.Context, stageID, errorMessage string) error {
	return ErrChainWritesUnsupported
}

func (b *GCPTraceBackend) RecomputeChainAggregates(ctx context.Context, chainID string) error {
	return ErrChainWritesUnsupported
}

var _ Backend = (*GCPTraceBackend)(nil)
