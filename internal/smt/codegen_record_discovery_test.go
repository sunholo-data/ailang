package smt

import (
	"strings"
	"testing"

	"github.com/sunholo/ailang/internal/core"
	"github.com/sunholo/ailang/internal/types"
)

// --- Record type discovery tests (M-SMT-RECORD-DISCOVERY) ---

func TestCollectAndDeclareRecordTypes_ReturnType(t *testing.T) {
	// Record type appears in return type annotation only, not in params
	ctx := NewSMTContext()
	result := &EncodeResult{}
	activeRecordTypes = make(map[string]*RecordTypeInfo)
	activeFieldSetToSort = make(map[string]string)
	defer func() { activeRecordTypes = nil; activeFieldSetToSort = nil }()

	params := []FunctionParam{
		{Name: "x", Type: &types.TCon{Name: "int"}},
		{Name: "y", Type: &types.TCon{Name: "int"}},
	}
	returnType := &types.TRecord{
		Fields:   map[string]types.Type{"x": &types.TCon{Name: "int"}, "y": &types.TCon{Name: "int"}},
		TypeName: "Point",
	}

	collectAndDeclareRecordTypes(params, "Point", returnType, nil, nil, ctx, result)

	// The Point record should be discovered and declared
	if _, ok := activeRecordTypes["Point"]; !ok {
		t.Error("expected Point record type to be discovered from return type")
	}
	if !ctx.DeclaredTypes["Point"] {
		t.Error("expected Point to be declared in ctx.DeclaredTypes")
	}
	if len(result.Declarations) != 1 {
		t.Errorf("expected 1 declaration, got %d", len(result.Declarations))
	}
	if !strings.Contains(result.Declarations[0], "Point") {
		t.Errorf("expected Point in declaration, got %q", result.Declarations[0])
	}
}

func TestCollectAndDeclareRecordTypes_BodyRecord(t *testing.T) {
	// Record appears only in function body (let binding), not in params or return type
	ctx := NewSMTContext()
	result := &EncodeResult{}
	activeRecordTypes = make(map[string]*RecordTypeInfo)
	activeFieldSetToSort = make(map[string]string)
	defer func() { activeRecordTypes = nil; activeFieldSetToSort = nil }()

	params := []FunctionParam{
		{Name: "a", Type: &types.TCon{Name: "int"}},
		{Name: "b", Type: &types.TCon{Name: "int"}},
	}
	body := &core.Let{
		Name: "delta",
		Value: &core.Record{
			Fields: map[string]core.CoreExpr{
				"dx": &core.Lit{Kind: core.IntLit, Value: int64(0)},
				"dy": &core.Lit{Kind: core.IntLit, Value: int64(0)},
			},
		},
		Body: &core.Var{Name: "delta"},
	}

	collectAndDeclareRecordTypes(params, "Int", nil, body, nil, ctx, result)

	// The anonymous record with dx,dy should be discovered
	if len(activeRecordTypes) != 1 {
		t.Errorf("expected 1 record type discovered from body, got %d", len(activeRecordTypes))
	}
	// Check the field-set lookup works
	if _, ok := activeFieldSetToSort["dx,dy"]; !ok {
		t.Error("expected dx,dy field-set in activeFieldSetToSort")
	}
}

func TestCollectAndDeclareRecordTypes_EnsuresClause(t *testing.T) {
	// Record appears in an ensures clause expression
	ctx := NewSMTContext()
	result := &EncodeResult{}
	activeRecordTypes = make(map[string]*RecordTypeInfo)
	activeFieldSetToSort = make(map[string]string)
	defer func() { activeRecordTypes = nil; activeFieldSetToSort = nil }()

	params := []FunctionParam{
		{Name: "x", Type: &types.TCon{Name: "int"}},
	}
	ensuresBody := &core.Record{
		Fields: map[string]core.CoreExpr{
			"a": &core.Lit{Kind: core.IntLit, Value: int64(1)},
			"b": &core.Lit{Kind: core.IntLit, Value: int64(2)},
		},
	}
	contracts := []*core.Contract{
		{Kind: core.EnsuresKind, Expr: ensuresBody},
	}

	collectAndDeclareRecordTypes(params, "Int", nil, nil, contracts, ctx, result)

	if len(activeRecordTypes) != 1 {
		t.Errorf("expected 1 record type from ensures clause, got %d", len(activeRecordTypes))
	}
	if _, ok := activeFieldSetToSort["a,b"]; !ok {
		t.Error("expected a,b field-set in activeFieldSetToSort")
	}
}

func TestCollectRecordTypesFromBody_NestedInIf(t *testing.T) {
	// Record in an if-then branch
	ctx := NewSMTContext()
	result := &EncodeResult{}
	activeRecordTypes = make(map[string]*RecordTypeInfo)
	activeFieldSetToSort = make(map[string]string)
	defer func() { activeRecordTypes = nil; activeFieldSetToSort = nil }()

	body := &core.If{
		Cond: &core.Lit{Kind: core.BoolLit, Value: true},
		Then: &core.Record{
			Fields: map[string]core.CoreExpr{
				"x": &core.Lit{Kind: core.IntLit, Value: int64(1)},
				"y": &core.Lit{Kind: core.IntLit, Value: int64(2)},
			},
		},
		Else: &core.Record{
			Fields: map[string]core.CoreExpr{
				"x": &core.Lit{Kind: core.IntLit, Value: int64(3)},
				"y": &core.Lit{Kind: core.IntLit, Value: int64(4)},
			},
		},
	}

	collectRecordTypesFromBody(body, ctx, result)

	// Both branches have the same record shape — should discover once
	if len(activeRecordTypes) != 1 {
		t.Errorf("expected 1 record type (both branches same shape), got %d", len(activeRecordTypes))
	}
}

