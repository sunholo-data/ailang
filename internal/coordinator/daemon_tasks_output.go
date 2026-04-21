package coordinator

import (
	"encoding/json"
	"fmt"

	"github.com/sunholo-data/ailang/internal/messaging"
)

// postTaskStatus posts a status update to the task's thread
func (d *Daemon) postTaskStatus(task *TaskRecord, status, message string) {
	if task.ThreadID == "" || d.msgStore == nil {
		return
	}

	content := fmt.Sprintf("**Status: %s**\n\n%s", status, message)
	_, err := d.msgStore.CreateMessage(
		task.ThreadID,
		"ailang_instance", "coordinator", // from
		"human", "user", // to (for visibility)
		"status",
		content,
		"",
	)
	if err != nil {
		d.logger.Printf("Failed to post status to thread %s: %v", task.ThreadID, err)
	}
}

// postTaskResult posts the execution result to the task's thread and records metrics
func (d *Daemon) postTaskResult(task *TaskRecord, result *ExecuteResult, execErr error) {
	if task.ThreadID == "" || d.msgStore == nil {
		return
	}

	var content string
	var kind string

	if execErr != nil {
		kind = "status" // DB schema only allows: directive, question, proposal, status, result
		content = fmt.Sprintf("**Task Failed**\n\n❌ Error: %v", execErr)
	} else if result != nil {
		if result.Success {
			kind = "result"
			content = fmt.Sprintf("**Task Completed Successfully**\n\n"+
				"- **Provider:** %s\n"+
				"- **Duration:** %s\n"+
				"- **Cost:** $%.4f\n"+
				"- **Tokens:** %d (in: %d, out: %d)\n\n"+
				"---\n\n%s",
				result.Provider, result.Duration, result.Cost,
				result.TokensUsed, result.InputTokens, result.OutputTokens, result.Output)

			if len(result.FilesCreated) > 0 {
				content += fmt.Sprintf("\n\n**Files Created:** %v", result.FilesCreated)
			}
			if len(result.FilesModified) > 0 {
				content += fmt.Sprintf("\n\n**Files Modified:** %v", result.FilesModified)
			}
		} else {
			kind = "status" // DB schema only allows: directive, question, proposal, status, result
			content = fmt.Sprintf("**Task Failed**\n\n"+
				"- **Provider:** %s\n"+
				"- **Duration:** %s\n\n"+
				"**Error:** %s",
				result.Provider, result.Duration, result.Error)
		}
	} else {
		kind = "status" // DB schema only allows: directive, question, proposal, status, result
		content = "**Task Failed**\n\n❌ Unknown error"
	}

	// Create metadata with execution_stats format expected by metrics system
	metadataJSON := ""
	if result != nil {
		metadata := map[string]interface{}{
			"execution_stats": map[string]interface{}{
				"duration_ms":    result.Duration.Milliseconds(),
				"input_tokens":   result.InputTokens,
				"output_tokens":  result.OutputTokens,
				"cost":           result.Cost, // In dollars
				"files_created":  result.FilesCreated,
				"files_modified": result.FilesModified,
			},
		}
		if data, err := json.Marshal(metadata); err == nil {
			metadataJSON = string(data)
		}
	}

	_, err := d.msgStore.CreateMessage(
		task.ThreadID,
		"ailang_instance", "coordinator",
		"human", "user",
		kind,
		content,
		metadataJSON,
	)
	if err != nil {
		d.logger.Printf("Failed to post result to thread %s: %v", task.ThreadID, err)
		return
	}

	// Record metrics at global, agent, and thread levels
	if result != nil {
		stats := &messaging.MessageExecutionStats{
			DurationMS:   int(result.Duration.Milliseconds()),
			InputTokens:  result.InputTokens,
			OutputTokens: result.OutputTokens,
			CostCents:    int(result.Cost * 100), // Convert dollars to cents
			FilesCreated: result.FilesCreated,
		}
		if err := d.msgStore.RecordMetrics(task.ThreadID, "coordinator", stats); err != nil {
			d.logger.Printf("Failed to record metrics: %v", err)
		} else {
			d.logger.Printf("Recorded metrics: thread=%s, tokens=%d, cost=$%.4f",
				task.ThreadID, stats.InputTokens+stats.OutputTokens, result.Cost)
		}
	}
}
