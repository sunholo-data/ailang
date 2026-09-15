package smt

// from_ast.go — the surface-AST → SMT translation used by `ailang verify` and
// `ailang ai-check`: type annotations to SMT sorts and types.Type, ADT / record
// declarations to declare-datatype inputs, the callee-sort encodability gate,
// and the Core-body lookups the verification driver needs. Moved here from
// cmd/ailang (M-V1-SIMPLIFY-S2 M3) so an embedded or WASM caller can verify
// without the CLI.

import (
	"fmt"
	"strings"

	"github.com/sunholo-data/ailang/internal/ast"
	"github.com/sunholo-data/ailang/internal/core"
	"github.com/sunholo-data/ailang/internal/types"
)

// ADTExtraction holds ADT types, inline record types from ADT constructor fields,
// and named record type aliases (type X = {fields}) extracted from one surface file.
type ADTExtraction struct {
	ADTTypes      map[string][]ADTVariant
	RecordDecls   []string                  // SMT-LIB declare-datatype for inline record types in ADT fields
	RecordAliases map[string]*types.TRecord // Named record type aliases (e.g., "TableCell" → TRecord)
}

// ExtractADTTypesWithRecords extracts ADT type definitions and record type aliases
// from the Surface AST and converts them to SMT-compatible formats.
// Handles three cases:
//  1. Record type aliases (type TableCell = {text: string, ...}) → RecordAliases
//  2. ADTs (type Block = TextBlock(...) | ...) → ADTTypes
//  3. Inline record types in ADT constructor fields → RecordDecls
func ExtractADTTypesWithRecords(file *ast.File) ADTExtraction {
	result := ADTExtraction{
		ADTTypes:      make(map[string][]ADTVariant),
		RecordAliases: make(map[string]*types.TRecord),
	}
	// Track record types found in ADT fields that need declaration
	recordsSeen := make(map[string]bool)

	for _, decl := range file.Decls {
		typeDecl, ok := decl.(*ast.TypeDecl)
		if !ok {
			continue
		}

		// Case 1: Record type alias (type TableCell = {text: string, colSpan: int, ...})
		if recType, ok := typeDecl.Definition.(*ast.RecordType); ok {
			collectNamedRecordAlias(typeDecl.Name, recType, &result, recordsSeen)
			continue
		}

		// Case 2: ADT (type Block = TextBlock(...) | TableBlock(...) | ...)
		algType, ok := typeDecl.Definition.(*ast.AlgebraicType)
		if !ok {
			continue
		}

		var variants []ADTVariant
		for _, ctor := range algType.Constructors {
			variant := ADTVariant{Name: ctor.Name}
			for _, field := range ctor.Fields {
				sortName := ASTTypeToSMTSort(field.Type)
				fieldName := field.Name
				if fieldName == "" {
					// Prefix with constructor name to ensure uniqueness across
					// all constructors in the datatype (Z3 requirement).
					fieldName = fmt.Sprintf("%s_%d", ctor.Name, len(variant.Fields))
				}
				variant.Fields = append(variant.Fields, ADTField{
					Name: fieldName,
					Sort: sortName,
				})
				// If this field is an inline record type, collect its declaration
				if recType, ok := field.Type.(*ast.RecordType); ok {
					collectRecordDeclFromAST(recType, &result, recordsSeen)
				}
			}
			variants = append(variants, variant)
		}
		result.ADTTypes[typeDecl.Name] = variants
	}

	return result
}

