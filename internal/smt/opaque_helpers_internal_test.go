package smt

import (
	"errors"
	"strings"
	"testing"

	"github.com/sunholo-data/ailang/internal/ast"
	"github.com/sunholo-data/ailang/internal/core"
)

// A record update whose type is unknown picks the one declared record that has
// every updated field, deterministically; with several it is a skip, not a
// guess (the first in map order built the wrong record: a Z3 ERROR).
func TestLookupRecordForUpdateByFields(t *testing.T) {
	previous := activeRecordTypes
	defer func() { activeRecordTypes = previous }()
	activeRecordTypes = map[string]*RecordTypeInfo{
		"InlineRun":              {SortName: "InlineRun", FieldSorts: map[string]string{"text": "String", "href": "String"}},
		"Record_level_runs_text": {SortName: "Record_level_runs_text", FieldSorts: map[string]string{"text": "String", "level": "Int", "runs": "(Seq InlineRun)"}},
	}
	ru := &core.RecordUpdate{Base: &core.Var{Name: "n"}, Updates: map[string]core.CoreExpr{"href": &core.Lit{Kind: core.StringLit, Value: "x"}}}
	info, err := lookupRecordForUpdate(ru)
	if err != nil || info.SortName != "InlineRun" {
		t.Fatalf("unique candidate: %v %v", info, err)
	}
	ru.Updates = map[string]core.CoreExpr{"text": &core.Lit{Kind: core.StringLit, Value: "x"}}
	for i := 0; i < 20; i++ {
		if _, err := lookupRecordForUpdate(ru); err == nil || !strings.Contains(err.Error(), "InlineRun, Record_level_runs_text") {
			t.Fatalf("ambiguous update: %v", err)
		}
	}
}

// A helper's uninterpreted function whose signature names an undeclared sort
// is a skip (ErrUnresolvableTypes), not a Z3 ERROR.
func TestValidateDeclarationsChecksDeclareFunSignatures(t *testing.T) {
	ctx := NewSMTContext()
	if err := validateDeclarations([]string{`(declare-fun $spec_collect (Int) Record_count_rest)`}, ctx); !errors.Is(err, ErrUnresolvableTypes) ||
		!strings.Contains(err.Error(), `"Record_count_rest" in the signature of $spec_collect`) {
		t.Fatalf("undeclared result sort: %v", err)
	}
	if err := validateDeclarations([]string{`(declare-fun $spec_f ((Seq Missing) Int) Int)`}, ctx); !errors.Is(err, ErrUnresolvableTypes) {
		t.Fatalf("undeclared parameter sort: %v", err)
	}
	ok := []string{`(declare-datatype Box ((mk_Box (v Int))))`, `(declare-fun $spec_g ((Seq Box) String) (Option Int))`, `(declare-datatype Option (par (a) ((Some (Some_0 a)) (None))))`}
	if err := validateDeclarations([]string{ok[2], ok[0], ok[1]}, ctx); err != nil {
		t.Fatalf("declared sorts: %v", err)
	}
}

// An inline record in an ADT payload with a field of no sort (a type
// variable here) is never declared without it: declaring the rest would
// model two payloads differing only there as equal.
func TestInlineRecordWithUnsortedFieldNotDeclared(t *testing.T) {
	rec := &ast.RecordType{Fields: []*ast.RecordField{
		{Name: "tag", Type: &ast.SimpleType{Name: "int"}},
		{Name: "v", Type: &ast.TypeVar{Name: "a"}},
	}}
	file := &ast.File{Decls: []ast.Node{&ast.TypeDecl{
		Name:       "Holder",
		TypeParams: []string{"a"},
		Definition: &ast.AlgebraicType{Constructors: []*ast.Constructor{{Name: "H", Fields: []*ast.ConstructorField{{Type: rec}}}}},
	}}}
	got := ExtractADTTypesWithRecords(file)
	for _, d := range got.RecordDecls {
		if strings.Contains(d, "Record_tag_v") {
			t.Fatalf("declared without its unsorted field: %s", d)
		}
	}
	ok := &ast.RecordType{Fields: []*ast.RecordField{{Name: "tag", Type: &ast.SimpleType{Name: "int"}}}}
	file.Decls = append(file.Decls, &ast.TypeDecl{Name: "Other", Definition: &ast.AlgebraicType{Constructors: []*ast.Constructor{{Name: "O", Fields: []*ast.ConstructorField{{Type: ok}}}}}})
	got = ExtractADTTypesWithRecords(file)
	if len(got.RecordDecls) != 1 || !strings.Contains(got.RecordDecls[0], "Record_tag") {
		t.Fatalf("a record with every field sorted is declared: %v", got.RecordDecls)
	}
}
