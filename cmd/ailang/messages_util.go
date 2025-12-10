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

// Utility functions and watch/cleanup operations for messages

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
