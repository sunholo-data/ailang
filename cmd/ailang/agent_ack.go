package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/sunholo/ailang/internal/messaging"
)

// agentAckCommand handles acknowledgment of messages
func agentAckCommand() {
	fs := flag.NewFlagSet("agent ack", flag.ExitOnError)
	stateDir := fs.String("state-dir", getDefaultStateDir(), "State directory")
	all := fs.Bool("all", false, "Acknowledge all unread messages")

	if err := fs.Parse(flag.Args()[2:]); err != nil {
		fmt.Fprintf(os.Stderr, "%s: %v\n", red("Error"), err)
		os.Exit(1)
	}

	// Get message IDs or patterns
	args := fs.Args()

	if *all {
		ackAllMessages(*stateDir)
		return
	}

	if len(args) < 1 {
		fmt.Fprintf(os.Stderr, "%s: missing message ID or pattern\n", red("Error"))
		fmt.Fprintf(os.Stderr, "Usage: ailang agent ack <message-id>\n")
		fmt.Fprintf(os.Stderr, "       ailang agent ack --all\n")
		fmt.Fprintf(os.Stderr, "Examples:\n")
		fmt.Fprintf(os.Stderr, "  ailang agent ack msg_20251025_154821_ff2abd75af09\n")
		fmt.Fprintf(os.Stderr, "  ailang agent ack --all\n")
		os.Exit(1)
	}

	messageID := args[0]
	if err := ackMessage(*stateDir, messageID); err != nil {
		fmt.Fprintf(os.Stderr, "%s: %v\n", red("Error"), err)
		os.Exit(1)
	}

	fmt.Printf("%s Message %s acknowledged\n", green("✓"), messageID)
}

// ackMessage acknowledges a single message by moving it from unread to read/processed
func ackMessage(stateDir, messageID string) error {
	// Check if SQLite database exists
	dbPath := messaging.GetDefaultDatabasePath()
	if messaging.DatabaseExists(dbPath) {
		return ackMessageSQLite(dbPath, messageID)
	}

	// Fall back to file-based inbox
	// Try to acknowledge from different inbox locations
	locations := []struct {
		dir  string
		name string
	}{
		{filepath.Join(stateDir, "messages", "inbox", "user", "_unread"), "user inbox"},
		{filepath.Join(".", ".ailang", "state", "messages", "claude-code"), "claude-code inbox"},
	}

	for _, loc := range locations {
		// Try different filename patterns
		patterns := []string{
			messageID + ".json",
			messageID + ".pending.json",
			messageID, // In case user provides full filename
		}

		for _, pattern := range patterns {
			srcPath := filepath.Join(loc.dir, pattern)
			if _, err := os.Stat(srcPath); err == nil {
				// Found the message, move it to processed/read
				var dstDir string
				if strings.Contains(loc.dir, "claude-code") {
					dstDir = filepath.Join(".", ".ailang", "state", "messages", "claude-code", "_processed")
				} else {
					dstDir = filepath.Join(stateDir, "messages", "inbox", "user", "_read")
				}

				if err := os.MkdirAll(dstDir, 0755); err != nil {
					return fmt.Errorf("failed to create directory: %w", err)
				}

				dstPath := filepath.Join(dstDir, filepath.Base(srcPath))
				if err := os.Rename(srcPath, dstPath); err != nil {
					return fmt.Errorf("failed to move message: %w", err)
				}

				return nil
			}
		}
	}

	return fmt.Errorf("message not found: %s", messageID)
}

// ackAllMessages acknowledges all unread messages
func ackAllMessages(stateDir string) {
	// Check if SQLite database exists
	dbPath := messaging.GetDefaultDatabasePath()
	if messaging.DatabaseExists(dbPath) {
		ackAllMessagesSQLite(dbPath)
		return
	}

	// Fall back to file-based inbox
	count := 0
	locations := []string{
		filepath.Join(stateDir, "messages", "inbox", "user", "_unread"),
		filepath.Join(".", ".ailang", "state", "messages", "claude-code"),
	}

	for _, srcDir := range locations {
		files, err := filepath.Glob(filepath.Join(srcDir, "*.json"))
		if err != nil {
			continue
		}

		for _, srcPath := range files {
			messageID := strings.TrimSuffix(filepath.Base(srcPath), ".json")
			if err := ackMessage(stateDir, messageID); err == nil {
				count++
			}
		}
	}

	if count == 0 {
		fmt.Println("No messages to acknowledge")
	} else {
		fmt.Printf("%s Acknowledged %d message%s\n", green("✓"), count, pluralize(count))
	}
}

