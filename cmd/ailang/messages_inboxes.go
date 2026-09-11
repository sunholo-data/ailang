package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/sunholo-data/ailang/internal/coordinator"
	"github.com/sunholo-data/ailang/internal/messaging"
	"github.com/sunholo-data/ailang/internal/storage"
)

// `ailang messages inboxes` — what does sending to this name actually DO?
//
// Nothing answered that before. The registry is a 700-line YAML in a GCS bucket,
// the effect of a name is spread across agent config, wildcard patterns and a
// `triage_only_inboxes` list, and the only feedback a sender got was silence.
// Measured 2026-09-08..10: fifteen messages to `ailang-core`,
// `pkg:sunholo/ailang` and `pkg:sunholo/email_parse` — none of which exist —
// were accepted and never dispatched, and nobody noticed for two days.
//
// Every inbox is exactly one of three things, and the command's job is to say
// which, in the sender's terms:
//
//	DISPATCHES  an agent is registered; a send creates a task and work starts
//	TRIAGE      declared human-triage; a send is filed for a person, on purpose
//	NOTHING     neither; a send is accepted and silently goes nowhere
//
// The third row is the whole point. It is the state that looks identical to the
// first from where a sender stands.

type inboxEffect string

const (
	effectDispatches inboxEffect = "DISPATCHES"
	effectTriage     inboxEffect = "TRIAGE"
	effectNothing    inboxEffect = "NOTHING"
)

type inboxRow struct {
	Inbox   string      `json:"inbox"`
	Effect  inboxEffect `json:"effect"`
	Agent   string      `json:"agent,omitempty"`
	Model   string      `json:"model,omitempty"`
	Pattern bool        `json:"pattern,omitempty"`
	Detail  string      `json:"detail"`
	// Unread is how many messages are sitting in it right now. On a NOTHING row
	// this is the count of things that went nowhere.
	Unread int `json:"unread"`
	// Recent counts messages of ANY status in the window. Without it a mistyped
	// inbox disappears from this listing the moment someone reads its messages —
	// which is exactly when you still need to go and fix the sender. Measured
	// while building this: four `ailang-core` reports were triaged mid-session
	// and the bad name vanished with them.
	Recent int `json:"recent"`
}

type inboxesOutput struct {
	Registry string     `json:"registry"`
	Store    string     `json:"store,omitempty"`
	Rows     []inboxRow `json:"inboxes"`
}

func messagesInboxesCommand(args []string) error {
	fs := flag.NewFlagSet("messages inboxes", flag.ExitOnError)
	registryPath := fs.String("registry", "", "config to judge against (default: this machine's ~/.ailang/config.yaml)")
	asJSON := fs.Bool("json", false, "emit as JSON")
	all := fs.Bool("all", false, "include registered inboxes with no recent traffic")
	days := fs.Int("days", 14, "window for the traffic counts")
	_ = fs.Parse(args)

	// WHICH REGISTRY — the part that decides whether this command informs or
	// misleads. Judged against this machine's config (2 agents) while reading the
	// prod store (35), every correctly-routed inbox renders as NOTHING: sixteen
	// false alarms, and the one real problem buried among them. `messages health`
	// has the same hazard and settles for a warning; a command whose entire
	// output is a routing verdict cannot.
	//
	// So it RESOLVES the registry that matches the store rather than assuming
	// one, and says which it used. Explicit --registry always wins.
	registry, regPath, err := resolveInboxRegistry(*registryPath)
	if err != nil {
		return err
	}

	// Counting what is actually sitting in each inbox is what turns a config
	// listing into an answer. A NOTHING row with 4 unread is a live problem; the
	// same row with 0 is a name nobody has used.
	unread, recent := map[string]int{}, map[string]int{}
	if msgStore, sErr := openStore(); sErr == nil {
		defer func() { _ = msgStore.Close() }()
		since := time.Now().AddDate(0, 0, -*days).Format("2006-01-02")
		if msgs, lErr := msgStore.ListInboxMessages(messaging.InboxListOptions{StartDate: since}); lErr == nil {
			for _, m := range msgs {
				recent[m.ToInbox]++
				if m.Status == "unread" {
					unread[m.ToInbox]++
				}
			}
		}
	}

	rows := buildInboxRows(registry, unread, recent, *all)
	out := inboxesOutput{Registry: regPath, Rows: rows}

	if *asJSON {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(out)
	}
	printInboxRows(out)
	return nil
}

