// Package coordinator provides task coordination and event formatting.
package coordinator

import (
	"strings"
	"testing"
	"time"
)

// Edge Case Tests

func TestFormatEventsAsText_EmptyEventIDFields(t *testing.T) {
	now := time.Now()
	events := []*TaskEventRecord{
		{ID: 0, StreamType: "turn_start", TurnNum: 1, CreatedAt: now},
		{ID: 0, StreamType: "text", TurnNum: 1, Text: "Content", CreatedAt: now},
		{ID: 0, StreamType: "turn_end", TurnNum: 1, CreatedAt: now},
	}

	result := FormatEventsAsText(events, nil)
	if !strings.Contains(result, "Turn 1") {
		t.Errorf("expected Turn 1 header, got: %s", result)
	}
	if !strings.Contains(result, "Content") {
		t.Errorf("expected content, got: %s", result)
	}
}

func TestFormatEventsAsText_NegativeTurnNumbers(t *testing.T) {
	now := time.Now()
	events := []*TaskEventRecord{
		{ID: 1, StreamType: "turn_start", TurnNum: -1, CreatedAt: now},
		{ID: 2, StreamType: "text", TurnNum: -1, Text: "Invalid turn", CreatedAt: now},
		{ID: 3, StreamType: "turn_end", TurnNum: -1, CreatedAt: now},
	}

	result := FormatEventsAsText(events, nil)
	// Negative turn numbers should be skipped (TurnNum > 0 check in code)
	if strings.Contains(result, "Turn -1") {
		t.Errorf("expected negative turns to be skipped, got: %s", result)
	}
}

func TestFormatEventsAsText_SpecialCharactersInText(t *testing.T) {
	now := time.Now()
	specialText := `Special chars: \n\t"'<>&`
	events := []*TaskEventRecord{
		{ID: 1, StreamType: "turn_start", TurnNum: 1, CreatedAt: now},
		{ID: 2, StreamType: "text", TurnNum: 1, Text: specialText, CreatedAt: now},
		{ID: 3, StreamType: "turn_end", TurnNum: 1, CreatedAt: now},
	}

	result := FormatEventsAsText(events, nil)
	if !strings.Contains(result, "Special chars") {
		t.Errorf("expected special characters in output, got: %s", result)
	}
}

func TestFormatEventsAsText_MultipleToolsInTurn(t *testing.T) {
	now := time.Now()
	events := []*TaskEventRecord{
		{ID: 1, StreamType: "turn_start", TurnNum: 1, CreatedAt: now},
		{ID: 2, StreamType: "tool_use", TurnNum: 1, ToolName: "Read", ToolInput: `{"path":"/tmp/file1"}`, CreatedAt: now},
		{ID: 3, StreamType: "tool_result", TurnNum: 1, ToolOutput: "file1 content", CreatedAt: now},
		{ID: 4, StreamType: "tool_use", TurnNum: 1, ToolName: "Write", ToolInput: `{"path":"/tmp/file2"}`, CreatedAt: now},
		{ID: 5, StreamType: "tool_result", TurnNum: 1, ToolOutput: "written", CreatedAt: now},
		{ID: 6, StreamType: "turn_end", TurnNum: 1, CreatedAt: now},
	}

	result := FormatEventsAsText(events, nil)
	if !strings.Contains(result, "[TOOL] Read") {
		t.Errorf("expected Read tool, got: %s", result)
	}
	if !strings.Contains(result, "[TOOL] Write") {
		t.Errorf("expected Write tool, got: %s", result)
	}
}

func TestFormatEventsAsText_ZeroTurnNumbers(t *testing.T) {
	now := time.Now()
	events := []*TaskEventRecord{
		{ID: 1, StreamType: "text", TurnNum: 0, Text: "Invalid", CreatedAt: now},
	}

	result := FormatEventsAsText(events, nil)
	// Turn 0 should be skipped (TurnNum > 0 check in code)
	if result != "" {
		t.Errorf("expected empty string for turn 0 events, got: %q", result)
	}
}

