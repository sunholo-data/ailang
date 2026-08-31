package main

// `ailang messages health` — is the message plane actually delivering?
//
// M-MESSAGE-PLANE-TRUST M3. Every seam on the send → dispatch path used to fail
// silently: the write succeeded, a green tick printed, and nothing ran. This
// command exists to make one number visible that should always be zero —
// messages that are filed, routable, and undelivered.

import (
	"flag"
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/sunholo-data/ailang/internal/coordinator"
	"github.com/sunholo-data/ailang/internal/messaging"
	"github.com/sunholo-data/ailang/internal/storage"
)

// inboxBucket classifies an unread message by what SHOULD happen to it.
type inboxBucket int

const (
	bucketRoutable   inboxBucket = iota // an agent is registered: should have been dispatched
	bucketTriage                        // declared human-triage: sitting unread is correct
	bucketUnroutable                    // no agent AND not declared: a config gap
)

func runMessagesHealth(args []string) {
	fs := flag.NewFlagSet("messages health", flag.ExitOnError)
	registryPath := fs.String("registry", "", "Agent config to judge routing against (default: ~/.ailang/config.yaml)")
	strict := fs.Bool("strict", false, "Exit non-zero when the verdict is not HEALTHY (for CI / the morning report)")
	if err := fs.Parse(args); err != nil {
		return
	}

	mode, project := messagesTarget()
	fmt.Println()
	fmt.Println(bold("Message plane health"))
	fmt.Println()

	if desc := describeMessageStore(); desc != "" {
		fmt.Printf("  %s\n", desc) // describeMessageStore renders its own "store: " prefix
	} else {
		fmt.Printf("  store:    local SQLite (%s)\n", messaging.GetDefaultDatabasePath())
	}

	// Which registry are we judging against? This is load-bearing and easy to
	// get wrong: the CLOUD coordinator reads its agents from a GCS bucket, not
	// from this machine. Measured 2026-08-31 — the local config carried 41
	// agents while prod carried 34. A routing verdict computed from the wrong
	// registry is worse than none, so name the source every time.
	var registry *coordinator.AgentRegistry
	var err error
	if *registryPath != "" {
		registry, err = coordinator.LoadAgentRegistryFrom(*registryPath)
	} else {
		registry, err = coordinator.LoadAgentRegistry()
	}
	if err != nil {
		fmt.Fprintf(os.Stderr, "\n%s cannot load the agent registry: %v\n", red("Error"), err)
		fmt.Fprintf(os.Stderr, "  Routing cannot be judged without it. Refusing to guess.\n")
		os.Exit(1)
	}
	src := *registryPath
	if src == "" {
		src = messaging.GetConfigPath()
	}
	fmt.Printf("  registry: %s (%d agents)\n", src, len(registry.ListAgents()))
	// Only warn when the registry was IMPLIED. An operator who named one with
	// --registry has already made the choice this warning exists to prompt, and
	// repeating it there would be false: the path shown may well BE the cloud
	// coordinator's config, pulled from the bucket.
	if mode == storage.ModeGCP && *registryPath == "" {
		fmt.Printf("            %s this is THIS machine's config, not the cloud coordinator's\n", yellow("!"))
		fmt.Printf("            (that reads gs://%s-ailang-config/config.yaml — they can diverge;\n", project)
		fmt.Printf("             pass --registry <path> to judge against a copy of that one)\n")
	}

	store, err := openStore()
	if err != nil {
		fmt.Fprintf(os.Stderr, "\n%s %v\n", red("Error"), err)
		os.Exit(1)
	}
	defer func() { _ = store.Close() }()

	msgs, err := store.ListInboxMessages(messaging.InboxListOptions{UnreadOnly: true, Collapsed: true})
	if err != nil {
		reportListFailure(err)
		os.Exit(1)
	}

	counts := map[inboxBucket]int{}
	byInbox := map[inboxBucket]map[string]int{
		bucketRoutable: {}, bucketTriage: {}, bucketUnroutable: {},
	}
	for _, m := range msgs {
		b := classifyInbox(registry, m.ToInbox)
		counts[b]++
		byInbox[b][m.ToInbox]++
	}

	fmt.Println()
	fmt.Printf("  Unread total:                 %d\n", len(msgs))
	fmt.Printf("  ├─ routable (agent exists):   %s\n", emphasizeIfNonZero(counts[bucketRoutable]))
	fmt.Printf("  ├─ human-triage (by design):  %d\n", counts[bucketTriage])
	fmt.Printf("  └─ no agent, not declared:    %s\n", emphasizeIfNonZero(counts[bucketUnroutable]))

	printInboxBreakdown("routable but undelivered", byInbox[bucketRoutable])
	printInboxBreakdown("config gap (no agent, not declared triage)", byInbox[bucketUnroutable])

	// Send path: can THIS machine announce a message at all?
	fmt.Println()
	notifyOK := reportSendPath(mode, project)

	fmt.Println()
	healthy := counts[bucketRoutable] == 0 && counts[bucketUnroutable] == 0 && notifyOK
	switch {
	case healthy:
		fmt.Printf("  %s messages that should have been dispatched: 0\n", green("HEALTHY"))
	case counts[bucketRoutable] > 0:
		fmt.Printf("  %s %d message(s) filed, routable, and never dispatched.\n",
			red("DEGRADED"), counts[bucketRoutable])
		fmt.Printf("           In a healthy plane this is always 0 — push delivers everything.\n")
	default:
		fmt.Printf("  %s see the flagged rows above.\n", yellow("DEGRADED"))
	}
	fmt.Println()

	if *strict && !healthy {
		os.Exit(1)
	}
}

