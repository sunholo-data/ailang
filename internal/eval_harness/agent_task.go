// agent_task.go owns shared executor task assembly, below workspace lifecycle.
package eval_harness

import (
	"fmt"
	"github.com/google/uuid"
	"github.com/sunholo-data/ailang/internal/executor"
	"github.com/sunholo-data/ailang/internal/modelreg"
	"strings"
	"time"
)

// BuildEvalExecutorTask preserves the one-shot executor controls for retained
// stage callers. It does not create, reset or clean a workspace, dispatch a
// model, discover benchmarks or alter the supplied frozen prompt.
func BuildEvalExecutorTask(spec *BenchmarkSpec, config *MultiExecutorConfig, modelName, workspace, systemPrompt, taskPrompt string) (*executor.Task, error) {
	timeoutSeconds := config.TimeoutSeconds
	if spec.Timeout > 0 {
		timeoutSeconds = spec.Timeout
	}
	task := &executor.Task{
		ID:           fmt.Sprintf("%s_%s", spec.ID, uuid.New().String()[:8]),
		Directive:    taskPrompt,
		SystemPrompt: systemPrompt,
		// Teaching-prompt delivery (M-EVAL-OPENCODE-SYSTEM-PROMPT). The
		// 2026-06-06 delivery experiment found turn-1 concatenation + a small
		// front-loaded traps card beats re-injecting the full prompt every turn
		// via AGENTS.md ("MOVE", which bloated context and scored 1/6 vs 3/6).
		// So persistence now defaults OFF; set AILANG_EVAL_PERSIST_PROMPT=1 to
		// re-enable (opencode → AGENTS.md; executors without a persistent
		// channel ignore it).
		PersistentSystemPrompt: persistentSystemPromptEnabled(),
		Workspace:              workspace,
		Timeout:                time.Duration(timeoutSeconds) * time.Second,
		Model:                  modelName,
		Metadata:               buildChainMetadata(config.ChainID, config.StageID),
		MaxTokensPerBench:      config.MaxTokensPerBench,        // M-EVAL-OS-LONGITUDINAL Phase 1
		MaxOutputTokens:        modelMaxOutputTokens(modelName), // M-OLLAMA-PER-MODEL-MAX-TOKENS
		EvalShellGuard:         true,                            // per-command limit + no whole-disk find (2026-10-02)
	}
	// A lane run executes under its OWN policy: fs_sandbox = this run's
	// workspace (executor.MaterializeRunPolicy). Passing the operator's file
	// through unchanged pointed every run at one shared sandbox root.
	if config.PolicyPath != "" {
		runPolicy, perr := executor.MaterializeRunPolicy(config.PolicyPath, workspace)
		if perr != nil {
			return nil, perr
		}
		task.PolicyPath = runPolicy
	}

	// The ailang_only lane (M-AGENT-AILANG-ONLY-EXECUTION D5): a tool policy
	// is a harness boundary, so it is explicit, banked, and never a silent
	// default. An unknown profile fails the run before any model is called.
	if config.ToolPolicy != "" {
		tools, err := executor.ProfileTools(config.ToolPolicy)
		if err != nil {
			return nil, fmt.Errorf("tool-policy: %w", err)
		}
		task.AllowedTools = tools
	}
	// config.PolicyPath reaches the agent as AILANG_AGENT_POLICY through
	// task.PolicyPath (set above): BuildEnvironment derives it, and ExtraEnv
	// may not carry it. It used to ride ExtraEnv, where the agent_env block
	// below replaced the whole map and silently dropped it.

	// Export benchmark agent_env to the executor subprocess (M-EVAL-REIMPLEMENT-BENCH).
	// This is the ACTIVE executor path (RunAgentBenchmarkWithExecutor → exec.ExecuteStreaming);
	// the legacy RunHeadlessSessionStreaming injection never ran for executor-based agents
	// (motoko/opencode/pi), so MOTOKO_AST_AUTOREAD silently never reached them. ${WORKSPACE} →
	// the per-run workspace. BuildEnvironment merges task.ExtraEnv into the agent's env.
	if len(spec.AgentEnv) > 0 {
		if task.ExtraEnv == nil {
			task.ExtraEnv = make(map[string]string, len(spec.AgentEnv))
		}
		for k, v := range spec.AgentEnv {
			task.ExtraEnv[k] = strings.ReplaceAll(v, "${WORKSPACE}", workspace)
		}
	}

	// Apply per-model TTFT / generation timeouts from models.yml
	lookupKey := evalModelLookupKey(config)
	if modelreg.GlobalModelsConfig != nil {
		if cfg, ok := modelreg.GlobalModelsConfig.Models[lookupKey]; ok {
			if cfg.TTFTTimeoutSeconds > 0 {
				task.TTFTTimeout = time.Duration(cfg.TTFTTimeoutSeconds) * time.Second
			}
			if cfg.GenerationTimeoutSeconds > 0 {
				task.IdleTimeout = time.Duration(cfg.GenerationTimeoutSeconds) * time.Second
			}
			// M-OLLAMA-PER-MODEL-MAX-TOKENS (fix 2026-06-19): resolve
			// max_output_tokens via the SAME registry key as TTFT (lookupKey =
			// ConfigKey) rather than `modelName` (the agent display name, e.g.
			// "ollama/qwen3.6:..."), which is NOT a registry key — GetModel(modelName)
			// missed → returned 0, so the per-model 32768 budget never reached
			// motoko and the wire stayed at the 16384 floor.
			if cfg.MaxOutputTokens > 0 {
				task.MaxOutputTokens = cfg.MaxOutputTokens
			}
			// reasoning_effort reached standard mode (ai_agent.go) but was
			// dropped on the agent path, so an entry could declare a dial that
			// never left the registry. Empty stays empty: no reasoning control
			// is sent and the model thinks at its provider default.
			task.ReasoningEffort = cfg.ReasoningEffort
			if cfg.Provider == "openrouter" && modelreg.GlobalModelsConfig != nil {
				task.OpenRouterProvider = modelreg.GlobalModelsConfig.EffectiveOpenRouterProvider(cfg.APIName)
			}
			if cfg.GCPProject != "" {
				task.GCPProject = cfg.GCPProject
			}
			if cfg.GCPLocation != "" {
				task.GCPLocation = cfg.GCPLocation
			}
			if cfg.MotokoProfile != "" {
				if task.Metadata == nil {
					task.Metadata = make(map[string]string)
				}
				task.Metadata["motoko_profile"] = cfg.MotokoProfile
			}
			// M-EVAL-COST-AND-SPEED-BUDGETS (v0.16.0): populate Task.Budget
			// from the resolved per-model cost ceiling. Wall-clock Timeout
			// is bumped to the budgets:hard_timeout_secs default (600s) to
			// give cost the primary gate; cost.go does the kill-on-exceed.
			if maxCost := cfg.ResolvedMaxCostUSD(); maxCost > 0 {
				task.Budget = executor.NewCostBudget(maxCost, cfg.Pricing.InputPer1K, cfg.Pricing.OutputPer1K)
			}
			// Report cost at the SAME per-model rates the budget enforces at.
			// Set unconditionally: Budget exists only when enforcement is on
			// (maxCost > 0), but a cost_usd is banked either way, and before
			// this the two came from different price tables — a codex row could
			// bank $0.34 while the budget that spared it saw $0.27.
			//
			// Through the one adapter, never a hand copy: CacheReadCost was
			// omitted until 2026-09-02 and CacheWriteCost until 2026-09-15,
			// each silently zeroing that rate for EVERY agent-mode row —
			// this struct overrides the executor's own CostModel via
			// ResolveCostModel, so an executor that priced the cache
			// correctly had that pricing discarded here.
			task.Pricing = executor.CostModelFromPricing(cfg.Provider, lookupKey, cfg.Pricing)
			// The WORK gate. A per-model budgets:max_tokens_per_bench overrides
			// the global --max-tokens-per-bench flag; this is what makes the
			// agent suite comparable, because the dollar gate above buys work
			// in inverse proportion to price (see Budgets doc).
			task.MaxTokensPerBench = cfg.ResolvedMaxTokensPerBench(config.MaxTokensPerBench)
			// M-EVAL-LOCAL-OLLAMA (v0.22.0): take the MAX of spec.Timeout and
			// model.HardTimeoutSecs rather than letting spec.Timeout veto. The
			// benchmark spec timeout is cloud-tuned (Sonnet 4.6 speeds); local
			// Ollama models need their own slower budget to apply. For cloud
			// models with cost budgets, the wall-clock bump is a no-op because
			// cost trips first; for local models (pricing=0, no cost gate) this
			// is the only gate, so it must reflect the model's actual speed.
			specT := spec.Timeout
			hardSecs := cfg.ResolvedHardTimeoutSecs()
			effective := specT
			if hardSecs > effective {
				effective = hardSecs
			}
			if effective > 0 {
				task.Timeout = time.Duration(effective) * time.Second
			}
		}
	}

	return task, nil
}

func evalModelLookupKey(config *MultiExecutorConfig) string {
	if config.ConfigKey != "" {
		return config.ConfigKey
	}
	return config.ModelName
}
