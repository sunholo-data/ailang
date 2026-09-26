package firestore

import (
	"encoding/json"
	"time"

	"github.com/sunholo-data/ailang/internal/mapval"
	"github.com/sunholo-data/ailang/internal/messaging"
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
		ID:            mapval.String(data, "id"),
		Title:         mapval.String(data, "title"),
		CreatedAt:     snapshotToTime(data, "created_at"),
		CreatedByType: mapval.String(data, "created_by_type"),
		CreatedByID:   mapval.String(data, "created_by_id"),
		Status:        mapval.String(data, "status"),
		ContextJSON:   mapval.String(data, "context_json"),
		TargetAgent:   mapval.String(data, "target_agent"),
		Workspace:     mapval.String(data, "workspace"),
		LastSeq:       mapval.Int(data, "last_seq"),
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
		ID:            mapval.String(data, "id"),
		ThreadID:      mapval.String(data, "thread_id"),
		MessageSeq:    mapval.Int(data, "message_seq"),
		CreatedAt:     snapshotToTime(data, "created_at"),
		FromType:      mapval.String(data, "from_type"),
		FromID:        mapval.String(data, "from_id"),
		ToType:        mapval.String(data, "to_type"),
		ToID:          mapval.String(data, "to_id"),
		Kind:          mapval.String(data, "kind"),
		Content:       mapval.String(data, "content"),
		MetadataJSON:  mapval.String(data, "metadata_json"),
		DeliveryState: mapval.String(data, "delivery_state"),
		BusinessState: mapval.String(data, "business_state"),
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
		ID:             mapval.String(data, "id"),
		MessageID:      mapval.String(data, "message_id"),
		CorrelationID:  mapval.String(data, "correlation_id"),
		FromAgent:      mapval.String(data, "from_agent"),
		ToInbox:        mapval.String(data, "to_inbox"),
		MessageType:    mapval.String(data, "message_type"),
		Title:          mapval.String(data, "title"),
		Payload:        mapval.String(data, "payload"),
		Category:       mapval.String(data, "category"),
		GitHubRepo:     mapval.String(data, "github_repo"),
		DupOf:          mapval.String(data, "dup_of"),
		Embedding:      mapval.String(data, "embedding"),
		EmbeddingModel: mapval.String(data, "embedding_model"),
		ParentTaskID:   mapval.String(data, "parent_task_id"),
		ChainID:        mapval.String(data, "chain_id"),
		Iteration:      mapval.Int(data, "iteration"),
		Status:         mapval.String(data, "status"),
		CreatedAt:      snapshotToTime(data, "created_at"),
	}
	if v := mapval.Int(data, "github_issue"); v != 0 {
		m.GitHubIssue = &v
	}
	if v := mapval.Int64(data, "simhash"); v != 0 {
		m.Simhash = &v
	}
	if v := mapval.Int64(data, "embedding_updated_at"); v != 0 {
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
		ID:              mapval.String(data, "id"),
		ThreadID:        mapval.String(data, "thread_id"),
		ThreadTitle:     mapval.String(data, "thread_title"),
		InstanceID:      mapval.String(data, "instance_id"),
		CreatedAt:       snapshotToTime(data, "created_at"),
		EffectDeltaJSON: mapval.String(data, "effect_delta_json"),
		Proposal:        mapval.String(data, "proposal"),
		Impact:          mapval.String(data, "impact"),
		EstimatedCost:   mapval.Float(data, "estimated_cost"),
		Status:          mapval.String(data, "status"),
		ReviewedBy:      mapval.String(data, "reviewed_by"),
		ReviewedAt:      snapshotToTime(data, "reviewed_at"),
		ReviewNotes:     mapval.String(data, "review_notes"),
		CapabilityToken: mapval.String(data, "capability_token"),
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
		ID:              mapval.String(data, "id"),
		ApprovalID:      mapval.String(data, "approval_id"),
		ThreadID:        mapval.String(data, "thread_id"),
		AgentID:         mapval.String(data, "agent_id"),
		Action:          mapval.String(data, "action"),
		Actor:           mapval.String(data, "actor"),
		Proposal:        mapval.String(data, "proposal"),
		Impact:          mapval.String(data, "impact"),
		CapabilityToken: mapval.String(data, "capability_token"),
		CreatedAt:       mapval.Int64(data, "created_at"),
	}
	if v, ok := data["estimated_cost"]; ok && v != nil {
		cost := mapval.Float(data, "estimated_cost")
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
		ID:            mapval.String(data, "id"),
		AgentID:       mapval.String(data, "agent_id"),
		InstanceID:    mapval.String(data, "instance_id"),
		StartedAt:     mapval.Int64(data, "started_at"),
		TotalTokens:   mapval.Int(data, "total_tokens"),
		TotalCostCent: mapval.Int(data, "total_cost_cents"),
		ThreadCount:   mapval.Int(data, "thread_count"),
	}
	if v := mapval.Int64(data, "ended_at"); v != 0 {
		e.EndedAt = &v
	}
	if v, ok := data["exit_code"]; ok && v != nil {
		code := mapval.Int(data, "exit_code")
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
		ID:     mapval.String(data, "agent_id"),
		Label:  mapval.String(data, "label"),
		Status: mapval.String(data, "status"),
	}
}
