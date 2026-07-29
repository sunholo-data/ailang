package eval_harness

import (
	"strings"
	"testing"
)

// TestAssertResolvedProfile is M4's core: a row must measure what it claims to
// measure. All ten cloud motoko entries spent weeks silently defaulting to the
// `dogfood` profile — running without ailang_docs/microrag and with a verify
// gate that cannot work in a benchmark workspace — while models.yml advertised
// "DP7 verifier + microRAG context". Nothing detected it because the claim was
// never compared to reality.
func TestAssertResolvedProfile(t *testing.T) {
	tests := []struct {
		name       string
		claimed    string
		resolved   string
		wantValid  bool
		wantReason string
	}{
		{
			name: "match", claimed: "cloud", resolved: "cloud", wantValid: true,
		},
		{
			// The exact production bug.
			name: "claimed cloud but ran dogfood", claimed: "cloud", resolved: "dogfood",
			wantValid: false, wantReason: ReasonConfigMismatch,
		},
		{
			// Executors that broadcast nothing must be unaffected — M4 is
			// opt-in per executor, exactly like the canary.
			name:    "no resolved value: cannot assert, so do not fail",
			claimed: "cloud", resolved: "", wantValid: true,
		},
		{
			// models.yml sets no motoko_profile: nothing was claimed, so there
			// is nothing to contradict.
			name: "no claim: nothing to contradict", claimed: "", resolved: "ollama", wantValid: true,
		},
		{
			name: "neither present", claimed: "", resolved: "", wantValid: true,
		},
		{
			// models.yml omitting motoko_profile means the executor default
			// ("dogfood"), so an explicit "dogfood" resolution is consistent.
			name:    "empty claim resolving to the default is fine",
			claimed: "", resolved: "dogfood", wantValid: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v := AssertResolvedProfile(tt.claimed, tt.resolved)

			if tt.wantValid {
				if v != nil && !v.Valid {
					t.Fatalf("expected no mismatch, got invalid(%s)", v.Reason)
				}
				return
			}
			if v == nil || v.Valid {
				t.Fatalf("expected a mismatch to be flagged, got %+v", v)
			}
			if v.Reason != tt.wantReason {
				t.Errorf("reason = %q, want %q", v.Reason, tt.wantReason)
			}
		})
	}
}

// TestAssertResolvedProfile_MessageNamesBothSides: an operator reading the
// banked row must be able to see WHAT was claimed and WHAT ran, without
// re-deriving it from two other files.
func TestAssertResolvedProfile_MessageNamesBothSides(t *testing.T) {
	v := AssertResolvedProfile("cloud", "dogfood")
	if v == nil {
		t.Fatal("expected a mismatch")
	}
	if got := v.Detail; got == "" {
		t.Fatal("mismatch must carry a detail naming both sides")
	}
	for _, want := range []string{"cloud", "dogfood"} {
		if !strings.Contains(v.Detail, want) {
			t.Errorf("detail %q must name %q", v.Detail, want)
		}
	}
}
