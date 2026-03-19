package traverse

import (
	"testing"

	"github.com/sunholo/ailang/internal/types"
)

func TestVisitor_SimpleTypes(t *testing.T) {
	tests := []struct {
		name string
		typ  types.Type
	}{
		{"TVar", &types.TVar{Name: "a"}},
		{"TVar2", &types.TVar2{Name: "b", Kind: types.Star}},
		{"TCon", &types.TCon{Name: "int"}},
		{"RowVar", &types.RowVar{Name: "r", Kind: types.RecordRow}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var visited []types.Type
			Walk(tt.typ, func(typ types.Type) {
				visited = append(visited, typ)
			})

			if len(visited) != 1 {
				t.Errorf("expected 1 visited type, got %d", len(visited))
			}
			if visited[0] != tt.typ {
				t.Errorf("expected %v, got %v", tt.typ, visited[0])
			}
		})
	}
}

func TestVisitor_TFunc(t *testing.T) {
	// int -> string -> bool
	intType := &types.TCon{Name: "int"}
	stringType := &types.TCon{Name: "string"}
	boolType := &types.TCon{Name: "bool"}

	funcType := &types.TFunc2{
		Params: []types.Type{intType, stringType},
		Return: boolType,
	}

	var visited []types.Type
	Walk(funcType, func(typ types.Type) {
		visited = append(visited, typ)
	})

	// Should visit: TFunc2, int, string, bool
	if len(visited) != 4 {
		t.Errorf("expected 4 visited types, got %d", len(visited))
	}
}

func TestVisitor_TFunc2WithEffects(t *testing.T) {
	intType := &types.TCon{Name: "int"}
	effectRow := &types.Row{
		Labels: map[string]types.Type{"IO": nil},
		Kind:   types.EffectRow,
	}

	funcType := &types.TFunc2{
		Params:    []types.Type{intType},
		Return:    intType,
		EffectRow: effectRow,
	}

	var visited []types.Type
	Walk(funcType, func(typ types.Type) {
		visited = append(visited, typ)
	})

	// Should visit: TFunc2, int (param), int (return), Row
	if len(visited) != 4 {
		t.Errorf("expected 4 visited types, got %d", len(visited))
	}
}

func TestVisitor_TList(t *testing.T) {
	elemType := &types.TCon{Name: "int"}
	listType := &types.TList{Element: elemType}

	var visited []types.Type
	Walk(listType, func(typ types.Type) {
		visited = append(visited, typ)
	})

	if len(visited) != 2 {
		t.Errorf("expected 2 visited types, got %d", len(visited))
	}
}

func TestVisitor_TTuple(t *testing.T) {
	intType := &types.TCon{Name: "int"}
	stringType := &types.TCon{Name: "string"}
	tupleType := &types.TTuple{Elements: []types.Type{intType, stringType}}

	var visited []types.Type
	Walk(tupleType, func(typ types.Type) {
		visited = append(visited, typ)
	})

	// Should visit: TTuple, int, string
	if len(visited) != 3 {
		t.Errorf("expected 3 visited types, got %d", len(visited))
	}
}

func TestVisitor_TRecord(t *testing.T) {
	intType := &types.TCon{Name: "int"}
	stringType := &types.TCon{Name: "string"}
	recordType := &types.TRecord{
		Fields: map[string]types.Type{
			"x": intType,
			"y": stringType,
		},
	}

	var visited []types.Type
	Walk(recordType, func(typ types.Type) {
		visited = append(visited, typ)
	})

	// Should visit: TRecord, int, string
	if len(visited) != 3 {
		t.Errorf("expected 3 visited types, got %d", len(visited))
	}
}

func TestVisitor_TApp(t *testing.T) {
	// List[int]
	listCon := &types.TCon{Name: "List"}
	intType := &types.TCon{Name: "int"}
	appType := &types.TApp{
		Constructor: listCon,
		Args:        []types.Type{intType},
	}

	var visited []types.Type
	Walk(appType, func(typ types.Type) {
		visited = append(visited, typ)
	})

	// Should visit: TApp, List, int
	if len(visited) != 3 {
		t.Errorf("expected 3 visited types, got %d", len(visited))
	}
}

