package modelreg

import (
	"errors"
	"strings"
	"testing"
)

func resolveFixture() *ModelsConfig {
	return &ModelsConfig{Models: map[string]ModelConfig{
		"claude-sonnet-4-5": {APIName: "claude-sonnet-4-5-20250929", Provider: "anthropic",
			Pricing: Pricing{InputPer1K: 0.003, OutputPer1K: 0.015, CacheReadPer1K: 0.0003, CacheWritePer1K: 0.00375}},
		"gemini-2-5-pro": {APIName: "gemini-2.5-pro", Provider: "google",
			Pricing: Pricing{InputPer1K: 0.00125, OutputPer1K: 0.01}},
		"or-glm-5-3-flash": {APIName: "z-ai/glm-5.3-flash", Provider: "openrouter", Aliases: []string{"glm-5.3-flash"},
			Pricing: Pricing{InputPer1K: 0.00006, OutputPer1K: 0.00022, CacheReadPer1K: 0.000016}},
		"gpt5-5":          {APIName: "gpt-5.5", Provider: "openai", Pricing: Pricing{InputPer1K: 0.005, OutputPer1K: 0.03}},
		"opencode-gpt5-5": {APIName: "gpt-5.5", Provider: "openai", Pricing: Pricing{InputPer1K: 0.005, OutputPer1K: 0.03}},
	}}
}

func TestResolve_KeyAPINameAliasAndNormalisedForms(t *testing.T) {
	c := resolveFixture()
	cases := map[string]string{
		"claude-sonnet-4-5":          "claude-sonnet-4-5", // key
		"claude-sonnet-4-5-20250929": "claude-sonnet-4-5", // api_name
		"gemini-2.5-pro":             "gemini-2-5-pro",    // api_name with dots
		"gemini-2-5-pro-20260101":    "gemini-2-5-pro",    // dated key
		"z-ai/glm-5.3-flash":         "or-glm-5-3-flash",  // OpenRouter slug = api_name
		"glm-5.3-flash":              "or-glm-5-3-flash",  // declared alias
		"gpt-5.5":                    "gpt5-5",            // shared api_name → sorted-first twin
		"  gpt5-5  ":                 "gpt5-5",            // whitespace tolerated
	}
	for in, want := range cases {
		key, m, err := c.Resolve(in)
		if err != nil {
			t.Errorf("Resolve(%q): %v", in, err)
			continue
		}
		if key != want {
			t.Errorf("Resolve(%q) = %q, want %q", in, key, want)
		}
		if m == nil || m.APIName != c.Models[want].APIName {
			t.Errorf("Resolve(%q) returned the wrong row", in)
		}
	}
}

func TestResolve_UnknownIsErrUnknownModel(t *testing.T) {
	c := resolveFixture()
	for _, in := range []string{"", "   ", "this-model-does-not-exist", "z-ai/glm-9"} {
		_, _, err := c.Resolve(in)
		if !errors.Is(err, ErrUnknownModel) {
			t.Errorf("Resolve(%q) err = %v, want ErrUnknownModel", in, err)
		}
	}
}

// The observatory's private table (deleted 2026-09-15) mapped these wire names
// to registry keys. Every one must still resolve to the same key through the
// registry, or a deployed dashboard would start pricing rows it used to price.
func TestResolve_ShippedRegistryCoversTheOldObservatoryTable(t *testing.T) {
	if err := InitModelsConfig(); err != nil {
		t.Fatalf("init: %v", err)
	}
	old := map[string]string{
		"gpt-5":                      "gpt5",
		"gpt-5-mini":                 "gpt5-mini",
		"gpt-5.1":                    "gpt5-1",
		"gpt-5.1-chat-latest":        "gpt5-1-instant",
		"gpt-5.2":                    "gpt5-2",
		"gpt-5.2-chat-latest":        "gpt5-2-instant",
		"claude-sonnet-4-6":          "claude-sonnet-4-6",
		"claude-sonnet-4-5":          "claude-sonnet-4-5",
		"claude-haiku-4-5":           "claude-haiku-4-5",
		"claude-opus-4-5":            "claude-opus-4-5",
		"claude-opus-4-6":            "claude-opus-4-6",
		"claude-sonnet-4-5-20250929": "claude-sonnet-4-5",
		"claude-haiku-4-5-20251001":  "claude-haiku-4-5",
		"claude-opus-4-5-20251101":   "claude-opus-4-5",
		"gemini-2.5-pro":             "gemini-2-5-pro",
		"gemini-2.5-flash":           "gemini-2-5-flash",
		"gemini-3-flash-preview":     "gemini-3-flash",
		"gemini-3-pro-preview":       "gemini-3-pro",
	}
	for in, want := range old {
		key, m, err := GlobalModelsConfig.Resolve(in)
		if err != nil {
			t.Errorf("%q no longer resolves: %v", in, err)
			continue
		}
		// The key may legitimately differ when the wire name has since become
		// its own row (gemini-3-flash-preview); what must not change is the price.
		if key != want && m.Pricing != GlobalModelsConfig.Models[want].Pricing {
			t.Errorf("%q resolves to %q (%+v), the old table said %q (%+v)",
				in, key, m.Pricing, want, GlobalModelsConfig.Models[want].Pricing)
		}
	}
}

