package testing

import (
	"github.com/sunholo-data/ailang/internal/ast"
	"github.com/sunholo-data/ailang/internal/eval"
)

// maxDeriveDepth is the default recursion depth budget for structural type
// derivation. The design doc recommends a default of 3 (see
// m-property-generator-coverage.md, "Recursion bound"). M3 threads the budget
// through record/tuple/named-type descent so mutually-referential types cannot
// recurse unboundedly; M4 adds the ADT-aware behaviour at the bound.
const maxDeriveDepth = 3

// deriveCtx carries the generation budget through structural type derivation.
//
// depth is decremented on every derivation step, bounding recursion through
// mutually-referential types (record → list → named type → record ...). At the
// bound the derivation fails (nil, nil) — an honest vacuous skip, never a
// fabricated generator.
//
// sizeBudget bounds collection lengths inside a derivation. M3 carries it for
// M4's derivation-internal list cap (F-4); top-level list[<scalar>] keeps
// today's 0..MaxSize so Lane A's measured behaviour is unchanged.
type deriveCtx struct {
	depth         int
	sizeBudget    int
	substitutions map[string]ast.Type
	recursionMemo map[*ast.TypeDecl][]bool
}

// newDeriveCtx returns the root derivation context: the max derivation depth
// and the default generation size budget.
func newDeriveCtx() deriveCtx {
	config := DefaultConfig()
	return deriveCtx{
		depth:         maxDeriveDepth,
		sizeBudget:    config.MaxSize,
		substitutions: make(map[string]ast.Type),
		recursionMemo: make(map[*ast.TypeDecl][]bool),
	}
}

// descend returns the context for one derivation step deeper, decrementing the
// depth budget.
func (c deriveCtx) descend() deriveCtx {
	c.depth--
	return c
}

// exhausted reports whether the depth budget is spent; callers must then stop
// deriving and return nil, nil.
func (c deriveCtx) exhausted() bool {
	return c.depth <= 0
}

// createGeneratorForType creates a generator and shrinker for the given type.
//
// It is the derivation entry point wired to all three generator call sites —
// runProperty (forall) and runRequiresProperty in runner.go, runEnsuresProperty
// in contract_domain.go — through the genForTypeSeam, so no further wiring is
// needed when new arms are added (M3 wires records/tuples/unit/aliases; M4 adds
// ADTs and user-type TypeApp substitution).
//
// It is a thin wrapper over deriveType with a fresh root context, so every
// recursive descent is budgeted.
func (r *Runner) createGeneratorForType(typ ast.Type) (Generator, Shrinker) {
	return r.deriveType(typ, newDeriveCtx())
}

