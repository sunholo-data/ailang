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

	"github.com/sunholo-data/ailang/internal/config"
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
	registryPath := fs.String("registry", "", "config to judge against: a path, or `cloud` for the plane's own registry (default: $AILANG_CONFIG if it declares agents, else the plane's, else this machine's)")
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
			row.Detail = dispatchDetail(agent)
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
			// The REPO, not just the agent id. An agent name is not a
			// destination: `design-doc-creator` sounds generic and works only on
			// sunholo-data/ailang, while `daneel-design` — the one that reads
			// repo-specific — serves sunholo-data/daneel. A sender with AILANG
			// feedback had no way to tell from this listing which of the two was
			// theirs, and the obvious-sounding name (`ailang-core`) bounces.
			row.Detail = dispatchDetail(agent)
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
	plane, _ := messagesTarget()
	return resolveInboxRegistryForPlane(flagPath, plane)
}

// resolveInboxRegistryForPlane is resolveInboxRegistry with the plane named by
// the caller rather than read off the MESSAGE store.
//
// The approvals path needs this. Its plane comes from `--remote`, not from
// $AILANG_STORAGE_MESSAGING, and the two disagree constantly: `coordinator
// approve --remote gcp` on a laptop with no messaging override resolved the
// plane as local, loaded this machine's registry, and found no cloud agent —
// so checkRegistryCanDispatch refused. See the fault it caused at that call
// site.
func resolveInboxRegistryForPlane(flagPath string, plane storage.Mode) (*coordinator.AgentRegistry, string, error) {
	// `--registry cloud`: name the plane explicitly, whatever the environment
	// says. Daneel asked for this spelling by name — a caller that KNOWS it
	// means the shared plane should not have to arrange for an env var to be
	// absent to say so.
	if strings.TrimSpace(flagPath) == "cloud" {
		reg, label, err := loadCloudRegistry()
		if err != nil {
			return nil, "", fmt.Errorf("--registry cloud: cannot read the plane's registry: %w", err)
		}
		return reg, label, nil
	}
	if flagPath != "" {
		reg, declared, err := coordinator.LoadAgentRegistryFromDeclared(flagPath)
		if err != nil {
			return nil, "", fmt.Errorf("cannot load the agent registry %s: %w", flagPath, err)
		}
		if !declared {
			// An explicit path that declares nothing is a user error, and the
			// registry handed back would be AILANG's built-in default fleet.
			return nil, "", fmt.Errorf("%s has no `coordinator:` section, so it declares no agents — reading it as a registry would answer from AILANG's built-in default (one agent, `coordinator`), not from any real deployment.\n  Point --registry at a config with a coordinator section, or use --registry cloud for the plane's own", flagPath)
		}
		return reg, flagPath, nil
	}
	// $AILANG_CONFIG names the ailang config, which is NOT necessarily a
	// registry. A file with a `pubsub:` block and no `coordinator:` section is a
	// perfectly good send config and declares no agents at all — and this branch
	// used to accept it, hand back an empty registry, and label it as the
	// authority.
	//
	// Measured by Daneel 2026-09-17 (v0.2.5 → v0.2.11): the daneel user has no
	// ~/.ailang/config.yaml, so every `messages send` sets $AILANG_CONFIG to a
	// pubsub-only file to get the notification published. The dispatch programs
	// then ran `messages inboxes --all --json` in the same process and got
	// `"registry": ".../ailang-messaging.yaml ($AILANG_CONFIG)"`. Their own guard
	// — "refuse unless the registry is the shared plane's" — correctly refused
	// every dispatch, and two of Mark's morning design requests sat deferred for
	// seven hours.
	//
	// The report called that an empty registry; it is one agent, `coordinator`,
	// from AILANG's built-in default (measured here, not assumed). Same effect,
	// and worse without Daneel's guard: `messages health` would call every other
	// inbox a config gap, and the send guard would judge routing against a fleet
	// that does not exist.
	//
	// Eight commands share this resolver (health, inboxes, prs, approvals, the
	// send guard, pipeline, lint, agents), so the empty answer was not confined
	// to one report: with such a config set, `messages health` calls every inbox
	// a config gap.
	//
	// So: a config that declares no agents is not a registry. Say it once on
	// stderr and keep resolving, exactly as if the variable were unset.
	if p := config.Raw(config.EnvConfigFile); p != "" {
		reg, declared, err := coordinator.LoadAgentRegistryFromDeclared(p)
		if err != nil {
			return nil, "", fmt.Errorf("cannot load $AILANG_CONFIG %s: %w", p, err)
		}
		if declared {
			return reg, p + " ($AILANG_CONFIG)", nil
		}
		fmt.Fprintf(os.Stderr, "%s $AILANG_CONFIG (%s) has no `coordinator:` section, so it declares no agents — resolving the plane's own registry instead of AILANG's built-in defaults.\n",
			yellow("!"), p)
	}

	// Reading a remote store: fetch the registry that plane actually dispatches
	// from. Best effort — a laptop without GCS credentials still gets an answer,
	// clearly labelled as the local one.
	if plane == storage.ModeGCP {
		if reg, label, err := loadCloudRegistry(); err == nil {
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

// dispatchDetail is the one description of what sending here does.
//
// There were two, in the two branches that build this listing, and they drifted
// the moment one gained the repository — which is the fault this whole session
// kept finding. One function, both callers.
func dispatchDetail(a *coordinator.AgentConfig) string {
	if repo := agentRepoLabel(a); repo != "" {
		return fmt.Sprintf("creates a task for %s on %s", a.ID, repo)
	}
	return fmt.Sprintf("creates a task for %s", a.ID)
}

// agentRepoLabel is the repository an agent's work lands in.
//
// ResolveRepo first, because `repo` is the explicit declaration; Workspace is
// the older field that doubles as a path on a local coordinator, so it is only
// useful here when it looks like owner/name.
func agentRepoLabel(a *coordinator.AgentConfig) string {
	if a == nil {
		return ""
	}
	if r := a.ResolveRepo(); r != "" {
		return r
	}
	if w := a.Workspace; strings.Count(w, "/") == 1 && !strings.HasPrefix(w, "/") {
		return w
	}
	return ""
}

// loadCloudInboxRegistry reads the coordinator's own config out of GCS.
// loadCloudRegistry is loadCloudInboxRegistry behind a variable so a test can
// pin WHICH registry a given plane reads. The wiring is the thing that broke:
// the approvals path resolved its registry from this machine's config for
// sixteen days and no test could see it, because every test that mattered
// would have had to reach GCS.
var loadCloudRegistry = loadCloudInboxRegistry

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
	return reg, fmt.Sprintf("gs://%s/%s (the plane's own registry)", store.bucket, store.object), nil
}
