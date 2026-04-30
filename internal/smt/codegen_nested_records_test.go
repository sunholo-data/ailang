package smt

import (
	"strings"
	"testing"

	"github.com/sunholo-data/ailang/internal/core"
	"github.com/sunholo-data/ailang/internal/types"
)

// --- Nested record type discovery tests (M1_NESTED_RECORDS) ---

func TestCollectRecordType_NestedOneLevelDeep(t *testing.T) {
	// Outer record has a field that is itself a record:
	// {name: string, pos: {x: int, y: int}}
	ctx := NewSMTContext()
	result := &EncodeResult{}
	activeRecordTypes = make(map[string]*RecordTypeInfo)
	activeFieldSetToSort = make(map[string]string)
	defer func() { activeRecordTypes = nil; activeFieldSetToSort = nil }()

	innerRec := &types.TRecord{
		Fields: map[string]types.Type{
			"x": &types.TCon{Name: "int"},
			"y": &types.TCon{Name: "int"},
		},
	}
	outerRec := &types.TRecord{
		Fields: map[string]types.Type{
			"name": &types.TCon{Name: "string"},
			"pos":  innerRec,
		},
	}

	collectRecordType(outerRec, ctx, result)

	// Both inner and outer should be declared
	if len(activeRecordTypes) != 2 {
		t.Errorf("expected 2 record types (inner + outer), got %d", len(activeRecordTypes))
	}

	// Inner record should be declared
	innerSortName := MapRecordSortName(innerRec)
	if _, ok := activeRecordTypes[innerSortName]; !ok {
		t.Errorf("inner record type %q not found in activeRecordTypes", innerSortName)
	}

	// Outer record should be declared
	outerSortName := MapRecordSortName(outerRec)
	if _, ok := activeRecordTypes[outerSortName]; !ok {
		t.Errorf("outer record type %q not found in activeRecordTypes", outerSortName)
	}
}

func TestCollectRecordType_NestedTwoLevelsDeep(t *testing.T) {
	// Three-level nesting:
	// {label: string, details: {addr: {street: string, zip: int}, count: int}}
	ctx := NewSMTContext()
	result := &EncodeResult{}
	activeRecordTypes = make(map[string]*RecordTypeInfo)
	activeFieldSetToSort = make(map[string]string)
	defer func() { activeRecordTypes = nil; activeFieldSetToSort = nil }()

	addrRec := &types.TRecord{
		Fields: map[string]types.Type{
			"street": &types.TCon{Name: "string"},
			"zip":    &types.TCon{Name: "int"},
		},
	}
	detailsRec := &types.TRecord{
		Fields: map[string]types.Type{
			"addr":  addrRec,
			"count": &types.TCon{Name: "int"},
		},
	}
	topRec := &types.TRecord{
		Fields: map[string]types.Type{
			"label":   &types.TCon{Name: "string"},
			"details": detailsRec,
		},
	}

	collectRecordType(topRec, ctx, result)

	// All three levels should be declared
	if len(activeRecordTypes) != 3 {
		t.Errorf("expected 3 record types (addr + details + top), got %d", len(activeRecordTypes))
	}
}

// --- Topological sort tests ---

