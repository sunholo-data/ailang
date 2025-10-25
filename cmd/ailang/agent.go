package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/sunholo/ailang/internal/agentprotocol"
)

// agentCommand handles the 'ailang agent' command and its subcommands.
func agentCommand() {
	if flag.NArg() < 2 {
		printAgentHelp()
		os.Exit(1)
	}

	subcommand := flag.Arg(1)

	switch subcommand {
	case "top":
		agentTopCommand()
	case "dlq":
		agentDLQCommand()
	case "send":
		agentSendCommand()
	case "inbox":
		agentInboxCommand()
	case "ack", "acknowledge":
		agentAckCommand()
	case "unack", "unacknowledge":
		agentUnackCommand()
	case "help", "--help", "-h":
		printAgentHelp()
	default:
		fmt.Fprintf(os.Stderr, "%s: unknown agent subcommand: %s\n", red("Error"), subcommand)
		fmt.Fprintf(os.Stderr, "Run 'ailang agent help' for usage.\n")
		os.Exit(1)
	}
}

func printAgentHelp() {
	fmt.Println("Usage: ailang agent <subcommand> [options]")
	fmt.Println()
	fmt.Println("Subcommands:")
	fmt.Println("  send     Send a message to an agent")
	fmt.Println("  inbox    Check inbox for messages")
	fmt.Println("  ack      Acknowledge and mark messages as read")
	fmt.Println("  unack    Move acknowledged messages back to unread")
	fmt.Println("  top      Show agent queue status and metrics")
	fmt.Println("  dlq      Manage dead letter queue")
	fmt.Println("  help     Show this help message")
	fmt.Println()
	fmt.Println("Examples:")
	fmt.Println("  ailang agent send sprint-planner '{\"task\": \"plan\"}'")
	fmt.Println("  ailang agent send --to-user '{\"status\": \"complete\"}'")
	fmt.Println("  ailang agent inbox user                # Check user inbox")
	fmt.Println("  ailang agent inbox --unread-only       # Only show unread")
	fmt.Println("  ailang agent ack msg_20251025_154821   # Acknowledge message")
	fmt.Println("  ailang agent ack --all                 # Acknowledge all messages")
	fmt.Println("  ailang agent unack msg_20251025_154821 # Move back to unread")
	fmt.Println("  ailang agent top                       # Show current status")
	fmt.Println("  ailang agent dlq --list                # List failed messages")
}

// agentTopCommand shows agent queue status and metrics.
func agentTopCommand() {
	// Parse flags
	fs := flag.NewFlagSet("agent top", flag.ExitOnError)
	stateDir := fs.String("state-dir", getDefaultStateDir(), "State directory")
	watchMode := fs.Bool("watch", false, "Watch mode (refresh every 2 seconds)")
	_ = watchMode // TODO: implement watch mode

	if err := fs.Parse(flag.Args()[2:]); err != nil {
		fmt.Fprintf(os.Stderr, "%s: %v\n", red("Error"), err)
		os.Exit(1)
	}

	// Open database
	db, err := agentprotocol.NewDB(*stateDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s: failed to open database: %v\n", red("Error"), err)
		os.Exit(1)
	}
	defer db.Close()

	// Print header
	fmt.Println(bold("AILANG Agent Status"))
	fmt.Println(strings.Repeat("=", 80))
	fmt.Println()

	// Get active agents
	agents, err := db.ListActiveAgents()
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s: failed to list agents: %v\n", red("Error"), err)
		os.Exit(1)
	}

	if len(agents) == 0 {
		fmt.Println(yellow("No active agents"))
		return
	}

	// Display agent info
	fmt.Printf("%s:\n", bold("Active Agents"))
	for _, agent := range agents {
		fmt.Printf("  %s %s\n", green("•"), cyan(agent.AgentID))
		fmt.Printf("    Status: %s\n", getStatusColor(agent.Status))
		fmt.Printf("    Last Heartbeat: %s\n", formatDuration(time.Since(agent.LastHeartbeat)))

		// Get stats for this agent
		stats, err := db.GetAgentStats(agent.AgentID)
		if err != nil {
			fmt.Printf("    %s: failed to get stats: %v\n", yellow("Warning"), err)
			continue
		}

		// Display message counts
		if msgCounts, ok := stats["message_counts"].(map[string]int); ok && len(msgCounts) > 0 {
			fmt.Printf("    Messages:\n")
			for status, count := range msgCounts {
				fmt.Printf("      %s: %d\n", status, count)
			}
		}

		// Display metrics
		if metrics, ok := stats["metrics"].(map[string]float64); ok && len(metrics) > 0 {
			fmt.Printf("    Metrics (1h avg):\n")
			metricNames := make([]string, 0, len(metrics))
			for name := range metrics {
				metricNames = append(metricNames, name)
			}
			sort.Strings(metricNames)
			for _, name := range metricNames {
				fmt.Printf("      %s: %.2f\n", name, metrics[name])
			}
		}

		fmt.Println()
	}

	// Check for DLQ entries
	dlq := agentprotocol.NewDeadLetterQueue(*stateDir)
	dlqEntries, err := dlq.GetDeadLetterMessages()
	if err != nil {
		fmt.Printf("%s: failed to check DLQ: %v\n", yellow("Warning"), err)
	} else if len(dlqEntries) > 0 {
		fmt.Printf("%s %s: %d messages in dead letter queue\n",
			red("⚠"),
			bold("Warning"),
			len(dlqEntries))
		fmt.Println("  Run 'ailang agent dlq --list' to see details")
		fmt.Println()
	}

	// Check user inbox
	inbox := agentprotocol.NewUserInbox(*stateDir)
	unread, err := inbox.GetUnreadMessages()
	if err != nil {
		fmt.Printf("%s: failed to check user inbox: %v\n", yellow("Warning"), err)
	} else if len(unread) > 0 {
		fmt.Printf("%s: %d unread messages in user inbox\n",
			cyan("ℹ"),
			len(unread))
		fmt.Println("  Run 'check-inbox user' to view them")
		fmt.Println()
	}
}

