package ai

import (
	"fmt"
	"sort"
	"strings"
	"sync"
)

// ProviderRegistry holds config-driven AI providers registered at runtime
// from [[ai_provider]] blocks in installed packages' ailang.toml manifests.
// See design_docs/planned/v0_16_0/m-ai-provider-config.md.
//
// Built-in providers (openai, anthropic, gemini, ollama, openrouter) are NOT
// stored here — they remain hardcoded in cmd/ailang/{exec,ai_handlers}.go for
// the features they need (tool use, image input, OpenRouter routing) that the
// v1 [[ai_provider]] schema doesn't yet cover.
//
// Resolution order at dispatch time: built-in providers first, registry
// second. On name collision the built-in wins (with warning) — see D4 in
// the master sequence doc.
type ProviderRegistry struct {
	mu        sync.RWMutex
	providers map[string]*registeredProvider
}

type registeredProvider struct {
	provider Provider
	source   string // path to the ailang.toml that declared this
}

// builtInProviderNames are reserved — config-driven providers cannot shadow
// them. We don't enforce this in Register because the registry is built
// lazily from package manifests; instead, the dispatch layer in cmd/ailang
// consults built-ins first. For diagnostic clarity, registration of a name
// matching a built-in is allowed but flagged via Diagnostics().
var builtInProviderNames = map[string]bool{
	"openai":     true,
	"anthropic":  true,
	"gemini":     true,
	"ollama":     true,
	"openrouter": true,
}

// NewProviderRegistry returns an empty registry. Most callers want
// GlobalProviderRegistry instead — created at process start, populated
// once after package resolution.
func NewProviderRegistry() *ProviderRegistry {
	return &ProviderRegistry{providers: map[string]*registeredProvider{}}
}

// GlobalProviderRegistry is populated at process startup from installed
// packages' ailang.toml manifests. Tests should call Reset() in setup or
// construct a fresh ProviderRegistry where possible.
var GlobalProviderRegistry = NewProviderRegistry()

// Register adds a provider to the registry. Returns a structured error if
// the same name is already registered (cross-package conflict per D11 in
// the master sequence doc). The error message names both source manifests
// so the user can resolve by removing one or aliasing.
func (r *ProviderRegistry) Register(name string, p Provider, source string) error {
	if name == "" {
		return fmt.Errorf("provider name is empty")
	}
	if p == nil {
		return fmt.Errorf("provider %q: nil provider passed to Register", name)
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	if existing, ok := r.providers[name]; ok {
		// Source-based idempotency: re-registering the same name from the
		// same source manifest is a no-op (the provider may be a fresh
		// struct each time — e.g. the harvest function reconstructs them
		// on every CLI invocation). Different sources with the same name
		// is a true cross-package conflict.
		if existing.source == source {
			return nil
		}
		return fmt.Errorf("AI provider name %q is already registered\n  first declared:  %s\n  also declared:   %s\n\nResolve by removing one of the [[ai_provider]] blocks, or rename one of the providers (the routing prefix used in call(\"<name>/<model>\", ...) must be unique across all installed packages).",
			name, existing.source, source)
	}
	r.providers[name] = &registeredProvider{provider: p, source: source}
	return nil
}

// Lookup returns the registered provider for the given name, or (nil, false)
// if not registered. Built-in providers are NOT stored here; dispatch must
// consult built-ins first.
func (r *ProviderRegistry) Lookup(name string) (Provider, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	rp, ok := r.providers[name]
	if !ok {
		return nil, false
	}
	return rp.provider, true
}

// Names returns the registered provider names in deterministic (alphabetical)
// order. Used for diagnostics, dispatch fallback ordering, and CLI listing.
func (r *ProviderRegistry) Names() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]string, 0, len(r.providers))
	for k := range r.providers {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

// SourceOf returns the manifest path that declared the given provider, or
// "" if not registered. Used for error messages.
func (r *ProviderRegistry) SourceOf(name string) string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	if rp, ok := r.providers[name]; ok {
		return rp.source
	}
	return ""
}

// Reset clears the registry. Tests should call this in setup to avoid
// pollution between tests that exercise the global instance.
func (r *ProviderRegistry) Reset() {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.providers = map[string]*registeredProvider{}
}

// Diagnostics returns informational warnings about the current registry
// state — currently: registered names that shadow built-in provider names.
// Returns nil if no diagnostics. Caller decides whether to log/print/error.
func (r *ProviderRegistry) Diagnostics() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	var warnings []string
	for name, rp := range r.providers {
		if builtInProviderNames[name] {
			warnings = append(warnings, fmt.Sprintf(
				"WARNING: AI provider %q (declared in %s) shadows a built-in provider name; the built-in will win at dispatch time. Rename the [[ai_provider]] block to avoid confusion.",
				name, rp.source))
		}
	}
	sort.Strings(warnings)
	return warnings
}

// IsBuiltinName reports whether the given name is one of the hardcoded
// built-in provider names. Useful for cmd/ailang dispatch logic that
// short-circuits to built-ins before consulting the registry.
func IsBuiltinName(name string) bool {
	return builtInProviderNames[strings.ToLower(name)]
}
