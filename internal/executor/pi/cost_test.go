package pi

import (
	"context"
	"encoding/json"
	"math"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/sunholo-data/ailang/internal/executor"
	"github.com/sunholo-data/ailang/internal/modelreg"
)

// One measured pi run on anthropic/claude-haiku-5-5 (2026-10-12). At Haiku 5.5 list
// price ($0.10/M in, $0.50/M out, $0.01/M cache read, $0.125/M cache write):
//
//	30×0.10 + 7370×0.50 + 1,485,217×0.01 + 146,923×0.125  (per 1M) = $0.036905545
//
// pi's own price table put six such runs at $15.49 (~$2.58 each, ~56x over).
const (
	haiku55Input      = 30
	haiku55Output     = 7370
	haiku55CacheRead  = 1485217
	haiku55CacheWrite = 146923
	haiku55WantUSD    = 0.000003 + 0.003685 + 0.01485217 + 0.018365375
	piSelfReportedUSD = 2.58 // what pi claimed for a run like this — must NOT be the cost
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

func TestPiRunCostIsRegistryPriced(t *testing.T) {
	cases := []struct {
		name     string
		pricing  *executor.CostModel // Task.Pricing; nil = resolve from the model string
		model    string
		wantUSD  float64
		wantProv executor.CostProvenance
	}{
		{
			name:     "Task.Pricing from the haiku-5-5 row",
			pricing:  registryPricing(t, "claude-haiku-5-5"),
			model:    "anthropic/claude-haiku-5-5",
			wantUSD:  haiku55WantUSD,
			wantProv: executor.CostMetered,
		},
		{
			name:     "no Task.Pricing: pi provider/model resolves via the bare id",
			model:    "anthropic/claude-haiku-5-5",
			wantUSD:  haiku55WantUSD,
			wantProv: executor.CostMetered,
		},
		{
			name:     "unknown model is unpriced and says so",
			model:    "anthropic/claude-nonexistent-9-9",
			wantUSD:  0,
			wantProv: executor.CostProvenanceUnknown,
		},
		{
			// The bare id resolves to an anthropic row; an ollama-prefixed name must
			// not borrow its price.
			name:     "provider mismatch never borrows another provider's row",
			model:    "ollama/claude-haiku-5-5",
			wantUSD:  0,
			wantProv: executor.CostProvenanceUnknown,
		},
		{
			name:     "zero-rate Task.Pricing is free-local",
			pricing:  &executor.CostModel{ProviderName: "ollama"},
			model:    "ollama/gemma4:31b",
			wantUSD:  0,
			wantProv: executor.CostFreeLocal,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			task := &executor.Task{Model: tc.model, Pricing: tc.pricing}
			got, prov := piRunCost(task, tc.model, haiku55Input, haiku55Output, 0, haiku55CacheRead, haiku55CacheWrite)
			if math.Abs(got-tc.wantUSD) > 1e-9 {
				t.Errorf("cost = $%.9f, want $%.9f", got, tc.wantUSD)
			}
			if prov != tc.wantProv {
				t.Errorf("provenance = %q, want %q", prov, tc.wantProv)
			}
		})
	}
}

// Reasoning tokens are disjoint from OutputTokens on the Result but bill at the
// output rate; dropping them would under-report a thinking model.
func TestPiRunCostBillsReasoningAtOutputRate(t *testing.T) {
	task := &executor.Task{Pricing: registryPricing(t, "claude-haiku-5-5")}
	without, _ := piRunCost(task, "anthropic/claude-haiku-5-5", 0, 1000, 0, 0, 0)
	with, _ := piRunCost(task, "anthropic/claude-haiku-5-5", 0, 1000, 1000, 0, 0)
	if math.Abs(with-2*without) > 1e-12 || without == 0 {
		t.Errorf("1000 out + 1000 reasoning = $%.9f, want 2 × $%.9f", with, without)
	}
}

