package firestore

import (
	"time"

	"github.com/sunholo-data/ailang/internal/coordinator"
)

// taskToMap converts a TaskRecord to a Firestore document map.
func taskToMap(t *coordinator.TaskRecord) map[string]interface{} {
	m := map[string]interface{}{
		"id":               t.ID,
		"message_id":       t.MessageID,
		"thread_id":        t.ThreadID,
		"parent_task_id":   t.ParentTaskID,
		"title":            t.Title,
		"content":          t.Content,
		"type":             string(t.Type),
		"kind":             t.Kind,
		"source":           t.Source, // M-PKG-AUTONOMOUS-CASCADE-SAFE M1: Pub/Sub topic origin
		"priority":         t.Priority,
		"status":           string(t.Status),
		"provider":         t.Provider,
		"agent_id":         t.AgentID,
		"worktree_id":      t.WorktreeID,
		"worktree_path":    t.WorktreePath,
		"base_branch":      t.BaseBranch,
		"base_commit":      t.BaseCommit,
		"session_id":       t.SessionID,
		"iteration":        t.Iteration,
		"workspace":        t.Workspace,
		"chain_id":         t.ChainID,
		"stage_id":         t.StageID,
		"github_issue":     t.GithubIssue,
		"github_repo":      t.GithubRepo,
		"stage":            string(t.Stage),
		"design_doc_path":  t.DesignDocPath,
		"sprint_plan_path": t.SprintPlanPath,
		"created_at":       timeToFirestore(t.CreatedAt),
		"started_at":       timePtrToFirestore(t.StartedAt),
		"completed_at":     timePtrToFirestore(t.CompletedAt),
		"duration":         int64(t.Duration),
		"error":            t.Error,
		"output":           t.Output,
		"cost":             t.Cost,
		"tokens_used":      t.TokensUsed,
		"input_tokens":     t.InputTokens,
		"output_tokens":    t.OutputTokens,
		"peak_cpu":         t.PeakCPU,
		"peak_memory_mb":   t.PeakMemory,
		"impact_level":     t.ImpactLevel,
		"estimated_cost":   t.EstimatedCost,
		// M-PKG-CASCADE-DETERMINISTIC-FIRST: cascade envelope persistence
		"root_package":        t.RootPackage,
		"root_change_class":   t.RootChangeClass,
		"from_version":        t.FromVersion,
		"to_version":          t.ToVersion,
		"from_interface_hash": t.FromInterfaceHash,
		"to_interface_hash":   t.ToInterfaceHash,
		"from_content_hash":   t.FromContentHash,
		"to_content_hash":     t.ToContentHash,
		"effects_widened":     t.EffectsWidened,
		"prev_effect_ceiling": t.PrevEffectCeiling,
		"new_effect_ceiling":  t.NewEffectCeiling,
	}

	// Convert capabilities array
	if len(t.Capabilities) > 0 {
		caps := make([]map[string]interface{}, len(t.Capabilities))
		for i, c := range t.Capabilities {
			caps[i] = map[string]interface{}{
				"type":         string(c.Type),
				"paths":        c.Paths,
				"budget_delta": c.BudgetDelta,
			}
		}
		m["capabilities"] = caps
	}

	return m
}

