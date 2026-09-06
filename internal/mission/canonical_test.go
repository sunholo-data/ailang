package mission

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// knownNonCanonical is a RATCHET, not a target.
//
// The fleet wrote the same record five ways, and supporting all of them cost this package
// two separate rounds of "the parser silently under-indexed". 375 headings were normalised
// to one shape on 2026-09-06; these are the genuinely irregular remainder — date ranges,
// attended-ratification stamps with no iteration number, a heading whose date is not in the
// date position at all.
//
// The count may FALL, never rise. Fixing them is welcome; adding a sixth variant is not.
const knownNonCanonical = 30

// missionDocs are the record streams. Charters are excluded deliberately: they are curated
// prose, and a lint that demanded canonical headings there would be linting an essay.
func missionDocs(t *testing.T) []string {
	t.Helper()
	var out []string
	for _, dir := range []string{
		filepath.Join("..", "..", "design_docs"),
		filepath.Join("..", "..", "..", "ailang-world", "design_docs"),
	} {
		for _, pat := range []string{"*-mission-log.md", "*-mission-log-archive.md",
			"*-mission-status-archive.md", "*-mission-status-archive-old.md"} {
			m, _ := filepath.Glob(filepath.Join(dir, pat))
			out = append(out, m...)
		}
	}
	return out
}

func TestMissionDocHeadingsStayCanonical(t *testing.T) {
	docs := missionDocs(t)
	if len(docs) == 0 {
		t.Skip("no mission docs on this machine")
	}
	var offenders []string
	for _, d := range docs {
		body, err := os.ReadFile(d) //nolint:gosec // repo-relative doc
		if err != nil {
			continue
		}
		for _, l := range strings.Split(string(body), "\n") {
			if !strings.HasPrefix(l, "## ") {
				continue
			}
			if canonicalEntryRe.MatchString(l) || canonicalStatusRe.MatchString(l) {
				continue
			}
			// Only RECORDS are linted. A structural section or a preamble heading is
			// legitimate prose and must not be forced into a record shape.
			if !looksLikeRecord(l) {
				continue
			}
			offenders = append(offenders, filepath.Base(d)+": "+l)
		}
	}
	switch {
	case len(offenders) > knownNonCanonical:
		t.Errorf("non-canonical record headings rose to %d (limit %d). A sixth variant is how "+
			"the parser ends up with five grammars for one sentence — run "+
			"`ailang mission normalize --apply`, do not raise the constant.\n  first: %s",
			len(offenders), knownNonCanonical, offenders[knownNonCanonical])
	case len(offenders) < knownNonCanonical:
		t.Errorf("non-canonical record headings FELL to %d — good. Lower knownNonCanonical to %d "+
			"so the ratchet holds at the new level", len(offenders), len(offenders))
	}
}

// The canonical patterns must actually accept what the normaliser emits, or every run
// would rewrite the same headings forever and the ratchet would never settle.
func TestNormaliserOutputIsAcceptedAsCanonical(t *testing.T) {
	for _, in := range []string{
		"## Iteration 141 — 2026-08-31 — row 51 LANDED",
		"## ITERATION 0 — 2026-08-28T06:41Z (first unattended fire)",
		"## ITERATION 1 — 2026-08-28T07:26Z",
		"## STATUS 2026-09-02 — ITERATION 33 COMPLETE: **headline**",
		"## STATUS 2026-09-03 (iteration 152) — **headline**",
	} {
		out, _, ok := normalizeHeading(in)
		if !ok {
			t.Errorf("refused a variant this pass is meant to convert: %q", in)
			continue
		}
		if !canonicalEntryRe.MatchString(out) && !canonicalStatusRe.MatchString(out) {
			t.Errorf("normaliser output is not canonical, so it would rewrite forever:\n  in  %s\n  out %s", in, out)
		}
	}
}
