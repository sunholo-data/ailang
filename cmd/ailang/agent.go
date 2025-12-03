package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
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
	fmt.Println("  ailang agent inbox --unread-only user  # Only show unread")
	fmt.Println("  ailang agent inbox --full user         # Show full message content")
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
	full := fs.Bool("full", false, "Show full message content without truncation")

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
		showUserInbox(*stateDir, *unreadOnly, *readOnly, *archived, *archive, *limit, *full)
	case "claude-code":
		showClaudeCodeInbox(*unreadOnly, *limit, *full)
	default:
		showAgentInbox(*stateDir, agentID, *limit, *full)
	}
}
