package coordinator

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/sunholo/ailang/internal/messaging"
	"github.com/sunholo/ailang/internal/pubsub"
)

// CompletionHandler processes task completions and updates task status
// in the coordinator's store when Cloud Run Jobs finish.
// Completions arrive either via pull subscription (Start) or push HTTP endpoint
// (HandleCompletion).
type CompletionHandler struct {
	subscriber    *pubsub.Subscriber
	taskStore     Store
	msgStore      messaging.MessageStore // For posting completion notifications
	agentRegistry *AgentRegistry         // For checking skip_approval config
	logger        *log.Logger
}

// NewCompletionHandler creates a handler that processes task completions.
// msgStore and agentRegistry are optional — if nil, completion notifications
// and skip_approval handling are disabled (backwards compatible).
func NewCompletionHandler(subscriber *pubsub.Subscriber, taskStore Store, msgStore messaging.MessageStore, agentRegistry *AgentRegistry, logger *log.Logger) *CompletionHandler {
	return &CompletionHandler{
		subscriber:    subscriber,
		taskStore:     taskStore,
		msgStore:      msgStore,
		agentRegistry: agentRegistry,
		logger:        logger,
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

	// Build ExecuteResult from completion metrics so they're stored in Firestore.
	var execResult *ExecuteResult
	if completion.SessionID != "" || completion.NumTurns > 0 || completion.ToolCallCount > 0 {
		execResult = &ExecuteResult{
			Success:       completion.Status == "completed",
			SessionID:     completion.SessionID,
			NumTurns:      completion.NumTurns,
			ToolCallCount: completion.ToolCallCount,
			InputTokens:   completion.InputTokens,
			OutputTokens:  completion.OutputTokens,
			TokensUsed:    completion.InputTokens + completion.OutputTokens,
			Cost:          completion.CostUSD,
			Duration:      time.Duration(completion.DurationMS) * time.Millisecond,
		}
		if completion.ErrorMsg != "" {
			execResult.Error = completion.ErrorMsg
		}
	}

	switch completion.Status {
	case "completed":
		// Check if agent is configured to skip approval.
		skipApproval := false
		if h.agentRegistry != nil {
			if agent := h.agentRegistry.GetAgentByID(completion.AgentID); agent != nil {
				skipApproval = agent.SkipApproval
			}
		}

		if skipApproval {
			// Skip approval — mark completed directly.
			if err := h.taskStore.MarkTaskCompleted(ctx, completion.TaskID, execResult); err != nil {
				return fmt.Errorf("mark task completed: %w", err)
			}
			h.logger.Printf("CompletionHandler: task %s → completed (skip_approval, branch=%s, turns=%d, tools=%d)",
				completion.TaskID, completion.BranchName, completion.NumTurns, completion.ToolCallCount)
		} else {
			// Standard flow: mark pending_approval for human review.
			if err := h.taskStore.MarkTaskPendingApproval(ctx, completion.TaskID, "", completion.BranchName, "", "", execResult); err != nil {
				return fmt.Errorf("mark task pending_approval: %w", err)
			}
			h.logger.Printf("CompletionHandler: task %s → pending_approval (branch=%s, turns=%d, tools=%d)",
				completion.TaskID, completion.BranchName, completion.NumTurns, completion.ToolCallCount)
		}

		// Post completion notification to the agent's inbox so the portal/sidecar can detect it.
		h.postCompletionNotification(ctx, task, completion)

	case "failed":
		if err := h.taskStore.MarkTaskFailed(ctx, completion.TaskID, fmt.Errorf("%s", completion.ErrorMsg)); err != nil {
			return fmt.Errorf("mark task failed: %w", err)
		}
		h.logger.Printf("CompletionHandler: task %s → failed (error=%s)",
			completion.TaskID, completion.ErrorMsg)

		// Post failure notification too.
		h.postCompletionNotification(ctx, task, completion)

	default:
		h.logger.Printf("CompletionHandler: unknown completion status %q for task %s",
			completion.Status, completion.TaskID)
	}

	return nil
}

// postCompletionNotification sends a message to the agent's inbox with the
// completion status. This allows external clients (portal, sidecar) to poll
// for task completion via GET /api/messages.
func (h *CompletionHandler) postCompletionNotification(ctx context.Context, task *TaskRecord, completion pubsub.TaskCompletion) {
	if h.msgStore == nil {
		return
	}

	payload, _ := json.Marshal(map[string]string{
		"task_id":     completion.TaskID,
		"agent_id":    completion.AgentID,
		"status":      completion.Status,
		"branch_name": completion.BranchName,
		"error_msg":   completion.ErrorMsg,
	})

	msg := &messaging.InboxMessage{
		FromAgent:     completion.AgentID,
		ToInbox:       task.AgentID, // Same inbox as the agent
		MessageType:   "completion",
		Title:         fmt.Sprintf("Task %s: %s", completion.TaskID, completion.Status),
		Payload:       string(payload),
		CorrelationID: task.MessageID, // Links back to the original request message
	}

	if err := h.msgStore.InsertInboxMessage(msg); err != nil {
		h.logger.Printf("CompletionHandler: failed to post completion notification for task %s: %v",
			completion.TaskID, err)
	} else {
		h.logger.Printf("CompletionHandler: posted completion notification for task %s (correlation=%s)",
			completion.TaskID, task.MessageID)
	}
}
