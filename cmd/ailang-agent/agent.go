package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/sunholo/ailang/internal/messaging"
)

// Agent represents an AILANG agent that polls for messages and executes directives
type Agent struct {
	instanceID   string
	client       *messaging.Client
	pollInterval time.Duration
}

// NewAgent creates a new agent instance
func NewAgent(instanceID string, dbPath string, pollIntervalSec int) (*Agent, error) {
	// Create messaging client
	client, err := messaging.NewClient(dbPath, instanceID)
	if err != nil {
		return nil, fmt.Errorf("failed to create client: %w", err)
	}

	return &Agent{
		instanceID:   instanceID,
		client:       client,
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

	// For Phase 1, we just log the message
	// Phase 2 will add actual execution via Claude Code

	if msg.Kind == "directive" {
		log.Printf("  [DIRECTIVE] Would execute: %s", msg.Content)
		// TODO: Execute directive via Claude Code (Phase 2)
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
