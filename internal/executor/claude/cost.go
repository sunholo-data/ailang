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

// CostModel returns pricing information for cost calculations.
//
// NOT used for Result.CostUSD: the claude CLI reports its own
// total_cost_usd and the executor banks that figure directly. Kept because
// the Executor interface requires it and callers may use it for pre-flight
// estimates. Audited 2026-07-30 — do not assume this table is what gets
// banked. Note the CLI's figure is itself a list-price equivalent when the
// rig authenticates via OAuth subscription, not metered spend.
func (e *ClaudeExecutor) CostModel() *executor.CostModel {
	// Default to Haiku pricing
	return &executor.CostModel{
		ProviderName:    "anthropic",
		InputTokenCost:  0.001,  // $1.00 per 1M
		OutputTokenCost: 0.005,  // $5.00 per 1M
		CacheReadCost:   0.0001, // $0.10 per 1M
	}
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
