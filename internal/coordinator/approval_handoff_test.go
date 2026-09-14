package coordinator

import (
	"context"
	"strings"
	"testing"
)

func TestApprovalHandoffTargets_ExcludesAutoEdges(t *testing.T) {
	tests := []struct {
		name  string
		agent *AgentConfig
		want  []string
	}{
		{
			name:  "nil agent",
			agent: nil,
			want:  nil,
		},
		{
			name:  "no handoffs configured",
			agent: &AgentConfig{ID: "a"},
			want:  nil,
		},
		{
			name: "all auto — everything already fired at completion",
			agent: &AgentConfig{
				ID: "a", TriggerOnComplete: []string{"b", "c"}, AutoApproveHandoffs: true,
			},
			want: nil,
		},
		{
			name: "none auto — the pipeline shape",
			agent: &AgentConfig{
				ID: "design-doc-creator", TriggerOnComplete: []string{"sprint-planner"},
			},
			want: []string{"sprint-planner"},
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := approvalHandoffTargets(tc.agent)
			if len(got) != len(tc.want) {
				t.Fatalf("approvalHandoffTargets() = %v, want %v", got, tc.want)
			}
			for i := range got {
				if got[i] != tc.want[i] {
					t.Fatalf("approvalHandoffTargets() = %v, want %v", got, tc.want)
				}
			}
		})
	}
}

// TestHandoffTargetsPartition is the property that stops double-dispatch: every
// configured target fires exactly once, either at completion (auto) or at
// approval (non-auto) — never both, never neither.
func TestHandoffTargetsPartition(t *testing.T) {
	agent := &AgentConfig{
		ID:                  "a",
		TriggerOnComplete:   []string{"b", "c", "d"},
		AutoApproveHandoffs: false,
	}

	f := &finalizer{
		deps: &FinalizeDeps{AgentRegistry: registryWith(agent)},
		in:   FinalizeInput{Task: &TaskRecord{ID: "task-1", AgentID: "a"}},
	}
	auto := f.autoHandoffTargets()
	onApproval := approvalHandoffTargets(agent)

	seen := map[string]int{}
	for _, x := range auto {
		seen[x]++
	}
	for _, x := range onApproval {
		seen[x]++
	}
	for _, target := range agent.TriggerOnComplete {
		switch seen[target] {
		case 0:
			t.Errorf("target %q fires in NEITHER path — the handoff would be lost", target)
		case 1: // correct
		default:
			t.Errorf("target %q fires in BOTH paths — duplicate dispatch", target)
		}
	}
}

func registryWith(agents ...*AgentConfig) *AgentRegistry {
	r := NewAgentRegistry()
	for _, a := range agents {
		if a.Inbox == "" {
			a.Inbox = a.ID
		}
		_ = r.Register(a)
	}
	return r
}

// TestDispatchApprovalHandoffs_LoudWhenAgentUnknown pins the refusal that the
// twelve-day silent failure needed: a registry that cannot see the task's agent
// must not report "nothing to do".
func TestDispatchApprovalHandoffs_LoudWhenAgentUnknown(t *testing.T) {
	task := &TaskRecord{ID: "task-x", AgentID: "ghost"}
	_, err := dispatchApprovalHandoffs(context.Background(), NewAgentRegistry(), nil, task)
	if err == nil {
		t.Fatal("expected an error when the registry does not know the task's agent")
	}
}

// TestDispatchApprovalHandoffs_NoMessageStoreIsLoud: owing a handoff with no way
// to deliver it must not look like success.
func TestDispatchApprovalHandoffs_NoMessageStoreIsLoud(t *testing.T) {
	agent := &AgentConfig{ID: "a", Inbox: "a", TriggerOnComplete: []string{"b"}}
	target := &AgentConfig{ID: "b", Inbox: "b"}
	task := &TaskRecord{ID: "task-y", AgentID: "a"}

	_, err := dispatchApprovalHandoffs(context.Background(), registryWith(agent, target), nil, task)
	if err == nil {
		t.Fatal("expected an error: a handoff is owed but nothing can deliver it")
	}
}

// TestHandoffContent_NamesTheArtifact is the fact the next stage actually needs.
//
// design-doc-creator declares an output_marker of DESIGN_DOC_PATH:, the
// finalizer stores it on the task, and the handoff dropped it — so
// sprint-planner was asked to plan a design doc whose path it was never told,
// from a copy of the original request.
func TestHandoffContent_NamesTheArtifact(t *testing.T) {
	src := &AgentConfig{ID: "design-doc-creator", Label: "Design Doc Creator"}
	task := &TaskRecord{
		ID:            "task-08032ebc",
		Content:       "Design a secondary-model fallback for cloud executor agents",
		DesignDocPath: "design_docs/planned/m-secondary-model-fallback.md",
		BaseBranch:    "dev",
	}

	got := handoffContent(src, task, 0)

	if !strings.Contains(got, "design_docs/planned/m-secondary-model-fallback.md") {
		t.Errorf("the handoff must name the artifact the previous stage produced:\n%s", got)
	}
	if !strings.Contains(got, "Design Doc Creator") || !strings.Contains(got, "task-08032ebc") {
		t.Errorf("the handoff lost its provenance:\n%s", got)
	}
	if !strings.Contains(got, task.Content) {
		t.Errorf("the original request is still context the next stage needs:\n%s", got)
	}
}

// A cloud task has no GitHub issue, and "#0" is a reference to nothing that
// reads exactly like a real one.
func TestHandoffContent_OmitsAbsentIssueNumber(t *testing.T) {
	src := &AgentConfig{ID: "a", Label: "A"}
	task := &TaskRecord{ID: "task-x", Content: "do the thing"}

	if got := handoffContent(src, task, 0); strings.Contains(got, "#0") {
		t.Errorf("an absent issue must be omitted, not rendered as #0:\n%s", got)
	}
	if got := handoffContent(src, task, 1170); !strings.Contains(got, "#1170") {
		t.Errorf("a real issue number must still appear:\n%s", got)
	}
}

// A task with no artifact still produces a usable handoff — the fields are
// additive, not required.
func TestHandoffContent_SurvivesAnEmptyTask(t *testing.T) {
	got := handoffContent(&AgentConfig{ID: "a", Label: "A"}, &TaskRecord{ID: "task-y"}, 0)
	if !strings.Contains(got, "Please continue") {
		t.Errorf("a bare task must still hand off:\n%s", got)
	}
	for _, unwanted := range []string{"Design doc:", "Sprint plan:", "Branch:"} {
		if strings.Contains(got, unwanted) {
			t.Errorf("an unset field must be omitted, not rendered empty (%s):\n%s", unwanted, got)
		}
	}
}
