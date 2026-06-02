package eval_harness

import (
	_ "embed"
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

//go:embed models.yml
var embeddedModelsYAML []byte

// ModelConfig represents a single model configuration
type ModelConfig struct {
	APIName                  string  `yaml:"api_name"`
	Provider                 string  `yaml:"provider"`
	Description              string  `yaml:"description"`
	EnvVar                   string  `yaml:"env_var"`
	AgentCLI                 *string `yaml:"agent_cli"`          // CLI command for agent eval (e.g., "claude", "openai", "gemini"), nil if not supported
	AgentModelName           *string `yaml:"agent_model_name"`   // Model name to pass to agent CLI (e.g., "haiku", "sonnet")
	MaxOutputTokens          int     `yaml:"max_output_tokens"`  // Max output tokens (0 = handler default 4096)
	TTFTTimeoutSeconds       int     `yaml:"ttft_timeout"`       // Prefill budget in seconds (0 = executor default 30s)
	GenerationTimeoutSeconds int     `yaml:"generation_timeout"` // Per-token idle budget after first event (0 = executor default 3m)
	ModelFamily              string  `yaml:"model_family"`       // Logical model family for cross-harness grouping (e.g., "claude-sonnet-4-6"); empty = no grouping
	GCPProject               string  `yaml:"gcp_project"`        // Override GOOGLE_CLOUD_PROJECT for this model's evals (e.g. "ailang-dev")
	GCPLocation              string  `yaml:"gcp_location"`       // Override GOOGLE_CLOUD_LOCATION (e.g. "us-central1")
	MotokoProfile            string  `yaml:"motoko_profile"`     // Override MOTOKO_CONFIG profile (default: "dogfood"); used when agent_cli is "motoko"
	Pricing                  Pricing `yaml:"pricing"`
	Budgets                  Budgets `yaml:"budgets"` // M-EVAL-COST-AND-SPEED-BUDGETS (v0.16.0): cost-aware budget overrides
	Notes                    string  `yaml:"notes"`
}

// Pricing represents model pricing information
type Pricing struct {
	InputPer1K  float64 `yaml:"input_per_1k"`
	OutputPer1K float64 `yaml:"output_per_1k"`
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
type Budgets struct {
	MaxCostUSD              float64 `yaml:"max_cost_usd"`
	HardTimeoutSecs         int     `yaml:"hard_timeout_secs"`
	ExpectedTTFTSecs        int     `yaml:"expected_ttft_secs"`
	ExpectedTTFSolutionSecs int     `yaml:"expected_ttf_solution_secs"`
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

// SupportsAgentEval returns true if the model supports agent-based evaluation
func (c *ModelsConfig) SupportsAgentEval(name string) bool {
	model, err := c.GetModel(name)
	if err != nil {
		return false
	}
	return model.AgentCLI != nil && *model.AgentCLI != ""
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
