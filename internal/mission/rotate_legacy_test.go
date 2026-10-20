package mission

import (
	"crypto/sha256"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

// Generated from stapledons-godot origin/main 50d8d5ba; never replace this
// instrument with a hand-written approximation of the legacy headings.
func legacyFixture(t *testing.T) []string {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join("testdata", "stapledon-log-headings.txt"))
	if err != nil {
		t.Fatalf("INSTRUMENT FAILURE: stapledon headings: %v", err)
	}
	var headings []string
	for _, line := range strings.Split(string(raw), "\n") {
		line = strings.TrimSuffix(line, "\r")
		if line == "" {
			continue
		}
		if !strings.HasPrefix(line, "## ") {
			t.Fatalf("INSTRUMENT FAILURE: non-heading fixture line %q", line)
		}
		headings = append(headings, line)
	}
	if len(headings) != 41 {
		t.Fatalf("INSTRUMENT FAILURE: got %d non-empty headings, want 41", len(headings))
	}
	return headings
}

func legacyLog(headings []string) string {
	return "# Log\n\n" + strings.Join(headings, "\nbody\n\n") + "\nbody\n"
}

func legacyHeadings(entries []LogEntry) []string {
	var headings []string
	for _, e := range entries {
		headings = append(headings, strings.SplitN(e.Body, "\n", 2)[0])
	}
	return headings
}

func TestRotateLegacyFixtureParse(t *testing.T) {
	headings := legacyFixture(t)
	body := legacyLog(headings)
	pre, entries := parseLog(body)
	if len(entries) != 41 {
		t.Fatalf("fixture entries = %d, want 41 (38 numbered + 3 notes)", len(entries))
	}
	var notes int
	for _, e := range entries {
		if e.Num == -1 {
			notes++
		}
	}
	if notes != 3 {
		t.Errorf("fixture notes = %d, want 3", notes)
	}
	if strings.Contains(pre, "## ") || len(strayHeadings(body, entries)) != 0 || ignoredIndexHeadings(body) != 0 {
		t.Fatal("fixture has preamble record headings, stray or ignored headings")
	}
	if !reflect.DeepEqual(legacyHeadings(entries), headings) {
		t.Fatal("fixture parser changed heading sequence")
	}
	if entries[0].Num != -1 || entries[0].Date != "2026-09-27" || entries[0].Title != "pre-iteration 0 (attended bootstrap)" ||
		entries[2].Num != 0 || entries[2].Title != "(night) M1.1 photometry package LANDED [PRODUCT]" ||
		entries[4].Date != "2026-09-28" || entries[4].Title != "(afternoon) M1.2a catalogue acquire and parse LANDED [PRODUCT]" {
		t.Fatalf("fixture fields: first=%+v zero=%+v afternoon=%+v", entries[0], entries[2], entries[4])
	}
}

func TestRotateLegacyFixtureSplitOrder(t *testing.T) {
	headings := legacyFixture(t)
	live, archive, index := fixturePaths(t, "log")
	indexWrite(t, live, legacyLog(headings))
	res, err := RotateLog(live, 20)
	if err != nil {
		t.Fatal(err)
	}
	if res.Total != 41 || res.Archived != 21 || res.Kept != 20 {
		t.Fatalf("fixture split total/archive/keep = %d/%d/%d, want 41/21/20", res.Total, res.Archived, res.Kept)
	}
	_, old := parseLog(indexRead(t, archive))
	_, kept := parseLog(indexRead(t, live))
	if !reflect.DeepEqual(legacyHeadings(old), headings[:21]) {
		t.Fatalf("archive heading order differs: got %v, want %v (October 2 note must stay between 8 and 9)", legacyHeadings(old), headings[:21])
	}
	if !reflect.DeepEqual(legacyHeadings(kept), headings[21:]) {
		t.Fatal("kept heading order differs")
	}
	want := []int{0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 11, 12, 14, 16, 17, 18, 19, 20, 21, 22, 23, 24, 26, 27, 28, 29, 30, 31, 32, 33, 34, 35, 36, 37, 38, 39, 40, 41}
	idx := indexRead(t, index)
	if !reflect.DeepEqual(indexNumbers(idx), want) || res.IndexEntries != 41 || len(res.OrphanNumbers) != 0 {
		t.Fatalf("fixture index numbers = %v; metadata %+v", indexNumbers(idx), res)
	}
	for _, note := range []string{"pre-iteration 0 (attended bootstrap)", "(late) armed, attended", "attended, M2 journey core LANDED (sprint R1-M2-JOURNEY) [PRODUCT]"} {
		if !strings.Contains(idx, "| "+note+" |") {
			t.Errorf("fixture index missing note %q", note)
		}
	}
}

func TestRotateLegacyParserFields(t *testing.T) {
	for _, tc := range []struct {
		heading string
		num     int
		title   string
	}{
		{"## 2026-09-27: pre-iteration 0 (attended bootstrap)", -1, "pre-iteration 0 (attended bootstrap)"},
		{"## 2026-09-27 (late): armed, attended", -1, "(late) armed, attended"},
		{"## 2026-09-27 (night): iteration 0, boot", 0, "(night) boot"},
		{"## 2026-09-27 (afternoon): ITERATION 2: title", 2, "(afternoon) title"},
		{"## 2026-09-27: iteration 3 title", 3, "title"},
	} {
		t.Run(tc.title, func(t *testing.T) {
			pre, es := parseLog("# Log\n" + tc.heading + "\nbody\n" + numbered(4, "next"))
			if len(es) != 2 {
				t.Fatalf("parser entries = %d, want 2, including legacy record before first canonical number", len(es))
			}
			if es[0].Num != tc.num || es[0].Date != "2026-09-27" || es[0].Title != tc.title {
				t.Fatalf("legacy fields Num/Date/Title = %d/%s/%q, want %d/2026-09-27/%q", es[0].Num, es[0].Date, es[0].Title, tc.num, tc.title)
			}
			if strings.Contains(pre, tc.heading) || es[0].Body != tc.heading+"\nbody" {
				t.Fatal("legacy record body or preamble incorrect")
			}
		})
	}
}