// agentDLQCommand manages the dead letter queue.
func agentDLQCommand() {
	// Parse flags
	fs := flag.NewFlagSet("agent dlq", flag.ExitOnError)
	stateDir := fs.String("state-dir", getDefaultStateDir(), "State directory")
	listFlag := fs.Bool("list", false, "List all DLQ entries")
	retryFlag := fs.String("retry", "", "Retry message by ID")
	deleteFlag := fs.String("delete", "", "Delete message by ID")

	if err := fs.Parse(flag.Args()[2:]); err != nil {
		fmt.Fprintf(os.Stderr, "%s: %v\n", red("Error"), err)
		os.Exit(1)
	}

	dlq := agentprotocol.NewDeadLetterQueue(*stateDir)

	if *listFlag {
		// List all DLQ entries
		entries, err := dlq.GetDeadLetterMessages()
		if err != nil {
			fmt.Fprintf(os.Stderr, "%s: failed to list DLQ entries: %v\n", red("Error"), err)
			os.Exit(1)
		}

		if len(entries) == 0 {
			fmt.Println(green("Dead letter queue is empty"))
			return
		}

		fmt.Printf("%s: %d messages\n\n", bold("Dead Letter Queue"), len(entries))

		for i, entry := range entries {
			fmt.Printf("%s %s\n", bold(fmt.Sprintf("[%d]", i+1)), cyan(entry.MessageID))
			fmt.Printf("  From: %s → To: %s\n", entry.FromAgent, entry.ToAgent)
			fmt.Printf("  Type: %s\n", entry.MessageType)
			fmt.Printf("  Failed At: %s\n", entry.FailedAt.Format(time.RFC3339))
			fmt.Printf("  Retry Count: %d\n", entry.RetryCount)
			fmt.Printf("  Reason: %s\n", red(entry.FailureReason))
			if entry.StackTrace != "" {
				fmt.Printf("  Stack Trace:\n%s\n", indentText(entry.StackTrace, 4))
			}
			fmt.Println()
		}

		return
	}

	if *retryFlag != "" {
		// Retry a specific message
		env, err := dlq.RetryFromDeadLetter(*retryFlag)
		if err != nil {
			fmt.Fprintf(os.Stderr, "%s: failed to retry message: %v\n", red("Error"), err)
			os.Exit(1)
		}

		fmt.Printf("%s Retrying message %s\n", green("✓"), cyan(*retryFlag))
		fmt.Printf("  From: %s → To: %s\n", env.FromAgent, env.ToAgent)
		fmt.Println()
		fmt.Println(yellow("Note: Message removed from DLQ. You'll need to re-send it manually."))
		return
	}

	if *deleteFlag != "" {
		// Delete a specific message
		if err := dlq.DeleteDeadLetterMessage(*deleteFlag); err != nil {
			fmt.Fprintf(os.Stderr, "%s: failed to delete message: %v\n", red("Error"), err)
			os.Exit(1)
		}

		fmt.Printf("%s Deleted message %s from DLQ\n", green("✓"), cyan(*deleteFlag))
		return
	}

	// No flags specified
	printAgentDLQHelp()
}

