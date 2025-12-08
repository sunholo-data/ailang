package core

import (
	"fmt"
	"testing"
)

// TestResolveValue_DirectLiteral tests resolving a literal value
func TestResolveValue_DirectLiteral(t *testing.T) {
	lit := &Lit{CoreNode: CoreNode{NodeID: 1}, Kind: IntLit, Value: 42}
	bindings := map[string]CoreExpr{}

	resolved := ResolveValue(lit, bindings)

	if resolved != lit {
		t.Errorf("Expected same literal, got %v", resolved)
	}
}

// TestResolveValue_SingleVarBinding tests resolving a single variable
func TestResolveValue_SingleVarBinding(t *testing.T) {
	lit := &Lit{CoreNode: CoreNode{NodeID: 1}, Kind: StringLit, Value: "hello"}
	v := &Var{CoreNode: CoreNode{NodeID: 2}, Name: "x"}

	bindings := map[string]CoreExpr{
		"x": lit,
	}

	resolved := ResolveValue(v, bindings)

	if resolved != lit {
		t.Errorf("Expected literal, got %v", resolved)
	}
}

// TestResolveValue_ChainedVars tests resolving a chain of variables (x → y → z → literal)
func TestResolveValue_ChainedVars(t *testing.T) {
	lit := &List{CoreNode: CoreNode{NodeID: 1}, Elements: []CoreExpr{}}

	bindings := map[string]CoreExpr{
		"x": lit,
		"y": &Var{CoreNode: CoreNode{NodeID: 2}, Name: "x"},
		"z": &Var{CoreNode: CoreNode{NodeID: 3}, Name: "y"},
	}

	v := &Var{CoreNode: CoreNode{NodeID: 4}, Name: "z"}
	resolved := ResolveValue(v, bindings)

	if resolved != lit {
		t.Errorf("Expected list literal, got %v", resolved)
	}
}

// TestResolveValue_CycleDetection tests that cycles are detected and fail-closed
func TestResolveValue_CycleDetection(t *testing.T) {
	// Create a cycle: a → b → c → a
	bindings := map[string]CoreExpr{
		"a": &Var{CoreNode: CoreNode{NodeID: 1}, Name: "b"},
		"b": &Var{CoreNode: CoreNode{NodeID: 2}, Name: "c"},
		"c": &Var{CoreNode: CoreNode{NodeID: 3}, Name: "a"},
	}

	v := &Var{CoreNode: CoreNode{NodeID: 4}, Name: "a"}

	// Should detect cycle and return the last resolvable var (fail-closed)
	resolved := ResolveValue(v, bindings)

	// Should return a Var (stopped at cycle)
	if _, ok := resolved.(*Var); !ok {
		t.Errorf("Expected Var (cycle detected), got %T", resolved)
	}
}

// TestResolveValue_UnboundVar tests that unbound variables are returned as-is
func TestResolveValue_UnboundVar(t *testing.T) {
	v := &Var{CoreNode: CoreNode{NodeID: 1}, Name: "unbound"}
	bindings := map[string]CoreExpr{}

	resolved := ResolveValue(v, bindings)

	if resolved != v {
		t.Errorf("Expected same var, got %v", resolved)
	}
}

// TestResolveValue_DeepChain tests a 20-link chain (mini-fuzz)
func TestResolveValue_DeepChain(t *testing.T) {
	// Create a 20-link chain
	bindings := map[string]CoreExpr{}
	terminalLit := &Lit{CoreNode: CoreNode{NodeID: 999}, Kind: IntLit, Value: 42}

	// Build chain: v0 → v1 → v2 → ... → v19 → literal
	bindings["v19"] = terminalLit
	for i := 0; i < 19; i++ {
		currName := fmt.Sprintf("v%d", i)
		nextName := fmt.Sprintf("v%d", i+1)
		bindings[currName] = &Var{CoreNode: CoreNode{NodeID: uint64(i)}, Name: nextName}
	}

	startVar := &Var{CoreNode: CoreNode{NodeID: 100}, Name: "v0"}
	resolved := ResolveValue(startVar, bindings)

	// Should resolve to the terminal literal
	if lit, ok := resolved.(*Lit); !ok || lit.Value != 42 {
		t.Errorf("Expected to resolve to literal 42, got %v", resolved)
	}
}

