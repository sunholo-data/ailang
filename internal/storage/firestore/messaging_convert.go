package firestore

import (
	"encoding/json"
	"time"

	"github.com/sunholo/ailang/internal/messaging"
)

// --- Thread conversion ---

func threadToMap(t *messaging.Thread) map[string]interface{} {
	return map[string]interface{}{
		"id":              t.ID,
		"title":           t.Title,
		"created_at":      timeToFirestore(t.CreatedAt),
		"created_by_type": t.CreatedByType,
		"created_by_id":   t.CreatedByID,
		"status":          t.Status,
		"context_json":    t.ContextJSON,
		"target_agent":    t.TargetAgent,
		"workspace":       t.Workspace,
		"last_seq":        t.LastSeq,
		"updated_at":      timeToFirestore(t.UpdatedAt),
	}
}

func mapToThread(data map[string]interface{}) *messaging.Thread {
	return &messaging.Thread{
		ID:            getString(data, "id"),
		Title:         getString(data, "title"),
		CreatedAt:     snapshotToTime(data, "created_at"),
		CreatedByType: getString(data, "created_by_type"),
		CreatedByID:   getString(data, "created_by_id"),
		Status:        getString(data, "status"),
		ContextJSON:   getString(data, "context_json"),
		TargetAgent:   getString(data, "target_agent"),
		Workspace:     getString(data, "workspace"),
		LastSeq:       getInt(data, "last_seq"),
		UpdatedAt:     snapshotToTime(data, "updated_at"),
	}
}

// --- Message conversion ---

func messageToMap(m *messaging.Message) map[string]interface{} {
	return map[string]interface{}{
		"id":             m.ID,
		"thread_id":      m.ThreadID,
		"message_seq":    m.MessageSeq,
		"created_at":     timeToFirestore(m.CreatedAt),
		"from_type":      m.FromType,
		"from_id":        m.FromID,
		"to_type":        m.ToType,
		"to_id":          m.ToID,
		"kind":           m.Kind,
		"content":        m.Content,
		"metadata_json":  m.MetadataJSON,
		"delivery_state": m.DeliveryState,
		"business_state": m.BusinessState,
	}
}

func mapToMessage(data map[string]interface{}) *messaging.Message {
	return &messaging.Message{
		ID:            getString(data, "id"),
		ThreadID:      getString(data, "thread_id"),
		MessageSeq:    getInt(data, "message_seq"),
		CreatedAt:     snapshotToTime(data, "created_at"),
		FromType:      getString(data, "from_type"),
		FromID:        getString(data, "from_id"),
		ToType:        getString(data, "to_type"),
		ToID:          getString(data, "to_id"),
		Kind:          getString(data, "kind"),
		Content:       getString(data, "content"),
		MetadataJSON:  getString(data, "metadata_json"),
		DeliveryState: getString(data, "delivery_state"),
		BusinessState: getString(data, "business_state"),
	}
}

// --- InboxMessage conversion ---

func inboxToMap(m *messaging.InboxMessage) map[string]interface{} {
	data := map[string]interface{}{
		"id":              m.ID,
		"message_id":      m.MessageID,
		"correlation_id":  m.CorrelationID,
		"from_agent":      m.FromAgent,
		"to_inbox":        m.ToInbox,
		"message_type":    m.MessageType,
		"title":           m.Title,
		"payload":         m.Payload,
		"category":        m.Category,
		"github_repo":     m.GitHubRepo,
		"simhash":         nil,
		"dup_of":          m.DupOf,
		"embedding":       m.Embedding,
		"embedding_model": m.EmbeddingModel,
		"parent_task_id":  m.ParentTaskID,
		"chain_id":        m.ChainID,
		"iteration":       m.Iteration,
		"status":          m.Status,
		"created_at":      timeToFirestore(m.CreatedAt),
	}
	if m.GitHubIssue != nil {
		data["github_issue"] = *m.GitHubIssue
	}
	if m.Simhash != nil {
		data["simhash"] = *m.Simhash
	}
	if m.EmbeddingUpdatedAt != nil {
		data["embedding_updated_at"] = *m.EmbeddingUpdatedAt
	}
	if m.ReadAt != nil {
		data["read_at"] = timeToFirestore(*m.ReadAt)
	}
	if m.ExpiresAt != nil {
		data["expires_at"] = timeToFirestore(*m.ExpiresAt)
	}
	return data
}

