package runner

import (
	"bytes"
	"path/filepath"
	"testing"

	"github.com/sunholo-data/ailang/internal/effects"
	"github.com/sunholo-data/ailang/internal/pipeline"
	"github.com/sunholo-data/ailang/internal/runtime"
	"github.com/sunholo-data/ailang/internal/testutil/prefixpkg"
)

// Review fxJ (#2224, round 2): a root package without a module_prefix whose
// main imports util.ail as `util` (a path) and as `acme/app/util` (its bare
// self-import). Before, the file was two modules and its module-level code
// ran twice; it is one module, initialised once.
func TestCanonicalModuleIdentity_RootPathSpellingRunsOnce(t *testing.T) {
	root := prefixpkg.Tree(t, map[string]string{
		"app/ailang.toml": prefixpkg.Manifest("acme/app", "", nil, nil, nil),
		"app/util.ail": "module acme/app/util\n\nimport std/debug (log)\n\nlet initMarker = {\n  log(\"INIT-UTIL\");\n  1\n}\n\n" +
			"export func mk() -> int = initMarker\n\nexport func get(n: int) -> int = n + initMarker\n",
		"app/main.ail": "module acme/app/main\n\nimport std/io (println)\nimport util (mk)\nimport acme/app/util (get)\n\n" +
			"export func main() -> () ! {IO} = println(show(get(mk())))\n",
	})
	app := filepath.Join(root, "app")
	t.Chdir(app)
	file := filepath.Join(app, "main.ail")
	res, err := pipeline.Run(pipeline.Config{Mode: pipeline.ModeCheck, NoCache: true}, pipeline.Source{Filename: file})
	if err != nil || len(res.Errors) > 0 {
		t.Fatalf("compile: %v %v", err, res.Errors)
	}
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
	if err := ExecuteModuleEntrypoint(rt, ModuleExecParams{
		Filename: file, Iface: res.Interface, Modules: res.Modules, Entry: "main", ArgsJSON: "null",
		NoPrint: true, Quiet: true, PipelineResult: &res,
	}); err != nil {
		t.Fatal(err)
	}
	if out.String() != "2\n" {
		t.Fatalf("stdout = %q, want \"2\\n\"", out.String())
	}
	inits := 0
	for _, e := range ctx.Debug.Collect().Logs {
		if e.Message == "INIT-UTIL" {
			inits++
		}
	}
	if inits != 1 {
		t.Fatalf("util's module-level code ran %d times, want 1", inits)
	}
}
