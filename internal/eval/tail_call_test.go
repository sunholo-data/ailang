package eval

import (
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/sunholo-data/ailang/internal/core"
)

// M-EVAL-TAIL-CALLS (#1486): a call in tail position of a function body runs
// in constant evaluator depth. Frames with post-body work (ensures, budget
// frames, rand mode) stay nested, and the trace call sequence is unchanged.

func ilit(n int) core.CoreExpr    { return &core.Lit{Kind: core.IntLit, Value: int64(n)} }
func v(name string) core.CoreExpr { return &core.Var{Name: name} }
func bin(op string, l, r core.CoreExpr) core.CoreExpr {
	return &core.BinOp{Op: op, Left: l, Right: r}
}
func call(fn string, args ...core.CoreExpr) core.CoreExpr {
	return &core.App{Func: v(fn), Args: args}
}

// countBody: if i >= n then i else count(i + 1, n)
func countBody() core.CoreExpr {
	return &core.If{
		Cond: bin(">=", v("i"), v("n")),
		Then: v("i"),
		Else: call("count", bin("+", v("i"), ilit(1)), v("n")),
	}
}

// newRecFn builds a self-referential function bound in env under name.
func newRecFn(env *Environment, name string, params []string, body core.CoreExpr) *FunctionValue {
	fn := &FunctionValue{Params: params, Body: body, Env: env}
	env.Set(name, fn)
	return fn
}

func newTestEvaluator() *CoreEvaluator {
	e := NewCoreEvaluator()
	e.SetExperimentalBinopShim(true)
	return e
}

func intResult(t *testing.T, v Value, err error) int {
	t.Helper()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, isTail := v.(*tailCall); isTail {
		t.Fatal("a *tailCall escaped a public entry point")
	}
	iv, ok := v.(*IntValue)
	if !ok {
		t.Fatalf("want *IntValue, got %T (%v)", v, v)
	}
	return iv.Value
}

const million = 1_000_000

func TestTailCall_SelfLoopViaIf(t *testing.T) {
	e := newTestEvaluator()
	fn := newRecFn(NewEnvironment(), "count", []string{"i", "n"}, countBody())
	got, err := e.CallFunction(fn, []Value{&IntValue{Value: 0}, &IntValue{Value: million}})
	if n := intResult(t, got, err); n != million {
		t.Fatalf("count = %d, want %d", n, million)
	}
}

func TestTailCall_ThroughLet(t *testing.T) {
	// let j = i + 1 in if j > n then i else count(j, n)
	body := &core.Let{Name: "j", Value: bin("+", v("i"), ilit(1)), Body: &core.If{
		Cond: bin(">", v("j"), v("n")), Then: v("i"), Else: call("count", v("j"), v("n")),
	}}
	e := newTestEvaluator()
	fn := newRecFn(NewEnvironment(), "count", []string{"i", "n"}, body)
	got, err := e.CallFunction(fn, []Value{&IntValue{Value: 0}, &IntValue{Value: million}})
	if n := intResult(t, got, err); n != million {
		t.Fatalf("count = %d, want %d", n, million)
	}
}

func TestTailCall_MutualRecursion(t *testing.T) {
	env := NewEnvironment()
	// even(i) = if i == 0 then 1 else odd(i - 1); odd(i) = if i == 0 then 0 else even(i - 1)
	newRecFn(env, "odd", []string{"i"}, &core.If{Cond: bin("==", v("i"), ilit(0)), Then: ilit(0), Else: call("even", bin("-", v("i"), ilit(1)))})
	even := newRecFn(env, "even", []string{"i"}, &core.If{Cond: bin("==", v("i"), ilit(0)), Then: ilit(1), Else: call("odd", bin("-", v("i"), ilit(1)))})
	got, err := newTestEvaluator().CallFunction(even, []Value{&IntValue{Value: million}})
	if n := intResult(t, got, err); n != 1 {
		t.Fatalf("even(%d) = %d, want 1", million, n)
	}
}

// The first frame reached through evalCore (not CallFunction) also trampolines.
func TestTailCall_FromLetRecBody(t *testing.T) {
	letrec := &core.LetRec{
		Bindings: []core.RecBinding{{Name: "count", Value: &core.Lambda{Params: []string{"i", "n"}, Body: countBody()}}},
		Body:     call("count", ilit(0), ilit(million)),
	}
	got, err := newTestEvaluator().evalCore(letrec)
	if n := intResult(t, got, err); n != million {
		t.Fatalf("count = %d, want %d", n, million)
	}
}