// collectNamedRecordAlias registers a named record type alias for Z3 encoding.
// For example: type TableCell = {text: string, colSpan: int, rowSpan: int, merged: bool}
// becomes: (declare-datatype TableCell ((mk_TableCell (colSpan Int) (merged Bool) (rowSpan Int) (text String))))
func collectNamedRecordAlias(name string, recType *ast.RecordType, result *ADTExtraction, seen map[string]bool) {
	if seen[name] {
		return
	}
	seen[name] = true

	typeFields := make(map[string]types.Type, len(recType.Fields))
	for _, f := range recType.Fields {
		ft := ConvertASTTypeToType(f.Type)
		if ft == nil {
			continue
		}
		typeFields[f.Name] = ft
	}
	if len(typeFields) > 0 {
		result.RecordAliases[name] = &types.TRecord{
			Fields:   typeFields,
			TypeName: name,
		}
	}

	// Also check if any field is itself an inline record type that needs declaration
	for _, f := range recType.Fields {
		if innerRec, ok := f.Type.(*ast.RecordType); ok {
			collectRecordDeclFromAST(innerRec, result, seen)
		}
	}
}

// collectRecordDeclFromAST generates a declare-datatype for a record type
// found in an ADT constructor field, so it's declared before the ADT.
func collectRecordDeclFromAST(recType *ast.RecordType, result *ADTExtraction, seen map[string]bool) {
	rec := ConvertASTTypeToType(recType)
	if rec == nil {
		return
	}
	trec, ok := rec.(*types.TRecord)
	if !ok {
		return
	}
	sortName := MapRecordSortName(trec)
	if seen[sortName] {
		return
	}
	seen[sortName] = true

	// Build field sorts
	fields := make(map[string]string)
	for name, fieldType := range trec.Fields {
		sort, err := MapType(fieldType)
		if err != nil {
			continue // Skip unencodable fields
		}
		fields[name] = sort
	}
	if len(fields) > 0 {
		result.RecordDecls = append(result.RecordDecls, DeclareRecordDatatype(sortName, fields))
	}
}

// ASTTypeToSMTSort converts an AST type annotation to an SMT-LIB sort name.
func ASTTypeToSMTSort(t ast.Type) string {
	switch ty := t.(type) {
	case *ast.SimpleType:
		switch ty.Name {
		case "int":
			return "Int"
		case "float":
			return "Real"
		case "bool":
			return "Bool"
		case "string":
			return "String"
		default:
			return ty.Name // ADT type name
		}
	case *ast.ListType:
		elemSort := ASTTypeToSMTSort(ty.Element)
		return fmt.Sprintf("(Seq %s)", elemSort)
	case *ast.TypeApp:
		// TypeApp{Constructor: "list", Args: [int]} → (Seq Int)
		if ty.Constructor == "list" && len(ty.Args) == 1 {
			elemSort := ASTTypeToSMTSort(ty.Args[0])
			return fmt.Sprintf("(Seq %s)", elemSort)
		}
		return ty.Constructor // ADT type name
	case *ast.RecordType:
		rec := ConvertASTTypeToType(ty)
		if rec == nil {
			return "Int" // Fallback
		}
		trec, ok := rec.(*types.TRecord)
		if !ok {
			return "Int"
		}
		return MapRecordSortName(trec)
	case *ast.LabelledType:
		// Strip IFC label metadata — labels do not affect SMT sorts.
		return ASTTypeToSMTSort(ty.Base)
	default:
		return "Int" // Fallback for complex types
	}
}

// M-SMT-CALLEE-SORT-GATE: the caller-side encodability gate.

// CollectMonomorphicTypeNames returns the set of type names (ADTs and record
// aliases) declared WITHOUT type parameters across the given files. These are the
// only user types that can be emitted as concrete SMT sorts. Parametric types
// (e.g. Option[a], Result[e,a]) are excluded: the encoder cannot monomorphize them,
// so a signature that mentions them must be gated rather than encoded.
func CollectMonomorphicTypeNames(files []*ast.File) map[string]bool {
	names := make(map[string]bool)
	for _, f := range files {
		if f == nil {
			continue
		}
		for _, decl := range f.Decls {
			if td, ok := decl.(*ast.TypeDecl); ok && len(td.TypeParams) == 0 {
				names[td.Name] = true
			}
		}
	}
	return names
}