// buildInboxRows merges the registry with what is actually in the store.
//
// Inboxes that exist ONLY in the store — a name someone sent to that the
// registry has never heard of — are included unconditionally. They are the
// failure this command exists to surface, and omitting them because they are
// not configured would reproduce the original blindness exactly.
func buildInboxRows(registry *coordinator.AgentRegistry, unread, recent map[string]int, all bool) []inboxRow {
	seen := map[string]bool{}
	var rows []inboxRow

	for _, inbox := range registry.ListInboxes() {
		seen[inbox] = true
		n := unread[inbox]
		if recent[inbox] == 0 && !all {
			// Still list a pattern: it is the least discoverable kind of name.
			if !strings.ContainsRune(inbox, '*') {
				continue
			}
		}
		row := inboxRow{Inbox: inbox, Effect: effectDispatches, Unread: n, Recent: recent[inbox], Pattern: strings.ContainsRune(inbox, '*')}
		if agent := registry.GetAgentForInbox(inbox); agent != nil {
			row.Agent = agent.ID
			row.Model = agent.Model
			row.Detail = fmt.Sprintf("creates a task for %s", agent.ID)
		} else if row.Pattern {
			row.Detail = "wildcard — any matching name dispatches"
		}
		rows = append(rows, row)
	}

	// Everything the store has seen, whether or not the registry knows it.
	for inbox, n := range recent {
		if seen[inbox] {
			continue
		}
		seen[inbox] = true
		row := inboxRow{Inbox: inbox, Unread: unread[inbox], Recent: n}
		switch {
		case registry.GetAgentForInbox(inbox) != nil:
			agent := registry.GetAgentForInbox(inbox)
			row.Effect, row.Agent, row.Model = effectDispatches, agent.ID, agent.Model
			row.Detail = fmt.Sprintf("creates a task for %s", agent.ID)
		case registry.IsTriageOnly(inbox):
			row.Effect = effectTriage
			row.Detail = "filed for a human, on purpose"
		default:
			row.Effect = effectNothing
			row.Detail = "accepted and never dispatched — no agent, not declared triage"
			if s := coordinator.SuggestInbox(inbox, registry.ListInboxes()); s != "" {
				row.Detail += fmt.Sprintf("; did you mean %q?", s)
			}
		}
		rows = append(rows, row)
	}

	// Declared triage inboxes with nothing in them still deserve a line under
	// --all: their whole purpose is to make "unrouted" mean intended.
	if all {
		for _, inbox := range registry.TriageInboxes() {
			if seen[inbox] {
				continue
			}
			rows = append(rows, inboxRow{
				Inbox: inbox, Effect: effectTriage, Detail: "filed for a human, on purpose",
			})
		}
	}

	// Problems first, then busiest, then alphabetical — the order someone
	// scanning this actually needs.
	sort.Slice(rows, func(i, j int) bool {
		ri, rj := rows[i], rows[j]
		if (ri.Effect == effectNothing) != (rj.Effect == effectNothing) {
			return ri.Effect == effectNothing
		}
		if ri.Recent != rj.Recent {
			return ri.Recent > rj.Recent
		}
		return ri.Inbox < rj.Inbox
	})
	return rows
}

