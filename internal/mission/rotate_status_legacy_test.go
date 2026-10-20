package mission

import (
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"testing"
)

// worldStatusFixture is generated from ailang-world origin/dev 17c28324.
// The pattern retains the date, qualifier and numbered discriminator, cutting at
// a pattern rather than a UTF-8 byte offset. Exact generator command (from repo root):
//
//	set -o pipefail
//	git -C /Users/voightkampff/dev/sunholo-data/ailang-world show 17c28324:design_docs/world-mission-status-archive.md | grep '^## ' | python3 -c 'import re, sys; pattern = re.compile(r"^(## STATUS\s+\d{4}-\d{2}-\d{2}(?:\s*\([^)]*\))?\s+—\s+)(ITERATION\s+\d+(?:\s+[A-Za-z]+)?\s*:?\s*)?"); lines = sys.stdin.read().splitlines(); matches = [pattern.match(line) for line in lines]; assert all(matches), "unrecognised source heading"; sys.stdout.writelines(m.group(0) + "HEADLINE\n" for m in matches)' > internal/mission/testdata/world-status-archive-heading-prefixes.txt
func worldStatusFixture(t *testing.T) []string {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join("testdata", "world-status-archive-heading-prefixes.txt"))
	if err != nil {
		t.Fatalf("INSTRUMENT FAILURE: world STATUS headings: %v", err)
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
	if len(headings) != 281 {
		t.Fatalf("INSTRUMENT FAILURE: world STATUS got %d non-empty headings, want 281", len(headings))
	}
	return headings
}

func TestRotateStatusLegacyParserFields(t *testing.T) {
	for _, tc := range []struct {
		heading, date string
		num           int
	}{
		{"## STATUS 2026-09-03 (iteration 152) — **h**", "2026-09-03", 152},
		{"## STATUS 2026-07-24 (M1 GO, attended) — **h**", "2026-07-24", -1},
	} {
		t.Run(tc.heading, func(t *testing.T) {
			_, entries := parseLog(tc.heading + "\nbody\n")
			if len(entries) != 1 {
				t.Fatalf("STATUS entries = %d, want 1", len(entries))
			}
			e := entries[0]
			if e.Num != tc.num || e.Date != tc.date || e.Title != "**h**" {
				t.Fatalf("STATUS Num/Date/Title = %d/%s/%q, want %d/%s/**h**", e.Num, e.Date, e.Title, tc.num, tc.date)
			}
		})
	}
}

func TestRotateWorldStatusLegacyFixture(t *testing.T) {
	headings := worldStatusFixture(t)
	var body strings.Builder
	for i, heading := range headings {
		fmt.Fprintf(&body, "%s\nworld body %d\n\n", heading, i)
	}
	pre, entries := parseLog(body.String())
	if len(entries) != 281 {
		t.Fatalf("world STATUS entries = %d, want 281", len(entries))
	}
	var numbers []int
	parenCount := 0
	for i, e := range entries {
		if statusParenIterRe.MatchString(headings[i]) {
			parenCount++
			if e.Num < 0 {
				t.Fatalf("world paren-form heading parsed as note: %s", headings[i])
			}
		}
		if e.Num >= 0 {
			numbers = append(numbers, e.Num)
		}
	}
	if parenCount != 100 {
		t.Fatalf("world paren-form count = %d, want 100", parenCount)
	}
	if pre != "" || len(strayHeadings(body.String(), entries)) != 0 || ignoredIndexHeadings(body.String()) != 0 {
		t.Fatal("world fixture has preamble, stray or ignored headings")
	}
	sort.Ints(numbers)
	want := []int{267, 268, 269, 270, 271, 272, 273, 274, 275, 276, 277, 278, 279, 280, 281, 282, 283, 284, 285, 286}
	if !reflect.DeepEqual(numbers[len(numbers)-20:], want) {
		t.Fatalf("world 20 highest iteration numbers = %v, want %v", numbers[len(numbers)-20:], want)
	}
	live := filepath.Join(t.TempDir(), "world-mission-status-archive.md")
	indexWrite(t, live, body.String())
	res, err := RotateLog(live, 20)
	if err != nil {
		t.Fatal(err)
	}
	if res.Total != 281 || res.Archived != 261 || res.Kept != 20 {
		t.Fatalf("world total/archive/keep = %d/%d/%d, want 281/261/20", res.Total, res.Archived, res.Kept)
	}
	_, kept := parseLog(indexRead(t, live))
	_, archived := parseLog(indexRead(t, res.ArchivePath))
	var got []int
	keptBodies := map[string]bool{}
	for _, e := range kept {
		got = append(got, e.Num)
		keptBodies[strings.TrimSpace(e.Body)] = true
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("world kept iterations = %v, want the 20 highest %v", got, want)
	}
	archivedBodies := map[string]bool{}
	for _, e := range archived {
		archivedBodies[strings.TrimSpace(e.Body)] = true
	}
	if len(archived) != 261 || len(kept)+len(archived) != 281 {
		t.Fatalf("world output entries kept/archive = %d/%d, want 20/261", len(kept), len(archived))
	}
	for _, e := range entries {
		entryBody := strings.TrimSpace(e.Body)
		if keptBodies[entryBody] {
			if archivedBodies[entryBody] {
				t.Errorf("world kept body also archived: %s", legacyHeadings([]LogEntry{e})[0])
			}
		} else if !archivedBodies[entryBody] {
			t.Errorf("world archived entry body missing: %s", legacyHeadings([]LogEntry{e})[0])
		}
	}
}
