package elaborate

import (
	"fmt"

	"github.com/sunholo/ailang/internal/ast"
	"github.com/sunholo/ailang/internal/core"
)

// Elaborate transforms a surface program to Core ANF
func (e *Elaborator) Elaborate(prog *ast.Program) (*core.Program, error) {
	// Check new File structure first (for REPL and bare expressions)
	if prog.File != nil && prog.File.Module == nil && len(prog.File.Statements) > 0 {
		// First, process type declarations to register constructors
		for _, stmt := range prog.File.Statements {
			if typeDecl, ok := stmt.(*ast.TypeDecl); ok {
				_, err := e.elaborateTypeDecl(typeDecl)
				if err != nil {
					return nil, fmt.Errorf("failed to process type declaration %s: %w", typeDecl.Name, err)
				}
			}
		}

		// Process bare expressions from REPL
		var coreDecls []core.CoreExpr
		for _, stmt := range prog.File.Statements {
			if expr, ok := stmt.(ast.Expr); ok {
				coreExpr, err := e.elaborateExpr(expr)
				if err != nil {
					return nil, err
				}
				if coreExpr != nil {
					coreDecls = append(coreDecls, coreExpr)
				}
			}
		}
		return &core.Program{Decls: coreDecls, Meta: make(map[string]*core.DeclMeta)}, nil
	}

	// Legacy: check Module field
	if prog.Module == nil {
		// For simple expressions without a module, return empty program
		// Use ElaborateExpr for bare expressions
		return &core.Program{Meta: make(map[string]*core.DeclMeta)}, nil
	}

	var coreDecls []core.CoreExpr
	for _, decl := range prog.Module.Decls {
		coreExpr, err := e.elaborateNode(decl)
		if err != nil {
			return nil, err
		}
		if coreExpr != nil {
			coreDecls = append(coreDecls, coreExpr)
		}
	}

	return &core.Program{Decls: coreDecls, Meta: make(map[string]*core.DeclMeta)}, nil
}

// ElaborateExpr transforms a single expression to Core ANF (for testing)
func (e *Elaborator) ElaborateExpr(expr ast.Expr) (core.CoreExpr, error) {
	return e.elaborateExpr(expr)
}

