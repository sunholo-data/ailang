package pipeline

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/sunholo-data/ailang/internal/core"
)

// TestContractExpressionsFullyLowered verifies that contract expressions
// (requires/ensures clauses) are fully processed by all pipeline passes.
// This is a regression test for M-CONTRACT-OPLOWERING-FIX: previously,
// the type checker, dictionary elaboration, and OpLowering all independently
// skipped contract expressions, leaving raw Intrinsic/BinOp nodes that
// failed at runtime.
func TestContractExpressionsFullyLowered(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "ailang-contract-pipeline-test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Module with int and float comparison contracts
	content := `module contract_test

export func absVal(x: int) -> int
requires { x >= -1000 }
ensures { result >= 0 }
= if x >= 0 then x else 0 - x

export func clamp01(x: float) -> float
ensures { result >= 0.0 && result <= 1.0 }
= if x >= 1.0 then 1.0 else if x <= 0.0 then 0.0 else x

export func main() -> () ! {IO} = {
  println(show(absVal(5)))
}
`
	filePath := filepath.Join(tempDir, "contract_test.ail")
	if err := os.WriteFile(filePath, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	// Save and restore working directory (pipeline uses CWD for module resolution)
	origDir, _ := os.Getwd()
	os.Chdir(tempDir)
	defer os.Chdir(origDir)

	cfg := Config{
		Mode:         ModeCheck,
		RelaxModules: true,
	}
	src := Source{
		Code:     content,
		Filename: "contract_test.ail",
	}

	result, err := RunWithContext(context.Background(), cfg, src)
	if err != nil {
		t.Fatalf("pipeline failed: %v", err)
	}

	// Find the compiled module
	if result.Modules == nil {
		t.Fatal("no modules in result")
	}

	var foundModule *core.Program
	for _, mod := range result.Modules {
		if mod.Core != nil && mod.Core.Meta != nil {
			foundModule = mod.Core
			break
		}
	}
	if foundModule == nil {
		t.Fatal("no module with Meta found in result")
	}

	// Verify all contract expressions are fully lowered
	for funcName, meta := range foundModule.Meta {
		for i, contract := range meta.Contracts {
			if contract.Expr == nil {
				continue
			}
			intrinsics := findNodes(contract.Expr, func(e core.CoreExpr) bool {
				_, ok := e.(*core.Intrinsic)
				return ok
			})
			if len(intrinsics) > 0 {
				t.Errorf("contract %s[%d] (%s) has %d un-lowered Intrinsic nodes — OpLowering missed contract expressions",
					funcName, i, contract.Kind.String(), len(intrinsics))
			}

			binops := findNodes(contract.Expr, func(e core.CoreExpr) bool {
				_, ok := e.(*core.BinOp)
				return ok
			})
			if len(binops) > 0 {
				t.Errorf("contract %s[%d] (%s) has %d un-elaborated BinOp nodes — dictionary elaboration missed contract expressions",
					funcName, i, contract.Kind.String(), len(binops))
			}
		}
	}
}

// findNodes walks a Core expression tree and returns all nodes matching the predicate.
func findNodes(expr core.CoreExpr, pred func(core.CoreExpr) bool) []core.CoreExpr {
	var results []core.CoreExpr
	walkExpr(expr, func(e core.CoreExpr) {
		if pred(e) {
			results = append(results, e)
		}
	})
	return results
}

// walkExpr recursively visits all nodes in a Core expression tree.
func walkExpr(expr core.CoreExpr, fn func(core.CoreExpr)) {
	if expr == nil {
		return
	}
	fn(expr)

	switch e := expr.(type) {
	case *core.Let:
		walkExpr(e.Value, fn)
		walkExpr(e.Body, fn)
	case *core.LetRec:
		for _, b := range e.Bindings {
			walkExpr(b.Value, fn)
		}
		walkExpr(e.Body, fn)
	case *core.Lambda:
		walkExpr(e.Body, fn)
	case *core.App:
		walkExpr(e.Func, fn)
		for _, arg := range e.Args {
			walkExpr(arg, fn)
		}
	case *core.If:
		walkExpr(e.Cond, fn)
		walkExpr(e.Then, fn)
		walkExpr(e.Else, fn)
	case *core.Match:
		walkExpr(e.Scrutinee, fn)
		for _, arm := range e.Arms {
			walkExpr(arm.Guard, fn)
			walkExpr(arm.Body, fn)
		}
	case *core.Intrinsic:
		for _, arg := range e.Args {
			walkExpr(arg, fn)
		}
	case *core.BinOp:
		walkExpr(e.Left, fn)
		walkExpr(e.Right, fn)
	case *core.UnOp:
		walkExpr(e.Operand, fn)
	case *core.Record:
		for _, v := range e.Fields {
			walkExpr(v, fn)
		}
	case *core.RecordAccess:
		walkExpr(e.Record, fn)
	case *core.RecordUpdate:
		walkExpr(e.Base, fn)
		for _, v := range e.Updates {
			walkExpr(v, fn)
		}
	case *core.List:
		for _, elem := range e.Elements {
			walkExpr(elem, fn)
		}
	case *core.Tuple:
		for _, elem := range e.Elements {
			walkExpr(elem, fn)
		}
	case *core.DictApp:
		walkExpr(e.Dict, fn)
		for _, arg := range e.Args {
			walkExpr(arg, fn)
		}
	case *core.DictAbs:
		walkExpr(e.Body, fn)
		// Leaf nodes: Var, VarGlobal, Lit, DictRef — no children
	}
}