func TestVisitor_Row(t *testing.T) {
	intType := &types.TCon{Name: "int"}
	tail := &types.RowVar{Name: "r", Kind: types.RecordRow}
	row := &types.Row{
		Labels: map[string]types.Type{"x": intType},
		Tail:   tail,
		Kind:   types.RecordRow,
	}

	var visited []types.Type
	Walk(row, func(typ types.Type) {
		visited = append(visited, typ)
	})

	// Should visit: Row, int, RowVar
	if len(visited) != 3 {
		t.Errorf("expected 3 visited types, got %d", len(visited))
	}
}

func TestVisitor_CycleDetection(t *testing.T) {
	// Create a cyclic type: List[T] where T refers back to List[T]
	// Simulate with a TApp that references itself

	listCon := &types.TCon{Name: "List"}
	appType := &types.TApp{
		Constructor: listCon,
		Args:        nil, // Will set to self
	}
	// Create cycle: args contains self
	appType.Args = []types.Type{appType}

	cycleCount := 0
	v := NewVisitor().WithOnCycle(func(typ types.Type) {
		cycleCount++
	})

	visitCount := 0
	v.Visit(appType, func(typ types.Type) {
		visitCount++
	})

	// Should visit: TApp, List, then detect cycle on TApp
	if cycleCount != 1 {
		t.Errorf("expected 1 cycle detection, got %d", cycleCount)
	}
	if visitCount != 2 {
		t.Errorf("expected 2 visits (before cycle), got %d", visitCount)
	}
}

func TestVisitor_CycleDetection_SilentSkip(t *testing.T) {
	// Without OnCycle, cycles should be silently skipped
	listCon := &types.TCon{Name: "List"}
	appType := &types.TApp{
		Constructor: listCon,
		Args:        nil,
	}
	appType.Args = []types.Type{appType}

	visitCount := 0
	Walk(appType, func(typ types.Type) {
		visitCount++
	})

	// Should visit: TApp, List, then silently skip cycle
	if visitCount != 2 {
		t.Errorf("expected 2 visits, got %d", visitCount)
	}
}

func TestVisitor_DepthLimit(t *testing.T) {
	// Create a deeply nested type
	var typ types.Type = &types.TCon{Name: "int"}
	for i := 0; i < 5; i++ {
		typ = &types.TList{Element: typ}
	}

	// With depth limit of 3, should panic
	v := NewVisitor().WithMaxDepth(3)

	defer func() {
		if r := recover(); r == nil {
			t.Error("expected panic on depth limit, got none")
		}
	}()

	v.Visit(typ, func(typ types.Type) {})
}

func TestVisitor_NilType(t *testing.T) {
	// Should handle nil gracefully
	var visited []types.Type
	Walk(nil, func(typ types.Type) {
		visited = append(visited, typ)
	})

	if len(visited) != 0 {
		t.Errorf("expected 0 visited types for nil, got %d", len(visited))
	}
}

func TestVisitor_NestedCycles(t *testing.T) {
	// Test more complex cycle: A -> B -> A
	aCon := &types.TCon{Name: "A"}
	bCon := &types.TCon{Name: "B"}

	aApp := &types.TApp{Constructor: aCon, Args: nil}
	bApp := &types.TApp{Constructor: bCon, Args: []types.Type{aApp}}
	aApp.Args = []types.Type{bApp}

	cycleCount := 0
	v := NewVisitor().WithOnCycle(func(typ types.Type) {
		cycleCount++
	})

	visitCount := 0
	v.Visit(aApp, func(typ types.Type) {
		visitCount++
	})

	// Should visit: aApp, A, bApp, B, then detect cycle on aApp
	if cycleCount != 1 {
		t.Errorf("expected 1 cycle, got %d", cycleCount)
	}
	if visitCount != 4 {
		t.Errorf("expected 4 visits, got %d", visitCount)
	}
}

func TestWalkWithCycleCallback(t *testing.T) {
	// Test convenience function
	listCon := &types.TCon{Name: "List"}
	appType := &types.TApp{
		Constructor: listCon,
		Args:        nil,
	}
	appType.Args = []types.Type{appType}

	cycleCount := 0
	visitCount := 0

	WalkWithCycleCallback(appType,
		func(typ types.Type) { visitCount++ },
		func(typ types.Type) { cycleCount++ },
	)

	if cycleCount != 1 {
		t.Errorf("expected 1 cycle, got %d", cycleCount)
	}
	if visitCount != 2 {
		t.Errorf("expected 2 visits, got %d", visitCount)
	}
}

