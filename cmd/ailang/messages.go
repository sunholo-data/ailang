package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/sunholo/ailang/internal/messaging"
)

// messagesCommand handles the 'messages' (alias: 'msg') subcommand.
// This uses the unified collaboration.db for both CLI and dashboard access.
func messagesCommand() {
	if len(os.Args) < 3 {
		runMessagesList([]string{})
		return
	}

	subCmd := os.Args[2]
	args := os.Args[3:]

	switch subCmd {
	case "list", "ls":
		runMessagesList(args)
	case "ack":
		runMessagesAck(args)
	case "unack":
		runMessagesUnack(args)
	case "send":
		runMessagesSend(args)
	case "read":
		runMessagesRead(args)
	case "watch":
		runMessagesWatch(args)
	case "cleanup":
		runMessagesCleanup(args)
	case "--help", "-h", "help":
		printMessagesHelp()
	default:
		fmt.Fprintf(os.Stderr, "%s: unknown subcommand '%s'\n", red("Error"), subCmd)
		printMessagesHelp()
		os.Exit(1)
	}
}

// openStore opens the unified collaboration database
func openStore() (*messaging.Store, error) {
	dbPath := messaging.GetDefaultDatabasePath()
	return messaging.OpenStore(dbPath)
}

func runMessagesList(args []string) {
	fs := flag.NewFlagSet("messages list", flag.ExitOnError)
	inbox := fs.String("inbox", "", "Filter by inbox (user, claude-code, etc.)")
	unread := fs.Bool("unread", false, "Show only unread messages")
	from := fs.String("from", "", "Filter by sender agent")
	limit := fs.Int("limit", 20, "Maximum messages to show")
	jsonOut := fs.Bool("json", false, "Output as JSON")

	if err := fs.Parse(args); err != nil {
		fmt.Fprintf(os.Stderr, "%s: %v\n", red("Error"), err)
		os.Exit(1)
	}

	store, err := openStore()
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s: %v\n", red("Error"), err)
		os.Exit(1)
	}
	defer store.Close()

	opts := messaging.InboxListOptions{
		Inbox:      *inbox,
		FromAgent:  *from,
		Limit:      *limit,
		UnreadOnly: *unread,
	}

	messages, err := store.ListInboxMessages(opts)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s: %v\n", red("Error"), err)
		os.Exit(1)
	}

	if *jsonOut {
		data, _ := json.MarshalIndent(messages, "", "  ")
		fmt.Println(string(data))
		return
	}

	if len(messages) == 0 {
		fmt.Println("No messages found.")
		return
	}

	// Get counts for summary
	counts, _ := store.CountInboxMessagesByStatus(*inbox)

	// Print summary
	fmt.Printf("\n%s\n\n", bold("Messages"))
	if counts[messaging.InboxStatusUnread] > 0 {
		fmt.Printf("  Unread: %s\n", yellow(fmt.Sprintf("%d", counts[messaging.InboxStatusUnread])))
	}
	if counts[messaging.InboxStatusRead] > 0 {
		fmt.Printf("  Read: %d\n", counts[messaging.InboxStatusRead])
	}
	fmt.Println()

	// Print messages
	for _, msg := range messages {
		printInboxMessage(msg, false)
	}
}

func runMessagesAck(args []string) {
	fs := flag.NewFlagSet("messages ack", flag.ExitOnError)
	all := fs.Bool("all", false, "Acknowledge all unread messages")
	inbox := fs.String("inbox", "", "Filter by inbox when using --all")

	if err := fs.Parse(args); err != nil {
		fmt.Fprintf(os.Stderr, "%s: %v\n", red("Error"), err)
		os.Exit(1)
	}

	store, err := openStore()
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s: %v\n", red("Error"), err)
		os.Exit(1)
	}
	defer store.Close()

	if *all {
		count, err := store.MarkAllInboxMessagesRead(*inbox)
		if err != nil {
			fmt.Fprintf(os.Stderr, "%s: %v\n", red("Error"), err)
			os.Exit(1)
		}
		fmt.Printf("%s %d message(s) marked as read.\n", green("✓"), count)
		return
	}

	if fs.NArg() == 0 {
		fmt.Fprintf(os.Stderr, "%s: message ID required (or use --all)\n", red("Error"))
		os.Exit(1)
	}

	msgID := fs.Arg(0)
	if err := store.MarkInboxMessageRead(msgID); err != nil {
		fmt.Fprintf(os.Stderr, "%s: %v\n", red("Error"), err)
		os.Exit(1)
	}
	fmt.Printf("%s Message marked as read.\n", green("✓"))
}

