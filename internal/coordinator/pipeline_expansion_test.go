package coordinator

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"
)

// M-PIPELINE-RECONCILIATION M4 (D2, ratified 2026-08-26): chain-as-data.
//
// The stapledon-* and twilight-* agent entries were six copy-paste clones of
// the same three-stage chain — zero messages ever received, hand-edited in
// lockstep or (in practice) not at all. A pipeline declares the stages ONCE;
// a binding is one line per project.
const pipelineFixture = `
coordinator:
  pipelines:
    - name: project-dev
      stages:
        - suffix: design-doc
          label_suffix: "Design Doc Creator"
          capabilities: [research, docs]
          skill: design-doc-creator
          output_markers: ["DESIGN_DOC_PATH:"]
          artifact_patterns: ["design_docs/**/*.md"]
        - suffix: sprint-planner
          label_suffix: "Sprint Planner"
          capabilities: [research, docs, planning]
          skill: sprint-planner
          output_markers: ["SPRINT_PLAN_PATH:"]
          artifact_patterns: ["design_docs/**/*.md"]
        - suffix: sprint-executor
          label_suffix: "Sprint Executor"
          capabilities: [code, test, docs]
          skill: sprint-executor
          output_markers: ["IMPLEMENTATION_COMPLETE:"]
          artifact_patterns: ["**/*.go", "**/*.ail"]
      bindings:
        - project: stapledon
          label: Stapledon
          workspace: sunholo-data/stapledons_voyage
          merge_branch: main
        - project: twilight
          label: Twilight
          workspace: MarkEdmondson1234/TwilightGame
          merge_branch: main
  agents: []
`

func expandFixture(t *testing.T) []*AgentConfig {
	t.Helper()
	var file struct {
		Coordinator CoordinatorConfig `yaml:"coordinator"`
	}
	if err := yaml.Unmarshal([]byte(pipelineFixture), &file); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	agents, err := file.Coordinator.ExpandPipelines()
	if err != nil {
		t.Fatalf("expand: %v", err)
	}
	return agents
}

func TestPipelineExpansion_Shape(t *testing.T) {
	agents := expandFixture(t)
	if len(agents) != 6 {
		t.Fatalf("2 bindings x 3 stages must expand to 6 agents, got %d", len(agents))
	}

	byID := map[string]*AgentConfig{}
	for _, a := range agents {
		byID[a.ID] = a
	}

	dd := byID["stapledon-design-doc"]
	if dd == nil {
		t.Fatal("expected stapledon-design-doc")
	}
	if dd.Inbox != "stapledon-design-doc" {
		t.Errorf("inbox: %q", dd.Inbox)
	}
	if dd.Workspace != "sunholo-data/stapledons_voyage" {
		t.Errorf("workspace: %q", dd.Workspace)
	}
	if dd.MergeBranch != "main" {
		t.Errorf("merge_branch: %q", dd.MergeBranch)
	}
	// The chain is rewired WITHIN the binding: stage N triggers stage N+1 of
	// the SAME project.
	if len(dd.TriggerOnComplete) != 1 || dd.TriggerOnComplete[0] != "stapledon-sprint-planner" {
		t.Errorf("trigger chain: %v", dd.TriggerOnComplete)
	}
	ex := byID["twilight-sprint-executor"]
	if ex == nil || len(ex.TriggerOnComplete) != 0 {
		t.Errorf("last stage must trigger nothing, got %+v", ex)
	}
	if dd.Invoke == nil || dd.Invoke.Type != "skill" || dd.Invoke.Name != "design-doc-creator" {
		t.Errorf("invoke must be the stage's skill, got %+v", dd.Invoke)
	}
	if dd.Label != "Stapledon Design Doc Creator" {
		t.Errorf("label: %q", dd.Label)
	}
}

