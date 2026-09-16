package pi

import (
	"fmt"
	"strings"

	"github.com/sunholo-data/ailang/internal/executor"
)

// buildPiArgs assembles the pi CLI argument vector.
//
// Flags used:
//
//	--mode json        NDJSON event stream
//	-p                 non-interactive (process prompt and exit)
//	--model <prov/id>  model selection via provider-prefix shorthand
//	--no-session       ephemeral run (avoids ~/.pi/sessions/ pollution)
//	--no-tools         when AllowedTools is empty; otherwise --tools <list>
//	--thinking <lvl>   only when the registry declares reasoning_effort
//
// The directive is the trailing positional argument.
//
// NOT expressible here: the per-request output budget. pi has no max-tokens
// flag — it reads maxTokens from ~/.pi/agent/models.json and falls back to
// 16384 for any model that omits it (model-registry.js). task.MaxOutputTokens
// therefore CANNOT be forwarded from this side; the registry's declared budget
// reaches the wire only if the pi config carries the same number. Canonical
// copy: tools/pi-extensions/models.mission.json, drift-tested against
// models.yml by TestPiModelsConfigMatchesRegistry.
func buildPiArgs(model string, task *executor.Task, directive string) ([]string, error) {
	args := []string{
		"--mode", "json",
		"--model", model,
		"--no-session",
		"-p",
	}

	// Empty = send no dial at all; provider default thinking. Validated here
	// rather than passed through, because pi rejects an unknown level with a
	// usage dump that reads like a harness bug.
	if task.ReasoningEffort != "" {
		if !validPiThinkingLevels[task.ReasoningEffort] {
			return nil, fmt.Errorf("pi: invalid reasoning_effort %q (want one of off, minimal, low, medium, high, xhigh, max)", task.ReasoningEffort)
		}
		args = append(args, "--thinking", task.ReasoningEffort)
	}

	// AGENTS.md / CLAUDE.md discovery is ON by default in pi. A frozen mission stage must
	// not inherit whatever those files happen to say today.
	if task.IsolateFromAmbientContext {
		args = append(args, "--no-context-files")
	}

	// Both flags or neither — see isolationArgs.
	args = append(args, isolationArgs(task)...)

	switch {
	case task.AllowedTools == nil:
		// nil = caller does not specify; let pi's defaults apply.
	case len(task.AllowedTools) == 0:
		args = append(args, "--no-tools")
	default:
		args = append(args, "--tools", strings.Join(task.AllowedTools, ","))
	}

	args = append(args, directive)
	return args, nil
}

// validPiThinkingLevels is pi's --thinking vocabulary (cli/args.js
// VALID_THINKING_LEVELS). Wider than the registry's off/low/medium/high, so a
// models.yml value is always accepted; the extra two are pi-only.
var validPiThinkingLevels = map[string]bool{
	"off": true, "minimal": true, "low": true,
	"medium": true, "high": true, "xhigh": true, "max": true,
}