func runMessagesUnack(args []string) {
	fs := flag.NewFlagSet("messages unack", flag.ExitOnError)

	if err := fs.Parse(args); err != nil {
		fmt.Fprintf(os.Stderr, "%s: %v\n", red("Error"), err)
		os.Exit(1)
	}

	if fs.NArg() == 0 {
		fmt.Fprintf(os.Stderr, "%s: message ID required\n", red("Error"))
		os.Exit(1)
	}

	store, err := openStore()
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s: %v\n", red("Error"), err)
		os.Exit(1)
	}
	defer store.Close()

	msgID := fs.Arg(0)
	if err := store.MarkInboxMessageUnread(msgID); err != nil {
		fmt.Fprintf(os.Stderr, "%s: %v\n", red("Error"), err)
		os.Exit(1)
	}
	fmt.Printf("%s Message marked as unread.\n", green("✓"))
}

func runMessagesSend(args []string) {
	fs := flag.NewFlagSet("messages send", flag.ExitOnError)
	jsonPayload := fs.String("json", "", "Send JSON payload instead of plain text")
	title := fs.String("title", "", "Message title")
	from := fs.String("from", "cli", "Sender agent name")
	correlationID := fs.String("correlation", "", "Correlation ID for grouping messages")

	if err := fs.Parse(args); err != nil {
		fmt.Fprintf(os.Stderr, "%s: %v\n", red("Error"), err)
		os.Exit(1)
	}

	if fs.NArg() < 1 {
		fmt.Fprintf(os.Stderr, "%s: inbox required\n", red("Error"))
		fmt.Println("Usage: ailang messages send <inbox> [message]")
		fmt.Println("       ailang messages send <inbox> --json '{...}'")
		os.Exit(1)
	}

	inbox := fs.Arg(0)
	var payload string

	if *jsonPayload != "" {
		payload = *jsonPayload
	} else if fs.NArg() >= 2 {
		payload = strings.Join(fs.Args()[1:], " ")
	} else {
		fmt.Fprintf(os.Stderr, "%s: message content required\n", red("Error"))
		os.Exit(1)
	}

	store, err := openStore()
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s: %v\n", red("Error"), err)
		os.Exit(1)
	}
	defer store.Close()

	// Determine message title
	msgTitle := *title
	if msgTitle == "" {
		msgTitle = truncateString(payload, 50)
	}

	msg := &messaging.InboxMessage{
		FromAgent:     *from,
		ToInbox:       inbox,
		MessageType:   messaging.InboxTypeNotification,
		Title:         msgTitle,
		Payload:       payload,
		CorrelationID: *correlationID,
	}

	if err := store.InsertInboxMessage(msg); err != nil {
		fmt.Fprintf(os.Stderr, "%s: %v\n", red("Error"), err)
		os.Exit(1)
	}

	fmt.Printf("%s Message sent to '%s' (ID: %s)\n", green("✓"), inbox, msg.MessageID)
}

func runMessagesRead(args []string) {
	fs := flag.NewFlagSet("messages read", flag.ExitOnError)
	peek := fs.Bool("peek", false, "Show without marking as read")
	jsonOut := fs.Bool("json", false, "Output as JSON")

	if err := fs.Parse(args); err != nil {
		fmt.Fprintf(os.Stderr, "%s: %v\n", red("Error"), err)
		os.Exit(1)
	}

	if fs.NArg() == 0 {
		fmt.Fprintf(os.Stderr, "%s: message ID required\n", red("Error"))
		os.Exit(1)
	}

	store, err := openStore()
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s: %v\n", red("Error"), err)
		os.Exit(1)
	}
	defer store.Close()

	msgID := fs.Arg(0)
	msg, err := store.GetInboxMessage(msgID)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s: %v\n", red("Error"), err)
		os.Exit(1)
	}

	if msg == nil {
		fmt.Fprintf(os.Stderr, "%s: message not found\n", red("Error"))
		os.Exit(1)
	}

	// Mark as read unless peeking
	if !*peek && msg.Status == messaging.InboxStatusUnread {
		_ = store.MarkInboxMessageRead(msgID) // Ignore error for auto-mark
	}

	if *jsonOut {
		data, _ := json.MarshalIndent(msg, "", "  ")
		fmt.Println(string(data))
		return
	}

	printInboxMessage(*msg, true)
}

