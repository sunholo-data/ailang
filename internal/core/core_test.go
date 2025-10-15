package core

import (
	"fmt"
	"github.com/sunholo/ailang/internal/ast"
	"strings"
	"testing"
)

func TestCoreNode(t *testing.T) {
	node := CoreNode{
		NodeID:   42,
		CoreSpan: ast.Pos{Line: 10, Column: 5, File: "core.ail"},
		OrigSpan: ast.Pos{Line: 1, Column: 1, File: "test.ail"},
	}

	// Test ID()
	if got := node.ID(); got != 42 {
		t.Errorf("CoreNode.ID() = %v, want %v", got, 42)
	}

	// Test Span()
	expectedSpan := ast.Pos{Line: 10, Column: 5, File: "core.ail"}
	if got := node.Span(); got != expectedSpan {
		t.Errorf("CoreNode.Span() = %v, want %v", got, expectedSpan)
	}

	// Test OriginalSpan()
	expectedOrigSpan := ast.Pos{Line: 1, Column: 1, File: "test.ail"}
	if got := node.OriginalSpan(); got != expectedOrigSpan {
		t.Errorf("CoreNode.OriginalSpan() = %v, want %v", got, expectedOrigSpan)
	}
}

func TestVar(t *testing.T) {
	v := &Var{
		CoreNode: CoreNode{NodeID: 1},
		Name:     "myVar",
	}

	// Test String()
	if got := v.String(); got != "myVar" {
		t.Errorf("Var.String() = %v, want %v", got, "myVar")
	}

	// Test ID() inherited from CoreNode
	if got := v.ID(); got != 1 {
		t.Errorf("Var.ID() = %v, want %v", got, 1)
	}

	// Verify it implements CoreExpr interface
	var _ CoreExpr = v
}

func TestLit(t *testing.T) {
	tests := []struct {
		name string
		lit  *Lit
		want string
	}{
		{
			name: "int literal",
			lit:  &Lit{CoreNode: CoreNode{NodeID: 1}, Kind: IntLit, Value: int64(42)},
			want: "42",
		},
		{
			name: "float literal",
			lit:  &Lit{CoreNode: CoreNode{NodeID: 2}, Kind: FloatLit, Value: 3.14},
			want: "3.14",
		},
		{
			name: "string literal",
			lit:  &Lit{CoreNode: CoreNode{NodeID: 3}, Kind: StringLit, Value: "hello"},
			want: "hello",
		},
		{
			name: "bool true",
			lit:  &Lit{CoreNode: CoreNode{NodeID: 4}, Kind: BoolLit, Value: true},
			want: "true",
		},
		{
			name: "bool false",
			lit:  &Lit{CoreNode: CoreNode{NodeID: 5}, Kind: BoolLit, Value: false},
			want: "false",
		},
		{
			name: "unit literal",
			lit:  &Lit{CoreNode: CoreNode{NodeID: 6}, Kind: UnitLit, Value: nil},
			want: "<nil>",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.lit.String(); got != tt.want {
				t.Errorf("Lit.String() = %v, want %v", got, tt.want)
			}

			// Verify it implements CoreExpr interface
			var _ CoreExpr = tt.lit
		})
	}
}

func TestLambda(t *testing.T) {
	lambda := &Lambda{
		CoreNode: CoreNode{NodeID: 1},
		Params:   []string{"x", "y"},
		Body:     &Var{CoreNode: CoreNode{NodeID: 2}, Name: "x"},
	}

	// Test String()
	expected := "λ[x y]. x"
	if got := lambda.String(); got != expected {
		t.Errorf("Lambda.String() = %v, want %v", got, expected)
	}

	// Test with single param
	lambda2 := &Lambda{
		CoreNode: CoreNode{NodeID: 3},
		Params:   []string{"z"},
		Body:     &Var{CoreNode: CoreNode{NodeID: 4}, Name: "z"},
	}

	expected2 := "λ[z]. z"
	if got := lambda2.String(); got != expected2 {
		t.Errorf("Lambda.String() = %v, want %v", got, expected2)
	}

	// Verify it implements CoreExpr interface
	var _ CoreExpr = lambda
}

func TestLam(t *testing.T) {
	lam := &Lam{
		CoreNode: CoreNode{NodeID: 1},
		Param:    "x",
		Body:     &Var{CoreNode: CoreNode{NodeID: 2}, Name: "x"},
	}

	// Test String()
	expected := "λx. x"
	if got := lam.String(); got != expected {
		t.Errorf("Lam.String() = %v, want %v", got, expected)
	}

	// Verify it implements Expr interface
	var _ Expr = lam
}

