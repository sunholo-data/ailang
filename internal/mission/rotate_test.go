package mission

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"unicode/utf8"
)

func sampleLog(n int) string {
	var b strings.Builder
	b.WriteString("# V1 Mission Log\n\nPreamble that must survive rotation.\n")
	for i := 1; i <= n; i++ {
		fmt.Fprintf(&b, "\n## %d — 2026-09-%02d — Did thing number %d [TAG%d]\n\n**Shipped.** body line for %d\nmore body\n", i, (i%28)+1, i, i%3, i)
	}
	return b.String()
}

func TestRotateLog_KeepsNewestArchivesRestIndexesAll(t *testing.T) {
	dir := t.TempDir()
	logPath := filepath.Join(dir, "v1-mission-log.md")
	if err := os.WriteFile(logPath, []byte(sampleLog(50)), 0o600); err != nil {
		t.Fatal(err)
	}
	res, err := RotateLog(logPath, 10)
	if err != nil {
		t.Fatalf("RotateLog: %v", err)
	}
	if res.Kept != 10 || res.Archived != 40 || res.Total != 50 {
		t.Errorf("kept/archived/total = %d/%d/%d, want 10/40/50", res.Kept, res.Archived, res.Total)
	}

	live, _ := os.ReadFile(logPath)
	ls := string(live)
	if !strings.Contains(ls, "Preamble that must survive rotation.") {
		t.Error("the preamble was dropped")
	}
	if !strings.Contains(ls, "## 50 —") || !strings.Contains(ls, "## 41 —") {
		t.Error("the newest 10 entries must remain in the live log")
	}
	if strings.Contains(ls, "## 40 —") || strings.Contains(ls, "## 1 —") {
		t.Error("older entries must be gone from the live log")
	}
	if res.LogAfter >= res.LogBefore {
		t.Errorf("live log did not shrink: %d -> %d", res.LogBefore, res.LogAfter)
	}

	// The archive holds the full body of what was rotated out — not just its heading.
	arch, err := os.ReadFile(res.ArchivePath)
	if err != nil {
		t.Fatalf("archive: %v", err)
	}
	if !strings.Contains(string(arch), "body line for 1") {
		t.Error("the archive must hold the FULL entry, or rotation is deletion")
	}

	// THE POINT: the index covers EVERY iteration, archived and live.
	idx, err := os.ReadFile(res.IndexPath)
	if err != nil {
		t.Fatalf("index: %v", err)
	}
	is := string(idx)
	for _, n := range []int{1, 25, 40, 41, 50} {
		if !strings.Contains(is, fmt.Sprintf("| %d |", n)) {
			t.Errorf("index is missing iteration %d — an incomplete index cannot stop repeated work", n)
		}
	}
	if res.IndexEntries != 50 {
		t.Errorf("IndexEntries = %d, want 50", res.IndexEntries)
	}
	// And it must stay small enough to actually read.
	if len(idx) > 50*250 {
		t.Errorf("index is %d B for 50 entries — too fat to read whole, which is its only job", len(idx))
	}
}

