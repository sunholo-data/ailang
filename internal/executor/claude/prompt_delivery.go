package claude

import (
	"fmt"
	"io"
	"strings"

	"github.com/sunholo-data/ailang/internal/executor"
)

// Prompt delivery keeps every prompt byte OUT of the child's argv.
//
// argv is world-readable: `ps`, and therefore `pkill -f` / `pgrep -f`, match against it.
// The system prompt carries the whole AILANG teaching prompt, so every eval agent's argv
// used to contain "ailang run --entry main ...", and a Python run's carried "solution.py".
// Agents under test clean up their own hung test runs with `pkill -f "ailang run"` (and
// `pkill -f "solution.py"`), and that matched EVERY other agent CLI on the machine: every
// concurrent trial, every other eval-suite, and the launchd rotation's opencode. BSD pkill
// skips its own ancestors, so the agent that ran it survived and its siblings got SIGTERM —
// banked as `exit status 143` api_error rows, clustered to the second across processes.
// Measured 2026-10-27/28 on the rig: seven clusters, each a pkill in one eval agent's
// transcript, the kills 0.1-5s later.
//
// So the system prompt goes in a 0600 temp file (--system-prompt-file) and the directive
// on stdin (claude -p reads the prompt from stdin when none is given). The model receives
// the same bytes either way; only what the process table shows changes.

// promptDelivery is the argv fragment, the stdin and the cleanup for one run.
type promptDelivery struct {
	args    []string
	stdin   io.Reader
	cleanup func()
}

// deliverPrompt prepares task's prompts for a claude child without putting them on argv.
// cleanup must be called once the child has exited; it removes the system-prompt file.
func deliverPrompt(task *executor.Task) (*promptDelivery, error) {
	d := &promptDelivery{
		args:    []string{"-p"},
		stdin:   strings.NewReader(task.Directive),
		cleanup: func() {},
	}
	if task.SystemPrompt == "" {
		return d, nil
	}
	// Shared with the quorum's claude seat (internal/mission/quorum).
	path, cleanup, err := executor.WritePromptFile("ailang-claude-system-*.md", task.SystemPrompt)
	if err != nil {
		return nil, fmt.Errorf("claude: system prompt file: %w", err)
	}
	d.args = append(d.args, "--system-prompt-file", path)
	d.cleanup = cleanup
	return d, nil
}
