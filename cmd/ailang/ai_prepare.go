package main

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/sunholo-data/ailang/internal/ai"
	"github.com/sunholo-data/ailang/internal/ai/configdriven"
	"github.com/sunholo-data/ailang/internal/ai/factory"
	"github.com/sunholo-data/ailang/internal/ai/openrouter"
	"github.com/sunholo-data/ailang/internal/config"
	"github.com/sunholo-data/ailang/internal/effects"
	"github.com/sunholo-data/ailang/internal/eval_harness"
	"github.com/sunholo-data/ailang/internal/modelreg"
)

// prepareRunAIHandler resolves/fixes structural configuration before main, but
// defers credential file reads, client creation and readiness until an AI effect.
// Explicit setupAIHandler callers (including serve-api) retain eager setup.
func prepareRunAIHandler(effCtx *effects.EffContext, cfg aiSetup, routing *ai.AIRoutingPolicy, attr *ai.Attribution) error {
	if cfg.Stub || cfg.Model == "" || cfg.StubFixtures != "" {
		if err := setupAIHandler(effCtx, cfg, routing, attr); err != nil {
			return err
		}
		effCtx.WithProviderLookup(snapshotNamedProviders())
		return nil
	}
	cwd, _ := os.Getwd()
	if err := HarvestAndRegisterFromDir(cwd); err != nil {
		return fmt.Errorf("AI provider registration failed: %w", err)
	}
	for _, diag := range ai.GlobalProviderRegistry.Diagnostics() {
		fmt.Fprintln(os.Stderr, diag)
	}

	binding, err := prepareRunAIBinding(cfg.Model)
	if err != nil {
		return err
	}

	path := cfg.KeyFile
	if path == "" {
		path = config.AIKeyFile()
	}
	if path != "" {
		var err error
		path, err = filepath.Abs(path)
		if err != nil {
			return fmt.Errorf("--ai-key-file path: %w", err)
		}
	}
	frozen := factory.FreezeConfiguration(binding.keyEnv)
	opts := []ai.HandlerOption{ai.WithAttribution(openrouter.FrozenAttribution(attr))}
	if routing != nil {
		copy := *routing
		copy.Order = append([]string(nil), routing.Order...)
		copy.Require = append([]ai.AICapability(nil), routing.Require...)
		opts = append(opts, ai.WithRoutingPolicy(&copy))
	}
	if binding.maxTokens > 0 {
		opts = append(opts, ai.WithMaxTokens(binding.maxTokens))
	}
	prepared := preparedRunAI{binding: binding, path: path, noADC: cfg.NoADC, frozen: frozen, opts: opts}
	effCtx.AI = effects.NewLazyAIContext(prepared.initialize)
	effCtx.WithProviderLookup(snapshotNamedProviders())
	return nil
}

type preparedRunAI struct {
	binding runAIBinding
	path    string
	noADC   bool
	frozen  factory.Option
	opts    []ai.HandlerOption
}

func (p preparedRunAI) initialize() (*effects.AIContext, error) {
	fopts := []factory.Option{p.frozen, factory.WithNoADC(p.noADC), factory.WithAPIKeyEnv(p.binding.keyEnv), factory.WithConfigDriven(func(name string) ai.Provider {
		if name == p.binding.provider {
			return p.binding.configDriven
		}
		return nil
	})}
	if p.path != "" {
		key, err := factory.ReadKeyFile(p.path)
		if err != nil {
			return nil, lazyAuthError(err)
		}
		fopts = append(fopts, factory.WithAPIKey(key))
	}
	client, err := factory.New(p.binding.provider, fopts...)
	if err != nil {
		return nil, lazyAuthError(err)
	}
	if err := readyForCalls(client, p.path != ""); err != nil {
		return nil, err
	}
	return effects.NewAIContext(ai.NewHandler(client.Provider, p.binding.model, p.opts...)).WithModelResolver(p.binding.resolver), nil
}

// Initialization auth failures now enter the operation's established error
// channel: plain error for call, Err(AIError) for Result APIs. Preserve a
// provider's pre-typed error rather than replacing capability/model codes.
func lazyAuthError(err error) error {
	var typed *ai.AIError
	if errors.As(err, &typed) {
		return err
	}
	return ai.NewAIError(ai.CodeAuthFailed, err.Error(), false)
}

type runAIBinding struct {
	provider, model, keyEnv string
	maxTokens               int
	resolver                effects.ModelResolver
	configDriven            ai.Provider
}

