// AI effect handlers for the CLI
//
// Provides real AI handlers for --ai flag using the unified internal/ai package.
// Supports multiple providers: anthropic, openai, google
// Uses models.yml configuration for model lookup.

package main

import (
	"context"
	"fmt"
	"github.com/sunholo-data/ailang/internal/modelreg"
	"os"
	"strings"

	"github.com/sunholo-data/ailang/internal/ai"
	"github.com/sunholo-data/ailang/internal/ai/factory"
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
	model, err := modelreg.GlobalModelsConfig.GetModel(aiModel)
	if err != nil {
		// Model not in config - try direct usage with guessed provider
		return setupAIHandlerDirect(effCtx, aiModel, routingPolicy, attr)
	}

	return setupAIHandlerFromConfig(effCtx, model, aiModel, routingPolicy, attr)
}

// setupAIHandlerFromConfig configures the AI effect handler from a resolved
// models.yml entry. Extracted from setupAIHandler so tests can drive the
// dispatch path (built-in switch + config-driven registry default) without
// going through modelreg.GlobalModelsConfig.
func setupAIHandlerFromConfig(effCtx *effects.EffContext, model *eval_harness.ModelConfig, aiModel string, routingPolicy *ai.AIRoutingPolicy, attr *ai.Attribution) error {
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

	// One factory resolves the credential (from the model's env_var), the
	// endpoint and the lane; built-ins win over a same-named [[ai_provider]]
	// block (M-AI-PROVIDER-CONFIG D4).
	client, err := factory.New(model.Provider,
		factory.WithAPIKeyEnv(model.EnvVar),
		factory.WithConfigDriven(LookupConfigDrivenProvider))
	if err != nil {
		if names := ai.GlobalProviderRegistry.Names(); len(names) > 0 {
			return fmt.Errorf("%w (model %s; config-driven providers: %v)", err, aiModel, names)
		}
		return fmt.Errorf("%w (model %s)", err, aiModel)
	}
	if err := readyForCalls(client); err != nil {
		return err
	}

	handler := ai.NewHandler(client.Provider, model.APIName, opts...)
	effCtx.AI = effects.NewAIContext(handler).WithModelResolver(makeModelResolver(model.Provider))
	return nil
}

// readyForCalls announces the Google lane (so an operator with both ADC and
// a key knows which one is billing) and probes a local ollama daemon before
// the first call, so a stopped daemon reads as "Ollama not running", not as a
// mid-program AI error.
func readyForCalls(client *factory.Client) error {
	switch client.Lane {
	case factory.LaneADC:
		fmt.Fprintf(os.Stderr, "AI: Using Vertex AI (ADC)\n")
	case factory.LaneAPIKey:
		if client.Type == ai.ProviderGoogle {
			fmt.Fprintf(os.Stderr, "AI: Using Google AI Studio (GOOGLE_API_KEY)\n")
		}
	}
	if probe, ok := client.Provider.(interface{ CheckConnection(context.Context) error }); ok {
		return probe.CheckConnection(context.Background())
	}
	return nil
}

// makeModelResolver builds the per-call model resolver injected into the
// AIContext. It lets step()/stepWithCache()/stepWithStream() accept models.yml
// FRIENDLY names (e.g. "gemini-2-5-flash") the same way the --ai flag does,
// instead of forcing hand-written api_names ("gemini-2.5-flash").
//
// boundProvider is the provider of the --ai-selected handler; per-call routing
// stays within it. See effects.ModelResolver for the full contract.
func makeModelResolver(boundProvider string) effects.ModelResolver {
	return func(model string) (string, error) {
		apiName, provider, err := eval_harness.ResolveModelName(model)
		if err != nil {
			// Unknown to models.yml — assume it is already an api_name and let
			// the provider be the final authority (preserves the pre-resolver
			// path where step("gemini-2.5-flash") already worked).
			return model, nil
		}
		if !strings.EqualFold(provider, boundProvider) {
			// Cross-vendor per-call is not supported: the handler is bound to
			// one provider. Return a typed, non-retryable AIError with a hint.
			return "", ai.NewAIError(ai.CodeModelNotAllowed, fmt.Sprintf(
				"per-call model %q resolves to provider %q, but the bound --ai handler is %q; "+
					"per-call routing stays within the bound provider (use --ai %s to switch providers)",
				model, provider, boundProvider, model), false)
		}
		return apiName, nil
	}
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

	if provider == "" {
		// M-AI-PROVIDER-CONFIG: try config-driven provider via "<name>/<model>"
		// prefix. GuessProvider above already handles known OpenRouter vendor
		// prefixes; anything else with a "/" might be a config-driven provider.
		if slash := strings.Index(modelName, "/"); slash > 0 {
			if cd := LookupConfigDrivenProvider(modelName[:slash]); cd != nil {
				effCtx.AI = effects.NewAIContext(ai.NewHandler(cd, modelName[slash+1:], opts...))
				return nil
			}
		}
		return fmt.Errorf("cannot determine provider for model %s (use models.yml or prefix with claude-/gpt-/gemini-/ollama: or vendor/model for OpenRouter, or install a package declaring an [[ai_provider]] block)", modelName)
	}

	client, err := factory.New(string(provider))
	if err != nil {
		return err
	}
	if err := readyForCalls(client); err != nil {
		return err
	}
	// Strip the routing prefix the guess consumed: "ollama:" for the local
	// daemon; "openrouter:" for the gateway, whose model names are
	// "vendor/model" (e.g. "anthropic/claude-sonnet-4.5") or "openrouter/auto".
	model := strings.TrimPrefix(strings.TrimPrefix(modelName, "ollama:"), "openrouter:")
	effCtx.AI = effects.NewAIContext(ai.NewHandler(client.Provider, model, opts...))
	return nil
}
