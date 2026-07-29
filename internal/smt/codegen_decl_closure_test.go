package smt

import (
	"errors"
	"strings"
	"testing"

	"github.com/sunholo-data/ailang/internal/core"
	"github.com/sunholo-data/ailang/internal/types"
)

func TestEncodeFunction_RecordAliasReachesADT(t *testing.T) {
	params := []FunctionParam{{Name: "p", Type: &types.TCon{Name: "Proposal"}}}
	meta := &core.DeclMeta{Name: "hasName", IsPure: true}
	opts := EncodeFunctionOpts{RecordTypeAliases: map[string]*types.TRecord{
		"Proposal": {Fields: map[string]types.Type{
			"name":     &types.TCon{Name: "string"},
			"evidence": &types.TList{Element: &types.TCon{Name: "Evidence"}},
		}},
	}}
	adts := map[string][]ADTVariant{
		"Evidence": {{Name: "CompilerOutput", Fields: []ADTField{{Name: "CompilerOutput_0", Sort: "String"}}}},
	}
	result, err := EncodeFunction("hasName", params, &core.Lit{Kind: core.BoolLit, Value: true}, "Bool", meta, adts, opts)
	if err != nil {
		t.Fatalf("EncodeFunction: %v", err)
	}
	evidenceAt := strings.Index(result.SMTLib, "(declare-datatype Evidence ")
	proposalAt := strings.Index(result.SMTLib, "(declare-datatype Proposal ")
	if evidenceAt < 0 || proposalAt < 0 || evidenceAt > proposalAt {
		t.Fatalf("expected Evidence before Proposal:\n%s", result.SMTLib)
	}
}

func TestEncodeFunction_RecordADTCycleUsesPluralGroup(t *testing.T) {
	params := []FunctionParam{{Name: "d", Type: &types.TCon{Name: "Doc"}}}
	opts := EncodeFunctionOpts{
		RecordTypeAliases: map[string]*types.TRecord{"Doc": {Fields: map[string]types.Type{
			"blocks": &types.TList{Element: &types.TCon{Name: "Block"}},
		}}},
		ExtraDeclarations: []string{
			`(declare-datatype Record_blocks_kind ((mk_Record_blocks_kind (blocks (Seq Block)) (kind String))))`,
		},
	}
	adts := map[string][]ADTVariant{"Block": {
		{Name: "Para", Fields: []ADTField{{Name: "Para_0", Sort: "String"}}},
		{Name: "Container", Fields: []ADTField{{Name: "Container_0", Sort: "Record_blocks_kind"}}},
	}}
	result, err := EncodeFunction("titleOf", params, &core.Lit{Kind: core.StringLit, Value: ""}, "String",
		&core.DeclMeta{Name: "titleOf", IsPure: true}, adts, opts)
	if err != nil {
		t.Fatalf("EncodeFunction: %v", err)
	}
	if got := strings.Count(result.SMTLib, "(declare-datatypes (("); got != 1 {
		t.Fatalf("plural group count = %d, want 1:\n%s", got, result.SMTLib)
	}
	for _, name := range []string{"(Block 0)", "(Record_blocks_kind 0)"} {
		if !strings.Contains(result.SMTLib, name) {
			t.Fatalf("missing %s:\n%s", name, result.SMTLib)
		}
	}
}

func TestValidateDeclarations_ConstantsAndPluralGroups(t *testing.T) {
	ctx := NewSMTContext()
	if err := validateDeclarations([]string{`(declare-const $p_p Missing)`}, ctx); !errors.Is(err, ErrUnresolvableTypes) ||
		!strings.Contains(err.Error(), "$p_p") || !strings.Contains(err.Error(), "Missing") {
		t.Fatalf("dangling constant error = %v", err)
	}
	evidence := `(declare-datatype Evidence ((CompilerOutput (CompilerOutput_0 String))))`
	if err := validateDeclarations([]string{evidence, `(declare-const xs (Seq Evidence))`}, ctx); err != nil {
		t.Fatalf("declared Seq element rejected: %v", err)
	}
	if err := validateDeclarations([]string{`(declare-const xs (Seq Evidence))`, evidence}, ctx); !errors.Is(err, ErrUnresolvableTypes) {
		t.Fatalf("forward Seq element error = %v", err)
	}
	plural := `(declare-datatypes ((A 0) (B 0)) (((MkA (b B))) ((MkB (a A)))))`
	if err := validateDeclarations([]string{plural, `(declare-const x A)`, `(declare-const y B)`}, ctx); err != nil {
		t.Fatalf("atomic plural declaration rejected: %v", err)
	}
	if err := validateDeclarations([]string{plural, `(declare-const z C)`}, ctx); !errors.Is(err, ErrUnresolvableTypes) ||
		!strings.Contains(err.Error(), "C") {
		t.Fatalf("unknown post-group sort error = %v", err)
	}
}

