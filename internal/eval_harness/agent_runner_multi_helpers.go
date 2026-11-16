package eval_harness

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/sunholo-data/ailang/internal/config"
	"github.com/sunholo-data/ailang/internal/eval_harness/langreg"
	"github.com/sunholo-data/ailang/internal/executor"
	"github.com/sunholo-data/ailang/internal/modelreg"
)

// Helpers split out of agent_runner_multi.go at the 800-line gate: the traps
// card, token-usage projection, row provenance and workspace setup.

// maybePrependTrapsCard front-loads the compact "dialect traps" card into the
// turn-1 task message — a tiny, un-buryable reminder of the highest-frequency
// rule violations, distilled from the 14-failure analysis.
//
// Default ON. The 2026-06-06 prompt-delivery experiment (local qwen3.5, n=2)
// showed the card sharply cuts flailing (symbolic_diff 1/2→2/2, 880k→246k
// tokens) by front-loading the import/syntax rules the model otherwise misses.
// It loads trapsCardDefaultPath unless AILANG_EVAL_TRAPS_CARD overrides the
// path; set AILANG_EVAL_TRAPS_CARD=off (or 0/false/no/none) to disable. If the
// card file is unreadable the directive is returned unchanged — this is an
// additive salience aid, not a data-integrity path.
func maybePrependTrapsCard(directive string) string {
	path := config.EvalTrapsCard()
	switch strings.ToLower(path) {
	case "off", "0", "false", "no", "none":
		return directive // explicitly disabled
	case "":
		path = trapsCardDefaultPath // default on
	}
	card, err := os.ReadFile(path)
	if err != nil {
		fmt.Fprintf(os.Stderr, "[eval] traps card unreadable (%s): %v — continuing without card\n", path, err)
		return directive
	}
	return strings.TrimRight(string(card), "\n") + "\n\n---\n\n" + directive
}

// persistentSystemPromptEnabled reports whether the FULL teaching prompt should
// be delivered via a persistent system-prompt channel (opencode AGENTS.md),
// re-injected every turn, instead of concatenated once into the first user
// message.
//
// Defaults to FALSE. The 2026-06-05/06 prompt-delivery experiment (local
// qwen3.5, n=2) showed re-injecting the full ~22k prompt every turn ("MOVE")
// was the WORST delivery — 1/6 vs 3/6 for turn-1 concatenation — because it
// bloated the context (up to 39 turns / 2.4M tokens) and the model lost the
// signal. Set AILANG_EVAL_PERSIST_PROMPT=1/true/on to re-enable for A/B testing.
func persistentSystemPromptEnabled() bool { return config.EvalPersistPrompt() }

// modelMaxOutputTokens returns the registry's declared max_output_tokens for a
// model (its per-request output strength), or 0 if unknown. Forwarded on the Task
// to executors that drive a separate runtime so a reasoning model isn't truncated
// mid-<think> by a small default (M-OLLAMA-PER-MODEL-MAX-TOKENS).
func modelMaxOutputTokens(modelName string) int {
	if modelreg.GlobalModelsConfig == nil {
		return 0
	}
	if m, err := modelreg.GlobalModelsConfig.GetModel(modelName); err == nil {
		return m.MaxOutputTokens
	}
	return 0
}

// tokenUsageFromResult maps executor token counts into the banked TokenUsage.
//
// This mapping is a proven silent-data-loss point. The standard path had the
// same shape and dropped reasoning tokens + finish_reason for every provider
// until 43333e7a8, which is why the whole v0.30.0 standard baseline banked
// cost figures that understate real spend (see
// eval_results/baselines/v0.30.0/CAVEATS.md). The agent path dropped them for
// even longer.
//
// It is a named function, not an inline literal, so the boundary itself is
// covered by a test rather than only the parsers feeding it — a field that is
// parsed correctly but never copied here is indistinguishable, in the banked
// data, from a field the provider never reported.
func tokenUsageFromResult(result *executor.Result) TokenUsage {
	if result == nil {
		return TokenUsage{}
	}
	return TokenUsage{
		InputTokens:              result.InputTokens,
		OutputTokens:             result.OutputTokens,
		ReasonTokens:             result.ReasonTokens,
		CacheReadInputTokens:     result.CacheReadInputTokens,
		CacheCreationInputTokens: result.CacheCreationInputTokens,
	}
}

