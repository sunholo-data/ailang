package mission

import (
	"fmt"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestRotatePreambleRecordShapes(t *testing.T) {
	for _, tc := range []struct {
		heading string
		refuse  bool
	}{
		{"## 2026-10-01 — iteration 9 the dash form", true},
		{"## 2026-10-01 broken dated record", true},
		{"## 2026-10-01", true},
		{"## 2026-10-01 (night) broken record", true},
		{"## 12 — no date here", true},
		{"## Iteration 12 — no date here", true},
		{"## ITERATION 12 —", true},
		{"## 12 & 13 — no date here", true},
		{"## STATUS 2026-10-01 broken record", true},
		{"## STATUS 2026-10-01 (iteration 12) missing dash", true},
		{"## STATUS 2026-", true},
		{"## 1. Purpose", false},
		{"## How to run iteration 5", false},
		{"## 2026 roadmap", false},
		{"## How to read this file", false},
		{"## N — YYYY-MM-DD — <headline>", false},
		{"## (Charter sections — moved back 2026-10-01)", false},
		{"## Archived 2026-09-30: fork-era charter (x)", false},
		{"## 2. Scope (iteration 3 onward)", false},
	} {
		t.Run(tc.heading, func(t *testing.T) {
			live, archive, index := fixturePaths(t, "log")
			pre := "# Log\n\n" + tc.heading + "\nhand-written preamble\n"
			indexWrite(t, live, pre+"## 5 — 2026-01-05 — a\na body\n## 6 — 2026-01-06 — b\nb body\n")
			indexWrite(t, archive, numbered(0, "archived"))
			indexWrite(t, index, tableHeader+"| 0 | old | preserved |\n")
			before := map[string]string{}
			for _, p := range []string{live, archive, index} {
				before[p] = indexRead(t, p)
			}
			res, err := RotateLog(live, 1)
			if tc.refuse {
				if err == nil || !strings.Contains(err.Error(), tc.heading) {
					t.Errorf("record-shaped heading must be refused by name: %v", err)
				}
				for p, want := range before {
					if indexRead(t, p) != want {
						t.Errorf("refusal changed %s bytes", filepath.Base(p))
					}
				}
				return
			}
			if err != nil {
				t.Fatalf("structural preamble must rotate normally: %v", err)
			}
			if res.Archived != 1 || res.Kept != 1 || !strings.HasPrefix(indexRead(t, live), pre) {
				t.Fatalf("rotation must archive one entry and preserve preamble: %+v", res)
			}
			_, old := parseLog(indexRead(t, archive))
			_, kept := parseLog(indexRead(t, live))
			if !reflect.DeepEqual(legacyHeadings(old), []string{strings.SplitN(numbered(0, "archived"), "\n", 2)[0], "## 5 — 2026-01-05 — a"}) ||
				!reflect.DeepEqual(legacyHeadings(kept), []string{"## 6 — 2026-01-06 — b"}) {
				t.Fatal("structural preamble changed the archive/live entry split")
			}
		})
	}
}

func TestRotateDamagedArchiveNotices(t *testing.T) {
	lines := strings.Split(strings.TrimSuffix(legacyNotice, "\n"), "\n")
	for _, tc := range []struct {
		name, damaged, after string
	}{
		{"handwritten_then_real", strings.Join(lines[:2], "\n") + "\n> my own quoted prose", "\n" + legacyNotice},
		// No blank or fourth notice line before the entry: the bounds check must
		// protect the fourth-line access even when the first three prefixes match.
		{"last_three_lines", strings.Join(lines[:3], "\n"), "\n"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			live, _, _ := fixturePaths(t, "log")
			pre := "# Log\n\n" + tc.damaged
			indexWrite(t, live, pre+tc.after+numbered(1, "old")+numbered(2, "new"))
			res, err := RotateLog(live, 1)
			if err != nil {
				t.Fatal(err)
			}
			got := indexRead(t, live)
			if !strings.HasPrefix(got, pre+"\n") {
				t.Fatalf("damaged notice and handwritten prose must survive verbatim:\n%s", got)
			}
			wantNotice := strings.Replace(legacyNotice, "newest 20.", "newest 1.", 1)
			if res.Archived != 1 || strings.Count(got, wantNotice) != 1 || strings.Contains(got, legacyNotice) {
				t.Fatalf("real rotation must replace old complete notices with exactly one generated notice:\n%s", got)
			}
		})
	}
}

