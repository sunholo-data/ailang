package runner

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/sunholo-data/ailang/internal/pipeline"
)

// writeTree writes files (relative path -> source) under root.
func writeTree(t *testing.T, root string, files map[string]string) {
	t.Helper()
	for rel, src := range files {
		p := filepath.Join(root, rel)
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(src), 0o644); err != nil {
			t.Fatal(err)
		}
	}
}

const reexportColor = "module lib/color\n" +
	"export type Color = Red | Green | Blue(int)\n" +
	"export pure func mk() -> Color = Blue(7)\n" +
	"export pure func name(c: Color) -> string = match c { Red => \"red\", Green => \"green\", Blue(n) => \"blue ${show(n)}\" }\n"

// TST-PKGPAR-06
// A constructor imported through a facade (`export import`) is the origin's
// constructor: values built on either side match the other's patterns, on the
// evaluator and the strict VM (M-PKG-PARALLEL-AGENTS C1). Before C1 this was
// IMP010 ("symbol 'Red' not exported by 'lib/facade'").
func TestReexport_IdentityOnBothEngines(t *testing.T) {
	cases := []struct {
		name  string
		files map[string]string
		want  string
	}{
		{
			name: "constructors and type through a facade",
			want: "red\nblue 7\n1\n",
			files: map[string]string{
				"lib/facade.ail": "module lib/facade\nexport import lib/color (Color, Red, Blue, mk, name)\n",
				"main.ail": "module main\nimport std/io (println)\nimport lib/facade (Color, Red, Blue, mk, name)\n" +
					"pure func code(c: Color) -> int = match c { Red => 1, Blue(_) => 2, _ => 3 }\n" +
					"export func main() -> () ! {IO} { println(name(Red)); println(name(mk())); println(show(code(Red))) }\n",
			},
		},
		{
			name: "a value from the origin matched by the facade's constructor",
			want: "blue\n",
			files: map[string]string{
				"lib/facade.ail": "module lib/facade\nexport import lib/color (Color, Blue)\n",
				"main.ail": "module main\nimport std/io (println)\nimport lib/color (mk)\nimport lib/facade (Color, Blue)\n" +
					"pure func isBlue(c: Color) -> bool = match c { Blue(_) => true, _ => false }\n" +
					"export func main() -> () ! {IO} = println(if isBlue(mk()) then \"blue\" else \"other\")\n",
			},
		},
		{
			name: "aliases on both sides, and the facade's own export in the same import",
			want: "red\nfacade-own\n",
			files: map[string]string{
				"lib/facade.ail": "module lib/facade\nexport import lib/color (Red as Crimson, name)\n" +
					"export pure func own() -> string = \"facade-own\"\n",
				"main.ail": "module main\nimport std/io (println)\nimport lib/facade (Crimson as R, name as describe, own)\n" +
					"export func main() -> () ! {IO} { println(describe(R)); println(own()) }\n",
			},
		},
		{
			// What a module alias supports: functions and types. (Qualified
			// constructors, `C.Red`, are IMP010 through any module alias today,
			// facade or not.) The match names the origin's constructors.
			name: "module alias access through a facade",
			want: "blue 7\n2\n",
			files: map[string]string{
				"lib/facade.ail": "module lib/facade\nexport import lib/color (Color, Red, Blue, mk, name)\n",
				"main.ail": "module main\nimport std/io (println)\nimport lib/facade as F\n" +
					"pure func code(c: F.Color) -> int = match c { Red => 1, Blue(_) => 2, _ => 3 }\n" +
					"export func main() -> () ! {IO} { println(F.name(F.mk())); println(show(code(F.mk()))) }\n",
			},
		},
		{
			name: "a chain of two facades",
			want: "green\n",
			files: map[string]string{
				"lib/inner.ail": "module lib/inner\nexport import lib/color (Green, name)\n",
				"lib/outer.ail": "module lib/outer\nexport import lib/inner (Green, name)\n",
				"main.ail":      "module main\nimport std/io (println)\nimport lib/outer (Green, name)\nexport func main() -> () ! {IO} = println(name(Green))\n",
			},
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			root := t.TempDir()
			t.Chdir(root)
			c.files["lib/color.ail"] = reexportColor
			writeTree(t, root, c.files)
			runOnEngines(t, filepath.Join(root, "main.ail"), c.want, []string{"eval", "strict-bytecode"})
		})
	}
}

// compileErr compiles entry under root and returns the joined compile error
// text ("" when it compiles).
func compileErr(t *testing.T, root, entry string, noCache bool) string {
	t.Helper()
	res, err := pipeline.Run(pipeline.Config{Mode: pipeline.ModeCheck, NoCache: noCache}, pipeline.Source{Filename: filepath.Join(root, entry)})
	var parts []string
	if err != nil {
		parts = append(parts, err.Error())
	}
	for _, e := range res.Errors {
		parts = append(parts, fmt.Sprint(e))
	}
	return strings.Join(parts, "\n")
}