func runMessagesWatch(args []string) {
	fs := flag.NewFlagSet("messages watch", flag.ExitOnError)
	inbox := fs.String("inbox", "", "Filter by inbox")
	interval := fs.Duration("interval", time.Second, "Poll interval")

	if err := fs.Parse(args); err != nil {
		fmt.Fprintf(os.Stderr, "%s: %v\n", red("Error"), err)
		os.Exit(1)
	}

	store, err := openStore()
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s: %v\n", red("Error"), err)
		os.Exit(1)
	}
	defer store.Close()

	fmt.Println("Watching for new messages... (Ctrl+C to stop)")
	fmt.Println()

	seen := make(map[string]bool)

	// First, mark all existing messages as seen
	existing, _ := store.ListInboxMessages(messaging.InboxListOptions{
		Inbox:      *inbox,
		UnreadOnly: true,
	})
	for _, msg := range existing {
		seen[msg.ID] = true
	}

	for {
		messages, err := store.ListInboxMessages(messaging.InboxListOptions{
			Inbox:      *inbox,
			UnreadOnly: true,
		})
		if err != nil {
			fmt.Fprintf(os.Stderr, "%s: %v\n", red("Error"), err)
			time.Sleep(*interval)
			continue
		}

		for _, msg := range messages {
			if !seen[msg.ID] {
				seen[msg.ID] = true
				fmt.Printf("%s New message:\n", green("→"))
				printInboxMessage(msg, false)
			}
		}

		time.Sleep(*interval)
	}
}

func runMessagesCleanup(args []string) {
	fs := flag.NewFlagSet("messages cleanup", flag.ExitOnError)
	olderThan := fs.Duration("older-than", 7*24*time.Hour, "Remove messages older than this (e.g., 7d, 24h)")
	expired := fs.Bool("expired", false, "Remove only expired messages")
	dryRun := fs.Bool("dry-run", false, "Show what would be deleted without deleting")

	if err := fs.Parse(args); err != nil {
		fmt.Fprintf(os.Stderr, "%s: %v\n", red("Error"), err)
		os.Exit(1)
	}

	store, err := openStore()
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s: %v\n", red("Error"), err)
		os.Exit(1)
	}
	defer store.Close()

	if *dryRun {
		// Just show counts
		counts, _ := store.CountInboxMessagesByStatus("")
		fmt.Printf("Would clean up messages older than %v:\n", *olderThan)
		fmt.Printf("  Deleted: %d\n", counts[messaging.InboxStatusDeleted])
		fmt.Printf("  (Dry run - no changes made)\n")
		return
	}

	count, err := store.CleanupInboxMessages(*olderThan, *expired)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s: %v\n", red("Error"), err)
		os.Exit(1)
	}

	fmt.Printf("%s Cleaned up %d message(s).\n", green("✓"), count)
}

// printInboxMessage formats and prints a message to stdout.
func printInboxMessage(msg messaging.InboxMessage, full bool) {
	statusIcon := "○"
	if msg.Status == messaging.InboxStatusUnread {
		statusIcon = yellow("●")
	} else if msg.Status == messaging.InboxStatusRead {
		statusIcon = green("○")
	}

	age := formatAge(msg.CreatedAt)

	fmt.Printf("  %s [%s] %s • %s\n",
		statusIcon,
		cyan(msg.ToInbox),
		msg.FromAgent,
		age,
	)

	if msg.Title != "" {
		fmt.Printf("    %s\n", bold(msg.Title))
	}

	// Truncate payload for list view
	payload := msg.Payload
	if !full && len(payload) > 100 {
		payload = payload[:97] + "..."
	}

	// Try to pretty-print JSON
	if strings.HasPrefix(strings.TrimSpace(payload), "{") {
		var parsed map[string]interface{}
		if err := json.Unmarshal([]byte(payload), &parsed); err == nil {
			if full {
				formatted, _ := json.MarshalIndent(parsed, "    ", "  ")
				fmt.Printf("    %s\n", string(formatted))
			} else {
				// Compact summary
				if content, ok := parsed["content"].(string); ok {
					if len(content) > 100 {
						content = content[:97] + "..."
					}
					fmt.Printf("    %s\n", content)
				} else {
					fmt.Printf("    %s\n", payload)
				}
			}
		} else {
			fmt.Printf("    %s\n", payload)
		}
	} else {
		fmt.Printf("    %s\n", payload)
	}

	if full {
		fmt.Printf("\n    ID: %s\n", msg.MessageID)
		if msg.CorrelationID != "" {
			fmt.Printf("    Correlation: %s\n", msg.CorrelationID)
		}
		fmt.Printf("    Status: %s\n", msg.Status)
		fmt.Printf("    Created: %s\n", msg.CreatedAt.Format(time.RFC3339))
		if msg.ReadAt != nil {
			fmt.Printf("    Read: %s\n", msg.ReadAt.Format(time.RFC3339))
		}
	}
	fmt.Println()
}

