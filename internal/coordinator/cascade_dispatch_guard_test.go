package coordinator

import (
	"strings"
	"testing"
)

// The cascade-repair dispatch guard (M-CASCADE-DISPATCH-GUARD).
//
// Origin: task-a8c0e096 (2026-10) and a recurring family in inbox_17 —
// messages with EMPTY cascade context dispatched to pkg-* agents under the
// pkg-update.md (cascade-repair) template, each costing a full paid agent run
// before the template's defense-in-depth note stopped the agent. These tests
// pin the invariant at dispatch time: source=cascade plus a complete envelope,
// or no dispatch.

func TestIsCascadeRepairTemplate(t *testing.T) {
	pkgUpdate := "- `BUMP_RESULT: success` (or `failed: <reason>`)\n"
	if !IsCascadeRepairTemplate(pkgUpdate) {
		t.Error("template carrying the BUMP_RESULT output contract should be recognized as cascade-repair")
	}
	// A directive about the guard (this bug's own reports quote the word
	// "cascade-repair") must NOT be mistaken for the template.
	if IsCascadeRepairTemplate("file a [bug] cascade routing issue and stop") {
		t.Error("prose quoting 'cascade-repair' must not be recognized as the template")
	}
	if IsCascadeRepairTemplate("") {
		t.Error("empty template content must not be recognized as cascade-repair")
	}
}

func TestCascadeDispatchGaps(t *testing.T) {
	// Authoritative cascade with a complete envelope: no gaps.
	if gaps := CascadeDispatchGaps("cascade", "sunholo/auth", "C", "0.2.0"); len(gaps) != 0 {
		t.Errorf("complete cascade dispatch reported gaps: %v", gaps)
	}
	// The misroute that actually happened: empty everything.
	gaps := CascadeDispatchGaps("", "", "", "")
	want := []string{"source", "root_package", "change_class", "to_version"}
	if len(gaps) != len(want) {
		t.Fatalf("got %d gaps (%v), want %d", len(gaps), gaps, len(want))
	}
	for i, w := range want {
		if !strings.Contains(gaps[i], w) {
			t.Errorf("gap %d = %q, want it to name %q", i, gaps[i], w)
		}
	}
	// Source=cascade but the envelope never arrived (the "EMPTY dispatch").
	gaps = CascadeDispatchGaps("cascade", "", "", "")
	if len(gaps) != 3 {
		t.Errorf("source=cascade with empty envelope: got gaps %v, want 3 envelope gaps", gaps)
	}
	// Right topic, wrong envelope: a non-cascade source must always be a gap
	// even when envelope fields are present (public-routed feedback with a
	// fabricated root_package is exactly the spoof the source guard exists for).
	if gaps := CascadeDispatchGaps("public", "sunholo/auth", "C", "0.2.0"); len(gaps) != 1 || !strings.Contains(gaps[0], "source") {
		t.Errorf("non-cascade source with full envelope: got %v, want exactly the source gap", gaps)
	}
}

func TestValidateCascadeDirective(t *testing.T) {
	cascadeDirective := "BUMP_RESULT: success\n[cascade-repair] adapt to change\n"
	task := &TaskRecord{ID: "task-test", Source: "cascade", RootPackage: "sunholo/auth", RootChangeClass: "C", ToVersion: "0.2.0"}

	if err := ValidateCascadeDirective(task, cascadeDirective); err != nil {
		t.Errorf("authoritative cascade with complete envelope: unexpected error: %v", err)
	}

	// The actual incident: rendered cascade-repair directive, empty context.
	empty := &TaskRecord{ID: "task-test"}
	err := ValidateCascadeDirective(empty, cascadeDirective)
	if err == nil {
		t.Fatal("cascade-repair directive with empty context must be refused")
	}
	for _, want := range []string{"source", "root_package", "change_class", "to_version", "task-test"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("error %q does not name %q", err.Error(), want)
		}
	}

	// Other workflows pass through untouched — the guard must not gate the
	// whole coordinator, only the cascade-repair template.
	if err := ValidateCascadeDirective(empty, "You are a sprint planner. Plan the sprint."); err != nil {
		t.Errorf("non-cascade directive refused: %v", err)
	}
	if err := ValidateCascadeDirective(nil, cascadeDirective); err != nil {
		t.Errorf("nil task with cascade directive: unexpected error %v (caller has other nil checks)", err)
	}
}

// The template must name the repo the wrapper ACTUALLY cloned. task-a8c0e096
// ran in sunholo-data/docparse while the directive hardcoded
// "sunholo-data/ailang-packages" — the agent was told it was somewhere else.
func TestBuildTemplateDirectiveRendersActualRepo(t *testing.T) {
	agent := &AgentConfig{
		ID:           "pkg-sunholo-docparse",
		Workspace:    "sunholo-data/docparse",
		Subdirectory: "packages/docparse",
		Invoke: &InvokeConfig{
			Type:     "prompt",
			Template: "Repo: {{.Repo}}\nDir: {{.Subdirectory}}\nSource: {{.Source}}",
		},
	}
	task := &TaskRecord{ID: "task-test", Source: "cascade", RootPackage: "sunholo/docparse", ToVersion: "1.2.3"}

	directive := BuildDirectiveFromConfig(task, agent)
	if !strings.Contains(directive, "Repo: sunholo-data/docparse") {
		t.Errorf("directive must render the agent's resolved repo, got:\n%s", directive)
	}
	if !strings.Contains(directive, "Dir: packages/docparse") {
		t.Errorf("directive must render the agent's subdirectory, got:\n%s", directive)
	}

	// A local-path workspace (no org/repo coordinate) renders as the workspace
	// rather than empty — the truth about where the agent is standing.
	agent.Workspace = "/home/dev/mk-docparse"
	agent.Repo = ""
	directive = BuildDirectiveFromConfig(task, agent)
	if !strings.Contains(directive, "Repo: /home/dev/mk-docparse") {
		t.Errorf("local workspace should render as-is rather than blank, got:\n%s", directive)
	}
}