func TestFormatEventsAsText_EmptyToolName(t *testing.T) {
	now := time.Now()
	events := []*TaskEventRecord{
		{ID: 1, StreamType: "turn_start", TurnNum: 1, CreatedAt: now},
		{ID: 2, StreamType: "tool_use", TurnNum: 1, ToolName: "", ToolInput: "input", CreatedAt: now},
		{ID: 3, StreamType: "turn_end", TurnNum: 1, CreatedAt: now},
	}

	result := FormatEventsAsText(events, nil)
	if !strings.Contains(result, "[TOOL]") {
		t.Errorf("expected tool marker, got: %s", result)
	}
}

func TestFormatEventsAsText_EmptyErrorMessage(t *testing.T) {
	now := time.Now()
	events := []*TaskEventRecord{
		{ID: 1, StreamType: "turn_start", TurnNum: 1, CreatedAt: now},
		{ID: 2, StreamType: "error", TurnNum: 1, ErrorMsg: "", CreatedAt: now},
		{ID: 3, StreamType: "turn_end", TurnNum: 1, CreatedAt: now},
	}

	result := FormatEventsAsText(events, nil)
	if !strings.Contains(result, "[ERROR]") {
		t.Errorf("expected error marker even with empty message, got: %s", result)
	}
}

func TestFormatEventsAsText_VeryLargeTurnNumbers(t *testing.T) {
	now := time.Now()
	events := []*TaskEventRecord{
		{ID: 1, StreamType: "turn_start", TurnNum: 9999, CreatedAt: now},
		{ID: 2, StreamType: "text", TurnNum: 9999, Text: "Far future turn", CreatedAt: now},
		{ID: 3, StreamType: "turn_end", TurnNum: 9999, CreatedAt: now},
	}

	result := FormatEventsAsText(events, nil)
	if !strings.Contains(result, "Turn 9999") {
		t.Errorf("expected Turn 9999, got: %s", result)
	}
}

func TestFormatEventsAsText_MaxTextLengthEdgeCases(t *testing.T) {
	now := time.Now()

	// Test with MaxTextLength exactly matching text length
	events := []*TaskEventRecord{
		{ID: 1, StreamType: "turn_start", TurnNum: 1, CreatedAt: now},
		{ID: 2, StreamType: "text", TurnNum: 1, Text: "exact", CreatedAt: now},
		{ID: 3, StreamType: "turn_end", TurnNum: 1, CreatedAt: now},
	}

	opts := &FormatOptions{MaxTextLength: 5}
	result := FormatEventsAsText(events, opts)
	if !strings.Contains(result, "exact") {
		t.Errorf("expected exact text when MaxTextLength matches, got: %s", result)
	}
	if strings.Contains(result, "...") {
		t.Errorf("should not truncate when MaxTextLength equals text length, got: %s", result)
	}

	// Test with MaxTextLength = 1 (extreme truncation)
	opts.MaxTextLength = 1
	events[1].Text = "hello"
	result = FormatEventsAsText(events, opts)
	if !strings.Contains(result, "...") {
		t.Errorf("expected truncation with MaxTextLength=1, got: %s", result)
	}
}

func TestFormatEventsAsText_UnknownStreamType(t *testing.T) {
	now := time.Now()
	events := []*TaskEventRecord{
		{ID: 1, StreamType: "turn_start", TurnNum: 1, CreatedAt: now},
		{ID: 2, StreamType: "unknown_type", TurnNum: 1, Text: "unknown", CreatedAt: now},
		{ID: 3, StreamType: "turn_end", TurnNum: 1, CreatedAt: now},
	}

	result := FormatEventsAsText(events, nil)
	// Unknown types should be silently ignored
	if strings.Contains(result, "unknown") {
		t.Errorf("expected unknown types to be ignored, got: %s", result)
	}
}

