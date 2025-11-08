package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/sunholo/ailang/internal/agentprotocol"
	"github.com/sunholo/ailang/internal/messaging"
)

// showUserInbox displays messages from the user inbox
func showUserInbox(stateDir string, unreadOnly, readOnly, archivedFlag, archiveAfter bool, limit int) {
	// Check if SQLite database exists
	dbPath := messaging.GetDefaultDatabasePath()
	if messaging.DatabaseExists(dbPath) {
		showUserInboxSQLite(dbPath, unreadOnly, limit)
		return
	}

	// Fall back to file-based inbox
	inbox := agentprotocol.NewUserInbox(stateDir)

	// Determine which folder to read from
	var messages []*agentprotocol.Envelope
	var err error
	var folder string

	if archivedFlag {
		folder = "archived"
		messages, err = inbox.GetArchivedMessages()
	} else if readOnly {
		folder = "read"
		messages, err = inbox.GetReadMessages()
	} else if unreadOnly {
		folder = "unread"
		messages, err = inbox.GetUnreadMessages()
	} else {
		// Default: show unread first
		folder = "unread + read"
		unread, err1 := inbox.GetUnreadMessages()
		read, err2 := inbox.GetReadMessages()
		if err1 != nil {
			err = err1
		} else if err2 != nil {
			err = err2
		} else {
			messages = append(unread, read...)
		}
	}

	if err != nil {
		fmt.Fprintf(os.Stderr, "%s: failed to read inbox: %v\n", red("Error"), err)
		os.Exit(1)
	}

	// Apply limit
	if len(messages) > limit {
		messages = messages[:limit]
	}

	// Display messages
	if len(messages) == 0 {
		fmt.Printf("%s No messages in %s folder\n", green("✓"), folder)
		return
	}

	fmt.Printf("%s %s Inbox (%d message%s)\n", bold("📬"), cyan("User"), len(messages), pluralize(len(messages)))
	fmt.Println(strings.Repeat("=", 80))
	fmt.Println()

	for i, msg := range messages {
		fmt.Printf("%s Message %d/%d\n", bold("▶"), i+1, len(messages))
		fmt.Printf("  ID: %s\n", cyan(msg.MessageID))
		fmt.Printf("  From: %s\n", msg.FromAgent)
		fmt.Printf("  Type: %s\n", msg.MessageType)
		fmt.Printf("  Timestamp: %s\n", msg.Timestamp)

		if msg.CorrelationID != "" {
			fmt.Printf("  Correlation: %s\n", msg.CorrelationID)
		}

		// Display payload preview
		if msg.Payload != nil {
			payloadJSON, _ := json.Marshal(msg.Payload)
			preview := string(payloadJSON)
			if len(preview) > 200 {
				preview = preview[:197] + "..."
			}
			fmt.Printf("  Payload: %s\n", preview)
		}

		fmt.Println()

		// Mark as read if unread
		if !readOnly && !archivedFlag {
			if err := inbox.MarkAsRead(msg.MessageID); err != nil {
				fmt.Fprintf(os.Stderr, "%s: failed to mark as read: %v\n", yellow("Warning"), err)
			}
		}

		// Archive if requested
		if archiveAfter && !archivedFlag {
			if err := inbox.MarkAsArchived(msg.MessageID); err != nil {
				fmt.Fprintf(os.Stderr, "%s: failed to archive: %v\n", yellow("Warning"), err)
			}
		}
	}

	if archiveAfter {
		fmt.Printf("%s All messages archived\n", green("✓"))
	}
}

// showClaudeCodeInbox displays messages from the claude-code inbox
func showClaudeCodeInbox(unreadOnly bool, limit int) {
	// Claude-code inbox is in project directory
	inboxDir := ".ailang/state/messages/claude-code"
	processedDir := filepath.Join(inboxDir, "_processed")

	var messages []*agentprotocol.Envelope
	var folder string

	if unreadOnly {
		folder = "unread"
		// Read *.pending.json files from main directory
		files, err := filepath.Glob(filepath.Join(inboxDir, "*.pending.json"))
		if err != nil {
			fmt.Fprintf(os.Stderr, "%s: failed to scan inbox: %v\n", red("Error"), err)
			os.Exit(1)
		}

		for _, filePath := range files {
			data, err := os.ReadFile(filePath)
			if err != nil {
				fmt.Fprintf(os.Stderr, "%s: failed to read %s: %v\n", yellow("Warning"), filePath, err)
				continue
			}

			var msg agentprotocol.Envelope
			if err := json.Unmarshal(data, &msg); err != nil {
				fmt.Fprintf(os.Stderr, "%s: failed to parse %s: %v\n", yellow("Warning"), filePath, err)
				continue
			}

			messages = append(messages, &msg)
		}
	} else {
		folder = "unread + processed"
		// Read both unread (*.pending.json) and processed (_processed/*.json)
		unreadFiles, _ := filepath.Glob(filepath.Join(inboxDir, "*.pending.json"))
		processedFiles, _ := filepath.Glob(filepath.Join(processedDir, "*.json"))

		allFiles := append(unreadFiles, processedFiles...)

		for _, filePath := range allFiles {
			data, err := os.ReadFile(filePath)
			if err != nil {
				continue
			}

			var msg agentprotocol.Envelope
			if err := json.Unmarshal(data, &msg); err != nil {
				continue
			}

			messages = append(messages, &msg)
		}
	}

	// Apply limit
	if len(messages) > limit {
		messages = messages[:limit]
	}

	// Display messages
	if len(messages) == 0 {
		fmt.Printf("%s No messages in %s folder\n", green("✓"), folder)
		return
	}

	fmt.Printf("%s %s Inbox (%d message%s)\n", bold("📬"), cyan("claude-code"), len(messages), pluralize(len(messages)))
	fmt.Println(strings.Repeat("=", 80))
	fmt.Println()

	for i, msg := range messages {
		fmt.Printf("%s Message %d/%d\n", bold("▶"), i+1, len(messages))
		fmt.Printf("  ID: %s\n", cyan(msg.MessageID))
		fmt.Printf("  From: %s\n", msg.FromAgent)
		fmt.Printf("  Type: %s\n", msg.MessageType)
		fmt.Printf("  Timestamp: %s\n", msg.Timestamp)

		if msg.CorrelationID != "" {
			fmt.Printf("  Correlation: %s\n", msg.CorrelationID)
		}

		// Display payload preview
		if msg.Payload != nil {
			payloadJSON, _ := json.Marshal(msg.Payload)
			preview := string(payloadJSON)
			if len(preview) > 200 {
				preview = preview[:197] + "..."
			}
			fmt.Printf("  Payload: %s\n", preview)
		}

		fmt.Println()
	}
}

