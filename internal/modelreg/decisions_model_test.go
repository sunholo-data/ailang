package modelreg

import (
	"math"
	"testing"
)

// M-AI-DECIDE-SYSTEM-ONE M2: TypeSafe Jev is a `text->decisions` model on
// OpenRouter. The observatory prices Broadcast spans through Resolve, and the
// wire name it sees is the DATED endpoint id ("typesafe/jev-1.13-20260917").
// Resolve normalises only the query (date stripped, dots to dashes), so the
// row must declare the normalised spelling as an alias or the dated name
// prices as `unpriced`. Both spellings must land on the same row.
func TestResolve_TypeSafeJevDatedAndBareWireNames(t *testing.T) {
	if err := InitModelsConfig(); err != nil {
		t.Fatalf("init: %v", err)
	}
	for _, name := range []string{"typesafe/jev-1.13", "typesafe/jev-1.13-20260917", "typesafe/jev-1-13"} {
		key, m, err := GlobalModelsConfig.Resolve(name)
		if err != nil {
			t.Errorf("Resolve(%q): %v", name, err)
			continue
		}
		if key != "or-typesafe-jev-1-13" {
			t.Errorf("Resolve(%q) = %q, want or-typesafe-jev-1-13", name, key)
		}
		if m.Provider != "openrouter" {
			t.Errorf("Resolve(%q).Provider = %q, want openrouter", name, m.Provider)
		}
	}
}

// The spike's measured call: 424 in / 73 out, OpenRouter billed $1.7808e-05
// ($0.042/MTok input, output free). Output must price at exactly zero — this is
// a positive input price with a zero output price, NOT a free model.
func TestCostForName_TypeSafeJevMatchesOpenRouterBill(t *testing.T) {
	if err := InitModelsConfig(); err != nil {
		t.Fatalf("init: %v", err)
	}
	cost, err := GlobalModelsConfig.CostForName("typesafe/jev-1.13-20260917", 424, 73, 0, 0)
	if err != nil {
		t.Fatalf("CostForName: %v", err)
	}
	if math.Abs(cost-1.7808e-05) > 1e-12 {
		t.Errorf("cost = %.12g, want 1.7808e-05 (OpenRouter usage.cost for the banked spike call)", cost)
	}
	_, m, _ := GlobalModelsConfig.Resolve("typesafe/jev-1.13")
	if m.Pricing.InputPer1K <= 0 {
		t.Errorf("input price must be positive (%.9f) — a 0/0 row would read as free", m.Pricing.InputPer1K)
	}
	if m.Pricing.OutputPer1K != 0 {
		t.Errorf("output price = %v, want 0 (OpenRouter lists completion at $0)", m.Pricing.OutputPer1K)
	}
}

// A decisions model cannot answer a chat prompt. It must never be selectable by
// eval tooling: no suite membership, no agent harness fields. `eval-suite`
// with no --models runs dev_models × every tier (real spend) and would bank
// 100% api_error rows for it.
func TestModels_TypeSafeJevIsPricedButNotInAnySuite(t *testing.T) {
	if err := InitModelsConfig(); err != nil {
		t.Fatalf("init: %v", err)
	}
	const key = "or-typesafe-jev-1-13"
	m, ok := GlobalModelsConfig.Models[key]
	if !ok {
		t.Fatalf("row %q missing", key)
	}
	if m.AgentCLI != nil || m.AgentModelName != nil {
		t.Errorf("row %q carries agent fields (cli=%v model=%v); a text->decisions model is not an agent harness target", key, m.AgentCLI, m.AgentModelName)
	}
	suites := map[string][]string{
		"extended_suite":     GlobalModelsConfig.ExtendedSuite,
		"dev_models":         GlobalModelsConfig.DevModels,
		"agent_suite":        GlobalModelsConfig.AgentSuite,
		"benchmark_suite":    GlobalModelsConfig.BenchmarkSuite,
		"ollama_suite":       GlobalModelsConfig.OllamaSuite,
		"harness_suite":      GlobalModelsConfig.HarnessSuite,
		"lang_harness_suite": GlobalModelsConfig.LangHarnessSuite,
	}
	for suite, members := range suites {
		for _, member := range members {
			if member == key {
				t.Errorf("%s lists %q — a decisions model must not be an eval model", suite, key)
			}
		}
	}
}