// astTypeEncodable reports whether an AST type can appear in a callee signature
// that the SMT encoder can handle. Primitives, lists/sequences, records, and
// monomorphic ADTs (declared in `declarable`) are encodable. A parametric ADT
// application (TypeApp with args, e.g. Option[float]) or a bare type variable is
// NOT — those are exactly the shapes that leak an undeclared sort into Z3.
// A nil type (no annotation) is treated as encodable — we only gate on what we can see.
func astTypeEncodable(t ast.Type, declarable map[string]bool) bool {
	switch ty := t.(type) {
	case nil:
		return true
	case *ast.SimpleType:
		switch ty.Name {
		case "int", "float", "bool", "string":
			return true
		default:
			return declarable[ty.Name] // monomorphic user ADT/record
		}
	case *ast.ListType:
		return astTypeEncodable(ty.Element, declarable)
	case *ast.TypeApp:
		// list[T] is the one encodable type application (maps to (Seq T)).
		if ty.Constructor == "list" && len(ty.Args) == 1 {
			return astTypeEncodable(ty.Args[0], declarable)
		}
		return false // parametric ADT application (Option[float], Result[e,a], ...)
	case *ast.RecordType:
		return true // records map to declared record datatypes
	case *ast.LabelledType:
		return astTypeEncodable(ty.Base, declarable)
	case *ast.TypeVar:
		return false // unbound type variable in a signature
	default:
		return false // tuples, function types, and other shapes are not encodable
	}
}

// describeASTType renders an AST type for diagnostic messages.
func describeASTType(t ast.Type) string {
	switch ty := t.(type) {
	case nil:
		return "?"
	case *ast.SimpleType:
		return ty.Name
	case *ast.ListType:
		return "[" + describeASTType(ty.Element) + "]"
	case *ast.TypeApp:
		parts := make([]string, len(ty.Args))
		for i, a := range ty.Args {
			parts[i] = describeASTType(a)
		}
		return ty.Constructor + "[" + strings.Join(parts, ", ") + "]"
	case *ast.RecordType:
		return "{...}"
	case *ast.LabelledType:
		return describeASTType(ty.Base)
	case *ast.TypeVar:
		return ty.Name
	default:
		return fmt.Sprintf("%T", t)
	}
}

// FirstUnencodableCalleeType walks the cross-function call graph reachable from a
// contracted function's body and returns the first callee whose signature (a
// parameter type or the return type) is not SMT-encodable — e.g. a parametric ADT
// like Option[float]. Returns (calleeName, renderedType), or ("","") if all clean.
//
// This closes a gap in the fragment gate: the smt-side checks only inspect
// $builtin/stdlib call names, never the signature TYPES of a user cross-function
// callee. Without this, such a callee leaks an undeclared sort into the SMT script
// and Z3 hard-errors instead of the verifier skipping with UNENCODABLE_TYPE.
func FirstUnencodableCalleeType(
	funcName string,
	body core.CoreExpr,
	prog *core.Program,
	imported map[string]*core.Program,
	calleeASTFuncs map[string]*ast.FuncDecl,
	declarable map[string]bool,
) (string, string) {
	for _, name := range CollectCalleeNames(body, funcName, prog, imported) {
		fd, ok := calleeASTFuncs[name]
		if !ok {
			continue
		}
		if !astTypeEncodable(fd.ReturnType, declarable) {
			return name, describeASTType(fd.ReturnType)
		}
		for _, p := range fd.Params {
			if !astTypeEncodable(p.Type, declarable) {
				return name, describeASTType(p.Type)
			}
		}
	}
	return "", ""
}

// FindFunctionBody finds the body expression of a named function in the Core program.
func FindFunctionBody(prog *core.Program, funcName string) core.CoreExpr {
	for _, decl := range prog.Decls {
		switch d := decl.(type) {
		case *core.LetRec:
			for _, binding := range d.Bindings {
				if binding.Name == funcName {
					return binding.Value
				}
			}
		case *core.Let:
			if d.Name == funcName {
				return d.Value
			}
		}
	}
	return nil
}