func TestRotateLegacyNoArchive(t *testing.T) {
	for _, keep := range []int{2, 3} {
		for _, withArchive := range []bool{false, true} {
			t.Run(fmt.Sprintf("keep=%d/archive=%t", keep, withArchive), func(t *testing.T) {
				live, archive, index := fixturePaths(t, "log")
				body := "# Log\n\n" + legacyNotice + "\n" + numbered(1, "first") + "\n\n" + numbered(2, "next") + "\n\n"
				indexWrite(t, live, body)
				oldArchive := "# Archive\n" + numbered(0, "older")
				if withArchive {
					indexWrite(t, archive, oldArchive)
				}
				res, err := RotateLog(live, keep)
				if err != nil {
					t.Fatal(err)
				}
				if sha256.Sum256([]byte(indexRead(t, live))) != sha256.Sum256([]byte(body)) {
					t.Error("no-archive live SHA256 changed")
				}
				if res.Archived != 0 || res.LogAfter != res.LogBefore || res.LogBefore != len(body) {
					t.Errorf("no-archive metadata: %+v", res)
				}
				if !strings.Contains(indexRead(t, index), "| 2 |") {
					t.Error("no-archive index not written")
				}
				if withArchive {
					if indexRead(t, archive) != oldArchive {
						t.Error("no-archive existing archive changed")
					}
				} else if _, err := os.Stat(archive); !os.IsNotExist(err) {
					t.Errorf("no-archive created archive: %v", err)
				}
			})
		}
	}
}

const legacyNotice = "> **Older entries are ARCHIVED.** This file holds the newest 20. The full record of every\n> iteration is in `test-mission-log-archive.md`, and a one-line index of ALL of them —\n> the thing to grep before picking work, so the loop never repeats itself — is in\n> `test-mission-index.md`.\n"

func TestRotateLegacyNotices(t *testing.T) {
	for _, copies := range []int{1, 3} {
		t.Run(fmt.Sprintf("copies=%d", copies), func(t *testing.T) {
			live, _, _ := fixturePaths(t, "log")
			// Prose includes a different blockquote with a similar phrase: keep it.
			before := "# Log\n\n> Older entries are ARCHIVED by hand.\n\nleft prose\n"
			after := "right prose\n\n"
			indexWrite(t, live, before+strings.Repeat(legacyNotice, copies)+after+numbered(1, "old")+numbered(2, "new"))
			for pass := 0; pass < 2; pass++ {
				res, err := RotateLog(live, 1)
				if err != nil {
					t.Fatal(err)
				}
				got := indexRead(t, live)
				if res.Archived != 1 || strings.Count(got, "> **Older entries are ARCHIVED.**") != 1 {
					t.Fatalf("notice count = %d, want 1 after real rotation %d", strings.Count(got, "> **Older entries are ARCHIVED.**"), pass+1)
				}
				if !strings.Contains(got, before+after) {
					t.Fatal("notice stripping changed adjacent prose")
				}
				if pass == 0 {
					indexWrite(t, live, got+numbered(3, "appended"))
				}
			}
		})
	}
}

func TestRotateLegacyPreambleGuard(t *testing.T) {
	for _, bad := range []string{
		"## 2026-10-01 — iteration 9 the dash form",
		"## 2026-10-01 broken dated record",
		"## 123 — broken numbered record",
		"## ITERATION 9 — missing date",
	} {
		for _, withEntry := range []bool{false, true} {
			t.Run(fmt.Sprintf("%s/entry=%t", bad, withEntry), func(t *testing.T) {
				live, archive, index := fixturePaths(t, "log")
				body := "# Log\n" + bad + "\nbody\n## 999 second offender\n"
				if withEntry {
					body += numbered(1000, "last")
				}
				indexWrite(t, live, body)
				indexWrite(t, archive, numbered(0, "archived"))
				indexWrite(t, index, tableHeader+"| 0 | old | preserved |\n")
				before := map[string]string{}
				for _, p := range []string{live, archive, index} {
					before[p] = indexRead(t, p)
				}
				_, err := RotateLog(live, 1)
				if err == nil || !strings.Contains(err.Error(), bad) {
					t.Errorf("preamble refusal must name first offending heading %q: %v", bad, err)
				}
				for p, want := range before {
					if indexRead(t, p) != want {
						t.Errorf("preamble refusal changed %s", filepath.Base(p))
					}
				}
				if _, err := RegenerateIndex(live); err != nil {
					t.Fatalf("index-only must remain permissive: %v", err)
				}
			})
		}
	}
}

func TestRotateLegacyAllowedPreamble(t *testing.T) {
	allowed := []string{
		"## How to read this file",
		"## N — YYYY-MM-DD — <headline>",
		"## (Charter sections — moved back 2026-10-01)",
		"## Archived 2026-09-30: fork-era charter (...)",
	}
	live, _, _ := fixturePaths(t, "log")
	pre := strings.Join(allowed, "\n") + "\n"
	indexWrite(t, live, pre+numbered(1, "old")+numbered(2, "new"))
	if _, err := RotateLog(live, 1); err != nil {
		t.Fatalf("allowed preamble refused: %v", err)
	}
	if !strings.Contains(indexRead(t, live), pre) {
		t.Fatal("allowed preamble changed")
	}
}
