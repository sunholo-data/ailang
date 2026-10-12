package motoko

import (
	"math"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/sunholo-data/ailang/internal/executor"
	"github.com/sunholo-data/ailang/internal/modelreg"
)

// Two real run_summary usage blocks banked 2026-10-12 (mk-lanefix logfiles).
//
// Haiku 5.5, direct Anthropic (session_task-e0a711f0-motoko-hk3). Usage is
// DISJOINT: input 54 is fresh, cache counters are separate. Registry
// ($0.10 in / $0.50 out / $0.01 cache read / $0.125 cache write per 1M):
//
//	54×0.10 + 26,858×0.50 + 2,553,032×0.01 + 124,706×0.125  (per 1M)
//	= 0.0000054 + 0.013429 + 0.02553032 + 0.01558825 = $0.05455297
//
// motoko's meter priced input+output only: $0.01329.
//
// GLM-5.3-Flash via OpenRouter (session_task-503c588e-motoko-g4). Usage is
// cache-INCLUSIVE: input 13,272,269 contains the 12,998,912 cache reads, so
// fresh input is 273,357. Registry ($0.15 in / $0.50 out / $0.03 cache read):
//
//	273,357×0.15 + 12,998,912×0.03 + 24,330×0.50  (per 1M)
//	= 0.04100355 + 0.38996736 + 0.012165 = $0.44313591
//
// motoko's meter billed all 13.27M input at $0.15/M: $2.00187 (and its cost cap
// killed the run there).
const (
	hk3Input, hk3Output, hk3CacheRead, hk3CacheWrite = 54, 26858, 2553032, 124706
	hk3WantUSD, hk3MotokoUSD                         = 0.05455297, 0.01329

	glmInput, glmOutput, glmCacheRead = 13272269, 24330, 12998912
	glmFresh                          = glmInput - glmCacheRead // 273,357
	glmWantUSD, glmMotokoUSD          = 0.44313591, 2.00187
)

func registryPricing(t *testing.T, key string) *executor.CostModel {
	t.Helper()
	if modelreg.GlobalModelsConfig == nil {
		if err := modelreg.InitModelsConfig(); err != nil {
			t.Fatalf("InitModelsConfig: %v", err)
		}
	}
	m, ok := modelreg.GlobalModelsConfig.Models[key]
	if !ok {
		t.Fatalf("registry has no row %q", key)
	}
	return executor.CostModelFromPricing(m.Provider, key, m.Pricing)
}

