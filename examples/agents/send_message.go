//go:build ignore

package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/sunholo/ailang/internal/agentprotocol"
)

// send_message is a utility for sending messages to agents or users.
//
// Usage:
//
//	send_message [flags] <to-agent> <payload-json>
//
// Flags:
//
//	--to-user       Send to user inbox instead of agent
//	--wait <dur>    Wait for response (e.g., "30s", "5m")
//	--from <agent>  Sender agent ID (default: "cli-sender")
//
// Examples:
//
//	send_message echo-agent '{"message": "Hello!"}'
//	send_message --to-user '{"message": "Task complete!"}'
//	send_message --wait 30s eval-analyzer '{"action": "analyze"}'
func main() {
	// Parse flags
	var (
		toUser    = flag.Bool("to-user", false, "Send to user inbox instead of agent")
		waitFlag  = flag.String("wait", "", "Wait for response (e.g., '30s', '5m')")
		fromAgent = flag.String("from", "cli-sender", "Sender agent ID")
		helpFlag  = flag.Bool("help", false, "Show usage")
	)
	flag.Parse()

	if *helpFlag {
		flag.Usage()
		os.Exit(0)
	}

	args := flag.Args()

	// Parse arguments based on --to-user flag
	var toAgent string
	var payloadStr string

	if *toUser {
		// Format: send_message --to-user <payload-json>
		if len(args) < 1 {
			fmt.Println("Usage: send_message --to-user <payload-json>")
			fmt.Println("\nExamples:")
			fmt.Println(`  send_message --to-user '{"message": "Task complete!"}'`)
			os.Exit(1)
		}
		toAgent = "user"
		payloadStr = args[0]
	} else {
		// Format: send_message <to-agent> <payload-json>
		if len(args) < 2 {
			fmt.Println("Usage: send_message [flags] <to-agent> <payload-json>")
			fmt.Println("\nFlags:")
			fmt.Println("  --to-user       Send to user inbox instead of agent")
			fmt.Println("  --wait <dur>    Wait for response (e.g., '30s', '5m')")
			fmt.Println("  --from <agent>  Sender agent ID (default: 'cli-sender')")
			fmt.Println("\nExamples:")
			fmt.Println(`  send_message echo-agent '{"message": "Hello!"}'`)
			fmt.Println(`  send_message --to-user '{"message": "Task complete!"}'`)
			fmt.Println(`  send_message --wait 30s eval-analyzer '{"action": "analyze"}'`)
			os.Exit(1)
		}
		toAgent = args[0]
		payloadStr = args[1]
	}

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

	// Create message envelope
	msg := &agentprotocol.Envelope{
		ProtocolVersion: "1.0.0",
		SchemaVersion:   "1.0.0",
		MessageID:       agentprotocol.GenerateMessageID(),
		CorrelationID:   agentprotocol.GenerateCorrelationID(),
		TraceID:         agentprotocol.GenerateTraceID(),
		Timestamp:       time.Now().UTC().Format(time.RFC3339),
		TTLSeconds:      300,
		FromAgent:       *fromAgent,
		ToAgent:         toAgent,
		MessageType:     "request",
		PayloadSchema:   "cli.v1",
		Payload:         payload,
	}

	// Send message
	var msgPath string
	var err error

	if *toUser {
		// Send to user inbox
		inbox := agentprotocol.NewUserInbox(stateDir)
		msgPath, err = inbox.SendToUser(msg)
		if err != nil {
			log.Fatalf("Failed to send message to user: %v", err)
		}

		fmt.Printf("✓ Message sent to user inbox\n")
		fmt.Printf("  Message ID: %s\n", msg.MessageID)
		fmt.Printf("  Path: %s\n", msgPath)
		fmt.Printf("\nTo check user inbox:\n")
		fmt.Printf("  go run examples/agents/check_inbox.go user\n")
	} else {
		// Send to agent
		writer := agentprotocol.NewMessageWriter(stateDir)
		msgPath, err = writer.WriteMessage(msg)
		if err != nil {
			log.Fatalf("Failed to send message: %v", err)
		}

		fmt.Printf("✓ Message sent to %s\n", toAgent)
		fmt.Printf("  Message ID: %s\n", msg.MessageID)
		fmt.Printf("  Correlation ID: %s\n", msg.CorrelationID)
		fmt.Printf("  Path: %s\n", msgPath)
		fmt.Printf("\nTo check for response:\n")
		fmt.Printf("  go run examples/agents/check_inbox.go %s\n", *fromAgent)
	}

	// Wait for response if requested
	if *waitFlag != "" {
		waitDuration, err := time.ParseDuration(*waitFlag)
		if err != nil {
			log.Fatalf("Invalid wait duration: %v", err)
		}

		fmt.Printf("\nWaiting for response (timeout: %s)...\n", waitDuration)

		// Poll for response with correlation ID
		reader := agentprotocol.NewMessageReader(stateDir)
		deadline := time.Now().Add(waitDuration)
		pollInterval := 500 * time.Millisecond

		for time.Now().Before(deadline) {
			// Scan for messages to the sender
			pending, err := reader.ScanPendingMessages(*fromAgent)
			if err != nil {
				log.Printf("Warning: Failed to scan messages: %v", err)
				time.Sleep(pollInterval)
				continue
			}

			// Look for response with matching correlation ID
			for _, msgFile := range pending {
				env, err := reader.ReadMessage(msgFile)
				if err != nil {
					continue
				}

				if env == nil {
					continue // Already seen
				}

				if env.CorrelationID == msg.CorrelationID && env.MessageType == "response" {
					fmt.Printf("\n✓ Received response!\n")
					fmt.Printf("  Message ID: %s\n", env.MessageID)
					fmt.Printf("  From: %s\n", env.FromAgent)

					// Pretty print payload
					payloadJSON, _ := json.MarshalIndent(env.Payload, "  ", "  ")
					fmt.Printf("  Payload:\n  %s\n", string(payloadJSON))

					return // Success
				}
			}

			time.Sleep(pollInterval)
		}

		fmt.Printf("\n⚠ Timeout: No response received within %s\n", waitDuration)
		os.Exit(1)
	}
}
