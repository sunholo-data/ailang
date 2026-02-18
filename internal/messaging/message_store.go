// Package messaging provides the MessageStore interface for collaboration hub storage.
// This interface enables pluggable backends (SQLite, Firestore, etc.) for cloud deployment.
package messaging

import (
	"context"
	"time"
)

// MessageStore defines the abstract storage interface for the collaboration hub.
// The concrete SQLite implementation is Store; cloud backends (Firestore, etc.)
// can implement this interface to enable AILANG_STORAGE=gcp mode.
type MessageStore interface {
	// Lifecycle
	Close() error

	// === Thread Management ===

	CreateThread(title, createdByType, createdByID, targetAgent string) (*Thread, error)
	CreateThreadWithWorkspace(title, createdByType, createdByID, targetAgent, workspace string) (*Thread, error)
	GetOrCreateThreadWithWorkspace(title, createdByType, createdByID, targetAgent, workspace string) (*Thread, bool, error)
	GetThreadByTitleAndAgent(title, targetAgent string) (*Thread, error)
	GetThread(threadID string) (*Thread, error)
	SetThreadWorkspace(threadID, workspace string) error
	GetThreadWorkspace(threadID string) (string, error)
	SetThreadTargetAgent(threadID, targetAgent string) error
	UpdateThreadTitle(threadID, title string) error
	DeleteThread(threadID string) error
	GetThreadsByStatus(status string, limit int) ([]Thread, error)
	NewThreadFilter(status, workspace string, limit int) ThreadFilter
	GetThreadsFiltered(filter ThreadFilter) ([]Thread, error)
	GetDistinctWorkspaces() ([]string, error)
	GetThreadAggregateStats() (*ThreadAggregateStats, error)

	// === Message Management ===

	GetMessages(toType, toID string, deliveryState string) ([]Message, error)
	MarkAsAcked(messageID string) error
	MarkAsUnacked(messageID string) error
	ClaimMessage(messageID, claimedBy string) error
	MarkAllAsAcked(toType, toID string) (int64, error)
	CreateMessage(threadID, fromType, fromID, toType, toID, kind, content, metadataJSON string) (*Message, error)
	GetMessagesFromSeq(threadID string, fromSeq int, limit int) ([]Message, error)
	Subscribe(instanceID, threadID string) error
	UpdateAckSeq(instanceID, threadID string, ackSeq int) error

	// === Inbox Management ===

	InsertInboxMessage(msg *InboxMessage) error
	InsertInboxMessageWithContext(ctx context.Context, msg *InboxMessage) error
	ListInboxMessages(opts InboxListOptions) ([]InboxMessage, error)
	GetInboxMessage(id string) (*InboxMessage, error)
	MarkInboxMessageRead(id string) error
	MarkInboxMessageUnread(id string) error
	MarkAllInboxMessagesRead(inbox string) (int64, error)
	ForwardInboxMessage(id string, toInbox string) error
	InboxMessageExistsByGitHub(repo string, issueNumber int) (bool, error)
	InboxMessageExistsByTitle(inbox string, title string) (string, error)
	UpdateInboxMessageGitHub(messageID string, issueNumber int, repo string) error
	CleanupInboxMessages(olderThan time.Duration, expiredOnly bool) (int64, error)
	CountInboxMessagesByStatus(inbox string) (map[string]int64, error)
	GetMessageFlowEdges() ([]MessageFlowEdge, error)
	GetActiveAgents() ([]ActiveAgent, error)

	// === Approval Workflow ===

	CreateApproval(threadID, instanceID string, effectDelta *EffectDelta, proposal, impact string, estimatedCost float64) (*Approval, error)
	GetApproval(approvalID string) (*Approval, error)
	GetApprovalsByStatus(status string, limit int) ([]Approval, error)
	ApproveApproval(approvalID, reviewedBy string, reviewNotes string, tokenDuration time.Duration) error
	RejectApproval(approvalID, reviewedBy string, reviewNotes string) error

	// === History Tracking ===

	RecordApprovalHistory(approvalID, threadID, agentID, action, actor, proposal, impact string, estimatedCost *float64, capabilityToken string) error
	GetApprovalHistory(threadID string, limit int) ([]ApprovalHistoryEntry, error)
	RecordInstanceStart(agentID, instanceID string) error
	RecordInstanceEnd(instanceID string, exitCode int, totalTokens, totalCostCents, threadCount int) error
	GetInstanceHistory(agentID string, limit int) ([]InstanceHistoryEntry, error)
	CleanupOldHistory(retentionDays int) (int64, int64, error)

	// === Search & Deduplication ===

	SemanticSearch(opts SearchOptions) ([]SearchHit, error)
	FindSimilar(msgID string, threshold float64, limit int) ([]SearchHit, error)
	FindDuplicates(inbox string, threshold float64) ([]DuplicateGroup, error)
	ApplyDuplicates(groups []DuplicateGroup, runID string) error
	ClearDuplicateMarker(msgID string) error
	UpdateMessageEmbedding(msgID string, embedding []float32, model string) error

	// === Metrics & Analytics ===

	RecordMetrics(threadID, agentID string, stats *MessageExecutionStats) error
	GetMetrics(scopeType, scopeID string) (*AggregatedMetrics, error)
	GetGlobalMetrics() (*AggregatedMetrics, error)
	GetAgentMetrics(agentID string) (*AggregatedMetrics, error)
	GetThreadMetrics(threadID string) (*AggregatedMetrics, error)
	GetMetricsTrends(scopeType, scopeID, period string, limit int) ([]map[string]interface{}, error)

	// === Execution Hierarchy ===

	GetAggregatedExecutionStats() (*ExecutionStats, error)
	GetExecutionStatsByThread(threadID string) (*ExecutionStats, error)
	GetHierarchy() (*HierarchyResponse, error)
	GetAgentStats(agentID string) (*AgentStats, error)
	GetKnownAgents() ([]AgentInfo, error)

	// === Agent Registration ===

	RegisterAgent(agentID, label, status string) error
	UpdateAgentStatus(agentID, status string) error
	RecordAgentInstance(agentID, instanceID string) error
}

// Compile-time check that Store implements MessageStore.
var _ MessageStore = (*Store)(nil)
