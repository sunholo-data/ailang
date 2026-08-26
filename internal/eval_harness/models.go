package eval_harness

import (
	_ "embed"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"

	"github.com/sunholo-data/ailang/internal/executor"
)

//go:embed models.yml
var embeddedModelsYAML []byte

// ModelConfig represents a single model configuration
type ModelConfig struct {
	APIName                  string  `yaml:"api_name"`
	Provider                 string  `yaml:"provider"`
	Description              string  `yaml:"description"`
	EnvVar                   string  `yaml:"env_var"`
	AgentCLI                 *string `yaml:"agent_cli"`            // CLI command for agent eval (e.g., "claude", "openai", "gemini"), nil if not supported
	AgentModelName           *string `yaml:"agent_model_name"`     // Model name to pass to agent CLI (e.g., "haiku", "sonnet")
	MaxOutputTokens          int     `yaml:"max_output_tokens"`    // Max output tokens (0 = handler default 4096)
	DefaultThinking          string  `yaml:"default_thinking"`     // Thinking state when the harness sends NO thinking control (M-EVAL-TOKEN-HEADROOM). One of: "on" (thinks by default), "off" (needs an explicit ask), "always_on" (cannot be disabled — explicit disable is an API error), "none" (no thinking capability), "unknown" (NOT verified). Required on every entry: this is the column whose absence let GLM-5.2's truncated thinking read as a capability regression for a month. A row marked "unknown" must NOT have its token counts read as efficiency data
	ReasoningMaxTokens       int     `yaml:"reasoning_max_tokens"` // Cap on hidden thinking tokens (0 = provider default / uncapped). OpenRouter only; best-effort — third-party upstreams may ignore it (observed: glm-5.2 via Baidu/StreamLake)
	ReasoningEffort          string  `yaml:"reasoning_effort"`     // Vendor effort dial ("low"|"medium"|"high"; empty = vendor default). OpenRouter reasoning.effort — the DOCUMENTED control for effort-capable models (e.g. kimi-k3 Low/Standard/High/Max). Reasoning bills as output; record explicitly for eval reproducibility when deviating from default
	TTFTTimeoutSeconds       int     `yaml:"ttft_timeout"`         // Prefill budget in seconds (0 = executor default 30s)
	GenerationTimeoutSeconds int     `yaml:"generation_timeout"`   // Per-token idle budget after first event (0 = executor default 3m)
	ModelFamily              string  `yaml:"model_family"`         // Logical model family for cross-harness grouping (e.g., "claude-sonnet-4-6"); empty = no grouping
	GCPProject               string  `yaml:"gcp_project"`          // Override GOOGLE_CLOUD_PROJECT for this model's evals (e.g. "ailang-dev")
	GCPLocation              string  `yaml:"gcp_location"`         // Override GOOGLE_CLOUD_LOCATION (e.g. "us-central1")
	MotokoProfile            string  `yaml:"motoko_profile"`       // Override MOTOKO_CONFIG profile (default: "dogfood"); used when agent_cli is "motoko"
	Pricing                  Pricing `yaml:"pricing"`
	Budgets                  Budgets `yaml:"budgets"` // M-EVAL-COST-AND-SPEED-BUDGETS (v0.16.0): cost-aware budget overrides
	Notes                    string  `yaml:"notes"`
}

// Pricing represents model pricing information
type Pricing struct {
	InputPer1K  float64 `yaml:"input_per_1k"`
	OutputPer1K float64 `yaml:"output_per_1k"`
	// CacheReadPer1K prices prompt-cache READ tokens, which every major provider
	// bills far below fresh input (OpenRouter ~20% of input; deepseek-v4-flash
	// $0.016-0.028/M vs $0.08-0.14/M depending on the upstream host).
	//
	// Zero means "no cache rate declared", and the cost helpers then bill cache
	// reads at the FULL input rate rather than silently at $0 — an overstatement
	// is visible in a budget, whereas a $0 line looks like caching is free and
	// hides both the spend and a broken cache. Declare it wherever it is known.
	CacheReadPer1K float64 `yaml:"cache_read_per_1k"`

	// Expires is the last date (INCLUSIVE, "YYYY-MM-DD") on which the rates
	// above are the ones actually billed. Empty means "no scheduled change
	// known", which is the normal case.
	//
	// This exists because introductory pricing is a real and silent failure
	// mode: on 2026-08-13 Google launched Gemini 3.7 Flash at $0.75/$3.75 per
	// 1M with the rates DOUBLING to $1.50/$7.50 on 2027-01-01. Before this
	// field, that reversion lived only in a YAML comment — so on New Year's Day
	// every Gemini Flash cost figure would quietly become half the true spend,
	// with no test going red and nothing to notice. A date the checker can read
	// turns a comment nobody re-reads into a gate.
	//
	// Enforced offline by TestModels_PricingScheduleIsHonoured (so it fails in
	// CI on the day, with no network) and online by `make verify-model-pricing`.
	Expires string `yaml:"expires"`

	// Next is the schedule of rates that take effect the day AFTER Expires.
	// Required whenever Expires is set, and meaningless without it: recording
	// only that a price lapses, without what it lapses TO, leaves the row
	// failing with no way to fix it except re-researching the vendor's page.
	Next *ScheduledPricing `yaml:"next"`
}

