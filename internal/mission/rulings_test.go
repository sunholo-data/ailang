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

// M-FLEET-RULINGS-LEDGER M4: scope behaviour of CheckRulings, end to end, and the DoctorWith wiring.

// kindCount counts the findings of one kind.
func kindCount(fs []Finding, kind string) int {
	n := 0
	for _, f := range fs {
		if f.Kind == kind {
			n++
		}
	}
	return n
}

// linkOnlyInBlock is a charter whose only link sits inside its decision-ledger block.
const linkOnlyInBlock = "# C\n\n<!-- decision-ledger:start -->\n> **Fleet-wide rulings:** [fleet-rulings.md](fleet-rulings.md).\n<!-- decision-ledger:end -->\n"

func TestRulings_LinkMissing(t *testing.T) {
	f := newRulingsFixture(t)
	f.ledger(rowResolved1, "")
	m := f.inRepoMission("a")

	f.write("design_docs/a.md", "# C\n\nno link here\n")
	got := CheckRulings([]*Mission{m})
	if kindCount(got, "ruling-link-missing") != 1 {
		t.Fatalf("no link: want one ruling-link-missing, got %+v", got)
	}
	for _, g := range got {
		if g.Kind == "ruling-link-missing" && g.Mission != "a" {
			t.Fatalf("ruling-link-missing must name mission a, got %+v", g)
		}
	}

	f.write("design_docs/a.md", linkOnlyInBlock)
	got = CheckRulings([]*Mission{m})
	if kindCount(got, "ruling-link-missing") != 1 {
		t.Fatalf("link only inside the ledger block: want still one ruling-link-missing, got %+v", got)
	}

	f.write("design_docs/a.md", linkedCharter)
	got = CheckRulings([]*Mission{m})
	if kindCount(got, "ruling-link-missing") != 0 {
		t.Fatalf("link outside the block: want none, got %+v", got)
	}
}

func TestRulings_PerMissionScope(t *testing.T) {
	f := newRulingsFixture(t)
	const rows = rowResolved1 + "\n" + rowResolved2 + "\n| D-FR-3 | RESOLVED | ruling three | evidence three |"
	f.ledger(rows, strings.Join([]string{
		`D-FR-1 forbid charters "` + rulingLiteral + `"`,
		`D-FR-2 forbid registry "` + rulingLiteral + `"`,
		`D-FR-3 forbid skills "` + rulingLiteral + `"`,
	}, "\n"))
	wdA, wdB := filepath.Join(f.root, "wd", "a"), filepath.Join(f.root, "wd", "b")
	a := f.mission("a", sharedDriverRepo, wdA, "design_docs/a.md", "# "+rulingLiteral)
	b := f.mission("b", sharedDriverRepo, wdB, "design_docs/b.md", "# "+rulingLiteral)
	f.write("design_docs/a.md", linkedCharter+rulingLiteral+"\n")
	f.write("design_docs/b.md", linkedCharter+rulingLiteral+"\n")
	f.write(".claude/skills/mission-a/resources/g.md", rulingLiteral+"\n")

	owners := func(fs []Finding) map[string]int {
		out := map[string]int{}
		for _, c := range contradictions(fs) {
			out[c.Mission]++
		}
		return out
	}

	// Only a: a's charter and a's TOML carry hits; b's do not; the skills hit is still reported.
	got := owners(CheckRulings([]*Mission{a}))
	if got["a"] != 2 || got["b"] != 0 {
		t.Fatalf("slice {a}: want a=2 (charter+toml) b=0, got %v", got)
	}
	if got[rulingsMission] != 1 {
		t.Fatalf("slice {a}: want the skills hit (1), got %v", got)
	}

	// Both: each mission's charter and TOML report.
	got = owners(CheckRulings([]*Mission{a, b}))
	if got["a"] != 2 || got["b"] != 2 || got[rulingsMission] != 1 {
		t.Fatalf("slice {a,b}: want a=2 b=2 skills=1, got %v", got)
	}
}

