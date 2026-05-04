package types

import (
	"encoding/json"
	"reflect"
	"strings"
	"testing"

	"github.com/sunholo-data/ailang/internal/ast"
)

// M-EFFECT-REFINEMENT Phase 1 — M2 tests for parameterised effects.
//
// These tests cover:
//   - DefaultModeFor lookup table (Phase 1: only Rand has a default)
//   - ElaborateEffectRow / ElaborateEffectRowWithBudgets default-mode desugar
//   - User params win over defaults
//   - Row.String / FormatEffectRow emit params alphabetically
//   - Row.Equals invariant on Params
//   - unifyRows invariant on Params (5 cases per design doc)
//   - JSON round-trip preserves Params; old-format JSON unmarshals to nil

func TestDefaultModeFor(t *testing.T) {
	cases := []struct {
		effect    string
		wantKey   string
		wantValue string
		wantOK    bool
	}{
		{"Rand", "mode", "os", true},
		{"AI", "mode", "fixed", true}, // M-AI-EFFECT-MODES (v0.15.0)
		// Other effects: no entry yet. Their port sprints (Phase 5 of parent doc) add rows.
		{"Clock", "", "", false},
		{"Net", "", "", false},
		{"FS", "", "", false},
		{"IO", "", "", false},
		{"Unknown", "", "", false},
	}
	for _, tc := range cases {
		t.Run(tc.effect, func(t *testing.T) {
			k, v, ok := DefaultModeFor(tc.effect)
			if k != tc.wantKey || v != tc.wantValue || ok != tc.wantOK {
				t.Errorf("DefaultModeFor(%q) = (%q, %q, %v), want (%q, %q, %v)",
					tc.effect, k, v, ok, tc.wantKey, tc.wantValue, tc.wantOK)
			}
		})
	}
}

