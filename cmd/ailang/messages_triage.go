package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"sort"

	"github.com/sunholo-data/ailang/internal/messaging"
)

// Triage subcommand: cluster unread messages by envelope similarity.
// Clustering itself lives in internal/messaging (ClusterMessages) so the
// coordinator's triage router can reuse it.

func runMessagesTriage(args []string) {
	fs := flag.NewFlagSet("messages triage", flag.ExitOnError)
	inbox := fs.String("inbox", "", "Filter by inbox")
	clusterBy := fs.String("cluster-by", "intent", "Envelope slot to cluster on (intent, code, context, skill, resolution)")
	topN := fs.Int("top", 10, "Show top-N clusters")
	threshold := fs.Float64("threshold", 0.50, "Minimum similarity for clustering (0.0-1.0)")
	jsonOut := fs.Bool("json", false, "Output as JSON")

	if err := fs.Parse(args); err != nil {
		fmt.Fprintf(os.Stderr, "%s: %v\n", red("Error"), err)
		os.Exit(1)
	}

	// Validate slot
	if err := messaging.ValidateSlot(*clusterBy); err != nil {
		fmt.Fprintf(os.Stderr, "%s: %v\n", red("Error"), err)
		fmt.Fprintf(os.Stderr, "Valid slots: intent, code, context, skill, resolution\n")
		os.Exit(1)
	}

	store, err := openStore()
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s: %v\n", red("Error"), err)
		os.Exit(1)
	}
	defer store.Close()

	// Resolve inbox
	resolvedInbox := *inbox
	if resolvedInbox == "" {
		resolvedInbox = inferInbox()
		if resolvedInbox != "" {
			fmt.Printf("Using inbox: %s (inferred from repo)\n", cyan(resolvedInbox))
		}
	}

	// Get unread messages
	messages, err := store.ListInboxMessages(messaging.InboxListOptions{
		Inbox:  resolvedInbox,
		Status: "unread",
		Limit:  200,
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s: %v\n", red("Error"), err)
		os.Exit(1)
	}

	if len(messages) == 0 {
		fmt.Println("No unread messages to triage.")
		return
	}

	// Filter to messages with envelopes containing the requested slot
	var withSlot []messaging.InboxMessage
	for _, msg := range messages {
		if msg.Envelope != nil && msg.Envelope.Get(*clusterBy) != nil {
			withSlot = append(withSlot, msg)
		}
	}

	if len(withSlot) < 2 {
		// Not enough envelopes for clustering — show flat list
		fmt.Printf("\n%s (%d unread, %d have '%s' envelope)\n\n",
			bold("Triage"), len(messages), len(withSlot), *clusterBy)
		fmt.Println("Not enough envelope data for clustering. Showing flat list:")
		fmt.Println()
		for i, msg := range messages {
			fmt.Printf("  %d. ", i+1)
			printInboxMessage(msg, false)
		}
		printSearchFooter("SQLite", "triage", len(messages), *threshold)
		return
	}

	// Greedy threshold-based clustering
	clusters := messaging.ClusterMessages(withSlot, *clusterBy, *threshold)

	// Sort clusters by size (largest first)
	sort.Slice(clusters, func(i, j int) bool {
		return clusters[i].Count > clusters[j].Count
	})

	// Limit to top-N
	if len(clusters) > *topN {
		clusters = clusters[:*topN]
	}

	// Add any messages without envelopes as "Uncategorized"
	var uncategorized []messaging.InboxMessage
	for _, msg := range messages {
		if msg.Envelope == nil || msg.Envelope.Get(*clusterBy) == nil {
			uncategorized = append(uncategorized, msg)
		}
	}
	if len(uncategorized) > 0 {
		clusters = append(clusters, messaging.Cluster{
			Label:    "Uncategorized (no envelope)",
			Slot:     *clusterBy,
			Count:    len(uncategorized),
			Messages: uncategorized,
		})
	}

	if *jsonOut {
		data, _ := json.MarshalIndent(clusters, "", "  ")
		fmt.Println(string(data))
		return
	}

	// Print triage report
	fmt.Printf("\n%s — %d messages, %d clusters (by %s)\n\n",
		bold("Triage Report"), len(messages), len(clusters), *clusterBy)

	for i, c := range clusters {
		fmt.Printf("Cluster %d: %s (%d msgs)\n", i+1, bold(c.Label), c.Count)
		for _, msg := range c.Messages {
			shortID := msg.MessageID
			if len(shortID) > 8 {
				shortID = shortID[:8]
			}
			fmt.Printf("  %s %s %s\n", dim(shortID), msg.Title, dim(msg.FromAgent))
		}
		fmt.Println()
	}

	printSearchFooter("SQLite", "triage:"+*clusterBy, len(messages), *threshold)
}
