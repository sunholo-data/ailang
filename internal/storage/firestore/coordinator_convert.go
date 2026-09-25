package firestore

import (
	"time"

	"github.com/sunholo-data/ailang/internal/coordinator"
	"github.com/sunholo-data/ailang/internal/mapval"
)

// taskToMap converts a TaskRecord to a Firestore document map.
//
// A zero CreatedAt is stamped rather than written as null, matching
// observatory_tasks.go which has always done this. The asymmetry was load-bearing:
// timeToFirestore turns a zero time into a Firestore null, the task reads back with
// a zero CreatedAt, and the stale-task detector then aged it from the zero time and
// killed it seconds after dispatch. Persisting "unknown" as null let a missing
// timestamp travel; stamping it at the write boundary keeps every task row orderable
// and ageable. Fixing this does NOT excuse the caller from setting CreatedAt — the
// detector reports an unknowable age loudly rather than acting on one.
func taskToMap(t *coordinator.TaskRecord) map[string]interface{} {
	if t.CreatedAt.IsZero() {
		t.CreatedAt = time.Now()
	}
	m := map[string]interface{}{
		"id":             t.ID,
		"message_id":     t.MessageID,
		"thread_id":      t.ThreadID,
		"parent_task_id": t.ParentTaskID,
		"title":          t.Title,
		"content":        t.Content,
		"type":           string(t.Type),
		"kind":           t.Kind,
		"source":         t.Source, // M-PKG-AUTONOMOUS-CASCADE-SAFE M1: Pub/Sub topic origin
		"priority":       t.Priority,
		"status":         string(t.Status),
		"provider":       t.Provider,
		"agent_id":       t.AgentID,
		"worktree_id":    t.WorktreeID,
		"worktree_path":  t.WorktreePath,
		"base_branch":    t.BaseBranch,
		"base_commit":    t.BaseCommit,
		// M-COMPLETION-PATH-PARITY C1: the ledger must appear in BOTH directions
		// of this hand-written map. Omitting it from either silently drops the
		// ledger, and every redelivery would then re-run every effect.
		"finalization":     ledgerToMap(t.Finalization),
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
		"queued_at":        timePtrToFirestore(t.QueuedAt),
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
		ID:             mapval.String(data, "id"),
		MessageID:      mapval.String(data, "message_id"),
		ThreadID:       mapval.String(data, "thread_id"),
		ParentTaskID:   mapval.String(data, "parent_task_id"),
		Title:          mapval.String(data, "title"),
		Content:        mapval.String(data, "content"),
		Type:           coordinator.TaskType(mapval.String(data, "type")),
		Kind:           mapval.String(data, "kind"),
		Source:         mapval.String(data, "source"), // M-PKG-AUTONOMOUS-CASCADE-SAFE M1
		Priority:       mapval.Int(data, "priority"),
		Status:         coordinator.TaskStatus(mapval.String(data, "status")),
		Provider:       mapval.String(data, "provider"),
		AgentID:        mapval.String(data, "agent_id"),
		WorktreeID:     mapval.String(data, "worktree_id"),
		WorktreePath:   mapval.String(data, "worktree_path"),
		BaseBranch:     mapval.String(data, "base_branch"),
		BaseCommit:     mapval.String(data, "base_commit"),
		Finalization:   ledgerFromMap(data["finalization"]),
		SessionID:      mapval.String(data, "session_id"),
		Iteration:      mapval.Int(data, "iteration"),
		Workspace:      mapval.String(data, "workspace"),
		ChainID:        mapval.String(data, "chain_id"),
		StageID:        mapval.String(data, "stage_id"),
		GithubIssue:    mapval.Int(data, "github_issue"),
		GithubRepo:     mapval.String(data, "github_repo"),
		Stage:          coordinator.TaskStage(mapval.String(data, "stage")),
		DesignDocPath:  mapval.String(data, "design_doc_path"),
		SprintPlanPath: mapval.String(data, "sprint_plan_path"),
		CreatedAt:      snapshotToTime(data, "created_at"),
		StartedAt:      snapshotToTimePtr(data, "started_at"),
		QueuedAt:       snapshotToTimePtr(data, "queued_at"),
		CompletedAt:    snapshotToTimePtr(data, "completed_at"),
		Duration:       time.Duration(mapval.Int64(data, "duration")),
		Error:          mapval.String(data, "error"),
		Output:         mapval.String(data, "output"),
		Cost:           mapval.Float(data, "cost"),
		TokensUsed:     mapval.Int(data, "tokens_used"),
		InputTokens:    mapval.Int(data, "input_tokens"),
		OutputTokens:   mapval.Int(data, "output_tokens"),
		PeakCPU:        mapval.Float(data, "peak_cpu"),
		PeakMemory:     mapval.Float(data, "peak_memory_mb"),
		ImpactLevel:    mapval.String(data, "impact_level"),
		EstimatedCost:  mapval.Float(data, "estimated_cost"),
		// M-PKG-CASCADE-DETERMINISTIC-FIRST: cascade envelope hydration
		RootPackage:       mapval.String(data, "root_package"),
		RootChangeClass:   mapval.String(data, "root_change_class"),
		FromVersion:       mapval.String(data, "from_version"),
		ToVersion:         mapval.String(data, "to_version"),
		FromInterfaceHash: mapval.String(data, "from_interface_hash"),
		ToInterfaceHash:   mapval.String(data, "to_interface_hash"),
		FromContentHash:   mapval.String(data, "from_content_hash"),
		ToContentHash:     mapval.String(data, "to_content_hash"),
		EffectsWidened:    mapval.Bool(data, "effects_widened"),
		PrevEffectCeiling: mapval.Strings(data, "prev_effect_ceiling"),
		NewEffectCeiling:  mapval.Strings(data, "new_effect_ceiling"),
	}

	// Convert capabilities array
	if caps, ok := data["capabilities"].([]interface{}); ok {
		for _, c := range caps {
			if cm, ok := c.(map[string]interface{}); ok {
				cap := coordinator.Capability{
					Type:        coordinator.CapabilityType(mapval.String(cm, "type")),
					BudgetDelta: mapval.Float(cm, "budget_delta"),
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
		"evaluation":         a.Evaluation,
		"handoffs_triggered": false,
	}
}

// mapToApproval converts a Firestore document map to an ApprovalRequestRecord.
func mapToApproval(data map[string]interface{}) *coordinator.ApprovalRequestRecord {
	return &coordinator.ApprovalRequestRecord{
		ID:          mapval.String(data, "id"),
		TaskID:      mapval.String(data, "task_id"),
		Type:        mapval.String(data, "type"),
		Description: mapval.String(data, "description"),
		ContextJSON: mapval.String(data, "context_json"),
		Status:      mapval.String(data, "status"),
		ResolvedBy:  mapval.String(data, "resolved_by"),
		CreatedAt:   snapshotToTime(data, "created_at"),
		ResolvedAt:  snapshotToTimePtr(data, "resolved_at"),
		TimeoutAt:   snapshotToTimePtr(data, "timeout_at"),
		AutoReject:  mapval.Bool(data, "auto_reject"),
		Evaluation:  mapval.String(data, "evaluation"),
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
		ThreadID:    mapval.String(data, "thread_id"),
		StreamType:  mapval.String(data, "stream_type"),
		TurnNum:     mapval.Int(data, "turn_num"),
		Text:        mapval.String(data, "text"),
		ToolName:    mapval.String(data, "tool_name"),
		ToolInput:   mapval.String(data, "tool_input"),
		ToolOutput:  mapval.String(data, "tool_output"),
		ErrorMsg:    mapval.String(data, "error_msg"),
		Status:      mapval.String(data, "status"),
		TokensIn:    mapval.Int(data, "tokens_in"),
		TokensOut:   mapval.Int(data, "tokens_out"),
		Cost:        mapval.Float(data, "cost"),
		DurationSec: mapval.Int(data, "duration_sec"),
		CreatedAt:   snapshotToTime(data, "created_at"),
	}
}