func TestEncodeFunction_UnmappableNeededAliasFailsLoud(t *testing.T) {
	opts := EncodeFunctionOpts{RecordTypeAliases: map[string]*types.TRecord{
		"Broken": {Fields: map[string]types.Type{"value": &types.TVar{Name: "a"}}},
	}}
	_, err := EncodeFunction("f", []FunctionParam{{Name: "x", Type: &types.TCon{Name: "Broken"}}},
		&core.Lit{Kind: core.IntLit, Value: int64(1)}, "Int", &core.DeclMeta{Name: "f", IsPure: true}, nil, opts)
	if !errors.Is(err, ErrUnresolvableTypes) || !strings.Contains(err.Error(), "Broken") {
		t.Fatalf("unmappable alias error = %v", err)
	}
}

// TestEncodeFunction_DeclarationOrderIsDeterministic is the AC1.5 guard.
//
// Mutually INDEPENDENT ADTs (none references another) are emitted in `pending` order.
// If either the record-alias pass or the ADT pass iterates its Go map directly, that order
// is randomized per run and the generated SMT-LIB differs between invocations on identical
// input — a direct violation of design axiom A1 (Determinism).
//
// Red mutation: in EncodeFunction, replace the sorted `adtNames` iteration with a bare
// `for typeName, variants := range adtTypes`. It compiles, and this test fails.
// (Measured before the fix: 40 CLI invocations produced 3 distinct declaration orders.)
func TestEncodeFunction_DeclarationOrderIsDeterministic(t *testing.T) {
	encode := func() string {
		t.Helper()
		params := []FunctionParam{{Name: "p", Type: &types.TCon{Name: "Proposal"}}}
		opts := EncodeFunctionOpts{RecordTypeAliases: map[string]*types.TRecord{
			"Proposal": {Fields: map[string]types.Type{
				"name":      &types.TCon{Name: "string"},
				"evidence":  &types.TList{Element: &types.TCon{Name: "Evidence"}},
				"decisions": &types.TList{Element: &types.TCon{Name: "Decision"}},
				"verdicts":  &types.TList{Element: &types.TCon{Name: "Verdict"}},
			}},
			"Unused":  {Fields: map[string]types.Type{"a": &types.TCon{Name: "string"}}},
			"Another": {Fields: map[string]types.Type{"b": &types.TCon{Name: "int"}}},
		}}
		// Three ADTs, none referencing another: their relative order is unconstrained by
		// dependency, so only an explicit sort can pin it.
		adts := map[string][]ADTVariant{
			"Evidence": {{Name: "CompilerOutput", Fields: []ADTField{{Name: "CompilerOutput_0", Sort: "String"}}}},
			"Decision": {{Name: "Accept", Fields: []ADTField{{Name: "Accept_0", Sort: "String"}}}},
			"Verdict":  {{Name: "Pending", Fields: []ADTField{{Name: "Pending_0", Sort: "String"}}}},
		}
		result, err := EncodeFunction("hasName", params, &core.Lit{Kind: core.BoolLit, Value: true}, "Bool",
			&core.DeclMeta{Name: "hasName", IsPure: true}, adts, opts)
		if err != nil {
			t.Fatalf("EncodeFunction: %v", err)
		}
		return result.SMTLib
	}

	first := encode()
	for i := 1; i < 50; i++ {
		if got := encode(); got != first {
			t.Fatalf("SMT-LIB not byte-identical on encode #%d (A1 determinism violation)\n--- first ---\n%s\n--- got ---\n%s",
				i+1, first, got)
		}
	}
}
