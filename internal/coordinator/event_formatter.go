// Package coordinator provides task coordination and event formatting.
package coordinator

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

// EventsResponse is the JSON response for the events API endpoint.
type EventsResponse struct {
	TaskID      string             `json:"task_id"`
	TotalEvents int                `json:"total_events"`
	TotalTurns  int                `json:"total_turns"`
	Iteration   int                `json:"iteration,omitempty"`
	Events      []*TaskEventRecord `json:"events"`
}

// FormatOptions controls how events are formatted.
type FormatOptions struct {
	// ShowTimestamps includes timestamps in text output
	ShowTimestamps bool
	// ShowToolInputs includes full tool input in text output
	ShowToolInputs bool
	// MaxTextLength truncates long text blocks (0 = no limit)
	MaxTextLength int
	// TurnFilter only shows specific turn (0 = all turns)
	TurnFilter int
	// TypeFilter only shows specific event types (empty = all types)
	TypeFilter []string
}

// DefaultFormatOptions returns sensible defaults for CLI display.
func DefaultFormatOptions() *FormatOptions {
	return &FormatOptions{
		ShowTimestamps: true,
		ShowToolInputs: true,
		MaxTextLength:  0,
		TurnFilter:     0,
		TypeFilter:     nil,
	}
}

// FormatEventsAsText formats events for human-readable CLI output.
// Returns a string with turn separators and tool highlighting.
func FormatEventsAsText(events []*TaskEventRecord, opts *FormatOptions) string {
	if opts == nil {
		opts = DefaultFormatOptions()
	}

	if len(events) == 0 {
		return "No events recorded for this task."
	}

	var sb strings.Builder
	var currentTurn int
	var turnText strings.Builder

	// Filter events if needed
	filtered := filterEvents(events, opts)

	for _, event := range filtered {
		// Handle turn boundaries
		if event.TurnNum != currentTurn && event.TurnNum > 0 {
			// Flush previous turn
			if currentTurn > 0 {
				sb.WriteString(formatTurnBlock(currentTurn, turnText.String(), opts))
			}
			currentTurn = event.TurnNum
			turnText.Reset()
		}

		// Format based on event type
		switch event.StreamType {
		case "turn_start":
			// Turn header handled by formatTurnBlock
			continue

		case "text":
			text := event.Text
			if opts.MaxTextLength > 0 && len(text) > opts.MaxTextLength {
				text = text[:opts.MaxTextLength] + "..."
			}
			turnText.WriteString(text)

		case "tool_use":
			turnText.WriteString(formatToolUse(event, opts))

		case "tool_result":
			// Tool results are usually verbose, show summary
			if event.ToolOutput != "" {
				output := event.ToolOutput
				if len(output) > 200 {
					output = output[:200] + "..."
				}
				turnText.WriteString(fmt.Sprintf("\n  └─ Result: %s\n", output))
			}

		case "turn_end":
			// Will be handled by next turn_start or end of events
			continue

		case "error":
			turnText.WriteString(fmt.Sprintf("\n[ERROR] %s\n", event.ErrorMsg))

		case "status":
			// Status events are metadata, skip in text output
			continue

		case "human_feedback":
			turnText.WriteString(fmt.Sprintf("\n[HUMAN FEEDBACK]\n%s\n", event.Text))

		case "human_approval":
			turnText.WriteString(fmt.Sprintf("\n[APPROVED] %s\n", event.Text))

		case "iteration_start":
			turnText.WriteString(fmt.Sprintf("\n[ITERATION %d STARTED]\n", event.TurnNum))
		}
	}

	// Flush last turn
	if currentTurn > 0 {
		sb.WriteString(formatTurnBlock(currentTurn, turnText.String(), opts))
	}

	return sb.String()
}

// formatTurnBlock formats a single turn with header and content.
func formatTurnBlock(turnNum int, content string, opts *FormatOptions) string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("\n─── Turn %d ", turnNum))
	sb.WriteString(strings.Repeat("─", 50))
	sb.WriteString("\n")

	// Add box around content
	lines := strings.Split(strings.TrimSpace(content), "\n")
	for _, line := range lines {
		sb.WriteString(fmt.Sprintf("│ %s\n", line))
	}
	sb.WriteString("└" + strings.Repeat("─", 60) + "\n")

	return sb.String()
}

// formatToolUse formats a tool invocation for text display.
func formatToolUse(event *TaskEventRecord, opts *FormatOptions) string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("\n[TOOL] %s\n", event.ToolName))

	if opts.ShowToolInputs && event.ToolInput != "" {
		// Try to pretty-print JSON input
		var inputMap map[string]interface{}
		if err := json.Unmarshal([]byte(event.ToolInput), &inputMap); err == nil {
			for key, val := range inputMap {
				valStr := fmt.Sprintf("%v", val)
				if len(valStr) > 100 {
					valStr = valStr[:100] + "..."
				}
				sb.WriteString(fmt.Sprintf("  %s: %s\n", key, valStr))
			}
		} else {
			// Not JSON, show raw (truncated)
			input := event.ToolInput
			if len(input) > 200 {
				input = input[:200] + "..."
			}
			sb.WriteString(fmt.Sprintf("  input: %s\n", input))
		}
	}

	return sb.String()
}

// filterEvents applies turn and type filters to events.
func filterEvents(events []*TaskEventRecord, opts *FormatOptions) []*TaskEventRecord {
	if opts.TurnFilter == 0 && len(opts.TypeFilter) == 0 {
		return events
	}

	var filtered []*TaskEventRecord
	typeSet := make(map[string]bool)
	for _, t := range opts.TypeFilter {
		typeSet[t] = true
	}

	for _, event := range events {
		// Apply turn filter
		if opts.TurnFilter > 0 && event.TurnNum != opts.TurnFilter {
			continue
		}

		// Apply type filter
		if len(typeSet) > 0 && !typeSet[event.StreamType] {
			continue
		}

		filtered = append(filtered, event)
	}

	return filtered
}

// FormatEventsAsJSON formats events as a JSON response.
func FormatEventsAsJSON(taskID string, events []*TaskEventRecord, opts *FormatOptions) (*EventsResponse, error) {
	if opts == nil {
		opts = DefaultFormatOptions()
	}

	// Filter events if needed
	filtered := filterEvents(events, opts)

	// Count turns
	maxTurn := 0
	for _, event := range events {
		if event.TurnNum > maxTurn {
			maxTurn = event.TurnNum
		}
	}

	return &EventsResponse{
		TaskID:      taskID,
		TotalEvents: len(events),
		TotalTurns:  maxTurn,
		Events:      filtered,
	}, nil
}

// CountTurns returns the number of turns in the event list.
func CountTurns(events []*TaskEventRecord) int {
	maxTurn := 0
	for _, event := range events {
		if event.TurnNum > maxTurn {
			maxTurn = event.TurnNum
		}
	}
	return maxTurn
}

// GetTurnTimestamp returns the timestamp of the first event in a turn.
func GetTurnTimestamp(events []*TaskEventRecord, turnNum int) *time.Time {
	for _, event := range events {
		if event.TurnNum == turnNum {
			return &event.CreatedAt
		}
	}
	return nil
}

// SummarizeEvents returns a brief summary of the events.
func SummarizeEvents(events []*TaskEventRecord) string {
	turns := CountTurns(events)
	toolCount := 0
	textLen := 0

	for _, event := range events {
		if event.StreamType == "tool_use" {
			toolCount++
		}
		if event.StreamType == "text" {
			textLen += len(event.Text)
		}
	}

	return fmt.Sprintf("%d turns, %d tool calls, %d chars of text", turns, toolCount, textLen)
}
