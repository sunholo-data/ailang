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
	TypeParams    map[string][]string
	ADTTypes      map[string][]ADTVariant
	RecordDecls   []string                  // SMT-LIB declare-datatype for inline record types in ADT fields
	RecordAliases map[string]*types.TRecord // Named record type aliases (e.g., "TableCell" → TRecord)
	// RecordConflicts are inline record sort names declared twice with
	// different field sorts (the name is built from field names only).
	RecordConflicts []string
}

// ExtractADTTypesWithRecords extracts ADT type definitions and record type aliases
// from the Surface AST and converts them to SMT-compatible formats.
// Handles three cases:
//  1. Record type aliases (type TableCell = {text: string, ...}) → RecordAliases
//  2. ADTs (type Block = TextBlock(...) | ...) → ADTTypes
//  3. Inline record types in ADT constructor fields → RecordDecls
func ExtractADTTypesWithRecords(file *ast.File) ADTExtraction {
	result := ADTExtraction{
		TypeParams:    make(map[string][]string),
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
				// Declare every inline record type in the field, including one
				// nested in a list or type argument (Json's [{key, value}]).
				collectNestedRecordDecls(field.Type, &result, recordsSeen)
			}
			variants = append(variants, variant)
		}
		result.ADTTypes[typeDecl.Name] = variants
		result.TypeParams[typeDecl.Name] = append([]string(nil), typeDecl.TypeParams...)
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

	// A field with no type (a tuple, a function) makes the whole record
	// undeclarable: declaring the rest would model two values differing only
	// in that field as equal (MdParseState's itemIndents: [(int, int)] was
	// dropped, and Z3 rejected every literal that named it).
	typeFields := make(map[string]types.Type, len(recType.Fields))
	complete := true
	for _, f := range recType.Fields {
		ft := ConvertASTTypeToType(f.Type)
		if ft == nil {
			complete = false
			break
		}
		typeFields[f.Name] = ft
	}
	if complete && len(typeFields) > 0 {
		result.RecordAliases[name] = &types.TRecord{
			Fields:   typeFields,
			TypeName: name,
		}
	}

	// Also declare inline record types inside its fields, at any depth.
	for _, f := range recType.Fields {
		collectNestedRecordDecls(f.Type, result, seen)
	}
}

