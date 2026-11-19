package runner

import (
	"fmt"
	"math/rand"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/sunholo-data/ailang/internal/eval"
	"github.com/sunholo-data/ailang/internal/repl"
)

// TestShadowGeneratedREPLRegistry runs the generated shadowing corpus (see
// TestShadowGeneratedEngines) through the REPL's module path: the registry
// the wasm playground's ailangLoadModule/ailangCall and the REPL's imports
// use, which compiles a module through its own copy of the phases (it kept
// #2401's capture after the native pipelines were fixed). Every case, called
// with true and with false, must give the renamed-apart twin's output on the
// evaluator.
func TestShadowGeneratedREPLRegistry(t *testing.T) {
	r := rand.New(rand.NewSource(shadowGenSeed))
	cases := make([]shadowCase, shadowGenCases)
	for i := range cases {
		cases[i] = genShadowCase(r, i)
	}
	dir := t.TempDir()
	twinFile := filepath.Join(dir, "shadow_twin.ail")
	want := strings.Split(runOnEngine(t, twinFile, shadowModule("shadow_twin", cases, true), "eval"), "\n")
	if len(want) != 2*len(cases)+1 {
		t.Fatalf("twin on the evaluator printed %d lines, want %d", len(want)-1, 2*len(cases))
	}

	reg := repl.NewModuleRegistry()
	loadStdlibIntoRegistry(t, reg, "option", "result", "list", "string", "io")
	// The cases are private functions; export them so they can be called.
	src := strings.ReplaceAll(shadowModule("test/shadow_gen", cases, false), "\nfunc case", "\nexport func case")
	if _, err := reg.LoadModule("test/shadow_gen", src); err != nil {
		t.Fatalf("registry load: %v", err)
	}
	bad := 0
	for i, c := range cases {
		for j, b := range []bool{true, false} {
			v, err := reg.InvokeExport("test/shadow_gen", c.name, []eval.Value{&eval.BoolValue{Value: b}})
			got := fmt.Sprintf("error: %v", err)
			if err == nil {
				got = eval.Show(v)
			}
			if w := want[2*i+j]; got != w {
				bad++
				t.Errorf("registry: %s(b=%v) = %s, oracle %s\n  %s", c.name, b, got, w, c.desc)
			}
		}
	}
	if bad > 0 {
		t.Logf("registry: %d of %d results differ from the oracle", bad, 2*len(cases))
	}
}

// loadStdlibIntoRegistry loads std modules from the repository's std/ in the
// order given (dependencies first), as cmd/wasm loads the embedded copy.
func loadStdlibIntoRegistry(t *testing.T, reg *repl.ModuleRegistry, names ...string) {
	t.Helper()
	for _, name := range names {
		content, err := os.ReadFile(filepath.Join("..", "..", "std", name+".ail"))
		if err != nil {
			t.Fatal(err)
		}
		if _, err := reg.LoadModule("std/"+name, string(content)); err != nil {
			t.Fatalf("load std/%s: %v", name, err)
		}
	}
}
