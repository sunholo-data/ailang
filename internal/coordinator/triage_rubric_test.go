package coordinator

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const (
	skillPath = "../../.agents/skills/ailang-core-triage/SKILL.md"
	rowsDir   = "../../design_docs/planned/ailang-core-triage"
)

func TestParseTriageLimits(t *testing.T) {
	// The real skill: the gate is worthless if it cannot read the live document.
	b, err := os.ReadFile(skillPath)
	if err != nil {
		t.Fatalf("the rubric gate cannot read %s: %v", skillPath, err)
	}
	lim, err := ParseTriageLimits(string(b))
	if err != nil {
		t.Fatalf("thresholds unreadable from the live skill: %v", err)
	}
	if lim.MaxLines != 2 || lim.MaxFiles != 1 {
		// Not a failure if the skill was tuned — but say so loudly, because the
		// gate's strictness just changed.
		t.Logf("thresholds are now lines=%d files=%d (were 2/1)", lim.MaxLines, lim.MaxFiles)
	}

	// A skill that declares neither must fail, not default.
	if _, err := ParseTriageLimits("# no thresholds here"); err == nil {
		t.Error("missing thresholds must be an error: a gate that invents its own numbers is not reading the rubric")
	}
	if _, err := ParseTriageLimits("| `DIRECT_FIX_MAX_LINES` | **2** |"); err == nil {
		t.Error("half the thresholds is still missing thresholds")
	}
}

func TestEstimateLines(t *testing.T) {
	for _, tc := range []struct {
		in   string
		want int
	}{
		// The measured row. A range must read as its LARGEST end.
		{"~15–30 lines in the coordinator executor's pre-flight step", 30},
		{"~15-30 lines", 30},
		{"2 lines in internal/coordinator/store.go", 2},
		{"1 line in x.go", 1},
		{"a one-word change", 0},
		{"lines: unknown", 0},
	} {
		if got := EstimateLines(tc.in); got != tc.want {
			t.Errorf("EstimateLines(%q) = %d, want %d", tc.in, got, tc.want)
		}
	}
}

func TestCheckTriageRow(t *testing.T) {
	lim := TriageLimits{MaxLines: 2, MaxFiles: 1}
	for _, tc := range []struct {
		name    string
		row     TriageRow
		wantBad bool
		expect  string
	}{
		{"the measured violation", TriageRow{
			Recommend: "direct-fix",
			Estimate:  "~15–30 lines in the coordinator executor's pre-flight/PR step (`coordinator_cloud.go` area)",
		}, true, "DIRECT_FIX_MAX_LINES"},
		{"a real direct fix", TriageRow{
			Recommend: "direct-fix", Estimate: "2 lines in internal/coordinator/store.go",
		}, false, ""},
		{"direct-fix with no estimate", TriageRow{Recommend: "direct-fix"}, true, "no `- **Estimate**:`"},
		{"direct-fix with an uncountable estimate", TriageRow{
			Recommend: "direct-fix", Estimate: "small, in the executor",
		}, true, "names no line count"},
		{"direct-fix spanning two files", TriageRow{
			Recommend: "direct-fix", Estimate: "1 line in a.go and 1 line in b.go",
		}, true, "DIRECT_FIX_MAX_FILES"},
		// Everything that is not direct-fix carries no size claim to contradict.
		{"design-doc with a big estimate", TriageRow{
			Recommend: "design-doc", Estimate: "~15–30 lines in coordinator_cloud.go",
		}, false, ""},
		{"duplicate-of", TriageRow{Recommend: "duplicate-of design_docs/planned/x.md"}, false, ""},
		{"drop", TriageRow{Recommend: "drop"}, false, ""},
		{"no verdict at all", TriageRow{}, true, "states no verdict"},
		{"an invented verdict", TriageRow{Recommend: "maybe-later"}, true, "is not one of"},
	} {
		got := CheckTriageRow(tc.row, lim)
		if tc.wantBad && len(got) == 0 {
			t.Errorf("%s: expected a violation, got none", tc.name)
		}
		if !tc.wantBad && len(got) > 0 {
			t.Errorf("%s: expected no violation, got %v", tc.name, got)
		}
		if tc.expect != "" && !strings.Contains(strings.Join(got, " | "), tc.expect) {
			t.Errorf("%s: message %v does not mention %q", tc.name, got, tc.expect)
		}
	}
}

// The gate itself: every triage row in the tree obeys its own numbers.
//
// This is what turns the PR's `test` check red. The coordinator merges with
// `gh pr merge --auto` and no `--admin`, so a red check leaves the PR open
// instead of landing a row whose label contradicts its estimate.
func TestTriageRowsInTreeObeyTheRubric(t *testing.T) {
	b, err := os.ReadFile(skillPath)
	if err != nil {
		t.Fatalf("cannot read the rubric at %s: %v", skillPath, err)
	}
	lim, err := ParseTriageLimits(string(b))
	if err != nil {
		t.Fatalf("cannot read the thresholds: %v", err)
	}

	rows, err := filepath.Glob(filepath.Join(rowsDir, "*.md"))
	if err != nil {
		t.Fatalf("glob: %v", err)
	}
	// An empty search is a claim, not a fact: if the directory ever moves, this
	// test would pass by checking nothing.
	if _, err := os.Stat(rowsDir); err != nil {
		t.Fatalf("%s does not exist — the gate is checking nothing: %v", rowsDir, err)
	}
	for _, path := range rows {
		content, err := os.ReadFile(path)
		if err != nil {
			t.Errorf("%s: %v", path, err)
			continue
		}
		for _, problem := range CheckTriageRow(ParseTriageRow(string(content)), lim) {
			t.Errorf("%s: %s", filepath.Base(path), problem)
		}
	}
	t.Logf("checked %d triage row(s) against lines<=%d files<=%d", len(rows), lim.MaxLines, lim.MaxFiles)
}
