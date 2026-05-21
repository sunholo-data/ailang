package eval_harness

// langreg_wire.go wires the eval_harness runner constructors into the langreg
// package, breaking the circular-import constraint.
//
// langreg cannot import eval_harness (cycle), so it holds factory vars that
// eval_harness populates here via init(). After this init runs, langreg's
// Language.NewRunner() delegates to the real PythonRunner/AILANGRunner.

import (
	"context"

	"github.com/sunholo-data/ailang/internal/eval_harness/langreg"
)

func init() {
	langreg.SetPythonRunnerFactory(func(spec interface{}) interface{} {
		if spec == nil {
			return NewPythonRunner()
		}
		if bs, ok := spec.(*BenchmarkSpec); ok {
			return NewPythonRunnerWithSpec(bs)
		}
		return NewPythonRunner()
	})

	langreg.SetAILANGRunnerFactory(func(ctx context.Context, spec interface{}, taskID string) interface{} {
		var bs *BenchmarkSpec
		if spec != nil {
			if s, ok := spec.(*BenchmarkSpec); ok {
				bs = s
			}
		}
		var caps []string
		if bs != nil {
			caps = bs.Caps
		}
		return NewAILANGRunnerWithTask(ctx, "", caps, taskID, bs)
	})

	langreg.SetJSRunnerFactory(func(spec interface{}) interface{} {
		if bs, ok := spec.(*BenchmarkSpec); ok {
			return NewJSRunnerWithSpec(bs)
		}
		return NewJSRunner()
	})

	langreg.SetGoRunnerFactory(func(spec interface{}) interface{} {
		if bs, ok := spec.(*BenchmarkSpec); ok {
			return NewGoRunnerWithSpec(bs)
		}
		return NewGoRunner()
	})

	langreg.SetMoonbitRunnerFactory(func(spec interface{}) interface{} {
		if bs, ok := spec.(*BenchmarkSpec); ok {
			return NewMoonbitRunnerWithSpec(bs)
		}
		return NewMoonbitRunner()
	})
}
