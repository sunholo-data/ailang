package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"

	"github.com/sunholo/ailang/internal/messaging"
)

// CRUD operations for messages: list, ack, unack, read

func runMessagesList(args []string) {
	fs := flag.NewFlagSet("messages list", flag.ExitOnError)
	inbox := fs.String("inbox", "", "Filter by inbox (user, claude-code, etc.)")
	unread := fs.Bool("unread", false, "Show only unread messages")
	from := fs.String("from", "", "Filter by sender agent")
	limit := fs.Int("limit", 20, "Maximum messages to show")
	jsonOut := fs.Bool("json", false, "Output as JSON")
	similarTo := fs.String("similar-to", "", "Find messages similar to this message ID")
	collapsed := fs.Bool("collapsed", false, "Hide duplicate messages (where dup_of is set)")
	duplicatesOf := fs.String("duplicates-of", "", "Show only duplicates of this message ID")
	threshold := fs.Float64("threshold", 0.70, "Similarity threshold for --similar-to (0.0-1.0)")

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

	// Handle --similar-to mode
	if *similarTo != "" {
		hits, err := store.FindSimilar(*similarTo, *threshold, *limit)
		if err != nil {
			fmt.Fprintf(os.Stderr, "%s: %v\n", red("Error"), err)
			os.Exit(1)
		}

		if *jsonOut {
			data, _ := json.MarshalIndent(hits, "", "  ")
			fmt.Println(string(data))
			return
		}

		if len(hits) == 0 {
			fmt.Println("No similar messages found.")
			return
		}

		fmt.Printf("\n%s %s:\n\n", bold("Similar messages to"), *similarTo)
		for _, hit := range hits {
			fmt.Printf("  [%.0f%% similar] ", hit.Score*100)
			printInboxMessage(hit.Message, false)
		}
		printSearchFooter("SQLite", "simhash", len(hits), *threshold)
		return
	}

	opts := messaging.InboxListOptions{
		Inbox:      *inbox,
		FromAgent:  *from,
		Limit:      *limit,
		UnreadOnly: *unread,
		Collapsed:  *collapsed,
		DupOf:      *duplicatesOf,
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

	// Resolve short ID prefix to full ID
	msgID, err := resolveMessageID(store, fs.Arg(0))
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s: %v\n", red("Error"), err)
		os.Exit(1)
	}

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

	// Resolve short ID prefix to full ID
	msgID, err := resolveMessageID(store, fs.Arg(0))
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s: %v\n", red("Error"), err)
		os.Exit(1)
	}

	if err := store.MarkInboxMessageUnread(msgID); err != nil {
		fmt.Fprintf(os.Stderr, "%s: %v\n", red("Error"), err)
		os.Exit(1)
	}
	fmt.Printf("%s Message marked as unread.\n", green("✓"))
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

	// Resolve short ID prefix to full ID
	msgID, err := resolveMessageID(store, fs.Arg(0))
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s: %v\n", red("Error"), err)
		os.Exit(1)
	}

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

func runMessagesForward(args []string) {
	fs := flag.NewFlagSet("messages forward", flag.ExitOnError)
	toInbox := fs.String("to", "", "Target inbox (required)")
	reason := fs.String("reason", "", "Reason for forwarding (logged)")

	if err := fs.Parse(args); err != nil {
		fmt.Fprintf(os.Stderr, "%s: %v\n", red("Error"), err)
		os.Exit(1)
	}

	if *toInbox == "" {
		fmt.Fprintf(os.Stderr, "%s: --to flag is required\n", red("Error"))
		fmt.Fprintf(os.Stderr, "\nUsage: ailang messages forward <MSG_ID> --to <inbox>\n")
		fmt.Fprintf(os.Stderr, "\nAvailable inboxes: user, design-doc-creator, sprint-planner, sprint-executor, coordinator\n")
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

	// Resolve short ID prefix to full ID
	msgID, err := resolveMessageID(store, fs.Arg(0))
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s: %v\n", red("Error"), err)
		os.Exit(1)
	}

	// Get the message to verify it exists and show what we're forwarding
	msg, err := store.GetInboxMessage(msgID)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s: %v\n", red("Error"), err)
		os.Exit(1)
	}
	if msg == nil {
		fmt.Fprintf(os.Stderr, "%s: message not found\n", red("Error"))
		os.Exit(1)
	}

	oldInbox := msg.ToInbox

	// Forward the message (update to_id in database)
	if err := store.ForwardInboxMessage(msgID, *toInbox); err != nil {
		fmt.Fprintf(os.Stderr, "%s: %v\n", red("Error"), err)
		os.Exit(1)
	}

	// Log the forward
	reasonStr := ""
	if *reason != "" {
		reasonStr = fmt.Sprintf(" (reason: %s)", *reason)
	}

	fmt.Printf("%s Forwarded message from '%s' to '%s'%s\n",
		green("✓"), oldInbox, *toInbox, reasonStr)
	fmt.Printf("   Title: %s\n", msg.Title)
	fmt.Printf("   ID: %s\n", msgID[:8]+"...")
}