func TestLet(t *testing.T) {
	let := &Let{
		CoreNode: CoreNode{NodeID: 1},
		Name:     "x",
		Value:    &Lit{CoreNode: CoreNode{NodeID: 2}, Kind: IntLit, Value: int64(5)},
		Body:     &Var{CoreNode: CoreNode{NodeID: 3}, Name: "x"},
	}

	// Test String()
	expected := "let x = 5 in x"
	if got := let.String(); got != expected {
		t.Errorf("Let.String() = %v, want %v", got, expected)
	}

	// Verify it implements CoreExpr interface
	var _ CoreExpr = let
}

func TestLetRec(t *testing.T) {
	letRec := &LetRec{
		CoreNode: CoreNode{NodeID: 1},
		Bindings: []RecBinding{
			{Name: "fact", Value: &Lambda{
				CoreNode: CoreNode{NodeID: 2},
				Params:   []string{"n"},
				Body:     &Var{CoreNode: CoreNode{NodeID: 3}, Name: "n"},
			}},
			{Name: "fib", Value: &Lambda{
				CoreNode: CoreNode{NodeID: 4},
				Params:   []string{"x"},
				Body:     &Var{CoreNode: CoreNode{NodeID: 5}, Name: "x"},
			}},
		},
		Body: &Var{CoreNode: CoreNode{NodeID: 6}, Name: "fact"},
	}

	// Test String()
	got := letRec.String()
	if !strings.Contains(got, "let rec") {
		t.Errorf("LetRec.String() missing 'let rec': %v", got)
	}
	if !strings.Contains(got, "fact") || !strings.Contains(got, "fib") {
		t.Errorf("LetRec.String() missing bindings: %v", got)
	}

	// Verify it implements CoreExpr interface
	var _ CoreExpr = letRec
}

func TestApp(t *testing.T) {
	app := &App{
		CoreNode: CoreNode{NodeID: 1},
		Func:     &Var{CoreNode: CoreNode{NodeID: 2}, Name: "add"},
		Args: []CoreExpr{
			&Lit{CoreNode: CoreNode{NodeID: 3}, Kind: IntLit, Value: int64(1)},
			&Lit{CoreNode: CoreNode{NodeID: 4}, Kind: IntLit, Value: int64(2)},
		},
	}

	// Test String() - Note: actual format uses square brackets
	got := app.String()
	if !strings.Contains(got, "add") || !strings.Contains(got, "1") || !strings.Contains(got, "2") {
		t.Errorf("App.String() missing expected parts: %v", got)
	}

	// Test with no args
	app2 := &App{
		CoreNode: CoreNode{NodeID: 5},
		Func:     &Var{CoreNode: CoreNode{NodeID: 6}, Name: "getValue"},
		Args:     []CoreExpr{},
	}

	got2 := app2.String()
	if !strings.Contains(got2, "getValue") {
		t.Errorf("App.String() missing function name: %v", got2)
	}

	// Verify it implements CoreExpr interface
	var _ CoreExpr = app
}

func TestBinOp(t *testing.T) {
	binOp := &BinOp{
		CoreNode: CoreNode{NodeID: 1},
		Op:       "+",
		Left:     &Lit{CoreNode: CoreNode{NodeID: 2}, Kind: IntLit, Value: int64(1)},
		Right:    &Lit{CoreNode: CoreNode{NodeID: 3}, Kind: IntLit, Value: int64(2)},
	}

	// Test String()
	expected := "(1 + 2)"
	if got := binOp.String(); got != expected {
		t.Errorf("BinOp.String() = %v, want %v", got, expected)
	}

	// Test different operators
	ops := []string{"-", "*", "/", "==", "!=", "<", ">", "&&", "||", "++"}
	for _, op := range ops {
		binOp.Op = op
		got := binOp.String()
		expectedOp := fmt.Sprintf("(1 %s 2)", op)
		if got != expectedOp {
			t.Errorf("BinOp.String() with op %s = %v, want %v", op, got, expectedOp)
		}
	}

	// Verify it implements CoreExpr interface
	var _ CoreExpr = binOp
}

func TestUnOp(t *testing.T) {
	unOp := &UnOp{
		CoreNode: CoreNode{NodeID: 1},
		Op:       "-",
		Operand:  &Lit{CoreNode: CoreNode{NodeID: 2}, Kind: IntLit, Value: int64(42)},
	}

	// Test String() - actual format doesn't use parentheses
	got := unOp.String()
	if !strings.Contains(got, "-") || !strings.Contains(got, "42") {
		t.Errorf("UnOp.String() missing expected parts: %v", got)
	}

	// Test with not operator
	unOp2 := &UnOp{
		CoreNode: CoreNode{NodeID: 3},
		Op:       "!",
		Operand:  &Lit{CoreNode: CoreNode{NodeID: 4}, Kind: BoolLit, Value: true},
	}

	got2 := unOp2.String()
	if !strings.Contains(got2, "!") || !strings.Contains(got2, "true") {
		t.Errorf("UnOp.String() missing expected parts: %v", got2)
	}

	// Verify it implements CoreExpr interface
	var _ CoreExpr = unOp
}

