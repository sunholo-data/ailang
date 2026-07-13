package elaborate

import (
	"fmt"
	"os"

	"github.com/sunholo-data/ailang/internal/ast"
	"github.com/sunholo-data/ailang/internal/core"
	"github.com/sunholo-data/ailang/internal/types"
)

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

		// M-STREAM-DX/M4: Register type alias for single-constructor ADTs wrapping a record
		// This enables `t.name` field access on newtype-pattern ADTs like `type Item = Item({name: string})`
		// The unifier's expandAlias will expand TCon("Item") -> TRecord{name: string} during unification,
		// allowing TRecordOpen (from field access) to unify with the expanded record type.
		// Only applies when: exactly 1 constructor, exactly 1 field, field is a record type.
		if len(def.Constructors) == 1 && len(decl.TypeParams) == 0 {
			ctor := def.Constructors[0]
			if len(ctor.Fields) == 1 && ctor.Fields[0].Type != nil {
				if _, isRecord := ctor.Fields[0].Type.(*ast.RecordType); isRecord {
					recordType := e.astTypeToInternalType(ctor.Fields[0].Type)
					if recordType != nil {
						e.RegisterTypeAlias(typeName, recordType)
					}
				}
			}
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
			// M-XMOD-ALIAS-POLY: record the alias's type params (e.g. Box[a])
			// so applied uses (`Box[int]`) instantiate the body.
			e.RegisterTypeAliasParams(typeName, decl.TypeParams)
		}
		return nil, nil

	case *ast.TypeAlias:
		// M-BUGFIX: Register type alias for expansion during unification
		// This fixes: `type Coord = {x: int, y: int}` with `IsoTile(tile: Coord)`
		targetType := e.astTypeToInternalType(def.Target)
		if targetType != nil {
			e.RegisterTypeAlias(typeName, targetType)
			// M-XMOD-ALIAS-POLY: record the alias's type params (e.g. Ident[a],
			// Pair[a,b]) so applied uses instantiate the body.
			e.RegisterTypeAliasParams(typeName, decl.TypeParams)
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
		// Use an open effect row when the annotation doesn't specify effects.
		// A nil/closed row would mean "no effects allowed", which breaks
		// unification with functions that have effects like AI, IO, FS.
		e.freshVarNum++
		openEffectRow := &types.Row{
			Kind:   types.EffectRow,
			Labels: make(map[string]types.Type),
			Tail:   &types.RowVar{Name: fmt.Sprintf("ε_annot%d", e.freshVarNum), Kind: types.EffectRow},
		}
		return &types.TFunc2{
			Params:    params,
			EffectRow: openEffectRow,
			Return:    e.astTypeToInternalType(typ.Return),
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

	case *ast.LabelledType:
		// IFC label/refinement — strip label metadata, use base type for structural operations.
		return e.astTypeToInternalType(typ.Base)

	default:
		// Unknown type - return nil (will be handled gracefully)
		return nil
	}
}
