// Package lower transforms Core IR into Statement IR.
//
// This is a PROJECTION, not a translation: information is deliberately lost
// (type variables, effects, row polymorphism) because target languages cannot
// express these concepts. The loss is explicit and controlled.
//
// No file in this package may import internal/gen/golang/.
package lower

import (
	"sort"

	"github.com/sunholo/ailang/internal/ast"
	"github.com/sunholo/ailang/internal/gen/stmt"
	"github.com/sunholo/ailang/internal/types"
)

// ProjectType converts a types.Type (from the type checker) into a
// stmt.ResolvedType (for the emitter). This is the semantic boundary
// where AILANG's rich type system is projected into target-language types.
//
// Rules:
//   - Primitive types (int, float, bool, string, ()) → PrimitiveType
//   - Named types (ADTs, records) → NamedType with Pointer=true for ADTs
//   - List/Array → SliceType
//   - Tuple → TupleType
//   - Function → FuncType
//   - Type variables → InterfaceType (erased — the type was polymorphic)
//   - Unknown/nil → InterfaceType (defensive)
func ProjectType(t types.Type) stmt.ResolvedType {
	if t == nil {
		return stmt.InterfaceType{}
	}
	return projectType(t, 0)
}

const maxTypeDepth = 50 // cycle guard

func projectType(t types.Type, depth int) stmt.ResolvedType {
	if depth > maxTypeDepth {
		return stmt.InterfaceType{} // break infinite recursion on cyclic types
	}
	if t == nil {
		return stmt.InterfaceType{}
	}

	switch t := t.(type) {
	case *types.TCon:
		return projectTCon(t)

	case *types.TVar:
		// Type variable survived to codegen → polymorphic position.
		// Erase to interface{}.
		return stmt.InterfaceType{}

	case *types.TList:
		return stmt.SliceType{Elem: projectType(t.Element, depth+1)}

	case *types.TArray:
		return stmt.SliceType{Elem: projectType(t.Element, depth+1)}

	case *types.TTuple:
		elems := make([]stmt.ResolvedType, len(t.Elements))
		for i, e := range t.Elements {
			elems[i] = projectType(e, depth+1)
		}
		return stmt.TupleType{Elems: elems}

	case *types.TRecord:
		if t.TypeName != "" {
			// Nominal record type — use the name.
			return stmt.NamedType{Name: t.TypeName, Pointer: false}
		}
		// Structural record without a name — best we can do is interface{}.
		return stmt.InterfaceType{}

	case *types.TRecordOpen:
		// Open record (row-polymorphic) — erase to interface{}.
		return stmt.InterfaceType{}

	case *types.TFunc2:
		params := make([]stmt.ResolvedType, len(t.Params))
		for i, p := range t.Params {
			params[i] = projectType(p, depth+1)
		}
		ret := projectType(t.Return, depth+1)
		return stmt.FuncType{Params: params, Return: ret}

	case *types.TApp:
		return projectTApp(t, depth)

	case *types.TMap:
		// Maps don't have a direct Statement IR representation yet.
		// Project as interface{} for now.
		return stmt.InterfaceType{}

	default:
		// Unknown type variant — defensive fallback.
		return stmt.InterfaceType{}
	}
}

// projectTCon maps type constructors to primitives or named types.
func projectTCon(t *types.TCon) stmt.ResolvedType {
	switch t.Name {
	case "int":
		return stmt.PrimitiveType{Kind: stmt.PrimInt}
	case "float":
		return stmt.PrimitiveType{Kind: stmt.PrimFloat}
	case "bool":
		return stmt.PrimitiveType{Kind: stmt.PrimBool}
	case "string":
		return stmt.PrimitiveType{Kind: stmt.PrimString}
	case "()", "unit":
		return stmt.PrimitiveType{Kind: stmt.PrimUnit}
	case "bytes":
		// bytes → []byte in Go, but for Statement IR treat as SliceType of int.
		return stmt.SliceType{Elem: stmt.PrimitiveType{Kind: stmt.PrimInt}}
	default:
		// User-defined ADT or opaque type. ADTs use pointer semantics.
		return stmt.NamedType{Name: t.Name, Pointer: true}
	}
}