func TestTopologicalSortRecordDeclarations(t *testing.T) {
	// Inner record must be declared BEFORE outer record.
	// {pos: {x: int, y: int}} should produce:
	//   (declare-datatype Record_x_y ...)
	//   (declare-datatype Record_name_pos ...)
	ctx := NewSMTContext()
	result := &EncodeResult{}
	activeRecordTypes = make(map[string]*RecordTypeInfo)
	activeFieldSetToSort = make(map[string]string)
	defer func() { activeRecordTypes = nil; activeFieldSetToSort = nil }()

	innerRec := &types.TRecord{
		Fields: map[string]types.Type{
			"x": &types.TCon{Name: "int"},
			"y": &types.TCon{Name: "int"},
		},
	}
	outerRec := &types.TRecord{
		Fields: map[string]types.Type{
			"name": &types.TCon{Name: "string"},
			"pos":  innerRec,
		},
	}

	params := []FunctionParam{
		{Name: "o", Type: outerRec},
	}

	collectAndDeclareRecordTypes(params, "", nil, nil, nil, ctx, result)

	// Find the indices of declarations
	innerSortName := MapRecordSortName(innerRec)
	outerSortName := MapRecordSortName(outerRec)

	// Use precise prefix match: "(declare-datatype SortName " to avoid substring collisions
	innerPrefix := "(declare-datatype " + innerSortName + " "
	outerPrefix := "(declare-datatype " + outerSortName + " "

	innerIdx := -1
	outerIdx := -1
	for i, decl := range result.Declarations {
		if strings.HasPrefix(decl, innerPrefix) {
			innerIdx = i
		}
		if strings.HasPrefix(decl, outerPrefix) {
			outerIdx = i
		}
	}

	if innerIdx == -1 {
		t.Fatalf("inner record declaration not found in: %v", result.Declarations)
	}
	if outerIdx == -1 {
		t.Fatalf("outer record declaration not found in: %v", result.Declarations)
	}
	if innerIdx >= outerIdx {
		t.Errorf("inner record (index %d) should be declared before outer record (index %d)", innerIdx, outerIdx)
	}
}

func TestTopologicalSort_ThreeLevels(t *testing.T) {
	// Three levels: addr < details < top
	ctx := NewSMTContext()
	result := &EncodeResult{}
	activeRecordTypes = make(map[string]*RecordTypeInfo)
	activeFieldSetToSort = make(map[string]string)
	defer func() { activeRecordTypes = nil; activeFieldSetToSort = nil }()

	addrRec := &types.TRecord{
		Fields: map[string]types.Type{
			"street": &types.TCon{Name: "string"},
			"zip":    &types.TCon{Name: "int"},
		},
	}
	detailsRec := &types.TRecord{
		Fields: map[string]types.Type{
			"addr":  addrRec,
			"count": &types.TCon{Name: "int"},
		},
	}
	topRec := &types.TRecord{
		Fields: map[string]types.Type{
			"label":   &types.TCon{Name: "string"},
			"details": detailsRec,
		},
	}

	params := []FunctionParam{
		{Name: "t", Type: topRec},
	}

	collectAndDeclareRecordTypes(params, "", nil, nil, nil, ctx, result)

	addrSort := MapRecordSortName(addrRec)
	detailsSort := MapRecordSortName(detailsRec)
	topSort := MapRecordSortName(topRec)

	addrIdx := -1
	detailsIdx := -1
	topIdx := -1
	for i, decl := range result.Declarations {
		if strings.Contains(decl, "declare-datatype "+addrSort+" ") {
			addrIdx = i
		}
		if strings.Contains(decl, "declare-datatype "+detailsSort+" ") {
			detailsIdx = i
		}
		if strings.Contains(decl, "declare-datatype "+topSort+" ") {
			topIdx = i
		}
	}

	if addrIdx == -1 {
		t.Fatalf("addr declaration not found in: %v", result.Declarations)
	}
	if detailsIdx == -1 {
		t.Fatalf("details declaration not found in: %v", result.Declarations)
	}
	if topIdx == -1 {
		t.Fatalf("top declaration not found in: %v", result.Declarations)
	}

	if addrIdx >= detailsIdx {
		t.Errorf("addr (index %d) must come before details (index %d)", addrIdx, detailsIdx)
	}
	if detailsIdx >= topIdx {
		t.Errorf("details (index %d) must come before top (index %d)", detailsIdx, topIdx)
	}
}

// --- Self-referential record rejection tests ---

func TestCollectRecordType_SelfReferential_Rejected(t *testing.T) {
	// A record that references itself: {next: <self>}
	// This is a conceptual test — in practice, the user would write:
	//   type Node = {value: int, next: Node}
	// Which would produce a TRecord with a field typed as TRecord pointing to itself.
	//
	// We simulate this by creating a circular reference.
	ctx := NewSMTContext()
	result := &EncodeResult{}
	activeRecordTypes = make(map[string]*RecordTypeInfo)
	activeFieldSetToSort = make(map[string]string)
	defer func() { activeRecordTypes = nil; activeFieldSetToSort = nil }()

	selfRec := &types.TRecord{
		Fields: map[string]types.Type{
			"value": &types.TCon{Name: "int"},
		},
	}
	// Create circular reference
	selfRec.Fields["next"] = selfRec

	err := collectRecordTypeSafe(selfRec, ctx, result, nil)
	if err == nil {
		t.Error("expected error for self-referential record, got nil")
	}
	if err != nil && !strings.Contains(err.Error(), "self-referential") && !strings.Contains(err.Error(), "cycle") {
		t.Errorf("expected cycle/self-referential error, got: %v", err)
	}
}