// formatAge returns a human-readable age string.
func formatAge(t time.Time) string {
	d := time.Since(t)
	if d < time.Minute {
		return "just now"
	} else if d < time.Hour {
		return fmt.Sprintf("%dm ago", int(d.Minutes()))
	} else if d < 24*time.Hour {
		return fmt.Sprintf("%dh ago", int(d.Hours()))
	} else if d < 7*24*time.Hour {
		return fmt.Sprintf("%dd ago", int(d.Hours()/24))
	}
	return t.Format("Jan 2")
}

// truncateString truncates a string to maxLen and adds "..." if needed.
func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}

func printMessagesHelp() {
	fmt.Println("Usage: ailang messages <subcommand> [options]")
	fmt.Println()
	fmt.Println("Unified messaging system for agent-to-agent and human-agent communication.")
	fmt.Println("Messages are accessible from both CLI and Collaboration Hub dashboard.")
	fmt.Println()
	fmt.Println("Subcommands:")
	fmt.Printf("  %s                     List messages (default)\n", cyan("list"))
	fmt.Printf("  %s <id>                Mark message as read\n", cyan("ack"))
	fmt.Printf("  %s <id>              Mark message as unread\n", cyan("unack"))
	fmt.Printf("  %s <inbox> <msg>      Send a message\n", cyan("send"))
	fmt.Printf("  %s <id>               Show full message content\n", cyan("read"))
	fmt.Printf("  %s                    Watch for new messages\n", cyan("watch"))
	fmt.Printf("  %s                  Clean up old messages\n", cyan("cleanup"))
	fmt.Println()
	fmt.Println("List Flags:")
	fmt.Println("  --inbox <name>       Filter by inbox (user, claude-code, etc.)")
	fmt.Println("  --unread             Show only unread messages")
	fmt.Println("  --from <agent>       Filter by sender")
	fmt.Println("  --limit <n>          Maximum messages to show (default: 20)")
	fmt.Println("  --json               Output as JSON")
	fmt.Println()
	fmt.Println("Ack Flags:")
	fmt.Println("  --all                Acknowledge all unread messages")
	fmt.Println("  --inbox <name>       Filter by inbox when using --all")
	fmt.Println()
	fmt.Println("Send Flags:")
	fmt.Println("  --json <payload>     Send JSON payload")
	fmt.Println("  --title <text>       Message title")
	fmt.Println("  --from <agent>       Sender name (default: cli)")
	fmt.Println("  --correlation <id>   Correlation ID for grouping")
	fmt.Println()
	fmt.Println("Aliases: msg, messages")
	fmt.Println()
	fmt.Println("Examples:")
	fmt.Printf("  %s                         # List all messages\n", cyan("ailang messages list"))
	fmt.Printf("  %s               # List unread only\n", cyan("ailang messages list --unread"))
	fmt.Printf("  %s          # Filter by inbox\n", cyan("ailang messages list --inbox user"))
	fmt.Printf("  %s              # Mark as read\n", cyan("ailang messages ack MSG_ID"))
	fmt.Printf("  %s                     # Ack all unread\n", cyan("ailang messages ack --all"))
	fmt.Printf("  %s   # Send message\n", cyan("ailang messages send user \"Hello\""))
	fmt.Printf("  %s    # Watch for new\n", cyan("ailang messages watch --inbox user"))
}
