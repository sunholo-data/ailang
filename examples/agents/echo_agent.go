package main

import (
	"fmt"
	"log"
	"os"
	"time"

	"github.com/sunholo/ailang/internal/agentprotocol"
	"github.com/sunholo/ailang/internal/agentrunner"
)

// EchoAgent is a simple demo agent that echoes messages back to the sender.
//
// Usage:
//
//	go run examples/agents/echo_agent.go
//
// In another terminal:
//
//	go run examples/agents/send_message.go echo-agent '{"message": "Hello!"}'
//
// The echo agent will:
// 1. Receive the message
// 2. Log it
// 3. Send a response back with the same payload
func main() {
	// Parse command line arguments
	stateDir := ".ailang/state"
	if len(os.Args) > 1 {
		stateDir = os.Args[1]
	}

	fmt.Println("🤖 Echo Agent Starting...")
	fmt.Printf("   State dir: %s\n", stateDir)
	fmt.Printf("   Agent ID: echo-agent\n")
	fmt.Printf("   Poll interval: 2 seconds\n\n")

	// Create handler that echoes messages
	handler := agentrunner.NewFunctionHandler(func(msg *agentprotocol.Envelope) (map[string]interface{}, error) {
		log.Printf("📨 Received message from %s", msg.FromAgent)
		log.Printf("   Message ID: %s", msg.MessageID)
		log.Printf("   Correlation ID: %s", msg.CorrelationID)
		log.Printf("   Type: %s", msg.MessageType)
		log.Printf("   Payload: %v\n", msg.Payload)

		// Echo back the payload
		response := map[string]interface{}{
			"echo":        msg.Payload,
			"received_at": time.Now().UTC().Format(time.RFC3339),
			"message":     "Message echoed successfully",
		}

		log.Printf("✅ Echoing message back to %s\n", msg.FromAgent)

		return response, nil
	})

	// Configure the agent
	config := &agentrunner.AgentConfig{
		AgentID:       "echo-agent",
		StateDir:      stateDir,
		PollInterval:  2 * time.Second,
		LeaseDuration: 60,
		Handler:       handler,
		OnError: func(err error) {
			log.Printf("❌ Error: %v", err)
		},
	}

	// Create and start runner
	runner, err := agentrunner.NewRunner(config)
	if err != nil {
		log.Fatalf("Failed to create runner: %v", err)
	}
	defer runner.Stop()

	log.Println("✓ Echo agent started. Press Ctrl+C to stop.\n")

	// Run (blocks until stopped)
	if err := runner.Run(); err != nil {
		log.Fatalf("Runner failed: %v", err)
	}
}
