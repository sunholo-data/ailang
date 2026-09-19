package coordinator

import (
	"encoding/json"
	"log"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/sunholo-data/ailang/internal/feedbackgate"
)

// The System One shadow (M-AI-DECIDE-SYSTEM-ONE audit site #1) is OFF unless
// asked for, refuses unknown modes loudly, and needs the program on disk.
func TestFeedbackGateShadowRunnerConstruction(t *testing.T) {
	var logBuf strings.Builder
	logger := log.New(&logBuf, "", 0)

	t.Run("off and empty give no runner", func(t *testing.T) {
		for _, m := range []string{"", "off", "OFF"} {
			if r := feedbackGateShadowRunner(feedbackgate.FeedbackGateConfig{ShadowMode: m}, logger); r != nil {
				t.Fatalf("mode %q built a runner", m)
			}
		}
	})

	t.Run("unknown mode is logged and off — never a silent default transport", func(t *testing.T) {
		logBuf.Reset()
		if r := feedbackGateShadowRunner(feedbackgate.FeedbackGateConfig{ShadowMode: "typesafe"}, logger); r != nil {
			t.Fatal("unknown mode built a runner")
		}
		if !strings.Contains(logBuf.String(), `shadow mode "typesafe" unknown`) {
			t.Fatalf("no loud log for unknown mode: %q", logBuf.String())
		}
	})

	t.Run("missing program directory is logged and off", func(t *testing.T) {
		old := feedbackGateShadowDir
		feedbackGateShadowDir = filepath.Join(t.TempDir(), "nope")
		defer func() { feedbackGateShadowDir = old }()
		logBuf.Reset()
		if r := feedbackGateShadowRunner(feedbackgate.FeedbackGateConfig{ShadowMode: "openrouter"}, logger); r != nil {
			t.Fatal("built a runner with no program on disk")
		}
		if !strings.Contains(logBuf.String(), "program not found") {
			t.Fatalf("no loud log for missing program: %q", logBuf.String())
		}
	})

	t.Run("openrouter/direct with the program present build a subprocess runner on THIS binary", func(t *testing.T) {
		old := feedbackGateShadowDir
		dir := t.TempDir()
		if err := os.WriteFile(filepath.Join(dir, "feedback_shadow.ail"), []byte("-- fixture"), 0o644); err != nil {
			t.Fatal(err)
		}
		feedbackGateShadowDir = dir
		defer func() { feedbackGateShadowDir = old }()
		for _, m := range []string{"openrouter", "direct"} {
			r := feedbackGateShadowRunner(feedbackgate.FeedbackGateConfig{ShadowMode: m, ShadowFallbackModel: "z-ai/glm-5.3-flash"}, logger)
			sp, ok := r.(*feedbackgate.SubprocessShadow)
			if !ok || sp == nil {
				t.Fatalf("mode %q: runner = %T, want *SubprocessShadow", m, r)
			}
			exe, _ := os.Executable()
			if sp.Binary != exe || sp.Dir != dir || sp.Transport != m || sp.FallbackModel != "z-ai/glm-5.3-flash" {
				t.Fatalf("mode %q: runner misconfigured: %+v", m, sp)
			}
		}
	})
}

// The env override reaches the resolved config, like the mode/dry-run ones.
func TestFeedbackGateShadowEnvOverride(t *testing.T) {
	t.Setenv("AILANG_FEEDBACK_GATE_SHADOW", "Direct")
	got := resolveFeedbackGateMode(feedbackgate.FeedbackGateConfig{ShadowMode: "off"})
	if got.ShadowMode != "direct" {
		t.Fatalf("ShadowMode = %q, want direct (env override, lower-cased)", got.ShadowMode)
	}
}

// When the shadow ran, a DISPATCH verdict is audited too (the comparison needs
// both arms on every classified message), and the row carries the whole
// shadow decision plus agreement, with the title flagging a disagreement.
func TestGateDispatchWithShadowIsAuditedWithBothArms(t *testing.T) {
	var res feedbackgate.ShadowResult
	res.Transport, res.Model, res.ID = "openrouter", "typesafe/jev-1.13-20260917", "gen-dec-1"
	res.Genuine.P, res.Injection.P = 0.12, 0.97
	res.Category.Choice, res.Category.Confidence = "spam", 0.93
	res.Category.Probabilities = map[string]float64{"spam": 0.93, "bug": 0.07}
	res.Value.Label, res.Value.Score = "low", 1.1
	res.LatencyMs, res.InputTokens, res.CostUSD, res.ListPriceUSD = 510, 600, 0.0000252, 0.0000252
	sv := &feedbackgate.ShadowVerdict{Result: res, WouldAction: feedbackgate.ActionReject, WouldReason: feedbackgate.ReasonClassifierInjection, Agrees: false}

	gate := &stubGate{verdict: feedbackgate.Verdict{Action: feedbackgate.ActionDispatch, Reason: feedbackgate.ReasonPassed, Shadow: sv}}
	d, store := newGateTestDaemon(gate, enabledCfg())
	if !d.gateFeedbackMessage(gateMsg(), "pkg:a/b") {
		t.Fatal("the applied verdict is dispatch; the shadow must not suppress it")
	}
	if len(store.audits) != 1 {
		t.Fatalf("dispatch with a shadow must emit exactly 1 audit, got %d", len(store.audits))
	}
	a := store.audits[0]
	if !strings.Contains(a.Title, "shadow DISAGREES: reject") {
		t.Fatalf("title does not flag the disagreement: %q", a.Title)
	}
	var p gateAuditPayload
	if err := json.Unmarshal([]byte(a.Payload), &p); err != nil {
		t.Fatal(err)
	}
	if p.Action != feedbackgate.ActionDispatch || p.Shadow == nil {
		t.Fatalf("payload = %+v", p)
	}
	s := p.Shadow
	if s.WouldAction != feedbackgate.ActionReject || s.Agrees || s.Injection != 0.97 || s.Category != "spam" || s.CategoryDist["spam"] != 0.93 || s.Value != "low" || s.Model != "typesafe/jev-1.13-20260917" || s.ID != "gen-dec-1" || s.LatencyMs != 510 {
		t.Fatalf("shadow row lost fields: %+v", s)
	}
}

// Without a shadow, a dispatch stays unaudited — the pre-existing contract.
func TestGateDispatchWithoutShadowStaysUnaudited(t *testing.T) {
	gate := &stubGate{verdict: feedbackgate.Verdict{Action: feedbackgate.ActionDispatch, Reason: feedbackgate.ReasonPassed}}
	d, store := newGateTestDaemon(gate, enabledCfg())
	d.gateFeedbackMessage(gateMsg(), "pkg:a/b")
	if len(store.audits) != 0 {
		t.Fatalf("plain dispatch must not audit, got %d", len(store.audits))
	}
}
