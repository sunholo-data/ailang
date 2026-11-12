package golang

import (
	"strings"
	"testing"

	"github.com/sunholo-data/ailang/internal/core"
	"github.com/sunholo-data/ailang/internal/types"
)

// #2244: a match whose scrutinee's type is written as an alias of an ADT
// (`type Hue = Color`) switches on the ADT the elaborator resolved the
// constructors to. The alias declares no Kind constants: HueKindRed does not
// exist, and the emitted Go failed to compile.
func TestGenerateMatch_ScrutineeTypedByADTAlias(t *testing.T) {
	const scrutID, matchID = uint64(300), uint64(301)
	arm := func(name, body string) core.MatchArm {
		return core.MatchArm{
			Pattern: &core.ConstructorPattern{Name: name, TypeName: "Color", TypeModule: "main"},
			Body:    &core.Lit{Kind: core.StringLit, Value: body},
		}
	}
	prog := &core.Program{Decls: []core.CoreExpr{&core.Let{
		Name: "code",
		Value: &core.Lambda{Params: []string{"h"}, Body: &core.Match{
			CoreNode:   core.CoreNode{NodeID: matchID},
			Scrutinee:  &core.Var{CoreNode: core.CoreNode{NodeID: scrutID}, Name: "h"},
			Exhaustive: true,
			Arms:       []core.MatchArm{arm("Red", "r"), arm("Green", "g"), arm("Blue", "b")},
		}},
		Body: &core.Var{Name: "code"},
	}}}
	gen := New("test")
	for _, c := range []string{"Red", "Green", "Blue"} {
		gen.RegisterADTConstructor("Color", c, 0)
	}
	cti := make(types.CoreTypeInfo)
	cti[scrutID] = &types.TCon{Name: "Hue", Module: "main"}
	cti[matchID] = &types.TCon{Name: "string"}
	gen.SetCoreTypeInfo(cti)

	code, err := gen.Generate(prog)
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}
	s := string(code)
	if strings.Contains(s, "HueKind") || !strings.Contains(s, "case ColorKindRed:") || !strings.Contains(s, "_scrutinee.(*Color)") {
		t.Errorf("the switch must use Color's Kind constants and type, not the alias Hue's:\n%s", s)
	}
}

// The if-chain path (a guard forces it) tests each constructor against the
// pattern's own ADT. Two loaded ADTs declare Red; before, the bare-name
// lookup took whichever the map yielded first, and the emitted Go panicked
// with "interface {} is *Color, not *Light" (review probe p41/p43/p44).
// Generated repeatedly: the bare-name lookup iterates a Go map.
func TestGenerateMatch_IfChainUsesPatternADT(t *testing.T) {
	const scrutID, matchID = uint64(400), uint64(401)
	for run := 0; run < 32; run++ {
		pat := func(name string) *core.ConstructorPattern {
			return &core.ConstructorPattern{Name: name, TypeName: "Color", TypeModule: "lib/color"}
		}
		prog := &core.Program{Decls: []core.CoreExpr{&core.Let{
			Name: "code",
			Value: &core.Lambda{Params: []string{"h", "n"}, Body: &core.Match{
				CoreNode:   core.CoreNode{NodeID: matchID},
				Scrutinee:  &core.Var{CoreNode: core.CoreNode{NodeID: scrutID}, Name: "h"},
				Exhaustive: true,
				Arms: []core.MatchArm{
					{Pattern: pat("Red"), Guard: &core.Lit{Kind: core.BoolLit, Value: false}, Body: &core.Lit{Kind: core.StringLit, Value: "x"}},
					{Pattern: pat("Red"), Body: &core.Lit{Kind: core.StringLit, Value: "r"}},
					{Pattern: pat("Green"), Body: &core.Lit{Kind: core.StringLit, Value: "g"}},
				},
			}},
			Body: &core.Var{Name: "code"},
		}}}
		gen := New("test")
		for _, typ := range []string{"Light", "Color"} {
			gen.RegisterADTConstructor(typ, "Red", 0)
			gen.RegisterADTConstructor(typ, "Green", 0)
		}
		cti := make(types.CoreTypeInfo)
		cti[scrutID] = &types.TCon{Name: "Hue", Module: "lib/b"}
		cti[matchID] = &types.TCon{Name: "string"}
		gen.SetCoreTypeInfo(cti)
		code, err := gen.Generate(prog)
		if err != nil {
			t.Fatalf("Generate failed: %v", err)
		}
		s := string(code)
		start := strings.Index(s, "func code_impl(")
		end := strings.Index(s, "func code(")
		if start < 0 || end < start {
			t.Fatalf("run %d: generated match function not found:\n%s", run, s)
		}
		body := s[start:end]
		if strings.Contains(body, "*Light") || strings.Contains(body, "LightKind") || !strings.Contains(body, "ColorKindRed") {
			t.Fatalf("run %d: the if-chain must test Color's constructors:\n%s", run, body)
		}
	}
}
