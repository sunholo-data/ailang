package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/sunholo/ailang/internal/agentprotocol"
)

// send_message is a utility for sending messages to agents.
//
// Usage:
//   go run examples/agents/send_message.go <to-agent> <payload-json>
//
// Examples:
//   go run examples/agents/send_message.go echo-agent '{"message": "Hello!"}'
//   go run examples/agents/send_message.go eval-analyzer '{"action": "analyze"}'
func main() {
	if len(os.Args) < 3 {
		fmt.Println("Usage: send_message <to-agent> <payload-json>")
		fmt.Println("\nExamples:")
		fmt.Println(`  send_message echo-agent '{"message": "Hello!"}'`)
		fmt.Println(`  send_message eval-analyzer '{"action": "analyze"}'`)
		os.Exit(1)
	}

	toAgent := os.Args[1]
	payloadStr := os.Args[2]

	// Parse payload
	var payload map[string]interface{}
	if err := json.Unmarshal([]byte(payloadStr), &payload); err != nil {
		log.Fatalf("Invalid JSON payload: %v", err)
	}

	// Get state dir
	stateDir := ".ailang/state"
	if env := os.Getenv("STATE_DIR"); env != "" {
		stateDir = env
	}

	// Create message
	writer := agentprotocol.NewMessageWriter(stateDir)
	msg := &agentprotocol.Envelope{
		ProtocolVersion: "1.0.0",
		SchemaVersion:   "1.0.0",
		MessageID:       agentprotocol.GenerateMessageID(),
		CorrelationID:   agentprotocol.GenerateCorrelationID(),
		TraceID:         agentprotocol.GenerateTraceID(),
		Timestamp:       time.Now().UTC().Format(time.RFC3339),
		TTLSeconds:      300,
		FromAgent:       "cli-sender",
		ToAgent:         toAgent,
		MessageType:     "request",
		PayloadSchema:   "cli.v1",
		Payload:         payload,
	}

	// Send message
	msgPath, err := writer.WriteMessage(msg)
	if err != nil {
		log.Fatalf("Failed to send message: %v", err)
	}

	fmt.Printf("✓ Message sent to %s\n", toAgent)
	fmt.Printf("  Message ID: %s\n", msg.MessageID)
	fmt.Printf("  Correlation ID: %s\n", msg.CorrelationID)
	fmt.Printf("  Path: %s\n", msgPath)
	fmt.Printf("\nTo check for response:\n")
	fmt.Printf("  go run examples/agents/check_inbox.go cli-sender\n")
}