// withProvenance copies the fields that say WHICH harness and lane a run
// happened on — executor version, effective tool policy, policy digest — onto
// a row. Provenance is not a measurement, so it belongs on failed and
// diagnostic rows too: a failure that cannot be attributed to a harness
// version or a tool lane is the un-annotated boundary this repo keeps paying
// for. Safe on a nil result.
//
// FinishReason rides along for the same reason: it says how the run ENDED, not
// what it measured, and the caller's categoriser reads it ahead of the error
// string. Without it an executor's typed kill (pi's thrash_aborted on the token
// WORK gate) reached the caller as a bare string and banked as api_error — 7
// rotation rows by 2026-10-02.
func withProvenance(row *AgentBenchmarkResult, res *executor.Result) *AgentBenchmarkResult {
	if res == nil {
		return row
	}
	row.FinishReason = res.FinishReason
	row.ExecutorVersion = res.ExecutorVersion
	row.ToolPolicy = res.ToolPolicy
	row.PolicyDigest = res.PolicyDigest
	row.EnvNamesDigest = res.EnvNamesDigest
	return row
}

// prepareAgentWorkspace writes every file the agent starts with into an
// existing workspace — the minimal .git folder, the benchmark's input files,
// the fmt-hook settings (ON arm only) and the solution placeholder — and
// returns the solution path. It is the live agent-mode workspace setup
// (RunAgentBenchmarkWithExecutor), so it is what the hidden-test leak guard
// (TestHiddenTests_NeverReachPromptOrWorkspace) drives.
func prepareAgentWorkspace(workspace string, spec *BenchmarkSpec, fmtHook FmtHookMode, language string) (string, error) {
	if err := os.MkdirAll(filepath.Join(workspace, ".git"), 0755); err != nil {
		return "", fmt.Errorf("failed to create .git folder: %w", err)
	}

	// Seed benchmark input files so the agent can actually run/test its solution
	// (e.g. cli_args reads numbers.txt). Mirrors the standard-runner layout.
	if err := seedInputFiles(workspace, spec); err != nil {
		return "", err
	}

	// Fmt-hook A/B (M-EVAL-FMT-WEAKMODEL-AB): this is the ACTIVE executor path.
	// When ON, write a workspace .claude/settings.json registering the LANDED
	// format_ail.sh PostToolUse hook. The claude executor sets cmd.Dir = workspace
	// and loads .claude/settings.json naturally (see internal/executor/claude:165),
	// so no --settings flag is needed here — writing the file IS the wiring. When
	// OFF, nothing is written: byte-identical to today's path. This is the ONLY
	// per-arm difference in the experiment.
	fmtSettingsPath, fmtSettingsJSON, fmtErr := fmtHook.Apply(workspace)
	if fmtErr != nil {
		return "", fmt.Errorf("fmt-hook setup failed: %w", fmtErr)
	}
	if fmtSettingsPath != "" {
		fmt.Fprintf(os.Stderr, "[FMT-HOOK] on (executor path): %s\n%s\n", fmtSettingsPath, string(fmtSettingsJSON))
	}

	// Create solution file placeholder
	var solutionPath, placeholder string
	if language == "ailang" {
		benchmarkDir := filepath.Join(workspace, "benchmark")
		if err := os.MkdirAll(benchmarkDir, 0755); err != nil {
			return "", fmt.Errorf("failed to create benchmark dir: %w", err)
		}
		solutionPath = filepath.Join(benchmarkDir, "solution.ail")
		placeholder = `module benchmark/solution
// DO NOT CHANGE THE MODULE DECLARATION ABOVE!
// TODO: Add your solution code below

`
	} else {
		lang, langErr := langreg.Get(language)
		if langErr != nil {
			return "", fmt.Errorf("unknown language %q: %w", language, langErr)
		}
		solutionPath = filepath.Join(workspace, lang.SolutionFilename())
		placeholder = "# TODO: Write your solution here\n"
	}
	if err := os.WriteFile(solutionPath, []byte(placeholder), 0644); err != nil {
		return "", fmt.Errorf("failed to create solution placeholder: %w", err)
	}
	return solutionPath, nil
}
