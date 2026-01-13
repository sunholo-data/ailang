// Package coordinator provides task coordination and event formatting.
package coordinator

import (
	"encoding/json"
	"strings"
	"testing"
	"time"
)

func TestFormatEventsAsText_EmptyEvents(t *testing.T) {
	result := FormatEventsAsText(nil, nil)
	if result != "No events recorded for this task." {
		t.Errorf("expected empty message, got: %s", result)
	}

	result = FormatEventsAsText([]*TaskEventRecord{}, nil)
	if result != "No events recorded for this task." {
		t.Errorf("expected empty message, got: %s", result)
	}
}

func TestFormatEventsAsText_SingleTurn(t *testing.T) {
	now := time.Now()
	events := []*TaskEventRecord{
		{ID: 1, StreamType: "turn_start", TurnNum: 1, CreatedAt: now},
		{ID: 2, StreamType: "text", TurnNum: 1, Text: "Hello, world!", CreatedAt: now},
		{ID: 3, StreamType: "turn_end", TurnNum: 1, CreatedAt: now},
	}

	result := FormatEventsAsText(events, nil)

	if !strings.Contains(result, "Turn 1") {
		t.Errorf("expected Turn 1 header, got: %s", result)
	}
	if !strings.Contains(result, "Hello, world!") {
		t.Errorf("expected text content, got: %s", result)
	}
}

func TestFormatEventsAsText_MultipleTurns(t *testing.T) {
	now := time.Now()
	events := []*TaskEventRecord{
		{ID: 1, StreamType: "turn_start", TurnNum: 1, CreatedAt: now},
		{ID: 2, StreamType: "text", TurnNum: 1, Text: "First turn", CreatedAt: now},
		{ID: 3, StreamType: "turn_end", TurnNum: 1, CreatedAt: now},
		{ID: 4, StreamType: "turn_start", TurnNum: 2, CreatedAt: now},
		{ID: 5, StreamType: "text", TurnNum: 2, Text: "Second turn", CreatedAt: now},
		{ID: 6, StreamType: "turn_end", TurnNum: 2, CreatedAt: now},
	}

	result := FormatEventsAsText(events, nil)

	if !strings.Contains(result, "Turn 1") {
		t.Errorf("expected Turn 1 header, got: %s", result)
	}
	if !strings.Contains(result, "Turn 2") {
		t.Errorf("expected Turn 2 header, got: %s", result)
	}
	if !strings.Contains(result, "First turn") {
		t.Errorf("expected first turn content, got: %s", result)
	}
	if !strings.Contains(result, "Second turn") {
		t.Errorf("expected second turn content, got: %s", result)
	}
}

func TestFormatEventsAsText_ToolUse(t *testing.T) {
	now := time.Now()
	events := []*TaskEventRecord{
		{ID: 1, StreamType: "turn_start", TurnNum: 1, CreatedAt: now},
		{ID: 2, StreamType: "tool_use", TurnNum: 1, ToolName: "Read", ToolInput: `{"file_path":"/tmp/test.go"}`, CreatedAt: now},
		{ID: 3, StreamType: "tool_result", TurnNum: 1, ToolOutput: "file contents here", CreatedAt: now},
		{ID: 4, StreamType: "turn_end", TurnNum: 1, CreatedAt: now},
	}

	opts := &FormatOptions{ShowToolInputs: true}
	result := FormatEventsAsText(events, opts)

	if !strings.Contains(result, "[TOOL] Read") {
		t.Errorf("expected tool name, got: %s", result)
	}
	if !strings.Contains(result, "file_path") {
		t.Errorf("expected tool input, got: %s", result)
	}
	if !strings.Contains(result, "Result:") {
		t.Errorf("expected tool result, got: %s", result)
	}
}

func TestFormatEventsAsText_ToolUseWithoutInputs(t *testing.T) {
	now := time.Now()
	events := []*TaskEventRecord{
		{ID: 1, StreamType: "turn_start", TurnNum: 1, CreatedAt: now},
		{ID: 2, StreamType: "tool_use", TurnNum: 1, ToolName: "Read", ToolInput: `{"file_path":"/tmp/test.go"}`, CreatedAt: now},
		{ID: 3, StreamType: "turn_end", TurnNum: 1, CreatedAt: now},
	}

	opts := &FormatOptions{ShowToolInputs: false}
	result := FormatEventsAsText(events, opts)

	if !strings.Contains(result, "[TOOL] Read") {
		t.Errorf("expected tool name, got: %s", result)
	}
	if strings.Contains(result, "file_path") {
		t.Errorf("expected no tool input when ShowToolInputs=false, got: %s", result)
	}
}