func printInboxRows(out inboxesOutput) {
	fmt.Printf("Inboxes — what a send to each one does\n\n")
	fmt.Printf("  registry: %s\n", out.Registry)
	if out.Store != "" {
		fmt.Printf("  store:    %s\n", out.Store)
	}
	fmt.Println()

	if len(out.Rows) == 0 {
		fmt.Println("  (no inboxes registered and none seen in the store)")
		return
	}

	var problems int
	for _, r := range out.Rows {
		marker := " "
		label := string(r.Effect)
		switch r.Effect {
		case effectDispatches:
			label = green("DISPATCHES")
		case effectTriage:
			label = cyan("TRIAGE    ")
		case effectNothing:
			marker, label = "!", red("NOTHING   ")
			problems++
		}
		count := ""
		switch {
		case r.Recent > 0 && r.Unread > 0:
			count = fmt.Sprintf(" [%d recent, %d unread]", r.Recent, r.Unread)
		case r.Recent > 0:
			count = fmt.Sprintf(" [%d recent]", r.Recent)
		}
		fmt.Printf(" %s %s  %-34s %s%s\n", marker, label, r.Inbox, r.Detail, count)
	}

	fmt.Println()
	if problems > 0 {
		fmt.Printf(" %s %d inbox(es) accept messages and do nothing with them.\n", red("!"), problems)
		fmt.Println("   Register an agent for them, declare them triage_only_inboxes, or fix the sender.")
	}
	fmt.Println("   Judged against the registry named above — pass --registry <cloud config> to")
	fmt.Println("   judge against the plane the coordinator actually reads.")
}

// resolveInboxRegistry loads the registry that matches the store being read.
//
// Precedence: --registry, then $AILANG_CONFIG, then — when the store is a remote
// plane — the cloud config the coordinator itself reads, then this machine's.
// The returned label names the source, because a verdict from the wrong registry
// is indistinguishable from a verdict from the right one.
func resolveInboxRegistry(flagPath string) (*coordinator.AgentRegistry, string, error) {
	if flagPath != "" {
		reg, err := coordinator.LoadAgentRegistryFrom(flagPath)
		if err != nil {
			return nil, "", fmt.Errorf("cannot load the agent registry %s: %w", flagPath, err)
		}
		return reg, flagPath, nil
	}
	if p := os.Getenv("AILANG_CONFIG"); p != "" {
		reg, err := coordinator.LoadAgentRegistryFrom(p)
		if err != nil {
			return nil, "", fmt.Errorf("cannot load $AILANG_CONFIG %s: %w", p, err)
		}
		return reg, p + " ($AILANG_CONFIG)", nil
	}

	// Reading a remote store: fetch the registry that plane actually dispatches
	// from. Best effort — a laptop without GCS credentials still gets an answer,
	// clearly labelled as the local one.
	if mode, _ := messagesTarget(); mode == storage.ModeGCP {
		if reg, label, err := loadCloudInboxRegistry(); err == nil {
			return reg, label, nil
		} else if err != nil {
			fmt.Fprintf(os.Stderr, "%s could not read the cloud registry (%v)\n", yellow("!"), err)
			fmt.Fprintf(os.Stderr, "  falling back to this machine's config — routing verdicts below may be wrong.\n\n")
		}
	}

	reg, err := coordinator.LoadAgentRegistry()
	if err != nil {
		return nil, "", fmt.Errorf("cannot load the agent registry: %w", err)
	}
	return reg, "~/.ailang/config.yaml (this machine)", nil
}

// loadCloudInboxRegistry reads the coordinator's own config out of GCS.
func loadCloudInboxRegistry() (*coordinator.AgentRegistry, string, error) {
	ctx := context.Background()
	store, err := newGCSConfigStore(ctx)
	if err != nil {
		return nil, "", err
	}
	data, _, err := store.Read()
	if err != nil {
		return nil, "", err
	}
	tmp, err := os.CreateTemp("", "ailang-cloud-config-*.yaml")
	if err != nil {
		return nil, "", err
	}
	defer func() { _ = os.Remove(tmp.Name()) }()
	if _, err := tmp.Write(data); err != nil {
		_ = tmp.Close()
		return nil, "", err
	}
	if err := tmp.Close(); err != nil {
		return nil, "", err
	}
	reg, err := coordinator.LoadAgentRegistryFrom(tmp.Name())
	if err != nil {
		return nil, "", err
	}
	bucket, object := configLocation()
	return reg, fmt.Sprintf("gs://%s/%s (the plane's own registry)", bucket, object), nil
}
