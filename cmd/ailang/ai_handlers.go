// AI effect handlers for the CLI
//
// Provides real AI handlers for --ai flag using the unified internal/ai package.
// Supports multiple providers: anthropic, openai, google
// Uses models.yml configuration for model lookup.

package main

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/sunholo-data/ailang/internal/ai"
	"github.com/sunholo-data/ailang/internal/ai/anthropic"
	"github.com/sunholo-data/ailang/internal/ai/gemini"
	"github.com/sunholo-data/ailang/internal/ai/ollama"
	"github.com/sunholo-data/ailang/internal/ai/openai"
	"github.com/sunholo-data/ailang/internal/effects"
	"github.com/sunholo-data/ailang/internal/eval_harness"
)

// setupAIHandler configures the AI effect handler based on CLI flags.
// Uses the unified internal/ai package for all providers.
func setupAIHandler(effCtx *effects.EffContext, aiStub bool, aiModel string) error {
	if aiStub {
		effCtx.AI = effects.NewAIContext(effects.NewStubAIHandler())
		return nil
	}

	if aiModel == "" {
		// No AI handler configured - warn early if AI capability was granted
		if effCtx.HasCap("AI") {
			fmt.Fprintf(os.Stderr, "Warning: --caps AI requires --ai <model> flag.\n"+
				"  No AI model configured. Programs using AI.call will fail.\n"+
				"  Fix: ailang run --caps AI --ai gemini-2-5-flash ...\n"+
				"  Or for testing: ailang run --caps AI --ai-stub ...\n")
		}
		return nil
	}

	// Load models config to look up model details
	if err := eval_harness.InitModelsConfig(); err != nil {
		// Config not found - try to use model name directly with guessed provider
		return setupAIHandlerDirect(effCtx, aiModel)
	}

	// Look up model in config
	model, err := eval_harness.GlobalModelsConfig.GetModel(aiModel)
	if err != nil {
		// Model not in config - try direct usage with guessed provider
		return setupAIHandlerDirect(effCtx, aiModel)
	}

	// Get API key from environment (may be empty for Google ADC)
	apiKey := os.Getenv(model.EnvVar)

	// Build handler options from model config
	var opts []ai.HandlerOption
	if model.MaxOutputTokens > 0 {
		opts = append(opts, ai.WithMaxTokens(model.MaxOutputTokens))
	}

	// Create handler based on provider using unified ai package
	var handler effects.AIHandler
	switch ai.ProviderFromString(model.Provider) {
	case ai.ProviderAnthropic:
		if apiKey == "" {
			return fmt.Errorf("%s environment variable required for model %s", model.EnvVar, aiModel)
		}
		client := anthropic.NewClient(apiKey)
		handler = client.NewHandler(model.APIName, opts...)

	case ai.ProviderOpenAI:
		if apiKey == "" {
			return fmt.Errorf("%s environment variable required for model %s", model.EnvVar, aiModel)
		}
		client := openai.NewClient(apiKey)
		handler = client.NewHandler(model.APIName, opts...)

	case ai.ProviderGoogle:
		// Precedence: ADC first (if available), then GOOGLE_API_KEY.
		// Many users have GOOGLE_API_KEY set for other tools but prefer ADC
		// for Vertex AI access. Try ADC silently first; fall back to API key.
		if client, err := gemini.NewVertexAIClient(""); err == nil {
			fmt.Fprintf(os.Stderr, "AI: Using Vertex AI (ADC)\n")
			handler = client.NewHandler(model.APIName, opts...)
		} else if apiKey != "" {
			fmt.Fprintf(os.Stderr, "AI: Using Google AI Studio (GOOGLE_API_KEY)\n")
			client := gemini.NewClient(apiKey)
			handler = client.NewHandler(model.APIName, opts...)
		} else {
			return fmt.Errorf("Gemini auth failed: Application Default Credentials (ADC) not configured, and GOOGLE_API_KEY is not set.\n"+
				"  Option 1: gcloud auth application-default login  (recommended, for Vertex AI)\n"+
				"  Option 2: export GOOGLE_API_KEY=<key>  (get one at https://aistudio.google.com/apikey)\n"+
				"  ADC error: %w", err)
		}

	case ai.ProviderOllama:
		// Ollama is local, no API key needed
		client, err := ollama.NewClient()
		if err != nil {
			return fmt.Errorf("failed to create Ollama client: %w", err)
		}
		// Check connection before proceeding
		if err := client.CheckConnection(context.Background()); err != nil {
			return err
		}
		handler = client.NewHandler(model.APIName, opts...)

	default:
		return fmt.Errorf("unsupported AI provider: %s", model.Provider)
	}

	effCtx.AI = effects.NewAIContext(handler)
	return nil
}

// setupAIHandlerDirect creates an AI handler using the model name directly
// (fallback when models.yml is not available).
func setupAIHandlerDirect(effCtx *effects.EffContext, modelName string) error {
	// Guess provider from model name
	provider := ai.GuessProvider(modelName)

	var handler effects.AIHandler

	switch provider {
	case ai.ProviderAnthropic:
		apiKey := os.Getenv("ANTHROPIC_API_KEY")
		if apiKey == "" {
			return fmt.Errorf("ANTHROPIC_API_KEY environment variable required")
		}
		client := anthropic.NewClient(apiKey)
		handler = client.NewHandler(modelName)

	case ai.ProviderOpenAI:
		apiKey := os.Getenv("OPENAI_API_KEY")
		if apiKey == "" {
			return fmt.Errorf("OPENAI_API_KEY environment variable required")
		}
		client := openai.NewClient(apiKey)
		handler = client.NewHandler(modelName)

	case ai.ProviderGoogle:
		// Precedence: ADC first (if available), then GOOGLE_API_KEY.
		apiKey := os.Getenv("GOOGLE_API_KEY")
		if client, err := gemini.NewVertexAIClient(""); err == nil {
			fmt.Fprintf(os.Stderr, "AI: Using Vertex AI (ADC)\n")
			handler = client.NewHandler(modelName)
		} else if apiKey != "" {
			fmt.Fprintf(os.Stderr, "AI: Using Google AI Studio (GOOGLE_API_KEY)\n")
			client := gemini.NewClient(apiKey)
			handler = client.NewHandler(modelName)
		} else {
			return fmt.Errorf("Gemini auth failed: Application Default Credentials (ADC) not configured, and GOOGLE_API_KEY is not set.\n"+
				"  Option 1: gcloud auth application-default login  (recommended, for Vertex AI)\n"+
				"  Option 2: export GOOGLE_API_KEY=<key>  (get one at https://aistudio.google.com/apikey)\n"+
				"  ADC error: %w", err)
		}

	case ai.ProviderOllama:
		// Ollama is local, no API key needed
		client, err := ollama.NewClient()
		if err != nil {
			return fmt.Errorf("failed to create Ollama client: %w", err)
		}
		// Check connection before proceeding
		if err := client.CheckConnection(context.Background()); err != nil {
			return err
		}
		// Strip ollama: prefix if present
		model := strings.TrimPrefix(modelName, "ollama:")
		handler = client.NewHandler(model)

	default:
		return fmt.Errorf("cannot determine provider for model %s (use models.yml or prefix with claude-/gpt-/gemini-/ollama:)", modelName)
	}

	effCtx.AI = effects.NewAIContext(handler)
	return nil
}
