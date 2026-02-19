package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"math"
	"os"
	"sort"

	"github.com/sunholo/ailang/internal/messaging"
)

// Triage subcommand: cluster unread messages by envelope similarity

type triageCluster struct {
	Label    string                   `json:"label"`
	Slot     string                   `json:"slot"`
	Count    int                      `json:"count"`
	Messages []messaging.InboxMessage `json:"messages"`
}

func runMessagesTriage(args []string) {
	fs := flag.NewFlagSet("messages triage", flag.ExitOnError)
	inbox := fs.String("inbox", "", "Filter by inbox")
	clusterBy := fs.String("cluster-by", "intent", "Envelope slot to cluster on (intent, code, context, skill, resolution)")
	topN := fs.Int("top", 10, "Show top-N clusters")
	threshold := fs.Float64("threshold", 0.75, "Minimum similarity for clustering (0.0-1.0)")
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
	clusters := clusterMessages(withSlot, *clusterBy, *threshold)

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
		clusters = append(clusters, triageCluster{
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

// clusterMessages performs greedy threshold-based clustering on messages
// using the specified envelope slot.
func clusterMessages(messages []messaging.InboxMessage, slot string, threshold float64) []triageCluster {
	assigned := make([]bool, len(messages))
	var clusters []triageCluster

	for i := 0; i < len(messages); i++ {
		if assigned[i] {
			continue
		}
		assigned[i] = true

		vec := messages[i].Envelope.Get(slot)
		if vec == nil {
			continue
		}

		cluster := triageCluster{
			Label:    messages[i].Title,
			Slot:     slot,
			Count:    1,
			Messages: []messaging.InboxMessage{messages[i]},
		}

		// Find similar messages
		for j := i + 1; j < len(messages); j++ {
			if assigned[j] {
				continue
			}
			other := messages[j].Envelope.Get(slot)
			if other == nil {
				continue
			}
			sim := cosineSimilarity(vec.Vector, other.Vector)
			if sim >= threshold {
				assigned[j] = true
				cluster.Messages = append(cluster.Messages, messages[j])
				cluster.Count++
			}
		}

		clusters = append(clusters, cluster)
	}

	return clusters
}

// cosineSimilarity computes the cosine similarity between two vectors.
func cosineSimilarity(a, b []float32) float64 {
	if len(a) != len(b) || len(a) == 0 {
		return 0
	}
	var dot, normA, normB float64
	for i := range a {
		dot += float64(a[i]) * float64(b[i])
		normA += float64(a[i]) * float64(a[i])
		normB += float64(b[i]) * float64(b[i])
	}
	denom := math.Sqrt(normA) * math.Sqrt(normB)
	if denom == 0 {
		return 0
	}
	return dot / denom
}
