package elaborate

import (
	"fmt"
	"path/filepath"

	"github.com/sunholo/ailang/internal/ast"
	"github.com/sunholo/ailang/internal/core"
	"github.com/sunholo/ailang/internal/eval"
	"github.com/sunholo/ailang/internal/loader"
	"github.com/sunholo/ailang/internal/types"
)

// Elaborator transforms surface AST to Core ANF
type Elaborator struct {
	nextID       uint64
	surfaceSpans map[uint64]ast.Pos  // Map Core IDs to surface positions
	effectAnnots map[uint64][]string // Map Core IDs to effect annotations from AST
	freshVarNum  int                 // For generating fresh variable names
	moduleLoader *loader.ModuleLoader
	filePath     string                      // Current file path for relative imports
	globalEnv    map[string]core.GlobalRef   // Global environment for imports (name -> GlobalRef)
	constructors map[string]*ConstructorInfo // Available constructors (name -> info)
}

// ConstructorInfo holds information about an available constructor
type ConstructorInfo struct {
	TypeName   string // The ADT type name (e.g., "Option")
	CtorName   string // Constructor name (e.g., "Some")
	Arity      int    // Number of fields
	IsImported bool   // Whether this constructor is imported
}

// NewElaborator creates a new elaborator
func NewElaborator() *Elaborator {
	return &Elaborator{
		nextID:       1,
		surfaceSpans: make(map[uint64]ast.Pos),
		effectAnnots: make(map[uint64][]string),
		freshVarNum:  0,
		globalEnv:    make(map[string]core.GlobalRef),
		constructors: make(map[string]*ConstructorInfo),
	}
}

// NewElaboratorWithPath creates a new elaborator with file path for imports
func NewElaboratorWithPath(filePath string) *Elaborator {
	dir := filepath.Dir(filePath)
	return &Elaborator{
		nextID:       1,
		surfaceSpans: make(map[uint64]ast.Pos),
		effectAnnots: make(map[uint64][]string),
		freshVarNum:  0,
		moduleLoader: loader.NewModuleLoader(dir),
		filePath:     filePath,
		globalEnv:    make(map[string]core.GlobalRef),
		constructors: make(map[string]*ConstructorInfo),
	}
}

// SetGlobalEnv sets the global environment for import resolution
func (e *Elaborator) SetGlobalEnv(env map[string]core.GlobalRef) {
	e.globalEnv = env
}

// AddBuiltinsToGlobalEnv adds all builtin functions to the global environment
func (e *Elaborator) AddBuiltinsToGlobalEnv() {
	// Import eval package to access builtins
	builtins := eval.Builtins
	for name := range builtins {
		e.globalEnv[name] = core.GlobalRef{
			Module: "$builtin",
			Name:   name,
		}
	}
}

// RegisterConstructor adds a constructor to the elaborator's constructor map
func (e *Elaborator) RegisterConstructor(typeName, ctorName string, arity int, isImported bool) {
	e.constructors[ctorName] = &ConstructorInfo{
		TypeName:   typeName,
		CtorName:   ctorName,
		Arity:      arity,
		IsImported: isImported,
	}
}

// GetConstructors returns all constructors defined in this module (not imported)
func (e *Elaborator) GetConstructors() map[string]*ConstructorInfo {
	localConstructors := make(map[string]*ConstructorInfo)
	for name, info := range e.constructors {
		if !info.IsImported {
			localConstructors[name] = info
		}
	}
	return localConstructors
}