// mapToTask converts a Firestore document map to a TaskRecord.
func mapToTask(data map[string]interface{}) *coordinator.TaskRecord {
	t := &coordinator.TaskRecord{
		ID:             getString(data, "id"),
		MessageID:      getString(data, "message_id"),
		ThreadID:       getString(data, "thread_id"),
		ParentTaskID:   getString(data, "parent_task_id"),
		Title:          getString(data, "title"),
		Content:        getString(data, "content"),
		Type:           coordinator.TaskType(getString(data, "type")),
		Kind:           getString(data, "kind"),
		Source:         getString(data, "source"), // M-PKG-AUTONOMOUS-CASCADE-SAFE M1
		Priority:       getInt(data, "priority"),
		Status:         coordinator.TaskStatus(getString(data, "status")),
		Provider:       getString(data, "provider"),
		AgentID:        getString(data, "agent_id"),
		WorktreeID:     getString(data, "worktree_id"),
		WorktreePath:   getString(data, "worktree_path"),
		BaseBranch:     getString(data, "base_branch"),
		BaseCommit:     getString(data, "base_commit"),
		SessionID:      getString(data, "session_id"),
		Iteration:      getInt(data, "iteration"),
		Workspace:      getString(data, "workspace"),
		ChainID:        getString(data, "chain_id"),
		StageID:        getString(data, "stage_id"),
		GithubIssue:    getInt(data, "github_issue"),
		GithubRepo:     getString(data, "github_repo"),
		Stage:          coordinator.TaskStage(getString(data, "stage")),
		DesignDocPath:  getString(data, "design_doc_path"),
		SprintPlanPath: getString(data, "sprint_plan_path"),
		CreatedAt:      snapshotToTime(data, "created_at"),
		StartedAt:      snapshotToTimePtr(data, "started_at"),
		CompletedAt:    snapshotToTimePtr(data, "completed_at"),
		Duration:       time.Duration(getInt64(data, "duration")),
		Error:          getString(data, "error"),
		Output:         getString(data, "output"),
		Cost:           getFloat64(data, "cost"),
		TokensUsed:     getInt(data, "tokens_used"),
		InputTokens:    getInt(data, "input_tokens"),
		OutputTokens:   getInt(data, "output_tokens"),
		PeakCPU:        getFloat64(data, "peak_cpu"),
		PeakMemory:     getFloat64(data, "peak_memory_mb"),
		ImpactLevel:    getString(data, "impact_level"),
		EstimatedCost:  getFloat64(data, "estimated_cost"),
		// M-PKG-CASCADE-DETERMINISTIC-FIRST: cascade envelope hydration
		RootPackage:       getString(data, "root_package"),
		RootChangeClass:   getString(data, "root_change_class"),
		FromVersion:       getString(data, "from_version"),
		ToVersion:         getString(data, "to_version"),
		FromInterfaceHash: getString(data, "from_interface_hash"),
		ToInterfaceHash:   getString(data, "to_interface_hash"),
		FromContentHash:   getString(data, "from_content_hash"),
		ToContentHash:     getString(data, "to_content_hash"),
		EffectsWidened:    getBool(data, "effects_widened"),
		PrevEffectCeiling: getStringSlice(data, "prev_effect_ceiling"),
		NewEffectCeiling:  getStringSlice(data, "new_effect_ceiling"),
	}

	// Convert capabilities array
	if caps, ok := data["capabilities"].([]interface{}); ok {
		for _, c := range caps {
			if cm, ok := c.(map[string]interface{}); ok {
				cap := coordinator.Capability{
					Type:        coordinator.CapabilityType(getString(cm, "type")),
					BudgetDelta: getFloat64(cm, "budget_delta"),
				}
				if paths, ok := cm["paths"].([]interface{}); ok {
					for _, p := range paths {
						if ps, ok := p.(string); ok {
							cap.Paths = append(cap.Paths, ps)
						}
					}
				}
				t.Capabilities = append(t.Capabilities, cap)
			}
		}
	}

	return t
}

// approvalToMap converts an ApprovalRequestRecord to a Firestore document map.
func approvalToMap(a *coordinator.ApprovalRequestRecord) map[string]interface{} {
	return map[string]interface{}{
		"id":                 a.ID,
		"task_id":            a.TaskID,
		"type":               a.Type,
		"description":        a.Description,
		"context_json":       a.ContextJSON,
		"status":             a.Status,
		"resolved_by":        a.ResolvedBy,
		"created_at":         timeToFirestore(a.CreatedAt),
		"resolved_at":        timePtrToFirestore(a.ResolvedAt),
		"timeout_at":         timePtrToFirestore(a.TimeoutAt),
		"auto_reject":        a.AutoReject,
		"handoffs_triggered": false,
	}
}

// mapToApproval converts a Firestore document map to an ApprovalRequestRecord.
func mapToApproval(data map[string]interface{}) *coordinator.ApprovalRequestRecord {
	return &coordinator.ApprovalRequestRecord{
		ID:          getString(data, "id"),
		TaskID:      getString(data, "task_id"),
		Type:        getString(data, "type"),
		Description: getString(data, "description"),
		ContextJSON: getString(data, "context_json"),
		Status:      getString(data, "status"),
		ResolvedBy:  getString(data, "resolved_by"),
		CreatedAt:   snapshotToTime(data, "created_at"),
		ResolvedAt:  snapshotToTimePtr(data, "resolved_at"),
		TimeoutAt:   snapshotToTimePtr(data, "timeout_at"),
		AutoReject:  getBool(data, "auto_reject"),
	}
}

