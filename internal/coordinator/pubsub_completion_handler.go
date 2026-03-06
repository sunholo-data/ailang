package coordinator

import (
	"context"
	"fmt"
	"log"

	"github.com/sunholo/ailang/internal/pubsub"
)

// CompletionHandler processes task completions and updates task status
// in the coordinator's store when Cloud Run Jobs finish.
// Completions arrive either via pull subscription (Start) or push HTTP endpoint
// (HandleCompletion).
type CompletionHandler struct {
	subscriber *pubsub.Subscriber
	taskStore  Store
	logger     *log.Logger
}

// NewCompletionHandler creates a handler that processes task completions.
func NewCompletionHandler(subscriber *pubsub.Subscriber, taskStore Store, logger *log.Logger) *CompletionHandler {
	return &CompletionHandler{
		subscriber: subscriber,
		taskStore:  taskStore,
		logger:     logger,
	}
}

// HandleCompletion processes a task completion from either pull subscription
// or push HTTP endpoint. Returns nil to ack bad data (prevent retry loops).
func (h *CompletionHandler) HandleCompletion(data []byte, attrs map[string]string) error {
	completion, err := pubsub.DecodeTaskCompletion(data)
	if err != nil {
		h.logger.Printf("CompletionHandler: failed to decode completion: %v", err)
		return nil // Ack to avoid retry loop on bad data.
	}

	h.logger.Printf("CompletionHandler: received completion for task %s (status=%s, agent=%s)",
		completion.TaskID, completion.Status, completion.AgentID)

	if err := h.handleCompletion(context.Background(), completion); err != nil {
		h.logger.Printf("CompletionHandler: failed to handle completion for task %s: %v",
			completion.TaskID, err)
		return fmt.Errorf("handle completion: %w", err)
	}

	return nil
}

// Start begins listening for completions via pull subscription in the background.
// Not used in push mode — the HTTP handler calls HandleCompletion() directly.
func (h *CompletionHandler) Start(ctx context.Context) {
	go func() {
		err := h.subscriber.Subscribe(ctx, pubsub.SubCompletionsCoordinator, func(ctx context.Context, data []byte, attrs map[string]string) error {
			return h.HandleCompletion(data, attrs)
		})
		if err != nil && ctx.Err() == nil {
			h.logger.Printf("CompletionHandler: subscription error: %v", err)
		}
	}()
}

// handleCompletion updates the task store based on the completion status.
func (h *CompletionHandler) handleCompletion(ctx context.Context, completion pubsub.TaskCompletion) error {
	task, err := h.taskStore.GetTask(ctx, completion.TaskID)
	if err != nil {
		return fmt.Errorf("get task %s: %w", completion.TaskID, err)
	}
	if task == nil {
		h.logger.Printf("CompletionHandler: task %s not found, ignoring completion", completion.TaskID)
		return nil // Idempotent: task may not exist in this coordinator.
	}

	// Idempotency: skip if task is already in a terminal state.
	if task.Status == "completed" || task.Status == "failed" || task.Status == "cancelled" {
		h.logger.Printf("CompletionHandler: task %s already in terminal state %q, skipping",
			completion.TaskID, task.Status)
		return nil
	}

	switch completion.Status {
	case "completed":
		if err := h.taskStore.MarkTaskPendingApproval(ctx, completion.TaskID, "", completion.BranchName, "", "", nil); err != nil {
			return fmt.Errorf("mark task pending_approval: %w", err)
		}
		h.logger.Printf("CompletionHandler: task %s → pending_approval (branch=%s)",
			completion.TaskID, completion.BranchName)

	case "failed":
		if err := h.taskStore.MarkTaskFailed(ctx, completion.TaskID, fmt.Errorf("%s", completion.ErrorMsg)); err != nil {
			return fmt.Errorf("mark task failed: %w", err)
		}
		h.logger.Printf("CompletionHandler: task %s → failed (error=%s)",
			completion.TaskID, completion.ErrorMsg)

	default:
		h.logger.Printf("CompletionHandler: unknown completion status %q for task %s",
			completion.Status, completion.TaskID)
	}

	return nil
}
