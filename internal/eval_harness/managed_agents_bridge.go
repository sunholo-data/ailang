// Cross-environment file bridge for executors that advertise
// executor.CapRemoteSandbox (the Managed Agents API runs the agent in a
// Google-hosted Linux sandbox, so agent file edits don't touch the local
// workspace). The eval harness can't read solution.ail server-side, so we
// (a) instruct the agent to dump its solution in its text response as a
// fenced code block, and (b) extract that block + write it locally.
//
// Lives in eval_harness/ (not in the executor itself) because this is
// eval-harness POLICY — different backend callers might want different
// bridging strategies (e.g. GCS upload/download, no bridging, different
// file destinations). The executor stays policy-free.

package eval_harness

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/sunholo-data/ailang/internal/executor"
)

// executorHasCapability returns true if exec's Capabilities() slice contains
// the given capability. Tiny helper because Go's []Capability isn't a set.
func executorHasCapability(exec executor.Executor, want executor.Capability) bool {
	if exec == nil {
		return false
	}
	for _, c := range exec.Capabilities() {
		if c == want {
			return true
		}
	}
	return false
}

// managedAgentsBridgeInstruction is appended to the system prompt for
// CapRemoteSandbox executors so the agent's solution code surfaces in its
// text response (the only channel the eval harness can read).
const managedAgentsBridgeInstruction = "\n\nIMPORTANT — Cross-environment requirement: " +
	"Your sandbox file edits cannot be read by the evaluator running this task. " +
	"At the very end of your response, ALWAYS output your complete final solution " +
	"as a single fenced code block (```<lang>\n...\n```). " +
	"Even if you used file-edit tools in the sandbox, repeat the final file contents " +
	"verbatim in the fenced block. This is the only channel the evaluator can read."

// writeSolutionFromResponse extracts the agent's final code answer from the
// streamed text response and writes it to the eval harness's solution path.
//
// This is necessary because the Managed Agents API runs the agent in a
// Google-hosted sandbox — the agent's file edits land server-side and don't
// touch our local workspace. CLI-subprocess executors (claude, codex,
// opencode) don't have this problem because their agent processes share the
// local filesystem with the harness.
//
// Returns the absolute solution path written and the number of bytes if the
// write succeeded, or empty/0 if no code block could be extracted.
//
// Convention: the eval harness creates <workspace>/benchmark/solution.ail
// (for AILANG) or <workspace>/<lang-default> (for other langs) before
// invoking the executor. We honour the AILANG convention here; other
// languages fall through to a best-effort guess at the workspace root.
func writeSolutionFromResponse(workspace, response string) (path string, bytesWritten int, err error) {
	if workspace == "" || response == "" {
		return "", 0, nil
	}

	// Look for the AILANG solution path the eval harness placed.
	candidates := []string{
		filepath.Join(workspace, "benchmark", "solution.ail"),
		filepath.Join(workspace, "solution.ail"),
		filepath.Join(workspace, "solution.py"),
	}
	var solutionPath string
	for _, c := range candidates {
		if _, statErr := os.Stat(c); statErr == nil {
			solutionPath = c
			break
		}
	}
	if solutionPath == "" {
		// No placeholder found — nothing to overwrite; leave the workspace
		// alone rather than guessing where the harness wanted output.
		return "", 0, nil
	}

	code := extractCode(response, solutionPath)
	if code == "" {
		return solutionPath, 0, nil
	}

	if err := os.WriteFile(solutionPath, []byte(code), 0644); err != nil {
		return solutionPath, 0, fmt.Errorf("managed_agents: write solution: %w", err)
	}
	return solutionPath, len(code), nil
}

// extractCode pulls the most likely solution code out of the agent's text
// response, with two strategies tried in order:
//
//  1. The last fenced code block (```...```), with an optional language tag.
//     This is the canonical way LLMs surface code in chat responses.
//  2. If the response itself looks like raw code (contains a module
//     declaration matching the expected file), use the whole response.
//
// Returns empty string when neither strategy yields content. We prefer the
// LAST fenced block because models often print a "here's the final version"
// block after intermediate scratch.
func extractCode(response, solutionPath string) string {
	if response == "" {
		return ""
	}

	// Strategy 1: last fenced code block.
	if block := lastFencedBlock(response); block != "" {
		return block
	}

	// Strategy 2: raw code detection — only when the response itself OPENS
	// with a recognisable code shape (anchored, not just contains).
	// Without anchoring, agent chain-of-thought commentary like
	// "I will start by viewing the target file ... module benchmark/solution ..."
	// gets misclassified as code because it mentions the keyword.
	trimmed := strings.TrimLeft(response, " \t\n\r")
	switch filepath.Ext(solutionPath) {
	case ".ail":
		// AILANG modules always start with `module <path>` (optionally
		// preceded by comments). Accept response as raw code only when the
		// first non-comment line begins with `module `.
		if firstCodeLineStartsWith(trimmed, "module ") {
			return trimmed
		}
	case ".py":
		// Python solutions typically start with `import`, `from`, or `def`.
		if firstCodeLineStartsWith(trimmed, "import ") ||
			firstCodeLineStartsWith(trimmed, "from ") ||
			firstCodeLineStartsWith(trimmed, "def ") {
			return trimmed
		}
	}
	return ""
}

// firstCodeLineStartsWith returns true if the first non-comment, non-blank
// line of s starts with prefix. Comments are detected via `//` (AILANG/JS)
// and `#` (Python). Blank lines are skipped.
//
// Comment scanning is deliberately conservative — a bare `module` mention in
// a `// commentary about modules ...` line MUST NOT count as code start.
func firstCodeLineStartsWith(s, prefix string) bool {
	for _, line := range strings.Split(s, "\n") {
		stripped := strings.TrimSpace(line)
		if stripped == "" {
			continue
		}
		if strings.HasPrefix(stripped, "//") || strings.HasPrefix(stripped, "#") {
			continue
		}
		return strings.HasPrefix(stripped, prefix)
	}
	return false
}

// lastFencedBlock returns the content of the last ``` ... ``` block in s,
// stripped of the language hint on the opening fence. Returns "" if no
// complete fenced block is present.
func lastFencedBlock(s string) string {
	// Walk back from the end to find the last closing fence.
	endFence := strings.LastIndex(s, "```")
	if endFence < 0 {
		return ""
	}
	// Find the matching opening fence before endFence.
	openFence := strings.LastIndex(s[:endFence], "```")
	if openFence < 0 {
		return ""
	}
	// Skip past the opening ``` plus optional language tag + newline.
	inner := s[openFence+3 : endFence]
	if nl := strings.IndexByte(inner, '\n'); nl >= 0 {
		// If the first line is a language tag (no spaces, like "ailang"),
		// drop it. Otherwise keep the whole inner block.
		firstLine := strings.TrimSpace(inner[:nl])
		if firstLine != "" && !strings.ContainsAny(firstLine, " \t") {
			inner = inner[nl+1:]
		}
	}
	return strings.TrimSpace(inner) + "\n"
}