// deriveType derives a generator and shrinker for typ within ctx.
//
// M3 arms: scalars, unit (), anonymous/nested record types, tuple types, and
// same-file named type declarations (records and aliases). AlgebraicType (ADTs)
// and TypeApp over user types are intentionally M4 and fall through to
// nil, nil until then.
func (r *Runner) deriveType(typ ast.Type, ctx deriveCtx) (Generator, Shrinker) {
	if ctx.exhausted() {
		return nil, nil
	}
	if typeVar, ok := typ.(*ast.TypeVar); ok {
		replacement, found := ctx.substitutions[typeVar.Name]
		if !found {
			return nil, nil
		}
		return r.deriveType(replacement, ctx)
	}

	// Check for simple types
	if simpleType, ok := typ.(*ast.SimpleType); ok {
		if replacement, ok := ctx.substitutions[simpleType.Name]; ok {
			return r.deriveType(replacement, ctx)
		}
		switch simpleType.Name {
		case "int":
			config := DefaultConfig()
			return NewIntGenerator(config.MinInt, config.MaxInt), NewIntShrinker()
		case "float":
			config := DefaultConfig()
			return NewFloatGenerator(config.MinFloat, config.MaxFloat), NewFloatShrinker()
		case "bool":
			return NewBoolGenerator(), NewNoOpShrinker()
		case "string":
			config := DefaultConfig()
			return NewStringGenerator(0, config.MaxSize, ""), NewStringShrinker()
		case "()":
			// Unit: zero-arg functions parse to a `_ : ()` parameter
			// (parser_func.go S-CALL0 convention), and `()` in parameter
			// position parses to SimpleType{Name:"()"} (§1.5).
			return NewConstantGenerator(&eval.UnitValue{}), NewNoOpShrinker()
		}
		// Named type: same-file TypeDecl lookup. Imported types (no decl, or no
		// sourceFile) stay honest vacuous skips — Lane A already makes them loud.
		return r.deriveNamedType(simpleType.Name, ctx)
	}

	// Anonymous record types, including nested ones
	// ({ name: string, pos: { x: int, y: int } }).
	if recordType, ok := typ.(*ast.RecordType); ok {
		return r.deriveRecordType(recordType, ctx)
	}

	// Tuple types: (int, string).
	if tupleType, ok := typ.(*ast.TupleType); ok {
		elemGens := make([]Generator, 0, len(tupleType.Elements))
		for _, elem := range tupleType.Elements {
			elemGen, _ := r.deriveType(elem, ctx.descend())
			if elemGen == nil {
				// No silent substitution (CLAUDE.md §2): an underivable element
				// makes the whole tuple underivable.
				return nil, nil
			}
			elemGens = append(elemGens, elemGen)
		}
		return NewTupleGenerator(elemGens), NewNoOpShrinker()
	}

	// The parser represents both [a] and list[a] as TypeApp.
	// Element derivation must stay inside this derivation's depth budget
	// (deriveType, not createGeneratorForType): a fresh root context here lets
	// a record-type cycle routed through a list reset the budget every pass
	// and overflow the stack — reachable since M3 made named types derivable.
	if app, ok := typ.(*ast.TypeApp); ok && app.Constructor == "list" && len(app.Args) == 1 {
		elemGen, elemShrink := r.deriveType(app.Args[0], ctx.descend())
		if elemGen == nil {
			return nil, nil
		}
		return NewListGenerator(elemGen, 0, listMaxLen(ctx, app.Args[0])), NewListShrinker(elemShrink)
	}

	// A user type application substitutes its arguments for the declaration's
	// type parameters before deriving the definition. The builtin list arm must
	// remain above this lookup.
	if app, ok := typ.(*ast.TypeApp); ok {
		decl := r.findTypeDecl(app.Constructor)
		if decl == nil || len(decl.TypeParams) != len(app.Args) {
			return nil, nil
		}
		next := ctx.descend()
		next.substitutions = cloneSubstitutions(ctx.substitutions)
		for i, param := range decl.TypeParams {
			next.substitutions[param] = app.Args[i]
		}
		return r.deriveTypeDef(decl, next)
	}

	// Check for list types [a]
	if listType, ok := typ.(*ast.ListType); ok {
		// Create generator for element type (same budget rule as the TypeApp arm).
		elemGen, elemShrink := r.deriveType(listType.Element, ctx.descend())
		if elemGen == nil {
			return nil, nil
		}
		return NewListGenerator(elemGen, 0, listMaxLen(ctx, listType.Element)), NewListShrinker(elemShrink)
	}

	// Unsupported type
	return nil, nil
}

func cloneSubstitutions(src map[string]ast.Type) map[string]ast.Type {
	dst := make(map[string]ast.Type, len(src))
	for name, typ := range src {
		dst[name] = typ
	}
	return dst
}

func listMaxLen(ctx deriveCtx, elem ast.Type) int {
	config := DefaultConfig()
	if ctx.depth == maxDeriveDepth && isScalarType(elem) {
		return config.MaxSize
	}
	maxLen := ctx.depth
	if maxLen > ctx.sizeBudget {
		maxLen = ctx.sizeBudget
	}
	if maxLen < 0 {
		return 0
	}
	return maxLen
}

func isScalarType(typ ast.Type) bool {
	simple, ok := typ.(*ast.SimpleType)
	if !ok {
		return false
	}
	switch simple.Name {
	case "int", "float", "bool", "string", "()":
		return true
	default:
		return false
	}
}

// deriveRecordType derives a record generator over per-field derived
// generators. Any underivable field makes the whole record underivable
// (nil, nil) — no silent substitution (CLAUDE.md §2).
func (r *Runner) deriveRecordType(recordType *ast.RecordType, ctx deriveCtx) (Generator, Shrinker) {
	if ctx.exhausted() {
		return nil, nil
	}
	fieldGens := make(map[string]Generator, len(recordType.Fields))
	for _, field := range recordType.Fields {
		fieldGen, _ := r.deriveType(field.Type, ctx.descend())
		if fieldGen == nil {
			return nil, nil
		}
		fieldGens[field.Name] = fieldGen
	}
	return NewRecordGenerator(fieldGens), NewNoOpShrinker()
}

// deriveNamedType resolves a same-file named type (a SimpleType whose name is
// not a scalar, e.g. `Point` in `p: Point`) to its TypeDecl and derives from
// the declaration's definition. A nil executor sourceFile, an unresolvable
// name, or a definition kind M3 does not derive (AlgebraicType, ...) all
// return nil, nil — imported types stay honest vacuous skips (loud per Lane A).
func (r *Runner) deriveNamedType(name string, ctx deriveCtx) (Generator, Shrinker) {
	if ctx.exhausted() {
		return nil, nil
	}
	if decl := r.findTypeDecl(name); decl != nil {
		return r.deriveTypeDef(decl, ctx)
	}
	return nil, nil
}