func TestRulings_RootDerivationEndToEnd(t *testing.T) {
	if got := CheckRulings(nil); len(got) != 1 || got[0].Kind != "rulings-skipped" {
		t.Fatalf("nil missions: want exactly one rulings-skipped, got %+v", got)
	}

	hand := &Mission{Name: "a"}
	if got := CheckRulings([]*Mission{hand}); len(got) != 1 || got[0].Kind != "rulings-skipped" {
		t.Fatalf("hand-built mission: want exactly one rulings-skipped, got %+v", got)
	}

	f1, f2 := newRulingsFixture(t), newRulingsFixture(t)
	m1 := f1.mission("a", sharedDriverRepo, filepath.Join(f1.root, "wd", "a"), "doc.md", "")
	m2 := f2.mission("b", sharedDriverRepo, filepath.Join(f2.root, "wd", "b"), "doc.md", "")
	got := CheckRulings([]*Mission{m1, m2})
	if len(got) != 1 || got[0].Kind != "rulings-skipped" {
		t.Fatalf("two roots: want exactly one rulings-skipped, got %+v", got)
	}
	if !strings.Contains(got[0].Detail, f1.root) || !strings.Contains(got[0].Detail, f2.root) {
		t.Fatalf("two roots: detail must name both roots: %q", got[0].Detail)
	}

	// A one-mission slice from a loaded mission runs the check.
	f := newRulingsFixture(t)
	f.ledger(rowResolved1, `D-FR-1 forbid skills "`+rulingLiteral+`"`)
	f.write(".claude/skills/mission-a/resources/g.md", rulingLiteral+"\n")
	m := f.mission("a", sharedDriverRepo, filepath.Join(f.root, "wd", "a"), "design_docs/a.md", "")
	f.write("design_docs/a.md", linkedCharter)
	got = CheckRulings([]*Mission{m})
	if kindCount(got, "rulings-skipped") != 0 || len(contradictions(got)) < 1 {
		t.Fatalf("loaded one-mission slice: want the check to run and hit, got %+v", got)
	}
}

func TestRulings_CharterLocationEndToEnd(t *testing.T) {
	const detector = `D-FR-1 forbid charters "` + rulingLiteral + `"`

	// In-repo mission: the charter is read from the registry root, never the workdir.
	f := newRulingsFixture(t)
	f.ledger(rowResolved1, detector)
	wd := filepath.Join(f.root, "wd", "a")
	m := f.mission("a", sharedDriverRepo, wd, "design_docs/a.md", "")
	f.write("design_docs/a.md", linkedCharter+rulingLiteral+"\n")
	f.write(filepath.Join("wd", "a", "design_docs", "a.md"), linkedCharter)
	got := contradictions(CheckRulings([]*Mission{m}))
	if len(got) != 1 || !strings.Contains(got[0].Detail, "design_docs/a.md:") || strings.Contains(got[0].Detail, "/wd/") {
		t.Fatalf("root copy hits: want one hit under the root, got %+v", got)
	}

	// Swapped: the root copy is clean, the workdir copy has the literal. Zero hits.
	f2 := newRulingsFixture(t)
	f2.ledger(rowResolved1, detector)
	m2 := f2.mission("a", sharedDriverRepo, filepath.Join(f2.root, "wd", "a"), "design_docs/a.md", "")
	f2.write("design_docs/a.md", linkedCharter)
	f2.write(filepath.Join("wd", "a", "design_docs", "a.md"), linkedCharter+rulingLiteral+"\n")
	if got := contradictions(CheckRulings([]*Mission{m2})); len(got) != 0 {
		t.Fatalf("workdir-only hit must not count for an in-repo mission, got %+v", got)
	}

	// Foreign mission: the charter is read from its workdir; the root copy is ignored.
	f3 := newRulingsFixture(t)
	f3.ledger(rowResolved1, detector)
	m3 := f3.mission("x", "other/repo", filepath.Join(f3.root, "wd", "x"), "design_docs/x.md", "")
	f3.write("design_docs/x.md", linkedCharter)
	f3.write(filepath.Join("wd", "x", "design_docs", "x.md"), linkedCharter+rulingLiteral+"\n")
	if got := contradictions(CheckRulings([]*Mission{m3})); len(got) != 1 {
		t.Fatalf("foreign mission with workdir hit: want 1, got %+v", got)
	}

	// Foreign mission with no charter in its workdir: unreadable, attributed to that mission.
	f4 := newRulingsFixture(t)
	f4.ledger(rowResolved1, detector)
	m4 := f4.mission("y", "other/repo", filepath.Join(f4.root, "wd", "y"), "design_docs/y.md", "")
	f4.write("design_docs/y.md", linkedCharter+rulingLiteral+"\n")
	got = CheckRulings([]*Mission{m4})
	if kindCount(got, "ruling-file-unreadable") != 1 {
		t.Fatalf("missing foreign charter: want one ruling-file-unreadable, got %+v", got)
	}
	for _, g := range got {
		if g.Kind == "ruling-file-unreadable" && g.Mission != "y" {
			t.Fatalf("ruling-file-unreadable must name mission y, got %+v", g)
		}
	}
}