func TestFilterEvents_EmptyTypeFilter(t *testing.T) {
	now := time.Now()
	events := []*TaskEventRecord{
		{ID: 1, StreamType: "text", TurnNum: 1, CreatedAt: now},
		{ID: 2, StreamType: "tool_use", TurnNum: 1, CreatedAt: now},
	}

	opts := &FormatOptions{TypeFilter: []string{}}
	filtered := filterEvents(events, opts)

	if len(filtered) != 2 {
		t.Errorf("expected empty TypeFilter to match all, got %d", len(filtered))
	}
}

func TestFilterEvents_TurnFilterZero(t *testing.T) {
	now := time.Now()
	events := []*TaskEventRecord{
		{ID: 1, StreamType: "text", TurnNum: 1, CreatedAt: now},
		{ID: 2, StreamType: "text", TurnNum: 2, CreatedAt: now},
	}

	opts := &FormatOptions{TurnFilter: 0, TypeFilter: nil}
	filtered := filterEvents(events, opts)

	if len(filtered) != 2 {
		t.Errorf("expected TurnFilter=0 to match all, got %d", len(filtered))
	}
}

func TestCountTurns_WithGaps(t *testing.T) {
	events := []*TaskEventRecord{
		{TurnNum: 1},
		{TurnNum: 3}, // Gap: no turn 2
		{TurnNum: 5}, // Gap: no turn 4
	}

	count := CountTurns(events)
	if count != 5 {
		t.Errorf("expected 5 (max turn number), got %d", count)
	}
}

func TestCountTurns_DuplicateTurns(t *testing.T) {
	events := []*TaskEventRecord{
		{TurnNum: 1},
		{TurnNum: 1},
		{TurnNum: 1},
	}

	count := CountTurns(events)
	if count != 1 {
		t.Errorf("expected 1 (max turn number), got %d", count)
	}
}

func TestGetTurnTimestamp_MultipleEventsInTurn(t *testing.T) {
	now := time.Now()
	later := now.Add(time.Hour)
	events := []*TaskEventRecord{
		{TurnNum: 1, CreatedAt: now},
		{TurnNum: 1, CreatedAt: later}, // Later timestamp in same turn
	}

	ts := GetTurnTimestamp(events, 1)
	if ts == nil {
		t.Fatal("expected timestamp, got nil")
	}
	// Should return FIRST timestamp in turn
	if !ts.Equal(now) {
		t.Errorf("expected first timestamp %v, got %v", now, ts)
	}
}

func TestSummarizeEvents_NoEvents(t *testing.T) {
	summary := SummarizeEvents([]*TaskEventRecord{})
	expected := "0 turns, 0 tool calls, 0 chars of text"
	if summary != expected {
		t.Errorf("expected %q, got %q", expected, summary)
	}
}

func TestSummarizeEvents_OnlyToolsNoText(t *testing.T) {
	events := []*TaskEventRecord{
		{StreamType: "tool_use", TurnNum: 1},
		{StreamType: "tool_use", TurnNum: 1},
		{StreamType: "text", TurnNum: 1, Text: ""},
	}

	summary := SummarizeEvents(events)
	if !strings.Contains(summary, "2 tool calls") {
		t.Errorf("expected 2 tool calls, got: %s", summary)
	}
	if !strings.Contains(summary, "0 chars of text") {
		t.Errorf("expected 0 chars, got: %s", summary)
	}
}