// Prod OpenRouter Broadcast spans name models by slug. Sampled 2026-09-01..13
// from dashboard.ailang.sunholo.com: every token-bearing LLM Generation span
// carried one of these and every one of them was stored at cost_usd=null,
// because the old generic path never tried api_name.
func TestResolve_ShippedRegistryCoversProdOpenRouterSlugs(t *testing.T) {
	if err := InitModelsConfig(); err != nil {
		t.Fatalf("init: %v", err)
	}
	for _, slug := range []string{"z-ai/glm-5.3-flash", "moonshotai/kimi-k3", "deepseek/deepseek-v4-flash-0731"} {
		if _, _, err := GlobalModelsConfig.Resolve(slug); err != nil {
			t.Errorf("prod slug %q does not resolve: %v", slug, err)
		}
	}
}

func TestCostForName_UnknownModelIsAnErrorNotZero(t *testing.T) {
	c := resolveFixture()
	cost, err := c.CostForName("no-such-model", 1000, 1000, 1000, 1000)
	if !errors.Is(err, ErrUnknownModel) {
		t.Fatalf("err = %v, want ErrUnknownModel", err)
	}
	if cost != 0 {
		t.Errorf("cost on error = %v, want 0 (and the error is the signal)", cost)
	}
}

// A non-Anthropic model's cache tokens must be priced from its OWN declared
// rates, not the Anthropic 10%/125% multipliers the observatory hardcoded for
// every provider until 2026-09-15.
func TestCostForName_NonAnthropicCacheIsNotTenAndOneTwentyFivePercent(t *testing.T) {
	c := resolveFixture()
	const in, out, read, write = 0, 0, 100_000, 100_000
	got, err := c.CostForName("z-ai/glm-5.3-flash", in, out, read, write)
	if err != nil {
		t.Fatal(err)
	}
	m := c.Models["or-glm-5-3-flash"]
	anthropicStyle := float64(read)/1000*m.Pricing.InputPer1K*0.1 + float64(write)/1000*m.Pricing.InputPer1K*1.25
	// Declared read rate; write undeclared → full input rate (registry stance).
	want := float64(read)/1000*m.Pricing.CacheReadPer1K + float64(write)/1000*m.Pricing.InputPer1K
	if diff := got - want; diff > 1e-12 || diff < -1e-12 {
		t.Errorf("cost = %.10f, want %.10f", got, want)
	}
	if diff := got - anthropicStyle; diff < 1e-12 && diff > -1e-12 {
		t.Errorf("cost %.10f equals the Anthropic-multiplier figure; the provider's own rates were not used", got)
	}
}

// Anthropic rows declare the 10%/125% rates the observatory used to hardcode,
// so the only Anthropic delta from the migration is rounding.
func TestCostForName_AnthropicCacheMatchesVendorMultipliers(t *testing.T) {
	c := resolveFixture()
	got, err := c.CostForName("claude-sonnet-4-5-20250929", 1000, 500, 10_000, 2_000)
	if err != nil {
		t.Fatal(err)
	}
	const inRate = 0.003
	want := 1.0*inRate + 0.5*0.015 + 10.0*inRate*0.1 + 2.0*inRate*1.25
	if diff := got - want; diff > 1e-12 || diff < -1e-12 {
		t.Errorf("cost = %.10f, want %.10f", got, want)
	}
}

func TestValidate_AliasCollisionsAreRejected(t *testing.T) {
	cases := map[string]*ModelsConfig{
		"alias equals another key": {Models: map[string]ModelConfig{
			"a": {APIName: "a", Provider: "p", Aliases: []string{"b"}},
			"b": {APIName: "b", Provider: "p"},
		}},
		"alias declared twice": {Models: map[string]ModelConfig{
			"a": {APIName: "a", Provider: "p", Aliases: []string{"x"}},
			"b": {APIName: "b", Provider: "p", Aliases: []string{"x"}},
		}},
		"empty alias": {Models: map[string]ModelConfig{
			"a": {APIName: "a", Provider: "p", Aliases: []string{""}},
		}},
	}
	for name, c := range cases {
		err := c.Validate()
		if err == nil {
			t.Errorf("%s: expected a validation error", name)
			continue
		}
		if !strings.Contains(err.Error(), "alias") {
			t.Errorf("%s: error must name the alias problem; got %v", name, err)
		}
	}
	ok := &ModelsConfig{Models: map[string]ModelConfig{
		"a": {APIName: "a", Provider: "p", Aliases: []string{"x", "y"}},
	}}
	if err := ok.Validate(); err != nil {
		t.Errorf("distinct aliases must validate: %v", err)
	}
}

