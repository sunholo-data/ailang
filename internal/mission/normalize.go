package mission

import (
	"fmt"
	"os"
	"regexp"
	"strings"
)

// Heading normalisation.
//
// The fleet writes the same record four ways, and supporting all of them cost this package
// two rounds of "the parser silently under-indexed". Counted across every mission doc on
// 2026-09-06:
//
//	368  ## 317 — 2026-09-02 — title              <- CANONICAL, and already the majority
//	160  ## Iteration 141 — 2026-08-31 — title    (world)
//	 12  ## ITERATION 0 — 2026-08-28T06:41Z (n)   (docs)
//	  1  ## ITERATION 3 — died mid-flight ... (2026-09-02)   <- date not even in position
//
// Normalising to the majority shape means ONE pattern to parse, one to lint, and no
// variant left to rediscover. The alternative — keep widening the parser — is how you end
// up maintaining four grammars for one sentence.
//
// STATUS stamps keep their own canonical shape; they are a different stream, not a variant:
//
//	## STATUS 2026-09-06 — ITERATION 333: **headline**

// canonicalEntryRe matches a heading that is ALREADY canonical.
// A TITLE-LESS entry is canonical too: "## 1 — 2026-08-28". Ten of docs' twelve entries
// carry only a number and a timestamp, and refusing them would have left the majority of
// that mission permanently non-canonical — or forced the normaliser to invent a title,
// which is worse. The shape is what is being standardised; the prose is the author's.
var canonicalEntryRe = regexp.MustCompile(`^## \d+ — \d{4}-\d{2}-\d{2}( — \S|$)`)

// canonicalStatusRe matches an already-canonical STATUS stamp.
var canonicalStatusRe = regexp.MustCompile(`^## STATUS \d{4}-\d{2}-\d{2} — ITERATION \d+: \S`)

// wordyEntryRe matches "## Iteration 141 — 2026-08-31 — title" and the ALL-CAPS variant,
// including a trailing ISO time and a parenthetical qualifier on the date.
// The trailing parenthetical is NOT consumed as a qualifier: in docs' form it IS the
// title ("## ITERATION 0 — 2026-08-28T06:41Z (first unattended fire)"). Swallowing it left
// an empty title, which normalizeHeading correctly refused — so the heading silently
// stayed un-normalised. Capture it and unwrap it instead.
var wordyEntryRe = regexp.MustCompile(
	`^## (?i:iteration)\s+(\d+)\s+—\s+(\d{4}-\d{2}-\d{2})(?:T[0-9:]+Z?)?\s*(?:—\s*)?(.*)$`)

// statusVariantRe matches the STATUS variants that are not yet canonical: a parenthetical
// on the date, and/or a word between the number and the colon.
var statusVariantRe = regexp.MustCompile(
	`^## STATUS\s+(\d{4}-\d{2}-\d{2})(?:\s*\(([^)]*)\))?\s+—\s+ITERATION\s+(\d+)(?:\s+([A-Za-z]+))?\s*:?\s*(.*)$`)

// statusParenIterRe matches the variant that puts the iteration number INSIDE the
// parenthetical rather than after the word ITERATION:
//
//	## STATUS 2026-09-03 (iteration 152) — **headline**
//
// The fifth status shape, found only by running the normaliser across the fleet and
// reading what it refused. That is the argument for reporting refusals loudly rather than
// skipping them quietly: the long tail is invisible until something counts it.
var statusParenIterRe = regexp.MustCompile(
	`^## STATUS\s+(\d{4}-\d{2}-\d{2})\s*\(\s*(?i:iteration)\s+(\d+)\s*\)\s*—\s*(.*)$`)

// Rewrite is one heading change, reported so a human can see exactly what moved.
type Rewrite struct {
	Line int
	From string
	To   string
}

// NormalizeResult reports a normalisation.
type NormalizeResult struct {
	Path      string
	Rewrites  []Rewrite
	Unhandled []Rewrite // headings that look like records but could not be converted safely
	Applied   bool
}

