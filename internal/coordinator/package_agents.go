package coordinator

import (
	"fmt"
	"log"
	"sort"
	"strings"
	"time"

	"github.com/sunholo-data/ailang/internal/executor"
	"github.com/sunholo-data/ailang/internal/messaging"
	"github.com/sunholo-data/ailang/internal/pkg"
)

// M-PKG-QUALITY-LADDER M6 — every published package gets an agent inbox.
//
// The plane config declares ONE `package_agent_template` section (not an
// agent: it serves no inbox, so a `pkg:` inbox for a package that is not in
// the registry stays a visible typo). At load and on refresh, the registry
// index is read and, for every package with no exact `pkg:<name>` agent of
// its own, a per-package agent is derived from the template: id
// pkg-<vendor>-<name>, inbox pkg:<name>, workspace / merge_branch /
// subdirectory / artifact_patterns from metadata.repository. Hand-written
// entries always win (Register refuses a duplicate inbox), so nothing already
// configured moves.
//
// Measured 2026-09-17: 53 packages, 29 hand-written pkg agents + one
// motoko_ext_* family pattern, 12 packages with no inbox at all — messages to
// them were accepted and never dispatched.

// packageAgentRefresh bounds how often a daemon re-reads the index.
const packageAgentRefresh = 10 * time.Minute

// SetPackageAgentTemplate installs the `package_agent_template` section.
// nil disables derivation (the feature is opt-in by declaring the section).
func (r *AgentRegistry) SetPackageAgentTemplate(t *AgentConfig) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.packageTemplate = t
}

// PackageAgentTemplate returns the installed template, or nil.
func (r *AgentRegistry) PackageAgentTemplate() *AgentConfig {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.packageTemplate
}

// DerivePackageAgent builds the per-package agent from the template and the
// package's index entry. ok=false when metadata.repository is not a GitHub
// tree URL: the repo is NOT a function of the package name (sunholo/email
// lives in email-parse, sunholo/ailang_parse is a repo root, sunholo/duckdb is
// packages/duckdb in the monorepo), so a guessed workspace clones the right
// repo and finds nothing — the failure that sent every pkg-sunholo-ailang-parse
// dispatch nowhere. Such a package stays visibly unserved in `messages
// inboxes` (with the reason) and its publisher sees PUB021 until
// [metadata] repository is set.
func DerivePackageAgent(template *AgentConfig, entry pkg.IndexEntry) (*AgentConfig, bool) {
	if template == nil || entry.Name == "" {
		return nil, false
	}
	ref, ok := pkg.ParseRepositoryURL(entry.Repository)
	if !ok {
		return nil, false
	}
	derived := *template // shallow copy; slices below are re-allocated
	derived.ID = pkg.PackageAgentID(entry.Name)
	derived.Inbox = messaging.FormatPackageInbox(entry.Name) // registry spelling: underscores, never the repo dir's hyphens
	derived.Capabilities = append([]string(nil), template.Capabilities...)
	derived.Label = "Package: " + entry.Name + " (derived from registry)"
	derived.Workspace = ref.Workspace
	if ref.Branch != "" {
		derived.MergeBranch = ref.Branch
	}
	derived.Subdirectory = ref.Subdirectory
	// artifact_patterns are what the merge guard reads; unset means `**/*`,
	// which bounds nothing. Monorepo package → its subdirectory; the repo IS
	// the package → `**/*` is the honest bound, stated rather than defaulted.
	if ref.Subdirectory != "" {
		derived.ArtifactPatterns = []string{ref.Subdirectory + "/**/*"}
	} else {
		derived.ArtifactPatterns = []string{"**/*"}
	}
	return &derived, true
}

// templateUsable refuses a template that would derive agents dispatch cannot
// run: a `pkg:` inbox defaults to the ailang_only lane, and that lane executes
// nothing without a policy_path (dispatch fails closed since 2026-09-17). A
// generator that forgets policy_path produces an agent that can read and
// write files and run nothing — measured by the message-plane session; see
// design_docs/planned/HANDOVER-package-agent-autoprovision.md.
func templateUsable(t *AgentConfig) error {
	probe := *t
	probe.Inbox = messaging.FormatPackageInbox("probe/probe")
	if lane := probe.GetEffectiveToolPolicy(); lane != executor.ToolProfileFull && strings.TrimSpace(t.PolicyPath) == "" {
		return fmt.Errorf("package_agent_template runs the %s lane but declares no policy_path — derived agents could execute nothing; add policy_path (e.g. /etc/ailang-config/policies/pkg-ailang-only.toml)", lane)
	}
	return nil
}