// collectNestedRecordDecls declares every inline record type that occurs in t:
// t itself, a list or array element, a type argument, a tuple element, or a
// field of another inline record. Cycle-safe: AST types are finite trees and
// named types are not expanded.
func collectNestedRecordDecls(t ast.Type, result *ADTExtraction, seen map[string]bool) {
	switch ty := t.(type) {
	case nil, *ast.SimpleType, *ast.TypeVar, *ast.FuncType:
		return
	case *ast.RecordType:
		collectRecordDeclFromAST(ty, result, seen)
		for _, f := range ty.Fields {
			if f != nil {
				collectNestedRecordDecls(f.Type, result, seen)
			}
		}
	case *ast.ListType:
		collectNestedRecordDecls(ty.Element, result, seen)
	case *ast.ArrayType:
		collectNestedRecordDecls(ty.Element, result, seen)
	case *ast.TypeApp:
		for _, a := range ty.Args {
			collectNestedRecordDecls(a, result, seen)
		}
	case *ast.TupleType:
		for _, e := range ty.Elements {
			collectNestedRecordDecls(e, result, seen)
		}
	case *ast.LabelledType:
		collectNestedRecordDecls(ty.Base, result, seen)
	default:
		panic(fmt.Sprintf("collectNestedRecordDecls: unhandled ast.Type %T", t))
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

	// Build field sorts. A field with no sort makes the record undeclarable
	// (see collectNamedRecordAlias): it is never declared with the rest.
	fields := make(map[string]string)
	for name, fieldType := range trec.Fields {
		sort, err := MapType(fieldType)
		if err != nil {
			return
		}
		fields[name] = sort
	}
	if len(fields) == 0 {
		return
	}
	decl := DeclareRecordDatatype(sortName, fields)
	if seen[sortName] {
		// Two records with one name and two bodies: Choice(string, [{key,
		// desc: string}]) and ChoiceR(Json, [{key, desc: Json}]) are both
		// Record_desc_key by field names. Recorded as a conflict, which
		// extractVerifyTypes resolves by naming them by their field sorts
		// too (record_names.go); one declaration winning made a term over
		// the other ill-sorted (a Z3 ERROR).
		for _, d := range result.RecordDecls {
			if ExtractSortNameFromDecl(d) == sortName && d != decl {
				result.RecordConflicts = append(result.RecordConflicts, sortName)
				break
			}
		}
		return
	}
	seen[sortName] = true
	result.RecordDecls = append(result.RecordDecls, decl)
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
		case "()", "unit":
			return unitSortName
		default:
			return ty.Name // ADT type name
		}
	case *ast.ListType:
		elemSort := ASTTypeToSMTSort(ty.Element)
		if elemSort == "" {
			return ""
		}
		return fmt.Sprintf("(Seq %s)", elemSort)
	case *ast.TypeApp:
		// TypeApp{Constructor: "list", Args: [int]} → (Seq Int)
		if ty.Constructor == "list" && len(ty.Args) == 1 {
			elemSort := ASTTypeToSMTSort(ty.Args[0])
			if elemSort == "" {
				return ""
			}
			return fmt.Sprintf("(Seq %s)", elemSort)
		}
		args := make([]string, len(ty.Args))
		for i, arg := range ty.Args {
			args[i] = ASTTypeToSMTSort(arg)
			if args[i] == "" {
				return ""
			}
		}
		if len(args) == 0 {
			return ty.Constructor
		}
		return "(" + ty.Constructor + " " + strings.Join(args, " ") + ")"
	case *ast.RecordType:
		rec := ConvertASTTypeToType(ty)
		if rec == nil {
			return "" // Unsupported, never invent a numeric sort
		}
		trec, ok := rec.(*types.TRecord)
		if !ok {
			return ""
		}
		return MapRecordSortName(trec)
	case *ast.TypeVar:
		return ty.Name
	case *ast.TupleType:
		elems := make([]string, len(ty.Elements))
		for i, el := range ty.Elements {
			if elems[i] = ASTTypeToSMTSort(el); elems[i] == "" {
				return ""
			}
		}
		sort, err := tupleSortOf(elems)
		if err != nil {
			return ""
		}
		return sort
	case *ast.LabelledType:
		// Strip IFC label metadata — labels do not affect SMT sorts.
		return ASTTypeToSMTSort(ty.Base)
	default:
		return "" // Unsupported, never invent a numeric sort for complex types
	}
}

// M-SMT-CALLEE-SORT-GATE: the caller-side encodability gate.

// CollectDeclarableTypes maps every type name (ADT or record alias) declared
// across files to its number of type parameters. A monomorphic type (0) is a
// sort by name; a parametric one is a sort only applied to as many encodable
// arguments. Option and Result are built in.
func CollectDeclarableTypes(files []*ast.File) map[string]int {
	names := map[string]int{"Option": 1, "Result": 2}
	for _, f := range files {
		if f == nil {
			continue
		}
		for _, decl := range f.Decls {
			if td, ok := decl.(*ast.TypeDecl); ok {
				names[td.Name] = len(td.TypeParams)
			}
		}
	}
	return names
}