func TestFormatEventsAsJSON_NoTurnsField(t *testing.T) {
	events := []*TaskEventRecord{
		{StreamType: "text", Text: "hello"}, // No TurnNum set
	}

	resp, err := FormatEventsAsJSON("task-123", events, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if resp.TotalTurns != 0 {
		t.Errorf("expected 0 turns for events without TurnNum, got %d", resp.TotalTurns)
	}
}

func TestFormatToolUse_EmptyJSONObject(t *testing.T) {
	event := &TaskEventRecord{
		StreamType: "tool_use",
		ToolName:   "Test",
		ToolInput:  "{}",
	}

	opts := &FormatOptions{ShowToolInputs: true}
	result := formatToolUse(event, opts)

	if !strings.Contains(result, "[TOOL] Test") {
		t.Errorf("expected tool name, got: %s", result)
	}
}

func TestFormatToolUse_JSONWithComplexValues(t *testing.T) {
	event := &TaskEventRecord{
		StreamType: "tool_use",
		ToolName:   "Complex",
		ToolInput:  `{"nested": {"key": "value"}, "array": [1, 2, 3]}`,
	}

	opts := &FormatOptions{ShowToolInputs: true}
	result := formatToolUse(event, opts)

	if !strings.Contains(result, "[TOOL] Complex") {
		t.Errorf("expected tool name, got: %s", result)
	}
}

func TestFormatEventsAsText_AllEventTypesTogetherInTurn(t *testing.T) {
	now := time.Now()
	events := []*TaskEventRecord{
		{ID: 1, StreamType: "turn_start", TurnNum: 1, CreatedAt: now},
		{ID: 2, StreamType: "text", TurnNum: 1, Text: "text", CreatedAt: now},
		{ID: 3, StreamType: "tool_use", TurnNum: 1, ToolName: "Read", ToolInput: `{}`, CreatedAt: now},
		{ID: 4, StreamType: "tool_result", TurnNum: 1, ToolOutput: "result", CreatedAt: now},
		{ID: 5, StreamType: "error", TurnNum: 1, ErrorMsg: "error", CreatedAt: now},
		{ID: 6, StreamType: "human_feedback", TurnNum: 1, Text: "feedback", CreatedAt: now},
		{ID: 7, StreamType: "human_approval", TurnNum: 1, Text: "approval", CreatedAt: now},
		{ID: 8, StreamType: "status", TurnNum: 1, Text: "status", CreatedAt: now},
		{ID: 9, StreamType: "turn_end", TurnNum: 1, CreatedAt: now},
	}

	result := FormatEventsAsText(events, nil)

	if !strings.Contains(result, "Turn 1") {
		t.Errorf("expected Turn 1, got: %s", result)
	}
	if !strings.Contains(result, "[TOOL] Read") {
		t.Errorf("expected tool, got: %s", result)
	}
	if !strings.Contains(result, "[ERROR]") {
		t.Errorf("expected error, got: %s", result)
	}
	if !strings.Contains(result, "[HUMAN FEEDBACK]") {
		t.Errorf("expected feedback, got: %s", result)
	}
	if !strings.Contains(result, "[APPROVED]") {
		t.Errorf("expected approval, got: %s", result)
	}
	// Status should not appear in text output
	if strings.Contains(result, "status") {
		t.Errorf("expected status to be skipped, got: %s", result)
	}
}

func TestFormatOptions_MutationDoesntAffectGlobal(t *testing.T) {
	opts1 := DefaultFormatOptions()
	opts2 := DefaultFormatOptions()

	opts1.ShowTimestamps = false
	opts1.MaxTextLength = 999

	if !opts2.ShowTimestamps {
		t.Errorf("expected opts2 to be independent, ShowTimestamps affected")
	}
	if opts2.MaxTextLength != 0 {
		t.Errorf("expected opts2 to be independent, MaxTextLength affected")
	}
}

func TestFilterEvents_MultipleTypes(t *testing.T) {
	now := time.Now()
	events := []*TaskEventRecord{
		{ID: 1, StreamType: "text", TurnNum: 1, CreatedAt: now},
		{ID: 2, StreamType: "tool_use", TurnNum: 1, CreatedAt: now},
		{ID: 3, StreamType: "error", TurnNum: 1, CreatedAt: now},
		{ID: 4, StreamType: "text", TurnNum: 2, CreatedAt: now},
	}

	opts := &FormatOptions{TypeFilter: []string{"text", "error"}}
	filtered := filterEvents(events, opts)

	if len(filtered) != 3 {
		t.Errorf("expected 3 filtered events, got %d", len(filtered))
	}

	for _, event := range filtered {
		if event.StreamType != "text" && event.StreamType != "error" {
			t.Errorf("expected only text/error events, got %s", event.StreamType)
		}
	}
}
