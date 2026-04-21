package main

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/sunholo-data/ailang/internal/observatory"
)

// chainExport is the full JSON export structure
type chainExport struct {
	Chain  *observatory.ExecutionChain `json:"chain"`
	Stages []stageExport               `json:"stages"`
}

type stageExport struct {
	Stage       observatory.ChainStage `json:"stage"`
	Messages    []chatMessageExport    `json:"messages,omitempty"`     // Full chat with tool data
	DataStatus  string                 `json:"data_status,omitempty"`  // "available", "no_session_id", "not_synced"
	DataMessage string                 `json:"data_message,omitempty"` // Human-readable explanation
}

// jsonExportOptions controls JSON output filtering
type jsonExportOptions struct {
	SummaryOnly bool // Only show turn numbers and tool names
	ToolsOnly   bool // Only show tool_use blocks
	TurnFilter  int  // Show only specific turn (0 = all)
	StageFilter int  // Show only specific stage (0 = all, 1-based)
}

// printChainJSONFiltered outputs filtered chain data as JSON
func printChainJSONFiltered(chain *observatory.ExecutionChain, opts jsonExportOptions) {
	if opts.SummaryOnly {
		printChainJSONSummary(chain, opts)
		return
	}
	printChainJSONFull(chain, opts)
}

// printChainJSONSummary outputs a compact summary (turns and tools only)
func printChainJSONSummary(chain *observatory.ExecutionChain, opts jsonExportOptions) {
	type turnSummary struct {
		TurnNumber int      `json:"turn"`
		Tools      []string `json:"tools,omitempty"`
		ToolCount  int      `json:"tool_count"`
		TextLength int      `json:"text_length,omitempty"`
	}
	type stageSummary struct {
		StageNum   int           `json:"stage"`
		AgentID    string        `json:"agent"`
		Status     string        `json:"status"`
		TaskID     string        `json:"task_id,omitempty"`
		TurnCount  int           `json:"turns"`
		ToolCalls  int           `json:"tool_calls"`
		TurnDetail []turnSummary `json:"turn_detail,omitempty"`
	}

	summary := struct {
		ChainID    string         `json:"chain_id"`
		Status     string         `json:"status"`
		Source     string         `json:"source"`
		TotalCost  float64        `json:"total_cost"`
		StageCount int            `json:"stage_count"`
		Stages     []stageSummary `json:"stages"`
	}{
		ChainID:    chain.ID,
		Status:     string(chain.Status),
		Source:     string(chain.SourceType),
		TotalCost:  chain.TotalCost,
		StageCount: len(chain.Stages),
		Stages:     make([]stageSummary, 0),
	}

	for i, stage := range chain.Stages {
		stageIdx := i + 1
		if opts.StageFilter > 0 && stageIdx != opts.StageFilter {
			continue
		}

		ss := stageSummary{
			StageNum: stageIdx,
			AgentID:  stage.AgentID,
			Status:   string(stage.Status),
			TaskID:   stage.TaskID,
		}

		// Get messages - prefer deterministic task_id query (M-DETERMINISTIC-CHAT-LINKING)
		var messages []chatMessageExport
		if stage.TaskID != "" {
			messages = getChatMessagesForTask(stage.TaskID)
		}

		// Fallback to timestamp-based query if no deterministic results
		if len(messages) == 0 && stage.TaskID != "" {
			sessionInfo := getSessionInfoFromTask(stage.TaskID)
			if sessionInfo != nil && sessionInfo.SessionID != "" {
				messages = getChatMessagesInRange(sessionInfo.SessionID, sessionInfo.StartedAt, sessionInfo.CompletedAt)
			}
		}

		// Build turn summaries from chat messages
		turnMap := make(map[int]*turnSummary)
		if len(messages) > 0 {
			for _, msg := range messages {
				tn := msg.TurnNumber
				if opts.TurnFilter > 0 && tn != opts.TurnFilter {
					continue
				}
				if turnMap[tn] == nil {
					turnMap[tn] = &turnSummary{TurnNumber: tn}
				}
				ts := turnMap[tn]

				for _, block := range msg.Content {
					switch block.Type {
					case "tool_use":
						if block.ToolUse != nil {
							ts.Tools = append(ts.Tools, block.ToolUse.Name)
							ts.ToolCount++
						}
					case "text":
						ts.TextLength += len(block.Text)
					}
				}
			}
		} else if stage.TaskID != "" {
			// Fallback to streaming events
			events := getTaskEvents(stage.TaskID)
			for _, e := range events {
				tn := e.TurnNum
				if tn == 0 {
					continue
				}
				if opts.TurnFilter > 0 && tn != opts.TurnFilter {
					continue
				}
				if turnMap[tn] == nil {
					turnMap[tn] = &turnSummary{TurnNumber: tn}
				}
				ts := turnMap[tn]

				switch e.StreamType {
				case "tool_use":
					if e.ToolName != "" {
						ts.Tools = append(ts.Tools, e.ToolName)
						ts.ToolCount++
					}
				case "text":
					ts.TextLength += len(e.Text)
				}
			}
		}

		// Convert map to sorted slice
		for tn := 1; tn <= len(turnMap)+100; tn++ { // scan reasonable range
			if ts, ok := turnMap[tn]; ok {
				ts.Tools = dedupeTools(ts.Tools)
				ss.TurnDetail = append(ss.TurnDetail, *ts)
				ss.ToolCalls += ts.ToolCount
			}
		}
		ss.TurnCount = len(ss.TurnDetail)

		summary.Stages = append(summary.Stages, ss)
	}

	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	enc.Encode(summary)
}

