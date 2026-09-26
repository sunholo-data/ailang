package coordinator

import "testing"

// The real transcript, 2026-09-14. sprint-executor read the plan, found the
// sprint state file missing, and reported `no_changes` — indistinguishable from
// a package agent that checked its dependency and found it current.
func TestParseBlockedMarker_TheRealCase(t *testing.T) {
	transcript := `[TURN 1]
I'm invoking the sprint-executor skill as requested.
[TOOL] /bin/bash -lc "validate_sprint_json.sh M-OPENROUTER-EU-ROUTING"
Execution is blocked by the mandatory sprint-executor gate.

**BLOCKED:** the sprint state file has not been committed
**BLOCKED_ON:** ` + "`.ailang/state/sprints/sprint_M-OPENROUTER-EU-ROUTING.json`" + `
**IMPLEMENTATION_COMPLETE:** ` + "`false`"

	got, ok := ParseBlockedMarker(transcript)
	if !ok {
		t.Fatal("a BLOCKED: marker must be recognised")
	}
	if got.Reason != "the sprint state file has not been committed" {
		t.Errorf("reason = %q", got.Reason)
	}
	if got.On != ".ailang/state/sprints/sprint_M-OPENROUTER-EU-ROUTING.json" {
		t.Errorf("on = %q — the machine-readable half must survive the backticks", got.On)
	}
}

// Models emit "**BLOCKED:**" as readily as the bare form. A contract that only
// matches one is a contract most runs fail by accident.
func TestParseBlockedMarker_TolerantOfDecoration(t *testing.T) {
	for _, in := range []string{
		"BLOCKED: no config",
		"**BLOCKED:** no config",
		"  **BLOCKED:**   `no config`  ",
		`BLOCKED: "no config"`,
	} {
		got, ok := ParseBlockedMarker(in)
		if !ok || got.Reason != "no config" {
			t.Errorf("%q -> (%+v, %v), want reason %q", in, got, ok, "no config")
		}
	}
}

// BLOCKED_ON: also carries the BLOCKED: prefix. Testing in the wrong order
// would read the subject as the reason.
func TestParseBlockedMarker_OnIsNotMistakenForReason(t *testing.T) {
	got, ok := ParseBlockedMarker("BLOCKED_ON: some/path.json")
	if ok {
		t.Errorf("BLOCKED_ON alone is not a declaration: %+v", got)
	}
}

// An agent may narrate a blocker it then works around. Its LAST word describes
// the outcome; freezing an early worry into a terminal state would be worse
// than not having the marker at all.
func TestParseBlockedMarker_LastOccurrenceWins(t *testing.T) {
	got, ok := ParseBlockedMarker("BLOCKED: no config yet\nfound one, continuing\nBLOCKED: the API key is absent")
	if !ok || got.Reason != "the API key is absent" {
		t.Errorf("got %+v, want the final declaration", got)
	}
}

func TestParseBlockedMarker_AbsentOrEmpty(t *testing.T) {
	for _, in := range []string{
		"", "all done, two files changed",
		"BLOCKED:", "**BLOCKED:**   ",
		"the task was blocked on nothing in particular", // prose, not a marker
	} {
		if got, ok := ParseBlockedMarker(in); ok {
			t.Errorf("%q must not parse as blocked, got %+v", in, got)
		}
	}
}

// blocked must NOT suppress a later identical request: re-running once the
// precondition is fixed is the entire point.
func TestBlockedDoesNotSuppressDedup(t *testing.T) {
	if DedupSuppresses(TaskStatusBlocked) {
		t.Error("a blocked task must never suppress the re-run that follows the fix")
	}
}

// ...but it IS over. Leaving it non-terminal would have the stale detector
// re-dispatch it forever against a precondition no agent can satisfy.
func TestBlockedIsTerminal(t *testing.T) {
	if !IsTerminalStatus(TaskStatusBlocked) {
		t.Error("blocked is terminal for this attempt")
	}
}
