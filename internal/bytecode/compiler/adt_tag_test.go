package compiler

import (
	"strings"
	"testing"

	"github.com/sunholo-data/ailang/internal/gen/stmt"
)

// adtTagProgram declares `type Opt = Some(int) | None` and an exported
// zero-arg `probe` returning ADTTagEq{<ctor>, typeName, tag}.
func adtTagProgram(value stmt.Expr, typeName, tag string) *stmt.Program {
	return &stmt.Program{
		TypeDecls: []stmt.TypeDecl{{
			Name: "Opt",
			Kind: stmt.ADTDecl{Variants: []stmt.ADTVariant{
				{Tag: "Some", Fields: []stmt.ResolvedType{stmt.PrimitiveType{Kind: stmt.PrimInt}}},
				{Tag: "None"},
			}},
		}},
		FuncDecls: []stmt.FuncDecl{{
			Name:     "probe",
			Return:   stmt.ADTTagEq{Value: value, TypeName: typeName, Tag: tag},
			Exported: true,
		}},
	}
}

func TestCompile_ADTTagEq(t *testing.T) {
	some := stmt.ADTConstructor{TypeName: "Opt", Tag: "Some", Args: []stmt.Expr{stmt.LitInt{Value: 1}}}
	none := stmt.ADTConstructor{TypeName: "Opt", Tag: "None"}
	cases := []struct {
		name     string
		value    stmt.Expr
		typeName string
		tag      string
		want     bool
	}{
		{"named type, match", some, "Opt", "Some", true},
		{"named type, miss", none, "Opt", "Some", false},
		{"inferred type, match", none, "", "None", true},
		{"inferred type, miss", some, "", "None", false},
		// A type name the compiler has no TypeDecl for (e.g. an alias) falls
		// back to tag inference rather than failing.
		{"unknown type name, inferred", some, "Alias", "Some", true},
	}
	for _, c := range cases {
		got := runProgram(t, adtTagProgram(c.value, c.typeName, c.tag), "probe", nil)
		if got.Bool != c.want {
			t.Errorf("%s: got %v, want %v", c.name, got.Bool, c.want)
		}
	}
}

// A tag no ADT declares is a loud compile error (the function goes
// EvalOnly), never a guessed ordinal.
func TestCompile_ADTTagEq_UnknownTagIsLoud(t *testing.T) {
	none := stmt.ADTConstructor{TypeName: "Opt", Tag: "None"}
	for _, typeName := range []string{"Opt", ""} {
		img, err := Compile(adtTagProgram(none, typeName, "Nope"))
		if err != nil {
			t.Fatalf("type %q: per-function failures tag EvalOnly, not fail the image: %v", typeName, err)
		}
		proto := img.Prototypes[0]
		if !proto.EvalOnly || !strings.Contains(proto.EvalReason, "Nope") {
			t.Errorf("type %q: expected EvalOnly naming the unknown tag, got EvalOnly=%v reason=%q", typeName, proto.EvalOnly, proto.EvalReason)
		}
	}
}