func prepareRunAIBinding(modelName string) (runAIBinding, error) {
	// Both the chosen row and the per-call name resolver are immutable snapshots.
	var models *modelreg.ModelsConfig
	loadErr := eval_harness.InitModelsConfig()
	if loadErr != nil && config.ModelsPath() != "" {
		return runAIBinding{}, loadErr
	}
	if loadErr == nil && modelreg.GlobalModelsConfig != nil {
		// `ailang run` takes this deferred path, never setupAIHandler, so the
		// per-model OpenRouter routing options must be installed here too.
		// Measured 2026-10-13: without this, motoko's runtime sent no
		// provider.ignore and 118 GLM calls still landed on inference-net.
		openrouter.SetProviderDefaults(modelreg.GlobalModelsConfig.OpenRouterProviderByAPIName())
		models = &modelreg.ModelsConfig{Models: make(map[string]modelreg.ModelConfig)}
		for name, row := range modelreg.GlobalModelsConfig.Models {
			row.Aliases = append([]string(nil), row.Aliases...)
			models.Models[name] = row
		}
	}
	provider, model, maxTokens, keyEnv := "", modelName, 0, ""
	var resolver effects.ModelResolver
	if models != nil {
		if row, err := models.GetModel(modelName); err == nil {
			if strings.TrimSpace(row.Provider) == "" || strings.TrimSpace(row.APIName) == "" {
				return runAIBinding{}, fmt.Errorf("model %q requires provider and api_name", modelName)
			}
			provider, model, maxTokens, keyEnv = row.Provider, row.APIName, row.MaxOutputTokens, row.EnvVar
			resolver = frozenModelResolver(models, provider)
		} else if _, row, err := models.Resolve(modelName); err == nil {
			maxTokens = row.MaxOutputTokens
		}
	}
	if provider == "" {
		provider = string(ai.GuessProvider(modelName))
		model = strings.TrimPrefix(strings.TrimPrefix(modelName, "ollama:"), "openrouter:")
		if provider == "" {
			if prefix, suffix, ok := strings.Cut(modelName, "/"); ok && LookupConfigDrivenProvider(prefix) != nil {
				provider, model = prefix, suffix
			} else {
				return runAIBinding{}, fmt.Errorf("cannot determine provider for model %s (use models.yml or a provider/model prefix)", modelName)
			}
		}
	}
	cd := snapshotConfigDriven(LookupConfigDrivenProvider(provider))
	typ := ai.ProviderFromString(provider)
	if typ == ai.ProviderCodex {
		return runAIBinding{}, ai.NewAIError(ai.CodeCapabilityNotSupported, "codex AI-effect routing is unavailable; select chatgpt/<model> or provider openai", false)
	}
	if !ai.IsBuiltinName(string(typ)) && typ != ai.ProviderGoogle && typ != ai.ProviderChatGPT && cd == nil {
		return runAIBinding{}, fmt.Errorf("unsupported AI provider: %q", provider)
	}

	return runAIBinding{provider: provider, model: model, maxTokens: maxTokens, keyEnv: keyEnv, resolver: resolver, configDriven: cd}, nil
}

func frozenModelResolver(models *modelreg.ModelsConfig, bound string) effects.ModelResolver {
	return func(name string) (string, error) {
		row, err := models.GetModel(name)
		if err != nil {
			return name, nil
		}
		if !strings.EqualFold(row.Provider, bound) {
			return "", ai.NewAIError(ai.CodeModelNotAllowed, fmt.Sprintf("per-call model %q resolves to provider %q, but the bound --ai handler is %q; per-call routing stays within the bound provider (use --ai %s to switch providers)", name, row.Provider, bound, name), false)
		}
		return row.APIName, nil
	}
}

func snapshotConfigDriven(provider ai.Provider) ai.Provider {
	if p, ok := provider.(*configdriven.Provider); ok {
		spec := *p.Spec()
		spec.Models.Allowed = append([]string(nil), spec.Models.Allowed...)
		if spec.AuthHeaders != nil {
			spec.AuthHeaders = make(map[string]string, len(p.Spec().AuthHeaders))
			for name, value := range p.Spec().AuthHeaders {
				spec.AuthHeaders[name] = value
			}
		}
		return configdriven.New(&spec, configdriven.WithFrozenAuthEnvironment())
	}
	return provider
}

// Named streaming can run without a selected --ai handler. Capture its full
// registry independently so every CLI branch obeys the same freeze boundary.
func snapshotNamedProviders() func(string) (ai.Provider, bool) {
	providers := make(map[string]ai.Provider)
	for _, name := range ai.GlobalProviderRegistry.Names() {
		if provider, ok := ai.GlobalProviderRegistry.Lookup(name); ok {
			providers[name] = snapshotConfigDriven(provider)
		}
	}
	return func(name string) (ai.Provider, bool) {
		provider, ok := providers[name]
		return provider, ok
	}
}
