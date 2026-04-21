package bytecode_golden_test

// M-LOWER-FIX M1_PROBE: lowers each of the 4 problem .ail files via the same
// pipeline.Run + lower.LowerProgram path the parity gate uses, and dumps the
// resulting stmt.Program as a string. The dump verifies (or refutes) the
// hypotheses in design_docs/planned/v0_11_0/m-lower-fix.md §3 — in particular
// Bug C, where we suspect factorial/sumList hit the polymorphic
// `_dict_*` fallback in lowerDictApp.
//
// Run with:
//   go test -v -run TestLowerProbe ./tests/golden/bytecode/
//
// The test never fails — it always passes and emits its findings via t.Logf
// so the human can read them in the output.

import (
	"fmt"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"github.com/sunholo-data/ailang/internal/ast"
	"github.com/sunholo-data/ailang/internal/gen/lower"
	"github.com/sunholo-data/ailang/internal/gen/stmt"
	"github.com/sunholo-data/ailang/internal/pipeline"
)

func TestLowerProbe(t *testing.T) {
	files := []string{
		"tuples.ail",
		"match_patterns.ail",
		"arithmetic.ail",
		"functions.ail",
		"lists.ail",
		"string_ops.ail",
	}
	for _, fname := range files {
		fname := fname
		t.Run(strings.TrimSuffix(fname, ".ail"), func(t *testing.T) {
			prog, err := probeLowerFile(fname)
			if err != nil {
				t.Logf("=== %s: LOWER FAILED ===\n%v", fname, err)
				return
			}
			t.Logf("=== %s: lowered to %d funcs ===", fname, len(prog.FuncDecls))
			for _, fd := range prog.FuncDecls {
				t.Logf("--- func %s (exported=%v, %d params) ---",
					fd.Name, fd.Exported, len(fd.Params))
				t.Logf("Body:\n%s", dumpStmts(fd.Body, "  "))
				if fd.Return != nil {
					t.Logf("Return: %s", dumpExpr(fd.Return))
				} else {
					t.Logf("Return: <nil>")
				}
				flag := scanForBuiltinFallbacks(&fd)
				if len(flag) > 0 {
					sort.Strings(flag)
					t.Logf("⚠ unresolved BuiltinCalls: %s", strings.Join(flag, ", "))
				}
			}
		})
	}
}

func probeLowerFile(name string) (*stmt.Program, error) {
	abs, err := filepath.Abs(filepath.Join("..", "codegen", name))
	if err != nil {
		return nil, err
	}
	data, err := readFile(abs)
	if err != nil {
		return nil, err
	}
	res, err := pipeline.Run(pipeline.Config{
		Mode:         pipeline.ModeCheck,
		RelaxModules: true,
	}, pipeline.Source{Filename: abs, Code: string(data)})
	if err != nil {
		return nil, err
	}
	if len(res.Errors) > 0 {
		return nil, fmt.Errorf("pipeline errors: %v", res.Errors)
	}
	prog := &stmt.Program{Package: "probe"}
	if res.Artifacts.AST != nil {
		seen := map[string]bool{}
		for _, decl := range res.Artifacts.AST.Decls {
			if td, ok := decl.(*ast.TypeDecl); ok && !seen[td.Name] {
				seen[td.Name] = true
				prog.TypeDecls = append(prog.TypeDecls, lower.LowerTypeDecl(td))
			}
		}
	}
	fileProg, err := lower.LowerProgram(res.Artifacts.Core, res.Artifacts.CoreTI, res.Artifacts.AST, "probe")
	if err != nil {
		return nil, err
	}
	prog.FuncDecls = append(prog.FuncDecls, fileProg.FuncDecls...)
	return prog, nil
}

// dumpStmts pretty-prints a slice of stmt.Stmt with indentation.
func dumpStmts(stmts []stmt.Stmt, indent string) string {
	if len(stmts) == 0 {
		return indent + "<empty>"
	}
	var sb strings.Builder
	for _, s := range stmts {
		sb.WriteString(indent)
		sb.WriteString(dumpStmt(s, indent))
		sb.WriteString("\n")
	}
	return strings.TrimRight(sb.String(), "\n")
}

