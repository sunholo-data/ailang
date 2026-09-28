package main

import (
	"fmt"
	"slices"
	"strings"
)

// A coordinator PR used to carry one label, agent-task, whatever it was: a
// triage row, a design doc, a sprint plan or 530 lines of code all looked the
// same in the PR list, and nothing said that merging one hands the task to the
// next agent. On 2026-09-28 a sprint-plan PR was merged as "just a plan"; the
// landed-card daemon fired sprint-executor, which opened #1367 with code.
// These helpers make the stage and the consequence of merging visible.

// prStageLabels names the pipeline stage of the agents whose PRs a reviewer
// most needs to tell apart. Agents not listed get no stage label rather than a
// guessed one.
var prStageLabels = map[string]string{
	"ailang-core-triage": "stage:triage",
	"design-doc-creator": "stage:design",
	"sprint-planner":     "stage:plan",
	"sprint-executor":    "stage:code",
}

// agentPRLabels is the label set for a non-cascade coordinator PR. The labels
// are deliberately outside the coordinator's watch_labels (bug, feature,
// from:*), so labelling a PR can never import it as a new task.
func agentPRLabels(agentID string, nextAgents []string) []string {
	labels := []string{"agent-task"}
	if stage := prStageLabels[agentID]; stage != "" {
		labels = append(labels, stage)
	}
	for _, next := range nextAgents {
		labels = append(labels, "merge-fires:"+next)
	}
	return labels
}

// mergeFiresBanner is the first thing in the PR body when merging the PR starts
// another agent. Empty when the agent hands off to nobody.
func mergeFiresBanner(taskID string, nextAgents []string, autoMerge bool) string {
	if len(nextAgents) == 0 {
		return ""
	}
	names := make([]string, len(nextAgents))
	for i, a := range nextAgents {
		names[i] = "`" + a + "`"
	}
	var b strings.Builder
	fmt.Fprintf(&b, "> [!WARNING]\n> **Merging this PR starts %s.** A merge approves task `%s`, and the coordinator then hands it to the next agent", strings.Join(names, ", "), taskID)
	if slices.Contains(nextAgents, "sprint-executor") {
		b.WriteString(", which writes code")
	}
	b.WriteString(". Close the PR instead to stop the chain here.\n")
	if autoMerge {
		b.WriteString("> GitHub auto-merge is enabled: this PR merges itself when its checks pass. Disable auto-merge to hold it.\n")
	}
	b.WriteString("\n")
	return b.String()
}
