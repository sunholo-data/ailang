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

	"github.com/sunholo/ailang/internal/ai"
	"github.com/sunholo/ailang/internal/ai/anthropic"
	"github.com/sunholo/ailang/internal/ai/gemini"
	"github.com/sunholo/ailang/internal/ai/ollama"
	"github.com/sunholo/ailang/internal/ai/openai"
	"github.com/sunholo/ailang/internal/effects"
	"github.com/sunholo/ailang/internal/eval_harness"
)

// setupAIHandler configures the AI effect handler based on CLI flags.
// Uses the unified internal/ai package for all providers.
func setupAIHandler(effCtx *effects.EffContext, aiStub bool, aiModel string) error {
	if aiStub {
		effCtx.AI = effects.NewAIContext(effects.NewStubAIHandler())
		return nil
	}

	if aiModel == "" {
		// No AI handler configured - will fail at runtime if AI effect used
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

	// Create handler based on provider using unified ai package
	var handler effects.AIHandler
	switch ai.ProviderFromString(model.Provider) {
	case ai.ProviderAnthropic:
		if apiKey == "" {
			return fmt.Errorf("%s environment variable required for model %s", model.EnvVar, aiModel)
		}
		client := anthropic.NewClient(apiKey)
		handler = client.NewHandler(model.APIName)

	case ai.ProviderOpenAI:
		if apiKey == "" {
			return fmt.Errorf("%s environment variable required for model %s", model.EnvVar, aiModel)
		}
		client := openai.NewClient(apiKey)
		handler = client.NewHandler(model.APIName)

	case ai.ProviderGoogle:
		if apiKey != "" {
			fmt.Fprintf(os.Stderr, "AI: Using Google AI Studio (GOOGLE_API_KEY is set)\n")
			client := gemini.NewClient(apiKey)
			handler = client.NewHandler(model.APIName)
		} else {
			fmt.Fprintf(os.Stderr, "AI: Using Vertex AI (GOOGLE_API_KEY not set, falling back to ADC)\n")
			client, err := gemini.NewVertexAIClient("")
			if err != nil {
				return fmt.Errorf("Gemini auth failed: GOOGLE_API_KEY is not set, and Application Default Credentials (ADC) also failed.\n"+
					"  Option 1: export GOOGLE_API_KEY=<key>  (get one at https://aistudio.google.com/apikey)\n"+
					"  Option 2: gcloud auth application-default login  (for Vertex AI)\n"+
					"  Error: %w", err)
			}
			handler = client.NewHandler(model.APIName)
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
		handler = client.NewHandler(model.APIName)

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
		apiKey := os.Getenv("GOOGLE_API_KEY")
		if apiKey != "" {
			fmt.Fprintf(os.Stderr, "AI: Using Google AI Studio (GOOGLE_API_KEY is set)\n")
			client := gemini.NewClient(apiKey)
			handler = client.NewHandler(modelName)
		} else {
			fmt.Fprintf(os.Stderr, "AI: Using Vertex AI (GOOGLE_API_KEY not set, falling back to ADC)\n")
			client, err := gemini.NewVertexAIClient("")
			if err != nil {
				return fmt.Errorf("Gemini auth failed: GOOGLE_API_KEY is not set, and Application Default Credentials (ADC) also failed.\n"+
					"  Option 1: export GOOGLE_API_KEY=<key>  (get one at https://aistudio.google.com/apikey)\n"+
					"  Option 2: gcloud auth application-default login  (for Vertex AI)\n"+
					"  Error: %w", err)
			}
			handler = client.NewHandler(modelName)
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