// UnwrapLambdaParams unwraps Lambda nodes from the Core body,
// extracting parameters with types from the Surface AST and returning the inner body.
func UnwrapLambdaParams(
	funcName string,
	surfaceFuncs map[string]*ast.FuncDecl,
	body core.CoreExpr,
) ([]FunctionParam, core.CoreExpr) {
	// Unwrap Lambda nodes to get param names and inner body
	var coreParamNames []string
	innerBody := body
	for {
		lam, ok := innerBody.(*core.Lambda)
		if !ok {
			break
		}
		coreParamNames = append(coreParamNames, lam.Params...)
		innerBody = lam.Body
	}

	// Build params using Surface AST types (most reliable)
	if fd, ok := surfaceFuncs[funcName]; ok && len(fd.Params) > 0 {
		params := make([]FunctionParam, 0, len(fd.Params))
		for _, p := range fd.Params {
			// Skip unit params from zero-arg function desugaring (func f() → func f(_: ()))
			if isUnitParam(p) {
				continue
			}
			paramType := ConvertASTTypeToType(p.Type)
			if paramType != nil {
				params = append(params, FunctionParam{
					Name: p.Name,
					Type: paramType,
				})
			}
		}
		return params, innerBody
	}

	// Fallback: use Core param names with Int default (skip unit/dummy params)
	params := make([]FunctionParam, 0, len(coreParamNames))
	for _, name := range coreParamNames {
		if name == "_" {
			continue
		}
		params = append(params, FunctionParam{
			Name: name,
			Type: &types.TCon{Name: "int"},
		})
	}
	return params, innerBody
}

// isUnitParam returns true if the param is a unit parameter from zero-arg desugaring.
// The parser desugars `func f()` to `func f(_: ())`, producing a param named "_" with unit type.
func isUnitParam(p *ast.Param) bool {
	if p.Name != "_" {
		return false
	}
	if st, ok := p.Type.(*ast.SimpleType); ok {
		return st.Name == "()"
	}
	return false
}

// ConvertASTTypeToType converts an AST type annotation to a types.Type.
func ConvertASTTypeToType(t ast.Type) types.Type {
	if t == nil {
		return nil
	}
	switch ty := t.(type) {
	case *ast.SimpleType:
		return &types.TCon{Name: ty.Name}
	case *ast.ListType:
		elem := ConvertASTTypeToType(ty.Element)
		if elem == nil {
			return nil
		}
		return &types.TList{Element: elem}
	case *ast.TypeApp:
		// TypeApp{Constructor: "list", Args: [int]} → TList{Element: int}
		if ty.Constructor == "list" && len(ty.Args) == 1 {
			elem := ConvertASTTypeToType(ty.Args[0])
			if elem == nil {
				return nil
			}
			return &types.TList{Element: elem}
		}
		// Other TypeApps (e.g., Option[int]) → TApp
		if len(ty.Args) > 0 {
			args := make([]types.Type, len(ty.Args))
			for i, a := range ty.Args {
				at := ConvertASTTypeToType(a)
				if at == nil {
					return nil
				}
				args[i] = at
			}
			return &types.TApp{
				Constructor: &types.TCon{Name: ty.Constructor},
				Args:        args,
			}
		}
		return &types.TCon{Name: ty.Constructor}
	case *ast.RecordType:
		fields := make(map[string]types.Type, len(ty.Fields))
		for _, f := range ty.Fields {
			ft := ConvertASTTypeToType(f.Type)
			if ft == nil {
				return nil
			}
			fields[f.Name] = ft
		}
		return &types.TRecord{Fields: fields}
	case *ast.LabelledType:
		// Strip IFC label metadata — labels do not affect type structure.
		return ConvertASTTypeToType(ty.Base)
	default:
		return nil
	}
}