func dumpStmt(s stmt.Stmt, indent string) string {
	switch s := s.(type) {
	case stmt.VarDecl:
		return fmt.Sprintf("VarDecl %s = %s", s.Name, dumpExpr(s.Value))
	case stmt.AssignStmt:
		return fmt.Sprintf("Assign %s = %s", s.Name, dumpExpr(s.Value))
	case stmt.ReturnStmt:
		return fmt.Sprintf("Return %s", dumpExpr(s.Value))
	case stmt.ExprStmt:
		return fmt.Sprintf("Expr %s", dumpExpr(s.Value))
	case stmt.IfStmt:
		var sb strings.Builder
		sb.WriteString(fmt.Sprintf("If %s {\n", dumpExpr(s.Cond)))
		sb.WriteString(dumpStmts(s.Then, indent+"  "))
		if len(s.Else) > 0 {
			sb.WriteString("\n" + indent + "} else {\n")
			sb.WriteString(dumpStmts(s.Else, indent+"  "))
		}
		sb.WriteString("\n" + indent + "}")
		return sb.String()
	case stmt.SwitchStmt:
		var sb strings.Builder
		sb.WriteString(fmt.Sprintf("Switch %s on %q {\n", dumpExpr(s.Scrutinee), s.ADTName))
		for _, c := range s.Cases {
			binds := make([]string, len(c.Bindings))
			for i, b := range c.Bindings {
				binds[i] = fmt.Sprintf("%s@%d", b.Name, b.FieldIndex)
			}
			sb.WriteString(indent + "  case " + c.Tag + "(" + strings.Join(binds, ",") + "):\n")
			sb.WriteString(dumpStmts(c.Body, indent+"    ") + "\n")
		}
		if len(s.Default) > 0 {
			sb.WriteString(indent + "  default:\n")
			sb.WriteString(dumpStmts(s.Default, indent+"    ") + "\n")
		}
		sb.WriteString(indent + "}")
		return sb.String()
	}
	return fmt.Sprintf("<%T>", s)
}

func dumpExpr(e stmt.Expr) string {
	if e == nil {
		return "<nil>"
	}
	switch e := e.(type) {
	case stmt.LitInt:
		return fmt.Sprintf("%d", e.Value)
	case stmt.LitFloat:
		return fmt.Sprintf("%g", e.Value)
	case stmt.LitBool:
		return fmt.Sprintf("%v", e.Value)
	case stmt.LitString:
		return fmt.Sprintf("%q", e.Value)
	case stmt.LitUnit:
		return "()"
	case stmt.VarRef:
		return e.Name
	case stmt.GlobalRef:
		return fmt.Sprintf("Global(%s.%s)", e.Module, e.Name)
	case stmt.BinOp:
		return fmt.Sprintf("(%s %v %s)", dumpExpr(e.Left), e.Op, dumpExpr(e.Right))
	case stmt.UnOp:
		return fmt.Sprintf("(%v %s)", e.Op, dumpExpr(e.Operand))
	case stmt.Call:
		args := make([]string, len(e.Args))
		for i, a := range e.Args {
			args[i] = dumpExpr(a)
		}
		return fmt.Sprintf("Call(%s)(%s)", dumpExpr(e.Func), strings.Join(args, ", "))
	case stmt.BuiltinCall:
		args := make([]string, len(e.Args))
		for i, a := range e.Args {
			args[i] = dumpExpr(a)
		}
		return fmt.Sprintf("Builtin[%s](%s)", e.Name, strings.Join(args, ", "))
	case stmt.FieldAccess:
		return fmt.Sprintf("%s.%s", dumpExpr(e.Record), e.Field)
	case stmt.TupleLit:
		elems := make([]string, len(e.Elems))
		for i, el := range e.Elems {
			elems[i] = dumpExpr(el)
		}
		return "(" + strings.Join(elems, ", ") + ")"
	case stmt.ListLit:
		elems := make([]string, len(e.Elems))
		for i, el := range e.Elems {
			elems[i] = dumpExpr(el)
		}
		return "[" + strings.Join(elems, ", ") + "]"
	case stmt.RecordLit:
		flds := make([]string, len(e.Fields))
		for i, f := range e.Fields {
			flds[i] = fmt.Sprintf("%s: %s", f.Name, dumpExpr(f.Value))
		}
		return fmt.Sprintf("Record{%s}{%s}", e.TypeName, strings.Join(flds, ", "))
	case stmt.Cons:
		return fmt.Sprintf("Cons(%s, %s)", dumpExpr(e.Head), dumpExpr(e.Tail))
	case stmt.IfExpr:
		return fmt.Sprintf("IfExpr(%s, %s, %s)", dumpExpr(e.Cond), dumpExpr(e.Then), dumpExpr(e.Else))
	case stmt.Lambda:
		params := make([]string, len(e.Params))
		for i, p := range e.Params {
			params[i] = p.Name
		}
		return fmt.Sprintf("λ(%s) {...}", strings.Join(params, ","))
	case stmt.ADTConstructor:
		args := make([]string, len(e.Args))
		for i, a := range e.Args {
			args[i] = dumpExpr(a)
		}
		return fmt.Sprintf("%s(%s)", e.Tag, strings.Join(args, ", "))
	}
	return fmt.Sprintf("<%T>", e)
}