func TestFormatEventsAsText_ErrorEvent(t *testing.T) {
	now := time.Now()
	events := []*TaskEventRecord{
		{ID: 1, StreamType: "turn_start", TurnNum: 1, CreatedAt: now},
		{ID: 2, StreamType: "error", TurnNum: 1, ErrorMsg: "something went wrong", CreatedAt: now},
		{ID: 3, StreamType: "turn_end", TurnNum: 1, CreatedAt: now},
	}

	result := FormatEventsAsText(events, nil)

	if !strings.Contains(result, "[ERROR]") {
		t.Errorf("expected error marker, got: %s", result)
	}
	if !strings.Contains(result, "something went wrong") {
		t.Errorf("expected error message, got: %s", result)
	}
}

func TestFormatEventsAsText_HumanFeedback(t *testing.T) {
	now := time.Now()
	events := []*TaskEventRecord{
		{ID: 1, StreamType: "turn_start", TurnNum: 1, CreatedAt: now},
		{ID: 2, StreamType: "human_feedback", TurnNum: 1, Text: "Please fix the bug in parser.go", CreatedAt: now},
		{ID: 3, StreamType: "turn_end", TurnNum: 1, CreatedAt: now},
	}

	result := FormatEventsAsText(events, nil)

	if !strings.Contains(result, "[HUMAN FEEDBACK]") {
		t.Errorf("expected human feedback marker, got: %s", result)
	}
	if !strings.Contains(result, "Please fix the bug") {
		t.Errorf("expected feedback content, got: %s", result)
	}
}

func TestFormatEventsAsText_HumanApproval(t *testing.T) {
	now := time.Now()
	events := []*TaskEventRecord{
		{ID: 1, StreamType: "turn_start", TurnNum: 1, CreatedAt: now},
		{ID: 2, StreamType: "human_approval", TurnNum: 1, Text: "Approved by mark", CreatedAt: now},
		{ID: 3, StreamType: "turn_end", TurnNum: 1, CreatedAt: now},
	}

	result := FormatEventsAsText(events, nil)

	if !strings.Contains(result, "[APPROVED]") {
		t.Errorf("expected approval marker, got: %s", result)
	}
	if !strings.Contains(result, "Approved by mark") {
		t.Errorf("expected approval content, got: %s", result)
	}
}

func TestFormatEventsAsText_IterationStart(t *testing.T) {
	now := time.Now()
	events := []*TaskEventRecord{
		{ID: 1, StreamType: "iteration_start", TurnNum: 2, CreatedAt: now},
		{ID: 2, StreamType: "turn_start", TurnNum: 2, CreatedAt: now},
		{ID: 3, StreamType: "text", TurnNum: 2, Text: "Continuing work", CreatedAt: now},
		{ID: 4, StreamType: "turn_end", TurnNum: 2, CreatedAt: now},
	}

	result := FormatEventsAsText(events, nil)

	if !strings.Contains(result, "[ITERATION 2 STARTED]") {
		t.Errorf("expected iteration marker, got: %s", result)
	}
}

func TestFormatEventsAsText_TextTruncation(t *testing.T) {
	now := time.Now()
	longText := strings.Repeat("a", 500)
	events := []*TaskEventRecord{
		{ID: 1, StreamType: "turn_start", TurnNum: 1, CreatedAt: now},
		{ID: 2, StreamType: "text", TurnNum: 1, Text: longText, CreatedAt: now},
		{ID: 3, StreamType: "turn_end", TurnNum: 1, CreatedAt: now},
	}

	opts := &FormatOptions{MaxTextLength: 100}
	result := FormatEventsAsText(events, opts)

	if !strings.Contains(result, "...") {
		t.Errorf("expected truncation indicator, got: %s", result)
	}
	if strings.Contains(result, strings.Repeat("a", 200)) {
		t.Errorf("expected truncated text, got full text")
	}
}

func TestFilterEvents_TurnFilter(t *testing.T) {
	now := time.Now()
	events := []*TaskEventRecord{
		{ID: 1, StreamType: "text", TurnNum: 1, Text: "Turn 1", CreatedAt: now},
		{ID: 2, StreamType: "text", TurnNum: 2, Text: "Turn 2", CreatedAt: now},
		{ID: 3, StreamType: "text", TurnNum: 3, Text: "Turn 3", CreatedAt: now},
	}

	opts := &FormatOptions{TurnFilter: 2}
	filtered := filterEvents(events, opts)

	if len(filtered) != 1 {
		t.Errorf("expected 1 event, got %d", len(filtered))
	}
	if filtered[0].TurnNum != 2 {
		t.Errorf("expected turn 2, got turn %d", filtered[0].TurnNum)
	}
}

