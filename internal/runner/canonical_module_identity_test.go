package runner

import (
	"bytes"
	"path/filepath"
	"strings"
	"testing"

	"github.com/sunholo-data/ailang/internal/effects"
	"github.com/sunholo-data/ailang/internal/pipeline"
	"github.com/sunholo-data/ailang/internal/runtime"
	"github.com/sunholo-data/ailang/internal/testutil/prefixpkg"
)

// M1b (m-v1-nominal-type-identity): a dependency module that the dependency's
// own files import through module_prefix and the consumer imports by its
// published path is ONE module at run time. Before M1b it was two instances,
// and its module-level bindings were evaluated twice (measured: the fixture's
// init log printed twice, on the evaluator and with --bytecode). The bytecode
// arm is not repeated here: the VM cannot yet compile a module-level binding
// that calls Debug.log and falls back to the evaluator, so it would re-run
// this same arm.
func TestCanonicalModuleIdentity_ModuleLevelCodeRunsOnce(t *testing.T) {
	app, _ := prefixpkg.New(t, nil)
	file := filepath.Join(app, "main.ail")
	res, err := pipeline.Run(pipeline.Config{Mode: pipeline.ModeCheck, NoCache: true}, pipeline.Source{Filename: file})
	if err != nil || len(res.Errors) > 0 {
		t.Fatalf("compile: %v %v", err, res.Errors)
	}
	for _, engine := range []string{"eval"} {
		t.Run(engine, func(t *testing.T) {
			var out bytes.Buffer
			rt := runtime.NewModuleRuntime(".")
			if res.DictReg != nil {
				rt.GetEvaluator().SetDictionaryRegistry(res.DictReg)
			}
			ctx := effects.NewEffContext(nil)
			ctx.Grant(effects.Capability{Name: "IO"})
			ctx.Grant(effects.Capability{Name: "Debug"})
			ctx.IOWriter = &out
			ctx.Debug = effects.NewDebugContext()
			rt.GetEvaluator().SetEffContext(ctx)
			ctx.FnCaller = rt.GetEvaluator().CallValue
			ctx.FnCallerN = rt.GetEvaluator().CallValueN

			err := ExecuteModuleEntrypoint(rt, ModuleExecParams{
				Filename: file, Iface: res.Interface, Modules: res.Modules, Entry: "main", ArgsJSON: "null",
				NoPrint: true, Quiet: true, PipelineResult: &res,
			})
			if err != nil {
				t.Fatalf("%s: %v", engine, err)
			}
			if out.String() != prefixpkg.Want {
				t.Fatalf("%s: stdout = %q, want %q", engine, out.String(), prefixpkg.Want)
			}
			inits := 0
			for _, entry := range ctx.Debug.Collect().Logs {
				if entry.Message == prefixpkg.InitLog {
					inits++
				}
			}
			if inits != 1 {
				t.Fatalf("%s: the shape module's module-level code ran %d times, want 1", engine, inits)
			}
			var shapeInstances []string
			for _, id := range rt.ListInstances() {
				if strings.HasSuffix(id, "types/shape") {
					shapeInstances = append(shapeInstances, id)
				}
			}
			if len(shapeInstances) != 1 || shapeInstances[0] != prefixpkg.Canonical {
				t.Fatalf("%s: shape module instances = %v, want exactly [%s]", engine, shapeInstances, prefixpkg.Canonical)
			}
		})
	}
}
