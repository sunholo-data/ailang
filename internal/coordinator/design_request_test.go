package coordinator

import (
	"context"
	"strings"
	"testing"
)

func designRequestTask() *TaskRecord {
	return &TaskRecord{ID: "task-design", Kind: "request", Content: `{"workflow":"design-document-v1","project":"ailang","request":"Design receipts; do not implement"}`}
}

func TestDesignRequestScopeAndControl(t *testing.T) {
	agent := &AgentConfig{ID: "website-builder", Inbox: "website-builder", Workspace: "sunholo-data/sunholo-websites", Model: "existing-model", DesignRequests: true, Invoke: &InvokeConfig{Type: "prompt", Template: "ordinary maintenance {{.Content}}"}, SkipApproval: true, AutoMerge: true, SessionContinuity: true, TriggerOnComplete: []string{"sprint-planner"}, Subdirectory: "packages/auth"}
	task := designRequestTask()
	effective := AgentForTask(agent, task)
	if effective.ID != agent.ID || effective.Inbox != agent.Inbox || effective.Workspace != agent.Workspace || effective.Model != agent.Model {
		t.Fatal("changed routing or model")
	}
	if effective.SkipApproval || effective.AutoMerge || effective.SessionContinuity || len(effective.TriggerOnComplete) > 0 || effective.Subdirectory != "" {
		t.Fatal("design request retained implementation authority")
	}
	if effective.Invoke.Type != "skill" || effective.Invoke.Name != "design-doc-creator" {
		t.Fatal("did not choose existing skill")
	}
	directive := BuildDirectiveFromConfig(task, effective)
	for _, want := range []string{"design-doc-creator", "SKILL.md", "DESIGN_DOC_PATH:", task.Content} {
		if !strings.Contains(directive, want) {
			t.Fatalf("directive missing %q", want)
		}
	}
	if !agent.SkipApproval || !agent.AutoMerge || len(agent.TriggerOnComplete) != 1 {
		t.Fatal("mutated registry")
	}
	for _, content := range []string{`{"request":"ordinary maintenance"}`, `{"workflow":"another-workflow"}`, `quoted design-document-v1`} {
		control := &TaskRecord{Kind: "request", Content: content}
		if AgentForTask(agent, control) != agent {
			t.Fatal("changed ordinary request")
		}
		if !strings.HasPrefix(BuildDirectiveFromConfig(control, agent), "ordinary maintenance") {
			t.Fatal("lost ordinary template")
		}
	}
	task.Kind = "feedback"
	if AgentForTask(agent, task) != agent {
		t.Fatal("rerouted feedback")
	}
	task.Kind = "request"
	agent.DesignRequests = false
	if !strings.Contains(BuildDirectiveFromConfig(task, agent), "not enabled") {
		t.Fatal("unsupported route did not refuse")
	}
}

func TestDesignRequestCompletionNeverHandsOff(t *testing.T) {
	for _, auto := range []bool{false, true} {
		for _, design := range []bool{false, true} {
			name := "ordinary"
			if design {
				name = "design"
			}
			if auto {
				name += "-auto"
			}
			t.Run(name, func(t *testing.T) {
				agent := handoffAgent(auto)
				agent.DesignRequests = true
				h := newFinalizeHarness(t, agent)
				if design {
					h.task.Kind = "request"
					h.task.Content = designRequestTask().Content
				}
				h.finalize(t, OutcomeCompleted, false)
				approval := h.approval(t)
				if !design && approval == nil {
					t.Fatal("ordinary request lost branch review record")
				}
				count := len(h.handoffs(t))
				if design {
					if count != 0 {
						t.Fatal("design dispatched automatic successor")
					}
					if approval != nil {
						t.Fatal("design created a deferred approval")
					}
					targets, err := dispatchApprovalHandoffs(context.Background(), h.deps.AgentRegistry, nil, h.task)
					if err != nil || len(targets) != 0 {
						t.Fatalf("approval attempted successor: %v %v", targets, err)
					}
					// Replay must remain bounded even if registry still contains successor edges.
					h.finalize(t, OutcomeCompleted, true)
					if len(h.handoffs(t)) != 0 {
						t.Fatal("replay dispatched successor")
					}
				} else if auto {
					if count != 1 {
						t.Fatalf("positive control: ordinary auto handoffs=%d", count)
					}
				} else if !strings.Contains(approval.ContextJSON, "handoff_targets") {
					t.Fatal("positive control: ordinary approval lost successor")
				}
			})
		}
	}
}
