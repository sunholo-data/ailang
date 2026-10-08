package apiserver

import (
	"github.com/sunholo-data/ailang/internal/ast"
	"github.com/sunholo-data/ailang/internal/eval"
)

// Zero values for omitted record params, built from the DECLARED record type.
//
// ParamTypes collapses every record to the name "record", whose type-name
// zero (zeroValueForType) is the empty record {}. Bound to a param declared
// {download_url: string, file_id: string, …}, that {} crashed the function on
// its first field read ("record has no field: file_id") — on every surface
// that pads a missing arg: REST named/positional/multipart binding and an
// omitted @optional MCP param alike. The zero now carries every declared
// field at its own zero, so the record the function sees matches its type.
//
// A record with a field that has no zero (an ADT, a function, a type var)
// gets no AST zero and keeps the old {} — the same loud field-access failure
// as before, never a half-built record that type-checks by accident.

// maxZeroDepth bounds alias recursion (type A = {b: B}, type B = {a: A}).
const maxZeroDepth = 8

// extractParamZeros records, per param, the zero value its declared record
// type gives (nil for non-record params: the type-name zero serves them).
func extractParamZeros(params []*ast.Param, aliases map[string]*ast.RecordType) []any {
	zeros := make([]any, len(params))
	any_ := false
	for i, p := range params {
		if !isRecordParamType(p.Type, aliases) {
			continue
		}
		if z, ok := zeroFromAST(p.Type, aliases, 0); ok {
			zeros[i] = z
			any_ = true
		}
	}
	if !any_ {
		return nil
	}
	return zeros
}

func isRecordParamType(t ast.Type, aliases map[string]*ast.RecordType) bool {
	switch v := t.(type) {
	case *ast.RecordType:
		return true
	case *ast.SimpleType:
		return aliases[v.Name] != nil
	case *ast.LabelledType:
		return isRecordParamType(v.Base, aliases)
	}
	return false
}

// zeroFromAST is the zero value of declared type t (ok=false when t has none).
// Values are typed for the engine: an int field binds an int and a float field
// a FloatValue, so neither Call nor CallPreserveFloats re-reads 0 as the other.
func zeroFromAST(t ast.Type, aliases map[string]*ast.RecordType, depth int) (any, bool) {
	if depth > maxZeroDepth {
		return nil, false
	}
	switch v := t.(type) {
	case *ast.RecordType:
		rec := make(map[string]any, len(v.Fields))
		for _, f := range v.Fields {
			z, ok := zeroFromAST(f.Type, aliases, depth+1)
			if !ok {
				return nil, false
			}
			rec[f.Name] = z
		}
		return rec, true
	case *ast.SimpleType:
		switch v.Name {
		case "string":
			return "", true
		case "int":
			return int(0), true
		case "float":
			return &eval.FloatValue{Value: 0}, true
		case "bool":
			return false, true
		}
		if rec := aliases[v.Name]; rec != nil {
			return zeroFromAST(rec, aliases, depth+1)
		}
	case *ast.ListType, *ast.ArrayType:
		return []any{}, true
	case *ast.TypeApp: // [T] parses as list[T]
		if v.Constructor == "list" {
			return []any{}, true
		}
	case *ast.LabelledType:
		return zeroFromAST(v.Base, aliases, depth)
	}
	return nil, false
}

// paramZero is the value a missing param i binds: its declared-record zero
// when it has one (a fresh copy — the engine must not share one map across
// calls), else the type-name zero.
func paramZero(types []string, zeros []any, i int) any {
	if i < len(zeros) && zeros[i] != nil {
		return cloneZero(zeros[i])
	}
	if i < len(types) {
		return zeroValueForType(types[i])
	}
	return nil
}

func cloneZero(v any) any {
	switch z := v.(type) {
	case map[string]any:
		out := make(map[string]any, len(z))
		for k, fv := range z {
			out[k] = cloneZero(fv)
		}
		return out
	case []any:
		return []any{}
	}
	return v
}