// ElaborateFile transforms a complete file with module structure to Core ANF
func (e *Elaborator) ElaborateFile(file *ast.File) (*core.Program, error) {
	// For REPL/simple cases without module or funcs
	if file.Module == nil || (len(file.Imports) == 0 && len(file.Funcs) == 0) {
		// First, process type declarations to register constructors
		for _, stmt := range file.Statements {
			if typeDecl, ok := stmt.(*ast.TypeDecl); ok {
				_, err := e.elaborateTypeDecl(typeDecl)
				if err != nil {
					return nil, fmt.Errorf("failed to process type declaration %s: %w", typeDecl.Name, err)
				}
			}
		}

		// Then elaborate statements as expressions
		var coreDecls []core.CoreExpr
		for _, stmt := range file.Statements {
			if expr, ok := stmt.(ast.Expr); ok {
				coreExpr, err := e.elaborateExpr(expr)
				if err != nil {
					return nil, err
				}
				coreDecls = append(coreDecls, coreExpr)
			}
		}
		return &core.Program{Decls: coreDecls, Meta: make(map[string]*core.DeclMeta)}, nil
	}

	// First, process type declarations to register constructors
	// This must happen before function elaboration so constructors are available
	for _, decl := range file.Decls {
		if typeDecl, ok := decl.(*ast.TypeDecl); ok {
			_, err := e.elaborateTypeDecl(typeDecl)
			if err != nil {
				return nil, fmt.Errorf("failed to process type declaration %s: %w", typeDecl.Name, err)
			}
		}
	}

	// Build symbol table and imports map
	funcs := collectFuncSigs(file)
	imports := collectImports(file)
	symbols := make(map[string]*FuncSig)
	for _, f := range funcs {
		symbols[f.Name] = f
	}

	// Load imported modules and add their exports to symbols
	if e.moduleLoader != nil {
		for _, imp := range file.Imports {
			if len(imp.Symbols) > 0 {
				// Selective import
				for _, sym := range imp.Symbols {
					decl, err := e.moduleLoader.GetExport(imp.Path, sym)
					if err != nil {
						// Preserve structured error reports without wrapping
						return nil, err
					}
					// If decl is nil, it's a type or constructor - skip for now
					// (they'll be handled by the type checker and linker)
					if decl == nil {
						continue
					}
					// Convert imported func to FuncSig
					// The GetExport already returns *ast.FuncDecl
					sig := astFuncToSig(decl)
					symbols[sym] = sig
					// Mark as imported
					imports[sym] = imp.Path + "/" + sym
				}
			}
		}
	}

	// Build call graph for SCC detection
	graph := BuildCallGraph(funcs, symbols, imports)

	// Find SCCs for mutual recursion
	sccs := graph.SCCs()

	// Desugar functions based on SCCs
	var coreDecls []core.CoreExpr
	meta := make(map[string]*core.DeclMeta)
	for _, scc := range sccs {
		if len(scc) == 1 && !isSelfRecursive(scc[0], symbols) {
			// Single non-recursive function → Let
			f := symbols[scc[0]]
			lambda, err := e.funcToLambda(f)
			if err != nil {
				return nil, err
			}

			let := &core.Let{
				CoreNode: e.makeNodeFromFunc(f),
				Name:     f.Name,
				Value:    lambda,
				Body: &core.Var{
					CoreNode: e.makeNodeFromFunc(f),
					Name:     f.Name,
				},
			}
			// Track metadata from original AST function
			if astFunc := findASTFunc(file, f.Name); astFunc != nil {
				meta[f.Name] = &core.DeclMeta{
					Name:     f.Name,
					IsExport: astFunc.IsExport,
					IsPure:   astFunc.IsPure,
				}
			}
			coreDecls = append(coreDecls, let)
		} else {
			// Mutual or self-recursive → LetRec
			var bindings []core.RecBinding
			for _, fname := range scc {
				f := symbols[fname]
				lambda, err := e.funcToLambda(f)
				if err != nil {
					return nil, err
				}
				bindings = append(bindings, core.RecBinding{
					Name:  f.Name,
					Value: lambda,
				})
				// Track metadata for each binding
				if astFunc := findASTFunc(file, f.Name); astFunc != nil {
					meta[f.Name] = &core.DeclMeta{
						Name:     f.Name,
						IsExport: astFunc.IsExport,
						IsPure:   astFunc.IsPure,
					}
				}
			}

			// Create a LetRec that binds all functions and returns unit
			letRec := &core.LetRec{
				CoreNode: e.makeNode(ast.Pos{Line: 0, Column: 0}),
				Bindings: bindings,
				Body: &core.Lit{
					CoreNode: e.makeNode(ast.Pos{Line: 0, Column: 0}),
					Kind:     core.UnitLit,
					Value:    nil,
				},
			}
			coreDecls = append(coreDecls, letRec)
		}
	}

	// Add any non-func statements
	for _, stmt := range file.Statements {
		if expr, ok := stmt.(ast.Expr); ok {
			coreExpr, err := e.elaborateExpr(expr)
			if err != nil {
				return nil, err
			}
			coreDecls = append(coreDecls, coreExpr)
		}
	}

	return &core.Program{Decls: coreDecls, Meta: meta}, nil
}

// findASTFunc finds the AST function declaration by name
func findASTFunc(file *ast.File, name string) *ast.FuncDecl {
	for _, fn := range file.Funcs {
		if fn.Name == name {
			return fn
		}
	}
	return nil
}

// collectFuncSigs extracts function signatures from file
// astFuncToSig converts an AST FuncDecl to a FuncSig
func astFuncToSig(f *ast.FuncDecl) *FuncSig {
	// Extract parameter names
	params := make([]string, len(f.Params))
	for i, p := range f.Params {
		params[i] = p.Name
	}

	return &FuncSig{
		Name:     f.Name,
		NodeSID:  "", // TODO: Calculate surface SID
		Body:     f.Body,
		Params:   params,
		IsPure:   f.IsPure,
		IsExport: f.IsExport,
		Tests:    f.Tests,
		Props:    f.Properties,
		FuncDecl: f,
	}
}

