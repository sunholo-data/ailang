package eval_harness

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/sunholo-data/ailang/internal/executor"
	"github.com/sunholo-data/ailang/internal/modelreg"
	"github.com/sunholo-data/ailang/internal/testutil"
)

// TestStandardModeCostProvenance: standard mode reaches providers over metered
// HTTP APIs, so a priced model is genuinely billed — unlike agent mode, where
// the codex/claude CLIs run on subscriptions. Unresolvable must not guess.
func TestStandardModeCostProvenance(t *testing.T) {
	saved := modelreg.GlobalModelsConfig
	t.Cleanup(func() { modelreg.GlobalModelsConfig = saved })

	modelreg.GlobalModelsConfig = &ModelsConfig{Models: map[string]ModelConfig{
		"gpt5-6-luna": {Pricing: Pricing{InputPer1K: 0.0002, OutputPer1K: 0.0012}},
		"local-gemma": {Pricing: Pricing{}},
	}}

	tests := []struct{ model, want string }{
		{"gpt5-6-luna", string(executor.CostMetered)},
		{"local-gemma", string(executor.CostFreeLocal)},
		{"never-heard-of-it", string(executor.CostProvenanceUnknown)},
	}
	for _, tt := range tests {
		if got := standardModeCostProvenance(tt.model); got != tt.want {
			t.Errorf("%s: got %q, want %q", tt.model, got, tt.want)
		}
	}

	modelreg.GlobalModelsConfig = nil
	if got := standardModeCostProvenance("gpt5-6-luna"); got != string(executor.CostProvenanceUnknown) {
		t.Errorf("no config loaded: got %q, want unknown", got)
	}
}

// TestRunMetrics_CostProvenanceRoundTrip pins the banked JSON contract: the
// label ships with the row, and its ABSENCE on a pre-2026-07-30 baseline
// decodes as unknown rather than silently reading as metered.
func TestRunMetrics_CostProvenanceRoundTrip(t *testing.T) {
	saved := modelreg.GlobalModelsConfig
	t.Cleanup(func() { modelreg.GlobalModelsConfig = saved })
	modelreg.GlobalModelsConfig = &ModelsConfig{Models: map[string]ModelConfig{
		"gpt5-6-luna": {Pricing: Pricing{InputPer1K: 0.0002, OutputPer1K: 0.0012}},
	}}

	m := NewRunMetrics("fizzbuzz", "ailang", "gpt5-6-luna", 42)
	blob, err := json.Marshal(m)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var back RunMetrics
	if err := json.Unmarshal(blob, &back); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if back.CostProvenance != string(executor.CostMetered) {
		t.Errorf("round-tripped provenance = %q, want %q", back.CostProvenance, executor.CostMetered)
	}

	// A pre-fix banked row has no cost_provenance key at all.
	var legacy RunMetrics
	if err := json.Unmarshal([]byte(`{"id":"fizzbuzz","cost_usd":0.34259375}`), &legacy); err != nil {
		t.Fatalf("legacy unmarshal: %v", err)
	}
	if legacy.CostProvenance != "" {
		t.Errorf("legacy row provenance = %q, want empty (reads as unknown)", legacy.CostProvenance)
	}
}

// TestResolvedMaxTokensPerBench: the per-model work gate overrides the global
// --max-tokens-per-bench flag, and 0 from both means no enforcement.
func TestResolvedMaxTokensPerBench(t *testing.T) {
	tests := []struct {
		name     string
		perModel int
		flag     int
		want     int
	}{
		{"per-model wins over the flag", 3_000_000, 500_000, 3_000_000},
		{"flag applies when no per-model budget", 0, 500_000, 500_000},
		{"both unset = unlimited", 0, 0, 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := &ModelConfig{Budgets: Budgets{MaxTokensPerBench: tt.perModel}}
			if got := m.ResolvedMaxTokensPerBench(tt.flag); got != tt.want {
				t.Errorf("got %d, want %d", got, tt.want)
			}
		})
	}
}

// TestAgentSuiteGatesAreConsistent is the guard for the 2026-07-31 audit: a
// dollar-only gate buys work in inverse proportion to price, which let the
// suite span 24x in actual tokens and left claude-sonnet-4-6 — the longitudinal
// ANCHOR — with the tightest budget of all six. Every agent_suite model must
// now carry an explicit token gate, and the subscription lanes must all land on
// the SAME effective work ceiling.
func TestAgentSuiteGatesAreConsistent(t *testing.T) {
	cfg, err := LoadModelsConfig("../modelreg/models.yml")
	if err != nil {
		t.Fatalf("load models.yml: %v", err)
	}
	subscription := map[string]bool{"codex": true, "claude": true}
	var subWork []float64

	for _, name := range cfg.AgentSuite {
		m, ok := cfg.Models[name]
		if !ok {
			t.Fatalf("agent_suite references unknown model %q", name)
		}
		tok := m.Budgets.MaxTokensPerBench
		if tok == 0 {
			t.Errorf("%s: no max_tokens_per_bench — a dollar-only gate is not a work gate", name)
			continue
		}
		// Blended rate at the ~95/5 input/output mix agent runs actually show.
		blended := m.Pricing.InputPer1K*0.95 + m.Pricing.OutputPer1K*0.05
		byCost := m.ResolvedMaxCostUSD() / blended * 1000
		binds := byCost
		if float64(tok) < binds {
			binds = float64(tok)
		}
		cli := ""
		if m.AgentCLI != nil {
			cli = *m.AgentCLI
		}
		if subscription[cli] {
			// No spend to control here, so the token gate MUST be the binding one.
			if byCost < float64(tok) {
				t.Errorf("%s: dollar cap binds first (%.2fM tok) — token gate never fires on a subscription lane",
					name, byCost/1e6)
			}
			subWork = append(subWork, binds)
		}
	}
	for i := 1; i < len(subWork); i++ {
		if subWork[i] != subWork[0] {
			t.Errorf("subscription lanes disagree on work ceiling: %.2fM vs %.2fM",
				subWork[0]/1e6, subWork[i]/1e6)
		}
	}
	if len(subWork) < 2 {
		t.Fatalf("expected at least 2 subscription-lane models in agent_suite, got %d", len(subWork))
	}
}