// projectTApp handles generic type application like Option[int], Result[T, E].
func projectTApp(t *types.TApp, depth int) stmt.ResolvedType {
	// Extract constructor name.
	conName := ""
	if con, ok := t.Constructor.(*types.TCon); ok {
		conName = con.Name
	}

	switch conName {
	case "list":
		// list[T] → SliceType
		if len(t.Args) == 1 {
			return stmt.SliceType{Elem: projectType(t.Args[0], depth+1)}
		}
	case "Array":
		// Array[T] → SliceType
		if len(t.Args) == 1 {
			return stmt.SliceType{Elem: projectType(t.Args[0], depth+1)}
		}
	}

	// Generic ADT application (e.g., Option[int], Result[string, Error]).
	// Project as the named type — monomorphization has already specialized it.
	if conName != "" {
		return stmt.NamedType{Name: conName, Pointer: true}
	}

	return stmt.InterfaceType{}
}

// ProjectASTType converts an ast.Type (from surface syntax) into a
// stmt.ResolvedType. Used for type declarations where we have AST types
// rather than inferred types.Type.
func ProjectASTType(t ast.Type) stmt.ResolvedType {
	if t == nil {
		return stmt.InterfaceType{}
	}
	return projectASTType(t, 0)
}

func projectASTType(t ast.Type, depth int) stmt.ResolvedType {
	if depth > maxTypeDepth || t == nil {
		return stmt.InterfaceType{}
	}

	switch t := t.(type) {
	case *ast.SimpleType:
		return projectSimpleType(t.Name)

	case *ast.TypeVar:
		// Type parameter in a declaration — erase.
		return stmt.InterfaceType{}

	case *ast.FuncType:
		params := make([]stmt.ResolvedType, len(t.Params))
		for i, p := range t.Params {
			params[i] = projectASTType(p, depth+1)
		}
		ret := projectASTType(t.Return, depth+1)
		return stmt.FuncType{Params: params, Return: ret}

	case *ast.ListType:
		return stmt.SliceType{Elem: projectASTType(t.Element, depth+1)}

	case *ast.ArrayType:
		return stmt.SliceType{Elem: projectASTType(t.Element, depth+1)}

	case *ast.TupleType:
		elems := make([]stmt.ResolvedType, len(t.Elements))
		for i, e := range t.Elements {
			elems[i] = projectASTType(e, depth+1)
		}
		return stmt.TupleType{Elems: elems}

	case *ast.TypeApp:
		return projectSimpleType(t.Constructor)

	case *ast.RecordType:
		// Structural record in AST — no name available.
		return stmt.InterfaceType{}

	default:
		return stmt.InterfaceType{}
	}
}

func projectSimpleType(name string) stmt.ResolvedType {
	switch name {
	case "int":
		return stmt.PrimitiveType{Kind: stmt.PrimInt}
	case "float":
		return stmt.PrimitiveType{Kind: stmt.PrimFloat}
	case "bool":
		return stmt.PrimitiveType{Kind: stmt.PrimBool}
	case "string":
		return stmt.PrimitiveType{Kind: stmt.PrimString}
	case "()", "unit":
		return stmt.PrimitiveType{Kind: stmt.PrimUnit}
	default:
		return stmt.NamedType{Name: name, Pointer: true}
	}
}

// LowerTypeDecl converts an AST type declaration into a Statement IR TypeDecl.
func LowerTypeDecl(td *ast.TypeDecl) stmt.TypeDecl {
	result := stmt.TypeDecl{
		Name:     td.Name,
		Exported: td.Exported,
	}

	switch def := td.Definition.(type) {
	case *ast.AlgebraicType:
		variants := make([]stmt.ADTVariant, len(def.Constructors))
		for i, ctor := range def.Constructors {
			fields := make([]stmt.ResolvedType, len(ctor.Fields))
			for j, f := range ctor.Fields {
				fields[j] = ProjectASTType(f.Type)
			}
			variants[i] = stmt.ADTVariant{
				Tag:    ctor.Name,
				Fields: fields,
			}
		}
		result.Kind = stmt.ADTDecl{Variants: variants}

	case *ast.RecordType:
		fields := make([]stmt.RecordField, len(def.Fields))
		for i, f := range def.Fields {
			fields[i] = stmt.RecordField{
				Name: f.Name,
				Type: ProjectASTType(f.Type),
			}
		}
		// Sort fields for deterministic output.
		sort.Slice(fields, func(i, j int) bool {
			return fields[i].Name < fields[j].Name
		})
		result.Kind = stmt.RecordDecl{Fields: fields}

	case *ast.TypeAlias:
		result.Kind = stmt.TypeAliasDecl{Target: ProjectASTType(def.Target)}

	default:
		// Unknown type definition kind — should not happen after validation.
		result.Kind = stmt.ADTDecl{} // empty ADT as fallback
	}

	return result
}
