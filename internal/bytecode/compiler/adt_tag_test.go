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
	}
	for _, c := range cases {
		got := runProgram(t, adtTagProgram(c.value, c.typeName, c.tag), "probe", nil)
		if got.Bool != c.want {
			t.Errorf("%s: got %v, want %v", c.name, got.Bool, c.want)
		}
	}
}

// A tag check with no type, or a type the program does not declare, is a
// loud compile error (EvalOnly), never an ordinal inferred from the tag:
// two ADTs may declare the tag at different ordinals (#2303).
func TestCompile_ADTTagEq_NoTypeIsLoud(t *testing.T) {
	some := stmt.ADTConstructor{TypeName: "Opt", Tag: "Some", Args: []stmt.Expr{stmt.LitInt{Value: 1}}}
	for typeName, want := range map[string]string{"": "has no ADT", "Alias": `unknown ADT "Alias"`} {
		img, err := Compile(adtTagProgram(some, typeName, "Some"))
		if err != nil {
			t.Fatalf("type %q: per-function failures tag EvalOnly, not fail the image: %v", typeName, err)
		}
		if proto := img.Prototypes[0]; !proto.EvalOnly || !strings.Contains(proto.EvalReason, want) {
			t.Errorf("type %q: want EvalOnly with %q, got EvalOnly=%v reason=%q", typeName, want, proto.EvalOnly, proto.EvalReason)
		}
	}
}

// The switch path refuses a missing or unknown ADT the same way: two ADTs
// declare Some here, at different ordinals, so a tag-based guess could pick
// either (#2303).
func TestCompile_SwitchNoTypeIsLoud(t *testing.T) {
	for typeName, want := range map[string]string{"": "has no ADT", "Alias": `unknown ADT "Alias"`} {
		prog := adtTagProgram(stmt.LitInt{Value: 0}, "Opt", "Some")
		prog.TypeDecls = append(prog.TypeDecls, stmt.TypeDecl{Name: "Other", Kind: stmt.ADTDecl{Variants: []stmt.ADTVariant{{Tag: "None"}, {Tag: "Some"}}}})
		prog.FuncDecls[0] = stmt.FuncDecl{
			Name:   "probe",
			Params: []stmt.Param{{Name: "o", Type: stmt.NamedType{Name: "Opt"}}},
			Body: []stmt.Stmt{stmt.SwitchStmt{
				Scrutinee: stmt.VarRef{Name: "o"},
				ADTName:   typeName,
				Cases:     []stmt.SwitchCase{{Tag: "Some", Body: []stmt.Stmt{stmt.ReturnStmt{Value: stmt.LitBool{Value: true}}}}},
				Default:   []stmt.Stmt{stmt.ReturnStmt{Value: stmt.LitBool{Value: false}}},
			}},
			Return:   stmt.LitBool{Value: false},
			Exported: true,
		}
		img, err := Compile(prog)
		if err != nil {
			t.Fatalf("type %q: per-function failures tag EvalOnly, not fail the image: %v", typeName, err)
		}
		if proto := img.Prototypes[0]; !proto.EvalOnly || !strings.Contains(proto.EvalReason, want) {
			t.Errorf("type %q: want EvalOnly with %q, got EvalOnly=%v reason=%q", typeName, want, proto.EvalOnly, proto.EvalReason)
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