// classifyInbox decides what should have happened to a message.
func classifyInbox(registry *coordinator.AgentRegistry, inbox string) inboxBucket {
	if registry.GetAgentForInbox(inbox) != nil {
		return bucketRoutable
	}
	if registry.IsTriageOnly(inbox) {
		return bucketTriage
	}
	return bucketUnroutable
}

// reportSendPath reports whether a send from this machine would be announced.
// Returns false when a cloud store would receive a message nothing is told about.
func reportSendPath(mode storage.Mode, project string) bool {
	cfg, err := messaging.LoadConfig()
	if err != nil {
		fmt.Printf("  send path: %s messaging config unreadable: %v\n", red("UNKNOWN"), err)
		return false
	}
	enabled := cfg != nil && cfg.PubSub != nil && cfg.PubSub.Enabled
	if mode != storage.ModeGCP {
		fmt.Printf("  send path: local store — the daemon polls it directly, no notification needed\n")
		return true
	}
	if !enabled {
		fmt.Printf("  send path: %s pubsub disabled — a send from here would be FILED, NOT DISPATCHED\n", red("BROKEN"))
		fmt.Printf("             add a pubsub block to %s\n", messaging.GetConfigPath())
		return false
	}
	p := cfg.PubSub.ProjectID
	if p == "" {
		p = project
	}
	// A notification published to a DIFFERENT project than the one the message
	// was written to reaches a coordinator that will never see the message. The
	// write succeeds, the publish succeeds, both report ok — and the work is
	// invisible to the only process that could do it.
	//
	// Measured 2026-08-31: a probe written to ailang-multivac-dev published its
	// notification to ailang-multivac because the pubsub block pins project_id.
	// Both facts were already on this screen, one line apart, and reading them
	// as a pair is exactly what a health check is for.
	if p != project {
		fmt.Printf("  send path: %s pubsub publishes to %q but this store is %q\n", red("SPLIT"), p, project)
		fmt.Printf("             A notification sent to the wrong project reaches a coordinator that\n")
		fmt.Printf("             cannot see the message. Set pubsub.project_id to %q in %s,\n", project, messaging.GetConfigPath())
		fmt.Printf("             or unset it so it follows the store.\n")
		return false
	}
	fmt.Printf("  send path: %s pubsub enabled (project %s)\n", green("ok"), p)
	return true
}

// printInboxBreakdown lists the offending inboxes, largest first, so the output
// names what to act on rather than only how much is wrong.
func printInboxBreakdown(label string, m map[string]int) {
	if len(m) == 0 {
		return
	}
	type row struct {
		inbox string
		n     int
	}
	rows := make([]row, 0, len(m))
	for k, v := range m {
		rows = append(rows, row{k, v})
	}
	sort.Slice(rows, func(i, j int) bool {
		if rows[i].n != rows[j].n {
			return rows[i].n > rows[j].n
		}
		return rows[i].inbox < rows[j].inbox
	})
	fmt.Printf("\n  %s:\n", label)
	for _, r := range rows {
		fmt.Printf("    %4d  %s\n", r.n, r.inbox)
	}
}

func emphasizeIfNonZero(n int) string {
	if n == 0 {
		return fmt.Sprintf("%d", n)
	}
	return red(fmt.Sprintf("%d  ← should be 0", n))
}

// reportListFailure turns Firestore's missing-index error into an actionable
// finding instead of an opaque gRPC dump.
//
// A missing composite index is infrastructure that was never declared, and it
// reads as a query failure at every call site. Two were found this way on
// 2026-08-31 (obs_chain_stages, inbox_messages) — and the more dangerous of the
// pair was the one a LIST path swallowed into "0 stages", which is why this
// reports rather than degrades: a health command that silently answers with a
// partial query is worse than one that refuses.
func reportListFailure(err error) {
	msg := err.Error()
	if !strings.Contains(msg, "requires an index") && !strings.Contains(msg, "FailedPrecondition") {
		fmt.Fprintf(os.Stderr, "\n%s listing unread: %v\n", red("Error"), err)
		return
	}
	fmt.Fprintf(os.Stderr, "\n%s the unread query needs a Firestore composite index that does not exist.\n", red("BLOCKED"))
	fmt.Fprintf(os.Stderr, "  This is undeclared infrastructure, not a code fault — the same class of gap\n")
	fmt.Fprintf(os.Stderr, "  that made every cloud chain read \"0 stages\".\n\n")
	fmt.Fprintf(os.Stderr, "  Create it with:\n")
	fmt.Fprintf(os.Stderr, "    gcloud firestore indexes composite create \\\n")
	fmt.Fprintf(os.Stderr, "      --collection-group=inbox_messages \\\n")
	fmt.Fprintf(os.Stderr, "      --field-config=field-path=dup_of,order=ascending \\\n")
	fmt.Fprintf(os.Stderr, "      --field-config=field-path=status,order=ascending \\\n")
	fmt.Fprintf(os.Stderr, "      --field-config=field-path=created_at,order=descending \\\n")
	fmt.Fprintf(os.Stderr, "      --project=<project>\n\n")
	fmt.Fprintf(os.Stderr, "  Underlying error: %v\n", err)
}
