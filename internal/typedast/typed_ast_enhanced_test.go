package typedast

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/sunholo/ailang/internal/ast"
	"github.com/sunholo/ailang/internal/core"
)

func TestTypedExpr_InterfaceMethods(t *testing.T) {
	typedExpr := TypedExpr{
		NodeID:    42,
		Span:      ast.Pos{Line: 10, Column: 5, File: "test.ail"},
		Type:      "Int",
		EffectRow: "IO",
		Core: &core.Lit{
			CoreNode: core.CoreNode{NodeID: 42},
			Kind:     core.IntLit,
			Value:    5,
		},
	}

	assert.Equal(t, uint64(42), typedExpr.GetNodeID())
	assert.Equal(t, ast.Pos{Line: 10, Column: 5, File: "test.ail"}, typedExpr.GetSpan())
	assert.Equal(t, "Int", typedExpr.GetType())
	assert.Equal(t, "IO", typedExpr.GetEffectRow())
	assert.NotNil(t, typedExpr.GetCore())
}

func TestTypedVar_String(t *testing.T) {
	typedVar := &TypedVar{
		TypedExpr: TypedExpr{NodeID: 1, Type: "Int"},
		Name:      "myVar",
	}

	assert.Equal(t, "myVar", typedVar.String())
}

func TestTypedLit_String(t *testing.T) {
	tests := []struct {
		name     string
		kind     core.LitKind
		value    interface{}
		expected string
	}{
		{"int literal", core.IntLit, 42, "42"},
		{"string literal", core.StringLit, "hello", "hello"},
		{"bool literal", core.BoolLit, true, "true"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			typedLit := &TypedLit{
				TypedExpr: TypedExpr{NodeID: 1, Type: "test"},
				Kind:      tt.kind,
				Value:     tt.value,
			}
			assert.Equal(t, tt.expected, typedLit.String())
		})
	}
}

func TestTypedLambda_String(t *testing.T) {
	bodyVar := &TypedVar{
		TypedExpr: TypedExpr{NodeID: 2, Type: "Int"},
		Name:      "x",
	}

	typedLambda := &TypedLambda{
		TypedExpr:  TypedExpr{NodeID: 1, Type: "Int -> Int"},
		Params:     []string{"x", "y"},
		ParamTypes: []interface{}{"Int", "Int"},
		Body:       bodyVar,
	}

	result := typedLambda.String()
	assert.Contains(t, result, "λ")
	assert.Contains(t, result, "Int -> Int")
}

func TestTypedLet_String(t *testing.T) {
	value := &TypedLit{
		TypedExpr: TypedExpr{NodeID: 2, Type: "Int"},
		Kind:      core.IntLit,
		Value:     5,
	}

	body := &TypedVar{
		TypedExpr: TypedExpr{NodeID: 3, Type: "Int"},
		Name:      "x",
	}

	typedLet := &TypedLet{
		TypedExpr: TypedExpr{NodeID: 1, Type: "Int"},
		Name:      "x",
		Scheme:    "Int",
		Value:     value,
		Body:      body,
	}

	result := typedLet.String()
	assert.Contains(t, result, "let x")
	assert.Contains(t, result, "in")
}

func TestTypedLetRec_String(t *testing.T) {
	bodyVar := &TypedVar{
		TypedExpr: TypedExpr{NodeID: 2, Type: "Int"},
		Name:      "f",
	}

	typedLetRec := &TypedLetRec{
		TypedExpr: TypedExpr{NodeID: 1, Type: "Int"},
		Bindings: []TypedRecBinding{
			{Name: "f", Scheme: "Int -> Int", Value: bodyVar},
		},
		Body: bodyVar,
	}

	result := typedLetRec.String()
	assert.Contains(t, result, "let rec")
	assert.Contains(t, result, "f")
}

