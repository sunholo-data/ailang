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
	"github.com/sunholo-data/ailang/internal/ai/openrouter"
	"github.com/sunholo-data/ailang/internal/effects"
	"github.com/sunholo-data/ailang/internal/eval_harness"
)

// setupAIHandler configures the AI effect handler based on CLI flags.
// Uses the unified internal/ai package for all providers.
//
// routingPolicy is optional. When non-nil it is attached to the handler so
// every outgoing AI call carries the policy. Only the OpenRouter provider
// consumes it; other providers will surface ai.ErrRoutingNotSupported on
// the first call. If aiModel is empty (no AI configured) and the caller
// passed a non-nil routingPolicy, we treat that as a configuration mistake
// and warn — there is no handler to attach the policy to.
func setupAIHandler(effCtx *effects.EffContext, aiStub bool, aiModel string, routingPolicy *ai.AIRoutingPolicy, attr *ai.Attribution) error {
	// M-AI-PROVIDER-CONFIG: harvest [[ai_provider]] blocks from the project's
	// ailang.toml + dependency manifests before consulting the registry in
	// setupAIHandlerFromConfig / setupAIHandlerDirect. Idempotent — safe to
	// call repeatedly across multiple AI handler setups in the same process.
	cwd, _ := os.Getwd()
	if err := HarvestAndRegisterFromDir(cwd); err != nil {
		// Cross-package duplicate-name conflicts are surfaced here as fatal —
		// the user must resolve before the program can be reasoned about.
		return fmt.Errorf("AI provider registration failed: %w", err)
	}
	if diags := ai.GlobalProviderRegistry.Diagnostics(); len(diags) > 0 {
		for _, msg := range diags {
			fmt.Fprintln(os.Stderr, msg)
		}
	}

	if aiStub {
		// Stub handler ignores routing policy — that's fine, this is for
		// flag-shape testing without any real provider call.
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
		if routingPolicy.HasRouting() {
			fmt.Fprintln(os.Stderr, "Warning: --routing-* flags set but --ai not configured; routing policy ignored.")
		}
		return nil
	}

	// Load models config to look up model details
	if err := eval_harness.InitModelsConfig(); err != nil {
		// Config not found - try to use model name directly with guessed provider
		return setupAIHandlerDirect(effCtx, aiModel, routingPolicy, attr)
	}

	// Look up model in config
	model, err := eval_harness.GlobalModelsConfig.GetModel(aiModel)
	if err != nil {
		// Model not in config - try direct usage with guessed provider
		return setupAIHandlerDirect(effCtx, aiModel, routingPolicy, attr)
	}

	return setupAIHandlerFromConfig(effCtx, model, aiModel, routingPolicy, attr)
}

// setupAIHandlerFromConfig configures the AI effect handler from a resolved
// models.yml entry. Extracted from setupAIHandler so tests can drive the
// dispatch path (built-in switch + config-driven registry default) without
// going through eval_harness.GlobalModelsConfig.
func setupAIHandlerFromConfig(effCtx *effects.EffContext, model *eval_harness.ModelConfig, aiModel string, routingPolicy *ai.AIRoutingPolicy, attr *ai.Attribution) error {
	// Get API key from environment (may be empty for Google ADC)
	apiKey := os.Getenv(model.EnvVar)

	// Build handler options from model config
	var opts []ai.HandlerOption
	if model.MaxOutputTokens > 0 {
		opts = append(opts, ai.WithMaxTokens(model.MaxOutputTokens))
	}
	if routingPolicy != nil {
		opts = append(opts, ai.WithRoutingPolicy(routingPolicy))
	}
	if attr != nil {
		opts = append(opts, ai.WithAttribution(attr))
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
		customBaseURL := strings.TrimSpace(os.Getenv("OPENAI_BASE_URL"))
		if apiKey == "" && customBaseURL == "" {
			return fmt.Errorf("%s environment variable required for model %s (or set OPENAI_BASE_URL for a custom unauthenticated endpoint)", model.EnvVar, aiModel)
		}
		var clientOpts []openai.ClientOption
		if customBaseURL != "" {
			clientOpts = append(clientOpts, openai.WithBaseURL(customBaseURL))
		}
		client := openai.NewClient(apiKey, clientOpts...)
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
		// M-AI-PROVIDER-CONFIG: consult the config-driven provider registry.
		// Built-in dispatch above wins on collision (D4).
		if cd := LookupConfigDrivenProvider(model.Provider); cd != nil {
			handler = ai.NewHandler(cd, model.APIName, opts...)
		} else {
			names := ai.GlobalProviderRegistry.Names()
			if len(names) > 0 {
				return fmt.Errorf("unsupported AI provider: %q (built-in: openai, anthropic, gemini, ollama, openrouter; config-driven: %v)", model.Provider, names)
			}
			return fmt.Errorf("unsupported AI provider: %s", model.Provider)
		}
	}

	effCtx.AI = effects.NewAIContext(handler)
	return nil
}

