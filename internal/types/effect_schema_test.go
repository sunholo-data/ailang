package types

import (
	"strings"
	"testing"

	"github.com/sunholo-data/ailang/internal/ast"
)

// M-EFFECT-MODE-VALIDATION M2 — tests for the closed effectSchema enforcement in
// ElaborateEffectRowWithBudgets:
//   - full legal matrix elaborates clean
//   - the three error shapes: EFF_UNKNOWN_MODE / EFF_UNKNOWN_PARAM_KEY /
//     EFF_PARAMS_NOT_SUPPORTED
//   - stringSliceToEffectRow bridge carries nil params → validation is a no-op
//
// (The schema/defaults consistency guard is M1; see effect_schema_consistency_test.go.)

// TestEffectSchemaLegalMatrix verifies every registered (effect, key, value)
// triple elaborates clean, plus the bare forms of the schema-carrying effects.
func TestEffectSchemaLegalMatrix(t *testing.T) {
	legal := []struct {
		name string
		anns []ast.EffectAnnotation
	}{
		{"Rand[mode=os]", ann("Rand", "mode", "os")},
		{"Rand[mode=seeded]", ann("Rand", "mode", "seeded")},
		{"Rand[mode=crypto]", ann("Rand", "mode", "crypto")},
		{"AI[mode=fixed]", ann("AI", "mode", "fixed")},
		{"AI[mode=routeable]", ann("AI", "mode", "routeable")},
		{"AI[mode=replay-only]", ann("AI", "mode", "replay-only")},
		{"AI[scope=byok]", ann("AI", "scope", "byok")},
		{"AI[mode=routeable, scope=byok]", []ast.EffectAnnotation{
			{Name: "AI", Params: []ast.EffectParam{
				{Key: "mode", Value: "routeable"},
				{Key: "scope", Value: "byok"},
			}},
		}},
		{"bare Rand", []ast.EffectAnnotation{{Name: "Rand"}}},
		{"bare AI", []ast.EffectAnnotation{{Name: "AI"}}},
	}
	for _, tc := range legal {
		t.Run(tc.name, func(t *testing.T) {
			if _, err := ElaborateEffectRowWithBudgets(tc.anns); err != nil {
				t.Errorf("%s should elaborate clean, got error: %v", tc.name, err)
			}
		})
	}
}

// TestEffectSchemaErrorShapes verifies the three structured, fix-carrying errors.
// Each assertion checks the verbatim EFF_* code AND a stable legal-list / fix
// substring appear in the returned error text.
func TestEffectSchemaErrorShapes(t *testing.T) {
	cases := []struct {
		name     string
		anns     []ast.EffectAnnotation
		wantCode string
		wantSubs []string // additional substrings that must appear (legal list / doc name)
	}{
		{
			name:     "unknown mode value",
			anns:     ann("Rand", "mode", "banana"),
			wantCode: "EFF_UNKNOWN_MODE",
			wantSubs: []string{"os", "seeded", "crypto", "Fix:"},
		},
		{
			name:     "unknown param key",
			anns:     ann("Rand", "flavor", "hot"),
			wantCode: "EFF_UNKNOWN_PARAM_KEY",
			wantSubs: []string{"mode", "Fix:"},
		},
		{
			name:     "params on schema-less effect",
			anns:     ann("Clock", "mode", "pinned"),
			wantCode: "EFF_PARAMS_NOT_SUPPORTED",
			wantSubs: []string{"m-effect-clock-net-fs-modes", "Fix:"},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := ElaborateEffectRowWithBudgets(tc.anns)
			if err == nil {
				t.Fatalf("%s: expected an error, got nil", tc.name)
			}
			msg := err.Error()
			if !strings.Contains(msg, tc.wantCode) {
				t.Errorf("%s: want code %q in error, got:\n%s", tc.name, tc.wantCode, msg)
			}
			for _, sub := range tc.wantSubs {
				if !strings.Contains(msg, sub) {
					t.Errorf("%s: want substring %q in error, got:\n%s", tc.name, sub, msg)
				}
			}
		})
	}
}

// TestStringSliceBridgeNilParamsNoValidation is the D1 bridge regression: the
// stringSliceToEffectRow path (pipeline/validate_effects.go) builds rows from
// effect NAME strings only — Params is always nil. This proves validation is a
// no-op on that path, so the iface-cache / back-compat round-trip is untouched.
// (validateEffectParams(_, nil) must return nil for every effect, including
// schema-less ones.)
func TestStringSliceBridgeNilParamsNoValidation(t *testing.T) {
	for _, effect := range []string{"Rand", "AI", "Clock", "IO", "FS", "Net"} {
		if err := validateEffectParams(effect, nil); err != nil {
			t.Errorf("validateEffectParams(%q, nil) should be a no-op, got: %v", effect, err)
		}
		if err := validateEffectParams(effect, map[string]string{}); err != nil {
			t.Errorf("validateEffectParams(%q, empty) should be a no-op, got: %v", effect, err)
		}
	}
	// And the elaborator itself: a bare effect list (as the bridge produces via
	// ElaborateEffectRow) never triggers a param-validation error.
	if _, err := ElaborateEffectRow([]string{"Rand", "AI", "Clock", "IO"}); err != nil {
		t.Errorf("bare effect elaboration should not trigger validation, got: %v", err)
	}
}

// ann is a one-param annotation-list constructor for the tables above.
func ann(effect, key, value string) []ast.EffectAnnotation {
	return []ast.EffectAnnotation{
		{Name: effect, Params: []ast.EffectParam{{Key: key, Value: value}}},
	}
}
