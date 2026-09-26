package main

// `ailang messages health` — is the message plane actually delivering?
//
// M-MESSAGE-PLANE-TRUST M3. Every seam on the send → dispatch path used to fail
// silently: the write succeeded, a green tick printed, and nothing ran. This
// command exists to make one number visible that should always be zero —
// messages that are filed, routable, and undelivered.

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"strings"
	"time"

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
	bucketResult                        // an agent's own OUTPUT, filed in its own inbox
)

// resultMessageTypes are the types an agent EMITS. They land in the agent's own
// inbox as the durable record of what it did, and nothing is supposed to pick
// them up — a completion is not a request for more work.
//
// Counting them as "routable but never dispatched" is why this command read
// DEGRADED 87 on a plane whose real undispatched backlog was 4 (measured
// 2026-09-14: 61 of 65 were completion/approval_request rows the agents had
// just written themselves). A health number that is never zero in a healthy
// plane teaches the reader to ignore it, which is worse than not printing it.
//
// inbox_unrouted_notice is here too: it is the bounce the coordinator files
// when a send had nowhere to go. It reports a fault, it is not one.
var resultMessageTypes = map[string]bool{
	messaging.InboxTypeCompletion:      true,
	messaging.InboxTypeApprovalRequest: true,
	messaging.InboxTypeResponse:        true,
	bounceMessageType:                  true,
}

// bounceMessageType mirrors coordinator.bounceMessageType, which is unexported.
const bounceMessageType = "inbox_unrouted_notice"

func isResultMessageType(t string) bool { return resultMessageTypes[t] }

func runMessagesHealth(args []string) {
	fs := flag.NewFlagSet("messages health", flag.ExitOnError)
	registryPath := fs.String("registry", "", "Agent config to judge routing against: a path, or `cloud` for the plane's own registry (default: $AILANG_CONFIG if it declares agents, else the plane's, else ~/.ailang/config.yaml)")
	strict := fs.Bool("strict", false, "Exit non-zero when the verdict is not HEALTHY (for CI / the morning report)")
	since := fs.String("since", "24h", "Window the verdict judges: 24h, 7d, 90m, or 0 for all time")
	asJSON := fs.Bool("json", false, "Emit the summary as JSON (for hooks and agents)")
	if err := fs.Parse(args); err != nil {
		return
	}
	window, werr := parseHealthWindow(*since)
	if werr != nil {
		fmt.Fprintf(os.Stderr, "%s %v\n", red("Error"), werr)
		os.Exit(2)
	}

	mode, project := messagesTarget()
	// --json is the machine path: nothing but the document goes to stdout, so a
	// hook can read it without stripping a banner.
	out := os.Stdout
	if *asJSON {
		out = os.Stderr
	}
	fmt.Fprintln(out)
	fmt.Fprintln(out, bold("Message plane health")+dim("   window: "+humanWindow(window)))
	fmt.Fprintln(out)

	if desc := describeMessageStore(); desc != "" {
		// Name the PLANE, not just the project. `ailang-multivac` and
		// `ailang-multivac-dev` differ by a suffix and are entirely separate
		// planes; reading one and concluding about the other is easy and
		// expensive (M-COORDINATOR-EXECUTION-TRUST M4).
		if plane := planeLabel(project); plane != "" {
			fmt.Fprintf(out, "  %s  %s\n", desc, bold("["+strings.ToUpper(plane)+" PLANE]"))
		} else {
			fmt.Fprintf(out, "  %s\n", desc) // describeMessageStore renders its own "store: " prefix
		}
	} else {
		fmt.Fprintf(out, "  store:    local SQLite (%s)\n", messaging.GetDefaultDatabasePath())
	}

	// Which registry are we judging against? This is load-bearing and easy to
	// get wrong: the CLOUD coordinator reads its agents from a GCS bucket, not
	// from this machine. Measured 2026-08-31 — the local config carried 41
	// agents while prod carried 34. A routing verdict computed from the wrong
	// registry is worse than none, so name the source every time.
	//
	// resolveInboxRegistry is the SAME resolver `messages inboxes` and
	// `coordinator agents` use: when the store is a remote plane it fetches that
	// plane's own config out of GCS, and only falls back to this machine's on a
	// credential failure (loudly, on stderr).
	//
	// This command used to load ~/.ailang/config.yaml unconditionally and print a
	// warning above the verdict. On a laptop whose local config holds 2 agents it
	// reported a 177-message "config gap" against a plane whose real gap was 54 —
	// a wrong number with a caveat under it, which readers take as a number.
	// Sharing the resolver means the three commands cannot disagree about what
	// the plane's agents are.
	registry, src, err := resolveInboxRegistry(*registryPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "\n%s cannot load the agent registry: %v\n", red("Error"), err)
		fmt.Fprintf(os.Stderr, "  Routing cannot be judged without it. Refusing to guess.\n")
		os.Exit(1)
	}
	fmt.Fprintf(out, "  registry: %s (%d agents)\n", src, len(registry.ListAgents()))

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

	now := time.Now()
	hm := make([]healthMsg, 0, len(msgs))
	for _, m := range msgs {
		hm = append(hm, healthMsg{
			Inbox:  m.ToInbox,
			Type:   m.MessageType,
			Age:    now.Sub(m.CreatedAt),
			Bucket: classifyInbox(registry, m.ToInbox, m.MessageType),
		})
	}
	summary := summarizeHealth(hm, window)
	arrivals := countArrivals(store, registry, window, now)

	// Send path: can THIS machine announce a message at all?
	sendOK, sendLine := describeSendPath(mode, project)
	healthy := summary.LiveFaults() == 0 && sendOK

	if *asJSON {
		if err := printHealthJSON(summary, arrivals, healthy, sendLine, src); err != nil {
			fmt.Fprintf(os.Stderr, "%s %v\n", red("Error"), err)
			os.Exit(1)
		}
		if *strict && !healthy {
			os.Exit(1)
		}
		return
	}

	renderHealth(out, summary, arrivals, window, sendLine, healthy)

	if *strict && !healthy {
		os.Exit(1)
	}
}