func TestDoctor_RulingFindingsPrecedeMissionFindings(t *testing.T) {
	f := newFleet(t)
	installed := "MISSION_NAME=docs\nMISSION_REPO=sunholo-data/ailang\nMISSION_DOC=d.md\n" +
		`MISSION_PLANNER_ALLOWLIST="tools/*|docs/*"` + "\n"
	m := f.addMission("docs", sharedDriverRepo, Schedule{Mode: ModeInterval, IntervalSeconds: 21600}, installed, "", true)
	m.Workdir = filepath.Join(f.root, "repos", "docs-moved")
	if err := os.MkdirAll(filepath.Join(m.Workdir, "tools", "launchd"), 0o750); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(m.Workdir, "tools", "launchd", "mission-control.sh"),
		[]byte("#!/bin/bash\n"+pinSentinel+"\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	rep := f.run()
	if len(rep.Findings) < 2 {
		t.Fatalf("want ruling note plus mission findings, got %+v", findingStrings(rep))
	}
	if rep.Findings[0].Kind != "rulings-skipped" {
		t.Fatalf("ruling notes must PRECEDE mission findings (P8b): first is %+v", rep.Findings[0])
	}
	if last := rep.Findings[len(rep.Findings)-1]; last.Severity != Drift {
		t.Fatalf("last finding must stay the drift reason (lane-check reads it): %+v", last)
	}
	if rep.ExitCode() != 1 {
		t.Fatalf("drift must still fail the run: ExitCode=%d", rep.ExitCode())
	}
}

func TestDoctor_RulingCheckRunsForLoadedMission(t *testing.T) {
	f := newFleet(t)
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, "missions"), 0o750); err != nil {
		t.Fatal(err)
	}
	body := "name = \"loaded\"\nrepo = \"" + sharedDriverRepo + "\"\nworkdir = \"" + dir + "\"\ndoc = \"d.md\"\n" +
		"[schedule]\nmode = \"keepalive\"\nthrottle_seconds = 60\nboot_offset = 60\n"
	path := filepath.Join(dir, "missions", "loaded.toml")
	if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
	m, err := LoadFile(path)
	if err != nil {
		t.Fatalf("LoadFile: %v", err)
	}
	f.reg.Missions = append(f.reg.Missions, m)

	rep := DoctorWith(f.reg, f.p, nil)
	if got := findingsOfKind(rep, "rulings-ledger-absent"); len(got) != 1 {
		t.Fatalf("loaded mission: want exactly one rulings-ledger-absent, got %+v", findingStrings(rep))
	}
	if got := findingsOfKind(rep, "rulings-skipped"); len(got) != 0 {
		t.Fatalf("loaded mission must have a root: unexpected rulings-skipped %+v", got)
	}
}
