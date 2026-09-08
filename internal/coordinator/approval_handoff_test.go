package coordinator

import (
	"context"
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
