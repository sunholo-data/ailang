package main

import (
	"bufio"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/sunholo/ailang/internal/messaging"
	"github.com/sunholo/ailang/internal/pubsub"
	"golang.org/x/term"
)

// Utility functions and watch/cleanup operations for messages

// isTerminal returns true if stdin is a terminal (interactive mode).
func isTerminal() bool {
	return term.IsTerminal(int(os.Stdin.Fd()))
}

// runMessagesInteractive shows an interactive menu for managing messages.
func runMessagesInteractive() {
	store, err := openStore()
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s: %v\n", red("Error"), err)
		os.Exit(1)
	}
	defer store.Close()

	reader := bufio.NewReader(os.Stdin)
	selectedIdx := -1

	for {
		// Get messages
		messages, err := store.ListInboxMessages(messaging.InboxListOptions{
			Limit: 20,
		})
		if err != nil {
			fmt.Fprintf(os.Stderr, "%s: %v\n", red("Error"), err)
			fmt.Println("Press Enter to retry...")
			reader.ReadString('\n')
			continue // Stay in menu, retry on next loop
		}

		// Get unread count
		counts, _ := store.CountInboxMessagesByStatus("")
		unreadCount := counts[messaging.InboxStatusUnread]

		// Clear screen and print header
		fmt.Print("\033[H\033[2J") // ANSI clear screen
		fmt.Println("┌─────────────────────────────────────────────────────────────────┐")
		fmt.Printf("│ AILANG Messages (%d unread)                                      │\n", unreadCount)
		fmt.Println("├─────────────────────────────────────────────────────────────────┤")

		if len(messages) == 0 {
			fmt.Println("│ No messages.                                                    │")
		} else {
			for i, msg := range messages {
				unreadMark := " "
				if msg.Status == messaging.InboxStatusUnread {
					unreadMark = yellow("●")
				}

				// Truncate title to fit
				title := msg.Title
				if len(title) > 45 {
					title = title[:42] + "..."
				}

				// Show GitHub issue number if available
				issueStr := ""
				if msg.GitHubIssue != nil {
					issueStr = fmt.Sprintf("#%d", *msg.GitHubIssue)
				}

				// Highlight selected row
				prefix := " "
				if i == selectedIdx {
					prefix = cyan(">")
				}

				fmt.Printf("│%s[%d] %s %-45s %6s │\n", prefix, i+1, unreadMark, title, issueStr)
			}
		}

		fmt.Println("└─────────────────────────────────────────────────────────────────┘")
		fmt.Println()
		fmt.Println("Actions: [1-9] select  [r]ead  [f]orward  [a]ck  [A]ck all  [q]uit")
		fmt.Println()

		if selectedIdx >= 0 && selectedIdx < len(messages) {
			msg := messages[selectedIdx]
			fmt.Printf("Selected: %s (ID: %s)\n", bold(msg.Title), msg.ID[:8])
		}

		fmt.Print("Enter command: ")

		input, err := reader.ReadString('\n')
		if err != nil {
			// EOF (Ctrl+D) or other input error - graceful exit
			fmt.Println("\nExiting...")
			return
		}
		input = strings.TrimSpace(input)

		if input == "" {
			continue
		}

		// Handle single character commands
		switch input {
		case "q", "quit", "exit":
			fmt.Println("Goodbye!")
			return

		case "r", "read":
			if selectedIdx < 0 || selectedIdx >= len(messages) {
				fmt.Println(red("No message selected. Press 1-9 to select."))
				waitForEnter(reader)
				continue
			}
			msg := messages[selectedIdx]
			_ = store.MarkInboxMessageRead(msg.ID)
			printInboxMessage(msg, true)
			waitForEnter(reader)

		case "f", "forward":
			if selectedIdx < 0 || selectedIdx >= len(messages) {
				fmt.Println(red("No message selected. Press 1-9 to select."))
				waitForEnter(reader)
				continue
			}
			msg := messages[selectedIdx]
			fmt.Print("Forward to inbox (design-doc-creator/sprint-planner/coordinator): ")
			target, _ := reader.ReadString('\n')
			target = strings.TrimSpace(target)
			if target != "" {
				if err := store.ForwardInboxMessage(msg.ID, target); err != nil {
					fmt.Printf("%s: %v\n", red("Error"), err)
				} else {
					fmt.Printf("%s Forwarded to %s\n", green("✓"), target)
				}
				waitForEnter(reader)
			}

		case "a", "ack":
			if selectedIdx < 0 || selectedIdx >= len(messages) {
				fmt.Println(red("No message selected. Press 1-9 to select."))
				waitForEnter(reader)
				continue
			}
			msg := messages[selectedIdx]
			if err := store.MarkInboxMessageRead(msg.ID); err != nil {
				fmt.Printf("%s: %v\n", red("Error"), err)
			} else {
				fmt.Printf("%s Message marked as read.\n", green("✓"))
			}
			selectedIdx = -1 // Deselect
			time.Sleep(500 * time.Millisecond)

		case "A":
			count, err := store.MarkAllInboxMessagesRead("")
			if err != nil {
				fmt.Printf("%s: %v\n", red("Error"), err)
			} else {
				fmt.Printf("%s %d message(s) marked as read.\n", green("✓"), count)
			}
			selectedIdx = -1
			time.Sleep(500 * time.Millisecond)

		default:
			// Try to parse as number for selection
			if num, err := strconv.Atoi(input); err == nil && num >= 1 && num <= len(messages) {
				selectedIdx = num - 1
			} else {
				fmt.Printf("%s Unknown command: %s\n", yellow("?"), input)
				waitForEnter(reader)
			}
		}
	}
}