// GetEffectAnnotation returns the effect annotation for a Core node ID
func (e *Elaborator) GetEffectAnnotation(nodeID uint64) []string {
	return e.effectAnnots[nodeID]
}

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
			if imp.Symbols != nil && len(imp.Symbols) > 0 {
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

// elaborateExpr transforms surface expression to Core ANF
func (e *Elaborator) elaborateExpr(expr ast.Expr) (core.CoreExpr, error) {
	// First pass: desugar surface constructs
	desugared := e.desugar(expr)

	// Second pass: normalize to ANF
	return e.normalize(desugared)
}

// desugar handles surface syntax sugar
func (e *Elaborator) desugar(expr ast.Expr) ast.Expr {
	// For now, pass through - will add ? operator desugaring etc
	return expr
}

// normalize converts expression to A-Normal Form
func (e *Elaborator) normalize(expr ast.Expr) (core.CoreExpr, error) {
	switch ex := expr.(type) {
	case *ast.Literal:
		return e.normalizeLiteral(ex)

	case *ast.Identifier:
		// Check if this is a nullary constructor (e.g., None, True, False)
		if ctorInfo, isConstructor := e.constructors[ex.Name]; isConstructor && ctorInfo.Arity == 0 {
			// Nullary constructor: transform None → $adt.make_Option_None
			// Note: No App node needed - factory is a value, not a function call
			factoryName := fmt.Sprintf("make_%s_%s", ctorInfo.TypeName, ctorInfo.CtorName)
			return &core.VarGlobal{
				CoreNode: e.makeNode(ex.Position()),
				Ref: core.GlobalRef{
					Module: "$adt",
					Name:   factoryName,
				},
			}, nil
		}
		// Check if this is an imported symbol
		if ref, ok := e.globalEnv[ex.Name]; ok {
			return &core.VarGlobal{
				CoreNode: e.makeNode(ex.Position()),
				Ref:      ref,
			}, nil
		}
		// Otherwise it's a local variable
		return &core.Var{
			CoreNode: e.makeNode(ex.Position()),
			Name:     ex.Name,
		}, nil

	case *ast.Lambda:
		return e.normalizeLambda(ex)

	case *ast.BinaryOp:
		return e.normalizeBinaryOp(ex)

	case *ast.UnaryOp:
		return e.normalizeUnaryOp(ex)

	case *ast.If:
		return e.normalizeIf(ex)

	case *ast.Let:
		return e.normalizeLet(ex)

	case *ast.Block:
		return e.normalizeBlock(ex)

	case *ast.FuncCall:
		return e.normalizeFuncCall(ex)

	case *ast.Record:
		return e.normalizeRecord(ex)

	case *ast.RecordAccess:
		return e.normalizeRecordAccess(ex)

	case *ast.List:
		return e.normalizeList(ex)

	case *ast.Tuple:
		return e.normalizeTuple(ex)

	case *ast.Match:
		return e.normalizeMatch(ex)

	default:
		if expr == nil {
			return nil, fmt.Errorf("normalization received nil expression")
		}
		return nil, fmt.Errorf("normalization not implemented for %T", expr)
	}
}

// normalizeLiteral handles literals
func (e *Elaborator) normalizeLiteral(lit *ast.Literal) (core.CoreExpr, error) {
	var kind core.LitKind
	switch lit.Kind {
	case ast.IntLit:
		kind = core.IntLit
	case ast.FloatLit:
		kind = core.FloatLit
	case ast.StringLit:
		kind = core.StringLit
	case ast.BoolLit:
		kind = core.BoolLit
	case ast.UnitLit:
		kind = core.UnitLit
	default:
		return nil, fmt.Errorf("unknown literal kind: %v", lit.Kind)
	}

	return &core.Lit{
		CoreNode: e.makeNode(lit.Position()),
		Kind:     kind,
		Value:    lit.Value,
	}, nil
}

// normalizeLambda handles lambda expressions
func (e *Elaborator) normalizeLambda(lam *ast.Lambda) (core.CoreExpr, error) {
	// Extract parameter names
	params := make([]string, len(lam.Params))
	for i, p := range lam.Params {
		params[i] = p.Name
	}

	// Normalize body
	body, err := e.normalize(lam.Body)
	if err != nil {
		return nil, err
	}

	// Create Core Lambda node
	coreLam := &core.Lambda{
		CoreNode: e.makeNode(lam.Position()),
		Params:   params,
		Body:     body,
	}

	// Store effect annotations if present
	if len(lam.Effects) > 0 {
		// Validate and normalize effect names
		_, err := types.ElaborateEffectRow(lam.Effects)
		if err != nil {
			return nil, fmt.Errorf("invalid effect annotation: %w", err)
		}
		e.effectAnnots[coreLam.ID()] = lam.Effects
	}

	return coreLam, nil
}

// normalizeBinaryOp handles binary operations with ANF transformation
func (e *Elaborator) normalizeBinaryOp(binop *ast.BinaryOp) (core.CoreExpr, error) {
	// Normalize operands to atomic values
	left, leftBinds, err := e.normalizeToAtomic(binop.Left)
	if err != nil {
		return nil, err
	}

	right, rightBinds, err := e.normalizeToAtomic(binop.Right)
	if err != nil {
		return nil, err
	}

	// Map operator to intrinsic
	var op core.IntrinsicOp
	switch binop.Op {
	case "+":
		op = core.OpAdd
	case "-":
		op = core.OpSub
	case "*":
		op = core.OpMul
	case "/":
		op = core.OpDiv
	case "%":
		op = core.OpMod
	case "==":
		op = core.OpEq
	case "!=":
		op = core.OpNe
	case "<":
		op = core.OpLt
	case "<=":
		op = core.OpLe
	case ">":
		op = core.OpGt
	case ">=":
		op = core.OpGe
	case "++":
		op = core.OpConcat
	case "&&":
		op = core.OpAnd
	case "||":
		op = core.OpOr
	default:
		// For compatibility, still create BinOp for unknown operators
		result := &core.BinOp{
			CoreNode: e.makeNode(binop.Position()),
			Op:       binop.Op,
			Left:     left,
			Right:    right,
		}
		return e.wrapWithBindings(result, append(leftBinds, rightBinds...)), nil
	}

	// Create intrinsic operation
	result := &core.Intrinsic{
		CoreNode: e.makeNode(binop.Position()),
		Op:       op,
		Args:     []core.CoreExpr{left, right},
	}

	// Wrap with let bindings from normalization
	return e.wrapWithBindings(result, append(leftBinds, rightBinds...)), nil
}

// normalizeUnaryOp handles unary operations
func (e *Elaborator) normalizeUnaryOp(unop *ast.UnaryOp) (core.CoreExpr, error) {
	operand, binds, err := e.normalizeToAtomic(unop.Expr)
	if err != nil {
		return nil, err
	}

	// Map operator to intrinsic
	var op core.IntrinsicOp
	switch unop.Op {
	case "-":
		op = core.OpNeg
	case "not":
		op = core.OpNot
	default:
		// For compatibility, still create UnOp for unknown operators
		result := &core.UnOp{
			CoreNode: e.makeNode(unop.Position()),
			Op:       unop.Op,
			Operand:  operand,
		}
		return e.wrapWithBindings(result, binds), nil
	}

	// Create intrinsic operation
	result := &core.Intrinsic{
		CoreNode: e.makeNode(unop.Position()),
		Op:       op,
		Args:     []core.CoreExpr{operand},
	}

	return e.wrapWithBindings(result, binds), nil
}

// normalizeIf handles conditionals
func (e *Elaborator) normalizeIf(ifExpr *ast.If) (core.CoreExpr, error) {
	// Condition must be atomic
	cond, condBinds, err := e.normalizeToAtomic(ifExpr.Condition)
	if err != nil {
		return nil, err
	}

	// Branches can be complex
	thenBranch, err := e.normalize(ifExpr.Then)
	if err != nil {
		return nil, err
	}

	elseBranch, err := e.normalize(ifExpr.Else)
	if err != nil {
		return nil, err
	}

	result := &core.If{
		CoreNode: e.makeNode(ifExpr.Position()),
		Cond:     cond,
		Then:     thenBranch,
		Else:     elseBranch,
	}

	return e.wrapWithBindings(result, condBinds), nil
}

// normalizeLet handles let bindings
func (e *Elaborator) normalizeLet(let *ast.Let) (core.CoreExpr, error) {
	// Check if it's recursive (let rec)
	isRec := false
	// For now, detect recursion by checking if the value references the name
	// This is simplified - full implementation would analyze the value expression

	if isRec {
		// Handle recursive binding
		value, err := e.normalize(let.Value)
		if err != nil {
			return nil, err
		}

		body, err := e.normalize(let.Body)
		if err != nil {
			return nil, err
		}

		return &core.LetRec{
			CoreNode: e.makeNode(let.Position()),
			Bindings: []core.RecBinding{{Name: let.Name, Value: value}},
			Body:     body,
		}, nil
	} else {
		// Non-recursive let
		value, err := e.normalize(let.Value)
		if err != nil {
			return nil, err
		}

		body, err := e.normalize(let.Body)
		if err != nil {
			return nil, err
		}

		return &core.Let{
			CoreNode: e.makeNode(let.Position()),
			Name:     let.Name,
			Value:    value,
			Body:     body,
		}, nil
	}
}

// normalizeBlock converts a block of semicolon-separated expressions
// into nested Let expressions: { e1; e2; e3 } => let _ = e1 in let _ = e2 in e3
func (e *Elaborator) normalizeBlock(block *ast.Block) (core.CoreExpr, error) {
	// Empty block: should not happen but handle gracefully
	if len(block.Exprs) == 0 {
		// Return unit literal
		return &core.Lit{
			CoreNode: e.makeNode(block.Position()),
			Kind:     core.UnitLit,
			Value:    "()",
		}, nil
	}

	// Single expression: just normalize it directly
	if len(block.Exprs) == 1 {
		return e.normalize(block.Exprs[0])
	}

	// Multiple expressions: convert to nested Lets
	// Start with the last expression (the return value)
	result, err := e.normalize(block.Exprs[len(block.Exprs)-1])
	if err != nil {
		return nil, err
	}

	// Work backwards through the expressions, wrapping each in a Let
	for i := len(block.Exprs) - 2; i >= 0; i-- {
		value, err := e.normalize(block.Exprs[i])
		if err != nil {
			return nil, err
		}

		// Use a wildcard name for the binding since we're discarding the result
		// Generate unique names to avoid conflicts
		bindingName := fmt.Sprintf("_block_%d", i)

		result = &core.Let{
			CoreNode: e.makeNode(block.Position()),
			Name:     bindingName,
			Value:    value,
			Body:     result,
		}
	}

	return result, nil
}

// normalizeFuncCall handles function application
func (e *Elaborator) normalizeFuncCall(app *ast.FuncCall) (core.CoreExpr, error) {
	// Check if this is a constructor call
	if ident, ok := app.Func.(*ast.Identifier); ok {
		if ctorInfo, isConstructor := e.constructors[ident.Name]; isConstructor {
			// This is a constructor! Emit $adt factory call
			// Transform Some(x) → $adt.make_Option_Some(x)
			factoryName := fmt.Sprintf("make_%s_%s", ctorInfo.TypeName, ctorInfo.CtorName)

			// Normalize arguments to atomic values
			var allBindings []binding
			var atomicArgs []core.CoreExpr

			for _, arg := range app.Args {
				atomic, binds, err := e.normalizeToAtomic(arg)
				if err != nil {
					return nil, err
				}
				atomicArgs = append(atomicArgs, atomic)
				allBindings = append(allBindings, binds...)
			}

			// Create factory function reference
			factoryRef := &core.VarGlobal{
				CoreNode: e.makeNode(app.Position()),
				Ref: core.GlobalRef{
					Module: "$adt",
					Name:   factoryName,
				},
			}

			result := &core.App{
				CoreNode: e.makeNode(app.Position()),
				Func:     factoryRef,
				Args:     atomicArgs,
			}

			return e.wrapWithBindings(result, allBindings), nil
		}
	}

	// Not a constructor - handle as normal function call
	fun, err := e.normalize(app.Func)
	if err != nil {
		return nil, err
	}

	var allBindings []binding
	var atomicArgs []core.CoreExpr

	for _, arg := range app.Args {
		atomic, binds, err := e.normalizeToAtomic(arg)
		if err != nil {
			return nil, err
		}
		atomicArgs = append(atomicArgs, atomic)
		allBindings = append(allBindings, binds...)
	}

	// If function is not atomic, bind it too
	if !core.IsAtomic(fun) {
		freshVar := e.freshVar()
		allBindings = append(allBindings, binding{Name: freshVar, Value: fun})
		fun = &core.Var{
			CoreNode: e.makeNode(app.Position()),
			Name:     freshVar,
		}
	}

	result := &core.App{
		CoreNode: e.makeNode(app.Position()),
		Func:     fun,
		Args:     atomicArgs,
	}

	return e.wrapWithBindings(result, allBindings), nil
}

// normalizeRecord handles record construction
func (e *Elaborator) normalizeRecord(rec *ast.Record) (core.CoreExpr, error) {
	fields := make(map[string]core.CoreExpr)
	var allBindings []binding

	for i, field := range rec.Fields {
		value := field.Value
		name := field.Name
		atomic, binds, err := e.normalizeToAtomic(value)
		if err != nil {
			return nil, err
		}
		fields[name] = atomic
		allBindings = append(allBindings, binds...)
		_ = i // use i to avoid warning
	}

	result := &core.Record{
		CoreNode: e.makeNode(rec.Position()),
		Fields:   fields,
	}

	return e.wrapWithBindings(result, allBindings), nil
}

// normalizeRecordAccess handles field access
func (e *Elaborator) normalizeRecordAccess(acc *ast.RecordAccess) (core.CoreExpr, error) {
	record, binds, err := e.normalizeToAtomic(acc.Record)
	if err != nil {
		return nil, err
	}

	result := &core.RecordAccess{
		CoreNode: e.makeNode(acc.Position()),
		Record:   record,
		Field:    acc.Field,
	}

	return e.wrapWithBindings(result, binds), nil
}

// normalizeList handles list construction
func (e *Elaborator) normalizeList(list *ast.List) (core.CoreExpr, error) {
	var elements []core.CoreExpr
	var allBindings []binding

	for _, elem := range list.Elements {
		atomic, binds, err := e.normalizeToAtomic(elem)
		if err != nil {
			return nil, err
		}
		elements = append(elements, atomic)
		allBindings = append(allBindings, binds...)
	}

	result := &core.List{
		CoreNode: e.makeNode(list.Position()),
		Elements: elements,
	}

	return e.wrapWithBindings(result, allBindings), nil
}

// normalizeTuple handles tuple construction
func (e *Elaborator) normalizeTuple(tuple *ast.Tuple) (core.CoreExpr, error) {
	var elements []core.CoreExpr
	var allBindings []binding

	for _, elem := range tuple.Elements {
		atomic, binds, err := e.normalizeToAtomic(elem)
		if err != nil {
			return nil, err
		}
		elements = append(elements, atomic)
		allBindings = append(allBindings, binds...)
	}

	result := &core.Tuple{
		CoreNode: e.makeNode(tuple.Position()),
		Elements: elements,
	}

	return e.wrapWithBindings(result, allBindings), nil
}

// normalizeMatch handles pattern matching
func (e *Elaborator) normalizeMatch(match *ast.Match) (core.CoreExpr, error) {
	// Scrutinee must be atomic
	scrutinee, binds, err := e.normalizeToAtomic(match.Expr)
	if err != nil {
		return nil, err
	}

	// Convert arms
	var arms []core.MatchArm
	for _, caseClause := range match.Cases {
		pattern, err := e.elaboratePattern(caseClause.Pattern)
		if err != nil {
			return nil, err
		}

		body, err := e.normalize(caseClause.Body)
		if err != nil {
			return nil, err
		}

		arms = append(arms, core.MatchArm{
			Pattern: pattern,
			Body:    body,
		})
	}

	result := &core.Match{
		CoreNode:   e.makeNode(match.Position()),
		Scrutinee:  scrutinee,
		Arms:       arms,
		Exhaustive: false, // Will be set by typechecker
	}

	return e.wrapWithBindings(result, binds), nil
}

// elaboratePattern converts surface pattern to core pattern
func (e *Elaborator) elaboratePattern(pat ast.Pattern) (core.CorePattern, error) {
	switch p := pat.(type) {
	case *ast.Identifier:
		return &core.VarPattern{Name: p.Name}, nil
	case *ast.Literal:
		return &core.LitPattern{Value: p.Value}, nil
	case *ast.WildcardPattern:
		return &core.WildcardPattern{}, nil
	case *ast.ConstructorPattern:
		// Elaborate nested patterns
		var args []core.CorePattern
		for _, argPat := range p.Patterns {
			coreArg, err := e.elaboratePattern(argPat)
			if err != nil {
				return nil, err
			}
			args = append(args, coreArg)
		}
		return &core.ConstructorPattern{
			Name: p.Name,
			Args: args,
		}, nil
	case *ast.TuplePattern:
		// Elaborate tuple element patterns
		var elements []core.CorePattern
		for _, elemPat := range p.Elements {
			coreElem, err := e.elaboratePattern(elemPat)
			if err != nil {
				return nil, err
			}
			elements = append(elements, coreElem)
		}
		return &core.TuplePattern{
			Elements: elements,
		}, nil
	case *ast.ListPattern:
		// Elaborate list element patterns
		var elements []core.CorePattern
		for _, elemPat := range p.Elements {
			coreElem, err := e.elaboratePattern(elemPat)
			if err != nil {
				return nil, err
			}
			elements = append(elements, coreElem)
		}

		// Elaborate rest pattern if present
		var tail *core.CorePattern
		if p.Rest != nil {
			restCore, err := e.elaboratePattern(p.Rest)
			if err != nil {
				return nil, err
			}
			tail = &restCore
		}

		return &core.ListPattern{
			Elements: elements,
			Tail:     tail,
		}, nil
	default:
		return nil, fmt.Errorf("pattern elaboration not implemented for %T", pat)
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

// Helper types and functions

type binding struct {
	Name  string
	Value core.CoreExpr
}

// normalizeToAtomic ensures expression is atomic, introducing let bindings if needed
func (e *Elaborator) normalizeToAtomic(expr ast.Expr) (core.CoreExpr, []binding, error) {
	normalized, err := e.normalize(expr)
	if err != nil {
		return nil, nil, err
	}

	if core.IsAtomic(normalized) {
		return normalized, nil, nil
	}

	// Need to bind non-atomic expression
	freshName := e.freshVar()
	bind := binding{Name: freshName, Value: normalized}
	varRef := &core.Var{
		CoreNode: e.makeNode(expr.Position()),
		Name:     freshName,
	}

	return varRef, []binding{bind}, nil
}

// wrapWithBindings wraps expression with let bindings
func (e *Elaborator) wrapWithBindings(expr core.CoreExpr, bindings []binding) core.CoreExpr {
	result := expr
	// Apply bindings in reverse order (innermost first)
	for i := len(bindings) - 1; i >= 0; i-- {
		bind := bindings[i]
		result = &core.Let{
			CoreNode: e.makeNode(bind.Value.Span()),
			Name:     bind.Name,
			Value:    bind.Value,
			Body:     result,
		}
	}
	return result
}

// makeNode creates a new CoreNode with unique ID
func (e *Elaborator) makeNode(pos ast.Pos) core.CoreNode {
	id := e.nextID
	e.nextID++
	e.surfaceSpans[id] = pos
	return core.CoreNode{
		NodeID:   id,
		CoreSpan: pos,
		OrigSpan: pos,
	}
}

// freshVar generates a fresh variable name
func (e *Elaborator) freshVar() string {
	e.freshVarNum++
	return fmt.Sprintf("$tmp%d", e.freshVarNum)
}

// GetSurfaceSpan retrieves the original surface span for a Core node ID
func (e *Elaborator) GetSurfaceSpan(nodeID uint64) (ast.Pos, bool) {
	span, ok := e.surfaceSpans[nodeID]
	return span, ok
}

// ElaborateWithDictionaries transforms operators to dictionary calls
// This is the second pass after type checking
func ElaborateWithDictionaries(prog *core.Program, resolved map[uint64]*types.ResolvedConstraint) (*core.Program, error) {
	elaborator := &DictElaborator{
		resolved:    resolved,
		freshVarNum: 0,
	}

	// Transform each declaration
	var newDecls []core.CoreExpr
	for _, decl := range prog.Decls {
		transformed := elaborator.transformExpr(decl)
		newDecls = append(newDecls, transformed)
	}

	return &core.Program{Decls: newDecls}, nil
}

// DictElaborator handles dictionary transformation
type DictElaborator struct {
	resolved    map[uint64]*types.ResolvedConstraint
	freshVarNum int
}

// transformExpr recursively transforms Core expressions
func (de *DictElaborator) transformExpr(expr core.CoreExpr) core.CoreExpr {
	if expr == nil {
		return nil
	}

	switch e := expr.(type) {
	case *core.BinOp:
		// Check if this operator has a resolved constraint
		if rc, ok := de.resolved[e.ID()]; ok && rc.Method != "" {
			// Guard against nil Type in resolved constraint
			if rc.Type == nil {
				// Skip dictionary transformation if type is nil
				return &core.BinOp{
					CoreNode: e.CoreNode,
					Op:       e.Op,
					Left:     de.transformExpr(e.Left),
					Right:    de.transformExpr(e.Right),
				}
			}

			// Transform to dictionary application
			// First transform the operands
			left := de.transformExpr(e.Left)
			right := de.transformExpr(e.Right)

			// Create dictionary reference
			typeName := types.NormalizeTypeName(rc.Type)
			// fmt.Printf("DEBUG ELABORATE: BinOp NodeID=%d, Class=%s, Type=%v, NormalizedType=%s, Method=%s\n",
			// 	e.ID(), rc.ClassName, rc.Type, typeName, rc.Method)
			dictRef := &core.DictRef{
				CoreNode:  e.CoreNode,
				ClassName: rc.ClassName,
				TypeName:  typeName,
			}

			// Create dictionary application directly

			// Build the ANF structure properly:
			// For now, just use DictApp directly with DictRef as the dictionary
			// This is valid ANF since DictRef is atomic
			return &core.DictApp{
				CoreNode: e.CoreNode,
				Dict:     dictRef,
				Method:   rc.Method,
				Args:     []core.CoreExpr{left, right},
			}
		}

		// No dictionary transformation needed, just recurse
		return &core.BinOp{
			CoreNode: e.CoreNode,
			Op:       e.Op,
			Left:     de.transformExpr(e.Left),
			Right:    de.transformExpr(e.Right),
		}

	case *core.UnOp:
		// Check if this operator has a resolved constraint
		if rc, ok := de.resolved[e.ID()]; ok && rc.Method != "" {
			// Guard against nil Type in resolved constraint
			if rc.Type == nil {
				// Skip dictionary transformation if type is nil
				return &core.UnOp{
					CoreNode: e.CoreNode,
					Op:       e.Op,
					Operand:  de.transformExpr(e.Operand),
				}
			}

			// Transform to dictionary application
			operand := de.transformExpr(e.Operand)

			// Create dictionary reference
			typeName := types.NormalizeTypeName(rc.Type)
			dictRef := &core.DictRef{
				CoreNode:  e.CoreNode,
				ClassName: rc.ClassName,
				TypeName:  typeName,
			}

			// Create dictionary application directly

			// Build ANF structure properly with DictRef directly in DictApp
			return &core.DictApp{
				CoreNode: e.CoreNode,
				Dict:     dictRef,
				Method:   rc.Method,
				Args:     []core.CoreExpr{operand},
			}
		}

		// No transformation needed
		return &core.UnOp{
			CoreNode: e.CoreNode,
			Op:       e.Op,
			Operand:  de.transformExpr(e.Operand),
		}

	case *core.Intrinsic:
		// Intrinsic nodes pass through - they'll be handled by OpLowering pass
		args := make([]core.CoreExpr, len(e.Args))
		for i, arg := range e.Args {
			args[i] = de.transformExpr(arg)
		}
		return &core.Intrinsic{
			CoreNode: e.CoreNode,
			Op:       e.Op,
			Args:     args,
		}

	case *core.Let:
		return &core.Let{
			CoreNode: e.CoreNode,
			Name:     e.Name,
			Value:    de.transformExpr(e.Value),
			Body:     de.transformExpr(e.Body),
		}

	case *core.LetRec:
		var newBindings []core.RecBinding
		for _, binding := range e.Bindings {
			newBindings = append(newBindings, core.RecBinding{
				Name:  binding.Name,
				Value: de.transformExpr(binding.Value),
			})
		}
		return &core.LetRec{
			CoreNode: e.CoreNode,
			Bindings: newBindings,
			Body:     de.transformExpr(e.Body),
		}

	case *core.Lambda:
		return &core.Lambda{
			CoreNode: e.CoreNode,
			Params:   e.Params,
			Body:     de.transformExpr(e.Body),
		}

	case *core.App:
		var newArgs []core.CoreExpr
		for _, arg := range e.Args {
			newArgs = append(newArgs, de.transformExpr(arg))
		}
		return &core.App{
			CoreNode: e.CoreNode,
			Func:     de.transformExpr(e.Func),
			Args:     newArgs,
		}

	case *core.If:
		return &core.If{
			CoreNode: e.CoreNode,
			Cond:     de.transformExpr(e.Cond),
			Then:     de.transformExpr(e.Then),
			Else:     de.transformExpr(e.Else),
		}

	case *core.Match:
		var newArms []core.MatchArm
		for _, arm := range e.Arms {
			newArms = append(newArms, core.MatchArm{
				Pattern: arm.Pattern,
				Body:    de.transformExpr(arm.Body),
			})
		}
		return &core.Match{
			CoreNode:   e.CoreNode,
			Scrutinee:  de.transformExpr(e.Scrutinee),
			Arms:       newArms,
			Exhaustive: e.Exhaustive,
		}

	case *core.Record:
		newFields := make(map[string]core.CoreExpr)
		for k, v := range e.Fields {
			newFields[k] = de.transformExpr(v)
		}
		return &core.Record{
			CoreNode: e.CoreNode,
			Fields:   newFields,
		}

	case *core.RecordAccess:
		return &core.RecordAccess{
			CoreNode: e.CoreNode,
			Record:   de.transformExpr(e.Record),
			Field:    e.Field,
		}

	case *core.List:
		var newElements []core.CoreExpr
		for _, elem := range e.Elements {
			newElements = append(newElements, de.transformExpr(elem))
		}
		return &core.List{
			CoreNode: e.CoreNode,
			Elements: newElements,
		}

	// Atomic expressions - return as is
	case *core.Var, *core.Lit, *core.DictRef:
		return expr

	// Already dictionary nodes - preserve
	case *core.DictAbs, *core.DictApp:
		return expr

	default:
		// Unknown type - return as is
		return expr
	}
}
