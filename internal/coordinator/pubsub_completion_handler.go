package coordinator

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/sunholo-data/ailang/internal/messaging"
	"github.com/sunholo-data/ailang/internal/observatory"
	"github.com/sunholo-data/ailang/internal/pubsub"
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
	// obsBackend lets a cloud completion advance its chain and stage. Without it
	// this path wrote task status and nothing else, which is why every chain in
	// production sat "active" with a stage frozen at "pending" — 315 of them, the
	// oldest for four months (M-COMPLETION-PATH-PARITY).
	obsBackend observatory.Backend
	// instanceID names this coordinator on finalisation ledger claims.
	instanceID string
	logger     *log.Logger
}

// SetFinalizationDeps wires the collaborators cloud finalisation needs beyond the
// task store. Both are optional for construction but the completion path is
// materially degraded without them, and says so at startup rather than silently.
func (h *CompletionHandler) SetFinalizationDeps(obsBackend observatory.Backend, instanceID string) {
	h.obsBackend = obsBackend
	h.instanceID = instanceID
	if obsBackend == nil && h.logger != nil {
		h.logger.Printf("CompletionHandler: no observatory backend — cloud completions will not advance their chain or stage")
	}
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
	if IsTerminalStatus(task.Status) {
		h.logger.Printf("CompletionHandler: task %s already in terminal state %q, skipping",
			completion.TaskID, task.Status)
		return nil
	}

	// Build ExecuteResult from completion metrics so they're stored in Firestore.
	var execResult *ExecuteResult
	if completion.SessionID != "" || completion.NumTurns > 0 || completion.ToolCallCount > 0 || completion.ArtifactGCSPath != "" {
		execResult = &ExecuteResult{
			// A no_changes run is not a success — that distinction is the whole
			// point of the status (M-COORDINATOR-EXECUTION-TRUST M2).
			Success:         TaskStatus(completion.Status) == TaskStatusCompleted,
			SessionID:       completion.SessionID,
			NumTurns:        completion.NumTurns,
			ToolCallCount:   completion.ToolCallCount,
			InputTokens:     completion.InputTokens,
			OutputTokens:    completion.OutputTokens,
			TokensUsed:      completion.InputTokens + completion.OutputTokens,
			Cost:            completion.CostUSD,
			Duration:        time.Duration(completion.DurationMS) * time.Millisecond,
			ArtifactGCSPath: completion.ArtifactGCSPath,
		}
		if completion.ErrorMsg != "" {
			execResult.Error = completion.ErrorMsg
		}
	}

	// One finalisation path for both executors (M-COMPLETION-PATH-PARITY M1).
	//
	// This used to be a switch that set a status and posted a notification —
	// two effects, against the daemon path's ten. Everything in that difference
	// was dead in production: the approval record, the agent handoffs that
	// depend on it, and all chain and stage progression. The configured pipeline
	// design-doc-creator -> sprint-planner -> ... has therefore never advanced
	// past its first stage.
	outcome, ok := completionOutcome(completion.Status)
	if !ok {
		h.logger.Printf("CompletionHandler: unknown completion status %q for task %s", completion.Status, completion.TaskID)
		return nil
	}

	skipApproval := false
	if h.agentRegistry != nil {
		if agent := h.agentRegistry.GetAgentByID(completion.AgentID); agent != nil {
			skipApproval = agent.SkipApproval
		}
	}

	deps := &FinalizeDeps{
		TaskStore:     h.taskStore,
		MsgStore:      h.msgStore,
		ObsBackend:    h.obsBackend,
		AgentRegistry: h.agentRegistry,
		Logger:        h.logger,
		Owner:         h.instanceID,
	}
	report, err := FinalizeTaskCompletion(ctx, deps, FinalizeInput{
		Task:         task,
		Result:       execResult,
		Outcome:      outcome,
		BranchName:   completion.BranchName,
		SkipApproval: skipApproval,
	}, &CloudStrategy{Completion: completion})
	if err != nil {
		return fmt.Errorf("finalize task %s: %w", completion.TaskID, err)
	}
	h.logger.Printf("CompletionHandler: task %s -> %s (branch=%s, turns=%d, tools=%d) applied=%v skipped=%v",
		completion.TaskID, outcome, completion.BranchName, completion.NumTurns, completion.ToolCallCount,
		report.Applied, report.Skipped)

	// The agent's own inbox notice stays: the portal and sidecar poll for it.
	// Note this is NOT a reply to whoever sent the original request — completions
	// land in the agent's inbox and correlate home only via correlation_id.
	h.postCompletionNotification(ctx, task, completion)

	return nil
}

// postCompletionNotification sends a message to the agent's inbox with the
// completion status. This allows external clients (portal, sidecar) to poll
// for task completion via GET /api/messages.
func (h *CompletionHandler) postCompletionNotification(ctx context.Context, task *TaskRecord, completion pubsub.TaskCompletion) {
	if h.msgStore == nil {
		return
	}

	payload, _ := json.Marshal(map[string]interface{}{
		"task_id":       completion.TaskID,
		"agent_id":      completion.AgentID,
		"status":        completion.Status,
		"branch_name":   completion.BranchName,
		"error_msg":     completion.ErrorMsg,
		"changed_files": completion.ChangedFiles,
	})

	// Resolve the agent's INBOX, not its ID: they differ for package agents
	// (ID "pkg-sunholo-auth" watches inbox "pkg:sunholo/auth").
	toInbox := task.AgentID
	if inbox, ok := h.agentRegistry.InboxForAgent(task.AgentID); ok {
		toInbox = inbox
	}

	msg := &messaging.InboxMessage{
		FromAgent:     completion.AgentID,
		ToInbox:       toInbox,
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
