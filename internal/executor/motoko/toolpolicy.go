package motoko

import (
	"fmt"
	"strings"

	"github.com/sunholo-data/ailang/internal/executor"
)

// nativeTools is the canonical tool surface motoko ALWAYS exposes to the
// model: its native catalogue (BashExec, ReadFile, WriteFile, EditFile,
// Search) is advertised on every run and no profile can remove an entry
// (mk-main src/core/tool_catalog.ail; ToolConfig has no allowlist). ABI 8.0
// extensions can add tools and deny a call, but cannot hide a native one.
var nativeTools = []string{"Bash", "Read", "Write", "Edit", "Grep"}

// checkToolPolicy refuses a task whose tool restriction motoko cannot honour.
//
// Task.AllowedTools nil means "the executor's default" and is always fine. A
// list that includes every native tool is fine too: motoko runs with exactly
// that surface (this is what the eval harness passes for every agent run).
// A list that OMITS a native tool asks for it to be forbidden, and motoko
// would run with it anyway. Before this check that happened silently: an
// ailang_only task (no Bash, no native file tools) or a read-only question
// task ran with bash and write access, and the result banked ToolPolicy as
// nil ("unmeasured"). A program-policy file (Task.PolicyPath) is refused for
// the same reason: motoko does not route its tools through it yet.
//
// Lift this once motoko-ext-ailang-policy enforces the lane and motoko core can
// hide native tools (design: m-motoko-ailang-only-lane).
func checkToolPolicy(task *executor.Task) error {
	if task.PolicyPath != "" {
		return fmt.Errorf("motoko cannot run under a program policy yet (task.PolicyPath=%s): its tools do not go through `ailang policy-tool` / `ailang run --policy`, so the boundary would not hold. Use the pi executor for the ailang_only lane", task.PolicyPath)
	}
	if task.AllowedTools == nil {
		return nil
	}
	allowed := make(map[string]bool, len(task.AllowedTools))
	for _, t := range task.AllowedTools {
		allowed[t] = true
	}
	var forbidden []string
	for _, t := range nativeTools {
		if !allowed[t] {
			forbidden = append(forbidden, t)
		}
	}
	if len(forbidden) > 0 {
		return fmt.Errorf("motoko cannot honour this task's tool restriction: it always exposes %s, but the task forbids %s (allowed: %s). Refusing rather than running with wider tools than asked. Use the pi executor for restricted tool policies such as ailang_only",
			strings.Join(nativeTools, ", "), strings.Join(forbidden, ", "), strings.Join(task.AllowedTools, ", "))
	}
	return nil
}

// effectiveToolPolicy is the Result.ToolPolicy for a run checkToolPolicy let
// through: the native surface motoko actually exposed (extensions may add to it).
func effectiveToolPolicy() []string {
	return append([]string(nil), nativeTools...)
}
