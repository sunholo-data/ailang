package main

import (
	"context"
	"fmt"
	"log"
	"path/filepath"
	"time"

	"github.com/sunholo/ailang/internal/agent"
	"github.com/sunholo/ailang/internal/messaging"
)

// Agent represents an AILANG agent that polls for messages and executes directives
type Agent struct {
	instanceID   string
	client       *messaging.Client
	executor     *agent.DirectiveExecutor
	pollInterval time.Duration
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

	return &Agent{
		instanceID:   instanceID,
		client:       client,
		executor:     executor,
		pollInterval: time.Duration(pollIntervalSec) * time.Second,
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

	// Process each message
	for _, msg := range messages {
		if err := a.processMessage(ctx, msg); err != nil {
			log.Printf("Failed to process message %s: %v", msg.ID, err)
			// Continue processing other messages even if one fails
			continue
		}

		// Acknowledge message after successful processing
		if err := a.client.AcknowledgeMessage(msg.ID); err != nil {
			log.Printf("Failed to acknowledge message %s: %v", msg.ID, err)
		}
	}

	return nil
}

// processMessage processes a single message
func (a *Agent) processMessage(ctx context.Context, msg *messaging.Message) error {
	log.Printf("Processing message %s (thread=%s, kind=%s)", msg.ID, msg.ThreadID, msg.Kind)
	log.Printf("  From: %s/%s", msg.FromType, msg.FromID)
	log.Printf("  To: %s/%s", msg.ToType, msg.ToID)
	log.Printf("  Content: %s", msg.Content)

	if msg.Kind == "directive" {
		log.Printf("  [DIRECTIVE] Executing: %s", msg.Content)

		// Execute directive via Claude Code
		result, err := a.executor.Execute(msg.Content)
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
			return fmt.Errorf("directive failed: %s", result.Error)
		}

		// Phase 3 will send result back to UI via message bus
		// For now, we just log the result

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
