package coordinator

// Checking a triage row against its OWN numbers.
//
// The ailang-core-triage skill carries a countable rubric — `DIRECT_FIX_MAX_LINES`,
// `DIRECT_FIX_MAX_FILES` — precisely so that `direct-fix` vs `design-doc` is
// auditable rather than aesthetic. It is still only text the model reads.
//
// Measured 2026-09-15 on task-c0ca6301, a run whose clone CONTAINED the rubric
// (the branch is cut from the commit that added it): the row wrote
// `Estimate: ~15–30 lines` and `Recommend: direct-fix` in the same file, with
// the threshold at 2. The skill's own words — "a number you write and disregard
// is worse than no number, because it looks like evidence" — describe exactly
// what landed.
//
// So the rubric is enforced here instead of merely stated there. The thresholds
// are READ FROM THE SKILL rather than copied: a checker with its own copy of the
// numbers is the two-implementations fault this pipeline keeps producing, and
// tuning the skill would silently stop tuning the gate.
//
// The gate runs in `make test`, so a violating row turns the PR's `test` check
// red — and the coordinator merges with `gh pr merge --auto` and no `--admin`,
// so a red check leaves the PR open rather than landing the row.

import (
	"fmt"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

// TriageLimits are the countable thresholds from the skill.
type TriageLimits struct {
	MaxLines int
	MaxFiles int
}

var (
	limitRe = regexp.MustCompile("(?m)^\\|\\s*`(DIRECT_FIX_MAX_LINES|DIRECT_FIX_MAX_FILES)`\\s*\\|\\s*\\*\\*(\\d+)\\*\\*\\s*\\|")
	// Numbers immediately qualifying "lines", including a range: "~15–30 lines".
	// Both an en dash and a hyphen occur in practice.
	estLineRe = regexp.MustCompile(`(\d+)\s*(?:[–—-]\s*(\d+))?\s*lines?\b`)
	// File-looking tokens, so "more than one file" is countable rather than read.
	estFileRe = regexp.MustCompile(`[\w./-]+\.(?:go|md|ya?ml|sh|ail|json|txt)\b`)
	bulletRe  = regexp.MustCompile(`(?m)^\s*-\s+\*\*([A-Za-z]+)\*\*:\s*(.*)$`)
)

// ParseTriageLimits reads the thresholds out of the skill document.
//
// Returns an error rather than a default when either is missing: a gate that
// silently falls back to a number nobody wrote is not a gate.
func ParseTriageLimits(skillMD string) (TriageLimits, error) {
	var l TriageLimits
	for _, m := range limitRe.FindAllStringSubmatch(skillMD, -1) {
		n, err := strconv.Atoi(m[2])
		if err != nil {
			return l, fmt.Errorf("threshold %s is not a number: %q", m[1], m[2])
		}
		switch m[1] {
		case "DIRECT_FIX_MAX_LINES":
			l.MaxLines = n
		case "DIRECT_FIX_MAX_FILES":
			l.MaxFiles = n
		}
	}
	if l.MaxLines <= 0 || l.MaxFiles <= 0 {
		return l, fmt.Errorf("skill does not declare both DIRECT_FIX_MAX_LINES and DIRECT_FIX_MAX_FILES as a `| `NAME` | **n** |` row (got lines=%d files=%d)", l.MaxLines, l.MaxFiles)
	}
	return l, nil
}

// TriageRow is the header of one triage artifact.
type TriageRow struct {
	Recommend string
	Estimate  string
}

// ParseTriageRow reads the row's declared fields. Missing fields stay empty —
// CheckTriageRow decides which absences matter.
func ParseTriageRow(md string) TriageRow {
	var r TriageRow
	for _, m := range bulletRe.FindAllStringSubmatch(md, -1) {
		v := strings.TrimSpace(m[2])
		switch strings.ToLower(m[1]) {
		case "recommend":
			r.Recommend = v
		case "estimate":
			r.Estimate = v
		}
	}
	return r
}

// recommendLabel is the bare verdict: `duplicate-of <path>` carries an argument.
func recommendLabel(s string) string {
	s = strings.TrimSpace(strings.Trim(strings.TrimSpace(s), "`"))
	if i := strings.IndexAny(s, " \t"); i > 0 {
		s = s[:i]
	}
	return strings.ToLower(strings.Trim(s, "`"))
}

var validRecommends = map[string]bool{
	"design-doc": true, "direct-fix": true, "duplicate-of": true, "drop": true,
}

// EstimateLines is the LARGEST line count the estimate admits to.
//
// The largest, not the smallest: a range is a claim that the change might be
// that big, and a threshold that reads only the optimistic end is not a bound.
// Returns 0 when the estimate names no line count.
func EstimateLines(estimate string) int {
	max := 0
	for _, m := range estLineRe.FindAllStringSubmatch(estimate, -1) {
		for _, g := range m[1:] {
			if g == "" {
				continue
			}
			if n, err := strconv.Atoi(g); err == nil && n > max {
				max = n
			}
		}
	}
	return max
}

// EstimateFiles counts the distinct file paths the estimate names.
func EstimateFiles(estimate string) []string {
	seen := map[string]bool{}
	var out []string
	for _, f := range estFileRe.FindAllString(estimate, -1) {
		if !seen[f] {
			seen[f] = true
			out = append(out, f)
		}
	}
	sort.Strings(out)
	return out
}

// CheckTriageRow reports every way the row contradicts the rubric it was
// written under. An empty slice means the label is consistent with its numbers.
//
// It deliberately does NOT judge whether the estimate is correct — only whether
// the row obeys the estimate it states. That is the auditable part: when the fix
// lands, its real diff is a separate check on the estimate itself.
func CheckTriageRow(row TriageRow, limits TriageLimits) []string {
	var problems []string
	label := recommendLabel(row.Recommend)
	if label == "" {
		return []string{"no `- **Recommend**:` line — the row states no verdict"}
	}
	if !validRecommends[label] {
		return []string{fmt.Sprintf("Recommend %q is not one of design-doc | direct-fix | duplicate-of <path> | drop", row.Recommend)}
	}
	if label != "direct-fix" {
		return nil
	}
	if strings.TrimSpace(row.Estimate) == "" {
		return []string{"direct-fix with no `- **Estimate**:` line — the skill requires the estimate that justifies the label"}
	}
	n := EstimateLines(row.Estimate)
	if n == 0 {
		problems = append(problems, fmt.Sprintf(
			"direct-fix whose estimate names no line count (%q) — an estimate you cannot count is not an estimate", row.Estimate))
	} else if n > limits.MaxLines {
		problems = append(problems, fmt.Sprintf(
			"direct-fix estimated at %d lines, over DIRECT_FIX_MAX_LINES=%d — rubric row 6 makes this design-doc (estimate: %q)",
			n, limits.MaxLines, row.Estimate))
	}
	if files := EstimateFiles(row.Estimate); len(files) > limits.MaxFiles {
		problems = append(problems, fmt.Sprintf(
			"direct-fix naming %d files, over DIRECT_FIX_MAX_FILES=%d — rubric row 5 makes this design-doc (%s)",
			len(files), limits.MaxFiles, strings.Join(files, ", ")))
	}
	return problems
}
