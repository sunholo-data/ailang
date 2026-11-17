package mission

import (
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

// M-FLEET-RULINGS-LEDGER M3 tests for CheckRulings. Fixtures come from rulings_corpus_test.go.

const (
	rulingLiteral = "retired-phrase-x"
	rowOpen1      = "| D-FR-1 | OPEN | ruling one | evidence one |"
)

// linkedCharter has a link to the ledger outside its decision-ledger block.
const linkedCharter = "# C\n\n> **Fleet-wide rulings:** [fleet-rulings.md](fleet-rulings.md).\n\n" +
	"<!-- decision-ledger:start -->\n| ID | Status | Ruling | Evidence |\n|---|---|---|---|\n<!-- decision-ledger:end -->\n"

// ledger writes design_docs/fleet-rulings.md from raw ledger rows and raw detector lines.
func (f *rulingsFixture) ledger(rows, detectors string) {
	f.t.Helper()
	body := "# Fleet-wide rulings\n\n" + ledgerStartMarker + "\n| ID | Status | Ruling | Evidence |\n|---|---|---|---|\n" +
		rows + "\n" + ledgerEndMarker + "\n\n" + detectorsStartMarker + "\n" + detectors + "\n" + detectorsEndMarker + "\n"
	f.write(rulingsLedgerRel, body)
}

// inRepoMission is a mission whose charter lives under the registry root.
func (f *rulingsFixture) inRepoMission(name string) *Mission {
	f.t.Helper()
	return f.mission(name, sharedDriverRepo, filepath.Join(f.root, "wd", name), "design_docs/"+name+".md", "")
}

func contradictions(fs []Finding) []Finding {
	var out []Finding
	for _, f := range fs {
		if f.Kind == "ruling-contradiction" {
			out = append(out, f)
		}
	}
	return out
}

// skillCase builds a ledger with the given rows and detectors, one skill file holding the
// literal on line 3, and one linked in-repo mission. It returns CheckRulings' output.
func skillCase(t *testing.T, rows, detectors string) []Finding {
	t.Helper()
	f := newRulingsFixture(t)
	f.ledger(rows, detectors)
	f.write(".claude/skills/mission-a/resources/g.md", "one\ntwo\n"+rulingLiteral+" here\n")
	f.write("design_docs/a.md", linkedCharter)
	return CheckRulings([]*Mission{f.inRepoMission("a")})
}

func TestRulings_SkillHitNamesRulingFileAndLine(t *testing.T) {
	got := skillCase(t, rowResolved1, `D-FR-1 forbid skills "`+rulingLiteral+`"`)
	if len(got) != 1 {
		t.Fatalf("want exactly 1 finding, got %d: %+v", len(got), got)
	}
	g := got[0]
	if g.Kind != "ruling-contradiction" || g.Mission != rulingsMission || g.Severity != Note {
		t.Fatalf("finding shape wrong: %+v", g)
	}
	for _, want := range []string{"D-FR-1", ".claude/skills/mission-a/resources/g.md:3"} {
		if !strings.Contains(g.Detail, want) {
			t.Errorf("detail %q lacks %q", g.Detail, want)
		}
	}
}

func TestRulings_OpenAndSupersededAreQuiet(t *testing.T) {
	det := `D-FR-1 forbid skills "` + rulingLiteral + `"`
	cases := []struct {
		name, rows, dets string
		wantHits         int
	}{
		{"open ruling is quiet", rowOpen1, det, 0},
		{"superseded by RESOLVED is quiet", rowResolved1 + "\n" + rowResolved2, det + "\nD-FR-2 supersedes D-FR-1", 0},
		{"superseded by OPEN still hits", rowResolved1 + "\n" + rowOpen2, det + "\nD-FR-2 supersedes D-FR-1", 1},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if n := len(contradictions(skillCase(t, c.rows, c.dets))); n != c.wantHits {
				t.Fatalf("contradictions = %d, want %d", n, c.wantHits)
			}
		})
	}
}

func TestRulings_CharterLedgerBlockExcluded(t *testing.T) {
	// The literal sits inside the charter's own ledger block: it must not hit.
	inside := "<!-- decision-ledger:start -->\n| ID | Status | Ruling | Evidence |\n|---|---|---|---|\n" +
		"| D-FR-1 | RESOLVED | " + rulingLiteral + " | e |\n<!-- decision-ledger:end -->\n"
	linkLine := "> [fleet-rulings.md](fleet-rulings.md)\n\n"

	run := func(t *testing.T, charter string) []Finding {
		t.Helper()
		f := newRulingsFixture(t)
		f.ledger(rowResolved1, `D-FR-1 forbid charters "`+rulingLiteral+`"`)
		f.write("design_docs/a.md", charter)
		return CheckRulings([]*Mission{f.inRepoMission("a")})
	}

	if got := contradictions(run(t, "# C\n"+linkLine+inside)); len(got) != 0 {
		t.Fatalf("ledger-block literal hit: %+v", got)
	}
	got := contradictions(run(t, "# C\n"+linkLine+"outside "+rulingLiteral+"\n"+inside))
	if len(got) != 1 || got[0].Mission != "a" {
		t.Fatalf("outside literal: want 1 contradiction owned by a, got %+v", got)
	}
}