// printChainJSONFull outputs full chain data with filtering
func printChainJSONFull(chain *observatory.ExecutionChain, opts jsonExportOptions) {
	// Filter stages if StageFilter is set
	stagesToProcess := chain.Stages
	if opts.StageFilter > 0 {
		var filteredStages []*observatory.ChainStage
		for i, stage := range chain.Stages {
			if i+1 == opts.StageFilter {
				filteredStages = append(filteredStages, stage)
			}
		}
		stagesToProcess = filteredStages
	}

	// Create a copy of chain with filtered stages
	chainCopy := *chain
	chainCopy.Stages = stagesToProcess

	export := chainExport{
		Chain:  &chainCopy,
		Stages: make([]stageExport, 0, len(stagesToProcess)),
	}

	for _, stage := range stagesToProcess {

		se := stageExport{
			Stage: *stage,
		}

		// Get messages - prefer deterministic task_id query (M-DETERMINISTIC-CHAT-LINKING)
		var allMessages []chatMessageExport
		var sessionInfo *taskSessionInfo

		if stage.TaskID != "" {
			allMessages = getChatMessagesForTask(stage.TaskID)
		}

		// Fallback to timestamp-based query if no deterministic results
		if len(allMessages) == 0 {
			if stage.SessionID != "" {
				sessionInfo = &taskSessionInfo{SessionID: stage.SessionID}
			}
			if stage.TaskID != "" {
				if taskInfo := getSessionInfoFromTask(stage.TaskID); taskInfo != nil {
					sessionInfo = taskInfo
				}
			}
			if sessionInfo != nil && sessionInfo.SessionID != "" {
				allMessages = getChatMessagesInRange(sessionInfo.SessionID, sessionInfo.StartedAt, sessionInfo.CompletedAt)
			}
		}

		// Get filtered messages
		if len(allMessages) > 0 {

			// Apply turn and tools filters
			for _, msg := range allMessages {
				if opts.TurnFilter > 0 && msg.TurnNumber != opts.TurnFilter {
					continue
				}

				if opts.ToolsOnly {
					// Filter to only tool_use and tool_result blocks
					var filteredContent []contentBlock
					for _, block := range msg.Content {
						if block.Type == "tool_use" || block.Type == "tool_result" {
							filteredContent = append(filteredContent, block)
						}
					}
					if len(filteredContent) == 0 {
						continue // Skip messages with no tool content
					}
					msg.Content = filteredContent
				}

				se.Messages = append(se.Messages, msg)
			}
		}

		// Set data status
		if len(se.Messages) > 0 {
			se.DataStatus = "available"
		} else if sessionInfo == nil || sessionInfo.SessionID == "" {
			se.DataStatus = "no_session_id"
			se.DataMessage = "Task has no linked session. Run 'ailang observatory sync-chat' after session completes."
		} else {
			se.DataStatus = "not_synced"
			se.DataMessage = fmt.Sprintf("Session %s exists but no chat messages found. Run 'ailang observatory sync-chat' to import.", sessionInfo.SessionID)
		}

		export.Stages = append(export.Stages, se)
	}

	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	enc.Encode(export)
}

// printChainJSON outputs full chain data as JSON (legacy wrapper)
func printChainJSON(chain *observatory.ExecutionChain) {
	printChainJSONFull(chain, jsonExportOptions{})
}