// renderHealth prints the summary: the window first, because "is it working
// NOW" is the question, and the standing backlog second, because it is context.
func renderHealth(out *os.File, s *healthSummary, a arrivalStats, window time.Duration, sendLine string, healthy bool) {
	fmt.Fprintln(out)
	if window > 0 {
		fmt.Fprintf(out, "  %s\n", bold("In the "+humanWindow(window)+":"))
		switch {
		case a.Err != nil:
			fmt.Fprintf(out, "    arrived               %s  %s\n", yellow("unknown"), dim(briefError(a.Err.Error())))
		default:
			fmt.Fprintf(out, "    arrived               %d   ·  work %d  ·  agent output %d\n", a.Total, a.Work, a.Output)
		}
		fmt.Fprintf(out, "    undelivered           %s  %s\n",
			emphasizeIfNonZero(s.Routable.New), dim("work that arrived and is still not picked up"))
		fmt.Fprintf(out, "    unroutable            %s  %s\n",
			emphasizeIfNonZero(s.Unroutable.New), dim("sent to an inbox no agent serves"))
		if a.Err == nil {
			fmt.Fprintf(out, "    bounces               %s  %s\n",
				emphasizeIfNonZero(a.Bounced), dim("UNDELIVERED notices the coordinator filed"))
		}
		fmt.Fprintln(out)
	}

	fmt.Fprintf(out, "  %s %d\n", bold("Unread, all time:"), s.Unread)
	fmt.Fprintf(out, "  ├─ routable (agent exists):   %s\n", bucketLine(s.Routable, window))
	fmt.Fprintf(out, "  ├─ agent output (not work):   %d\n", s.Result.Total)
	fmt.Fprintf(out, "  ├─ human-triage (by design):  %d\n", s.Triage.Total)
	fmt.Fprintf(out, "  └─ no agent, not declared:    %s\n", bucketLine(s.Unroutable, window))

	printHealthRows(out, "routable but undelivered", s.Routable, window)
	printHealthRows(out, "config gap (no agent, not declared triage)", s.Unroutable, window)

	fmt.Fprintln(out)
	fmt.Fprintf(out, "  %s\n", sendLine)
	fmt.Fprintln(out)

	switch {
	case healthy && window == 0:
		fmt.Fprintf(out, "  %s no routable message on this plane is undispatched, at any age.\n", green("HEALTHY"))
	case healthy:
		fmt.Fprintf(out, "  %s nothing that arrived in the %s is undispatched.\n",
			green("HEALTHY"), humanWindow(window))
	case s.Routable.New > 0 && window == 0:
		fmt.Fprintf(out, "  %s %d message(s) are routable and were never dispatched, at any age.\n",
			red("DEGRADED"), s.Routable.New)
		fmt.Fprintf(out, "           In a healthy plane this is 0 — push delivers everything.\n")
	case s.Routable.New > 0:
		fmt.Fprintf(out, "  %s %d message(s) arrived in the %s, are routable, and were never dispatched.\n",
			red("DEGRADED"), s.Routable.New, humanWindow(window))
		fmt.Fprintf(out, "           In a healthy plane this is 0 — push delivers everything.\n")
	default:
		fmt.Fprintf(out, "  %s see the flagged rows above.\n", yellow("DEGRADED"))
	}

	// The backlog is real and is NOT the verdict. Saying so, with the commands
	// that act on it, is what stops a nine-day-old handoff reading as today's
	// outage — and what stops the banner being ignored when it finally is one.
	if b := s.Backlog(); b > 0 && window > 0 {
		fmt.Fprintf(out, "  %s %d older item(s) predate this window — standing backlog, not a live fault:\n",
			yellow("BACKLOG"), b)
		fmt.Fprintf(out, "      ailang messages health --since 0        judge the whole plane\n")
		fmt.Fprintf(out, "      ailang messages list --unread --json    the ids and bodies\n")
		fmt.Fprintf(out, "      ailang messages forward <id> <inbox>    re-route and dispatch one\n")
	}

	// This command sees the message plane and NOTHING downstream of it. A plane
	// can be perfectly healthy while every task it dispatched sits unapproved or
	// every PR is blocked — which is exactly the state on 2026-09-16, and it took
	// six commands to find. Name the other three so the next reader runs them.
	fmt.Fprintf(out, "\n  %s the coordinator half is not in this view:\n", dim("next:"))
	fmt.Fprintf(out, "      %s   decisions waiting on you\n", dim("ailang coordinator approvals --remote gcp"))
	fmt.Fprintf(out, "      %s        PRs the decisions left behind\n", dim("ailang coordinator prs --remote gcp"))
	fmt.Fprintf(out, "      %s   did the four-stage chain advance\n", dim("ailang coordinator pipeline --remote gcp"))
	fmt.Fprintln(out)
}