// showAgentInbox displays messages for a specific agent
func showAgentInbox(stateDir string, agentID string, limit int) {
	reader := agentprotocol.NewMessageReader(stateDir)

	// Get pending message file paths
	filePaths, err := reader.ScanPendingMessages(agentID)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s: failed to scan inbox: %v\n", red("Error"), err)
		os.Exit(1)
	}

	if len(filePaths) == 0 {
		fmt.Printf("%s No messages for agent %s\n", green("✓"), cyan(agentID))
		return
	}

	// Apply limit
	if len(filePaths) > limit {
		filePaths = filePaths[:limit]
	}

	// Read messages
	var messages []*agentprotocol.Envelope
	for _, filePath := range filePaths {
		msg, err := reader.ReadMessage(filePath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "%s: failed to read message %s: %v\n", yellow("Warning"), filePath, err)
			continue
		}
		messages = append(messages, msg)
	}

	fmt.Printf("%s %s Inbox (%d message%s)\n", bold("📬"), cyan(agentID), len(messages), pluralize(len(messages)))
	fmt.Println(strings.Repeat("=", 80))
	fmt.Println()

	for i, msg := range messages {
		fmt.Printf("%s Message %d/%d\n", bold("▶"), i+1, len(messages))
		fmt.Printf("  ID: %s\n", cyan(msg.MessageID))
		fmt.Printf("  From: %s\n", msg.FromAgent)
		fmt.Printf("  Type: %s\n", msg.MessageType)
		fmt.Printf("  Timestamp: %s\n", msg.Timestamp)

		if msg.CorrelationID != "" {
			fmt.Printf("  Correlation: %s\n", msg.CorrelationID)
		}

		if msg.Payload != nil {
			payloadJSON, _ := json.Marshal(msg.Payload)
			preview := string(payloadJSON)
			if len(preview) > 200 {
				preview = preview[:197] + "..."
			}
			fmt.Printf("  Payload: %s\n", preview)
		}

		fmt.Println()
	}
}

// showUserInboxSQLite displays messages from SQLite database
func showUserInboxSQLite(dbPath string, unreadOnly bool, limit int) {
	store, err := messaging.OpenStore(dbPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s: failed to open database: %v\n", red("Error"), err)
		fmt.Fprintf(os.Stderr, "Falling back to file-based inbox...\n")
		return
	}
	defer store.Close()

	// Determine delivery state filter
	deliveryState := ""
	if unreadOnly {
		deliveryState = "pending"
	}

	// Get messages for user
	messages, err := store.GetMessages("human", "user", deliveryState)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s: failed to retrieve messages: %v\n", red("Error"), err)
		os.Exit(1)
	}

	// Apply limit
	if len(messages) > limit {
		messages = messages[:limit]
	}

	// Display messages
	if len(messages) == 0 {
		folder := "all"
		if unreadOnly {
			folder = "unread"
		}
		fmt.Printf("%s No messages in %s folder\n", green("✓"), folder)
		return
	}

	fmt.Printf("%s %s Inbox (%d message%s) [SQLite]\n", bold("📬"), cyan("User"), len(messages), pluralize(len(messages)))
	fmt.Println(strings.Repeat("=", 80))
	fmt.Println()

	for i, msg := range messages {
		fmt.Printf("%s Message %d/%d\n", bold("▶"), i+1, len(messages))
		fmt.Printf("  ID: %s\n", cyan(msg.ID))
		fmt.Printf("  From: %s (%s)\n", msg.FromID, msg.FromType)
		fmt.Printf("  Kind: %s\n", msg.Kind)
		fmt.Printf("  Created: %s\n", msg.CreatedAt.Format("2006-01-02 15:04:05"))
		fmt.Printf("  State: %s\n", msg.DeliveryState)

		// Display content preview
		if msg.Content != "" {
			preview := msg.Content
			if len(preview) > 200 {
				preview = preview[:197] + "..."
			}
			fmt.Printf("  Content: %s\n", preview)
		}

		fmt.Println()
	}
}