// agentUnackCommand handles un-acknowledgment of messages (move back to unread)
func agentUnackCommand() {
	fs := flag.NewFlagSet("agent unack", flag.ExitOnError)
	stateDir := fs.String("state-dir", getDefaultStateDir(), "State directory")

	if err := fs.Parse(flag.Args()[2:]); err != nil {
		fmt.Fprintf(os.Stderr, "%s: %v\n", red("Error"), err)
		os.Exit(1)
	}

	// Get message IDs
	args := fs.Args()

	if len(args) < 1 {
		fmt.Fprintf(os.Stderr, "%s: missing message ID\n", red("Error"))
		fmt.Fprintf(os.Stderr, "Usage: ailang agent unack <message-id>\n")
		fmt.Fprintf(os.Stderr, "Examples:\n")
		fmt.Fprintf(os.Stderr, "  ailang agent unack msg_20251025_154821_ff2abd75af09\n")
		os.Exit(1)
	}

	messageID := args[0]
	if err := unackMessage(*stateDir, messageID); err != nil {
		fmt.Fprintf(os.Stderr, "%s: %v\n", red("Error"), err)
		os.Exit(1)
	}

	fmt.Printf("%s Message %s moved back to unread\n", green("✓"), messageID)
}

// unackMessage moves a message back from read/processed to unread
func unackMessage(stateDir, messageID string) error {
	// Check if SQLite database exists
	dbPath := messaging.GetDefaultDatabasePath()
	if messaging.DatabaseExists(dbPath) {
		return unackMessageSQLite(dbPath, messageID)
	}

	// Fall back to file-based inbox
	// Try to unacknowledge from different processed/read locations
	locations := []struct {
		srcDir string
		dstDir string
		name   string
	}{
		{
			srcDir: filepath.Join(stateDir, "messages", "inbox", "user", "_read"),
			dstDir: filepath.Join(stateDir, "messages", "inbox", "user", "_unread"),
			name:   "user inbox",
		},
		{
			srcDir: filepath.Join(".", ".ailang", "state", "messages", "claude-code", "_processed"),
			dstDir: filepath.Join(".", ".ailang", "state", "messages", "claude-code"),
			name:   "claude-code inbox",
		},
	}

	for _, loc := range locations {
		// Try different filename patterns
		patterns := []string{
			messageID + ".json",
			messageID + ".pending.json",
			messageID, // In case user provides full filename
		}

		for _, pattern := range patterns {
			srcPath := filepath.Join(loc.srcDir, pattern)
			if _, err := os.Stat(srcPath); err == nil {
				// Found the message, move it back to unread
				if err := os.MkdirAll(loc.dstDir, 0755); err != nil {
					return fmt.Errorf("failed to create directory: %w", err)
				}

				// For claude-code inbox, change filename from .json back to .pending.json
				dstFilename := filepath.Base(srcPath)
				if strings.Contains(loc.dstDir, "claude-code") && !strings.HasSuffix(dstFilename, ".pending.json") {
					dstFilename = strings.TrimSuffix(dstFilename, ".json") + ".pending.json"
				}

				dstPath := filepath.Join(loc.dstDir, dstFilename)
				if err := os.Rename(srcPath, dstPath); err != nil {
					return fmt.Errorf("failed to move message: %w", err)
				}

				return nil
			}
		}
	}

	return fmt.Errorf("message not found in processed/read folders: %s", messageID)
}

// ackMessageSQLite acknowledges a message in SQLite database
func ackMessageSQLite(dbPath, messageID string) error {
	store, err := messaging.OpenStore(dbPath)
	if err != nil {
		return fmt.Errorf("failed to open database: %w", err)
	}
	defer store.Close()

	return store.MarkAsAcked(messageID)
}

// ackAllMessagesSQLite acknowledges all messages in SQLite database
func ackAllMessagesSQLite(dbPath string) {
	store, err := messaging.OpenStore(dbPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s: failed to open database: %v\n", red("Error"), err)
		os.Exit(1)
	}
	defer store.Close()

	count, err := store.MarkAllAsAcked("human", "user")
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s: failed to acknowledge messages: %v\n", red("Error"), err)
		os.Exit(1)
	}

	if count == 0 {
		fmt.Println("No messages to acknowledge")
	} else {
		fmt.Printf("%s Acknowledged %d message%s\n", green("✓"), count, pluralize(int(count)))
	}
}

// unackMessageSQLite moves a message back to pending in SQLite database
func unackMessageSQLite(dbPath, messageID string) error {
	store, err := messaging.OpenStore(dbPath)
	if err != nil {
		return fmt.Errorf("failed to open database: %w", err)
	}
	defer store.Close()

	return store.MarkAsUnacked(messageID)
}
