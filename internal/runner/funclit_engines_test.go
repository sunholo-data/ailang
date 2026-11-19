package runner

import (
	"os"
	"path/filepath"
	"testing"
)

// #2238 review (probes d11j, d11e, d11k, d11h): an effect-polymorphic function
// literal — a callback parameter annotated `! {e}` — checks wherever it is
// bound, like the same top-level func, and runs identically on the evaluator
// and the strict VM.
func TestFuncLitEffectPolymorphic_RunsOnBothEngines(t *testing.T) {
	for _, c := range []struct{ name, src, want string }{
		{"d11j let-bound, literal declares ! {e}",
			"module m/main\nimport std/io (println)\nexport func main() -> () ! {IO} {\n  let run = func(g: int -> () ! {e}) -> () ! {e} { g(1) };\n  run(\\n. println(show(n)))\n}\n",
			"1\n"},
		{"d11e literal shares the enclosing func's e",
			"module m/main\nimport std/io (println)\nfunc outer(cb: int -> () ! {e}) -> () ! {e} {\n  let run = func(g: int -> () ! {e}, n: int) -> () ! {e} { g(n) };\n  run(cb, 5)\n}\n" +
				"export func main() -> () ! {IO} = outer(\\n. println(show(n)))\n",
			"5\n"},
		{"d11k module-level let",
			"module m/main\nimport std/io (println)\nlet run = func(g: int -> () ! {e}) -> () ! {e} { g(1) }\nexport func main() -> () ! {IO} = run(\\n. println(show(n)))\n",
			"1\n"},
		{"d11h used at IO and at pure",
			"module m/main\nimport std/io (println)\nexport func main() -> () ! {IO} {\n  let run = func(g: int -> () ! {e}, n: int) -> () ! {e} { g(n) };\n  run(\\n. println(show(n)), 7);\n  run(\\n. (), 8)\n}\n",
			"7\n"},
		{"d11b literal with no effect annotation of its own",
			"module m/main\nimport std/io (println)\nexport func main() -> () ! {IO} {\n  let run = func(g: int -> () ! {e}, n: int) -> () { g(n) };\n  run(\\n. println(show(n)), 7)\n}\n",
			"7\n"},
	} {
		t.Run(c.name, func(t *testing.T) {
			root := t.TempDir()
			t.Chdir(root)
			if err := os.MkdirAll("m", 0o755); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(filepath.Join("m", "main.ail"), []byte(c.src), 0o644); err != nil {
				t.Fatal(err)
			}
			runOnEngines(t, filepath.Join(root, "m", "main.ail"), c.want, []string{"eval", "strict-bytecode"})
		})
	}
}
