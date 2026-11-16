package coordinator

import (
	"testing"
	"time"
)

// TestCanRetrigger_NewTask tests task with iteration 0 (not yet started).
func TestCanRetrigger_NewTask(t *testing.T) {
	task := &TaskRecord{
		ID:        "task-new",
		Iteration: 0, // Not started
	}

	// Should be able to retrigger (iteration 0 is treated as 1)
	if !CanRetrigger(task) {
		t.Error("New task (iteration 0) should be able to retrigger")
	}
}

// TestHumanFeedback_Struct tests the HumanFeedback struct fields.
func TestHumanFeedback_Struct(t *testing.T) {
	feedback := &HumanFeedback{
		TaskID:    "task-feedback-test",
		Iteration: 2,
		Feedback:  "Need to improve error messages",
		Action:    "reject",
		Timestamp: time.Now(),
		UserID:    "cli-user",
	}

	if feedback.TaskID != "task-feedback-test" {
		t.Error("TaskID not set correctly")
	}
	if feedback.Iteration != 2 {
		t.Error("Iteration not set correctly")
	}
	if feedback.Action != "reject" {
		t.Error("Action not set correctly")
	}
}

// TestMaxAgentIterations tests the constant value.
func TestMaxAgentIterations(t *testing.T) {
	if MaxAgentIterations != 3 {
		t.Errorf("MaxAgentIterations = %d, want 3", MaxAgentIterations)
	}
}

// TestTruncateForSpan tests text truncation for OTEL attributes.
func TestTruncateForSpan(t *testing.T) {
	tests := []struct {
		name   string
		input  string
		maxLen int
		want   string
	}{
		{
			name:   "short text",
			input:  "short",
			maxLen: 10,
			want:   "short",
		},
		{
			name:   "exact length",
			input:  "exactly10c",
			maxLen: 10,
			want:   "exactly10c",
		},
		{
			name:   "needs truncation",
			input:  "this is a long feedback message that exceeds the limit",
			maxLen: 20,
			want:   "this is a long feedb...",
		},
		{
			name:   "empty string",
			input:  "",
			maxLen: 10,
			want:   "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := truncateForSpan(tt.input, tt.maxLen)
			if got != tt.want {
				t.Errorf("truncateForSpan(%q, %d) = %q, want %q",
					tt.input, tt.maxLen, got, tt.want)
			}
		})
	}
}
