package opencode

import (
	"context"
	"math"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/sunholo-data/ailang/internal/executor"
	"github.com/sunholo-data/ailang/internal/modelreg"
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

func TestOpencodeRunCostIsRegistryPriced(t *testing.T) {
	type tokens struct{ in, out, reason, cacheRead, cacheWrite int }
	cases := []struct {
		name     string
		model    string
		pricing  *executor.CostModel // Task.Pricing; nil = resolve from the model string
		tok      tokens
		wantUSD  float64
		wantProv executor.CostProvenance
	}{
		{
			// The mockEvents fixture: 2 in, 159 out, 34,436 cache write. Haiku 4.5
			// ($1 in / $5 out / $1.25 cache write per 1M):
			//   2×1 + 159×5 + 34,436×1.25 (per 1M) = 0.000002 + 0.000795 + 0.043045 = $0.043842
			name: "haiku 4.5 via the opencode row's agent_model_name", model: "anthropic/claude-haiku-4-5",
			tok: tokens{in: 2, out: 159, cacheWrite: 34436}, wantUSD: 0.043842, wantProv: executor.CostMetered,
		},
		{
			// GLM-5.3-Flash via OpenRouter ($0.15 in / $0.50 out / $0.03 cache read per 1M).
			// opencode reports cache EXCLUSIVE of input, and reasoning disjoint from output:
			//   40,000×0.15 + 360,000×0.03 + (12,000+8,000)×0.50 (per 1M)
			//   = 0.006 + 0.0108 + 0.010 = $0.0268
			name:  "glm-5.3-flash with Task.Pricing; reasoning bills at the output rate",
			model: "openrouter/z-ai/glm-5.3-flash", pricing: registryPricing(t, "opencode-or-glm-5-3-flash"),
			tok:     tokens{in: 40000, out: 12000, reason: 8000, cacheRead: 360000},
			wantUSD: 0.0268, wantProv: executor.CostMetered,
		},
		{
			// Ollama Cloud is a SUBSCRIPTION lane (D1): imputed $3 in / $15 out per 1M,
			//   10,000×3 + 2,000×15 (per 1M) = 0.03 + 0.03 = $0.06, never metered.
			name: "ollama cloud route is list-price-equivalent", model: "ollama/kimi-k3:cloud",
			tok: tokens{in: 10000, out: 2000}, wantUSD: 0.06, wantProv: executor.CostListPriceEquivalent,
		},
		{
			name: "local ollama model is free-local", model: "ollama/qwen3.6:35b-a3b-mxfp8",
			tok: tokens{in: 10000, out: 2000}, wantUSD: 0, wantProv: executor.CostFreeLocal,
		},
		{
			name: "provider mismatch never borrows a cloud row's price", model: "ollama/claude-haiku-4-5",
			tok: tokens{in: 10000, out: 2000}, wantUSD: 0, wantProv: executor.CostProvenanceUnknown,
		},
		{
			name: "unknown model is unpriced and says so", model: "openrouter/nobody/nonexistent-9",
			tok: tokens{in: 10000, out: 2000}, wantUSD: 0, wantProv: executor.CostProvenanceUnknown,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, prov := opencodeRunCost(&executor.Task{Model: tc.model, Pricing: tc.pricing}, tc.model,
				tc.tok.in, tc.tok.out, tc.tok.reason, tc.tok.cacheRead, tc.tok.cacheWrite)
			if math.Abs(got-tc.wantUSD) > 1e-9 {
				t.Errorf("cost = $%.9f, want $%.9f", got, tc.wantUSD)
			}
			if prov != tc.wantProv {
				t.Errorf("provenance = %q, want %q", prov, tc.wantProv)
			}
		})
	}
}

// End to end: a stream whose step_finish.cost is opencode's own (wrong) figure.
// Result.CostUSD must be registry × tokens; opencode's number survives only as
// ProviderData diagnostic.
func TestOpencodeResultCostUSDIgnoresSelfReportedCost(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip(skipWindows)
	}
	const selfReported = 1.25 // per step; two steps
	events := mockEvents()
	for i, ev := range events {
		if strings.Contains(ev, `"type":"step_finish"`) {
			events[i] = strings.NewReplacer(`"cost":0.02164725`, `"cost":1.25`, `"cost":0.02219475`, `"cost":1.25`).Replace(ev)
		}
	}
	dir := t.TempDir()
	_ = writeFakeOpenCode(t, dir, events)
	e, err := New(&executor.Config{OpenCodePath: filepath.Join(dir, "opencode"), OpenCodeModel: "anthropic/claude-haiku-4-5", TimeoutSeconds: 10})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	result, err := e.ExecuteStreaming(context.Background(), &executor.Task{
		ID: "oc-cost", Directive: "replay", Workspace: dir, Timeout: 10 * time.Second,
	}, &collectingHandler{})
	if err != nil {
		t.Fatalf("ExecuteStreaming: %v", err)
	}
	if math.Abs(result.CostUSD-0.043842) > 1e-9 {
		t.Errorf("CostUSD = $%.9f, want $0.043842 (registry × tokens) — opencode claimed $%.2f", result.CostUSD, 2*selfReported)
	}
	if result.CostProvenance != executor.CostMetered {
		t.Errorf("CostProvenance = %q, want metered", result.CostProvenance)
	}
	if got := result.ProviderData["opencode_self_reported_cost_usd"]; got != 2*selfReported {
		t.Errorf("opencode_self_reported_cost_usd = %v, want %v", got, 2*selfReported)
	}
}
