package eval_harness

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// ModelConfig represents a single model configuration
type ModelConfig struct {
	APIName     string  `yaml:"api_name"`
	Provider    string  `yaml:"provider"`
	Description string  `yaml:"description"`
	EnvVar      string  `yaml:"env_var"`
	Pricing     Pricing `yaml:"pricing"`
	Notes       string  `yaml:"notes"`
}

// Pricing represents model pricing information
type Pricing struct {
	InputPer1K  float64 `yaml:"input_per_1k"`
	OutputPer1K float64 `yaml:"output_per_1k"`
}

// ModelsConfig represents the entire models.yml configuration
type ModelsConfig struct {
	Models         map[string]ModelConfig `yaml:"models"`
	Default        string                 `yaml:"default"`
	BenchmarkSuite []string               `yaml:"benchmark_suite"`
	DevModels      []string               `yaml:"dev_models"`
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
	// Try to find models.yml in internal/eval_harness/ directory
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

// ResolveModelName resolves a user-provided model name to its API name
// Supports both friendly names (e.g., "claude-sonnet-4-5") and direct API names
func ResolveModelName(name string) (apiName, provider string, err error) {
	if GlobalModelsConfig == nil {
		// Try to initialize
		if err := InitModelsConfig(); err != nil {
			// Fallback: return name as-is and guess provider
			return name, guessProvider(name), nil
		}
	}

	// Try to get model from config
	model, err := GlobalModelsConfig.GetModel(name)
	if err != nil {
		// Not in config, use as-is
		return name, guessProvider(name), nil
	}

	return model.APIName, model.Provider, nil
}

// guessProvider attempts to guess the provider from model name
func guessProvider(modelName string) string {
	if len(modelName) >= 3 {
		prefix := modelName[:3]
		switch prefix {
		case "gpt":
			return "openai"
		case "cla":
			return "anthropic"
		case "gem":
			return "google"
		}
	}
	return "unknown"
}
