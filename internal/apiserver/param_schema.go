package apiserver

import "github.com/sunholo-data/ailang/internal/ast"

// ParamSchema retains the declared structure lost by ParamTypes. Unknown leaves
// deliberately use embed.FromGo so ADTs and polymorphic arguments keep working.
type ParamSchema struct {
	Kind    string
	Fields  []paramField
	Element *ParamSchema
}
type paramField struct {
	Name   string
	Schema ParamSchema
}

func schemaFromAST(t ast.Type, aliases map[string]*ast.RecordType, depth int) ParamSchema {
	if depth > maxZeroDepth {
		return ParamSchema{}
	}
	switch v := t.(type) {
	case *ast.SimpleType:
		switch v.Name {
		case "int", "float", "bool", "string", "unit":
			return ParamSchema{Kind: v.Name}
		}
		if r := aliases[v.Name]; r != nil {
			return schemaFromAST(r, aliases, depth+1)
		}
	case *ast.LabelledType:
		return schemaFromAST(v.Base, aliases, depth+1)
	case *ast.RecordType:
		s := ParamSchema{Kind: "record"}
		for _, f := range v.Fields {
			s.Fields = append(s.Fields, paramField{f.Name, schemaFromAST(f.Type, aliases, depth+1)})
		}
		return s
	case *ast.ListType:
		e := schemaFromAST(v.Element, aliases, depth+1)
		return ParamSchema{Kind: "list", Element: &e}
	case *ast.ArrayType:
		e := schemaFromAST(v.Element, aliases, depth+1)
		return ParamSchema{Kind: "array", Element: &e}
	case *ast.TypeApp:
		if (v.Constructor == "list" || v.Constructor == "Array") && len(v.Args) == 1 {
			e := schemaFromAST(v.Args[0], aliases, depth+1)
			kind := "list"
			if v.Constructor == "Array" {
				kind = "array"
			}
			return ParamSchema{Kind: kind, Element: &e}
		}
	}
	return ParamSchema{}
}

func extractParamSchemas(params []*ast.Param, aliases map[string]*ast.RecordType) []ParamSchema {
	out := make([]ParamSchema, len(params))
	for i, p := range params {
		out[i] = schemaFromAST(p.Type, aliases, 0)
	}
	return out
}
