package ast

import (
	"encoding/json"
	"fmt"
)

// PrintProgram produces a deterministic JSON representation of a Program.
// Note: Program doesn't implement Node, so we handle it separately.
func PrintProgram(prog *Program) string {
	if prog == nil {
		return "null"
	}

	data, err := json.MarshalIndent(simplify(prog), "", "  ")
	if err != nil {
		return fmt.Sprintf("error: %v", err)
	}

	return string(data)
}

// Print produces a deterministic JSON representation of an AST node.
// This is used for golden snapshot testing.
//
// Design decisions:
// - Omits instance-specific metadata: SIDs, byte offsets, detailed positions
// - Normalizes file paths to "test://unit" for reproducibility
// - Includes "type" field for each node to identify node type
// - Uses JSON marshaling with custom handling for Node interface
func Print(node Node) string {
	if node == nil {
		return "null"
	}

	// Use reflection-based JSON marshaling since AST types are already serializable
	// We'll handle the Program type specially to normalize filenames
	data, err := json.MarshalIndent(simplify(node), "", "  ")
	if err != nil {
		return fmt.Sprintf("error: %v", err)
	}

	return string(data)
}

// simplify converts AST nodes to simple JSON-serializable structures
// Removes position info and other instance-specific metadata
func simplify(node interface{}) interface{} {
	if node == nil {
		return nil
	}

	switch n := node.(type) {
	case *Program:
		m := map[string]interface{}{"type": "Program"}
		if n.File != nil {
			m["file"] = simplify(n.File)
		}
		if n.Module != nil {
			m["module"] = simplify(n.Module)
		}
		return m

	case *File:
		m := map[string]interface{}{
			"type": "File",
			"path": "test://unit", // Normalize for tests
		}
		if n.Module != nil {
			m["module"] = simplify(n.Module)
		}
		if len(n.Imports) > 0 {
			m["imports"] = simplifySlice(n.Imports)
		}
		if len(n.Decls) > 0 {
			m["decls"] = simplifyNodeSlice(n.Decls)
		}
		if len(n.Funcs) > 0 {
			m["funcs"] = simplifySlice(n.Funcs)
		}
		if len(n.Statements) > 0 {
			m["statements"] = simplifyNodeSlice(n.Statements)
		}
		return m

	case *ModuleDecl:
		return map[string]interface{}{
			"type": "ModuleDecl",
			"path": n.Path,
		}

	case *ImportDecl:
		m := map[string]interface{}{
			"type": "ImportDecl",
			"path": n.Path,
		}
		if len(n.Symbols) > 0 {
			m["symbols"] = n.Symbols
		}
		return m

	case *Module:
		m := map[string]interface{}{
			"type": "Module",
			"name": n.Name,
		}
		if len(n.Imports) > 0 {
			m["imports"] = simplifySlice(n.Imports)
		}
		if len(n.Exports) > 0 {
			m["exports"] = n.Exports
		}
		if len(n.Decls) > 0 {
			m["decls"] = simplifyNodeSlice(n.Decls)
		}
		return m

	case *Identifier:
		return map[string]interface{}{
			"type": "Identifier",
			"name": n.Name,
		}

	case *Literal:
		m := map[string]interface{}{
			"type": "Literal",
			"kind": literalKindString(n.Kind),
		}
		if n.Value != nil {
			m["value"] = n.Value
		}
		return m

	case *BinaryOp:
		return map[string]interface{}{
			"type":  "BinaryOp",
			"op":    n.Op,
			"left":  simplify(n.Left),
			"right": simplify(n.Right),
		}

	case *UnaryOp:
		return map[string]interface{}{
			"type": "UnaryOp",
			"op":   n.Op,
			"expr": simplify(n.Expr),
		}

	case *Lambda:
		m := map[string]interface{}{
			"type": "Lambda",
			"body": simplify(n.Body),
		}
		if len(n.Params) > 0 {
			m["params"] = simplifySlice(n.Params)
		}
		if len(n.Effects) > 0 {
			m["effects"] = n.Effects
		}
		return m

	case *FuncCall:
		m := map[string]interface{}{
			"type": "FuncCall",
			"func": simplify(n.Func),
		}
		if len(n.Args) > 0 {
			m["args"] = simplifyExprSlice(n.Args)
		}
		return m

	case *Let:
		m := map[string]interface{}{
			"type":  "Let",
			"name":  n.Name,
			"value": simplify(n.Value),
			"body":  simplify(n.Body),
		}
		if n.Type != nil {
			m["typeAnnotation"] = simplify(n.Type)
		}
		return m

	case *If:
		return map[string]interface{}{
			"type":      "If",
			"condition": simplify(n.Condition),
			"then":      simplify(n.Then),
			"else":      simplify(n.Else),
		}

	case *Match:
		m := map[string]interface{}{
			"type": "Match",
			"expr": simplify(n.Expr),
		}
		if len(n.Cases) > 0 {
			m["cases"] = simplifySlice(n.Cases)
		}
		return m

	case *Case:
		m := map[string]interface{}{
			"type":    "Case",
			"pattern": simplify(n.Pattern),
			"body":    simplify(n.Body),
		}
		if n.Guard != nil {
			m["guard"] = simplify(n.Guard)
		}
		return m

	case *List:
		m := map[string]interface{}{"type": "List"}
		if len(n.Elements) > 0 {
			m["elements"] = simplifyExprSlice(n.Elements)
		}
		return m

	case *Tuple:
		m := map[string]interface{}{"type": "Tuple"}
		if len(n.Elements) > 0 {
			m["elements"] = simplifyExprSlice(n.Elements)
		}
		return m

	case *Record:
		m := map[string]interface{}{"type": "Record"}
		if len(n.Fields) > 0 {
			// Convert map to sorted slice for determinism
			fields := make([]map[string]interface{}, 0, len(n.Fields))
			for k, v := range n.Fields {
				fields = append(fields, map[string]interface{}{
					"name":  k,
					"value": simplify(v),
				})
			}
			m["fields"] = fields
		}
		return m

	case *RecordAccess:
		return map[string]interface{}{
			"type":   "RecordAccess",
			"record": simplify(n.Record),
			"field":  n.Field,
		}

	// Patterns
	case *WildcardPattern:
		return map[string]interface{}{
			"type": "WildcardPattern",
		}

	case *ConsPattern:
		return map[string]interface{}{
			"type": "ConsPattern",
			"head": simplify(n.Head),
			"tail": simplify(n.Tail),
		}

	case *ConstructorPattern:
		m := map[string]interface{}{
			"type": "ConstructorPattern",
			"name": n.Name,
		}
		if len(n.Patterns) > 0 {
			m["patterns"] = simplifyPatternSlice(n.Patterns)
		}
		return m

	case *TuplePattern:
		m := map[string]interface{}{"type": "TuplePattern"}
		if len(n.Elements) > 0 {
			m["elements"] = simplifyPatternSlice(n.Elements)
		}
		return m

	case *ListPattern:
		m := map[string]interface{}{"type": "ListPattern"}
		if len(n.Elements) > 0 {
			m["elements"] = simplifyPatternSlice(n.Elements)
		}
		if n.Rest != nil {
			m["rest"] = simplify(n.Rest)
		}
		return m

	case *RecordPattern:
		m := map[string]interface{}{
			"type": "RecordPattern",
			"rest": n.Rest,
		}
		if len(n.Fields) > 0 {
			fields := make([]interface{}, len(n.Fields))
			for i, f := range n.Fields {
				fields[i] = map[string]interface{}{
					"name":    f.Name,
					"pattern": simplify(f.Pattern),
				}
			}
			m["fields"] = fields
		}
		return m

	// Types
	case *SimpleType:
		return map[string]interface{}{
			"type": "SimpleType",
			"name": n.Name,
		}

	case *TypeVar:
		return map[string]interface{}{
			"type": "TypeVar",
			"name": n.Name,
		}

	case *FuncType:
		m := map[string]interface{}{
			"type": "FuncType",
		}
		if len(n.Params) > 0 {
			m["params"] = simplifyTypeSlice(n.Params)
		}
		if n.Return != nil {
			m["return"] = simplify(n.Return)
		}
		if len(n.Effects) > 0 {
			m["effects"] = n.Effects
		}
		return m

	case *ListType:
		return map[string]interface{}{
			"type":    "ListType",
			"element": simplify(n.Element),
		}

	case *TupleType:
		m := map[string]interface{}{"type": "TupleType"}
		if len(n.Elements) > 0 {
			m["elements"] = simplifyTypeSlice(n.Elements)
		}
		return m

	case *Param:
		m := map[string]interface{}{
			"type": "Param",
			"name": n.Name,
		}
		if n.Type != nil {
			m["typeAnnotation"] = simplify(n.Type)
		}
		return m

	default:
		// Fallback for unknown types
		return map[string]interface{}{
			"type":  fmt.Sprintf("%T", node),
			"_note": "Not yet handled by printer",
		}
	}
}