// Test that TRecord2 with Row is traversed correctly
func TestVisitor_TRecord2(t *testing.T) {
	intType := &types.TCon{Name: "int"}
	row := &types.Row{
		Labels: map[string]types.Type{"x": intType},
		Kind:   types.RecordRow,
	}
	recordType := &types.TRecord2{Row: row}

	var visited []types.Type
	Walk(recordType, func(typ types.Type) {
		visited = append(visited, typ)
	})

	// Should visit: TRecord2, Row, int
	if len(visited) != 3 {
		t.Errorf("expected 3 visited types, got %d", len(visited))
	}
}

// Test TArray traversal
func TestVisitor_TArray(t *testing.T) {
	elemType := &types.TCon{Name: "int"}
	arrayType := &types.TArray{Element: elemType}

	var visited []types.Type
	Walk(arrayType, func(typ types.Type) {
		visited = append(visited, typ)
	})

	if len(visited) != 2 {
		t.Errorf("expected 2 visited types, got %d", len(visited))
	}
}

// Test TRecordOpen traversal
func TestVisitor_TRecordOpen(t *testing.T) {
	intType := &types.TCon{Name: "int"}
	rowVar := &types.RowVar{Name: "r", Kind: types.RecordRow}
	recordType := &types.TRecordOpen{
		Fields: map[string]types.Type{"x": intType},
		Row:    rowVar,
	}

	var visited []types.Type
	Walk(recordType, func(typ types.Type) {
		visited = append(visited, typ)
	})

	// Should visit: TRecordOpen, int, RowVar
	if len(visited) != 3 {
		t.Errorf("expected 3 visited types, got %d", len(visited))
	}
}

// ========== Wrapper Function Tests ==========

func TestCollectFreeVars(t *testing.T) {
	// a -> b -> c
	aVar := &types.TVar{Name: "a"}
	bVar := &types.TVar{Name: "b"}
	cVar := &types.TVar{Name: "c"}

	funcType := &types.TFunc2{
		Params: []types.Type{aVar, bVar},
		Return: cVar,
	}

	vars := CollectFreeVars(funcType)

	if len(vars) != 3 {
		t.Errorf("expected 3 free vars, got %d", len(vars))
	}
	for _, name := range []string{"a", "b", "c"} {
		if !vars[name] {
			t.Errorf("expected var %s to be collected", name)
		}
	}
}

func TestCollectFreeVars_TVar2(t *testing.T) {
	aVar := &types.TVar2{Name: "alpha", Kind: types.Star}
	bVar := &types.TVar2{Name: "beta", Kind: types.Star}

	funcType := &types.TFunc2{
		Params: []types.Type{aVar},
		Return: bVar,
	}

	vars := CollectFreeVars(funcType)

	if len(vars) != 2 {
		t.Errorf("expected 2 free vars, got %d", len(vars))
	}
	if !vars["alpha"] || !vars["beta"] {
		t.Errorf("expected alpha and beta, got %v", vars)
	}
}

func TestCollectFreeVars_RowVar(t *testing.T) {
	rowVar := &types.RowVar{Name: "r", Kind: types.RecordRow}
	row := &types.Row{
		Labels: map[string]types.Type{},
		Tail:   rowVar,
		Kind:   types.RecordRow,
	}

	vars := CollectFreeVars(row)

	if !vars["r"] {
		t.Errorf("expected row var 'r' to be collected")
	}
}

func TestCollectFreeVars_CyclicType(t *testing.T) {
	// Test that cyclic types don't hang
	listCon := &types.TCon{Name: "List"}
	aVar := &types.TVar{Name: "a"}
	appType := &types.TApp{
		Constructor: listCon,
		Args:        []types.Type{aVar},
	}
	// Create cycle
	appType.Args = append(appType.Args, appType)

	// This should not hang
	vars := CollectFreeVars(appType)

	if !vars["a"] {
		t.Errorf("expected 'a' to be collected even with cycle")
	}
}