// ScheduledPricing is a future rate card for a row whose current price is known
// to expire — see Pricing.Expires. It is deliberately NOT a *Pricing: a schedule
// that could itself carry a schedule invites a chain of future prices nobody has
// verified, and one hop is all any vendor has ever actually announced.
type ScheduledPricing struct {
	InputPer1K     float64 `yaml:"input_per_1k"`
	OutputPer1K    float64 `yaml:"output_per_1k"`
	CacheReadPer1K float64 `yaml:"cache_read_per_1k"`
}

// Budgets represents per-model cost-and-speed budget overrides
// (M-EVAL-COST-AND-SPEED-BUDGETS, v0.15.1).
//
// Defaults when omitted:
//
//	MaxCostUSD          = min($0.50, input_per_1k × 64 + output_per_1k × 32)
//	HardTimeoutSecs     = 600  (was 60-180s before this milestone)
//	ExpectedTTFTSecs    = 30   (legacy default, used for "abnormally slow" alerts)
//	ExpectedTTFSolutionSecs = 0  (no alert if 0)
//
// Cost is the primary gate; wall-clock HardTimeoutSecs is a safety net for
// hung connections, not a cost proxy.
//
// MaxTokensPerBench is the WORK gate, and it exists because a dollar gate is
// not one. A dollar ceiling buys work in inverse proportion to model price, so
// a flat $0.30 across the agent suite spanned 24x in actual tokens (audited
// 2026-07-31): claude-sonnet-4-6 — the longitudinal ANCHOR — got 0.14M tokens
// while opencode-or-deepseek-v4-flash got 3.40M. That silently handed the
// weakest models the most iteration and the reference model the least, which
// is backwards for a suite whose question is "does the agent loop rescue weak
// models". It also means a vendor price change silently re-scopes the gate,
// which is exactly what OpenAI's 2026-07-30 cut did to gpt5-6-luna.
//
// Policy (Mark, 2026-07-31): subscription lanes (codex, claude-on-OAuth) gate
// on TOKENS alone — equal work, and no spend to control. Metered lanes keep a
// dollar ceiling as a real spend control WITH the token gate as a second bound;
// whichever binds first wins. Their comparison stays confounded, but for a
// stated reason rather than by accident.
type Budgets struct {
	MaxCostUSD              float64 `yaml:"max_cost_usd"`
	MaxTokensPerBench       int     `yaml:"max_tokens_per_bench"`
	HardTimeoutSecs         int     `yaml:"hard_timeout_secs"`
	ExpectedTTFTSecs        int     `yaml:"expected_ttft_secs"`
	ExpectedTTFSolutionSecs int     `yaml:"expected_ttf_solution_secs"`
}

// ResolvedMaxTokensPerBench returns the effective cumulative-token ceiling for
// a model: the explicit per-model budget when set, otherwise flagFallback (the
// global --max-tokens-per-bench value, 0 = unlimited).
//
// Per-model wins because the flag is one number for a whole run, and the point
// of the token gate is that every model in a cohort gets the SAME work — which
// a shared flag already gives, but a mixed cohort (cloud + local) may want to
// split. 0 from both = no token enforcement (legacy behaviour).
func (m *ModelConfig) ResolvedMaxTokensPerBench(flagFallback int) int {
	if m.Budgets.MaxTokensPerBench > 0 {
		return m.Budgets.MaxTokensPerBench
	}
	return flagFallback
}

// ResolvedMaxCostUSD returns the effective cost ceiling for a model:
// the explicit Budgets.MaxCostUSD when set, otherwise the default formula.
// Returns 0 if pricing is also zero (free local models — no enforcement).
func (m *ModelConfig) ResolvedMaxCostUSD() float64 {
	if m.Budgets.MaxCostUSD > 0 {
		return m.Budgets.MaxCostUSD
	}
	// Default formula: min($0.50, input × 64 + output × 32)
	formula := m.Pricing.InputPer1K*64.0 + m.Pricing.OutputPer1K*32.0
	const ceilingUSD = 0.50
	if formula < ceilingUSD {
		return formula
	}
	return ceilingUSD
}