func TestCollectRecordType_MutuallyRecursive_Rejected(t *testing.T) {
	// Two records referencing each other: A -> B -> A
	ctx := NewSMTContext()
	result := &EncodeResult{}
	activeRecordTypes = make(map[string]*RecordTypeInfo)
	activeFieldSetToSort = make(map[string]string)
	defer func() { activeRecordTypes = nil; activeFieldSetToSort = nil }()

	recA := &types.TRecord{
		Fields: map[string]types.Type{
			"val": &types.TCon{Name: "int"},
		},
	}
	recB := &types.TRecord{
		Fields: map[string]types.Type{
			"ref": recA,
		},
	}
	// Create mutual reference
	recA.Fields["other"] = recB

	err := collectRecordTypeSafe(recA, ctx, result, nil)
	if err == nil {
		t.Error("expected error for mutually recursive records, got nil")
	}
	if err != nil && !strings.Contains(err.Error(), "self-referential") && !strings.Contains(err.Error(), "cycle") {
		t.Errorf("expected cycle/self-referential error, got: %v", err)
	}
}

// --- Nested field access encoding tests ---

func TestEncodeNestedRecordAccess(t *testing.T) {
	// o.pos.x → (x (pos o))
	// This is represented as RecordAccess{Record: RecordAccess{Record: Var("o"), Field: "pos"}, Field: "x"}
	setupNestedRecordTestContext()
	defer teardownNestedRecordTestContext()

	expr := &core.RecordAccess{
		Record: &core.RecordAccess{
			Record: &core.Var{Name: "o"},
			Field:  "pos",
		},
		Field: "x",
	}

	got, err := EncodeExpr(expr)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "(x (pos o))" {
		t.Errorf("got %q, want %q", got, "(x (pos o))")
	}
}

func TestEncodeTripleNestedRecordAccess(t *testing.T) {
	// a.b.c.d → (d (c (b a)))
	setupNestedRecordTestContext()
	defer teardownNestedRecordTestContext()

	expr := &core.RecordAccess{
		Record: &core.RecordAccess{
			Record: &core.RecordAccess{
				Record: &core.Var{Name: "a"},
				Field:  "b",
			},
			Field: "c",
		},
		Field: "d",
	}

	got, err := EncodeExpr(expr)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "(d (c (b a)))" {
		t.Errorf("got %q, want %q", got, "(d (c (b a)))")
	}
}

// --- Full EncodeFunction with nested records ---

