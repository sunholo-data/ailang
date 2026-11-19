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

// Review of #2284 (probes r01–r14, d03, d09c, q01): let-bound field readers
// over records stay polymorphic — also when the field's type is not in the
// reader's own type — and print what dev printed. Record-only rows run on the
// evaluator and the strict VM; rows touching a newtype-record ADT run on the
// evaluator only (the strict VM cannot read a newtype's field, pre-existing).
func TestFieldReaders_OutputEqualsBase(t *testing.T) {
	for _, c := range []struct {
		name, body, want string
		vm               bool
	}{
		{"r01", "let f = \\r. r.a.b;\n  println(\"${show(f({ a: { b: 1 } }))} ${f({ a: { b: \"s\" } })}\")", "1 s\n", true},
		{"r02", "let f = \\r. r.a.b;\n  println(\"${show(f({ a: { b: 1 } }))} ${show(f({ a: { b: 2, z: true } }))}\")", "1 2\n", true},
		{"r03", "let f = \\r. r.a.b;\n  println(\"${show(f({ a: { b: 1, z: 5 } }))} ${show(f({ a: { b: 2 } }))}\")", "1 2\n", true},
		{"r05", "let describe = \\p. show(p.x);\n  println(\"${describe({ x: 1 })} ${describe({ x: \"s\" })}\")", "1 s\n", true},
		{"r06", "let count = \\r. length(r.items);\n  println(show(count({ items: [1, 2] }) + count({ items: [\"a\"] })))", "3\n", true},
		{"r07", "let f = \\r. r.a.x + r.a.y;\n  println(show(f({ a: { x: 1, y: 2 } }) + f({ a: { x: 1, y: 2, z: 3 } })))", "6\n", true},
		{"r08", "let describe = \\p. show(p.x);\n  println(\"${describe({ x: 1 })} ${describe({ x: 2, y: true })}\")", "1 2\n", true},
		{"r09", "let same = \\p. p.a == p.b;\n  println(\"${show(same({ a: 1, b: 1 }))} ${show(same({ a: \"x\", b: \"y\" }))}\")", "true false\n", true},
		{"r14", "letrec describe = \\p. \\n. if n <= 0 then show(p.x) else describe(p)(n - 1);\n  println(\"${describe({ x: 1 })(2)} ${describe({ x: \"s\" })(1)}\")", "1 s\n", true},
		{"d09c", "let same = \\r. r.a == r.b;\n  let lt = \\r. r.a < r.b;\n  println(\"${show(same({ a: 3, b: 3 }))} ${show(same({ a: \"p\", b: \"q\" }))} ${show(lt({ a: 1, b: 2 }))} ${show(lt({ a: \"b\", b: \"a\" }))}\")", "true false true false\n", true},
		{"d03", "let f = \\r. r.a.b.c;\n  let rr = { a: { b: { c: 1 } } };\n  let rw = { a: M({ b: I({ c: 2 }) }) };\n  let mixed = { a: { b: I({ c: 3 }) } };\n  println(\"${show(f(rr))} ${show(f(rw))} ${show(f(mixed))} ${show(rw.a.b.c)}\")", "1 2 3 2\n", false},
		{"q01", "let f = \\t. t.x;\n  let v: V = W({ x: 2 });\n  println(\"${show(getv(W({ x: 1 })))} ${show(f(v))}\")", "1001 2\n", false},
		// Second review of #2284 (n12): settling `s.g` decides that `t` is a W.
		{"n12", "let f = \\s. { let t = s.g; getv(s.g) + t.x };\n  println(show(f({ g: W({ x: 7 }) })))", "1014\n", false},
		{"n12-reordered", "let f = \\s. { let t = s.g; let v = t.x; v + getv(s.g) };\n  println(show(f({ g: W({ x: 7 }) })))", "1014\n", false},
	} {
		t.Run(c.name, func(t *testing.T) {
			root := t.TempDir()
			t.Chdir(root)
			src := "module m/main\nimport std/io (println)\nimport std/list (length)\n" +
				"type I = I({ c: int })\ntype M = M({ b: I })\ntype W = W({ x: int })\ntype V = W\n" +
				"pure func getv(v: V) -> int = v.x + 1000\n" +
				"export func main() -> () ! {IO} {\n  " + c.body + "\n}\n"
			if err := os.MkdirAll("m", 0o755); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(filepath.Join("m", "main.ail"), []byte(src), 0o644); err != nil {
				t.Fatal(err)
			}
			engines := []string{"eval"}
			if c.vm {
				engines = append(engines, "strict-bytecode")
			}
			runMainOutput(t, filepath.Join(root, "m", "main.ail"), c.want, engines)
		})
	}
}

// runMainOutput compiles entry and runs its main on each engine, requiring
// stdout == want on every one.
func runMainOutput(t *testing.T, entry, want string, engines []string) {
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
