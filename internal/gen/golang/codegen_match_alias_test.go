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