func TestContainsType(t *testing.T) {
	intType := &types.TCon{Name: "int"}
	stringType := &types.TCon{Name: "string"}
	listType := &types.TList{Element: intType}

	if !ContainsType(listType, intType) {
		t.Error("expected intType to be contained in listType")
	}
	if ContainsType(listType, stringType) {
		t.Error("expected stringType NOT to be contained in listType")
	}
}

func TestContainsTypeByName(t *testing.T) {
	aVar := &types.TVar{Name: "a"}
	funcType := &types.TFunc2{
		Params: []types.Type{aVar},
		Return: &types.TCon{Name: "int"},
	}

	if !ContainsTypeByName(funcType, "a") {
		t.Error("expected 'a' to be found")
	}
	if ContainsTypeByName(funcType, "b") {
		t.Error("expected 'b' NOT to be found")
	}
}

func TestCountTypes(t *testing.T) {
	// int -> string
	intType := &types.TCon{Name: "int"}
	stringType := &types.TCon{Name: "string"}
	funcType := &types.TFunc2{
		Params: []types.Type{intType},
		Return: stringType,
	}

	count := CountTypes(funcType)
	// TFunc2, int, string
	if count != 3 {
		t.Errorf("expected 3 types, got %d", count)
	}
}

func TestCollectTypesByKind(t *testing.T) {
	intType := &types.TCon{Name: "int"}
	stringType := &types.TCon{Name: "string"}
	funcType := &types.TFunc2{
		Params: []types.Type{intType},
		Return: stringType,
	}

	// Collect all TCon types
	cons := CollectTypesByKind(funcType, func(typ types.Type) bool {
		_, ok := typ.(*types.TCon)
		return ok
	})

	if len(cons) != 2 {
		t.Errorf("expected 2 TCon types, got %d", len(cons))
	}
}

func TestAllTypes(t *testing.T) {
	intType := &types.TCon{Name: "int"}
	listType := &types.TList{Element: intType}

	all := AllTypes(listType)

	if len(all) != 2 {
		t.Errorf("expected 2 types, got %d", len(all))
	}
}

func TestHasCycles_NoCycle(t *testing.T) {
	intType := &types.TCon{Name: "int"}
	listType := &types.TList{Element: intType}

	if HasCycles(listType) {
		t.Error("expected no cycles in simple list type")
	}
}

func TestHasCycles_WithCycle(t *testing.T) {
	listCon := &types.TCon{Name: "List"}
	appType := &types.TApp{
		Constructor: listCon,
		Args:        nil,
	}
	appType.Args = []types.Type{appType} // Create cycle

	if !HasCycles(appType) {
		t.Error("expected cycle to be detected")
	}
}

func TestDepth(t *testing.T) {
	// int has depth 0
	intType := &types.TCon{Name: "int"}
	if d := Depth(intType); d != 0 {
		t.Errorf("expected depth 0 for TCon, got %d", d)
	}

	// List[int] has depth 1
	listType := &types.TList{Element: intType}
	if d := Depth(listType); d != 1 {
		t.Errorf("expected depth 1 for List[int], got %d", d)
	}

	// List[List[int]] has depth 2
	nestedList := &types.TList{Element: listType}
	if d := Depth(nestedList); d != 2 {
		t.Errorf("expected depth 2 for List[List[int]], got %d", d)
	}
}

func TestDepth_Nil(t *testing.T) {
	if d := Depth(nil); d != 0 {
		t.Errorf("expected depth 0 for nil, got %d", d)
	}
}

// ========== HasTypeVars / IsMonomorphic Tests ==========

func TestHasTypeVars_WithTVar(t *testing.T) {
	aVar := &types.TVar{Name: "a"}
	funcType := &types.TFunc2{
		Params: []types.Type{aVar},
		Return: &types.TCon{Name: "int"},
	}

	if !HasTypeVars(funcType) {
		t.Error("expected HasTypeVars to return true for type with TVar")
	}
}

func TestHasTypeVars_WithTVar2(t *testing.T) {
	aVar := &types.TVar2{Name: "alpha", Kind: types.Star}
	funcType := &types.TFunc2{
		Params: []types.Type{aVar},
		Return: &types.TCon{Name: "int"},
	}

	if !HasTypeVars(funcType) {
		t.Error("expected HasTypeVars to return true for type with TVar2")
	}
}