func TestElaborateEffectRow_DefaultModeApplied(t *testing.T) {
	// Bare !{Rand} desugars to params["Rand"] = {"mode": "os"}.
	row, err := ElaborateEffectRow([]string{"Rand"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got := row.Params["Rand"]; got["mode"] != "os" || len(got) != 1 {
		t.Errorf("expected Params[Rand] = {mode: os}, got %v", got)
	}

	// Effects with no registered default → Params is nil (back-compat).
	row, err = ElaborateEffectRow([]string{"IO", "FS"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if row.Params != nil {
		t.Errorf("expected Params == nil for IO/FS (no defaults), got %v", row.Params)
	}

	// Mixed: Rand gets default, IO does not.
	row, err = ElaborateEffectRow([]string{"Rand", "IO"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got := row.Params["Rand"]; got["mode"] != "os" {
		t.Errorf("expected Params[Rand] = {mode: os}, got %v", got)
	}
	if _, has := row.Params["IO"]; has {
		t.Errorf("expected Params[IO] unset, got %v", row.Params["IO"])
	}
}

func TestElaborateEffectRowWithBudgets_UserParamsOverrideDefault(t *testing.T) {
	// User-supplied params override the registered default.
	annotations := []ast.EffectAnnotation{
		{Name: "Rand", Params: []ast.EffectParam{{Key: "mode", Value: "crypto"}}},
	}
	row, err := ElaborateEffectRowWithBudgets(annotations)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got := row.Params["Rand"]; got["mode"] != "crypto" || len(got) != 1 {
		t.Errorf("expected Params[Rand] = {mode: crypto}, got %v", got)
	}

	// Multiple params preserved.
	annotations = []ast.EffectAnnotation{
		{Name: "Rand", Params: []ast.EffectParam{
			{Key: "mode", Value: "os"},
			{Key: "scope", Value: "identity"},
		}},
	}
	row, err = ElaborateEffectRowWithBudgets(annotations)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	got := row.Params["Rand"]
	if got["mode"] != "os" || got["scope"] != "identity" || len(got) != 2 {
		t.Errorf("expected Params[Rand] = {mode: os, scope: identity}, got %v", got)
	}

	// Bare AST annotation (no Params) → default applied.
	annotations = []ast.EffectAnnotation{{Name: "Rand"}}
	row, err = ElaborateEffectRowWithBudgets(annotations)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got := row.Params["Rand"]; got["mode"] != "os" {
		t.Errorf("expected default Params[Rand] = {mode: os}, got %v", got)
	}

	// Effects without registered defaults stay bare.
	annotations = []ast.EffectAnnotation{{Name: "IO"}, {Name: "FS"}}
	row, err = ElaborateEffectRowWithBudgets(annotations)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if row.Params != nil {
		t.Errorf("expected Params == nil for IO/FS (no defaults), got %v", row.Params)
	}
}

func TestUnifyRows_InvariantOnParams(t *testing.T) {
	// Helper: build effect row from annotations.
	mkRow := func(t *testing.T, effs []ast.EffectAnnotation, tail string) *Row {
		t.Helper()
		row, err := ElaborateEffectRowWithBudgets(effs)
		if err != nil {
			t.Fatalf("ElaborateEffectRowWithBudgets failed: %v", err)
		}
		if tail != "" {
			row.Tail = &RowVar{Name: tail, Kind: EffectRow}
		}
		return row
	}

	annRandOS := []ast.EffectAnnotation{
		{Name: "Rand", Params: []ast.EffectParam{{Key: "mode", Value: "os"}}},
	}
	annRandSeeded := []ast.EffectAnnotation{
		{Name: "Rand", Params: []ast.EffectParam{{Key: "mode", Value: "seeded"}}},
	}
	annRandBare := []ast.EffectAnnotation{{Name: "Rand"}}
	annRandOSFS := []ast.EffectAnnotation{
		{Name: "Rand", Params: []ast.EffectParam{{Key: "mode", Value: "os"}}},
		{Name: "FS"},
	}
	annFSRandOS := []ast.EffectAnnotation{
		{Name: "FS"},
		{Name: "Rand", Params: []ast.EffectParam{{Key: "mode", Value: "os"}}},
	}

	cases := []struct {
		name    string
		row1    *Row
		row2    *Row
		wantErr bool
	}{
		{
			name: "same params unify",
			row1: mkRow(t, annRandOS, ""),
			row2: mkRow(t, annRandOS, ""),
		},
		{
			name: "default-desugar match: Rand[mode=os] unifies with bare Rand",
			row1: mkRow(t, annRandOS, ""),
			row2: mkRow(t, annRandBare, ""),
		},
		{
			name:    "different params fail to unify",
			row1:    mkRow(t, annRandOS, ""),
			row2:    mkRow(t, annRandSeeded, ""),
			wantErr: true,
		},
		{
			// Polymorphic tails: row1 = !{Rand[mode=os] | a}, row2 = !{Rand[mode=os], FS | b}.
			// Common labels (Rand) must have matching params; the FS label is captured
			// by row1's tail 'a', and row2's tail 'b' captures any extra labels in row1.
			name: "polymorphic tail preserved",
			row1: mkRow(t, annRandOS, "a"),
			row2: mkRow(t, annRandOSFS, "b"),
		},
		{
			name: "row swap: order does not matter",
			row1: mkRow(t, annRandOSFS, ""),
			row2: mkRow(t, annFSRandOS, ""),
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			u := NewUnifier()
			_, err := u.unifyRows(tc.row1, tc.row2, Substitution{})
			if tc.wantErr {
				if err == nil {
					t.Errorf("expected unification error, got nil")
				} else if !strings.Contains(err.Error(), "param mismatch") {
					t.Errorf("expected param mismatch error, got: %v", err)
				}
				return
			}
			if err != nil {
				t.Errorf("unexpected unification error: %v", err)
			}
		})
	}
}

func TestRow_String_ParamsAlphabetical(t *testing.T) {
	// Run multiple iterations to catch map-iteration-order non-determinism.
	row := &Row{
		Kind:   EffectRow,
		Labels: map[string]Type{"Rand": Unit()},
		Params: map[string]map[string]string{
			"Rand": {"scope": "identity", "mode": "crypto"},
		},
	}
	want := "{Rand[mode=crypto, scope=identity]}"
	for i := 0; i < 50; i++ {
		got := row.String()
		if got != want {
			t.Fatalf("iteration %d: expected %q, got %q", i, want, got)
		}
	}
}

func TestFormatEffectRow_ParamsAlphabetical(t *testing.T) {
	row := &Row{
		Kind:   EffectRow,
		Labels: map[string]Type{"Rand": Unit()},
		Params: map[string]map[string]string{
			"Rand": {"zebra": "z", "alpha": "a", "mode": "os"},
		},
	}
	want := "! {Rand[alpha=a, mode=os, zebra=z]}"
	for i := 0; i < 20; i++ {
		got := FormatEffectRow(row)
		if got != want {
			t.Fatalf("iteration %d: expected %q, got %q", i, want, got)
		}
	}
}

func TestRow_Equals_InvariantParams(t *testing.T) {
	mkRow := func(params map[string]map[string]string) *Row {
		return &Row{
			Kind:   EffectRow,
			Labels: map[string]Type{"Rand": Unit()},
			Params: params,
		}
	}

	// Same params → equal.
	a := mkRow(map[string]map[string]string{"Rand": {"mode": "os"}})
	b := mkRow(map[string]map[string]string{"Rand": {"mode": "os"}})
	if !a.Equals(b) {
		t.Errorf("rows with identical params should be equal")
	}

	// Different params → not equal.
	a = mkRow(map[string]map[string]string{"Rand": {"mode": "os"}})
	b = mkRow(map[string]map[string]string{"Rand": {"mode": "seeded"}})
	if a.Equals(b) {
		t.Errorf("rows with different params should NOT be equal")
	}

	// nil and empty Params maps are equivalent.
	a = mkRow(nil)
	b = mkRow(map[string]map[string]string{})
	if !a.Equals(b) {
		t.Errorf("nil and empty Params maps should be equivalent")
	}

	// nil Params and Params with empty inner map are equivalent.
	a = mkRow(nil)
	b = mkRow(map[string]map[string]string{"Rand": {}})
	if !a.Equals(b) {
		t.Errorf("nil and {Rand: {}} Params maps should be equivalent")
	}
}

func TestRow_JSON_RoundTrip(t *testing.T) {
	original := &Row{
		Kind:   EffectRow,
		Labels: map[string]Type{"Rand": Unit(), "FS": Unit()},
		Params: map[string]map[string]string{
			"Rand": {"mode": "os", "scope": "identity"},
		},
	}

	// Marshal.
	data, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("marshal failed: %v", err)
	}

	// Unmarshal via the type-dispatcher (Row uses tag "row").
	got, err := UnmarshalType(data)
	if err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}
	gotRow, ok := got.(*Row)
	if !ok {
		t.Fatalf("expected *Row, got %T", got)
	}

	if !reflect.DeepEqual(original.Params, gotRow.Params) {
		t.Errorf("Params round-trip mismatch:\n  want: %v\n  got:  %v",
			original.Params, gotRow.Params)
	}
}

func TestRow_JSON_OldFormat_NoParams(t *testing.T) {
	// Old iface JSON without "params" field must unmarshal to Params == nil.
	oldJSON := `{
		"tag": "row",
		"data": {
			"kind": {"tag": "row", "data": {"tag": "effect"}},
			"labels": {"IO": {"tag": "tcon", "data": {"Name": "()"}}}
		}
	}`
	got, err := UnmarshalType([]byte(oldJSON))
	if err != nil {
		t.Fatalf("unmarshal old-format JSON failed: %v", err)
	}
	gotRow, ok := got.(*Row)
	if !ok {
		t.Fatalf("expected *Row, got %T", got)
	}
	if gotRow.Params != nil {
		t.Errorf("expected Params == nil for old-format JSON, got %v", gotRow.Params)
	}
}

func TestRow_JSON_EmptyParams_OmitEmpty(t *testing.T) {
	// Marshalling a Row with Params == nil should produce JSON without a "params" key.
	row := &Row{
		Kind:   EffectRow,
		Labels: map[string]Type{"IO": Unit()},
		// Params is nil
	}
	data, err := json.Marshal(row)
	if err != nil {
		t.Fatalf("marshal failed: %v", err)
	}
	if strings.Contains(string(data), `"params"`) {
		t.Errorf("expected no \"params\" field in JSON, got: %s", data)
	}

	// Same when Params has only empty inner maps.
	row.Params = map[string]map[string]string{"IO": {}}
	data, err = json.Marshal(row)
	if err != nil {
		t.Fatalf("marshal failed: %v", err)
	}
	if strings.Contains(string(data), `"params"`) {
		t.Errorf("expected no \"params\" field for empty inner map, got: %s", data)
	}
}

func TestSubsumeEffectRows_InvariantOnParams(t *testing.T) {
	mkRow := func(params map[string]map[string]string) *Row {
		return &Row{
			Kind:   EffectRow,
			Labels: map[string]Type{"Rand": Unit()},
			Params: params,
		}
	}

	// Same params → subsumed.
	a := mkRow(map[string]map[string]string{"Rand": {"mode": "os"}})
	b := mkRow(map[string]map[string]string{"Rand": {"mode": "os"}})
	if !SubsumeEffectRows(a, b) {
		t.Errorf("Rand[mode=os] should subsume Rand[mode=os]")
	}

	// Different params → not subsumed (Phase 5 routeable→fixed scenario).
	a = mkRow(map[string]map[string]string{"Rand": {"mode": "os"}})
	b = mkRow(map[string]map[string]string{"Rand": {"mode": "seeded"}})
	if SubsumeEffectRows(a, b) {
		t.Errorf("Rand[mode=os] should NOT subsume Rand[mode=seeded]")
	}
}

func TestParamMapsEqual(t *testing.T) {
	cases := []struct {
		name string
		a, b map[string]string
		want bool
	}{
		{"both nil", nil, nil, true},
		{"both empty", map[string]string{}, map[string]string{}, true},
		{"nil vs empty", nil, map[string]string{}, true},
		{"same single", map[string]string{"k": "v"}, map[string]string{"k": "v"}, true},
		{"different value", map[string]string{"k": "v"}, map[string]string{"k": "w"}, false},
		{"different key", map[string]string{"k": "v"}, map[string]string{"j": "v"}, false},
		{"size mismatch", map[string]string{"k": "v"}, map[string]string{"k": "v", "j": "w"}, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := paramMapsEqual(tc.a, tc.b); got != tc.want {
				t.Errorf("paramMapsEqual(%v, %v) = %v, want %v", tc.a, tc.b, got, tc.want)
			}
		})
	}
}

// =============================================================================
// M-AI-EFFECT-MODES (v0.15.0): tests for the AI row in defaultEffectModes
// =============================================================================

// TestElaborateEffectRow_AIDefault checks that bare !{AI} desugars to
// !{AI[mode=fixed]} via the M-AI-EFFECT-MODES default-mode entry.
func TestElaborateEffectRow_AIDefault(t *testing.T) {
	// Bare !{AI} → Params["AI"] = {"mode": "fixed"}
	row, err := ElaborateEffectRow([]string{"AI"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got := row.Params["AI"]; got["mode"] != "fixed" || len(got) != 1 {
		t.Errorf("expected Params[AI] = {mode: fixed}, got %v", got)
	}

	// !{AI, IO} — AI gets default, IO does not (no entry for IO).
	row, err = ElaborateEffectRow([]string{"AI", "IO"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got := row.Params["AI"]; got["mode"] != "fixed" {
		t.Errorf("expected Params[AI] = {mode: fixed}, got %v", got)
	}
	if _, has := row.Params["IO"]; has {
		t.Errorf("expected Params[IO] unset (no default registered), got %v", row.Params["IO"])
	}

	// !{AI, Rand} — both get their defaults (AI=fixed, Rand=os).
	row, err = ElaborateEffectRow([]string{"AI", "Rand"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got := row.Params["AI"]; got["mode"] != "fixed" {
		t.Errorf("expected Params[AI] = {mode: fixed}, got %v", got)
	}
	if got := row.Params["Rand"]; got["mode"] != "os" {
		t.Errorf("expected Params[Rand] = {mode: os}, got %v", got)
	}
}

// TestElaborateEffectRowWithBudgets_AIUserParamsOverrideDefault checks that
// user-supplied AI params override the registered default.
func TestElaborateEffectRowWithBudgets_AIUserParamsOverrideDefault(t *testing.T) {
	// !{AI[mode=routeable]} preserves user-supplied param (does NOT desugar to fixed).
	annotations := []ast.EffectAnnotation{
		{Name: "AI", Params: []ast.EffectParam{{Key: "mode", Value: "routeable"}}},
	}
	row, err := ElaborateEffectRowWithBudgets(annotations)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got := row.Params["AI"]; got["mode"] != "routeable" || len(got) != 1 {
		t.Errorf("expected Params[AI] = {mode: routeable}, got %v", got)
	}

	// !{AI[mode=replay-only]} also preserves user param (parser-accepted; runtime stub).
	annotations = []ast.EffectAnnotation{
		{Name: "AI", Params: []ast.EffectParam{{Key: "mode", Value: "replay-only"}}},
	}
	row, err = ElaborateEffectRowWithBudgets(annotations)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got := row.Params["AI"]; got["mode"] != "replay-only" {
		t.Errorf("expected Params[AI] = {mode: replay-only}, got %v", got)
	}

	// !{AI[mode=routeable, scope=byok]} preserves both params.
	annotations = []ast.EffectAnnotation{
		{Name: "AI", Params: []ast.EffectParam{
			{Key: "mode", Value: "routeable"},
			{Key: "scope", Value: "byok"},
		}},
	}
	row, err = ElaborateEffectRowWithBudgets(annotations)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got := row.Params["AI"]; got["mode"] != "routeable" || got["scope"] != "byok" || len(got) != 2 {
		t.Errorf("expected Params[AI] = {mode: routeable, scope: byok}, got %v", got)
	}

	// Bare !{AI} (no Params on the annotation) gets the default applied.
	annotations = []ast.EffectAnnotation{{Name: "AI"}}
	row, err = ElaborateEffectRowWithBudgets(annotations)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got := row.Params["AI"]; got["mode"] != "fixed" {
		t.Errorf("expected bare AI to default to {mode: fixed}, got %v", got)
	}
}

// TestEffectModeFor_AI verifies the read-path used by the CLI safety-gate
// (M-AI-EFFECT-MODES M2): EffectModeFor must return the declared mode for an
// effect in a row, applying the default-mode desugar so bare !{AI} reads as
// "fixed". Returns ("", false) for nil rows, missing effects, or effects with
// no mode param (and no default).
func TestEffectModeFor_AI(t *testing.T) {
	t.Run("nil row returns no mode", func(t *testing.T) {
		got, ok := EffectModeFor(nil, "AI")
		if got != "" || ok {
			t.Errorf("EffectModeFor(nil, \"AI\") = (%q, %v), want (\"\", false)", got, ok)
		}
	})

	t.Run("AI not in row returns no mode", func(t *testing.T) {
		row, err := ElaborateEffectRow([]string{"IO", "FS"})
		if err != nil {
			t.Fatalf("ElaborateEffectRow failed: %v", err)
		}
		got, ok := EffectModeFor(row, "AI")
		if got != "" || ok {
			t.Errorf("EffectModeFor(IO/FS row, \"AI\") = (%q, %v), want (\"\", false)", got, ok)
		}
	})

	t.Run("bare AI desugars to fixed", func(t *testing.T) {
		row, err := ElaborateEffectRow([]string{"AI"})
		if err != nil {
			t.Fatalf("ElaborateEffectRow failed: %v", err)
		}
		got, ok := EffectModeFor(row, "AI")
		if got != "fixed" || !ok {
			t.Errorf("EffectModeFor(bare AI row, \"AI\") = (%q, %v), want (\"fixed\", true)", got, ok)
		}
	})

	t.Run("explicit mode=routeable returns routeable", func(t *testing.T) {
		annotations := []ast.EffectAnnotation{
			{Name: "AI", Params: []ast.EffectParam{{Key: "mode", Value: "routeable"}}},
		}
		row, err := ElaborateEffectRowWithBudgets(annotations)
		if err != nil {
			t.Fatalf("ElaborateEffectRowWithBudgets failed: %v", err)
		}
		got, ok := EffectModeFor(row, "AI")
		if got != "routeable" || !ok {
			t.Errorf("EffectModeFor(AI[mode=routeable], \"AI\") = (%q, %v), want (\"routeable\", true)", got, ok)
		}
	})

	t.Run("explicit mode=replay-only returns replay-only", func(t *testing.T) {
		annotations := []ast.EffectAnnotation{
			{Name: "AI", Params: []ast.EffectParam{{Key: "mode", Value: "replay-only"}}},
		}
		row, err := ElaborateEffectRowWithBudgets(annotations)
		if err != nil {
			t.Fatalf("ElaborateEffectRowWithBudgets failed: %v", err)
		}
		got, ok := EffectModeFor(row, "AI")
		if got != "replay-only" || !ok {
			t.Errorf("EffectModeFor(AI[mode=replay-only], \"AI\") = (%q, %v), want (\"replay-only\", true)", got, ok)
		}
	})

	t.Run("effect with no mode param returns false", func(t *testing.T) {
		// Manually construct a row with AI label but no mode param entry —
		// simulates a row hand-built without going through ElaborateEffectRow.
		row := &Row{
			Kind:   EffectRow,
			Labels: map[string]Type{"IO": Unit()},
			Params: nil, // IO has no default mode, so no entry
		}
		got, ok := EffectModeFor(row, "IO")
		if got != "" || ok {
			t.Errorf("EffectModeFor(IO row, \"IO\") = (%q, %v), want (\"\", false)", got, ok)
		}
	})
}

// TestUnifyRows_AIModes covers the 5 unification cases for AI parameters,
// matching the matrix in the design doc Examples section.
func TestUnifyRows_AIModes(t *testing.T) {
	mkRow := func(t *testing.T, effs []ast.EffectAnnotation, tail string) *Row {
		t.Helper()
		row, err := ElaborateEffectRowWithBudgets(effs)
		if err != nil {
			t.Fatalf("ElaborateEffectRowWithBudgets failed: %v", err)
		}
		if tail != "" {
			row.Tail = &RowVar{Name: tail, Kind: EffectRow}
		}
		return row
	}

	annAIFixed := []ast.EffectAnnotation{
		{Name: "AI", Params: []ast.EffectParam{{Key: "mode", Value: "fixed"}}},
	}
	annAIRouteable := []ast.EffectAnnotation{
		{Name: "AI", Params: []ast.EffectParam{{Key: "mode", Value: "routeable"}}},
	}
	annAIBare := []ast.EffectAnnotation{{Name: "AI"}}
	annAIFixedFS := []ast.EffectAnnotation{
		{Name: "AI", Params: []ast.EffectParam{{Key: "mode", Value: "fixed"}}},
		{Name: "FS"},
	}
	annFSAIFixed := []ast.EffectAnnotation{
		{Name: "FS"},
		{Name: "AI", Params: []ast.EffectParam{{Key: "mode", Value: "fixed"}}},
	}

	cases := []struct {
		name    string
		row1    *Row
		row2    *Row
		wantErr bool
	}{
		{"same mode (fixed)", mkRow(t, annAIFixed, ""), mkRow(t, annAIFixed, ""), false},
		{"default-desugar match", mkRow(t, annAIFixed, ""), mkRow(t, annAIBare, ""), false},
		{"different modes (fixed vs routeable)", mkRow(t, annAIFixed, ""), mkRow(t, annAIRouteable, ""), true},
		{"polymorphic tail", mkRow(t, annAIFixed, "a"), mkRow(t, annAIFixedFS, "b"), false},
		{"row swap", mkRow(t, annAIFixedFS, ""), mkRow(t, annFSAIFixed, ""), false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			u := NewUnifier()
			_, err := u.unifyRows(tc.row1, tc.row2, Substitution{})
			if (err != nil) != tc.wantErr {
				t.Errorf("unifyRows: got err=%v, wantErr=%v", err, tc.wantErr)
			}
		})
	}
}