// ResolvedHardTimeoutSecs returns the effective wall-clock safety net.
// Default raised from 60-180s to 600s in v0.15.1 because cost is now the
// primary gate.
func (m *ModelConfig) ResolvedHardTimeoutSecs() int {
	if m.Budgets.HardTimeoutSecs > 0 {
		return m.Budgets.HardTimeoutSecs
	}
	return 600
}

// ModelsConfig represents the entire models.yml configuration
type ModelsConfig struct {
	Models           map[string]ModelConfig `yaml:"models"`
	Default          string                 `yaml:"default"`
	BenchmarkSuite   []string               `yaml:"benchmark_suite"`
	ExtendedSuite    []string               `yaml:"extended_suite"`
	DevModels        []string               `yaml:"dev_models"`
	AgentSuite       []string               `yaml:"agent_suite"`
	OllamaSuite      []string               `yaml:"ollama_suite"`
	HarnessSuite     []string               `yaml:"harness_suite"`
	LangHarnessSuite []string               `yaml:"lang_harness_suite"`
}

var (
	// GlobalModelsConfig is the loaded models configuration
	GlobalModelsConfig *ModelsConfig
)

// LoadModelsConfig loads the models.yml configuration
func LoadModelsConfig(path string) (*ModelsConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read models config: %w", err)
	}

	var config ModelsConfig
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse models YAML: %w", err)
	}

	return &config, nil
}

// InitModelsConfig loads the global models configuration
func InitModelsConfig() error {
	// Try embedded models.yml first (available in installed binary)
	if len(embeddedModelsYAML) > 0 {
		var config ModelsConfig
		if err := yaml.Unmarshal(embeddedModelsYAML, &config); err == nil {
			GlobalModelsConfig = &config
			return nil
		}
		// If embedded parse fails, fall through to file system
	}

	// Fall back to file system (for development)
	paths := []string{
		"internal/eval_harness/models.yml",
		"../internal/eval_harness/models.yml",
		"models.yml", // If already in the same directory
	}

	var lastErr error
	for _, path := range paths {
		config, err := LoadModelsConfig(path)
		if err == nil {
			GlobalModelsConfig = config
			return nil
		}
		lastErr = err
	}

	return fmt.Errorf("failed to load models config from any path: %w", lastErr)
}

// GetModel returns the configuration for a model by friendly name
func (c *ModelsConfig) GetModel(name string) (*ModelConfig, error) {
	model, ok := c.Models[name]
	if !ok {
		return nil, fmt.Errorf("model %s not found in configuration", name)
	}
	return &model, nil
}

// GetAPIName returns the API name for a model by friendly name
func (c *ModelsConfig) GetAPIName(name string) (string, error) {
	model, err := c.GetModel(name)
	if err != nil {
		return "", err
	}
	return model.APIName, nil
}

// GetProvider returns the provider for a model
func (c *ModelsConfig) GetProvider(name string) (string, error) {
	model, err := c.GetModel(name)
	if err != nil {
		return "", err
	}
	return model.Provider, nil
}

// IsOllamaCloudRoute reports whether a name selects an Ollama CLOUD model.
// Thin delegate to executor.IsOllamaCloudRoute — the grammar lives in the lower
// layer because cost provenance needs it too, and two copies could disagree.
func IsOllamaCloudRoute(name string) bool { return executor.IsOllamaCloudRoute(name) }

// UsesLocalGPU reports whether a model runs on the local Ollama GPU — the
// shared single-GPU rig that the rig lock protects. Cloud/API models return
// false. Unknown model names return false: they fail model validation later,
// and an ad-hoc cloud model name must not grab the rig lock.
//
// Ollama-provider rows are NOT automatically local: a `-cloud`-suffixed row is
// proxied to ollama.com and shares nothing with the rig, so serializing it
// behind the GPU lock buys nothing and costs wall-clock (M-OLLAMA-CLOUD-PROVIDER
// D4). The port-8080 concern for concurrent *motoko* runs is a separate
// constraint and deliberately not conflated with this one — motoko pins that
// port for local and cloud rows alike, so it is not a GPU question.
func (c *ModelsConfig) UsesLocalGPU(name string) bool {
	model, err := c.GetModel(name)
	if err != nil {
		return false
	}
	if IsOllamaCloudRoute(model.APIName) {
		return false
	}
	if model.AgentModelName != nil && IsOllamaCloudRoute(*model.AgentModelName) {
		return false
	}
	if model.Provider == "ollama" {
		return true
	}
	return model.AgentModelName != nil && strings.HasPrefix(*model.AgentModelName, "ollama/")
}

