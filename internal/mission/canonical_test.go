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
// to one shape on 2026-09-06; these are this repository's genuinely irregular remainder —
// date ranges, attended-ratification stamps with no iteration number, a heading whose date
// is not in the date position at all.
//
// The count may FALL, never rise. Fixing them is welcome; adding a sixth variant is not.
const knownNonCanonical = 11

// missionDocs are the record streams. Charters are excluded deliberately: they are curated
// prose, and a lint that demanded canonical headings there would be linting an essay.
func missionDocs(t *testing.T) []string {
	t.Helper()
	docs, err := missionDocsInRepo(filepath.Join("..", ".."))
	if err != nil {
		t.Fatal(err)
	}
	return docs
}

func missionDocsInRepo(repoRoot string) ([]string, error) {
	var out []string
	dir := filepath.Join(repoRoot, "design_docs")
	for _, pat := range []string{"*-mission-log.md", "*-mission-log-archive.md",
		"*-mission-status-archive.md", "*-mission-status-archive-old.md"} {
		m, err := filepath.Glob(filepath.Join(dir, pat))
		if err != nil {
			return nil, err
		}
		out = append(out, m...)
	}
	return out, nil
}

func nonCanonicalHeadings(name string, body []byte) []string {
	var offenders []string
	for _, l := range strings.Split(string(body), "\n") {
		// Git may materialise checked-in Markdown with CRLF on Windows. A carriage
		// return is a line delimiter, not part of the heading grammar.
		l = strings.TrimSuffix(l, "\r")
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
		offenders = append(offenders, filepath.Base(name)+": "+l)
	}
	return offenders
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
			t.Fatal(err)
		}
		offenders = append(offenders, nonCanonicalHeadings(d, body)...)
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

func TestMissionDocsAreRepoLocal(t *testing.T) {
	parent := t.TempDir()
	repo := filepath.Join(parent, "repo")
	localDoc := filepath.Join(repo, "design_docs", "v1-mission-log.md")
	siblingDoc := filepath.Join(parent, "ailang-world", "design_docs", "world-mission-log.md")
	for _, path := range []string{localDoc, siblingDoc} {
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte("## 1 — 2026-09-06\n"), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	docs, err := missionDocsInRepo(repo)
	if err != nil {
		t.Fatal(err)
	}
	if len(docs) != 1 || docs[0] != localDoc {
		t.Fatalf("mission docs = %v, want only repo-local %s", docs, localDoc)
	}
}

func TestNonCanonicalHeadingsAreLineEndingIndependent(t *testing.T) {
	const lf = "## 1 — 2026-09-06\n## STATUS 2026-09-06 — attended ratification\n"
	crlf := strings.ReplaceAll(lf, "\n", "\r\n")

	gotLF := nonCanonicalHeadings("mission.md", []byte(lf))
	gotCRLF := nonCanonicalHeadings("mission.md", []byte(crlf))
	if strings.Join(gotLF, "\n") != strings.Join(gotCRLF, "\n") {
		t.Fatalf("line endings changed offenders:\nLF:   %q\nCRLF: %q", gotLF, gotCRLF)
	}
	if len(gotLF) != 1 {
		t.Fatalf("offenders = %q, want the one genuinely irregular STATUS heading", gotLF)
	}
}

// knownStrayHeadings is a RATCHET over the files rotate-log treats as LIVE (*-mission-log.md,
// *-mission-status-archive.md): "## " headings interleaved with entries, which make RotateLog
// refuse. 11 are motoko's status-archive structure (deliberately unrotated), 1 is docs-log's
// irregular "## ITERATION 3 — died mid-flight…" record. Archives are excluded: never rotated as live.
// May FALL (lower it), never rise: a new one should fail CI where it is added, not at a later rotation.
const knownStrayHeadings = 12

func TestCanonicalRecordStreamsStrayHeadingsRatchet(t *testing.T) {
	var live []string
	for _, d := range missionDocs(t) {
		b := filepath.Base(d)
		if strings.HasSuffix(b, "-mission-log.md") || strings.HasSuffix(b, "-mission-status-archive.md") {
			live = append(live, d)
		}
	}
	if len(live) == 0 {
		t.Skip("no mission record streams on this machine")
	}
	var stray []string
	for _, d := range live {
		raw, err := os.ReadFile(d) //nolint:gosec // repo-relative doc
		if err != nil {
			t.Fatal(err)
		}
		body := strings.ReplaceAll(string(raw), "\r\n", "\n") // Windows checkout (F11)
		_, entries := parseLog(body)
		for _, h := range strayHeadings(body, entries) {
			stray = append(stray, filepath.Base(d)+": "+h)
		}
	}
	switch {
	case len(stray) > knownStrayHeadings:
		t.Errorf("interleaved structural headings ROSE to %d (limit %d) — rotate-log will refuse these files. "+
			"Move the section above the first entry; do not raise the constant.\n  %s",
			len(stray), knownStrayHeadings, strings.Join(stray, "\n  "))
	case len(stray) < knownStrayHeadings:
		t.Errorf("interleaved structural headings FELL to %d — good. Lower knownStrayHeadings to %d.\n  %s",
			len(stray), len(stray), strings.Join(stray, "\n  "))
	}
}
