package main

import (
	"testing"

	"github.com/sunholo-data/ailang/internal/eval_harness"
)

// The incident this guards, measured on the prod plane 2026-09-17 23:55:09:
//
//	{"models":["--help"], "total_jobs":180, ...}
//	"Eval Suite partial: 0/180 passed (0.0%)"   after 0.053 seconds
//
// `--models --help` binds the flag as the flag's own VALUE (Go's flag package
// takes the next argument unconditionally), so help never printed, "--help"
// became a model, and 180 trials failed instantly. Nothing downstream objected:
// the 0% was banked and published like a measurement.
func TestUnknownEvalModels(t *testing.T) {
	cfg, err := eval_harness.LoadModelsConfig("../../internal/modelreg/models.yml")
	if err != nil {
		t.Skipf("models.yml not available: %v", err)
	}
	if len(cfg.DevModels) == 0 {
		t.Skip("dev_models empty in models.yml; nothing known to contrast against")
	}
	known := cfg.DevModels[0]

	// Control: the instrument must see a positive before an empty result means
	// anything. A registry key that IS present has to come back clean, or
	// "no unknowns" below would pass even if the lookup were broken.
	if got := unknownEvalModels([]string{known}, cfg); len(got) != 0 {
		t.Fatalf("control failed: known model %q reported unknown (%v) — the lookup is broken, "+
			"so every other case in this test is meaningless", known, got)
	}

	cases := []struct {
		name  string
		input []string
		want  []string
	}{
		{"the incident", []string{"--help"}, []string{"--help"}},
		{"short flag form", []string{"-agent"}, []string{"-agent"}},
		{"typo in a real name", []string{known + "-typo"}, []string{known + "-typo"}},
		{"known model passes", []string{known}, nil},
		{"empty list passes", nil, nil},
		// The mixed case is the one that matters operationally: a list where
		// most entries resolve must still refuse, or a single bad entry in a
		// rotation silently contributes a column of zeros to a comparison.
		{"one bad entry among good", []string{known, "--help"}, []string{"--help"}},
		// Order is preserved so the error names them as the operator typed them.
		{"two bad entries keep order", []string{"zzz", known, "aaa"}, []string{"zzz", "aaa"}},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := unknownEvalModels(tc.input, cfg)
			if len(got) != len(tc.want) {
				t.Fatalf("unknownEvalModels(%v) = %v, want %v", tc.input, got, tc.want)
			}
			for i := range got {
				if got[i] != tc.want[i] {
					t.Errorf("unknownEvalModels(%v)[%d] = %q, want %q", tc.input, i, got[i], tc.want[i])
				}
			}
		})
	}
}

// A nil registry must NOT refuse. resolveEvalModelList falls back to hardcoded
// model lists when models.yml fails to load; validating those against a nil
// config would report every name unknown and refuse every run — turning a
// degraded-but-working path into a hard outage.
func TestUnknownEvalModels_NilConfigRefusesNothing(t *testing.T) {
	if got := unknownEvalModels([]string{"--help", "anything", "at-all"}, nil); got != nil {
		t.Errorf("unknownEvalModels(..., nil) = %v, want nil — a missing registry must not "+
			"refuse the hardcoded fallback lists", got)
	}
}