func printAgentDLQHelp() {
	fmt.Println("Usage: ailang agent dlq [options]")
	fmt.Println()
	fmt.Println("Options:")
	fmt.Println("  --list           List all messages in DLQ")
	fmt.Println("  --retry <id>     Retry a specific message")
	fmt.Println("  --delete <id>    Delete a specific message")
	fmt.Println("  --state-dir <dir> State directory (default: ~/.ailang/state)")
	fmt.Println()
	fmt.Println("Examples:")
	fmt.Println("  ailang agent dlq --list")
	fmt.Println("  ailang agent dlq --retry msg_20251025_143000_abc123")
	fmt.Println("  ailang agent dlq --delete msg_20251025_143000_abc123")
}

// Helper functions

func getDefaultStateDir() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ".ailang/state"
	}
	return filepath.Join(home, ".ailang", "state")
}

func getStatusColor(status string) string {
	switch status {
	case "active":
		return green(status)
	case "paused":
		return yellow(status)
	case "error":
		return red(status)
	default:
		return status
	}
}

func formatDuration(d time.Duration) string {
	if d < time.Minute {
		return fmt.Sprintf("%.0fs ago", d.Seconds())
	} else if d < time.Hour {
		return fmt.Sprintf("%.0fm ago", d.Minutes())
	} else if d < 24*time.Hour {
		return fmt.Sprintf("%.1fh ago", d.Hours())
	} else {
		return fmt.Sprintf("%.1fd ago", d.Hours()/24)
	}
}

func indentText(text string, spaces int) string {
	lines := strings.Split(text, "\n")
	indent := strings.Repeat(" ", spaces)
	for i, line := range lines {
		lines[i] = indent + line
	}
	return strings.Join(lines, "\n")
}

// agentSendCommand sends a message to an agent or user.
func agentSendCommand() {
	fs := flag.NewFlagSet("agent send", flag.ExitOnError)
	stateDir := fs.String("state-dir", getDefaultStateDir(), "State directory")
	fromAgent := fs.String("from", "cli-sender", "Sender agent ID")
	toUser := fs.Bool("to-user", false, "Send to user inbox instead of agent")
	wait := fs.Duration("wait", 0, "Wait for response (e.g., 30s, 5m)")
	correlationID := fs.String("correlation-id", "", "Correlation ID for tracking")

	if err := fs.Parse(flag.Args()[2:]); err != nil {
		fmt.Fprintf(os.Stderr, "%s: %v\n", red("Error"), err)
		os.Exit(1)
	}

	// Validate arguments
	if fs.NArg() < 1 {
		fmt.Fprintf(os.Stderr, "%s: missing arguments\n", red("Error"))
		fmt.Fprintf(os.Stderr, "Usage: ailang agent send [flags] <agent-id> <payload-json>\n")
		fmt.Fprintf(os.Stderr, "   or: ailang agent send --to-user [flags] <payload-json>\n")
		os.Exit(1)
	}

	var toAgent string
	var payloadJSON string

	if *toUser {
		// Send to user: ailang agent send --to-user '{"status": "done"}'
		toAgent = "user"
		payloadJSON = fs.Arg(0)
	} else {
		// Send to agent: ailang agent send sprint-planner '{"task": "plan"}'
		if fs.NArg() < 2 {
			fmt.Fprintf(os.Stderr, "%s: missing payload argument\n", red("Error"))
			fmt.Fprintf(os.Stderr, "Usage: ailang agent send <agent-id> <payload-json>\n")
			os.Exit(1)
		}
		toAgent = fs.Arg(0)
		payloadJSON = fs.Arg(1)
	}

	// Parse payload JSON
	var payload map[string]interface{}
	if err := json.Unmarshal([]byte(payloadJSON), &payload); err != nil {
		fmt.Fprintf(os.Stderr, "%s: invalid JSON payload: %v\n", red("Error"), err)
		os.Exit(1)
	}

	// Generate IDs if not provided
	corrID := *correlationID
	if corrID == "" {
		corrID = agentprotocol.GenerateCorrelationID()
	}

	// Create message envelope
	msg := &agentprotocol.Envelope{
		ProtocolVersion: "1.0.0",
		SchemaVersion:   "1.0.0",
		MessageID:       agentprotocol.GenerateMessageID(),
		CorrelationID:   corrID,
		TraceID:         agentprotocol.GenerateTraceID(),
		Timestamp:       time.Now().UTC().Format(time.RFC3339),
		TTLSeconds:      300,
		ToAgent:         toAgent,
		FromAgent:       *fromAgent,
		MessageType:     "request",
		Payload:         payload,
	}

	// Send message
	var filePath string
	var err error

	if toAgent == "user" {
		// Send to user inbox (uses _unread folder)
		inbox := agentprotocol.NewUserInbox(*stateDir)
		filePath, err = inbox.SendToUser(msg)
	} else {
		// Send to agent inbox (pending folder)
		writer := agentprotocol.NewMessageWriter(*stateDir)
		filePath, err = writer.WriteMessage(msg)
	}

	if err != nil {
		fmt.Fprintf(os.Stderr, "%s: failed to send message: %v\n", red("Error"), err)
		os.Exit(1)
	}

	fmt.Printf("%s Message sent successfully\n", green("✓"))
	fmt.Printf("  Message ID: %s\n", cyan(msg.MessageID))
	fmt.Printf("  To: %s\n", toAgent)
	fmt.Printf("  From: %s\n", *fromAgent)
	fmt.Printf("  Correlation ID: %s\n", corrID)
	fmt.Printf("  File: %s\n", filePath)

	// Wait for response if requested
	if *wait > 0 {
		fmt.Printf("\n%s Waiting for response (timeout: %s)...\n", yellow("⏳"), *wait)
		// TODO: Implement wait logic (poll for response message with matching correlation_id)
		fmt.Printf("%s Wait mode not yet implemented\n", yellow("⚠"))
	}
}

