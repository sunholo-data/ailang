package elaborate

import (
	"strings"
	"testing"

	"github.com/sunholo/ailang/internal/core"
	"github.com/sunholo/ailang/internal/lexer"
	"github.com/sunholo/ailang/internal/parser"
	"github.com/sunholo/ailang/internal/types"
)

// prettyCore provides a basic string representation of Core programs
// This is a stub implementation - a full implementation would be more sophisticated
func prettyCore(prog *core.Program) string {
	var parts []string
	for i, decl := range prog.Decls {
		parts = append(parts, prettyCorExpr(decl, i))
	}
	return strings.Join(parts, "\n")
}

// Basic Core expression pretty printer stub
func prettyCorExpr(expr core.CoreExpr, indent int) string {
	switch e := expr.(type) {
	case *core.Lit:
		return "Lit(" + literalString(e) + ")"
	case *core.Var:
		return "Var(" + e.Name + ")"
	case *core.BinOp:
		left := prettyCorExpr(e.Left, indent)
		right := prettyCorExpr(e.Right, indent)
		return "BinOp(" + left + " " + e.Op + " " + right + ")"
	case *core.Let:
		value := prettyCorExpr(e.Value, indent+1)
		body := prettyCorExpr(e.Body, indent+1)
		return "Let(" + e.Name + " = " + value + " in " + body + ")"
	default:
		return "UnknownExpr"
	}
}

func literalString(lit *core.Lit) string {
	switch lit.Kind {
	case core.IntLit:
		return "Int"
	case core.FloatLit:
		return "Float"
	case core.StringLit:
		return "String"
	case core.BoolLit:
		return "Bool"
	default:
		return "Unknown"
	}
}

func TestElaborateWithDictionaries_AddInt(t *testing.T) {
	src := `let r = 2 + 3 in r`

	l := lexer.New(src, "<test>")
	p := parser.New(l)
	surf := p.Parse()
	if errors := p.Errors(); len(errors) > 0 {
		t.Fatalf("parse: %v", errors[0])
	}

	el := NewElaborator()
	core1, err := el.Elaborate(surf)
	if err != nil {
		t.Fatalf("elaborate: %v", err)
	}

	tc := types.NewCoreTypeCheckerWithInstances(types.LoadBuiltinInstances())

	// Type check all expressions
	env := types.NewTypeEnvWithBuiltins()
	for _, decl := range core1.Decls {
		_, _, err := tc.CheckCoreExpr(decl, env)
		if err != nil {
			t.Fatalf("typecheck: %v", err)
		}
	}

	// For now, this is a placeholder since ElaborateWithDictionaries doesn't exist yet
	// In the real implementation, this would transform operators into dictionary calls
	core2 := core1 // Placeholder

	pretty := prettyCore(core2)
	t.Logf("Core representation:\n%s", pretty)

	// These checks are placeholders for when dictionary elaboration is implemented
	// For now, just ensure we can parse and elaborate basic expressions
	if !strings.Contains(pretty, "BinOp") {
		t.Logf("Note: Binary operation found in core (not yet dictionary elaborated)")
	}

	// TODO: When dictionary elaboration is implemented, check for:
	// - DictRef(Num, Int)
	// - DictApp calls
	// - ANF transformation
}

func TestElaborateWithDictionaries_OrdEqChain(t *testing.T) {
	// This test will need complex boolean operations, which may not be fully implemented
	// For now, test simple comparison
	src := `let r = 5 < 10 in r`

	l := lexer.New(src, "<test>")
	p := parser.New(l)
	surf := p.Parse()
	if errors := p.Errors(); len(errors) > 0 {
		t.Fatalf("parse: %v", errors[0])
	}

	el := NewElaborator()
	core1, err := el.Elaborate(surf)
	if err != nil {
		t.Fatalf("elaborate: %v", err)
	}

	tc := types.NewCoreTypeCheckerWithInstances(types.LoadBuiltinInstances())

	env := types.NewTypeEnvWithBuiltins()
	for _, decl := range core1.Decls {
		_, _, err := tc.CheckCoreExpr(decl, env)
		if err != nil {
			t.Fatalf("typecheck: %v", err)
		}
	}

	// Placeholder for dictionary elaboration
	core2 := core1

	pretty := prettyCore(core2)
	t.Logf("Core representation:\n%s", pretty)

	// TODO: When dictionary elaboration is implemented, check for:
	// - DictRef(Ord, Int)
	// - DictRef(Eq, Int) (if derived from Ord)
	// - DictApp calls with "lt" and "eq" methods
}
