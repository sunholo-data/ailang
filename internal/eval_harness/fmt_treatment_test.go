package eval_harness

import "testing"

// TestResolveFmtArm is the fix for a mislabelled experiment.
//
// The motoko-extension fmt A/B banked fmt_hook_state:"off" on BOTH arms,
// because that field was resolved from the Claude `-fmt-hook` flag — which a
// motoko-profile arm never sets. Both arms carried the same label, so the
// comparison could not distinguish them at all, and M6's verification gate
// (fmt_hook_events > 0 on the ON arm) had nothing to assert against.
//
// The arm must be derived from what was actually LOADED, not from a flag that
// belongs to a different mechanism.
func TestResolveFmtArm(t *testing.T) {
	tests := []struct {
		name       string
		flagMode   FmtHookMode
		extensions string // resolved_extensions from the subject's step-0 broadcast
		want       string
	}{
		{
			name: "claude hook on, no extensions", flagMode: FmtHookModeOn, extensions: "", want: "on",
		},
		{
			name: "claude hook off, no extensions", flagMode: FmtHookModeOff, extensions: "", want: "off",
		},
		{
			// THE BUG: motoko ollama_fmt profile loads the fmt extension, but the
			// Claude flag is off. This used to report "off" — mislabelling the
			// treatment arm as the control.
			name: "motoko fmt extension loaded", flagMode: FmtHookModeOff,
			extensions: "compaction_ai,context_mode,fmt", want: "on",
		},
		{
			// The REAL on-the-wire format: the registry stamps "<name>#<index>".
			// Matching the bare name against "fmt#2" failed, so a live ON arm was
			// labelled off and then falsely reported as a contaminated control.
			name: "registry id format with index suffix", flagMode: FmtHookModeOff,
			extensions: "compaction_ai#0,context_mode#1,fmt#2", want: "on",
		},
		{
			name: "indexed ids without fmt", flagMode: FmtHookModeOff,
			extensions: "compaction_ai#0,context_mode#1", want: "off",
		},
		{
			name: "indexed lookalike does not count", flagMode: FmtHookModeOff,
			extensions: "fmtx#0,not_fmt_really#1", want: "off",
		},
		{
			name: "motoko without fmt extension", flagMode: FmtHookModeOff,
			extensions: "compaction_ai,context_mode", want: "off",
		},
		{
			// Substring safety: an extension merely CONTAINING "fmt" is not the
			// fmt extension.
			name: "lookalike extension name does not count", flagMode: FmtHookModeOff,
			extensions: "compaction_ai,fmtx,not_fmt_really", want: "off",
		},
		{
			name: "both mechanisms on", flagMode: FmtHookModeOn,
			extensions: "fmt", want: "on",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ResolveFmtArm(tt.flagMode, tt.extensions); got != tt.want {
				t.Errorf("ResolveFmtArm(%q, %q) = %q, want %q", tt.flagMode, tt.extensions, got, tt.want)
			}
		})
	}
}

// TestAssertFmtTreatmentIntegrity enforces M6's verification gate and M5's void
// clause: an ON arm that cannot show the treatment fired is not a null result,
// it is not a result at all.
func TestAssertFmtTreatmentIntegrity(t *testing.T) {
	oneEvent := []FmtHookEvent{{Status: "formatted"}}
	allErrors := []FmtHookEvent{{Status: "error"}, {Status: "error"}, {Status: "error"}}
	mixed := []FmtHookEvent{{Status: "error"}, {Status: "formatted"}, {Status: "error"}}

	tests := []struct {
		name       string
		arm        string
		events     []FmtHookEvent
		wantValid  bool
		wantReason string
	}{
		{name: "on arm with events is proven", arm: "on", events: oneEvent, wantValid: true},
		{
			// The exact state the fmt A/B is in today.
			name: "on arm with NO events is void", arm: "on", events: nil,
			wantValid: false, wantReason: ReasonTreatmentUnproven,
		},
		{name: "off arm with no events is correct", arm: "off", events: nil, wantValid: true},
		{
			// Firing is not delivering. Events exist, but nothing was ever
			// formatted — the treatment did not apply.
			name: "on arm where every invocation errored is void", arm: "on", events: allErrors,
			wantValid: false, wantReason: ReasonTreatmentUnproven,
		},
		{
			// The real shape observed on run_length_encode: some errors, but
			// formats did happen, so the treatment demonstrably applied.
			name: "on arm with mixed errors and formats is proven", arm: "on", events: mixed,
			wantValid: true,
		},
		{
			// Contamination: the control arm was supposed to have no treatment.
			name: "off arm WITH events is contaminated", arm: "off", events: oneEvent,
			wantValid: false, wantReason: ReasonTreatmentUnproven,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v := AssertFmtTreatmentIntegrity(tt.arm, tt.events)
			if tt.wantValid {
				if v != nil && !v.Valid {
					t.Fatalf("expected no violation, got invalid(%s): %s", v.Reason, v.Detail)
				}
				return
			}
			if v == nil || v.Valid {
				t.Fatalf("expected a violation, got %+v", v)
			}
			if v.Reason != tt.wantReason {
				t.Errorf("reason = %q, want %q", v.Reason, tt.wantReason)
			}
			if v.Detail == "" {
				t.Error("a void result must explain what was expected vs observed")
			}
		})
	}
}