// normalizeHeading returns the canonical form of a heading, and whether it changed.
//
// Returns ok=false for a heading it cannot convert CONFIDENTLY. Those are reported rather
// than guessed at: a wrong rewrite silently changes a record's identity, and the whole
// point of this pass is to stop maintaining variants, not to invent a fifth.
func normalizeHeading(l string) (out string, changed bool, ok bool) {
	switch {
	case canonicalEntryRe.MatchString(l), canonicalStatusRe.MatchString(l):
		return l, false, true

	case statusParenIterRe.MatchString(l):
		m := statusParenIterRe.FindStringSubmatch(l)
		title := strings.TrimSpace(m[3])
		if title == "" {
			return l, false, false
		}
		return fmt.Sprintf("## STATUS %s — ITERATION %s: %s", m[1], m[2], title), true, true

	case statusVariantRe.MatchString(l):
		m := statusVariantRe.FindStringSubmatch(l)
		title := strings.TrimSpace(m[5])
		// A qualifier like "(midday)" and a word like "COMPLETE" are real information;
		// fold them into the headline rather than dropping them on the floor.
		if q := strings.TrimSpace(m[2]); q != "" {
			title = "(" + q + ") " + title
		}
		if w := strings.TrimSpace(m[4]); w != "" {
			title = w + ": " + title
		}
		if title == "" {
			return l, false, false
		}
		return fmt.Sprintf("## STATUS %s — ITERATION %s: %s", m[1], m[3], title), true, true

	case wordyEntryRe.MatchString(l):
		m := wordyEntryRe.FindStringSubmatch(l)
		title := strings.TrimSpace(m[3])
		// A title that is entirely parenthesised loses the wrapper, so
		// "(first unattended fire)" reads as a title rather than an aside.
		if strings.HasPrefix(title, "(") && strings.HasSuffix(title, ")") &&
			!strings.Contains(title[1:len(title)-1], "(") {
			title = strings.TrimSpace(title[1 : len(title)-1])
		}
		if title == "" {
			// No title in the source, so none in the output. Emitting a dangling
			// separator would be tidier-looking and less honest.
			return fmt.Sprintf("## %s — %s", m[1], m[2]), true, true
		}
		return fmt.Sprintf("## %s — %s — %s", m[1], m[2], title), true, true
	}
	return l, false, false
}

// looksLikeRecord reports whether a heading is plausibly an iteration record rather than a
// structural section, so the report can separate "could not convert" from "not a record".
func looksLikeRecord(l string) bool {
	low := strings.ToLower(l)
	return strings.Contains(low, "iteration") || strings.Contains(low, "## status") ||
		regexp.MustCompile(`^## \d`).MatchString(l)
}

// Normalize rewrites a mission document's headings to canonical form.
//
// apply=false is a dry run: it reports every rewrite and changes nothing. Bodies are NEVER
// touched — only the heading line itself — and a caller can verify that by diffing
// everything except the reported lines.
func Normalize(path string, apply bool) (*NormalizeResult, error) {
	raw, err := os.ReadFile(path) //nolint:gosec // caller-supplied mission doc
	if err != nil {
		return nil, fmt.Errorf("failed to read %s: %w", path, err)
	}
	lines := strings.Split(string(raw), "\n")
	res := &NormalizeResult{Path: path, Applied: apply}
	for i, l := range lines {
		if !strings.HasPrefix(l, "## ") {
			continue
		}
		out, changed, ok := normalizeHeading(l)
		switch {
		case ok && changed:
			res.Rewrites = append(res.Rewrites, Rewrite{i + 1, l, out})
			lines[i] = out
		case !ok && looksLikeRecord(l):
			res.Unhandled = append(res.Unhandled, Rewrite{i + 1, l, ""})
		}
	}
	if apply && len(res.Rewrites) > 0 {
		if werr := writeAtomic(path, []byte(strings.Join(lines, "\n")), 0o644); werr != nil { //nolint:gosec // a doc
			return nil, werr
		}
	}
	return res, nil
}