// Rotating twice must not lose the first rotation's entries from the index, and must not
// duplicate them. This is the case an append-only index gets wrong.
func TestRotateLog_IsIdempotentAndCumulative(t *testing.T) {
	dir := t.TempDir()
	logPath := filepath.Join(dir, "v1-mission-log.md")
	if err := os.WriteFile(logPath, []byte(sampleLog(30)), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := RotateLog(logPath, 5); err != nil {
		t.Fatal(err)
	}
	// A later iteration appends to the trimmed log, then we rotate again.
	f, _ := os.OpenFile(logPath, os.O_APPEND|os.O_WRONLY, 0o600)
	fmt.Fprintf(f, "\n## 31 — 2026-09-07 — A brand new thing [NEW]\n\nbody 31\n")
	_ = f.Close()

	res, err := RotateLog(logPath, 5)
	if err != nil {
		t.Fatalf("second rotate: %v", err)
	}
	idx, _ := os.ReadFile(res.IndexPath)
	is := string(idx)
	for _, n := range []int{1, 15, 26, 31} {
		if !strings.Contains(is, fmt.Sprintf("| %d |", n)) {
			t.Errorf("iteration %d fell out of the index across two rotations", n)
		}
	}
	if c := strings.Count(is, "| 26 |"); c != 1 {
		t.Errorf("iteration 26 appears %d times — the index must be regenerated, not appended", c)
	}
	if res.IndexEntries != 31 {
		t.Errorf("IndexEntries = %d, want 31", res.IndexEntries)
	}
}

// Rotation must never be able to lose an entry: everything that leaves the live log has
// to be in the archive, and everything ever seen has to be in the index.
func TestRotateLog_NothingIsEverLost(t *testing.T) {
	dir := t.TempDir()
	logPath := filepath.Join(dir, "v1-mission-log.md")
	if err := os.WriteFile(logPath, []byte(sampleLog(40)), 0o600); err != nil {
		t.Fatal(err)
	}
	res, err := RotateLog(logPath, 8)
	if err != nil {
		t.Fatal(err)
	}
	live, _ := os.ReadFile(logPath)
	arch, _ := os.ReadFile(res.ArchivePath)
	both := string(live) + string(arch)
	for i := 1; i <= 40; i++ {
		if !strings.Contains(both, fmt.Sprintf("body line for %d\n", i)) {
			t.Fatalf("iteration %d's BODY exists in neither the live log nor the archive", i)
		}
	}
}

func TestRotateLog_RefusesNonsenseAndEmptyLogs(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "v1-mission-log.md")
	_ = os.WriteFile(p, []byte("# just a preamble, no entries\n"), 0o600)
	if _, err := RotateLog(p, 5); err == nil {
		t.Error("a log with no parseable entries must be refused, not silently emptied")
	}
	_ = os.WriteFile(p, []byte(sampleLog(3)), 0o600)
	if _, err := RotateLog(p, 0); err == nil {
		t.Error("keep=0 must be refused — it would archive everything")
	}
}

// Fewer entries than the keep threshold: a no-op for the log, but the index is still
// written, because a mission that has never rotated still needs one.
func TestRotateLog_UnderThresholdStillWritesTheIndex(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "v1-mission-log.md")
	_ = os.WriteFile(p, []byte(sampleLog(3)), 0o600)
	res, err := RotateLog(p, 20)
	if err != nil {
		t.Fatal(err)
	}
	if res.Archived != 0 || res.Kept != 3 {
		t.Errorf("kept/archived = %d/%d, want 3/0", res.Kept, res.Archived)
	}
	idx, err := os.ReadFile(res.IndexPath)
	if err != nil || !strings.Contains(string(idx), "| 3 |") {
		t.Error("the index must be written even when nothing rotated")
	}
}

// A strict heading pattern silently UNDER-INDEXED v1's real log: 335 headings, 331
// matched, and three real records — a date range, a combined two-slot entry, and an
// attended note — were left out. No bytes were lost, but an index that quietly omits
// three iterations is precisely the failure it exists to prevent: the loop looks, does
// not find, and redoes the work.
func TestRotateLog_IndexesTheAwkwardRealHeadings(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "v1-mission-log.md")
	body := `# V1 Mission Log

## N — YYYY-MM-DD — <headline>

A template in the preamble. Must NOT be indexed as an iteration.

## 5 — 2026-07-09 — An ordinary entry [OK]

body 5

## 6 — 2026-07-10/11 — A date RANGE, which strict parsing rejected [OK]

body 6

## 321 & 322 — 2026-09-02 — NO ENTRY: both slots died mid-flight [DEAD]

body 321

## ATTENDED NOTE — 2026-08-27 (Mark + interactive session; not an iteration retro)

body note
`
	if err := os.WriteFile(p, []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
	res, err := RotateLog(p, 2)
	if err != nil {
		t.Fatalf("RotateLog: %v", err)
	}
	idx, _ := os.ReadFile(res.IndexPath)
	is := string(idx)

	if !strings.Contains(is, "| 6 |") {
		t.Error("a date-RANGE entry (2026-07-10/11) must be indexed")
	}
	if !strings.Contains(is, "| 321 |") {
		t.Error("a combined '321 & 322' entry must be indexed")
	}
	if !strings.Contains(is, "321 & 322") {
		t.Error("a combined entry must SAY it covers both slots, or 322 looks unrecorded")
	}
	if !strings.Contains(is, "ATTENDED NOTE") {
		t.Error("an attended note is a real record and belongs in the index")
	}
	// ...and the preamble template must NOT masquerade as an iteration.
	if strings.Contains(is, "YYYY-MM-DD") {
		t.Error("the '## N — YYYY-MM-DD' template was indexed as a real iteration")
	}
	// Nothing lost, as always.
	live, _ := os.ReadFile(p)
	arch, _ := os.ReadFile(res.ArchivePath)
	both := string(live) + string(arch)
	for _, want := range []string{"body 5", "body 6", "body 321", "body note"} {
		if !strings.Contains(both, want) {
			t.Errorf("%q survived neither the live log nor the archive", want)
		}
	}
}

