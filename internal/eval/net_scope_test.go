package eval

import (
	"errors"
	"testing"

	"github.com/sunholo-data/ailang/internal/types"
)

// M-NET-SCOPE-PUBLIC (#1522): the declared Net scope reaches the effect
// context the way the declared Rand mode does.

func netScopeFnType(params map[string]string) types.Type {
	row := &types.Row{Kind: types.EffectRow, Labels: map[string]types.Type{"Net": types.Unit()}}
	if params != nil {
		row.Params = map[string]map[string]string{"Net": params}
	}
	return &types.TFunc2{Params: []types.Type{types.TString}, Return: types.TString, EffectRow: row}
}

func TestNetScope_ExtractedFromType(t *testing.T) {
	if got := extractNetScope(netScopeFnType(map[string]string{"scope": "public"})); got != "public" {
		t.Fatalf("Net[scope=public] extracted as %q", got)
	}
	if got := extractNetScope(netScopeFnType(nil)); got != "" {
		t.Fatalf("bare Net extracted as %q, want \"\"", got)
	}
	if got := extractNetScope(types.TString); got != "" {
		t.Fatalf("non-function extracted as %q", got)
	}
}

// netScopeRecorder is a fake effect context that records scope pushes/pops.
type netScopeRecorder struct {
	fakeEffCtx
	depth, maxDepth int
	pushes          []string
}

func (r *netScopeRecorder) PushNetScope(scope string) {
	r.pushes = append(r.pushes, "+"+scope)
	r.depth++
	if r.depth > r.maxDepth {
		r.maxDepth = r.depth
	}
}

func (r *netScopeRecorder) PopNetScope(scope string) {
	r.pushes = append(r.pushes, "-"+scope)
	r.depth--
}

func TestNetScope_PushedAndPoppedAroundCall(t *testing.T) {
	ctx := &netScopeRecorder{}
	e := newTestEvaluator()
	e.SetEffContext(ctx)
	fn := &FunctionValue{Params: []string{"x"}, Body: v("x"), Env: NewEnvironment(), EffectNetScope: "public"}
	if _, err := e.CallFunction(fn, []Value{&IntValue{Value: 1}}); err != nil {
		t.Fatal(err)
	}
	if ctx.maxDepth != 1 || ctx.depth != 0 {
		t.Fatalf("scope not pushed once and popped: max=%d final=%d events=%v", ctx.maxDepth, ctx.depth, ctx.pushes)
	}
	// Bare Net pushes nothing.
	bare := &netScopeRecorder{}
	e.SetEffContext(bare)
	plain := &FunctionValue{Params: []string{"x"}, Body: v("x"), Env: NewEnvironment()}
	if _, err := e.CallFunction(plain, []Value{&IntValue{Value: 1}}); err != nil {
		t.Fatal(err)
	}
	if len(bare.pushes) != 0 {
		t.Fatalf("bare function pushed a scope: %v", bare.pushes)
	}
}

// A scoped frame has post-body work (the pop), so it must never be replaced
// by a tail call — otherwise the pop would run early or not at all.
func TestNetScope_FrameNotTailReplaced(t *testing.T) {
	e := newTestEvaluator()
	e.SetEffContext(&netScopeRecorder{})
	e.SetMaxRecursionDepth(5)
	fn := newRecFn(NewEnvironment(), "count", []string{"i", "n"}, countBody())
	fn.EffectNetScope = "public"
	_, err := e.CallFunction(fn, []Value{&IntValue{Value: 0}, &IntValue{Value: 10}})
	var rec *RecursionLimitError
	if !errors.As(err, &rec) {
		t.Fatalf("err = %v: a Net-scoped frame was tail-replaced (want nested calls hitting the limit)", err)
	}
}