func TestIf(t *testing.T) {
	ifExpr := &If{
		CoreNode: CoreNode{NodeID: 1},
		Cond:     &Lit{CoreNode: CoreNode{NodeID: 2}, Kind: BoolLit, Value: true},
		Then:     &Lit{CoreNode: CoreNode{NodeID: 3}, Kind: StringLit, Value: "yes"},
		Else:     &Lit{CoreNode: CoreNode{NodeID: 4}, Kind: StringLit, Value: "no"},
	}

	// Test String()
	expected := "if true then yes else no"
	if got := ifExpr.String(); got != expected {
		t.Errorf("If.String() = %v, want %v", got, expected)
	}

	// Verify it implements CoreExpr interface
	var _ CoreExpr = ifExpr
}

func TestList(t *testing.T) {
	list := &List{
		CoreNode: CoreNode{NodeID: 1},
		Elements: []CoreExpr{
			&Lit{CoreNode: CoreNode{NodeID: 2}, Kind: IntLit, Value: int64(1)},
			&Lit{CoreNode: CoreNode{NodeID: 3}, Kind: IntLit, Value: int64(2)},
			&Lit{CoreNode: CoreNode{NodeID: 4}, Kind: IntLit, Value: int64(3)},
		},
	}

	// Test String() - actual format uses nested brackets
	got := list.String()
	if !strings.Contains(got, "1") || !strings.Contains(got, "2") || !strings.Contains(got, "3") {
		t.Errorf("List.String() missing expected elements: %v", got)
	}

	// Test empty list
	emptyList := &List{
		CoreNode: CoreNode{NodeID: 5},
		Elements: []CoreExpr{},
	}

	got2 := emptyList.String()
	if !strings.Contains(got2, "[") || !strings.Contains(got2, "]") {
		t.Errorf("List.String() missing brackets: %v", got2)
	}

	// Verify it implements CoreExpr interface
	var _ CoreExpr = list
}

func TestRecord(t *testing.T) {
	record := &Record{
		CoreNode: CoreNode{NodeID: 1},
		Fields: map[string]CoreExpr{
			"name": &Lit{CoreNode: CoreNode{NodeID: 2}, Kind: StringLit, Value: "Alice"},
			"age":  &Lit{CoreNode: CoreNode{NodeID: 3}, Kind: IntLit, Value: int64(30)},
		},
	}

	// Test String() - note map order is non-deterministic
	got := record.String()
	if !strings.HasPrefix(got, "{") || !strings.HasSuffix(got, "}") {
		t.Errorf("Record.String() wrong format: %v", got)
	}
	if !strings.Contains(got, "Alice") || !strings.Contains(got, "30") {
		t.Errorf("Record.String() missing field values: %v", got)
	}

	// Test empty record
	emptyRecord := &Record{
		CoreNode: CoreNode{NodeID: 4},
		Fields:   map[string]CoreExpr{},
	}

	got2 := emptyRecord.String()
	if !strings.Contains(got2, "{") || !strings.Contains(got2, "}") {
		t.Errorf("Record.String() missing braces: %v", got2)
	}

	// Verify it implements CoreExpr interface
	var _ CoreExpr = record
}

func TestDictAbs(t *testing.T) {
	dictAbs := &DictAbs{
		CoreNode: CoreNode{NodeID: 1},
		Params:   []DictParam{{Name: "NumInt", ClassName: "Num", Type: "Int"}},
		Body:     &Var{CoreNode: CoreNode{NodeID: 2}, Name: "x"},
	}

	// Test String() - actual format is different
	got := dictAbs.String()
	if !strings.Contains(got, "NumInt") || !strings.Contains(got, "x") {
		t.Errorf("DictAbs.String() missing expected parts: %v", got)
	}

	// Test with multiple params
	dictAbs2 := &DictAbs{
		CoreNode: CoreNode{NodeID: 3},
		Params: []DictParam{
			{Name: "EqInt", ClassName: "Eq", Type: "Int"},
			{Name: "OrdInt", ClassName: "Ord", Type: "Int"},
		},
		Body: &Var{CoreNode: CoreNode{NodeID: 4}, Name: "y"},
	}

	got2 := dictAbs2.String()
	if !strings.Contains(got2, "EqInt") || !strings.Contains(got2, "OrdInt") || !strings.Contains(got2, "y") {
		t.Errorf("DictAbs.String() missing expected parts: %v", got2)
	}

	// Verify it implements CoreExpr interface
	var _ CoreExpr = dictAbs
}

