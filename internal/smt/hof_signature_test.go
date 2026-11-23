package smt

import (
	"testing"

	"github.com/sunholo-data/ailang/internal/core"
	"github.com/sunholo-data/ailang/internal/types"
)

var intList = &types.TList{Element: &types.TCon{Name: "int"}}

// typeHOFCall gives a hand-built HOF call App and its list argument the node IDs
// and elaborated types the real pipeline would, so InlineHOFCalls can read the
// specialisation's signature from them (it never assumes an element sort).
func typeHOFCall(call *core.App, listType, resultType types.Type) types.CoreTypeInfo {
	ti := types.CoreTypeInfo{}
	call.NodeID = 9001
	ti.Set(call.NodeID, resultType)
	list := call.Args[len(call.Args)-1]
	if v, ok := list.(*core.Var); ok {
		v.NodeID = 9002
		ti.Set(v.NodeID, listType)
	}
	return ti
}

// A literal-lambda HOF call whose types are unknown is refused, never encoded
// at an assumed (Seq Int): that default sent every non-int list to Z3 as a
// sort error (#215).
func TestInlineHOFCalls_RefusesUnknownTypes(t *testing.T) {
	body := &core.App{
		Func: &core.VarGlobal{Ref: core.GlobalRef{Module: "std/list", Name: "map"}},
		Args: []core.CoreExpr{
			&core.Lambda{Params: []string{"x"}, Body: &core.Var{Name: "x"}},
			&core.Var{Name: "xs"},
		},
	}
	restore := activateADTContext(nil, nil)
	defer restore()
	result := InlineHOFCalls("f", body, 3)
	if result == nil || result.Unsupported == "" {
		t.Fatalf("want a refusal for an untyped map call, got %+v", result)
	}
	if len(result.Specializations) != 0 {
		t.Fatalf("refused call must not emit a specialisation")
	}
}
