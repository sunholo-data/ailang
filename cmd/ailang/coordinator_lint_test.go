package main

import (
	"strings"
	"testing"

	"github.com/sunholo-data/ailang/internal/coordinator"
)

func agent(id string, mut ...func(*coordinator.AgentConfig)) *coordinator.AgentConfig {
	a := &coordinator.AgentConfig{ID: id, Inbox: id, Workspace: "/tmp/t"}
	for _, m := range mut {
		m(a)
	}
	return a
}

func findingRules(fs []lintFinding) string {
	var r []string
	for _, f := range fs {
		r = append(r, f.Rule)
	}
	return strings.Join(r, ",")
}

// A registry with nothing wrong must produce nothing. A linter that always
// reports something gets ignored, which is the failure mode this whole exercise
// is about.
func TestLintRegistry_CleanChainIsSilent(t *testing.T) {
	agents := []*coordinator.AgentConfig{
		agent("designer", func(a *coordinator.AgentConfig) {
			a.TriggerOnComplete = []string{"planner"}
			a.AutoMerge = true
			a.ArtifactPatterns = []string{"design_docs/**/*.md"}
		}),
		agent("planner", func(a *coordinator.AgentConfig) {
			a.TriggerOnComplete = []string{"executor"}
			a.AutoApproveHandoffTo = []string{"executor"}
		}),
		agent("executor", func(a *coordinator.AgentConfig) {
			a.TriggerOnComplete = []string{"evaluator"}
			a.AutoApproveHandoffTo = []string{"evaluator"}
		}),
		agent("evaluator", func(a *coordinator.AgentConfig) { a.SkipApproval = true }),
	}
	if got := lintRegistry(agents); len(got) != 0 {
		t.Errorf("clean registry produced %d finding(s): %s", len(got), findingRules(got))
	}
}

// THE LANDMINE. skip_approval means applyApproval never runs, so nothing embeds
// the handoff target; a non-auto edge means nothing dispatches it at completion
// either. The edge is present, reads as configured, and fires never.
func TestLintRegistry_SkipApprovalWithGatedEdgeCanNeverFire(t *testing.T) {
	agents := []*coordinator.AgentConfig{
		agent("writer", func(a *coordinator.AgentConfig) {
			a.SkipApproval = true
			a.TriggerOnComplete = []string{"reviewer"}
		}),
		agent("reviewer"),
	}
	got := lintRegistry(agents)
	if len(got) != 1 || got[0].Rule != "handoff-can-fire" {
		t.Fatalf("want one handoff-can-fire finding, got %d: %s", len(got), findingRules(got))
	}
	if !strings.Contains(got[0].Msg, "reviewer") {
		t.Errorf("the finding must name the target it cannot reach: %q", got[0].Msg)
	}
}

// Same agent, edge made auto: now it dispatches at completion and is fine.
func TestLintRegistry_SkipApprovalWithAutoEdgeIsFine(t *testing.T) {
	agents := []*coordinator.AgentConfig{
		agent("writer", func(a *coordinator.AgentConfig) {
			a.SkipApproval = true
			a.TriggerOnComplete = []string{"reviewer"}
			a.AutoApproveHandoffTo = []string{"reviewer"}
		}),
		agent("reviewer"),
	}
	if got := lintRegistry(agents); len(got) != 0 {
		t.Errorf("an auto edge has a dispatch path; got %s", findingRules(got))
	}
}

func TestLintRegistry_EdgeToUnknownAgent(t *testing.T) {
	agents := []*coordinator.AgentConfig{
		agent("planner", func(a *coordinator.AgentConfig) { a.TriggerOnComplete = []string{"sprint-exectuor"} }),
	}
	got := lintRegistry(agents)
	if len(got) != 1 || got[0].Rule != "edge-target-exists" {
		t.Fatalf("want edge-target-exists, got %d: %s", len(got), findingRules(got))
	}
}