func TestTypedApp_String(t *testing.T) {
	fn := &TypedVar{
		TypedExpr: TypedExpr{NodeID: 1, Type: "(Int, Int) -> Int"},
		Name:      "add",
	}

	arg1 := &TypedLit{
		TypedExpr: TypedExpr{NodeID: 2, Type: "Int"},
		Kind:      core.IntLit,
		Value:     1,
	}

	typedApp := &TypedApp{
		TypedExpr: TypedExpr{NodeID: 4, Type: "Int"},
		Func:      fn,
		Args:      []TypedNode{arg1},
	}

	result := typedApp.String()
	assert.Contains(t, result, "add")
	assert.Contains(t, result, "Int")
}

func TestTypedIf_String(t *testing.T) {
	cond := &TypedLit{
		TypedExpr: TypedExpr{NodeID: 1, Type: "Bool"},
		Kind:      core.BoolLit,
		Value:     true,
	}

	thenBranch := &TypedLit{
		TypedExpr: TypedExpr{NodeID: 2, Type: "Int"},
		Kind:      core.IntLit,
		Value:     1,
	}

	elseBranch := &TypedLit{
		TypedExpr: TypedExpr{NodeID: 3, Type: "Int"},
		Kind:      core.IntLit,
		Value:     2,
	}

	typedIf := &TypedIf{
		TypedExpr: TypedExpr{NodeID: 4, Type: "Int"},
		Cond:      cond,
		Then:      thenBranch,
		Else:      elseBranch,
	}

	result := typedIf.String()
	assert.Contains(t, result, "if")
	assert.Contains(t, result, "then")
	assert.Contains(t, result, "else")
}

func TestTypedMatch_String(t *testing.T) {
	scrutinee := &TypedVar{
		TypedExpr: TypedExpr{NodeID: 1, Type: "Int"},
		Name:      "x",
	}

	typedMatch := &TypedMatch{
		TypedExpr:  TypedExpr{NodeID: 2, Type: "String"},
		Scrutinee:  scrutinee,
		Arms:       []TypedMatchArm{},
		Exhaustive: true,
	}

	result := typedMatch.String()
	assert.Contains(t, result, "match")
	assert.Contains(t, result, "x")
}

func TestTypedBinOp_String(t *testing.T) {
	left := &TypedLit{
		TypedExpr: TypedExpr{NodeID: 1, Type: "Int"},
		Kind:      core.IntLit,
		Value:     1,
	}

	right := &TypedLit{
		TypedExpr: TypedExpr{NodeID: 2, Type: "Int"},
		Kind:      core.IntLit,
		Value:     2,
	}

	typedBinOp := &TypedBinOp{
		TypedExpr: TypedExpr{NodeID: 3, Type: "Int"},
		Op:        "+",
		Left:      left,
		Right:     right,
	}

	result := typedBinOp.String()
	assert.Contains(t, result, "+")
	assert.Contains(t, result, "Int")
}

func TestTypedUnOp_String(t *testing.T) {
	operand := &TypedLit{
		TypedExpr: TypedExpr{NodeID: 1, Type: "Int"},
		Kind:      core.IntLit,
		Value:     42,
	}

	typedUnOp := &TypedUnOp{
		TypedExpr: TypedExpr{NodeID: 2, Type: "Int"},
		Op:        "-",
		Operand:   operand,
	}

	result := typedUnOp.String()
	assert.Contains(t, result, "-")
	assert.Contains(t, result, "Int")
}

func TestTypedRecord_String(t *testing.T) {
	typedRecord := &TypedRecord{
		TypedExpr: TypedExpr{NodeID: 1, Type: "{x: Int, y: Int}"},
		Fields: map[string]TypedNode{
			"x": &TypedLit{TypedExpr: TypedExpr{Type: "Int"}, Kind: core.IntLit, Value: 10},
			"y": &TypedLit{TypedExpr: TypedExpr{Type: "Int"}, Kind: core.IntLit, Value: 20},
		},
	}

	result := typedRecord.String()
	assert.Contains(t, result, "{...}")
}

func TestTypedRecordAccess_String(t *testing.T) {
	record := &TypedVar{
		TypedExpr: TypedExpr{NodeID: 1, Type: "{x: Int}"},
		Name:      "r",
	}

	typedRecordAccess := &TypedRecordAccess{
		TypedExpr: TypedExpr{NodeID: 2, Type: "Int"},
		Record:    record,
		Field:     "x",
	}

	result := typedRecordAccess.String()
	assert.Contains(t, result, "r.x")
}

