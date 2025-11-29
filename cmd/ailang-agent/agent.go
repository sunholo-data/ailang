package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"path/filepath"
	"sync"
	"time"

	"github.com/sunholo/ailang/internal/agent"
	"github.com/sunholo/ailang/internal/messaging"
)

// Agent represents an AILANG agent that polls for messages and executes directives
type Agent struct {
	instanceID   string
	client       *messaging.Client
	executor     *agent.DirectiveExecutor
	detector     *agent.CapabilityDetector
	pollInterval time.Duration
	// Track active tasks for concurrency
	activeTasks map[string]bool
	taskMutex   sync.Mutex
}

// NewAgent creates a new agent instance
func NewAgent(instanceID string, dbPath string, pollIntervalSec int) (*Agent, error) {
	// Create messaging client
	client, err := messaging.NewClient(dbPath, instanceID)
	if err != nil {
		return nil, fmt.Errorf("failed to create client: %w", err)
	}

	// Create directive executor
	// Use .ailang/state/workspaces for execution workspaces
	workspaceBase := filepath.Join(filepath.Dir(dbPath), "workspaces")
	executor := agent.NewDirectiveExecutor(workspaceBase)

	// Create capability detector
	detector := agent.NewCapabilityDetector()

	return &Agent{
		instanceID:   instanceID,
		client:       client,
		executor:     executor,
		detector:     detector,
		pollInterval: time.Duration(pollIntervalSec) * time.Second,
		activeTasks:  make(map[string]bool),
	}, nil
}

// Run starts the agent polling loop
func (a *Agent) Run(ctx context.Context) error {
	log.Printf("Agent %s starting (poll interval: %v)", a.instanceID, a.pollInterval)

	ticker := time.NewTicker(a.pollInterval)
	defer ticker.Stop()

	// Do initial poll immediately
	if err := a.poll(ctx); err != nil {
		log.Printf("Initial poll error: %v", err)
	}

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			if err := a.poll(ctx); err != nil {
				log.Printf("Poll error: %v", err)
			}
		}
	}
}

// poll checks for new messages and processes them
func (a *Agent) poll(ctx context.Context) error {
	// Get pending messages for this agent instance
	messages, err := a.client.PollMessages()
	if err != nil {
		return fmt.Errorf("failed to poll messages: %w", err)
	}

	if len(messages) == 0 {
		// No messages, silent poll
		return nil
	}

	log.Printf("Received %d message(s)", len(messages))

	// Process each message concurrently
	for _, msg := range messages {
		// Check if we're already processing this message
		a.taskMutex.Lock()
		if a.activeTasks[msg.ID] {
			a.taskMutex.Unlock()
			log.Printf("Message %s already being processed, skipping", msg.ID)
			continue
		}
		a.taskMutex.Unlock()

		// Try to claim the message (atomic operation to prevent duplicate work)
		if err := a.client.ClaimMessage(msg.ID); err != nil {
			log.Printf("Failed to claim message %s (another agent may have claimed it): %v", msg.ID, err)
			// Another agent claimed this message, skip it
			continue
		}

		log.Printf("Successfully claimed message %s", msg.ID)

		// Mark as active
		a.taskMutex.Lock()
		a.activeTasks[msg.ID] = true
		a.taskMutex.Unlock()

		// Process the claimed message in a goroutine (non-blocking)
		go func(msg *messaging.Message) {
			defer func() {
				// Remove from active tasks when done
				a.taskMutex.Lock()
				delete(a.activeTasks, msg.ID)
				a.taskMutex.Unlock()
			}()

			if err := a.processMessage(ctx, msg); err != nil {
				log.Printf("Failed to process message %s: %v", msg.ID, err)
				return
			}

			// Acknowledge message after successful processing
			if err := a.client.AcknowledgeMessage(msg.ID); err != nil {
				log.Printf("Failed to acknowledge message %s: %v", msg.ID, err)
			}
		}(msg)
	}

	return nil
}

// ActiveTaskCount returns the number of currently active tasks
func (a *Agent) ActiveTaskCount() int {
	a.taskMutex.Lock()
	defer a.taskMutex.Unlock()
	return len(a.activeTasks)
}

// parseWorkspace extracts workspace path from message metadata
func parseWorkspace(metadataJSON string) string {
	if metadataJSON == "" {
		return ""
	}
	var metadata struct {
		Workspace string `json:"workspace"`
	}
	if err := json.Unmarshal([]byte(metadataJSON), &metadata); err != nil {
		return ""
	}
	return metadata.Workspace
}