// TestIsListValue_List tests that list literals are detected
func TestIsListValue_List(t *testing.T) {
	list := &List{CoreNode: CoreNode{NodeID: 1}, Elements: []CoreExpr{}}
	bindings := map[string]CoreExpr{}

	if !IsListValue(list, bindings) {
		t.Errorf("Expected IsListValue to return true for List")
	}
}

// TestIsListValue_ListThroughVar tests that lists are detected through variable bindings
func TestIsListValue_ListThroughVar(t *testing.T) {
	list := &List{CoreNode: CoreNode{NodeID: 1}, Elements: []CoreExpr{}}
	bindings := map[string]CoreExpr{
		"x": list,
		"y": &Var{CoreNode: CoreNode{NodeID: 2}, Name: "x"},
	}

	v := &Var{CoreNode: CoreNode{NodeID: 3}, Name: "y"}

	if !IsListValue(v, bindings) {
		t.Errorf("Expected IsListValue to return true for var → var → list")
	}
}

// TestIsListValue_NotList tests that non-lists return false
func TestIsListValue_NotList(t *testing.T) {
	lit := &Lit{CoreNode: CoreNode{NodeID: 1}, Kind: StringLit, Value: "hello"}
	bindings := map[string]CoreExpr{}

	if IsListValue(lit, bindings) {
		t.Errorf("Expected IsListValue to return false for string literal")
	}
}

// TestIsStringValue_String tests that string literals are detected
func TestIsStringValue_String(t *testing.T) {
	lit := &Lit{CoreNode: CoreNode{NodeID: 1}, Kind: StringLit, Value: "hello"}
	bindings := map[string]CoreExpr{}

	if !IsStringValue(lit, bindings) {
		t.Errorf("Expected IsStringValue to return true for string literal")
	}
}

// TestIsStringValue_NotString tests that non-strings return false
func TestIsStringValue_NotString(t *testing.T) {
	lit := &Lit{CoreNode: CoreNode{NodeID: 1}, Kind: IntLit, Value: 42}
	bindings := map[string]CoreExpr{}

	if IsStringValue(lit, bindings) {
		t.Errorf("Expected IsStringValue to return false for int literal")
	}
}

// TestIsIntValue_Int tests that int literals are detected
func TestIsIntValue_Int(t *testing.T) {
	lit := &Lit{CoreNode: CoreNode{NodeID: 1}, Kind: IntLit, Value: 42}
	bindings := map[string]CoreExpr{}

	if !IsIntValue(lit, bindings) {
		t.Errorf("Expected IsIntValue to return true for int literal")
	}
}

// TestIsIntValue_NotInt tests that non-ints return false
func TestIsIntValue_NotInt(t *testing.T) {
	lit := &Lit{CoreNode: CoreNode{NodeID: 1}, Kind: FloatLit, Value: 3.14}
	bindings := map[string]CoreExpr{}

	if IsIntValue(lit, bindings) {
		t.Errorf("Expected IsIntValue to return false for float literal")
	}
}

// TestIsFloatValue_Float tests that float literals are detected
func TestIsFloatValue_Float(t *testing.T) {
	lit := &Lit{CoreNode: CoreNode{NodeID: 1}, Kind: FloatLit, Value: 3.14}
	bindings := map[string]CoreExpr{}

	if !IsFloatValue(lit, bindings) {
		t.Errorf("Expected IsFloatValue to return true for float literal")
	}
}

// TestIsFloatValue_NotFloat tests that non-floats return false
func TestIsFloatValue_NotFloat(t *testing.T) {
	lit := &Lit{CoreNode: CoreNode{NodeID: 1}, Kind: IntLit, Value: 42}
	bindings := map[string]CoreExpr{}

	if IsFloatValue(lit, bindings) {
		t.Errorf("Expected IsFloatValue to return false for int literal")
	}
}

