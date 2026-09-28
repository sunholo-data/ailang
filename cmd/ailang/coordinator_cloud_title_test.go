package main

import (
	"testing"

	"github.com/sunholo-data/ailang/internal/config"
)

// Real directives from the coordinator PRs that prompted this (2026-09-28).
const (
	// #1347 — first stage: the request arrives as a JSON payload whose request
	// opens with an email "Subject:" line.
	directive1347 = `{"project":"ailang","target":"ailang","request":"Subject: AILANG package registry\n\nImprove AILANG documentation website to show the agents.md for an overview of the package.\n\nSent from my iPhone"}

Invoke the design-doc-creator skill to complete this task.`

	// #1356 — a handoff: the original request survives only truncated at 500
	// characters, so it is not decodable JSON.
	directive1356 = `**Handoff from Design Doc Creator**

Task: task-f8212936
Artifact: design_docs/planned/v0_47_1/m-pkg-registry-discoverability.md
Work branch: coordinator/task-f8212936

Original Request: {"project":"ailang","target":"ailang","request":"Subject: AILANG package registry\n\nImprove AILANG documentation website to show the agents

Previous work has been approved. Please continue.

Invoke the sprint-planner skill to complete this task.`
)

// MU: return the cleaned task title unconditionally (skip the opaque check)
// and the hash cases fail.
func TestTaskSubject_OpaqueTitleUsesTheRequest(t *testing.T) {
	cases := []struct {
		name, title, directive, want string
	}{
		{"first stage, Daneel hash", "Daneel design b5f7599fba5f0e7e33510da94c533146f98d43cef80ae8b0", directive1347, "AILANG package registry"},
		{"handoff, truncated request", "Handoff: Daneel design b5f7599fba5f0e7e33510da94c533146f98d43cef80ae8b0", directive1356, "AILANG package registry"},
		{"double handoff", "Handoff: Handoff: Daneel design b5f7599fba5f0e7e33510da94c533146f98d43cef80ae8b0 (approved)", directive1356, "AILANG package registry"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Setenv(config.EnvTaskTitle, c.title)
			if got := taskSubject(c.directive); got != c.want {
				t.Errorf("subject = %q, want %q", got, c.want)
			}
		})
	}
}

// A human title is kept — only the handoff wrapping is removed.
// MU: drop stripHandoffWrapping and the first case keeps "Handoff: ".
func TestTaskSubject_HumanTitleKeptWithoutHandoffNoise(t *testing.T) {
	t.Setenv(config.EnvTaskTitle, "Handoff: Bug: --strict-bytecode GET_FIELD uses the wrong slot (approved)")
	if got := taskSubject("irrelevant"); got != "Bug: --strict-bytecode GET_FIELD uses the wrong slot" {
		t.Errorf("subject = %q", got)
	}
	t.Setenv(config.EnvTaskTitle, "Plan M-RIG-GPU-ADMISSION-GATEWAY Phase 2 (cutover)")
	if got := taskSubject("irrelevant"); got != "Plan M-RIG-GPU-ADMISSION-GATEWAY Phase 2 (cutover)" {
		t.Errorf("a clean human title was changed: %q", got)
	}
}

// When nothing better exists, the cleaned title still beats nothing.
func TestTaskSubject_OpaqueTitleNoRequestFallsBack(t *testing.T) {
	t.Setenv(config.EnvTaskTitle, "Handoff: Daneel design b5f7599fba5f0e7e33510da94c533146")
	if got := taskSubject(""); got != "Daneel design b5f7599fba5f0e7e33510da94c533146" {
		t.Errorf("subject = %q", got)
	}
}

// Stock lead-ins give way to the subject.
// MU: return s unchanged from stripRequestBoilerplate and this fails.
func TestTaskSubject_StripsRequestLeadIn(t *testing.T) {
	t.Setenv(config.EnvTaskTitle, "Handoff: Daneel design 812b11b7485f50cb7ababc47600fbad0b56be04716d35")
	dir := `Original Request: {"request":"Create a design document for a salvage/retry policy for cloud executor tasks that time out`
	if got := taskSubject(dir); got != "Salvage/retry policy for cloud executor tasks that time out" {
		t.Errorf("subject = %q", got)
	}
}