func TestRulings_MalformedLedgerRunsNothing(t *testing.T) {
	f := newRulingsFixture(t)
	f.ledger(rowResolved1, "D-FR-1 forbid skills \""+rulingLiteral+"\"\nD-FR-1 forbid skills \"short\"")
	f.write(".claude/skills/mission-a/resources/g.md", rulingLiteral+"\n")
	f.write("design_docs/a.md", "# C\n")
	got := CheckRulings([]*Mission{f.inRepoMission("a")})
	if len(got) != 1 || got[0].Kind != "rulings-malformed" {
		t.Fatalf("want exactly one rulings-malformed, got %+v", got)
	}
	data, err := os.ReadFile(filepath.Join(f.root, rulingsLedgerRel))
	if err != nil {
		t.Fatal(err)
	}
	wantLine := fmt.Sprintf("line %d:", lineOf(t, string(data), `D-FR-1 forbid skills "short"`))
	if !strings.Contains(got[0].Detail, wantLine) {
		t.Fatalf("detail %q does not name the bad line (%s)", got[0].Detail, wantLine)
	}
}

func TestRulings_DetectorCapEndToEnd(t *testing.T) {
	hit := `D-FR-1 forbid skills "` + rulingLiteral + `"`
	run := func(t *testing.T, dets []string) ([]Finding, string) {
		t.Helper()
		f := newRulingsFixture(t)
		f.ledger(rowResolved1, strings.Join(dets, "\n"))
		f.write(".claude/skills/mission-a/resources/g.md", rulingLiteral+"\n")
		f.write("design_docs/a.md", linkedCharter)
		got := CheckRulings([]*Mission{f.inRepoMission("a")})
		data, err := os.ReadFile(filepath.Join(f.root, rulingsLedgerRel))
		if err != nil {
			t.Fatal(err)
		}
		return got, string(data)
	}

	// 65 detector lines, the 65th is the hit: one malformed finding naming its line.
	dets65 := append(genDetectors(64), hit)
	got, doc := run(t, dets65)
	if len(got) != 1 || got[0].Kind != "rulings-malformed" {
		t.Fatalf("65 lines: want one rulings-malformed, got %+v", got)
	}
	if want := fmt.Sprintf("line %d:", lineOf(t, doc, hit)); !strings.Contains(got[0].Detail, want) {
		t.Fatalf("65 lines: detail %q lacks %q", got[0].Detail, want)
	}

	// 64 detector lines, the 64th is the hit: runs normally.
	got, _ = run(t, append(genDetectors(63), hit))
	if len(contradictions(got)) < 1 {
		t.Fatalf("64 lines: want >=1 contradiction, got %+v", got)
	}
	for _, g := range got {
		if g.Kind == "rulings-malformed" {
			t.Fatalf("64 lines: unexpected malformed finding %+v", g)
		}
	}
}

func TestRulings_AbsentLedgerIsOneNote(t *testing.T) {
	f := newRulingsFixture(t)
	f.write("design_docs/a.md", linkedCharter)
	got := CheckRulings([]*Mission{f.inRepoMission("a")})
	if len(got) != 1 || got[0].Kind != "rulings-ledger-absent" {
		t.Fatalf("want exactly one rulings-ledger-absent, got %+v", got)
	}
	if code := (&Report{Findings: got}).ExitCode(); code != 0 {
		t.Fatalf("absent ledger must not fail the run: exit %d", code)
	}
}

func TestRulings_EveryFindingIsNote(t *testing.T) {
	f := newRulingsFixture(t)
	f.ledger(rowResolved1, `D-FR-1 forbid skills "`+rulingLiteral+`"`)
	f.write(".claude/skills/mission-a/resources/g.md", rulingLiteral+"\n")
	f.write("design_docs/a.md", "# C\n") // unlinked: ruling-link-missing
	// foreign mission whose charter is absent: ruling-file-unreadable
	missing := f.mission("b", "other/repo", filepath.Join(f.root, "wd", "b"), "c.md", "")
	got := CheckRulings([]*Mission{f.inRepoMission("a"), missing})

	kinds := map[string]bool{}
	for _, g := range got {
		kinds[g.Kind] = true
		if g.Severity != Note {
			t.Errorf("finding %+v is not a Note", g)
		}
	}
	for _, k := range []string{"ruling-contradiction", "ruling-link-missing", "ruling-file-unreadable"} {
		if !kinds[k] {
			t.Errorf("fixture did not produce %s: %+v", k, got)
		}
	}
	if code := (&Report{Findings: got}).ExitCode(); code != 0 {
		t.Fatalf("notes must not fail the run: exit %d", code)
	}
}

func TestRulings_HitCapTwentyThenSummary(t *testing.T) {
	f := newRulingsFixture(t)
	f.ledger(rowResolved1, `D-FR-1 forbid skills "`+rulingLiteral+`"`)
	var b strings.Builder
	for i := 0; i < 23; i++ {
		fmt.Fprintf(&b, "%s line %d\n", rulingLiteral, i)
	}
	f.write(".claude/skills/mission-a/resources/g.md", b.String())
	f.write("design_docs/a.md", linkedCharter)
	got := contradictions(CheckRulings([]*Mission{f.inRepoMission("a")}))
	if len(got) != 21 {
		t.Fatalf("23 hits: want 20 listed + 1 summary = 21, got %d", len(got))
	}
	if !strings.Contains(got[20].Detail, "3 more") {
		t.Fatalf("summary detail %q lacks %q", got[20].Detail, "3 more")
	}
}

func TestRulings_OutputIsSorted(t *testing.T) {
	f := newRulingsFixture(t)
	f.ledger(rowResolved1, "")
	f.write("design_docs/a.md", "# C\n")
	f.write("design_docs/b.md", "# C\n")
	// Passed in b, a order: each unlinked, so each yields one link-missing finding.
	missions := []*Mission{f.inRepoMission("b"), f.inRepoMission("a")}
	first := CheckRulings(missions)
	second := CheckRulings(missions)
	if len(first) != 2 || first[0].Mission != "a" || first[1].Mission != "b" {
		t.Fatalf("want a before b, got %+v", first)
	}
	if !reflect.DeepEqual(first, second) {
		t.Fatalf("nondeterministic output:\n%+v\n%+v", first, second)
	}
}