func TestApplyRegistryCost(t *testing.T) {
	cases := []struct {
		name                           string
		model                          string
		pricing                        *executor.CostModel // Task.Pricing; nil = resolve from the model string
		in, out, cacheRead, cacheWrite int
		motokoUSD                      float64
		wantIn                         int
		wantUSD                        float64
		wantProv                       executor.CostProvenance
	}{
		{
			name:  "haiku 5.5 direct anthropic: disjoint cache now priced (was $0.013)",
			model: "anthropic/claude-haiku-5-5", in: hk3Input, out: hk3Output,
			cacheRead: hk3CacheRead, cacheWrite: hk3CacheWrite, motokoUSD: hk3MotokoUSD,
			wantIn: hk3Input, wantUSD: hk3WantUSD, wantProv: executor.CostMetered,
		},
		{
			name:  "haiku 5.5 with Task.Pricing from the registry row",
			model: "anthropic/claude-haiku-5-5", pricing: registryPricing(t, "claude-haiku-5-5"),
			in: hk3Input, out: hk3Output, cacheRead: hk3CacheRead, cacheWrite: hk3CacheWrite, motokoUSD: hk3MotokoUSD,
			wantIn: hk3Input, wantUSD: hk3WantUSD, wantProv: executor.CostMetered,
		},
		{
			name:  "glm-5.3-flash via openrouter: cache reads leave input, priced at cache rate (was $2.00)",
			model: "openrouter/z-ai/glm-5.3-flash", in: glmInput, out: glmOutput,
			cacheRead: glmCacheRead, motokoUSD: glmMotokoUSD,
			wantIn: glmFresh, wantUSD: glmWantUSD, wantProv: executor.CostMetered,
		},
		{
			name:  "glm-5.3-flash with Task.Pricing from motoko-lane row",
			model: "openrouter/z-ai/glm-5.3-flash", pricing: registryPricing(t, "motoko-lane-or-glm-5-3-flash"),
			in: glmInput, out: glmOutput, cacheRead: glmCacheRead, motokoUSD: glmMotokoUSD,
			wantIn: glmFresh, wantUSD: glmWantUSD, wantProv: executor.CostMetered,
		},
		{
			// motoko-chatgpt-gpt-6-1-sol: $2/M in, $10/M out, no cache rate declared →
			// cache reads bill at the input rate. 100k inclusive, 80k cached, 5k out:
			//   20,000×2 + 80,000×2 + 5,000×10 (per 1M) = 0.04 + 0.16 + 0.05 = $0.25
			name:  "chatgpt subscription lane is list-price-equivalent, never metered",
			model: "chatgpt/gpt-6.1-sol", in: 100000, out: 5000, cacheRead: 80000, motokoUSD: 0.21,
			wantIn: 20000, wantUSD: 0.25, wantProv: executor.CostListPriceEquivalent,
		},
		{
			name:  "local ollama model is free-local whatever motoko's meter said",
			model: "ollama/qwen3.6:35b-a3b-mxfp8", in: 579, out: 101, cacheRead: 128, motokoUSD: 0.000813,
			wantIn: 451, wantUSD: 0, wantProv: executor.CostFreeLocal,
		},
		{
			// The bare id resolves to the anthropic row; an ollama route must not
			// borrow its price.
			name:  "provider mismatch never borrows a cloud row's price",
			model: "ollama/claude-haiku-5-5", in: 1000, out: 100, motokoUSD: 0.001,
			wantIn: 1000, wantUSD: 0, wantProv: executor.CostProvenanceUnknown,
		},
		{
			name:  "unknown model is unpriced and says so",
			model: "openrouter/nobody/nonexistent-9", in: 1000, out: 100, motokoUSD: 0.5,
			wantIn: 1000, wantUSD: 0, wantProv: executor.CostProvenanceUnknown,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			res := &executor.Result{
				InputTokens: tc.in, OutputTokens: tc.out,
				CacheReadInputTokens: tc.cacheRead, CacheCreationInputTokens: tc.cacheWrite,
				CostUSD: tc.motokoUSD, ProviderData: map[string]any{},
			}
			applyRegistryCost(res, &executor.Task{Model: tc.model, Pricing: tc.pricing}, tc.model)
			if math.Abs(res.CostUSD-tc.wantUSD) > 1e-9 {
				t.Errorf("CostUSD = $%.9f, want $%.9f (motoko said $%v)", res.CostUSD, tc.wantUSD, tc.motokoUSD)
			}
			if res.CostProvenance != tc.wantProv {
				t.Errorf("CostProvenance = %q, want %q", res.CostProvenance, tc.wantProv)
			}
			if res.InputTokens != tc.wantIn {
				t.Errorf("InputTokens = %d, want %d fresh", res.InputTokens, tc.wantIn)
			}
			if res.CacheReadInputTokens != tc.cacheRead {
				t.Errorf("CacheReadInputTokens = %d, want %d unchanged", res.CacheReadInputTokens, tc.cacheRead)
			}
			if got := res.ProviderData["motoko_self_reported_cost_usd"]; got != tc.motokoUSD {
				t.Errorf("motoko_self_reported_cost_usd = %v, want %v", got, tc.motokoUSD)
			}
		})
	}
}

// A cache-inclusive route whose cache reads exceed its input breaks the subset
// relationship: the split is unknown, so the run is unpriced, not clamped.
func TestApplyRegistryCostRefusesImpossibleCacheSplit(t *testing.T) {
	res := &executor.Result{InputTokens: 100, OutputTokens: 10, CacheReadInputTokens: 500, CostUSD: 0.2, ProviderData: map[string]any{}}
	applyRegistryCost(res, &executor.Task{}, "openrouter/z-ai/glm-5.3-flash")
	if res.CostUSD != 0 || res.CostProvenance != executor.CostProvenanceUnknown {
		t.Errorf("cost = $%v (%q), want $0 unknown", res.CostUSD, res.CostProvenance)
	}
	if _, ok := res.ProviderData["motoko_cost_unpriced_reason"]; !ok {
		t.Error("no motoko_cost_unpriced_reason recorded")
	}
}