// scanForBuiltinFallbacks walks all expressions in a function and collects
// the names of every BuiltinCall that doesn't look like a "real" builtin
// (i.e. starts with `_dict_` or contains a class name like `_Fractional_`,
// `_Num_`, etc. — the descriptive fallback shapes from lowerDictMethod).
func scanForBuiltinFallbacks(fd *stmt.FuncDecl) []string {
	seen := map[string]bool{}
	var visit func(stmt.Expr)
	visit = func(e stmt.Expr) {
		if e == nil {
			return
		}
		switch e := e.(type) {
		case stmt.BuiltinCall:
			if isFallbackName(e.Name) {
				seen[e.Name] = true
			}
			for _, a := range e.Args {
				visit(a)
			}
		case stmt.BinOp:
			visit(e.Left)
			visit(e.Right)
		case stmt.UnOp:
			visit(e.Operand)
		case stmt.Call:
			visit(e.Func)
			for _, a := range e.Args {
				visit(a)
			}
		case stmt.FieldAccess:
			visit(e.Record)
		case stmt.TupleLit:
			for _, el := range e.Elems {
				visit(el)
			}
		case stmt.ListLit:
			for _, el := range e.Elems {
				visit(el)
			}
		case stmt.RecordLit:
			for _, f := range e.Fields {
				visit(f.Value)
			}
		case stmt.Cons:
			visit(e.Head)
			visit(e.Tail)
		case stmt.IfExpr:
			visit(e.Cond)
			visit(e.Then)
			visit(e.Else)
		case stmt.ADTConstructor:
			for _, a := range e.Args {
				visit(a)
			}
		}
	}
	var visitStmts func(ss []stmt.Stmt)
	visitStmts = func(ss []stmt.Stmt) {
		for _, s := range ss {
			switch s := s.(type) {
			case stmt.VarDecl:
				visit(s.Value)
			case stmt.AssignStmt:
				visit(s.Value)
			case stmt.ReturnStmt:
				visit(s.Value)
			case stmt.ExprStmt:
				visit(s.Value)
			case stmt.IfStmt:
				visit(s.Cond)
				visitStmts(s.Then)
				visitStmts(s.Else)
			case stmt.SwitchStmt:
				visit(s.Scrutinee)
				for _, c := range s.Cases {
					visitStmts(c.Body)
				}
				visitStmts(s.Default)
			}
		}
	}
	visitStmts(fd.Body)
	visit(fd.Return)
	out := make([]string, 0, len(seen))
	for k := range seen {
		out = append(out, k)
	}
	return out
}

func isFallbackName(name string) bool {
	if strings.HasPrefix(name, "_dict_") {
		return true
	}
	for _, prefix := range []string{
		"_Num_", "_Eq_", "_Ord_", "_Show_",
		"_Fractional_", "_Read_", "_Bounded_",
	} {
		if strings.HasPrefix(name, prefix) {
			return true
		}
	}
	return false
}
