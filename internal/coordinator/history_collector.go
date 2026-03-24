package coordinator

import (
	"context"
	"fmt"
	"sort"
	"time"

	"github.com/sunholo/ailang/internal/messaging"
	"github.com/sunholo/ailang/internal/pkg"
)

// CollectVersionHistory builds a VersionHistory by aggregating existing data sources:
// - The triggering inbox message (from messaging store)
// - Task events (from task store: turns, tool uses, completions, errors)
// - Approval history (from messaging store: created, approved, rejected)
//
// No new data capture needed — everything is already recorded during execution.
func CollectVersionHistory(
	ctx context.Context,
	msgStore messaging.MessageStore,
	taskStore Store,
	task *TaskRecord,
	pkgName, version, previousVersion string,
) *pkg.VersionHistory {
	history := &pkg.VersionHistory{
		Schema:    pkg.VersionHistorySchema,
		Package:   pkgName,
		Version:   version,
		Previous:  previousVersion,
		CreatedAt: time.Now().UTC().Format(time.RFC3339),
	}

	// 1. Add the triggering message as the first entry
	if task.MessageID != "" && msgStore != nil {
		if msg, err := msgStore.GetInboxMessage(task.MessageID); err == nil && msg != nil {
			history.Messages = append(history.Messages, pkg.HistoryEntry{
				Timestamp: msg.CreatedAt.Format(time.RFC3339),
				Kind:      msg.Category,
				From:      msg.FromAgent,
				Title:     msg.Title,
				Detail:    truncateForHistory(msg.Payload, 500),
				Status:    "received",
			})
		}
	}

	// 2. Add task events (turns, tool uses, status changes, errors)
	if taskStore != nil {
		events, err := taskStore.GetTaskEvents(ctx, task.ID, 100)
		if err == nil {
			for _, evt := range events {
				entry := taskEventToHistoryEntry(evt)
				if entry != nil {
					history.Messages = append(history.Messages, *entry)
				}
			}
		}
	}

	// 3. Add approval history entries
	if task.ThreadID != "" && msgStore != nil {
		approvals, err := msgStore.GetApprovalHistory(task.ThreadID, 20)
		if err == nil {
			for _, a := range approvals {
				history.Messages = append(history.Messages, pkg.HistoryEntry{
					Timestamp: time.Unix(a.CreatedAt, 0).UTC().Format(time.RFC3339),
					Kind:      "approval-" + a.Action,
					From:      a.Actor,
					Title:     a.Action + " by " + a.Actor,
					Detail:    truncateForHistory(a.Proposal, 500),
					Status:    "completed",
				})
			}
		}
	}

	// 4. Add task completion entry
	if task.Status == TaskStatusCompleted || task.Status == TaskStatusFailed {
		completedAt := time.Now().UTC()
		if task.CompletedAt != nil {
			completedAt = *task.CompletedAt
		}
		history.Messages = append(history.Messages, pkg.HistoryEntry{
			Timestamp: completedAt.Format(time.RFC3339),
			Kind:      "task-" + string(task.Status),
			From:      task.AgentID,
			Title:     "Task " + string(task.Status),
			Detail:    taskCompletionDetail(task),
			Status:    string(task.Status),
		})
	}

	// Sort all entries by timestamp
	sort.Slice(history.Messages, func(i, j int) bool {
		return history.Messages[i].Timestamp < history.Messages[j].Timestamp
	})

	return history
}

// taskEventToHistoryEntry maps a TaskEventRecord to a HistoryEntry.
// Returns nil for events that aren't meaningful for the version history
// (e.g., individual text chunks, tool outputs).
func taskEventToHistoryEntry(evt *TaskEventRecord) *pkg.HistoryEntry {
	switch evt.StreamType {
	case "status":
		return &pkg.HistoryEntry{
			Timestamp: evt.CreatedAt.Format(time.RFC3339),
			Kind:      "status",
			From:      "executor",
			Title:     "Status: " + evt.Status,
			Detail:    evt.Text,
			Status:    evt.Status,
		}
	case "turn_start":
		return &pkg.HistoryEntry{
			Timestamp: evt.CreatedAt.Format(time.RFC3339),
			Kind:      "turn-start",
			From:      "executor",
			Title:     "Turn started",
			Status:    "running",
		}
	case "turn_end":
		detail := ""
		if evt.TokensIn > 0 || evt.TokensOut > 0 {
			detail = tokenSummary(evt.TokensIn, evt.TokensOut, evt.Cost)
		}
		return &pkg.HistoryEntry{
			Timestamp: evt.CreatedAt.Format(time.RFC3339),
			Kind:      "turn-end",
			From:      "executor",
			Title:     "Turn completed",
			Detail:    detail,
			Status:    "completed",
		}
	case "error":
		return &pkg.HistoryEntry{
			Timestamp: evt.CreatedAt.Format(time.RFC3339),
			Kind:      "error",
			From:      "executor",
			Title:     "Error: " + truncateForHistory(evt.ErrorMsg, 100),
			Detail:    evt.ErrorMsg,
			Status:    "failed",
		}
	default:
		// Skip text, tool_use, tool_result — too granular for version history
		return nil
	}
}

func taskCompletionDetail(task *TaskRecord) string {
	if task.InputTokens > 0 || task.OutputTokens > 0 {
		return tokenSummary(task.InputTokens, task.OutputTokens, task.Cost)
	}
	return ""
}

func tokenSummary(in, out int, cost float64) string {
	if cost > 0 {
		return fmt.Sprintf("Tokens: %d in, %d out. Cost: $%.4f", in, out, cost)
	}
	return fmt.Sprintf("Tokens: %d in, %d out", in, out)
}

func truncateForHistory(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}