// Titles are full of em-dashes, and slicing a Go string truncates BYTES. The first real
// run produced an index that was not valid UTF-8 because a dash was cut in half — and an
// index no tool can read is worse than a long line.
func TestRotateLog_IndexIsValidUTF8WhenTitlesAreTruncated(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "v1-mission-log.md")
	long := strings.Repeat("an em—dash title that runs on ", 12) // ~360 chars, many multi-byte
	body := "# Log\n\n## 1 — 2026-09-06 — " + long + "\n\nbody\n"
	if err := os.WriteFile(p, []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
	res, err := RotateLog(p, 1)
	if err != nil {
		t.Fatal(err)
	}
	idx, _ := os.ReadFile(res.IndexPath)
	if !utf8.Valid(idx) {
		t.Fatal("the index is not valid UTF-8 — a multi-byte rune was cut in half by a byte slice")
	}
	if !strings.Contains(string(idx), "...") {
		t.Error("a long title should be truncated")
	}
}

// The fleet uses THREE heading formats. A parser that knows only v1's rotates nothing for
// docs (it refused loudly, which was right) and would mis-parse world's. All three are
// real, copied from the live logs on 2026-09-06.
func TestParseLog_HandlesEveryFormatTheFleetActuallyUses(t *testing.T) {
	cases := []struct {
		name, heading string
		wantNum       int
		wantDate      string
	}{
		{"v1", "## 337 — 2026-09-06 — Bank the pi runner evidence [HARNESS]", 337, "2026-09-06"},
		{"docs", "## ITERATION 11 — 2026-09-06T06:00Z", 11, "2026-09-06"},
		{"docs with note", "## ITERATION 0 — 2026-08-28T06:41Z (first unattended fire)", 0, "2026-08-28"},
		{"world", "## Iteration 2 — 2026-07-23 — queue HUMAN-BLOCKED on D1", 2, "2026-07-23"},
		{"v1 date range", "## 6 — 2026-07-10/11 — Iteration 5 landed", 6, "2026-07-10"},
		{"v1 combined", "## 321 & 322 — 2026-09-02 — NO ENTRY: both slots died", 321, "2026-09-02"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, entries := parseLog(tc.heading + "\n\nbody\n")
			if len(entries) != 1 {
				t.Fatalf("parsed %d entries from %q, want 1 — an unparsed heading means this "+
					"mission's log never rotates and never indexes", len(entries), tc.heading)
			}
			if entries[0].Num != tc.wantNum {
				t.Errorf("Num = %d, want %d", entries[0].Num, tc.wantNum)
			}
			if entries[0].Date != tc.wantDate {
				t.Errorf("Date = %q, want %q", entries[0].Date, tc.wantDate)
			}
		})
	}
}

