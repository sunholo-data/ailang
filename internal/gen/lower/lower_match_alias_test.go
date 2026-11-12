package lower

import (
	"testing"

	"github.com/sunholo-data/ailang/internal/core"
	"github.com/sunholo-data/ailang/internal/gen/stmt"
	"github.com/sunholo-data/ailang/internal/types"
)

// #2244: a constructor test names the pattern's own ADT — never the
// scrutinee's written type, which may be an alias (Hue) that declares no
// tags — at the top of a switch and at any depth of an if-chain. Only a
// pattern carrying no ADT falls back to the scrutinee's type.
func TestLowerMatch_DispatchesOnPatternADT(t *testing.T) {
	hue := &types.TCon{Name: "Hue", Module: "m"}
	color := func(name string, args ...core.CorePattern) *core.ConstructorPattern {
		return &core.ConstructorPattern{Name: name, TypeName: "Color", TypeModule: "m", Args: args}
	}

	sw := LowerMatchStmt(&core.Match{
		CoreNode:  core.CoreNode{NodeID: 1},
		Scrutinee: coreVar(2, "h"),
		Arms: []core.MatchArm{
			{Pattern: color("Red"), Body: litInt(3, 1)},
			{Pattern: color("Green"), Body: litInt(4, 2)},
		},
	}, makeCTI(map[uint64]types.Type{2: hue}))
	if s, ok := sw.(stmt.SwitchStmt); !ok || s.ADTName != "Color" {
		t.Errorf("switch on a Hue: got %#v, want ADTName Color", sw)
	}

	// Some(Red): the outer pattern is Option's, the nested one Color's.
	some := &core.ConstructorPattern{Name: "Some", TypeName: "Option", TypeModule: "std/option", Args: []core.CorePattern{color("Red")}}
	cond := patternCond(stmt.VarRef{Name: "o"}, &types.TApp{Constructor: &types.TCon{Name: "Opt", Module: "m"}, Args: []types.Type{hue}}, some)
	var tags []stmt.ADTTagEq
	var collect func(e stmt.Expr)
	collect = func(e stmt.Expr) {
		switch x := e.(type) {
		case stmt.ADTTagEq:
			tags = append(tags, x)
		case stmt.BinOp:
			collect(x.Left)
			collect(x.Right)
		}
	}
	collect(cond)
	if len(tags) != 2 || tags[0].TypeName != "Option" || tags[1].TypeName != "Color" {
		t.Errorf("nested tag tests = %+v, want Option then Color", tags)
	}

	// No ADT on the pattern: the scrutinee's type is all there is.
	if got := patternADTName(&core.ConstructorPattern{Name: "Red"}, &types.TCon{Name: "Color", Module: "m"}); got != "Color" {
		t.Errorf("unstamped pattern: %q, want the scrutinee's Color", got)
	}
	if got := patternADTName(&core.ConstructorPattern{Name: "Red"}, nil); got != "" {
		t.Errorf("unstamped pattern, no type: %q, want \"\"", got)
	}
}
