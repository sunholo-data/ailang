package elaborate

import (
	"fmt"

	"github.com/sunholo-data/ailang/internal/ast"
	"github.com/sunholo-data/ailang/internal/core"
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

// ModuleLet represents a module-level let binding
type ModuleLet struct {
	Name  string
	Value ast.Expr
	Pos   ast.Pos
}

// collectModuleLets extracts module-level let bindings from file statements
// These bindings should be in scope for all function bodies in the module
func collectModuleLets(file *ast.File) []*ModuleLet {
	var lets []*ModuleLet
	for _, stmt := range file.Statements {
		if letExpr, ok := stmt.(*ast.Let); ok {
			lets = append(lets, &ModuleLet{
				Name:  letExpr.Name,
				Value: letExpr.Value,
				Pos:   letExpr.Position(),
			})
		}
	}
	return lets
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

	// Collect module-level let bindings BEFORE function elaboration
	// These must be in scope for function bodies (M-BUG-MODULE-LET-SCOPE fix)
	moduleLets := collectModuleLets(file)

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
					// Use the alias name if present, otherwise the original name.
					// This prevents imported functions from overwriting local
					// definitions with the same name (e.g., "import foo (bar as baz)"
					// stores as "baz", not "bar").
					bindName := sym
					if imp.SymbolAliases != nil {
						if alias, ok := imp.SymbolAliases[sym]; ok {
							bindName = alias
						}
					}
					// Convert imported func to FuncSig
					// The GetExport already returns *ast.FuncDecl
					sig := astFuncToSig(decl)
					// Don't overwrite local function definitions with imports.
					// Local functions take precedence; the import is still
					// accessible via globalEnv (VarGlobal) at elaboration time.
					if _, isLocal := symbols[bindName]; !isLocal {
						symbols[bindName] = sig
					}
					// Mark as imported using the bind name
					imports[bindName] = imp.Path + "/" + sym
				}
			}
		}
	}

	// M-MODULE-LET-FUNC-RESOLUTION (#366): module-level lets are first-class
	// call-graph nodes, not a wrapping special case. Build a name->let lookup so
	// the SCC emitter can tell funcs from lets and emit each in dependency order.
	letByName := make(map[string]*ModuleLet, len(moduleLets))
	for _, ml := range moduleLets {
		letByName[ml.Name] = ml
	}

	// Build call graph for SCC detection over BOTH funcs and module lets.
	graph := BuildCallGraph(funcs, moduleLets, symbols, imports)

	// Find SCCs for mutual recursion
	sccs := graph.SCCs()

	// Emit each SCC in topological order. Every node is either a module func
	// (emitted as a lambda-valued let/letrec binding) or a module let (emitted
	// with its elaborated value). Interleaving lets and funcs in dependency
	// order is what lets a module-let value resolve a module func — the fix.
	var coreDecls []core.CoreExpr
	meta := make(map[string]*core.DeclMeta)

	// emitFuncMeta records DeclMeta/contracts for a func node. Let nodes carry no
	// contracts and must NOT collide with func-keyed meta.
	emitFuncMeta := func(f *FuncSig) error {
		astFunc := findASTFunc(file, f.Name)
		if astFunc == nil {
			return nil
		}
		contracts, err := e.elaborateContracts(astFunc.Properties)
		if err != nil {
			return fmt.Errorf("elaborating contracts for %s: %w", f.Name, err)
		}
		dm := &core.DeclMeta{
			Name:      f.Name,
			IsExport:  astFunc.IsExport,
			IsPure:    astFunc.IsPure,
			Contracts: contracts,
		}
		if astFunc.VerifyDepth != nil {
			dm.VerifyDepth = *astFunc.VerifyDepth
		}
		meta[f.Name] = dm
		return nil
	}

	// bindingValue elaborates the core value for a node: a lambda for funcs, the
	// elaborated value expression for module lets.
	bindingValue := func(nodeName string) (core.CoreExpr, ast.Pos, error) {
		if f, isFunc := symbols[nodeName]; isFunc {
			if err := emitFuncMeta(f); err != nil {
				return nil, ast.Pos{}, err
			}
			lambda, err := e.funcToLambda(f)
			if err != nil {
				return nil, ast.Pos{}, err
			}
			return lambda, f.FuncDecl.Position(), nil
		}
		ml := letByName[nodeName]
		value, err := e.elaborateExpr(ml.Value)
		if err != nil {
			return nil, ast.Pos{}, fmt.Errorf("error elaborating module-level let '%s': %w", ml.Name, err)
		}
		return value, ml.Pos, nil
	}

	for _, scc := range sccs {
		if len(scc) == 1 && !isSelfRecursive(scc[0], symbols, letByName) {
			// Single non-recursive node -> Let
			nodeName := scc[0]
			value, pos, err := bindingValue(nodeName)
			if err != nil {
				return nil, err
			}
			coreDecls = append(coreDecls, &core.Let{
				CoreNode: e.makeNode(pos),
				Name:     nodeName,
				Value:    value,
				Body: &core.Var{
					CoreNode: e.makeNode(pos),
					Name:     nodeName,
				},
			})
		} else {
			// Mutual or self-recursive group -> LetRec.
			//
			// Tarjan yields SCC members in reverse topological (post-)order, so a
			// self/mutual-recursive group that mixes lets and funcs is bound
			// together. The runtime LetRec evaluator (eval_expressions.go) handles
			// non-lambda RHS with a strict-eval self-cycle detector, so a module
			// letrec self-reference emits an honest cycle error rather than a false
			// "undefined variable" (see #366 M2 decision).
			var bindings []core.RecBinding
			for _, nodeName := range scc {
				value, _, err := bindingValue(nodeName)
				if err != nil {
					return nil, err
				}
				bindings = append(bindings, core.RecBinding{
					Name:  nodeName,
					Value: value,
				})
			}
			coreDecls = append(coreDecls, &core.LetRec{
				CoreNode: e.makeNode(ast.Pos{Line: 0, Column: 0}),
				Bindings: bindings,
				Body: &core.Lit{
					CoreNode: e.makeNode(ast.Pos{Line: 0, Column: 0}),
					Kind:     core.UnitLit,
					Value:    nil,
				},
			})
		}
	}

	// Add any non-func, non-let statements (e.g., a bare main() call). These are
	// emitted AFTER all func/let decls, so they see every module binding via the
	// forward-threaded env — no wrapping needed.
	for _, stmt := range file.Statements {
		if expr, ok := stmt.(ast.Expr); ok {
			// Skip let expressions - they are module-let decls handled above.
			if _, isLet := expr.(*ast.Let); isLet {
				continue
			}
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
		// Skip extern functions - they have no body to elaborate
		// Extern functions are handled separately in codegen (extern_stubs.go)
		if f.IsExtern {
			continue
		}
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

// isSelfRecursive checks whether a module node (func OR let) references its own
// name in its body/value. Self-recursive nodes must be emitted as LetRec so the
// name is in scope for its own definition (a recursive func) or so the runtime's
// strict-eval cycle detector fires an honest error (a self-referential let, #366).
func isSelfRecursive(name string, symbols map[string]*FuncSig, lets map[string]*ModuleLet) bool {
	var body ast.Expr
	if f, ok := symbols[name]; ok && f != nil {
		body = f.Body
	} else if ml, ok := lets[name]; ok && ml != nil {
		body = ml.Value
	} else {
		return false
	}

	for _, ref := range findReferences(body) {
		if ref == name {
			return true
		}
	}
	return false
}

// makeNodeFromFunc creates CoreNode from FuncSig
func (e *Elaborator) makeNodeFromFunc(f *FuncSig) core.CoreNode {
	pos := f.FuncDecl.Position()
	return e.makeNode(pos)
}