func collectFuncSigs(file *ast.File) []*FuncSig {
	var funcs []*FuncSig
	for _, f := range file.Funcs {
		funcs = append(funcs, astFuncToSig(f))
	}
	return funcs
}

// collectImports builds import name map
func collectImports(file *ast.File) map[string]string {
	imports := make(map[string]string)
	for _, imp := range file.Imports {
		if imp.Symbols != nil {
			// Selective import
			for _, sym := range imp.Symbols {
				imports[sym] = imp.Path + "/" + sym
			}
		}
		// TODO: Handle wildcard imports
	}
	return imports
}

// isSelfRecursive checks if function calls itself
func isSelfRecursive(fname string, symbols map[string]*FuncSig) bool {
	f := symbols[fname]
	if f == nil {
		return false
	}

	refs := findReferences(f.Body)
	for _, ref := range refs {
		if ref == fname {
			return true
		}
	}
	return false
}

// funcToLambda converts function to lambda
func (e *Elaborator) funcToLambda(f *FuncSig) (core.CoreExpr, error) {
	body, err := e.elaborateExpr(f.Body)
	if err != nil {
		return nil, err
	}

	lambda := &core.Lambda{
		CoreNode: e.makeNodeFromFunc(f),
		Params:   f.Params,
		Body:     body,
	}

	// TODO: Preserve metadata (pure, export, tests, props) in CoreNode.Meta

	return lambda, nil
}

// makeNodeFromFunc creates CoreNode from FuncSig
func (e *Elaborator) makeNodeFromFunc(f *FuncSig) core.CoreNode {
	pos := f.FuncDecl.Position()
	return e.makeNode(pos)
}

// elaborateNode handles any AST node
func (e *Elaborator) elaborateNode(node ast.Node) (core.CoreExpr, error) {
	switch n := node.(type) {
	case ast.Expr:
		return e.elaborateExpr(n)
	case *ast.FuncDecl:
		return e.elaborateFuncDecl(n)
	case *ast.TypeDecl:
		// Type declarations don't produce Core expressions
		// They register constructors for use in expressions
		return e.elaborateTypeDecl(n)
	default:
		return nil, fmt.Errorf("elaboration not implemented for %T", node)
	}
}

// elaborateTypeDecl processes a type declaration and registers its constructors
// Type declarations don't produce Core expressions - they have side effects:
// 1. Register constructors in the elaborator's constructor map
// 2. Add constructors to the module interface (for exports)
func (e *Elaborator) elaborateTypeDecl(decl *ast.TypeDecl) (core.CoreExpr, error) {
	// Extract type name
	typeName := decl.Name

	// Process the type definition
	switch def := decl.Definition.(type) {
	case *ast.AlgebraicType:
		// Process each constructor in the ADT
		for _, ctor := range def.Constructors {
			// Register constructor in elaborator's map
			e.RegisterConstructor(typeName, ctor.Name, len(ctor.Fields), false)
		}
		// Type declarations don't produce code, return nil
		return nil, nil

	case *ast.RecordType:
		// Record types don't have constructors (they're structural)
		// TODO: Handle record type declarations if needed
		return nil, nil

	default:
		return nil, fmt.Errorf("unknown type definition: %T", def)
	}
}

// elaborateFuncDecl handles function declarations
func (e *Elaborator) elaborateFuncDecl(fn *ast.FuncDecl) (core.CoreExpr, error) {
	// Convert to lambda
	lambda := &ast.Lambda{
		Params: fn.Params,
		Body:   fn.Body,
		Pos:    fn.Pos,
	}

	value, err := e.normalizeLambda(lambda)
	if err != nil {
		return nil, err
	}

	// Wrap in let rec if recursive
	return &core.LetRec{
		CoreNode: e.makeNode(fn.Position()),
		Bindings: []core.RecBinding{{Name: fn.Name, Value: value}},
		Body:     &core.Var{CoreNode: e.makeNode(fn.Position()), Name: fn.Name},
	}, nil
}
