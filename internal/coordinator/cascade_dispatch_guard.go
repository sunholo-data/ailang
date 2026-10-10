package coordinator

import (
	"fmt"
	"strings"

	"github.com/sunholo-data/ailang/internal/pubsub"
)

// The cascade-repair dispatch guard.
//
// Origin (recurring, measured across inbox_17 and again on task-a8c0e096,
// 2026-10): messages with NO cascade context were dispatched to pkg-* agents
// under the pkg-update.md template — the agents' DEFAULT template_file — so the
// agent received a "cascade-repair" directive with every cascade field blank
// ({{.RootPackage}}, {{.Source}}, {{.ToVersion}}, ...) and, because the
// template hardcoded the ailang-packages monorepo, sometimes in the WRONG
// repository entirely. Each misroute cost a full paid agent run before the
// template's own defense-in-depth note ("if Source is empty or anything other
// than cascade, the wrapper has misrouted you — stop without committing")
// stopped the agent.
//
// The invariant the template defends already exists in the dispatch contract
// (M-PKG-AUTONOMOUS-CASCADE-SAFE M1, PublishCascadeWithEnvelope): an
// authoritative cascade message carries source=cascade plus a complete
// envelope. Enforcing it here, at dispatch time, is CLAUDE.md Principle 3 —
// one guard where the dispatch happens, instead of every agent re-deriving it
// after the money is spent. A non-cascade message under the cascade template is
// a routing bug: fail loudly, dispatch nothing.

// cascadeRepairTemplateMarker is the template's machine-readable output
// contract, defined in pkg-update.md and parsed by the wrapper. A rendered
// directive containing it came from the cascade-repair workflow — a user
// message that merely QUOTES the marker would have to also survive rendering,
// which is why identity is checked on the marker rather than on prose like
// "cascade-repair" that any bug report about this very guard will contain.
const cascadeRepairTemplateMarker = "BUMP_RESULT:"

// IsCascadeRepairTemplate reports whether template content is the
// cascade-repair (pkg-update.md) workflow.
func IsCascadeRepairTemplate(templateContent string) bool {
	return strings.Contains(templateContent, cascadeRepairTemplateMarker)
}

// CascadeDispatchGaps returns the reasons a task is not an authoritative
// cascade dispatch — the checks the pkg-update.md defense note asks the agent
// to perform, evaluated before the agent exists. Empty slice = dispatch is
// authorized.
func CascadeDispatchGaps(source, rootPackage, changeClass, toVersion string) []string {
	var gaps []string
	if source != pubsub.SourceCascade {
		gaps = append(gaps, fmt.Sprintf("source=%q (want %q)", source, pubsub.SourceCascade))
	}
	if rootPackage == "" {
		gaps = append(gaps, "root_package is empty")
	}
	if changeClass == "" {
		gaps = append(gaps, "change_class is empty")
	}
	if toVersion == "" {
		gaps = append(gaps, "to_version is empty")
	}
	return gaps
}

// ValidateCascadeDirective enforces the guard on a RENDERED directive: a
// cascade-repair directive may only be produced for an authoritative cascade
// with a complete envelope. Returns nil for every other workflow.
func ValidateCascadeDirective(task *TaskRecord, directive string) error {
	if task == nil || !IsCascadeRepairTemplate(directive) {
		return nil
	}
	gaps := CascadeDispatchGaps(task.Source, task.RootPackage, task.RootChangeClass, task.ToVersion)
	if len(gaps) == 0 {
		return nil
	}
	return fmt.Errorf("cascade-repair directive rendered for task %s but it is not an authoritative cascade dispatch: %s",
		task.ID, strings.Join(gaps, "; "))
}