func TestFilterEvents_TypeFilter(t *testing.T) {
	now := time.Now()
	events := []*TaskEventRecord{
		{ID: 1, StreamType: "text", TurnNum: 1, Text: "Hello", CreatedAt: now},
		{ID: 2, StreamType: "tool_use", TurnNum: 1, ToolName: "Read", CreatedAt: now},
		{ID: 3, StreamType: "text", TurnNum: 1, Text: "World", CreatedAt: now},
	}

	opts := &FormatOptions{TypeFilter: []string{"tool_use"}}
	filtered := filterEvents(events, opts)

	if len(filtered) != 1 {
		t.Errorf("expected 1 event, got %d", len(filtered))
	}
	if filtered[0].StreamType != "tool_use" {
		t.Errorf("expected tool_use, got %s", filtered[0].StreamType)
	}
}

func TestFilterEvents_CombinedFilters(t *testing.T) {
	now := time.Now()
	events := []*TaskEventRecord{
		{ID: 1, StreamType: "text", TurnNum: 1, CreatedAt: now},
		{ID: 2, StreamType: "tool_use", TurnNum: 1, CreatedAt: now},
		{ID: 3, StreamType: "text", TurnNum: 2, CreatedAt: now},
		{ID: 4, StreamType: "tool_use", TurnNum: 2, CreatedAt: now},
	}

	opts := &FormatOptions{TurnFilter: 2, TypeFilter: []string{"tool_use"}}
	filtered := filterEvents(events, opts)

	if len(filtered) != 1 {
		t.Errorf("expected 1 event, got %d", len(filtered))
	}
	if filtered[0].ID != 4 {
		t.Errorf("expected event ID 4, got %d", filtered[0].ID)
	}
}

func TestFilterEvents_NoFilter(t *testing.T) {
	now := time.Now()
	events := []*TaskEventRecord{
		{ID: 1, StreamType: "text", TurnNum: 1, CreatedAt: now},
		{ID: 2, StreamType: "text", TurnNum: 2, CreatedAt: now},
	}

	opts := &FormatOptions{TurnFilter: 0, TypeFilter: nil}
	filtered := filterEvents(events, opts)

	if len(filtered) != 2 {
		t.Errorf("expected 2 events (no filter), got %d", len(filtered))
	}
}

func TestFormatEventsAsJSON(t *testing.T) {
	now := time.Now()
	events := []*TaskEventRecord{
		{ID: 1, StreamType: "turn_start", TurnNum: 1, CreatedAt: now},
		{ID: 2, StreamType: "text", TurnNum: 1, Text: "Hello", CreatedAt: now},
		{ID: 3, StreamType: "turn_end", TurnNum: 1, CreatedAt: now},
		{ID: 4, StreamType: "turn_start", TurnNum: 2, CreatedAt: now},
		{ID: 5, StreamType: "text", TurnNum: 2, Text: "World", CreatedAt: now},
		{ID: 6, StreamType: "turn_end", TurnNum: 2, CreatedAt: now},
	}

	resp, err := FormatEventsAsJSON("task-123", events, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if resp.TaskID != "task-123" {
		t.Errorf("expected task-123, got %s", resp.TaskID)
	}
	if resp.TotalEvents != 6 {
		t.Errorf("expected 6 events, got %d", resp.TotalEvents)
	}
	if resp.TotalTurns != 2 {
		t.Errorf("expected 2 turns, got %d", resp.TotalTurns)
	}
	if len(resp.Events) != 6 {
		t.Errorf("expected 6 events in response, got %d", len(resp.Events))
	}
}

func TestFormatEventsAsJSON_WithFilter(t *testing.T) {
	now := time.Now()
	events := []*TaskEventRecord{
		{ID: 1, StreamType: "text", TurnNum: 1, Text: "Hello", CreatedAt: now},
		{ID: 2, StreamType: "tool_use", TurnNum: 1, ToolName: "Read", CreatedAt: now},
		{ID: 3, StreamType: "text", TurnNum: 2, Text: "World", CreatedAt: now},
	}

	opts := &FormatOptions{TypeFilter: []string{"text"}}
	resp, err := FormatEventsAsJSON("task-456", events, opts)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// TotalEvents should reflect original count
	if resp.TotalEvents != 3 {
		t.Errorf("expected 3 total events, got %d", resp.TotalEvents)
	}
	// But filtered events only returns matching
	if len(resp.Events) != 2 {
		t.Errorf("expected 2 filtered events, got %d", len(resp.Events))
	}
}

func TestFormatEventsAsJSON_Serializable(t *testing.T) {
	now := time.Now()
	events := []*TaskEventRecord{
		{ID: 1, StreamType: "text", TurnNum: 1, Text: "Test", CreatedAt: now},
	}

	resp, _ := FormatEventsAsJSON("task-789", events, nil)

	// Should be JSON serializable
	data, err := json.Marshal(resp)
	if err != nil {
		t.Fatalf("failed to marshal response: %v", err)
	}

	var decoded EventsResponse
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}

	if decoded.TaskID != "task-789" {
		t.Errorf("expected task-789 after round-trip, got %s", decoded.TaskID)
	}
}