func TestCollectRecordTypesFromBody_NestedInMatch(t *testing.T) {
	// Record in a match arm
	ctx := NewSMTContext()
	result := &EncodeResult{}
	activeRecordTypes = make(map[string]*RecordTypeInfo)
	activeFieldSetToSort = make(map[string]string)
	defer func() { activeRecordTypes = nil; activeFieldSetToSort = nil }()

	body := &core.Match{
		Scrutinee: &core.Var{Name: "x"},
		Arms: []core.MatchArm{
			{
				Pattern: &core.ConstructorPattern{Name: "A"},
				Body: &core.Record{
					Fields: map[string]core.CoreExpr{
						"name":  &core.Lit{Kind: core.StringLit, Value: "test"},
						"count": &core.Lit{Kind: core.IntLit, Value: int64(0)},
					},
				},
			},
		},
	}

	collectRecordTypesFromBody(body, ctx, result)

	if len(activeRecordTypes) != 1 {
		t.Errorf("expected 1 record type from match arm, got %d", len(activeRecordTypes))
	}
	if _, ok := activeFieldSetToSort["count,name"]; !ok {
		t.Error("expected count,name field-set")
	}
}

func TestInferRecordTypeFromLiteral_IntFields(t *testing.T) {
	rec := &core.Record{
		Fields: map[string]core.CoreExpr{
			"x": &core.Lit{Kind: core.IntLit, Value: int64(5)},
			"y": &core.Lit{Kind: core.IntLit, Value: int64(10)},
		},
	}
	result := inferRecordTypeFromLiteral(rec)
	if result == nil {
		t.Fatal("expected TRecord, got nil")
	}
	if len(result.Fields) != 2 {
		t.Errorf("expected 2 fields, got %d", len(result.Fields))
	}
	for name, typ := range result.Fields {
		tcon, ok := typ.(*types.TCon)
		if !ok {
			t.Errorf("field %q: expected TCon, got %T", name, typ)
			continue
		}
		if tcon.Name != "int" {
			t.Errorf("field %q: expected int, got %s", name, tcon.Name)
		}
	}
}

func TestInferRecordTypeFromLiteral_UninferableField(t *testing.T) {
	// A field with a variable (not a literal) can't be inferred
	rec := &core.Record{
		Fields: map[string]core.CoreExpr{
			"x": &core.Var{Name: "unknown"},
		},
	}
	result := inferRecordTypeFromLiteral(rec)
	if result != nil {
		t.Error("expected nil for uninferrable field type, got non-nil")
	}
}

func TestEncodeFunction_ReturnOnlyRecord(t *testing.T) {
	// Function returns a record type not present in any parameter
	params := []FunctionParam{
		{Name: "x", Type: &types.TCon{Name: "int"}},
		{Name: "y", Type: &types.TCon{Name: "int"}},
	}
	body := &core.Record{
		Fields: map[string]core.CoreExpr{
			"x": &core.Var{Name: "x"},
			"y": &core.Var{Name: "y"},
		},
	}
	meta := &core.DeclMeta{
		Name:   "makePoint",
		IsPure: true,
		Contracts: []*core.Contract{
			{
				Kind: core.RequiresKind,
				Expr: &core.Lit{Kind: core.BoolLit, Value: true},
			},
			{
				Kind: core.EnsuresKind,
				Expr: &core.App{
					Func: &core.VarGlobal{Ref: core.GlobalRef{Module: "$builtin", Name: "eq_Int"}},
					Args: []core.CoreExpr{
						&core.RecordAccess{Record: &core.Var{Name: "result"}, Field: "x"},
						&core.Var{Name: "x"},
					},
				},
			},
		},
	}

	returnType := &types.TRecord{
		Fields: map[string]types.Type{
			"x": &types.TCon{Name: "int"},
			"y": &types.TCon{Name: "int"},
		},
	}

	opts := EncodeFunctionOpts{
		ReturnType: returnType,
		Body:       body,
		Contracts:  meta.Contracts,
	}

	result, err := EncodeFunction("makePoint", params, body, "Rec_x_y", meta, nil, opts)
	if err != nil {
		t.Fatalf("EncodeFunction with return-only record: unexpected error: %v", err)
	}

	// Should contain record type declaration (discovered from return type)
	if !strings.Contains(result.SMTLib, "declare-datatype") {
		t.Errorf("expected record datatype declaration in SMT-LIB:\n%s", result.SMTLib)
	}
	// Should contain mk_ constructor
	if !strings.Contains(result.SMTLib, "mk_") {
		t.Errorf("expected mk_ constructor in SMT-LIB:\n%s", result.SMTLib)
	}
	// Should NOT contain "unknown record type" error
	if strings.Contains(result.SMTLib, "unknown record type") {
		t.Errorf("unexpected 'unknown record type' in SMT-LIB:\n%s", result.SMTLib)
	}
}
