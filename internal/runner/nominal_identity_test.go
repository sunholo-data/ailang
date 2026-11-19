package runner

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/sunholo-data/ailang/internal/effects"
	"github.com/sunholo-data/ailang/internal/pipeline"
	"github.com/sunholo-data/ailang/internal/runtime"
)

// M-V1-NOMINAL-TYPE-IDENTITY M2, run time: programs that use two same-named
// ADTs from different modules correctly must run on the evaluator and on the
// strict bytecode VM. h2 used to fail on the evaluator only ("ambiguous
// constructor T.A found in multiple modules") because both modules' factories
// were named make_T_A; they are now make_T_A@<module>.
func TestNominalIdentity_RunsOnBothEngines(t *testing.T) {
	cases := []struct {
		name  string
		files map[string]string
		entry string
		want  string
		// strict: the strict VM can run it (h7-shaped programs with two
		// same-named ADTs of DIFFERENT constructors are M4: BC001 today).
		strict bool
	}{
		{
			name: "v15 disjoint use", entry: "v15/main.ail", want: "a.B\nb.A hi\n", strict: true,
			files: map[string]string{
				"v15/a.ail": "module v15/a\nexport type T = A(int) | B\nexport pure func mk() -> T = B\n" +
					"export pure func desc(t: T) -> string = match t { A(n) => \"a.A ${show(n)}\", B => \"a.B\" }\n",
				"v15/b.ail": "module v15/b\nexport type T = B | C(string) | A(string)\nexport pure func mk() -> T = A(\"hi\")\n" +
					"export pure func desc(t: T) -> string = match t { B => \"b.B\", C(s) => \"b.C\", A(s) => \"b.A ${s}\" }\n",
				"v15/main.ail": "module v15/main\nimport std/io (println)\nimport v15/a as Ma (mk, desc)\nimport v15/b as Mb (mk, desc)\n" +
					"export func main() -> () ! {IO} { println(Ma.desc(Ma.mk())); println(Mb.desc(Mb.mk())) }\n",
			},
		},
		{
			name: "h2 qualified constructor next to an auto-imported twin", entry: "h2/main.ail", want: "7\n", strict: true,
			files: map[string]string{
				"h2/a.ail":    "module h2/a\nexport type T = A(int)\nexport pure func get(t: T) -> int = match t { A(n) => n }\n",
				"h2/b.ail":    "module h2/b\nexport type T = A(string)\nexport pure func getb(t: T) -> string = match t { A(s) => s }\n",
				"h2/main.ail": "module h2/main\nimport std/io (println)\nimport h2/b (getb)\nimport h2/a (A, get)\nexport func main() -> () ! {IO} = println(show(get(A(7))))\n",
			},
		},
		{
			name: "h7 different constructors, evaluator", entry: "h7/main.ail", want: "a 1\nb x\n",
			files: map[string]string{
				"h7/a.ail":    "module h7/a\nexport type T = A(int)\nexport pure func mk() -> T = A(1)\nexport pure func desc(t: T) -> string = match t { A(n) => \"a ${show(n)}\" }\n",
				"h7/b.ail":    "module h7/b\nexport type T = B(string)\nexport pure func mk() -> T = B(\"x\")\nexport pure func desc(t: T) -> string = match t { B(s) => \"b ${s}\" }\n",
				"h7/main.ail": "module h7/main\nimport std/io (println)\nimport h7/a as Ma (mk, desc)\nimport h7/b as Mb (mk, desc)\nexport func main() -> () ! {IO} { println(Ma.desc(Ma.mk())); println(Mb.desc(Mb.mk())) }\n",
			},
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			root := t.TempDir()
			t.Chdir(root)
			for rel, src := range c.files {
				if err := os.MkdirAll(filepath.Dir(rel), 0o755); err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(rel, []byte(src), 0o644); err != nil {
					t.Fatal(err)
				}
			}
			engines := []string{"eval"}
			if c.strict {
				engines = append(engines, "strict-bytecode")
			}
			runOnEngines(t, filepath.Join(root, c.entry), c.want, engines)
		})
	}
}

// runOnEngines compiles entry and runs its main on each engine ("eval",
// "strict-bytecode"), requiring stdout == want on every one.
func runOnEngines(t *testing.T, entry, want string, engines []string) {
	t.Helper()
	res, err := pipeline.Run(pipeline.Config{Mode: pipeline.ModeCheck, NoCache: true}, pipeline.Source{Filename: entry})
	if err != nil || len(res.Errors) > 0 {
		t.Fatalf("compile: %v %v", err, res.Errors)
	}
	for _, engine := range engines {
		var out bytes.Buffer
		rt := runtime.NewModuleRuntime(".")
		if res.DictReg != nil {
			rt.GetEvaluator().SetDictionaryRegistry(res.DictReg)
		}
		ctx := effects.NewEffContext(nil)
		ctx.Grant(effects.Capability{Name: "IO"})
		ctx.IOWriter = &out
		rt.GetEvaluator().SetEffContext(ctx)
		ctx.FnCaller = rt.GetEvaluator().CallValue
		ctx.FnCallerN = rt.GetEvaluator().CallValueN
		err := ExecuteModuleEntrypoint(rt, ModuleExecParams{
			Filename: entry, Iface: res.Interface, Modules: res.Modules, Entry: "main", ArgsJSON: "null",
			NoPrint: true, Quiet: true, PipelineResult: &res,
			BytecodeMode: engine != "eval", StrictBytecode: engine != "eval",
		})
		if err != nil {
			t.Fatalf("%s: %v", engine, err)
		}
		if out.String() != want {
			t.Fatalf("%s: stdout = %q, want %q", engine, out.String(), want)
		}
	}
}
