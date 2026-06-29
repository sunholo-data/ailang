package main

import (
	"fmt"
	"os"

	"github.com/sunholo-data/ailang/internal/eval_harness"
)

// resolveEvalModelList determines the model list for an eval-suite run from the
// model-selection flags (--models / --full / default dev models) and enforces the
// agent-only safety guard. It exits the process on misconfiguration (an agent-only
// model selected without -agent would otherwise degrade silently to empty/garbage
// results — the 2026-05-23 incident: 102 trials, total_tokens=0, provider=ollama).
func resolveEvalModelList(models string, fullSuite, agent bool) []string {
	// Determine model list
	var modelList []string
	if models != "" {
		// User specified models explicitly. Recognize named suites as a
		// single token (e.g. --models agent_suite) and expand them to the
		// composite from models.yml. Otherwise fall back to comma-split.
		modelList = expandModelSuite(models, eval_harness.GlobalModelsConfig)
	} else if fullSuite {
		// Full suite: use extended suite (5 models) from models.yml
		if eval_harness.GlobalModelsConfig != nil && len(eval_harness.GlobalModelsConfig.ExtendedSuite) > 0 {
			modelList = eval_harness.GlobalModelsConfig.ExtendedSuite
		} else {
			// Fallback if models.yml not loaded
			modelList = []string{"gpt5-2-codex", "claude-opus-4-6", "claude-sonnet-4-6", "gemini-3-pro", "gemini-2-5-pro"}
		}
	} else {
		// Default: use dev models from models.yml
		if eval_harness.GlobalModelsConfig != nil && len(eval_harness.GlobalModelsConfig.DevModels) > 0 {
			modelList = eval_harness.GlobalModelsConfig.DevModels
		} else {
			// Fallback if models.yml not loaded
			modelList = []string{"gpt5-mini", "claude-haiku-4-5", "gemini-2-5-flash"}
		}
	}
	// SAFETY: block standard (direct-API) mode only for models that are GENUINELY
	// agent-only — i.e. have an agent_cli AND no cloud standard path (provider is
	// local/CLI-bound, e.g. ollama). Without this guard those degrade silently to
	// junk (the 2026-05-23 incident: 102 trials, total_tokens=0, provider=ollama).
	// Cloud models (anthropic/openai/google/openrouter) that ALSO have an agent_cli
	// — Claude, GPT, Gemini — are dual-mode and run standard natively; they must
	// NOT be blocked (they have hundreds of standard-mode runs across baselines).
	if !agent && eval_harness.GlobalModelsConfig != nil {
		var agentOnlyModels []string
		for _, m := range modelList {
			if eval_harness.GlobalModelsConfig.SupportsAgentEval(m) &&
				!eval_harness.GlobalModelsConfig.SupportsStandardEval(m) {
				cli, _ := eval_harness.GlobalModelsConfig.GetAgentCLI(m)
				agentOnlyModels = append(agentOnlyModels, fmt.Sprintf("%s (agent_cli: %q)", m, cli))
			}
		}
		if len(agentOnlyModels) > 0 {
			fmt.Fprintf(os.Stderr, "Error: model(s) require agent mode but -agent was not passed:\n")
			for _, m := range agentOnlyModels {
				fmt.Fprintf(os.Stderr, "  - %s\n", m)
			}
			fmt.Fprintf(os.Stderr, "\nThese models are only usable via their agent CLI; standard-mode runs would silently produce empty/garbage results.\n")
			fmt.Fprintf(os.Stderr, "Re-run with -agent and -benchmarks <list>, e.g.:\n")
			fmt.Fprintf(os.Stderr, "  ailang eval-suite -agent -benchmarks fizzbuzz,adt_option -models %s\n", modelList[0])
			os.Exit(1)
		}
	}
	return modelList
}
