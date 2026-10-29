package runner

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"github.com/sunholo-data/ailang/internal/bytecode"
	"github.com/sunholo-data/ailang/internal/pipeline"
	"github.com/sunholo-data/ailang/internal/types"
)

// runCapturingStdout is runOnEngine plus whatever the program wrote to the
// process's real stdout. The evaluator's prelude `println` builtin writes
// with fmt.Print, not through the effect context's IOWriter, so its output
// only shows up there.
func runCapturingStdout(t *testing.T, file, source, engine string) string {
	t.Helper()
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	orig := os.Stdout
	os.Stdout = w
	done := make(chan string)
	go func() {
		var b bytes.Buffer
		_, _ = io.Copy(&b, r)
		done <- b.String()
	}()
	ioOut := func() string {
		defer func() { os.Stdout = orig }()
		return runOnEngine(t, file, source, engine)
	}()
	_ = w.Close()
	return <-done + ioOut
}

// M-VM-COVERAGE G2: an entry module's prelude `println`, used without an
// import, runs natively on the VM. The four cases are the design doc's
// required tests, plus println used as a value.
func TestPreludePrintlnOnVM(t *testing.T) {
	cases := []struct {
		name, source, want string
	}{
		{
			name: "prelude_no_import",
			source: `module prelude_no_import
export func main() -> () ! {IO} {
  println("a");
  println(show(1 + 2))
}
`,
			want: "a\n3\n",
		},
		{
			// V21: a user func println shadows the prelude, and still compiles.
			name: "user_func_shadows",
			source: `module user_func_shadows
func println(s: string) -> () ! {IO} = ()
export func main() -> () ! {IO} {
  println("must not print")
}
`,
			want: "",
		},
		{
			name: "explicit_import",
			source: `module explicit_import
import std/io (println)
export func main() -> () ! {IO} {
  println("explicit")
}
`,
			want: "explicit\n",
		},
		{
			// V4 shape: std/io imported for something else, println from the prelude.
			name: "selective_import_without_println",
			source: `module selective_import_without_println
import std/io (readLine)
import std/list (forEachE)
export func main() -> () ! {IO} {
  forEachE(println, ["x", "y"]);
  let p = println;
  p("value")
}
`,
			want: "x\ny\nvalue\n",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			file := filepath.Join(t.TempDir(), tc.name+".ail")
			for _, engine := range allEngines {
				if got := runCapturingStdout(t, file, tc.source, engine); got != tc.want {
					t.Errorf("%s: stdout = %q, want %q", engine, got, tc.want)
				}
			}
		})
	}
}

// A library module (no exported main) gets no prelude, so a bare println is
// still a type error: the VM resolution must not widen the prelude.
func TestPreludePrintlnStillRejectedInLibrary(t *testing.T) {
	file := filepath.Join(t.TempDir(), "lib_bare_println.ail")
	source := "module lib_bare_println\nexport func hello() -> () ! {IO} = println(\"lib\")\n"
	if err := os.WriteFile(file, []byte(source), 0600); err != nil {
		t.Fatal(err)
	}
	res, err := pipeline.Run(pipeline.Config{Mode: pipeline.ModeCheck, TransientRoot: true}, pipeline.Source{Filename: file, Code: source})
	if err == nil && len(res.Errors) == 0 {
		t.Fatal("a library module's bare println type-checked; the prelude is entry-only")
	}
}

// pipeline.PreludeSurface() is the single source of truth for the prelude;
// bytecode.PreludeFuncs only says how the VM runs each entry. They must
// name the same functions, with matching arity.
func TestPreludeSurfaceHasVMImplementation(t *testing.T) {
	var surface, vmNames []string
	for _, b := range pipeline.PreludeSurface() {
		surface = append(surface, b.Name)
		pf, ok := bytecode.PreludeFuncs[b.Name]
		if !ok {
			continue
		}
		fn, isFn := b.Scheme.Type.(*types.TFunc2)
		if !isFn || len(fn.Params) != pf.Arity {
			t.Errorf("prelude %s: VM arity %d does not match scheme %s", b.Name, pf.Arity, b.Scheme.Type)
		}
		found := false
		for _, n := range bytecode.EffectBuiltinNames {
			found = found || n == pf.Effect
		}
		if !found {
			t.Errorf("prelude %s: VM effect %q is not in bytecode.EffectBuiltinNames", b.Name, pf.Effect)
		}
	}
	for n := range bytecode.PreludeFuncs {
		vmNames = append(vmNames, n)
	}
	sort.Strings(surface)
	sort.Strings(vmNames)
	if strings.Join(surface, ",") != strings.Join(vmNames, ",") {
		t.Fatalf("prelude surface %v and VM prelude table %v disagree", surface, vmNames)
	}
}
