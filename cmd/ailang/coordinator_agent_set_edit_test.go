package main

import "strings"

import "testing"

// The editor must not reflow the document. config.cloud.yaml carries its
// rulings in comments — who decided a thing and what breaks if you widen it —
// and a YAML round-trip moves or drops them, turning a one-word change into an
// unreviewable diff.
const sampleConfig = `  agents:
    - id: design-doc-creator
      label: "Design Doc Creator"
      inbox: design-doc-creator
      # kimi-k3 is the lane that failed its first real use.
      model: openrouter/moonshotai/kimi-k3
      timeout: "2h"
      auto_merge: false

    - id: design-doc-creator-daneel
      label: "Design Doc Creator (Daneel)"
      inbox: daneel-design
      model: openrouter/z-ai/glm-5.3-flash
      auto_merge: true
`

func TestSetAgentField_ChangesOneLineOnly(t *testing.T) {
	out, before, after, err := setAgentField(sampleConfig, "design-doc-creator", "model", "openrouter/z-ai/glm-5.3")
	if err != nil {
		t.Fatal(err)
	}
	if before != "openrouter/moonshotai/kimi-k3" || after != "openrouter/z-ai/glm-5.3" {
		t.Fatalf("before=%q after=%q", before, after)
	}
	if !strings.Contains(out, "      model: openrouter/z-ai/glm-5.3\n") {
		t.Error("the new value is not on the line")
	}
	if !strings.Contains(out, "# kimi-k3 is the lane that failed its first real use.") {
		t.Error("the comment above the field must survive")
	}
	// Exactly one line differs.
	oldLines, newLines := strings.Split(sampleConfig, "\n"), strings.Split(out, "\n")
	if len(oldLines) != len(newLines) {
		t.Fatalf("line count changed: %d -> %d", len(oldLines), len(newLines))
	}
	diff := 0
	for i := range oldLines {
		if oldLines[i] != newLines[i] {
			diff++
		}
	}
	if diff != 1 {
		t.Errorf("%d lines changed, want exactly 1", diff)
	}
}

// design-doc-creator is a PREFIX of design-doc-creator-daneel. Editing the
// wrong agent because one id starts with another is the kind of mistake that
// looks fine in review.
func TestSetAgentField_DoesNotMatchOnPrefix(t *testing.T) {
	out, before, _, err := setAgentField(sampleConfig, "design-doc-creator-daneel", "model", "openrouter/z-ai/glm-5.3")
	if err != nil {
		t.Fatal(err)
	}
	if before != "openrouter/z-ai/glm-5.3-flash" {
		t.Fatalf("edited the wrong agent: previous value was %q", before)
	}
	if !strings.Contains(out, "      model: openrouter/moonshotai/kimi-k3\n") {
		t.Error("the OTHER agent's model must be untouched")
	}
}

func TestSetAgentField_AddsAnAbsentField(t *testing.T) {
	out, before, after, err := setAgentField(sampleConfig, "design-doc-creator", "merge_branch", "dev")
	if err != nil {
		t.Fatal(err)
	}
	if before != "" || after != "dev" {
		t.Fatalf("before=%q after=%q", before, after)
	}
	if !strings.Contains(out, "      merge_branch: dev") {
		t.Error("the field was not added")
	}
	// It must land inside the right block, before the next agent.
	idx, nextAgent := strings.Index(out, "merge_branch: dev"), strings.Index(out, "- id: design-doc-creator-daneel")
	if idx < 0 || idx > nextAgent {
		t.Error("the field landed outside the agent's own block")
	}
}

func TestSetAgentField_UnknownAgentIsAnError(t *testing.T) {
	if _, _, _, err := setAgentField(sampleConfig, "daneel-design", "model", "x"); err == nil {
		t.Fatal("an inbox name is not an agent id — this must fail loudly, not edit nothing silently")
	}
}

// Measured on agent-set's first real run, 2026-09-13: the ladder reported three
// SUCCESSes in about a second and pushed prod before test had finished
// building. It matched the trigger NAME and read the newest build for it —
// which, minutes after any other config change, is a SUCCESS from a different
// commit. A wait that cannot fail is not a wait.
func TestFindBuildStatus_IsScopedToTheCommit(t *testing.T) {
	// Newest-first, exactly as `gcloud builds list --sort-by=~createTime` emits.
	listing := strings.Join([]string{
		"WORKING\tailang-multivac-config-prod\t16a189d",
		"WORKING\tailang-multivac-config-test\t16a189d",
		"SUCCESS\tailang-multivac-config-prod\tdeb0c47",
		"SUCCESS\tailang-multivac-config-test\tdeb0c47",
		"SUCCESS\tailang-multivac-config-dev\tdeb0c47",
		"SUCCESS\t\t", // a build with no trigger and no sha
	}, "\n")

	// The bug: an older SUCCESS for the same trigger must not answer for us.
	if st, found := findBuildStatus(listing, "ailang-multivac-config-prod", "16a189d"); !found || st != "WORKING" {
		t.Errorf("our own build is WORKING; got found=%v status=%q", found, st)
	}
	// Our dev build has not appeared yet — "not found" must not read as done.
	if _, found := findBuildStatus(listing, "ailang-multivac-config-dev", "16a189d"); found {
		t.Error("a build that has not appeared must report not-found, not the previous commit's SUCCESS")
	}
	// The older commit still resolves correctly when asked about explicitly.
	if st, found := findBuildStatus(listing, "ailang-multivac-config-dev", "deb0c47"); !found || st != "SUCCESS" {
		t.Errorf("deb0c47 dev build: found=%v status=%q", found, st)
	}
	if _, found := findBuildStatus("", "ailang-multivac-config-dev", "16a189d"); found {
		t.Error("an empty listing finds nothing")
	}
}

func TestIsTerminalBuildStatus(t *testing.T) {
	for _, s := range []string{"SUCCESS", "FAILURE", "CANCELLED", "TIMEOUT", "INTERNAL_ERROR", "EXPIRED"} {
		if !isTerminalBuildStatus(s) {
			t.Errorf("%q is terminal", s)
		}
	}
	for _, s := range []string{"WORKING", "QUEUED", "PENDING", ""} {
		if isTerminalBuildStatus(s) {
			t.Errorf("%q is not terminal — waiting must continue", s)
		}
	}
}
