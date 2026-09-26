package modelreg

import (
	"errors"
	"fmt"
	"sort"
	"strings"
)

// M-V1-SIMPLIFY-S3 M2: one alias table, one resolver.
//
// Until 2026-09-15 the observatory carried a private map of ~20 wire names to
// registry keys (`normalizeModelName`) plus its own date-suffix stripper, and
// its generic OTLP path did not even try api_name — so every OpenRouter span
// whose gen_ai.request.model was a slug like "z-ai/glm-5.3-flash" priced at
// $0.00 in prod, silently. The registry already knew the slug; the lookup was
// in the wrong package. Aliases now live on the row they belong to, and there
// is exactly one function that turns a wire name into a registry key.

// ErrUnknownModel is wrapped by Resolve when no row matches. Callers that price
// tokens MUST branch on it rather than reading $0: an unresolvable model is
// "unpriced", which is a different fact from "free".
var ErrUnknownModel = errors.New("model not in registry")

// ErrAmbiguousModel is wrapped by Resolve when a name matches several rows
// that do NOT agree on price. Picking one would price the call at a rate
// nobody chose; the caller treats it as unpriced, and the fix is in the
// registry (declare an alias on the intended row).
var ErrAmbiguousModel = errors.New("model name matches rows with different pricing")

// Resolve maps a name a caller actually has — a friendly key, an api_name, a
// declared alias, a harness wire name, or a dated/dotted variant of any of
// those — to the registry key and row it belongs to.
//
// Precedence: exact key → api_name → alias → the same three over a normalised
// form (trailing -YYYYMMDD stripped, "." replaced by "-") → agent_model_name.
// Within one level keys are tried in sorted order so two rows sharing an
// api_name (harness twins such as gpt5-5 / opencode-gpt5-5) resolve
// deterministically; the pricing gate in eval_harness
// (TestModels_PricingIsSlugConsistent) keeps such twins priced identically.
//
// agent_model_name is the LAST tier because it is what a harness passes to a
// CLI, not a vendor id — the rig banks it as the stage model ("openrouter/
// deepseek/deepseek-v4-flash-0731", "ollama/deepseek-v4-flash:0731-cloud" in
// the local observatory), so refusing it leaves real rows unpriced, but CLI
// short names ("sonnet", "opus") move between vendor models with CLI releases.
// The guard: every row sharing that wire name must agree on price, or the
// name is ErrAmbiguousModel rather than a silent pick.
func (c *ModelsConfig) Resolve(name string) (string, *ModelConfig, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return "", nil, fmt.Errorf("%w: empty model name", ErrUnknownModel)
	}
	if m, ok := c.Models[name]; ok {
		return name, &m, nil
	}
	if key, ok := c.matchAPINameOrAlias(name); ok {
		m := c.Models[key]
		return key, &m, nil
	}
	if n := normalizeModelName(name); n != name {
		if m, ok := c.Models[n]; ok {
			return n, &m, nil
		}
		if key, ok := c.matchAPINameOrAlias(n); ok {
			m := c.Models[key]
			return key, &m, nil
		}
	}
	if keys := c.matchAgentModelName(name); len(keys) > 0 {
		first := c.Models[keys[0]]
		for _, k := range keys[1:] {
			if !sameRates(c.Models[k].Pricing, first.Pricing) {
				return "", nil, fmt.Errorf("%w: %q is the wire name of %v", ErrAmbiguousModel, name, keys)
			}
		}
		return keys[0], &first, nil
	}
	return "", nil, fmt.Errorf("%w: %q", ErrUnknownModel, name)
}

// sameRates compares the billed rates only — not the schedule pointer, which
// is per-row state and would make two identically-priced rows read as different.
func sameRates(a, b Pricing) bool {
	return a.InputPer1K == b.InputPer1K && a.OutputPer1K == b.OutputPer1K &&
		a.CacheReadPer1K == b.CacheReadPer1K && a.CacheWritePer1K == b.CacheWritePer1K
}

// matchAgentModelName returns, in sorted order, every key whose
// agent_model_name is exactly name.
func (c *ModelsConfig) matchAgentModelName(name string) []string {
	var keys []string
	for _, k := range c.sortedKeys() {
		if a := c.Models[k].AgentModelName; a != nil && *a == name {
			keys = append(keys, k)
		}
	}
	return keys
}

// sortedKeys is the deterministic iteration order every lookup here shares.
func (c *ModelsConfig) sortedKeys() []string {
	keys := make([]string, 0, len(c.Models))
	for k := range c.Models {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

func (c *ModelsConfig) matchAPINameOrAlias(name string) (string, bool) {
	keys := c.sortedKeys()
	for _, k := range keys {
		if c.Models[k].APIName == name {
			return k, true
		}
	}
	for _, k := range keys {
		for _, a := range c.Models[k].Aliases {
			if a == name {
				return k, true
			}
		}
	}
	return "", false
}

// normalizeModelName strips a trailing -YYYYMMDD snapshot date and turns dots
// into dashes ("claude-sonnet-4-5-20250929" → "claude-sonnet-4-5",
// "gemini-2.5-pro" → "gemini-2-5-pro"). Pure string grammar; it never consults
// the registry, so Resolve applies it only after the literal name has failed.
func normalizeModelName(model string) string {
	if len(model) > 9 {
		suffix := model[len(model)-9:]
		if suffix[0] == '-' && isAllDigits(suffix[1:]) {
			model = model[:len(model)-9]
		}
	}
	return strings.ReplaceAll(model, ".", "-")
}

func isAllDigits(s string) bool {
	for _, c := range s {
		if c < '0' || c > '9' {
			return false
		}
	}
	return len(s) > 0
}

// CostForName prices a run for a model named by anything Resolve accepts.
//
// It is CalculateCostForModelWithCache behind Resolve: the one entry point the
// observatory, the executors' fallback cost model and the quorum estimate all
// share, so a wire name that the registry knows is priced the same everywhere.
// Returns ErrUnknownModel (wrapped) when nothing matches — never $0.
func (c *ModelsConfig) CostForName(name string, inputTokens, outputTokens, cacheReadTokens, cacheWriteTokens int) (float64, error) {
	key, _, err := c.Resolve(name)
	if err != nil {
		return 0, err
	}
	return c.CalculateCostForModelWithCache(key, inputTokens, outputTokens, cacheReadTokens, cacheWriteTokens)
}

// validateAliases reports aliases that collide with a key, an api_name of a
// different row, or another row's alias. Resolve would still be deterministic
// with a collision (sorted order), but a registry where one wire name points
// at two rows is a registry that prices the same call two ways depending on
// spelling — exactly what this table exists to remove.
func (c *ModelsConfig) validateAliases() []string {
	var problems []string
	owner := map[string]string{}
	for _, k := range c.sortedKeys() {
		for _, a := range c.Models[k].Aliases {
			a = strings.TrimSpace(a)
			if a == "" {
				problems = append(problems, fmt.Sprintf("model %q declares an empty alias", k))
				continue
			}
			if a == k {
				problems = append(problems, fmt.Sprintf("model %q lists its own key as an alias", k))
				continue
			}
			if _, isKey := c.Models[a]; isKey {
				problems = append(problems, fmt.Sprintf("model %q alias %q is another model's key", k, a))
				continue
			}
			if prev, dup := owner[a]; dup && prev != k {
				problems = append(problems, fmt.Sprintf("alias %q is declared by both %q and %q", a, prev, k))
				continue
			}
			owner[a] = k
		}
	}
	return problems
}