// setupAIHandlerDirect creates an AI handler using the model name directly
// (fallback when models.yml is not available).
//
// routingPolicy is optional and threaded onto the handler via WithRoutingPolicy
// when non-nil. Only OpenRouter consumes it; passing a routing policy with a
// non-OpenRouter provider yields ai.ErrRoutingNotSupported on the first call.
func setupAIHandlerDirect(effCtx *effects.EffContext, modelName string, routingPolicy *ai.AIRoutingPolicy, attr *ai.Attribution) error {
	// Guess provider from model name
	provider := ai.GuessProvider(modelName)

	// Build handler options once; reused across providers.
	var opts []ai.HandlerOption
	if routingPolicy != nil {
		opts = append(opts, ai.WithRoutingPolicy(routingPolicy))
	}
	if attr != nil {
		opts = append(opts, ai.WithAttribution(attr))
	}

	var handler effects.AIHandler

	switch provider {
	case ai.ProviderAnthropic:
		apiKey := os.Getenv("ANTHROPIC_API_KEY")
		if apiKey == "" {
			return fmt.Errorf("ANTHROPIC_API_KEY environment variable required")
		}
		client := anthropic.NewClient(apiKey)
		handler = client.NewHandler(modelName, opts...)

	case ai.ProviderOpenAI:
		apiKey := os.Getenv("OPENAI_API_KEY")
		if apiKey == "" {
			return fmt.Errorf("OPENAI_API_KEY environment variable required")
		}
		client := openai.NewClient(apiKey)
		handler = client.NewHandler(modelName, opts...)

	case ai.ProviderGoogle:
		// Precedence: ADC first (if available), then GOOGLE_API_KEY.
		apiKey := os.Getenv("GOOGLE_API_KEY")
		if client, err := gemini.NewVertexAIClient(""); err == nil {
			fmt.Fprintf(os.Stderr, "AI: Using Vertex AI (ADC)\n")
			handler = client.NewHandler(modelName, opts...)
		} else if apiKey != "" {
			fmt.Fprintf(os.Stderr, "AI: Using Google AI Studio (GOOGLE_API_KEY)\n")
			client := gemini.NewClient(apiKey)
			handler = client.NewHandler(modelName, opts...)
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
		handler = client.NewHandler(model, opts...)

	case ai.ProviderOpenRouter:
		apiKey := os.Getenv("OPENROUTER_API_KEY")
		if apiKey == "" {
			return fmt.Errorf("OPENROUTER_API_KEY environment variable required")
		}
		client := openrouter.NewClient(apiKey)
		// Strip optional explicit "openrouter:" prefix; the model name itself
		// is "vendor/model" (e.g., "anthropic/claude-sonnet-4.5") or
		// "openrouter/auto" for the auto-router.
		model := strings.TrimPrefix(modelName, "openrouter:")
		handler = client.NewHandler(model, opts...)

	default:
		// M-AI-PROVIDER-CONFIG: try config-driven provider via "<name>/<model>"
		// prefix. GuessProvider above already handles known OpenRouter vendor
		// prefixes; anything else with a "/" might be a config-driven provider.
		if slash := strings.Index(modelName, "/"); slash > 0 {
			providerName := modelName[:slash]
			modelPart := modelName[slash+1:]
			if cd := LookupConfigDrivenProvider(providerName); cd != nil {
				handler = ai.NewHandler(cd, modelPart, opts...)
				break
			}
		}
		return fmt.Errorf("cannot determine provider for model %s (use models.yml or prefix with claude-/gpt-/gemini-/ollama: or vendor/model for OpenRouter, or install a package declaring an [[ai_provider]] block)", modelName)
	}

	effCtx.AI = effects.NewAIContext(handler)
	return nil
}
