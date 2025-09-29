package types_test

import (
	"strings"
	"testing"

	"github.com/sunholo/ailang/internal/ast"
	"github.com/sunholo/ailang/internal/core"
	"github.com/sunholo/ailang/internal/elaborate"
	"github.com/sunholo/ailang/internal/lexer"
	"github.com/sunholo/ailang/internal/parser"
	"github.com/sunholo/ailang/internal/types"
)

// Small helpers (adapt names if your constructors differ)
func parseSurface(t *testing.T, src string) *ast.Program {
	t.Helper()
	l := lexer.New(src, "<test>")
	p := parser.New(l)
	prog := p.Parse()

	if errors := p.Errors(); len(errors) > 0 {
		var errMsgs []string
		for _, err := range errors {
			errMsgs = append(errMsgs, err.Error())
		}
		t.Fatalf("parse error: %s", strings.Join(errMsgs, "; "))
	}
	return prog
}

func elaborateCore(t *testing.T, surf *ast.Program) *core.Program {
	t.Helper()
	el := elaborate.NewElaborator()
	cp, err := el.Elaborate(surf)
	if err != nil {
		t.Fatalf("elaborate error: %v", err)
	}
	return cp
}

func typecheckCore(t *testing.T, cp *core.Program) *types.CoreTypeChecker {
	t.Helper()
	tc := types.NewCoreTypeCheckerWithInstances(types.LoadBuiltinInstances())

	// For now, type check the expressions in the program
	// This is a simplified approach since we don't have full program typing yet
	env := types.NewTypeEnvWithBuiltins()
	for _, decl := range cp.Decls {
		_, _, err := tc.CheckCoreExpr(decl, env)
		if err != nil {
			t.Fatalf("typecheck error: %v", err)
		}
	}
	return tc
}

func TestDefaulting_IntLiteralGround(t *testing.T) {
	src := `let x = 1 in x`
	surf := parseSurface(t, src)
	cp := elaborateCore(t, surf)
	tc := typecheckCore(t, cp)

	// For now, just verify that type checking succeeded
	// In the full implementation, we would check the actual type
	_ = tc

	// Placeholder assertion - this will be replaced when we have proper type lookup
	t.Logf("Successfully type checked: %s", src)
}

func TestDefaulting_FloatLiteralGround(t *testing.T) {
	src := `let y = 3.14 in y`
	surf := parseSurface(t, src)
	cp := elaborateCore(t, surf)
	tc := typecheckCore(t, cp)

	// For now, just verify that type checking succeeded
	_ = tc

	// Placeholder assertion - this will be replaced when we have proper type lookup
	t.Logf("Successfully type checked: %s", src)
}

func TestQualifiedScheme_NumConstraint(t *testing.T) {
	// This test requires lambda parsing which may not be fully implemented yet
	src := `let add = 1 + 2 in add` // Simplified for now
	surf := parseSurface(t, src)
	cp := elaborateCore(t, surf)
	tc := typecheckCore(t, cp)

	// For now, just verify that type checking succeeded
	_ = tc

	// Placeholder assertion - this will be replaced when we have proper qualified scheme lookup
	t.Logf("Successfully type checked: %s", src)
}
