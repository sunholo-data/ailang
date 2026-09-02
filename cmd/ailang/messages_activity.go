package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/sunholo-data/ailang/internal/messaging"
)

// M-COORDINATOR-EXECUTION-TRUST M4, part 2 — the fleet activity view.
//
// The condition that decides whether a person PREFERS the message plane over an
// interactive session is not correctness, it is legibility: you have to be able
// to see what the fleet did without asking an agent to go and look. Every number
// below was assembled by hand during the 2026-09-02 audit, out of `gcloud run
// jobs executions list`, coordinator logs and Firestore. The inputs already
// existed; nothing joined them.
//
// Deliberately a CLI that prints the truth — no dashboard, no alerting, and no
// dependency on the GCP APIs, so it works against any store the CLI can already
// read.

type activityCounts struct {
	Plane        string         `json:"plane,omitempty"`
	Project      string         `json:"project,omitempty"`
	WindowHours  float64        `json:"window_hours"`
	Total        int            `json:"total"`
	ByType       map[string]int `json:"by_type"`
	ByInbox      map[string]int `json:"by_inbox"`
	ByOutcome    map[string]int `json:"outcomes"`
	Unread       int            `json:"unread"`
	OldestUnread string         `json:"oldest_unread,omitempty"`
	Senders      map[string]int `json:"senders"`
}

// summarizeActivity folds a message window into the counts above. Pure, so the
// shape of the report is testable without a store.
func summarizeActivity(msgs []messaging.InboxMessage, since time.Time, hours float64) activityCounts {
	a := activityCounts{
		WindowHours: hours,
		ByType:      map[string]int{},
		ByInbox:     map[string]int{},
		ByOutcome:   map[string]int{},
		Senders:     map[string]int{},
	}
	var oldestUnread time.Time

	for i := range msgs {
		m := msgs[i]
		if m.CreatedAt.Before(since) {
			continue
		}
		a.Total++
		a.ByType[strings.TrimSpace(m.MessageType)]++
		a.ByInbox[m.ToInbox]++
		a.Senders[m.FromAgent]++

		if m.Status != "read" {
			a.Unread++
			if oldestUnread.IsZero() || m.CreatedAt.Before(oldestUnread) {
				oldestUnread = m.CreatedAt
			}
		}

		// A completion carries the outcome that actually matters — including
		// no_changes, which is the whole point of M2 and is invisible in a plain
		// message listing.
		if m.MessageType == "completion" {
			var payload struct {
				Status string `json:"status"`
			}
			if err := json.Unmarshal([]byte(m.Payload), &payload); err == nil && payload.Status != "" {
				a.ByOutcome[payload.Status]++
			} else {
				a.ByOutcome["unparseable"]++
			}
		}
	}
	if !oldestUnread.IsZero() {
		a.OldestUnread = oldestUnread.UTC().Format(time.RFC3339)
	}
	return a
}

func runMessagesActivity(args []string) {
	fs := flag.NewFlagSet("messages activity", flag.ExitOnError)
	hours := fs.Float64("hours", 24, "Window to summarize, in hours")
	asJSON := fs.Bool("json", false, "Emit the summary as JSON")
	if err := fs.Parse(args); err != nil {
		return
	}

	store, err := openStore()
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s cannot open the message store: %v\n", red("Error"), err)
		os.Exit(1)
	}
	defer func() { _ = store.Close() }()

	// A large limit rather than a time filter: the store's list options do not
	// take one, and quietly summarizing a truncated window would be its own
	// silent-wrong-answer bug.
	msgs, err := store.ListInboxMessages(messaging.InboxListOptions{Limit: 5000})
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s cannot list messages: %v\n", red("Error"), err)
		os.Exit(1)
	}

	_, project := messagesTarget()
	since := time.Now().Add(-time.Duration(*hours * float64(time.Hour)))
	a := summarizeActivity(msgs, since, *hours)
	a.Project = project
	a.Plane = planeLabel(project)

	if *asJSON {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		_ = enc.Encode(a)
		return
	}

	fmt.Println()
	plane := a.Plane
	if plane == "" {
		plane = "unknown plane"
	} else {
		plane = strings.ToUpper(plane) + " plane"
	}
	fmt.Printf("%s  %s\n", bold(fmt.Sprintf("Fleet activity — last %gh", *hours)), bold("["+plane+"]"))
	if a.Project != "" {
		fmt.Printf("  project: %s\n", a.Project)
	}
	fmt.Println()

	if a.Total == 0 {
		fmt.Printf("  no messages in the window — the fleet did nothing, or is not reaching this store\n\n")
		return
	}

	fmt.Printf("  messages:  %d   (unread: %d)\n", a.Total, a.Unread)
	if a.OldestUnread != "" {
		fmt.Printf("  oldest unread: %s\n", a.OldestUnread)
	}
	printCountSection("by type", a.ByType)
	printCountSection("task outcomes", a.ByOutcome)
	printCountSection("busiest inboxes", a.ByInbox)
	printCountSection("busiest senders", a.Senders)

	// The outcome that used to be invisible gets called out by name.
	if n := a.ByOutcome[string("no_changes")]; n > 0 {
		fmt.Printf("\n  %d task(s) ran and changed nothing. That is now reported rather than\n", n)
		fmt.Printf("  counted as success — check whether they were meant to be no-ops.\n")
	}
	fmt.Println()
}

// printCountSection prints a count map highest-first, capped so one noisy inbox
// cannot bury the rest.
func printCountSection(title string, counts map[string]int) {
	if len(counts) == 0 {
		return
	}
	type kv struct {
		k string
		v int
	}
	rows := make([]kv, 0, len(counts))
	for k, v := range counts {
		if k == "" {
			k = "(unset)"
		}
		rows = append(rows, kv{k, v})
	}
	sort.Slice(rows, func(i, j int) bool {
		if rows[i].v != rows[j].v {
			return rows[i].v > rows[j].v
		}
		return rows[i].k < rows[j].k
	})
	fmt.Printf("\n  %s:\n", title)
	for i, r := range rows {
		if i >= 8 {
			fmt.Printf("      … and %d more\n", len(rows)-i)
			break
		}
		fmt.Printf("    %5d  %s\n", r.v, r.k)
	}
}
