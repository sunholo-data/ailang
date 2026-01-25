package elaborate

import (
	"fmt"
	"os"

	"github.com/sunholo/ailang/internal/ast"
	"github.com/sunholo/ailang/internal/core"
	"github.com/sunholo/ailang/internal/types"
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
				// M-VERIFY: Elaborate contracts for this function
				contracts, err := e.elaborateContracts(astFunc.Properties)
				if err != nil {
					return nil, fmt.Errorf("elaborating contracts for %s: %w", f.Name, err)
				}
				meta[f.Name] = &core.DeclMeta{
					Name:      f.Name,
					IsExport:  astFunc.IsExport,
					IsPure:    astFunc.IsPure,
					Contracts: contracts,
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
					// M-VERIFY: Elaborate contracts for this function
					contracts, err := e.elaborateContracts(astFunc.Properties)
					if err != nil {
						return nil, fmt.Errorf("elaborating contracts for %s: %w", f.Name, err)
					}
					meta[f.Name] = &core.DeclMeta{
						Name:      f.Name,
						IsExport:  astFunc.IsExport,
						IsPure:    astFunc.IsPure,
						Contracts: contracts,
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

	// Elaborate module-level lets and wrap all function declarations in them
	// This ensures module-level lets are in scope for function bodies
	if len(moduleLets) > 0 {
		// Elaborate module-level let values once
		elaboratedLets := make([]struct {
			Name  string
			Value core.CoreExpr
			Pos   ast.Pos
		}, len(moduleLets))
		for i, ml := range moduleLets {
			value, err := e.elaborateExpr(ml.Value)
			if err != nil {
				return nil, fmt.Errorf("error elaborating module-level let '%s': %w", ml.Name, err)
			}
			elaboratedLets[i] = struct {
				Name  string
				Value core.CoreExpr
				Pos   ast.Pos
			}{ml.Name, value, ml.Pos}
		}

		// Wrap each function declaration in the module-level lets
		for i, decl := range coreDecls {
			coreDecls[i] = e.wrapInLets(decl, elaboratedLets)
		}
	}

	// Add any non-func, non-let statements (e.g., main() call)
	for _, stmt := range file.Statements {
		if expr, ok := stmt.(ast.Expr); ok {
			// Skip let expressions - they're already processed above
			if _, isLet := expr.(*ast.Let); isLet {
				continue
			}
			coreExpr, err := e.elaborateExpr(expr)
			if err != nil {
				return nil, err
			}
			// Wrap non-func statements in module-level lets too
			if len(moduleLets) > 0 {
				// Re-elaborate for wrapping (values are simple, this is fine)
				elaboratedLets := make([]struct {
					Name  string
					Value core.CoreExpr
					Pos   ast.Pos
				}, len(moduleLets))
				for i, ml := range moduleLets {
					value, _ := e.elaborateExpr(ml.Value)
					elaboratedLets[i] = struct {
						Name  string
						Value core.CoreExpr
						Pos   ast.Pos
					}{ml.Name, value, ml.Pos}
				}
				coreExpr = e.wrapInLets(coreExpr, elaboratedLets)
			}
			coreDecls = append(coreDecls, coreExpr)
		}
	}

	return &core.Program{Decls: coreDecls, Meta: meta}, nil
}

// wrapInLets wraps a Core expression in a series of let bindings
// The lets are applied in order, so the innermost let is the last in the slice
func (e *Elaborator) wrapInLets(expr core.CoreExpr, lets []struct {
	Name  string
	Value core.CoreExpr
	Pos   ast.Pos
}) core.CoreExpr {
	result := expr
	// Wrap in reverse order so first let is outermost
	for i := len(lets) - 1; i >= 0; i-- {
		l := lets[i]
		result = &core.Let{
			CoreNode: e.makeNode(l.Pos),
			Name:     l.Name,
			Value:    l.Value,
			Body:     result,
		}
	}
	return result
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

	// M-FIX-FLOAT-OP: Preserve parameter type annotations for the type checker
	// This ensures float annotations aren't lost during elaboration
	if f.FuncDecl != nil && len(f.FuncDecl.Params) > 0 {
		paramTypes := make([]types.Type, len(f.FuncDecl.Params))
		hasAnnotations := false
		for i, param := range f.FuncDecl.Params {
			if param.Type != nil {
				paramTypes[i] = e.astTypeToInternalType(param.Type)
				hasAnnotations = true
				if os.Getenv("DEBUG_PARAM_ANNOTS") != "" {
					fmt.Fprintf(os.Stderr, "[DEBUG] funcToLambda %s: param %s has type %v\n",
						f.Name, param.Name, paramTypes[i])
				}
			}
		}
		if hasAnnotations {
			e.paramTypeAnnots[lambda.ID()] = paramTypes
			if os.Getenv("DEBUG_PARAM_ANNOTS") != "" {
				fmt.Fprintf(os.Stderr, "[DEBUG] Stored param annotations for Lambda ID %d, func %s: %v\n",
					lambda.ID(), f.Name, paramTypes)
			}
		}
	}

	// M-FIX-FLOAT-OP: Preserve return type annotations for the type checker
	// This ensures PI() -> float ACTUALLY constrains inference to return float
	if f.FuncDecl != nil && f.FuncDecl.ReturnType != nil {
		returnType := e.astTypeToInternalType(f.FuncDecl.ReturnType)
		if returnType != nil {
			e.returnTypeAnnots[lambda.ID()] = returnType
			if os.Getenv("DEBUG_PARAM_ANNOTS") != "" {
				fmt.Fprintf(os.Stderr, "[DEBUG] funcToLambda %s: stored return type annotation %v for Lambda ID %d\n",
					f.Name, returnType, lambda.ID())
			}
		}
	}

	// M-CAPABILITY-BUDGETS: Preserve effect annotations for the type checker
	// This ensures @limit=N budget annotations are preserved through elaboration
	if f.FuncDecl != nil && len(f.FuncDecl.Effects) > 0 {
		effectNames := ast.EffectNames(f.FuncDecl.Effects)
		e.effectAnnots[lambda.ID()] = effectNames
		e.effectAnnotsFull[lambda.ID()] = f.FuncDecl.Effects
	}

	return lambda, nil
}

// elaborateContracts elaborates contract properties to Core contracts.
// M-VERIFY: Contract expressions reference function parameters by name.
func (e *Elaborator) elaborateContracts(props []*ast.Property) ([]*core.Contract, error) {
	if len(props) == 0 {
		return nil, nil
	}

	var contracts []*core.Contract
	for _, prop := range props {
		// Only process requires/ensures contracts (skip forall properties)
		if prop.Kind != ast.RequiresKind && prop.Kind != ast.EnsuresKind && prop.Kind != ast.InvariantKind {
			continue
		}

		// Elaborate the predicate expression to Core
		coreExpr, err := e.elaborateExpr(prop.Expr)
		if err != nil {
			return nil, fmt.Errorf("elaborating contract: %w", err)
		}

		// Map AST kind to Core kind
		var kind core.ContractKind
		switch prop.Kind {
		case ast.RequiresKind:
			kind = core.RequiresKind
		case ast.EnsuresKind:
			kind = core.EnsuresKind
		case ast.InvariantKind:
			kind = core.InvariantKind
		}

		// Generate message from expression if not provided
		message := prop.Name
		if message == "" {
			message = prop.Expr.String()
		}

		// Format location
		location := fmt.Sprintf("%s:%d:%d", prop.Pos.File, prop.Pos.Line, prop.Pos.Column)

		contracts = append(contracts, &core.Contract{
			Kind:     kind,
			Expr:     coreExpr,
			Message:  message,
			Location: location,
		})
	}

	return contracts, nil
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
// 3. M-DX19: Track types with `deriving (Eq)` for automatic Eq instance generation
func (e *Elaborator) elaborateTypeDecl(decl *ast.TypeDecl) (core.CoreExpr, error) {
	// Extract type name
	typeName := decl.Name

	// M-DX19: Check for deriving (Eq) clause
	hasDerivingEq := false
	for _, d := range decl.Deriving {
		if d == ast.DeriveEq {
			hasDerivingEq = true
			break
		}
	}

	// M-DX19: Validate deriving (Eq) constraints
	if hasDerivingEq {
		// Reject polymorphic types (require typeclass constraints which aren't supported yet)
		if len(decl.TypeParams) > 0 {
			return nil, fmt.Errorf("cannot derive Eq for polymorphic type %s[%s] without Eq constraints (deferred to v0.7+)",
				typeName, decl.TypeParams[0])
		}
		// Record this type as having derived Eq
		e.derivedEqTypes[typeName] = true
	}

	// Process the type definition
	switch def := decl.Definition.(type) {
	case *ast.AlgebraicType:
		// Process each constructor in the ADT
		// M-TAPP-FIX: Track type parameter count for proper TApp generation
		typeParamCount := len(decl.TypeParams)
		for _, ctor := range def.Constructors {
			// M-POLY-ADT: Convert AST field types to internal types
			// This is critical for correctly typing constructors like Err(string) in Result[a]
			var fieldTypes []types.Type
			for _, field := range ctor.Fields {
				if field.Type != nil {
					fieldTypes = append(fieldTypes, e.astTypeToInternalType(field.Type))
				} else {
					fieldTypes = append(fieldTypes, nil)
				}
			}
			// Register constructor with actual field types and type param names
			e.RegisterConstructorWithFields(typeName, ctor.Name, len(ctor.Fields), false, typeParamCount, decl.TypeParams, fieldTypes)
		}
		// Type declarations don't produce code, return nil
		return nil, nil

	case *ast.RecordType:
		// M-FIX-RECORD-UPDATE: Register record type aliases for expansion during unification
		// This allows `type NPC = { pos: Pos, name: string }` to work with record update
		// When we have `{ npc | pos: ... }`, unification needs to expand NPC to its record type
		recordType := e.astTypeToInternalType(def)
		if recordType != nil {
			e.RegisterTypeAlias(typeName, recordType)
		}
		return nil, nil

	case *ast.TypeAlias:
		// M-BUGFIX: Register type alias for expansion during unification
		// This fixes: `type Coord = {x: int, y: int}` with `IsoTile(tile: Coord)`
		targetType := e.astTypeToInternalType(def.Target)
		if targetType != nil {
			e.RegisterTypeAlias(typeName, targetType)
		}
		return nil, nil

	default:
		return nil, fmt.Errorf("unknown type definition: %T", def)
	}
}

// elaborateFuncDecl handles function declarations
func (e *Elaborator) elaborateFuncDecl(fn *ast.FuncDecl) (core.CoreExpr, error) {
	// Debug trace entry
	if os.Getenv("DEBUG_PARAM_ANNOTS") != "" {
		fmt.Fprintf(os.Stderr, "[DEBUG] elaborateFuncDecl called for %s, params=%d\n", fn.Name, len(fn.Params))
	}

	// Convert to lambda
	// M-CAPABILITY-BUDGETS: Copy effect annotations from FuncDecl to Lambda
	lambda := &ast.Lambda{
		Params:  fn.Params,
		Body:    fn.Body,
		Effects: fn.Effects, // Preserve effect annotations including budgets
		Pos:     fn.Pos,
	}

	value, err := e.normalizeLambda(lambda)
	if err != nil {
		return nil, err
	}

	// M-FIX-FLOAT-OP: Preserve parameter type annotations for the type checker
	// This ensures float annotations aren't lost during elaboration
	if os.Getenv("DEBUG_PARAM_ANNOTS") != "" {
		fmt.Fprintf(os.Stderr, "[DEBUG] value type: %T\n", value)
	}
	if coreLambda, ok := value.(*core.Lambda); ok {
		paramTypes := make([]types.Type, len(fn.Params))
		hasAnnotations := false
		for i, param := range fn.Params {
			if param.Type != nil {
				paramTypes[i] = e.astTypeToInternalType(param.Type)
				hasAnnotations = true
				if os.Getenv("DEBUG_PARAM_ANNOTS") != "" {
					fmt.Fprintf(os.Stderr, "[DEBUG]   param %s has type annotation: %v\n", param.Name, paramTypes[i])
				}
			}
		}
		if hasAnnotations {
			e.paramTypeAnnots[coreLambda.ID()] = paramTypes
			// Debug trace
			if os.Getenv("DEBUG_PARAM_ANNOTS") != "" {
				fmt.Fprintf(os.Stderr, "[DEBUG] Stored param annotations for Lambda ID %d, func %s: %v\n",
					coreLambda.ID(), fn.Name, paramTypes)
			}
		}

		// M-FIX-FLOAT-OP: Preserve return type annotations for the type checker
		// This ensures PI() -> float ACTUALLY constrains inference to return float
		if fn.ReturnType != nil {
			returnType := e.astTypeToInternalType(fn.ReturnType)
			if returnType != nil {
				e.returnTypeAnnots[coreLambda.ID()] = returnType
				if os.Getenv("DEBUG_PARAM_ANNOTS") != "" {
					fmt.Fprintf(os.Stderr, "[DEBUG] elaborateFuncDecl %s: stored return type annotation %v for Lambda ID %d\n",
						fn.Name, returnType, coreLambda.ID())
				}
			}
		}
	}

	// Wrap in let rec if recursive
	return &core.LetRec{
		CoreNode: e.makeNode(fn.Position()),
		Bindings: []core.RecBinding{{Name: fn.Name, Value: value}},
		Body:     &core.Var{CoreNode: e.makeNode(fn.Position()), Name: fn.Name},
	}, nil
}

// astTypeToInternalType converts an AST type to an internal types.Type
// M-BUGFIX: Used for type alias registration during elaboration
func (e *Elaborator) astTypeToInternalType(t ast.Type) types.Type {
	if t == nil {
		return nil
	}

	switch typ := t.(type) {
	case *ast.SimpleType:
		switch typ.Name {
		case "int":
			return types.TInt
		case "float":
			return types.TFloat
		case "string":
			return types.TString
		case "bool":
			return types.TBool
		case "()", "unit":
			return types.TUnit
		case "bytes":
			return types.TBytes
		default:
			// Type constructor (e.g., user-defined ADT)
			return &types.TCon{Name: typ.Name}
		}

	case *ast.TypeVar:
		// M-FIX-FLOAT-OP: Handle type variables in function type annotations
		// e.g., in `func map[a,b](f: (a) -> b, ...)` the `a` and `b` are TypeVars
		return &types.TVar2{Name: typ.Name, Kind: types.Star}

	case *ast.RecordType:
		// Convert record type to TRecord or TRecordOpen
		fields := make(map[string]types.Type)
		for _, field := range typ.Fields {
			fields[field.Name] = e.astTypeToInternalType(field.Type)
		}
		// M-GAP4: If row variable present, create open record type
		if typ.Row != nil {
			rowVar := &types.RowVar{Name: typ.Row.Name, Kind: &types.KRow{ElemKind: &types.KRecord{}}}
			return &types.TRecordOpen{Fields: fields, Row: rowVar}
		}
		return &types.TRecord{Fields: fields}

	case *ast.ListType:
		return &types.TList{Element: e.astTypeToInternalType(typ.Element)}

	case *ast.ArrayType:
		return &types.TArray{Element: e.astTypeToInternalType(typ.Element)}

	case *ast.TupleType:
		elements := make([]types.Type, len(typ.Elements))
		for i, elem := range typ.Elements {
			elements[i] = e.astTypeToInternalType(elem)
		}
		return &types.TTuple{Elements: elements}

	case *ast.FuncType:
		params := make([]types.Type, len(typ.Params))
		for i, p := range typ.Params {
			params[i] = e.astTypeToInternalType(p)
		}
		return &types.TFunc{
			Params: params,
			Return: e.astTypeToInternalType(typ.Return),
		}

	case *ast.TypeApp:
		// M-TAPP-FIX: Handle type application (e.g., Option[int], Result[T, E])
		// Convert to internal TApp type with constructor and args
		args := make([]types.Type, len(typ.Args))
		for i, arg := range typ.Args {
			args[i] = e.astTypeToInternalType(arg)
		}
		return &types.TApp{
			Constructor: &types.TCon{Name: typ.Constructor},
			Args:        args,
		}

	default:
		// Unknown type - return nil (will be handled gracefully)
		return nil
	}
}