func TestEncodeFunction_NestedRecordParam(t *testing.T) {
	// func getInnerX(o: {name: string, pos: {x: int, y: int}}) -> int
	// requires { o.pos.x >= 0 }
	// ensures { result >= 0 }
	// { o.pos.x }
	innerRec := &types.TRecord{
		Fields: map[string]types.Type{
			"x": &types.TCon{Name: "int"},
			"y": &types.TCon{Name: "int"},
		},
	}
	outerRec := &types.TRecord{
		Fields: map[string]types.Type{
			"name": &types.TCon{Name: "string"},
			"pos":  innerRec,
		},
	}

	params := []FunctionParam{
		{Name: "o", Type: outerRec},
	}

	body := &core.RecordAccess{
		Record: &core.RecordAccess{
			Record: &core.Var{Name: "o"},
			Field:  "pos",
		},
		Field: "x",
	}

	// requires { o.pos.x >= 0 }
	requiresExpr := &core.App{
		Func: &core.VarGlobal{Ref: core.GlobalRef{Module: "$builtin", Name: "ge_Int"}},
		Args: []core.CoreExpr{
			&core.RecordAccess{
				Record: &core.RecordAccess{
					Record: &core.Var{Name: "o"},
					Field:  "pos",
				},
				Field: "x",
			},
			&core.Lit{Kind: core.IntLit, Value: int64(0)},
		},
	}

	// ensures { result >= 0 }
	ensuresExpr := &core.App{
		Func: &core.VarGlobal{Ref: core.GlobalRef{Module: "$builtin", Name: "ge_Int"}},
		Args: []core.CoreExpr{
			&core.Var{Name: "result"},
			&core.Lit{Kind: core.IntLit, Value: int64(0)},
		},
	}

	meta := &core.DeclMeta{
		IsPure: true,
		Contracts: []*core.Contract{
			{Kind: core.RequiresKind, Expr: requiresExpr},
			{Kind: core.EnsuresKind, Expr: ensuresExpr},
		},
	}

	innerSortName := MapRecordSortName(innerRec)
	outerSortName := MapRecordSortName(outerRec)

	opts := EncodeFunctionOpts{
		ReturnType: &types.TCon{Name: "int"},
		Body:       body,
		Contracts:  meta.Contracts,
	}

	result, err := EncodeFunction("getInnerX", params, body, "Int", meta, nil, opts)
	if err != nil {
		t.Fatalf("EncodeFunction with nested record: unexpected error: %v", err)
	}

	// Both inner and outer record types must be declared
	if !strings.Contains(result.SMTLib, "declare-datatype "+innerSortName) {
		t.Errorf("inner record type %q not declared in SMT-LIB:\n%s", innerSortName, result.SMTLib)
	}
	if !strings.Contains(result.SMTLib, "declare-datatype "+outerSortName) {
		t.Errorf("outer record type %q not declared in SMT-LIB:\n%s", outerSortName, result.SMTLib)
	}

	// Inner must come before outer
	innerPos := strings.Index(result.SMTLib, "declare-datatype "+innerSortName)
	outerPos := strings.Index(result.SMTLib, "declare-datatype "+outerSortName)
	if innerPos >= outerPos {
		t.Errorf("inner record must be declared before outer record in SMT-LIB:\n%s", result.SMTLib)
	}

	// Body should encode nested access: (x (pos $p_o)) — parameter `o` is
	// renamed via the `$p_` prefix to avoid record-accessor collisions.
	if result.BodyExpr != "(x (pos $p_o))" {
		t.Errorf("body expression = %q, want %q", result.BodyExpr, "(x (pos $p_o))")
	}
}

// --- Nested record field sort mapping ---

func TestMapRecordFields_NestedRecord(t *testing.T) {
	// A record with a nested record field should map the inner field
	// to the inner record's sort name
	innerRec := &types.TRecord{
		Fields: map[string]types.Type{
			"x": &types.TCon{Name: "int"},
			"y": &types.TCon{Name: "int"},
		},
	}
	outerRec := &types.TRecord{
		Fields: map[string]types.Type{
			"name": &types.TCon{Name: "string"},
			"pos":  innerRec,
		},
	}

	fields, err := MapRecordFields(outerRec)
	if err != nil {
		t.Fatalf("MapRecordFields with nested record: unexpected error: %v", err)
	}

	if fields["name"] != "String" {
		t.Errorf("field 'name': got %q, want %q", fields["name"], "String")
	}

	innerSortName := MapRecordSortName(innerRec)
	if fields["pos"] != innerSortName {
		t.Errorf("field 'pos': got %q, want %q (inner record sort name)", fields["pos"], innerSortName)
	}
}

// --- Helper functions ---

func setupNestedRecordTestContext() {
	activeRecordTypes = map[string]*RecordTypeInfo{
		"Record_x_y": {
			SortName:   "Record_x_y",
			CtorName:   "mk_Record_x_y",
			FieldNames: []string{"x", "y"},
			FieldSorts: map[string]string{"x": "Int", "y": "Int"},
		},
		"Record_name_pos": {
			SortName:   "Record_name_pos",
			CtorName:   "mk_Record_name_pos",
			FieldNames: []string{"name", "pos"},
			FieldSorts: map[string]string{"name": "String", "pos": "Record_x_y"},
		},
	}
	activeFieldSetToSort = map[string]string{
		"x,y":      "Record_x_y",
		"name,pos": "Record_name_pos",
	}
}

func teardownNestedRecordTestContext() {
	activeRecordTypes = nil
	activeFieldSetToSort = nil
}