// GetEnvVar returns the environment variable name for a model's API key
func (c *ModelsConfig) GetEnvVar(name string) (string, error) {
	model, err := c.GetModel(name)
	if err != nil {
		return "", err
	}
	return model.EnvVar, nil
}

// CalculateCostForModel calculates the cost for a model using its pricing config
func (c *ModelsConfig) CalculateCostForModel(name string, inputTokens, outputTokens int) (float64, error) {
	model, err := c.GetModel(name)
	if err != nil {
		// NO FALLBACK - return error to caller
		// This prevents infinite recursion and silent failures
		return 0.0, err
	}

	inputCost := float64(inputTokens) / 1000.0 * model.Pricing.InputPer1K
	outputCost := float64(outputTokens) / 1000.0 * model.Pricing.OutputPer1K

	return inputCost + outputCost, nil
}

// CalculateCostForModelWithCache prices a run whose prompt-cache activity is known.
//
// inputTokens must be FRESH input only — disjoint from cacheReadTokens. Callers
// holding a cache-inclusive total must subtract before calling, or they will pay
// twice for the same tokens.
//
// Why this exists (2026-08-11): agent mode folded cache reads into InputTokens for
// reporting while costing the cache-exclusive value, so the banked input_tokens and
// cost_usd described different quantities and the documented "~25-30% undercount"
// could not be checked from banked data. Measured on the pi/OpenRouter lane: an
// unpinned deepseek-v4-flash call cached 0 of ~27.7k prompt tokens, the same call
// price-pinned cached 27,392 — a 4.8x cost difference that was invisible.
//
// A model with no declared cache rate bills cache reads at the full input rate:
// overstating is visible, whereas $0 hides both the spend and a broken cache.
func (c *ModelsConfig) CalculateCostForModelWithCache(name string, inputTokens, outputTokens, cacheReadTokens int) (float64, error) {
	model, err := c.GetModel(name)
	if err != nil {
		// NO FALLBACK - same stance as CalculateCostForModel.
		return 0.0, err
	}

	cacheRate := model.Pricing.CacheReadPer1K
	if cacheRate == 0 {
		cacheRate = model.Pricing.InputPer1K
	}

	inputCost := float64(inputTokens) / 1000.0 * model.Pricing.InputPer1K
	cacheCost := float64(cacheReadTokens) / 1000.0 * cacheRate
	outputCost := float64(outputTokens) / 1000.0 * model.Pricing.OutputPer1K

	return inputCost + cacheCost + outputCost, nil
}

// SupportsAgentEval returns true if the model supports agent-based evaluation
func (c *ModelsConfig) SupportsAgentEval(name string) bool {
	model, err := c.GetModel(name)
	if err != nil {
		return false
	}
	return model.AgentCLI != nil && *model.AgentCLI != ""
}

// SupportsStandardEval returns true if the model can be evaluated in standard
// (direct-API) mode. A model is standard-capable when its provider is a cloud
// HTTP provider the standard runner can reach (anthropic, openai, google,
// openrouter). Local/CLI-bound providers (ollama) are NOT standard-capable in
// practice: running them via the direct-API path silently degrades to junk
// (the 2026-05-23 incident: provider=ollama, total_tokens=0). A model can have
// an agent_cli AND still be standard-capable — Claude/GPT/Gemini have both paths.
func (c *ModelsConfig) SupportsStandardEval(name string) bool {
	model, err := c.GetModel(name)
	if err != nil {
		return false
	}
	switch model.Provider {
	case "anthropic", "openai", "google", "gemini", "vertex", "openrouter":
		return true
	default:
		// ollama (local) and unknown providers are agent-only in practice.
		return false
	}
}

// GetAgentCLI returns the agent CLI command for a model (e.g., "claude")
func (c *ModelsConfig) GetAgentCLI(name string) (string, error) {
	model, err := c.GetModel(name)
	if err != nil {
		return "", err
	}
	if model.AgentCLI == nil || *model.AgentCLI == "" {
		return "", fmt.Errorf("model %s does not support agent evaluation", name)
	}
	return *model.AgentCLI, nil
}

