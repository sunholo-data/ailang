package mission

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// Every variant counted across the live fleet on 2026-09-06, with the count that made the
// majority shape canonical. These are copied verbatim, not invented.
func TestNormalizeHeading_ConvertsEveryLiveVariant(t *testing.T) {
	cases := []struct {
		name, in, want string
		changed        bool
	}{
		{"canonical is left alone (368 of these)",
			"## 317 — 2026-09-02 — dev was red, V1 owns the repo [ADMIN]",
			"## 317 — 2026-09-02 — dev was red, V1 owns the repo [ADMIN]", false},
		{"world's wordy form (160)",
			"## Iteration 141 — 2026-08-31 — row 51 LANDED: 7 of 9 arms",
			"## 141 — 2026-08-31 — row 51 LANDED: 7 of 9 arms", true},
		{"docs' ISO-time form (12)",
			"## ITERATION 0 — 2026-08-28T06:41Z (first unattended fire)",
			"## 0 — 2026-08-28 — first unattended fire", true},
		{"canonical status stamp is left alone",
			"## STATUS 2026-09-06 — ITERATION 333: **M4/4 landed**",
			"## STATUS 2026-09-06 — ITERATION 333: **M4/4 landed**", false},
		{"motoko's COMPLETE variant keeps the word",
			"## STATUS 2026-09-02 — ITERATION 33 COMPLETE: **the judge's finding**",
			"## STATUS 2026-09-02 — ITERATION 33: COMPLETE: **the judge's finding**", true},
		{"v1's early parenthetical keeps the qualifier",
			"## STATUS 2026-07-14 (midday) — ITERATION 29: m-dx-examples LANDED",
			"## STATUS 2026-07-14 — ITERATION 29: (midday) m-dx-examples LANDED", true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, changed, ok := normalizeHeading(tc.in)
			if !ok {
				t.Fatalf("refused to convert a known-live variant: %q", tc.in)
			}
			if changed != tc.changed {
				t.Errorf("changed = %v, want %v", changed, tc.changed)
			}
			if got != tc.want {
				t.Errorf("\n  in   %s\n  got  %s\n  want %s", tc.in, got, tc.want)
			}
		})
	}
}

// Information in a variant must survive the rewrite. A "(midday)" qualifier and a
// "COMPLETE" marker are real content; dropping them to make the shape fit would be the
// tidy-looking kind of data loss.
func TestNormalizeHeading_NeverDropsInformation(t *testing.T) {
	for _, in := range []string{
		"## STATUS 2026-07-14 (midday) — ITERATION 29: headline here",
		"## STATUS 2026-09-02 — ITERATION 33 COMPLETE: headline here",
		"## ITERATION 0 — 2026-08-28T06:41Z (first unattended fire)",
	} {
		got, _, ok := normalizeHeading(in)
		if !ok {
			t.Fatalf("refused %q", in)
		}
		for _, token := range []string{"midday", "COMPLETE", "first unattended fire"} {
			if strings.Contains(in, token) && !strings.Contains(got, token) {
				t.Errorf("rewriting %q dropped %q -> %s", in, token, got)
			}
		}
	}
}

// A heading it cannot convert confidently is REPORTED, never guessed at. The live example
// is "## ITERATION 3 — died mid-flight, credited retroactively by iteration 4 (2026-09-02)"
// — the date is not in the date position, so any rewrite would be an invention.
func TestNormalize_ReportsWhatItCannotConvertRatherThanGuessing(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "m-mission-log.md")
	body := "# Log\n\n## ITERATION 3 — died mid-flight, credited retroactively by iteration 4 (2026-09-02)\n\nbody\n" +
		"\n## Iteration 4 — 2026-09-02 — a convertible one\n\nbody\n"
	if err := os.WriteFile(p, []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
	res, err := Normalize(p, false)
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Unhandled) != 1 {
		t.Fatalf("expected 1 unhandled heading, got %d", len(res.Unhandled))
	}
	if !strings.Contains(res.Unhandled[0].From, "died mid-flight") {
		t.Errorf("wrong heading reported: %s", res.Unhandled[0].From)
	}
	if len(res.Rewrites) != 1 {
		t.Errorf("the convertible one should still be rewritten; got %d", len(res.Rewrites))
	}
}

