// Package coordinator provides human interaction handling for the feedback loop.
package coordinator

import (
	"context"
	"fmt"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

var humanTracer = otel.Tracer("coordinator.human")

// HumanFeedback represents feedback provided by a human reviewer.
type HumanFeedback struct {
	TaskID    string    // Task being reviewed
	Iteration int       // Current iteration of the task
	Feedback  string    // Human's feedback text
	Action    string    // "reject" or "approve"
	Timestamp time.Time // When feedback was provided
	UserID    string    // Who provided feedback (optional)
}

// StoreFeedbackEvent stores human feedback as a task event for audit trail.
func StoreFeedbackEvent(ctx context.Context, store Store, feedback *HumanFeedback) error {
	// Create OTEL span for human feedback
	ctx, span := humanTracer.Start(ctx, "human.feedback",
		trace.WithAttributes(
			attribute.String("task.id", feedback.TaskID),
			attribute.Int("task.iteration", feedback.Iteration),
			attribute.String("feedback.action", feedback.Action),
			attribute.String("feedback.text", truncateForSpan(feedback.Feedback, 200)),
		),
	)
	defer span.End()

	// Store as task event
	event := &TaskEventRecord{
		TaskID:     feedback.TaskID,
		StreamType: "human_feedback",
		Text:       feedback.Feedback,
		TurnNum:    feedback.Iteration, // Use iteration as turn for context
		Status:     feedback.Action,
		CreatedAt:  feedback.Timestamp,
	}

	if err := store.StoreTaskEvent(ctx, event); err != nil {
		span.RecordError(err)
		return fmt.Errorf("failed to store feedback event: %w", err)
	}

	return nil
}

// StoreApprovalEvent stores human approval as a task event.
func StoreApprovalEvent(ctx context.Context, store Store, taskID string, approvedBy string) error {
	// Create OTEL span for human approval
	ctx, span := humanTracer.Start(ctx, "human.approval",
		trace.WithAttributes(
			attribute.String("task.id", taskID),
			attribute.String("approved.by", approvedBy),
		),
	)
	defer span.End()

	event := &TaskEventRecord{
		TaskID:     taskID,
		StreamType: "human_approval",
		Text:       fmt.Sprintf("Approved by %s", approvedBy),
		Status:     "approved",
		CreatedAt:  time.Now(),
	}

	if err := store.StoreTaskEvent(ctx, event); err != nil {
		span.RecordError(err)
		return fmt.Errorf("failed to store approval event: %w", err)
	}

	return nil
}

// StoreIterationStartEvent marks the start of a new task iteration.
func StoreIterationStartEvent(ctx context.Context, store Store, taskID string, iteration int) error {
	ctx, span := humanTracer.Start(ctx, "task.iteration_start",
		trace.WithAttributes(
			attribute.String("task.id", taskID),
			attribute.Int("task.iteration", iteration),
		),
	)
	defer span.End()

	event := &TaskEventRecord{
		TaskID:     taskID,
		StreamType: "iteration_start",
		TurnNum:    iteration,
		Text:       fmt.Sprintf("Iteration %d started", iteration),
		CreatedAt:  time.Now(),
	}

	if err := store.StoreTaskEvent(ctx, event); err != nil {
		span.RecordError(err)
		return fmt.Errorf("failed to store iteration start event: %w", err)
	}

	return nil
}

// PrepareTaskForRetrigger prepares feedback content for a re-triggered task.
// NOTE (M-TASK-HIERARCHY): This function no longer modifies task status.
// The new workflow creates a NEW task via the message system with parent_task_id linking,
// and the old task is marked as "rejected" separately via store.MarkTaskRejected().
//
// This function now only:
// - Increments iteration count for tracking
// - Appends feedback to content for context propagation
func PrepareTaskForRetrigger(task *TaskRecord, feedback string) {
	// Increment iteration
	if task.Iteration == 0 {
		task.Iteration = 1 // First run was iteration 1
	}
	task.Iteration++

	// NOTE: Status is NOT modified here. The caller is responsible for:
	// 1. Marking this task as "rejected" via store.MarkTaskRejected()
	// 2. Creating a NEW task via the message system with parent_task_id linking

	// Clear previous completion data (still useful for tracking)
	task.CompletedAt = nil
	task.Error = ""
	task.Output = ""

	// Append feedback to content for context
	if feedback != "" {
		task.Content = fmt.Sprintf("%s\n\n[HUMAN FEEDBACK - Iteration %d]\n%s",
			task.Content, task.Iteration, feedback)
	}
}

// MaxIterations is the limit to prevent infinite feedback loops.
const MaxIterations = 3

// CanRetrigger checks if a task can be re-triggered (hasn't exceeded iteration limit).
func CanRetrigger(task *TaskRecord) bool {
	iteration := task.Iteration
	if iteration == 0 {
		iteration = 1
	}
	return iteration < MaxIterations
}

// truncateForSpan truncates text for OTEL span attributes.
func truncateForSpan(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}