// End to end: a pi stream whose usage carries the measured Haiku 5.5 run and pi's
// inflated cost.total. Result.CostUSD must be the registry figure; pi's number
// survives only as ProviderData diagnostic.
func TestPiResultCostUSDIgnoresPiSelfReportedCost(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip(skipWindows)
	}
	dir := t.TempDir()
	writeFakePi(t, dir, setAssistantUsage(t, loadFixtureLines(t, "v0_85_1/fizzbuzz.ndjson")))
	e, err := New(&executor.Config{PiPath: filepath.Join(dir, "pi"), PiModel: "anthropic/claude-haiku-5-5"})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	result, err := e.ExecuteStreaming(context.Background(), &executor.Task{
		ID: "pi-cost", Directive: "replay", Workspace: dir, Timeout: 30 * time.Second,
	}, &collectingHandler{})
	if err != nil {
		t.Fatalf("ExecuteStreaming: %v", err)
	}
	if result.CacheReadInputTokens != haiku55CacheRead || result.CacheCreationInputTokens != haiku55CacheWrite {
		t.Fatalf("fixture patch did not land: cache %d/%d", result.CacheReadInputTokens, result.CacheCreationInputTokens)
	}
	if math.Abs(result.CostUSD-haiku55WantUSD) > 1e-9 {
		t.Errorf("CostUSD = $%.6f, want $%.6f (registry × tokens) — pi claimed $%.2f",
			result.CostUSD, haiku55WantUSD, piSelfReportedUSD)
	}
	if result.CostProvenance != executor.CostMetered {
		t.Errorf("CostProvenance = %q, want metered", result.CostProvenance)
	}
	if got := result.ProviderData["pi_self_reported_cost_usd"]; got != piSelfReportedUSD {
		t.Errorf("pi_self_reported_cost_usd = %v, want %v", got, piSelfReportedUSD)
	}
}

// The live budget gate prices the same tokens at the same registry rates: a $0.10
// cap must NOT trip on this $0.037 run (pi's $2.58 would have).
func TestPiBudgetGateUsesRegistryRatesIncludingCache(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip(skipWindows)
	}
	cm := registryPricing(t, "claude-haiku-5-5")
	run := func(maxUSD float64) *executor.Result {
		dir := t.TempDir()
		writeFakePi(t, dir, setAssistantUsage(t, loadFixtureLines(t, "v0_85_1/fizzbuzz.ndjson")))
		e, err := New(&executor.Config{PiPath: filepath.Join(dir, "pi"), PiModel: "anthropic/claude-haiku-5-5"})
		if err != nil {
			t.Fatalf("New: %v", err)
		}
		r, err := e.ExecuteStreaming(context.Background(), &executor.Task{
			ID: "pi-budget", Directive: "replay", Workspace: dir, Timeout: 30 * time.Second,
			Pricing: cm,
			Budget:  executor.NewCostBudgetWithCache(maxUSD, cm.InputTokenCost, cm.OutputTokenCost, cm.CacheReadCost, cm.CacheWriteCost),
		}, &collectingHandler{})
		if err != nil {
			t.Fatalf("ExecuteStreaming: %v", err)
		}
		return r
	}
	if r := run(0.10); r.CostKilledAt != 0 || r.FinishReason == executor.FinishCostExhausted {
		t.Errorf("$0.10 cap killed a $%.4f run (CostKilledAt=%.4f)", haiku55WantUSD, r.CostKilledAt)
	}
	// Cache is ~90% of this run's bill: a cap between the fresh-only and the full
	// figure trips only if the gate sees cache.
	if r := run(0.02); r.FinishReason != executor.FinishCostExhausted {
		t.Errorf("$0.02 cap did not trip on a $%.4f run — cache tokens not reaching the gate (finish=%q)",
			haiku55WantUSD, r.FinishReason)
	}
}

// setAssistantUsage rewrites every assistant message_end/turn_end usage to the
// measured Haiku 5.5 run (the fizzbuzz fixture has exactly one assistant turn).
func setAssistantUsage(t *testing.T, events []string) []string {
	t.Helper()
	out := make([]string, 0, len(events))
	for _, ln := range events {
		if !strings.Contains(ln, `"role":"assistant"`) || !(strings.Contains(ln, `"type":"message_end"`) || strings.Contains(ln, `"type":"turn_end"`)) {
			out = append(out, ln)
			continue
		}
		var ev map[string]any
		if err := json.Unmarshal([]byte(ln), &ev); err != nil {
			t.Fatalf("setAssistantUsage: %v", err)
		}
		msg, _ := ev["message"].(map[string]any)
		usage, _ := msg["usage"].(map[string]any)
		if usage == nil {
			out = append(out, ln)
			continue
		}
		usage["input"], usage["output"], usage["reasoning"] = haiku55Input, haiku55Output, 0
		usage["cacheRead"], usage["cacheWrite"] = haiku55CacheRead, haiku55CacheWrite
		if cost, _ := usage["cost"].(map[string]any); cost != nil {
			cost["total"] = piSelfReportedUSD
		}
		b, err := json.Marshal(ev)
		if err != nil {
			t.Fatalf("setAssistantUsage: %v", err)
		}
		out = append(out, string(b))
	}
	return out
}
