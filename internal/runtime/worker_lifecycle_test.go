package runtime

import (
	"errors"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/sunholo-data/ailang/internal/core"
	"github.com/sunholo-data/ailang/internal/eval"
)

type requestWorkerContext struct {
	closed  *atomic.Int32
	cleanup error
}

func (c *requestWorkerContext) Clone() interface{} {
	return &requestWorkerContext{closed: c.closed, cleanup: c.cleanup}
}
func (c *requestWorkerContext) CloseWorkers() error { c.closed.Add(1); return c.cleanup }
func (c *requestWorkerContext) BindWorkerScope()    {}

func TestEntrypointWorkerLifecycle_ReturnAndError(t *testing.T) {
	for _, fail := range []bool{false, true} {
		t.Run(map[bool]string{false: "return", true: "error"}[fail], func(t *testing.T) {
			var closed atomic.Int32
			ctx := &requestWorkerContext{closed: &closed}
			rt := NewModuleRuntime(t.TempDir())
			rt.GetEvaluator().SetEffContext(ctx)
			var body core.CoreExpr = &core.Lit{Kind: core.IntLit, Value: int64(3)}
			if fail {
				body = &core.Var{Name: "missing_for_worker_test"}
			}
			fn := &eval.FunctionValue{Body: body, Env: eval.NewEnvironment()}
			inst := &ModuleInstance{Path: "worker_test", Exports: map[string]eval.Value{"run": fn}}
			_, err := CallEntrypoint(rt, inst, "run", nil)
			if fail && err == nil {
				t.Fatal("expected primary evaluator error")
			}
			if !fail && err != nil {
				t.Fatal(err)
			}
			if closed.Load() != 1 {
				t.Fatal("request worker owner was not closed")
			}
		})
	}
}
func TestEntrypointWorkerLifecycle_PreparePanicCleans(t *testing.T) {
	var closed atomic.Int32
	rt := NewModuleRuntime(t.TempDir())
	rt.GetEvaluator().SetEffContext(&requestWorkerContext{closed: &closed})
	inst := &ModuleInstance{Path: "worker_test", Exports: map[string]eval.Value{"run": &eval.FunctionValue{Body: &core.Lit{Kind: core.IntLit, Value: int64(3)}, Env: eval.NewEnvironment()}}}
	func() {
		defer func() {
			if recover() != "prepare panic" {
				t.Error("primary panic changed")
			}
		}()
		_, _ = CallEntrypointPrepared(rt, inst, "run", nil, func(interface{}) { panic("prepare panic") })
	}()
	if closed.Load() != 1 {
		t.Fatal("preparation panic leaked request owner")
	}
}
func TestEntrypointWorkerLifecycle_CleanupFailureVisible(t *testing.T) {
	for _, primaryFailure := range []bool{false, true} {
		t.Run(map[bool]string{false: "return", true: "primary_error"}[primaryFailure], func(t *testing.T) {
			var closed atomic.Int32
			rt := NewModuleRuntime(t.TempDir())
			rt.GetEvaluator().SetEffContext(&requestWorkerContext{closed: &closed, cleanup: errors.New("worker fixture cleanup failed")})
			var body core.CoreExpr = &core.Lit{Kind: core.IntLit, Value: int64(3)}
			if primaryFailure {
				body = &core.Var{Name: "missing_primary_fixture"}
			}
			inst := &ModuleInstance{Path: "worker_test", Exports: map[string]eval.Value{"run": &eval.FunctionValue{Body: body, Env: eval.NewEnvironment()}}}
			result, err := CallEntrypoint(rt, inst, "run", nil)
			if result != nil || err == nil || !strings.Contains(err.Error(), "worker fixture cleanup failed") {
				t.Fatalf("cleanup failure lost: result=%v err=%v", result, err)
			}
			if primaryFailure && !strings.Contains(err.Error(), "missing_primary_fixture") {
				t.Fatalf("primary failure lost while reporting cleanup: %v", err)
			}
		})
	}
}

func TestEntrypointWorkerLifecycle_CallPanicAndExitPreserved(t *testing.T) {
	for _, sentinel := range []interface{}{"call fixture panic", &eval.EvalExitCode{Code: 7}} {
		var closed atomic.Int32
		rt := NewModuleRuntime(t.TempDir())
		rt.GetEvaluator().SetEffContext(&requestWorkerContext{closed: &closed})
		env := eval.NewEnvironment()
		env.Set("explode", &eval.BuiltinFunction{Name: "explode", Fn: func([]eval.Value) (eval.Value, error) { panic(sentinel) }})
		fn := &eval.FunctionValue{Body: &core.App{Func: &core.Var{Name: "explode"}}, Env: env}
		inst := &ModuleInstance{Path: "worker_test", Exports: map[string]eval.Value{"run": fn}}
		func() {
			defer func() {
				if recovered := recover(); recovered != sentinel {
					t.Errorf("primary panic changed: %v", recovered)
				}
			}()
			_, _ = CallEntrypoint(rt, inst, "run", nil)
		}()
		if closed.Load() != 1 {
			t.Fatal("call panic leaked worker owner")
		}
	}
}