// astTypeEncodable reports whether an AST type can appear in a callee signature
// that the SMT encoder can handle: primitives, lists, records, declared
// monomorphic ADTs, and declared parametric ADTs applied to encodable
// arguments. A bare type variable, a tuple or a function type is not — those
// are the shapes that would leak an undeclared sort into Z3. A nil type (no
// annotation) is treated as encodable — we only gate on what we can see.
func astTypeEncodable(t ast.Type, declarable map[string]int) bool {
	switch ty := t.(type) {
	case nil:
		return true
	case *ast.SimpleType:
		switch ty.Name {
		case "int", "float", "bool", "string", "()", "unit":
			return true
		default:
			n, ok := declarable[ty.Name]
			return ok && n == 0 // monomorphic user ADT/record
		}
	case *ast.ListType:
		return astTypeEncodable(ty.Element, declarable)
	case *ast.TypeApp:
		// list[T] is the one encodable type application (maps to (Seq T)).
		if ty.Constructor == "list" && len(ty.Args) == 1 {
			return astTypeEncodable(ty.Args[0], declarable)
		}
		// A parametric ADT applied to encodable arguments (Option[int],
		// Result[float, string], a user Box[int]) is a concrete datatype sort:
		// the encoder declares the ADT with Z3 binders and annotates its
		// constructors at the instance (parametric_*.go).
		if n := declarable[ty.Constructor]; n == 0 || n != len(ty.Args) {
			return false
		}
		for _, a := range ty.Args {
			if !astTypeEncodable(a, declarable) {
				return false
			}
		}
		return true
	case *ast.RecordType:
		return true // records map to declared record datatypes
	case *ast.LabelledType:
		return astTypeEncodable(ty.Base, declarable)
	case *ast.TypeVar:
		return false // unbound type variable in a signature
	case *ast.TupleType:
		// A tuple is the datatype TupleN of its components (tuple_sorts.go).
		if len(ty.Elements) < minTupleArity || len(ty.Elements) > maxTupleArity {
			return false
		}
		for _, el := range ty.Elements {
			if !astTypeEncodable(el, declarable) {
				return false
			}
		}
		return true
	default:
		return false // function types and other shapes are not encodable
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
		return t.String() // tuples, function types: AILANG syntax, never a Go type name
	}
}

// FirstUnencodableCalleeType walks the cross-function call graph reachable from a
// contracted function's body and returns the first callee whose signature (a
// parameter type or the return type) is not SMT-encodable — e.g. a type variable
// or a tuple. Returns (calleeName, renderedType), or ("","") if all clean.
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
	declarable map[string]int,
) (string, string) {
	// Only the callees funcName calls itself (in its body, or in a lambda its
	// map/filter/foldl calls are specialised from): a call needs the callee's
	// sorts at the call. A deeper callee with an unencodable signature is
	// never defined (the resolver refuses it, UnencodableSignatures), so the
	// helper calling it is called opaquely; it used to skip the function
	// although its contract never reached that callee (opaque_helpers.go).
	for _, name := range collectDirectCalls(body, funcName, prog, imported) {
		fd, ok := calleeASTFuncs[name]
		if !ok {
			continue
		}
		if bad := unencodableSignatureType(fd, declarable); bad != "" {
			return name, bad
		}
	}
	return "", ""
}

// unencodableSignatureType describes the first type in fd's signature with no
// SMT sort, or "" when every type has one.
func unencodableSignatureType(fd *ast.FuncDecl, declarable map[string]int) string {
	if !astTypeEncodable(fd.ReturnType, declarable) {
		return describeASTType(fd.ReturnType)
	}
	for _, p := range fd.Params {
		// A zero-arg function desugars to `f(_: ())`; the unit param carries
		// no information and the encoder drops it (see UnwrapLambdaParams), so
		// it must not make the callee unencodable. A nullary pure callee is a
		// named constant.
		if isUnitParam(p) {
			continue
		}
		if !astTypeEncodable(p.Type, declarable) {
			return describeASTType(p.Type)
		}
	}
	return ""
}

// SurfaceReturnTypes maps each function with a return annotation to that type.
func SurfaceReturnTypes(funcs map[string]*ast.FuncDecl) map[string]types.Type {
	out := make(map[string]types.Type, len(funcs))
	for name, fd := range funcs {
		if fd.ReturnType == nil {
			continue
		}
		if t := ConvertASTTypeToType(fd.ReturnType); t != nil {
			out[name] = t
		}
	}
	return out
}

