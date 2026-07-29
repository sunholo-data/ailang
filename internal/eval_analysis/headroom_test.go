package eval_analysis

import (
	"strings"
	"testing"
)

// TestHeadroom_HaikuFmtConfigurationTriggers is the concrete case this rule
// exists to prevent repeating. The fmt A/B was run against haiku, where both
// arms sat at ~96%: 30/30 vs 29/30 and 42/45 vs 43/45. At that ceiling the
// experiment was structurally incapable of showing the effect it was built to
// measure, and it consumed rig time to produce nothing.
func TestHeadroom_HaikuFmtConfigurationTriggers(t *testing.T) {
	h := CheckHeadroom(29, 30, DefaultHeadroomCeiling) // the real OFF arm

	if !h.Warn {
		t.Fatal("a control arm at 96.7% must warn — a small effect cannot be resolved there")
	}
	if !strings.Contains(h.Message, "96") {
		t.Errorf("message should state the observed rate, got: %s", h.Message)
	}
	// The operator needs to know WHY, not just that something is off.
	for _, want := range []string{"ceiling", "headroom"} {
		if !strings.Contains(strings.ToLower(h.Message), want) {
			t.Errorf("message should explain the ceiling problem (missing %q): %s", want, h.Message)
		}
	}
}

func TestCheckHeadroom(t *testing.T) {
	tests := []struct {
		name      string
		pass, tot int
		wantWarn  bool
	}{
		{name: "plenty of headroom", pass: 54, tot: 84, wantWarn: false}, // 64.3% — the real microRAG OFF arm
		{name: "just under the ceiling", pass: 89, tot: 100, wantWarn: false},
		{name: "exactly at the ceiling", pass: 90, tot: 100, wantWarn: true},
		{name: "above the ceiling", pass: 96, tot: 100, wantWarn: true},
		{name: "perfect score", pass: 20, tot: 20, wantWarn: true},
		{
			// No data is not a reason to warn — that would fire on every new
			// experiment and be trained away as noise.
			name: "no observations", pass: 0, tot: 0, wantWarn: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := CheckHeadroom(tt.pass, tt.tot, DefaultHeadroomCeiling)
			if h.Warn != tt.wantWarn {
				t.Errorf("Warn = %v, want %v (%d/%d)", h.Warn, tt.wantWarn, tt.pass, tt.tot)
			}
			if !h.Warn && h.Message != "" {
				t.Errorf("no warning should carry no message, got: %s", h.Message)
			}
		})
	}
}

// TestCheckHeadroom_NeverBlocks: the rule is advisory. A ceiling is a reason to
// doubt a null result, not a reason to refuse to run — sometimes you genuinely
// want the regression-guard signal from a saturated arm.
func TestCheckHeadroom_NeverBlocks(t *testing.T) {
	h := CheckHeadroom(100, 100, DefaultHeadroomCeiling)
	if !h.Warn {
		t.Fatal("100% must warn")
	}
	// The type carries no error and no blocking signal — asserted structurally
	// by there being nothing to assert. If a Fatal/Block field is ever added,
	// this test should be revisited deliberately.
	if h.Rate < 0.999 {
		t.Errorf("Rate = %v, want 1.0", h.Rate)
	}
}

// TestPairArms_SurfacesHeadroom wires the rule into the comparison so it is
// seen at the moment someone reads a result, not only at setup.
func TestPairArms_SurfacesHeadroom(t *testing.T) {
	// Control (off) arm at 100%: 4/4.
	on := []*BenchmarkResult{pass("a", 1), pass("b", 1), pass("c", 1), pass("d", 1)}
	off := []*BenchmarkResult{pass("a", 1), pass("b", 1), pass("c", 1), pass("d", 1)}

	p := PairArms(on, off)
	if !p.Headroom.Warn {
		t.Error("a saturated control arm must be surfaced on the paired result")
	}
}