// D5: a tail call does not deepen recursionDepth, so a tiny limit still runs a
// long loop; non-tail recursion is still bounded.
func TestTailCall_DepthLimitCountsPendingFramesOnly(t *testing.T) {
	e := newTestEvaluator()
	e.SetMaxRecursionDepth(5)
	fn := newRecFn(NewEnvironment(), "count", []string{"i", "n"}, countBody())
	got, err := e.CallFunction(fn, []Value{&IntValue{Value: 0}, &IntValue{Value: 1000}})
	if n := intResult(t, got, err); n != 1000 {
		t.Fatalf("count = %d", n)
	}

	// sum(i) = if i == 0 then 0 else 1 + sum(i - 1)  — not a tail call
	sum := newRecFn(NewEnvironment(), "sum", []string{"i"}, &core.If{
		Cond: bin("==", v("i"), ilit(0)), Then: ilit(0), Else: bin("+", ilit(1), call("sum", bin("-", v("i"), ilit(1)))),
	})
	_, err = e.CallFunction(sum, []Value{&IntValue{Value: 100}})
	var rec *RecursionLimitError
	if !errors.As(err, &rec) {
		t.Fatalf("non-tail recursion past the limit: err = %v, want RecursionLimitError", err)
	}
}

// D4: frames with post-body work are never replaced — observed through the
// depth limit, which a nested chain of 10 exceeds at max depth 5.
func TestTailCall_DisqualifiedFramesStayNested(t *testing.T) {
	cases := map[string]func(fn *FunctionValue){
		"ensures (checking on)": func(fn *FunctionValue) {
			fn.Postconditions = []*ContractSpec{{Kind: "ensures", Expr: &core.Lit{Kind: core.BoolLit, Value: true}}}
		},
		"@limit budget": func(fn *FunctionValue) { fn.EffectBudgets = map[string]int{"IO": 100} },
		"@min budget":   func(fn *FunctionValue) { fn.EffectMinBudgets = map[string]int{"IO": 0} },
		"rand mode":     func(fn *FunctionValue) { fn.EffectRandMode = "seeded" },
	}
	for name, mark := range cases {
		t.Run(name, func(t *testing.T) {
			e := newTestEvaluator()
			e.SetEffContext(&fakeEffCtx{contracts: true})
			e.SetMaxRecursionDepth(5)
			fn := newRecFn(NewEnvironment(), "count", []string{"i", "n"}, countBody())
			mark(fn)
			_, err := e.CallFunction(fn, []Value{&IntValue{Value: 0}, &IntValue{Value: 10}})
			var rec *RecursionLimitError
			if !errors.As(err, &rec) {
				t.Fatalf("err = %v: a %s frame was replaced (want nested calls hitting the limit)", err, name)
			}
		})
	}
}

// ensures with contract checking OFF does no post-body work, so the frame is
// replaceable.
func TestTailCall_EnsuresWithCheckingOffIsReplaceable(t *testing.T) {
	e := newTestEvaluator()
	e.SetEffContext(&fakeEffCtx{contracts: false})
	e.SetMaxRecursionDepth(5)
	fn := newRecFn(NewEnvironment(), "count", []string{"i", "n"}, countBody())
	fn.Postconditions = []*ContractSpec{{Kind: "ensures", Expr: &core.Lit{Kind: core.BoolLit, Value: true}}}
	got, err := e.CallFunction(fn, []Value{&IntValue{Value: 0}, &IntValue{Value: 100}})
	if n := intResult(t, got, err); n != 100 {
		t.Fatalf("count = %d", n)
	}
}

// --- trace replay (D6) -------------------------------------------------------

type fakeEffCtx struct {
	contracts bool
	events    []string
}

func (f *fakeEffCtx) HasTraceCollector() bool    { return true }
func (f *fakeEffCtx) RecordsFunctionCalls() bool { return true }
func (f *fakeEffCtx) RenderTraceValue(v Value) string {
	if v == nil {
		return "<nil>"
	}
	return v.String()
}
func (f *fakeEffCtx) RecordFunctionEnter(name string, args []string) {
	f.events = append(f.events, fmt.Sprintf("E %s(%s)", name, strings.Join(args, ",")))
}
func (f *fakeEffCtx) RecordFunctionExit(name string, result string) {
	f.events = append(f.events, fmt.Sprintf("X %s=%s", name, result))
}
func (f *fakeEffCtx) CheckRequires(cond bool, msg, loc string) error {
	if !cond {
		return fmt.Errorf("requires failed: %s", msg)
	}
	return nil
}
func (f *fakeEffCtx) CheckEnsures(cond bool, msg, loc string) error {
	if !cond {
		return fmt.Errorf("ensures failed: %s", msg)
	}
	return nil
}
func (f *fakeEffCtx) IsContractCheckingEnabled() bool { return f.contracts }

func eventsString(f *fakeEffCtx) string { return strings.Join(f.events, " | ") }

// The sequence nested evaluation produces for count(0,2) entered via evalCore.
func TestTailCall_TraceSequenceMatchesNested(t *testing.T) {
	ctx := &fakeEffCtx{}
	e := newTestEvaluator()
	e.SetEffContext(ctx)
	letrec := &core.LetRec{
		Bindings: []core.RecBinding{{Name: "count", Value: &core.Lambda{Params: []string{"i", "n"}, Body: countBody()}}},
		Body:     call("count", ilit(0), ilit(2)),
	}
	got, err := e.evalCore(letrec)
	intResult(t, got, err)
	want := "E count(0,2) | E count(1,2) | E count(2,2) | X count=2 | X count=2 | X count=2"
	if s := eventsString(ctx); s != want {
		t.Fatalf("trace\n got: %s\nwant: %s", s, want)
	}
}