// UnencodableSignatures maps each function whose signature has a type with no
// SMT sort to that type; the resolver never defines one (EncodeFunctionOpts).
func UnencodableSignatures(funcs map[string]*ast.FuncDecl, declarable map[string]int) map[string]string {
	out := make(map[string]string)
	for name, fd := range funcs {
		if bad := unencodableSignatureType(fd, declarable); bad != "" {
			out[name] = bad
		}
	}
	return out
}

// firstUnencodableCalleeInFunc applies FirstUnencodableCalleeType to a function's
// body and then to each of its contract predicates.
func firstUnencodableCalleeInFunc(
	funcName string,
	body core.CoreExpr,
	contracts []*core.Contract,
	prog *core.Program,
	imported map[string]*core.Program,
	calleeASTFuncs map[string]*ast.FuncDecl,
	declarable map[string]int,
) (string, string) {
	if callee, bad := FirstUnencodableCalleeType(funcName, body, prog, imported, calleeASTFuncs, declarable); callee != "" {
		return callee, bad
	}
	for _, c := range contracts {
		if c == nil || c.Expr == nil {
			continue
		}
		if callee, bad := FirstUnencodableCalleeType(funcName, c.Expr, prog, imported, calleeASTFuncs, declarable); callee != "" {
			return callee, bad
		}
	}
	return "", ""
}

// firstUnannotatedCalleeParam returns the first callee reachable from body or
// a contract that has a parameter without a type annotation, and that
// parameter (S5). Such a parameter has no sort; encoding the callee would
// leave it an unknown constant in Z3.
func firstUnannotatedCalleeParam(
	funcName string,
	body core.CoreExpr,
	contracts []*core.Contract,
	prog *core.Program,
	imported map[string]*core.Program,
	calleeASTFuncs map[string]*ast.FuncDecl,
) (string, string) {
	roots := []core.CoreExpr{body}
	for _, c := range contracts {
		if c != nil && c.Expr != nil {
			roots = append(roots, c.Expr)
		}
	}
	for _, root := range roots {
		for _, name := range CollectCalleeNames(root, funcName, prog, imported) {
			if param := firstUnannotatedParam(calleeASTFuncs[name]); param != "" {
				return name, param
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
			if paramType == nil && p.Type != nil {
				paramType = &types.TVar{Name: fmt.Sprintf("unsupported parameter type %T", p.Type)}
			}
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

// SurfaceFunctionParams returns the SMT-relevant parameters of a surface function
// declaration, used to build callee define-fun signatures. The unit parameter of a
// zero-arg function (`func f()` → `f(_: ())`) is dropped, so a nullary pure function
// becomes a nullary define-fun — an SMT-LIB constant — and call sites drop the unit
// argument to match (see encodeUserFunctionCall).
func SurfaceFunctionParams(fd *ast.FuncDecl) []FunctionParam {
	var params []FunctionParam
	for _, p := range fd.Params {
		if isUnitParam(p) {
			continue
		}
		if pt := ConvertASTTypeToType(p.Type); pt != nil {
			params = append(params, FunctionParam{Name: p.Name, Type: pt})
		}
	}
	return params
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
	case *ast.TypeVar:
		return &types.TVar{Name: ty.Name}
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
		if ty.Row != nil {
			return &types.TRecordOpen{Row: &types.TVar{Name: ty.Row.Name}}
		}
		fields := make(map[string]types.Type, len(ty.Fields))
		for _, f := range ty.Fields {
			ft := ConvertASTTypeToType(f.Type)
			if ft == nil {
				return nil
			}
			fields[f.Name] = ft
		}
		return &types.TRecord{Fields: fields}
	case *ast.TupleType:
		elems := make([]types.Type, len(ty.Elements))
		for i, el := range ty.Elements {
			if elems[i] = ConvertASTTypeToType(el); elems[i] == nil {
				return nil
			}
		}
		return &types.TTuple{Elements: elems}
	case *ast.LabelledType:
		// Strip IFC label metadata — labels do not affect type structure.
		return ConvertASTTypeToType(ty.Base)
	default:
		return nil
	}
}