func TestTypedList_String(t *testing.T) {
	typedList := &TypedList{
		TypedExpr: TypedExpr{NodeID: 1, Type: "[Int]"},
		Elements: []TypedNode{
			&TypedLit{TypedExpr: TypedExpr{Type: "Int"}, Kind: core.IntLit, Value: 1},
			&TypedLit{TypedExpr: TypedExpr{Type: "Int"}, Kind: core.IntLit, Value: 2},
		},
	}

	result := typedList.String()
	assert.Contains(t, result, "[...]")
	assert.Contains(t, result, "[Int]")
}

func TestTypedTuple_String(t *testing.T) {
	typedTuple := TypedTuple{
		TypedExpr: TypedExpr{NodeID: 1, Type: "(Int, String)"},
		Elements: []TypedNode{
			&TypedLit{TypedExpr: TypedExpr{Type: "Int"}, Kind: core.IntLit, Value: 42},
			&TypedLit{TypedExpr: TypedExpr{Type: "String"}, Kind: core.StringLit, Value: "hello"},
		},
	}

	result := typedTuple.String()
	assert.Contains(t, result, "(...)")
	assert.Contains(t, result, "(Int, String)")
}

func TestTypedPatterns_String(t *testing.T) {
	tests := []struct {
		name     string
		pattern  TypedPattern
		expected string
	}{
		{
			name:     "var pattern",
			pattern:  TypedVarPattern{Name: "x", Type: "Int"},
			expected: "x",
		},
		{
			name:     "lit pattern",
			pattern:  TypedLitPattern{Value: 42},
			expected: "42",
		},
		{
			name: "constructor pattern",
			pattern: TypedConstructorPattern{
				Name: "Some",
				Args: []TypedPattern{TypedVarPattern{Name: "x"}},
			},
			expected: "Some",
		},
		{
			name:     "wildcard pattern",
			pattern:  TypedWildcardPattern{},
			expected: "_",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.pattern.String()
			assert.Contains(t, result, tt.expected)
		})
	}
}

func TestTypedTuplePattern_String(t *testing.T) {
	pattern := TypedTuplePattern{
		Elements: []TypedPattern{
			TypedVarPattern{Name: "x"},
			TypedVarPattern{Name: "y"},
		},
	}

	result := pattern.String()
	assert.Contains(t, result, "x")
	assert.Contains(t, result, "y")
	assert.Contains(t, result, "(")
	assert.Contains(t, result, ")")
}

func TestTypedListPattern_String(t *testing.T) {
	tests := []struct {
		name     string
		pattern  TypedListPattern
		contains []string
	}{
		{
			name: "simple list pattern",
			pattern: TypedListPattern{
				Elements: []TypedPattern{
					TypedVarPattern{Name: "x"},
					TypedVarPattern{Name: "y"},
				},
			},
			contains: []string{"x", "y", "[", "]"},
		},
		{
			name: "list pattern with tail",
			pattern: TypedListPattern{
				Elements: []TypedPattern{
					TypedVarPattern{Name: "head"},
				},
				Tail: func() *TypedPattern {
					var p TypedPattern = TypedVarPattern{Name: "rest"}
					return &p
				}(),
			},
			contains: []string{"head", "rest", "..."},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.pattern.String()
			for _, s := range tt.contains {
				assert.Contains(t, result, s)
			}
		})
	}
}