func TestRotateStableLargeNoteGroup(t *testing.T) {
	headings := []string{"## 1 — 2026-10-01 — anchor"}
	for i := 0; i < 30; i++ {
		headings = append(headings, fmt.Sprintf("## 2026-10-01: note %02d", i))
	}
	for i := 2; i <= 11; i++ {
		headings = append(headings, fmt.Sprintf("## %d — 2026-10-01 — numbered", i))
	}
	// A late, lower-numbered entry forces partitioning rather than letting an
	// unstable sort's already-sorted fast path hide reordering of equal keys.
	zero := "## 0 — 2026-10-01 — late zero"
	headings = append(headings, zero)
	live, archive, _ := fixturePaths(t, "log")
	indexWrite(t, live, legacyLog(headings))
	res, err := RotateLog(live, 1)
	if err != nil {
		t.Fatal(err)
	}
	_, old := parseLog(indexRead(t, archive))
	want := append([]string{zero}, headings[:len(headings)-2]...)
	if res.Total != 42 || res.Archived != 41 || !reflect.DeepEqual(legacyHeadings(old), want) {
		t.Fatalf("archive must keep exact file order within equal keys: got %v, want %v", legacyHeadings(old), want)
	}
}

func TestRotateNotesAroundIterationZero(t *testing.T) {
	headings := []string{
		"## 2026-10-01: note A",
		"## 0 — 2026-10-01 — zero",
		"## 2026-10-01: note B",
		"## 1 — 2026-10-01 — one",
		"## 2 — 2026-10-01 — two",
	}
	for _, tc := range []struct {
		name     string
		headings []string
	}{
		{"ascending", headings},
		// Ascending input alone cannot distinguish keys -1 and 0: stable sort
		// retains its order either way. A preceding positive anchor exposes the
		// mutant's failure to reset the key at iteration zero.
		{"positive_before_zero", []string{headings[0], headings[3], headings[1], headings[2], headings[4]}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			live, archive, _ := fixturePaths(t, "log")
			indexWrite(t, live, legacyLog(tc.headings))
			if _, err := RotateLog(live, 1); err != nil {
				t.Fatal(err)
			}
			_, old := parseLog(indexRead(t, archive))
			if !reflect.DeepEqual(legacyHeadings(old), headings[:4]) {
				t.Fatalf("note B must archive AFTER iteration 0 and BEFORE iteration 1: got %v, want %v", legacyHeadings(old), headings[:4])
			}
		})
	}
}

func TestParseLogLegacyIterationNumberBoundary(t *testing.T) {
	for _, tc := range []struct {
		suffix string
		num    int
		title  string
	}{
		{"12a, x", -1, "iteration 12a, x"},
		{"12, x", 12, "x"},
	} {
		t.Run(tc.suffix, func(t *testing.T) {
			heading := "## 2026-10-01: iteration " + tc.suffix
			pre, entries := parseLog(heading + "\nbody\n")
			if pre != "" || len(entries) != 1 {
				t.Fatalf("expected one legacy record, got preamble %q, entries %v", pre, entries)
			}
			if e := entries[0]; e.Num != tc.num || e.Date != "2026-10-01" || e.Title != tc.title {
				t.Fatalf("legacy number boundary: got %+v, want Num=%d Date=2026-10-01 Title=%q", e, tc.num, tc.title)
			}
		})
	}
}