// TestStandardModeCostProvenance_AnthropicOAuthIsNotMetered pins the billing
// integrity rule added with standard-mode OAuth (M-EVAL-STANDARD-OAUTH).
//
// Standard mode can now authenticate to Anthropic with a subscription access
// token. The token count and the arithmetic are identical either way, so the
// ONLY thing distinguishing "we were charged $12" from "we spent quota" is this
// label. Getting it wrong is not cosmetic: every cost-per-pass comparison and
// every budget guard reads it, and a metered label on a subscription run
// invents spend that never happened — the precise defect CostListPriceEquivalent
// exists to prevent.
func TestStandardModeCostProvenance_AnthropicOAuthIsNotMetered(t *testing.T) {
	saved := modelreg.GlobalModelsConfig
	t.Cleanup(func() { modelreg.GlobalModelsConfig = saved })

	modelreg.GlobalModelsConfig = &ModelsConfig{Models: map[string]ModelConfig{
		"claude-fable-5-1": {Provider: "anthropic", Pricing: Pricing{InputPer1K: 0.010, OutputPer1K: 0.050}},
		// A non-Anthropic priced row must be unaffected by Anthropic env state.
		"gpt5-6-luna": {Provider: "openai", Pricing: Pricing{InputPer1K: 0.0002, OutputPer1K: 0.0012}},
	}}

	tests := []struct {
		name              string
		apiKey, authToken string
		model             string
		want              string
	}{
		{
			name:      "anthropic on OAuth is list-price-equivalent",
			authToken: "oauth-tok", model: "claude-fable-5-1",
			want: string(executor.CostListPriceEquivalent),
		},
		{
			name:   "anthropic on an API key stays metered",
			apiKey: "sk-ant-x", model: "claude-fable-5-1",
			want: string(executor.CostMetered),
		},
		{
			name:   "both set: metered key wins, so the cost is metered",
			apiKey: "sk-ant-x", authToken: "oauth-tok", model: "claude-fable-5-1",
			want: string(executor.CostMetered),
		},
		{
			name:  "no credential anywhere stays metered rather than guessing",
			model: "claude-fable-5-1",
			want:  string(executor.CostMetered),
		},
		{
			name:      "an OAuth token does not relabel a non-Anthropic row",
			authToken: "oauth-tok", model: "gpt5-6-luna",
			want: string(executor.CostMetered),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Isolate HOME so the developer's real ~/.claude/.credentials.json
			// cannot decide the outcome — the resolver falls back to it, so
			// without this the "no credential" row passes or fails depending on
			// whose machine runs the suite.
			testutil.SetHomeDir(t, t.TempDir())
			t.Setenv("CLAUDE_CODE_OAUTH_TOKEN", "")
			t.Setenv("ANTHROPIC_API_KEY", tt.apiKey)
			t.Setenv("ANTHROPIC_AUTH_TOKEN", tt.authToken)
			if got := standardModeCostProvenance(tt.model); got != tt.want {
				t.Errorf("%s: got %q, want %q", tt.model, got, tt.want)
			}
		})
	}
}

// The credential FILE is the lane most runs will actually take (it is what the
// `claude` CLI reads), so it must drive the label too — otherwise the common
// case reports subscription usage as money.
func TestStandardModeCostProvenance_CredentialFileIsSubscription(t *testing.T) {
	saved := modelreg.GlobalModelsConfig
	t.Cleanup(func() { modelreg.GlobalModelsConfig = saved })
	modelreg.GlobalModelsConfig = &ModelsConfig{Models: map[string]ModelConfig{
		"claude-fable-5-1": {Provider: "anthropic", Pricing: Pricing{InputPer1K: 0.010, OutputPer1K: 0.050}},
	}}

	home := t.TempDir()
	testutil.SetHomeDir(t, home)
	t.Setenv("ANTHROPIC_API_KEY", "")
	t.Setenv("ANTHROPIC_AUTH_TOKEN", "")
	t.Setenv("CLAUDE_CODE_OAUTH_TOKEN", "")
	if err := os.MkdirAll(filepath.Join(home, ".claude"), 0o700); err != nil {
		t.Fatal(err)
	}
	body := fmt.Sprintf(`{"claudeAiOauth":{"accessToken":"tok","expiresAt":%d}}`,
		time.Now().Add(time.Hour).UnixMilli())
	if err := os.WriteFile(filepath.Join(home, ".claude", ".credentials.json"), []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}

	if got := standardModeCostProvenance("claude-fable-5-1"); got != string(executor.CostListPriceEquivalent) {
		t.Errorf("got %q, want list-price-equivalent — a run on the CLI's own credential spends quota, not money", got)
	}
}