// A dry run must change nothing. This is the property that makes it safe to point at a
// live mission document.
func TestNormalize_DryRunTouchesNothing(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "m-mission-log.md")
	body := "# Log\n\n## Iteration 7 — 2026-09-02 — convertible\n\nbody\n"
	if err := os.WriteFile(p, []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
	res, err := Normalize(p, false)
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Rewrites) != 1 {
		t.Fatalf("expected 1 planned rewrite, got %d", len(res.Rewrites))
	}
	after, _ := os.ReadFile(p)
	if string(after) != body {
		t.Error("a DRY RUN modified the file")
	}
}

// Applying must change ONLY heading lines. Bodies are the mission's record; a normaliser
// that touched them would be rewriting history, not formatting it.
func TestNormalize_ApplyChangesOnlyHeadings(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "m-mission-log.md")
	body := "# Log\n\n## Iteration 7 — 2026-09-02 — convertible\n\n**Shipped.** a body line — with an em-dash\nanother line\n"
	if err := os.WriteFile(p, []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
	res, err := Normalize(p, true)
	if err != nil {
		t.Fatal(err)
	}
	after, _ := os.ReadFile(p)
	beforeLines, afterLines := strings.Split(body, "\n"), strings.Split(string(after), "\n")
	if len(beforeLines) != len(afterLines) {
		t.Fatalf("line count changed: %d -> %d", len(beforeLines), len(afterLines))
	}
	changed := map[int]bool{}
	for _, r := range res.Rewrites {
		changed[r.Line-1] = true
	}
	for i := range beforeLines {
		if !changed[i] && beforeLines[i] != afterLines[i] {
			t.Errorf("line %d changed but was not reported as a rewrite:\n  before %q\n  after  %q", i+1, beforeLines[i], afterLines[i])
		}
	}
	if !strings.Contains(string(after), "## 7 — 2026-09-02 — convertible") {
		t.Error("the heading was not normalised")
	}
}

// After normalising, the rotator's parser must read the file — that is the whole point of
// standardising, and it is the property that lets the parser eventually shrink.
func TestNormalize_OutputIsParseableByTheRotator(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "m-mission-log.md")
	body := "# Log\n\n## Iteration 7 — 2026-09-02 — one\n\nb1\n\n## ITERATION 8 — 2026-09-03T06:00Z (two)\n\nb2\n"
	if err := os.WriteFile(p, []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := Normalize(p, true); err != nil {
		t.Fatal(err)
	}
	after, _ := os.ReadFile(p)
	_, entries := parseLog(string(after))
	if len(entries) != 2 {
		t.Fatalf("normalised output parsed as %d entries, want 2", len(entries))
	}
	for _, e := range entries {
		if !canonicalEntryRe.MatchString(strings.Split(e.Body, "\n")[0]) {
			t.Errorf("entry %d heading is not canonical: %q", e.Num, strings.Split(e.Body, "\n")[0])
		}
	}
}

// Ten of docs' twelve entries carry only a number and a timestamp. Refusing them would
// have left that mission permanently non-canonical; inventing a title would be worse. The
// SHAPE is what is standardised — the prose stays the author's.
func TestNormalizeHeading_TitlelessEntriesAreCanonicalToo(t *testing.T) {
	got, changed, ok := normalizeHeading("## ITERATION 1 — 2026-08-28T07:26Z")
	if !ok || !changed {
		t.Fatalf("a title-less entry must normalise; ok=%v changed=%v", ok, changed)
	}
	if got != "## 1 — 2026-08-28" {
		t.Errorf("got %q, want %q", got, "## 1 — 2026-08-28")
	}
	if strings.HasSuffix(got, "—") || strings.HasSuffix(got, "— ") {
		t.Error("a dangling separator is tidier-looking and less honest")
	}
	// And the result must be recognised as canonical, or it would be rewritten forever.
	if !canonicalEntryRe.MatchString(got) {
		t.Error("the normaliser's own output is not canonical — it would rewrite on every run")
	}
	// Idempotent.
	again, changed2, _ := normalizeHeading(got)
	if changed2 || again != got {
		t.Errorf("not idempotent: %q -> %q", got, again)
	}
}