// processMessage processes a single message
func (a *Agent) processMessage(ctx context.Context, msg *messaging.Message) error {
	log.Printf("Processing message %s (thread=%s, kind=%s)", msg.ID, msg.ThreadID, msg.Kind)
	log.Printf("  From: %s/%s", msg.FromType, msg.FromID)
	log.Printf("  To: %s/%s", msg.ToType, msg.ToID)
	log.Printf("  Content: %s", msg.Content)

	// Get workspace from thread (persisted at thread level)
	// Falls back to message metadata for backward compatibility
	workspace, err := a.client.GetThreadWorkspace(msg.ThreadID)
	if err != nil {
		log.Printf("  [WORKSPACE] Warning: could not get thread workspace: %v", err)
		workspace = ""
	}
	if workspace == "" {
		// Fallback to message metadata (legacy support)
		workspace = parseWorkspace(msg.MetadataJSON)
	}
	if workspace != "" {
		log.Printf("  [WORKSPACE] Using: %s", workspace)
	} else {
		log.Printf("  [WORKSPACE] None set - will create temporary workspace")
	}

	if msg.Kind == "directive" {
		log.Printf("  [DIRECTIVE] Executing: %s", msg.Content)

		// Detect required capabilities
		deltas := a.detector.DetectCapabilities(msg.Content)
		if len(deltas) > 0 {
			log.Printf("  [APPROVAL] Directive requires capabilities: %v", deltas)

			// Request approval
			proposal := a.detector.FormatProposal(deltas)
			impactLevel := a.detector.ClassifyImpact(deltas)             // "low", "medium", or "high"
			impactDesc := a.detector.FormatImpact(deltas)                // Human-readable description
			estimatedCost := a.detector.CalculateTotalCost(deltas, 0.01) // Base execution cost $0.01

			// Include impact description in proposal for human review
			fullProposal := fmt.Sprintf("%s\n\nImpact: %s", proposal, impactDesc)

			approvalID, err := a.client.RequestApproval(msg.ThreadID, deltas[0], fullProposal, impactLevel, estimatedCost)
			if err != nil {
				log.Printf("  [ERROR] Failed to request approval: %v", err)
				return fmt.Errorf("failed to request approval: %w", err)
			}

			log.Printf("  [APPROVAL] Requested approval %s (cost: $%.2f)", approvalID, estimatedCost)

			// Wait for approval (24 hour timeout - effectively indefinite)
			approved, err := a.client.WaitForApproval(approvalID, 24*time.Hour)
			if err != nil {
				log.Printf("  [APPROVAL] Approval timed out: %v", err)
				// Send timeout message to UI
				_, _ = a.client.SendResult(msg.ThreadID, fmt.Sprintf("⏱️ Approval request timed out\n\nThe directive required approval but timed out after 24 hours.\n\nDirective: %s", msg.Content))
				return fmt.Errorf("approval timeout: %w", err)
			}

			if !approved {
				log.Printf("  [APPROVAL] Rejected")
				// Send rejection message to UI
				_, _ = a.client.SendResult(msg.ThreadID, fmt.Sprintf("❌ Directive rejected\n\nThe directive was rejected by the user.\n\nDirective: %s", msg.Content))
				return fmt.Errorf("directive rejected")
			}

			log.Printf("  [APPROVAL] Approved! Proceeding with execution")
		} else {
			log.Printf("  [APPROVAL] No special capabilities required, proceeding")
		}

		// Execute directive via Claude Code (optionally in specified workspace)
		result, err := a.executor.ExecuteInWorkspace(msg.Content, workspace)
		if err != nil {
			log.Printf("  [ERROR] Execution failed: %v", err)
			return fmt.Errorf("directive execution failed: %w", err)
		}

		// Log execution results
		log.Printf("  [RESULT] Success=%v, Duration=%dms, Cost=$%.4f, Turns=%d",
			result.Success, result.DurationMS, result.Cost, result.NumTurns)
		log.Printf("  [RESULT] Output: %s", result.Output)

		if len(result.FilesCreated) > 0 {
			log.Printf("  [FILES] Created %d file(s):", len(result.FilesCreated))
			for _, file := range result.FilesCreated {
				log.Printf("    - %s", file)
			}
		}

		if !result.Success {
			log.Printf("  [ERROR] %s", result.Error)
		}

		// Format and send result back to UI
		formatted := agent.FormatResult(result)
		log.Printf("  [SENDING] Publishing result to thread %s", msg.ThreadID)
		if _, err := a.client.SendResult(msg.ThreadID, formatted); err != nil {
			log.Printf("  [ERROR] Failed to send result: %v", err)
			return fmt.Errorf("failed to send result: %w", err)
		}

		log.Printf("  [SUCCESS] Result published to UI")

		// Return error if directive failed (for proper error handling)
		if !result.Success {
			return fmt.Errorf("directive failed: %s", result.Error)
		}

	} else if msg.Kind == "question" {
		log.Printf("  [QUESTION] Answering: %s", msg.Content)

		// Questions don't require approval - they're just read-only queries
		// Use thread workspace if set, otherwise creates a temporary one
		result, err := a.executor.ExecuteInWorkspace(msg.Content, workspace)
		if err != nil {
			log.Printf("  [ERROR] Question answering failed: %v", err)
			return fmt.Errorf("question answering failed: %w", err)
		}

		// Format as a simple response (no workspace link needed for questions)
		var response string
		if result.Success {
			response = fmt.Sprintf("## Answer\n\n%s\n\n---\n*Duration: %dms, Cost: $%.4f*",
				result.Output, result.DurationMS, result.Cost)
		} else {
			response = fmt.Sprintf("## Failed to answer\n\n%s", result.Error)
		}

		if _, err := a.client.SendResult(msg.ThreadID, response); err != nil {
			log.Printf("  [ERROR] Failed to send answer: %v", err)
			return fmt.Errorf("failed to send answer: %w", err)
		}

		log.Printf("  [SUCCESS] Answer published to UI")

	} else {
		log.Printf("  [INFO] Message kind '%s' not yet supported", msg.Kind)
	}

	return nil
}

// Close closes the agent and releases resources
func (a *Agent) Close() error {
	if a.client != nil {
		return a.client.Close()
	}
	return nil
}
