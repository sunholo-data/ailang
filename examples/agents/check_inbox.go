package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/sunholo/ailang/internal/agentprotocol"
)

// check_inbox displays pending messages for an agent.
//
// Usage:
//   go run examples/agents/check_inbox.go <agent-id>
//
// Example:
//   go run examples/agents/check_inbox.go cli-sender
func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: check_inbox <agent-id>")
		os.Exit(1)
	}

	agentID := os.Args[1]

	// Get state dir
	stateDir := ".ailang/state"
	if env := os.Getenv("STATE_DIR"); env != "" {
		stateDir = env
	}

	// Scan for messages
	reader := agentprotocol.NewMessageReader(stateDir)
	pending, err := reader.ScanPendingMessages(agentID)
	if err != nil {
		log.Fatalf("Failed to scan messages: %v", err)
	}

	if len(pending) == 0 {
		fmt.Printf("No pending messages for %s\n", agentID)
		return
	}

	fmt.Printf("📬 Pending messages for %s (%d total):\n\n", agentID, len(pending))

	for i, msgPath := range pending {
		msg, err := reader.ReadMessage(msgPath)
		if err != nil {
			log.Printf("Failed to read %s: %v", msgPath, err)
			continue
		}

		if msg == nil {
			continue // Already seen
		}

		fmt.Printf("─────────────────────────────────────────────────────────\n")
		fmt.Printf("Message #%d\n", i+1)
		fmt.Printf("─────────────────────────────────────────────────────────\n")
		fmt.Printf("Message ID:     %s\n", msg.MessageID)
		fmt.Printf("From:           %s\n", msg.FromAgent)
		fmt.Printf("To:             %s\n", msg.ToAgent)
		fmt.Printf("Type:           %s\n", msg.MessageType)
		fmt.Printf("Correlation ID: %s\n", msg.CorrelationID)

		if msg.ParentMessageID != nil {
			fmt.Printf("Parent ID:      %s\n", *msg.ParentMessageID)
		}

		// Parse timestamp
		created, err := time.Parse(time.RFC3339, msg.Timestamp)
		if err == nil {
			age := time.Since(created).Round(time.Second)
			fmt.Printf("Created:        %v ago\n", age)
		}

		// Pretty print payload
		if msg.Payload != nil {
			payloadJSON, _ := json.MarshalIndent(msg.Payload, "                ", "  ")
			fmt.Printf("Payload:\n                %s\n", string(payloadJSON))
		}

		fmt.Printf("Path:           %s\n", msgPath)
		fmt.Println()
	}

	fmt.Printf("Total: %d message(s)\n", len(pending))
}
