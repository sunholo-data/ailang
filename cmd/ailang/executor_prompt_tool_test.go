package main

import (
	"encoding/json"
	"strings"
	"testing"
)

// The hosted executor's `ailang_prompt` serves the teaching prompt cut for its
// sandbox, in two parts (executor_tools/executor.ail sandboxPrompt): "core"
// (the sections in sandboxCore) and "reference" (every other kept section).
// The cut is by heading, against whatever prompt the binary ships, so this
// test is what keeps it in step with prompt edits:
//   - every heading sandboxOmitted and sandboxCore name still exists;
//   - the sections an executor caller needs are in core;
//   - every kept section is in exactly one part, and no omitted one in either;
//   - each part stays inside its budget.

// executorPromptPartBudget is the most one part may return, in characters as
// `ailang run` prints the result, which is within a few percent of the wire.
// Claude Code refuses an MCP result over 25k tokens, counted by the API, and
// this text runs at about 2.2 characters a token: on prod, 77,090 characters
// (v0.62.1) and then 56,411 (v0.62.2) were both refused, 2026-11-26. 45k
// characters is about 20k tokens.
const executorPromptPartBudget = 45_000

// The sections a caller of the executor cannot do without; they must be in core.
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
	omitted := headingList(t, bin, "sandboxOmitted")
	core := headingList(t, bin, "sandboxCore")
	headings := promptHeadings(full)
	for _, h := range append(append([]string{}, omitted...), core...) {
		if !anyHasPrefix(headings, h) {
			t.Errorf("executor.ail names %q, which is no longer a heading in `ailang prompt`: rename it to match, or drop it", h)
		}
	}

	parts := map[string]string{}
	for _, p := range []string{"core", "reference"} {
		out, stderr, rc := runAilangBin(t, bin, "run", "--quiet", "--caps", "Clock,Debug,Env,Process",
			"--entry", "ailangPrompt", "--args-json", `"`+p+`"`, "executor_tools/executor.ail")
		if rc != 0 {
			t.Fatalf("ailangPrompt(%q) failed (exit %d): %s", p, rc, stderr)
		}
		if len(out) > executorPromptPartBudget {
			t.Errorf("ailang_prompt part %q returns %d characters, over its %d budget (Claude Code refuses results over 25k tokens): move sections between sandboxCore and reference, or omit ones a sandboxed program cannot use", p, len(out), executorPromptPartBudget)
		}
		t.Logf("ailang_prompt part %q: %d characters", p, len(out))
		parts[p] = out
	}

	for _, h := range executorPromptMustKeep {
		if !anyHasPrefix(headings, h) {
			t.Errorf("the prompt no longer has %q: update executorPromptMustKeep to the section that replaced it", h)
		} else if !strings.Contains(parts["core"], h) {
			t.Errorf("ailang_prompt core dropped %q, which an executor caller needs", h)
		}
	}
	for _, h := range keptHeadings(full, omitted) {
		in := 0
		for _, out := range parts {
			if strings.Contains(out, `\n`+escapeShown(h)+`\n`) {
				in++
			}
		}
		if in != 1 {
			t.Errorf("kept section %q is in %d parts, want exactly 1", h, in)
		}
	}
	for _, h := range omitted {
		for p, out := range parts {
			if strings.Contains(out, `\n`+escapeShown(h)) {
				t.Errorf("ailang_prompt part %q still contains omitted section %q", p, h)
			}
		}
	}

	bad, _, rc := runAilangBin(t, bin, "run", "--quiet", "--caps", "Clock,Debug,Env,Process",
		"--entry", "ailangPrompt", "--args-json", `"everything"`, "executor_tools/executor.ail")
	if rc != 0 || !strings.Contains(bad, "invalid_part") {
		t.Errorf("ailangPrompt(\"everything\") should refuse with invalid_part, got (exit %d): %s", rc, bad)
	}
}

func headingList(t *testing.T, bin, entry string) []string {
	t.Helper()
	out, stderr, rc := runAilangBin(t, bin, "run", "--quiet", "--entry", entry, "executor_tools/executor.ail")
	if rc != 0 {
		t.Fatalf("%s failed (exit %d): %s", entry, rc, stderr)
	}
	var hs []string
	if err := json.Unmarshal([]byte(strings.TrimSpace(out)), &hs); err != nil || len(hs) == 0 {
		t.Fatalf("%s is not a list of headings (%v): %s", entry, err, out)
	}
	return hs
}

// promptHeadings returns the "# " and "## " lines outside code fences.
func promptHeadings(full string) []string {
	var hs []string
	fence := false
	for _, l := range strings.Split(full, "\n") {
		if strings.HasPrefix(l, "```") {
			fence = !fence
		} else if !fence && (strings.HasPrefix(l, "# ") || strings.HasPrefix(l, "## ")) {
			hs = append(hs, l)
		}
	}
	return hs
}

// keptHeadings returns the "## " headings the cut keeps: not omitted, and not
// under an omitted "# " section.
func keptHeadings(full string, omitted []string) []string {
	var hs []string
	off := false
	for _, h := range promptHeadings(full) {
		if strings.HasPrefix(h, "# ") {
			off = anyPrefixOf(h, omitted)
		} else if !off && !anyPrefixOf(h, omitted) {
			hs = append(hs, h)
		}
	}
	return hs
}

func anyHasPrefix(lines []string, prefix string) bool {
	for _, l := range lines {
		if strings.HasPrefix(l, prefix) {
			return true
		}
	}
	return false
}

func anyPrefixOf(line string, prefixes []string) bool {
	for _, p := range prefixes {
		if strings.HasPrefix(line, p) {
			return true
		}
	}
	return false
}

// escapeShown escapes a line the way `ailang run` prints it inside a string.
func escapeShown(s string) string {
	return strings.NewReplacer(`\`, `\\`, `"`, `\"`).Replace(s)
}
