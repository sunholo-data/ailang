// Package golang provides Go code generation from AILANG Core AST.
package golang

import (
	"strings"
	"testing"

	"github.com/sunholo/ailang/internal/core"
)

// TestNeedsBoolAssertion tests the needsBoolAssertion helper function.
// M-CODEGEN-BOOL-ASSERTIONS: Verifies detection of expressions needing .(bool) assertions.
func TestNeedsBoolAssertion(t *testing.T) {
	g := &Generator{
		typedLocalVars:    make(map[string]string),
		currentFuncParams: make(map[string]string),
	}

	tests := []struct {
		name     string
		expr     core.CoreExpr
		setup    func()
		expected bool
	}{
		{
			name: "DictApp with eq method",
			expr: &core.DictApp{
				Dict:   &core.DictRef{ClassName: "Eq", TypeName: "Color"},
				Method: "eq",
				Args:   []core.CoreExpr{},
			},
			expected: true,
		},
		{
			name: "DictApp with neq method",
			expr: &core.DictApp{
				Dict:   &core.DictRef{ClassName: "Eq", TypeName: "Color"},
				Method: "neq",
				Args:   []core.CoreExpr{},
			},
			expected: true,
		},
		{
			name: "DictApp with lt method",
			expr: &core.DictApp{
				Dict:   &core.DictRef{ClassName: "Ord", TypeName: "Int"},
				Method: "lt",
				Args:   []core.CoreExpr{},
			},
			expected: true,
		},
		{
			name: "DictApp with add method (not bool)",
			expr: &core.DictApp{
				Dict:   &core.DictRef{ClassName: "Num", TypeName: "Int"},
				Method: "add",
				Args:   []core.CoreExpr{},
			},
			expected: false,
		},
		{
			name: "Var not in typedLocalVars",
			expr: &core.Var{Name: "sameColor"},
			setup: func() {
				g.typedLocalVars = make(map[string]string)
				g.currentFuncParams = make(map[string]string)
			},
			expected: true,
		},
		{
			name: "Var in typedLocalVars",
			expr: &core.Var{Name: "typedBool"},
			setup: func() {
				g.typedLocalVars = map[string]string{"typedBool": "bool"}
				g.currentFuncParams = make(map[string]string)
			},
			expected: false,
		},
		{
			name: "Var as function param with interface{}",
			expr: &core.Var{Name: "x"},
			setup: func() {
				g.typedLocalVars = make(map[string]string)
				g.currentFuncParams = map[string]string{"x": "interface{}"}
			},
			expected: true,
		},
		{
			name: "Var as function param with bool",
			expr: &core.Var{Name: "x"},
			setup: func() {
				g.typedLocalVars = make(map[string]string)
				g.currentFuncParams = map[string]string{"x": "bool"}
			},
			expected: false,
		},
		{
			name: "Literal (never needs assertion)",
			expr: &core.Lit{Value: true},
			setup: func() {
				g.typedLocalVars = make(map[string]string)
				g.currentFuncParams = make(map[string]string)
			},
			expected: false,
		},
		{
			name: "BinOp comparison (handled by DictApp)",
			expr: &core.BinOp{
				Op:    "==",
				Left:  &core.Var{Name: "a"},
				Right: &core.Var{Name: "b"},
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.setup != nil {
				tt.setup()
			}
			result := g.needsBoolAssertion(tt.expr)
			if result != tt.expected {
				t.Errorf("needsBoolAssertion() = %v, want %v", result, tt.expected)
			}
		})
	}
}

// TestGenerateExprWithBoolAssertion tests the bool assertion wrapper.
// M-CODEGEN-BOOL-ASSERTIONS: Verifies .(bool) is added when needed.
func TestGenerateExprWithBoolAssertion(t *testing.T) {
	g := New("test")
	g.typedLocalVars = make(map[string]string)
	g.currentFuncParams = make(map[string]string)

	tests := []struct {
		name     string
		expr     core.CoreExpr
		contains string
	}{
		{
			name: "DictApp eq adds .(bool)",
			expr: &core.DictApp{
				Dict:   &core.DictRef{ClassName: "Eq", TypeName: "Color"},
				Method: "eq",
				Args:   []core.CoreExpr{&core.Lit{Value: int64(1)}, &core.Lit{Value: int64(2)}},
			},
			contains: ".(bool)",
		},
		{
			name: "Var adds .(bool) when not typed",
			expr: &core.Var{Name: "sameColor"},
			contains: ".(bool)",
		},
		{
			name: "Literal does not add .(bool)",
			expr: &core.Lit{Value: true},
			contains: "true",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g.buf.Reset()
			err := g.generateExprWithBoolAssertion(tt.expr)
			if err != nil {
				t.Fatalf("generateExprWithBoolAssertion() error = %v", err)
			}
			output := g.buf.String()
			if !strings.Contains(output, tt.contains) {
				t.Errorf("output = %q, want to contain %q", output, tt.contains)
			}
		})
	}
}

// TestLogicalOperatorsBoolAssertion tests && and || with interface{} operands.
// M-CODEGEN-BOOL-ASSERTIONS: Verifies both operands get .(bool) when needed.
func TestLogicalOperatorsBoolAssertion(t *testing.T) {
	g := New("test")
	g.typedLocalVars = make(map[string]string)
	g.currentFuncParams = make(map[string]string)

	tests := []struct {
		name      string
		op        string
		wantCount int // number of .(bool) assertions expected
	}{
		{
			name:      "&& operator",
			op:        "&&",
			wantCount: 2, // Both operands need .(bool)
		},
		{
			name:      "|| operator",
			op:        "||",
			wantCount: 2, // Both operands need .(bool)
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g.buf.Reset()
			expr := &core.BinOp{
				Op:    tt.op,
				Left:  &core.Var{Name: "left"},
				Right: &core.Var{Name: "right"},
			}
			err := g.generateBinOp(expr)
			if err != nil {
				t.Fatalf("generateBinOp() error = %v", err)
			}
			output := g.buf.String()
			count := strings.Count(output, ".(bool)")
			if count != tt.wantCount {
				t.Errorf("output = %q, has %d .(bool), want %d", output, count, tt.wantCount)
			}
		})
	}
}

// TestLogicalOperatorsNoAssertionForLiterals tests that literals don't get .(bool).
// M-CODEGEN-BOOL-ASSERTIONS: Verifies no unnecessary assertions for concrete types.
func TestLogicalOperatorsNoAssertionForLiterals(t *testing.T) {
	g := New("test")
	g.typedLocalVars = make(map[string]string)
	g.currentFuncParams = make(map[string]string)

	g.buf.Reset()
	expr := &core.BinOp{
		Op:    "&&",
		Left:  &core.Lit{Value: true},
		Right: &core.Lit{Value: false},
	}
	err := g.generateBinOp(expr)
	if err != nil {
		t.Fatalf("generateBinOp() error = %v", err)
	}
	output := g.buf.String()

	// Literals should not have .(bool)
	if strings.Contains(output, ".(bool)") {
		t.Errorf("output = %q, should not contain .(bool) for literals", output)
	}
	if !strings.Contains(output, "true") || !strings.Contains(output, "false") {
		t.Errorf("output = %q, should contain both true and false", output)
	}
}
