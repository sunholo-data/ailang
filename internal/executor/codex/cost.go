package codex

import (
	"encoding/json"
	"os"
	"path/filepath"

	"github.com/sunholo-data/ailang/internal/executor"
)

// Cost and auth-lane resolution for the codex harness.
//
// Split out of codex.go 2026-07-31 when the executor crossed the 800-line
// check-file-sizes gate. These two concerns belong together: what a run costs
// and whether anyone was charged for it are the same question asked twice.

// authLane reports whether codex runs are charged per token.
//
// Codex authenticates from ~/.codex/auth.json, written by `codex login`. An
// OPENAI_API_KEY in the environment does NOT override it — probe-verified
// 2026-07-30 against codex-cli 0.145.0 with auth_mode "chatgpt", where a
// deliberately invalid env key still ran clean. Reading the env var here would
// therefore report "billed" for a run the ChatGPT plan covered.
//
// Unreadable or unrecognised → Unknown. A wrong "metered" is the failure mode
// this exists to prevent, so it is never the fallback.
func (e *CodexExecutor) authLane() executor.AuthLane {
	home, err := os.UserHomeDir()
	if err != nil {
		return executor.AuthLaneUnknown
	}
	data, err := os.ReadFile(filepath.Join(home, ".codex", "auth.json"))
	if err != nil {
		return executor.AuthLaneUnknown
	}
	var auth struct {
		AuthMode string `json:"auth_mode"`
		APIKey   string `json:"OPENAI_API_KEY"`
	}
	if err := json.Unmarshal(data, &auth); err != nil {
		return executor.AuthLaneUnknown
	}
	switch auth.AuthMode {
	case "chatgpt":
		return executor.AuthLaneSubscription
	case "apikey":
		return executor.AuthLaneBilled
	}
	// Older codex releases wrote the key with no auth_mode discriminator.
	if auth.APIKey != "" {
		return executor.AuthLaneBilled
	}
	return executor.AuthLaneUnknown
}

// CostModel returns pricing for gpt-5-codex (the default Codex model).
// Source: https://platform.openai.com/docs/pricing
// gpt-5-codex: $1.25/$10.00 per 1M tokens = $0.00125/$0.01 per 1K.
//
// FALLBACK ONLY. The codex CLI runs whatever `--model` it is handed, so this
// table is correct for exactly one of them. Result.CostUSD is billed via
// executor.ResolveCostModel, which prefers Task.Pricing (the per-model rates
// from models.yml). This is reached only when a caller supplies no pricing.
func (e *CodexExecutor) CostModel() *executor.CostModel {
	return &executor.CostModel{
		ProviderName:    "openai",
		InputTokenCost:  0.00125,
		OutputTokenCost: 0.01,
		CacheReadCost:   0.000125,
	}
}