// TST-PKGPAR-09
// A re-export chain that returns to itself is MOD016, naming the cycle.
// Re-exporting a name the source does not export is the existing IMP010,
// naming the source module.
func TestReexport_CycleAndMissingName(t *testing.T) {
	t.Run("cycle", func(t *testing.T) {
		root := t.TempDir()
		t.Chdir(root)
		writeTree(t, root, map[string]string{
			"lib/a.ail": "module lib/a\nexport import lib/b (X)\n",
			"lib/b.ail": "module lib/b\nexport import lib/a (X)\n",
			"main.ail":  "module main\nimport lib/a (X)\nexport func main() -> int = 1\n",
		})
		got := compileErr(t, root, "main.ail", true)
		// The cycle is reported from whichever of its modules is visited first.
		if !strings.Contains(got, "MOD016") || !(strings.Contains(got, "lib/a.X -> lib/b.X -> lib/a.X") || strings.Contains(got, "lib/b.X -> lib/a.X -> lib/b.X")) {
			t.Fatalf("want MOD016 naming the cycle, got:\n%s", got)
		}
	})
	t.Run("missing name", func(t *testing.T) {
		root := t.TempDir()
		t.Chdir(root)
		writeTree(t, root, map[string]string{
			"lib/color.ail":  reexportColor,
			"lib/facade.ail": "module lib/facade\nexport import lib/color (Purple)\n",
			"main.ail":       "module main\nimport lib/facade (Purple)\nexport func main() -> int = 1\n",
		})
		got := compileErr(t, root, "main.ail", true)
		if !strings.Contains(got, "IMP010") || !strings.Contains(got, "'Purple' not exported by 'lib/color'") {
			t.Fatalf("want IMP010 naming lib/color, got:\n%s", got)
		}
	})
}

// The importer keeps the facade in its cache closure: a warm, cached
// dependent must recompile and fail once the facade stops re-exporting a name
// it uses, never be served its stale success.
func TestReexport_DroppedReexportMissesCache(t *testing.T) {
	t.Setenv("AILANG_CACHE_DIR", t.TempDir())
	root := t.TempDir()
	t.Chdir(root)
	writeTree(t, root, map[string]string{
		"lib/color.ail":  reexportColor,
		"lib/facade.ail": "module lib/facade\nexport import lib/color (Color, Red, name)\n",
		"main.ail":       "module main\nimport lib/facade (Red, name)\nexport func main() -> string = name(Red)\n",
	})
	for i := 0; i < 2; i++ { // cold, then warm
		if got := compileErr(t, root, "main.ail", false); got != "" {
			t.Fatalf("run %d: %s", i, got)
		}
	}
	writeTree(t, root, map[string]string{"lib/facade.ail": "module lib/facade\nexport import lib/color (Color, name)\n"})
	if got := compileErr(t, root, "main.ail", false); !strings.Contains(got, "Red") {
		t.Fatalf("a dropped re-export must fail the cached dependent, got %q", got)
	}
}

// A name imported through a re-export line marked @deprecated("path") warns
// TC_DEPRECATED_001 at the name in the import list, for a function, a type and
// a constructor alike, with the fix naming the new path. A re-export line
// without @deprecated warns nothing.
func TestReexport_MovedWarning(t *testing.T) {
	warnings := func(facadeLine string) []string {
		root := t.TempDir()
		t.Chdir(root)
		writeTree(t, root, map[string]string{
			"lib/color.ail":  reexportColor,
			"lib/facade.ail": "module lib/facade\n" + facadeLine + "\n",
			"main.ail":       "module main\nimport lib/facade (Color, Red, name)\nexport func main() -> string = name(Red)\n",
		})
		res, err := pipeline.Run(pipeline.Config{Mode: pipeline.ModeCheck, NoCache: true}, pipeline.Source{Filename: filepath.Join(root, "main.ail")})
		if err != nil || len(res.Errors) > 0 {
			t.Fatalf("compile: %v %v", err, res.Errors)
		}
		var out []string
		type coded interface {
			Code() string
			Position() (int, int)
		}
		for _, w := range res.Warnings {
			if c, ok := w.(coded); ok && c.Code() == pipeline.CodeDeprecated {
				line, col := c.Position()
				out = append(out, fmt.Sprintf("%d:%d %s", line, col, w.String()))
			}
		}
		return out
	}
	got := warnings(`export import lib/color (Color, Red, name) @deprecated("pkg/demo/core/color")`)
	if len(got) != 3 {
		t.Fatalf("want 3 warnings (type, constructor, function), got %d:\n%s", len(got), strings.Join(got, "\n"))
	}
	for i, want := range []string{"2:20", "2:27", "2:32"} {
		if !strings.HasPrefix(got[i], want+" ") {
			t.Errorf("warning %d at %q, want %s", i, got[i], want)
		}
	}
	for _, name := range []string{"Color", "Red", "name"} {
		fix := "import " + name + " from pkg/demo/core/color"
		if !strings.Contains(strings.Join(got, "\n"), fix) {
			t.Errorf("no warning has the fix %q:\n%s", fix, strings.Join(got, "\n"))
		}
	}
	if !strings.Contains(got[0], "lib/facade") {
		t.Errorf("the warning should name the facade imported through: %s", got[0])
	}
	if quiet := warnings(`export import lib/color (Color, Red, name)`); len(quiet) != 0 {
		t.Fatalf("a re-export without @deprecated warned:\n%s", strings.Join(quiet, "\n"))
	}
}
