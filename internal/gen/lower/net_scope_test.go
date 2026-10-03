package lower

import (
	"strings"
	"testing"

	"github.com/sunholo-data/ailang/internal/core"
	"github.com/sunholo-data/ailang/internal/types"
)

// M-NET-SCOPE-PUBLIC design D-C: the bytecode VM has no moded-frame hook, so
// a function declaring ! {Net[scope=public]} is lowered as an EvalOnly stub
// and the call routes to the evaluator, which enforces the scope. Dropping
// the scope silently on the VM path would be a silent fallback.
func netScopeProgram(params map[string]string) (*core.Program, types.CoreTypeInfo) {
	lamID := uint64(100)
	row := &types.Row{Kind: types.EffectRow, Labels: map[string]types.Type{"Net": types.Unit()}}
	if params != nil {
		row.Params = map[string]map[string]string{"Net": params}
	}
	cti := makeCTI(map[uint64]types.Type{
		lamID: &types.TFunc2{Params: []types.Type{types.TString}, Return: types.TString, EffectRow: row},
		101:   types.TString,
	})
	prog := &core.Program{
		Decls: []core.CoreExpr{&core.Let{
			CoreNode: core.CoreNode{NodeID: 1},
			Name:     "fetch",
			Value: &core.Lambda{
				CoreNode: core.CoreNode{NodeID: lamID},
				Params:   []string{"u"},
				Body:     coreVar(101, "u"),
			},
			Body: &core.Lit{CoreNode: core.CoreNode{NodeID: 2}, Kind: core.UnitLit},
		}},
		Meta: map[string]*core.DeclMeta{"fetch": {Name: "fetch", IsExport: true}},
	}
	return prog, cti
}

func TestNetScopePublic_LoweredEvalOnly(t *testing.T) {
	prog, cti := netScopeProgram(map[string]string{"scope": "public"})
	out, err := LowerProgram(prog, cti, nil, "main")
	if err != nil {
		t.Fatal(err)
	}
	if len(out.FuncDecls) != 1 {
		t.Fatalf("want 1 func, got %d", len(out.FuncDecls))
	}
	fd := out.FuncDecls[0]
	if !strings.Contains(fd.LowerError, "Net[scope=public]") {
		t.Fatalf("a Net[scope=public] function must lower EvalOnly, LowerError=%q", fd.LowerError)
	}
	if len(fd.Params) != 1 {
		t.Fatalf("EvalOnly stub must keep its arity, got %d params", len(fd.Params))
	}
}

func TestNetScopeBare_LoweredNormally(t *testing.T) {
	prog, cti := netScopeProgram(nil)
	out, err := LowerProgram(prog, cti, nil, "main")
	if err != nil {
		t.Fatal(err)
	}
	if fd := out.FuncDecls[0]; fd.LowerError != "" {
		t.Fatalf("bare Net must lower normally, got LowerError=%q", fd.LowerError)
	}
}