func mapToInbox(data map[string]interface{}) *messaging.InboxMessage {
	m := &messaging.InboxMessage{
		ID:             getString(data, "id"),
		MessageID:      getString(data, "message_id"),
		CorrelationID:  getString(data, "correlation_id"),
		FromAgent:      getString(data, "from_agent"),
		ToInbox:        getString(data, "to_inbox"),
		MessageType:    getString(data, "message_type"),
		Title:          getString(data, "title"),
		Payload:        getString(data, "payload"),
		Category:       getString(data, "category"),
		GitHubRepo:     getString(data, "github_repo"),
		DupOf:          getString(data, "dup_of"),
		Embedding:      getString(data, "embedding"),
		EmbeddingModel: getString(data, "embedding_model"),
		ParentTaskID:   getString(data, "parent_task_id"),
		ChainID:        getString(data, "chain_id"),
		Iteration:      getInt(data, "iteration"),
		Status:         getString(data, "status"),
		CreatedAt:      snapshotToTime(data, "created_at"),
	}
	if v := getInt(data, "github_issue"); v != 0 {
		m.GitHubIssue = &v
	}
	if v := getInt64(data, "simhash"); v != 0 {
		m.Simhash = &v
	}
	if v := getInt64(data, "embedding_updated_at"); v != 0 {
		m.EmbeddingUpdatedAt = &v
	}
	m.ReadAt = snapshotToTimePtr(data, "read_at")
	m.ExpiresAt = snapshotToTimePtr(data, "expires_at")
	return m
}

// --- Approval conversion ---

func approvalMsgToMap(a *messaging.Approval) map[string]interface{} {
	data := map[string]interface{}{
		"id":                a.ID,
		"thread_id":         a.ThreadID,
		"thread_title":      a.ThreadTitle,
		"instance_id":       a.InstanceID,
		"created_at":        timeToFirestore(a.CreatedAt),
		"effect_delta_json": a.EffectDeltaJSON,
		"proposal":          a.Proposal,
		"impact":            a.Impact,
		"estimated_cost":    a.EstimatedCost,
		"status":            a.Status,
		"reviewed_by":       a.ReviewedBy,
		"reviewed_at":       timeToFirestore(a.ReviewedAt),
		"review_notes":      a.ReviewNotes,
		"capability_token":  a.CapabilityToken,
		"token_expires_at":  timeToFirestore(a.TokenExpiresAt),
	}
	return data
}

func mapToApprovalMsg(data map[string]interface{}) *messaging.Approval {
	return &messaging.Approval{
		ID:              getString(data, "id"),
		ThreadID:        getString(data, "thread_id"),
		ThreadTitle:     getString(data, "thread_title"),
		InstanceID:      getString(data, "instance_id"),
		CreatedAt:       snapshotToTime(data, "created_at"),
		EffectDeltaJSON: getString(data, "effect_delta_json"),
		Proposal:        getString(data, "proposal"),
		Impact:          getString(data, "impact"),
		EstimatedCost:   getFloat64(data, "estimated_cost"),
		Status:          getString(data, "status"),
		ReviewedBy:      getString(data, "reviewed_by"),
		ReviewedAt:      snapshotToTime(data, "reviewed_at"),
		ReviewNotes:     getString(data, "review_notes"),
		CapabilityToken: getString(data, "capability_token"),
		TokenExpiresAt:  snapshotToTime(data, "token_expires_at"),
	}
}

// --- History entry conversion ---

