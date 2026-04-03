package lower

import (
	"fmt"
	"strings"

	"github.com/sunholo/ailang/internal/ast"
	"github.com/sunholo/ailang/internal/core"
	"github.com/sunholo/ailang/internal/gen/stmt"
	"github.com/sunholo/ailang/internal/types"
)

// LowerProgram converts a Core program + AST (for type declarations) into
// a Statement IR Program ready for emission.
//
// This is the main entry point for the lowering pipeline:
//
//	pipeline.Run(ModeCheck) → Core + CoreTypeInfo + AST
//	lower.LowerProgram(core, cti, ast) → stmt.Program
//	emitter.Emit(program) → Go/Rust/etc. source code
func LowerProgram(
	coreProg *core.Program,
	cti types.CoreTypeInfo,
	astFile *ast.File,
	pkgName string,
) (*stmt.Program, error) {
	prog := &stmt.Program{
		Package: pkgName,
	}

	// Phase 1: Lower type declarations from AST.
	if astFile != nil {
		prog.TypeDecls = lowerTypeDecls(astFile)
	}

	// Phase 2: Lower function declarations from Core.
	if coreProg != nil {
		funcs, err := lowerFuncDecls(coreProg, cti)
		if err != nil {
			return nil, fmt.Errorf("lowering functions: %w", err)
		}
		prog.FuncDecls = funcs
	}

	// Phase 3: Collect required imports.
	prog.Imports = collectImports(prog)

	// Phase 4: Validate the result.
	if err := stmt.Validate(prog); err != nil {
		return nil, fmt.Errorf("Statement IR validation failed: %w", err)
	}

	return prog, nil
}

// LowerMultiModule handles multi-module compilation by lowering each module
// and merging the results into a single Program.
func LowerMultiModule(
	modules map[string]*ModuleInput,
	pkgName string,
) (*stmt.Program, error) {
	prog := &stmt.Program{
		Package: pkgName,
	}

	for modName, mod := range modules {
		// Lower type declarations.
		if mod.AST != nil {
			typeDecls := lowerTypeDecls(mod.AST)
			prog.TypeDecls = append(prog.TypeDecls, typeDecls...)
		}

		// Lower function declarations.
		if mod.Core != nil {
			funcs, err := lowerFuncDecls(mod.Core, mod.CTI)
			if err != nil {
				return nil, fmt.Errorf("module %s: %w", modName, err)
			}
			// Tag functions with their source module.
			for i := range funcs {
				funcs[i].Module = modName
			}
			prog.FuncDecls = append(prog.FuncDecls, funcs...)
		}
	}

	prog.Imports = collectImports(prog)

	if err := stmt.Validate(prog); err != nil {
		return nil, fmt.Errorf("Statement IR validation failed: %w", err)
	}

	return prog, nil
}

// ModuleInput bundles the artifacts needed to lower one module.
type ModuleInput struct {
	Core *core.Program
	CTI  types.CoreTypeInfo
	AST  *ast.File
}

// lowerTypeDecls extracts type declarations from the AST.
func lowerTypeDecls(astFile *ast.File) []stmt.TypeDecl {
	var result []stmt.TypeDecl

	for _, node := range astFile.Decls {
		if td, ok := node.(*ast.TypeDecl); ok {
			result = append(result, LowerTypeDecl(td))
		}
	}
	// Also check Statements (some type decls may be there).
	for _, node := range astFile.Statements {
		if td, ok := node.(*ast.TypeDecl); ok {
			result = append(result, LowerTypeDecl(td))
		}
	}

	return result
}

// lowerFuncDecls converts Core top-level declarations into FuncDecls.
func lowerFuncDecls(coreProg *core.Program, cti types.CoreTypeInfo) ([]stmt.FuncDecl, error) {
	var result []stmt.FuncDecl

	for _, decl := range coreProg.Decls {
		fd, err := lowerTopLevelDecl(decl, coreProg.Meta, cti)
		if err != nil {
			return nil, err
		}
		if fd != nil {
			result = append(result, *fd)
		}
	}

	return result, nil
}

// lowerTopLevelDecl converts a single Core top-level expression into a FuncDecl.
// Top-level Core expressions are Let bindings wrapping Lambdas.
func lowerTopLevelDecl(
	e core.CoreExpr,
	meta map[string]*core.DeclMeta,
	cti types.CoreTypeInfo,
) (*stmt.FuncDecl, error) {
	// Top-level declarations are Let bindings: Let("funcName", Lambda(...), body)
	let, ok := e.(*core.Let)
	if !ok {
		// Could be a top-level expression (e.g., in REPL). Skip.
		return nil, nil
	}

	name := let.Name

	// Get metadata (export status, etc.).
	dm := meta[name]
	exported := dm != nil && dm.IsExport

	// The value should be a Lambda (or DictAbs wrapping a Lambda).
	value := let.Value
	// Unwrap DictAbs — erase dictionary abstraction.
	if da, ok := value.(*core.DictAbs); ok {
		value = da.Body
	}

	lam, ok := value.(*core.Lambda)
	if !ok {
		// Top-level value binding (not a function). Lower as a 0-arg function.
		retType := resolveExprType(let.Value, cti)
		body, retExpr := FlattenBlock(let.Value, cti)
		return &stmt.FuncDecl{
			Name:       name,
			ReturnType: retType,
			Body:       body,
			Return:     retExpr,
			Exported:   exported,
		}, nil
	}

	// Lower the lambda into a FuncDecl.
	params := lowerParams(lam, cti)
	retType := resolveReturnType(lam, cti)
	body, retExpr := FlattenBlock(lam.Body, cti)

	return &stmt.FuncDecl{
		Name:       name,
		Params:     params,
		ReturnType: retType,
		Body:       body,
		Return:     retExpr,
		Exported:   exported,
	}, nil
}

