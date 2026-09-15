package main

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/sunholo-data/ailang/internal/coordinator"
)

// Repairing a card must replace the superseded run's evidence completely — a
// half-repaired card is the same bug with better manners.
func TestRewriteCardFromBranch(t *testing.T) {
	old := `{"changed_files":["design_docs/planned/ailang-core-backlog.md"],` +
		`"diff_stat":"1 file changed, 7 insertions(+)",` +
		`"diff":"--- a/design_docs/planned/ailang-core-backlog.md\n+++ b/…",` +
		`"work_id":"deadbeef","handoff_targets":["sprint-planner"]}`
	branch := []string{"design_docs/planned/ailang-core-triage/coordinator-completion-wrong-ref.md"}

	out, err := rewriteCardFromBranch(old, branch, 1225, "gcp (project ailang-multivac)")
	if err != nil {
		t.Fatalf("rewrite: %v", err)
	}
	var obj map[string]interface{}
	if err := json.Unmarshal([]byte(out), &obj); err != nil {
		t.Fatalf("output is not JSON: %v", err)
	}

	got := coordinator.CardFilesFromContext(out)
	if len(got) != 1 || got[0] != branch[0] {
		t.Errorf("changed_files = %v, want the branch's file", got)
	}
	if _, stillThere := obj["diff"]; stillThere {
		t.Error("the superseded run's patch survived — `--full` would print the old change back")
	}
	if strings.Contains(out, "deadbeef") {
		t.Error("the old work id survived: a replay of the NEW run would then read as new work")
	}
	if wid, _ := obj["work_id"].(string); wid != coordinator.WorkIDForApproval(branch, obj["diff_stat"].(string)) {
		t.Errorf("work_id is not derived from the repaired evidence: %q", wid)
	}
	if src, _ := obj["card_source"].(string); !strings.Contains(src, "#1225") {
		t.Errorf("the repair must record where the evidence came from, got %q", src)
	}
	// Everything else on the card is someone else's field and must survive.
	if _, ok := obj["handoff_targets"]; !ok {
		t.Error("handoff_targets was dropped — the card would stop warning that approving dispatches")
	}
	// A stat that claimed line counts GitHub never gave us would be a worse lie
	// than the stale card.
	if stat, _ := obj["diff_stat"].(string); strings.Contains(stat, "insertion") {
		t.Errorf("re-derived stat invents line counts: %q", stat)
	}

	if _, err := rewriteCardFromBranch("{not json", branch, 1, "local"); err == nil {
		t.Error("an unreadable card must be refused, not silently replaced")
	}
}