// waitForEnter waits for user to press Enter.
func waitForEnter(reader *bufio.Reader) {
	fmt.Print("\nPress Enter to continue...")
	reader.ReadString('\n')
}

// resolveMessageID resolves a short ID prefix to a full message ID.
// Returns error if no match or multiple matches (ambiguous prefix).
func resolveMessageID(store messaging.MessageStore, prefix string) (string, error) {
	// If prefix looks like a full UUID, use it directly
	if len(prefix) >= 36 {
		return prefix, nil
	}

	// Query messages where ID starts with prefix
	messages, err := store.ListInboxMessages(messaging.InboxListOptions{
		Limit: 100, // Reasonable limit for prefix search
	})
	if err != nil {
		return "", err
	}

	var matches []string
	for _, msg := range messages {
		if strings.HasPrefix(msg.ID, prefix) {
			matches = append(matches, msg.ID)
		}
	}

	switch len(matches) {
	case 0:
		return "", fmt.Errorf("no message found with prefix '%s'", prefix)
	case 1:
		return matches[0], nil
	default:
		return "", fmt.Errorf("ambiguous prefix '%s' matches %d messages, use a longer prefix", prefix, len(matches))
	}
}

func runMessagesWatch(args []string) {
	fs := flag.NewFlagSet("messages watch", flag.ExitOnError)
	inbox := fs.String("inbox", "", "Filter by inbox")
	interval := fs.Duration("interval", time.Second, "Poll interval")
	usePubSub := fs.Bool("pubsub", false, "Use Pub/Sub pull subscription instead of SQLite polling (M-PUBSUB)")

	if err := fs.Parse(args); err != nil {
		fmt.Fprintf(os.Stderr, "%s: %v\n", red("Error"), err)
		os.Exit(1)
	}

	// Pub/Sub mode: pull from subscription
	if *usePubSub {
		runMessagesWatchPubSub(*inbox)
		return
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

// runMessagesWatchPubSub uses Pub/Sub pull subscription for real-time message watching.
func runMessagesWatchPubSub(inbox string) {
	cfg, err := messaging.LoadConfig()
	if err != nil || cfg == nil || cfg.PubSub == nil || !cfg.PubSub.Enabled {
		fmt.Fprintf(os.Stderr, "%s: Pub/Sub not enabled in config\n", red("Error"))
		fmt.Fprintln(os.Stderr, "Add to ~/.ailang/config.yaml:")
		fmt.Fprintln(os.Stderr, "  pubsub:")
		fmt.Fprintln(os.Stderr, "    enabled: true")
		fmt.Fprintln(os.Stderr, "    project_id: your-project-id")
		os.Exit(1)
	}

	notifier, err := messaging.NewPubSubNotifier(cfg.PubSub)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s: %v\n", red("Error"), err)
		os.Exit(1)
	}
	if notifier != nil {
		defer notifier.Close()
	}

	// Create a subscriber directly for pull mode
	projectID := cfg.PubSub.ProjectID
	if projectID == "" {
		projectID = os.Getenv("AILANG_CLOUD_PROJECT")
	}
	if projectID == "" {
		projectID = os.Getenv("GOOGLE_CLOUD_PROJECT")
	}

	prefix := cfg.PubSub.TopicPrefix
	if prefix == "" {
		prefix = pubsub.DefaultTopicPrefix
	}

	ctx := context.Background()
	client, err := pubsub.NewClient(ctx, projectID, prefix)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s: %v\n", red("Error"), err)
		os.Exit(1)
	}
	defer client.Close()

	subscriber := pubsub.NewSubscriber(client)

	subName := pubsub.SubMessagesLaptop
	fmt.Printf("Watching Pub/Sub subscription '%s' for messages... (Ctrl+C to stop)\n\n",
		client.SubscriptionName(subName))

	err = subscriber.Subscribe(ctx, subName, func(ctx context.Context, data []byte, attrs map[string]string) error {
		notification, decErr := pubsub.DecodeMessageNotification(data)
		if decErr != nil {
			fmt.Fprintf(os.Stderr, "%s: decode error: %v\n", yellow("!"), decErr)
			return nil
		}

		msgAttrs := pubsub.AttributesFromMap(attrs)

		// Filter by inbox if specified
		if inbox != "" && msgAttrs.Inbox != inbox {
			return nil
		}

		fmt.Printf("%s New message via Pub/Sub:\n", green("→"))
		fmt.Printf("  ID:    %s\n", notification.MessageID)
		fmt.Printf("  Inbox: %s\n", msgAttrs.Inbox)
		fmt.Printf("  From:  %s\n", msgAttrs.FromAgent)
		if msgAttrs.Category != "" {
			fmt.Printf("  Type:  %s\n", msgAttrs.Category)
		}
		fmt.Println()
		return nil
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s: subscription error: %v\n", red("Error"), err)
		os.Exit(1)
	}
}

func runMessagesCleanup(args []string) {
	fs := flag.NewFlagSet("messages cleanup", flag.ExitOnError)
	olderThan := humanDuration(7 * 24 * time.Hour) // Default: 7 days
	fs.Var(&olderThan, "older-than", "Remove messages older than this (e.g., 7d, 30d, 168h)")
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
		fmt.Printf("Would clean up messages older than %v:\n", time.Duration(olderThan))
		fmt.Printf("  Deleted: %d\n", counts[messaging.InboxStatusDeleted])
		fmt.Printf("  (Dry run - no changes made)\n")
		return
	}

	count, err := store.CleanupInboxMessages(time.Duration(olderThan), *expired)
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
