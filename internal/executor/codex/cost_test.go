package codex

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/sunholo-data/ailang/internal/executor"
	"github.com/sunholo-data/ailang/internal/modelreg"
	"github.com/sunholo-data/ailang/internal/testutil"
)

// TestAuthLane_ReadsTheFile: the cost record depends on this. An api-key
// auth.json misreported as subscription would under-count spend, and a wrong
// "billed" is never the fallback.
func TestAuthLane_ReadsTheFile(t *testing.T) {
	for _, c := range []struct {
		name, file string
		want       executor.AuthLane
	}{
		{"chatgpt", `{"auth_mode":"chatgpt","tokens":{"refresh_token":"r"}}`, executor.AuthLaneSubscription},
		{"apikey", `{"auth_mode":"apikey","OPENAI_API_KEY":"sk-x"}`, executor.AuthLaneBilled},
		{"legacy key, no discriminator", `{"OPENAI_API_KEY":"sk-x"}`, executor.AuthLaneBilled},
		{"unparseable", `{`, executor.AuthLaneUnknown},
		{"absent", "", executor.AuthLaneUnknown},
	} {
		t.Run(c.name, func(t *testing.T) {
			home := t.TempDir()
			testutil.SetHomeDir(t, home)
			t.Setenv("CODEX_HOME", "")
			if c.file != "" {
				if err := os.MkdirAll(filepath.Join(home, ".codex"), 0o700); err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(authPath(home), []byte(c.file), 0o600); err != nil {
					t.Fatal(err)
				}
			}
			if got := (&CodexExecutor{}).authLane(); got != c.want {
				t.Errorf("authLane = %v, want %v", got, c.want)
			}
		})
	}
}

// TestCodexCostModel: the fallback rate card is the registry row of the model
// the executor is configured with, not gpt-5-codex's table applied to every
// model (M-V1-SIMPLIFY-S3 M2).
func TestCodexCostModel(t *testing.T) {
	cfg := testConfig()
	cfg.CodexModel = "gpt-5.2-codex" // an api_name; resolves to gpt5-2-codex
	exec, _ := New(cfg)
	cm := exec.CostModel()

	if cm == nil || cm.Unpriced {
		t.Fatalf("a registry api_name must price; got %+v", cm)
	}
	if cm.ProviderName != "openai" {
		t.Errorf("expected provider 'openai', got %q", cm.ProviderName)
	}
	if err := modelreg.InitModelsConfig(); err != nil {
		t.Fatal(err)
	}
	row := modelreg.GlobalModelsConfig.Models["gpt5-2-codex"].Pricing
	if cm.InputTokenCost != row.InputPer1K || cm.OutputTokenCost != row.OutputPer1K {
		t.Errorf("rate card %+v does not match the registry row %+v", cm, row)
	}

	// 1000 in + 500 out at the row's rates, through the one formula.
	got := cm.CalculateCost(executor.TokenUsage{InputTokens: 1000, OutputTokens: 500})
	want := row.Cost(1000, 500, 0, 0)
	if diff := got - want; diff > 1e-9 || diff < -1e-9 {
		t.Errorf("expected cost %v, got %v", want, got)
	}
}

// A model the registry does not know ("gpt-5-codex" is not a row) yields an
// explicit Unpriced card rather than silently billing at another model's rates.
func TestCodexCostModel_UnknownModelIsUnpriced(t *testing.T) {
	exec, _ := New(testConfig()) // CodexModel = "gpt-5-codex"
	cm := exec.CostModel()
	if cm == nil || !cm.Unpriced {
		t.Fatalf("expected an Unpriced card for gpt-5-codex, got %+v", cm)
	}
	if cm.CalculateCost(executor.TokenUsage{InputTokens: 1_000_000}) != 0 {
		t.Error("an unpriced card must bill nothing")
	}
}
