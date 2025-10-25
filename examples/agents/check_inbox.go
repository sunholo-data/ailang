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

// check_inbox displays pending messages for an agent or user.
//
// Usage:
//
//	check_inbox [flags] <agent-id|user>
//
// Flags:
//
//	--archive       Move messages to archive after viewing
//	--unread-only   Show only unread messages
//	--read-only     Show only read messages (for user inbox)
//	--archived      Show archived messages (for user inbox)
//
// Examples:
//
//	check_inbox cli-sender
//	check_inbox user
//	check_inbox --unread-only user
//	check_inbox --archive user
func main() {
	// Parse flags
	var (
		archiveFlag  = flag.Bool("archive", false, "Move messages to archive after viewing")
		unreadOnly   = flag.Bool("unread-only", false, "Show only unread messages")
		readOnly     = flag.Bool("read-only", false, "Show only read messages (for user inbox)")
		archivedFlag = flag.Bool("archived", false, "Show archived messages (for user inbox)")
		helpFlag     = flag.Bool("help", false, "Show usage")
	)
	flag.Parse()

	if *helpFlag {
		flag.Usage()
		os.Exit(0)
	}

	args := flag.Args()

	if len(args) < 1 {
		fmt.Println("Usage: check_inbox [flags] <agent-id|user>")
		fmt.Println("\nFlags:")
		fmt.Println("  --archive       Move messages to archive after viewing")
		fmt.Println("  --unread-only   Show only unread messages")
		fmt.Println("  --read-only     Show only read messages (for user inbox)")
		fmt.Println("  --archived      Show archived messages (for user inbox)")
		fmt.Println("\nExamples:")
		fmt.Println("  check_inbox cli-sender")
		fmt.Println("  check_inbox user")
		fmt.Println("  check_inbox --unread-only user")
		fmt.Println("  check_inbox --archive user")
		os.Exit(1)
	}

	agentID := args[0]

	// Get state dir
	stateDir := ".ailang/state"
	if env := os.Getenv("STATE_DIR"); env != "" {
		stateDir = env
	}

	// Handle user inbox separately
	if agentID == "user" {
		handleUserInbox(stateDir, *archiveFlag, *unreadOnly, *readOnly, *archivedFlag)
		return
	}

	// Handle agent inbox (original behavior)
	handleAgentInbox(stateDir, agentID)
}

func handleUserInbox(stateDir string, archive, unreadOnly, readOnly, archivedFlag bool) {
	inbox := agentprotocol.NewUserInbox(stateDir)

	var messages []*agentprotocol.Envelope
	var err error
	var category string

	// Determine which category to show
	if archivedFlag {
		messages, err = inbox.GetArchivedMessages()
		category = "Archived"
	} else if readOnly {
		messages, err = inbox.GetReadMessages()
		category = "Read"
	} else if unreadOnly {
		messages, err = inbox.GetUnreadMessages()
		category = "Unread"
	} else {
		// Show all unread messages by default
		messages, err = inbox.GetUnreadMessages()
		category = "Unread"
	}

	if err != nil {
		log.Fatalf("Failed to get messages: %v", err)
	}

	if len(messages) == 0 {
		fmt.Printf("No %s messages for user\n", category)
		return
	}

	fmt.Printf("📬 %s messages for user (%d total):\n\n", category, len(messages))

	for i, msg := range messages {
		fmt.Printf("─────────────────────────────────────────────────────────\n")
		fmt.Printf("Message #%d\n", i+1)
		fmt.Printf("─────────────────────────────────────────────────────────\n")
		fmt.Printf("Message ID:     %s\n", msg.MessageID)
		fmt.Printf("From:           %s\n", msg.FromAgent)
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

		fmt.Println()

		// Mark as read if showing unread messages (unless archived flag is set)
		if category == "Unread" && !archive {
			if err := inbox.MarkAsRead(msg.MessageID); err != nil {
				log.Printf("Warning: Failed to mark message as read: %v", err)
			} else {
				fmt.Printf("  ✓ Marked as read\n")
			}
		}

		// Archive if requested
		if archive {
			if err := inbox.MarkAsArchived(msg.MessageID); err != nil {
				log.Printf("Warning: Failed to archive message: %v", err)
			} else {
				fmt.Printf("  ✓ Archived\n")
			}
		}
	}

	fmt.Printf("Total: %d message(s)\n", len(messages))

	// Show next steps
	if !archive && category == "Unread" {
		fmt.Printf("\nAll messages marked as read.\n")
		fmt.Printf("To archive messages: check_inbox --archive user\n")
	}
}

func handleAgentInbox(stateDir string, agentID string) {
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