// agentInboxCommand checks an agent's inbox for messages.
func agentInboxCommand() {
	fs := flag.NewFlagSet("agent inbox", flag.ExitOnError)
	stateDir := fs.String("state-dir", getDefaultStateDir(), "State directory")
	unreadOnly := fs.Bool("unread-only", false, "Show only unread messages")
	readOnly := fs.Bool("read-only", false, "Show only read messages")
	archived := fs.Bool("archived", false, "Show archived messages")
	archive := fs.Bool("archive", false, "Move messages to archive after viewing")
	limit := fs.Int("limit", 10, "Maximum number of messages to show")

	if err := fs.Parse(flag.Args()[2:]); err != nil {
		fmt.Fprintf(os.Stderr, "%s: %v\n", red("Error"), err)
		os.Exit(1)
	}

	// Get agent ID
	var agentID string
	if fs.NArg() < 1 {
		fmt.Fprintf(os.Stderr, "%s: missing agent ID\n", red("Error"))
		fmt.Fprintf(os.Stderr, "Usage: ailang agent inbox [flags] <agent-id>\n")
		fmt.Fprintf(os.Stderr, "Examples:\n")
		fmt.Fprintf(os.Stderr, "  ailang agent inbox user\n")
		fmt.Fprintf(os.Stderr, "  ailang agent inbox --unread-only claude-code\n")
		fmt.Fprintf(os.Stderr, "  ailang agent inbox --unread-only user\n")
		os.Exit(1)
	}
	agentID = fs.Arg(0)

	// Special handling for user and claude-code inboxes
	switch agentID {
	case "user":
		showUserInbox(*stateDir, *unreadOnly, *readOnly, *archived, *archive, *limit)
	case "claude-code":
		showClaudeCodeInbox(*unreadOnly, *limit)
	default:
		showAgentInbox(*stateDir, agentID, *limit)
	}
}

func showUserInbox(stateDir string, unreadOnly, readOnly, archivedFlag, archiveAfter bool, limit int) {
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

func pluralize(count int) string {
	if count == 1 {
		return ""
	}
	return "s"
}

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

func ackMessage(stateDir, messageID string) error {
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

func ackAllMessages(stateDir string) {
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

func unackMessage(stateDir, messageID string) error {
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