func TestMotokoInputIncludesCacheReads(t *testing.T) {
	for model, want := range map[string]bool{
		"anthropic/claude-haiku-5-5":            false, // direct Anthropic: disjoint
		"claude-haiku-5-5":                      false, // bare claude → anthropic
		"openrouter/z-ai/glm-5.3-flash":         true,
		"openrouter/anthropic/claude-haiku-4.5": true, // OpenRouter normalises to the OpenAI shape
		"chatgpt/gpt-6.1-sol":                   true,
		"openai/gpt-6.1-sol":                    true,
		"google/gemini-3-flash":                 true,
		"ollama/qwen3.6:35b-a3b-mxfp8":          true,
	} {
		if got := motokoInputIncludesCacheReads(model); got != want {
			t.Errorf("motokoInputIncludesCacheReads(%q) = %v, want %v", model, got, want)
		}
	}
}

// Parser + cost together, from the real GLM run_summary line: the Result must
// carry fresh input and the registry cost, with motoko's $2.00 as diagnostic.
func TestParsedGLMRunIsRegistryPriced(t *testing.T) {
	line := `{"schema_version":"1","session_id":"s","type":"run_summary","model":"openrouter/z-ai/glm-5.3-flash","finish_reason":"cost_exhausted","steps_executed":198,"usage":{"input_tokens":13272269,"output_tokens":24330,"cache_read_input_tokens":12998912,"cache_creation_input_tokens":0,"total_tokens":13296599},"total_cost_usd":2.00187,"total_cost_millicents":200187,"duration_ms":462093,"error":"cost cap reached: 200187 of 200000 millicents used"}` + "\n"
	path := filepath.Join(t.TempDir(), "s.jsonl")
	if err := os.WriteFile(path, []byte(line), 0o644); err != nil {
		t.Fatal(err)
	}
	res, err := parseSessionJSONL(path)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	applyRegistryCost(res, &executor.Task{Model: "openrouter/z-ai/glm-5.3-flash"}, "openrouter/z-ai/glm-5.3-flash")
	if math.Abs(res.CostUSD-glmWantUSD) > 1e-9 || res.InputTokens != glmFresh {
		t.Errorf("CostUSD = $%.8f, InputTokens = %d; want $%.8f, %d", res.CostUSD, res.InputTokens, glmWantUSD, glmFresh)
	}
	if got := res.ProviderData["motoko_reported_input_tokens"]; got != glmInput {
		t.Errorf("motoko_reported_input_tokens = %v, want %d", got, glmInput)
	}
	// The kill point stays the figure motoko's own cap compared against.
	if res.CostKilledAt != glmMotokoUSD {
		t.Errorf("CostKilledAt = %v, want motoko's %v", res.CostKilledAt, glmMotokoUSD)
	}
}

// costRateEnv forwards all four rates and the cap in motoko's units, and
// omits what the budget leaves unpriced (motoko then prices cache tokens at
// the input rate).
func TestCostRateEnv(t *testing.T) {
	b := executor.NewCostBudgetWithCache(2, 0.00015, 0.0005, 0.00003, 0)
	got := strings.Join(costRateEnv(b), " ")
	for _, want := range []string{
		"MOTOKO_COST_INPUT_PER_1M_MILLICENTS=15000",
		"MOTOKO_COST_OUTPUT_PER_1M_MILLICENTS=50000",
		"MOTOKO_COST_CACHE_READ_PER_1M_MILLICENTS=3000",
		"AI_MAX_COST_USD_CENTS=200",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("costRateEnv = %q, missing %s", got, want)
		}
	}
	if strings.Contains(got, "CACHE_WRITE") {
		t.Errorf("an unpriced cache-write rate must be omitted, got %q", got)
	}
	if costRateEnv(nil) != nil {
		t.Error("no budget, no env")
	}
}