func approvalHistoryToMap(e *messaging.ApprovalHistoryEntry) map[string]interface{} {
	data := map[string]interface{}{
		"id":               e.ID,
		"approval_id":      e.ApprovalID,
		"thread_id":        e.ThreadID,
		"agent_id":         e.AgentID,
		"action":           e.Action,
		"actor":            e.Actor,
		"proposal":         e.Proposal,
		"impact":           e.Impact,
		"capability_token": e.CapabilityToken,
		"created_at":       e.CreatedAt,
	}
	if e.EstimatedCost != nil {
		data["estimated_cost"] = *e.EstimatedCost
	}
	return data
}

func mapToApprovalHistory(data map[string]interface{}) messaging.ApprovalHistoryEntry {
	e := messaging.ApprovalHistoryEntry{
		ID:              getString(data, "id"),
		ApprovalID:      getString(data, "approval_id"),
		ThreadID:        getString(data, "thread_id"),
		AgentID:         getString(data, "agent_id"),
		Action:          getString(data, "action"),
		Actor:           getString(data, "actor"),
		Proposal:        getString(data, "proposal"),
		Impact:          getString(data, "impact"),
		CapabilityToken: getString(data, "capability_token"),
		CreatedAt:       getInt64(data, "created_at"),
	}
	if v, ok := data["estimated_cost"]; ok && v != nil {
		cost := getFloat64(data, "estimated_cost")
		e.EstimatedCost = &cost
	}
	return e
}

func instanceHistoryToMap(e *messaging.InstanceHistoryEntry) map[string]interface{} {
	data := map[string]interface{}{
		"id":               e.ID,
		"agent_id":         e.AgentID,
		"instance_id":      e.InstanceID,
		"started_at":       e.StartedAt,
		"total_tokens":     e.TotalTokens,
		"total_cost_cents": e.TotalCostCent,
		"thread_count":     e.ThreadCount,
	}
	if e.EndedAt != nil {
		data["ended_at"] = *e.EndedAt
	}
	if e.ExitCode != nil {
		data["exit_code"] = *e.ExitCode
	}
	return data
}

func mapToInstanceHistory(data map[string]interface{}) messaging.InstanceHistoryEntry {
	e := messaging.InstanceHistoryEntry{
		ID:            getString(data, "id"),
		AgentID:       getString(data, "agent_id"),
		InstanceID:    getString(data, "instance_id"),
		StartedAt:     getInt64(data, "started_at"),
		TotalTokens:   getInt(data, "total_tokens"),
		TotalCostCent: getInt(data, "total_cost_cents"),
		ThreadCount:   getInt(data, "thread_count"),
	}
	if v := getInt64(data, "ended_at"); v != 0 {
		e.EndedAt = &v
	}
	if v, ok := data["exit_code"]; ok && v != nil {
		code := getInt(data, "exit_code")
		e.ExitCode = &code
	}
	return e
}

// --- Metrics conversion ---

func metricsToMap(threadID, agentID string, stats *messaging.MessageExecutionStats) map[string]interface{} {
	filesJSON, _ := json.Marshal(stats.FilesCreated)
	return map[string]interface{}{
		"thread_id":     threadID,
		"agent_id":      agentID,
		"duration_ms":   stats.DurationMS,
		"input_tokens":  stats.InputTokens,
		"output_tokens": stats.OutputTokens,
		"cost_cents":    stats.CostCents,
		"files_created": string(filesJSON),
		"created_at":    time.Now(),
	}
}

// --- Agent info conversion ---

func agentInfoToMap(agentID, label, status string) map[string]interface{} {
	return map[string]interface{}{
		"agent_id":   agentID,
		"label":      label,
		"status":     status,
		"updated_at": time.Now(),
	}
}

func mapToAgentInfo(data map[string]interface{}) messaging.AgentInfo {
	return messaging.AgentInfo{
		ID:     getString(data, "agent_id"),
		Label:  getString(data, "label"),
		Status: getString(data, "status"),
	}
}
