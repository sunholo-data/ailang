package smt

import (
	"errors"
	"strings"
	"testing"

	"github.com/sunholo-data/ailang/internal/ast"
	"github.com/sunholo-data/ailang/internal/types"
)

func TestParametricSortArgumentsRetained(t *testing.T) {
	typ := &ast.TypeApp{Constructor: "Result", Args: []ast.Type{&ast.SimpleType{Name: "int"}, &ast.TypeApp{Constructor: "Option", Args: []ast.Type{&ast.SimpleType{Name: "string"}}}}}
	if got := ASTTypeToSMTSort(typ); got != "(Result Int (Option String))" {
		t.Fatalf("arguments erased: %s", got)
	}
	if got, err := MapType(ConvertASTTypeToType(typ)); err != nil || got != "(Result Int (Option String))" {
		t.Fatalf("checked type mapping: %s %v", got, err)
	}
	if _, err := MapType(&types.TVar{Name: "a"}); err == nil {
		t.Fatal("open type accepted")
	}
}
func TestParametricDefinitionRetainsBinderFields(t *testing.T) {
	file := &ast.File{Decls: []ast.Node{&ast.TypeDecl{Name: "Result", TypeParams: []string{"a", "e"}, Definition: &ast.AlgebraicType{Constructors: []*ast.Constructor{{Name: "Ok", Fields: []*ast.ConstructorField{{Type: &ast.TypeVar{Name: "a"}}}}, {Name: "Err", Fields: []*ast.ConstructorField{{Type: &ast.TypeVar{Name: "e"}}}}}}}}}
	// Real source extraction is covered by CLI fixtures; this test requires the
	// additive binder metadata without altering the monomorphic ADT map API.
	got := ExtractADTTypesWithRecords(file)
	if strings.Join(got.TypeParams["Result"], ",") != "a,e" {
		t.Fatalf("missing generic binders: %+v", got)
	}
}

func TestParametricDefinitionsAndSCCDeterministic(t *testing.T) {
	variants := map[string][]ADTVariant{
		"Even": {{Name: "ELeaf", Fields: []ADTField{{Name: "ELeaf_0", Sort: "a"}}}, {Name: "ENode", Fields: []ADTField{{Name: "ENode_0", Sort: "(Odd a)"}}}},
		"Odd":  {{Name: "ONode", Fields: []ADTField{{Name: "ONode_0", Sort: "(Even a)"}}}},
	}
	first := ""
	for i := 0; i < 20; i++ {
		names := []string{"Even", "Odd"}
		decls := make([]string, 0, len(names))
		for _, name := range names {
			decls = append(decls, DeclareParametricDatatype(name, []string{"a"}, variants[name]))
		}
		grouped := DeclareDatatypesMutual(decls)
		if i == 0 {
			first = grouped
		} else if grouped != first {
			t.Fatal("nondeterministic declaration")
		}
		if err := validateDeclarations([]string{grouped, DeclareConst("e", "(Even Int)")}, NewSMTContext()); err != nil {
			t.Fatal(err)
		}
	}
	if !strings.Contains(first, "(Even 1)") || !strings.Contains(first, "(Odd 1)") {
		t.Fatal(first)
	}
}
func TestParametricMalformedSortRejected(t *testing.T) {
	for _, bad := range []string{"", "(Seq )", "(Result Int)", "Result"} {
		defs := map[string][]ADTVariant{"Wrapper": {{Name: "Wrap", Fields: []ADTField{{Name: "Wrap_0", Sort: bad}}}}}
		err := validateParametricDefinitions(defs, map[string][]string{"Wrapper": nil, "Result": {"a", "e"}})
		if !errors.Is(err, ErrUnresolvableTypes) {
			t.Fatalf("malformed %q accepted: %v", bad, err)
		}
	}
}
