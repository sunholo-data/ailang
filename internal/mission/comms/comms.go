// Package comms carries the mission↔human GitHub channel.
//
// It exists because that channel is measurably broken. The v1 bookkeeping thread
// for the week of 2026-08-31 held 27 comments and 52,677 characters, of which the
// human wrote exactly one comment, six characters long — a signal ratio of about
// 1:8,800. The thread had already lost decisions inside itself: it carries a
// "Retraction — iteration 308 re-asked two decisions you had already answered".
//
// See design_docs/planned/v0_36_0/m-mission-comms-into-the-binary.md.
package comms

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"unicode/utf8"
)

// MaxReportChars caps a single iteration report.
//
// The design doc's success metric is a drop from ~52,677 chars/week to under
// 8,000, which assumes roughly this per report. It is pinned by a test so that
// raising it is a deliberate act with a visible diff, not a quiet drift back to
// the 1,951-char mean that made the thread unreadable.
const MaxReportChars = 400

// truncationMark ends a report that did not fit. A report silently cut at the cap
// is one the reader trusts and shouldn't — Critical Principle 2 applies to
// presentation as much as to control flow.
const truncationMark = "…[truncated]"

// missionRepos maps a mission to its repository. Repos are stable, unlike the
// weekly bookkeeping issue, so this is the one part that belongs in code.
var missionRepos = map[string]string{
	"v1":     "sunholo-data/ailang",
	"world":  "sunholo-data/ailang-world",
	"docs":   "sunholo-data/ailang",
	"motoko": "sunholo-data/ailang",
}

// Mission is a resolved mission identity: which repo and which thread.
//
// The driver currently answers this by passing MISSION_GH_ISSUE around as a bare
// env string at six call sites. Resolving it once, in one place, is most of the
// point of the port.
type Mission struct {
	Name  string
	Repo  string
	Issue int
}

// stateDir mirrors the driver's resolution so the two cannot disagree about where
// mission state lives.
func stateDir() string {
	if d := os.Getenv("AILANG_STATE_DIR"); d != "" {
		return d
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return ".ailang/state"
	}
	return filepath.Join(home, ".ailang", "state")
}

// ResolveMission resolves a mission name to its repo and live bookkeeping issue.
//
// The issue is READ FROM STATE, never hardcoded: the threads rotate weekly, so a
// constant would be stale within days. An unknown mission, a missing state file
// or an unparseable issue are all hard errors — silently defaulting would post
// one mission's report onto another mission's thread, or onto issue 0.
func ResolveMission(name string) (Mission, error) {
	if name == "" {
		return Mission{}, fmt.Errorf("mission name is empty")
	}
	repo, ok := missionRepos[name]
	if !ok {
		known := make([]string, 0, len(missionRepos))
		for k := range missionRepos {
			known = append(known, k)
		}
		// Sorted so the error message is deterministic (A1).
		sortStrings(known)
		return Mission{}, fmt.Errorf("unknown mission %q (known: %s)", name, strings.Join(known, ", "))
	}

	m := Mission{Name: name, Repo: repo}

	// The driver honours MISSION_GH_ISSUE as an override; the port must too, or a
	// cutover cannot be rehearsed against a scratch issue.
	if v := strings.TrimSpace(os.Getenv("MISSION_GH_ISSUE")); v != "" {
		n, err := strconv.Atoi(v)
		if err != nil {
			return Mission{}, fmt.Errorf("MISSION_GH_ISSUE=%q is not a number: %w", v, err)
		}
		m.Issue = n
		return m, nil
	}

	// ALWAYS the namespaced file. mission-control.sh:78-83 records that v1 alone
	// once read the fleet-shared bare `mission-gh-issue`, which holds a CLOSED
	// thread (745) while `mission-v1-gh-issue` holds the live one. Reading the
	// bare file would post every v1 report into a dead thread, silently.
	path := filepath.Join(stateDir(), "mission-"+name+"-gh-issue")
	raw, err := os.ReadFile(path) //nolint:gosec // path is built from a known mission name
	if err != nil {
		return Mission{}, fmt.Errorf("no issue for mission %q: %s unreadable: %w", name, path, err)
	}
	first := strings.TrimSpace(strings.SplitN(string(raw), "\n", 2)[0])
	n, err := strconv.Atoi(first)
	if err != nil {
		return Mission{}, fmt.Errorf("issue file %s does not hold a number (%q): %w", path, first, err)
	}
	if n <= 0 {
		return Mission{}, fmt.Errorf("issue file %s holds a non-positive issue number %d", path, n)
	}
	m.Issue = n
	return m, nil
}

// Report is one iteration's worth of what a human needs.
//
// Deliberately a small fixed set of fields rather than free prose: the prose is
// what grew to 1,951 chars a comment, and it already exists in the mission log,
// which is git-tracked. The GitHub comment is a pointer to that record, not a
// second copy of it.
type Report struct {
	Mission      string
	Iteration    int
	Landed       string
	GoalDistance string
	CostUSD      float64
	LogCommit    string
}

// RenderReport produces the capped, deterministic report body.
//
// Deterministic by construction: fixed field order, no map iteration, no
// timestamps. The same iteration state renders byte-identically every time, which
// is what makes the comms path replayable (A1, A2).
func RenderReport(r Report) string {
	var b strings.Builder
	fmt.Fprintf(&b, "**%s iter %d**", r.Mission, r.Iteration)
	if r.Landed != "" {
		fmt.Fprintf(&b, " — %s", r.Landed)
	}
	if r.GoalDistance != "" {
		fmt.Fprintf(&b, "\nGoal: %s", r.GoalDistance)
	}
	fmt.Fprintf(&b, "\nCost: $%.2f", r.CostUSD)
	if r.LogCommit != "" {
		fmt.Fprintf(&b, " · log: %s", r.LogCommit)
	}
	return truncate(b.String(), MaxReportChars)
}

// truncate cuts to at most max BYTES while never splitting a rune, and marks that
// it did. Byte-based because the cap is about payload size; rune-safe because the
// reports are full of em dashes and arrows and a byte-boundary cut would emit
// invalid UTF-8 into a GitHub comment.
func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	keep := max - len(truncationMark)
	if keep < 0 {
		keep = 0
	}
	// Back off to a rune boundary.
	for keep > 0 && !utf8.RuneStart(s[keep]) {
		keep--
	}
	return s[:keep] + truncationMark
}

// sortStrings is a tiny insertion sort, used only to make an error message
// deterministic. Avoids pulling in "sort" for one call on a 4-element slice.
func sortStrings(xs []string) {
	for i := 1; i < len(xs); i++ {
		for j := i; j > 0 && xs[j] < xs[j-1]; j-- {
			xs[j], xs[j-1] = xs[j-1], xs[j]
		}
	}
}