// GetAgentModelName returns the model name to pass to the agent CLI
func (c *ModelsConfig) GetAgentModelName(name string) (string, error) {
	model, err := c.GetModel(name)
	if err != nil {
		return "", err
	}
	if model.AgentModelName == nil || *model.AgentModelName == "" {
		return "", fmt.Errorf("model %s does not have an agent_model_name configured", name)
	}
	return *model.AgentModelName, nil
}

// FilterAgentSupportedModels filters a list of models to only those that support agent eval
func (c *ModelsConfig) FilterAgentSupportedModels(models []string) []string {
	var supported []string
	for _, model := range models {
		if c.SupportsAgentEval(model) {
			supported = append(supported, model)
		}
	}
	return supported
}

// ListModels returns all configured model names
func (c *ModelsConfig) ListModels() []string {
	models := make([]string, 0, len(c.Models))
	for name := range c.Models {
		models = append(models, name)
	}
	return models
}

// GetBenchmarkSuite returns the recommended models for comprehensive evaluation
func (c *ModelsConfig) GetBenchmarkSuite() []string {
	return c.BenchmarkSuite
}

// GetAgentSuite returns the cross-harness agent eval suite (claude+gemini+codex+opencode).
// Only models with non-null agent_cli participate in agent-mode runs; text-only
// models in the suite are skipped cleanly.
func (c *ModelsConfig) GetAgentSuite() []string {
	return c.AgentSuite
}

// GetDefaultModel returns the default model name
func (c *ModelsConfig) GetDefaultModel() string {
	return c.Default
}

// FindModelsConfig searches for models.yml starting from a directory
func FindModelsConfig(startDir string) (string, error) {
	// Walk up the directory tree looking for benchmarks/models.yml
	dir, err := filepath.Abs(startDir)
	if err != nil {
		return "", err
	}

	for {
		modelsPath := filepath.Join(dir, "benchmarks", "models.yml")
		if _, err := os.Stat(modelsPath); err == nil {
			return modelsPath, nil
		}

		// Move up one directory
		parent := filepath.Dir(dir)
		if parent == dir {
			// Reached root
			break
		}
		dir = parent
	}

	return "", fmt.Errorf("models.yml not found")
}

// ResolveModelName resolves a user-provided model name to its API name.
// Supports both friendly names (e.g., "claude-sonnet-4-5") and direct API names.
// Returns error if model is not found in configuration -- NO SILENT FALLBACKS.
func ResolveModelName(name string) (apiName, provider string, err error) {
	if GlobalModelsConfig == nil {
		// Try to initialize
		if err := InitModelsConfig(); err != nil {
			return "", "", fmt.Errorf("models.yml not loaded and could not be initialized: %w", err)
		}
	}

	// Look up model from config -- fail if not found
	model, err := GlobalModelsConfig.GetModel(name)
	if err != nil {
		return "", "", fmt.Errorf("model %q not found in models.yml: %w\n"+
			"Available models: %v", name, err, GlobalModelsConfig.ListModels())
	}

	return model.APIName, model.Provider, nil
}

// GetExecutorForModel returns the appropriate executor for a model
// Returns the executor name (e.g., "claude", "codex", "managed_agents") and the model name to use
func (c *ModelsConfig) GetExecutorForModel(name string) (executorName string, modelName string, err error) {
	model, err := c.GetModel(name)
	if err != nil {
		return "", "", err
	}

	if model.AgentCLI == nil || *model.AgentCLI == "" {
		return "", "", fmt.Errorf("model %s does not support agent evaluation (no agent_cli configured)", name)
	}

	executorName = *model.AgentCLI

	// M-MANAGED-AGENTS (v0.22.0): Gemini CLI was retired. Reject any model config
	// still requesting agent_cli: "gemini" with a clear next-step message.
	if executorName == "gemini" {
		return "", "", fmt.Errorf(
			"model %q has agent_cli: \"gemini\", but Gemini CLI was retired in AILANG v0.22.0 "+
				"(Google deprecates gemini-cli on 2026-06-18). "+
				"For gemini-3-5-flash agent-mode, use agent_cli: \"managed_agents\" (Vertex AI Managed Agents API via ADC). "+
				"Older Gemini models (2.5, 3, 3.1) lose agent-mode coverage; use standard-mode (direct Vertex generateContent) instead. "+
				"See design_docs/implemented/v0_22_0/m-antigravity-cli-migration.md for context.",
			name,
		)
	}

	if model.AgentModelName != nil && *model.AgentModelName != "" {
		modelName = *model.AgentModelName
	} else {
		modelName = model.APIName
	}

	return executorName, modelName, nil
}