func (r *Runner) findTypeDecl(name string) *ast.TypeDecl {
	if r.executor == nil || r.executor.sourceFile == nil {
		return nil
	}
	for _, node := range r.executor.sourceFile.Decls {
		if decl, ok := node.(*ast.TypeDecl); ok && decl.Name == name {
			return decl
		}
	}
	return nil
}

// deriveTypeDef derives a generator from a TypeDecl's Definition.
//
// M3 handles RecordType definitions and TypeAlias targets — every
// `type X = ...` alias (record, tuple, list, scalar) recurses on .Target (§1.5:
// the design doc omits TypeAlias entirely, but without it every alias-typed
// parameter stays vacuous). AlgebraicType (ADTs) is M4 and stays nil.
func (r *Runner) deriveTypeDef(decl *ast.TypeDecl, ctx deriveCtx) (Generator, Shrinker) {
	if ctx.exhausted() {
		return nil, nil
	}
	switch def := decl.Definition.(type) {
	case *ast.RecordType:
		return r.deriveRecordType(def, ctx)
	case *ast.TypeAlias:
		// Recurse on the alias target.
		return r.deriveType(def.Target, ctx.descend())
	case *ast.AlgebraicType:
		return r.deriveAlgebraicType(decl, def, ctx)
	default:
		// AlgebraicType and anything else: M4.
		return nil, nil
	}
}

func (r *Runner) deriveAlgebraicType(decl *ast.TypeDecl, def *ast.AlgebraicType, ctx deriveCtx) (Generator, Shrinker) {
	recursive, ok := ctx.recursionMemo[decl]
	if !ok {
		recursive = make([]bool, len(def.Constructors))
		for i, ctor := range def.Constructors {
			for _, field := range ctor.Fields {
				if r.typeReferencesDecl(field.Type, decl.Name, map[string]bool{}) {
					recursive[i] = true
					break
				}
			}
		}
		ctx.recursionMemo[decl] = recursive
	}

	modulePath := ""
	if r.executor != nil && r.executor.sourceFile != nil && r.executor.sourceFile.Module != nil {
		modulePath = r.executor.sourceFile.Module.Path
	}
	constructors := make([]Generator, 0, len(def.Constructors))
	for i, ctor := range def.Constructors {
		if ctx.depth <= 1 && recursive[i] {
			continue
		}
		fieldCtx := ctx.descend()
		if ctx.depth <= 1 {
			fieldCtx = ctx
		}
		fieldGens := make([]Generator, 0, len(ctor.Fields))
		for _, field := range ctor.Fields {
			fieldGen, _ := r.deriveType(field.Type, fieldCtx)
			if fieldGen == nil {
				return nil, nil
			}
			fieldGens = append(fieldGens, fieldGen)
		}
		constructors = append(constructors,
			NewADTGenerator(ctor.Name, fieldGens, true, modulePath, decl.Name))
	}
	if len(constructors) == 0 {
		return nil, nil
	}
	return NewOneOfGenerator(constructors...), NewNoOpShrinker()
}

func (r *Runner) typeReferencesDecl(typ ast.Type, target string, visiting map[string]bool) bool {
	switch t := typ.(type) {
	case *ast.SimpleType:
		if t.Name == target {
			return true
		}
		if visiting[t.Name] {
			return false
		}
		decl := r.findTypeDecl(t.Name)
		if decl == nil {
			return false
		}
		visiting[t.Name] = true
		found := r.typeDefReferencesDecl(decl.Definition, target, visiting)
		delete(visiting, t.Name)
		return found
	case *ast.TypeApp:
		if t.Constructor == target {
			return true
		}
		for _, arg := range t.Args {
			if r.typeReferencesDecl(arg, target, visiting) {
				return true
			}
		}
	case *ast.ListType:
		return r.typeReferencesDecl(t.Element, target, visiting)
	case *ast.RecordType:
		for _, field := range t.Fields {
			if r.typeReferencesDecl(field.Type, target, visiting) {
				return true
			}
		}
	case *ast.TupleType:
		for _, elem := range t.Elements {
			if r.typeReferencesDecl(elem, target, visiting) {
				return true
			}
		}
	}
	return false
}

func (r *Runner) typeDefReferencesDecl(def ast.TypeDef, target string, visiting map[string]bool) bool {
	switch d := def.(type) {
	case *ast.TypeAlias:
		return r.typeReferencesDecl(d.Target, target, visiting)
	case *ast.RecordType:
		return r.typeReferencesDecl(d, target, visiting)
	case *ast.AlgebraicType:
		for _, ctor := range d.Constructors {
			for _, field := range ctor.Fields {
				if r.typeReferencesDecl(field.Type, target, visiting) {
					return true
				}
			}
		}
	}
	return false
}