// THE equivalence test: expansion must reproduce the six literal clone entries
// from the live config (2026-08-26 fixture) on every field that affects
// behavior — otherwise deleting the clones changes the fleet.
func TestPipelineExpansion_EquivalentToLiveClones(t *testing.T) {
	data, err := os.ReadFile(filepath.Join("testdata", "live_cloud_config_20260826.yaml"))
	if err != nil {
		t.Fatalf("fixture: %v", err)
	}
	var file struct {
		Coordinator CoordinatorConfig `yaml:"coordinator"`
	}
	if err := yaml.Unmarshal(data, &file); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	literal := map[string]*AgentConfig{}
	for _, a := range file.Coordinator.Agents {
		switch a.ID {
		case "stapledon-design-doc", "stapledon-sprint-planner", "stapledon-sprint-executor",
			"twilight-design-doc", "twilight-sprint-planner", "twilight-sprint-executor":
			literal[a.ID] = a
		}
	}
	if len(literal) != 6 {
		t.Fatalf("fixture must contain the 6 clone entries, found %d", len(literal))
	}

	for _, exp := range expandFixture(t) {
		lit := literal[exp.ID]
		if lit == nil {
			t.Errorf("expanded %q has no literal counterpart", exp.ID)
			continue
		}
		if exp.Inbox != lit.Inbox || exp.Workspace != lit.Workspace || exp.MergeBranch != lit.MergeBranch {
			t.Errorf("%s: routing fields diverge: %q/%q/%q vs %q/%q/%q",
				exp.ID, exp.Inbox, exp.Workspace, exp.MergeBranch, lit.Inbox, lit.Workspace, lit.MergeBranch)
		}
		if len(exp.TriggerOnComplete) != len(lit.TriggerOnComplete) ||
			(len(exp.TriggerOnComplete) > 0 && exp.TriggerOnComplete[0] != lit.TriggerOnComplete[0]) {
			t.Errorf("%s: chain diverges: %v vs %v", exp.ID, exp.TriggerOnComplete, lit.TriggerOnComplete)
		}
		// The three twilight-* entries are the ONE deliberate non-equivalence:
		// all had drifted to `invoke: type: prompt` with template files under
		// /etc/ailang/templates/ — a path that is not even the config mount
		// (/etc/ailang-config/), so those files have never existed on any
		// deployment, and the inboxes have received zero messages ever. Clones
		// pointing at nothing are exactly the drift class D2 deletes; the
		// expansion produces the working skill invokes instead. Found BY this
		// test on 2026-08-26 — as was the fact that twilight's workspace is
		// MarkEdmondson1234/TwilightGame, a different org than assumed.
		if !strings.HasPrefix(exp.ID, "twilight-") {
			if exp.Invoke == nil || lit.Invoke == nil || exp.Invoke.Name != lit.Invoke.Name || exp.Invoke.Type != lit.Invoke.Type {
				t.Errorf("%s: invoke diverges: %+v vs %+v", exp.ID, exp.Invoke, lit.Invoke)
			}
		}
		if exp.Provider != lit.Provider || exp.SessionContinuity != lit.SessionContinuity ||
			exp.MaxConcurrentTasks != lit.MaxConcurrentTasks || exp.AutoMerge != lit.AutoMerge ||
			exp.AutoApproveHandoffs != lit.AutoApproveHandoffs {
			t.Errorf("%s: behavior fields diverge", exp.ID)
		}
	}
}

// An expanded id colliding with a literal agent is a config error, loudly.
func TestPipelineExpansion_CollisionIsError(t *testing.T) {
	src := pipelineFixture + `
`
	var file struct {
		Coordinator CoordinatorConfig `yaml:"coordinator"`
	}
	if err := yaml.Unmarshal([]byte(src), &file); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	file.Coordinator.Agents = []*AgentConfig{{ID: "stapledon-design-doc", Inbox: "x", Workspace: "y"}}
	if _, err := file.Coordinator.ExpandPipelines(); err == nil {
		t.Fatal("an expanded id colliding with a literal agent must be an error")
	}
}
