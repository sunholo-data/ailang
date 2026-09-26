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

// CostModel returns the registry rate card for the executor's configured
// model (M-V1-SIMPLIFY-S3 M2: this used to be gpt-5-codex's $1.25/$10 table
// applied to every codex-run model — `git log -S 'gpt-5-codex: $1.25/$10.00'`).
//
// FALLBACK ONLY. Result.CostUSD is billed via executor.ResolveCostModel, which
// prefers Task.Pricing (the per-model rates the harness resolved); this is
// reached only when a caller supplies none. A model the registry cannot
// resolve yields an explicit Unpriced card rather than another model's rates.
func (e *CodexExecutor) CostModel() *executor.CostModel {
	cm, err := executor.CostModelFor(e.model)
	if err != nil {
		return executor.UnpricedCostModel("openai", e.model)
	}
	return cm
}
