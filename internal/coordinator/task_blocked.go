package coordinator

// An agent saying "I could not start, and here is why".
//
// Until now an agent had NO voice in its own outcome. The job classified purely
// mechanically — ClassifyCompletionStatus(changedFiles, branchPushed,
// expectChanges) — so "did any files change?" was the entire vocabulary. A
// blocked precondition, a refusal on principle, and a discovered impossibility
// all collapsed into `no_changes`, which ALSO means "I did the work and nothing
// needed changing". Those are different facts and they need different responses.
//
// Measured 2026-09-14: sprint-executor read the plan, found
// `.ailang/state/sprints/sprint_M-OPENROUTER-EU-ROUTING.json` missing, and said
// so in one clear sentence. It reported `no_changes`, indistinguishable from a
// package agent that checked its dependency and found it already current. One
// of those needs a human; the other needs nobody.
//
// WHY A MARKER AND NOT A NEW CLI. Every executor already streams its transcript
// and the harness already parses markers out of it (DESIGN_DOC_PATH:,
// IMPLEMENTATION_COMPLETE:). A second reporting channel would be a second
// implementation of "how an agent tells us what happened" — and six of the
// faults fixed on 2026-09-14 were precisely two implementations of one thing
// disagreeing. The marker rides a path that already works on claude, codex, pi
// and opencode alike, with no binary to install and nothing to keep in step.
//
// The asymmetry that makes this safe: nothing DEPENDS on the marker being
// present. An agent that forgets it gets today's behaviour, which is no worse.
// Contrast DesignDocPath, where a forgotten marker broke a downstream gate for
// nineteen tasks — a dependency on a model remembering, which is the shape to
// avoid.

import "strings"

// BlockedMarker is what an agent writes when it cannot proceed.
//
//	BLOCKED: the sprint state file has not been committed
//	BLOCKED_ON: .ailang/state/sprints/sprint_M-FOO.json
//
// BLOCKED_ON is optional and is the machine-readable half: the thing to fix.
const (
	BlockedMarker   = "BLOCKED:"
	BlockedOnMarker = "BLOCKED_ON:"
)

// BlockedReport is an agent's declaration that it did not attempt the work.
type BlockedReport struct {
	Reason string // human-readable, required
	On     string // machine-readable subject, optional
}

// ParseBlockedMarker extracts a blocked declaration from a transcript.
//
// The LAST occurrence wins. An agent may narrate a blocker it then works around
// ("BLOCKED: no config — creating one"), and its final word is the one that
// describes the outcome. Taking the first would freeze an early worry into a
// terminal state.
func ParseBlockedMarker(transcript string) (BlockedReport, bool) {
	var out BlockedReport
	var found bool
	for _, line := range strings.Split(transcript, "\n") {
		t := strings.TrimSpace(stripMarkdownEmphasis(line))
		switch {
		case strings.HasPrefix(t, BlockedOnMarker):
			// Checked FIRST: BLOCKED_ON: also has the BLOCKED: prefix, and
			// testing in the other order would read the subject as the reason.
			out.On = cleanMarkerValue(strings.TrimPrefix(t, BlockedOnMarker))
		case strings.HasPrefix(t, BlockedMarker):
			if r := cleanMarkerValue(strings.TrimPrefix(t, BlockedMarker)); r != "" {
				out.Reason = r
				found = true
			}
		}
	}
	if !found {
		return BlockedReport{}, false
	}
	return out, true
}

// stripMarkdownEmphasis removes the ** an agent wraps markers in. Models emit
// "**BLOCKED:** reason" as often as the bare form, and a marker contract that
// only matches one of them is a contract most runs fail by accident.
func stripMarkdownEmphasis(s string) string {
	return strings.ReplaceAll(strings.TrimSpace(s), "**", "")
}

// cleanMarkerValue trims the backticks and quotes models decorate values with.
func cleanMarkerValue(s string) string {
	return strings.Trim(strings.TrimSpace(s), "`\"' ")
}