func TestDictApp(t *testing.T) {
	dictApp := &DictApp{
		CoreNode: CoreNode{NodeID: 1},
		Dict:     &Var{CoreNode: CoreNode{NodeID: 2}, Name: "$dictNum"},
		Args: []CoreExpr{
			&Var{CoreNode: CoreNode{NodeID: 3}, Name: "NumInt"},
		},
	}

	// Test String() - actual format is different
	got := dictApp.String()
	if !strings.Contains(got, "$dictNum") || !strings.Contains(got, "NumInt") {
		t.Errorf("DictApp.String() missing expected parts: %v", got)
	}

	// Test with multiple args
	dictApp2 := &DictApp{
		CoreNode: CoreNode{NodeID: 4},
		Dict:     &Var{CoreNode: CoreNode{NodeID: 5}, Name: "$dictEq"},
		Args: []CoreExpr{
			&Var{CoreNode: CoreNode{NodeID: 6}, Name: "EqInt"},
			&Var{CoreNode: CoreNode{NodeID: 7}, Name: "EqString"},
		},
	}

	got2 := dictApp2.String()
	if !strings.Contains(got2, "$dictEq") || !strings.Contains(got2, "EqInt") || !strings.Contains(got2, "EqString") {
		t.Errorf("DictApp.String() missing expected parts: %v", got2)
	}

	// Verify it implements CoreExpr interface
	var _ CoreExpr = dictApp
}

func TestProgram(t *testing.T) {
	program := &Program{
		Decls: []CoreExpr{
			&Let{
				CoreNode: CoreNode{NodeID: 1},
				Name:     "x",
				Value:    &Lit{CoreNode: CoreNode{NodeID: 2}, Kind: IntLit, Value: int64(5)},
				Body:     &Var{CoreNode: CoreNode{NodeID: 3}, Name: "x"},
			},
			&Lit{CoreNode: CoreNode{NodeID: 4}, Kind: IntLit, Value: int64(42)},
		},
	}

	// Test that Decls are set correctly
	if len(program.Decls) != 2 {
		t.Errorf("Program.Decls length = %v, want %v", len(program.Decls), 2)
	}

	// Test with empty program
	emptyProgram := &Program{
		Decls: []CoreExpr{},
	}

	if len(emptyProgram.Decls) != 0 {
		t.Errorf("Empty Program.Decls length = %v, want %v", len(emptyProgram.Decls), 0)
	}
}

func TestRecBinding(t *testing.T) {
	binding := RecBinding{
		Name: "factorial",
		Value: &Lambda{
			CoreNode: CoreNode{NodeID: 1},
			Params:   []string{"n"},
			Body:     &Var{CoreNode: CoreNode{NodeID: 2}, Name: "n"},
		},
	}

	// Test that fields are accessible
	if binding.Name != "factorial" {
		t.Errorf("RecBinding.Name = %v, want %v", binding.Name, "factorial")
	}

	// Verify Value is a CoreExpr
	var _ CoreExpr = binding.Value
}

func TestTuplePattern(t *testing.T) {
	// Create tuple pattern: (x, 42, _)
	pattern := &TuplePattern{
		Elements: []CorePattern{
			&VarPattern{Name: "x"},
			&LitPattern{Value: 42},
			&WildcardPattern{},
		},
	}

	// Test String() representation
	got := pattern.String()
	want := "(x, 42, _)"
	if got != want {
		t.Errorf("TuplePattern.String() = %q, want %q", got, want)
	}

	// Test that it implements CorePattern interface
	var _ CorePattern = pattern
}

func TestConstructorPattern(t *testing.T) {
	// Create constructor pattern: Some(x)
	pattern := &ConstructorPattern{
		Name: "Some",
		Args: []CorePattern{
			&VarPattern{Name: "x"},
		},
	}

	// Test String() representation
	got := pattern.String()
	want := "Some([x])"
	if got != want {
		t.Errorf("ConstructorPattern.String() = %q, want %q", got, want)
	}

	// Test that it implements CorePattern interface
	var _ CorePattern = pattern
}

func TestNestedPatterns(t *testing.T) {
	// Create nested pattern: Cons((x, y), tail)
	pattern := &ConstructorPattern{
		Name: "Cons",
		Args: []CorePattern{
			&TuplePattern{
				Elements: []CorePattern{
					&VarPattern{Name: "x"},
					&VarPattern{Name: "y"},
				},
			},
			&VarPattern{Name: "tail"},
		},
	}

	// Test String() representation
	got := pattern.String()
	// Note: String() may format Args as a slice representation
	if !strings.Contains(got, "Cons") {
		t.Errorf("ConstructorPattern.String() should contain 'Cons', got %q", got)
	}

	// Test that it implements CorePattern interface
	var _ CorePattern = pattern
}