// MaterializePackageAgents registers a derived agent for every index package
// that lacks an exact one. Idempotent: a second call with the same index adds
// nothing. Returns the ids it added, sorted.
func (r *AgentRegistry) MaterializePackageAgents(index *pkg.RegistryIndex) []string {
	template := r.PackageAgentTemplate()
	if template == nil || index == nil {
		return nil
	}
	if err := templateUsable(template); err != nil {
		log.Printf("package agents: %v — deriving nothing", err)
		return nil
	}
	var added []string
	for _, entry := range index.Packages {
		inbox := messaging.FormatPackageInbox(entry.Name)
		if r.hasExactInbox(inbox) {
			continue
		}
		agent, ok := DerivePackageAgent(template, entry)
		if !ok {
			continue
		}
		if r.HasAgent(agent.ID) {
			continue
		}
		if err := r.Register(agent); err != nil {
			continue // a hand-written entry raced us; it wins
		}
		added = append(added, agent.ID)
	}
	sort.Strings(added)
	return added
}

// hasExactInbox reports a literal (non-pattern) registration for inbox.
func (r *AgentRegistry) hasExactInbox(inbox string) bool {
	r.mu.RLock()
	defer r.mu.RUnlock()
	_, ok := r.byInbox[inbox]
	return ok
}

// MaterializePackageAgentsFromRegistry fetches the live index and materializes.
// Best-effort by design: an unreachable registry means no derivation this
// round — a package inbox stays unserved (visible in `messages inboxes`)
// rather than dispatching on a guess — and it is logged; the daemon retries
// on its next refresh.
func (r *AgentRegistry) MaterializePackageAgentsFromRegistry(logf func(string, ...interface{})) []string {
	if r.PackageAgentTemplate() == nil {
		return nil
	}
	if logf == nil {
		logf = log.Printf
	}
	index, err := pkg.NewRegistryClient().FetchIndex()
	if err != nil {
		logf("package agents: registry index unavailable (%v) — no package agents derived this round", err)
		return nil
	}
	added := r.MaterializePackageAgents(index)
	if len(added) > 0 {
		logf("package agents: derived %d inbox agent(s) from the registry index: %s", len(added), strings.Join(added, ", "))
	}
	return added
}

// refreshPackageAgents is the daemon's periodic re-read (new packages appear
// without a config roll). Cheap: one public GCS GET every packageAgentRefresh.
func (d *Daemon) refreshPackageAgents() {
	if d.agentRegistry == nil || d.agentRegistry.PackageAgentTemplate() == nil {
		return
	}
	if !d.lastPackageAgentRefresh.IsZero() && time.Since(d.lastPackageAgentRefresh) < packageAgentRefresh {
		return
	}
	d.lastPackageAgentRefresh = time.Now()
	d.agentRegistry.MaterializePackageAgentsFromRegistry(func(f string, a ...interface{}) {
		d.logger.Printf(f, a...)
	})
}

// PackageInboxStatus explains, for `ailang messages inboxes`, why a `pkg:`
// inbox does or does not dispatch when only the template serves it.
func PackageInboxStatus(entry pkg.IndexEntry) string {
	if _, ok := pkg.ParseRepositoryURL(entry.Repository); ok {
		return fmt.Sprintf("derived from registry (%s)", entry.Repository)
	}
	if entry.Repository == "" {
		return "no [metadata] repository in ailang.toml — no agent can be derived (the repo is not a function of the name); republish with a GitHub tree URL"
	}
	return fmt.Sprintf("[metadata] repository %q is not a GitHub tree URL — no agent can be derived; republish with https://github.com/<owner>/<repo>/tree/<branch>[/<subdir>]", entry.Repository)
}
