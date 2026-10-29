package claude

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/sunholo-data/ailang/internal/executor"
)

// Registration, capabilities, health check and the finish-reason vocabulary:
// the executor surface around ExecuteStreaming (claude.go).

// normalizeClaudeFinishReason maps the Claude Code headless result subtype onto
// the executor.Result.FinishReason vocabulary.
//
// The subtype was already parsed and used for the success boolean, but the
// reason for a failure was discarded — so a run that burned through its turn
// budget banked identically to one that failed on the task itself.
// "error_max_turns" maps to "max_turns", which CategorizeAgentError already
// aliases to step_exhausted.
func normalizeClaudeFinishReason(subtype string) string {
	switch subtype {
	case "":
		return ""
	case "success":
		return "stop"
	case "error_max_turns":
		return "max_turns"
	case "error_during_execution":
		return "error"
	default:
		return strings.ToLower(subtype)
	}
}

// Capabilities returns the list of features this executor supports
func (e *ClaudeExecutor) Capabilities() []executor.Capability {
	return []executor.Capability{
		executor.CapStreaming,
		executor.CapToolControl,
		executor.CapSessionResume,
		executor.CapLocalWorkspace,
	}
}

// HealthCheck verifies the executor is configured and accessible
func (e *ClaudeExecutor) HealthCheck(ctx context.Context) error {
	// Check if claude binary exists
	_, err := exec.LookPath(e.claudePath)
	if err != nil {
		// Also try the resolved path directly (LookPath fails for absolute paths on some systems)
		if _, statErr := os.Stat(e.claudePath); statErr != nil {
			return fmt.Errorf("claude CLI not found: %w (install from https://claude.ai/code)", err)
		}
	}
	return nil
}

func Register() {
	executor.GlobalFactory().Register("claude", func(cfg *executor.Config) (executor.Executor, error) {
		return New(cfg)
	})
}

func init() {
	Register()
}