// An untraced CallFunction entry that tail-calls a helper: the helper frames are
// traced exactly as when reached through evalCoreApp today (quorum round 2).
func TestTailCall_CallFunctionEntryTracesTailCalledHelpers(t *testing.T) {
	ctx := &fakeEffCtx{}
	e := newTestEvaluator()
	e.SetEffContext(ctx)
	env := NewEnvironment()
	newRecFn(env, "count", []string{"i", "n"}, countBody())
	main := &FunctionValue{Params: []string{"n"}, Body: call("count", ilit(0), v("n")), Env: env}
	got, err := e.CallFunction(main, []Value{&IntValue{Value: 1}})
	intResult(t, got, err)
	want := "E count(0,1) | E count(1,1) | X count=1 | X count=1"
	if s := eventsString(ctx); s != want {
		t.Fatalf("trace\n got: %s\nwant: %s", s, want)
	}
}

// requires fails at iteration k=2: that frame emits no exit, the frames below
// it do (nested: E0 E1 E2 X1 X0).
func TestTailCall_RequiresFailureTraceReplay(t *testing.T) {
	ctx := &fakeEffCtx{contracts: true}
	e := newTestEvaluator()
	e.SetEffContext(ctx)
	// step(i) requires i < 2 = if i >= 5 then i else step(i + 1)
	body := &core.If{Cond: bin(">=", v("i"), ilit(5)), Then: v("i"), Else: call("step", bin("+", v("i"), ilit(1)))}
	// Build the function directly so the precondition can be attached.
	env := NewEnvironment()
	fn := newRecFn(env, "step", []string{"i"}, body)
	fn.Preconditions = []*ContractSpec{{Kind: "requires", Expr: bin("<", v("i"), ilit(2)), Message: "i < 2"}}
	entry := &FunctionValue{Params: []string{"_"}, Body: call("step", ilit(0)), Env: env}
	_, err := e.CallFunction(entry, []Value{&UnitValue{}})
	if err == nil || !strings.Contains(err.Error(), "requires failed") {
		t.Fatalf("err = %v, want requires failure", err)
	}
	want := "E step(0) | E step(1) | E step(2) | X step=<nil> | X step=<nil>"
	if s := eventsString(ctx); s != want {
		t.Fatalf("trace\n got: %s\nwant: %s", s, want)
	}
}

// A body error at the innermost frame: every frame emits its exit (nil result).
func TestTailCall_BodyErrorTraceReplay(t *testing.T) {
	ctx := &fakeEffCtx{}
	e := newTestEvaluator()
	e.SetEffContext(ctx)
	env := NewEnvironment()
	env.Set("boom", &BuiltinFunction{Name: "boom", Fn: func([]Value) (Value, error) { return nil, errors.New("boom") }})
	// walk(i) = if i >= 2 then boom(i) else walk(i + 1)
	body := &core.If{Cond: bin(">=", v("i"), ilit(2)), Then: call("boom", v("i")), Else: call("walk", bin("+", v("i"), ilit(1)))}
	newRecFn(env, "walk", []string{"i"}, body)
	entry := &FunctionValue{Params: []string{"_"}, Body: call("walk", ilit(0)), Env: env}
	_, err := e.CallFunction(entry, []Value{&UnitValue{}})
	if err == nil || !strings.Contains(err.Error(), "boom") {
		t.Fatalf("err = %v, want boom", err)
	}
	want := "E walk(0) | E walk(1) | E walk(2) | X walk=<nil> | X walk=<nil> | X walk=<nil>"
	if s := eventsString(ctx); s != want {
		t.Fatalf("trace\n got: %s\nwant: %s", s, want)
	}
}

// Env and resolver are restored for the caller after a tail chain, on success
// and on error.
func TestTailCall_RestoresCallerEnv(t *testing.T) {
	e := newTestEvaluator()
	before := e.env
	fn := newRecFn(NewEnvironment(), "count", []string{"i", "n"}, countBody())
	if _, err := e.CallFunction(fn, []Value{&IntValue{Value: 0}, &IntValue{Value: 50}}); err != nil {
		t.Fatal(err)
	}
	if e.env != before {
		t.Fatal("caller env not restored after a tail chain")
	}
	env := NewEnvironment()
	env.Set("boom", &BuiltinFunction{Name: "boom", Fn: func([]Value) (Value, error) { return nil, errors.New("boom") }})
	walk := newRecFn(env, "walk", []string{"i"}, &core.If{Cond: bin(">=", v("i"), ilit(3)), Then: call("boom", v("i")), Else: call("walk", bin("+", v("i"), ilit(1)))})
	_, _ = e.CallFunction(walk, []Value{&IntValue{Value: 0}})
	if e.env != before {
		t.Fatal("caller env not restored after a failing tail chain")
	}
}
