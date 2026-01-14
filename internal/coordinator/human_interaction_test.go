package coordinator

import (
	"testing"
	"time"
)

// TestPrepareTaskForRetrigger_Iteration1 tests first retry (iteration 1 -> 2).
func TestPrepareTaskForRetrigger_Iteration1(t *testing.T) {
	task := &TaskRecord{
		ID:        "task-12345678",
		Content:   "Original task content",
		Status:    TaskStatusPendingApproval,
		Iteration: 1, // First run
	}

	feedback := "Need to add error handling for edge cases"
	PrepareTaskForRetrigger(task, feedback)

	// Verify iteration incremented
	if task.Iteration != 2 {
		t.Errorf("Iteration = %d, want 2", task.Iteration)
	}

	// Verify status reset
	if task.Status != TaskStatusQueued {
		t.Errorf("Status = %s, want %s", task.Status, TaskStatusQueued)
	}

	// Verify feedback appended
	if task.Content == "Original task content" {
		t.Error("Content should be modified with feedback")
	}

	// Verify can still retrigger
	if !CanRetrigger(task) {
		t.Error("Should be able to retrigger at iteration 2")
	}
}

// TestPrepareTaskForRetrigger_Iteration2 tests second retry (iteration 2 -> 3).
func TestPrepareTaskForRetrigger_Iteration2(t *testing.T) {
	task := &TaskRecord{
		ID:        "task-23456789",
		Content:   "Task with first feedback\n\n[HUMAN FEEDBACK - Iteration 2]\nPrevious feedback",
		Status:    TaskStatusPendingApproval,
		Iteration: 2, // First retry
	}

	feedback := "Still needs better test coverage"
	PrepareTaskForRetrigger(task, feedback)

	// Verify iteration incremented to final
	if task.Iteration != 3 {
		t.Errorf("Iteration = %d, want 3 (final)", task.Iteration)
	}

	// Verify status reset
	if task.Status != TaskStatusQueued {
		t.Errorf("Status = %s, want %s", task.Status, TaskStatusQueued)
	}

	// At iteration 3, we've hit the limit - cannot retrigger again
	if CanRetrigger(task) {
		t.Error("Should NOT be able to retrigger at iteration 3 (max)")
	}
}

// TestPrepareTaskForRetrigger_Iteration3_MaxReached tests that iteration 3 is final.
func TestPrepareTaskForRetrigger_Iteration3_MaxReached(t *testing.T) {
	task := &TaskRecord{
		ID:        "task-34567890",
		Content:   "Task at max iteration",
		Status:    TaskStatusPendingApproval,
		Iteration: 3, // Max iteration
	}

	// Verify cannot retrigger
	if CanRetrigger(task) {
		t.Error("Should NOT be able to retrigger at iteration 3")
	}

	// If we try to prepare for retrigger anyway...
	PrepareTaskForRetrigger(task, "This should go to iteration 4")

	// Iteration would be 4 but CanRetrigger check should prevent this
	if task.Iteration != 4 {
		t.Errorf("Iteration = %d, want 4 (but would be blocked)", task.Iteration)
	}
}

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

// TestMaxIterations tests the constant value.
func TestMaxIterations(t *testing.T) {
	if MaxIterations != 3 {
		t.Errorf("MaxIterations = %d, want 3", MaxIterations)
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