// anthropicCacheReadMultiplier records the models whose cache-hit rate is NOT
// Anthropic's standard 0.1x of base input. Source: the "Cache read (hit)" row of
// platform.claude.com/docs/en/about-claude/pricing (fetched 2026-09-22). Before
// this table existed the test pinned every row to 0.1x, which ENFORCED a 4x
// overstatement on claude-fable-5-1 ($1.00/M declared vs $0.25/M billed).
var anthropicCacheReadMultiplier = map[string]float64{
	"claude-fable-5-1": 0.025,
	"claude-opus-5-5":  0.05,
}

// Every anthropic-provider row must declare its cache rates: the undeclared
// default bills reads at 100% of input, a 10x OVERSTATEMENT on the vendor whose
// cache we lean on hardest.
func TestModels_AnthropicRowsDeclareCacheRates(t *testing.T) {
	if err := InitModelsConfig(); err != nil {
		t.Fatalf("init: %v", err)
	}
	for _, k := range GlobalModelsConfig.sortedKeys() {
		m := GlobalModelsConfig.Models[k]
		if m.Provider != "anthropic" || m.Pricing.InputPer1K == 0 {
			continue
		}
		if m.Pricing.CacheReadPer1K == 0 || m.Pricing.CacheWritePer1K == 0 {
			t.Errorf("%s: anthropic row without cache_read_per_1k/cache_write_per_1k", k)
			continue
		}
		want, ok := anthropicCacheReadMultiplier[m.APIName]
		if !ok {
			want = 0.1
		}
		if r := m.Pricing.CacheReadPer1K / m.Pricing.InputPer1K; r < want*0.9 || r > want*1.1 {
			t.Errorf("%s: cache read is %.3fx input, Anthropic bills %gx for %s", k, r, want, m.APIName)
		}
		if w := m.Pricing.CacheWritePer1K / m.Pricing.InputPer1K; w < 1.24 || w > 1.26 {
			t.Errorf("%s: cache write is %.3fx input, Anthropic bills 1.25x", k, w)
		}
	}
}

func strp(s string) *string { return &s }

// The rig banks a harness's wire name as the stage model. It resolves as the
// last tier — and only when every row that carries it agrees on price.
func TestResolve_AgentModelNameIsLastTierWithAmbiguityGuard(t *testing.T) {
	c := &ModelsConfig{Models: map[string]ModelConfig{
		"pi-deepseek": {APIName: "deepseek/deepseek-v4-flash-0731", Provider: "openrouter",
			AgentModelName: strp("openrouter/deepseek/deepseek-v4-flash-0731"),
			Pricing:        Pricing{InputPer1K: 0.00008, OutputPer1K: 0.00018}},
		"claude-opus-4-7": {APIName: "claude-opus-4-7", Provider: "anthropic", AgentModelName: strp("opus"),
			Pricing: Pricing{InputPer1K: 0.005, OutputPer1K: 0.025}},
		"claude-opus-4-8": {APIName: "claude-opus-4-8", Provider: "anthropic", AgentModelName: strp("opus"),
			Pricing: Pricing{InputPer1K: 0.005, OutputPer1K: 0.025}},
		"claude-sonnet-4-6": {APIName: "claude-sonnet-4-6", Provider: "anthropic", AgentModelName: strp("sonnet"),
			Pricing: Pricing{InputPer1K: 0.003, OutputPer1K: 0.015}},
		"claude-sonnet-5": {APIName: "claude-sonnet-5", Provider: "anthropic", AgentModelName: strp("sonnet"),
			Pricing: Pricing{InputPer1K: 0.002, OutputPer1K: 0.010}},
	}}
	key, _, err := c.Resolve("openrouter/deepseek/deepseek-v4-flash-0731")
	if err != nil || key != "pi-deepseek" {
		t.Errorf("wire name: got (%q, %v), want pi-deepseek", key, err)
	}
	key, _, err = c.Resolve("opus")
	if err != nil || key != "claude-opus-4-7" {
		t.Errorf("consistently-priced short name: got (%q, %v), want claude-opus-4-7 (sorted-first)", key, err)
	}
	_, _, err = c.Resolve("sonnet")
	if !errors.Is(err, ErrAmbiguousModel) {
		t.Errorf("a short name whose rows disagree on price must be ErrAmbiguousModel, got %v", err)
	}
	if _, err := c.CostForName("sonnet", 1000, 1000, 0, 0); err == nil {
		t.Error("CostForName must refuse an ambiguous name rather than pick a rate")
	}
}

// The two pi wire names seen unpriced in the local observatory sample
// (2026-09-15) must resolve through the shipped registry.
func TestResolve_ShippedRegistryCoversRigWireNames(t *testing.T) {
	if err := InitModelsConfig(); err != nil {
		t.Fatalf("init: %v", err)
	}
	for _, w := range []string{"openrouter/deepseek/deepseek-v4-flash-0731", "ollama/deepseek-v4-flash:0731-cloud"} {
		if _, _, err := GlobalModelsConfig.Resolve(w); err != nil {
			t.Errorf("rig wire name %q does not resolve: %v", w, err)
		}
	}
}
