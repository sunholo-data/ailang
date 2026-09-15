package codex

import (
	"context"
	"fmt"
	"os"
	"os/exec"

	"github.com/sunholo-data/ailang/internal/executor"
)

// The executor.Executor surface that is not the streaming turn itself:
// capabilities, health, model resolution and factory registration
// (M-V1-SIMPLIFY-S4 M4 moved these out of codex.go at the 800-line gate).

// Capabilities returns the list of features this executor supports.
func (e *CodexExecutor) Capabilities() []executor.Capability {
	return []executor.Capability{
		executor.CapStreaming,
		executor.CapLocalWorkspace,
		executor.CapMCP,
	}
}

// HealthCheck verifies the codex binary exists on PATH and responds.
func (e *CodexExecutor) HealthCheck(ctx context.Context) error {
	codexPath := e.codexPath
	if _, err := exec.LookPath(codexPath); err != nil {
		if _, statErr := os.Stat(codexPath); statErr != nil {
			return fmt.Errorf("codex CLI not found: %w (install with: npm i -g @openai/codex)", err)
		}
	}
	checkCmd := exec.CommandContext(ctx, codexPath, "--version")
	configureProcessTree(checkCmd)
	if err := checkCmd.Run(); err != nil {
		return fmt.Errorf("codex --version failed: %w", err)
	}
	// Auth comes from ~/.codex/auth.json, written by `codex login`. OPENAI_API_KEY
	// in the environment does NOT override it — probe-verified 2026-07-30 against
	// codex-cli 0.145.0 with auth_mode "chatgpt": a deliberately invalid key in the
	// env still ran clean. So its absence is not a warning condition, and its
	// presence is not proof that runs are metered (this rig is on a ChatGPT
	// subscription, where cost_usd is a list-price equivalent, never billed spend).
	if os.Getenv("DEBUG_AGENT") != "" {
		fmt.Fprintf(os.Stderr, "[DEBUG_CODEX] auth: ~/.codex/auth.json (codex login); OPENAI_API_KEY is not consulted\n")
	}
	return nil
}

// Close releases any resources held by the executor.
func (e *CodexExecutor) Close() error {
	return nil
}

func (e *CodexExecutor) getModel(task *executor.Task) string {
	if task.Model != "" {
		return task.Model
	}
	return e.model
}

// requireModel is the D2(a) fail-loud point (M-MODEL-REGISTRY-SINGLE-SOURCE M6).
//
// The check lives at the ENTRY to execution rather than at construction,
// because the coordinator builds an executor before it knows the task and
// then supplies Task.Model per task — rejecting an empty model at
// construction would break the normal path. It lives here rather than inside
// getModel to avoid threading an error through every call site of a helper
// that runs after this guard has already passed.
func (e *CodexExecutor) requireModel(task *executor.Task) error {
	if e.getModel(task) == "" {
		return executor.ErrUnresolvedModel("codex", "CodexModel")
	}
	return nil
}

// Register registers the Codex executor with the global factory.
func Register() {
	executor.GlobalFactory().Register("codex", func(cfg *executor.Config) (executor.Executor, error) {
		return New(cfg)
	})
}

func init() {
	Register()
}
