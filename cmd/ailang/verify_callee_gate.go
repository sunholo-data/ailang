package main

import (
	"fmt"
	"strings"

	"github.com/sunholo-data/ailang/internal/ast"
	"github.com/sunholo-data/ailang/internal/core"
	"github.com/sunholo-data/ailang/internal/smt"
	"github.com/sunholo-data/ailang/internal/types"
)

// M-SMT-CALLEE-SORT-GATE: the caller-side encodability gate. Split out of
// verify.go to keep both under the 800-line AI-maintainability CI gate
// (make check-file-sizes) — no logic changes.

// collectMonomorphicTypeNames returns the set of type names (ADTs and record
// aliases) declared WITHOUT type parameters across the given files. These are the
// only user types that can be emitted as concrete SMT sorts. Parametric types
// (e.g. Option[a], Result[e,a]) are excluded: the encoder cannot monomorphize them,
// so a signature that mentions them must be gated rather than encoded.
func collectMonomorphicTypeNames(files []*ast.File) map[string]bool {
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

// firstUnencodableCalleeType walks the cross-function call graph reachable from a
// contracted function's body and returns the first callee whose signature (a
// parameter type or the return type) is not SMT-encodable — e.g. a parametric ADT
// like Option[float]. Returns (calleeName, renderedType), or ("","") if all clean.
//
// This closes a gap in the fragment gate: the smt-side checks only inspect
// $builtin/stdlib call names, never the signature TYPES of a user cross-function
// callee. Without this, such a callee leaks an undeclared sort into the SMT script
// and Z3 hard-errors instead of the verifier skipping with UNENCODABLE_TYPE.
func firstUnencodableCalleeType(
	funcName string,
	body core.CoreExpr,
	prog *core.Program,
	imported map[string]*core.Program,
	calleeASTFuncs map[string]*ast.FuncDecl,
	declarable map[string]bool,
) (string, string) {
	for _, name := range smt.CollectCalleeNames(body, funcName, prog, imported) {
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

// findFunctionBody finds the body expression of a named function in the Core program.
func findFunctionBody(prog *core.Program, funcName string) core.CoreExpr {
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

// unwrapLambdaParams unwraps Lambda nodes from the Core body,
// extracting parameters with types from the Surface AST and returning the inner body.
func unwrapLambdaParams(
	funcName string,
	surfaceFuncs map[string]*ast.FuncDecl,
	body core.CoreExpr,
) ([]smt.FunctionParam, core.CoreExpr) {
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
		params := make([]smt.FunctionParam, 0, len(fd.Params))
		for _, p := range fd.Params {
			// Skip unit params from zero-arg function desugaring (func f() → func f(_: ()))
			if isUnitParam(p) {
				continue
			}
			paramType := convertASTTypeToType(p.Type)
			if paramType != nil {
				params = append(params, smt.FunctionParam{
					Name: p.Name,
					Type: paramType,
				})
			}
		}
		return params, innerBody
	}

	// Fallback: use Core param names with Int default (skip unit/dummy params)
	params := make([]smt.FunctionParam, 0, len(coreParamNames))
	for _, name := range coreParamNames {
		if name == "_" {
			continue
		}
		params = append(params, smt.FunctionParam{
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

// convertASTTypeToType converts an AST type annotation to a types.Type.
func convertASTTypeToType(t ast.Type) types.Type {
	if t == nil {
		return nil
	}
	switch ty := t.(type) {
	case *ast.SimpleType:
		return &types.TCon{Name: ty.Name}
	case *ast.ListType:
		elem := convertASTTypeToType(ty.Element)
		if elem == nil {
			return nil
		}
		return &types.TList{Element: elem}
	case *ast.TypeApp:
		// TypeApp{Constructor: "list", Args: [int]} → TList{Element: int}
		if ty.Constructor == "list" && len(ty.Args) == 1 {
			elem := convertASTTypeToType(ty.Args[0])
			if elem == nil {
				return nil
			}
			return &types.TList{Element: elem}
		}
		// Other TypeApps (e.g., Option[int]) → TApp
		if len(ty.Args) > 0 {
			args := make([]types.Type, len(ty.Args))
			for i, a := range ty.Args {
				at := convertASTTypeToType(a)
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
			ft := convertASTTypeToType(f.Type)
			if ft == nil {
				return nil
			}
			fields[f.Name] = ft
		}
		return &types.TRecord{Fields: fields}
	case *ast.LabelledType:
		// Strip IFC label metadata — labels do not affect type structure.
		return convertASTTypeToType(ty.Base)
	default:
		return nil
	}
}
