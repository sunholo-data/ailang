package main

import (
	"encoding/json"
	"strings"
	"testing"
)

// The hosted executor's `ailang_prompt` serves the teaching prompt cut for its
// sandbox (executor_tools/executor.ail sandboxPrompt): the whole prompt is
// ~120 KB, and Claude Code refuses an MCP result over 25k tokens. That was
// measured on prod 2026-11-25: 127,495 characters, "exceeds maximum allowed
// tokens". The cut is by heading, against whatever prompt the binary ships,
// so this test is what keeps it in step with prompt edits:
//   - every heading sandboxOmitted names still exists in the prompt (a
//     renamed section would otherwise come back silently and grow the result);
//   - the sections an executor caller needs are still there;
//   - the result stays inside its budget.

// executorPromptBudget is the most the tool may return, in characters as
// `ailang run` prints the result. Over MCP the same result is ~8% longer
// (84,808 bytes for 78,719 here, 2026-11-25). Claude Code's limit is 25k
// tokens, ~100k characters at ~4 a token; 80k printed is ~86k on the wire,
// which leaves room for the token estimate to be off.
const executorPromptBudget = 80_000

// The sections a caller of the executor cannot do without.
var executorPromptMustKeep = []string{
	"## Required Program Structure",
	"## Effects",
	"## Pattern Matching",
	"## Contracts & Verification",
	"## Common Mistakes",
}

func TestExecutorPromptTool_InSyncWithPrompt(t *testing.T) {
	requireShForFakeSolver(t) // the tool execs the binary; skips on Windows like its siblings
	bin := buildAilang(t)
	t.Setenv("EXECUTOR_AILANG_BIN", bin)
	t.Setenv("AILANG_NO_CACHE", "1")

	full, stderr, rc := runAilangBin(t, bin, "prompt")
	if rc != 0 {
		t.Fatalf("ailang prompt failed (exit %d): %s", rc, stderr)
	}
	headings := map[string]bool{}
	for _, l := range strings.Split(full, "\n") {
		if strings.HasPrefix(l, "#") {
			headings[l] = true
		}
	}
	hasHeading := func(prefix string) bool {
		for h := range headings {
			if strings.HasPrefix(h, prefix) {
				return true
			}
		}
		return false
	}

	listOut, stderr, rc := runAilangBin(t, bin, "run", "--quiet", "--entry", "sandboxOmitted", "executor_tools/executor.ail")
	if rc != 0 {
		t.Fatalf("sandboxOmitted failed (exit %d): %s", rc, stderr)
	}
	var omitted []string
	if err := json.Unmarshal([]byte(strings.TrimSpace(listOut)), &omitted); err != nil || len(omitted) == 0 {
		t.Fatalf("sandboxOmitted is not a list of headings (%v): %s", err, listOut)
	}
	for _, h := range omitted {
		if !hasHeading(h) {
			t.Errorf("sandboxOmitted names %q, which is no longer a heading in `ailang prompt`: rename it to match, or drop it", h)
		}
	}

	out, stderr, rc := runAilangBin(t, bin, "run", "--quiet", "--caps", "Clock,Debug,Env,Process",
		"--entry", "ailangPrompt", "executor_tools/executor.ail")
	if rc != 0 {
		t.Fatalf("ailangPrompt failed (exit %d): %s", rc, stderr)
	}
	if len(out) > executorPromptBudget {
		t.Errorf("ailang_prompt returns %d characters, over its %d budget (Claude Code refuses results over 25k tokens): add the new or grown sections a sandboxed program cannot use to sandboxOmitted", len(out), executorPromptBudget)
	}
	if len(out) < len(full)/4 {
		t.Errorf("ailang_prompt returns %d characters of a %d-character prompt: the cut took too much", len(out), len(full))
	}
	for _, h := range executorPromptMustKeep {
		if !hasHeading(h) {
			t.Errorf("the prompt no longer has %q: update executorPromptMustKeep to the section that replaced it", h)
		} else if !strings.Contains(out, h) {
			t.Errorf("ailang_prompt dropped %q, which an executor caller needs", h)
		}
	}
	for _, h := range omitted {
		if strings.Contains(out, "\\n"+h) || strings.HasPrefix(out, h) {
			t.Errorf("ailang_prompt still contains omitted section %q", h)
		}
	}
	t.Logf("ailang_prompt: %d characters of %d", len(out), len(full))
}