func TestFormatType(t *testing.T) {
	tests := []struct {
		name     string
		input    interface{}
		expected string
	}{
		{
			name:     "nil type",
			input:    nil,
			expected: "<unknown>",
		},
		{
			name:     "string type",
			input:    "Int",
			expected: "Int",
		},
		{
			name: "stringer type",
			input: &TypedVar{
				TypedExpr: TypedExpr{Type: "Int"},
				Name:      "x",
			},
			expected: "x",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := FormatType(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestPrintTypedProgram(t *testing.T) {
	decl1 := &TypedLit{
		TypedExpr: TypedExpr{NodeID: 1, Type: "Int"},
		Kind:      core.IntLit,
		Value:     42,
	}

	decl2 := &TypedVar{
		TypedExpr: TypedExpr{NodeID: 2, Type: "String"},
		Name:      "result",
	}

	typedProgram := &TypedProgram{
		Decls: []TypedNode{decl1, decl2},
	}

	result := PrintTypedProgram(typedProgram)
	assert.Contains(t, result, "42")
	assert.Contains(t, result, "result")
	// Should have newlines between declarations
	assert.True(t, strings.Count(result, "\n") >= 2)
}

func TestTypedMatchArm(t *testing.T) {
	pattern := TypedVarPattern{Name: "x", Type: "Int"}
	guard := &TypedLit{
		TypedExpr: TypedExpr{Type: "Bool"},
		Kind:      core.BoolLit,
		Value:     true,
	}
	body := &TypedLit{
		TypedExpr: TypedExpr{Type: "String"},
		Kind:      core.StringLit,
		Value:     "matched",
	}

	arm := TypedMatchArm{
		Pattern: pattern,
		Guard:   guard,
		Body:    body,
	}

	assert.NotNil(t, arm.Pattern)
	assert.NotNil(t, arm.Guard)
	assert.NotNil(t, arm.Body)
}

func TestTypedRecBinding(t *testing.T) {
	value := &TypedLambda{
		TypedExpr:  TypedExpr{Type: "Int -> Int"},
		Params:     []string{"x"},
		ParamTypes: []interface{}{"Int"},
		Body: &TypedVar{
			TypedExpr: TypedExpr{Type: "Int"},
			Name:      "x",
		},
	}

	binding := TypedRecBinding{
		Name:   "factorial",
		Scheme: "Int -> Int",
		Value:  value,
	}

	assert.Equal(t, "factorial", binding.Name)
	assert.Equal(t, "Int -> Int", binding.Scheme)
	assert.NotNil(t, binding.Value)
}

func TestAllTypedNodesImplementInterface(t *testing.T) {
	// Verify all typed node types implement the TypedNode interface
	var _ TypedNode = &TypedVar{}
	var _ TypedNode = &TypedLit{}
	var _ TypedNode = &TypedLambda{}
	var _ TypedNode = &TypedLet{}
	var _ TypedNode = &TypedLetRec{}
	var _ TypedNode = &TypedApp{}
	var _ TypedNode = &TypedIf{}
	var _ TypedNode = &TypedMatch{}
	var _ TypedNode = &TypedBinOp{}
	var _ TypedNode = &TypedUnOp{}
	var _ TypedNode = &TypedRecord{}
	var _ TypedNode = &TypedRecordAccess{}
	var _ TypedNode = &TypedList{}
	var _ TypedNode = &TypedTuple{}
}

func TestAllPatternsImplementInterface(t *testing.T) {
	// Verify all pattern types implement the TypedPattern interface
	var _ TypedPattern = TypedVarPattern{}
	var _ TypedPattern = TypedLitPattern{}
	var _ TypedPattern = TypedConstructorPattern{}
	var _ TypedPattern = TypedWildcardPattern{}
	var _ TypedPattern = TypedTuplePattern{}
	var _ TypedPattern = TypedListPattern{}
}

func TestTypedProgram_EmptyDecls(t *testing.T) {
	prog := &TypedProgram{Decls: []TypedNode{}}
	result := PrintTypedProgram(prog)
	assert.Equal(t, "", result)
}

func TestTypedConstructorPattern_WithMultipleArgs(t *testing.T) {
	pattern := TypedConstructorPattern{
		Name: "Cons",
		Args: []TypedPattern{
			TypedVarPattern{Name: "head"},
			TypedVarPattern{Name: "tail"},
		},
	}

	result := pattern.String()
	require.Contains(t, result, "Cons")
	assert.Contains(t, result, "head")
	assert.Contains(t, result, "tail")
}