func TestHasTypeVars_WithRowVar(t *testing.T) {
	rowVar := &types.RowVar{Name: "r", Kind: types.RecordRow}
	row := &types.Row{
		Labels: map[string]types.Type{},
		Tail:   rowVar,
		Kind:   types.RecordRow,
	}

	if !HasTypeVars(row) {
		t.Error("expected HasTypeVars to return true for type with RowVar")
	}
}

func TestHasTypeVars_Concrete(t *testing.T) {
	// int -> string (no type variables)
	funcType := &types.TFunc2{
		Params: []types.Type{&types.TCon{Name: "int"}},
		Return: &types.TCon{Name: "string"},
	}

	if HasTypeVars(funcType) {
		t.Error("expected HasTypeVars to return false for concrete type")
	}
}

func TestHasTypeVars_Cyclic(t *testing.T) {
	// Cyclic type with type variable - should not hang
	listCon := &types.TCon{Name: "List"}
	aVar := &types.TVar{Name: "a"}
	appType := &types.TApp{
		Constructor: listCon,
		Args:        []types.Type{aVar},
	}
	// Create cycle
	appType.Args = append(appType.Args, appType)

	// Should complete without hanging
	result := HasTypeVars(appType)
	if !result {
		t.Error("expected HasTypeVars to return true for cyclic type with TVar")
	}
}

func TestHasTypeVars_CyclicConcrete(t *testing.T) {
	// Cyclic type without type variables - should not hang
	listCon := &types.TCon{Name: "List"}
	intType := &types.TCon{Name: "int"}
	appType := &types.TApp{
		Constructor: listCon,
		Args:        []types.Type{intType},
	}
	// Create cycle
	appType.Args = append(appType.Args, appType)

	// Should complete without hanging
	result := HasTypeVars(appType)
	if result {
		t.Error("expected HasTypeVars to return false for cyclic concrete type")
	}
}

func TestIsMonomorphic_Concrete(t *testing.T) {
	funcType := &types.TFunc2{
		Params: []types.Type{&types.TCon{Name: "int"}},
		Return: &types.TCon{Name: "string"},
	}

	if !IsMonomorphic(funcType) {
		t.Error("expected IsMonomorphic to return true for concrete type")
	}
}

func TestIsMonomorphic_Polymorphic(t *testing.T) {
	aVar := &types.TVar{Name: "a"}
	funcType := &types.TFunc2{
		Params: []types.Type{aVar},
		Return: aVar,
	}

	if IsMonomorphic(funcType) {
		t.Error("expected IsMonomorphic to return false for polymorphic type")
	}
}

func TestHasTypeVars_NestedType(t *testing.T) {
	// List[a] where a is a type variable
	aVar := &types.TVar{Name: "a"}
	listType := &types.TList{Element: aVar}

	if !HasTypeVars(listType) {
		t.Error("expected HasTypeVars to return true for List[a]")
	}
}

func TestHasTypeVars_DeeplyNestedVar(t *testing.T) {
	// List[List[List[a]]]
	aVar := &types.TVar{Name: "a"}
	list1 := &types.TList{Element: aVar}
	list2 := &types.TList{Element: list1}
	list3 := &types.TList{Element: list2}

	if !HasTypeVars(list3) {
		t.Error("expected HasTypeVars to return true for deeply nested type var")
	}
}

func TestHasTypeVars_RecordWithVar(t *testing.T) {
	aVar := &types.TVar{Name: "a"}
	recordType := &types.TRecord{
		Fields: map[string]types.Type{
			"x": &types.TCon{Name: "int"},
			"y": aVar,
		},
	}

	if !HasTypeVars(recordType) {
		t.Error("expected HasTypeVars to return true for record with type var")
	}
}

func TestHasTypeVars_TFunc2WithEffects(t *testing.T) {
	// a -> int ! {IO}
	aVar := &types.TVar2{Name: "a", Kind: types.Star}
	effectRow := &types.Row{
		Labels: map[string]types.Type{"IO": nil},
		Kind:   types.EffectRow,
	}
	funcType := &types.TFunc2{
		Params:    []types.Type{aVar},
		Return:    &types.TCon{Name: "int"},
		EffectRow: effectRow,
	}

	if !HasTypeVars(funcType) {
		t.Error("expected HasTypeVars to return true for TFunc2 with type var")
	}
}