// eventToMap converts a TaskEventRecord to a Firestore document map.
func eventToMap(e *coordinator.TaskEventRecord) map[string]interface{} {
	return map[string]interface{}{
		"task_id":      e.TaskID,
		"thread_id":    e.ThreadID,
		"stream_type":  e.StreamType,
		"turn_num":     e.TurnNum,
		"text":         e.Text,
		"tool_name":    e.ToolName,
		"tool_input":   e.ToolInput,
		"tool_output":  e.ToolOutput,
		"error_msg":    e.ErrorMsg,
		"status":       e.Status,
		"tokens_in":    e.TokensIn,
		"tokens_out":   e.TokensOut,
		"cost":         e.Cost,
		"duration_sec": e.DurationSec,
		"created_at":   timeToFirestore(e.CreatedAt),
	}
}

// mapToEvent converts a Firestore document map to a TaskEventRecord.
func mapToEvent(data map[string]interface{}, taskID string) *coordinator.TaskEventRecord {
	return &coordinator.TaskEventRecord{
		TaskID:      taskID,
		ThreadID:    getString(data, "thread_id"),
		StreamType:  getString(data, "stream_type"),
		TurnNum:     getInt(data, "turn_num"),
		Text:        getString(data, "text"),
		ToolName:    getString(data, "tool_name"),
		ToolInput:   getString(data, "tool_input"),
		ToolOutput:  getString(data, "tool_output"),
		ErrorMsg:    getString(data, "error_msg"),
		Status:      getString(data, "status"),
		TokensIn:    getInt(data, "tokens_in"),
		TokensOut:   getInt(data, "tokens_out"),
		Cost:        getFloat64(data, "cost"),
		DurationSec: getInt(data, "duration_sec"),
		CreatedAt:   snapshotToTime(data, "created_at"),
	}
}

// --- Type-safe Firestore value extractors ---

func getString(data map[string]interface{}, key string) string {
	v, ok := data[key]
	if !ok || v == nil {
		return ""
	}
	if s, ok := v.(string); ok {
		return s
	}
	return ""
}

func getInt(data map[string]interface{}, key string) int {
	v, ok := data[key]
	if !ok || v == nil {
		return 0
	}
	// Firestore returns numbers as int64 or float64
	switch n := v.(type) {
	case int64:
		return int(n)
	case float64:
		return int(n)
	case int:
		return n
	}
	return 0
}

func getInt64(data map[string]interface{}, key string) int64 {
	v, ok := data[key]
	if !ok || v == nil {
		return 0
	}
	switch n := v.(type) {
	case int64:
		return n
	case float64:
		return int64(n)
	case int:
		return int64(n)
	}
	return 0
}

func getFloat64(data map[string]interface{}, key string) float64 {
	v, ok := data[key]
	if !ok || v == nil {
		return 0
	}
	switch n := v.(type) {
	case float64:
		return n
	case int64:
		return float64(n)
	case int:
		return float64(n)
	}
	return 0
}

func getBool(data map[string]interface{}, key string) bool {
	v, ok := data[key]
	if !ok || v == nil {
		return false
	}
	if b, ok := v.(bool); ok {
		return b
	}
	return false
}

// getStringSlice extracts a []string from a Firestore document field that
// may have arrived as []interface{} (Firestore's native array type) or as
// a real []string. Returns nil for missing keys (so omitempty JSON works).
// M-PKG-CASCADE-DETERMINISTIC-FIRST.
func getStringSlice(data map[string]interface{}, key string) []string {
	v, ok := data[key]
	if !ok || v == nil {
		return nil
	}
	switch arr := v.(type) {
	case []string:
		return arr
	case []interface{}:
		out := make([]string, 0, len(arr))
		for _, x := range arr {
			if s, ok := x.(string); ok {
				out = append(out, s)
			}
		}
		return out
	}
	return nil
}