// A typo'd target must not ALSO be reported as an unreleasable handoff — one
// fault, one finding, or the real one gets lost in the noise.
func TestLintRegistry_UnknownTargetReportsOnce(t *testing.T) {
	agents := []*coordinator.AgentConfig{
		agent("writer", func(a *coordinator.AgentConfig) {
			a.SkipApproval = true
			a.TriggerOnComplete = []string{"nope"}
		}),
	}
	if got := lintRegistry(agents); len(got) != 1 {
		t.Errorf("want exactly one finding for one typo, got %d: %s", len(got), findingRules(got))
	}
}

// The evaluator loop-back the pipeline wants, wired the naive way: it would
// re-dispatch on PASS as well as FAIL, unbounded, because there is no round
// counter anywhere in the codebase.
func TestLintRegistry_CycleIsReportedOnce(t *testing.T) {
	agents := []*coordinator.AgentConfig{
		agent("executor", func(a *coordinator.AgentConfig) {
			a.TriggerOnComplete = []string{"evaluator"}
			a.AutoApproveHandoffTo = []string{"evaluator"}
		}),
		agent("evaluator", func(a *coordinator.AgentConfig) {
			a.TriggerOnComplete = []string{"executor"}
			a.AutoApproveHandoffTo = []string{"executor"}
		}),
	}
	got := lintRegistry(agents)
	var cycles int
	for _, f := range got {
		if f.Rule == "no-chain-cycle" {
			cycles++
			if !strings.Contains(f.Msg, "->") {
				t.Errorf("a cycle finding must show the path: %q", f.Msg)
			}
		}
	}
	if cycles != 1 {
		t.Errorf("want the cycle reported exactly once, got %d (%s)", cycles, findingRules(got))
	}
}

// A self-edge is the degenerate cycle and must still be caught.
func TestLintRegistry_SelfEdgeIsACycle(t *testing.T) {
	agents := []*coordinator.AgentConfig{
		agent("loop", func(a *coordinator.AgentConfig) {
			a.TriggerOnComplete = []string{"loop"}
			a.AutoApproveHandoffTo = []string{"loop"}
		}),
	}
	got := lintRegistry(agents)
	if len(got) != 1 || got[0].Rule != "no-chain-cycle" {
		t.Fatalf("want no-chain-cycle, got %d: %s", len(got), findingRules(got))
	}
}

// A diamond is not a cycle: two paths converging must not be reported.
func TestLintRegistry_DiamondIsNotACycle(t *testing.T) {
	agents := []*coordinator.AgentConfig{
		agent("a", func(x *coordinator.AgentConfig) {
			x.TriggerOnComplete = []string{"b", "c"}
			x.AutoApproveHandoffs = true
		}),
		agent("b", func(x *coordinator.AgentConfig) {
			x.TriggerOnComplete = []string{"d"}
			x.AutoApproveHandoffs = true
		}),
		agent("c", func(x *coordinator.AgentConfig) {
			x.TriggerOnComplete = []string{"d"}
			x.AutoApproveHandoffs = true
		}),
		agent("d"),
	}
	if got := lintRegistry(agents); len(got) != 0 {
		t.Errorf("a diamond is acyclic; got %s", findingRules(got))
	}
}

func TestLintRegistry_AutoMergeNeedsDeclaredPatterns(t *testing.T) {
	agents := []*coordinator.AgentConfig{
		agent("merger", func(a *coordinator.AgentConfig) { a.AutoMerge = true }),
	}
	got := lintRegistry(agents)
	if len(got) != 1 || got[0].Rule != "automerge-is-bounded" {
		t.Fatalf("want automerge-is-bounded, got %d: %s", len(got), findingRules(got))
	}
}

// daneel-design vs daneel-design-ailang is NOT this rule (they differ by a whole
// word). What this catches is the separator/case class: daneel_design.
func TestLintRegistry_NearDuplicateInboxes(t *testing.T) {
	agents := []*coordinator.AgentConfig{
		agent("one", func(a *coordinator.AgentConfig) { a.Inbox = "daneel-design" }),
		agent("two", func(a *coordinator.AgentConfig) { a.Inbox = "daneel_design" }),
	}
	got := lintRegistry(agents)
	if len(got) != 1 || got[0].Rule != "inbox-not-near-duplicate" {
		t.Fatalf("want inbox-not-near-duplicate, got %d: %s", len(got), findingRules(got))
	}
}
