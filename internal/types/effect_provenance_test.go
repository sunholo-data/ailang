package types_test

import (
	"strings"
	"testing"

	"github.com/sunholo-data/ailang/internal/ast"
	"github.com/sunholo-data/ailang/internal/types"
)

// TestEffectRowMismatch_ExtraLabel_NamesCallSite verifies that when the actual
// effect row has labels not present in the expected row, the error message
// names the call site that introduced each extra label (via Provenance).
func TestEffectRowMismatch_ExtraLabel_NamesCallSite(t *testing.T) {
	callSpan := ast.Span{Start: ast.Pos{File: "src/agent.ail", Line: 10, Column: 5}}

	expected := &types.Row{
		Kind:   types.EffectRow,
		Labels: map[string]types.Type{"Env": types.TUnit},
	}
	actual := &types.Row{
		Kind:       types.EffectRow,
		Labels:     map[string]types.Type{"Env": types.TUnit, "AI": types.TUnit},
		Provenance: map[string]ast.Span{"AI": callSpan},
	}

	err := types.NewRowMismatchError(expected, actual, []string{"function register"})
	if err == nil {
		t.Fatal("expected a mismatch error, got nil")
	}

	msg := err.Error()
	if !strings.Contains(msg, "src/agent.ail:10:5") {
		t.Errorf("error message should contain call site src/agent.ail:10:5\ngot: %s", msg)
	}
	if !strings.Contains(msg, "AI") {
		t.Errorf("error message should name the extra effect AI\ngot: %s", msg)
	}
}

// TestEffectRowMismatch_MissingLabel_NamesSlot verifies that when the expected
// row has labels absent from the actual row, the error message includes the
// source location recorded in the expected row's Provenance for those labels.
func TestEffectRowMismatch_MissingLabel_NamesSlot(t *testing.T) {
	slotSpan := ast.Span{Start: ast.Pos{File: "src/iface.ail", Line: 3, Column: 20}}

	// expected carries provenance: FS was declared at this slot position
	expected := &types.Row{
		Kind:       types.EffectRow,
		Labels:     map[string]types.Type{"Env": types.TUnit, "FS": types.TUnit},
		Provenance: map[string]ast.Span{"FS": slotSpan},
	}
	actual := &types.Row{
		Kind:   types.EffectRow,
		Labels: map[string]types.Type{"Env": types.TUnit},
	}

	err := types.NewRowMismatchError(expected, actual, []string{"function call"})
	if err == nil {
		t.Fatal("expected a mismatch error, got nil")
	}

	msg := err.Error()
	if !strings.Contains(msg, "src/iface.ail:3:20") {
		t.Errorf("error message should contain slot location src/iface.ail:3:20\ngot: %s", msg)
	}
	if !strings.Contains(msg, "FS") {
		t.Errorf("error message should name the missing effect FS\ngot: %s", msg)
	}
}

// TestEffectRowMismatch_NoProvenance_StillWorks verifies that the existing
// error formatting continues to work when no Provenance is set (backward compat).
func TestEffectRowMismatch_NoProvenance_StillWorks(t *testing.T) {
	expected := &types.Row{
		Kind:   types.EffectRow,
		Labels: map[string]types.Type{"Env": types.TUnit},
	}
	actual := &types.Row{
		Kind:   types.EffectRow,
		Labels: map[string]types.Type{"Env": types.TUnit, "AI": types.TUnit},
	}

	err := types.NewRowMismatchError(expected, actual, []string{"function foo"})
	if err == nil {
		t.Fatal("expected a mismatch error, got nil")
	}

	msg := err.Error()
	// Should still mention the extra effect even without provenance
	if !strings.Contains(msg, "AI") {
		t.Errorf("error message should name the extra effect AI\ngot: %s", msg)
	}
}

// TestRowEquals_ProvenanceIgnored verifies that two Rows are considered equal
// when their Labels and Kind match, even if Provenance differs — so provenance
// never causes spurious unification failures.
func TestRowEquals_ProvenanceIgnored(t *testing.T) {
	span := ast.Span{Start: ast.Pos{File: "a.ail", Line: 1, Column: 1}}

	r1 := &types.Row{
		Kind:       types.EffectRow,
		Labels:     map[string]types.Type{"AI": types.TUnit},
		Provenance: map[string]ast.Span{"AI": span},
	}
	r2 := &types.Row{
		Kind:   types.EffectRow,
		Labels: map[string]types.Type{"AI": types.TUnit},
	}

	if !r1.Equals(r2) {
		t.Error("rows with same Kind and Labels should be equal regardless of Provenance")
	}
}