// bucketLine renders a bucket total with its new/backlog split.
func bucketLine(st *bucketStat, window time.Duration) string {
	if window == 0 || st.Total == 0 {
		return emphasizeIfNonZero(st.Total)
	}
	head := fmt.Sprintf("%d", st.Total)
	if st.New > 0 {
		head = red(fmt.Sprintf("%d", st.Total))
	}
	oldest := time.Duration(0)
	for _, r := range st.Rows {
		if r.Oldest > oldest {
			oldest = r.Oldest
		}
	}
	return fmt.Sprintf("%-4s %s", head,
		dim(fmt.Sprintf("new %d · backlog %d (oldest %s)", st.New, st.Backlog, humanAge(oldest))))
}

// printHealthRows names what to act on, newest-first, with each row's age.
func printHealthRows(out *os.File, label string, st *bucketStat, window time.Duration) {
	if st.Total == 0 {
		return
	}
	fmt.Fprintf(out, "\n  %s:\n", label)
	for _, r := range st.Rows {
		age := dim(fmt.Sprintf("oldest %s", humanAge(r.Oldest)))
		switch {
		case window == 0:
			// No window: nothing is "backlog" relative to anything.
			fmt.Fprintf(out, "    %4d  %-28s %s\n", r.Total, r.Inbox, age)
		case r.New > 0:
			fmt.Fprintf(out, "    %4d  %-28s %-8s %s\n", r.Total, r.Inbox, red(fmt.Sprintf("%d new", r.New)), age)
		default:
			fmt.Fprintf(out, "    %4d  %-28s %-8s %s\n", r.Total, r.Inbox, dim("backlog"), age)
		}
	}
}

// classifyInbox decides what should have happened to a message.
//
// The message TYPE is load-bearing, not decoration. An agent's inbox holds both
// the work sent to it and the completions it wrote itself, and only the first
// kind can be "undelivered". Judging by inbox alone counts an agent's own output
// as a failure to dispatch it.
func classifyInbox(registry *coordinator.AgentRegistry, inbox, msgType string) inboxBucket {
	if registry.GetAgentForInbox(inbox) != nil {
		if isResultMessageType(msgType) {
			return bucketResult
		}
		return bucketRoutable
	}
	if registry.IsTriageOnly(inbox) {
		return bucketTriage
	}
	// Deliberately NOT short-circuited on the result type: a bounce filed to an
	// inbox no agent serves is a bounce that itself went nowhere, and that is a
	// config gap worth seeing (measured 2026-09-13 — five UNDELIVERED notices
	// addressed to "ailang", which is also unregistered).
	return bucketUnroutable
}

