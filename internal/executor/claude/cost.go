package claude

import (
	"os"

	"github.com/sunholo-data/ailang/internal/executor"
)

// Cost, auth-lane and model resolution for the claude harness.
//
// Split out of claude.go 2026-07-31 when the executor crossed the 800-line
// check-file-sizes gate. Grouped because all three answer "which model ran,
// under which account, at what price".

// authLane reports whether claude runs are charged per token.
//
// Mirrors the M-CLOUD-DUAL-AUTH branch in Execute: AILANG_AUTH_MODE=apikey means
// ANTHROPIC_API_KEY drives a metered account; anything else is the OAuth
// subscription lane, where the CLI still emits a non-zero total_cost_usd that
// nobody is charged. On the eval rig the key is deliberately stripped, so the
// default is the common case, not an edge case.
func (e *ClaudeExecutor) authLane() executor.AuthLane {
	if os.Getenv("AILANG_AUTH_MODE") == "apikey" {
		return executor.AuthLaneBilled
	}
	return executor.AuthLaneSubscription
}

// CostModel returns the registry rate card for the executor's configured
// model (M-V1-SIMPLIFY-S3 M2: this used to be a hard-coded Haiku table
// applied to every Claude model — `git log -S 'Default to Haiku pricing'`).
//
// NOT used for Result.CostUSD: the claude CLI reports its own
// total_cost_usd and the executor banks that figure directly. Kept because
// the Executor interface requires it and callers may use it for pre-flight
// estimates. Note the CLI's figure is itself a list-price equivalent when the
// rig authenticates via OAuth subscription, not metered spend.
//
// A model the registry cannot resolve — the CLI short names "haiku"/"sonnet"
// /"opus" are deliberately NOT aliases, since what they point at moves with
// CLI releases — yields an explicit Unpriced card, never Haiku's rates.
func (e *ClaudeExecutor) CostModel() *executor.CostModel {
	cm, err := executor.CostModelFor(e.model)
	if err != nil {
		return executor.UnpricedCostModel("anthropic", e.model)
	}
	return cm
}

// Close releases any resources held by the executor
func (e *ClaudeExecutor) Close() error {
	return nil
}

func (e *ClaudeExecutor) getModel(task *executor.Task) string {
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
func (e *ClaudeExecutor) requireModel(task *executor.Task) error {
	if e.getModel(task) == "" {
		return executor.ErrUnresolvedModel("claude", "ClaudeModel")
	}
	return nil
}

// claudeHeadlessResult matches Claude CLI output structure
type claudeHeadlessResult struct {
	Type         string      `json:"type"`
	Subtype      string      `json:"subtype"`
	IsError      bool        `json:"is_error"`
	Result       string      `json:"result"`
	NumTurns     int         `json:"num_turns"`
	DurationMS   int         `json:"duration_ms"`
	TotalCostUSD float64     `json:"total_cost_usd"`
	SessionID    string      `json:"session_id"`
	Usage        claudeUsage `json:"usage"`
}

type claudeUsage struct {
	InputTokens              int `json:"input_tokens"`
	OutputTokens             int `json:"output_tokens"`
	CacheReadInputTokens     int `json:"cache_read_input_tokens"`
	CacheCreationInputTokens int `json:"cache_creation_input_tokens"`
}

// Register registers the Claude executor with the global factory
