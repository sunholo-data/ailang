package apiserver

import (
	"fmt"

	"github.com/sunholo-data/ailang/internal/ast"
)

// Declared shapes are checked at route registration. Imported aliases and type
// variables remain runtime-checked; the hosting boundary does not infer types.
func declaredResponseHeadersIssue(fn *ast.FuncDecl, file *ast.File) string {
	aliases := make(map[string]*ast.TypeDecl)
	for _, node := range file.Decls {
		decl, ok := node.(*ast.TypeDecl)
		if !ok {
			continue
		}
		aliases[decl.Name] = decl
	}
	result := resolveHeaderType(fn.ReturnType, aliases, nil, 0)
	if app, ok := result.(*ast.TypeApp); ok && app.Constructor == "Result" && len(app.Args) > 0 {
		result = resolveHeaderType(app.Args[0], aliases, nil, 0)
	}
	rec, ok := result.(*ast.RecordType)
	if !ok {
		return ""
	}
	for _, field := range rec.Fields {
		if field.Name != "_headers" {
			continue
		}
		h := resolveHeaderType(field.Type, aliases, nil, 0)
		if fields, ok := h.(*ast.RecordType); ok {
			for _, f := range fields.Fields {
				typ := resolveHeaderType(f.Type, aliases, nil, 0)
				if simple, ok := typ.(*ast.SimpleType); ok && simple.Name == "string" {
					continue
				}
				// Generic and imported types cannot be resolved from this AST alone.
				if headerTypeUnknown(typ, aliases) {
					continue
				}
				return fmt.Sprintf("%s; field %q is declared %s", responseHeadersShape, f.Name, typ)
			}
			return ""
		}
		if simple, ok := h.(*ast.SimpleType); ok && (simple.Name == "Json" || simple.Name == "std/json.Json") {
			return ""
		}
		if headerTypeUnknown(h, aliases) {
			return ""
		}
		return fmt.Sprintf("%s; _headers is declared %s", responseHeadersShape, h)
	}
	return ""
}

func headerTypeUnknown(t ast.Type, aliases map[string]*ast.TypeDecl) bool {
	if t == nil {
		return true
	}
	if _, ok := t.(*ast.TypeVar); ok {
		return true
	}
	if s, ok := t.(*ast.SimpleType); ok {
		return !ast.IsBuiltinTypeName(s.Name) && aliases[s.Name] == nil
	}
	return false
}

// Resolve local aliases (including generic aliases) without changing AST nodes.
func resolveHeaderType(t ast.Type, aliases map[string]*ast.TypeDecl, bindings map[string]ast.Type, depth int) ast.Type {
	if t == nil || depth > maxZeroDepth {
		return t
	}
	switch v := t.(type) {
	case *ast.LabelledType:
		return resolveHeaderType(v.Base, aliases, bindings, depth+1)
	case *ast.TypeVar:
		if bound := bindings[v.Name]; bound != nil {
			return resolveHeaderType(bound, aliases, nil, depth+1)
		}
	case *ast.SimpleType:
		if decl := aliases[v.Name]; decl != nil {
			if target := headerAliasTarget(decl); target != nil {
				return resolveHeaderType(target, aliases, bindings, depth+1)
			}
		}
	case *ast.TypeApp:
		args := make([]ast.Type, len(v.Args))
		for i, arg := range v.Args {
			args[i] = resolveHeaderType(arg, aliases, bindings, depth+1)
		}
		if decl := aliases[v.Constructor]; decl != nil && len(decl.TypeParams) == len(args) {
			bound := make(map[string]ast.Type, len(args))
			for i, name := range decl.TypeParams {
				bound[name] = args[i]
			}
			if target := headerAliasTarget(decl); target != nil {
				return resolveHeaderType(target, aliases, bound, depth+1)
			}
		}
		return &ast.TypeApp{Constructor: v.Constructor, Args: args, Pos: v.Pos}
	case *ast.RecordType:
		copy := *v
		copy.Fields = make([]*ast.RecordField, len(v.Fields))
		for i, f := range v.Fields {
			field := *f
			field.Type = resolveHeaderType(f.Type, aliases, bindings, depth+1)
			copy.Fields[i] = &field
		}
		return &copy
	}
	return t
}

func headerAliasTarget(decl *ast.TypeDecl) ast.Type {
	switch def := decl.Definition.(type) {
	case *ast.TypeAlias:
		return def.Target
	case *ast.RecordType:
		return def
	}
	return nil
}