func TestCountTurns(t *testing.T) {
	events := []*TaskEventRecord{
		{TurnNum: 1},
		{TurnNum: 1},
		{TurnNum: 2},
		{TurnNum: 3},
		{TurnNum: 3},
	}

	count := CountTurns(events)
	if count != 3 {
		t.Errorf("expected 3 turns, got %d", count)
	}
}

func TestCountTurns_Empty(t *testing.T) {
	count := CountTurns(nil)
	if count != 0 {
		t.Errorf("expected 0 turns for nil, got %d", count)
	}

	count = CountTurns([]*TaskEventRecord{})
	if count != 0 {
		t.Errorf("expected 0 turns for empty, got %d", count)
	}
}

func TestGetTurnTimestamp(t *testing.T) {
	now := time.Now()
	later := now.Add(time.Hour)
	events := []*TaskEventRecord{
		{TurnNum: 1, CreatedAt: now},
		{TurnNum: 1, CreatedAt: later},
		{TurnNum: 2, CreatedAt: later},
	}

	ts := GetTurnTimestamp(events, 1)
	if ts == nil {
		t.Fatal("expected timestamp, got nil")
	}
	if !ts.Equal(now) {
		t.Errorf("expected first event timestamp, got %v", ts)
	}
}

func TestGetTurnTimestamp_NotFound(t *testing.T) {
	events := []*TaskEventRecord{
		{TurnNum: 1, CreatedAt: time.Now()},
	}

	ts := GetTurnTimestamp(events, 99)
	if ts != nil {
		t.Errorf("expected nil for missing turn, got %v", ts)
	}
}

func TestSummarizeEvents(t *testing.T) {
	events := []*TaskEventRecord{
		{StreamType: "turn_start", TurnNum: 1},
		{StreamType: "text", TurnNum: 1, Text: "Hello"},
		{StreamType: "tool_use", TurnNum: 1},
		{StreamType: "text", TurnNum: 1, Text: " World"},
		{StreamType: "turn_end", TurnNum: 1},
		{StreamType: "turn_start", TurnNum: 2},
		{StreamType: "tool_use", TurnNum: 2},
		{StreamType: "turn_end", TurnNum: 2},
	}

	summary := SummarizeEvents(events)

	if !strings.Contains(summary, "2 turns") {
		t.Errorf("expected 2 turns in summary, got: %s", summary)
	}
	if !strings.Contains(summary, "2 tool calls") {
		t.Errorf("expected 2 tool calls in summary, got: %s", summary)
	}
	if !strings.Contains(summary, "11 chars") {
		t.Errorf("expected 11 chars in summary, got: %s", summary)
	}
}

func TestDefaultFormatOptions(t *testing.T) {
	opts := DefaultFormatOptions()

	if !opts.ShowTimestamps {
		t.Error("expected ShowTimestamps to be true")
	}
	if !opts.ShowToolInputs {
		t.Error("expected ShowToolInputs to be true")
	}
	if opts.MaxTextLength != 0 {
		t.Errorf("expected MaxTextLength 0, got %d", opts.MaxTextLength)
	}
	if opts.TurnFilter != 0 {
		t.Errorf("expected TurnFilter 0, got %d", opts.TurnFilter)
	}
	if opts.TypeFilter != nil {
		t.Errorf("expected TypeFilter nil, got %v", opts.TypeFilter)
	}
}

func TestFormatToolUse_NonJSONInput(t *testing.T) {
	event := &TaskEventRecord{
		StreamType: "tool_use",
		ToolName:   "Bash",
		ToolInput:  "ls -la /tmp",
	}

	opts := &FormatOptions{ShowToolInputs: true}
	result := formatToolUse(event, opts)

	if !strings.Contains(result, "[TOOL] Bash") {
		t.Errorf("expected tool name, got: %s", result)
	}
	if !strings.Contains(result, "input: ls -la") {
		t.Errorf("expected raw input, got: %s", result)
	}
}

func TestFormatToolUse_LongInput(t *testing.T) {
	longInput := strings.Repeat("x", 500)
	event := &TaskEventRecord{
		StreamType: "tool_use",
		ToolName:   "Write",
		ToolInput:  longInput,
	}

	opts := &FormatOptions{ShowToolInputs: true}
	result := formatToolUse(event, opts)

	// Should be truncated
	if !strings.Contains(result, "...") {
		t.Errorf("expected truncation, got: %s", result)
	}
}

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
	// Returns empty string since all events are filtered out
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