// lowerParams extracts parameter types from a Lambda's type info.
func lowerParams(lam *core.Lambda, cti types.CoreTypeInfo) []stmt.Param {
	params := make([]stmt.Param, len(lam.Params))

	// Try to get function type from CoreTypeInfo.
	var funcType *types.TFunc2
	if t, ok := cti[lam.ID()]; ok {
		funcType, _ = t.(*types.TFunc2)
	}

	for i, name := range lam.Params {
		paramType := stmt.ResolvedType(stmt.InterfaceType{})
		if funcType != nil && i < len(funcType.Params) {
			paramType = ProjectType(funcType.Params[i])
		}
		params[i] = stmt.Param{Name: name, Type: paramType}
	}

	return params
}

// resolveReturnType extracts the return type of a lambda.
func resolveReturnType(lam *core.Lambda, cti types.CoreTypeInfo) stmt.ResolvedType {
	if t, ok := cti[lam.ID()]; ok {
		if ft, ok := t.(*types.TFunc2); ok {
			return ProjectType(ft.Return)
		}
	}
	return stmt.InterfaceType{}
}

// resolveExprType gets the type of an expression from CoreTypeInfo.
func resolveExprType(e core.CoreExpr, cti types.CoreTypeInfo) stmt.ResolvedType {
	if e == nil {
		return stmt.InterfaceType{}
	}
	if t, ok := cti[e.ID()]; ok {
		return ProjectType(t)
	}
	return stmt.InterfaceType{}
}

// collectImports determines which imports are needed based on the program's
// usage of builtins, string operations, etc.
func collectImports(prog *stmt.Program) []stmt.ImportSpec {
	needs := map[string]bool{}

	// Walk all expressions to find what's used.
	for _, fd := range prog.FuncDecls {
		walkExprsInFunc(&fd, func(e stmt.Expr) {
			switch e := e.(type) {
			case stmt.BuiltinCall:
				if strings.HasPrefix(e.Name, "_str_") || strings.HasPrefix(e.Name, "_show") {
					needs["strings"] = true
					needs["fmt"] = true
				}
				if strings.HasPrefix(e.Name, "_list_") {
					// Lists use slices — no import needed.
				}
			case stmt.LitString:
				// String literals may need fmt for formatting.
			}
		})
	}

	var imports []stmt.ImportSpec
	for path := range needs {
		imports = append(imports, stmt.ImportSpec{Path: path})
	}
	return imports
}

// walkExprsInFunc visits all expressions in a function declaration.
func walkExprsInFunc(fd *stmt.FuncDecl, visit func(stmt.Expr)) {
	for _, s := range fd.Body {
		walkExprsInStmt(s, visit)
	}
	if fd.Return != nil {
		walkExpr(fd.Return, visit)
	}
}

func walkExprsInStmt(s stmt.Stmt, visit func(stmt.Expr)) {
	switch s := s.(type) {
	case stmt.VarDecl:
		if s.Value != nil {
			walkExpr(s.Value, visit)
		}
	case stmt.AssignStmt:
		walkExpr(s.Value, visit)
	case stmt.ReturnStmt:
		walkExpr(s.Value, visit)
	case stmt.ExprStmt:
		walkExpr(s.Value, visit)
	case stmt.IfStmt:
		walkExpr(s.Cond, visit)
		for _, ts := range s.Then {
			walkExprsInStmt(ts, visit)
		}
		for _, es := range s.Else {
			walkExprsInStmt(es, visit)
		}
	case stmt.SwitchStmt:
		walkExpr(s.Scrutinee, visit)
		for _, c := range s.Cases {
			for _, bs := range c.Body {
				walkExprsInStmt(bs, visit)
			}
		}
		for _, ds := range s.Default {
			walkExprsInStmt(ds, visit)
		}
	}
}

func walkExpr(e stmt.Expr, visit func(stmt.Expr)) {
	if e == nil {
		return
	}
	visit(e)

	switch e := e.(type) {
	case stmt.BinOp:
		walkExpr(e.Left, visit)
		walkExpr(e.Right, visit)
	case stmt.UnOp:
		walkExpr(e.Operand, visit)
	case stmt.Call:
		walkExpr(e.Func, visit)
		for _, a := range e.Args {
			walkExpr(a, visit)
		}
	case stmt.FieldAccess:
		walkExpr(e.Record, visit)
	case stmt.RecordLit:
		for _, f := range e.Fields {
			walkExpr(f.Value, visit)
		}
	case stmt.RecordUpdate:
		walkExpr(e.Base, visit)
		for _, f := range e.Fields {
			walkExpr(f.Value, visit)
		}
	case stmt.ListLit:
		for _, el := range e.Elems {
			walkExpr(el, visit)
		}
	case stmt.ArrayLit:
		for _, el := range e.Elems {
			walkExpr(el, visit)
		}
	case stmt.TupleLit:
		for _, el := range e.Elems {
			walkExpr(el, visit)
		}
	case stmt.Cons:
		walkExpr(e.Head, visit)
		walkExpr(e.Tail, visit)
	case stmt.ADTConstructor:
		for _, a := range e.Args {
			walkExpr(a, visit)
		}
	case stmt.Lambda:
		for _, s := range e.Body {
			walkExprsInStmt(s, visit)
		}
		if e.Return != nil {
			walkExpr(e.Return, visit)
		}
	case stmt.TypeAssert:
		walkExpr(e.Value, visit)
	case stmt.IfExpr:
		walkExpr(e.Cond, visit)
		walkExpr(e.Then, visit)
		walkExpr(e.Else, visit)
	case stmt.BuiltinCall:
		for _, a := range e.Args {
			walkExpr(a, visit)
		}
	}
}
