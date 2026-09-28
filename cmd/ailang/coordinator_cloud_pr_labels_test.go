package main

import (
	"reflect"
	"strings"
	"testing"
)

func TestAgentPRLabels(t *testing.T) {
	for _, tc := range []struct {
		agent string
		next  []string
		want  []string
	}{
		{"sprint-planner", []string{"sprint-executor"}, []string{"agent-task", "stage:plan", "merge-fires:sprint-executor"}},
		{"design-doc-creator", []string{"sprint-planner"}, []string{"agent-task", "stage:design", "merge-fires:sprint-planner"}},
		{"sprint-executor", nil, []string{"agent-task", "stage:code"}},
		{"pkg-sunholo-x", nil, []string{"agent-task"}},
	} {
		if got := agentPRLabels(tc.agent, tc.next); !reflect.DeepEqual(got, tc.want) {
			t.Errorf("agentPRLabels(%q, %v) = %v, want %v", tc.agent, tc.next, got, tc.want)
		}
	}
}

// A label that is one of the coordinator's watch_labels would import the PR as a
// new task. Guard the whole generated set, not just today's values.
func TestAgentPRLabelsAvoidWatchLabels(t *testing.T) {
	for agent := range prStageLabels {
		for _, l := range agentPRLabels(agent, []string{"sprint-executor"}) {
			if l == "bug" || l == "feature" || strings.HasPrefix(l, "from:") {
				t.Errorf("agent %s gets watched label %q", agent, l)
			}
		}
	}
}

func TestMergeFiresBanner(t *testing.T) {
	if got := mergeFiresBanner("task-1", nil, true); got != "" {
		t.Fatalf("no handoff must mean no banner, got %q", got)
	}
	plan := mergeFiresBanner("task-86da7498", []string{"sprint-executor"}, false)
	for _, want := range []string{"Merging this PR starts `sprint-executor`", "task-86da7498", "which writes code", "Close the PR"} {
		if !strings.Contains(plan, want) {
			t.Errorf("plan banner missing %q:\n%s", want, plan)
		}
	}
	if strings.Contains(plan, "auto-merge") {
		t.Errorf("banner mentions auto-merge when it is off:\n%s", plan)
	}
	design := mergeFiresBanner("task-f8212936", []string{"sprint-planner"}, true)
	if strings.Contains(design, "writes code") {
		t.Errorf("a design handoff is not a code handoff:\n%s", design)
	}
	if !strings.Contains(design, "merges itself when its checks pass") {
		t.Errorf("auto-merge PR must say it merges itself:\n%s", design)
	}
}