// A FOURTH shape: the STATUS-stamp archive inverts the field order, putting the date
// first and the iteration number second. v1's is 1.69 MB / ~423k tokens across 295
// entries — append-only and unbounded, exactly like a log, so it rotates the same way.
func TestRotateLog_HandlesStatusStampArchives(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "v1-mission-status-archive.md")
	var b strings.Builder
	b.WriteString("# V1 Mission — STATUS stamp archive (rotated out of the charter)\n")
	for i := 1; i <= 30; i++ {
		fmt.Fprintf(&b, "\n## STATUS 2026-09-%02d — ITERATION %d: **headline %d**\n\nstatus body %d\n", (i%28)+1, i, i, i)
	}
	// All FOUR real variants, copied from the live archives 2026-09-06. A parser that
	// knows only the current one left 23 of v1's 295 entries and ALL of motoko's out.
	b.WriteString("\n## STATUS 2026-09-02 — ITERATION 91 COMPLETE: **motoko's variant**\n\nstatus body 91\n")
	b.WriteString("\n## STATUS 2026-07-14 (midday) — ITERATION 92: v1's early variant\n\nstatus body 92\n")
	b.WriteString("\n## STATUS 2026-07-12 (morning) — v1.0 SCOPE SET, no iteration number\n\nstatus body 93\n")
	if err := os.WriteFile(p, []byte(b.String()), 0o600); err != nil {
		t.Fatal(err)
	}
	res, err := RotateLog(p, 10)
	if err != nil {
		t.Fatalf("a status archive must rotate like a log: %v", err)
	}
	idxAll, _ := os.ReadFile(res.IndexPath)
	for _, want := range []string{"| 91 |", "| 92 |", "SCOPE SET"} {
		if !strings.Contains(string(idxAll), want) {
			t.Errorf("status variant %q missing from the index", want)
		}
	}
	for _, want := range []string{"status body 91", "status body 92", "status body 93"} {
		lv, _ := os.ReadFile(p)
		av, _ := os.ReadFile(res.ArchivePath)
		if !strings.Contains(string(lv)+string(av), want) {
			t.Errorf("%q survived neither file", want)
		}
	}
	if res.Total != 33 || res.Kept != 10 {
		t.Errorf("total/kept = %d/%d, want 33/10", res.Total, res.Kept)
	}
	// Its index must not collide with the LOG's index for the same mission.
	if strings.HasSuffix(res.IndexPath, "v1-mission-index.md") {
		t.Errorf("the status index collided with the log index: %s", res.IndexPath)
	}
	if !strings.Contains(res.IndexPath, "status") {
		t.Errorf("index path should name the status stream; got %s", res.IndexPath)
	}
	idx, _ := os.ReadFile(res.IndexPath)
	for _, n := range []int{1, 15, 30} {
		if !strings.Contains(string(idx), fmt.Sprintf("| %d |", n)) {
			t.Errorf("status iteration %d missing from the index", n)
		}
	}
	// Nothing lost.
	live, _ := os.ReadFile(p)
	arch, _ := os.ReadFile(res.ArchivePath)
	both := string(live) + string(arch)
	for i := 1; i <= 30; i++ {
		if !strings.Contains(both, fmt.Sprintf("status body %d\n", i)) {
			t.Fatalf("status entry %d survived neither file", i)
		}
	}
}

// Rotation assumes every "## " heading after the first entry IS an entry. A file that
// interleaves STRUCTURAL sections breaks that: the section becomes part of the preceding
// entry's body and is archived away with it.
//
// Measured on motoko's status archive, which carries eight — a Backlog, a Routing rule, a
// Skill section, "How the mission runs". Rotating it moved live reference material into an
// archive nothing loads. The tool cannot tell a record from structure, so it refuses.
func TestRotateLog_RefusesAFileWithInterleavedStructure(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "m-mission-log.md")
	body := `# Log

## 1 — 2026-09-01 — first [OK]

body 1

## Backlog (prioritized — top = next)

This is live reference material, not a record. It must not be archived.

## 2 — 2026-09-02 — second [OK]

body 2
`
	if err := os.WriteFile(p, []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
	_, err := RotateLog(p, 1)
	if err == nil {
		t.Fatal("a file with interleaved structural sections must be REFUSED — rotating it " +
			"silently relocates live reference material into an archive")
	}
	if !strings.Contains(err.Error(), "Backlog") {
		t.Errorf("the refusal must NAME the offending heading so it can be moved; got: %v", err)
	}
	// And it must have changed nothing.
	after, _ := os.ReadFile(p)
	if string(after) != body {
		t.Error("a refused rotation must leave the file untouched")
	}
}

// A heading BEFORE the first entry is preamble, not structure-in-the-stream, and must not
// trip the guard — otherwise no real log would ever rotate, since they all have a title.
func TestRotateLog_PreambleHeadingsDoNotTripTheGuard(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "m-mission-log.md")
	body := `# Log

## How to read this file

Preamble prose, above every entry.

## 1 — 2026-09-01 — first [OK]

body 1

## 2 — 2026-09-02 — second [OK]

body 2
`
	if err := os.WriteFile(p, []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := RotateLog(p, 1); err != nil {
		t.Fatalf("a preamble heading must not block rotation: %v", err)
	}
}