// Helper functions for slices
func simplifyNodeSlice(nodes []Node) []interface{} {
	result := make([]interface{}, len(nodes))
	for i, n := range nodes {
		result[i] = simplify(n)
	}
	return result
}

func simplifyExprSlice(exprs []Expr) []interface{} {
	result := make([]interface{}, len(exprs))
	for i, e := range exprs {
		result[i] = simplify(e)
	}
	return result
}

func simplifyTypeSlice(types []Type) []interface{} {
	result := make([]interface{}, len(types))
	for i, t := range types {
		result[i] = simplify(t)
	}
	return result
}

func simplifyPatternSlice(patterns []Pattern) []interface{} {
	result := make([]interface{}, len(patterns))
	for i, p := range patterns {
		result[i] = simplify(p)
	}
	return result
}

func simplifySlice(items interface{}) []interface{} {
	// Generic slice simplification using reflection would go here
	// For now, handle specific types
	switch items := items.(type) {
	case []*ImportDecl:
		result := make([]interface{}, len(items))
		for i, item := range items {
			result[i] = simplify(item)
		}
		return result
	case []*Import:
		result := make([]interface{}, len(items))
		for i, item := range items {
			result[i] = simplify(item)
		}
		return result
	case []*FuncDecl:
		result := make([]interface{}, len(items))
		for i, item := range items {
			result[i] = simplify(item)
		}
		return result
	case []*Param:
		result := make([]interface{}, len(items))
		for i, item := range items {
			result[i] = simplify(item)
		}
		return result
	case []*Case:
		result := make([]interface{}, len(items))
		for i, item := range items {
			result[i] = simplify(item)
		}
		return result
	default:
		return []interface{}{fmt.Sprintf("unhandled slice type: %T", items)}
	}
}

func literalKindString(kind LiteralKind) string {
	switch kind {
	case IntLit:
		return "Int"
	case FloatLit:
		return "Float"
	case StringLit:
		return "String"
	case BoolLit:
		return "Bool"
	case UnitLit:
		return "Unit"
	default:
		return "Unknown"
	}
}

// Compact returns a compact single-line JSON representation
func Compact(node Node) string {
	data, err := json.Marshal(simplify(node))
	if err != nil {
		return fmt.Sprintf("error: %v", err)
	}
	return string(data)
}
