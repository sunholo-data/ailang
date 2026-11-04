package apiserver

import (
	"github.com/sunholo-data/ailang/internal/ast"
	"testing"
)

func TestParamSchemaExtraction(t *testing.T) {
	s := schemaFromAST(&ast.ListType{Element: &ast.SimpleType{Name: "int"}}, nil, 0)
	if s.Kind != "list" || s.Element.Kind != "int" {
		t.Fatalf("schema: %+v", s)
	}
}

func TestParamSchemaAliasesAndBounds(t *testing.T) {
	aliases := map[string]*ast.RecordType{"Loop": {Fields: []*ast.RecordField{{Name: "next", Type: &ast.SimpleType{Name: "Loop"}}}}}
	s := schemaFromAST(&ast.SimpleType{Name: "Loop"}, aliases, 0)
	for i := 0; i < maxZeroDepth; i++ {
		if len(s.Fields) == 0 {
			break
		}
		s = s.Fields[0].Schema
	}
	if schemaFromAST(&ast.SimpleType{Name: "Loop"}, aliases, maxZeroDepth+1).Kind != "" {
		t.Fatal("unbounded alias")
	}
	labelled := schemaFromAST(&ast.LabelledType{Base: &ast.SimpleType{Name: "float"}}, nil, 0)
	if labelled.Kind != "float" {
		t.Fatalf("labelled: %v", labelled)
	}
	if schemaFromAST(&ast.SimpleType{Name: "Option"}, nil, 0).Kind != "" {
		t.Fatal("ADT should stay unknown")
	}
}

func TestParamSchemaNestedRecord(t *testing.T) {
	record := &ast.RecordType{Fields: []*ast.RecordField{{Name: "items", Type: &ast.TypeApp{Constructor: "list", Args: []ast.Type{&ast.RecordType{Fields: []*ast.RecordField{{Name: "ratio", Type: &ast.SimpleType{Name: "float"}}}}}}}}}
	aliases := map[string]*ast.RecordType{"Options": record}
	schema := schemaFromAST(&ast.SimpleType{Name: "Options"}, aliases, 0)
	if schema.Kind != "record" || schema.Fields[0].Schema.Element.Fields[0].Schema.Kind != "float" {
		t.Fatalf("nested alias: %+v", schema)
	}
	array := schemaFromAST(&ast.ArrayType{Element: &ast.SimpleType{Name: "int"}}, nil, 0)
	if array.Element.Kind != "int" {
		t.Fatal(array)
	}
}
