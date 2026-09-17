package coordinator

import (
	"fmt"
	"log"
	"sort"
	"strings"
	"time"

	"github.com/sunholo-data/ailang/internal/messaging"
	"github.com/sunholo-data/ailang/internal/pkg"
)

// M-PKG-QUALITY-LADDER M6 — every published package gets an agent inbox.
//
// The plane config declares ONE template agent whose inbox is exactly
// `pkg:*` (the catch-all pattern the registry already supports). At load and
// on refresh, the registry index is read and, for every package that has a
// parseable metadata.repository and no exact `pkg:<name>` agent of its own, a
// per-package agent is derived from the template: id pkg-<vendor>-<name>,
// inbox pkg:<name>, workspace / merge_branch / subdirectory /
// artifact_patterns from the repository URL. Hand-written entries always win
// (Register refuses a duplicate inbox), so nothing already configured moves.
//
// Measured 2026-09-17: 53 packages, 29 hand-written pkg agents + one
// motoko_ext_* family pattern, 12 packages with no inbox at all — messages to
// them were accepted and never dispatched.

// PackageAgentTemplateInbox is the inbox pattern that marks the template.
const PackageAgentTemplateInbox = "pkg:*"

// packageAgentRefresh bounds how often a daemon re-reads the index.
const packageAgentRefresh = 10 * time.Minute

// PackageAgentTemplate returns the `pkg:*` template agent, or nil when the
// config declares none (then no derivation happens — the feature is opt-in
// by declaring the template).
func (r *AgentRegistry) PackageAgentTemplate() *AgentConfig {
	r.mu.RLock()
	defer r.mu.RUnlock()
	for _, w := range r.wildcards {
		if w.prefix == strings.TrimSuffix(PackageAgentTemplateInbox, "*") {
			return w.agent
		}
	}
	return nil
}

// DerivePackageAgent builds the per-package agent from the template and the
// package's index entry. ok=false when the entry has no usable repository.
func DerivePackageAgent(template *AgentConfig, entry pkg.IndexEntry) (*AgentConfig, bool) {
	ref, ok := pkg.ParseRepositoryURL(entry.Repository)
	if !ok || template == nil {
		return nil, false
	}
	derived := *template // shallow copy; slices below are re-allocated
	derived.ID = pkg.PackageAgentID(entry.Name)
	derived.Label = "Package: " + entry.Name + " (derived from registry)"
	derived.Inbox = messaging.FormatPackageInbox(entry.Name)
	derived.Workspace = ref.Workspace
	if ref.Branch != "" {
		derived.MergeBranch = ref.Branch
	}
	derived.Subdirectory = ref.Subdirectory
	if ref.Subdirectory != "" {
		derived.ArtifactPatterns = []string{ref.Subdirectory + "/**/*"}
	} else {
		derived.ArtifactPatterns = append([]string(nil), template.ArtifactPatterns...)
	}
	derived.Capabilities = append([]string(nil), template.Capabilities...)
	return &derived, true
}

// MaterializePackageAgents registers a derived agent for every index package
// that lacks an exact one. Idempotent: a second call with the same index adds
// nothing. Returns the ids it added, sorted.
func (r *AgentRegistry) MaterializePackageAgents(index *pkg.RegistryIndex) []string {
	template := r.PackageAgentTemplate()
	if template == nil || index == nil {
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
// Best-effort by design: an unreachable registry leaves the `pkg:*` template
// serving new packages with ITS workspace (the packages monorepo), which is
// the same guess a hand-written entry would have made — and it is logged.
func (r *AgentRegistry) MaterializePackageAgentsFromRegistry(logf func(string, ...interface{})) []string {
	if r.PackageAgentTemplate() == nil {
		return nil
	}
	if logf == nil {
		logf = log.Printf
	}
	index, err := pkg.NewRegistryClient().FetchIndex()
	if err != nil {
		logf("package agents: registry index unavailable (%v) — new packages fall back to the pkg:* template's workspace", err)
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
		return "no metadata.repository — served by the pkg:* template's default workspace"
	}
	return fmt.Sprintf("metadata.repository %q is not a GitHub tree URL — served by the pkg:* template's default workspace", entry.Repository)
}