// describeSendPath reports whether a send from this machine would be announced,
// and the line that says so. Returning the line rather than printing it is what
// lets --json carry the same finding as the text view.
func describeSendPath(mode storage.Mode, project string) (bool, string) {
	cfg, err := messaging.LoadConfig()
	if err != nil {
		return false, fmt.Sprintf("send path: %s messaging config unreadable: %v", red("UNKNOWN"), err)
	}
	enabled := cfg != nil && cfg.PubSub != nil && cfg.PubSub.Enabled
	if mode != storage.ModeGCP {
		return true, "send path: local store — the daemon polls it directly, no notification needed"
	}
	if !enabled {
		return false, fmt.Sprintf("send path: %s pubsub disabled — a send from here would be FILED, NOT DISPATCHED\n             add a pubsub block to %s",
			red("BROKEN"), messaging.GetConfigPath())
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
		return false, fmt.Sprintf("send path: %s pubsub publishes to %q but this store is %q\n"+
			"             A notification sent to the wrong project reaches a coordinator that\n"+
			"             cannot see the message. Set pubsub.project_id to %q in %s, or unset it.",
			red("SPLIT"), p, project, project, messaging.GetConfigPath())
	}
	return true, fmt.Sprintf("send path: %s pubsub enabled (project %s)", green("ok"), p)
}

// arrivalStats is what actually came in during the window — the throughput half
// of the question. A plane with zero undelivered messages and zero arrivals is
// not healthy, it is idle, and the old output could not tell those apart.
type arrivalStats struct {
	Total   int   `json:"total"`
	Work    int   `json:"work"`
	Output  int   `json:"agent_output"`
	Bounced int   `json:"bounced"`
	Err     error `json:"-"`
}

// countArrivals counts everything created inside the window, read or unread.
//
// StartDate has DAY granularity in the store, so it over-fetches and the exact
// cut is applied here. Over-fetching is the safe direction: the alternative is a
// window that silently drops the first hours of the day.
//
// Collapsed is deliberately NOT set, and the duplicates are dropped in memory
// instead. Asking Firestore for dup_of AND created_at together needs a
// composite index that does not exist, and the whole throughput line then reads
// "unknown" — a health command that requires new infrastructure to answer its
// own question is not one. Same rows either way.
func countArrivals(store messaging.MessageStore, registry *coordinator.AgentRegistry, window time.Duration, now time.Time) arrivalStats {
	var a arrivalStats
	if window == 0 {
		return a // all-time: the unread buckets already say everything
	}
	cut := now.Add(-window)
	msgs, err := store.ListInboxMessages(messaging.InboxListOptions{
		IncludeRead: true,
		StartDate:   cut.Format("2006-01-02"),
	})
	if err != nil {
		a.Err = err
		return a
	}
	for _, m := range msgs {
		if m.CreatedAt.Before(cut) || m.DupOf != "" {
			continue
		}
		a.Total++
		if m.MessageType == bounceMessageType {
			a.Bounced++
		}
		if isResultMessageType(m.MessageType) {
			a.Output++
		} else {
			a.Work++
		}
	}
	return a
}

// healthDoc is the --json shape. Flat on purpose: a hook reads
// `.live_faults` and `.healthy` and needs nothing else.
type healthDoc struct {
	Healthy     bool         `json:"healthy"`
	WindowHours float64      `json:"window_hours"`
	LiveFaults  int          `json:"live_faults"`
	Backlog     int          `json:"backlog"`
	Registry    string       `json:"registry"`
	SendPath    string       `json:"send_path"`
	Arrived     arrivalStats `json:"arrived"`
	Unread      int          `json:"unread_total"`
	Routable    *bucketStat  `json:"routable"`
	Unroutable  *bucketStat  `json:"unroutable"`
	AgentOutput *bucketStat  `json:"agent_output"`
	HumanTriage *bucketStat  `json:"human_triage"`
}

func printHealthJSON(s *healthSummary, a arrivalStats, healthy bool, sendLine, registrySrc string) error {
	doc := healthDoc{
		Healthy: healthy, WindowHours: s.WindowHours,
		LiveFaults: s.LiveFaults(), Backlog: s.Backlog(),
		Registry: registrySrc, SendPath: stripANSI(sendLine),
		Arrived: a, Unread: s.Unread,
		Routable: s.Routable, Unroutable: s.Unroutable,
		AgentOutput: s.Result, HumanTriage: s.Triage,
	}
	b, err := json.MarshalIndent(doc, "", "  ")
	if err != nil {
		return err
	}
	fmt.Println(string(b))
	return nil
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