// TestIsBoolValue_Bool tests that bool literals are detected
func TestIsBoolValue_Bool(t *testing.T) {
	lit := &Lit{CoreNode: CoreNode{NodeID: 1}, Kind: BoolLit, Value: true}
	bindings := map[string]CoreExpr{}

	if !IsBoolValue(lit, bindings) {
		t.Errorf("Expected IsBoolValue to return true for bool literal")
	}
}

// TestIsBoolValue_NotBool tests that non-bools return false
func TestIsBoolValue_NotBool(t *testing.T) {
	lit := &Lit{CoreNode: CoreNode{NodeID: 1}, Kind: StringLit, Value: "true"}
	bindings := map[string]CoreExpr{}

	if IsBoolValue(lit, bindings) {
		t.Errorf("Expected IsBoolValue to return false for string literal")
	}
}

// TestTypeHelpers_ChainResolution tests all type helpers with chained variables
func TestTypeHelpers_ChainResolution(t *testing.T) {
	tests := []struct {
		name     string
		value    CoreExpr
		isInt    bool
		isFloat  bool
		isString bool
		isBool   bool
		isList   bool
	}{
		{
			name:     "int literal",
			value:    &Lit{CoreNode: CoreNode{NodeID: 1}, Kind: IntLit, Value: 42},
			isInt:    true,
			isFloat:  false,
			isString: false,
			isBool:   false,
			isList:   false,
		},
		{
			name:     "float literal",
			value:    &Lit{CoreNode: CoreNode{NodeID: 2}, Kind: FloatLit, Value: 3.14},
			isInt:    false,
			isFloat:  true,
			isString: false,
			isBool:   false,
			isList:   false,
		},
		{
			name:     "string literal",
			value:    &Lit{CoreNode: CoreNode{NodeID: 3}, Kind: StringLit, Value: "hello"},
			isInt:    false,
			isFloat:  false,
			isString: true,
			isBool:   false,
			isList:   false,
		},
		{
			name:     "bool literal",
			value:    &Lit{CoreNode: CoreNode{NodeID: 4}, Kind: BoolLit, Value: true},
			isInt:    false,
			isFloat:  false,
			isString: false,
			isBool:   true,
			isList:   false,
		},
		{
			name:     "list literal",
			value:    &List{CoreNode: CoreNode{NodeID: 5}, Elements: []CoreExpr{}},
			isInt:    false,
			isFloat:  false,
			isString: false,
			isBool:   false,
			isList:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a chain: v0 → v1 → v2 → value
			bindings := map[string]CoreExpr{
				"v2": tt.value,
				"v1": &Var{CoreNode: CoreNode{NodeID: 10}, Name: "v2"},
				"v0": &Var{CoreNode: CoreNode{NodeID: 11}, Name: "v1"},
			}

			v := &Var{CoreNode: CoreNode{NodeID: 12}, Name: "v0"}

			// Test all type helpers
			if IsIntValue(v, bindings) != tt.isInt {
				t.Errorf("IsIntValue: expected %v, got %v", tt.isInt, IsIntValue(v, bindings))
			}
			if IsFloatValue(v, bindings) != tt.isFloat {
				t.Errorf("IsFloatValue: expected %v, got %v", tt.isFloat, IsFloatValue(v, bindings))
			}
			if IsStringValue(v, bindings) != tt.isString {
				t.Errorf("IsStringValue: expected %v, got %v", tt.isString, IsStringValue(v, bindings))
			}
			if IsBoolValue(v, bindings) != tt.isBool {
				t.Errorf("IsBoolValue: expected %v, got %v", tt.isBool, IsBoolValue(v, bindings))
			}
			if IsListValue(v, bindings) != tt.isList {
				t.Errorf("IsListValue: expected %v, got %v", tt.isList, IsListValue(v, bindings))
			}
		})
	}
}
